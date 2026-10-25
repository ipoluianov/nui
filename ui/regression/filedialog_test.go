package regression

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// fileDialogNames are the names the dialog lists
func fileDialogNames(d *uitest.Window) []string {
	table := d.FindAll(func(w ui.Widgeter) bool { _, ok := w.(*ui.Table); return ok })[0].(*ui.Table)
	var names []string
	m := table.Model()
	for row := range m.RowCount() {
		names = append(names, m.Text(row, 0))
	}
	return names
}

func fileDialogDir(t *testing.T) string {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "sub"), 0o755)
	for _, name := range []string{"b.txt", "A.txt", "c.md", ".hidden.txt", "sub/inner.txt"} {
		os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644)
	}
	return dir
}

func TestBuiltinOpenFileDialog(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	dir := fileDialogDir(t)
	form := newForm(400, 300)
	w := uitest.Show(t, form)

	var got []string
	calls := 0
	form.ShowOpenFileDialog(ui.OpenFileDialogOptions{
		Title: "Open", DefaultDirectory: dir, AllowMultiple: true,
		Filters: []ui.FileDialogFilter{{DisplayName: "Text", Patterns: []string{"*.txt"}}, {DisplayName: "All", Patterns: []string{"*"}}},
	}, func(paths []string, err error) { got, calls = paths, calls+1 })
	d := w.Dialog()
	if d == nil {
		t.Fatal("no dialog")
	}
	// Folders first, by name ignoring the case; the filter and the hidden files
	if names := fileDialogNames(d); !slices.Equal(names, []string{"sub", "A.txt", "b.txt"}) {
		t.Errorf("listed %v", names)
	}
	d.Click(d.Find("Show hidden files"))
	if names := fileDialogNames(d); !slices.Equal(names, []string{"sub", ".hidden.txt", "A.txt", "b.txt"}) {
		t.Errorf("with hidden: %v", names)
	}
	d.Click(d.Find("Show hidden files"))

	// Into the folder by Enter on its row, back by the up button
	d.Click(d.FindAll(func(x ui.Widgeter) bool { _, ok := x.(*ui.Table); return ok })[0])
	d.Key(ui.KeyHome, ui.KeyModifiers{})
	d.Key(ui.KeyEnter, ui.KeyModifiers{})
	if names := fileDialogNames(d); !slices.Equal(names, []string{"inner.txt"}) {
		t.Errorf("in sub: %v", names)
	}
	d.Click(d.Find("Up"))

	// Two names typed: two paths, after the dialog closes
	name := d.FindAll(func(x ui.Widgeter) bool { tb, ok := x.(*ui.TextBox); return ok && tb.Text() == "" })[0]
	d.Click(name)
	d.Type(`"A.txt" "b.txt"`)
	d.Key(ui.KeyEnter, ui.KeyModifiers{})
	w.Settle()
	if calls != 1 || !slices.Equal(got, []string{filepath.Join(dir, "A.txt"), filepath.Join(dir, "b.txt")}) {
		t.Errorf("result %v (%d calls)", got, calls)
	}
	if w.Blocked() {
		t.Error("the dialog stays open")
	}

	// Cancelled by Esc: the result is empty, once
	form.ShowOpenFileDialog(ui.OpenFileDialogOptions{DefaultDirectory: dir}, func(paths []string, err error) { got, calls = paths, calls+1 })
	w.Dialog().Key(ui.KeyEsc, ui.KeyModifiers{})
	w.Settle()
	if calls != 2 || got != nil {
		t.Errorf("after Esc: %v (%d calls)", got, calls)
	}
}

func TestBuiltinSaveFileDialog(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	dir := fileDialogDir(t)
	form := newForm(400, 300)
	w := uitest.Show(t, form)

	var got string
	save := func(name string) {
		form.ShowSaveFileDialog(ui.SaveFileDialogOptions{DefaultDirectory: dir, DefaultFileName: name,
			Filters: []ui.FileDialogFilter{{DisplayName: "Text", Patterns: []string{"*.txt"}}}},
			func(path string, err error) { got = path })
	}

	// The name selected at the start; the extension of the filter added
	save("notes")
	w.Dialog().Key(ui.KeyEnter, ui.KeyModifiers{})
	w.Settle()
	if got != filepath.Join(dir, "notes.txt") {
		t.Errorf("saved as %q", got)
	}

	// An existing file: asked before replacing
	got = ""
	save("b.txt")
	d := w.Dialog()
	d.Key(ui.KeyEnter, ui.KeyModifiers{})
	q := d.Dialog()
	if q == nil {
		t.Fatal("not asked to replace")
	}
	q.Click(q.Find("No"))
	if got != "" || !w.Blocked() {
		t.Errorf("replaced after No: %q", got)
	}
	d.Key(ui.KeyEnter, ui.KeyModifiers{})
	d.Dialog().Click(d.Dialog().Find("Yes"))
	w.Settle()
	if got != filepath.Join(dir, "b.txt") {
		t.Errorf("after Yes: %q", got)
	}

	// A folder typed is opened
	save("")
	d = w.Dialog()
	d.Type("sub")
	d.Key(ui.KeyEnter, ui.KeyModifiers{})
	if names := fileDialogNames(d); !slices.Equal(names, []string{"inner.txt"}) {
		t.Errorf("after typing a folder: %v", names)
	}
}

func TestBuiltinSelectDirectoryDialog(t *testing.T) {
	ui.UseBuiltinFileDialogs(true)
	defer ui.UseBuiltinFileDialogs(false)
	dir := fileDialogDir(t)
	form := newForm(400, 300)
	w := uitest.Show(t, form)

	var got string
	form.ShowSelectDirectoryDialog(ui.SelectDirectoryDialogOptions{DefaultDirectory: dir}, func(path string, err error) { got = path })
	d := w.Dialog()
	if names := fileDialogNames(d); !slices.Equal(names, []string{"sub"}) {
		t.Errorf("listed %v", names)
	}
	d.Click(d.Find("Select Folder"))
	w.Settle()
	if got != dir {
		t.Errorf("selected %q, want %q", got, dir)
	}
}
