package ex04dialog

import "github.com/ipoluianov/nui/ui"

type DialogNewItem struct {
	ui.DialogContent

	txtBox *ui.TextBox

	OnOK     func(name string)
	OnCancel func()
}

func NewDialogNewItem() *DialogNewItem {
	var c DialogNewItem
	c.InitWidget()

	panelEnterName := c.AddPanel(0, 0)
	panelEnterName.AddWidget(0, 0, ui.NewLabel("Enter Name:"))
	c.txtBox = ui.NewTextBox()
	panelEnterName.AddWidget(0, 1, c.txtBox)

	c.AddVSpacer(1, 0)

	panelButtons := c.AddPanel(2, 0)
	panelButtons.AddHSpacer(0, 0)
	panelButtons.AddButton(0, 1, "OK", func() {
		if c.OnOK != nil {
			c.OnOK(c.txtBox.Text())
		}
		c.Form().Close()
	})
	panelButtons.AddButton(0, 2, "Cancel", func() {
		if c.OnCancel != nil {
			c.OnCancel()
		}
		c.Form().Close()
	})

	c.OnDialogShow = c.OnShow
	c.OnDialogReject = c.OnReject

	return &c
}

func (c *DialogNewItem) OnShow() {
	c.txtBox.Form().SetTitle("New Item Dialog")
	c.txtBox.Form().SetSize(300, 150)
	c.txtBox.Form().MoveToCenterOfParent()
	c.txtBox.Focus()
}

func (c *DialogNewItem) OnReject() bool {
	return true
}
