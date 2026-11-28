# Audit: text editing and text layout (nui/ui)

Area: TextBox, TextLayout / shaping, DrawText / glyph rasterizing, NumBox, EditableComboBox, text utils.
Host: Linux, go 1.27, headless tests (`newTestForm`).

All reproducers are in `doc/audit/_repro/text/zz_audit_text_test.go` (tests `TestAuditF1…F14`). To run one, copy it into `ui/` and run
`go test ./ui -run TestAuditF -count=1 -v`. Each test logs what it shows; panics are caught and logged by `catch()`.
The other files in `doc/audit/_repro/text/` are the fuzzers I used: `zz_audit_text_fuzz_test.go` (TextLayout / caret API),
`zz_audit_text_box_test.go` (random TextBox operations with invariant checks), `zz_audit_text_glyphs_test.go` (every rune
0x20..0x2FFF through DrawText / DrawTextF / Path.Text at sizes 0.3..61, hinting on), `zz_audit_text_race_test.go` (concurrent
MeasureText / DrawText / TextLayout under `-race`). After the issues below were filtered out, they found nothing else:
40000 TextLayout cases, 4000×80 TextBox operations, all glyphs, and no data race.

Helpers the snippets use (they are defined in the reproducer file or in existing tests):

```go
func catch(f func()) (msg string) {
	defer func() { if r := recover(); r != nil { msg = fmt.Sprint("PANIC: ", r) } }()
	f(); return "ok"
}
func paintForm(form *Form) { form.paintScaled(image.NewRGBA(image.Rect(0, 0, 500, 400)), 1) }
// typeKey / typeText: existing helpers in ui/treeview_test.go and ui/textbox_undo_test.go
```

---

## F1. A paragraph-separator character in shaped text panics in the layout: MeasureText, DrawText, NewLabel, TextBox

- **Severity:** critical. The panic is easy to hit: a label or log line that ends in "\r" (Windows line endings split on "\n") and has an emoji, Hebrew/Arabic or a combining accent crashes the program. Typing or pasting U+2029 into a TextBox crashes it too.
- **Where:** `ui/canvas_text_layout.go:255-266` (`layout()` splits paragraphs only at `'\n'`), `ui/canvas_text_layout.go:395` (`wrapper.WrapParagraphF`). The panic is raised inside go-text: `shaping.inclusiveGlyphRange` ← `cutRun` ← `processBreakOption`.
- **Root cause:** `TextLayout.layout()` cuts the text into paragraphs only at `'\n'`. The characters of Unicode bidi class B (paragraph separators), `\r`, U+001C..U+001E, U+0085 and U+2029, stay inside a paragraph. The go-text line wrapper takes them as mandatory breaks, and nui's runs, which are cut by face and by bidi level, then hand it an empty run, so `cutRun` indexes `glyphs[0]` of a run that has no glyphs. Without a width the panic needs shaped text (`needsShaping` sends the line to `shapedLine` → `TextLayout`). With a wrap width, plain ASCII is enough (`"a\r"`).
- **Reproducer:**
  ```go
  for _, s := range []string{"😀 done\r", "שלום\rעולם", "é x", "😀\u0085x"} {
      t.Logf("MeasureText(%q): %s", s, catch(func() { MeasureText(FontFamilySans, 12, s) }))
  }
  for _, s := range []string{"a\r", "a\x1cb", "a ", "a\r\n"} {
      t.Logf("%q: %s", s, catch(func() { l := NewTextLayout(s, TextStyle{Size: 12}); l.SetWidth(1); l.Size() }))
  }
  t.Logf("%s", catch(func() { NewLabel("😀 done\r") }))
  t.Logf("%s", catch(func() { box := NewTextBox(); newTestForm(t, box); box.SetText("é x") }))
  t.Logf("%s", catch(func() {       // the user types é, U+2029, x
      box := NewTextBox(); form := newTestForm(t, box); box.Focus()
      for _, r := range "é x" { form.processChar(r) }
  }))
  ```
  Output: every call gives `PANIC: runtime error: index out of range [0] with length 0`. Stack head:
  ```
  github.com/go-text/typesetting/shaping.inclusiveGlyphRange(...)        wrapping.go:213
  github.com/go-text/typesetting/shaping.cutRun(...)                      wrapping.go:241
  github.com/go-text/typesetting/shaping.(*LineWrapper).processBreakOption wrapping.go:1228
  ...WrapParagraphF
  github.com/ipoluianov/nui/ui.(*TextLayout).layoutParagraph              canvas_text_layout.go:395
  github.com/ipoluianov/nui/ui.(*TextLayout).layout                       canvas_text_layout.go:264
  github.com/ipoluianov/nui/ui.shapedLine / shapedWidth / measureText / MeasureText
  github.com/ipoluianov/nui/ui.measureMultilineTextWidth / (*Label).updateInnerSize / NewLabel
  ```
