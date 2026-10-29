package ui

import (
	"math"
)

// ChartPane is a plot of a chart: a rectangle with its axes, the series in
// it and the layers over and under them. The panes of a chart are stacked
// top to bottom (the price over the volume), each as tall as its weight
// says, and share the horizontal axis of the chart unless given their own.
type ChartPane struct {
	view   *ChartView
	weight float64
	xAxis  *ChartAxis
	// The axes of the pane: "y" on the left, and the ones added
	axes   []*ChartAxis
	series []ChartSeries
	under  []ChartLayer
	over   []ChartLayer
	// The annotations, made on the first use
	annotations *ChartAnnotations
	// The stack groups shown in percent of their total
	percent map[string]bool
	// The categories of the radars of the pane: the spokes
	radarCats []string

	// The plot of the last paint, in logical pixels of the widget
	x, y, w, h float64
	// The legend strip above the plot and its entries
	legendY, legendH float64
	legendItems      []legendHit
}

// ChartLayer is drawn in a pane over or under its series: annotations, zones,
// the drawing of an application's own
type ChartLayer interface {
	DrawLayer(cnv *Canvas, v *ChartViewport)
}

// ChartLayerFunc is a ChartLayer of a function
type ChartLayerFunc func(cnv *Canvas, v *ChartViewport)

// DrawLayer implements ChartLayer
func (f ChartLayerFunc) DrawLayer(cnv *Canvas, v *ChartViewport) { f(cnv, v) }

type legendHit struct {
	x, y, w, h float64
	series     ChartSeries
	index      int
}

func newChartPane(view *ChartView) *ChartPane {
	p := &ChartPane{view: view, weight: 1}
	p.xAxis = view.xAxis
	p.axes = []*ChartAxis{newChartAxis(view, "y", ChartAxisLeft, NewLinearScale(0, 1))}
	return p
}

// View returns the chart of the pane
func (p *ChartPane) View() *ChartView { return p.view }

// SetWeight sets how tall the pane is next to the others: a pane of weight
// 3 over one of weight 1 takes three quarters of the height
func (p *ChartPane) SetWeight(weight float64) *ChartPane {
	p.weight = math.Max(weight, 0.01)
	p.view.Update()
	return p
}

// XAxis returns the horizontal axis of the pane, shared by all the panes of
// the chart unless SetXAxis gave it its own
func (p *ChartPane) XAxis() *ChartAxis { return p.xAxis }

// SetXAxis gives the pane a horizontal axis of its own
func (p *ChartPane) SetXAxis(a *ChartAxis) *ChartPane {
	if a != nil {
		a.view = p.view
		p.xAxis = a
		p.view.Update()
	}
	return p
}

// NewAxis makes an axis for SetXAxis
func (p *ChartPane) NewAxis(name string, pos ChartAxisPosition, scale ChartScale) *ChartAxis {
	return newChartAxis(p.view, name, pos, scale)
}

// YAxis returns the vertical axis "y" of the pane
func (p *ChartPane) YAxis() *ChartAxis { return p.axes[0] }

// AddAxis adds an axis to the pane: a second vertical axis on the right for
// a series of other units. The series are put on it by SetAxisNames.
func (p *ChartPane) AddAxis(name string, pos ChartAxisPosition, scale ChartScale) *ChartAxis {
	if scale == nil {
		scale = NewLinearScale(0, 1)
	}
	a := newChartAxis(p.view, name, pos, scale)
	if pos != ChartAxisBottom && pos != ChartAxisTop {
		a.grid = false // the grid of the first vertical axis is enough
	}
	p.axes = append(p.axes, a)
	p.view.Update()
	return a
}

// Axis returns the axis of the name: the pane's own or the horizontal one;
// nil when there is none
func (p *ChartPane) Axis(name string) *ChartAxis {
	for _, a := range p.axes {
		if a.name == name {
			return a
		}
	}
	if p.xAxis != nil && (p.xAxis.name == name || name == "x") {
		return p.xAxis
	}
	return nil
}

// Axes returns the vertical axes of the pane and those it added
func (p *ChartPane) Axes() []*ChartAxis { return p.axes }

