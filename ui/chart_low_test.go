package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
	"time"
)

// A domain at the float limit widened past it to +Inf: the symlog ticks
// then counted decades from Inf down and never ended
func TestChartScaleDomainAtFloatLimit(t *testing.T) {
	big := math.MaxFloat64
	for _, v := range []float64{big, -big, big / 2} {
		for name, s := range map[string]ChartScale{
			"linear": NewLinearScale(v, v),
			"symlog": NewSymlogScale(v, v),
		} {
			lo, hi := s.Domain()
			if !chartFinite(lo) || !chartFinite(hi) || !(lo < hi) || !(lo <= v && v <= hi) {
				t.Errorf("%s scale of %g..%g: domain %g..%g, want a finite range around it", name, v, v, lo, hi)
			}
			chartWithin(t, 5*time.Second, fmt.Sprint(name, " ticks at ", v), func() {
				s.Ticks(300, func(string) float64 { return 20 })
			})
		}
	}
	// A tiny constant with a huge domain: c·10^k leaves the float range
	s := NewSymlogScale(-big, big)
	s.Constant = 1e-300
	chartWithin(t, 5*time.Second, "symlog ticks of a tiny constant", func() {
		if n := len(s.Ticks(300, func(string) float64 { return 20 })); n > 2000 {
			t.Errorf("%d ticks", n)
		}
	})
	ch := NewLineChart()
	ch.Series("a").SetValues([]float64{1, 2, 3})
	ch.Pane(0).YAxis().SetScale(NewSymlogScale(0, 1)).SetRange(big, big)
	chartWithin(t, 5*time.Second, "a symlog chart at the float limit", func() { ch.Render(400, 300, 1, nil) })
}

// A non-finite selection or range made SaveState return nothing: the whole
// view was lost, and RestoreState of it failed
func TestChartSaveStateNonFinite(t *testing.T) {
	inf := math.Inf(1)
	ch := NewLineChart()
	ch.Series("a").SetValues([]float64{1, 2, 3})
	ch.Pane(0).YAxis().SetRange(0, 10)
	ch.SetSelection(0, inf)
	ch.Pane(0).XAxis().SetRange(math.NaN(), 2)
	if _, _, ok := ch.Selection(); ok {
		t.Errorf("a selection to +Inf is taken")
	}
	if _, _, manual := ch.Pane(0).XAxis().Range(); manual {
		t.Errorf("a range from NaN is taken")
	}
	// The state of the fields set some other way is still saved without them
	ch.hasSel, ch.selFrom, ch.selTo = true, 0, inf
	st := ch.SaveState()
	if len(st) == 0 {
		t.Fatalf("SaveState is empty")
	}
	ch2 := NewLineChart()
	ch2.Series("a").SetValues([]float64{1, 2, 3})
	if err := ch2.RestoreState(st); err != nil {
		t.Fatalf("RestoreState(%s): %v", st, err)
	}
	if lo, hi, manual := ch2.Pane(0).YAxis().Range(); !manual || lo != 0 || hi != 10 {
		t.Errorf("restored y range %g..%g (manual %v), want 0..10", lo, hi, manual)
	}
}

// The candle autoscale took only Low and High: open or close outside
// high..low were drawn off the plot
func TestChartCandleRangeAllValues(t *testing.T) {
	ch := NewXYChart()
	ch.Pane(0).AddCandles("c").AddOHLC(0, 100, 50, 40, 45).AddOHLC(1, 45, 50, 40, 48)
	ch.Render(400, 300, 1, nil)
	lo, hi := ch.Pane(0).YAxis().scale.Domain()
	if !(lo <= 40 && hi >= 100) {
		t.Errorf("y domain %g..%g, want to cover 40..100", lo, hi)
	}
	ch = NewXYChart()
	ch.Pane(0).AddCandles("c").AddOHLC(0, 10, 9, 11, 10) // high below low
	ch.Render(400, 300, 1, nil)
	lo, hi = ch.Pane(0).YAxis().scale.Domain()
	if !(lo <= 9 && hi >= 11 && hi-lo < 10) {
		t.Errorf("y domain of an inverted candle %g..%g, want about 9..11", lo, hi)
	}
}

