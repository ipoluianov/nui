package ui

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"testing"
	"time"
)

// chartWithin runs f and fails the test when it does not return in time:
// a regression that loops forever is reported instead of hanging the run
func chartWithin(t *testing.T, d time.Duration, what string, f func()) {
	t.Helper()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		f()
	}()
	select {
	case p := <-done:
		if p != nil {
			t.Fatalf("%s panics: %v", what, p)
		}
	case <-time.After(d):
		t.Fatalf("%s does not return in %v", what, d)
	}
}

// chartPolarRings returns the rings of the polar frame of the pane
func chartPolarRings(p *ChartPane) []float64 {
	cnv := NewCanvas(image.NewRGBA(image.Rect(0, 0, 10, 10)))
	v := &ChartViewport{X: 0, Y: 0, W: 400, H: 300, FontSize: 12, Theme: p.view.theme, XAxis: p.xAxis, YAxis: p.YAxis()}
	return p.polarFrameOf(v, cnv).rings
}

// The rings of a radar are about 4 for any scale of the values: the loop up
// to the rim had an absolute epsilon, so values far below 1e-9 gave millions
// of rings, and a rim of Inf (an Inf value, or a rim past the largest float)
// was never reached and the rings grew until the memory ran out
func TestPolarRingsBoundedAtAnyScale(t *testing.T) {
	// The finite small values first: there the old loop ends (with 2 million
	// rings at 1e-15), so a regression fails here before the cases below,
	// which would loop forever
	for _, top := range []float64{1, 7.5, 1e-9, 1e-12, 1e-15, 1e-300, 1e-320, 5e-324, 1e300, math.MaxFloat64} {
		ch := NewXYChart()
		ch.Pane(0).AddRadar("r").Set("a", top).Set("b", top/2).Set("c", top/3)
		var rings []float64
		chartWithin(t, 5*time.Second, fmt.Sprint("the rings of top ", top), func() { rings = chartPolarRings(ch.Pane(0)) })
		if len(rings) < 1 || len(rings) > 10 {
			t.Fatalf("top %g: %d rings, want about 4", top, len(rings))
		}
		if last := rings[len(rings)-1]; !chartFinite(last) || last < top {
			t.Fatalf("top %g: the rim is %g", top, last)
		}
	}
	// Non-finite values are ignored: the scale is that of the others
	for _, bad := range []float64{math.Inf(1), math.Inf(-1), math.NaN()} {
		ch := NewXYChart()
		ch.Pane(0).AddRadar("r").Set("a", 1).Set("b", bad).Set("c", 2)
		ch.Pane(0).AddPolar("p").Add(0, 1).Add(90, bad).Add(180, 2)
		var rings []float64
		chartWithin(t, 5*time.Second, fmt.Sprint("the rings with ", bad), func() { rings = chartPolarRings(ch.Pane(0)) })
		if len(rings) == 0 || rings[len(rings)-1] != 2 {
			t.Fatalf("with %g: rings %v, want a rim of 2", bad, rings)
		}
		chartWithin(t, 5*time.Second, fmt.Sprint("the paint with ", bad), func() { ch.Render(400, 300, 1, nil) })
	}
}

// A dense line (a sample per pixel, drawn as columns) whose neighbour sample
// outside the view lies far away, past a gap in the data or a deep zoom:
// the columns towards it were filled for every pixel of the distance, not
// only over the plot, and a paint took hours
func TestLineFarNeighbourDrawsQuickly(t *testing.T) {
	ch := NewLineChart()
	s := ch.Series("s")
	for i := 0; i <= 1000; i++ {
		s.AddXY(float64(i), float64(i%7))
	}
	s.AddXY(1e11, 0)
	ch.SetXRange(0, 1000)
	chartWithin(t, 10*time.Second, "the paint of a dense line with a far neighbour", func() { ch.Render(400, 300, 1, nil) })
}

// A point of Inf or NaN in a 2D histogram (grid or hexagons) panicked
// SetPoints with an index out of range: int(NaN) of the cell; such points
// are not counted, and a range that overflows a float does not panic
func TestHistogram2DSkipsNonFinitePoints(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	cases := [][]ChartPoint{
		{{1, 1}, {nan, 2}, {3, 3}},
		{{1, 1}, {inf, 2}, {3, 3}},
		{{1, 1}, {2, -inf}, {3, 3}},
		{{1, 1}, {-1e308, 1}, {1.7e308, 3}},
		{{math.MaxFloat64, 1}, {math.MaxFloat64, 2}, {nan, nan}},
		{{nan, nan}},
	}
	for _, pts := range cases {
		finite := 0
		for _, p := range pts {
			if chartFinite(p.X) && chartFinite(p.Y) {
				finite++
			}
		}
		for _, hex := range []bool{false, true} {
			ch := NewXYChart()
			var h *Histogram2DSeries
			chartWithin(t, 5*time.Second, fmt.Sprint("SetPoints ", pts), func() {
				h = ch.Pane(0).AddHistogram2D("h").SetPoints(pts)
				if hex {
					h.SetHex(0)
				}
			})
			count := 0.0
			for _, row := range h.grid {
				for _, n := range row {
					if !math.IsNaN(n) {
						count += n
					}
				}
			}
			if count != float64(finite) {
				t.Fatalf("%v: %g points counted, want %d", pts, count, finite)
			}
			chartWithin(t, 5*time.Second, fmt.Sprint("the paint of ", pts), func() { ch.Render(400, 300, 1, nil) })
		}
	}
}

