package ui

import ()

type tabWidgetButton struct {
	Widget

	pressed       bool
	text          string
	onButtonClick func(btn *tabWidgetButton)
}

func NewTabWidgetButton(text string) *tabWidgetButton {
	var c tabWidgetButton
	c.InitWidget()
	c.SetTypeName("TabWidgetButton")
	c.SetMinWidth(100)
	c.SetMaxWidth(10000)
	c.setThemeHeight(ThemeControlHeight, true)
	c.SetMouseCursor(MouseCursorPointer)
	c.SetText("Button")

	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.buttonProcessMouseDown)
	c.SetOnMouseUp(c.buttonProcessMouseUp)

	c.SetText(text)

	c.SetCanBeFocused(true)

	return &c
}

func (c *tabWidgetButton) Text() string {
	return c.text
}

func (c *tabWidgetButton) SetText(text string) {
	c.text = text
	c.form.Update()
}

func (c *tabWidgetButton) SetOnButtonClick(fn func(btn *tabWidgetButton)) {
	c.onButtonClick = fn
}

func (c *tabWidgetButton) draw(cnv *Canvas) {
	cnv.SetHAlign(HAlignCenter)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(c.ForegroundColor())
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.DrawText(0, 0, c.Width(), c.Height(), c.text)
}

func (c *tabWidgetButton) buttonProcessMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	c.pressed = true
	return true
}

func (c *tabWidgetButton) buttonProcessMouseUp(button MouseButton, x int, y int, mods KeyModifiers) bool {
	c.pressed = false

	if x < 0 || x >= c.Width() || y < 0 || y >= c.Height() {
		// MouseUp outside the button area, ignore
		return false
	}

	hoverWidgeter := c.form.hoverWidget
	var localWidgeter Widgeter = c
	if hoverWidgeter == localWidgeter {
		if c.onButtonClick != nil {
			c.onButtonClick(c)
		}
	}

	return true
}
