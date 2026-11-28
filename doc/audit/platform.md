# Audit: native layer (internal/platforms, ui/atspi, ui/clipboard, ui/trayicon, ui/dragdrop)

Reproducer files (removed from the worktree, copied here):
`doc/audit/_repro/platform/`

- `zz_audit_platform_test.go` -> `internal/platforms/` (pure Go: URI round trip, headless NaN scale, DIB fuzz)
- `zz_audit_platform_x11_test.go` -> `internal/platforms/` (needs Xvfb; clipboard tests)
- `zz_audit_platform_drag_test.go` -> `internal/platforms/` (needs Xvfb + libXtst; drag-out hang)
- `ui_zz_audit_platform_test.go` -> `ui/zz_audit_platform_test.go` (headless: clipboard PNG bomb, aliasing, AT-SPI NaN)

To run the X11 tests: `Xvfb :97 -screen 0 1280x1024x24 -nolisten tcp &` then
`NUI_AUDIT_X=1 DISPLAY=:97 go test -p 1 -run 'TestAudit<Name>' -count=1 -v -timeout 60s ./internal/platforms/`.
Each X11 test should run in its own `go test` call (they share the process-wide X connection).

---

## 1. Pasting an image from the clipboard can kill the process: PNG header dimensions are not limited

- **Severity:** high
- **File:** ui/clipboard.go:96 (`ClipboardGetImage` -> `png.Decode`)
- **Root cause:** `ClipboardGetImage` passes whatever PNG is on the clipboard straight to `png.Decode`. Any other application can put that data there. Go's PNG decoder allocates the whole image from the IHDR width and height *before* it reads any pixel data. It only checks for integer overflow. An 89-byte PNG that claims 1000000x1000000 RGBA makes the decoder ask for a 4 TB block. The result is `fatal error: out of memory`, which `recover` cannot catch, so the app dies on Paste. Smaller claims, such as 30000x30000, really allocate gigabytes. A real 8000x8000 PNG of 434 KB allocated 61 MB.
- **Reproducer** (ui package, headless; run under `ulimit -v` so the test machine is safe):
  ```go
  func TestAuditClipboardImageHugeHeader(t *testing.T) {
      chunk := func(typ string, data []byte) []byte {
          b := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
          b = append(b, typ...); b = append(b, data...)
          return binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(append([]byte(typ), data...)))
      }
      ihdr := binary.BigEndian.AppendUint32(nil, 1000000)
      ihdr = binary.BigEndian.AppendUint32(ihdr, 1000000)
      ihdr = append(ihdr, 8, 6, 0, 0, 0)
      var z bytes.Buffer; zw := zlib.NewWriter(&z); zw.Write(make([]byte, 100)); zw.Close()
      p := []byte("\x89PNG\r\n\x1a\n")
      p = append(p, chunk("IHDR", ihdr)...); p = append(p, chunk("IDAT", z.Bytes())...); p = append(p, chunk("IEND", nil)...)
      ClipboardSetData(MIMEPNG, p)
      _, err := ClipboardGetImage()
      t.Logf("survived: err=%v", err)
  }
  ```
  `go test -c ./ui/ && (ulimit -v 3000000; NUI_AUDIT_BOMB=1 ./ui.test -test.run TestAuditClipboardImageHugeHeader -test.v)`
- **Actual output:**
  ```
  clipboard payload 89 bytes; pasting
  runtime: out of memory: cannot allocate 4000002867200-byte block (7995392 in use)
  fatal error: out of memory
  runtime.(*mcache).allocLarge ... runtime.mallocgcLarge ...
  ```
- **Expected:** an error such as "image too large", with no crash.
- **Fix direction:** call `png.DecodeConfig` first and refuse any image where `Width*Height` is above a limit (for example 64 Mpx, or `Width` or `Height` above 32768), then decode. The same applies to any other place that decodes images from outside data (macOS TIFF on the clipboard, `loadPngFromBytes`).

## 2. Drag-out never returns if its window is closed during the drag (UI thread hangs)