- **Expected:** no panic. `\r` is drawn as nothing (or a "\r\n" pair as one break, as `Canvas.DrawText` promises). U+2029 / U+0085 / U+001C-1E act as line breaks or are ignored.
- **Fix direction:** in `TextLayout.layout()`, also end a paragraph at every bidi class-B rune (`\r`, `\x1c`-`\x1e`, `\u0085`, ` `; treat `\r\n` as one break), so they never reach `WrapParagraphF` inside a paragraph. Another way is to map them to `'\n'` or drop them in `layoutParagraph` before shaping. Add a regression test with the strings above, with and without `SetWidth`. Also consider wrapping the go-text calls in `layoutParagraph` with a recover that falls back to an unshaped line, so a future go-text edge case can't crash an app.

---

## F2. TextLayout.SetMaxLines on a text with several paragraphs panics (levelAt index out of range)

- **Severity:** high. Public API, documented use, a 5-character input.
- **Where:** `ui/canvas_text_layout.go:345-351` (`levelAt`), called at `ui/canvas_text_layout.go:400`.
- **Root cause:** when `maxLines > 0` and the paragraph is followed by more text (`TextContinues = broken`), go-text adds the truncator run ("…"). Its `Runes.Offset` is `len(para)`, one past the last rune. `levelAt(out.Runes.Offset)` reads `levels[r]` without a bounds check.
- **Reproducer:**
  ```go
  l := NewTextLayout("a b\nc", TextStyle{Size: 12})
  l.SetMaxLines(1)
  l.Size()
  ```
  Output: `PANIC: runtime error: index out of range [3] with length 3` (in `layoutParagraph.func1`, i.e. `levelAt`).
- **Expected:** one line "a b…".
- **Fix direction:** in `levelAt`, clamp `r` to `len(levels)-1`, or give a run with `Runes.Offset >= len(levels)` (the truncator) the paragraph's base level (0, or 1 for RTL). `runOf` already returns -1 for it, so its style falls back to `firstStyle`, which is fine.

---

## F3. The cache of shaped lines has no memory budget: editing a long line with one emoji eats GBs

- **Severity:** high (memory DoS from ordinary use).
- **Where:** `ui/text_complex.go:77-96` (`shapedLine`, the cache is limited to `2048` entries only).
- **Root cause:** every distinct text that needs shaping (any emoji, RTL, combining mark) is laid out and its whole `TextLayout` (glyphs, clusters, runes, offsets) is cached under the full text as key. The only limit is the number of entries (2048), not their size. In a TextBox, every keystroke in a line changes the line, so `updateInnerSize`/`caretX` add a new entry of the whole line. For a 20k-rune line one entry is about 2.7 MB, so the cache can reach about 5.5 GB before it is cleared. LogView lines and `truncateTextToWidth` probes (binary search, each prefix a new key) feed the same cache.
- **Reproducer:**
  ```go
  clearShapedLines(); runtime.GC(); var m0, m1 runtime.MemStats; runtime.ReadMemStats(&m0)
  box := NewTextBox(); form := newTestForm(t, box)
  box.SetText("😀" + strings.Repeat("x", 20000)); box.Focus(); box.MoveCursorToEnd()
  for i := 0; i < 200; i++ { form.processChar('y') }
  runtime.GC(); runtime.ReadMemStats(&m1)
  ```
  Output: `200 chars typed at the end of a 20001-rune line with one emoji: 3.2s, live heap +543 MB, shapedLines entries 201 (limit 2048)`. Measuring alone gives the same: 300 versions of the line → +809 MB.
- **Expected:** bounded memory, on the order of tens of MB.
- **Fix direction:** keep a byte budget (sum of `len(text)` × a per-rune estimate, or the real slice sizes) like `colorImagesBudget` in `text_color.go`, with LRU eviction. Don't cache very long lines, or cache only the last N versions per widget. The TextBox could keep its own per-line layout and drop the old one when the line changes.

---

## F4. A password TextBox copies and cuts its plaintext to the clipboard

