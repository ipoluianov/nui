# Window API

`platforms.Window` is the interface returned by every window constructor. Same interface on Linux/Windows/macOS.

## Create a window

```go
win := platforms.CreateWindow(title string, posX, posY, width, height int, center bool, maximized bool) platforms.Window
win := platforms.CreateDefaultWindow() platforms.Window // title "App", 800x600, centered
```

- `center`: if `true`, ignores `posX`/`posY` and centers the window on screen.
- `maximized`: if `true`, opens maximized.
- Creating a window does not show it yet; call `Show()`, `Exec()`, or add it to `platforms.Run(...)`.

## Running the event loop

```go
win.Exec()      // Show() + EventLoop(), blocks current goroutine until closed
win.Show()      // shows the window
win.EventLoop() // blocks current goroutine, pumps events until closed
```

Run one window per goroutine. See [multi-window.md](multi-window.md) for running several windows.

## Closing

```go
win.Close()
```

Safe to call from any goroutine (including from another window's callback). Does not block.

```go
win.OnCloseRequest(func() bool {
	return true // true = allow close, false = cancel (e.g. unsaved changes)
})
```

`OnCloseRequest` fires when the user clicks the OS close button, before the window actually closes.

## Repainting

```go
win.Update() // request a repaint (coalesced, ~max 25 fps)
win.OnPaint(func(rgba *image.RGBA) {
	// draw into rgba, e.g. via canvas.NewCanvas(rgba)
})
```

`rgba` is a fresh per-window buffer already filled with the window's background color. Do not store the `*image.RGBA` pointer across calls — draw and return.

## Properties

```go
win.SetTitle(title string)
win.SetAppIcon(icon *image.RGBA)
win.SetBackgroundColor(color.RGBA)
win.SetMouseCursor(cursor platforms.MouseCursor)

win.Move(x, y int)
win.MoveToCenterOfScreen()
win.Resize(width, height int)
win.MinimizeWindow()
win.MaximizeWindow()
win.IsMaximized() bool

win.Size() (width, height int)
win.Pos() (x, y int)
win.PosX() int
win.PosY() int
win.Width() int
win.Height() int
win.KeyModifiers() platforms.KeyModifiers // live modifier state
win.DrawTimeUs() int64                // average paint time, microseconds
```

## Events

All `On*` setters take a callback and replace any previously set one (only one callback per event).

```go
win.OnCreated(func())

win.OnKeyDown(func(key platforms.Key, mods platforms.KeyModifiers) bool) // return true = handled
win.OnKeyUp(func(key platforms.Key, mods platforms.KeyModifiers))
win.OnChar(func(char rune)) // text input, after layout/keysym translation

win.OnMouseEnter(func())
win.OnMouseLeave(func())
win.OnMouseMove(func(x, y int))
win.OnMouseButtonDown(func(btn platforms.MouseButton, x, y int))
win.OnMouseButtonUp(func(btn platforms.MouseButton, x, y int))
win.OnMouseButtonDblClick(func(btn platforms.MouseButton, x, y int))
win.OnMouseWheel(func(deltaX, deltaY int))

win.OnMove(func(x, y int))
win.OnResize(func(width, height int))
win.OnCloseRequest(func() bool)
win.OnTimer(func()) // fires every ~10ms while the window is open
```

All callbacks run on the goroutine that is executing that window's `EventLoop()`/`Exec()`. Do not block inside them.

## Modal dialogs

```go
dlg := platforms.CreateWindow("Dialog", 0, 0, 300, 150, true, false)
dlg.OnKeyDown(func(k platforms.Key, m platforms.KeyModifiers) bool {
	if k == platforms.KeyEsc {
		dlg.Close()
	}
	return true
})
dlg.ShowModal(parentWin)
```

`ShowModal` returns immediately (non-blocking); the dialog runs on its own goroutine. The window manager blocks input to `parentWin` while the dialog is open (parent keeps repainting/timers). See [multi-window.md](multi-window.md).
