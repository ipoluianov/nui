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
	c.setThemeHeight(c.heightForText, false)
	c.updateInnerSize()
	return &c
}

// heightForText is the label's height: a control's height, or more for a
// larger font or more lines (big text used to be cut off)
func (c *Label) heightForText() int {
	_, textHeight, err := MeasureText(c.FontFamily(), c.FontSize(), "Ag")
	if err != nil {
		return ThemeControlHeight()
	}
	lines := len(textLines(c.Text()))
	return max(ThemeControlHeight(), textHeight*lines+controlPaddingY*2)
}

// Paint handler for the Label widget
func (c *Label) onPaint(cnv *Canvas) {
	// Draw the label text
	cnv.SetHAlign(c.GetHAlign("align", HAlignLeft))
	cnv.SetVAlign(VAlignCenter)
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.SetColor(c.ForegroundColor())
	// Under each line of the text, wherever the text is (not at the height
	// of a control: a taller label has it lower)
	cnv.SetUnderline(c.GetPropBool("underline", false))
	cnv.DrawText(0, 0, c.Width(), c.Height(), c.GetPropString("text", ""))
	cnv.SetUnderline(false)
}

// ///////////////////////////////////////////////////////////////////////////////
// Props
func (c *Label) ProcessPropChange(key string, value interface{}) {
	// A new minimum size lays the form out again (see SetMinWidth); a text
	// that fits in the same room only repaints
	c.updateInnerSize()
	if key != "text" {
		c.form.UpdateLayout()
	}
	c.Update()
}

/////////////////////////////////////////////////////////////////////////////////
// Label methods

// Text returns the text of the Label.
func (c *Label) Text() string {
	return c.GetPropString("text", "")
}

// SetText sets the text of the Label.
func (c *Label) SetText(text string) {
	if text == c.Text() && c.GetProp("text") != nil {
		return
	}
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
	if c.themeHeight != nil {
		height := c.minHeight
		c.applyThemeHeight()
		c.innerHeight = c.minHeight
		if c.minHeight != height {
			c.form.UpdateLayout()
		}
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
	c.updateInnerSize()
}

// SetBuddy makes the label the name of the widget for a screen reader: a
// text field without a text of its own is read as "Name: text field".
// FormLayout.AddRow does it for its rows.
func (c *Label) SetBuddy(w Widgeter) {
	if w != nil {
		w.widgetBase().accessibleLabel = c
	}
}
