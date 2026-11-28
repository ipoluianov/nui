# nui audit: parsers, persistence, app services

Scope: layout_xml.go, settings.go, formstate.go, instance*.go, markdownview.go, language.go, i18n/, strings*.go, shortcut.go, datepicker.go, timepicker.go, calendar.go, colorpicker.go, print*.go, filedialog_builtin.go.

Reproducers (full files) are in `doc/audit/_repro/parsing/`:
- `zz_audit_parsing_test.go` goes in `ui/` (package `ui`)
- `regression_zz_audit_parsing_test.go` goes in `ui/regression/` as `zz_audit_parsing_test.go` (package `regression`)

Run them with `go test ./ui -run 'TestAudit<Name>' -count=1 -v` (add `-race` where noted).

Checked and found clean (no finding): XML layout with NaN/Inf/huge/negative numeric attributes, bad splitters, tabs and tables, namespaces, invalid UTF-8 (50 hand-written cases, built and painted at 500x400, 10x10 and 0x0, plus 45 s of fuzzing), nesting 50k deep; `ParseShortcut`, `parseMnemonic`, `parsePageRange`, `pdfString`, `parseFileNames` (fuzzed); PDF output with titles holding parentheses, backslashes, Cyrillic, emoji, RTL, invalid UTF-8 and NUL (checked with `pdfinfo`); format verbs and empty fields of every built-in UIStrings language; `i18n.Plural` with MinInt.

---

## 1. Text layout panics on an RTL character plus a lone CR / paragraph separator (CRITICAL)