// An Inf or a value that overflows when scaled to pixels panicked the paint
// of filled series in the rasterizer (an integer divide by zero): the line's
// fill, the band, the funnel, the digital signal, the stacked polar series
// and the percent stack of tiny values. The charts treat a non-finite value
// as a missing sample and keep the pixels of the paths near the plot, so
// they do not rely on the canvas surviving any float.
func TestChartsNonFiniteValuesPaint(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	sets := map[string][]float64{
		"inf":  {1, inf, 3},
		"-inf": {1, -inf, 3},
		"nan":  {1, nan, 3},
		"huge": {1e308, -1e308, 1.7e308},
		"maxf": {math.MaxFloat64, math.MaxFloat64},
		"tiny": {1e-320, 2e-320},
	}
	builders := map[string]func(d []float64) *ChartView{
		"line-fill": func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.Series("a").SetFill(true, 0.3, false).SetValues(d)
			return &ch.ChartView
		},
		"line-spline-gradient": func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.Series("a").SetShape(ChartLineSpline).SetFill(true, 0.3, true).SetMarkers(ChartMarkersAlways, MarkerCircle, 6).SetValues(d)
			return &ch.ChartView
		},
		"line-step-percent": func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.Series("a").SetShape(ChartLineStepAfter).SetStack("g").SetFill(true, 0.3, false).SetValues(d)
			ch.Series("b").SetStack("g").SetFill(true, 0.3, false).SetValues(d)
			ch.Pane(0).SetStackPercent("g", true)
			return &ch.ChartView
		},
		"band": func(d []float64) *ChartView {
			ch := NewXYChart()
			b := ch.Pane(0).AddBand("b")
			e := ch.Pane(0).AddBand("e").SetErrorBars(true)
			for i, v := range d {
				b.Add(float64(i), v, v+1, v-1)
				e.Add(float64(i), v, v-1, v+1)
			}
			b.Add(1.5, 1, 0, inf)
			return ch
		},
		"funnel": func(d []float64) *ChartView {
			ch := NewXYChart()
			f := ch.Pane(0).AddFunnel("f")
			p := ch.AddPane().AddFunnel("p").SetPyramid(true)
			for i, v := range d {
				f.Add(fmt.Sprint(i), v)
				p.Add(fmt.Sprint(i), v)
			}
			return ch
		},
		"digital": func(d []float64) *ChartView {
			ch := NewXYChart()
			s := ch.Pane(0).AddDigital("d")
			b := ch.Pane(0).AddDigital("bus").SetBus(true)
			for i, v := range d {
				s.AddState(float64(i), v)
				b.AddState(v, v)
			}
			return ch
		},
		"polar": func(d []float64) *ChartView {
			ch := NewXYChart()
			for k := 0; k < 4; k++ {
				p := ch.Pane(0).AddPolar(fmt.Sprint("p", k)).SetKind(ChartPolarKind(k % 3)).SetStack("s")
				for i, v := range d {
					p.Add(float64(i)*30, v)
				}
			}
			return ch
		},
		"radar": func(d []float64) *ChartView {
			ch := NewXYChart()
			r := ch.Pane(0).AddRadar("r")
			for i, v := range d {
				r.Set(fmt.Sprint(i), v)
			}
			r.Set("x", 1).Set("y", 2)
			return ch
		},
	}
	for bn, build := range builders {
		for dn, d := range sets {
			chartWithin(t, 10*time.Second, bn+"/"+dn, func() {
				c := build(d)
				c.SetSize(400, 300)
				c.Render(400, 300, 1, nil)
			})
		}
	}
}

// An Inf in a filled line is a missing sample: the line and its fill break
// there as at a NaN
func TestLineInfDrawsAsGap(t *testing.T) {
	render := func(bad float64) []byte {
		ch := NewLineChart()
		ch.Series("a").SetFill(true, 0.3, false).SetValues([]float64{1, 2, bad, 3, 1})
		ch.Pane(0).YAxis().SetRange(0, 4)
		return ch.Render(400, 300, 1, nil).Pix
	}
	gap := render(math.NaN())
	for _, bad := range []float64{math.Inf(1), math.Inf(-1)} {
		var got []byte
		chartWithin(t, 10*time.Second, fmt.Sprint("a fill with ", bad), func() { got = render(bad) })
		if !bytes.Equal(got, gap) {
			t.Fatalf("a fill with %g differs from a fill with a gap", bad)
		}
	}
}
