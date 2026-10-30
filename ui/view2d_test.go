package ui_test

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// newView2DForm shows a view of a drawing: a red disc of radius 10 at
// (20, 10), and many small black lines around it
func newView2DForm(t *testing.T) (*ui.View2D, *uitest.Window, *int) {
	view := ui.NewView2D()
	view.SetBackground(color.RGBA{255, 255, 255, 255})
	paints := 0
	view.SetPainter(func(p *ui.View2DPaint) {
		paints++
		lastVisible = p.Visible
		disc := ui.NewPath()
		disc.Circle(20, 10, 10)
		p.Canvas.FillPath(disc, color.RGBA{220, 30, 30, 255})
		lines := ui.NewPath()
		for i := -50; i <= 50; i++ {
			lines.MoveTo(float64(i)*3, -40)
			lines.LineTo(float64(i)*3+2, -38)
		}
		p.Canvas.StrokePath(lines, ui.StrokeStyle{Width: 1, NonScaling: true}, color.RGBA{0, 0, 0, 255})
	})
	view.SetExtents(ui.RectF{MinX: 10, MinY: 0, MaxX: 30, MaxY: 20})
	form := ui.NewForm()
	form.SetSize(400, 300)
	form.Panel().AddWidget(0, 0, view)
	w := uitest.Show(t, form)
	return view, w, &paints
}

// lastVisible is the part of the drawing the last paint drew
var lastVisible ui.RectF

func samePoint(a, b ui.PointF) bool { return math.Hypot(a.X-b.X, a.Y-b.Y) < 1e-6 }

// The coordinates of the drawing and of the widget convert both ways, y up
// and down
func TestView2DCoordinates(t *testing.T) {
	view, _, _ := newView2DForm(t)
	view.SetView(ui.PointF{X: 100, Y: 50}, 4)
	c := view.ToScreen(ui.PointF{X: 100, Y: 50})
	if math.Abs(c.X-float64(view.Width())/2) > 1e-9 || math.Abs(c.Y-float64(view.Height())/2) > 1e-9 {
		t.Fatalf("the center is at %v", c)
	}
	up := view.ToScreen(ui.PointF{X: 100, Y: 51})
	if math.Abs(up.Y-(c.Y-4)) > 1e-9 {
		t.Fatalf("y up: %v", up)
	}
	for _, yUp := range []bool{true, false} {
		view.SetYUp(yUp)
		p := ui.PointF{X: 123.25, Y: -7.5}
		if q := view.ToWorld(view.ToScreen(p).X, view.ToScreen(p).Y); !samePoint(p, q) {
			t.Fatalf("y up %v: %v back as %v", yUp, p, q)
		}
		m := view.WorldMatrix()
		if !samePoint(m.ApplyPoint(p), view.ToScreen(p)) {
			t.Fatal("WorldMatrix differs from ToScreen")
		}
	}
	r := view.VisibleRect()
	if math.Abs(r.Width()-float64(view.Width())/4) > 1e-9 {
		t.Fatalf("visible %v", r)
	}
}

// ZoomExtents shows the extents; the wheel zooms keeping the point under
// the pointer; the middle button drags; the arrows move
func TestView2DControls(t *testing.T) {
	view, w, _ := newView2DForm(t)
	view.ZoomExtents()
	w.Settle()
	center := view.ToScreen(ui.PointF{X: 20, Y: 10})
	if math.Abs(center.X-float64(view.Width())/2) > 0.5 {
		t.Fatalf("the extents are not centered: %v", center)
	}
	img := w.Screenshot()
	at := uitest.At(view, int(center.X), int(center.Y))
	if c := img.RGBAAt(at.X, at.Y); c.R < 200 || c.G > 60 {
		t.Fatalf("the disc at the center is %v", c)
	}

	scale := view.Scale()
	pt := uitest.At(view, 100, 80)
	under := view.ToWorld(100.5, 80.5)
	w.Wheel(pt, 0, 2)
	if view.Scale() <= scale {
		t.Fatal("the wheel up did not zoom in")
	}
	if !samePoint(view.ToWorld(100.5, 80.5), under) {
		t.Fatalf("the point under the pointer moved: %v -> %v", under, view.ToWorld(100.5, 80.5))
	}

	before := view.Center()
	a, b := uitest.At(view, 200, 150), uitest.At(view, 260, 110)
	w.Move(a)
	w.Press(ui.MouseButtonMiddle, a)
	w.Move(b)
	w.Release(ui.MouseButtonMiddle, b)
	after := view.Center()
	k := view.Scale()
	if math.Abs((before.X-after.X)*k-60) > 1e-6 || math.Abs((before.Y-after.Y)*k-40) > 1e-6 {
		t.Fatalf("the drag of 60, -40 moved the center by %v, %v pixels", (before.X-after.X)*k, (before.Y-after.Y)*k)
	}

	view.Focus()
	c0 := view.Center()
	w.Key(ui.KeyArrowRight, ui.KeyModifiers{})
	if view.Center().X <= c0.X {
		t.Fatal("the right arrow did not move the view right")
	}
	w.Key(ui.KeyHome, ui.KeyModifiers{})
	if !samePoint(view.Center(), ui.PointF{X: 20, Y: 10}) {
		t.Fatalf("Home: %v", view.Center())
	}
}

