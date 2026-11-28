package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/big"
	"testing"
	"time"
)

// The y autoscale fits the finite values: an Inf or NaN sample (a missing
// reading) made the range NaN or Inf, and the axis fell back to 0..1 with
// the data off the plot
func TestChartAutoscaleSkipsNonFinite(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	yDomain := func(ch *ChartView) (float64, float64) {
		ch.Render(400, 300, 1, nil)
		return ch.Pane(0).YAxis().scale.Domain()
	}
	cases := map[string]func(bad float64) (float64, float64){
		"line": func(bad float64) (float64, float64) {
			ch := NewLineChart()
			ch.AddValues(100, 101, bad, 102, 103)
			return yDomain(&ch.ChartView)
		},
		"line fill": func(bad float64) (float64, float64) {
			ch := NewLineChart()
			ch.Series("a").SetFill(true, 0.3, false).SetFillBase(100).SetValues([]float64{100, 101, bad, 102, 103})
			return yDomain(&ch.ChartView)
		},
		"candle": func(bad float64) (float64, float64) {
			ch := NewXYChart()
			ch.Pane(0).AddCandles("c").AddOHLC(0, 101, 103, 100, 102).AddOHLC(1, 101, bad, 100, 102).AddOHLC(2, 101, 102, bad, 101)
			return yDomain(ch)
		},
		"band": func(bad float64) (float64, float64) {
			ch := NewXYChart()
			ch.Pane(0).AddBand("b").Add(0, 101, 100, 102).Add(1, 102, 101, bad).Add(2, bad, 101, 103).Add(3, 102, 101, 103)
			return yDomain(ch)
		},
		"waterfall": func(bad float64) (float64, float64) {
			ch := NewXYChart()
			ch.Pane(0).AddWaterfall("w").AddStep("a", 100).AddStep("b", bad).AddStep("c", 3)
			return yDomain(ch)
		},
	}
	for name, mk := range cases {
		for _, bad := range []float64{nan, inf, -inf} {
			var lo, hi float64
			chartWithin(t, 5*time.Second, fmt.Sprint(name, " with ", bad), func() { lo, hi = mk(bad) })
			if !(lo <= 100 && hi >= 103 && hi-lo < 120) {
				t.Errorf("%s with %g: y domain %g..%g, want about 100..103", name, bad, lo, hi)
			}
		}
	}
	// The spans fit the time axis: a span with a NaN or Inf end is left out
	for _, bad := range []float64{nan, inf, -inf} {
		ch := NewXYChart()
		s := ch.Pane(0).AddSpans("s")
		s.Add(ChartSpan{Row: "r", Start: 0, End: 5}).Add(ChartSpan{Row: "r", Start: 1, End: bad})
		lo, hi, ok := s.Range(ch.Pane(0).xAxis, nil)
		if !ok || lo != 0 || hi != 5 {
			t.Errorf("spans with an end of %g: x range %g..%g (%v), want 0..5", bad, lo, hi, ok)
		}
	}
}

// A labelled span with a NaN end panicked the text drawing: the test of
// visibility is false for NaN, so it was drawn with its label at x = NaN,
// and int(NaN) reached the glyph blending
func TestChartSpanNonFiniteLabel(t *testing.T) {
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		ch := NewXYChart()
		s := ch.Pane(0).AddSpans("s").SetLabels(true)
		s.Add(ChartSpan{ID: "a", Row: "r", Start: 0, End: 5, Label: "ok"})
		s.Add(ChartSpan{ID: "b", Row: "r", Start: 1, End: bad, Label: "x"})
		s.Add(ChartSpan{ID: "c", Row: "r", Start: bad, End: bad, Label: "m"})
		chartWithin(t, 5*time.Second, fmt.Sprint("the paint of a span ending at ", bad), func() { ch.Render(400, 300, 1, nil) })
		if len(s.rects) != 1 {
			t.Errorf("end %g: %d spans drawn, want only the finite one", bad, len(s.rects))
		}
	}
	// Any chart label at a position that does not fit an int is skipped
	cnv := NewCanvas(image.NewRGBA(image.Rect(0, 0, 100, 100)))
	for _, p := range [][2]float64{{math.NaN(), 10}, {10, math.Inf(1)}, {-1e300, 10}, {10, 1e19}} {
		chartWithin(t, 5*time.Second, fmt.Sprint("a label at ", p), func() {
			chartDrawLabel(cnv, "label", p[0], p[1], HAlignLeft, VAlignCenter, color.RGBA{A: 255})
		})
	}
}

