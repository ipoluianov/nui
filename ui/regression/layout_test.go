package regression

import (
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// The layout is done once for many changes, and before anything looks at
// the places of the widgets (doc/production-readiness.md 3.1)

// Bug #1, fixed: RemoveWidget lays the parent out, the widget to the right
// of the removed one moves to its place
func TestRemoveWidgetLayout(t *testing.T) {
	form := newForm(400, 200)
	p := form.Panel()
	left, right := ui.NewTextBox(), ui.NewTextBox()
	p.AddWidget(0, 0, left)
	p.AddWidget(0, 1, right)
	uitest.Show(t, form)
	before := right.X()
	p.RemoveWidget(left)
	if right.X() >= before || right.Width() <= 300 {
		t.Errorf("after removing the left one: x %d (was %d), width %d", right.X(), before, right.Width())
	}
}

// Bug #2, fixed: widgets in one grid cell lie on top of each other, each
// laid out to the cell
func TestTwoWidgetsInCell(t *testing.T) {
	form := newForm(400, 200)
	p := form.Panel()
	a, b := ui.NewButton("A"), ui.NewButton("B")
	p.AddWidget(0, 0, a)
	p.AddWidget(0, 0, b)
	p.AddVSpacer(1, 0)
	w := uitest.Show(t, form)
	if b.X() != a.X() || b.Y() != a.Y() || b.Width() != a.Width() || b.Height() != a.Height() {
		t.Errorf("A at %d,%d %dx%d, B at %d,%d %dx%d", a.X(), a.Y(), a.Width(), a.Height(), b.X(), b.Y(), b.Width(), b.Height())
	}
	// The one added last is on top: it gets the click
	clicks := 0
	b.SetOnClick(func() { clicks++ })
	w.Click(b)
	if clicks != 1 {
		t.Errorf("clicks on B: %d", clicks)
	}
}

// Hiding a widget gives its room to the others; showing it takes it back
func TestSetVisibleLaysOutSiblings(t *testing.T) {
	form := newForm(400, 200)
	p := form.Panel()
	first, second := ui.NewButton("First"), ui.NewButton("Second")
	p.AddWidget(0, 0, first)
	p.AddWidget(1, 0, second)
	p.AddVSpacer(2, 0)
	uitest.Show(t, form)
	y := second.Y()
	first.SetVisible(false)
	if second.Y() >= y {
		t.Errorf("second at y %d with the first hidden, %d before", second.Y(), y)
	}
	first.SetVisible(true)
	if second.Y() != y {
		t.Errorf("second at y %d with the first shown again, want %d", second.Y(), y)
	}
}

// A new minimum size lays the form out
func TestMinSizeLaysOut(t *testing.T) {
	form := newForm(400, 200)
	p := form.Panel()
	box := ui.NewPanel()
	box.SetMinSize(50, 40)
	next := ui.NewButton("Next")
	p.AddWidget(0, 0, box)
	p.AddWidget(1, 0, next)
	p.AddVSpacer(2, 0)
	uitest.Show(t, form)
	y := next.Y()
	box.SetMinHeight(100)
	if next.Y() != y+60 {
		t.Errorf("next at y %d, want %d", next.Y(), y+60)
	}
}

// The minimum sizes are those of the current widgets, before any frame
func TestMinSizeAfterChange(t *testing.T) {
	form := newForm(200, 100)
	p := form.Panel()
	box := ui.NewPanel()
	box.SetMinSize(50, 40)
	p.AddWidget(0, 0, box)
	uitest.Show(t, form)
	_, h := form.MinSize()
	big := ui.NewPanel()
	big.SetMinSize(50, 300)
	p.AddWidget(1, 0, big)
	if _, h2 := form.MinSize(); h2 < h+300 {
		t.Errorf("min height %d after adding 300 px, %d before", h2, h)
	}
	if got := p.MinHeight(); got < 340 {
		t.Errorf("panel min height %d", got)
	}
}

// Adding many widgets to a shown form lays it out once, not per widget
func TestAddManyWidgets(t *testing.T) {
	form := newForm(400, 300)
	w := uitest.Show(t, form)
	p := form.Panel()
	var last *ui.Label
	for i := range 2000 {
		last = ui.NewLabel("Label")
		p.AddWidget(i, 0, last)
	}
	w.Settle()
	if last.Y() <= 0 {
		t.Errorf("the last label at y %d: not laid out", last.Y())
	}
}

// RequestRepaint from a worker goroutine: many requests, one repaint
func TestRequestRepaintFromGoroutine(t *testing.T) {
	form := newForm(200, 100)
	box := paintBox(func(cnv *ui.Canvas) {})
	form.Panel().AddWidget(0, 0, box)
	w := uitest.Show(t, form)
	done := make(chan struct{})
	go func() {
		for range 1000 {
			box.RequestRepaint()
		}
		close(done)
	}()
	<-done
	if n := countPosted(w); n > 1 {
		t.Errorf("%d repaints posted for 1000 requests", n)
	}
}

// countPosted runs the posted functions and returns how many there were
func countPosted(w *uitest.Window) int {
	return uitest.RunPosted()
}