// Render panicked for a NaN or Inf scale (image of size MinInt64)
func TestChartRenderBadScale(t *testing.T) {
	for _, sc := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, -1} {
		ch := NewLineChart()
		ch.Series("a").SetValues([]float64{1, 2, 3})
		var img *image.RGBA
		chartWithin(t, 10*time.Second, fmt.Sprint("Render at scale ", sc), func() { img = ch.Render(100, 80, sc, nil) })
		b := img.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 || b.Dx()*b.Dy() > maxChartRenderPixels+b.Dx()+b.Dy()+1 {
			t.Errorf("Render at scale %g: image %v", sc, b)
		}
		var buf bytes.Buffer
		chartWithin(t, 10*time.Second, fmt.Sprint("WritePNG at scale ", sc), func() {
			if err := ch.WritePNG(&buf, 100, 80, sc, nil); err != nil {
				t.Errorf("WritePNG at scale %g: %v", sc, err)
			}
		})
	}
	ch := NewLineChart()
	chartWithin(t, 10*time.Second, "Render of a negative size", func() { ch.Render(-10, -10, 1, nil) })
}

// SetWeight(NaN) went through math.Max: the weights summed to NaN and no
// pane had a height
func TestChartPaneWeightNonFinite(t *testing.T) {
	for _, w := range []float64{math.NaN(), math.Inf(1), math.MaxFloat64} {
		ch := NewLineChart()
		ch.Series("a").SetValues([]float64{1, 2, 3})
		p := ch.AddPane().SetWeight(w)
		p.AddLine("b").SetValues([]float64{1, 2})
		ch.AddPane().SetWeight(w).AddLine("c").SetValues([]float64{1, 2})
		ch.Render(400, 300, 1, nil)
		for i, p := range ch.panes {
			if !chartFinite(p.h) || p.h < 0 {
				t.Errorf("weight %g: pane %d height %g", w, i, p.h)
			}
		}
		if !(ch.panes[1].h > 0) {
			t.Errorf("weight %g: the pane of that weight has height %g", w, ch.panes[1].h)
		}
	}
}

// A constant sparkline of 5e16 was widened by ±1, which is nothing past
// 2^53: the range stayed 0 and the line was NaN
func TestSparklineConstantLarge(t *testing.T) {
	count := func(vals []float64, kind ChartSparklineKind) int {
		img := image.NewRGBA(image.Rect(0, 0, 100, 30))
		cnv := NewCanvas(img)
		cnv.SetDirectTranslateAndClip(0, 0, 100, 30)
		DrawSparkline(cnv, 0, 0, 100, 30, vals, kind, color.RGBA{255, 0, 0, 255}, nil)
		n := 0
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] != 0 {
				n++
			}
		}
		return n
	}
	for _, v := range []float64{5e16, -5e16, math.MaxFloat64, -math.MaxFloat64} {
		if n := count([]float64{v, v, v}, ChartSparkLine); n == 0 {
			t.Errorf("a constant sparkline of %g draws nothing", v)
		}
	}
	if n := count([]float64{5, 5, 5}, ChartSparkLine); n == 0 {
		t.Errorf("a constant sparkline of 5 draws nothing")
	}
}