// A histogram whose values lie far apart for its bin width: the index of
// the bin overflowed int (a panic), and below that every value past bin
// 10000 was counted in the last one. Now the bins are widened to fit the
// values, and each value is counted in the bin that holds it.
func TestHistogramWideRange(t *testing.T) {
	cases := []struct {
		width float64
		vals  []float64
	}{
		{1e-9, []float64{0, 0.5, 1e10}},
		{0, []float64{-1e308, 1.7e308}},
		{0, []float64{-math.MaxFloat64, 0, math.MaxFloat64}},
		{0.001, []float64{0, 1, 2, 3, 100}},
		{1, []float64{-5e300, 1, 2, 3}},
	}
	for _, c := range cases {
		ch := NewXYChart()
		h := ch.Pane(0).AddHistogram("h").SetBins(0, c.width).SetValues(c.vals)
		chartWithin(t, 5*time.Second, fmt.Sprint("the histogram of ", c.vals), func() { ch.Render(400, 300, 1, nil) })
		edges, counts := h.Bins()
		if len(counts) == 0 || len(counts) > histMaxBins || len(edges) != len(counts)+1 {
			t.Fatalf("%v: %d bins, %d edges", c.vals, len(counts), len(edges))
		}
		total := 0.0
		for _, n := range counts {
			total += n
		}
		if total != float64(len(c.vals)) {
			t.Errorf("%v: %g values counted, want %d", c.vals, total, len(c.vals))
		}
		for _, v := range c.vals {
			if v < edges[0] || v > edges[len(edges)-1] {
				t.Errorf("%v: the value %g is outside the bins %g..%g", c.vals, v, edges[0], edges[len(edges)-1])
			}
		}
		for i, e := range edges {
			if !chartFinite(e) || (i > 0 && e < edges[i-1]) {
				t.Fatalf("%v: edge %d is %g", c.vals, i, e)
			}
		}
	}
	// The value 100 of a bin width of 0.001 has a bin of its own, not the
	// last of 10000 bins ending at 10
	ch := NewXYChart()
	h := ch.Pane(0).AddHistogram("h").SetBins(0, 0.001).SetValues([]float64{0, 1, 2, 3, 100})
	ch.Render(400, 300, 1, nil)
	edges, counts := h.Bins()
	if last := len(counts) - 1; counts[last] != 1 || edges[last] > 100 || edges[last+1] < 100 {
		t.Errorf("the last bin %g..%g counts %g, want the value 100 alone", edges[last], edges[last+1], counts[last])
	}
	for i := range counts[:len(counts)-1] {
		if edges[i] <= 3 && edges[i+1] > 3 && counts[i] != 1 {
			t.Errorf("the bin of the value 3 counts %g", counts[i])
		}
	}
}

// Zooming in with the wheel on millisecond data of a time axis: once the
// span was below what the floats resolve at 1.7e9 s, the scale took the
// range for a single value and widened it by a tenth of it - years. The
// zoom stops at the shortest span instead.
func TestChartZoomInStopsAtFloatResolution(t *testing.T) {
	ch := NewLineChart()
	ch.XAxis().SetScale(&TimeScale{Location: time.UTC})
	s := ch.Series("a")
	for i := 0; i < 1000; i++ {
		s.AddXY(1.7e9+float64(i)*1e-3, math.Sin(float64(i)/10))
	}
	img := image.NewRGBA(image.Rect(0, 0, 500, 300))
	paint := func() {
		cnv := NewCanvas(img)
		cnv.SetDirectTranslateAndClip(0, 0, 500, 300)
		ch.paintFunc(cnv)
	}
	ch.SetSize(500, 300)
	paint()
	prev := math.Inf(1)
	for i := 0; i < 60; i++ {
		ch.onWheel(&WheelEvent{DY: 3, X: 250, Y: 150})
		paint()
		lo, hi := ch.XRange()
		if span := hi - lo; span > prev*1.01 || lo < 1.7e9-1 || hi > 1.7e9+2 {
			t.Fatalf("step %d: the zoom in shows %f..%f (span %g, was %g)", i, lo, hi, span, prev)
		}
		prev = hi - lo
	}
	if prev <= 0 || prev > 0.01 {
		t.Errorf("the zoom stops at a span of %g s, want a few ms", prev)
	}
}

