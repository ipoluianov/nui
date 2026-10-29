package ui

import (
	"math"
)

// BarSeries draws bars: a value per category of a CategoryScale (a column
// chart, a bar chart turned on its side), or a bar per sample along a
// numeric or a time axis (the volume under candles, a histogram). The bars
// of several series on the same categories stand side by side; those of
// one stack group (SetStack) stand on each other.
//
//	s := chart.AddBars("Sales")
//	s.Set("Jan", 120).Set("Feb", 95).Set("Mar", 140)
type BarSeries struct {
	ChartSeriesBase
	// values by category index; NaN - none
	values []float64
	// source: the bars along a numeric axis come from it
	source ChartSource
	memory *ChartMemorySource

	horizontal bool
	stack      string
	// gap is the part of a category's band left between the groups of bars
	gap float64
	// innerGap is the part of a bar's slot left between the bars of a group
	innerGap float64
	labels   bool
	upDown   bool
	radius   float64

	// The samples of the last Prepare (along a numeric axis), and the
	// rectangles of the bars drawn, for Nearest
	samples []ChartSample
	rects   []barRect
}

type barRect struct {
	x, y, w, h float64
	index      int
	value      float64
	cat        string
}

// NewBarSeries makes bars of the name; add them to a pane with
// ChartPane.AddSeries, or make them with ChartPane.AddBars
func NewBarSeries(name string) *BarSeries {
	s := &BarSeries{gap: 0.2, innerGap: 0.1, radius: 2}
	s.name = name
	return s
}

// categoryAxis returns the axis of the categories: the horizontal one, or
// the vertical one of horizontal bars
func (s *BarSeries) categoryAxis(v *ChartViewport) *ChartAxis {
	if s.horizontal {
		return v.YAxis
	}
	return v.XAxis
}

func (s *BarSeries) valueAxis(v *ChartViewport) *ChartAxis {
	if s.horizontal {
		return v.XAxis
	}
	return v.YAxis
}

// categories returns the category scale the series' categories are on, from
// its pane; nil before the series is in a pane with one
func (s *BarSeries) categories() *CategoryScale {
	if s.pane == nil {
		return nil
	}
	xName, yName := s.AxisNames()
	name := xName
	if s.horizontal {
		name = yName
	}
	if a := s.pane.Axis(name); a != nil {
		if cs, ok := a.scale.(*CategoryScale); ok {
			return cs
		}
	}
	return nil
}

// Set sets the value of the category of the name, adding the category to
// the axis when it has none of the name
func (s *BarSeries) Set(category string, value float64) *BarSeries {
	cs := s.categories()
	if cs == nil {
		return s
	}
	i := cs.Index(category)
	if i < 0 {
		cs.SetCategories(append(append([]string(nil), cs.categories...), category))
		i = len(cs.categories) - 1
	}
	s.SetAt(i, value)
	return s
}

// Add is Set: chart.Add("Jan", 120).Add("Feb", 95)
func (s *BarSeries) Add(category string, value float64) *BarSeries {
	return s.Set(category, value)
}

// SetAt sets the value of the category number i
func (s *BarSeries) SetAt(i int, value float64) *BarSeries {
	if i < 0 {
		return s
	}
	for len(s.values) <= i {
		s.values = append(s.values, math.NaN())
	}
	s.values[i] = value
	s.update()
	return s
}

// SetValues replaces the values of the categories 0, 1, 2...
func (s *BarSeries) SetValues(values []float64) *BarSeries {
	s.values = append(s.values[:0], values...)
	s.update()
	return s
}

// Value returns the value of the category number i; NaN when it has none
func (s *BarSeries) Value(i int) float64 {
	if i < 0 || i >= len(s.values) {
		return math.NaN()
	}
	return s.values[i]
}

// shown returns the value of the category as drawn: in percent of the
// stack's total when the pane shows the group so
func (s *BarSeries) shown(i int) float64 {
	v := s.Value(i)
	if s.pane == nil || !s.pane.stackPercent(s.stack) || !chartFinite(v) {
		return v
	}
	total := 0.0
	for _, o := range s.pane.series {
		if b, ok := o.(*BarSeries); ok && b.stack == s.stack && b.Visible() && b.horizontal == s.horizontal {
			if ov := b.Value(i); chartFinite(ov) {
				total += math.Abs(ov)
			}
		}
	}
	if total == 0 {
		return 0
	}
	return v / total * 100
}