// AddSeries adds the series to the pane, drawn over those added before
func (p *ChartPane) AddSeries(s ChartSeries) ChartSeries {
	b := s.base()
	if b.pane != nil {
		b.pane.RemoveSeries(s)
	}
	b.pane = p
	if !b.paletteSet && b.col == nil {
		b.palette = p.view.nextPalette
		p.view.nextPalette++
	}
	p.series = append(p.series, s)
	switch ls := s.(type) {
	case *LineSeries:
		ls.watched = false
		ls.watchMemory()
	case *BarSeries:
		if ls.memory != nil {
			ls.SetSource(ls.memory)
		}
	case *CandleSeries:
		ls.watch()
	}
	p.view.Update()
	return s
}

// RemoveSeries takes the series out of the pane
func (p *ChartPane) RemoveSeries(s ChartSeries) {
	for i, o := range p.series {
		if o == s {
			p.series = append(p.series[:i:i], p.series[i+1:]...)
			s.base().pane = nil
			p.view.Update()
			return
		}
	}
}

// Series returns the series of the pane, in the order they are drawn
func (p *ChartPane) Series() []ChartSeries { return p.series }

// AddLine adds a line of the name
func (p *ChartPane) AddLine(name string) *LineSeries {
	s := NewLineSeries(name)
	p.AddSeries(s)
	return s
}

// AddArea adds a line with the area under it filled
func (p *ChartPane) AddArea(name string) *LineSeries {
	s := NewLineSeries(name)
	s.fill = true
	p.AddSeries(s)
	return s
}

// AddBars adds bars of the name; on a category axis the categories are
// added by BarSeries.Set
func (p *ChartPane) AddBars(name string) *BarSeries {
	s := NewBarSeries(name)
	p.AddSeries(s)
	return s
}

// AddCandles adds candles of the name
func (p *ChartPane) AddCandles(name string) *CandleSeries {
	s := NewCandleSeries(name)
	p.AddSeries(s)
	return s
}

// AddPie adds a pie of the name; a pane with only pies has no axes
func (p *ChartPane) AddPie(name string) *PieSeries {
	s := NewPieSeries(name)
	p.AddSeries(s)
	return s
}

// AddLayer adds a layer drawn over the series, or under them
func (p *ChartPane) AddLayer(layer ChartLayer, over bool) {
	if over {
		p.over = append(p.over, layer)
	} else {
		p.under = append(p.under, layer)
	}
	p.view.Update()
}

// Annotations returns the annotations of the pane: lines, zones and labels
// over its series
func (p *ChartPane) Annotations() *ChartAnnotations {
	if p.annotations == nil {
		p.annotations = &ChartAnnotations{pane: p}
	}
	return p.annotations
}

// Plot returns the plot of the pane of the last paint, in logical pixels of
// the widget
func (p *ChartPane) Plot() (x, y, w, h float64) { return p.x, p.y, p.w, p.h }

// noAxes reports whether the pane draws no axes: it has only pies
func (p *ChartPane) noAxes() bool {
	if len(p.series) == 0 {
		return false
	}
	for _, s := range p.series {
		if _, ok := s.(interface{ chartNoAxes() }); !ok {
			return false
		}
	}
	return true
}

// seriesAxes returns the axes the series is drawn on in the pane
func (p *ChartPane) seriesAxes(s ChartSeries) (x, y *ChartAxis) {
	xn, yn := s.AxisNames()
	x, y = p.Axis(xn), p.Axis(yn)
	if x == nil {
		x = p.xAxis
	}
	if y == nil {
		y = p.axes[0]
	}
	return x, y
}

// viewport returns the viewport of the series' axes over the plot
func (p *ChartPane) viewport(x, y *ChartAxis) *ChartViewport {
	return &ChartViewport{
		Pane: p, XAxis: x, YAxis: y,
		X: p.x, Y: p.y, W: p.w, H: p.h,
		Theme: p.view.theme, FontFamily: p.view.FontFamily(), FontSize: p.view.FontSize(),
	}
}

// barSlot returns the place of the bars among the bar series of the pane on
// the same categories: the stack groups and the bars without one stand side
// by side
func (p *ChartPane) barSlot(s *BarSeries) (slot, slots int) {
	slot = -1
	var groups []string
	for _, o := range p.series {
		b, ok := o.(*BarSeries)
		if !ok || !b.Visible() || !b.byCategory() || b.horizontal != s.horizontal {
			continue
		}
		key := b.stack
		if key == "" {
			key = "\x00" + string(rune(len(groups)))
		}
		idx := -1
		for i, g := range groups {
			if g == key {
				idx = i
			}
		}
		if idx < 0 {
			groups = append(groups, key)
			idx = len(groups) - 1
		}
		if b == s {
			slot = idx
		}
	}
	if slot < 0 {
		return 0, 1
	}
	return slot, len(groups)
}

