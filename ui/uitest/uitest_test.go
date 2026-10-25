package uitest_test

import (
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// A small form: a field, a check box, a combo box and buttons
type testForm struct {
	form   *ui.Form
	name   *ui.TextBox
	agree  *ui.Checkbox
	combo  *ui.ComboBox
	clicks int
	menu   string
}

func newTestForm() *testForm {
	f := &testForm{form: ui.NewForm()}
	f.form.SetTitle("Test")
	f.form.SetSize(400, 260)
	p := f.form.Panel()
	p.AddLabel(0, 0, "Name:")
	f.name = ui.NewTextBox()
	f.name.SetName("name")
	p.AddWidget(0, 1, f.name)
	f.agree = ui.NewCheckbox("I agree")
	p.AddWidget(1, 1, f.agree)
	f.combo = ui.NewComboBox()
	f.combo.AddItem("Red", nil)
	f.combo.AddItem("Green", nil)
	f.combo.AddItem("Blue", nil)
	p.AddWidget(2, 1, f.combo)
	p.AddButton(3, 1, "Count", func() { f.clicks++ })
	menu := ui.NewContextMenu(nil)
	menu.AddItem("Copy", func() { f.menu = "copy" })
	menu.AddItem("Paste", func() { f.menu = "paste" })
	f.name.SetContextMenu(menu)
	p.AddVSpacer(4, 0)
	return f
}

func TestClickAndType(t *testing.T) {
	f := newTestForm()
	w := uitest.Show(t, f.form)

	w.Click(w.Find("Count"))
	w.Click(w.Find("Count"))
	if f.clicks != 2 {
		t.Errorf("clicks = %d", f.clicks)
	}

	w.Click(w.Find("name"))
	if w.Focused() != ui.Widgeter(f.name) {
		t.Fatalf("focused: %v", w.Focused())
	}
	w.Type("Hello, World")
	w.Key(ui.KeyBackspace, ui.KeyModifiers{})
	if got := f.name.Text(); got != "Hello, Worl" {
		t.Errorf("text = %q", got)
	}
	w.Shortcut("Ctrl+A")
	w.Type("Привет")
	if got := f.name.Text(); got != "Привет" {
		t.Errorf("text after select all = %q", got)
	}

	w.Click(f.agree)
	if !f.agree.Checked() {
		t.Error("the check box is not checked")
	}
}

// The combo box's list is a popup window: it opens on a click, an item is
// chosen in it
func TestPopup(t *testing.T) {
	f := newTestForm()
	w := uitest.Show(t, f.form)
	w.Click(f.combo)
	if len(f.form.Panel().PopupWidgets) != 1 {
		t.Fatalf("popups: %d", len(f.form.Panel().PopupWidgets))
	}
	// The list's items are painted, not widgets: chosen with the keys
	w.Key(ui.KeyArrowDown, ui.KeyModifiers{})
	w.Key(ui.KeyArrowDown, ui.KeyModifiers{})
	w.Key(ui.KeyEnter, ui.KeyModifiers{})
	if f.combo.SelectedIndex() != 2 {
		t.Errorf("selected %d", f.combo.SelectedIndex())
	}
	if len(f.form.Panel().PopupWidgets) != 0 {
		t.Error("the list stays open")
	}

	// The context menu of the field
	w.RightClick(f.name)
	w.Click(w.Find("Paste"))
	if f.menu != "paste" {
		t.Errorf("menu: %q", f.menu)
	}
}

// A message box is a modal window over the form: the form gets no input
// until it is closed
func TestModalDialog(t *testing.T) {
	f := newTestForm()
	w := uitest.Show(t, f.form)
	ui.ShowMessageBox(f.form.Panel(), "Note", "Saved")
	w.Settle()
	box := w.Dialog()
	if box.Title() != "Note" || !w.Blocked() {
		t.Fatalf("dialog %v, blocked %v", box, w.Blocked())
	}
	w.Click(w.Find("Count"))
	if f.clicks != 0 {
		t.Error("the form got a click under the modal dialog")
	}
	box.Click(box.Find("OK"))
	if !box.Closed() || w.Blocked() {
		t.Errorf("after OK: closed %v, blocked %v", box.Closed(), w.Blocked())
	}
	w.Click(w.Find("Count"))
	if f.clicks != 1 {
		t.Error("the form gets no clicks after the dialog")
	}
}

// Invoke from another goroutine runs when the test settles
func TestInvoke(t *testing.T) {
	f := newTestForm()
	w := uitest.Show(t, f.form)
	done := make(chan struct{})
	go func() {
		f.form.Invoke(func() { f.name.SetText("from a goroutine") })
		close(done)
	}()
	<-done
	if f.name.Text() != "" {
		t.Fatal("ran before Settle")
	}
	w.Settle()
	if f.name.Text() != "from a goroutine" {
		t.Errorf("text = %q", f.name.Text())
	}
}

func TestGolden(t *testing.T) {
	f := newTestForm()
	w := uitest.Show(t, f.form)
	w.Click(f.agree)
	w.MatchGolden("test-form")
	w.Click(f.combo)
	w.MatchGolden("test-form-combo-open")
}
