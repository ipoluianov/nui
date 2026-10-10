# Getting started

## Install

```
go get github.com/ipoluianov/nui
```

## Build requirements

### Linux
No packages needed to build - the Linux backend talks to Xlib at runtime
via [purego](https://github.com/ebitengine/purego), not cgo, so
`GOOS=linux go build` also cross-compiles from macOS/Windows. `libX11.so`
must be present on whatever machine *runs* the binary (true of virtually
any Linux desktop, GNOME/KDE included, even under Wayland via XWayland).

### Windows
```
go build -ldflags="-H=windowsgui" .
```

### macOS
No packages needed to build - the macOS backend talks to Cocoa/AppKit at
runtime via [purego](https://github.com/ebitengine/purego) (including its
`objc` Objective-C runtime bindings), not cgo, so `GOOS=darwin go build` also
cross-compiles from Linux/Windows. AppKit itself must be present on whatever
machine *runs* the binary (true of every macOS install).

## Minimal app

```go
package main

import "github.com/ipoluianov/nui/internal/platforms"

func main() {
	platforms.CreateDefaultWindow().Exec()
}
```

`Exec()` shows the window and runs the event loop until the window is closed. Call it from the main goroutine: all windows and their callbacks live on the UI thread, the main OS thread (see [multi-window.md](multi-window.md)).

## Window with drawing and input

```go
package main

import (
	"image"
	"image/color"

	"github.com/ipoluianov/nui/internal/canvas"
	"github.com/ipoluianov/nui/internal/platforms"
)

func main() {
	win := platforms.CreateWindow("My App", 100, 100, 800, 600, true, false)

	win.OnPaint(func(rgba *image.RGBA) {
		cnv := canvas.NewCanvas(rgba)
		cnv.SetColor(color.RGBA{255, 255, 255, 255})
		cnv.DrawFixedString(10, 10, "Hello, nui!", 2)
	})

	win.OnKeyDown(func(key platforms.Key, mods platforms.KeyModifiers) bool {
		if key == platforms.KeyEsc {
			win.Close()
		}
		return true
	})

	win.Exec()
}
```

- `OnPaint` is called whenever the window needs to redraw. Draw only inside it.
- Call `win.Update()` after changing any state that affects the picture, to request a repaint.
- Callback return values of `true` mean "event handled" (see [window.md](window.md)).

## Run examples in this repo

```
go run ./main.go
```

`main.go` runs the launcher of the example applications (`examples/examples.go`);
`go run . notepad` runs one of them by name (run with an unknown name for the list).
