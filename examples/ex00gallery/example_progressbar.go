package ex00gallery

import (
	"fmt"

	"github.com/ipoluianov/nui/ui"
)

type ExamplePageProgressBar struct {
	ui.Widget

	lblStatus *ui.Label

	pbInteractive *ui.ProgressBar
	pbAuto        *ui.ProgressBar
	autoGrowing   bool
}

func NewExamplePageProgressBar() *ExamplePageProgressBar {
	var c ExamplePageProgressBar
	c.InitWidget()

	row := 0

	c.lblStatus = c.AddLabel(row, 0, "Progress bars show a value between a min and a max")
	c.lblStatus.SetUnderline(true)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Basic progress bar (0 to 100, at 35)")
	pbBasic := ui.NewProgressBar(0, 100, 35)
	c.AddWidget(row, 0, pbBasic)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Custom range (-50 to 50, at 0) with a text overlay")
	pbRange := ui.NewProgressBar(-50, 50, 0)
	pbRange.SetText("0 (midway)")
	c.AddWidget(row, 0, pbRange)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Drive it with buttons")
	c.pbInteractive = ui.NewProgressBar(0, 100, 50)
	c.pbInteractive.SetText(fmt.Sprintf("%.0f%%", c.pbInteractive.Value()))
	c.AddWidget(row, 0, c.pbInteractive)
	c.AddButton(row, 1, "-10%", func() { c.stepInteractive(-10) })
	c.AddButton(row, 2, "+10%", func() { c.stepInteractive(10) })
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Drive it with a NumBox")
	pbFromNumBox := ui.NewProgressBar(0, 100, 20)
	pbFromNumBox.SetText(fmt.Sprintf("%.0f%%", pbFromNumBox.Value()))
	c.AddWidget(row, 0, pbFromNumBox)
	nbPercent := ui.NewNumBox()
	nbPercent.SetMin(0)
	nbPercent.SetMax(100)
	nbPercent.SetDecimals(0)
	nbPercent.SetValue(20)
	nbPercent.SetOnValueChanged(func() {
		pbFromNumBox.SetValue(nbPercent.Value())
		pbFromNumBox.SetText(fmt.Sprintf("%.0f%%", nbPercent.Value()))
	})
	c.AddWidget(row, 1, nbPercent)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Animate automatically")
	c.pbAuto = ui.NewProgressBar(0, 100, 0)
	c.AddWidget(row, 0, c.pbAuto)
	c.AddButton(row, 1, "Start", func() { c.autoGrowing = true })
	c.AddButton(row, 2, "Stop", func() { c.autoGrowing = false })
	row = addSectionGap(&c.Widget, row+1)

	c.AddVSpacer(row, 0)

	c.AddTimer(100, c.onAutoTick)

	return &c
}

func (c *ExamplePageProgressBar) stepInteractive(delta float64) {
	c.pbInteractive.SetValue(c.pbInteractive.Value() + delta)
	c.setStatus(fmt.Sprintf("Interactive progress: %.0f%%", c.pbInteractive.Value()))
	c.pbInteractive.SetText(fmt.Sprintf("%.0f%%", c.pbInteractive.Value()))
}

// onAutoTick advances pbAuto by one step every 100ms while running, wrapping
// back to 0 once it reaches the max - a minimal stand-in for a real
// long-running task's progress.
func (c *ExamplePageProgressBar) onAutoTick() {
	if !c.autoGrowing {
		return
	}
	next := c.pbAuto.Value() + 2
	if next > 100 {
		next = 0
	}
	c.pbAuto.SetValue(next)
}

func (c *ExamplePageProgressBar) setStatus(text string) {
	c.lblStatus.SetText(text)
}