// The picture is painted again only when the view or the drawing change;
// a drag paints the strips that come in, and the result is the same as a
// whole paint
func TestView2DCacheAndScroll(t *testing.T) {
	view, w, paints := newView2DForm(t)
	view.SetView(ui.PointF{X: 0, Y: 0}, 3)
	w.Screenshot()
	n := *paints
	view.Update()
	w.Screenshot()
	if *paints != n {
		t.Fatalf("a repaint painted the drawing again: %d paints", *paints-n)
	}
	view.Invalidate()
	w.Screenshot()
	if *paints != n+1 {
		t.Fatalf("Invalidate: %d paints", *paints-n)
	}
	// Drag by whole pixels: painted in strips
	for i, d := range [][2]int{{17, 0}, {0, -9}, {-23, 31}} {
		a := uitest.At(view, 200, 150)
		b := uitest.At(view, 200+d[0], 150+d[1])
		w.Move(a)
		w.Press(ui.MouseButtonMiddle, a)
		before := *paints
		w.Move(b)
		w.Release(ui.MouseButtonMiddle, b)
		scrolled := w.Screenshot()
		full := view.VisibleRect()
		if *paints == before || lastVisible.Width() > full.Width()/4 && lastVisible.Height() > full.Height()/4 {
			t.Fatalf("drag %d: %d paints, the last of %v of %v", i, *paints-before, lastVisible, full)
		}
		view.Invalidate()
		whole := w.Screenshot()
		r := image.Rect(uitest.At(view, 0, 0).X, uitest.At(view, 0, 0).Y, uitest.At(view, view.Width(), view.Height()).X, uitest.At(view, view.Width(), view.Height()).Y)
		if d := maxDiff(scrolled, whole, r); d > 4 {
			t.Fatalf("drag %d: the scrolled picture differs from a whole paint by %d", i, d)
		}
	}
}

func maxDiff(a, b *image.RGBA, r image.Rectangle) int {
	m := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ca, cb := a.RGBAAt(x, y), b.RGBAAt(x, y)
			for _, d := range []int{int(ca.R) - int(cb.R), int(ca.G) - int(cb.G), int(ca.B) - int(cb.B)} {
				m = max(m, d, -d)
			}
		}
	}
	return m
}

// A box dragged to the right selects inside, to the left crossing; the
// application takes the press for itself by Handled
func TestView2DBoxSelection(t *testing.T) {
	view, w, _ := newView2DForm(t)
	view.SetBoxSelection(true)
	view.SetView(ui.PointF{}, 1)
	var got []ui.BoxSelection
	view.OnBoxSelected.Connect(func(b ui.BoxSelection) { got = append(got, b) })
	w.Drag(uitest.At(view, 100, 100), uitest.At(view, 150, 140))
	w.Drag(uitest.At(view, 150, 140), uitest.At(view, 100, 100))
	if len(got) != 2 || got[0].Crossing || !got[1].Crossing {
		t.Fatalf("boxes %+v", got)
	}
	r := got[0].Rect
	if math.Abs(r.Width()-50) > 1e-6 || math.Abs(r.Height()-40) > 1e-6 {
		t.Fatalf("box %v", r)
	}
	view.OnMouseDown.Connect(func(e *ui.MouseEvent) { e.Handled = true })
	w.Drag(uitest.At(view, 100, 100), uitest.At(view, 150, 140))
	if len(got) != 2 {
		t.Fatal("a box was dragged on a press the application took")
	}
	// A zoom window zooms to the box
	view2, w2, _ := newView2DForm(t)
	view2.SetView(ui.PointF{}, 1)
	view2.StartZoomWindow()
	w2.Drag(uitest.At(view2, 100, 100), uitest.At(view2, 140, 130))
	if view2.Scale() < 5 {
		t.Fatalf("zoom window: scale %v", view2.Scale())
	}
}

// AddText reads upright with y up, at the size asked
func TestView2DText(t *testing.T) {
	view, _, _ := newView2DForm(t)
	p := ui.NewPath()
	view.AddText(p, ui.PointF{X: 10, Y: 20}, "Hello", ui.FontFamilySans, 5, 0, ui.HAlignLeft, ui.VAlignBottom)
	x0, y0, _, y1, ok := p.Bounds()
	if !ok || x0 < 10 || y0 < 20-0.01 || y1 > 26 {
		t.Fatalf("text bounds %v %v %v", x0, y0, y1)
	}
}

// On a screen of the scale 2 and 1.5 the strips meet the moved picture
func TestView2DScrollHiDPI(t *testing.T) {
	defer uitest.SetScreen(1920, 1080, 1)
	for _, scale := range []float64{2, 1.5} {
		uitest.SetScreen(1920, 1080, scale)
		view, w, _ := newView2DForm(t)
		view.SetView(ui.PointF{X: 3, Y: -20}, 2.5)
		w.Screenshot()
		for i, d := range [][2]int{{10, 0}, {0, 7}, {-12, -6}, {3, 5}} {
			a := uitest.At(view, 200, 150)
			b := uitest.At(view, 200+d[0], 150+d[1])
			w.Move(a)
			w.Press(ui.MouseButtonMiddle, a)
			w.Move(b)
			w.Release(ui.MouseButtonMiddle, b)
			scrolled := w.Screenshot()
			view.Invalidate()
			whole := w.Screenshot()
			// The antialiasing of a strip may differ by a level or two
			if d := maxDiff(scrolled, whole, scrolled.Rect); d > 4 {
				t.Fatalf("scale %v drag %d: the scrolled picture differs from a whole paint by %d", scale, i, d)
			}
		}
	}
}
