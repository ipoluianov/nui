package ui

import (
	"math"
	"testing"
	"time"
)

// noPanicNoHang runs f and fails the test if it panics or does not return
// within a few seconds
func noPanicNoHang(t *testing.T, name string, f func()) {
	t.Helper()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		f()
	}()
	select {
	case r := <-done:
		if r != nil {
			t.Errorf("%s: panic: %v", name, r)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: does not return", name)
	}
}

// A dash that ends exactly at a corner, and a dash of no length, are drawn:
// a repeated point in a dash used to give NaN corners that erased the whole
// stroke
func TestStrokePathDashEndsAtVertex(t *testing.T) {
	rect := NewPath()
	rect.Rect(10, 10, 20, 20)
	cases := []struct {
		name string
		draw func(cnv *Canvas)
	}{
		// 20 is 2.5 periods: a dash ends at every corner
		{"rect dash 4 4", func(cnv *Canvas) { cnv.StrokePath(rect, StrokeStyle{Width: 2, Dash: []float64{4, 4}}, white) }},
		{"polyline dash 10 5", func(cnv *Canvas) {
			cnv.DrawPolyline([]PointF{{0, 10}, {10, 10}, {10, 30}}, StrokeStyle{Width: 2, Dash: []float64{10, 5}}, white)
		}},
		{"dotted round", func(cnv *Canvas) {
			cnv.StrokePath(rect, StrokeStyle{Width: 2, Dash: []float64{0, 4}, Cap: LineCapRound}, white)
		}},
		{"dotted square", func(cnv *Canvas) {
			cnv.StrokePath(rect, StrokeStyle{Width: 2, Dash: []float64{0, 4}, Cap: LineCapSquare}, white)
		}},
	}
	for _, tc := range cases {
		cnv, img := newTestCanvas(40, 40)
		tc.draw(cnv)
		if inkSum(img) == 0 {
			t.Errorf("%s: nothing drawn", tc.name)
		}
	}

	// The dots of a dotted line: a dot as of a line of no length at every 4
	// pixels of the 80 long outline, 20 of them
	cnv, img := newTestCanvas(40, 40)
	dot := NewPath()
	dot.MoveTo(20, 20)
	dot.LineTo(20, 20)
	cnv.StrokePath(dot, StrokeStyle{Width: 2, Cap: LineCapRound}, white)
	want := 20 * inkSum(img)
	cnv, img = newTestCanvas(40, 40)
	cnv.StrokePath(rect, StrokeStyle{Width: 2, Dash: []float64{0, 4}, Cap: LineCapRound}, white)
	if got := inkSum(img); want == 0 || math.Abs(got-want) > 0.5 {
		t.Errorf("dotted round: ink %.1f, want %.1f", got, want)
	}
	// A zero-length dash with butt caps draws nothing, as in SVG
	cnv, img = newTestCanvas(40, 40)
	cnv.StrokePath(rect, StrokeStyle{Width: 2, Dash: []float64{0, 4}}, white)
	if s := inkSum(img); s != 0 {
		t.Errorf("dotted butt: ink %.1f, want none", s)
	}
	// The same pattern as {4, 4} shifted so that no dash ends at a corner
	// draws as much ink
	cnv, img = newTestCanvas(40, 40)
	cnv.StrokePath(rect, StrokeStyle{Width: 2, Dash: []float64{4, 4}}, white)
	if s := inkSum(img); math.Abs(s-80) > 8 {
		t.Errorf("rect dash 4 4: ink %.1f, want about 80", s)
	}
}

// One polygon of a stroke with NaN corners does not erase the others
func TestRasterizePolysSkipsNaNPolygon(t *testing.T) {
	cnv, img := newTestCanvas(20, 20)
	pb := &polyBuilder{}
	pb.add(PointF{2, 2}, PointF{10, 2}, PointF{10, 10}, PointF{2, 10})
	pb.add(PointF{12, 12}, PointF{math.NaN(), math.NaN()}, PointF{12, 18})
	cnv.rasterizePolys(pb, func(off int, cover []uint8, _, _ int) { cnv.blendSolid(off, cover, 255, 255, 255, 255) })
	if s := inkSum(img); s != 64 {
		t.Errorf("ink %.1f, want 64 (the square)", s)
	}
}

