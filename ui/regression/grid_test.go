package regression

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

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

// Bug #17, fixed: the layout costs the occupied columns, not the numbers
// between them
func TestSparseGrid(t *testing.T) {
	form := newForm(400, 200)
	a, b := fixedPanel(50, 20), fixedPanel(50, 20)
	form.Panel().AddWidget(0, 0, a)
	form.Panel().AddWidget(1_000_000, 1_000_000, b)
	w := uitest.Show(t, form)
	start := time.Now()
	for range 20 {
		form.UpdateLayout()
		a.Width()
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Errorf("20 layouts took %v", d)
	}
	if b.X() <= a.X() || b.Y() <= a.Y() {
		t.Errorf("b at %d,%d, a at %d,%d", b.X(), b.Y(), a.X(), a.Y())
	}
	_ = w
}

func TestScrollBar(t *testing.T) {
	form := newForm(300, 300)
	bar := ui.NewScrollBar(true)
	bar.SetRange(0, 1000)
	bar.SetPageStep(50)
	form.Panel().AddWidget(0, 0, bar)
	w := uitest.Show(t, form)
	changed, activated := count(&bar.OnValueChanged), count(&bar.OnActivated)

	bar.SetValue(500)
	if bar.Value() != 500 || changed.n != 1 || activated.n != 0 {
		t.Fatalf("SetValue: %d, changed %d, activated %d", bar.Value(), changed.n, activated.n)
	}
	// The arrow at the bottom: one step
	w.Click(uitest.At(bar, bar.Width()/2, bar.Height()-3))
	if bar.Value() != 501 || activated.n != 1 {
		t.Errorf("down arrow: %d", bar.Value())
	}
	// The track below the thumb: a page
	w.Click(uitest.At(bar, bar.Width()/2, bar.Height()-bar.Width()-4))
	if bar.Value() != 551 {
		t.Errorf("page down: %d", bar.Value())
	}
	// The wheel: three steps up
	w.Wheel(bar, 0, 1)
	if bar.Value() != 548 {
		t.Errorf("wheel: %d", bar.Value())
	}
	bar.SetValue(5000)
	if bar.Value() != 1000 {
		t.Errorf("past the end: %d", bar.Value())
	}
}

// checker is a 8x8 image of 2x2 squares, to see the zoom
func checker() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			c := color.RGBA{230, 230, 230, 255}
			if (x/2+y/2)%2 == 1 {
				c = color.RGBA{40, 90, 200, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestZoomView(t *testing.T) {
	form := newForm(300, 240)
	view := ui.NewZoomView()
	view.SetImage(checker())
	view.SetSmooth(false)
	form.Panel().AddWidget(0, 0, view)
	w := uitest.Show(t, form)
	view.FitToView()
	if z := view.Zoom(); z < 25 || z > 32 {
		t.Errorf("fit 8x8 into ~%dx%d: zoom %v", view.Width(), view.Height(), z)
	}
	w.MatchGolden("zoom-view-fit")

	// Ctrl+wheel zooms around the mouse: the point under it stays
	view.SetZoom(40) // larger than the view: a smaller content is centered
	mx, my := 100, 80
	cx, cy := view.ToContent(mx, my)
	w.Shortcut("Ctrl+A") // holds Ctrl for the wheel
	w.Wheel(uitest.At(view, mx, my), 0, 2)
	cx2, cy2 := view.ToContent(mx, my)
	if math.Abs(cx-cx2) > 0.05 || math.Abs(cy-cy2) > 0.05 || view.Zoom() <= 40 {
		t.Errorf("zoom around the mouse: %.2f,%.2f -> %.2f,%.2f at zoom %v", cx, cy, cx2, cy2, view.Zoom())
	}

	// The middle button drags the content
	x0, _ := view.ToContent(0, 0)
	w.Press(ui.MouseButtonMiddle, uitest.At(view, 150, 100))
	w.Move(uitest.At(view, 110, 100))
	w.Release(ui.MouseButtonMiddle, uitest.At(view, 110, 100))
	if x1, _ := view.ToContent(0, 0); x1 <= x0 {
		t.Errorf("drag left: the content start %v -> %v", x0, x1)
	}

	// 0: the actual size
	w.Click(view)
	w.Key(ui.Key0, ui.KeyModifiers{})
	if view.Zoom() != 1 {
		t.Errorf("0: zoom %v", view.Zoom())
	}
}
