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

// RunInParent runs f after the current handler of the dialog returns, then
// updates the window the dialog was shown over. Use it for the dialog's result
// callbacks: they run once the dialog is closed, and the parent is repainted
// with the changes they made (see Form.Invoke).
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

// ShowDialog shows the widget, a DialogContent mostly, in a modal window
// over the widget's form and returns at once. The window is sized in the
// dialog's OnDialogShow (Form().SetSize); without that, it is as large as
// the dialog's widgets need.
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
	form.parentForm = c.form
	form.width, form.height = 0, 0
	if w, ok := centralWidget.(Dialoger); ok {
		w.onDialogShow()
	}
	// A dialog that set no size is as large as its widgets need, and not
	// narrower than dialogMinWidth
	if form.width == 0 && form.height == 0 {
		form.growToMinSize()
		form.width = max(form.width, dialogMinWidth)
	}
	form.ShowModal(c.form)
}

// dialogMinWidth is the width of a dialog that set no size, at least
const dialogMinWidth = 320
