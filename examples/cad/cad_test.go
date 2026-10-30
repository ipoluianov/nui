package cad

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func newTestApp(t *testing.T) (*app, *uitest.Window) {
	form := NewForm()
	w := uitest.Show(t, form)
	a := form.Panel().Widgets()[0].(*app)
	w.Screenshot()
	return a, w
}

// at returns the point of the window over the point of the drawing
func at(a *app, p ui.PointF) uitest.Point {
	s := a.view.ToScreen(p)
	return uitest.At(a.view, int(s.X), int(s.Y))
}

// The sample is drawn: the outline of the plate is in the view in the ink
func TestDrawingShows(t *testing.T) {
	a, w := newTestApp(t)
	if len(a.doc.shapes) < 50 {
		t.Fatalf("%d shapes in the sample", len(a.doc.shapes))
	}
	a.view.SetView(frontAt, 3)
	img := w.Screenshot()
	// The left side of the plate
	p := at(a, ui.PointF{X: frontAt.X - plateW/2, Y: frontAt.Y})
	found := false
	for dx := -2; dx <= 2; dx++ {
		if c := img.RGBAAt(p.X+dx, p.Y); c.R > 180 && c.G > 180 {
			found = true
		}
	}
	if !found {
		t.Fatal("the outline of the plate is not drawn")
	}
	// The layers hide
	a.layerChecks[layOutline].SetChecked(false)
	img = w.Screenshot()
	for dx := -2; dx <= 2; dx++ {
		if c := img.RGBAAt(p.X+dx, p.Y); c.R > 180 && c.G > 180 && c.B > 180 {
			t.Fatal("a hidden layer is drawn")
		}
	}
}

// The tools draw: a line snapped to the corner of the frame, a circle, a
// rectangle, a dimension
func TestTools(t *testing.T) {
	a, w := newTestApp(t)
	a.view.SetView(ui.PointF{X: 30, Y: 30}, 4)
	n := len(a.doc.shapes)
	w.Key(ui.KeyL, ui.KeyModifiers{})
	if a.tool != toolLine {
		t.Fatalf("the L key: tool %v", a.tool)
	}
	// Near the corner of the inner frame (10, 10): snapped to it
	near := at(a, ui.PointF{X: 10.6, Y: 9.5})
	w.Move(near)
	if a.snap.kind != snapEnd && a.snap.kind != snapIntersection || a.snap.p != (ui.PointF{X: 10, Y: 10}) {
		t.Fatalf("snap %v at %v", snapNames[a.snap.kind], a.snap.p)
	}
	w.Click(near)
	w.Click(at(a, ui.PointF{X: 40, Y: 25}))
	if len(a.doc.shapes) != n+1 {
		t.Fatalf("%d shapes after a line", len(a.doc.shapes)-n)
	}
	l := a.doc.shapes[len(a.doc.shapes)-1]
	if l.kind != kLine || l.p[0] != (ui.PointF{X: 10, Y: 10}) {
		t.Fatalf("the line %v", l.p)
	}
	// The line goes on from its end; the right button ends it
	w.ClickButton(ui.MouseButtonRight, at(a, ui.PointF{X: 50, Y: 50}))
	if len(a.pts) != 0 {
		t.Fatal("the right button did not end the line")
	}
	w.Key(ui.KeyC, ui.KeyModifiers{})
	w.Click(at(a, ui.PointF{X: 60, Y: 60}))
	w.Click(at(a, ui.PointF{X: 65, Y: 60}))
	w.Key(ui.KeyR, ui.KeyModifiers{})
	w.Click(at(a, ui.PointF{X: 70, Y: 70}))
	w.Click(at(a, ui.PointF{X: 80, Y: 76}))
	w.Key(ui.KeyD, ui.KeyModifiers{})
	w.Click(at(a, ui.PointF{X: 70, Y: 40}))
	w.Click(at(a, ui.PointF{X: 85, Y: 40}))
	w.Click(at(a, ui.PointF{X: 85, Y: 45}))
	if got := len(a.doc.shapes) - n; got != 1+1+4+1 {
		t.Fatalf("%d shapes drawn", got)
	}
	w.Screenshot()
}

