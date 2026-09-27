package ex04dialog

import "github.com/ipoluianov/nui/ui"

// Settings is the data DialogSettings edits.
type Settings struct {
	UserName      string
	Email         string
	Notifications bool
}

// DialogSettings edits application settings. Like DialogEditItem, it takes
// the current settings and pre-fills the form with them.
type DialogSettings struct {
	ui.DialogContent

	txtUserName      *ui.TextBox
	txtEmail         *ui.TextBox
	chkNotifications *ui.Checkbox

	OnOK     func(settings Settings)
	OnCancel func()
}

func NewDialogSettings(settings Settings) *DialogSettings {
	var c DialogSettings
	c.InitWidget()

	panelUserName := c.AddPanel(0, 0)
	panelUserName.AddWidget(0, 0, ui.NewLabel("User Name:"))
	c.txtUserName = ui.NewTextBox()
	c.txtUserName.SetText(settings.UserName)
	panelUserName.AddWidget(0, 1, c.txtUserName)

	panelEmail := c.AddPanel(1, 0)
	panelEmail.AddWidget(0, 0, ui.NewLabel("Email:"))
	c.txtEmail = ui.NewTextBox()
	c.txtEmail.SetText(settings.Email)
	panelEmail.AddWidget(0, 1, c.txtEmail)

	c.chkNotifications = ui.NewCheckbox("Enable notifications")
	c.chkNotifications.SetChecked(settings.Notifications)
	c.AddWidget(2, 0, c.chkNotifications)

	c.AddVSpacer(3, 0)

	panelButtons := c.AddPanel(4, 0)
	panelButtons.AddHSpacer(0, 0)
	panelButtons.AddButton(0, 1, "OK", func() {
		if c.OnOK != nil {
			c.OnOK(Settings{
				UserName:      c.txtUserName.Text(),
				Email:         c.txtEmail.Text(),
				Notifications: c.chkNotifications.Checked(),
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

func (c *DialogSettings) OnShow() {
	c.Form().SetTitle("Settings")
	c.Form().SetSize(400, 220)
	c.Form().MoveToCenterOfParent()
	c.txtUserName.Focus()
	c.txtUserName.SelectAllText()
}

func (c *DialogSettings) OnReject() bool {
	return false
}
