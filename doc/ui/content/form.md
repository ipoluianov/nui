# Form

`Form` is the main application window.

## Create and run

```go
form := ui.NewForm()
form.SetTitle("My app")
form.SetSize(800, 600)
form.Exec()
```

## Key methods

- **`Panel() *ui.Panel`**: form root container.
- **`SetMainWidget(w ui.Widgeter)`**: replace the root content with a single widget.
- **`Exec()` / `ExecMaximized()`**: run the window event loop.
- **`Close()`**: close the window.
- **`SetIcon(img image.Image)`**: set the icon of this window (title bar / taskbar), overriding the application icon. Works before and after the window is shown.

## Application icon

- **`ui.SetAppIcon(img image.Image)`**: set the default icon for all windows. Call it at startup, before showing any form; every form created afterwards uses it unless it has its own `SetIcon`.

## Global events

- **`SetOnGlobalKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool)`**: intercept key presses before focused widget.

