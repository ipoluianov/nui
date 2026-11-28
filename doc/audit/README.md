# Bug audit of nui, 2026-11-27

An audit of the whole library at commit `4342672`, for the agent (or developer) who fixes the bugs.
Each finding was reproduced by a test, unless it is marked "verified by reading". The details of each
finding are in the file of its area: root cause, a minimal reproducer, the actual output, the expected
behavior, and a direction for the fix.

| Area | File | Findings |
|---|---|---|
| Text editing, text layout, NumBox, EditableComboBox | [text.md](text.md) | 14 |
| Table, ListView, TreeView, ComboBox, PropertyGrid | [views.md](views.md) | 10 |
| 2D canvas, paths, effects, SVG recorder, internal/canvas | [canvas.md](canvas.md) | 15 |
| Charts and TimeChart | [charts.md](charts.md) | 17 |
| Parsers, settings, single instance, dates, file dialog, PDF | [parsing.md](parsing.md) | 17 |
| Core: layout, events, forms, popups, menus, docks | [core.md](core.md) | 15 |
| Native layer: X11 clipboard, drag and drop, AT-SPI, Windows COM | [platform.md](platform.md) | 12 |
| Other widgets, 3D, View2D, ZoomView, fonts | [misc.md](misc.md) | 13 |

## Running the reproducers

The reproducers are in [_repro/](_repro). They are internal tests (`package ui`, `package platforms`, and so on).
Go ignores directories whose names start with `_`, so they are not built and not run by `go test ./...`.
To run the reproducers of one area:

```sh
cp doc/audit/_repro/views/zz_*_test.go ui/
go test ./ui -run 'TestAudit' -count=1 -v -timeout 120s
rm ui/zz_audit_*_test.go
```

- **Copy the files of one area at a time.** Areas define helpers with the same names (`auditPaint`, ...), so the files of two areas do not compile together. Each area compiles on its own (checked).
- **Where each file goes:** the md file of the area says so. For canvas, `_repro/canvas/ui/` goes to `ui/` and `_repro/canvas/internal/canvas/` goes to `internal/canvas/`. For parsing, `regression_zz_audit_parsing_test.go` goes to `ui/regression/zz_audit_parsing_test.go`. For platform, `ui_zz_audit_platform_test.go` goes to `ui/zz_audit_platform_test.go` and the rest to `internal/platforms/`. The X11 tests need Xvfb (see platform.md).
- **Run hang and OOM tests one at a time, under a memory cap:** `go test -c -o /tmp/ui.test ./ui && (ulimit -v 3000000; /tmp/ui.test -test.run '^TestAuditX$' -test.v)`. Some reproducers allocate until the machine runs out of memory (CHART-1, PLT-1, TXT-5), and some never return (CORE-1, CORE-3, MISC-1, CNV-3, CNV-5, CHART-2, CHART-10, CHART-11).
- Several tests do not fail. They print what they show (`t.Logf`) and catch panics. When a fix is made, turn the reproducer into a real regression test with an assertion: in `ui/regression` for behavior an application sees, or next to the code (see [doc/testing.md](../testing.md)).

## How to fix

- Fix by **root cause**, not by symptom. Several findings share one cause (see below): one fix in the canvas or the text layout closes findings in several areas.
- Each fix comes with a regression test, and `go test ./...` must pass. The CI runs on Linux, macOS and Windows.
- Follow [doc/rules.md](../rules.md). Do not change the public API without need. When a function can't return an error, invalid input (NaN, Inf, negative sizes) is ignored or clamped.
- Once a finding is fixed, mark it here (e.g. ~~TXT-1~~ ✅) or remove it.

## Status

2026-11-28: all critical and high findings are fixed (marked ✅), and CORE-4 with them (the same pattern as CORE-5). Each fix has a regression test that fails on the code before it:

