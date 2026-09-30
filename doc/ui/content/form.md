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
- **`Close()`**: close the window at once, without calling `OnClose`. Safe from any goroutine. A window with an open modal dialog closes as soon as the dialog does.
- **`RequestClose() bool`**: close the window as if the user clicked its close button: `OnClose` runs first and can keep the window open. Call it on the UI thread (a handler or `Invoke`).
- **`Invoke(f func())`**: run `f` on the UI thread, then repaint the form. Safe from any goroutine.
- **`Hide()` / `Show()` / `IsHidden()`**: hide the window (taskbar button and all) and bring it back; the form stays open. See [TrayIcon](trayicon.md).
- **`ShowToast(text, kind)` / `ShowToastFor(text, kind, duration)`**: a short message in the corner, see [Toasts](toast.md).
- **`SetOnFilesDropped(func(files []string, x, y int))`**: files dropped from the system outside the drop targets, see [Drag and drop](dragdrop.md).

## File dialogs

The system dialogs don't block the form; the result comes on the UI thread (empty when cancelled).

```go
form.ShowOpenFileDialog(ui.OpenFileDialogOptions{Title: "Open", AllowMultiple: true,
	Filters: []ui.FileDialogFilter{{DisplayName: "Text", Patterns: []string{"*.txt"}}}},
	func(paths []string, err error) { ... })
form.ShowSaveFileDialog(ui.SaveFileDialogOptions{DefaultFileName: "report.txt"},
	func(path string, err error) { ... })
form.ShowSelectDirectoryDialog(ui.SelectDirectoryDialogOptions{Title: "Folder"},
	func(path string, err error) { ... })
```

On Linux the dialogs need `zenity` or `kdialog` (`ui.ErrNoFileDialog` otherwise).

## Threading

All forms live on the UI thread, the main OS thread: `Exec()` called from `main()` runs the event loop of every form, and all the handlers of all the forms run there, one at a time. A handler of one form can therefore change the widgets of another form directly.

Other goroutines must not touch forms and widgets directly. They hand the work over with `ui.Invoke(f)` / `form.Invoke(f)` (asynchronous) or `ui.InvokeSync(f)` (waits). `Show`, `ShowModal` and `Close` do this hand-over themselves.

```go
go func() {
	data := loadData() // slow work off the UI thread
	form.Invoke(func() { table.SetData(data) })
}()
```
- **`OnClose func() bool`**: called when the user closes the window (close button, Alt+F4) or on `RequestClose`; return `false` to keep it open. Not called by `Close()`.
- **`SetIcon(img image.Image)`**: set the icon of this window (title bar / taskbar), overriding the application icon. Works before and after the window is shown.

## Application icon

- **`ui.SetAppIcon(img image.Image)`**: set the default icon for all windows. Call it at startup, before showing any form; every form created afterwards uses it unless it has its own `SetIcon`.

## Global events

- **`SetOnGlobalKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool)`**: intercept key presses before focused widget.

