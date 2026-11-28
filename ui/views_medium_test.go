package ui

import (
	"image"
	"math"
	"slices"
	"testing"
	"time"
)

// Regression tests of the medium findings of the item views audit
// (doc/audit/views.md, VIEW-3 to VIEW-10).

type vmRowsModel struct {
	ModelSignal
	rows, cols int
}

func (m *vmRowsModel) RowCount() int            { return m.rows }
func (m *vmRowsModel) ColumnCount() int         { return m.cols }
func (m *vmRowsModel) Text(row, col int) string { return "x" }

func vmPaint(form *Form) {
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	form.paintScaled(img, 1)
}

// VIEW-3: a selection handler run by Remove sees the node out of the tree:
// it can neither remove it again (that panicked) nor make it current.
func TestTreeRemoveHandlerSeesNodeRemoved(t *testing.T) {
	tree := NewTreeView()
	newTestForm(t, tree)
	a := tree.AddNode(nil, "a")
	tree.AddNode(nil, "b")
	tree.SetCurrentNode(a)
	inTree := true
	tree.OnSelectionChanged.Connect(func(*TreeNode) {
		inTree = a.Tree() != nil
		a.Remove()             // "remove a if still there": a no-op now
		tree.SetCurrentNode(a) // "restore the selection": rejected
	})
	a.Remove()
	if inTree {
		t.Error("the removed node is still in the tree while the handlers run")
	}
	if cur := tree.CurrentNode(); cur == nil || cur.Tree() == nil || cur.Text(0) != "b" {
		t.Errorf("current node %v, want b", cur)
	}
	if tree.IsNodeSelected(a) {
		t.Error("the removed node is selected")
	}
}

// VIEW-3: Remove of a node that is not among its siblings anymore is a no-op
func TestTreeRemoveTwiceInHandler(t *testing.T) {
	tree := NewTreeView()
	newTestForm(t, tree)
	a := tree.AddNode(nil, "a")
	b := tree.AddNode(nil, "b")
	tree.SetCurrentNode(a)
	tree.OnSelectionChanged.Connect(func(*TreeNode) {
		if a.Tree() != nil {
			a.Remove()
		}
	})
	a.Remove()
	if n := len(tree.nodes); n != 1 || tree.nodes[0] != b {
		t.Errorf("top nodes %d, want b only", n)
	}
}

// VIEW-4: the Shift anchor is clamped when the model shrinks
func TestListViewAnchorClampedOnShrink(t *testing.T) {
	list := NewListView()
	newTestForm(t, list)
	list.SetMultiselect(true)
	model := NewListModel("a", "b", "c", "d", "e", "f", "g", "h", "i", "j")
	list.SetModel(model)
	list.SetCurrentRow(9)
	model.SetItems([]string{"a", "b", "c"})
	list.ProcessKeyDown(KeyArrowUp, KeyModifiers{Shift: true})
	if got := list.SelectedRows(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("selected rows %v, want [1 2]", got)
	}
}

// VIEW-5: a model that shrinks to nothing clears the current cell with a
// signal, and the old one doesn't come back when the rows do
func TestTableShrinkToZeroSignals(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	m := &vmRowsModel{rows: 10, cols: 2}
	table.SetModel(m)
	table.SetCurrentCell2(5, 0)
	var got []TableCell
	table.OnSelectionChanged.Connect(func(c TableCell) { got = append(got, c) })
	m.rows = 0
	m.Changed()
	if len(got) != 1 || got[0].Row != -1 {
		t.Errorf("signals after the shrink %v, want one of row -1", got)
	}
	m.rows = 10
	m.Changed()
	if r := table.CurrentRow(); r != 0 {
		t.Errorf("current row after refill %d, want 0 (as a new table)", r)
	}
	if sel := table.SelectedRows(); !slices.Equal(sel, []int{0}) {
		t.Errorf("selected %v, want the first row", sel)
	}
}

