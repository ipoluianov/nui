package ex06timechart

import "github.com/ipoluianov/nui/ui"

type Example struct {
	ui.Widget

	tabWidget *ui.TabWidget
}

func NewExample() *Example {
	var c Example
	c.InitWidget()

	// Expandable labels only require 10 px, so a long text is clipped
	// instead of making the whole column (and the chart) at least as wide
	// as the text. The text is set after SetXExpandable because a detached
	// Label keeps the min width computed when it was created.
	top := c.AddPanel(0, 0)
	top.SetPanelPadding(0)
	hint := top.AddLabel(0, 0, "")
	hint.SetXExpandable(true)
	hint.SetText("Left drag - pan, right drag right - zoom time, Ctrl + right drag right - zoom rectangle, right drag left or double click - zoom back, Esc - default zoom, Shift + left drag - select, wheel - zoom at pointer")

	light := ui.NewCheckbox("Light theme")
	light.SetChecked(!ui.IsDarkTheme)
	light.SetOnStateChanged(func() {
		if light.Checked() {
			ui.ApplyLightTheme()
		} else {
			ui.ApplyDarkTheme()
		}
		c.Form().Update()
	})
	top.AddWidget(0, 1, light)

	c.tabWidget = ui.NewTabWidget()
	c.AddWidget(1, 0, c.tabWidget)

	c.tabWidget.AddPage("Ping", NewPagePing())
	c.tabWidget.AddPage("Jitter", NewPageJitter())
	c.tabWidget.AddPage("Candles", NewPageCandles())

	return &c
}

func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Example 06 - TimeChart")
	form.SetSize(1000, 700)
	form.Panel().AddWidget(0, 0, NewExample())
	return form
}