| Findings | Tests |
|---|---|
| TXT-1..4 | `TestTextParagraphSeparators`, `TestTextLayoutMaxLinesSeveralParagraphs` (ui/regression/text_layout_test.go), `TestShapedLinesCacheMemoryBudget` (ui/text_complex_test.go), `TestTextBoxPasswordNotCopied` (ui/regression/textbox_test.go) |
| CNV-1..3, CHART-5 (canvas side) | ui/canvas_path_robust_test.go |
| CHART-1, 2, 3, 5 | ui/chart_robust_test.go |
| CORE-1, 4, 5, 7 | `TestStretchedColumnAtMaxSizeEndsLayout` (ui/regression/layout_test.go), `TestKeyDownHandlerRemovesFocusedWidget`, `TestMouseMoveHandlerOpensModal`, `TestMouseMoveHandlerRemovesPressedWidget` (ui/regression/events_test.go), `TestShowModalTwice`, `TestShowMaximizedTwice` (ui/regression/dialogs_test.go) |
| VIEW-1, VIEW-2, MISC-1 | `TestTableImageWithoutRoom` (ui/regression/table_test.go), `TestTreeViewLazyLoadSelectsChild` (ui/treeview_test.go), `TestView2DGridFarFromOrigin` (ui/view2d_test.go) |
| PRS-2, PRS-3 | `TestInstanceCloseWhileAccepting` (ui/instance_test.go), ui/regression/calendar_test.go, `TestCalendarGridWithoutMidnight`, `TestCalendarSelectionDrawnAfterSkippedMidnight` (ui/controls_test.go) |
| PLT-1, PLT-2 | `TestClipboardImageHugeHeader` (ui/regression/clipboard_test.go), `TestDragOutEndsWhenWindowCloses` (internal/platforms/dragout_linux_test.go; opt-in: `NUI_TEST_X11=1 DISPLAY=<Xvfb>`, it grabs the pointer) |

Notes on the fixes:
- PRS-3: `DateOnly`, `Calendar.Date` and `DatePicker.Date` return midnight as before; on a day whose midnight is skipped (Chile in September) they return the first instant of the day (01:00). The audit reproducers `TestAuditCalendarDST*` copy the old draw loop into the test, so they still "fail"; the library is fixed.
- PLT-1: images on the clipboard are limited to 32768 px a side and 64 Mpx (`platforms.ErrClipboardImageTooLarge`).
- TXT-1: in a multi-line `TextLayout` the paragraph separators break lines; in one-line text (Label, TextBox, DrawText) they are drawn as nothing.
- Still open from the CHART-5 area: CHART-6 (the y autoscale with Inf/NaN).

## Root causes shared by several findings

1. **Bidi paragraph separators in a shaped paragraph** (`\r`, U+001C-U+001E, U+0085, U+2029). TXT-1 = PRS-1 (the same bug, found twice).
2. **Non-finite or overflowing coordinates in the rasterizer.** CNV-2 is the root. CHART-5 (series with Inf), CNV-4, CNV-7 and CNV-8 are other doors to it. The canvas must survive any float; the charts should also filter non-finite values themselves.
3. **`DrawText` at coordinates near ±2^63** (`int(NaN)`) overflows in `blendGlyphMask` (text_glyph.go:316). CHART-9 (span labels) and MISC-2 (3D labels). A guard in `DrawText`/`drawText` closes both; the callers should skip non-finite positions too.
4. **Go `min`/`max`/`math.Max` let NaN through**, so a clamp of the form `max(lo, min(hi, v))` does nothing: PLT-7 = part of MISC-6 (Slider NaN through accessibility), MISC-5, CHART-15, CNV-4, PLT-11.
5. **Float loop counters and epsilon loops** that never end at large magnitudes: CNV-3, CHART-1, CHART-10, CHART-11, MISC-1.
6. **Size products that overflow `int` and pass a guard**: CNV-4 (`mw*mh`), CNV-8 (`pw*ph`).
7. **Handlers that change the widget tree during event dispatch** (a field read again after the handler ran): CORE-4, CORE-5, CORE-6, VIEW-2, VIEW-3.
8. **`OpenURL` does not wait for the child process**: PRS-17 = MISC-9.

## All findings

An ID is the area and the number of the section in the area's file: TXT-3 is "F3" of text.md, VIEW-3 is "3." of views.md, and so on (TXT text.md, VIEW views.md, CNV canvas.md, CHART charts.md, PRS parsing.md, CORE core.md, PLT platform.md, MISC misc.md).

