# Form

`Form` is the main application window.

## Create and run

```go
form := ui.NewForm()
form.SetTitle("My app")
form.SetSize(800, 600)
form.Panel().AddWidget(0, 0, ui.NewLabel("Hello"))
form.Show()
form.Exec() // runs the event loop until the form is closed
```

`Exec` needs a shown form: call `Show`, `ShowMaximized` or `ShowModal` first.

## Key methods

- **`Panel() *ui.Panel`**: form root container.
- **`SetMainWidget(w ui.Widgeter)`**: replace the root content with a single widget.
- **`Show()` / `ShowMaximized()`**: open the window and return at once.
- **`ShowModal(parent *ui.Form)`**: open the form as a dialog over `parent`, which gets no input until it closes (see [Dialog](dialog.md)).
- **`Exec()`**: wait until the form is closed. From `main()` it runs the event loop of all the forms; from a handler, a nested loop.
- **`SetTitle(text)` / `SetTitleFunc(f)`** (follows the language), **`SetSize(w, h)`**, **`Move(x, y)`**, **`Maximize()` / `Minimize()` / `Restore()`**.
- **`OnClose func() bool`**: called when the user closes the window (close button, Alt+F4) or on `RequestClose`; return `false` to keep it open. Not called by `Close()`.
- **`Close()`**: close the window at once, without calling `OnClose`. Safe from any goroutine. A window with an open modal dialog closes as soon as the dialog does.
- **`RequestClose() bool`**: close the window as if the user clicked its close button: `OnClose` runs first and can keep the window open. Call it on the UI thread (a handler or `Invoke`).
- **`Invoke(f func())`**: run `f` on the UI thread, then repaint the form. Safe from any goroutine.
- **`Update()`**: repaint the whole form with the next frame. A widget that changed what it draws calls its own `Update()`, which repaints only its rectangle.
- **`AddShortcut(shortcut, f)`**: run `f` on a key wherever the focus is, e.g. `form.AddShortcut("Mod+F", search.Focus)` ("Mod" is Cmd on macOS, Ctrl elsewhere). `AddAction(a)` does the same for an [Action](actions.md).
- **`SetAcceptButton(btn)` / `SetCancelButton(btn)`**: the buttons pressed by Enter and Esc.
- **`SetMenuBar(bar)`**: the main menu, see [MenuBar](menu_bar.md).
- **`SetOnActivated(f)` / `SetOnDeactivated(f)`**, **`IsActive()`**: the window got or lost the keyboard focus.
- **`FocusedWidget()`**, **`WidgetById(id)`**.
- **`Hide()` / `Show()` / `IsHidden()`**: hide the window (taskbar button and all) and bring it back; the form stays open. See [TrayIcon](trayicon.md).
- **`ShowToast(text, kind)` / `ShowToastFor(text, kind, duration)`**: a short message in the corner, see [Toasts](toast.md).
- **`KeyModifiers() ui.KeyModifiers`**: which of Shift, Ctrl, Alt (Cmd on macOS) are held right now, not only inside a key event - e.g. to show what the function keys do with them; read it in a timer. Call it on the UI thread.
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

On Linux the system dialogs are `zenity` or `kdialog`. Without either, nui shows its own dialog:

- the path of the folder with Up and Home;
- the folder's files (folders first);
- the name, the filter and the hidden files.

Enter or a double click opens a folder or picks a file. A folder or a path typed as the name is opened. Several names can be typed in quotes (`"a.txt" "b.txt"`). On save, the dialog asks before replacing a file and adds the filter's extension to a name without one.

`ui.UseBuiltinFileDialogs(true)` makes the forms use nui's dialog everywhere: the same on every system, and testable with `uitest`, where the system dialogs return "cancelled".

## Threading

All forms live on the UI thread, the main OS thread: `Exec()` called from `main()` runs the event loop of every form, and all the handlers of all the forms run there, one at a time. A handler of one form can therefore change the widgets of another form directly.

Other goroutines must not touch forms and widgets directly. They hand the work over with `ui.Invoke(f)` / `form.Invoke(f)` (asynchronous) or `ui.InvokeSync(f)` (waits). `Show`, `ShowModal` and `Close` do this hand-over themselves.

```go
go func() {
	data := loadData() // slow work off the UI thread
	form.Invoke(func() { list.SetItems(data) }) // data []string
}()
```

## Icons

- **`form.SetIcon(img image.Image)`**: set the icon of this window (title bar / taskbar), overriding the application icon. Works before and after the window is shown.
- **`ui.SetAppIcon(img image.Image)`**: set the default icon for all windows. Call it at startup, before showing any form; every form created afterwards uses it unless it has its own `SetIcon`.

## Global events

- **`SetOnGlobalKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool)`**: intercept key presses before the focused widget (a focused text field still takes its editing keys, e.g. Ctrl+Z, first); return `true` to stop the key there.

