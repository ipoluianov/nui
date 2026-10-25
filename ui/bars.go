package ui

import "time"

// ToolBar is a row of flat buttons, of actions mostly:
//
//	toolbar := ui.NewToolBar()
//	toolbar.AddAction(newAction)
//	toolbar.AddAction(saveAction)
//	toolbar.AddSeparator()
//	toolbar.AddAction(boldAction) // checkable: shown pressed when checked
//	form.Panel().AddWidget(0, 0, toolbar)
type ToolBar struct {
	HBox
	buttonSize int
	end        *HSpacer // the free width, after the last widget
}

// toolBarButtonSize is the side of the buttons at the theme's font size
func toolBarButtonSize() int {
	return ThemeControlHeight() + 2
}

// NewToolBar makes an empty toolbar
func NewToolBar() *ToolBar {
	var c ToolBar
	c.InitWidget()
	c.SetTypeName("ToolBar")
	c.SetPanelPadding(0)
	c.SetCellPadding(2)
	c.buttonSize = toolBarButtonSize()
	// The free width stays at the end: the buttons keep their size
	c.end = NewHSpacer()
	c.AddWidget(0, 0, c.end)
	return &c
}

// Add puts the widget at the end of the toolbar and returns it
func (c *ToolBar) Add(w Widgeter) Widgeter {
	c.end.SetGridPosition(0, c.count+1)
	c.AddWidget(0, c.count, w)
	c.count++
	return w
}

// AddStretch puts an expandable space: the widgets after it go to the
// right edge
func (c *ToolBar) AddStretch() {
	c.Add(NewHSpacer())
}

// AddSpacing puts an empty space of the width
func (c *ToolBar) AddSpacing(width int) {
	c.Add(fixedSpace(width, 0))
}

// AddAction adds a button of the action: its icon (its text if it has
// none), its tooltip, enabled and checked state, following its changes; a
// click runs it
func (c *ToolBar) AddAction(a *Action) Widgeter {
	if a.icon == nil {
		b := NewButton("")
		b.SetOnClick(a.Trigger)
		sync := func(*Action) {
			text, _, _ := parseMnemonic(a.text)
			b.SetText(text)
			b.SetTooltip(a.Tooltip())
			b.SetEnabled(a.enabled)
			// A checked one in the accent color, as a pressed tool button
			if a.checkable && a.checked {
				b.SetRole("primary")
			} else {
				b.SetRole("")
			}
		}
		sync(a)
		a.OnChanged.Connect(sync)
		return c.Add(b)
	}
	b := NewToolButton(a.icon, a.Tooltip(), a.Trigger)
	b.SetFlat(true)
	b.SetButtonSize(c.buttonSize, c.buttonSize)
	sync := func(*Action) {
		b.SetImage(a.icon)
		b.SetTooltip(a.Tooltip())
		b.SetEnabled(a.enabled)
		b.SetChecked(a.checkable && a.checked)
	}
	sync(a)
	a.OnChanged.Connect(sync)
	return c.Add(b)
}

// AddSeparator adds a gap between groups of buttons
func (c *ToolBar) AddSeparator() {
	c.AddSpacing(c.buttonSize / 3)
}

// StatusBar is the line at the bottom of a window: a text at the left, the
// sections added at the right (a position, a zoom, a progress bar), and
// messages shown for a while over the text
//
//	status := ui.NewStatusBar()
//	position := status.AddLabel("Ln 1, Col 1")
//	status.ShowMessage("Saved", 3*time.Second)
//	form.Panel().AddWidget(2, 0, status)
type StatusBar struct {
	HBox
	main    *Label
	text    string
	message *Timer
	// showing is set while a message is in the place of the text
	showing bool
}

// NewStatusBar makes a status bar with an empty text
func NewStatusBar() *StatusBar {
	var c StatusBar
	c.InitWidget()
	c.SetTypeName("StatusBar")
	c.SetPanelPadding(0)
	c.SetCellPadding(12)
	c.main = NewLabel("")
	c.main.SetXExpandable(true)
	c.Add(c.main)
	return &c
}

// SetText sets the text at the left
func (c *StatusBar) SetText(text string) {
	c.text = text
	if !c.showing {
		c.main.SetText(text)
	}
}

func (c *StatusBar) Text() string {
	return c.text
}

// ShowMessage shows the message in the place of the text for the duration
// (until the next message or ClearMessage for 0); then the text is back
func (c *StatusBar) ShowMessage(message string, d time.Duration) {
	if c.message != nil {
		c.message.Stop()
	}
	c.showing = true
	c.main.SetText(message)
	if d > 0 {
		c.message = c.AfterFunc(d, c.ClearMessage)
	}
}

// ClearMessage shows the text again in the place of a message
func (c *StatusBar) ClearMessage() {
	if c.message != nil {
		c.message.Stop()
	}
	c.showing = false
	c.main.SetText(c.text)
}

// AddLabel adds a section with the text at the right; set its text later
// with the label's SetText
func (c *StatusBar) AddLabel(text string) *Label {
	l := NewLabel(text)
	c.Add(l)
	return l
}
