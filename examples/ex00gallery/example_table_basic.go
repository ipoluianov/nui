package ex00gallery

import "github.com/ipoluianov/nui/ui"

type ExampleTableBasic struct {
	ui.Widget
}

func NewExampleTableBasic() *ExampleTableBasic {
	var c ExampleTableBasic
	c.InitWidget()

	lbl := c.AddLabel(0, 0, "A basic read-only table with a few rows of static data")
	lbl.SetUnderline(true)

	table := ui.NewTable()
	table.SetColumnCount(3)
	table.SetColumnName(0, "Name")
	table.SetColumnName(1, "Age")
	table.SetColumnName(2, "City")
	table.SetColumnWidth(0, 200)
	table.SetColumnWidth(1, 100)
	table.SetColumnWidth(2, 200)

	rows := [][3]string{
		{"Alice", "30", "New York"},
		{"Bob", "25", "Los Angeles"},
		{"Charlie", "35", "Chicago"},
		{"Diana", "28", "Houston"},
		{"Evan", "40", "Phoenix"},
	}
	table.SetRowCount(len(rows))
	for row, r := range rows {
		table.SetCellText2(row, 0, r[0])
		table.SetCellText2(row, 1, r[1])
		table.SetCellText2(row, 2, r[2])
	}

	c.AddWidget(1, 0, table)

	return &c
}
