package regression

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// rowPoint is the middle of the row of the list, in the window
func rowPoint(list *ui.ListView, row int) uitest.Point {
	_, y, w, h := list.ItemRect(row)
	return uitest.At(list, w/2, y+h/2)
}

func TestListView(t *testing.T) {
	form := newForm(300, 300)
	list := ui.NewListView()
	model := ui.NewListModel("zero", "one", "two", "three")
	list.SetModel(model)
	list.SetMultiselect(true)
	form.Panel().AddWidget(0, 0, list)
	w := uitest.Show(t, form)
	selection, activated := count(&list.OnSelectionChanged), count(&list.OnActivated)

	w.Click(rowPoint(list, 1))
	if list.CurrentRow() != 1 || selection.last != 1 {
		t.Fatalf("click: current %d", list.CurrentRow())
	}
	w.Key(ui.KeyArrowDown, ui.KeyModifiers{Shift: true})
	if got := list.SelectedRows(); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("shift+down: %v", got)
	}
	w.Press(ui.MouseButtonLeft, rowPoint(list, 0)) // Ctrl+click adds
	w.Release(ui.MouseButtonLeft, rowPoint(list, 0))
	w.Key(ui.KeyEnter, ui.KeyModifiers{})
	if activated.n != 1 || activated.last != 0 {
		t.Errorf("Enter: activated %d (%d)", activated.n, activated.last)
	}
	w.DoubleClick(rowPoint(list, 3))
	if activated.last != 3 {
		t.Errorf("double click: %d", activated.last)
	}

	// The model changes: the list follows
	model.Remove(3)
	if list.RowCount() != 3 || list.CurrentRow() != 2 {
		t.Errorf("after Remove: %d rows, current %d", list.RowCount(), list.CurrentRow())
	}
	model.Append("four", nil)
	w.Key(ui.KeyEnd, ui.KeyModifiers{})
	if list.CurrentRow() != 3 {
		t.Errorf("End: %d", list.CurrentRow())
	}
}

// A list of a million rows: the model is asked only for the visible ones
func TestListViewVirtual(t *testing.T) {
	form := newForm(300, 300)
	list := ui.NewListView()
	asked := 0
	list.SetModel(ui.NewFuncModel(func() int { return 1_000_000 }, 1, func(row, col int) string {
		asked++
		return fmt.Sprint("Row ", row)
	}))
	form.Panel().AddWidget(0, 0, list)
	w := uitest.Show(t, form)
	w.Click(list)
	w.Key(ui.KeyEnd, ui.KeyModifiers{})
	asked = 0
	w.Screenshot()
	if list.CurrentRow() != 999_999 || asked == 0 || asked > 50 {
		t.Errorf("current %d, texts asked for a frame: %d", list.CurrentRow(), asked)
	}
}

// coloredModel styles its rows
type coloredModel struct{ *ui.FuncModel }

func (m coloredModel) Style(row, col int) ui.ItemStyle {
	if row%2 == 1 {
		return ui.ItemStyle{Color: color.RGBA{200, 0, 0, 255}, Bold: true}
	}
	return ui.ItemStyle{}
}

func TestListViewGolden(t *testing.T) {
	form := newForm(240, 200)
	list := ui.NewListView()
	list.SetModel(coloredModel{ui.NewFuncModel(func() int { return 20 }, 1, func(row, col int) string {
		return fmt.Sprint("Item ", row)
	})})
	form.Panel().AddWidget(0, 0, list)
	empty := ui.NewListView()
	empty.SetEmptyText("No results")
	form.Panel().AddWidget(1, 0, empty)
	w := uitest.Show(t, form)
	w.Click(rowPoint(list, 2))
	w.MatchGolden("listview")
}

// editableModel is a table of its own
type editableModel struct {
	ui.ModelSignal
	cells [][]string
}

func (m *editableModel) RowCount() int            { return len(m.cells) }
func (m *editableModel) ColumnCount() int         { return 2 }
func (m *editableModel) Text(row, col int) string { return m.cells[row][col] }
func (m *editableModel) ColumnName(col int) string {
	return []string{"Name", "Value"}[col]
}
func (m *editableModel) SetText(row, col int, text string) bool {
	if col == 0 {
		return false // the names are fixed
	}
	m.cells[row][col] = text
	return true
}

func TestTableModel(t *testing.T) {
	form := newForm(400, 300)
	table := ui.NewTable()
	model := &editableModel{cells: [][]string{{"a", "1"}, {"b", "2"}}}
	table.SetModel(model)
	form.Panel().AddWidget(0, 0, table)
	uitest.Show(t, form)
	if table.RowCount() != 2 || table.GetCellText2(1, 0) != "b" {
		t.Fatalf("rows %d, cell %q", table.RowCount(), table.GetCellText2(1, 0))
	}
	changed := count(&table.OnCellChanged)
	table.SetCellText2(1, 1, "20")
	table.SetCellText2(1, 0, "x") // rejected
	if model.cells[1][1] != "20" || model.cells[1][0] != "b" || changed.n != 1 {
		t.Errorf("edits: %v, changed %d", model.cells, changed.n)
	}
	model.cells = append(model.cells, []string{"c", "3"})
	model.Changed()
	if table.RowCount() != 3 || table.GetCellText2(2, 1) != "3" {
		t.Errorf("after Changed: %d rows", table.RowCount())
	}
}
