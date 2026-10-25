package notepad

import (
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/ipoluianov/nui/ui"
)

// The edits of the commands go through InsertText, so Undo takes them back
// as a typing would be

// changeCase converts the selected text, or the whole text if nothing is
// selected
func (n *notepad) changeCase(convert func(string) string) {
	if !n.text.HasSelection() {
		n.text.SelectAllText()
	}
	n.text.InsertText(convert(n.text.SelectedText()))
}

func (n *notepad) insertDateTime() {
	n.text.InsertText(time.Now().Format("2006-01-02 15:04"))
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
	n.text.SetSelection(n.text.PosAt(pos), n.text.PosAt(pos+len(query)))
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
		n.text.SetCursorPosition(ui.TextPos{Line: line - 1})
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
