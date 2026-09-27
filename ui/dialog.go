package ui

type Dialoger interface {
	onDialogShow()
	onDialogReject() bool
}

type DialogContent struct {
	Widget

	OnDialogShow   func()
	OnDialogReject func() bool
}

func (c *DialogContent) InitWidget() {
	c.Widget.InitWidget()
}

// RunInParent runs f on the goroutine of the window the dialog was shown over.
// Use it for the dialog's result callbacks: they change the widgets of that
// window, which must not be touched from the dialog's own goroutine (see Form.Invoke).
// Read what f needs from the dialog's widgets before calling RunInParent.
func (c *DialogContent) RunInParent(f func()) {
	if f == nil {
		return
	}
	if c.Form() == nil || c.Form().ParentForm() == nil {
		f()
		return
	}
	c.Form().ParentForm().Invoke(f)
}

func (c *DialogContent) onDialogShow() {
	if c.OnDialogShow != nil {
		c.OnDialogShow()
	}
}

func (c *DialogContent) onDialogReject() bool {
	if c.OnDialogReject != nil {
		return c.OnDialogReject()
	}
	return true
}

func (c *Widget) ShowDialog(centralWidget Widgeter) {
	form := NewForm()
	form.SetAllowMinimize(false)
	form.SetAllowMaximize(false)
	form.Panel().AddWidget(0, 0, centralWidget)
	form.OnClose = func() bool {
		if w, ok := centralWidget.(Dialoger); ok {
			return w.onDialogReject()
		}
		return true
	}
	// The show handler sets the size, title and focus: it runs before the window
	// is created, so the dialog appears at once as it should, without resizing
	// (and no other goroutine handles the dialog yet)
	form.parentForm = c.form
	if w, ok := centralWidget.(Dialoger); ok {
		w.onDialogShow()
	}
	form.ShowModal(c.form)
}
