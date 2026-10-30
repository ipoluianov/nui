package ui

import (
	"image/color"
	"math"
	"testing"
)

// The outlines of a text are as wide as the text DrawText draws, and stand
// on the baseline
func TestTextOutline(t *testing.T) {
	w, asc, desc := MeasureTextOutline("Hello, world", FontFamilySans, 20)
	mw, _, _ := MeasureText(FontFamilySans, 20, "Hello, world")
	if math.Abs(w-float64(mw)) > 3 {
		t.Errorf("width %v, DrawText's %v", w, mw)
	}
	if asc < 15 || asc > 25 || desc < 3 || desc > 8 {
		t.Errorf("ascent %v descent %v", asc, desc)
	}
	p := NewPath()
	p.Text(10, 50, "Hx", FontFamilySans, 20)
	x0, y0, x1, y1, ok := p.Bounds()
	if !ok || x0 < 10 || y1 > 50.5 || y0 > 50-12 || x1 > 10+w {
		t.Errorf("bounds %v %v %v %v", x0, y0, x1, y1)
	}
	// Filled, the H has its two stems and the gap between them
	cnv, img := NewImageCanvas(100, 60, 1)
	cnv.FillPath(p, color.RGBA{0, 0, 0, 255})
	row := ""
	for x := 10; x < 26; x++ {
		if img.RGBAAt(x, 42).A > 128 {
			row += "#"
		} else {
			row += "."
		}
	}
	if row[:1] == "#" || !containsRun(row, "#..") {
		t.Errorf("the H at y 42: %q", row)
	}
}

func containsRun(s, run string) bool {
	for i := 0; i+len(run) <= len(s); i++ {
		if s[i:i+len(run)] == run {
			return true
		}
	}
	return false
}

// Aligned text: centered on the point both ways
func TestTextAligned(t *testing.T) {
	p := NewPath()
	p.TextAligned(100, 100, "MMMM", FontFamilySans, 30, HAlignCenter, VAlignCenter)
	x0, _, x1, _, _ := p.Bounds()
	if c := (x0 + x1) / 2; math.Abs(c-100) > 2 {
		t.Errorf("center x %v", c)
	}
	q := NewPath()
	q.TextAligned(100, 100, "MMMM", FontFamilySans, 30, HAlignRight, VAlignTop)
	_, y0, x1, _, _ := q.Bounds()
	if x1 > 100.5 || y0 < 100 {
		t.Errorf("right top: x1 %v y0 %v", x1, y0)
	}
}
