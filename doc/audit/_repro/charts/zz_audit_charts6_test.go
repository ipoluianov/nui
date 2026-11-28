package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"
)

// Small radar values: the rings loop runs to an absolute epsilon of 1e-9,
// so rings of a tiny step are made by the million
func TestAuditRadarSmallValues(t *testing.T) {
	for _, top := range []float64{1, 1e-12, 1e-14, 1e-15} {
		ch := NewXYChart()
		ch.Pane(0).AddRadar("r").Set("a", top).Set("b", top/2).Set("c", top/3)
		ch.SetSize(400, 300)
		auditPaint(ch, 400, 300)
		p := ch.Pane(0)
		v := p.viewport(p.xAxis, p.YAxis())
		cnv := NewCanvas(image.NewRGBA(image.Rect(0, 0, 10, 10)))
		start := time.Now()
		f := p.polarFrameOf(v, cnv)
		t.Logf("top %g: %d rings (rim %g), frame in %v", top, len(f.rings), f.max, time.Since(start))
		if len(f.rings) > 10 {
			t.Errorf("top %g: %d rings instead of about 4", top, len(f.rings))
		}
	}
}

func TestAuditSpanNaN(t *testing.T) {
	auditTry(t, "span with NaN start, labelled", func() {
		ch := NewXYChart()
		s := ch.Pane(0).AddSpans("s").SetLabels(true)
		s.Add(ChartSpan{ID: "a", Row: "r", Start: 0, End: 5, Label: "ok"})
		s.Add(ChartSpan{ID: "b", Row: "r", Start: 1, End: math.NaN(), Label: "x"})
		ch.Render(400, 300, 1, nil)
	})
}

func TestAuditBandInf(t *testing.T) {
	auditTry(t, "band with an Inf high", func() {
		ch := NewXYChart()
		ch.Pane(0).AddBand("b").Add(0, 1, 0, 2).Add(1, 2, 1, math.Inf(1)).Add(2, 1, 0, 2)
		ch.Render(400, 300, 1, nil)
	})
}

// Zooming in with the wheel on a time axis: the visible span must shrink
// monotonically, never jump out to years
func TestAuditZoomInJumpsOut(t *testing.T) {
	ch := NewLineChart()
	ch.XAxis().SetScale(&TimeScale{Location: time.UTC})
	s := ch.Series("a")
	base := 1.7e9
	for i := 0; i < 1000; i++ {
		s.AddXY(base+float64(i)*1e-3, math.Sin(float64(i)/10))
	}
	ch.SetSize(500, 300)
	auditPaint(&ch.ChartView, 500, 300)
	prev := math.Inf(1)
	for i := 0; i < 60; i++ {
		ch.onWheel(&WheelEvent{DY: 3, X: 250, Y: 150})
		auditPaint(&ch.ChartView, 500, 300)
		lo, hi := ch.XRange()
		span := hi - lo
		if span > prev*1.01 {
			t.Fatalf("step %d: zoom in made the span grow from %g s to %g s (range %f..%f)", i, prev, span, lo, hi)
		}
		prev = span
	}
	t.Logf("final span %g", prev)
}

func TestAuditHist2DNaN(t *testing.T) {
	auditTry(t, "hist2d NaN point", func() {
		ch := NewXYChart()
		ch.Pane(0).AddHistogram2D("h").SetPoints([]ChartPoint{{1, 1}, {math.NaN(), 2}, {3, 3}})
	})
	auditTry(t, "hist2d Inf point", func() {
		ch := NewXYChart()
		ch.Pane(0).AddHistogram2D("h").SetPoints([]ChartPoint{{1, 1}, {math.Inf(1), 2}, {3, 3}})
	})
}

