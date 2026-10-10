package notepad

import (
	"fmt"

	"github.com/ipoluianov/nui/ui"
)

// findDialog asks what to find. It stays open while the user presses Find
// Next: each press selects the next match in the editor.
type findDialog struct {
	ui.DialogContent

	txtFind  *ui.TextBox
	chkCase  *ui.Checkbox
	btnFind  *ui.Button
	btnClose *ui.Button

	// OnFindNext selects the next match and returns false if there is none
	OnFindNext func(text string, matchCase bool) bool
	OnClose    func()
}

func newFindDialog(text string, matchCase bool) *findDialog {
	var c findDialog
	c.InitWidget()

	row := c.AddPanel(0, 0)
	row.SetPanelPadding(0)
	row.AddLabel(0, 0, "Find what:")
	c.txtFind = ui.NewTextBox()
	c.txtFind.SetText(text)
	c.txtFind.SetOnTextChanged(func() {
		c.btnFind.SetEnabled(c.txtFind.Text() != "")
	})
	row.AddWidget(0, 1, c.txtFind)

	c.chkCase = ui.NewCheckbox("Match case")
	c.chkCase.SetChecked(matchCase)
	c.AddWidget(1, 0, c.chkCase)

	c.AddVSpacer(2, 0)

	buttons := c.AddPanel(3, 0)
	buttons.SetPanelPadding(0)
	buttons.AddHSpacer(0, 0)
	c.btnFind = buttons.AddButton(0, 1, "Find Next", func() {
		if c.OnFindNext != nil && !c.OnFindNext(c.txtFind.Text(), c.chkCase.Checked()) {
			ui.ShowToast(c.Form().ParentForm().Panel(), "Cannot find \""+c.txtFind.Text()+"\"", ui.ToastWarning)
		}
	})
	c.btnFind.SetEnabled(text != "")
	c.btnClose = buttons.AddButton(0, 2, "Close", func() {
		c.RunInParent(c.OnClose)
		c.Form().Close()
	})

	c.OnDialogShow = func() {
		c.Form().SetTitle("Find")
		c.Form().SetSize(380, 125)
		c.Form().MoveToCenterOfParent()
		c.Form().SetAcceptButton(c.btnFind)
		c.Form().SetCancelButton(c.btnClose)
		c.txtFind.Focus()
		c.txtFind.SelectAllText()
	}
	c.OnDialogReject = func() bool {
		c.RunInParent(c.OnClose)
		return true
	}
	return &c
}

// goToLineDialog asks for a line number
type goToLineDialog struct {
	ui.DialogContent

	numLine *ui.NumBox
	btnOK   *ui.Button

	OnOK func(line int)
}

func newGoToLineDialog(lines int) *goToLineDialog {
	var c goToLineDialog
	c.InitWidget()

	row := c.AddPanel(0, 0)
	row.SetPanelPadding(0)
	row.AddLabel(0, 0, fmt.Sprintf("Line number (1 - %d):", lines))
	c.numLine = ui.NewNumBox()
	c.numLine.SetDecimals(0)
	c.numLine.SetMin(1)
	c.numLine.SetMax(float64(lines))
	c.numLine.SetValue(1)
	row.AddWidget(0, 1, c.numLine)

	c.AddVSpacer(1, 0)

	buttons := c.AddPanel(2, 0)
	buttons.SetPanelPadding(0)
	buttons.AddHSpacer(0, 0)
	c.btnOK = buttons.AddButton(0, 1, "Go To", func() {
		line := int(c.numLine.Value())
		if c.OnOK != nil {
			c.RunInParent(func() { c.OnOK(line) })
		}
		c.Form().Close()
	})
	btnCancel := buttons.AddButton(0, 2, "Cancel", func() { c.Form().Close() })

	c.OnDialogShow = func() {
		c.Form().SetTitle("Go To Line")
		c.Form().SetSize(340, 100)
		c.Form().MoveToCenterOfParent()
		c.Form().SetAcceptButton(c.btnOK)
		c.Form().SetCancelButton(btnCancel)
		c.numLine.Focus()
	}
	return &c
}
