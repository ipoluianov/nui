package ex00gallery

import "github.com/ipoluianov/nui/ui"

type Example struct {
	ui.Widget

	tabWidget *ui.TabWidget
}

func NewExample() *Example {
	var c Example
	c.InitWidget()

	c.tabWidget = ui.NewTabWidget()
	c.AddWidget(0, 0, c.tabWidget)

	c.tabWidget.AddPage("Label", NewExamplePageLabel())
	c.tabWidget.AddPage("TextEdit", NewExamplePageTextBox())
	c.tabWidget.AddPage("ContextMenu", NewExamplePageContextMenu())
	c.tabWidget.AddPage("Button", NewExamplePageButton())
	c.tabWidget.AddPage("CheckBox", NewExamplePageCheckbox())
	c.tabWidget.AddPage("ComboBox", NewExamplePageComboBox())
	c.tabWidget.AddPage("ImageBox", NewExamplePageImageBox())
	c.tabWidget.AddPage("NumBox", NewExamplePageNumBox())
	c.tabWidget.AddPage("ProgressBar", NewExamplePageProgressBar())
	c.tabWidget.AddPage("Table", NewExamplePageTable())

	return &c
}

func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetSize(1100, 900)
	form.Panel().AddWidget(0, 0, NewExample())
	return form
}
