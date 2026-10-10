package notepad

import (
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/ipoluianov/nui/ui"
)

// The text box has no API to put the cursor or the selection at a place of
// a multiline text, so these helpers move it with the keys, as the user
// would. Typing the text in (KeyChar) keeps the undo history, which SetText
// would clear.

// selectAt selects count runes from line, col (from 0)
func (n *notepad) selectAt(line, col, count int) {
	none, shift := ui.KeyModifiers{}, ui.KeyModifiers{Shift: true}
	n.text.SelectRange(0, 0) // the cursor to the start
	for range line {
		n.text.KeyDown(ui.KeyArrowDown, none)
	}
	for range col {
		n.text.KeyDown(ui.KeyArrowRight, none)
	}
	for range count {
		n.text.KeyDown(ui.KeyArrowRight, shift)
	}
}

// typeText replaces the selection with s, or inserts it at the cursor
func (n *notepad) typeText(s string) {
	for _, ch := range s {
		if ch == '\n' {
			n.text.KeyDown(ui.KeyEnter, ui.KeyModifiers{})
		} else {
			n.text.KeyChar(ch, ui.KeyModifiers{})
		}
	}
}

// changeCase converts the selected text, or the whole text if nothing is
// selected
func (n *notepad) changeCase(convert func(string) string) {
	if sel := n.text.SelectedText(); sel != "" {
		n.typeText(convert(sel))
		return
	}
	n.text.SetText(convert(n.text.Text()))
}

func (n *notepad) insertDateTime() {
	n.typeText(time.Now().Format("2006-01-02 15:04"))
}

// find selects the next match of the last search after findPos, going
// around to the start of the text; it returns false if there is none
func (n *notepad) find() bool {
	text, query := []rune(n.text.Text()), []rune(n.findText)
	if len(query) == 0 {
		return false
	}
	if !n.matchCase {
		text, query = lowerRunes(text), lowerRunes(query)
	}
	start := min(n.findPos, len(text))
	pos := indexRunes(text[start:], query)
	if pos >= 0 {
		pos += start
	} else if pos = indexRunes(text, query); pos < 0 {
		return false
	}
	line := 0
	lineStart := 0
	for i, ch := range text[:pos] {
		if ch == '\n' {
			line++
			lineStart = i + 1
		}
	}
	n.selectAt(line, pos-lineStart, len(query))
	n.findPos = pos + len(query)
	return true
}

// findNext repeats the last search (F3), or asks what to find
func (n *notepad) findNext() {
	if n.findText == "" {
		n.showFind()
		return
	}
	if !n.find() {
		n.form.ShowToast("Cannot find \""+n.findText+"\"", ui.ToastWarning)
	}
	n.text.Focus()
}

func (n *notepad) showFind() {
	dialog := newFindDialog(n.findText, n.matchCase)
	dialog.OnFindNext = func(text string, matchCase bool) bool {
		n.findText, n.matchCase = text, matchCase
		return n.find()
	}
	dialog.OnClose = func() { n.text.Focus() }
	n.form.Panel().ShowDialog(dialog)
}

func (n *notepad) showGoToLine() {
	lines := strings.Count(n.text.Text(), "\n") + 1
	dialog := newGoToLineDialog(lines)
	dialog.OnOK = func(line int) {
		n.selectAt(line-1, 0, 0)
		n.text.Focus()
	}
	n.form.Panel().ShowDialog(dialog)
}

func lowerRunes(s []rune) []rune {
	out := make([]rune, len(s))
	for i, ch := range s {
		out[i] = unicode.ToLower(ch)
	}
	return out
}

// indexRunes is strings.Index for runes: positions in runes are what the
// text box selects by
func indexRunes(s, sub []rune) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if slices.Equal(s[i:i+len(sub)], sub) {
			return i
		}
	}
	return -1
}
