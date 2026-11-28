package regression

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// A folder dialog whose default folder does not exist: it used to show no
// folder, and OK gave "", the same as Cancel. Now it opens the nearest
// folder that exists, and OK gives that.
func TestDirDialogMissingDefaultFallsBack(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	dir := t.TempDir()
	form := newForm(400, 300)
	w := uitest.Show(t, form)
	got, calls := "x", 0
	form.ShowSelectDirectoryDialog(ui.SelectDirectoryDialogOptions{Title: "Pick", DefaultDirectory: filepath.Join(dir, "gone", "deeper")},
		func(path string, err error) { got, calls = path, calls+1 })
	d := w.Dialog()
	d.Click(d.Find("Select Folder"))
	w.Settle()
	if calls != 1 || got != dir {
		t.Errorf("calls=%d path=%q, want the nearest existing folder %q", calls, got, dir)
	}
}

// findTable returns the table of the file dialog
func findTable(t *testing.T, d *uitest.Window) *ui.Table {
	t.Helper()
	tables := d.FindAll(func(x ui.Widgeter) bool { _, ok := x.(*ui.Table); return ok })
	if len(tables) == 0 {
		t.Fatal("no table in the dialog")
	}
	return tables[0].(*ui.Table)
}

// A file whose name starts with a quote, activated with Enter: its name
// used to be parsed back from the name field and split at the quotes, so
// the dialog looked for "q" and said it was not found.
func TestOpenDialogActivateQuotedName(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	dir := t.TempDir()
	name := `"q" x.txt`
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
		t.Skip("such a name can't be made here:", err)
	}
	form := newForm(400, 300)
	w := uitest.Show(t, form)
	var got []string
	calls := 0
	form.ShowOpenFileDialog(ui.OpenFileDialogOptions{Title: "Open", DefaultDirectory: dir},
		func(paths []string, err error) { got, calls = paths, calls+1 })
	d := w.Dialog()
	d.Click(findTable(t, d))
	d.Key(ui.KeyHome, ui.KeyModifiers{})
	d.Key(ui.KeyEnter, ui.KeyModifiers{})
	w.Settle()
	if calls != 1 || len(got) != 1 || got[0] != filepath.Join(dir, name) {
		t.Errorf("file %q not opened: calls=%d got=%q", name, calls, got)
	}
}

// Several files selected, one with a quote in its name and one starting
// with a space: the names put in the field in quotes used to be split
// wrong, and the space trimmed. Now the selected names are taken as they
// are while the field is not edited.
func TestOpenDialogMultiSelectOddNames(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	dir := t.TempDir()
	names := []string{` lead.txt`, `a "b".txt`}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Skip("such a name can't be made here:", err)
		}
	}
	form := newForm(400, 300)
	w := uitest.Show(t, form)
	var got []string
	calls := 0
	form.ShowOpenFileDialog(ui.OpenFileDialogOptions{Title: "Open", DefaultDirectory: dir, AllowMultiple: true},
		func(paths []string, err error) { got, calls = paths, calls+1 })
	d := w.Dialog()
	findTable(t, d).SetSelectedRows([]int{0, 1})
	d.Click(d.Find("Open"))
	w.Settle()
	var want []string
	for _, n := range names {
		want = append(want, filepath.Join(dir, n))
	}
	slices.Sort(got)
	slices.Sort(want)
	if calls != 1 || !slices.Equal(got, want) {
		t.Errorf("calls=%d got=%q, want %q (blocked=%v)", calls, got, want, w.Blocked())
	}
}

// A state saved on a screen left of or above the main one (a negative X or
// Y) used to be neither restored nor kept on the screen: the window opened
// centered. Now it is moved there, kept on the screens there are.
func TestFormStateNegativePositionClamped(t *testing.T) {
	uitest.SetScreen(1920, 1080, 1)
	for _, c := range []struct {
		state        ui.FormState
		wantX, wantY int
	}{
		{ui.FormState{X: -3000, Y: 100, Width: 400, Height: 300}, 0, 100},
		{ui.FormState{X: 100, Y: -3000, Width: 400, Height: 300}, 100, 0},
		{ui.FormState{X: -50, Y: -60, Width: 400, Height: 300}, 0, 0},
		{ui.FormState{X: 5000, Y: 100, Width: 400, Height: 300}, 1920 - 400, 100},
	} {
		form := newForm(400, 300)
		form.SetState(c.state)
		uitest.Show(t, form)
		got := form.State()
		if got.X != c.wantX || got.Y != c.wantY {
			t.Errorf("saved %+v: shown at %d,%d, want %d,%d", c.state, got.X, got.Y, c.wantX, c.wantY)
		}
		form.Close()
	}
}