// SetSource draws a bar per sample of the source along a numeric or a time
// axis, instead of a bar per category
func (s *BarSeries) SetSource(src ChartSource) *BarSeries {
	s.source = src
	s.memory, _ = src.(*ChartMemorySource)
	if s.memory != nil && s.pane != nil && s.pane.view != nil {
		view := s.pane.view
		s.memory.watch(func() { view.RequestRepaint() })
	}
	s.update()
	return s
}

// AddXY adds a bar of the value at x along a numeric or a time axis
func (s *BarSeries) AddXY(x, value float64) *BarSeries {
	if s.memory == nil {
		s.SetSource(NewChartMemorySource())
	}
	s.memory.AppendValue(x, value)
	return s
}

// SetHorizontal lays the bars on their side: the categories go down the
// vertical axis and the values along the horizontal one
func (s *BarSeries) SetHorizontal(horizontal bool) *BarSeries {
	s.horizontal = horizontal
	s.update()
	return s
}

// SetStack stands the bars on those of the series of the same group added
// before; "" - side by side
func (s *BarSeries) SetStack(group string) *BarSeries {
	s.stack = group
	s.update()
	return s
}

// SetGap sets the part of a category's band left empty between the groups of
// bars (0.2 by default) and the part of a bar's slot between the bars of a
// group (0.1)
func (s *BarSeries) SetGap(gap, inner float64) *BarSeries {
	s.gap = math.Max(0, math.Min(gap, 0.95))
	s.innerGap = math.Max(0, math.Min(inner, 0.95))
	s.update()
	return s
}

// SetLabels writes the value on (or by) each bar
func (s *BarSeries) SetLabels(show bool) *BarSeries {
	s.labels = show
	s.update()
	return s
}

// SetUpDownColors colors a bar along a numeric axis by its sample: the up
// color of the theme when Last >= First, the down one otherwise (volume)
func (s *BarSeries) SetUpDownColors(on bool) *BarSeries {
	s.upDown = on
	s.update()
	return s
}

// SetCornerRadius rounds the end of each bar
func (s *BarSeries) SetCornerRadius(r float64) *BarSeries {
	s.radius = math.Max(0, r)
	s.update()
	return s
}

// byCategory reports whether the bars are a value per category
func (s *BarSeries) byCategory() bool {
	return s.source == nil
}

// Range implements ChartSeries
func (s *BarSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	catAxis := axis.Horizontal() != s.horizontal
	if s.byCategory() {
		if catAxis {
			if len(s.values) == 0 {
				return 0, 0, false
			}
			return -0.5, float64(len(s.values)) - 0.5, true
		}
		lo, hi := 0.0, 0.0
		for i := range s.values {
			val := s.shown(i)
			if !chartFinite(val) {
				continue
			}
			base := s.stackBase(i, val)
			lo, hi = math.Min(lo, base+val), math.Max(hi, base+val)
		}
		return lo, hi, true
	}
	if catAxis {
		if e, ok := s.source.(ChartSourceExtent); ok {
			return e.Extent()
		}
		return 0, 0, false
	}
	lo, hi := 0.0, 0.0
	xmin, xmax := v.XRange()
	for _, p := range s.samples {
		if p.HasValue() && p.X >= xmin && p.X <= xmax {
			lo, hi = math.Min(lo, p.Last), math.Max(hi, p.Last)
		}
	}
	return lo, hi, true
}

// stackBase returns what the bar of the category stands on: the bars of its
// stack group added before it, on the same side of zero
func (s *BarSeries) stackBase(i int, val float64) float64 {
	if s.stack == "" || s.pane == nil {
		return 0
	}
	base := 0.0
	for _, other := range s.pane.series {
		if other == ChartSeries(s) {
			break
		}
		b, ok := other.(*BarSeries)
		if !ok || b.stack != s.stack || !b.Visible() || b.horizontal != s.horizontal {
			continue
		}
		ov := b.shown(i)
		if chartFinite(ov) && (ov >= 0) == (val >= 0) {
			base += ov
		}
	}
	return base
}

// Prepare implements ChartSeries
func (s *BarSeries) Prepare(v *ChartViewport) {
	s.samples = nil
	if s.source != nil {
		xmin, xmax := v.XRange()
		s.samples = s.source.GetData(xmin, xmax, v.Group())
	}
}

