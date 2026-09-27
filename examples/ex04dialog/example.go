package ex04dialog

import (
	"strconv"

	"github.com/ipoluianov/nui/ui"
)

func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.Panel().AddButton(0, 0, "Dialog New Item", func() {
		dialog := NewDialogNewItem()
		dialog.OnOK = func(name string) {
			ui.ShowMessageBox(form.Panel(), "OK", "OK clicked: "+name)
		}
		dialog.OnCancel = func() {
			ui.ShowMessageBox(form.Panel(), "Cancel", "Cancel clicked")
		}
		form.Panel().ShowDialog(dialog)
	})
	item := ItemToEdit{Name: "Item #1", Description: "First item", IsActive: true}
	form.Panel().AddButton(0, 1, "Dialog Edit Item", func() {
		dialog := NewDialogEditItem(item)
		dialog.OnOK = func(edited ItemToEdit) {
			item = edited
			ui.ShowMessageBox(form.Panel(), "OK", "Item saved: "+item.Name)
		}
		dialog.OnCancel = func() {
			ui.ShowMessageBox(form.Panel(), "Cancel", "Edit cancelled")
		}
		form.Panel().ShowDialog(dialog)
	})
	settings := Settings{UserName: "user", Email: "user@example.com", Notifications: true}
	form.Panel().AddButton(0, 2, "Dialog Settings", func() {
		dialog := NewDialogSettings(settings)
		dialog.OnOK = func(edited Settings) {
			settings = edited
			ui.ShowMessageBox(form.Panel(), "Settings Saved",
				"User Name: "+settings.UserName+"\r\n"+
					"Email: "+settings.Email+"\r\n"+
					"Notifications: "+strconv.FormatBool(settings.Notifications))
		}
		dialog.OnCancel = func() {
			ui.ShowMessageBox(form.Panel(), "Cancel", "Settings not changed")
		}
		form.Panel().ShowDialog(dialog)
	})
	form.Panel().AddButton(0, 3, "Simple Message Box", func() {
		ui.ShowMessageBox(form.Panel(), "Message", "This is a simple message box")
	})
	form.Panel().AddHSpacer(0, 10)
	form.Panel().AddVSpacer(10, 0)
	return form
}
