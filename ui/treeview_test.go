package ui

import (
	"reflect"
	"testing"
)

// newTestTree makes a tree in a form:
//
//	A
//	  A1
//	  A2
//	    A2a
//	B
//	C
func newTestTree(t *testing.T) (*TreeView, map[string]*TreeNode) {
	t.Helper()
	form := NewForm()
	form.processResize(400, 300)
	tree := NewTreeView()
	tree.SetColumnCount(2)
	form.Panel().AddWidget(0, 0, tree)

	n := map[string]*TreeNode{}
	n["A"] = tree.AddNode(nil, "A")
	n["A1"] = n["A"].AddChild("A1")
	n["A2"] = n["A"].AddChild("A2")
	n["A2a"] = n["A2"].AddChild("A2a")
	n["B"] = tree.AddNode(nil, "B")
	n["C"] = tree.AddNode(nil, "C")
	return tree, n
}

func shownTexts(tree *TreeView) []string {
	tree.ensureRows()
	texts := []string{}
	for _, r := range tree.rows {
		texts = append(texts, r.node.Text(0))
	}
	return texts
}

func selectedTexts(tree *TreeView) []string {
	texts := []string{}
	for _, node := range tree.SelectedNodes() {
		texts = append(texts, node.Text(0))
	}
	return texts
}

func TestTreeViewExpandCollapse(t *testing.T) {
	tree, n := newTestTree(t)
	if got := shownTexts(tree); !reflect.DeepEqual(got, []string{"A", "B", "C"}) {
		t.Fatalf("collapsed rows = %v", got)
	}
	n["A"].Expand()
	n["A2"].Expand()
	if got := shownTexts(tree); !reflect.DeepEqual(got, []string{"A", "A1", "A2", "A2a", "B", "C"}) {
		t.Fatalf("expanded rows = %v", got)
	}
	if n["A2a"].Level() != 2 || tree.rows[3].level != 2 {
		t.Errorf("A2a level = %d, row level %d", n["A2a"].Level(), tree.rows[3].level)
	}

	// Collapsing takes the focus out of the hidden children
	tree.SetCurrentNode(n["A2a"])
	n["A"].Collapse()
	if tree.CurrentNode() != n["A"] || !reflect.DeepEqual(selectedTexts(tree), []string{"A"}) {
		t.Errorf("after collapse: current %v, selected %v", tree.CurrentNode().Text(0), selectedTexts(tree))
	}
	// SetCurrentNode shows a hidden node
	tree.SetCurrentNode(n["A2a"])
	if !n["A"].IsExpanded() || !n["A2"].IsExpanded() {
		t.Error("SetCurrentNode didn't expand the parents")
	}
}

func TestTreeViewKeyboard(t *testing.T) {
	tree, n := newTestTree(t)
	press := func(key Key, mods KeyModifiers) {
		t.Helper()
		if !tree.ProcessKeyDown(key, mods) {
			t.Fatalf("key %v not handled", key)
		}
	}
	press(KeyArrowDown, KeyModifiers{}) // no current node: goes to the first
	if tree.CurrentNode() != n["A"] {
		t.Fatalf("current = %v, want A", tree.CurrentNode())
	}
	press(KeyArrowRight, KeyModifiers{}) // expands
	if !n["A"].IsExpanded() || tree.CurrentNode() != n["A"] {
		t.Fatal("Right didn't expand A")
	}
	press(KeyArrowRight, KeyModifiers{}) // goes to the first child
	if tree.CurrentNode() != n["A1"] {
		t.Fatalf("current = %s, want A1", tree.CurrentNode().Text(0))
	}
	press(KeyArrowLeft, KeyModifiers{}) // goes to the parent
	if tree.CurrentNode() != n["A"] {
		t.Fatalf("current = %s, want A", tree.CurrentNode().Text(0))
	}
	press(KeyArrowLeft, KeyModifiers{}) // collapses
	if n["A"].IsExpanded() {
		t.Fatal("Left didn't collapse A")
	}
	press(KeyEnd, KeyModifiers{})
	if tree.CurrentNode() != n["C"] {
		t.Fatalf("End: current = %s", tree.CurrentNode().Text(0))
	}
	press(KeyNumpadAsterisk, KeyModifiers{})
	press(KeyHome, KeyModifiers{})
	press(KeyNumpadAsterisk, KeyModifiers{})
	if !n["A2"].IsExpanded() {
		t.Error("* didn't expand the whole branch")
	}
	// Arrows reach nothing else when there's nowhere to go
	press(KeyArrowUp, KeyModifiers{})
	if tree.CurrentNode() != n["A"] {
		t.Errorf("Up at the top: current = %s", tree.CurrentNode().Text(0))
	}
}

