package ex00gallery

import (
	"image/color"

	"github.com/ipoluianov/nui/ui"
)

type ExampleTableCustomCells struct {
	ui.Widget
}

func NewExampleTableCustomCells() *ExampleTableCustomCells {
	var c ExampleTableCustomCells
	c.InitWidget()

	lbl := c.AddLabel(0, 0, "Cells can have custom colors, alignment, images and truncate long text")
	lbl.SetUnderline(true)

	table := ui.NewTable()
	table.SetColumnCount(4)
	table.SetColumnName(0, "Icon")
	table.SetColumnName(1, "Status")
	table.SetColumnName(2, "Description")
	table.SetColumnName(3, "Amount")
	table.SetColumnWidth(0, 70)
	table.SetColumnWidth(1, 120)
	table.SetColumnWidth(2, 220)
	table.SetColumnWidth(3, 120)

	sampleIcon := newSampleImage(32, 32)

	rows := []struct {
		status string
		color  color.RGBA
		desc   string
		amount string
	}{
		{"OK", color.RGBA{R: 60, G: 180, B: 80, A: 255}, "Everything is running as expected, no action needed", "120.00"},
		{"Warning", color.RGBA{R: 220, G: 160, B: 40, A: 255}, "Disk usage is approaching the configured threshold", "0.00"},
		{"Error", color.RGBA{R: 210, G: 60, B: 60, A: 255}, "Background job failed after three retries", "-45.50"},
	}

	table.SetRowCount(len(rows))
	for row, r := range rows {
		table.SetCellImage(row, 0, sampleIcon, 24)

		table.SetCellText2(row, 1, r.status)
		table.SetCellColor(row, 1, r.color)

		table.SetCellText2(row, 2, r.desc)
		table.SetCellContraction(row, 2, true)

		table.SetCellText2(row, 3, r.amount)
		table.SetCellHAlign(row, 3, ui.HAlignRight)
	}

	c.AddWidget(1, 0, table)

	return &c
}
