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

// F2 that opens the editor of a cell reaches a focused tree or table before
// a shortcut on F2; on a cell that cannot be edited the shortcut gets it
func TestEditKeysGoToTreeAndTable(t *testing.T) {
	tree := NewTreeView()
	form := newTestForm(t, tree)
	tree.SetColumnCount(2)
	tree.SetColumnReadOnly(0, true)
	tree.SetEditTriggerF2(true)
	a := tree.AddNode(nil, "a")
	b := tree.AddNode(nil, "b")
	b.SetReadOnly(1, true)
	shortcut := 0
	form.AddShortcut("F2", func() { shortcut++ })
	tree.Focus()
	tree.SetCurrentNode(a)
	tree.SetCurrentColumn(0) // the name: the value is edited
	typeKey(form, KeyF2, 0, KeyModifiers{})
	if !tree.IsEditing() || tree.editColumn != 1 || shortcut != 0 {
		t.Fatalf("tree: editing %v, shortcut %d", tree.IsEditing(), shortcut)
	}
	tree.cancelEdit()
	tree.Focus()
	tree.SetCurrentNode(b)
	typeKey(form, KeyF2, 0, KeyModifiers{})
	if tree.IsEditing() || shortcut != 1 {
		t.Fatalf("read-only cell: editing %v, shortcut %d", tree.IsEditing(), shortcut)
	}

	table := NewTable()
	form = newTestForm(t, table)
	table.SetColumnCount(2)
	table.SetRowCount(2)
	table.SetCellEditTriggerF2(0, 1, true)
	shortcut = 0
	form.AddShortcut("F2", func() { shortcut++ })
	table.Focus()
	table.SetCurrentCell2(0, 1)
	typeKey(form, KeyF2, 0, KeyModifiers{})
	if table.editorTextBox == nil || shortcut != 0 {
		t.Fatalf("table: editing %v, shortcut %d", table.editorTextBox != nil, shortcut)
	}
	table.RemoveWidget(table.editorTextBox)
	table.editorTextBox = nil
	table.Focus()
	table.SetCurrentCell2(1, 1)
	typeKey(form, KeyF2, 0, KeyModifiers{})
	if table.editorTextBox != nil || shortcut != 1 {
		t.Fatalf("table, a cell without the trigger: shortcut %d", shortcut)
	}
}
