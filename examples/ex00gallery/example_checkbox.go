package ex00gallery

import (
	"fmt"
	"strings"

	"github.com/ipoluianov/nui/ui"
)

type ExamplePageCheckbox struct {
	ui.Widget

	lblStatus *ui.Label

	cbBold      *ui.Checkbox
	cbItalic    *ui.Checkbox
	cbUnderline *ui.Checkbox
}

func NewExamplePageCheckbox() *ExamplePageCheckbox {
	var c ExamplePageCheckbox
	c.InitWidget()

	row := 0

	c.lblStatus = c.AddLabel(row, 0, "Toggle a checkbox below")
	c.lblStatus.SetUnderline(true)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Basic checkbox")
	cbBasic := ui.NewCheckbox("Enable notifications")
	cbBasic.SetOnStateChanged(func() {
		c.setStatus(fmt.Sprintf("Basic checkbox: %s", checkedText(cbBasic.Checked())))
	})
	c.AddWidget(row, 0, cbBasic)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Pre-checked checkbox")
	cbPrechecked := ui.NewCheckbox("Remember me")
	cbPrechecked.SetChecked(true)
	cbPrechecked.SetOnStateChanged(func() {
		c.setStatus(fmt.Sprintf("Pre-checked checkbox: %s", checkedText(cbPrechecked.Checked())))
	})
	c.AddWidget(row, 0, cbPrechecked)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Group of checkboxes")
	c.cbBold = ui.NewCheckbox("Bold")
	c.cbBold.SetOnStateChanged(func() { c.updateGroupStatus() })
	c.AddWidget(row, 0, c.cbBold)

	c.cbItalic = ui.NewCheckbox("Italic")
	c.cbItalic.SetOnStateChanged(func() { c.updateGroupStatus() })
	c.AddWidget(row, 1, c.cbItalic)

	c.cbUnderline = ui.NewCheckbox("Underline")
	c.cbUnderline.SetOnStateChanged(func() { c.updateGroupStatus() })
	c.AddWidget(row, 2, c.cbUnderline)
	row = addSectionGap(&c.Widget, row+1)

	c.AddVSpacer(row, 0)

	return &c
}

func (c *ExamplePageCheckbox) updateGroupStatus() {
	var selected []string
	if c.cbBold.Checked() {
		selected = append(selected, "Bold")
	}
	if c.cbItalic.Checked() {
		selected = append(selected, "Italic")
	}
	if c.cbUnderline.Checked() {
		selected = append(selected, "Underline")
	}
	if len(selected) == 0 {
		c.setStatus("Group: none selected")
		return
	}
	c.setStatus("Group: " + strings.Join(selected, ", "))
}

func (c *ExamplePageCheckbox) setStatus(text string) {
	c.lblStatus.SetText(text)
}

func checkedText(checked bool) string {
	if checked {
		return "checked"
	}
	return "unchecked"
}
