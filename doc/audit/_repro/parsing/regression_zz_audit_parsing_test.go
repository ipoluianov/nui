package regression

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// A folder dialog lists a symlink to a folder; the target goes away before
// the user activates the row: the row is no longer a folder, activate()
// treats it as a file and writes to d.name, which a folder dialog lacks.
func TestAuditDirDialogDanglingLink(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	os.Mkdir(real, 0o755)
	os.Symlink(real, filepath.Join(dir, "link"))

	form := newForm(400, 300)
	w := uitest.Show(t, form)
	form.ShowSelectDirectoryDialog(ui.SelectDirectoryDialogOptions{Title: "Pick", DefaultDirectory: dir},
		func(path string, err error) {})
	d := w.Dialog()
	table := d.FindAll(func(x ui.Widgeter) bool { _, ok := x.(*ui.Table); return ok })[0].(*ui.Table)
	t.Logf("rows: %d", table.Model().RowCount())
	os.Remove(real) // the link dangles now
	d.Click(table)
	d.Key(ui.KeyEnd, ui.KeyModifiers{})
	d.Key(ui.KeyEnter, ui.KeyModifiers{}) // panics
}

// A state saved on a screen that is gone, left of the main one (negative
// X): SetState promises to keep the window on the screen
func TestAuditFormStateNegativeX(t *testing.T) {
	uitest.SetScreen(1920, 1080, 1)
	for _, st := range []ui.FormState{
		{X: 5000, Y: 100, Width: 400, Height: 300},
		{X: -3000, Y: 100, Width: 400, Height: 300},
		{X: 100, Y: -3000, Width: 400, Height: 300},
	} {
		form := newForm(400, 300)
		form.SetState(st)
		uitest.Show(t, form)
		got := form.State()
		t.Logf("saved %+v -> shown at %d,%d", st, got.X, got.Y)
		if got.X < 0 || got.Y < 0 || got.X > 1920-100 || got.Y > 1080-50 {
			t.Errorf("saved %+v: window off the screen at %d,%d", st, got.X, got.Y)
		}
		form.Close()
	}
}

// A folder dialog whose default directory doesn't exist: the list is empty,
// the path is empty, and OK gives "" - the same as Cancel.
func TestAuditDirDialogMissingDefault(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	form := newForm(400, 300)
	w := uitest.Show(t, form)
	got, calls := "x", 0
	form.ShowSelectDirectoryDialog(ui.SelectDirectoryDialogOptions{Title: "Pick", DefaultDirectory: "/nonexistent/dir"},
		func(path string, err error) { got, calls = path, calls+1 })
	d := w.Dialog()
	d.Click(d.Find("Select Folder"))
	w.Settle()
	t.Logf("calls=%d path=%q", calls, got)
	if calls == 1 && got == "" {
		t.Errorf("OK in the folder dialog gave an empty path (looks like Cancel)")
	}
}

// Open dialog: a file whose name starts with a quote can't be opened by
// activating it: the name field is split on quotes.
func TestAuditOpenDialogQuoteName(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	dir := t.TempDir()
	name := `"q" x.txt`
	os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644)
	form := newForm(400, 300)
	w := uitest.Show(t, form)
	var got []string
	calls := 0
	form.ShowOpenFileDialog(ui.OpenFileDialogOptions{Title: "Open", DefaultDirectory: dir},
		func(paths []string, err error) { got, calls = paths, calls+1 })
	d := w.Dialog()
	table := d.FindAll(func(x ui.Widgeter) bool { _, ok := x.(*ui.Table); return ok })[0].(*ui.Table)
	d.Click(table)
	d.Key(ui.KeyHome, ui.KeyModifiers{})
	d.Key(ui.KeyEnter, ui.KeyModifiers{})
	w.Settle()
	t.Logf("calls=%d got=%q blocked=%v", calls, got, w.Blocked())
	if calls != 1 || len(got) != 1 || got[0] != filepath.Join(dir, name) {
		t.Errorf("file %q not opened: calls=%d got=%q", name, calls, got)
	}
}
