package ex02messagebox

import "github.com/ipoluianov/nui/ui"

func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	mainPanel := form.Panel()
	mainPanel.AddButton(0, 0, "MessageBox", func() {
		ui.ShowMessageBox(mainPanel, "", "")
	})
	mainPanel.AddButton(0, 1, "MessageBox", func() {
		ui.ShowMessageBox(mainPanel, "Header", "Text of the message")
	})
	mainPanel.AddVSpacer(10, 0)

	return form
}
