# nui - Native UI Library for Go

Native gateway between OS UI & Golang with minimum dependencies, plus a
forms/widgets library built on top of it.

Documentation: [doc/README.md](doc/README.md)

```
go get github.com/ipoluianov/nui
```

# Packages

| Package | Import path | Purpose |
|---|---|---|
| `ui` | `github.com/ipoluianov/nui/ui` | forms, widgets, dialogs, themes |
| `i18n` | `github.com/ipoluianov/nui/ui/i18n` | translations |

Keyboard and mouse types (`ui.Key`, `ui.KeyModifiers`, `ui.MouseButton`,
`ui.MouseCursor` and their constants) are part of `ui`. The native layer lives
in `internal/` and is not importable by applications.

# Example

```go
package main

import "github.com/ipoluianov/nui/ui"

func main() {
	form := ui.NewForm()
	form.SetTitle("Example 01 - Base Form")
	form.Exec()
}
```

Runnable demos: `go run ./main.go` (widgets gallery, `examples/`).

# Operating Systems
- Linux
- Windows
- MacOS

# Base Widgets
- Button
- ComboBox
- ContextMenu
- Dialog
- HSpacer
- Label
- Panel
- Splitter
- Table
- TabWidget
- TextBox
- VSpacer

# Linux build
- go build -o bin/nui ./main.go

No C compiler or X11 headers needed to build (the Linux backend talks to
Xlib at runtime via [purego](https://github.com/ebitengine/purego), not
cgo), so this also cross-compiles from macOS/Windows with a plain
`GOOS=linux go build`. `libX11.so` still has to be present on whatever
machine actually runs the binary - true of virtually any Linux desktop
(GNOME/KDE, even under Wayland via XWayland).

# Windows build
- go build -o bin/nui.exe -ldflags="-H=windowsgui"

# macOS build
- go build -o bin/nui ./main.go

No C compiler or Xcode command line tools needed to build (the macOS backend
talks to Cocoa/AppKit at runtime via [purego](https://github.com/ebitengine/purego),
not cgo), so this also cross-compiles from Linux/Windows with a plain
`GOOS=darwin go build`. AppKit still has to be present on whatever machine
actually runs the binary - true of every macOS install.