func TestAuditHistogramWideRange(t *testing.T) {
	auditTry(t, "histogram fixed width, far value", func() {
		ch := NewXYChart()
		ch.Pane(0).AddHistogram("h").SetBins(0, 1e-9).SetValues([]float64{0, 0.5, 1e10})
		ch.Render(400, 300, 1, nil)
	})
	auditTry(t, "histogram huge range", func() {
		ch := NewXYChart()
		ch.Pane(0).AddHistogram("h").SetValues([]float64{-1e308, 1.7e308})
		ch.Render(400, 300, 1, nil)
	})
	ch := NewXYChart()
	h := ch.Pane(0).AddHistogram("h").SetBins(0, 0.001).SetValues([]float64{0, 1, 2, 3, 100})
	ch.Render(400, 300, 1, nil)
	e, c := h.Bins()
	t.Logf("bins %d, last edge %v, last count %v", len(c), e[len(e)-1], c[len(c)-1])
}

func TestAuditRemoveDuringInteraction(t *testing.T) {
	mk := func() (*LineChart, *ChartPane) {
		ch := NewLineChart()
		ch.Series("a").SetValues([]float64{1, 3, 2, 5, 4})
		p := ch.AddPane()
		p.AddLine("b").SetValues([]float64{5, 4, 3, 2, 1})
		ch.SetSize(500, 350)
		return ch, p
	}
	{
		ch, p := mk()
		ch.OnHover.Connect(func(h ChartHit) {
			ch.RemovePane(p)
			if h.Series != nil {
				if pn := h.Series.base().Pane(); pn != nil {
					pn.RemoveSeries(h.Series)
				}
			}
		})
		auditTry(t, "remove on hover", func() { auditInteract(&ch.ChartView, 500, 350) })
	}
	{
		ch, p := mk()
		ch.OnBeforeAction.Connect(func(e *ChartActionEvent) {
			ch.RemovePane(p)
			for _, s := range ch.Pane(0).Series() {
				ch.Pane(0).RemoveSeries(s)
			}
		})
		auditTry(t, "remove on before action", func() { auditInteract(&ch.ChartView, 500, 350) })
	}
	{
		ch, _ := mk()
		ch.OnViewChanged.Connect(func(*ChartView) {
			ch.Series("a").Clear()
			ch.Series("b").SetValues(nil)
		})
		ch.OnSeriesToggled.Connect(func(s ChartSeries) {
			if pn := s.base().Pane(); pn != nil {
				pn.RemoveSeries(s)
			}
		})
		auditTry(t, "clear on view changed", func() { auditInteract(&ch.ChartView, 500, 350) })
	}
	{
		a, _ := mk()
		b, _ := mk()
		_ = NewChartSyncGroup().Add(&a.ChartView, &b.ChartView)
		g2 := NewChartSyncGroup().Add(&b.ChartView, &a.ChartView)
		_ = g2
		a.OnViewChanged.Connect(func(*ChartView) { b.ResetZoom() })
		b.OnSelectionChanged.Connect(func(*ChartView) { a.ClearSelection() })
		auditTry(t, "sync groups", func() {
			auditInteract(&a.ChartView, 500, 350)
			auditInteract(&b.ChartView, 500, 350)
		})
	}
}

func TestAuditSparklineConstLarge(t *testing.T) {
	count := func(vals []float64) int {
		img := image.NewRGBA(image.Rect(0, 0, 100, 30))
		cnv := NewCanvas(img)
		cnv.SetDirectTranslateAndClip(0, 0, 100, 30)
		DrawSparkline(cnv, 0, 0, 100, 30, vals, ChartSparkLine, color.RGBA{255, 0, 0, 255}, nil)
		n := 0
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] != 0 {
				n++
			}
		}
		return n
	}
	small, large := count([]float64{5, 5, 5}), count([]float64{5e16, 5e16, 5e16})
	t.Logf("painted pixels: constant 5 -> %d, constant 5e16 -> %d", small, large)
	if large == 0 {
		t.Errorf("a constant sparkline of 5e16 draws nothing")
	}
}

// Pie hover with a huge start angle: sliceAt never ends
func TestAuditPieStartHuge(t *testing.T) {
	ch := NewPieChart()
	ch.Add("a", 1).Add("b", 2)
	ch.Pie().SetAngles(1e20, 2*math.Pi)
	ch.SetSize(400, 300)
	auditPaint(&ch.ChartView, 400, 300)
	auditTry(t, "pie hover with start 1e20", func() {
		ch.onMouseMove(200+30, 150+30, KeyModifiers{})
		ch.updateHits()
	})
}
