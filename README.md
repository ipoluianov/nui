# nui - Native UI Library for Go

Desktop applications in Go for Linux, Windows and macOS: forms, widgets, dialogs, charts, a 2D and 3D canvas. The library talks to the systems directly, without cgo, so a program builds and cross-compiles with a plain `go build`.

```
go get github.com/ipoluianov/nui
```

## Example

```go
package main

import "github.com/ipoluianov/nui/ui"

func main() {
	form := ui.NewForm()
	form.SetTitle("Hello")
	form.SetSize(320, 160)

	name := ui.NewTextBox()
	greeting := ui.NewLabel("")
	button := ui.NewButton("Greet")
	button.OnClick.Connect(func(ui.Widgeter) {
		greeting.SetText("Hello, " + name.Text() + "!")
	})

	fields := ui.NewFormLayout()
	fields.AddRow("Name:", name)
	p := form.Panel()
	p.AddWidget(0, 0, fields)
	p.AddWidget(1, 0, button)
	p.AddWidget(2, 0, greeting)
	p.AddVSpacer(3, 0)

	form.Show()
	form.Exec() // the event loop, until the window closes
}
```

`go run .` in this repository opens the launcher of the example applications in [examples/](examples): notepad, calculator, contacts, file explorer, paint, tasks, network monitor, unit converter, function plotter, chart gallery, canvas, 3D viewer, CAD. `go run . notepad` runs one of them.

## Documentation

- [Recipes](doc/recipes.md): the main window, dialogs, background work, tables, translations, the tray, tests: common tasks in short code.
- [Widgets and the API of `ui`](doc/ui/README.md): a page for each widget and topic.
- [Rules](doc/rules.md): the UI thread, repaint, events, dialogs: what the signatures do not show. Read before the first change.
- [Testing](doc/testing.md): forms tested without a display, golden pictures, benchmarks.
- [To do](doc/todo.md): what is known to be missing.

## What there is

- **Widgets:** buttons, text boxes (with a code editor's line numbers and highlighting), number boxes, check boxes, combo boxes, sliders, date and time pickers, color picker, tables and lists on models of millions of rows, trees, property grid, tabs, splitters, dock panels, menus, toolbars, status bar, log view, Markdown view, toasts.
- **Layouts:** grid with spans and weights, HBox, VBox, FormLayout, dialog buttons in the order of the platform, XML layouts.
- **Charts:** lines, bars, pies, scatter, candles, heat maps, live time charts.
- **Canvas:** paths, transforms, gradients, shadows, layers, SVG; 3D scenes and models; 2D drawings for CAD.
- **Application services:** settings, the window where the user left it, one copy of the application, background tasks with progress, clipboard with images, drag and drop of files, tray icon, printing and PDF, accessibility (AT-SPI on Linux).
- **Text:** built-in Noto Sans and JetBrains Mono with bold and italic, system fonts, color emoji, shaping and right-to-left text, input methods for Chinese, Japanese, Korean.
- **Languages:** the library's own texts in English, Russian, Chinese, Japanese, Korean, Turkish, Indonesian, Vietnamese and the main languages of Europe; translations of the application as Go code ([i18n](ui/i18n)).
- **HiDPI** on all systems: sizes in logical pixels, drawn at the scale of the screen.

## Packages

| Package | Import path | Purpose |
|---|---|---|
| `ui` | `github.com/ipoluianov/nui/ui` | forms, widgets, dialogs, themes, canvas, charts |
| `i18n` | `github.com/ipoluianov/nui/ui/i18n` | translations |
| `uitest` | `github.com/ipoluianov/nui/ui/uitest` | tests of forms without a display |

The native layer lives in `internal/` and is not importable by applications ([its docs](doc/platforms/README.md), for contributors).

## Building

| System | Build | Needs at run time |
|---|---|---|
| Linux | `go build` | `libX11.so` (any desktop; Wayland through XWayland) |
| Windows | `go build -ldflags="-H=windowsgui"` (no console window) | - |
| macOS | `go build` | - (AppKit) |

No C compiler, headers or SDKs are needed: Linux and macOS are reached at run time through [purego](https://github.com/ebitengine/purego), Windows through `syscall`. Any system builds for any other with `GOOS=linux`, `GOOS=windows` or `GOOS=darwin`.
