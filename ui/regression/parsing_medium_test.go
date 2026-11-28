package regression

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// A folder dialog lists a link to a folder whose target goes away before
// the row is activated: the row is no longer a folder, and activating it
// used to treat it as a file and write to the name field a folder dialog
// does not have (a nil dereference on the UI thread). Now the list is
// read again and the dialog stays open.
func TestDirDialogDanglingLinkActivate(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "link")); err != nil {
		t.Skip("no symlinks:", err)
	}

	form := newForm(400, 300)
	w := uitest.Show(t, form)
	finished := false
	form.ShowSelectDirectoryDialog(ui.SelectDirectoryDialogOptions{Title: "Pick", DefaultDirectory: dir},
		func(path string, err error) { finished = true })
	d := w.Dialog()
	if d == nil {
		t.Fatal("no dialog")
	}
	table := d.FindAll(func(x ui.Widgeter) bool { _, ok := x.(*ui.Table); return ok })[0].(*ui.Table)
	if n := table.Model().RowCount(); n != 2 {
		t.Fatalf("rows before: %d, want 2 (real, link)", n)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	d.Click(table)
	d.Key(ui.KeyEnd, ui.KeyModifiers{})
	d.Key(ui.KeyEnter, ui.KeyModifiers{})
	if finished {
		t.Error("the dialog finished on a row that is not a folder")
	}
	if n := table.Model().RowCount(); n != 0 {
		t.Errorf("rows after: %d, want 0 (the dangling link is no folder)", n)
	}
}
