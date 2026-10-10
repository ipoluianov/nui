package ui

import "testing"

// The keys that edit text reach a focused text box before the shortcuts and
// the global key handler of the form; the others still reach them
func TestTextEditKeysGoToTextBox(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("abc")
	box.Focus()
	box.MoveCursorToEnd()
	ctrl := KeyModifiers{Ctrl: true}

	var global []Key
	form.SetOnGlobalKeyDown(func(k Key, m KeyModifiers) bool {
		if m.Ctrl {
			global = append(global, k)
			return true
		}
		return false
	})
	undone := false
	form.AddShortcut("Ctrl+Z", func() { undone = true })

	typeText(form, "d")
	typeKey(form, KeyZ, 0, ctrl)
	if box.Text() != "abc" || undone || len(global) != 0 {
		t.Fatalf("Ctrl+Z: text %q, the form's undo %v, global %v", box.Text(), undone, global)
	}
	typeKey(form, KeyA, 0, ctrl)
	if box.SelectedText() != "abc" || len(global) != 0 {
		t.Fatalf("Ctrl+A: selected %q, global %v", box.SelectedText(), global)
	}
	// Not a text key: the form's handler gets it
	typeKey(form, KeyF, 0, ctrl)
	if len(global) != 1 || global[0] != KeyF {
		t.Fatalf("Ctrl+F: global %v", global)
	}
	// A disabled box does not take them
	box.SetEnabled(false)
	typeKey(form, KeyZ, 0, ctrl)
	if len(global) != 2 {
		t.Fatalf("Ctrl+Z in a disabled box: global %v", global)
	}
}