- **Severity:** high
- **File:** internal/platforms/dragout_linux.go:76-80 (the `for { if !d.nextEvent(...) { continue } ... }` loop) together with dragout_linux.go:113 (`w.idle(false)` for all windows, including the dragging one)
- **Root cause:** `startFileDragSync` runs a nested loop. That loop only ends on Button-1 release, Esc or a drop. `nextEvent` runs `idle` for every open window, and `idle` calls `doClose` when `closeRequested` is set. `closeRequested` gets set by `Close()`, which is documented as safe from any goroutine, or by a WM_DELETE_WINDOW ClientMessage that `nextEvent` hands to `processEvent` → `onCloseRequest` → `Close`. Once `doClose` destroys the X window, the server drops the pointer and keyboard grabs. The button release then goes to some other window and is passed to that window, so the drag loop never sees it and never sees Esc either. The loop runs forever. `runPosted` is never called, so `InvokeSync` callers deadlock, and timers stop ticking.
- **Reproducer** (Xvfb + XTest; see `zz_audit_platform_drag_test.go`):
  ```go
  w := createWindow("drag", 0, 0, 300, 300, false, false); w.Show(); time.Sleep(300*time.Millisecond); processXEvents()
  d2 := xOpenDisplay(0)
  go func() {
      time.Sleep(300 * time.Millisecond)
      w.Close()                              // or a WM_DELETE_WINDOW from the WM
      time.Sleep(300 * time.Millisecond)
      XTestFakeMotionEvent(d2, 0, 150, 150, 0)
      XTestFakeButtonEvent(d2, 1, 1, 0); XTestFakeButtonEvent(d2, 1, 0, 0); xFlush(d2)
  }()
  go func() { w.startFileDragSync([]string{"/tmp/audit.txt"}); ret <- true }()
  // wait 4 s
  ```
