package ui

import (
	"encoding/xml"
	"image/color"
	"io"
	"math"
	"strings"
	"testing"
)

// CohenSutherland finds where a line far out of the box crosses it: the
// products in its int math used to overflow, and the diagonal through the
// box was reported as invisible
func TestCohenSutherlandLargeCoords(t *testing.T) {
	x1, y1, x2, y2, ok := CohenSutherland(-3000000000, -3000000000, 3000000000, 3000000000, 0, 0, 39, 39)
	if !ok || x1 != 0 || y1 != 0 || x2 != 39 || y2 != 39 {
		t.Errorf("got (%d,%d)-(%d,%d) %v, want (0,0)-(39,39) true", x1, y1, x2, y2, ok)
	}
	// Small coordinates keep the int math exactly
	x1, y1, x2, y2, ok = CohenSutherland(-10, 5, 50, 25, 0, 0, 39, 39)
	if !ok || x1 != 0 || y1 != 8 || x2 != 39 || y2 != 21 {
		t.Errorf("got (%d,%d)-(%d,%d) %v, want (0,8)-(39,21) true", x1, y1, x2, y2, ok)
	}
	// Extreme ends do not panic and stay within the box
	for _, c := range [][4]int{
		{math.MinInt, math.MinInt, math.MaxInt, math.MaxInt},
		{math.MinInt, 20, math.MaxInt, 21},
		{-1 << 62, 1 << 62, 1 << 62, -1 << 62},
	} {
		x1, y1, x2, y2, ok := CohenSutherland(c[0], c[1], c[2], c[3], 0, 0, 39, 39)
		if ok && (x1 < 0 || x1 > 39 || y1 < 0 || y1 > 39 || x2 < 0 || x2 > 39 || y2 < 0 || y2 > 39) {
			t.Errorf("%v: got (%d,%d)-(%d,%d), out of the box", c, x1, y1, x2, y2)
		}
	}

	cnv, img := newTestCanvas(40, 40)
	white := color.RGBA{255, 255, 255, 255}
	cnv.DrawLine(-3000000000, -3000000000, 3000000000, 3000000000, 1, white)
	if got := img.RGBAAt(20, 20); got.R < 128 {
		t.Errorf("the diagonal is not drawn: pixel (20,20) = %v", got)
	}
}

// Control characters XML does not allow are dropped from the text of the
// SVG: the document stays well-formed
func TestSVGRecorderControlChars(t *testing.T) {
	rec := NewSVGRecorder(100, 100)
	cnv, _ := newTestCanvas(100, 100)
	cnv.SetRecorder(rec)
	cnv.DrawText(0, 0, 100, 20, "a\x01b\x00c\x1b￾￿d\te")
	cnv.SetFontFamily("Font\x01Name")
	cnv.DrawText(0, 20, 100, 20, "x")
	s := rec.String()
	dec := xml.NewDecoder(strings.NewReader(s))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("the SVG is not well-formed: %v\n%s", err, s)
		}
	}
	if !strings.Contains(s, ">abcd\te<") {
		t.Errorf("the text is not kept without the control characters:\n%s", s)
	}
}

// A rectangle infinite or far larger than the picture invalidates all of
// it: its ends converted to int used to give an empty clip
func TestPaintCacheInvalidateHugeRect(t *testing.T) {
	cnv, _ := newTestCanvas(40, 40)
	for _, r := range []RectF{
		{math.Inf(-1), math.Inf(-1), math.Inf(1), math.Inf(1)},
		{-1e300, -1e300, 1e300, 1e300},
		{-1e300, 10, 1e300, 20},
	} {
		pc := NewPaintCache()
		calls := 0
		paint := func(*Canvas) { calls++ }
		pc.Draw(cnv, 0, 0, 40, 40, paint)
		pc.InvalidateRect(r)
		pc.Draw(cnv, 0, 0, 40, 40, paint)
		if calls != 2 {
			t.Errorf("%v: paint called %d times, want 2", r, calls)
		}
	}
	// A NaN rectangle is no rectangle
	pc := NewPaintCache()
	pc.Draw(cnv, 0, 0, 40, 40, func(*Canvas) {})
	pc.InvalidateRect(RectF{math.NaN(), 0, 10, 10})
	if !pc.Valid() {
		t.Errorf("a NaN rectangle invalidated the picture")
	}
}

// FillTriangle and the wide lines (fillPolygonPixels) give the true
// coverage at a sharp tip, as the paths do (CNV-14): the fixed-point
// rasterizer of x/image/vector gave the pixel (9, 288) of the first
// triangle 108 instead of 13
func TestFillTriangleSharpTipCoverage(t *testing.T) {
	tris := [][6]int{
		{0, 0, 400, 37, 10, 290},
		{2, 5, 390, 9, 3, 7}, // a sliver, as the end of a wide line
	}
	for _, tr := range tris {
		cnv, img := newTestCanvas(400, 300)
		cnv.FillTriangle(tr[0], tr[1], tr[2], tr[3], tr[4], tr[5], white)
		p := NewPath()
		p.Polygon(PointF{float64(tr[0]), float64(tr[1])}, PointF{float64(tr[2]), float64(tr[3])}, PointF{float64(tr[4]), float64(tr[5])})
		worst, wx, wy, wwant := 0.0, 0, 0, 0.0
		for y := 0; y < 300; y++ {
			for x := 0; x < 400; x++ {
				got := float64(img.RGBAAt(x, y).R)
				// The supersampling only where the pixel is not plainly
				// in or out: the test stays fast
				if got == 0 || got == 255 {
					if p.Contains(float64(x)+0.5, float64(y)+0.5) == (got == 255) {
						continue
					}
				}
				want := supersampled(p, float64(x), float64(y))
				if d := math.Abs(got - want); d > worst {
					worst, wx, wy, wwant = d, x, y, want
				}
			}
		}
		if worst > 6 {
			t.Errorf("triangle %v: pixel (%d,%d) = %d, want %.0f", tr, wx, wy, img.RGBAAt(wx, wy).R, wwant)
		}
	}
}
