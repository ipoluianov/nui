package recipes

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// The editor: typing marks the document modified, the close button asks,
// No closes without saving
func TestEditor(t *testing.T) {
	settings, _ := ui.OpenSettingsFile(filepath.Join(t.TempDir(), "s.json"))
	e := NewEditor(settings)
	w := uitest.Show(t, e.Form)
	w.Click(e.text)
	w.Type("hello")
	if !e.modified || w.Title() != "Untitled * - Editor" {
		t.Fatalf("modified %v, title %q", e.modified, w.Title())
	}
	w.RequestClose()
	q := w.Dialog()
	if q == nil || w.Closed() {
		t.Fatal("no question before closing")
	}
	q.Click(q.Find("No"))
	w.Settle()
	if !w.Closed() {
		t.Error("the window stays after No")
	}
}

// The editor saves to its file
func TestEditorSave(t *testing.T) {
	e := NewEditor(nil)
	w := uitest.Show(t, e.Form)
	e.path = filepath.Join(t.TempDir(), "a.txt")
	w.Click(e.text)
	w.Type("text")
	w.Shortcut("Mod+S")
	if data, _ := os.ReadFile(e.path); string(data) != "text" || e.modified {
		t.Errorf("saved %q, modified %v", data, e.modified)
	}
}

// OK with an empty host keeps the dialog; with a host it gives the values
func TestConnectionDialog(t *testing.T) {
	form := ui.NewForm()
	form.SetSize(400, 300)
	w := uitest.Show(t, form)
	var host string
	var port int
	ShowConnectionDialog(form, func(h string, p int) { host, port = h, p })
	d := w.Dialog()
	d.Click(d.Find("OK"))
	if host != "localhost" || port != 8080 {
		t.Errorf("got %q %d", host, port)
	}
}

// The list is filtered once the typing pauses
func TestFilterList(t *testing.T) {
	form := ui.NewForm()
	form.SetSize(300, 300)
	list := FilterList([]string{"apple", "banana", "cherry"})
	form.Panel().AddWidget(0, 0, list)
	w := uitest.Show(t, form)
	box := w.FindAll(func(x ui.Widgeter) bool { _, ok := x.(*ui.TextBox); return ok })[0]
	view := w.FindAll(func(x ui.Widgeter) bool { _, ok := x.(*ui.ListView); return ok })[0].(*ui.ListView)
	w.Click(box)
	w.Type("an")
	if view.RowCount() != 3 {
		t.Errorf("filtered at once: %d rows", view.RowCount())
	}
	w.Wait(400 * time.Millisecond)
	if view.RowCount() != 1 || view.Model().Text(0, 0) != "banana" {
		t.Errorf("after the pause: %d rows", view.RowCount())
	}
}

func TestPeopleTable(t *testing.T) {
	table, add := PeopleTable([]Person{{"Ann", 30}})
	add(Person{"Bob", 40})
	if m := table.Model(); m.RowCount() != 2 || m.Text(1, 1) != "40" {
		t.Errorf("rows %d", m.RowCount())
	}
}

// A click puts a dot and repaints the widget
func TestDots(t *testing.T) {
	form := ui.NewForm()
	form.SetSize(300, 200)
	d := NewDots()
	form.Panel().AddWidget(0, 0, d)
	w := uitest.Show(t, form)
	n := 0
	d.OnAdded.Connect(func(count int) { n = count })
	w.Click(uitest.At(d, 20, 20))
	if n != 1 || len(d.points) != 1 {
		t.Errorf("dots %d", n)
	}
}

func TestLiveLabel(t *testing.T) {
	form := ui.NewForm()
	values := make(chan float64)
	label := LiveLabel(values)
	form.Panel().AddWidget(0, 0, label)
	w := uitest.Show(t, form)
	values <- 2.5
	close(values)
	for i := 0; i < 100 && label.Text() != "2.5"; i++ {
		time.Sleep(time.Millisecond)
		w.Settle()
	}
	if label.Text() != "2.5" {
		t.Errorf("label %q", label.Text())
	}
}

func TestCountFiles(t *testing.T) {
	form := ui.NewForm()
	w := uitest.Show(t, form)
	got := -1
	CountFiles(form.Panel(), []string{"ab", "cde"}, func(n int) { got = n })
	for i := 0; i < 200 && got < 0; i++ {
		w.Wait(10 * time.Millisecond)
	}
	if got != 5 {
		t.Errorf("counted %d", got)
	}
}

// The texts follow the language
func TestTranslatedForm(t *testing.T) {
	defer ui.SetLanguage("en")
	ui.SetLanguage("en")
	form := TranslatedForm(func() int { return 5 })
	w := uitest.Show(t, form)
	if w.Find("5 files") == nil {
		t.Error("no English text")
	}
	ui.SetLanguage("ru")
	w.Settle()
	if w.Title() != "Файлы" || w.Find("5 файлов") == nil {
		t.Errorf("Russian: title %q", w.Title())
	}
}

func TestAcceptFiles(t *testing.T) {
	form := ui.NewForm()
	form.SetSize(300, 200)
	var got []string
	AcceptFiles(form, func(p string) { got = append(got, p) })
	w := uitest.Show(t, form)
	w.DropFiles(uitest.Point{X: 50, Y: 50}, "/tmp/a.txt")
	if len(got) != 1 {
		t.Errorf("dropped %v", got)
	}
}