// Draw implements ChartSeries
func (s *BarSeries) Draw(cnv *Canvas, v *ChartViewport) {
	s.rects = s.rects[:0]
	col := s.Color(v.Theme)
	cat, val := s.categoryAxis(v), s.valueAxis(v)
	zero := val.ToPixel(0)
	// A bar of 0 to a value off the axis' range is cut at the plot's edge
	clampV := func(p float64) float64 {
		if s.horizontal {
			return math.Max(v.X-2, math.Min(v.X+v.W+2, p))
		}
		return math.Max(v.Y-2, math.Min(v.Y+v.H+2, p))
	}
	if !chartFinite(zero) {
		// A log axis has no zero: the bars start at its bottom
		if s.horizontal {
			zero = v.X
		} else {
			zero = v.Y + v.H
		}
	}
	addBar := func(c0, c1, v0, v1 float64, index int, value float64, catName string) {
		v0, v1 = clampV(v0), clampV(v1)
		var r barRect
		if s.horizontal {
			r = barRect{x: math.Min(v0, v1), y: c0, w: math.Abs(v1 - v0), h: c1 - c0}
		} else {
			r = barRect{x: c0, y: math.Min(v0, v1), w: c1 - c0, h: math.Abs(v1 - v0)}
		}
		r.index, r.value, r.cat = index, value, catName
		s.rects = append(s.rects, r)
	}
	if s.byCategory() {
		slot, slots := v.Pane.barSlot(s)
		band := math.Abs(cat.ToPixel(1) - cat.ToPixel(0))
		group := band * (1 - s.gap)
		w := group / float64(slots)
		inner := w * s.innerGap
		cs, _ := cat.scale.(*CategoryScale)
		for i := range s.values {
			value := s.shown(i)
			if !chartFinite(value) {
				continue
			}
			center := cat.ToPixel(float64(i))
			c0 := center - group/2 + float64(slot)*w + inner/2
			c1 := c0 + w - inner
			base := s.stackBase(i, value)
			name := ""
			if cs != nil {
				name = cs.name(i)
			}
			addBar(c0, c1, val.ToPixel(base), val.ToPixel(base+value), i, value, name)
		}
	} else {
		// As wide as the closest two samples are apart, at most 30 pixels
		step := 30.0
		for i := 1; i < len(s.samples); i++ {
			if d := cat.ToPixel(s.samples[i].X) - cat.ToPixel(s.samples[i-1].X); d > 0 && d < step {
				step = d
			}
		}
		w := math.Max(1, step*(1-s.gap))
		for i, p := range s.samples {
			if !p.HasValue() {
				continue
			}
			c := cat.ToPixel(p.X)
			addBar(c-w/2, c+w/2, zero, val.ToPixel(p.Last), i, p.Last, "")
		}
	}

	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	for _, r := range s.rects {
		c := col
		if s.upDown && !s.byCategory() {
			sm := s.samples[r.index]
			c = v.Theme.Up
			if sm.Last < sm.First {
				c = v.Theme.Down
			}
		}
		p.Reset()
		x, y, w, h := r.x, r.y, r.w, r.h
		// Whole pixels: the edges of the bars sharp
		if w >= 3 {
			x, w = math.Round(x), math.Max(1, math.Round(x+w)-math.Round(x))
		}
		if h >= 3 {
			y, h = math.Round(y), math.Max(1, math.Round(y+h)-math.Round(y))
		}
		rad := math.Min(s.radius, math.Min(w, h)/2)
		if rad > 0.5 {
			addBarShape(p, x, y, w, h, rad, s.horizontal, r.value < 0)
		} else {
			p.Rect(x, y, w, h)
		}
		cnv.FillPath(p, c)
	}
	if s.labels {
		s.drawLabels(cnv, v)
	}
}

// addBarShape adds a bar rounded at its end only: the top of a column, the
// bottom of a negative one, the right end of a horizontal bar
func addBarShape(p *Path, x, y, w, h, r float64, horizontal, negative bool) {
	tl, tr, br, bl := false, false, false, false
	switch {
	case !horizontal && !negative:
		tl, tr = true, true
	case !horizontal && negative:
		bl, br = true, true
	case horizontal && !negative:
		tr, br = true, true
	default:
		tl, bl = true, true
	}
	rr := func(on bool) float64 {
		if on {
			return r
		}
		return 0
	}
	p.MoveTo(x+rr(tl), y)
	p.LineTo(x+w-rr(tr), y)
	if tr {
		p.Arc(x+w-r, y+r, r, -math.Pi/2, 0)
	}
	p.LineTo(x+w, y+h-rr(br))
	if br {
		p.Arc(x+w-r, y+h-r, r, 0, math.Pi/2)
	}
	p.LineTo(x+rr(bl), y+h)
	if bl {
		p.Arc(x+r, y+h-r, r, math.Pi/2, math.Pi)
	}
	p.LineTo(x, y+rr(tl))
	if tl {
		p.Arc(x+r, y+r, r, math.Pi, 3*math.Pi/2)
	}
	p.Close()
}

