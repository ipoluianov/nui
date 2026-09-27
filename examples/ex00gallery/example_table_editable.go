package ex00gallery

import (
	"fmt"

	"github.com/ipoluianov/nui/ui"
)

type ExampleTableEditable struct {
	ui.Widget

	lblStatus *ui.Label
}

func NewExampleTableEditable() *ExampleTableEditable {
	var c ExampleTableEditable
	c.InitWidget()

	c.lblStatus = c.AddLabel(0, 0, "Double-click, press Enter or F2 on a cell to edit it - empty values are rejected")
	c.lblStatus.SetUnderline(true)

	table := ui.NewTable()
	table.SetColumnCount(2)
	table.SetColumnName(0, "Item")
	table.SetColumnName(1, "Quantity")
	table.SetColumnWidth(0, 220)
	table.SetColumnWidth(1, 120)

	table.SetEditTriggerDoubleClick(true)
	table.SetEditTriggerEnter(true)
	table.SetEditTriggerF2(true)

	items := [][2]string{
		{"Apples", "12"},
		{"Bananas", "6"},
		{"Cherries", "150"},
	}
	table.SetRowCount(len(items))
	for row, r := range items {
		table.SetCellText2(row, 0, r[0])
		table.SetCellText2(row, 1, r[1])
	}

	table.SetOnCellChanged(func(row, col int, text string, data interface{}) bool {
		if len(text) == 0 {
			c.setStatus("Rejected: cell value cannot be empty")
			return false
		}
		c.setStatus(fmt.Sprintf("Cell (%d, %d) changed to %q", row, col, text))
		return true
	})

	c.AddWidget(1, 0, table)

	return &c
}

func (c *ExampleTableEditable) setStatus(text string) {
	c.lblStatus.SetText(text)
}
