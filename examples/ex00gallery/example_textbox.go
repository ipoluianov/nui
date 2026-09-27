package ex00gallery

import "github.com/ipoluianov/nui/ui"

type ExamplePageTextBox struct {
	ui.Widget

	lblEcho *ui.Label
}

func NewExamplePageTextBox() *ExamplePageTextBox {
	var c ExamplePageTextBox
	c.InitWidget()

	row := 0

	row = addSectionHeader(&c.Widget, row, "Basic text box")
	tbBasic := ui.NewTextBox()
	tbBasic.SetText("Hello, world!")
	c.AddWidget(row, 0, tbBasic)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Hint text")
	tbHint := ui.NewTextBox()
	tbHint.SetHint("Enter your name...")
	c.AddWidget(row, 0, tbHint)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Read-only")
	tbReadOnly := ui.NewTextBox()
	tbReadOnly.SetText("You can't edit this text")
	tbReadOnly.SetReadOnly(true)
	c.AddWidget(row, 0, tbReadOnly)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Password")
	tbPassword := ui.NewTextBox()
	tbPassword.SetText("secret")
	tbPassword.SetIsPassword(true)
	c.AddWidget(row, 0, tbPassword)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Multiline")
	tbMultiline := ui.NewTextBox()
	tbMultiline.SetMultiline(true)
	// SetMultiline alone doesn't take effect until the textbox is attached
	// to a form, so it stays vertically collapsed to one line in a page
	// like this one that has no other expandable row to push it open.
	// Setting YExpandable directly works regardless of attachment order.
	tbMultiline.SetYExpandable(true)
	tbMultiline.SetText("First line\nSecond line\nThird line")
	tbMultiline.SetMinSize(300, ui.DefaultUiLineHeight*4)
	c.AddWidget(row, 0, tbMultiline)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Live text change")
	tbEcho := ui.NewTextBox()
	tbEcho.SetHint("Type something...")
	c.lblEcho = ui.NewLabel("You typed: ")
	tbEcho.SetOnTextChanged(func() {
		c.lblEcho.SetText("You typed: " + tbEcho.Text())
	})
	c.AddWidget(row, 0, tbEcho)
	row++
	c.AddWidget(row, 0, c.lblEcho)
	row++
	row = addSectionGap(&c.Widget, row)

	c.AddVSpacer(row, 0)

	return &c
}
