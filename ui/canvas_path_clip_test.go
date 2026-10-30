package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
	"time"
)

// A circle so large that only a bit of it is seen - a drawing zoomed far
// into an arc - is accurate where it is seen. The bit is in the middle of
// one of the curves the circle is made of, far from their ends.
func TestHugeCircleAccurate(t *testing.T) {
	black := color.RGBA{0, 0, 0, 255}
	for _, r := range []float64{1e5, 1e7, 1e9} {
		// The point of the circle at the angle a is at (200, 200)
		a := -0.9
		nx, ny := math.Cos(a), math.Sin(a) // out of the circle
		tx, ty := -ny, nx                  // along the edge
		p := NewPath()
		p.Circle(200-r*nx, 200-r*ny, r)
		cnv, img := NewImageCanvas(400, 400, 1)
		cnv.FillPath(p, black)
		scnv, simg := NewImageCanvas(400, 400, 1)
		scnv.StrokePath(p, StrokeStyle{Width: 2}, black)
		at := func(img *image.RGBA, d, n float64) uint8 {
			return img.RGBAAt(int(math.Floor(200+d*tx+n*nx)), int(math.Floor(200+d*ty+n*ny))).A
		}
		for _, d := range []float64{-150, -60, 0, 60, 150} {
			// Along the edge, then 3 pixels out of the circle and into it
			if out, in := at(img, d, 3), at(img, d, -3); out != 0 || in != 255 {
				t.Errorf("r %g at %v: fill %d out, %d in", r, d, out, in)
			}
			if on, off := at(simg, d, 0), at(simg, d, 3); on < 100 || off != 0 {
				t.Errorf("r %g at %v: line %d on, %d off", r, d, on, off)
			}
		}
	}
}

// A dashed line far longer than the canvas is quick, and its dashes are
// where they are on a short one
func TestHugeDashedLine(t *testing.T) {
	black := color.RGBA{0, 0, 0, 255}
	style := StrokeStyle{Width: 2, Dash: []float64{6, 4}}
	short, simg := NewImageCanvas(200, 20, 1)
	p := NewPath()
	p.MoveTo(-1000, 10)
	p.LineTo(1000, 10)
	short.StrokePath(p, style, black)

	long, limg := NewImageCanvas(200, 20, 1)
	q := NewPath()
	q.MoveTo(-1000, 10)
	q.LineTo(1e9, 10)
	start := time.Now()
	long.StrokePath(q, style, black)
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Errorf("a dashed line of 1e9 took %v", d)
	}
	for x := 0; x < 200; x++ {
		if a, b := simg.RGBAAt(x, 10).A, limg.RGBAAt(x, 10).A; int(a)-int(b) > 2 || int(b)-int(a) > 2 {
			t.Fatalf("dash at %d: %d, on the short line %d", x, b, a)
		}
	}
}

func TestClipSegment(t *testing.T) {
	r := [4]float64{0, 0, 10, 10}
	t0, t1, ok := clipSegment(PointF{-10, 5}, PointF{20, 5}, r)
	if !ok || math.Abs(t0-1.0/3) > 1e-12 || math.Abs(t1-2.0/3) > 1e-12 {
		t.Errorf("through: %v %v %v", t0, t1, ok)
	}
	if _, _, ok := clipSegment(PointF{-10, -5}, PointF{20, -5}, r); ok {
		t.Error("above: clipped in")
	}
	if t0, t1, ok := clipSegment(PointF{2, 2}, PointF{3, 3}, r); !ok || t0 != 0 || t1 != 1 {
		t.Errorf("inside: %v %v %v", t0, t1, ok)
	}
}

// A circle of a drawing zoomed in a million times by the canvas'
// transform is accurate where it is seen, with a line as thick as asked
func TestZoomedArcUnderTransform(t *testing.T) {
	black := color.RGBA{0, 0, 0, 255}
	const zoom = 1e6
	a := -0.9
	nx, ny := math.Cos(a), math.Sin(a)
	tx, ty := -ny, nx
	for _, nonScaling := range []bool{false, true} {
		cnv, img := NewImageCanvas(400, 400, 1)
		// The point of the unit circle at the angle a is at (200, 200)
		cnv.SetTransform(ScalingMatrix(zoom, zoom).Then(TranslationMatrix(200-zoom*nx, 200-zoom*ny)))
		p := NewPath()
		p.Circle(0, 0, 1)
		width := 2.0
		if !nonScaling {
			width = 2 / zoom
		}
		start := time.Now()
		cnv.StrokePath(p, StrokeStyle{Width: width, NonScaling: nonScaling}, black)
		if d := time.Since(start); d > 100*time.Millisecond {
			t.Errorf("non-scaling %v: %v", nonScaling, d)
		}
		for _, d := range []float64{-150, 0, 150} {
			on := img.RGBAAt(int(math.Floor(200+d*tx)), int(math.Floor(200+d*ty))).A
			off := img.RGBAAt(int(math.Floor(200+d*tx+3*nx)), int(math.Floor(200+d*ty+3*ny))).A
			if on < 100 || off != 0 {
				t.Errorf("non-scaling %v at %v: line %d on, %d off", nonScaling, d, on, off)
			}
		}
	}
}

// A line of NonScaling is as wide in pixels whatever the scale
func TestNonScalingStroke(t *testing.T) {
	black := color.RGBA{0, 0, 0, 255}
	for _, s := range []float64{0.1, 1, 10} {
		cnv, img := NewImageCanvas(100, 100, 1)
		cnv.ScaleBy(s, s)
		p := NewPath()
		p.MoveTo(0, 50/s)
		p.LineTo(100/s, 50/s)
		cnv.StrokePath(p, StrokeStyle{Width: 2, NonScaling: true}, black)
		n := 0
		for y := 0; y < 100; y++ {
			if img.RGBAAt(50, y).A > 0 {
				n++
			}
		}
		if n != 2 {
			t.Errorf("scale %v: %d pixels across, want 2", s, n)
		}
	}
}

// Arcs stay exact arcs: transformed, they are on the transformed ellipse;
// their box is that of the points they pass
func TestArcExact(t *testing.T) {
	p := NewPath()
	p.Arc(10, 20, 5, 0, math.Pi) // from the right, down, to the left
	x0, y0, x1, y1, ok := p.Bounds()
	if !ok || math.Abs(x0-5) > 1e-9 || math.Abs(x1-15) > 1e-9 || math.Abs(y0-20) > 1e-9 || math.Abs(y1-25) > 1e-9 {
		t.Errorf("bounds %v %v %v %v", x0, y0, x1, y1)
	}
	if pt, _ := p.CurrentPoint(); math.Abs(pt.X-5) > 1e-9 || math.Abs(pt.Y-20) > 1e-9 {
		t.Errorf("current point %v", pt)
	}
	q := NewPath()
	q.Circle(0, 0, 1)
	m := RotationMatrix(0.3).Then(SkewMatrix(0.5, 0)).Then(ScalingMatrix(3, 2))
	q.Transform(m)
	inv, _ := m.Invert()
	fp := &flatPath{}
	q.flatten(fp, 0, 0, 100)
	for _, pt := range fp.pts {
		x, y := inv.Apply(pt.X/100, pt.Y/100)
		if r := math.Hypot(x, y); math.Abs(r-1) > 1e-9 {
			t.Fatalf("a point of the transformed circle is at %v from the center", r)
		}
	}
	if d := svgPathData(q, 0, 0); !strings.Contains(d, "C") {
		t.Errorf("SVG of an arc: %q", d)
	}
}
