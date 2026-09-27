package ui

import ()

type RadioButton struct {
	Widget
	checked        bool
	text           string
	onStateChanged func(btn *RadioButton, checked bool)
}

func NewRadioButton(text string) *RadioButton {
	var c RadioButton
	c.InitWidget()
	c.SetTypeName("RadioButton")
	c.SetMinWidth(150)
	c.SetMaxWidth(10000)
	c.setThemeHeight(ThemeControlHeight, true)
	c.SetMouseCursor(MouseCursorPointer)
	c.SetText("RadioButton")
	c.SetCanBeFocused(true)

	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.buttonProcessMouseDown)
	c.SetOnMouseUp(c.buttonProcessMouseUp)

	c.SetText(text)

	return &c
}

func (c *RadioButton) Text() string {
	return c.text
}

func (c *RadioButton) SetText(text string) {
	c.text = text
	c.form.Update()
}

func (c *RadioButton) SetOnStateChanged(fn func(btn *RadioButton, checked bool)) {
	c.onStateChanged = fn
}

func (c *RadioButton) SetChecked(checked bool) {
	if c.checked == checked {
		return
	}

	if checked {
		// Uncheck all other radio buttons in the same group
		parentWidget := c.ParentWidget()
		if parentWidget != nil {
			for _, child := range parentWidget.Widgets() {
				if radioButton, ok := child.(*RadioButton); ok && radioButton.Id() != c.Id() {
					radioButton.SetChecked(false)
				}
			}
		}
	}

	c.checked = checked
	if c.onStateChanged != nil {
		c.onStateChanged(c, checked)
	}
}

func (c *RadioButton) Checked() bool {
	return c.checked
}

func (c *RadioButton) draw(cnv *Canvas) {
	size := ThemeIndicatorSize()
	padding := (c.Height() - size) / 2
	textX := padding + size + indicatorTextGap

	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(indicatorTextColor(&c.Widget))
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.DrawText(textX, 0, c.Width()-textX, c.Height(), c.text)

	// A circle; checked - an accent circle with a dot in the middle
	fill, border, mark := indicatorColors(&c.Widget, c.checked)
	cnv.FillFrame(padding, padding, size, size, size/2, fill, border)
	if c.checked {
		dot := size * 2 / 5
		offset := padding + (size-dot)/2
		cnv.FillRoundedRectAA(offset, offset, dot, dot, dot/2, mark)
	}
}

func (c *RadioButton) buttonProcessMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	return true
}

func (c *RadioButton) buttonProcessMouseUp(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if x < 0 || x >= c.Width() || y < 0 || y >= c.Height() {
		// MouseUp outside the button area, ignore
		return false
	}

	hoverWidgeter := c.form.hoverWidget
	var localWidgeter Widgeter = c
	if hoverWidgeter == localWidgeter {
		if !c.checked {
			c.SetChecked(true)
		}
	}

	return true
}
