package ui

import (
	"testing"
	"time"
)

type auditRowsModel struct {
	ModelSignal
	rows, cols int
}

func (m *auditRowsModel) RowCount() int            { return m.rows }
func (m *auditRowsModel) ColumnCount() int         { return m.cols }
func (m *auditRowsModel) Text(row, col int) string { return "x" }

func TestAuditTableModelShrinkToZeroSignal(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	m := &auditRowsModel{rows: 10, cols: 2}
	table.SetModel(m)
	table.SetCurrentCell2(5, 0)
	var got []TableCell
	table.OnSelectionChanged.Connect(func(c TableCell) { got = append(got, c) })
	m.rows = 0
	m.Changed()
	t.Logf("after shrink: CurrentRow=%d currentCellY=%d signals=%v", table.CurrentRow(), table.currentCellY, got)
	if table.CurrentRow() == -1 && len(got) == 0 {
		t.Errorf("current row went 5 -> -1 without OnSelectionChanged")
	}
	m.rows = 10
	m.Changed()
	t.Logf("after regrow: CurrentRow=%d selected=%v", table.CurrentRow(), table.SelectedRows())
	if table.CurrentRow() == 5 {
		t.Errorf("stale current row 5 came back after the model was emptied and refilled")
	}
}

func TestAuditTableShrinkPageUpStuck(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	table.SetColumnCount(1)
	table.SetRowCount(100)
	table.SetCurrentCell2(90, 0)
	table.SetRowCount(10)
	table.ProcessKeyDown(KeyPageUp, KeyModifiers{})
	if table.CurrentRow() < 0 {
		t.Errorf("PageUp after SetRowCount(10) with current row 90: CurrentRow=%d currentCellY=%d", table.CurrentRow(), table.currentCellY)
	}
}

func TestAuditTableEditorRowRemoved(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	m := &auditRowsModel{rows: 10, cols: 2}
	table.SetModel(m)
	table.SetCurrentCell2(5, 1)
	table.EditCurrentCell("")
	ed := table.editorTextBox
	if ed == nil {
		t.Fatal("no editor")
	}
	y0 := ed.Y()
	m.rows = 3
	m.Changed()
	t.Logf("editor y before %d after %d, still open %v", y0, ed.Y(), table.editorTextBox != nil)
	if table.editorTextBox != nil && ed.Y() < table.headerHeight() {
		t.Errorf("the editor of the removed row 5 stays open over the header at y=%d", ed.Y())
	}
}

func TestAuditTableManyColumnsPaint(t *testing.T) {
	for _, n := range []int{250, 500, 1000} {
		table := NewTable()
		form := newTestForm(t, table)
		table.SetColumnCount(n)
		table.SetRowCount(20)
		for i := 0; i < n; i++ {
			table.SetColumnWidth(i, 60)
		}
		start := time.Now()
		auditPaint(form)
		p := time.Since(start)
		start = time.Now()
		table.cellByPosition(60*n-1, 40)
		cb := time.Since(start)
		t.Logf("%d columns: paint %v, cellByPosition %v", n, p, cb)
	}
}

func TestAuditTableSelectAllBigModel(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	m := &auditRowsModel{rows: 1_000_000, cols: 5}
	table.SetModel(m)
	table.SetMultiselect(true)
	table.SetSelectingRows(false)
	start := time.Now()
	table.ProcessKeyDown(KeyA, KeyModifiers{Ctrl: true})
	t.Logf("Ctrl+A on 1M x 5 in cell mode: %v, %d entries", time.Since(start), len(table.selectedCells))
}

func TestAuditTreeRemoveChildrenSelected(t *testing.T) {
	for _, n := range []int{2000, 8000} {
		tree := NewTreeView()
		newTestForm(t, tree)
		tree.SetMultiselect(true)
		root := tree.AddNode(nil, "root")
		for i := 0; i < n; i++ {
			root.AddChild("c")
		}
		root.Expand()
		tree.SelectAll()
		start := time.Now()
		root.RemoveChildren()
		t.Logf("%d selected children: RemoveChildren %v", n, time.Since(start))
	}
}

func TestAuditTreeReentrantRemove(t *testing.T) {
	tree := NewTreeView()
	newTestForm(t, tree)
	a := tree.AddNode(nil, "a")
	tree.AddNode(nil, "b")
	tree.SetCurrentNode(a)
	saved := a
	once := false
	tree.OnSelectionChanged.Connect(func(n *TreeNode) {
		if !once {
			once = true
			tree.SetCurrentNode(saved) // restore a remembered selection
		}
	})
	a.Remove()
	if cur := tree.CurrentNode(); cur != nil && cur.Tree() == nil {
		t.Errorf("the current node is a removed node %q (Tree()==nil)", cur.Text(0))
	}
}

func TestAuditTreeReentrantRemoveSame(t *testing.T) {
	tree := NewTreeView()
	newTestForm(t, tree)
	a := tree.AddNode(nil, "a")
	tree.AddNode(nil, "b")
	tree.SetCurrentNode(a)
	tree.OnSelectionChanged.Connect(func(n *TreeNode) {
		if a.Tree() != nil {
			a.Remove() // "remove a if it is still in the tree"
		}
	})
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	a.Remove()
}
