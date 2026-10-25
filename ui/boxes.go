package ui

import "runtime"

// Layouts over the grid: a row or a column of widgets (HBox, VBox), the
// rows of labeled fields (FormLayout) and the buttons of a dialog
// (DialogButtonBox). Each is a Panel: it goes into a grid cell as any
// widget, and holds widgets of any kind, other boxes too.
//
//	row := ui.NewHBox()
//	row.Add(ui.NewLabel("Find:"))
//	row.Add(findBox) // an expandable widget takes the free room
//	row.AddSpacing(12)
//	row.Add(findButton)

// HBox lays widgets out in a row, left to right
type HBox struct {
	Panel
	count int
}

// NewHBox makes an empty row, without the padding around (the spacing
// between the widgets is the panel's, see SetCellPadding)
func NewHBox() *HBox {
	var c HBox
	c.InitWidget()
	c.SetTypeName("HBox")
	c.SetPanelPadding(0)
	return &c
}

// Add puts the widget at the end of the row and returns it
func (c *HBox) Add(w Widgeter) Widgeter {
	c.AddWidget(0, c.count, w)
	c.count++
	return w
}

// AddStretch puts an empty expandable space: the widgets after it go to the
// right edge
func (c *HBox) AddStretch() {
	c.Add(NewHSpacer())
}

// AddSpacing puts an empty space of the width
func (c *HBox) AddSpacing(width int) {
	c.Add(fixedSpace(width, 0))
}

// VBox lays widgets out in a column, top to bottom
type VBox struct {
	Panel
	count int
}

// NewVBox makes an empty column, without the padding around
func NewVBox() *VBox {
	var c VBox
	c.InitWidget()
	c.SetTypeName("VBox")
	c.SetPanelPadding(0)
	return &c
}

// Add puts the widget at the bottom of the column and returns it
func (c *VBox) Add(w Widgeter) Widgeter {
	c.AddWidget(c.count, 0, w)
	c.count++
	return w
}

// AddStretch puts an empty expandable space: the widgets after it go to the
// bottom edge
func (c *VBox) AddStretch() {
	c.Add(NewVSpacer())
}

// AddSpacing puts an empty space of the height
func (c *VBox) AddSpacing(height int) {
	c.Add(fixedSpace(0, height))
}

// fixedSpace is an empty widget of the size
func fixedSpace(width, height int) Widgeter {
	p := NewPanel()
	p.SetMinSize(width, height)
	p.SetMaxSize(width, height)
	return p
}

// FormLayout lays out the fields of a form: a label and its field a row,
// the labels in a column of their own, the fields taking the rest of the
// width (QFormLayout)
//
//	form := ui.NewFormLayout()
//	form.AddRow("Name:", nameBox)
//	form.AddRow("Port:", portBox)
//	form.AddRowWidget(rememberCheckbox) // the whole width
type FormLayout struct {
	Panel
	rows int
}

// NewFormLayout makes an empty form layout
func NewFormLayout() *FormLayout {
	var c FormLayout
	c.InitWidget()
	c.SetTypeName("FormLayout")
	c.SetPanelPadding(0)
	c.SetColumnStretch(1, 1)
	return &c
}

// AddRow adds a row of a label with the text and the field; it returns the
// label
func (c *FormLayout) AddRow(label string, field Widgeter) *Label {
	l := NewLabel(label)
	l.SetBuddy(field)
	c.AddWidget(c.rows, 0, l)
	c.AddWidget(c.rows, 1, field)
	c.rows++
	return l
}

// AddRowWidget adds a row of a widget taking the whole width
func (c *FormLayout) AddRowWidget(w Widgeter) {
	c.AddWidgetSpan(c.rows, 0, 1, 2, w)
	c.rows++
}

// RowCount returns the number of the rows
func (c *FormLayout) RowCount() int {
	return c.rows
}

// ButtonRole is what a button of a dialog does, see DialogButtonBox
type ButtonRole int

const (
	// ButtonAccept accepts the dialog: OK, Yes, Save. Enter presses it.
	ButtonAccept ButtonRole = iota
	// ButtonReject cancels the dialog: Cancel, Close. Esc presses it.
	ButtonReject
	// ButtonOther is any other button: it goes before the accept and the
	// reject ones
	ButtonOther
)

// DialogButtonBox is the row of the buttons of a dialog, at its right edge
// in the order of the platform: OK Cancel on Windows and Linux, Cancel OK
// on macOS. In a form it makes its accept button the form's accept button
// (Enter) and its reject button the cancel button (Esc).
//
//	buttons := ui.NewDialogButtonBox()
//	buttons.AddOK(func() { apply(); c.Form().Close() })
//	buttons.AddCancel(func() { c.Form().Close() })
//	dialog.AddWidget(2, 0, buttons)
type DialogButtonBox struct {
	Panel
	buttons []dialogButton
}

type dialogButton struct {
	button *Button
	role   ButtonRole
}

// NewDialogButtonBox makes an empty button box
func NewDialogButtonBox() *DialogButtonBox {
	var c DialogButtonBox
	c.InitWidget()
	c.SetTypeName("DialogButtonBox")
	c.SetPanelPadding(0)
	return &c
}

// AddButton adds a button with the role
func (c *DialogButtonBox) AddButton(text string, role ButtonRole, onClick func()) *Button {
	b := NewButton(text)
	b.SetOnClick(onClick)
	c.buttons = append(c.buttons, dialogButton{b, role})
	c.arrange()
	return b
}

// AddOK adds the accept button "OK", in the language of the interface
func (c *DialogButtonBox) AddOK(onClick func()) *Button {
	b := c.AddButton("", ButtonAccept, onClick)
	b.SetTextFunc(func() string { return UIText().OK })
	return b
}

// AddCancel adds the reject button "Cancel", in the language of the
// interface
func (c *DialogButtonBox) AddCancel(onClick func()) *Button {
	b := c.AddButton("", ButtonReject, onClick)
	b.SetTextFunc(func() string { return UIText().Cancel })
	return b
}

// Button returns the first button of the role, nil if none
func (c *DialogButtonBox) Button(role ButtonRole) *Button {
	for _, b := range c.buttons {
		if b.role == role {
			return b.button
		}
	}
	return nil
}

// arrange lays the buttons out in the order of the platform
func (c *DialogButtonBox) arrange() {
	c.RemoveAllWidgets()
	order := []ButtonRole{ButtonOther, ButtonAccept, ButtonReject}
	if runtime.GOOS == "darwin" {
		order = []ButtonRole{ButtonOther, ButtonReject, ButtonAccept}
	}
	c.AddHSpacer(0, 0)
	col := 1
	for _, role := range order {
		for _, b := range c.buttons {
			if b.role == role {
				c.AddWidget(0, col, b.button)
				col++
			}
		}
	}
	c.bindToForm()
}

// bindToForm makes the accept and the reject buttons the form's
func (c *DialogButtonBox) bindToForm() {
	if c.form == nil {
		return
	}
	if b := c.Button(ButtonAccept); b != nil {
		c.form.SetAcceptButton(b)
	}
	if b := c.Button(ButtonReject); b != nil {
		c.form.SetCancelButton(b)
	}
}

func (c *DialogButtonBox) attachToForm(self Widgeter, form *Form) {
	c.Panel.attachToForm(self, form)
	c.bindToForm()
}