// The buckets of the time chart's aggregation used UnixNano, which wraps
// around before 1678 and after 2262: points of such dates were put at
// wrong times (2300 came out in 1715)
func TestTimeChartDownsampleFarDates(t *testing.T) {
	for _, year := range []int{1, 1600, 1677, 2263, 2300, 3000, 100000} {
		base := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		var pts []TimeChartPoint
		for i := 0; i < 100; i++ {
			pts = append(pts, NewTimeChartValue(base.Add(time.Duration(i)*time.Minute), float64(i)))
		}
		res := TimeChartDownsample(pts, base, base.Add(100*time.Minute), 10*time.Minute)
		if len(res) != 10 || !res[0].DT.Equal(base) || !res[9].DT.Equal(base.Add(90*time.Minute)) {
			t.Fatalf("year %d: %d buckets from %v", year, len(res), res[0].DT.UTC())
		}
		if res[3].First != 30 || res[3].Last != 39 {
			t.Errorf("year %d: the bucket of 00:30 holds %g..%g", year, res[3].First, res[3].Last)
		}
	}
	// The grid of the buckets is that of the Unix epoch for any date: the
	// start of a bucket is a whole number of groups after 1970
	ns := func(t time.Time) *big.Int {
		v := big.NewInt(t.Unix())
		v.Mul(v, big.NewInt(1e9))
		return v.Add(v, big.NewInt(int64(t.Nanosecond())))
	}
	for _, g := range []time.Duration{7 * time.Minute, 1300 * time.Millisecond, 3 * time.Nanosecond, 24 * time.Hour, 200 * 365 * 24 * time.Hour} {
		for _, dt := range []time.Time{
			time.Date(1500, 3, 4, 5, 6, 7, 8, time.UTC),
			time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 4, 5, 6, 7, 8, time.UTC),
			time.Date(2500, 3, 4, 5, 6, 7, 8, time.UTC),
		} {
			b := NewTimeChartAggregator(g, 0).bucketStart(dt)
			if b.After(dt) || dt.Sub(b) >= g {
				t.Errorf("group %v, %v: bucket at %v", g, dt, b.UTC())
			}
			if r := new(big.Int).Mod(ns(b), big.NewInt(int64(g))); r.Sign() != 0 {
				t.Errorf("group %v, %v: bucket at %v is %v ns off the grid", g, dt, b.UTC(), r)
			}
		}
	}
}

// The pie's hover brought the angle into a slice's turn by adding 2π in a
// loop: a start of 1e12 took minutes, and from 1e20 on adding 2π no longer
// changed the angle and the loop never ended
func TestPieHoverAnyStartAngle(t *testing.T) {
	hitAt := func(start float64) int {
		ch := NewPieChart()
		ch.Add("a", 1).Add("b", 2)
		ch.Pie().SetAngles(start, 2*math.Pi)
		ch.Render(400, 300, 1, nil)
		pie := ch.Pie()
		var hit int
		chartWithin(t, 5*time.Second, fmt.Sprint("the hover with a start of ", start), func() {
			hit = pie.sliceAt(pie.cx+pie.r/2, pie.cy+pie.r/4)
		})
		return hit
	}
	for _, start := range []float64{1e12, 1e20, -1e300, math.MaxFloat64} {
		want := hitAt(math.Mod(start, 2*math.Pi))
		if got := hitAt(start); got != want || got < 0 {
			t.Errorf("start %g: slice %d under the pointer, want %d", start, got, want)
		}
	}
	// A start of Inf or NaN is ignored: the default, the top
	for _, bad := range []float64{math.NaN(), math.Inf(1)} {
		ch := NewPieChart()
		ch.Pie().SetAngles(bad, 2*math.Pi)
		if s := ch.Pie().start; s != -math.Pi/2 {
			t.Errorf("SetAngles(%g): the start is %g", bad, s)
		}
	}
	// The slice under the pointer for slices drawn far from the first turn
	s := &PieSeries{cx: 100, cy: 100, r: 50}
	for _, a0 := range []float64{0, 1e6, -1e6, 1e15} {
		s.drawn = []pieDrawn{{a0: a0, a1: a0 + math.Pi}, {a0: a0 + math.Pi, a1: a0 + 2*math.Pi}}
		for _, a := range []float64{0.3, 2, 3.5, 5} {
			// a turned by a0 is in slice 0 when (a-a0) mod 2π < π
			d := math.Mod(math.Mod(a-a0, 2*math.Pi)+2*math.Pi, 2*math.Pi)
			want := 0
			if d >= math.Pi {
				want = 1
			}
			got := -2
			chartWithin(t, 5*time.Second, "sliceAt", func() { got = s.sliceAt(100+20*math.Cos(a), 100+20*math.Sin(a)) })
			if got != want {
				t.Errorf("a0 %g, angle %g: slice %d, want %d", a0, a, got, want)
			}
		}
	}
}
