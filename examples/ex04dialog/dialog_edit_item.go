package ex04dialog

import "github.com/ipoluianov/nui/ui"

// ItemToEdit is the data DialogEditItem edits.
type ItemToEdit struct {
	Name        string
	Description string
	IsActive    bool
}

// DialogEditItem edits an already existing ItemToEdit. Unlike DialogNewItem,
// which always starts from blank fields, NewDialogEditItem takes the item's
// current value and pre-fills the form with it, so the user edits in place
// instead of typing everything from scratch.
type DialogEditItem struct {
	ui.DialogContent

	txtName        *ui.TextBox
	txtDescription *ui.TextBox
	chkActive      *ui.Checkbox

	OnOK     func(item ItemToEdit)
	OnCancel func()
}

func NewDialogEditItem(item ItemToEdit) *DialogEditItem {
	var c DialogEditItem
	c.InitWidget()

	panelName := c.AddPanel(0, 0)
	panelName.AddWidget(0, 0, ui.NewLabel("Name:"))
	c.txtName = ui.NewTextBox()
	c.txtName.SetText(item.Name)
	panelName.AddWidget(0, 1, c.txtName)

	panelDescription := c.AddPanel(1, 0)
	panelDescription.AddWidget(0, 0, ui.NewLabel("Description:"))
	c.txtDescription = ui.NewTextBox()
	c.txtDescription.SetText(item.Description)
	panelDescription.AddWidget(0, 1, c.txtDescription)

	c.chkActive = ui.NewCheckbox("Active")
	c.chkActive.SetChecked(item.IsActive)
	c.AddWidget(2, 0, c.chkActive)

	c.AddVSpacer(3, 0)

	panelButtons := c.AddPanel(4, 0)
	panelButtons.AddHSpacer(0, 0)
	panelButtons.AddButton(0, 1, "OK", func() {
		if c.OnOK != nil {
			c.OnOK(ItemToEdit{
				Name:        c.txtName.Text(),
				Description: c.txtDescription.Text(),
				IsActive:    c.chkActive.Checked(),
			})
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

func (c *DialogEditItem) OnShow() {
	c.Form().SetTitle("Edit Item Dialog")
	c.Form().SetSize(400, 220)
	c.Form().MoveToCenterOfParent()
	c.txtName.Focus()
	c.txtName.SelectAllText()
}

func (c *DialogEditItem) OnReject() bool {
	return false
}