// SetStackPercent shows the lines and the bars of the stack group in
// percent of their total: a 100% stacked area or bar chart
func (p *ChartPane) SetStackPercent(group string, percent bool) *ChartPane {
	if p.percent == nil {
		p.percent = map[string]bool{}
	}
	p.percent[group] = percent
	p.view.Update()
	return p
}

// stackPercent reports whether the group is stacked in percent
func (p *ChartPane) stackPercent(group string) bool {
	return group != "" && p.percent[group]
}

// computeStacks puts each stacked line on the lines of its group added
// before it, in the order of the series, once all of them are prepared: the
// base under each sample, and its factor to percent of the group's total
func (p *ChartPane) computeStacks() {
	for _, o := range p.series {
		s, ok := o.(*LineSeries)
		if !ok || s.stack == "" || !s.Visible() {
			continue
		}
		s.bases, s.factors = nil, nil
		if p.stackPercent(s.stack) {
			s.factors = make([]float64, len(s.samples))
			for i, sm := range s.samples {
				if total := p.groupTotal(s.stack, sm.X); total != 0 {
					s.factors[i] = 100 / total
				}
			}
		}
		// The line stands on the top of the line of its group just under it,
		// which stands on the ones under that
		var under *LineSeries
		for _, o2 := range p.series {
			if o2 == o {
				break
			}
			if ls, ok := o2.(*LineSeries); ok && ls.stack == s.stack && ls.Visible() && len(ls.samples) > 0 {
				under = ls
			}
		}
		bases := make([]float64, len(s.samples))
		if under != nil {
			for i, sm := range s.samples {
				bases[i] = interpolateStacked(under, sm.X)
			}
		}
		s.bases = bases
	}
}

// groupTotal returns the sum of the values of the lines of the group at x,
// interpolated between their samples
func (p *ChartPane) groupTotal(group string, x float64) float64 {
	total := 0.0
	for _, o := range p.series {
		ls, ok := o.(*LineSeries)
		if !ok || ls.stack != group || !ls.Visible() {
			continue
		}
		total += interpolateRaw(ls.samples, x)
	}
	return total
}

// interpolateRaw returns the value of the samples at x, interpolated
// between them; 0 outside them or in a gap
func interpolateRaw(sm []ChartSample, x float64) float64 {
	j := nearestSampleIndex(sm, x)
	if j < 0 {
		return 0
	}
	if sm[j].X == x {
		if sm[j].HasValue() {
			return sm[j].Last
		}
		return 0
	}
	a, b := j, j
	if sm[j].X < x {
		b = j + 1
	} else {
		a = j - 1
	}
	if a < 0 || b >= len(sm) || !sm[a].HasValue() || !sm[b].HasValue() || sm[b].X == sm[a].X {
		return 0
	}
	t := (x - sm[a].X) / (sm[b].X - sm[a].X)
	return sm[a].Last + (sm[b].Last-sm[a].Last)*t
}

// interpolateStacked returns the top of the stacked line at x: its value
// with its own base, interpolated between its samples, 0 outside them
func interpolateStacked(ls *LineSeries, x float64) float64 {
	sm := ls.samples
	j := nearestSampleIndex(sm, x)
	if j < 0 {
		return 0
	}
	top := func(k int) (float64, bool) {
		if !sm[k].HasValue() {
			return 0, false
		}
		return ls.stacked(k, sm[k].Last), true
	}
	if sm[j].X == x {
		v, _ := top(j)
		return v
	}
	// The samples on both sides of x
	a, b := j, j
	if sm[j].X < x {
		b = j + 1
	} else {
		a = j - 1
	}
	if a < 0 || b >= len(sm) {
		return 0
	}
	va, okA := top(a)
	vb, okB := top(b)
	if !okA || !okB || sm[b].X == sm[a].X {
		return 0
	}
	t := (x - sm[a].X) / (sm[b].X - sm[a].X)
	return va + (vb-va)*t
}