- **Severity:** critical (crashes the application from any widget that shows the text)
- **Location:** `ui/canvas_text_layout.go:395` (`layoutParagraph` → `shaping.LineWrapper.WrapParagraphF` → go-text `cutRun` / `inclusiveGlyphRange`). This is outside the assigned area. MarkdownView fuzzing found it, and it affects every text path (`MeasureText`, `Label`, `DrawText`...).
- **Root cause:** a paragraph can hold an RTL rune (Hebrew, Arabic, Thaana...) and a rune of bidi class B (paragraph separator): `\r` alone, U+001C..U+001E, U+0085 or U+2029. nui splits paragraphs only at `\n`. The bidi segmentation then gives a run that is shaped to zero glyphs (or a run boundary that doesn't match), and go-text's line wrapper indexes `glyphs[0]` of an empty slice. A lone `\r` is common (old Mac line endings, pasted text, file contents, chat messages).
- **Reproducer** (`TestAuditShapingPanic`):
  ```go
  func TestAuditShapingPanic(t *testing.T) {
      MeasureText(FontFamilySans, 14, "א\r0") // also "aא\rb", "א b", "א\u0085b", "0־\x1e0", "0ا\x1e0"
      l := NewLabel("00־\x1e0")
      form := newTestForm(t, l)
      form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
  }
  ```
  Output:
  ```
  panic: runtime error: index out of range [0] with length 0
  github.com/go-text/typesetting/shaping.inclusiveGlyphRange(...) shaping/wrapping.go:213
  github.com/go-text/typesetting/shaping.cutRun(...) shaping/wrapping.go:241
  ...(*LineWrapper).WrapParagraphF
  github.com/ipoluianov/nui/ui.(*TextLayout).layoutParagraph  ui/canvas_text_layout.go:395
  github.com/ipoluianov/nui/ui.(*TextLayout).layout           ui/canvas_text_layout.go:264
  github.com/ipoluianov/nui/ui.shapedLine                     ui/text_complex.go:86
  github.com/ipoluianov/nui/ui.measureText                    ui/text_renderer.go:400
  ```
  The same text without an RTL rune (`"abc\rdef"`) does not panic, and neither does `"א\r\n0"`.
- **Expected:** no panic. The separator is drawn as nothing (or treated as a line break).
- **Fix direction:** before the bidi/shaping step, either split paragraphs at every bidi class-B rune (`\r`, U+001C-U+001E, U+0085, U+2029) as well as `\n`, or replace those runes with a space or zero-width char. Drop shaped runs with `len(out.Glyphs)==0` before `WrapParagraphF`. Add a regression test with the strings above.

---

## 2. SingleInstance.Close races with the accept goroutine, which then panics (HIGH)

- **Severity:** high (an unrecoverable panic in a background goroutine kills the process; also a data race)
- **Location:** `ui/instance.go:86` (`c.listener.Accept()` read on every loop iteration) and `ui/instance.go:116` (`c.listener = nil` in `Close`)
- **Root cause:** the goroutine started by `OnActivate` reads the field `c.listener` on each iteration, and `Close` sets it to nil from the UI thread without synchronization. If `Close` runs while the goroutine is handling a connection (after `Accept` returned and before the next `Accept`), the next iteration calls `Accept` on a nil interface.
- **Reproducer** (`TestAuditInstanceCloseRace`):
  ```go
  func TestAuditInstanceCloseRace(t *testing.T) {
      t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
      for i := 0; i < 50; i++ {
          dir := t.TempDir()
          first, _ := acquireSingleInstance(dir, nil)
          first.OnActivate(func([]string) {})
          sock := instanceSocketPath(dir)
          done := make(chan struct{})
          go func() {
              defer close(done)
              for j := 0; j < 20; j++ {
                  if c, err := net.Dial("unix", sock); err == nil { c.Close() }
              }
          }()
          time.Sleep(time.Millisecond)
          first.Close()
          <-done
      }
  }
  ```
  `go test ./ui -run TestAuditInstanceCloseRace -race -count=1` prints `WARNING: DATA RACE` (write at instance.go:116, read at instance.go:86) and then:
  ```
  panic: runtime error: invalid memory address or nil pointer dereference
  goroutine 84 [running]:
  github.com/ipoluianov/nui/ui.(*SingleInstance).OnActivate.func1()  ui/instance.go:86
  ```
  Without `-race`, `-count=40` also crashes.
- **Expected:** `Close` stops the goroutine cleanly.
- **Fix direction:** capture the listener in a local variable before starting the goroutine (`ln := c.listener; go func(){ for { conn, err := ln.Accept() ...` ). Also guard the field with a mutex, or never nil it while the goroutine may read it.

---

## 3. Calendar / DatePicker break in time zones where DST starts at midnight (HIGH)

- **Severity:** high (wrong dates shown and chosen, keyboard navigation stuck). The affected zones are in use now: America/Santiago (Chile, every September), Africa/Cairo, America/Havana, Asia/Beirut. `zdump` shows that 2026 midnight doesn't exist in any of them.
- **Location:** `ui/calendar.go:54` (`DateOnly` builds local midnight), `ui/calendar.go:451` (draw loop `day = day.AddDate(0,0,1)`), `ProcessKeyDown`/`selectDate` (`calendar.go:328`, `:175`), `DatePicker.SetDate/changeDate`, `firstOfMonth`, `addMonthsClamped`.
- **Root cause:** dates are kept as "local midnight". On a day whose 00:00 does not exist, `time.Date(y,m,d,0,0,0,0,Local)` normalizes to 23:00 of the previous day. Then `DateOnly(DateOnly(x))` is not the same day. Iterated `AddDate` keeps the 23:00 offset, so every later cell is labeled with the previous day's number. `Equal` comparisons against `c.date` fail, and `selectDate(DateOnly(next day))` lands back on the same day.
- **Reproducer** (`TestAuditCalendarDSTSantiago`, `TestAuditCalendarDST`):
  ```go
  func TestAuditCalendarDSTSantiago(t *testing.T) {
      loc, _ := time.LoadLocation("America/Santiago")
      old := time.Local; time.Local = loc; defer func() { time.Local = old }()
      c := NewCalendar()
      c.SetFirstDayOfWeek(time.Monday)
      c.SetDate(time.Date(2026, 9, 1, 12, 0, 0, 0, loc))
      day := c.gridStart()
      var labels []int
      for i := 0; i < 14; i++ { labels = append(labels, day.Day()); day = day.AddDate(0, 0, 1) }
      t.Logf("labels: %v", labels)
      t.Logf("DateOnly(Sep 6 12:00) = %v", DateOnly(time.Date(2026, 9, 6, 12, 0, 0, 0, loc)))
      c.SetDate(time.Date(2026, 9, 4, 0, 0, 0, 0, loc))
      for i := 0; i < 4; i++ { c.ProcessKeyDown(KeyArrowRight, KeyModifiers{}) }
      if c.Date().Day() != 8 { t.Errorf("after 4 x Right from Sep 4: %v", c.Date()) }
  }
  ```
  Output:
  ```
  labels of the first two rows of Sep 2026: [31 1 2 3 4 5 5 6 7 8 9 10 11 12]   <- "5" twice, every later cell off by one
  DateOnly(Sep 6 2026 12:00) = 2026-09-05 23:00:00 -0400 -04                   <- Sep 6 cannot be selected
  Right x4 from Sep 4: [Sep 5 00:00 Sep 5 00:00 Sep 5 00:00 Sep 5 00:00]         <- stuck
  ```
  With `America/Sao_Paulo` in Nov 2018 (`TestAuditCalendarDST`), the selected Nov 20 never matches a drawn cell, so it is not highlighted. `partAt` computes cell dates directly from `gridStart` (not iteratively): the cell labeled "20" is Nov 21, so a click picks the next day.
- **Expected:** every day is shown once, can be selected, and is highlighted. The arrow keys move one day at a time.
- **Fix direction:** do not do date arithmetic on local midnight. Keep the selection as a civil date (year, month, day) or as `time.Date(y,m,d,12,0,0,0,Local)` (noon always exists). Compare dates by `Date()` triple. In `draw`, compute each cell as `gridStart.AddDate(0,0,row*7+col)` from a noon-based start, as `partAt` does. The returned `Date()` can still be normalized for the API, but then document it (or return the noon or the first existing instant of the day).

---

## 4. Single-instance protocol loses or truncates arguments with a newline or over 64 KB (MEDIUM)

- **Severity:** medium (a file name with `\n` is legal on Unix: the running copy gets wrong files or isn't activated at all; also a way to inject a fake "show")
- **Location:** `ui/instance.go:161` (`msg.WriteString(instanceArgCommand + arg + "\n")`, no escaping), `ui/instance.go:92-104` (line-based `bufio.Scanner` with the default 64 KB token limit; an unknown line ends the read without "show")
- **Reproducer** (`TestAuditInstanceNewlineArg`): the first copy is acquired in a temp dir, then the second starts with the args below. The UI queue is pumped with `platforms.RunHeadlessPosted()`.
  ```
  sent ["/tmp/a\nshow" "/tmp/b"] -> got ["/tmp/a"]            (truncated; "show" injected)
  sent ["/tmp/x\ny.txt"]          -> NOTHING delivered (no activation)
  sent ["aaaa...(70000 bytes)"]   -> NOTHING delivered (no activation)
  ```
- **Expected:** the running copy gets exactly the arguments passed, then "show".
- **Fix direction:** encode each argument (e.g. `strconv.Quote`/`Unquote`, base64, or JSON-encode the whole `[]string`, one message per connection). Raise the scanner buffer (`sc.Buffer(..., N)`) or read the whole connection with a size cap. When an unexpected line or a scan error is seen, still deliver the activation (or reject explicitly).

---

## 5. Built-in folder dialog: nil dereference in `activate` when a listed symlink stops being a folder (MEDIUM)

- **Severity:** medium (panic on the UI thread)
- **Location:** `ui/filedialog_builtin.go:349` (`d.name.SetText(e.Name())`). `d.name` is nil in `fileDialogDirectory` mode.
- **Root cause:** in folder mode the list shows folders and symlinks to folders (`isDirLink`). `activate` re-checks `isDirLink` when the row is activated. If the link's target is gone or changed, the row falls through to the file branch, which assumes a name field exists.
- **Reproducer** (`ui/regression`, `TestAuditDirDialogDanglingLink`):
  ```go
  ui.UseBuiltinFileDialogs(true); defer ui.UseBuiltinFileDialogs(false)
  dir := t.TempDir(); real := filepath.Join(dir, "real")
  os.Mkdir(real, 0o755); os.Symlink(real, filepath.Join(dir, "link"))
  form := newForm(400, 300); w := uitest.Show(t, form)
  form.ShowSelectDirectoryDialog(ui.SelectDirectoryDialogOptions{DefaultDirectory: dir}, func(string, error) {})
  d := w.Dialog()
  table := d.FindAll(func(x ui.Widgeter) bool { _, ok := x.(*ui.Table); return ok })[0]
  os.Remove(real)
  d.Click(table); d.Key(ui.KeyEnd, ui.KeyModifiers{}); d.Key(ui.KeyEnter, ui.KeyModifiers{})
  ```
  Output:
  ```
  panic: runtime error: invalid memory address or nil pointer dereference
  ui.(*TextBox).SetText(0x0, ...)            ui/textbox.go:308
  ui.(*fileDialog).activate(...)             ui/filedialog_builtin.go:349
  ```
- **Expected:** nothing happens, or the list is refreshed.
- **Fix direction:** in `activate`, `if d.name == nil { return }` (or `d.openDir(d.dir)` to refresh) before treating the row as a file.

---

## 6. A settings file containing `null` makes every later `Set` panic (MEDIUM)

- **Severity:** medium (the app crashes at the first `Set`/`RememberState` close after a corrupted or hand-edited settings file; `OpenSettings` reports no error)
- **Location:** `ui/settings.go:67` (`json.Unmarshal(data, &s.values)`), panic at `ui/settings.go:79` (`s.values[key] = raw`)
- **Root cause:** unmarshaling JSON `null` into a map sets the map to nil.
- **Reproducer** (`TestAuditSettingsNull`):
  ```go
  path := filepath.Join(t.TempDir(), "settings.json")
  os.WriteFile(path, []byte("null"), 0o644)
  s, err := OpenSettingsFile(path) // err == nil
  s.Set("theme", "dark")           // panic: assignment to entry in nil map
  ```
- **Expected:** empty settings that work.
- **Fix direction:** after `Unmarshal`, `if s.values == nil { s.values = map[string]json.RawMessage{} }`. Also unmarshal into a temporary map and keep an empty map on any error, so a failed load never leaves the map nil or partly filled.

---

## 7. `Settings.Save` is documented "safe for concurrent use" but concurrent saves fail (LOW-MEDIUM)

- **Severity:** low-medium (spurious errors; an older snapshot can overwrite a newer one)
- **Location:** `ui/settings.go:140-146`
- **Root cause:** every `Save` writes the same `settings.json.tmp` and renames it outside the mutex. While one goroutine truncates or writes the tmp file, another renames it. The rename that comes second fails with ENOENT, and the order of the final rename doesn't follow the order of the snapshots.
- **Reproducer** (`TestAuditSettingsConcurrentSave`): 200 keys, 8 goroutines x 50 `Save()`.
  ```
  failed saves: 43 of 400, first: rename .../settings.json.tmp .../settings.json: no such file or directory
  ```
- **Expected:** every `Save` succeeds, and the file holds the last snapshot.
- **Fix direction:** serialize `Save` with a dedicated save mutex held across marshal+write+rename, or use `os.CreateTemp(dir, "settings-*.tmp")` per call. Optionally `f.Sync()` before the rename, so the "a crash doesn't leave half of them" promise holds on power loss.

---

## 8. Built-in folder dialog with a missing default directory: OK returns "" (looks like Cancel) (LOW)

- **Severity:** low
- **Location:** `ui/filedialog_builtin.go:225-230` (`OnDialogShow`: `openDir(start)` fails and leaves `d.dir == ""`), `:358` (`d.finish([]string{d.dir})`)
- **Root cause:** when `DefaultDirectory` doesn't exist or can't be read, `openDir` returns early. The list and path stay empty, and `d.dir` stays "". In folder mode, OK then returns "", which callers read as cancel. In open/save mode a typed name becomes a path relative to the process cwd (`filepath.Join("", name)`).
- **Reproducer** (`ui/regression`, `TestAuditDirDialogMissingDefault`): `ShowSelectDirectoryDialog(..., DefaultDirectory: "/nonexistent/dir")`, then click "Select Folder". Output: `calls=1 path=""`.
- **Expected:** fall back to the nearest existing parent, home, or cwd. Never return an empty or relative path on OK.
- **Fix direction:** in `OnDialogShow`, if `openDir(start)` didn't set `d.dir`, try the parents of `start`, then `os.UserHomeDir()`, then `os.Getwd()`. In `accept`, refuse when `d.dir == ""`.

---

## 9. Built-in file dialog cannot open files whose names start with `"` or have leading/trailing spaces (LOW)

- **Severity:** low
- **Location:** `ui/filedialog_builtin.go:420-436` (`parseFileNames`), used by `activate` (`:349-350`)
- **Root cause:** activating a row puts the raw name into the text field, and `accept` re-parses it. Names are `TrimSpace`d, and a name starting with `"` is split at quotes. Selecting several files joins names in quotes without escaping embedded quotes.
- **Reproducer** (`ui/regression`, `TestAuditOpenDialogQuoteName`): a file `"q" x.txt` exists, and Enter is pressed on its row. Output: `calls=0 got=[] blocked=true`. The dialog stays open with a "file not found" message box for `q`.
- **Expected:** the activated file is returned as is.
- **Fix direction:** in `activate`, finish with `filepath.Join(d.dir, e.Name())` directly instead of going through the text field. For typed or multi-selected names, use an escaping scheme (or keep the selected paths in a field and use them when the text was not edited).

---

## 10. MarkdownView keeps the old layout after a font change (LOW)

- **Severity:** low (wrong rendering: text keeps the old size, wrapping and hit-testing of links don't match)
- **Location:** `ui/markdownview.go:251-256` (`relayout` returns when `width == c.width`). Nothing resets `c.width` on a font size, family or theme-metrics change.
- **Reproducer** (`TestAuditMarkdownFontChange`):
  ```go
  v := NewMarkdownView(); form := newTestForm(t, v)
  v.SetMarkdown("Hello **world** and some more words to wrap")
  form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
  v.SetFontSize(30); form.UpdateLayout()
  form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
  // v.lines[0].parts[0].size == 14, v.FontSize() == 30
  ```
- **Expected:** the text is laid out again at 30.
- **Fix direction:** keep the font key (family, size, scale) used for the layout and re-layout when it differs. Or override `ProcessPropChange`/`applyThemeMetrics` to set `c.width = 0` and call `relayout`.

---

## 11. TimePicker.SetShowSeconds(false) changes the value without OnTimeChanged (LOW)

- **Severity:** low (breaks the rule in doc/rules.md that "On…Changed fires on every change of the value, by the code too")
- **Location:** `ui/timepicker.go:107-112`
- **Reproducer** (`TestAuditTimePickerHideSeconds`): `SetShowSeconds(true); SetTime(10,20,30)`, then connect `OnTimeChanged`, then `SetShowSeconds(false)`. Output: `duration now 10h20m0s, OnTimeChanged fired 0 times`.
- **Fix direction:** when `values[2]` was non-zero, emit `OnTimeChanged` after zeroing it, as `SetTime` does.

---

## 12. ParseHexColor accepts malformed strings (LOW)

- **Severity:** low (wrong colors from invalid input: the color picker's hex box, the XML `color` attribute)
- **Location:** `ui/colorpicker.go:734` (`fmt.Sscanf(pair, "%02x", &b)`)
- **Root cause:** `Sscanf` skips leading spaces, accepts a `0x` prefix, and stops early.
- **Reproducer** (`TestAuditHex`):
  ```
  "# f0000"  -> {15 0 0 255} true     (space accepted, "#0f0000")
  "#\tf0000" -> {15 0 0 255} true
  "#0x1234"  -> {0 18 52 255} true    ("0x" taken as 00)
  "#ff00 0"  -> {255 0 0 255} true
  ```
- **Expected:** `false` for anything that isn't exactly 3/6/8 hex digits.
- **Fix direction:** check every byte with a hex-digit test, then decode with `strconv.ParseUint(pair, 16, 8)` or `hex.DecodeString`.

---

## 13. PrintJob: an invalid page size panics; zero-size or zero-page jobs give an invalid PDF (LOW)

- **Severity:** low (public API input, no validation)
- **Location:** `ui/print.go:117-125` (`pageSize`), `:136` (`image.NewRGBA`), `:168-178` (`WritePDF`)
- **Reproducer** (`TestAuditPDF`):
  - `&PrintJob{Pages: 1, DPI: math.Inf(1)}.SavePDF(p)` and `&PrintJob{Pages: 1, DPI: 10, Paper: PaperSize{"x", math.Inf(1), 10}}` panic with `image: NewRGBA Rectangle has huge or negative dimensions` (`renderPage` print.go:136 with width `0x8000000000000000`). A huge finite DPI (e.g. 1e6) asks for terabytes.
  - `Paper{"x", 0.0001, 0.0001}` writes `/Width 0 /Height 0`. `pdftoppm`: `Syntax Error: Bad image parameters`.
  - `Pages: 0` or `Pages: -3` writes `/Count 0 /Kids [ ]`. `pdfinfo`: `Syntax Error: Invalid page count 0`.
  - `SavePDF` leaves the partly written or empty file behind on error or panic.
- **Expected:** an error for a non-finite or out-of-range DPI or paper size, pages clamped to at least 1 pixel, and an error (or one blank page) for zero pages.
- **Fix direction:** validate in `WritePDF`: `math.IsInf/IsNaN`, a cap on `w*h` (e.g. 2^28 pixels), `w,h >= 1`, `count >= 1`. In `SavePDF`, remove the file on error.

---

## 14. Saved window state with a negative X or Y is neither restored nor clamped (LOW, verified by reading)

- **Severity:** low
- **Location:** `ui/form.go:751` (`if c.posX >= 0 && c.posY >= 0 { if c.clampOnShow { c.clampToScreen() }; c.wnd.Move(...) }`) and `ui/form.go:700` (`centerOnScreen := c.posX < 0 && c.posY < 0`)
- **Root cause:** `-1` is used as "no position". `SetState` stores any position, but `createWindow` only moves (and clamps) when both coordinates are non-negative. On Windows and macOS, a monitor left of or above the primary has negative coordinates. A window closed there is not restored to its place on the next start: on Windows the window is created at `CW_USEDEFAULT` and never moved (`internal/platforms/native_window_windows.go:126`). `clampToScreen`, which is meant to bring back a window from a monitor that is gone, never runs for these states. In the headless backend, a saved `{X:-3000,Y:100}` opens centered at 760,390.
- **Fix direction:** use a separate `positionSet bool` instead of the `-1` sentinel. When it is set, always `clampToScreen()` then `Move`, for any sign of coordinates.

---

## 15. Quadratic parsing time: markdown `[`/backticks and XML character data (LOW)

- **Severity:** low (a DoS only if the markdown or XML comes from untrusted input)
- **Location:** `ui/markdownview.go:222-224` (for each `[`, `strings.Index` to the end of the paragraph; the same for an unmatched backtick at `:211`), `ui/layout_xml.go:116` (`InnerText += string(t)` per CharData token)
- **Reproducer** (`TestAuditMarkdownQuadratic`, `TestAuditXMLCharData`):
  ```
  parseMarkdown(strings.Repeat("[", n)):  n=20000 5ms, 40000 22ms, 80000 96ms   (x4 per x2; ~1 MB -> ~15 s)
  parseLayoutXML("<column>"+strings.Repeat("aaaaaaaaaa<!---->", n)+"</column>"):
        166 KB 0.16 s, 332 KB 0.60 s, 664 KB 2.2 s
  ```
- **Fix direction:** markdown: remember "no `](` / no closing backtick after position p" and skip the search for later brackets (or scan once for the positions of `](`, `)` and the backticks). XML: collect the CharData in a `strings.Builder` per node, or drop `InnerText` (no element uses it).

---

## 16. Single-instance socket falls back to the shared /tmp (LOW, verified by reading)

- **Severity:** low (local, other user)
- **Location:** `ui/instance.go:135-145`
- **Root cause:** without `XDG_RUNTIME_DIR` (su, cron, ssh sessions without systemd-logind), and with a settings path longer than 103 bytes, the socket goes to `os.TempDir()/nui-<sha256(dir)[:8]>.sock`. On Linux that is the shared `/tmp`, and the name can be predicted from the path. Another local user can create that socket first. `os.Remove` fails (sticky bit), `net.Listen` fails, and the real first copy runs without a listener. Every later start of the victim then connects to the attacker's socket and sends its command-line arguments (file paths) to it.
- **Fix direction:** on Linux, fall back to a private directory (`os.MkdirTemp`-style `/tmp/nui-<uid>/`, mode 0700, owner checked) or to an abstract unix socket (`"@nui-..."`, Linux-only). Check the socket's owner before writing arguments to it.

---

## 17. MarkdownView links go to OpenURL without a scheme check, and the child process is never reaped (LOW, verified by reading)

- **Severity:** low
- **Location:** `ui/markdownview.go:444` → `ui/link.go:154-167`
- **Root cause:** (a) `OpenURL` runs `cmd.Start()` and never calls `Wait()`. On Unix every opened link leaves a zombie `xdg-open`/`open` process until the application exits. (b) Any link text from markdown is passed as is: `file:///...`, a local path to a `.desktop`/`.command` file, or on macOS a value starting with `-` (taken as an option of `open`, since there is no `--`).
- **Fix direction:** `go cmd.Wait()` after `Start`. In MarkdownView (default handler), allow only `http`, `https` and `mailto` unless the application opts in. Pass `--` before the URL where the opener supports it, and reject values starting with `-`.
