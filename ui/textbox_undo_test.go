package ui

import (
	"testing"
	"time"
)

func typeText(form *Form, s string) {
	for _, r := range s {
		if r == ' ' {
			typeKey(form, KeySpace, ' ', KeyModifiers{})
			continue
		}
		form.processChar(r)
	}
}

func TestTextBoxUndoRedo(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("start")
	box.Focus()
	box.MoveCursorToEnd()
	ctrl := KeyModifiers{Ctrl: true}

	typeText(form, " one")
	if box.Text() != "start one" {
		t.Fatalf("typed: %q", box.Text())
	}
	// Typing in a run is one step
	typeKey(form, KeyZ, 0, ctrl)
	if box.Text() != "start" || box.CanUndo() {
		t.Fatalf("undo typing: %q, can undo %v", box.Text(), box.CanUndo())
	}
	typeKey(form, KeyY, 0, ctrl)
	if box.Text() != "start one" {
		t.Fatalf("redo: %q", box.Text())
	}

	// A run of Backspace is one step, separate from the typing
	typeKey(form, KeyBackspace, 0, KeyModifiers{})
	typeKey(form, KeyBackspace, 0, KeyModifiers{})
	if box.Text() != "start o" {
		t.Fatalf("backspace: %q", box.Text())
	}
	typeKey(form, KeyZ, 0, ctrl)
	if box.Text() != "start one" {
		t.Fatalf("undo backspace: %q", box.Text())
	}

	// Typing after a pause, or elsewhere, is a new step
	typeText(form, "!")
	box.lastEditTime = time.Now().Add(-time.Minute)
	typeText(form, "?")
	typeKey(form, KeyZ, 0, ctrl)
	if box.Text() != "start one!" {
		t.Fatalf("after a pause: %q", box.Text())
	}
	typeKey(form, KeyHome, 0, KeyModifiers{})
	typeText(form, ">")
	typeKey(form, KeyZ, 0, ctrl)
	if box.Text() != "start one!" {
		t.Fatalf("typing elsewhere: %q", box.Text())
	}

	// A new edit drops the redo; Ctrl+Shift+Z redoes too
	typeKey(form, KeyZ, 0, ctrl)
	if box.Text() != "start one" || !box.CanRedo() {
		t.Fatalf("undo: %q", box.Text())
	}
	typeKey(form, KeyZ, 0, KeyModifiers{Ctrl: true, Shift: true})
	if box.Text() != "start one!" {
		t.Fatalf("Ctrl+Shift+Z: %q", box.Text())
	}
	typeKey(form, KeyZ, 0, ctrl)
	typeText(form, "x")
	if box.CanRedo() {
		t.Error("an edit after Undo must drop the redo")
	}

	// SetText from the code starts a new history; read-only can't undo
	box.SetText("new")
	if box.CanUndo() {
		t.Error("SetText kept the history")
	}
	box.MoveCursorToEnd()
	typeText(form, "er")
	box.SetReadOnly(true)
	typeKey(form, KeyZ, 0, ctrl)
	if box.Text() != "newer" {
		t.Errorf("read-only undo: %q", box.Text())
	}
}

func TestTextBoxUndoKeepsShortcutsAway(t *testing.T) {
	form, box, clicks := newMenuTestForm(t)
	form.menuBar.items[1].menu.AddItem("&Undo", func() { clicks["undo"]++ }).SetShortcut("Ctrl+Z")
	box.Focus()
	box.MoveCursorToEnd()
	typeText(form, "abc")
	typeKey(form, KeyZ, 0, KeyModifiers{Ctrl: true})
	if box.Text() != "" || clicks["undo"] != 0 {
		t.Errorf("Ctrl+Z in the text box: %q, menu undo %d", box.Text(), clicks["undo"])
	}
}
