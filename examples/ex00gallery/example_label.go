package ex00gallery

import "github.com/ipoluianov/nui/ui"

type ExamplePageLabel struct {
	ui.Widget
}

func NewExamplePageLabel() *ExamplePageLabel {
	var c ExamplePageLabel
	c.InitWidget()

	row := 0

	row = addSectionHeader(&c.Widget, row, "Basic label")
	c.AddLabel(row, 0, "This is a simple label")
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Text alignment")
	c.addAlignmentDemo(row)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Underline")
	lblUnderline := c.AddLabel(row, 0, "This label is underlined")
	lblUnderline.SetUnderline(true)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Font size")
	c.addFontSizeDemo(row)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Font family")
	c.AddLabel(row, 0, "Noto Sans (default): 0123456789 Illegal1 WWW iii")
	row++
	lblMono := c.AddLabel(row, 0, "JetBrains Mono: 0123456789 Illegal1 WWW iii")
	lblMono.SetFontFamily(ui.FontFamilyMono)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Foreground color")
	c.addColorDemo(row)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Background color")
	c.addBackgroundColorDemo(row)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Multiline text")
	// Canvas.DrawText only breaks lines on "\r\n" (not plain "\n"), and Label
	// does not grow its own height automatically, so the min height has to
	// be set explicitly to fit all the lines.
	lblMultiline := c.AddLabel(row, 0, "First line\r\nSecond line\r\nThird line")
	lblMultiline.SetMinHeight(ui.DefaultUiLineHeight * 3)
	row++

	c.AddVSpacer(row, 0)

	return &c
}

// addAlignmentDemo shows the same label text left, center and right aligned inside fixed-width boxes.
func (c *ExamplePageLabel) addAlignmentDemo(row int) {
	aligns := []struct {
		name  string
		align ui.HAlign
	}{
		{"Left", ui.HAlignLeft},
		{"Center", ui.HAlignCenter},
		{"Right", ui.HAlignRight},
	}

	for col, a := range aligns {
		box := c.AddPanel(row, col)
		box.SetMinWidth(160)
		box.SetXExpandable(false)
		box.SetRole("base")
		box.SetAutoFillBackground(true)

		lbl := box.AddLabel(0, 0, a.name)
		lbl.SetXExpandable(true)
		lbl.SetTextAlign(a.align)
	}
}

// addFontSizeDemo shows the same label rendered with different font sizes.
func (c *ExamplePageLabel) addFontSizeDemo(row int) {
	sizes := []float64{12, ui.DefaultFontSize, 24, 32}
	for col, size := range sizes {
		lbl := c.AddLabel(row, col, "Aa")
		lbl.SetFontSize(size)
	}
}

// addColorDemo shows labels rendered with different foreground colors.
func (c *ExamplePageLabel) addColorDemo(row int) {
	colors := []string{"#E53935", "#43A047", "#1E88E5", "#FDD835"}
	for col, hex := range colors {
		lbl := c.AddLabel(row, col, hex)
		lbl.SetForegroundColor(ui.ColorFromHex(hex))
	}
}

// addBackgroundColorDemo shows labels rendered with different background colors.
func (c *ExamplePageLabel) addBackgroundColorDemo(row int) {
	colors := []string{"#E53935", "#43A047", "#1E88E5", "#FDD835"}
	for col, hex := range colors {
		lbl := c.AddLabel(row, col, hex)
		lbl.SetBackgroundColor(ui.ColorFromHex(hex))
		lbl.SetAutoFillBackground(true)
	}
}