Severity: critical, high, medium, low, as the area's file gives it. The suggested order of the fixes: critical and high first, then the medium ones that crash or hang, then the rest.

### Critical and high

| ID | Sev. | Where | What |
|---|---|---|---|
| ~~TXT-1 (= PRS-1)~~ ✅ | critical | ui/canvas_text_layout.go:255-266, 395 | `\r`/U+2029/U+0085 together with emoji, RTL or a combining mark (or with a wrap width, even plain ASCII) panic in go-text: MeasureText, Label, DrawText, TextBox, typing |
| ~~TXT-2~~ ✅ | high | ui/canvas_text_layout.go:345-351, 400 | `TextLayout.SetMaxLines` on a text of several paragraphs panics (`levelAt` index out of range) |
| ~~TXT-3~~ ✅ | high | ui/text_complex.go:77-96 | The cache of shaped lines has no byte budget: editing a long line with an emoji takes GBs |
| ~~TXT-4~~ ✅ | high | ui/textbox.go:568-596, 1410-1418 | A password TextBox copies and cuts the password to the clipboard in plain text |
| ~~VIEW-1~~ ✅ | high | ui/table.go:1736-1749, 1888-1901, 2497 | Table paint panics when the image height is negative (small row height or large padding) |
| ~~VIEW-2~~ ✅ | high | ui/treeview.go:1783 | Lazy loading in `OnExpanding` that selects a child recurses forever (stack overflow) |
| ~~CNV-1~~ ✅ | high | ui/canvas_path.go:1427, 1321 | A dashed stroke disappears entirely when a dash ends exactly at a vertex, and so does a zero-length dash |
| ~~CNV-2~~ ✅ | high | ui/canvas_path.go:1169, ui/canvas_path_fill.go:75 | Inf coordinates panic the rasterizer (divide by zero / index out of range) |
| ~~CNV-3~~ ✅ | high | ui/canvas_path.go:316-340 | `Path.Bounds()` loops forever on an arc with an Inf or huge start angle |
| ~~CHART-1~~ ✅ | high | ui/chart_polar.go:33-39 | Radar/polar: millions of rings for small values; endless loop and OOM for Inf/MaxFloat64 |
| ~~CHART-2~~ ✅ | high | ui/chart_series_line.go:404-413 | A dense line freezes the UI when its neighbor sample is far outside the view |
| ~~CHART-3~~ ✅ | high | ui/chart_series_heatmap.go:419-440 | Histogram2D/hexbin: one NaN or Inf point panics `SetPoints` |
| ~~CHART-5~~ ✅ | high | chart_series_line.go:600, chart_series_stats.go:690 and others | Inf or overflowing values panic the paint of fills, bands, funnels, digital series, stacked percents (root CNV-2) |
| ~~PRS-2~~ ✅ | high | ui/instance.go:86, 116 | `SingleInstance.Close` races with the accept goroutine, which panics (nil listener); data race |
| ~~PRS-3~~ ✅ | high | ui/calendar.go:54, 451 and others | Calendar/DatePicker break where DST starts at midnight (Chile, Egypt, Cuba, Lebanon): days repeated, arrows stuck |
| ~~CORE-1~~ ✅ | high | ui/widget.go:2755-2767, 3000-3012 | Grid layout loops forever when a column/row with a larger stretch reaches its max size |
| ~~CORE-5~~ ✅ | high | ui/form.go:1281-1284 | A drag handler that opens a message box or removes the pressed widget: nil dereference in `processMouseMove` |
| ~~CORE-7~~ ✅ | high | ui/form.go:813-830, 858-865, 696 | `ShowModal`/`ShowMaximized` on an open form create a second window; the parent stays blocked for good |
| ~~PLT-1~~ ✅ | high | ui/clipboard.go:96 | Pasting an image: a PNG of 89 bytes that claims 1000000x1000000 kills the process (fatal OOM) |
| ~~PLT-2~~ ✅ | high | internal/platforms/dragout_linux.go:76-80, 113 | Drag-out never returns if its window is closed during the drag: the UI thread hangs |
| ~~MISC-1~~ ✅ | high | ui/view2d.go:693-717 | View2D paint hangs forever far from the origin (a float loop counter stuck at 2^53) |

