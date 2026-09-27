package ui

import "strings"

const (
	// messageBoxMaxLines is the maximum number of text lines shown in a message box.
	messageBoxMaxLines = 20

	messageBoxMinWidth = 250
	messageBoxMaxWidth = 600

	// messageBoxTextHPadding is the horizontal space reserved around the text
	// (panel/cell paddings, window borders, etc.) when computing the window width.
	messageBoxTextHPadding = 40

	// messageBoxChromeHeight is the vertical space taken by everything except
	// the text itself (spacer, button row, paddings).
	messageBoxChromeHeight = 120
)

// runOnParent runs a message box callback on the goroutine of the parent window,
// which owns the widgets the callback usually changes (see Form.Invoke)
func runOnParent(parentWidget Widgeter, f func()) {
	if f == nil {
		return
	}
	if parentWidget == nil || parentWidget.Form() == nil {
		f()
		return
	}
	parentWidget.Form().Invoke(f)
}

func ShowMessageBox(parentWidget Widgeter, title string, text string) {
	c, buttonPanel := newMessageBoxForm(title, text)

	buttonPanel.AddHSpacer(0, 0)
	okButton := buttonPanel.AddButton(0, 1, "", func() { c.Close() })
	okButton.SetTextFunc(func() string { return UIText().OK })
	buttonPanel.AddHSpacer(0, 2)

	c.SetAcceptButton(okButton)
	c.SetCancelButton(okButton)
	okButton.Focus()

	showMessageBoxForm(c, parentWidget)
}

func ShowQuestionMessageBoxYesNo(parentWidget Widgeter, title string, text string, onYes func(), onNo func()) {
	c, buttonPanel := newMessageBoxForm(title, text)

	buttonPanel.AddHSpacer(0, 0)
	yesButton := buttonPanel.AddButton(0, 1, "", func() {
		c.Close()
		runOnParent(parentWidget, onYes)
	})
	yesButton.SetTextFunc(func() string { return UIText().Yes })
	buttonPanel.AddHSpacer(0, 2)
	noButton := buttonPanel.AddButton(0, 3, "", func() {
		c.Close()
		runOnParent(parentWidget, onNo)
	})
	noButton.SetTextFunc(func() string { return UIText().No })
	buttonPanel.AddHSpacer(0, 4)

	c.SetAcceptButton(yesButton)
	c.SetCancelButton(noButton)
	yesButton.Focus()

	showMessageBoxForm(c, parentWidget)
}

func ShowQuestionMessageBoxOKCancel(parentWidget Widgeter, title string, text string, onOK func(), onCancel func()) {
	c, buttonPanel := newMessageBoxForm(title, text)

	buttonPanel.AddHSpacer(0, 0)
	okButton := buttonPanel.AddButton(0, 1, "", func() {
		c.Close()
		runOnParent(parentWidget, onOK)
	})
	okButton.SetTextFunc(func() string { return UIText().OK })
	buttonPanel.AddHSpacer(0, 2)
	cancelButton := buttonPanel.AddButton(0, 3, "", func() {
		c.Close()
		runOnParent(parentWidget, onCancel)
	})
	cancelButton.SetTextFunc(func() string { return UIText().Cancel })
	buttonPanel.AddHSpacer(0, 4)

	c.SetAcceptButton(okButton)
	c.SetCancelButton(cancelButton)
	okButton.Focus()

	showMessageBoxForm(c, parentWidget)
}

// newMessageBoxForm builds a message box form sized to fit the wrapped text
// and returns it along with the empty panel reserved for the button row.
func newMessageBoxForm(title string, text string) (*Form, *Panel) {
	c := NewForm()
	c.SetTitle(title)
	c.SetAllowMinimize(false)
	c.SetAllowMaximize(false)

	fontFamily := ThemeFontFamily()
	fontSize := DefaultFontSize

	maxTextWidth := messageBoxMaxWidth - messageBoxTextHPadding
	lines := wrapTextLines(text, fontFamily, fontSize, maxTextWidth)
	if len(lines) > messageBoxMaxLines {
		lines = lines[:messageBoxMaxLines]
		lines[len(lines)-1] = lines[len(lines)-1] + "..."
	}

	maxLineWidth := 0
	for _, line := range lines {
		lineWidth, _, err := MeasureText(fontFamily, fontSize, line)
		if err == nil && lineWidth > maxLineWidth {
			maxLineWidth = lineWidth
		}
	}

	_, lineHeight, err := MeasureText(fontFamily, fontSize, "Йg")
	if err != nil || lineHeight <= 0 {
		lineHeight = DefaultUiLineHeight
	}
	textHeight := lineHeight * len(lines)

	width := maxLineWidth + messageBoxTextHPadding
	if width < messageBoxMinWidth {
		width = messageBoxMinWidth
	}
	if width > messageBoxMaxWidth {
		width = messageBoxMaxWidth
	}

	height := textHeight + messageBoxChromeHeight

	c.SetSize(width, height)

	lbl := c.Panel().AddLabel(0, 0, strings.Join(lines, "\r\n"))
	lbl.SetTextAlign(HAlignCenter)
	lbl.SetMinHeight(textHeight)
	c.Panel().AddVSpacer(1, 0)
	buttonPanel := c.Panel().AddPanel(2, 0)

	return c, buttonPanel
}

func showMessageBoxForm(c *Form, parentWidget Widgeter) {
	if parentWidget != nil {
		c.ShowModal(parentWidget.Form())
	} else {
		c.Show()
	}
}

// wrapTextLines splits text into lines that fit within maxWidth pixels when
// rendered with the given font. Existing line breaks ("\n" or "\r\n") in the
// input are preserved as paragraph breaks.
func wrapTextLines(text string, fontFamily string, fontSize float64, maxWidth int) []string {
	var lines []string

	text = strings.ReplaceAll(text, "\r\n", "\n")
	paragraphs := strings.Split(text, "\n")

	for _, paragraph := range paragraphs {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		currentLine := ""
		for _, word := range words {
			candidate := word
			if currentLine != "" {
				candidate = currentLine + " " + word
			}

			candidateWidth, _, err := MeasureText(fontFamily, fontSize, candidate)
			if err == nil && candidateWidth <= maxWidth {
				currentLine = candidate
				continue
			}

			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = ""
			}

			wordWidth, _, err := MeasureText(fontFamily, fontSize, word)
			if err == nil && wordWidth > maxWidth {
				// The word alone is wider than maxWidth - break it by characters.
				currentLine = breakLongWord(word, fontFamily, fontSize, maxWidth, &lines)
			} else {
				currentLine = word
			}
		}

		if currentLine != "" {
			lines = append(lines, currentLine)
		}
	}

	return lines
}

// breakLongWord splits a single word that is wider than maxWidth into chunks,
// appending all but the last chunk to lines and returning the remaining tail.
func breakLongWord(word string, fontFamily string, fontSize float64, maxWidth int, lines *[]string) string {
	chunk := ""
	for _, r := range word {
		candidate := chunk + string(r)
		candidateWidth, _, err := MeasureText(fontFamily, fontSize, candidate)
		if err == nil && candidateWidth > maxWidth && chunk != "" {
			*lines = append(*lines, chunk)
			chunk = string(r)
			continue
		}
		chunk = candidate
	}
	return chunk
}
