package ex01base

import "github.com/ipoluianov/nui/ui"

func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetSize(400, 200)
	form.Panel().AddLabel(0, 0, "LABEL")
	return form
}
