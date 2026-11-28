package regression

import (
	"image"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func copyTable(t *testing.T, selectingRows bool) *ui.Table {
	t.Helper()
	form := newForm(400, 300)
	table := ui.NewTable()
	table.SetMultiselect(true)
	table.SetSelectingRows(selectingRows)
	table.SetColumnCount(3)
	table.SetRowCount(3)
	for r, row := range [][]string{{"a", "b", "c"}, {"d", "e", "f"}, {"g", "h", "i"}} {
		for c, text := range row {
			table.SetCellText2(r, c, text)
		}
	}
	form.Panel().AddWidget(0, 0, table)
	uitest.Show(t, form)
	return table
}

// The whole selection is copied, as spreadsheets paste it
func TestTableCopySelection(t *testing.T) {
	table := copyTable(t, false)
	table.SetCurrentCell2(0, 0)
	table.SelectAll()
	table.CopySelectionToClipboard()
	if text, _ := ui.ClipboardGetText(); text != "a\tb\tc\nd\te\tf\ng\th\ti" {
		t.Errorf("all cells: %q", text)
	}

	rows := copyTable(t, true)
	rows.SetSelectedRows([]int{0, 2})
	rows.CopySelectionToClipboard()
	if text, _ := ui.ClipboardGetText(); text != "a\tb\tc\ng\th\ti" {
		t.Errorf("rows 1 and 3: %q", text)
	}
}

// iconModel gives every cell an icon
type iconModel struct {
	ui.ModelSignal
	icon image.Image
}

func (m *iconModel) RowCount() int            { return 3 }
func (m *iconModel) ColumnCount() int         { return 2 }
func (m *iconModel) Text(row, col int) string { return "x" }
func (m *iconModel) Style(row, col int) ui.ItemStyle {
	return ui.ItemStyle{Icon: m.icon}
}

// A row too low for its padding, or a negative image width, leaves no room
// for the cell and header images: they are skipped, the paint doesn't panic
func TestTableImageWithoutRoom(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, 16, 16))

	form := newForm(400, 300)
	table := ui.NewTable()
	table.SetModel(&iconModel{icon: icon})
	table.SetRowHeight(4)
	form.Panel().AddWidget(0, 0, table)
	uitest.Show(t, form).Screenshot()

	form = newForm(400, 300)
	table = ui.NewTable()
	table.SetColumnCount(1)
	table.SetRowCount(1)
	table.SetCellImage(0, 0, icon, -1)
	table.SetColumnImage(0, icon, -1)
	form.Panel().AddWidget(0, 0, table)
	w := uitest.Show(t, form)
	w.Screenshot()

	table.SetCellPadding(20)
	table.SetCellImage(0, 0, icon, 16)
	table.SetColumnImage(0, icon, 16)
	w.Screenshot()
}