### Medium

| ID | Where | What |
|---|---|---|
| TXT-5 | ui/text_glyph.go:198-268 | `DrawText` at a large font size makes whole-glyph masks: hundreds of MB, then a fatal OOM; 1e9 panics |
| TXT-6 | ui/numbox.go:842-848, 250 | NumBox: stepping a large value gives +Inf; `SetValue(+Inf)` is accepted |
| TXT-7 | ui/textbox.go:815-842 | TextBox: a click on certain pixels (between characters of odd width) is ignored |
| TXT-8 | ui/textbox.go:1095 | A single-line TextBox keeps the line breaks of `SetText`: hidden lines the user can edit |
| TXT-12 | ui/canvas_text_layout.go:650-655, ui/textbox.go:1564-1585 | `TextLayout.IndexAt` is quadratic in the line length: a click on a long shaped line freezes |
| VIEW-3 | ui/treeview.go:1660-1676, 994 | Handlers re-entered during `TreeNode.Remove`: slice panic, or a removed node becomes current |
| VIEW-4 | ui/listview.go:105, 260 | ListView: the Shift anchor is not clamped when the model shrinks; rows past the end get selected |
| VIEW-5 | ui/table.go:583, 2456-2475, 1426, 1485 | Table: the current cell is not clamped when rows or columns shrink; no signal; navigation gets stuck |
| VIEW-6 | ui/table.go:452, 522, 2456 | Table: the cell editor stays open over the header after its row is removed (low-medium) |
| VIEW-7 | ui/table.go:822, 872, 985, 1576, 2408 | Table: reads create cell objects for arbitrary cells, an unbounded leak (low-medium) |
| VIEW-8 | ui/combobox.go:195, 454, 586 | ComboBox: a pick from a popup opened before the items changed reports an index that does not exist |
| VIEW-9 | ui/propertygrid.go:520-525, 573 | PropertyGrid: a float32 field gets ±Inf from the editor (low-medium) |
| VIEW-10 | ui/table.go:2003, 2073, 483, 1052, 1872; ui/treeview.go:994 | Quadratic cost: many columns, RemoveChildren with a large selection, SelectAll on a big model, an unclamped header span |
| CNV-4 | ui/canvas_effects.go:140-185, ui/canvas_frame.go:340 | `DrawShadow` panics on NaN/Inf/huge blur, NaN radius, non-finite position (medium/high) |
| CNV-5 | ui/canvas_effects.go:224-266 | `Canvas.Blur` hangs on a large radius (O(radius) per row) and its uint32 sum overflows |
| CNV-6 | ui/canvas_path_fill.go:44-106 | Even-odd fill: wrong coverage in the first column of a rasterized rectangle (up to 2x) |
| CNV-7 | ui/canvas_frame.go:236-261 | `FillTriangle`, `DrawLine` (width > 1) panic at coordinates >= 2^25 |
| CNV-8 | ui/canvas_transform.go:500-513 | `DrawText` under a large zoom panics in `image.NewRGBA` (`pw*ph` overflows) |
| CNV-9 | ui/canvas_clip.go:47-53 | `ClipPath` with very large coordinates clips everything away |
| CNV-11 | ui/canvas_svg.go:340-358 | SVG recorder: the font family is not escaped, so attributes can be injected into the SVG (security) |
| CHART-4 | ui/chart_series_stats.go:127-137 | Histogram: a far value panics (bin index overflow); values past bin 10000 pile into the last bin |
| CHART-6 | chart_series_line.go:282-298, chart_series_candle.go:83-90, chart_series_stats.go:631, chart_series_spans.go:158 | One NaN or Inf sample breaks the y autoscale: the axis shows 0..1, the data is off-screen |
| CHART-7 | ui/chart_scale.go:133-136 | Zooming in far enough makes the view jump out (to about 10 years on a time axis) |
| CHART-8 | ui/timechart_source.go:100-131 | TimeChart: dates before 1678 or after 2262 are bucketed at wrong times (UnixNano overflow) |
| CHART-9 | ui/chart_series_spans.go:219-223, ui/chart_view.go:932-950 | A span with a NaN end and a label panics in the text drawing |
| CHART-10 | ui/chart_series_pie.go:286-300 | Pie: a large start angle makes hover hang (low-medium) |
| PRS-4 | ui/instance.go:161, 92-104 | Single instance: arguments with a newline or over 64 KB are lost or truncated; "show" can be injected |
| PRS-5 | ui/filedialog_builtin.go:349 | Built-in folder dialog: nil dereference when a listed symlink stops being a folder |
| PRS-6 | ui/settings.go:67, 79 | A settings file holding `null` makes every later `Set` panic |
| PRS-7 | ui/settings.go:140-146 | `Settings.Save`, documented as safe for concurrent use, fails when called concurrently (low-medium) |
| CORE-3 | ui/widget.go:2675, 2920, 2491 | Grid layout hangs with a huge row or column span |
| ~~CORE-4~~ ✅ | ui/form.go:1489 | `processKeyDown`: nil dereference when the focused widget's key handler removes it |
| CORE-6 | ui/widget.go:1891-1892, 2113-2114 | `ProcessMouseDown`/`ProcessMouseDblClick`: index out of range when a handler removes siblings |
| CORE-8 | ui/widget.go:2267-2352, ui/action.go | Context menus shown and closed stay in the widget registry with their action handlers until the form closes |
| CORE-9 | ui/widget.go:3238-3241 | `SetContextMenu(nil)` panics; a replaced menu leaks |
| CORE-11 | ui/form.go:939-946, 419; ui/formstate.go:52; ui/dock.go:281-285 | `RememberState` of a form and of a DockLayout stop saving after the form is closed and shown again |
| CORE-13 | ui/boxes.go:225-247, ui/form.go:1373, 1553 | Enter/Esc run the OK/Cancel of a DialogButtonBox removed from the form (low/medium) |
| CORE-14 | ui/toast.go:84-95, 136; ui/tooltip.go:107-113 | Toast: no line limit, so a popup window millions of pixels tall |
| PLT-3 | internal/platforms/clipboard_linux.go:445-458 | X11: a copied text larger than the request limit (~16 MB) is "delivered" as nothing |
| PLT-4 | internal/platforms/clipboard_linux.go:328-350 | X11: pasting from a slow INCR owner blocks the UI thread and memory without limit |
| PLT-6 | internal/platforms/dragout_linux.go:52, dnd_linux.go:219 | Drag-out of a relative path makes its first segment the host; a drop takes `file://otherhost/...` as a local path |
| PLT-7 (part of MISC-6) | ui/accessible.go:152 | AT-SPI: `CurrentValue = NaN` from any D-Bus client makes a slider's value NaN |
| MISC-2 | ui/canvas3d_render.go:1243 | A 3D label at a non-finite or far point panics the render |
| MISC-3 | ui/font_registry.go:136 | The face cache grows without bound: a face per distinct font size (smooth zoom, NaN size) |
| MISC-4 | ui/tool_button.go:79 | `ToolButton.SetEnabled(false)` is ignored while a parent is disabled |
| MISC-5 | ui/zoomview.go:132, 181-191, 114 | ZoomView: a NaN zoom or a zoom range from 0 leaves the offset NaN for good (low-medium) |
| MISC-6 | ui/slider.go:94-160, ui/accessible.go:148 | Slider accepts NaN in the value, range and step, and is then stuck (low-medium) |
| MISC-8 | ui/canvas3d_render.go:605, canvas3d_mesh.go:100, 218, canvas3d_scene.go:343 | A negative mesh index panics the renderer, `ComputeNormals` and `Pick` (low-medium) |

