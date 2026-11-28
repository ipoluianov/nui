package ui

import (
	"image"
	"image/color"
	"testing"
	"time"
)

func auditPaint(form *Form) {
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	form.paintScaled(img, 1)
}

type auditIconModel struct {
	ModelSignal
	rows int
	icon image.Image
}

func (m *auditIconModel) RowCount() int            { return m.rows }
func (m *auditIconModel) ColumnCount() int         { return 2 }
func (m *auditIconModel) Text(row, col int) string { return "x" }
func (m *auditIconModel) Style(row, col int) ItemStyle {
	return ItemStyle{Icon: m.icon}
}

// Small row height with a model icon: imgHeight negative -> uint overflow in resize
func TestAuditTableSmallRowIcon(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	icon := image.NewRGBA(image.Rect(0, 0, 16, 16))
	table.SetModel(&auditIconModel{rows: 3, icon: icon})
	table.SetRowHeight(4) // padding 3 -> rowHeight-2*padding = -2
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		auditPaint(form)
	}()
	select {
	case r := <-done:
		if r != nil {
			t.Fatalf("paint panicked: %v", r)
		}
	case <-time.After(20 * time.Second):
		t.Fatalf("paint hangs")
	}
}

// A zero-height image in a cell
func TestAuditTableEmptyImage(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	table.SetColumnCount(1)
	table.SetRowCount(1)
	table.SetCellImage(0, 0, image.NewRGBA(image.Rect(0, 0, 10, 0)), 16)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("paint panicked: %v", r)
		}
	}()
	auditPaint(form)
}

func TestAuditTableNegativeImageWidth(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	table.SetColumnCount(1)
	table.SetRowCount(1)
	table.SetCellImage(0, 0, image.NewRGBA(image.Rect(0, 0, 10, 10)), -1)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("paint panicked: %v", r)
		}
	}()
	auditPaint(form)
}

// SetCurrentCell2 out of range creates cells
func TestAuditTableSetCurrentCellLeak(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	table.SetColumnCount(2)
	table.SetRowCount(2)
	for i := 0; i < 1000; i++ {
		table.SetCurrentCell2(100+i, 100+i)
		table.SetCurrentCell2(-1-i, 0)
	}
	if len(table.rows) > 2 {
		t.Errorf("out-of-range SetCurrentCell2 made %d rows of cells", len(table.rows))
	}
}

// Tree: a lazy loader that selects the first loaded child recurses forever
func TestAuditTreeExpandingSelectsChild(t *testing.T) {
	tree := NewTreeView()
	newTestForm(t, tree)
	folder := tree.AddNode(nil, "folder")
	folder.SetHasChildren(true)
	calls := 0
	tree.OnExpanding.Connect(func(n *TreeNode) {
		calls++
		if calls > 50 {
			return // stop the recursion for the test
		}
		child := n.AddChild("file")
		tree.SetCurrentNode(child)
	})
	folder.Expand()
	if calls != 1 {
		t.Errorf("OnExpanding called %d times for one Expand (re-entrant Expand via SetCurrentNode)", calls)
	}
	if n := folder.ChildCount(); n != 1 {
		t.Errorf("children loaded %d times", n)
	}
}

// ListView: anchor left past the end after the model shrank
func TestAuditListViewStaleAnchor(t *testing.T) {
	list := NewListView()
	newTestForm(t, list)
	list.SetMultiselect(true)
	model := NewListModel("a", "b", "c", "d", "e", "f", "g", "h", "i", "j")
	list.SetModel(model)
	list.SetCurrentRow(9)
	model.SetItems([]string{"a", "b", "c"})
	// Shift+Up extends from the anchor
	list.ProcessKeyDown(KeyArrowUp, KeyModifiers{Shift: true})
	for _, r := range list.SelectedRows() {
		if r >= list.RowCount() {
			t.Errorf("selected rows %v past the end (%d rows)", list.SelectedRows(), list.RowCount())
			break
		}
	}
}

// ComboBox: a pick in an open popup after the items were removed
func TestAuditComboStalePopupPick(t *testing.T) {
	cb := NewComboBox()
	newTestForm(t, cb)
	cb.AddItem("a", nil)
	cb.AddItem("b", nil)
	cb.AddItem("c", nil)
	var activated []int
	userChanged := 0
	cb.OnActivated.Connect(func(i int) { activated = append(activated, i) })
	cb.SetOnSelectedIndexChanged(func() { userChanged++ })
	cb.OpenPopup()
	popup := cb.form.Panel().PopupWidgets[len(cb.form.Panel().PopupWidgets)-1].(*comboBoxPopup)
	cb.Clear() // e.g. the items reloaded by a timer while the list is open
	popup.processKeys(KeyEnd)
	popup.processKeys(KeyEnter)
	if len(activated) > 0 && activated[0] >= cb.ItemCount() {
		t.Errorf("OnActivated(%d) for a combo box of %d items; SelectedIndexChanged called %d", activated[0], cb.ItemCount(), userChanged)
	}
}

type auditF32 struct {
	F float32
}

func TestAuditPropertyGridFloat32Overflow(t *testing.T) {
	grid := NewPropertyGrid()
	newTestForm(t, grid)
	obj := &auditF32{F: 1}
	if err := grid.SetObject(obj); err != nil {
		t.Fatal(err)
	}
	box := grid.Property("F").Editor().(*NumBox)
	box.SetValue(1e300)
	grid.Property("F").NotifyChanged()
	if f := float64(obj.F); f > 3.5e38 || f < -3.5e38 {
		t.Errorf("float32 field got %v from 1e300 (editor shows %v)", obj.F, box.Value())
	}
}

var _ = color.RGBA{}