func TestTreeViewMultiselect(t *testing.T) {
	tree, n := newTestTree(t)
	tree.SetMultiselect(true)
	tree.SetCurrentNode(n["A"])
	tree.ProcessKeyDown(KeyArrowDown, KeyModifiers{Shift: true})
	tree.ProcessKeyDown(KeyArrowDown, KeyModifiers{Shift: true})
	if got := selectedTexts(tree); !reflect.DeepEqual(got, []string{"A", "B", "C"}) {
		t.Errorf("Shift+Down range = %v", got)
	}
	// Ctrl+Up moves the focus only, Ctrl+Space toggles
	tree.ProcessKeyDown(KeyArrowUp, KeyModifiers{Ctrl: true})
	tree.ProcessKeyDown(KeySpace, KeyModifiers{Ctrl: true})
	if got := selectedTexts(tree); !reflect.DeepEqual(got, []string{"A", "C"}) {
		t.Errorf("after Ctrl+Space on B = %v", got)
	}
	n["A"].Expand()
	tree.ProcessKeyDown(KeyA, KeyModifiers{Ctrl: true})
	if len(tree.SelectedNodes()) != 5 {
		t.Errorf("Ctrl+A selected %v", selectedTexts(tree))
	}
	tree.SetMultiselect(false)
	if len(tree.SelectedNodes()) != 1 {
		t.Errorf("without multiselect: %v", selectedTexts(tree))
	}
}

func TestTreeViewRemove(t *testing.T) {
	tree, n := newTestTree(t)
	tree.SetCurrentNode(n["B"])
	n["B"].Remove()
	if tree.CurrentNode() != n["C"] || n["B"].Tree() != nil {
		t.Errorf("after removing B: current %v", tree.CurrentNode())
	}
	tree.SetCurrentNode(n["A2a"])
	n["A"].Remove() // removes the current node's ancestor
	if tree.CurrentNode() != n["C"] || n["A2a"].Tree() != nil {
		t.Errorf("after removing A: current %v", tree.CurrentNode())
	}
	if got := shownTexts(tree); !reflect.DeepEqual(got, []string{"C"}) {
		t.Errorf("rows = %v", got)
	}
	tree.Clear()
	if tree.CurrentNode() != nil || len(shownTexts(tree)) != 0 {
		t.Error("Clear left nodes")
	}
}

func TestTreeViewEditing(t *testing.T) {
	tree, n := newTestTree(t)
	var changes []string
	tree.SetOnCellChanged(func(node *TreeNode, col int, text string) bool {
		changes = append(changes, node.Text(0)+":"+text)
		return text != "bad"
	})
	tree.SetCurrentNode(n["B"])
	tree.Focus()

	// No trigger: F2 does nothing
	if tree.ProcessKeyDown(KeyF2, KeyModifiers{}) || tree.IsEditing() {
		t.Fatal("F2 edited without the trigger")
	}
	tree.SetEditTriggerF2(true)
	tree.ProcessKeyDown(KeyF2, KeyModifiers{})
	if !tree.IsEditing() || tree.editorTextBox.Text() != "B" {
		t.Fatal("F2 didn't open the editor on B")
	}
	// Keys the editor leaves go nowhere while editing
	tree.ProcessKeyDown(KeyArrowDown, KeyModifiers{})
	if tree.CurrentNode() != n["B"] {
		t.Fatal("Down moved away from the edited node")
	}
	tree.editorTextBox.SetText("Beta")
	tree.CommitEdit()
	if n["B"].Text(0) != "Beta" || tree.IsEditing() {
		t.Errorf("after commit: %q, editing %v", n["B"].Text(0), tree.IsEditing())
	}

	// Rejected by SetOnCellChanged
	tree.EditCurrentCell("")
	tree.editorTextBox.SetText("bad")
	tree.CommitEdit()
	if n["B"].Text(0) != "Beta" {
		t.Errorf("rejected text applied: %q", n["B"].Text(0))
	}

	// Cancel keeps the text
	tree.EditCurrentCell("")
	tree.editorTextBox.SetText("zzz")
	tree.CancelEdit()
	if n["B"].Text(0) != "Beta" {
		t.Errorf("cancelled text applied: %q", n["B"].Text(0))
	}

	// Second column, typing starts editing
	tree.SetEditTriggerKeyDown(true)
	tree.SetCurrentColumn(1)
	tree.onChar('7', KeyModifiers{})
	if !tree.IsEditing() || tree.editColumn != 1 || tree.editorTextBox.Text() != "7" {
		t.Fatal("typing didn't start editing column 1 with the typed letter")
	}
	tree.CommitEdit()
	if n["B"].Text(1) != "7" {
		t.Errorf("column 1 = %q", n["B"].Text(1))
	}

	// Read-only
	tree.SetColumnReadOnly(1, true)
	n["C"].SetReadOnly(0, true)
	if tree.IsCellEditable(n["B"], 1) || tree.IsCellEditable(n["C"], 0) || !tree.IsCellEditable(n["B"], 0) {
		t.Error("read-only cells are editable")
	}

	// Removing the edited node closes the editor
	tree.SetCurrentColumn(0)
	tree.EditCurrentCell("")
	n["B"].Remove()
	if tree.IsEditing() {
		t.Error("the editor stayed open on a removed node")
	}
	if !reflect.DeepEqual(changes, []string{"B:Beta", "Beta:bad", "Beta:7"}) {
		t.Errorf("changes = %v", changes)
	}
}

