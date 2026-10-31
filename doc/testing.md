# Testing nui and the applications on it

There are five tools:

| Tool | What it checks | Run |
|---|---|---|
| `ui/uitest` | forms without a display: clicks, typing, menus, dialogs, golden pictures | `go test ./...` |
| `ui/regression` | the behavior of nui as an application sees it: bugs fixed and the rules around them | `go test ./ui/regression` |
| `ui/benchmarks` | performance: TextBox, tables, layout, painting, charts | `go test ./ui/benchmarks -run xxx -bench . -benchmem` |
| `TestSnapshots` and `cmd/snapdiff` | that a change of painting did not change a picture where it should not: screenshots of all the examples before and after | see [below](#snapdiff-screenshots-before-and-after) |
| `scripts/check-clients.sh` | that the applications on nui next to it build and pass their tests with the working copy of nui | `scripts/check-clients.sh` |

The CI ([.github/workflows/test.yml](../.github/workflows/test.yml)) runs `go build ./...` and `go test ./...` on Linux, macOS and Windows; on Linux it also cross-builds for the other systems. When a test fails, the pictures `*.actual.png` and `*.diff.png` are attached to the run as the artifact `golden-<OS>`.

---

## uitest: forms without a display

Importing `github.com/ipoluianov/nui/ui/uitest` puts the whole test binary in the headless mode:
- the windows live in memory only, nothing shows on the screen;
- no display is needed: it runs in CI and over SSH;
- Linux, Windows and macOS.

A test shows a form, acts as the user and checks the state of the widgets or the picture:

```go
func TestEditing(t *testing.T) {
	n := newNotepad()
	w := uitest.Show(t, n.form)        // the form is closed at the end of the test

	w.Shortcut("Mod+A")                // Cmd on macOS, Ctrl on the others
	w.Type("hello world")
	w.Click(w.Find("Edit"))            // a menu of the menu bar, by its text
	w.Click(w.Find("Upper Case"))      // an item of the open menu

	if got := n.text.Text(); got != "HELLO WORLD" {
		t.Errorf("text = %q", got)
	}
	w.MatchGolden("notepad-upper")     // compared with testdata/golden/notepad-upper.png
}
```

A working example is [examples/notepad/notepad_ui_test.go](../examples/notepad/notepad_ui_test.go); each feature alone is in [ui/uitest/uitest_test.go](../ui/uitest/uitest_test.go).

### What a `Window` does

| Group | Methods |
|---|---|
| Mouse | `Click`, `RightClick`, `DoubleClick`, `Drag(from, to)`, `Move`, `Press`/`Release`, `Wheel(at, dx, dy)`, `DropFiles(at, files...)` |
| Keyboard | `Type("text\n")`, `Key(ui.KeyF5, mods)`, `Shortcut("Ctrl+Shift+S")` |
| Finding widgets | `Find("name or text")`, `FindAll(func)`, `Focused()`, `Popup()` |
| Windows | `Dialog()` - the modal window over the form, `Top()` - the window opened last, `Blocked()`, `Closed()`, `Title()`, `RequestClose()` - a click on the close button, `Resize` |
| Time | `Settle()` - runs the pending `Invoke` calls, `Tick()` - fires the timers once, `Wait(d)` - waits with the timers running |
| Pictures | `Screenshot()`, `MatchGolden(name)` |

The target of an action is either:
- **a widget**: the action happens at its center;
- **a point**: `uitest.At(widget, dx, dy)`, or `uitest.Point{X, Y}` in the coordinates of the window.

### Golden pictures

`MatchGolden("name")` compares the picture of the window with `testdata/golden/name.png` of the test's package.

- **No golden picture yet.** The test writes it and fails: look at the picture and run the test again.
- **A difference.** `name.actual.png` (what came out) and `name.diff.png` (the differences in red) are written next to the golden picture.
- **Writing the golden pictures again:** `go test ./pkg -update-golden` or `NUI_UPDATE_GOLDEN=1 go test ./pkg`. Look at the new pictures before a commit: from then on the test expects them.
- **A golden picture of a system.** The order of the buttons of a dialog (Cancel, OK on macOS) and the names of the shortcuts (⌃S for Ctrl+S) differ, so such pictures have a golden picture of their own in `testdata/golden/<GOOS>/name.png` (`darwin`, `windows`, `linux`). When it is there the test takes it, otherwise the common one. To make one for a system, copy the common one to `testdata/golden/<GOOS>/` and run the test with `-update-golden` on that system: the update writes where the golden picture is, the common one is left.
- **Tolerances.** `uitest.GoldenTolerance` is how much a channel of a pixel may differ, 8 of 255 by default; `uitest.GoldenMaxDiff` is the share of the pixels that may differ, 0 by default.

For the same pixels on every machine `uitest` turns on the light theme and draws the fonts with the rasterizer of nui (`NativeFontRendering = false`). The screen is set with `uitest.SetScreen(w, h, scale)`; a `scale` of 2 gives pictures as on a Retina screen.

### The headless mode and a real window

| What | Headless | Why |
|---|---|---|
| The UI thread | the test goroutine | an `Invoke` from other goroutines runs on `Settle` and after each action |
| Timers | fire only on `Tick`/`Wait` | otherwise the pictures would depend on the time, e.g. on the blink of the caret |
| `Exec()` | returns at once | the test drives the events |
| `ShowModal` | does not block on any system | as on Windows and Linux. A modal window blocks the input of its parent: clicks on the parent are ignored |
| Popups (menus, lists) | painted over the form, get the clicks | - |
| Items of ComboBox lists, rows of Table and TreeView | painted, not widgets, so `Find` does not find them | chosen with keys, or with a click at `uitest.At(w.Popup(), x, y)` |
| Clipboard | of the process | `ui.ClipboardGetText()` in a test reads what the application copied |
| File dialogs | return "cancel" at once | - |
| Printing, tray | no printers, the tray icon is not shown | - |
| Dragging files out | does not start | - |
| Links (`ui.OpenURL`, `Link`) | not opened | the test would open a browser, mail, Skype (`tel:`) |
| The buffer of the window | keeps the last frame, as a real window's | a frame repaints only what changed, over the last one |

---

## snapdiff: screenshots before and after

Golden pictures check a few forms. A change of painting itself (Canvas, the output to the screen, the theme) is checked wider: by screenshots of all the applications of `examples` and of a form with all the widgets, in the dark and the light theme, at scales 1, 1.5 and 2.

```
NUI_SNAPSHOT_DIR=/tmp/before go test ./examples -run TestSnapshots -count=1
# ... the change ...
NUI_SNAPSHOT_DIR=/tmp/after go test ./examples -run TestSnapshots -count=1
go run ./cmd/snapdiff -out /tmp/diff /tmp/before /tmp/after
```

`snapdiff` lists the pictures that differ, and in each the areas of the pixels that differ. With `-out` it writes a picture of each area: before on the left, after on the right, enlarged so that every pixel is seen. It exits with 1 when something differs.

- A change that must not change the pictures (a speedup, a refactoring) gives `N same, 0 differ`.
- For a change that changes them, look at every area: only what was meant may differ.
- `-tolerance n` does not count a difference of the components up to `n`: a change of rounding in color blending moves many pixels by 1-2, and so what else changed is seen. The largest difference of each area is printed.
- The pictures are the same from run to run. The network monitor is not among them: its data are random and its time axis is the clock. The mark of today in the calendar moves at midnight: take "before" and "after" on one day.

---

## Partial repaint

A frame repaints only the rectangles of the widgets that asked for it (`Widget.Update()`), over the last frame in the buffer of the window (see `ui/form_repaint.go`). A bug here looks like an old picture left on the screen: a widget changed without asking to be repainted. The tests that catch it:

- **`NUI_CHECK_REPAINT`.** With `NUI_CHECK_REPAINT=1` every frame repainted in part is compared with a whole repaint. A difference is written to stderr: the area, the rectangles repainted and the widgets in the area. With `NUI_CHECK_REPAINT=panic` a difference panics (the tests run so). With `NUI_CHECK_REPAINT_DIR=dir` both pictures are saved, for `snapdiff`. It works in a real window too: run the application with the variable and use it.
- **`TestRepaintMonkey`** (`examples`). Opens all the applications and the form with all the widgets and does random actions: moves, hovers, the wheel, drags, clicks, keys, waits for timers; the frame is compared after each. 60 actions at scales 1 and 1.5 by default; 30 at 1.5 with `-short`; `NUI_MONKEY=long` gives 400 at 1, 1.5 and 2, about 10 minutes. The sequence is the same from run to run (the random numbers are seeded by the name of the test); a failure prints the last actions.
- **`TestPaintClipped`** (`ui`). Painting clipped to a part of the frame gives the pixels of the whole frame. For that all the geometry (the clipping of paths, the pieces of rasterization, the clipping of lines, the corners of frames) is worked out in the clip of the whole frame (`Canvas.shapeClip`), and only the pixels of the part are written (`Canvas.clipRect`). A new drawing primitive follows the same rule. The exception is `Canvas.Blur`: at the edge of the part the blur lacks the neighbors.
- **A paint draws what the next one would.** A widget that lays itself out while painting (from what it measures) must settle within one paint: a part of the frame repainted later draws with the layout of that later paint. `TestTimeChartPaintAsNext` checks it for the charts.

---

## regression: the behavior kept

[ui/regression](../ui/regression) holds tests of the behavior of nui as an application sees it, by area: layout, events, signals, text boxes, tables, dialogs, the canvas, the XML layout. Most began as a bug fixed: the test keeps it from coming back. `TestDocsMentionExistingAPI` checks that every name of the package `ui` the documentation mentions exists.

---

## benchmarks: performance

The forms are shown in the headless mode, so the numbers are the library's own, without the system's drawing.

```
go test ./ui/benchmarks -run xxx -bench . -benchmem          # all, about 1.5 minutes
go test ./ui/benchmarks -run xxx -bench . -benchmem -short   # without the 20 000 lines, about 15 seconds
```

| Benchmark | What |
|---|---|
| `TextBoxSetText/N` | loading a text of N lines into a multiline TextBox |
| `TextBoxKey/N` | typing one character into a TextBox of N lines (press, character, release), without painting |
| `TextBoxPaint/N` | a frame of a form with such a TextBox, e.g. the caret blinking |
| `AddWidgets` | 1000 labels added one by one to a shown form |
| `LabelSetText` | the text of a label changed on a form of 300 labels |
| `TableFill` | a table of 100 000 × 5 filled with `SetCellText2` |
| `TableModel`, `TableModelPaint` | a table on a model of 1 000 000 × 5: setting the model and a frame |
| `ListViewPaint` | a frame of a list on a model of 1 000 000 rows |
| `TablePaint` | a frame of a form with such a table |
| `TablePaintLarge` | a frame of a table of 10 000 × 10 over a window of 2560×1440: much text, lines and fills |
| `EmptyFormPaint` | a frame of an empty form of 2560×1440, the floor of any repaint (the allocation of the buffer, 14.7 MB, included) |
| `ChartLines/N` | 10 lines of N points in memory, all seen, a chart of 1920×600: `ChartView.Render`, two passes of layout and drawing |
| `ChartLive` | 20 lines of a live stream (the last 3600 points each): a new point in each and `Render` at 1280×500 |
| `ChartScatterMillion` | a million points drawn as density, `Render` at 1280×800 |
| `ChartHeatmap` | a heat map of 2000×2000, `Render` at 1280×800 |

Run them before and after a change of layout, painting or text, on the same machine, to see what it gained and that nothing went back.

---

## check-clients.sh: the applications next to nui

```
scripts/check-clients.sh                 # every module next to nui that requires nui
scripts/check-clients.sh ../altfm        # the ones named
QUICK=1 scripts/check-clients.sh         # build and vet only, no tests
```

Each client is checked twice:
- **pinned**, with the version of nui of its `go.mod`;
- **local**, with the working copy of nui, put in by a temporary `go.work`; the files of the clients are not changed.

The script fails when a client worked pinned and broke local. The logs of the local failures are kept in `$TMPDIR/nui-clients`.

Run it before changing the public API: it shows which clients the change touches.