// drawLabels writes the values by the bars: past the end of each, or
// inside it when the plot has no room past it; none on a bar too small
func (s *BarSeries) drawLabels(cnv *Canvas, v *ChartViewport) {
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.85)
	val := s.valueAxis(v)
	_, lineH := cnv.MeasureTextRotated("0", 0)
	col := s.Color(v.Theme)
	for _, r := range s.rects {
		text := chartValueLabel(val.scale, r.value)
		tw := chartTextWidth(cnv, text)
		cnv.SetColor(v.Theme.Text)
		if s.horizontal {
			if r.h < lineH*0.8 {
				continue
			}
			neg := r.value < 0
			x, h := r.x+r.w+3, HAlignLeft
			if neg {
				x, h = r.x-3, HAlignRight
			}
			// No room past the end: inside it
			if (!neg && x+tw > v.X+v.W) || (neg && x-tw < v.X) {
				if r.w < tw+6 {
					continue
				}
				x, h = r.x+r.w-3, HAlignRight
				if neg {
					x, h = r.x+3, HAlignLeft
				}
				cnv.SetColor(chartContrastText(col))
			}
			cnv.SetHAlign(h)
			cnv.SetVAlign(VAlignCenter)
			cnv.DrawTextRotated(x, r.y+r.h/2, 0, text)
			continue
		}
		if tw > r.w+8 {
			continue
		}
		neg := r.value < 0
		y, va := r.y-2, VAlignBottom
		if neg {
			y, va = r.y+r.h+2, VAlignTop
		}
		if (!neg && y-lineH < v.Y) || (neg && y+lineH > v.Y+v.H) {
			if r.h < lineH+4 {
				continue
			}
			y, va = r.y+2, VAlignTop
			if neg {
				y, va = r.y+r.h-2, VAlignBottom
			}
			cnv.SetColor(chartContrastText(col))
		}
		cnv.SetHAlign(HAlignCenter)
		cnv.SetVAlign(va)
		cnv.DrawTextRotated(r.x+r.w/2, y, 0, text)
	}
}

// chartValueLabel formats a value as a label of its own (on a bar, a slice):
// as many decimals as the value has, at most three significant digits,
// with the unit and the formatter of the scale
func chartValueLabel(scale ChartScale, v float64) string {
	switch s := scale.(type) {
	case *LinearScale:
		if s.Formatter != nil {
			return s.Formatter(v)
		}
		// Three significant digits, without the zeros at the end
		step := 1.0
		if a := math.Abs(v); a > 0 {
			step = math.Pow(10, math.Floor(math.Log10(a))-2)
		}
		return trimChartZeros(FormatChartNumber(v, step, s.SI)) + s.Unit
	case *LogScale:
		return s.label(v)
	case *SymlogScale:
		return s.label(v)
	}
	return scale.Format(v)
}

// Nearest implements ChartSeries
func (s *BarSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	best := -1
	bestD := math.Inf(1)
	for i, r := range s.rects {
		// The distance to the bar, 0 inside it; along the categories only
		// when byX
		dx := math.Max(0, math.Max(r.x-px, px-(r.x+r.w)))
		dy := math.Max(0, math.Max(r.y-py, py-(r.y+r.h)))
		d := math.Hypot(dx, dy)
		if byX {
			if s.horizontal {
				d = dy
			} else {
				d = dx
			}
		}
		if d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return ChartHit{}, false
	}
	r := s.rects[best]
	cat, val := s.categoryAxis(v), s.valueAxis(v)
	h := ChartHit{
		Series: s, Index: r.index, Y: r.value, Name: s.name, Color: s.Color(v.Theme),
		YLabel: val.scale.Format(r.value), dist: bestD,
		PX: r.x + r.w/2, PY: r.y,
	}
	if s.horizontal {
		h.PX, h.PY = r.x+r.w, r.y+r.h/2
	}
	if s.byCategory() {
		h.X = float64(r.index)
		h.XLabel = r.cat
	} else {
		h.Sample = s.samples[r.index]
		h.X = h.Sample.X
		h.XLabel = cat.scale.Format(h.X)
	}
	return h, true
}

// Legend implements ChartSeries
func (s *BarSeries) Legend(th *ChartTheme) []ChartLegendItem {
	return s.legendItem(th, ChartLegendBox)
}

// chartXMargin asks the horizontal axis for a margin when the bars stand
// along it by their samples: the first and the last are not cut in half
func (s *BarSeries) chartXMargin() float64 {
	if s.byCategory() || s.horizontal {
		return 0
	}
	return 0.02
}