- **Severity:** high (security: the secret leaks to the system clipboard, where clipboard managers and other apps can read it).
- **Where:** `ui/textbox.go:568-596` (`cutSelected`, `copySelected`). The same applies to the public `Copy()` / `Cut()` (`ui/textbox.go:1410-1418`).
- **Root cause:** neither function checks the `ispassword` property. The masking is only visual (`lineToPasswordChars`).
- **Reproducer:**
  ```go
  box := NewTextBox(); form := newTestForm(t, box)
  box.SetIsPassword(true); box.SetText("s3cret"); box.Focus(); box.SelectAllText()
  typeKey(form, KeyC, 0, KeyModifiers{Ctrl: true})
  got, _ := ClipboardGetText()
  ```
  Output: `clipboard after Ctrl+A, Ctrl+C in a password box: "s3cret"`. Ctrl+X also puts `"s3cret"` on the clipboard and empties the box.
- **Expected:** as in Qt, GTK and browsers: in a password box Copy and Cut do nothing (Cut may still delete). Paste stays allowed.
- **Fix direction:** in `copySelected`/`cutSelected`, return early (cut: delete without `ClipboardSetText`) when `GetPropBool("ispassword", false)`. Also check that a drag of the selection, if added later, follows the same rule.

---

## F5. Canvas.DrawText at a large font size rasterizes and caches whole-glyph masks: hundreds of MB, then OOM