// The builders were quadratic: a copy of the category list per new bar
// category, a scan of the slices per pie slice, a copy of the tail per
// sample appended out of order
func TestChartBuildersLinear(t *testing.T) {
	const n = 60000
	chartWithin(t, 5*time.Second, "BarSeries.Set of many categories", func() {
		ch := NewBarChart()
		s := ch.Series("a")
		for i := 0; i < n; i++ {
			s.Set(fmt.Sprint("c", i), float64(i))
		}
		s.Set("c7", -1)
		cs := s.categories()
		if len(cs.Categories()) != n || cs.Index("c7") != 7 || s.Value(7) != -1 || cs.Index("none") != -1 {
			t.Errorf("%d categories, c7 at %d = %g", len(cs.Categories()), cs.Index("c7"), s.Value(7))
		}
		if lo, hi := cs.Domain(); lo != -0.5 || hi != n-0.5 {
			t.Errorf("category domain %g..%g", lo, hi)
		}
	})
	chartWithin(t, 5*time.Second, "PieSeries.Add of many slices", func() {
		ch := NewPieChart()
		p := ch.Pie()
		for i := 0; i < 2*n; i++ {
			p.Add(fmt.Sprint("s", i), 1)
		}
		p.Add("s3", 5)
		if len(p.slices) != 2*n || p.slices[3].value != 5 {
			t.Errorf("%d slices, s3 = %g", len(p.slices), p.slices[3].value)
		}
		p.Clear()
		p.Add("s3", 2)
		if len(p.slices) != 1 || p.slices[0].value != 2 {
			t.Errorf("after Clear: %+v", p.slices)
		}
	})
	chartWithin(t, 5*time.Second, "an out-of-order Append", func() {
		src := NewChartMemorySource()
		src.Append(ChartValue(0, -1), ChartValue(n, -2))
		batch := make([]ChartSample, 0, n)
		for i := n - 1; i >= 0; i-- {
			batch = append(batch, ChartValue(float64(i), float64(i)))
		}
		src.Append(batch...)
		got := src.Samples()
		if len(got) != n+2 {
			t.Fatalf("%d samples", len(got))
		}
		for i := 1; i < len(got); i++ {
			if got[i].X < got[i-1].X {
				t.Fatalf("samples out of order at %d: %g after %g", i, got[i].X, got[i-1].X)
			}
		}
		// The old sample of the same X stays before the new one
		if got[0].Last != -1 || got[1].X != 0 || got[1].Last != 0 || got[n+1].Last != -2 {
			t.Errorf("samples %v %v ... %v", got[0], got[1], got[n+1])
		}
	})
}

// An out-of-order Append keeps the order of arrival among equal X
func TestChartMemorySourceAppendStable(t *testing.T) {
	src := NewChartMemorySource()
	src.Append(ChartValue(1, 10), ChartValue(3, 30))
	src.Append(ChartValue(2, 21), ChartValue(1, 11), ChartValue(2, 22), ChartValue(5, 50), ChartValue(1, 12))
	var got []float64
	for _, p := range src.Samples() {
		got = append(got, p.Last)
	}
	want := []float64{10, 11, 12, 21, 22, 30, 50}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("samples %v, want %v", got, want)
	}
}

// The error bars drew every point every paint, however far it was zoomed
// in; only those over the plot are drawn now
func TestBandDrawsVisibleOnly(t *testing.T) {
	ch := NewXYChart()
	b := ch.Pane(0).AddBand("b").SetErrorBars(true)
	b.Add(5, 1, 0, 2).Add(1, 1, 0, 2) // out of order: kept sorted
	if b.samples[0].X != 1 || b.samples[1].X != 5 {
		t.Errorf("band samples %v", b.samples)
	}
	for i := 6; i < 100000; i++ {
		b.Add(float64(i), 1, 0, 2)
	}
	ch.Pane(0).XAxis().SetRange(100, 110)
	ch.Render(400, 300, 1, nil)
	v := &ChartViewport{X: 0, Y: 0, W: 400, H: 300, XAxis: ch.Pane(0).XAxis(), YAxis: ch.Pane(0).YAxis()}
	vis := b.visibleSamples(v)
	if len(vis) < 11 || len(vis) > 20 || vis[0].X > 99 || vis[len(vis)-1].X < 111 {
		t.Errorf("%d visible samples, %g..%g", len(vis), vis[0].X, vis[len(vis)-1].X)
	}
}
