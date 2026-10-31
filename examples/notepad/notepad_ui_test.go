package notepad

import (
	"strings"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// Typing marks the document modified and updates the status bar; Edit >
// Upper Case works from the main menu
func TestEditing(t *testing.T) {
	n := newNotepad()
	w := uitest.Show(t, n.form)

	w.Shortcut("Mod+A")
	w.Type("hello world")
	if got := n.text.Text(); got != "hello world" {
		t.Fatalf("text = %q", got)
	}
	if !strings.HasPrefix(w.Title(), "Untitled*") {
		t.Errorf("title %q: not marked modified", w.Title())
	}
	if !strings.Contains(n.lblCounts.Text(), "Words: 2") {
		t.Errorf("status %q", n.lblCounts.Text())
	}

	w.Click(w.Find("Edit"))
	w.Click(w.Find("Upper Case"))
	if got := n.text.Text(); got != "HELLO WORLD" {
		t.Errorf("after Upper Case: %q", got)
	}

	// The command's edit is undone as a typing would be
	w.Shortcut("Mod+Z")
	if got := n.text.Text(); got != "hello world" {
		t.Errorf("after Undo: %q", got)
	}
}

// Find selects the match, Go To Line puts the cursor on the line
func TestFindAndGoToLine(t *testing.T) {
	n := newNotepad()
	w := uitest.Show(t, n.form)
	n.text.SetText("alpha\nbeta gamma\ndelta")
	n.findText = "gamma"
	if !n.find() {
		t.Fatal("not found")
	}
	if got := n.text.SelectedText(); got != "gamma" {
		t.Errorf("selected %q", got)
	}
	from, _ := n.text.Selection()
	if from != (ui.TextPos{Line: 1, Col: 5}) {
		t.Errorf("selection from %+v", from)
	}
	n.text.SetCursorPosition(ui.TextPos{Line: 2})
	w.Type("X")
	if got := n.text.Line(2); got != "Xdelta" {
		t.Errorf("line 3 %q", got)
	}
}

// Closing with unsaved changes asks first; "Yes" saves, "No" discards
func TestCloseAsks(t *testing.T) {
	n := newNotepad()
	w := uitest.Show(t, n.form)
	w.Type("x")
	if w.RequestClose() {
		t.Fatal("closed without asking")
	}
	question := w.Dialog()
	question.Click(question.Find("No"))
	if !w.Closed() {
		t.Error("No doesn't close")
	}
}

// Escape dismisses the question and keeps the window with
// its changes
func TestCloseQuestionEscape(t *testing.T) {
	n := newNotepad()
	w := uitest.Show(t, n.form)
	w.Type("x")
	w.RequestClose()
	w.Dialog().Key(ui.KeyEsc, ui.KeyModifiers{})
	if w.Closed() || !strings.HasPrefix(n.text.Text(), "x") {
		t.Errorf("Escape: closed %v, text %q", w.Closed(), n.text.Text())
	}
	// The close button of the question is a Cancel too
	w.RequestClose()
	w.Dialog().RequestClose()
	if w.Closed() {
		t.Error("closing the question closed the window")
	}
}

// The items of Edit are grayed when they can't do anything: Undo of the new
// document, Copy without a selection, Paste with no text to paste
func TestEditMenuGrayed(t *testing.T) {
	n := newNotepad()
	w := uitest.Show(t, n.form)
	ui.ClipboardSetText("")

	enabled := func(item string) bool {
		w.Click(w.Find("Edit"))
		defer w.Key(ui.KeyEsc, ui.KeyModifiers{})
		return w.Find(item).Enabled()
	}
	if enabled("Undo") || enabled("Redo") || enabled("Copy") || enabled("Cut") {
		t.Error("Undo, Redo, Copy or Cut enabled in the new document")
	}
	w.Click(n.text)
	w.Type("x")
	if !enabled("Undo") {
		t.Error("Undo grayed after typing")
	}
	w.Shortcut("Mod+A")
	if !enabled("Copy") || !enabled("Cut") {
		t.Error("Copy or Cut grayed with the text selected")
	}
	ui.ClipboardSetText("clip")
	if !enabled("Paste") {
		t.Error("Paste grayed with text in the clipboard")
	}
}