func TestTreeViewLazyLoadAndTypeAhead(t *testing.T) {
	tree, _ := newTestTree(t)
	lazy := tree.AddNode(nil, "Lazy")
	lazy.SetHasChildren(true)
	loads := 0
	tree.SetOnExpand(func(node *TreeNode) {
		if node == lazy && node.ChildCount() == 0 {
			loads++
			node.AddChild("Loaded")
		}
	})
	lazy.Expand()
	lazy.Collapse()
	lazy.Expand()
	if loads != 1 || lazy.ChildCount() != 1 {
		t.Errorf("loads = %d, children = %d", loads, lazy.ChildCount())
	}

	tree.SetCurrentNode(nil)
	tree.onChar('l', KeyModifiers{})
	if tree.CurrentNode() != lazy {
		t.Fatalf("typing l: current = %v", tree.CurrentNode())
	}
	tree.onChar('o', KeyModifiers{}) // "lo": the next match is Loaded
	if tree.CurrentNode().Text(0) != "Loaded" {
		t.Errorf("typing lo: current = %s", tree.CurrentNode().Text(0))
	}
}

func TestTreeViewMouse(t *testing.T) {
	tree, n := newTestTree(t)
	rowY := func(row int) int { return tree.headerHeight() + row*tree.RowHeight() + 2 }

	// A click on the arrow expands without selecting
	tree.onMouseDown(MouseButtonLeft, tree.arrowX(0)+2, rowY(0), KeyModifiers{})
	tree.onMouseUp(MouseButtonLeft, 0, 0, KeyModifiers{})
	if !n["A"].IsExpanded() || tree.CurrentNode() != nil {
		t.Fatalf("arrow click: expanded %v, current %v", n["A"].IsExpanded(), tree.CurrentNode())
	}
	// A click on the text selects; in the second column it sets the column
	tree.onMouseDown(MouseButtonLeft, tree.columnOffset(1)+5, rowY(2), KeyModifiers{})
	tree.onMouseUp(MouseButtonLeft, 0, 0, KeyModifiers{})
	if tree.CurrentNode() != n["A2"] || tree.CurrentColumn() != 1 {
		t.Errorf("click: current %v, column %d", tree.CurrentNode().Text(0), tree.CurrentColumn())
	}
	// A double click on a node with children toggles it and activates it
	activated := 0
	tree.SetOnNodeActivated(func(node *TreeNode) { activated++ })
	tree.onMouseDblClick(MouseButtonLeft, 60, rowY(2), KeyModifiers{})
	if !n["A2"].IsExpanded() || activated != 1 {
		t.Errorf("double click: expanded %v, activated %d", n["A2"].IsExpanded(), activated)
	}
	// Shift+click selects a range with multiselect
	tree.SetMultiselect(true)
	tree.onMouseDown(MouseButtonLeft, 60, rowY(4), KeyModifiers{Shift: true})
	tree.onMouseUp(MouseButtonLeft, 0, 0, KeyModifiers{})
	if got := selectedTexts(tree); !reflect.DeepEqual(got, []string{"A2", "A2a", "B"}) {
		t.Errorf("Shift+click = %v", got)
	}
}

