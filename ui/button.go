package ui

import (
	"image/color"
)

type Button struct {
	Widget
	pressed bool

	// textMinWidth is the minimum width updateSizeFromText set last, to
	// tell it from a minimum width set explicitly
	textMinWidth int
}

const (
	DefaultButtonMinWidth = 100
)

func NewButton(text string) *Button {
	var c Button
	c.InitWidget()

	c.dontAllowOnClickIfDisabled = true

	c.SetTypeName("Button")
	c.SetMinWidth(DefaultButtonMinWidth)
	c.setThemeHeight(c.heightForText, false)
	c.SetMouseCursor(MouseCursorPointer)
	c.SetText("Button")
	c.SetCanBeFocused(true)
	c.SetElevation(3)

	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.buttonProcessMouseDown)
	c.SetOnMouseUp(c.buttonProcessMouseUp)
	// c.SetOnKeyDown(c.onKeyDown)

	c.SetText(text)
	c.SetProp("padding", 6)
	// SetProp only dispatches ProcessPropChange (which would normally grow
	// the button to fit this text) once the button is attached to a form -
	// c.form is still nil here, during construction, so that dispatch is a
	// no-op. Size from the text directly so the very first, most natural
	// construction path (NewButton(text) then AddWidget(...)) is sized
	// correctly too, not just a later SetText call made after attaching.
	c.updateSizeFromText()

	return &c
}

func (c *Button) Text() string {
	return c.GetPropString("text", "")
}

func (c *Button) SetText(text string) {
	c.SetProp("text", text)
	c.form.Update()
}

// Push simulates a click on the button, triggering the "onclick" handler
// programmatically (used e.g. by Form's accept/cancel buttons on Enter/Esc).
func (c *Button) Push() {
	if !c.Enabled() {
		return
	}
	if f := c.GetPropFunction("onclick"); f != nil {
		f()
	}
}

func (c *Button) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	return c.onKeyDown(key, mods)
}

func (c *Button) onKeyDown(key Key, mods KeyModifiers) bool {
	if key == KeyEnter || key == KeySpace {
		if f := c.GetPropFunction("onclick"); f != nil {
			f()
			return true
		}
	}
	return false
}

func (c *Button) draw(cnv *Canvas) {
	p := CurrentPalette()
	primary := c.Role() == "primary" || c.Role() == "secondary"

	fill, border := p.Button, p.Border
	if primary {
		fill, border = p.Highlight, p.Highlight
	}
	if c.backgroundColor != nil {
		fill = colorToRGBA(c.backgroundColor)
	}
	textColor := colorToRGBA(c.ForegroundColor())

	switch {
	case !c.Enabled():
		textColor = p.DisabledText
		if primary {
			fill = MixColors(fill, p.Window, 0.5)
			border = fill
		}
	case c.pressed && primary:
		// Towards the text would lighten the accent; a pressed accent darkens
		fill = MixColors(fill, color.RGBA{A: 255}, 0.15)
	case c.pressed:
		fill = pressedColor(fill, textColor)
	case c.IsHovered():
		fill = hoverColor(fill, textColor)
	}

	if c.IsFocused() && c.Enabled() {
		border = p.Highlight
		if primary {
			border = MixColors(p.Highlight, p.WindowText, 0.45)
		}
	}

	cnv.FillFrame(0, 0, c.Width(), c.Height(), themeControlRadius, fill, border)

	cnv.SetHAlign(HAlignCenter)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(textColor)
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.DrawText(0, 0, c.Width(), c.Height(), c.Text())
}

func (c *Button) ProcessPropChange(key string, value interface{}) {
	if key == "enabled" {
		if c.Enabled() {
			c.SetMouseCursor(MouseCursorPointer)
		} else {
			c.SetMouseCursor(MouseCursorNotDefined)
		}
	}

	c.updateSizeFromText()
	c.form.UpdateLayout()
}

// updateSizeFromText grows the button's min size to fit its current text.
// Compares against c.minWidth directly rather than the cached MinWidth()
// getter, since SetMinWidth doesn't invalidate that cache - reading it here
// (e.g. on the very first call, from NewButton before any layout pass has
// run) would otherwise lock in a stale value once a real layout pass reads
// MinWidth() later.
func (c *Button) updateSizeFromText() {
	padding := c.GetPropInt("padding", 6)

	textWidth, _, err := MeasureText(c.FontFamily(), c.FontSize(), c.Text())
	if err != nil {
		return
	}

	c.applyThemeHeight()

	// Fit the text; a minimum width set explicitly can only grow
	minWidth := max(DefaultButtonMinWidth, textWidth+padding*2)
	if c.minWidth == c.textMinWidth || minWidth > c.minWidth {
		c.SetMinWidth(minWidth)
		c.textMinWidth = minWidth
	}
}

func (c *Button) buttonProcessMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if !c.Enabled() {
		return false
	}
	c.pressed = true
	return true
}

func (c *Button) buttonProcessMouseUp(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if !c.Enabled() {
		return false
	}

	wasPressed := c.pressed
	c.pressed = false

	if x < 0 || x >= c.Width() || y < 0 || y >= c.Height() {
		// Released outside the button - cancel the click.
		return false
	}

	if wasPressed {
		if f := c.GetPropFunction("onclick"); f != nil {
			f()
		}
	}

	return true
}

// heightForText is the button's height: a control's height, or more for a
// larger font set on the button.
func (c *Button) heightForText() int {
	_, textHeight, err := MeasureText(c.FontFamily(), c.FontSize(), "Ag")
	if err != nil {
		return ThemeControlHeight()
	}
	return max(ThemeControlHeight(), textHeight+controlPaddingY*2)
}

func (c *Button) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.updateSizeFromText()
}
