package ui

import "image"

// The standard dialogs: a line of text to enter, the progress of a long
// work, the about box. Each is modal over the form of the widget given and
// returns at once; the result comes in a callback, after the dialog closes.

// InputDialog asks for a line of text, see ShowInputDialog
type InputDialog struct {
	Title    string
	Prompt   string // the text above the field
	Value    string // the text in the field at the start, selected
	Password bool   // the field shows stars
	// Validate, if set, checks the text when OK is pressed: an error is
	// shown under the field and the dialog stays open
	Validate func(text string) error
}

// ShowInputDialog asks for a line of text; onOK gets it when the user
// presses OK or Enter (Cancel, Esc and the close button call nothing)
//
//	ui.ShowInputDialog(form.Panel(), ui.InputDialog{Title: "Rename", Prompt: "New name:", Value: name},
//		func(text string) { rename(text) })
func ShowInputDialog(parent Widgeter, d InputDialog, onOK func(text string)) {
	var c DialogContent
	c.InitWidget()
	content := NewVBox()
	if d.Prompt != "" {
		content.Add(NewLabel(d.Prompt))
	}
	field := NewTextBox()
	field.SetIsPassword(d.Password)
	field.SetText(d.Value)
	field.SetMinWidth(300)
	content.Add(field)
	errorLabel := NewLabel("")
	errorLabel.SetForegroundColor(CurrentPalette().Error)
	errorLabel.SetVisible(false)
	content.Add(errorLabel)
	c.AddWidget(0, 0, content)
	c.AddVSpacer(1, 0)

	buttons := NewDialogButtonBox()
	buttons.AddOK(func() {
		text := field.Text()
		if d.Validate != nil {
			if err := d.Validate(text); err != nil {
				errorLabel.SetText(err.Error())
				errorLabel.SetVisible(true)
				field.Focus()
				return
			}
		}
		c.RunInParent(func() {
			if onOK != nil {
				onOK(text)
			}
		})
		c.Form().Close()
	})
	buttons.AddCancel(func() { c.Form().Close() })
	c.AddWidget(2, 0, buttons)

	c.OnDialogShow = func() {
		c.Form().SetTitle(d.Title)
		field.Focus()
		field.SelectAllText()
	}
	parent.widgetBase().ShowDialog(&c)
}

// ProgressDialog shows the progress of a long work, with a Cancel button,
// see ShowProgressDialog and RunWithProgress
type ProgressDialog struct {
	DialogContent
	text     *Label
	bar      *ProgressBar
	cancel   *Button
	onCancel func()
}

// ShowProgressDialog shows the dialog with the text and an empty bar;
// onCancel (Cancel, Esc, the close button) may be nil: then the dialog
// can't be cancelled. Close it with Close when the work is done. Its
// methods are for the UI thread: from a goroutine, see RunWithProgress.
func ShowProgressDialog(parent Widgeter, title, text string, onCancel func()) *ProgressDialog {
	c := &ProgressDialog{onCancel: onCancel}
	c.InitWidget()
	c.text = NewLabel(text)
	c.AddWidget(0, 0, c.text)
	c.bar = NewProgressBar(0, 1, 0)
	c.bar.SetMinWidth(320)
	c.AddWidget(1, 0, c.bar)
	c.AddVSpacer(2, 0)
	buttons := NewDialogButtonBox()
	c.cancel = buttons.AddCancel(c.cancelWork)
	c.cancel.SetEnabled(onCancel != nil)
	c.AddWidget(3, 0, buttons)
	c.OnDialogShow = func() { c.Form().SetTitle(title) }
	c.OnDialogReject = func() bool {
		c.cancelWork()
		return false // closed when the work stops
	}
	parent.widgetBase().ShowDialog(c)
	return c
}

func (c *ProgressDialog) cancelWork() {
	if c.onCancel == nil || !c.cancel.Enabled() {
		return
	}
	c.cancel.SetEnabled(false) // once
	c.text.SetText(UIText().Cancel + "...")
	c.onCancel()
}

// SetProgress shows the part done, from 0 to 1; a negative one shows the
// bar empty (the amount of the work unknown)
func (c *ProgressDialog) SetProgress(fraction float64) {
	c.bar.SetValue(max(0, min(1, fraction)))
}

// SetText changes the text above the bar
func (c *ProgressDialog) SetText(text string) {
	c.text.SetText(text)
}

// Close closes the dialog
func (c *ProgressDialog) Close() {
	if f := c.Form(); f != nil {
		f.Close()
	}
}

// AboutDialog is what the about box tells, see ShowAboutDialog
type AboutDialog struct {
	Name        string
	Version     string
	Description string
	Copyright   string
	Website     string
	Icon        image.Image // drawn at the left, 64x64 looks best
}

// ShowAboutDialog shows the name, the version and the rest of the
// application, with the link to its website
func ShowAboutDialog(parent Widgeter, d AboutDialog) {
	var c DialogContent
	c.InitWidget()
	row := NewHBox()
	row.SetCellPadding(16)
	if d.Icon != nil {
		icon := NewImageBox()
		icon.SetImage(d.Icon)
		b := d.Icon.Bounds()
		icon.SetMinSize(b.Dx(), b.Dy())
		icon.SetMaxSize(b.Dx(), b.Dy())
		row.Add(icon)
	}
	text := NewVBox()
	name := NewLabel(d.Name)
	name.SetFontBold(true)
	name.SetFontSize(ThemeFontSize() * 1.4)
	text.Add(name)
	for _, line := range []string{d.Version, d.Description, d.Copyright} {
		if line != "" {
			text.Add(NewLabel(line))
		}
	}
	if d.Website != "" {
		text.Add(NewLink(d.Website, func() { OpenURL(d.Website) }))
	}
	row.Add(text)
	c.AddWidget(0, 0, row)
	c.AddVSpacer(1, 0)
	buttons := NewDialogButtonBox()
	ok := buttons.AddOK(func() { c.Form().Close() })
	c.AddWidget(2, 0, buttons)
	c.OnDialogShow = func() {
		c.Form().SetTitle(d.Name)
		c.Form().SetCancelButton(ok) // Esc closes it too
		ok.Focus()
	}
	parent.widgetBase().ShowDialog(&c)
}
