package ui

type ProgressBar struct {
	Widget
	text     string
	value    float64
	minValue float64
	maxValue float64
}

func NewProgressBar(minValue, maxValue, initValue float64) *ProgressBar {
	var c ProgressBar
	c.InitWidget()
	c.SetTypeName("ProgressBar")
	c.SetMinWidth(100)
	c.SetMaxWidth(10000)
	c.setThemeHeight(ThemeControlHeight, true)
	c.SetOnPaint(c.draw)

	c.minValue = minValue
	c.maxValue = maxValue
	c.value = initValue

	return &c
}

func (c *ProgressBar) Text() string {
	return c.text
}

func (c *ProgressBar) SetText(text string) {
	c.text = text
	c.form.Update()
}

func (c *ProgressBar) SetValue(value float64) {
	if value < c.minValue {
		value = c.minValue
	}
	if value > c.maxValue {
		value = c.maxValue
	}
	if c.value == value {
		return
	}
	c.value = value
	c.form.Update()
}

func (c *ProgressBar) Value() float64 {
	return c.value
}

func (c *ProgressBar) SetMinValue(minValue float64) {
	c.minValue = minValue
	c.form.Update()
}

func (c *ProgressBar) SetMaxValue(maxValue float64) {
	c.maxValue = maxValue
	c.form.Update()
}

func (c *ProgressBar) draw(cnv *Canvas) {
	percents := (c.value - c.minValue) / (c.maxValue - c.minValue)
	if percents < 0 {
		percents = 0
	}
	if percents > 1 {
		percents = 1
	}

	// A sunken track with the accent bar inside
	p := CurrentPalette()
	cnv.FillFrame(0, 0, c.Width(), c.Height(), themeControlRadius, p.Base, p.Border)
	barWidth := int(float64(c.Width()-2) * percents)
	if barWidth > 0 {
		cnv.FillRoundedRectAA(1, 1, barWidth, c.Height()-2, themeControlRadius-1, p.Highlight)
	}

	if len(c.text) > 0 {
		cnv.SetHAlign(HAlignCenter)
		cnv.SetVAlign(VAlignCenter)
		cnv.SetColor(c.ForegroundColor())
		cnv.SetFontFamily(c.FontFamily())
		cnv.SetFontSize(c.FontSize())
		cnv.DrawText(0, 0, c.Width(), c.Height(), c.text)
	}
}