- **Severity:** medium. Large sizes are reachable with a zoomed canvas, charts or a user-set font size. ≥1e5 takes seconds and GBs, 1e6 kills the process (fatal OOM, which can't be recovered), and 1e9 panics.
- **Where:** `ui/text_glyph.go:198-268` (`rasterGlyph`: `cover := make([]uint8, r.Dx()*r.Dy())`), reached from `drawTextGlyphs` (`ui/text_glyph.go:389-437`) through `sfntGlyphMask`. The `renderText` path (no hinting) behaves the same way.
- **Root cause:** the mask of the glyph is made at full size regardless of the clip and the destination. `DrawTextLayout` switches to filling outlines above `maxMaskSize = 160` px (`ui/canvas_text_draw.go:109,146`), but `DrawText`'s own path has no such limit. The masks are then kept in `glyphMasks`, which is limited by count (16384) and not by bytes.
- **Reproducer:**
  ```go
  cnv := NewCanvasScaled(image.NewRGBA(image.Rect(0, 0, 100, 50)), 1)
  cnv.SetFontSize(16000)
  cnv.DrawText(0, 0, 100, 50, "ab")
  ```
  Output: size 4000 → 22 ms, 11 MB; size 16000 → 373 ms, 155 MB; size 1e5 → 21 s, 5.8 GB allocated; size 1e6 → `fatal error: runtime: out of memory` in `rasterGlyph` (text_glyph.go:268); size 1e9 → `PANIC: runtime error: makeslice: len out of range`.
- **Expected:** drawing cost and memory bounded by the visible area. No crash.
- **Fix direction:** in `drawText`/`drawTextGlyphs`, as `DrawTextLayout` does: above `maxMaskSize` (in pixels) fill the glyph outlines through the canvas, which is clipped, instead of making masks. Also clamp absurd sizes (NaN, Inf, > some maximum) in `SetFontSize`/`withFace`, and give `glyphMasks` a byte budget.

---

## F6. NumBox: stepping a large value gives +Inf (rounding overflow); SetValue(+Inf) is accepted

- **Severity:** medium (wrong value; the box then shows "+Inf" and the bound value becomes Inf).
- **Where:** `ui/numbox.go:842-848` (`stepBy`: `p := math.Pow10(c.decimals); v = math.Round(v*p) / p`), and `ui/numbox.go:250` (`SetValue` rejects only NaN).
- **Root cause:** `v*p` overflows to +Inf when |v| > MaxFloat64/10^decimals (1e307 with 2 decimals, 1e293 with 16), and `Inf/p` stays Inf. The clamp runs before the rounding, and with the default max (+Inf) the second clamp in `SetValue` also lets it through.
- **Reproducer:**
  ```go
  n := NewNumBox(); form := newTestForm(t, n)
  n.SetDecimals(2); n.SetValue(1e307); n.Focus()
  typeKey(form, KeyArrowUp, 0, KeyModifiers{})
  // n.Value(), n.Text()
  n.SetValue(math.Inf(1))
  ```
  Output: `decimals 2, value 1e307, Up: value=+Inf text="+Inf"`; `SetValue(+Inf): value=+Inf text="+Inf"`.
- **Expected:** the value stays finite (1e307, or the step is lost to precision). An infinite value is rejected like NaN (`parseText` already rejects Inf from the user).
- **Fix direction:** round only when `math.Abs(v*p)` is finite (`if r := math.Round(v*p) / p; !math.IsInf(r, 0) { v = r }`), and clamp after rounding. In `SetValue` (and the `"value"` prop path) ignore ±Inf as well as NaN.

---

## F7. TextBox: a click on certain pixels is ignored, so the cursor and the drag anchor stay where they were

- **Severity:** medium (common: one pixel at every character whose neighbor is an odd number of pixels wide. Clicking there does nothing, and a drag starting there selects from the old cursor, which may be on another line).
- **Where:** `ui/textbox.go:815-842` (`moveCursorNearPoint`, the non-shaped branch).
- **Root cause:** the hit zones are `[cp[pos] - (cp[pos]-cp[pos-1])/2, cp[pos] + (cp[pos+1]-cp[pos])/2)`. With integer division, the right edge of zone `pos-1` is `cp[pos-1] + d/2` and the left edge of zone `pos` is `cp[pos] - d/2`. They differ by `d % 2`, so when a character's width `d` is odd, one x belongs to no zone. No branch then calls `moveCursor`, and `MouseDown` copies the old cursor into the selection anchors.
- **Reproducer:**
  ```go
  box := NewTextBox(); newTestForm(t, box)
  box.SetText("line 52"); box.Focus()
  lx := box.lineX(0)
  box.SetCursorPosition(TextPos{Col: 7})
  x := lx + 12 // between "i" and "n": "n" is 9 px wide
  box.MouseDown(MouseButtonLeft, x, 5, KeyModifiers{}); box.MouseUp(MouseButtonLeft, x, 5, KeyModifiers{})
  ```
  Output: `caret xs [0 4 8 17 25 29 37 45] (+lineX 8); click at x=20: cursor {Line:0 Col:7}, nearest caret 2`. In a 100-line multiline box scrolled to line 52 (cursor on line 60), a click on that x of line 52 left the cursor on line 60.
- **Expected:** the cursor goes to the nearest caret (2).
- **Fix direction:** replace the hand-written zone search with `TextIndexAt(family, size, line, x-c.lineX(lineNumber))`, which picks the nearest caret and is already used for shaped lines. Or make the zones contiguous (`left = previous right`). `NumBox.cursorPosFromPixel` (`ui/numbox.go:569-598`) has the same formula and falls back to `len(runes)` (end of text) in the gap. With the built-in font's digit widths (8 px, "." 4 px) I found no gap there, but it has the same defect with other fonts or characters.

---

## F8. A single-line TextBox keeps the line breaks of SetText: hidden lines the user can move into and edit

- **Severity:** medium (data the user can't see is edited. Arrow Down and clicks in the lower half of the box go to the invisible line).
- **Where:** `ui/textbox.go:1095` (`modifyText`, case SetText: `splitLines(data.(string))` without checking `Multiline()`). The same comes through `EditableComboBox.SetText` (`ui/combobox_editable.go:87-92`) and `SetProp("text", …)`. Typing and `InsertText` drop `'\n'` in a single-line box (`ui/textbox.go:1112-1128`), so the behaviour is inconsistent.
- **Reproducer:**
  ```go
  box := NewTextBox(); form := newTestForm(t, box)   // single-line (default)
  box.SetText("visible\nhidden"); box.Focus()
  typeKey(form, KeyArrowDown, 0, KeyModifiers{})
  typeText(form, "X")
  ```
  Output: `single-line box: lines=2, cursor {Line:1 Col:1}, text "visible\nXhidden"`. Only "visible" is drawn: line 1 is painted below the box (y = yStaticOffset + 1·lineHeight), and the caret disappears.
- **Expected:** a single-line box holds one line. SetText either joins the lines (e.g. replacing "\n" with a space, as Qt's QLineEdit does) or drops the breaks, as typing does.
- **Fix direction:** in the SetText branch, when `!c.Multiline()`, normalise the text to one line before `splitLines`. Also, when `SetMultiline(false)` is called on a box with several lines, join them.

---

## F9. An edit that changes nothing records an undo step and clears the redo history

- **Severity:** low (lost Redo; a "dead" Ctrl+Z that does nothing; `CanUndo()` true with nothing to undo).
- **Where:** `ui/textbox.go:1038-1039` (`modifyText` calls `recordUndo(cmd)` first), `ui/textbox.go:1262-1268` (`recordUndo` pushes a state and sets `c.redoStack = nil` unconditionally).
- **Root cause:** the state is recorded before it is known whether the command changes the text. Backspace at (0,0), Delete at the end, `InsertText("")`, a paste of only control characters, or Enter in a read-only single-line box all push a step and drop Redo.
- **Reproducer:**
  ```go
  box := NewTextBox(); form := newTestForm(t, box)
  box.SetText("abc"); box.Focus(); box.MoveCursorToEnd()
  typeText(form, "d"); typeKey(form, KeyZ, 0, KeyModifiers{Ctrl: true})   // CanRedo = true
  typeKey(form, KeyHome, 0, KeyModifiers{}); typeKey(form, KeyBackspace, 0, KeyModifiers{})
  box.InsertText(""); box.InsertText("\x01")
  ```
  Output: `after typing d and Ctrl+Z: "abc" CanRedo=true` → `after Backspace at the start (nothing deleted): "abc" CanRedo=false undo steps=1` → `after InsertText("") and InsertText("\x01"): undo steps=3`.
- **Expected:** a no-op edit leaves both stacks unchanged.
- **Fix direction:** in `modifyText`, take the snapshot first, and push it (and clear redo) only `if changed`. Or pop the pushed state when `!changed` and restore the old redo stack.

---

## F10. Editing a line that has invalid UTF-8 silently rewrites its other bytes to U+FFFD

- **Severity:** low (data corruption of bytes the user did not touch: text loaded from a file in another encoding is changed by one keystroke elsewhere in the line).
- **Where:** `ui/textbox.go:1050` and the other `string(runes[...])` rebuilds in `modifyText` (lines ~1054, 1064, 1081, 1131) and `deleteSelection` (`ui/textbox.go:1011-1014`).
- **Root cause:** `[]rune(line)` turns every invalid byte into U+FFFD, and `string(runes)` writes it back as `EF BF BD`. Untouched lines keep their bytes, so `Text()` changes in a way that depends on which lines were edited.
- **Reproducer:**
  ```go
  box := NewTextBox(); form := newTestForm(t, box)
  box.SetText("a\xffb"); box.Focus(); box.MoveCursorToEnd()
  typeText(form, "Z")
  ```
  Output: `SetText("a\xffb"), typed Z at the end: Text()="a�bZ"` (the byte 0xFF became 3 bytes EF BF BD).
- **Expected:** either the box sanitises the text once in `SetText` (`strings.ToValidUTF8`, so the replacement is visible and consistent), or an edit splices bytes at rune offsets so the other bytes are kept.
- **Fix direction:** simplest: in `SetText`/`InsertText`, `strings.ToValidUTF8(text, "�")`, so every line is valid UTF-8 from the start and the rune arithmetic is exact. Or splice by byte offsets (`runeOffset`) instead of `[]rune` round-trips.

---

## F11. TextSelectionSpans (exported) panics when `from` is past the end of the text

- **Severity:** low (public helper; TextBox itself always passes valid ranges).
- **Where:** `ui/text_complex.go:202` (`from, to = max(0, from), min(to, len(positions)-1)`: `from` is not limited above).
- **Reproducer:** `TextSelectionSpans(FontFamilySans, 12, "ab", 5, 9)`
  Output: `PANIC: runtime error: index out of range [5] with length 3`.
- **Expected:** `nil` (an empty selection).
- **Fix direction:** `from = max(0, min(from, len(positions)-1))`, then `if from >= to { return nil }` after the clamping. `TextCaretX` already clamps this way.

---

## F12. TextLayout.IndexAt (a click in a shaped line) is quadratic in the line length

- **Severity:** medium (UI freeze: a click or a mouse drag on a long line that has one emoji, RTL letter or accent costs O(n²)).
- **Where:** `ui/canvas_text_layout.go:650-655` (`for r := ln.start; r <= ln.end; r++ { ln.caretX(r) }`). `caretX` (`ui/canvas_text_layout.go:595-620`) scans all clusters linearly. The same pattern exists in `TextBox.visualStep` (`ui/textbox.go:1564-1585`: `caretX` → `TextCaretX` → `CaretRect` for every grapheme on every arrow key when visual movement is on).
- **Reproducer:**
  ```go
  line := "😀" + strings.Repeat("a", n)
  TextCaretX(FontFamilySans, 12, line, 0)       // lay out once (cached)
  t0 := time.Now(); TextIndexAt(FontFamilySans, 12, line, 100)
  ```
  Output: 4001 runes 6 ms, 8001 → 27 ms, 16001 → 131 ms (×4 per doubling; 100k runes ≈ 5 s per click). `MouseMove` while dragging calls it for every event. One click in a TextBox on a 10001-rune line with an emoji took 52 ms.
- **Expected:** O(n) or O(log n) per hit test.
- **Fix direction:** in `IndexAt`, iterate the clusters of the line once, computing each cluster's caret edges (and inner positions for multi-rune clusters) directly, and keep the nearest. Or binary-search the clusters by x for LTR lines. In `visualStep`, compute all caret x once per call (one pass over the clusters), not per grapheme.

---

## F13. MakeLinesFromStringWithWordWrapping (exported) is quadratic: `cur += string(r)` per rune

- **Severity:** low (public helper, not used inside nui; a 100 KB word takes 1.4 s and 200 KB takes 17 s).
- **Where:** `ui/text_utils.go:52-79` (`splitOnNewlines` and `fields` build strings with `cur += string(r)`).
- **Reproducer:** `MakeLinesFromStringWithWordWrapping(strings.Repeat("x", 200000), 80)`
  Output: 25k chars 205 ms, 50k → 470 ms, 100k → 1.39 s, 200k → 17.0 s.
- **Expected:** linear time.
- **Fix direction:** use `strings.Split(text, "\n")` and `strings.Fields` (or a `strings.Builder` / slicing by byte index).

---

## F14. EditableComboBox: the highlighted match in the suggestion list is cut at wrong bytes when lower-casing shifts byte lengths

- **Severity:** low (wrong drawing: invalid UTF-8 fragments are drawn as the match and the rest of the text is garbled. No panic, because the total length is equal).
- **Where:** `ui/combobox_editable.go:436-447` (`completionPopup.draw`).
- **Root cause:** `start` is a byte index into `strings.ToLower(text)` but is used to slice `text`. The guard `len(strings.ToLower(text)) != len(text)` only catches a change of the total length. One character that grows (`Ⱥ` U+023A, 2 bytes → `ⱥ` U+2C65, 3 bytes) and one that shrinks (`ẞ` U+1E9E, 3 bytes → `ß`, 2 bytes) keep the total length but shift the offsets.
- **Reproducer** (the same computation as `draw`):
  ```go
  text, needle := "Ⱥẞabc", "ßab"
  start := strings.Index(strings.ToLower(text), needle)
  part := text[start : start+len(needle)]
  ```
  Output: `len(text)=8 len(lower)=8: the part drawn as the match of "ßab" is "\xba\x9eab"`. In the widget: `SetItems([]string{"Ⱥẞabc"})` then type "ßab", and the popup draws those three parts.
- **Expected:** the match "ẞab" is highlighted, or no highlight.
- **Fix direction:** find the match per rune: lower-case rune by rune while keeping a map from lower-case byte offsets to original byte offsets, or compare with `strings.EqualFold` on rune windows. Otherwise, when any rune changes its UTF-8 length under `unicode.ToLower`, skip the highlight (draw the plain truncated text).

---

### Checked and found OK (not bugs)
- TextBox: 4000 random sequences of 80 operations (SetText, InsertText, typing combining, ZWJ, RTL and invalid-UTF-8 text, all editing keys with Shift/Ctrl, mouse, SetSelection/SelectRange with out-of-range positions, password/multiline toggles, alignment, visual arrows, line numbers, a highlighter with out-of-range spans, painting). No panic. The cursor and selection always stay within the text. `Text()` always equals `Lines()` joined. `OffsetOf(PosAt(o)) == o` for all valid offsets.
- TextLayout / TextCaretX / TextIndexAt / graphemeBounds / truncateTextToWidth: 20000 random texts (without the F1/F2 triggers). No panic.
- NaN / ±Inf / negative / 1e300 coordinates, font sizes, widths, letter spacing and angles in DrawTextLayout, DrawTextF, DrawTextRotated, Path.Text, MeasureText: no panic or hang (except F5 for sizes in the 1e4..1e9 range).
- Every rune 0x20-0x2FFF in both built-in families, bold/italic, sizes 0.3-61, with hinting: no panic.
- Concurrent MeasureText / DrawText / TextLayout / Path.Text under `-race`: no race.