### Low

| ID | Where | What |
|---|---|---|
| TXT-9 | ui/textbox.go:1038, 1262-1268 | An edit that changes nothing records an undo step and clears the redo history |
| TXT-10 | ui/textbox.go:1050, 1011-1014 | Editing a line with invalid UTF-8 rewrites its other bytes to U+FFFD |
| TXT-11 | ui/text_complex.go:202 | `TextSelectionSpans` panics when `from` is past the end |
| TXT-13 | ui/text_utils.go:52-79 | `MakeLinesFromStringWithWordWrapping` is quadratic |
| TXT-14 | ui/combobox_editable.go:436-447 | EditableComboBox: the highlighted match is cut at wrong bytes when lower-casing changes byte lengths |
| CNV-10 | ui/canvas.go:938-1018 | `CohenSutherland` overflows int: a visible line is reported as invisible (low/medium) |
| CNV-12 | ui/canvas_svg.go:361-378 | SVG recorder: control characters in text give an invalid SVG |
| CNV-13 | ui/canvas_cache.go:121-129 | `PaintCache.InvalidateRect` with an infinite or huge rectangle repaints nothing |
| CNV-14 | ui/canvas_path.go:1049 | x/image/vector fixed-point rasterizer: wrong coverage at sharp vertices |
| CNV-15 | internal/canvas/* | internal/canvas: Save/Restore is a no-op, translucent blending too dark, `Clear` panics on non-RGBA colors, sub-images wrong |
| CHART-11 | ui/chart_scale.go:126-137, 547-555 | A domain at the float limit becomes Inf: Symlog `Ticks` loops forever |
| CHART-12 | ui/chart_sync.go:175-193 | `SaveState` returns empty data when a selection or a range is non-finite |
| CHART-13 | ui/chart_series_candle.go:83-90 | Candles: an inverted candle or open/close outside high..low are left out of the autoscale |
| CHART-14 | ui/chart_export.go:28-31 | `Render`/`WritePNG` panic for a NaN or +Inf scale |
| CHART-15 | ui/chart_pane.go:64-65 | `ChartPane.SetWeight(NaN)`: all panes get NaN heights, nothing is drawn |
| CHART-16 | ui/chart_sparkline.go:139-141 | A constant sparkline of large values (>= 1e16) draws nothing |
| CHART-17 | chart_series_bar.go:91-103, chart_series_pie.go:71-80, chart_series_stats.go:580, chart_data.go:144 | Quadratic builders: `BarSeries.Set`, `PieSeries.Add`, `BandSeries.Add`, out-of-order `Append` |
| PRS-8 | ui/filedialog_builtin.go:225-230, 358 | Built-in folder dialog with a missing default directory: OK returns "" |
| PRS-9 | ui/filedialog_builtin.go:420-436 | Built-in file dialog can't open files whose names start with `"` or have spaces at the ends |
| PRS-10 | ui/markdownview.go:251-256 | MarkdownView keeps the old layout after a font change |
| PRS-11 | ui/timepicker.go:107-112 | `TimePicker.SetShowSeconds(false)` changes the value without `OnTimeChanged` |
| PRS-12 | ui/colorpicker.go:734 | `ParseHexColor` accepts malformed strings (`"# f0000"`, `"#0x1234"`) |
| PRS-13 | ui/print.go:117-178 | PrintJob: an invalid page size panics; zero-size or zero-page jobs write an invalid PDF |
| PRS-14 | ui/form.go:751, 700 | A saved window position with a negative X or Y (a monitor left of the primary) is not restored (by reading) |
| PRS-15 | ui/markdownview.go:211-224, ui/layout_xml.go:116 | Quadratic parsing: markdown `[`/backticks, XML character data |
| PRS-16 | ui/instance.go:135-145 | Single-instance socket falls back to the shared /tmp: another local user can take it (by reading) |
| PRS-17 (= MISC-9) | ui/markdownview.go:444, ui/link.go:154-167 | `OpenURL` leaves a zombie per link; MarkdownView links have no scheme check (by reading) |
| CORE-2 | ui/widget.go:2759, 3004 | A huge stretch weight overflows: integer divide by zero |
| CORE-10 | ui/menu_bar.go:61-63 | `MenuBar.AddMenuItem` with a nil menu panics once the bar is in a form |
| CORE-12 | ui/dock.go:268-277, 126-155 | DockLayout: a corrupt saved area makes a dock disappear for good |
| CORE-15 | ui/messagebox.go:184-190 | `ShowMessageBox*` panic for a parent widget that is not in a form |
| PLT-5 | internal/platforms/clipboard_linux.go:315-321 | X11: Latin-1 STRING received through INCR is not converted to UTF-8 |
| PLT-8 | internal/platforms/dragout_windows.go:61-64, 184 | Windows: `comCall` passes pointers to stack variables as `uintptr` through an ordinary function (by reading). The other `go vet` warnings about unsafe.Pointer are not bugs (see platform.md) |
| PLT-9 | internal/platforms/clipboard_linux.go:143, 175; clipboard.go:52, 68 | The clipboard keeps the caller's byte slices: reusing the buffer changes what is pasted |
| PLT-10 | internal/platforms/clipboard.go:62-73 | Headless clipboard does not take "text/plain" as the text, unlike X11 |
| PLT-11 | internal/platforms/headless.go:56, 471 | Headless: a NaN or +Inf scale makes `Render` panic |
| PLT-12 | internal/platforms/native_window_linux.go:1381-1421 | An application icon of 2048x2048 or larger is silently not set on X11 |
| MISC-7 | ui/slider.go:154-160 | Slider: a value at the maximum is off the step grid when (max-min) is not a multiple of the step |
| MISC-10 | ui/view3d.go:623-655 | View3D: zoom keys have no limits; the distance becomes Inf or 0, the target NaN |
| MISC-11 | ui/accessible.go:73-74 | `AccessibleRole.String()` panics for a negative role |
| MISC-12 | ui/canvas3d_mesh.go:730 | Text STL: a repeated `endfacet` adds the triangle again; a facet without a normal keeps the previous one |
| MISC-13 | ui/systemfont.go:178-182, ui/font_system.go:286-290 | macOS: `registerSystemFontFile` leaks an open file per skipped style (by reading) |

## Findings about the tests themselves

| ID | Sev. | Where | What |
|---|---|---|---|
| TEST-1 | low | ui/font_style_test.go:75-120, ui/systemfont_test.go:96-122 | `TestSynthesizedStyles` and `TestUnknownFamilyFromSystem` fail on a second run in one process (`go test -count=2 ./ui`, or `-count=3 -shuffle=on`). TestSynthesizedStyles registers a real bold face `test-synth:bold` at its end and leaves it registered, so the next run measures a registered bold, not a synthesized one ("bold width 46, regular 46"). The cleanup of TestUnknownFamilyFromSystem deletes `fonts["dejavu sans mono"]` but not the rest of the state RegisterFont made (`registerNativeFont`, the forgotten system family), so on the next run the family no longer resolves to the system font ("width 76, default 76"). Not a library bug: RegisterFont twice with the same name works. Fix: unique family names per run (e.g. with a counter), or a cleanup that undoes all of RegisterFont. |
| TEST-2 | low | examples/repaint_test.go:49 | `go test -race ./...` fails with the default timeout of 10 minutes: `TestRepaintMonkey` in `examples` takes more than 10 minutes under the race detector (85 s without it). With `-timeout 60m` the package passes and the detector reports no race. The CI does not run `-race`. Fix: a shorter monkey run under race (the `race_on_test.go`/`race_off_test.go` pattern of `ui` exists), then `-race` in the CI on Linux. |

## What was checked and found clean

- `go build` and `go vet` for linux, windows, darwin (amd64 and arm64), linux/386, linux/arm, windows/386. FreeBSD does not build, but it is not a supported system.
- `go test ./...` passes. `go test -race` passes for all the packages (examples with a longer timeout). `-shuffle=on` gives no failures except the two tests of TEST-1 on repeated runs.
- Fuzzing: `ParseSVGPath` with fill, stroke, bounds and clip (1M runs); `imageFromDIB` (2.5M); XML layouts, markdown, shortcuts, page ranges, file name lists; `RegisterFont` with corrupt fonts; 40,000 TextLayout cases; 4000×80 random TextBox operations; every rune from 0x20 to 0x2FFF through all the text paths.
- Concurrent off-screen painting, MeasureText/DrawText and LogView.Write under `-race`.
- Signal Connect/Disconnect during Emit, timers changed in their own callbacks, Close from handlers, RunAsync after the owner closed.
- No command injection in `filedialog_linux.go` (no shell, arguments as single `--opt=value` values) or in `OpenURL` (no shell).
- The lists "Checked, no bug found" at the end of each area's file give more.