// Infinite, NaN and overflowing coordinates, widths and transforms do not
// panic the rasterizer; what is finite is drawn
func TestPathNonFiniteCoordinates(t *testing.T) {
	inf, nan := math.Inf(1), math.NaN()
	poly := func(rule FillRule, pts ...PointF) *Path {
		p := NewPath()
		p.Polygon(pts...)
		p.SetFillRule(rule)
		return p
	}
	rect := func(x, y, w, h float64) *Path {
		p := NewPath()
		p.Rect(x, y, w, h)
		return p
	}
	for _, rule := range []FillRule{FillRuleNonZero, FillRuleEvenOdd} {
		for _, v := range []float64{inf, -inf, 1e308, -1e308, math.MaxFloat64, nan} {
			p := poly(rule, PointF{0, 0}, PointF{v, 10}, PointF{0, 40}, PointF{30, 30})
			q := poly(rule, PointF{0, 0}, PointF{10, v}, PointF{0, 40}, PointF{30, 30})
			r := poly(rule, PointF{-v, -v}, PointF{v, -v}, PointF{v, v}, PointF{-v, v})
			noPanicNoHang(t, "fill polygon", func() {
				cnv, _ := newTestCanvas(50, 50)
				cnv.FillPath(p, white)
				cnv.FillPath(q, white)
				cnv.FillPath(r, white)
				// Wide: rasterized in bands
				cnv, _ = newTestCanvas(400, 300)
				cnv.FillPath(p, white)
				cnv.FillPath(q, white)
				cnv.FillPath(r, white)
			})
		}
	}
	// A rectangle to the infinity covers what is seen of it
	cnv, img := newTestCanvas(50, 50)
	noPanicNoHang(t, "fill rect", func() { cnv.FillPath(rect(0, 0, inf, 30), white) })
	if s := inkSum(img); s != 50*30 {
		t.Errorf("rect to +Inf: ink %.1f, want %d", s, 50*30)
	}
	cnv, img = newTestCanvas(50, 50)
	noPanicNoHang(t, "fill huge rect", func() {
		cnv.FillPath(poly(FillRuleNonZero, PointF{-1e308, -1e308}, PointF{1e308, -1e308}, PointF{1e308, 30}, PointF{-1e308, 30}), white)
	})
	if s := inkSum(img); s != 50*30 {
		t.Errorf("huge rect: ink %.1f, want %d", s, 50*30)
	}

	noPanicNoHang(t, "stroke", func() {
		cnv, _ := newTestCanvas(50, 50)
		for _, w := range []float64{inf, nan, 1e308} {
			cnv.StrokePath(rect(1, 1, 10, 10), StrokeStyle{Width: w}, white)
			cnv.StrokePath(rect(1, 1, 10, 10), StrokeStyle{Width: w, Join: LineJoinRound, Cap: LineCapRound}, white)
			cnv.StrokePath(rect(1, 1, 10, 10), StrokeStyle{Width: w, Dash: []float64{2, 2}}, white)
		}
		for _, v := range []float64{inf, -inf, 1e308, nan} {
			cnv.DrawPolyline([]PointF{{0, 0}, {v, 10}, {20, 30}}, StrokeStyle{Width: 3}, white)
			cnv.DrawPolyline([]PointF{{0, 0}, {10, v}, {20, 30}}, StrokeStyle{Width: 3, Dash: []float64{3, 2}}, white)
		}
	})

	// Finite paths that overflow under a transform
	p, err := ParseSVGPath("M0 0L1e308 1e308L0 1e308Z")
	if err != nil {
		t.Fatal(err)
	}
	noPanicNoHang(t, "transform", func() {
		cnv, _ := newTestCanvas(50, 50)
		cnv.Save()
		cnv.ScaleBy(2, 0.5)
		cnv.StrokePath(p, StrokeStyle{Width: 1}, white)
		cnv.StrokePath(p, StrokeStyle{Width: 1, NonScaling: true}, white)
		cnv.FillPath(p, white)
		cnv.FillPathPaint(p, &LinearGradient{X2: 10, Stops: []GradientStop{{0, white}, {1, black}}})
		cnv.Restore()
	})

	// Curves and arcs with non-finite or huge points and radii
	noPanicNoHang(t, "curves", func() {
		cnv, _ := newTestCanvas(50, 50)
		for _, v := range []float64{inf, -inf, 1e308, nan} {
			c := NewPath()
			c.MoveTo(0, 0)
			c.CubicTo(v, 0, 0, v, 40, 40)
			c.QuadTo(v, v, 10, 40)
			c.Close()
			cnv.FillPath(c, white)
			cnv.StrokePath(c, StrokeStyle{Width: 2}, white)
			a := NewPath()
			a.Arc(20, 20, v, 0, 3)
			a.Arc(v, 20, 10, 0, 3)
			cnv.FillPath(a, white)
			cnv.StrokePath(a, StrokeStyle{Width: 2, Dash: []float64{2, 2}}, white)
		}
	})
}

