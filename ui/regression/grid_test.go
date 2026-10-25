package regression

import (
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func fixedPanel(w, h int) *ui.Panel {
	p := ui.NewPanel()
	p.SetMinSize(w, h)
	return p
}

// A widget spanning columns takes them all, the padding between them too;
// the columns get the width it needs
func TestGridSpan(t *testing.T) {
	form := newForm(500, 300)
	grid := form.Panel()
	grid.SetCellPadding(6)
	a, b := fixedPanel(80, 20), fixedPanel(60, 20)
	grid.AddWidget(0, 0, a)
	grid.AddWidget(0, 1, b)
	wide := fixedPanel(300, 20)
	grid.AddWidgetSpan(1, 0, 1, 2, wide)
	tall := fixedPanel(40, 60)
	grid.AddWidgetSpan(0, 2, 2, 1, tall)
	grid.AddVSpacer(2, 0)
	grid.AddHSpacer(0, 3)
	uitest.Show(t, form)

	if wide.X() != a.X() || wide.X()+wide.Width() != b.X()+b.Width() {
		t.Errorf("wide: %d..%d, columns %d..%d", wide.X(), wide.X()+wide.Width(), a.X(), b.X()+b.Width())
	}
	if wide.Width() < 300 {
		t.Errorf("wide is %d, needs 300", wide.Width())
	}
	if tall.Y() != a.Y() || tall.Y()+tall.Height() != wide.Y()+wide.Height() {
		t.Errorf("tall: %d..%d, rows %d..%d", tall.Y(), tall.Y()+tall.Height(), a.Y(), wide.Y()+wide.Height())
	}
	if r, c := wide.GridSpan(); r != 1 || c != 2 {
		t.Errorf("GridSpan %d, %d", r, c)
	}
}

// The columns share the free width by their stretch weights
func TestGridStretch(t *testing.T) {
	form := newForm(400, 100)
	grid := form.Panel()
	grid.SetPanelPadding(0)
	grid.SetCellPadding(0)
	one, two := fixedPanel(0, 20), fixedPanel(0, 20)
	grid.AddWidget(0, 0, one)
	grid.AddWidget(0, 1, two)
	grid.SetColumnStretch(0, 1)
	grid.SetColumnStretch(1, 3)
	grid.AddVSpacer(1, 0)
	w := uitest.Show(t, form)
	if one.Width() != 100 || two.Width() != 300 {
		t.Errorf("1:3 of 400: %d and %d", one.Width(), two.Width())
	}
	w.Resize(800, 100)
	if one.Width() != 200 || two.Width() != 600 {
		t.Errorf("1:3 of 800: %d and %d", one.Width(), two.Width())
	}
}

// The boxes and the button box of a dialog sized by its widgets
type boxDialog struct {
	ui.DialogContent
	name    *ui.TextBox
	ok      bool
	buttons *ui.DialogButtonBox
}

func newBoxDialog() *boxDialog {
	var c boxDialog
	c.InitWidget()
	fields := ui.NewFormLayout()
	c.name = ui.NewTextBox()
	fields.AddRow("Name:", c.name)
	fields.AddRow("Port:", ui.NewNumBox())
	fields.AddRowWidget(ui.NewCheckbox("Remember the password"))
	c.AddWidget(0, 0, fields)
	c.AddVSpacer(1, 0)
	c.buttons = ui.NewDialogButtonBox()
	c.buttons.AddOK(func() { c.ok = true; c.Form().Close() })
	c.buttons.AddCancel(func() { c.Form().Close() })
	c.AddWidget(2, 0, c.buttons)
	c.OnDialogShow = func() { c.Form().SetTitle("Box dialog") }
	return &c
}

func TestBoxesAndDialogButtons(t *testing.T) {
	form := newForm(800, 600)
	w := uitest.Show(t, form)
	dialog := newBoxDialog()
	form.Panel().ShowDialog(dialog)
	d := w.Dialog()
	width, height := d.Size()
	if width < 320 || width > 500 || height > 250 {
		t.Errorf("dialog sized by its widgets: %dx%d", width, height)
	}
	// The fields fill the width; the button box is at the right edge
	if dialog.name.X()+dialog.name.Width() < width-20 {
		t.Errorf("the name box ends at %d of %d", dialog.name.X()+dialog.name.Width(), width)
	}
	cancel := dialog.buttons.Button(ui.ButtonReject)
	if x, _ := cancel.RectClientAreaOnWindow(); x+cancel.Width() < width-10 {
		t.Errorf("Cancel ends at x %d of %d", x+cancel.Width(), width)
	}
	d.MatchGolden("box-dialog")
	// Enter is the OK button
	d.Click(dialog.name)
	d.Type("x\n")
	if !dialog.ok || !d.Closed() {
		t.Errorf("Enter: ok %v, closed %v", dialog.ok, d.Closed())
	}
}

func TestHBoxVBox(t *testing.T) {
	form := newForm(400, 300)
	col := ui.NewVBox()
	row := ui.NewHBox()
	a := row.Add(ui.NewButton("A")).(*ui.Button)
	row.AddStretch()
	b := row.Add(ui.NewButton("B")).(*ui.Button)
	col.Add(row)
	col.AddSpacing(30)
	c := col.Add(ui.NewButton("C")).(*ui.Button)
	col.AddStretch()
	form.Panel().AddWidget(0, 0, col)
	uitest.Show(t, form)
	if b.X()+b.Width() < row.Width()-2 || a.X() > 2 {
		t.Errorf("row: A at %d, B ends at %d of %d", a.X(), b.X()+b.Width(), row.Width())
	}
	if c.Y()-(row.Y()+row.Height()) < 30 {
		t.Errorf("spacing: C at %d, the row ends at %d", c.Y(), row.Y()+row.Height())
	}
}