// A click selects, a box selects inside or crossing, Delete deletes
func TestSelection(t *testing.T) {
	a, w := newTestApp(t)
	a.view.SetView(frontAt, 3)
	w.Screenshot()
	// The bore of the front view, away from its center lines
	w.Click(at(a, ui.PointF{X: frontAt.X + boreR*math.Sqrt2/2, Y: frontAt.Y + boreR*math.Sqrt2/2}))
	if len(a.selected) != 1 {
		t.Fatalf("%d selected by a click", len(a.selected))
	}
	for s := range a.selected {
		if s.kind != kCircle || s.r != boreR {
			t.Fatalf("selected %v", s.kind)
		}
	}
	// A box around the upper right bolt hole, to the right: the hole and
	// its center lines
	c := ui.PointF{X: frontAt.X + boltX, Y: frontAt.Y + boltY}
	w.Drag(at(a, ui.PointF{X: c.X - 10, Y: c.Y + 10}), at(a, ui.PointF{X: c.X + 10, Y: c.Y - 10}))
	if len(a.selected) != 3 {
		t.Fatalf("%d selected inside the box", len(a.selected))
	}
	// The same box to the left crosses the circle of the bolts too
	w.Drag(at(a, ui.PointF{X: c.X + 10, Y: c.Y - 10}), at(a, ui.PointF{X: c.X - 10, Y: c.Y + 10}))
	if len(a.selected) <= 3 {
		t.Fatalf("%d selected crossing the box", len(a.selected))
	}
	n := len(a.doc.shapes)
	k := len(a.selected)
	w.Key(ui.KeyDelete, ui.KeyModifiers{})
	if len(a.doc.shapes) != n-k || len(a.selected) != 0 {
		t.Fatalf("Delete: %d shapes, %d selected", n-len(a.doc.shapes), len(a.selected))
	}
	w.Screenshot()
}

// A drawing of 100 000 shapes is drawn in a reasonable time, all of it
// and zoomed in, and dragged
func TestManyShapes(t *testing.T) {
	a, w := newTestApp(t)
	a.addMany(100000)
	start := time.Now()
	w.Screenshot()
	all := time.Since(start)
	r := a.doc.extents()
	a.view.SetView(r.Center(), 20)
	start = time.Now()
	w.Screenshot()
	zoomed := time.Since(start)
	t.Logf("100 000 shapes: all %v, zoomed in %v (paint %v)", all, zoomed, a.lastPaint)
	if all > 5*time.Second || zoomed > time.Second {
		t.Fatalf("too slow: all %v, zoomed %v", all, zoomed)
	}
	// A drag paints a strip
	from := uitest.Center(a.view)
	to := uitest.Point{X: from.X + 30, Y: from.Y}
	w.Move(from)
	w.Press(ui.MouseButtonMiddle, from)
	start = time.Now()
	w.Move(to)
	w.Release(ui.MouseButtonMiddle, to)
	w.Screenshot()
	t.Logf("a drag: %v", time.Since(start))
}

func TestExport(t *testing.T) {
	a, _ := newTestApp(t)
	dir := t.TempDir()
	for _, f := range []string{"svg", "png"} {
		path := filepath.Join(dir, "d."+f)
		if err := a.writeExport(path, f); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if len(data) < 1000 {
			t.Fatalf("%s: %d bytes", f, len(data))
		}
		if f == "svg" && (!strings.Contains(string(data), "<path") || !strings.Contains(string(data), "<svg")) {
			t.Fatal("the SVG has no paths")
		}
	}
}

// The 3D model: cut, picked, the gizmo
func TestModel3D(t *testing.T) {
	a, w := newTestApp(t)
	a.tabs.SetCurrentIndex(1)
	w.Screenshot()
	if len(a.scene.Objects) != 6 {
		t.Fatalf("%d objects", len(a.scene.Objects))
	}
	a.section.SetChecked(true)
	a.updateSection()
	if len(a.scene.ClipPlanes) != 1 {
		t.Fatal("no section")
	}
	img := w.Screenshot()
	// The red of the section is seen
	red := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 2 {
		for x := b.Min.X; x < b.Max.X; x += 2 {
			if c := img.RGBAAt(x, y); c.R > 200 && c.G < 90 && c.B < 90 {
				red++
			}
		}
	}
	if red < 100 {
		t.Fatalf("the section is not seen (%d red points)", red)
	}
	w.Click(uitest.Center(a.view3d))
	if a.picked == nil {
		t.Fatal("nothing picked at the center")
	}
}