- **Actual output:** the control run (no Close) returns: `drag returned (started=true), window closed=false`. The run with Close prints `drag still running 4 s after the button release; window closed=true`, which is a permanent hang.
- **Expected:** the drag ends (as a cancel) when its window closes.
- **Fix direction:** in the drag loop (and in `drop()`'s loops), return `DragEffectNone` once `d.c.platform.closed` becomes true or `closeRequested` is set. Alternatively, have `nextEvent` skip `doClose` for the drag's own window until the drag ends. Consider also running `runPosted()` inside `nextEvent`.

## 3. A copied text larger than the X request limit is "delivered" as nothing

- **Severity:** medium
- **File:** internal/platforms/clipboard_linux.go:445-458 (the text branch of `answerSelectionRequest`); the same applies in `clipboardHandOver`
- **Root cause:** for non-text items there is a `len(data) <= maxPropertySize(xDisplay)` check (the request is refused otherwise). The text branch has no such check, and INCR is not used for it. A text larger than the server's BIG-REQUESTS maximum (about 16 MB by default) goes into one `XChangeProperty`. The server answers BadLength, the property is never created, but the owner still sends SelectionNotify with `Property = property`, which means success. The application that pastes gets an empty or missing property. The same happens when the clipboard manager stores the text at exit.
- **Reproducer** (`TestAuditBigClipboardText`): nui owns the clipboard with 20 MB of `'a'`, and a second X connection requests UTF8_STRING and reads the property.
- **Actual output:**
  ```
  nui: X error 16 (request 97.0) on resource 0xf5          (BadLength)
  requestor got SelectionNotify, property=256               (success claimed)
  requestor reads property: status=0 type=0 format=0 nitems=0 (want 20971520)
  ```
- **Expected:** the requestor gets the text. If INCR is not implemented, the request should at least be refused (`Property = None`), as it is for the other formats.
- **Fix direction:** implement INCR on the owner side for large data: write the `INCR` type with the size, then write chunks on PropertyNotify(Delete) from the requestor. As a minimum, apply the `maxPropertySize` check to the text branch too and reply with `Property = None`.

## 4. Pasting from a slow INCR owner blocks the UI thread without limit (and memory without limit)

- **Severity:** medium
- **File:** internal/platforms/clipboard_linux.go:328-350 (`readClipboardIncr`)
- **Root cause:** the file says "a hung application must not hang this one" (`clipboardTimeout = 2s`). But `readClipboardIncr` calls `waitClipboardEvent` once per chunk, and each call starts a new 2 s deadline. There is no overall deadline and no limit on `result`. An owner that sends 1 byte every 1.9 s, or one that never sends the final zero-length chunk, keeps the UI thread inside `clipboardGetText` forever. A fast owner that never ends makes `result` grow until the process runs out of memory.
- **Reproducer** (`TestAuditIncrSlowDrip`): a second connection owns CLIPBOARD and answers with INCR, sending 6 one-byte chunks 1.5 s apart.
- **Actual output:** `clipboardGetText returned after 10.514579115s: len=6 err=<nil> (clipboardTimeout=2s)`. The UI thread was blocked for 10.5 s, and with more chunks the time has no limit.
- **Expected:** the whole paste is bounded, for example by an overall deadline (some multiple of clipboardTimeout) and a maximum size. The partial result is dropped and an error is returned.
- **Fix direction:** set one overall deadline for the whole INCR transfer, with an optional extension tied to progress but capped. Cap `len(result)` (for example at 256 MB) and abort with an error past that.

## 5. Latin-1 STRING received through INCR is not converted to UTF-8

- **Severity:** low
- **File:** internal/platforms/clipboard_linux.go:315-321 (`convertClipboard`)
- **Root cause:** `typ` is the type of the first property read. For a large transfer that type is `INCR`, so `data` is replaced by `readClipboardIncr`, but `typ` stays `INCR`. As a result the `typ == xXAString` → `latin1ToUTF8` branch is skipped, and raw Latin-1 bytes come back as an invalid UTF-8 Go string.
- **Reproducer** (`TestAuditIncrLatin1`): the owner refuses UTF8_STRING and answers STRING through INCR with the bytes `"caf\xe9"`.
- **Actual output:** `text="caf\xe9" err=<nil> (want "café")`
- **Expected:** `"café"`.
- **Fix direction:** decide on Latin-1 conversion from the requested target (STRING) or from the type of the INCR chunks, which `readClipboardProperty` returns.

## 6. Dragging out a relative path sends a URI with the first path segment as host; the receiver accepts remote-host URIs as local paths

- **Severity:** medium
- **File:** internal/platforms/dragout_linux.go:52 (sending); internal/platforms/dnd_linux.go:219 (receiving, `parseURIList`)
- **Root cause:** `(&url.URL{Scheme: "file", Path: f}).String()` with a relative `f` gives `file://docs/report.txt`, where `docs` becomes the host. `ui/dragdrop.go` passes `DragData.Files` through unchanged and does not make the paths absolute. On the receiving side, `parseURIList` ignores `u.Host`. So `file://docs/report.txt` is read as the local file `/report.txt`, which is a different file, and `file://report.txt` is dropped. A `file://otherhost/etc/passwd` sent by any XDND source is likewise turned into the local `/etc/passwd`, although by RFC 8089 it refers to another machine.
- **Reproducer** (`TestAuditDragOutURIRoundTrip`, pure Go):
  ```go
  for _, f := range []string{"/tmp/a b#c?d%e.txt", "docs/report.txt", "report.txt"} {
      uri := (&url.URL{Scheme: "file", Path: f}).String()
      got := parseURIList(uri + "\r\n")
  }
  ```
- **Actual output:**
  ```
  "/tmp/a b#c?d%e.txt" -> "file:///tmp/a%20b%23c%3Fd%25e.txt" -> ["/tmp/a b#c?d%e.txt"]   (ok)
  "docs/report.txt" -> "file://docs/report.txt" -> ["/report.txt"]                        (wrong file)
  "report.txt" -> "file://report.txt" -> []                                              (lost)
  ```
- **Expected:** the drag source makes paths absolute (`filepath.Abs`) before building the URI. The receiver accepts only an empty host or `localhost`, and possibly the local hostname.
- **Fix direction:** in `startFileDragSync`, run `filepath.Abs(f)` and use `url.URL{Scheme: "file", Path: abs}`. In `parseURIList`, skip a URI when `u.Host != "" && u.Host != "localhost"`.

## 7. AT-SPI `Value.CurrentValue = NaN` from any D-Bus client makes a Slider's value NaN

- **Severity:** medium
- **File:** ui/accessible.go:152 (`SetCurrent`), reached from ui/atspi_linux.go:954-966 (`atspiProperties.Set`)
- **Root cause:** `n.setValue(max(n.Min, min(n.Max, v)))`. Go's `min` and `max` return NaN when an argument is NaN, so the clamp does nothing. Any client on the accessibility bus can set the `CurrentValue` property to NaN (a double). `Slider.SetValue` keeps it: `snap(NaN)` is NaN. The slider then holds NaN, which leads to NaN positions while painting and NaN passed to `OnValueChanged` handlers. ScrollBar and NumBox happen to clamp NaN to 0 themselves.
- **Reproducer** (ui, headless):
  ```go
  s := NewSlider(); newTestForm(t, s); s.SetRange(0, 10); s.SetValue(5)
  accessibleNodeOf(s).SetCurrent(math.NaN())
  t.Logf("%v", s.Value())
  ```
- **Actual output:** `slider value after SetCurrent(NaN): NaN`
- **Expected:** NaN is rejected (`SetCurrent` returns false). ±Inf are already clamped to Min and Max.
- **Fix direction:** `if math.IsNaN(v) { return false }` in `AccessibleNode.SetCurrent`, or in `atspiProperties.Set`. `Slider.SetValue` should also ignore NaN.

## 8. Windows: `comCall` passes pointers to Go stack variables as plain `uintptr` through a normal function (GC/stack-move unsafe)

- **Severity:** low (latent memory corruption; verified by reading, Windows only)
- **File:** internal/platforms/dragout_windows.go:61-64 (`comCall`) and the caller at dragout_windows.go:184
- **Root cause:** the caller is `comCall(dataObject, c_IDataObjectSetData, uintptr(unsafe.Pointer(&fe)), uintptr(unsafe.Pointer(&sm)), 1)`. `fe` and `sm` are locals of `setPreferredDropEffect`. Their address only leaves the function as `uintptr`, so escape analysis keeps them on the goroutine stack. The `unsafe.Pointer` rule that keeps `uintptr(unsafe.Pointer(p))` valid applies only inside the argument list of `syscall.Syscall*` itself, or of functions marked `//go:uintptrescapes` such as `(*LazyProc).Call`. `comCall` is an ordinary variadic function. If its prologue, or the `append` inside it, grows the stack, the stack is copied, and the `uintptr` values in `args` still point at the old stack. `SetData` then reads freed or reused memory. The other vet warnings in the list are **not** bugs. clipboard_windows.go:78/124/141 and dragout_windows.go:178 point at GlobalLock memory owned by the OS. dpi_windows.go:97 and utils_windows.go:781 are `lParam` from the window procedure. dragout_windows.go:62 is a COM vtable. cocoa_darwin.go:495/1258 are a C string and NSBitmapImageRep memory. All of these are memory not owned by Go, so the conversions are fine.
- **Reproducer:** none (Windows-only and depends on timing). Verified by reading.
- **Fix direction:** add `//go:uintptrescapes` above `func comCall` (supported for variadic `...uintptr`, as `syscall.(*Proc).Call` does). Or call `syscall.SyscallN(vtbl[method], obj, uintptr(unsafe.Pointer(&fe)), ...)` directly at each call site. The same applies to the comCall uses in dragout_windows_test.go.

## 9. The clipboard keeps the caller's byte slices (X11 and headless): reusing the buffer after copying changes what is pasted

- **Severity:** low
- **File:** internal/platforms/clipboard_linux.go:143 (`clip.items[...] = item.Data`), :175 (returns the internal slice); internal/platforms/clipboard.go:52 (`slices.Clone(items)` is a shallow copy), :68-69
- **Root cause:** only the slice of items is cloned, not the `Data` slices. X11 serves `clip.items` later, from the event loop, so a caller that reuses its buffer after `ClipboardSetData` (for example a `bytes.Buffer` that is reset and reused) changes what other applications paste. `ClipboardGetData` also returns the internal slice, so changing the result changes the clipboard. Windows copies into global memory, so the platforms behave differently.
- **Reproducer** (ui, headless): `data := []byte("first"); ClipboardSetData("application/x-audit", data); copy(data, "XXXXX"); ClipboardGetData(...)`
- **Actual output:** `clipboard aliases the caller's buffer: "XXXXX"`, then after `got[0]='Z'` a second get returns `"ZXXXX"`.
- **Expected:** `"first"`.
- **Fix direction:** `bytes.Clone(item.Data)` in `clipboardSet` (linux) and in the headless `ClipboardSet`; also clone when returning the data the clipboard owns itself.

## 10. Headless clipboard does not treat "text/plain" as the text (differs from the real platforms)

- **Severity:** low
- **File:** internal/platforms/clipboard.go:62-73 (headless `ClipboardGet`, exact MIME match) compared with `isTextMIME` used on Linux
- **Root cause:** on X11, `ClipboardSetData("text/plain", b)` and the `;charset=UTF-8` variants count as the text, because `isTextMIME` is applied. The headless store compares MIME strings exactly, so `ClipboardGetText()` (which asks for `MIMEText`) returns "". Tests that use headless mode see different behaviour from production.
- **Reproducer:** `ClipboardSetData("text/plain", []byte("hello")); ClipboardGetText()`
- **Actual output:** `""`
- **Expected:** `"hello"`, as on X11.
- **Fix direction:** normalize text MIMEs to `MIMEText` (with `isTextMIME`) in the headless Set, Get and Formats.

## 11. Headless: a NaN or +Inf scale passes `max(scale, 1)` and makes `Render` panic

- **Severity:** low (test API)
- **File:** internal/platforms/headless.go:56 (`SetHeadlessScreen`), :471 (`MoveToScreen`)
- **Root cause:** Go's `max(NaN, 1)` is NaN and `max(+Inf, 1)` is +Inf. `px()` then gives `int(NaN)` or `int(Inf)`, which is the minimum int64 on amd64, so `image.NewRGBA` panics.
- **Reproducer:** `w := &HeadlessWindow{width: 10, height: 10, scale: max(math.NaN(), 1)}; w.Render()`
- **Actual output:** `panic: image: NewRGBA Rectangle has huge or negative dimensions` for both NaN and +Inf.
- **Expected:** invalid scales are rejected or treated as 1.
- **Fix direction:** `if !(scale >= 1 && scale <= 16) { scale = 1 }` (this also catches NaN).

## 12. An application icon of 2048x2048 or larger is silently not set on X11

- **Severity:** low
- **File:** internal/platforms/native_window_linux.go:1381-1421 (`SetAppIcon`)
- **Root cause:** `_NET_WM_ICON` is written in one `XChangeProperty` of `2+w*h` 32-bit words. For 2048x2048 that is 4194306 words, which is over the default BIG-REQUESTS maximum (4194303). The server answers BadLength, which is only logged, so the window has no icon. `ui.SetAppIcon` and `Form.SetIcon` do not scale the image down.
- **Reproducer** (`TestAuditBigIcon`): `w := createWindow(...); w.SetAppIcon(image.NewRGBA(image.Rect(0,0,2048,2048))); xSync(...)`
- **Actual output:** `nui: X error 16 (request 1.0) on resource 0x200001`. The icon is not applied.
- **Expected:** the icon is shown.
- **Fix direction:** scale the icon down to at most 256x256 (and if possible also add 16, 32, 48 and 64 px images to `_NET_WM_ICON`), or check the size against `maxPropertySize`.

---

### Checked, no bug found
- `imageFromDIB` (clipboard_dib.go): fuzzed for 60 s with 2.5 M executions, no panic. The size and stride checks keep reads in bounds.
- `parseDesktopScale`: NaN, Inf, 0, negative and huge values for NUI_SCALE and Xft.dpi are all clamped correctly (NaN fails `> 0`).
- filedialog_linux.go: arguments go through `exec.Command` with no shell. zenity values are single `--opt=value` arguments, so there is no command injection.
- `xErrorHandler` is installed, so X protocol errors are not fatal. The Xlib event struct layouts (Selection*, Property, ErrorEvent) match Xlib.h.
- AT-SPI `runeSlice`, `boundary`, `GetCharacterAtOffset` and `GetTextAtOffset` bound their indexes correctly.
- tray_linux: the menu is copied and guarded by a mutex, and IDs are looked up in a map.
