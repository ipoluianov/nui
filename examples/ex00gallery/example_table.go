package ex00gallery

import "github.com/ipoluianov/nui/ui"

// ExamplePageTable is the gallery's "Table" entry. Table has enough distinct
// features (editing, selection modes, custom cell drawing) that a single
// page would be cluttered, so this hosts its own nested TabWidget with one
// Table demo per sub-tab instead.
type ExamplePageTable struct {
	ui.Widget

	tabWidget *ui.TabWidget
}

func NewExamplePageTable() *ExamplePageTable {
	var c ExamplePageTable
	c.InitWidget()

	c.tabWidget = ui.NewTabWidget()
	c.AddWidget(0, 0, c.tabWidget)

	c.tabWidget.AddPage("Basic", NewExampleTableBasic())
	c.tabWidget.AddPage("Editable", NewExampleTableEditable())
	c.tabWidget.AddPage("Selection", NewExampleTableSelection())
	c.tabWidget.AddPage("Custom Cells", NewExampleTableCustomCells())

	return &c
}