// VIEW-5: a shrink moves the current cell to the last row and column, once
// signalled, and the keyboard goes on from there
func TestTableShrinkClampsCurrentCell(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	table.SetColumnCount(1)
	table.SetRowCount(100)
	table.SetCurrentCell2(90, 0)
	signals := 0
	table.OnSelectionChanged.Connect(func(TableCell) { signals++ })
	table.SetRowCount(10)
	if r := table.CurrentRow(); r != 9 || signals != 1 {
		t.Errorf("after SetRowCount(10): current row %d, %d signals; want 9, 1", r, signals)
	}
	if sel := table.SelectedRows(); !slices.Equal(sel, []int{9}) {
		t.Errorf("selected %v, want row 9", sel)
	}
	table.ProcessKeyDown(KeyPageUp, KeyModifiers{})
	if r := table.CurrentRow(); r < 0 || r >= 9 {
		t.Errorf("PageUp after the shrink: current row %d", r)
	}

	m := &vmRowsModel{rows: 5, cols: 10}
	table.SetModel(m)
	table.SetCurrentCell2(2, 8)
	m.cols = 2
	m.Changed()
	if col := table.CurrentColumn(); col != 1 {
		t.Errorf("current column after the columns shrank to 2: %d, want 1", col)
	}
	table.ProcessKeyDown(KeyArrowLeft, KeyModifiers{})
	if col := table.CurrentColumn(); col != 0 {
		t.Errorf("ArrowLeft after the shrink: column %d, want 0", col)
	}
}

// VIEW-6: the editors of a cell that is gone close instead of lying over
// the header
func TestTableEditorClosedWhenCellRemoved(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	m := &vmRowsModel{rows: 10, cols: 2}
	table.SetModel(m)
	table.SetCurrentCell2(5, 1)
	table.EditCurrentCell("")
	ed := table.editorTextBox
	if ed == nil {
		t.Fatal("no editor")
	}
	m.rows = 3
	m.Changed()
	if table.editorTextBox != nil || slices.ContainsFunc(table.innerWidgets, func(in *innerWidget) bool { return in.widget == ed }) {
		t.Error("the editor of the removed row is still open")
	}

	done := 0
	accepted := true
	table.EditCell(2, 1, 1, "t", func(_ string, ok bool) { done++; accepted = ok })
	if !table.IsEditingCell() {
		t.Fatal("EditCell opened nothing")
	}
	m.cols = 1
	m.Changed()
	if table.IsEditingCell() || done != 1 || accepted {
		t.Errorf("EditCell editor of a removed column: open %v, done %d, accepted %v", table.IsEditingCell(), done, accepted)
	}
}

// VIEW-7: reading a cell (current cell checks, hot tracking) makes no cell
// objects
func TestTableReadsCreateNoCells(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	table.SetColumnCount(2)
	table.SetRowCount(2)
	prevX, prevY := table.PreviousCurrentCellX(), table.PreviousCurrentCellY()
	for i := range 100 {
		table.SetCurrentCell2(100+i, 100+i)
		table.SetCurrentCell2(-1-i, 0)
	}
	if n := len(table.rows); n != 0 {
		t.Errorf("out-of-range SetCurrentCell2 made %d rows of cells", n)
	}
	if table.PreviousCurrentCellX() != prevX || table.PreviousCurrentCellY() != prevY {
		t.Error("a rejected SetCurrentCell2 changed the previous cell")
	}

	m := &vmRowsModel{rows: 1000, cols: 3}
	table.SetModel(m)
	table.SetHotTracking(true)
	for y := table.headerHeight(); y < 300; y += 5 {
		table.onMouseMove(10, y, KeyModifiers{})
		table.onMouseMove(250, y, KeyModifiers{})
	}
	table.ClearSelection()
	table.ProcessKeyDown(KeyEnter, KeyModifiers{})
	vmPaint(form)
	if n := len(table.rows); n != 0 {
		t.Errorf("hovering a model table made %d rows of cells", n)
	}
}

// VIEW-8: a pick from a popup opened before the items changed reports
// nothing for an item that is gone, and the item itself where it moved
func TestComboStalePopupPick(t *testing.T) {
	cb := NewComboBox()
	newTestForm(t, cb)
	cb.SetItems([]string{"a", "b", "c"})
	var activated []int
	changed := 0
	cb.OnActivated.Connect(func(i int) { activated = append(activated, i) })
	cb.SetOnSelectedIndexChanged(func() { changed++ })
	popup := func() *comboBoxPopup {
		cb.OpenPopup()
		pw := cb.form.Panel().PopupWidgets
		return pw[len(pw)-1].(*comboBoxPopup)
	}

	p := popup()
	cb.Clear()
	p.processKeys(KeyEnd)
	p.processKeys(KeyEnter)
	if len(activated) != 0 || changed != 0 {
		t.Errorf("pick of a removed item: OnActivated %v, changed %d", activated, changed)
	}
	cb.form.CloseTopPopup()

	cb.AddItem("a", nil)
	cb.AddItem("b", nil)
	cb.AddItem("c", nil)
	p = popup()
	cb.RemoveItem(0)
	p.processKeys(KeyEnd) // "c", now at 1
	p.processKeys(KeyEnter)
	if !slices.Equal(activated, []int{1}) || cb.SelectedItemText() != "c" {
		t.Errorf("pick of c after a removed before it: OnActivated %v, selected %q", activated, cb.SelectedItemText())
	}
}