// ExpandAll doesn't load children on demand: a tree whose every folder
// gets new subfolders when opened has no end
func TestTreeViewExpandAllLazy(t *testing.T) {
	tree, n := newTestTree(t)
	loads := 0
	tree.SetOnExpand(func(node *TreeNode) {
		if node.ChildCount() == 0 {
			loads++
			node.AddChild("sub").SetHasChildren(true)
		}
	})
	n["B"].SetHasChildren(true)

	tree.ExpandAll()
	if loads != 0 || !n["A2"].IsExpanded() || n["B"].IsExpanded() {
		t.Errorf("ExpandAll: loads %d, A2 expanded %v, lazy B expanded %v", loads, n["A2"].IsExpanded(), n["B"].IsExpanded())
	}
	// On a node ExpandAll loads that node only
	n["B"].ExpandAll()
	if loads != 1 || !n["B"].IsExpanded() || n["B"].Children()[0].IsExpanded() {
		t.Errorf("node ExpandAll: loads %d", loads)
	}
}

// Changes only mark the rows out of date: building a large tree and
// expanding it all doesn't rebuild the rows for each node
func TestTreeViewLargeTree(t *testing.T) {
	tree, _ := newTestTree(t)
	tree.Clear()
	for i := 0; i < 100; i++ {
		folder := tree.AddNode(nil, "folder")
		for j := 0; j < 100; j++ {
			sub := folder.AddChild("sub")
			for k := 0; k < 10; k++ {
				sub.AddChild("file")
			}
		}
	}
	tree.ExpandAll()
	if got := len(shownTexts(tree)); got != 100+100*100+100*100*10 {
		t.Fatalf("rows = %d", got)
	}
	tree.CollapseAll()
	if got := len(shownTexts(tree)); got != 100 {
		t.Fatalf("rows after CollapseAll = %d", got)
	}
	tree.Nodes()[0].RemoveChildren()
	if tree.Nodes()[0].ChildCount() != 0 {
		t.Fatal("RemoveChildren left children")
	}
}

// Scrolling right after changing the nodes uses the new size
func TestTreeViewScrollAfterChange(t *testing.T) {
	tree, _ := newTestTree(t)
	for i := 0; i < 100; i++ {
		tree.AddNode(nil, "more")
	}
	tree.SetScrollY(200)
	if tree.ScrollY() != 200 {
		t.Errorf("ScrollY = %d, want 200", tree.ScrollY())
	}
}

// typeKey sends a key press the way the platform does: the character only
// comes when the key down wasn't handled
func typeKey(form *Form, key Key, char rune, mods KeyModifiers) {
	if !form.processKeyDown(key, mods) && char != 0 {
		form.processChar(char)
	}
}

// Editing through the form's keyboard path: F2, typing, Enter
func TestTreeViewEditingThroughForm(t *testing.T) {
	tree, n := newTestTree(t)
	form := tree.Form()
	tree.SetEditTriggerF2(true)
	tree.SetCurrentNode(n["B"])
	tree.Focus()

	typeKey(form, KeyF2, 0, KeyModifiers{})
	if !tree.IsEditing() {
		t.Fatal("F2 didn't open the editor")
	}
	typeKey(form, KeyX, 'x', KeyModifiers{}) // replaces the selected text
	typeKey(form, KeyY, 'y', KeyModifiers{})
	typeKey(form, KeySpace, ' ', KeyModifiers{})
	typeKey(form, KeyZ, 'z', KeyModifiers{})
	// Navigation keys the editor leaves don't move the selection
	typeKey(form, KeyArrowDown, 0, KeyModifiers{})
	typeKey(form, KeyEnter, '\r', KeyModifiers{})
	if tree.IsEditing() || n["B"].Text(0) != "xy z" || tree.CurrentNode() != n["B"] {
		t.Errorf("after typing: editing %v, text %q, current %v", tree.IsEditing(), n["B"].Text(0), tree.CurrentNode().Text(0))
	}
	if form.FocusedWidget() != Widgeter(tree) {
		t.Error("the tree didn't get the focus back")
	}

	// Typing on the tree starts editing with the letter
	tree.SetEditTriggerKeyDown(true)
	typeKey(form, KeyQ, 'q', KeyModifiers{})
	typeKey(form, KeyW, 'w', KeyModifiers{})
	typeKey(form, KeyEsc, 0x1B, KeyModifiers{})
	if tree.IsEditing() || n["B"].Text(0) != "xy z" {
		t.Errorf("after Escape: editing %v, text %q", tree.IsEditing(), n["B"].Text(0))
	}
}
