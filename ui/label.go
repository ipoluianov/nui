package ui

type Label struct {
	Widget
}

/*
Properties:
- text: string - The text to display in the label.
- align: string - The horizontal alignment of the text ["left", "center", "right"].
- underline: bool - Whether the text is underlined.
*/

// It is the maximum width for a label when it is not expandable
// to avoid too large labels in case of long texts.
const labelMaxWidth = 1500

// //////////////////////////////////////////////////////////////////////////////////
// NewLabel creates a new Label widget with the specified text.
func NewLabel(text string) *Label {
	var c Label
	c.InitWidget()
	c.SetTypeName("Label")
	c.SetOnPaint(c.onPaint)
	c.SetText(text)
	c.innerHeight = ThemeControlHeight()
	c.setThemeHeight(ThemeControlHeight, false)
	c.updateInnerSize()
	return &c
}

// Paint handler for the Label widget
func (c *Label) onPaint(cnv *Canvas) {
	// Draw the label text
	cnv.SetHAlign(c.GetHAlign("align", HAlignLeft))
	cnv.SetVAlign(VAlignCenter)
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.SetColor(c.ForegroundColor())
	cnv.DrawText(0, 0, c.Width(), c.Height(), c.GetPropString("text", ""))

	// Draw underline if needed
	if c.GetPropBool("underline", false) {
		textWidth, _, err := MeasureText(c.FontFamily(), c.FontSize(), c.Text())
		if err == nil {
			cnv.SetColor(c.ForegroundColor())

			x := 0
			if c.TextAlign() == HAlignCenter {
				x = (c.Width() - textWidth) / 2
			} else if c.TextAlign() == HAlignRight {
				x = c.Width() - textWidth
			}

			textHeight := c.innerHeight
			cnv.DrawLine(x, textHeight-2, x+textWidth, textHeight-2, 1, c.ForegroundColor())
		}
	}
}

// ///////////////////////////////////////////////////////////////////////////////
// Props
func (c *Label) ProcessPropChange(key string, value interface{}) {
	c.updateInnerSize()
	c.form.UpdateLayout()
}

/////////////////////////////////////////////////////////////////////////////////
// Label methods

// Text returns the text of the Label.
func (c *Label) Text() string {
	return c.GetPropString("text", "")
}

// SetText sets the text of the Label.
func (c *Label) SetText(text string) {
	c.SetProp("text", text)
}

// TextAlign returns the horizontal alignment of the Label's text.
func (c *Label) TextAlign() HAlign {
	return c.GetHAlign("align", HAlignLeft)
}

// SetTextAlign sets the horizontal alignment of the Label's text.
func (c *Label) SetTextAlign(align HAlign) {
	c.SetProp("align", align.String())
}

// IsUnderline returns whether the Label's text is underlined.
func (c *Label) IsUnderline() bool {
	return c.GetPropBool("underline", false)
}

// SetUnderline sets whether the Label's text is underlined.
func (c *Label) SetUnderline(underline bool) {
	c.SetProp("underline", underline)
}

/////////////////////////////////////////////////////////////////////////////////
// Label private methods

// updateInnerSize updates the inner size of the Label based on its text.
func (c *Label) updateInnerSize() {
	textWidth, err := measureMultilineTextWidth(c.FontFamily(), c.FontSize(), c.Text())
	if err != nil {
		return
	}

	if !c.XExpandable() {
		// if not expandable, set inner width to text width
		// for example if in one row we have label + textbox
		// we want the label to be just wide enough for the text
		// so that the textbox gets the rest of the space
		c.innerWidth = textWidth
		if c.innerWidth > labelMaxWidth {
			c.innerWidth = labelMaxWidth
		}
		c.SetMinWidth(c.innerWidth)
	} else {
		// label can expand - if text is wider than real width - it will be clipped
		// it is useful when label is in a column and where are no other widgets in the same row
		c.innerWidth = labelMaxWidth
		c.SetMinWidth(10)
	}
}

func (c *Label) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.innerHeight = ThemeControlHeight()
	c.updateInnerSize()
}
