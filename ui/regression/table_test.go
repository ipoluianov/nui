package regression

import (
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