type vmFloats struct {
	F   float32
	Big float32 `max:"1e300"`
	D   float64
}

// VIEW-9: a float32 field stays finite whatever the editor is given
func TestPropertyGridFloat32Limits(t *testing.T) {
	grid := NewPropertyGrid()
	newTestForm(t, grid)
	obj := &vmFloats{F: 1, Big: 1, D: 1}
	grid.SetObject(obj)
	for _, name := range []string{"F", "Big"} {
		box := grid.Property(name).Editor().(*NumBox)
		box.SetValue(1e300)
		grid.Property(name).NotifyChanged()
	}
	box := grid.Property("D").Editor().(*NumBox)
	box.SetValue(1e300)
	grid.Property("D").NotifyChanged()
	if math.IsInf(float64(obj.F), 0) || math.IsInf(float64(obj.Big), 0) {
		t.Errorf("float32 fields got %v, %v from 1e300", obj.F, obj.Big)
	}
	if obj.F != math.MaxFloat32 || obj.D != 1e300 {
		t.Errorf("F = %v (want MaxFloat32), D = %v (want 1e300)", obj.F, obj.D)
	}
}

// VIEW-10: the cached column offsets and the binary search give what the
// widths give, with the stretched last column and widths of 0 and below
func TestTableColumnOffsetsCached(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	table.SetColumnCount(6)
	table.SetRowCount(3)
	check := func(stage string) {
		t.Helper()
		off := 0
		for col := range table.columnCount + 1 {
			if got := table.columnOffset(col); got != off {
				t.Fatalf("%s: columnOffset(%d) = %d, want %d", stage, col, got, off)
			}
			off += table.ownColumnWidth(col)
		}
		// The column under each x, by a linear scan
		for x := -5; x < table.contentWidth()+300; x += 3 {
			want := -1
			for col := range table.columnCount {
				o := table.columnOffset(col)
				if x >= o && x < o+table.columnWidth(col) {
					want = col
					break
				}
			}
			if got := table.columnAtX(x); got != want {
				t.Fatalf("%s: columnAtX(%d) = %d, want %d", stage, x, got, want)
			}
		}
	}
	check("default")
	table.SetColumnWidth(1, 0)
	table.SetColumnWidth(2, 0)
	table.SetColumnWidth(4, 37)
	check("widths")
	table.SetStretchLastColumn(true)
	check("stretched")
	table.SetColumnWidth(3, -20)
	check("negative")
	table.SetColumnCount(3)
	check("fewer columns")
}

// VIEW-10: a table of many columns paints only those in the view, and a
// header span past the columns is clamped
func TestTableManyColumnsBounded(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	table.SetColumnCount(100_000)
	table.SetRowCount(20)
	first, last := table.visibleColumns(table.scrollX, table.Width())
	if first != 0 || last-first > table.Width()/table.defaultColumnWidth+2 {
		t.Errorf("visible columns [%d, %d) of a %d wide view", first, last, table.Width())
	}
	table.SetColumnCount(2)
	table.SetHeaderCellSpan2(0, 0, 1, 50_000_000)
	if span, _ := table.headerSpan(0, 0, table.lookupHeaderCell(0, 0)); span != 2 {
		t.Errorf("header span %d over 2 columns", span)
	}
	start := time.Now()
	vmPaint(form)
	if d := time.Since(start); d > time.Second {
		t.Errorf("a paint with a huge header span took %v", d)
	}
}

// VIEW-10: removing many selected nodes is linear, not quadratic
func TestTreeRemoveChildrenSelectedLinear(t *testing.T) {
	tree := NewTreeView()
	newTestForm(t, tree)
	tree.SetMultiselect(true)
	root := tree.AddNode(nil, "root")
	for range 30_000 {
		root.AddChild("c")
	}
	root.Expand()
	tree.SelectAll()
	start := time.Now()
	root.RemoveChildren()
	// About 10 s before the fix, a few ms after
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("RemoveChildren of 30000 selected children took %v", d)
	}
	if root.ChildCount() != 0 || len(tree.selected) != 1 || !tree.selected[root] {
		t.Errorf("children %d, selected %v", root.ChildCount(), tree.selected)
	}
}
