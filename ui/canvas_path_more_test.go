package ui

import (
	"errors"
	"math"
	"testing"
)

func TestParseSVGPath(t *testing.T) {
	p, err := ParseSVGPath("M10 10 h20 v20 H10 z m5 5 l5 0 0 5z")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Contains(20, 20) || p.Contains(5, 5) {
		t.Fatal("square of the path")
	}
	if !near(p.Length(), 80+5+5+math.Sqrt(50), 1e-6) {
		t.Fatalf("length %v", p.Length())
	}
	// Curves, the shorthand ones, numbers written tight
	for _, d := range []string{
		"M0,0C10,0 20,10 20,20S30,40 40,40",
		"M0 0Q10-10 20 0T40 0",
		"M-1.5.5L1e1-2e0",
		"M10 30A20 20 0 0 1 50 30A20 20 0 0 1 90 30Q90 60 50 90Q10 60 10 30z",
		"m0 0a5 5 0 1010 0",
	} {
		if _, err := ParseSVGPath(d); err != nil {
			t.Fatalf("%q: %v", d, err)
		}
	}
	for _, d := range []string{"10 10", "M10", "M0 0 X5", "M0 0z5 5"} {
		if _, err := ParseSVGPath(d); !errors.Is(err, ErrSVGPath) {
			t.Fatalf("%q: error %v", d, err)
		}
	}
}

// An arc of SVG is the half circle it names
func TestSVGArc(t *testing.T) {
	p, err := ParseSVGPath("M0 0A10 10 0 0 1 20 0")
	if err != nil {
		t.Fatal(err)
	}
	if l := p.Length(); !near(l, math.Pi*10, 0.05) {
		t.Fatalf("half circle length %v", l)
	}
	// Clockwise on the screen from the left to the right: through the top
	pt, _, _ := p.PointAtLength(math.Pi * 5)
	if !near(pt.X, 10, 0.1) || !near(pt.Y, -10, 0.1) {
		t.Fatalf("middle of the arc at %v", pt)
	}
	// Radii too small are grown
	q, _ := ParseSVGPath("M0 0A1 1 0 0 0 20 0")
	if l := q.Length(); !near(l, math.Pi*10, 0.05) {
		t.Fatalf("grown half circle length %v", l)
	}
}

func TestArcTo(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.ArcTo(20, 0, 20, 20, 5)
	p.LineTo(20, 20)
	// 15 straight, a quarter of a circle of 5, 15 straight
	if l := p.Length(); !near(l, 30+math.Pi*2.5, 0.05) {
		t.Fatalf("length %v", l)
	}
	if p.StrokeContains(20, 0, 2) {
		t.Fatal("the corner is not rounded")
	}
	if !p.StrokeContains(10, 0, 2) || !p.StrokeContains(20, 10, 2) {
		t.Fatal("the lines of the corner")
	}
}

func TestPathShapes(t *testing.T) {
	p := NewPath()
	p.Star(50, 50, 40, 15, 5)
	if !p.Contains(50, 50) || !p.Contains(50, 15) || p.Contains(20, 20) {
		t.Fatal("star")
	}
	p.Reset()
	p.RegularPolygon(0, 0, 10, 6, 0)
	if l := p.Length(); !near(l, 60, 1e-9) {
		t.Fatalf("hexagon length %v", l)
	}
	// A spline goes through its points
	pts := []PointF{{0, 0}, {10, 10}, {20, 0}, {30, 10}}
	p.Reset()
	p.Spline(pts, false, 0)
	for _, pt := range pts {
		if !p.StrokeContains(pt.X, pt.Y, 0.01) {
			t.Fatalf("spline misses %v", pt)
		}
	}
	// Tension 1 is the polyline
	p.Reset()
	p.Spline(pts, false, 1)
	if l := p.Length(); !near(l, 3*math.Sqrt(200), 1e-6) {
		t.Fatalf("straight spline length %v", l)
	}
	q := NewPath()
	q.AddPath(p)
	if !near(q.Length(), p.Length(), 1e-9) {
		t.Fatal("AddPath")
	}
}

func TestPathContainsEvenOdd(t *testing.T) {
	p := NewPath()
	p.Circle(0, 0, 10)
	p.Circle(0, 0, 5)
	if !p.Contains(0, 0) {
		t.Fatal("nonzero")
	}
	p.SetFillRule(FillRuleEvenOdd)
	if p.Contains(0, 0) || !p.Contains(7, 0) {
		t.Fatal("even-odd")
	}
}
