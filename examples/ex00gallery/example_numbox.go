package ex00gallery

import (
	"fmt"

	"github.com/ipoluianov/nui/ui"
)

type ExamplePageNumBox struct {
	ui.Widget

	lblStatus *ui.Label
}

func NewExamplePageNumBox() *ExamplePageNumBox {
	var c ExamplePageNumBox
	c.InitWidget()

	row := 0

	c.lblStatus = c.AddLabel(row, 0, "Change a NumBox value below")
	c.lblStatus.SetUnderline(true)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Basic NumBox")
	nbBasic := ui.NewNumBox()
	nbBasic.SetOnValueChanged(func() {
		c.setStatus(fmt.Sprintf("Basic NumBox: %v", nbBasic.Value()))
	})
	c.AddWidget(row, 0, nbBasic)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Clamped to a range (0 to 100)")
	nbRange := ui.NewNumBox()
	nbRange.SetMin(0)
	nbRange.SetMax(100)
	nbRange.SetValue(50)
	nbRange.SetOnValueChanged(func() {
		c.setStatus(fmt.Sprintf("Range NumBox: %v (clamped to [0, 100])", nbRange.Value()))
	})
	c.AddWidget(row, 0, nbRange)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Decimals")
	nbInteger := ui.NewNumBox()
	nbInteger.SetDecimals(0)
	nbInteger.SetValue(1)
	nbInteger.SetOnValueChanged(func() {
		c.setStatus(fmt.Sprintf("Integer NumBox: %v", nbInteger.Value()))
	})
	c.AddWidget(row, 0, nbInteger)

	nbPrecise := ui.NewNumBox()
	nbPrecise.SetDecimals(3)
	nbPrecise.SetValue(3.14159)
	nbPrecise.SetOnValueChanged(func() {
		c.setStatus(fmt.Sprintf("Precise NumBox: %v", nbPrecise.Value()))
	})
	c.AddWidget(row, 1, nbPrecise)
	row++
	c.AddLabel(row, 0, "0 decimals")
	c.AddLabel(row, 1, "3 decimals")
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Custom step (spin buttons and mouse wheel move by 5)")
	nbStep := ui.NewNumBox()
	nbStep.SetStep(5)
	nbStep.SetValue(10)
	nbStep.SetOnValueChanged(func() {
		c.setStatus(fmt.Sprintf("Step NumBox: %v", nbStep.Value()))
	})
	c.AddWidget(row, 0, nbStep)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Read the value on demand")
	nbOnDemand := ui.NewNumBox()
	nbOnDemand.SetValue(42)
	c.AddWidget(row, 0, nbOnDemand)
	c.AddButton(row, 1, "Show value", func() {
		c.setStatus(fmt.Sprintf("On-demand NumBox: %v", nbOnDemand.Value()))
	})
	row = addSectionGap(&c.Widget, row+1)

	c.AddVSpacer(row, 0)

	return &c
}

func (c *ExamplePageNumBox) setStatus(text string) {
	c.lblStatus.SetText(text)
}
