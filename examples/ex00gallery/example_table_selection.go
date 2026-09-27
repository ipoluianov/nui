package ex00gallery

import (
	"fmt"
	"strings"

	"github.com/ipoluianov/nui/ui"
)

type ExampleTableSelection struct {
	ui.Widget

	lblStatus    *ui.Label
	lblSelection *ui.Label
	table        *ui.Table
}

func NewExampleTableSelection() *ExampleTableSelection {
	var c ExampleTableSelection
	c.InitWidget()

	c.lblStatus = c.AddLabel(0, 0, "Click a cell/row to select it. With Multiselect on: drag, Shift+click, Ctrl+click and Ctrl+A work as usual")
	c.lblStatus.SetUnderline(true)

	cbMultiselect := ui.NewCheckbox("Multiselect")
	c.AddWidget(1, 0, cbMultiselect)

	rbSelectRows := ui.NewRadioButton("Select whole rows")
	rbSelectRows.SetChecked(true)
	c.AddWidget(1, 1, rbSelectRows)

	rbSelectCells := ui.NewRadioButton("Select individual cells")
	c.AddWidget(1, 2, rbSelectCells)

	c.lblSelection = c.AddLabel(2, 0, "")

	c.table = ui.NewTable()
	c.table.SetColumnCount(3)
	c.table.SetColumnName(0, "Product")
	c.table.SetColumnName(1, "Price")
	c.table.SetColumnName(2, "In Stock")
	c.table.SetColumnWidth(0, 200)
	c.table.SetColumnWidth(1, 120)
	c.table.SetColumnWidth(2, 120)

	cbMultiselect.SetOnStateChanged(func() { c.table.SetMultiselect(cbMultiselect.Checked()); c.updateSelectionLabel() })
	rbSelectRows.SetOnStateChanged(func(btn *ui.RadioButton, checked bool) {
		if checked {
			c.table.SetSelectingRows(true)
			c.updateSelectionLabel()
		}
	})
	rbSelectCells.SetOnStateChanged(func(btn *ui.RadioButton, checked bool) {
		if checked {
			c.table.SetSelectingRows(false)
			c.updateSelectionLabel()
		}
	})

	rows := [][3]string{
		{"Keyboard", "49.99", "Yes"},
		{"Mouse", "19.99", "Yes"},
		{"Monitor", "199.99", "No"},
		{"Webcam", "39.99", "Yes"},
		{"Headset", "59.99", "Yes"},
		{"Microphone", "29.99", "No"},
	}
	c.table.SetRowCount(len(rows))
	for row, r := range rows {
		c.table.SetCellText2(row, 0, r[0])
		c.table.SetCellText2(row, 1, r[1])
		c.table.SetCellText2(row, 2, r[2])
	}

	c.table.SetOnSelectionChanged(func(row, col int) {
		c.updateSelectionLabel()
	})

	c.AddWidget(3, 0, c.table)

	c.updateSelectionLabel()

	return &c
}

func (c *ExampleTableSelection) updateSelectionLabel() {
	if c.table.SelectingRows() {
		rows := c.table.SelectedRows()
		parts := make([]string, len(rows))
		for i, r := range rows {
			parts[i] = fmt.Sprintf("%d", r)
		}
		c.setStatus(fmt.Sprintf("Selected rows (%d): %s", len(rows), strings.Join(parts, ", ")))
		return
	}

	cells := c.table.SelectedCells()
	parts := make([]string, len(cells))
	for i, cell := range cells {
		parts[i] = fmt.Sprintf("(%d,%d)", cell.Row, cell.Col)
	}
	c.setStatus(fmt.Sprintf("Selected cells (%d): %s", len(cells), strings.Join(parts, ", ")))
}

func (c *ExampleTableSelection) setStatus(text string) {
	c.lblSelection.SetText(text)
}
