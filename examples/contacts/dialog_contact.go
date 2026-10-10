package contacts

import (
	"strings"

	"github.com/ipoluianov/nui/ui"
)

// DialogContact adds a new contact or edits an existing one. OK checks the
// fields and keeps the dialog open with a message while they are wrong.
type DialogContact struct {
	ui.DialogContent

	title       string
	txtName     *ui.TextBox
	txtCompany  *ui.TextBox
	txtPhone    *ui.TextBox
	txtEmail    *ui.TextBox
	cmbGroup    *ui.ComboBox
	dpBirthday  *ui.DatePicker
	chkFavorite *ui.Checkbox
	txtNotes    *ui.TextBox
	lblError    *ui.Label
	btnOK       *ui.Button
	btnCancel   *ui.Button

	// OnOK gets the edited contact
	OnOK func(contact Contact)
}

func NewDialogContact(title string, contact Contact) *DialogContact {
	var c DialogContact
	c.InitWidget()
	c.title = title

	fields := c.AddPanel(0, 0)
	field := func(row int, label string, w ui.Widgeter) {
		fields.AddLabel(row, 0, label)
		fields.AddWidget(row, 1, w)
	}
	textBox := func(row int, label, text string) *ui.TextBox {
		txt := ui.NewTextBox()
		txt.SetText(text)
		field(row, label, txt)
		return txt
	}
	c.txtName = textBox(0, "Name:", contact.Name)
	c.txtCompany = textBox(1, "Company:", contact.Company)
	c.txtPhone = textBox(2, "Phone:", contact.Phone)
	c.txtEmail = textBox(3, "Email:", contact.Email)
	c.txtEmail.SetHint("name@example.com")

	c.cmbGroup = ui.NewComboBox()
	for i, g := range groups {
		c.cmbGroup.AddItem(g, g)
		if g == contact.Group {
			c.cmbGroup.SetSelectedIndex(i)
		}
	}
	field(4, "Group:", c.cmbGroup)

	c.dpBirthday = ui.NewDatePicker()
	c.dpBirthday.SetDate(contact.Birthday)
	field(5, "Birthday:", c.dpBirthday)

	c.chkFavorite = ui.NewCheckbox("Favorite")
	c.chkFavorite.SetChecked(contact.Favorite)
	fields.AddWidget(6, 1, c.chkFavorite)

	c.AddLabel(1, 0, "Notes:")
	c.txtNotes = ui.NewTextBox()
	c.txtNotes.SetMultiline(true)
	c.txtNotes.SetText(contact.Notes)
	c.txtNotes.SetYExpandable(true)
	c.AddWidget(2, 0, c.txtNotes)

	c.lblError = c.AddLabel(3, 0, "")
	c.lblError.SetForegroundColor(ui.ColorFromHex("#E5484D"))

	buttons := c.AddPanel(4, 0)
	buttons.AddHSpacer(0, 0)
	c.btnOK = buttons.AddButton(0, 1, "OK", c.accept)
	c.btnCancel = buttons.AddButton(0, 2, "Cancel", func() { c.Form().Close() })

	c.OnDialogShow = c.onShow
	return &c
}

func (c *DialogContact) onShow() {
	c.Form().SetTitle(c.title)
	c.Form().SetSize(440, 520)
	c.Form().MoveToCenterOfParent()
	// Enter (outside the notes) means OK, Esc means Cancel
	c.Form().SetAcceptButton(c.btnOK)
	c.Form().SetCancelButton(c.btnCancel)
	c.txtName.Focus()
	c.txtName.SelectAllText()
}

func (c *DialogContact) accept() {
	contact := Contact{
		Name:     strings.TrimSpace(c.txtName.Text()),
		Company:  strings.TrimSpace(c.txtCompany.Text()),
		Phone:    strings.TrimSpace(c.txtPhone.Text()),
		Email:    strings.TrimSpace(c.txtEmail.Text()),
		Group:    c.cmbGroup.SelectedItemText(),
		Birthday: c.dpBirthday.Date(),
		Favorite: c.chkFavorite.Checked(),
		Notes:    c.txtNotes.Text(),
	}
	switch {
	case contact.Name == "":
		c.showError("The name is required", c.txtName)
		return
	case contact.Email != "" && !strings.Contains(contact.Email, "@"):
		c.showError("The email address must contain \"@\"", c.txtEmail)
		return
	}
	onOK := c.OnOK
	c.RunInParent(func() {
		if onOK != nil {
			onOK(contact)
		}
	})
	c.Form().Close()
}

func (c *DialogContact) showError(text string, field *ui.TextBox) {
	c.lblError.SetText(text)
	field.Focus()
	field.SelectAllText()
}