// The fills of charts with infinite, overflowing and denormal values do not
// panic in the rasterizer (CHART-5)
func TestChartNonFiniteFillsDoNotPanic(t *testing.T) {
	noPanicNoHang(t, "line fill", func() {
		for _, vals := range [][]float64{{1, math.Inf(1), 3}, {1, math.Inf(-1), 3}, {1e308, -1e308, 1.7e308}} {
			for _, gradient := range []bool{false, true} {
				ch := NewLineChart()
				ch.Series("a").SetFill(true, 0.3, gradient).SetValues(vals)
				ch.Render(400, 300, 1, nil)
			}
		}
	})
	noPanicNoHang(t, "band", func() {
		b := NewXYChart()
		b.Pane(0).AddBand("b").Add(0, 1, 0, 2).Add(1, 2, 1, math.Inf(1)).Add(2, 1, 0, 2)
		b.Render(400, 300, 1, nil)
	})
	noPanicNoHang(t, "percent stack of denormals", func() {
		s := NewLineChart()
		s.Series("a").SetStack("g").SetFill(true, 0.3, false).SetValues([]float64{1e-320, 2e-320})
		s.Series("b").SetStack("g").SetValues([]float64{1e-320, 2e-320})
		s.Pane(0).SetStackPercent("g", true)
		s.Render(400, 300, 1, nil)
	})
}

// Path.Bounds returns for an arc of an infinite or huge start angle
func TestPathBoundsArcHugeAngle(t *testing.T) {
	noPanicNoHang(t, "inf start", func() {
		p := NewPath()
		p.Arc(0, 0, 10, math.Inf(1), 0)
		p.Bounds()
		p = NewPath()
		p.Arc(0, 0, 10, math.Inf(-1), math.Inf(1))
		p.Bounds()
		p = NewPath()
		p.Arc(0, 0, 10, math.NaN(), 1)
		p.Bounds()
	})
	var minX, minY, maxX, maxY float64
	noPanicNoHang(t, "huge start", func() {
		p := NewPath()
		p.Arc(0, 0, 10, 1e17, 1e17+16)
		minX, minY, maxX, maxY, _ = p.Bounds()
	})
	// More than a turn: the whole circle
	for _, v := range []float64{minX, minY, -maxX, -maxY} {
		if math.Abs(v+10) > 1e-6 {
			t.Errorf("huge start: bounds (%g, %g)-(%g, %g), want (-10, -10)-(10, 10)", minX, minY, maxX, maxY)
			break
		}
	}
	// A quarter at a start of many turns: as at the same angle near 0
	start := 1e6 * 2 * math.Pi
	p := NewPath()
	p.Arc(0, 0, 10, start, start+math.Pi/2)
	minX, minY, maxX, maxY, _ = p.Bounds()
	if math.Abs(minX) > 1e-6 || math.Abs(minY) > 1e-6 || math.Abs(maxX-10) > 1e-6 || math.Abs(maxY-10) > 1e-6 {
		t.Errorf("quarter at many turns: bounds (%g, %g)-(%g, %g), want (0, 0)-(10, 10)", minX, minY, maxX, maxY)
	}
}
