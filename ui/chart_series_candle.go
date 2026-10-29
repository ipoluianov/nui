package ui

import (
	"math"
)

// ChartCandleStyle is how a CandleSeries draws a bar
type ChartCandleStyle int

const (
	// ChartCandles draws Japanese candles: a body from the open to the
	// close and a wick from the low to the high
	ChartCandles ChartCandleStyle = iota
	// ChartOHLCBars draws a line from the low to the high with the open
	// as a tick to the left and the close to the right
	ChartOHLCBars
)

// CandleSeries draws a bar per sample of open (First), high, low and close
// (Last): green when it closed up, red when down. The samples come from a
// ChartSource, or from AddOHLC into the series' own memory.
type CandleSeries struct {
	ChartSeriesBase
	source ChartSource
	memory *ChartMemorySource
	style  ChartCandleStyle

	samples []ChartSample
}

// NewCandleSeries makes candles of the name
func NewCandleSeries(name string) *CandleSeries {
	s := &CandleSeries{}
	s.name = name
	s.memory = NewChartMemorySource()
	s.source = s.memory
	return s
}

// SetSource draws the candles from the source
func (s *CandleSeries) SetSource(src ChartSource) *CandleSeries {
	s.source = src
	s.memory, _ = src.(*ChartMemorySource)
	s.watch()
	s.update()
	return s
}

func (s *CandleSeries) watch() {
	if s.memory != nil && s.pane != nil && s.pane.view != nil {
		view := s.pane.view
		s.memory.watch(func() { view.RequestRepaint() })
	}
}

// Source returns the source of the candles
func (s *CandleSeries) Source() ChartSource { return s.source }

// Samples returns the samples drawn on the last paint
func (s *CandleSeries) Samples() []ChartSample { return s.samples }

// AddOHLC adds a bar at x
func (s *CandleSeries) AddOHLC(x, open, high, low, close float64) *CandleSeries {
	if s.memory == nil {
		s.SetSource(NewChartMemorySource())
	}
	s.memory.Append(ChartOHLC(x, open, high, low, close))
	return s
}

// SetStyle sets how a bar is drawn
func (s *CandleSeries) SetStyle(style ChartCandleStyle) *CandleSeries {
	s.style = style
	s.update()
	return s
}

// Range implements ChartSeries
func (s *CandleSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if axis.Horizontal() {
		if e, ok := s.source.(ChartSourceExtent); ok {
			return e.Extent()
		}
		return 0, 0, false
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	xmin, xmax := v.XRange()
	for _, p := range s.samples {
		if p.HasValue() && p.X >= xmin && p.X <= xmax {
			lo, hi = math.Min(lo, p.Low), math.Max(hi, p.High)
		}
	}
	return lo, hi, lo <= hi
}

// Prepare implements ChartSeries
func (s *CandleSeries) Prepare(v *ChartViewport) {
	s.samples = nil
	if s.source != nil {
		xmin, xmax := v.XRange()
		s.samples = s.source.GetData(xmin, xmax, v.Group())
	}
}

// Draw implements ChartSeries
func (s *CandleSeries) Draw(cnv *Canvas, v *ChartViewport) {
	drawBadSpans(cnv, v, s.samples)
	minDx := 30.0
	for i := 1; i < len(s.samples); i++ {
		if d := v.PX(s.samples[i].X) - v.PX(s.samples[i-1].X); d > 0 && d < minDx {
			minDx = d
		}
	}
	bodyW := math.Max(1, math.Floor(minDx*0.7))
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	for _, sm := range s.samples {
		if !sm.HasValue() {
			continue
		}
		col := v.Theme.Up
		if sm.Last < sm.First {
			col = v.Theme.Down
		}
		x := math.Floor(v.PX(sm.X)) + 0.5
		yh, yl := v.PY(sm.High), v.PY(sm.Low)
		yo, yc := v.PY(sm.First), v.PY(sm.Last)
		p.Reset()
		p.MoveTo(x, math.Round(yh))
		p.LineTo(x, math.Round(yl))
		if s.style == ChartOHLCBars {
			p.MoveTo(x-bodyW/2, chartCrisp(yo))
			p.LineTo(x, chartCrisp(yo))
			p.MoveTo(x, chartCrisp(yc))
			p.LineTo(x+bodyW/2, chartCrisp(yc))
			cnv.StrokePath(p, StrokeStyle{Width: 1}, col)
			continue
		}
		cnv.StrokePath(p, StrokeStyle{Width: 1}, col)
		top, bottom := math.Round(math.Min(yo, yc)), math.Round(math.Max(yo, yc))
		p.Reset()
		left := math.Round(x - bodyW/2)
		p.Rect(left, top, bodyW, math.Max(1, bottom-top))
		cnv.FillPath(p, col)
	}
}

// Nearest implements ChartSeries
func (s *CandleSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	i := nearestSampleIndex(s.samples, v.DataX(px))
	if i < 0 || !s.samples[i].HasValue() {
		return ChartHit{}, false
	}
	sm := s.samples[i]
	f := v.YAxis.scale.Format
	hx := v.PX(sm.X)
	col := v.Theme.Up
	if sm.Last < sm.First {
		col = v.Theme.Down
	}
	dist := math.Abs(hx - px)
	if !byX && (py < v.PY(sm.High)-4 || py > v.PY(sm.Low)+4) {
		dist += math.Min(math.Abs(py-v.PY(sm.High)), math.Abs(py-v.PY(sm.Low)))
	}
	return ChartHit{
		Series: s, Index: i, X: sm.X, Y: sm.Last, PX: hx, PY: v.PY(sm.Last),
		XLabel: v.XAxis.scale.Format(sm.X),
		YLabel: "O " + f(sm.First) + "  H " + f(sm.High) + "  L " + f(sm.Low) + "  C " + f(sm.Last),
		Name:   s.name, Color: col, Sample: sm, dist: dist,
	}, true
}

// Legend implements ChartSeries
func (s *CandleSeries) Legend(th *ChartTheme) []ChartLegendItem {
	items := s.legendItem(th, ChartLegendBox)
	items[0].Color = th.Up
	return items
}

// chartXMargin asks the horizontal axis for a margin: the first and the
// last candles are not cut in half at the edges of the plot
func (s *CandleSeries) chartXMargin() float64 { return 0.02 }
