# Dialog

A dialog is a widget embedding `ui.DialogContent`, shown in a modal window over a form with `ShowDialog`. Its `OnDialogShow` sets the window up (the title, the size, the focus) before it appears; `OnDialogReject` runs when the user closes the window (its close button, Esc with a cancel button) and may keep it open.

```go
type renameDialog struct {
	ui.DialogContent
	name *ui.TextBox
	// OnOK gets the new name
	OnOK func(name string)
}

func newRenameDialog(name string) *renameDialog {
	var c renameDialog
	c.InitWidget()
	c.name = ui.NewTextBox()
	c.name.SetText(name)
	c.AddWidget(0, 0, c.name)
	c.AddVSpacer(1, 0)
	buttons := c.AddPanel(2, 0)
	buttons.AddHSpacer(0, 0)
	ok := buttons.AddButton(0, 1, "OK", func() {
		name := c.name.Text()
		c.RunInParent(func() { c.OnOK(name) }) // after the dialog closes
		c.Form().Close()
	})
	cancel := buttons.AddButton(0, 2, "Cancel", func() { c.Form().Close() })
	c.OnDialogShow = func() {
		c.Form().SetTitle("Rename")
		c.Form().SetSize(360, 120)
		c.Form().SetAcceptButton(ok)     // Enter
		c.Form().SetCancelButton(cancel) // Esc
		c.name.Focus()
	}
	return &c
}

// Shown over the form of a widget:
dialog := newRenameDialog(file.Name)
dialog.OnOK = func(name string) { rename(file, name) }
form.Panel().ShowDialog(dialog)
```

`ShowDialog` returns at once: the result comes in a callback. `RunInParent` runs the callback after the dialog's handler returns, and repaints the window the dialog was shown over with the changes it made.

A complete example: the Find and Go To Line dialogs of `examples/notepad/dialogs.go`.

## The buttons

`ui.NewDialogButtonBox()` puts the buttons in the order of the platform (OK before Cancel on Windows and Linux, after it on macOS), in the language of the interface, and makes them the form's accept (Enter) and cancel (Esc) buttons:

```go
buttons := ui.NewDialogButtonBox()
buttons.AddOK(func() { c.RunInParent(func() { c.OnOK(name) }); c.Form().Close() })
buttons.AddCancel(func() { c.Form().Close() })
buttons.AddButton("Apply", ui.ButtonOther, apply)
c.AddWidget(2, 0, buttons)
```

## Keeping the dialog open

`OnDialogReject` returns `false` to keep the window open when the user closes it, e.g. to ask about unsaved changes first. `Form().Close()` from the code closes the window without calling it.
