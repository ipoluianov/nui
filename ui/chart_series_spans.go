package ui

import (
	"image/color"
	"math"
	"sort"
	"strconv"
)

// rowAxis returns the category scale of the rows of the pane, making the
// pane's y axis one: the first row at the top, no margin, no grid
func (p *ChartPane) rowAxis() *CategoryScale {
	a := p.YAxis()
	if cs, ok := a.scale.(*CategoryScale); ok {
		return cs
	}
	cs := NewCategoryScale()
	a.scale = cs
	a.inverted = true
	a.margin = 0
	a.grid = false
	a.includeZero = false
	return cs
}

// rowIndex returns the row of the name, adding it to the rows
func (p *ChartPane) rowIndex(name string) int {
	cs := p.rowAxis()
	if i := cs.Index(name); i >= 0 {
		return i
	}
	cs.SetCategories(append(append([]string(nil), cs.categories...), name))
	return len(cs.categories) - 1
}

// ---------------------------------------------------------------------------
// Gantt and state timeline

// ChartSpan is a span of a SpanSeries: a task of a Gantt chart, a state of
// a machine. A span with End equal to Start is a milestone.
type ChartSpan struct {
	ID         string
	Row        string
	Start, End float64
	Label      string
	// State colors the span by the series' colors of the states
	// (SetStateColor); without one, by Color or the row
	State string
	Color *color.RGBA
	// Progress 0..1 darkens the part of a task done
	Progress float64
}

// SpanSeries draws spans of time in rows: the tasks of a Gantt chart with
// their progress, milestones and dependencies, or the states of machines
// over a shift (working, idle, down). The rows are the categories of the
// pane's vertical axis, the first at the top.
//
//	g := chart.Pane(0).AddSpans("Plan")
//	g.Add(ui.ChartSpan{ID: "a", Row: "Design", Start: t0, End: t1, Progress: 1})
//	g.Add(ui.ChartSpan{ID: "b", Row: "Build", Start: t1, End: t2})
//	g.AddDependency("a", "b")
type SpanSeries struct {
	ChartSeriesBase
	spans  []ChartSpan
	states map[string]color.RGBA
	deps   [][2]string
	labels bool

	rects []spanRect
}

type spanRect struct {
	x, y, w, h float64
	index      int
}

// NewSpanSeries makes spans of the name
func NewSpanSeries(name string) *SpanSeries {
	s := &SpanSeries{labels: true}
	s.name = name
	return s
}

// AddSpans adds spans to the pane, its vertical axis becoming the rows
func (p *ChartPane) AddSpans(name string) *SpanSeries {
	p.rowAxis()
	s := NewSpanSeries(name)
	p.AddSeries(s)
	return s
}

// Add adds a span; its row is added to the rows when there is none of it
func (s *SpanSeries) Add(span ChartSpan) *SpanSeries {
	if span.End < span.Start {
		span.Start, span.End = span.End, span.Start
	}
	s.spans = append(s.spans, span)
	if s.pane != nil {
		s.pane.rowIndex(span.Row)
	}
	s.update()
	return s
}

// SetStateColor sets the color of the spans of the state
func (s *SpanSeries) SetStateColor(state string, col color.RGBA) *SpanSeries {
	if s.states == nil {
		s.states = map[string]color.RGBA{}
	}
	s.states[state] = col
	s.update()
	return s
}

// AddDependency draws an arrow from the end of the span of the id from to
// the start of the span of the id to
func (s *SpanSeries) AddDependency(from, to string) *SpanSeries {
	s.deps = append(s.deps, [2]string{from, to})
	s.update()
	return s
}

// SetLabels writes the labels on the spans (on by default)
func (s *SpanSeries) SetLabels(show bool) *SpanSeries {
	s.labels = show
	s.update()
	return s
}

// Spans returns the spans
func (s *SpanSeries) Spans() []ChartSpan { return s.spans }

func (s *SpanSeries) spanColor(sp ChartSpan, row int, th *ChartTheme) color.RGBA {
	if sp.State != "" {
		if c, ok := s.states[sp.State]; ok {
			return c
		}
	}
	if sp.Color != nil {
		return *sp.Color
	}
	if b := s.base(); b.col != nil {
		return *b.col
	}
	return th.SeriesColor(row)
}

// Range implements ChartSeries
func (s *SpanSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if len(s.spans) == 0 {
		return 0, 0, false
	}
	if !axis.Horizontal() {
		cs := s.pane.rowAxis()
		return -0.5, float64(len(cs.categories)) - 0.5, true
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, sp := range s.spans {
		lo, hi = math.Min(lo, sp.Start), math.Max(hi, sp.End)
	}
	return lo, hi, true
}

// chartXMargin: the first and the last spans, the milestones, are not at
// the very edges
func (s *SpanSeries) chartXMargin() float64 { return 0.04 }

// Prepare implements ChartSeries
func (s *SpanSeries) Prepare(v *ChartViewport) {}

// Draw implements ChartSeries
func (s *SpanSeries) Draw(cnv *Canvas, v *ChartViewport) {
	s.rects = s.rects[:0]
	cs := s.pane.rowAxis()
	band := math.Abs(v.PY(1) - v.PY(0))
	h := math.Max(4, band*0.62)
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.85)
	byID := map[string]spanRect{}
	for i, sp := range s.spans {
		row := cs.Index(sp.Row)
		if row < 0 {
			continue
		}
		col := s.spanColor(sp, row, v.Theme)
		cy := v.PY(float64(row))
		x1, x2 := v.PX(sp.Start), v.PX(sp.End)
		if x2 < v.X-20 || x1 > v.X+v.W+20 {
			continue
		}
		r := spanRect{x: x1, y: cy - h/2, w: x2 - x1, h: h, index: i}
		if sp.End == sp.Start {
			// A milestone
			d := h * 0.45
			cnv.FillPolygon([]PointF{{x1, cy - d}, {x1 + d, cy}, {x1, cy + d}, {x1 - d, cy}}, col)
			r.x, r.w = x1-d, 2*d
			if s.labels && sp.Label != "" {
				chartDrawLabel(cnv, sp.Label, x1+d+4, cy, HAlignLeft, VAlignCenter, v.Theme.Text)
			}
		} else {
			p.Reset()
			p.RoundedRect(math.Round(x1), math.Round(cy-h/2), math.Max(1, math.Round(x2)-math.Round(x1)), math.Round(h), math.Min(3, h/3))
			fill := col
			if sp.Progress > 0 {
				fill = MixColors(v.Theme.PlotBackground, col, 0.45)
			}
			cnv.FillPath(p, fill)
			if sp.Progress > 0 {
				p.Reset()
				done := (x2 - x1) * math.Min(1, sp.Progress)
				p.RoundedRect(math.Round(x1), math.Round(cy-h/2), math.Max(1, math.Round(done)), math.Round(h), math.Min(3, h/3))
				cnv.FillPath(p, col)
			}
			if s.labels && sp.Label != "" {
				tw := chartTextWidth(cnv, sp.Label)
				if tw+12 <= x2-x1 {
					chartDrawLabel(cnv, sp.Label, math.Max(x1, v.X)+7, cy, HAlignLeft, VAlignCenter, chartContrastText(col))
				} else if sp.ID != "" || sp.Progress > 0 {
					chartDrawLabel(cnv, sp.Label, x2+4, cy, HAlignLeft, VAlignCenter, v.Theme.Text)
				}
			}
		}
		s.rects = append(s.rects, r)
		if sp.ID != "" {
			byID[sp.ID] = r
		}
	}
	// The dependencies: from the end of one to the start of the other
	p.Reset()
	for _, d := range s.deps {
		a, okA := byID[d[0]]
		b, okB := byID[d[1]]
		if !okA || !okB {
			continue
		}
		ax, ay := a.x+a.w, a.y+a.h/2
		bx, by := b.x, b.y+b.h/2
		mid := math.Max(ax+6, math.Min(bx-6, ax+6))
		p.MoveTo(ax, chartCrisp(ay))
		p.LineTo(chartCrisp(mid), chartCrisp(ay))
		p.LineTo(chartCrisp(mid), chartCrisp(by))
		p.LineTo(bx-1, chartCrisp(by))
		cnv.FillPolygon([]PointF{{bx, by}, {bx - 5, by - 3.5}, {bx - 5, by + 3.5}}, v.Theme.Axis)
	}
	cnv.StrokePath(p, StrokeStyle{Width: 1}, v.Theme.Axis)
}

// Nearest implements ChartSeries
func (s *SpanSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	for k := len(s.rects) - 1; k >= 0; k-- {
		r := s.rects[k]
		if px >= r.x-2 && px <= r.x+r.w+2 && py >= r.y && py <= r.y+r.h {
			sp := s.spans[r.index]
			f := v.XAxis.scale.Format
			yl := f(sp.Start)
			if sp.End != sp.Start {
				yl += " … " + f(sp.End)
			}
			if sp.Progress > 0 {
				yl += " (" + strconv.Itoa(int(math.Round(sp.Progress*100))) + "%)"
			}
			name := sp.Label
			if name == "" {
				name = sp.State
			}
			cs := s.pane.rowAxis()
			return ChartHit{Series: s, Index: r.index, X: sp.Start, PX: px, PY: r.y + r.h/2,
				XLabel: sp.Row, YLabel: yl, Name: name, Color: s.spanColor(sp, cs.Index(sp.Row), v.Theme)}, true
		}
	}
	return ChartHit{}, false
}

// Legend implements ChartSeries: an entry per state
func (s *SpanSeries) Legend(th *ChartTheme) []ChartLegendItem {
	if len(s.states) == 0 {
		return nil
	}
	names := make([]string, 0, len(s.states))
	for n := range s.states {
		names = append(names, n)
	}
	sort.Strings(names)
	items := make([]ChartLegendItem, 0, len(names))
	for _, n := range names {
		items = append(items, ChartLegendItem{Name: n, Color: s.states[n], Kind: ChartLegendBox, Index: -1})
	}
	return items
}

// ---------------------------------------------------------------------------
// Digital signals

// DigitalSeries draws a digital signal in a row of its own, as a logic
// analyzer does: a binary one as a trace high and low, a bus (SetBus) as
// segments with their values written. Several signals share the time axis
// with the analog panes.
//
//	valve := chart.Pane(0).AddDigital("Valve open")
//	valve.AddState(t, 1)
type DigitalSeries struct {
	ChartSeriesBase
	source ChartSource
	memory *ChartMemorySource
	bus    bool
	names  map[float64]string

	samples []ChartSample
}

// AddDigital adds a digital signal to the pane: a row of its own named as
// the signal
func (p *ChartPane) AddDigital(name string) *DigitalSeries {
	s := &DigitalSeries{}
	s.name = name
	s.memory = NewChartMemorySource()
	s.source = s.memory
	p.rowIndex(name)
	p.AddSeries(s)
	s.watch()
	return s
}

func (s *DigitalSeries) watch() {
	if s.memory != nil && s.pane != nil && s.pane.view != nil {
		view := s.pane.view
		s.memory.watch(func() { view.RequestRepaint() })
	}
}

// AddState adds the value of the signal from x on: 0 and 1 for a binary
// one, any number for a bus
func (s *DigitalSeries) AddState(x, value float64) *DigitalSeries {
	if s.memory == nil {
		s.SetSource(NewChartMemorySource())
	}
	s.memory.AppendValue(x, value)
	return s
}

// SetSource draws the signal from the source: each sample is the value from
// its x to the next sample
func (s *DigitalSeries) SetSource(src ChartSource) *DigitalSeries {
	s.source = src
	s.memory, _ = src.(*ChartMemorySource)
	s.watch()
	s.update()
	return s
}

// SetBus draws the signal as a bus: segments with their values written
func (s *DigitalSeries) SetBus(bus bool) *DigitalSeries {
	s.bus = bus
	s.update()
	return s
}

// SetStateName writes the name instead of the value of a bus' segments:
// 0 "Idle", 1 "Run", 2 "Fault"
func (s *DigitalSeries) SetStateName(value float64, name string) *DigitalSeries {
	if s.names == nil {
		s.names = map[float64]string{}
	}
	s.names[value] = name
	s.update()
	return s
}

func (s *DigitalSeries) valueText(v float64) string {
	if n, ok := s.names[v]; ok {
		return n
	}
	return trimChartZeros(strconv.FormatFloat(v, 'f', 3, 64))
}

// Range implements ChartSeries
func (s *DigitalSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if !axis.Horizontal() {
		return -0.5, float64(len(s.pane.rowAxis().categories)) - 0.5, true
	}
	if e, ok := s.source.(ChartSourceExtent); ok {
		return e.Extent()
	}
	return 0, 0, false
}

// Prepare implements ChartSeries: the samples as they are, a change is
// never merged away
func (s *DigitalSeries) Prepare(v *ChartViewport) {
	s.samples = nil
	if s.source != nil {
		xmin, xmax := v.XRange()
		s.samples = s.source.GetData(xmin, xmax, 0)
	}
}

// Draw implements ChartSeries
func (s *DigitalSeries) Draw(cnv *Canvas, v *ChartViewport) {
	row := s.pane.rowAxis().Index(s.name)
	if row < 0 || len(s.samples) == 0 {
		return
	}
	col := s.Color(v.Theme)
	band := math.Abs(v.PY(1) - v.PY(0))
	cy := v.PY(float64(row))
	hi, lo := cy-band*0.3, cy+band*0.3
	end := v.X + v.W
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	if !s.bus {
		fill := col
		fill.A = 40
		p.Reset()
		var trace []PointF
		for i, sm := range s.samples {
			x := v.PX(sm.X)
			next := end
			if i+1 < len(s.samples) {
				next = v.PX(s.samples[i+1].X)
			}
			y := lo
			if sm.HasValue() && sm.Last != 0 {
				y = hi
				p.Rect(x, hi, next-x, lo-hi)
			}
			if !sm.HasValue() {
				y = cy
			}
			trace = append(trace, PointF{x, chartCrisp(y)}, PointF{next, chartCrisp(y)})
		}
		cnv.FillPath(p, fill)
		cnv.DrawPolyline(trace, StrokeStyle{Width: 1.5}, col)
		return
	}
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.8)
	for i, sm := range s.samples {
		x1 := v.PX(sm.X)
		x2 := end
		if i+1 < len(s.samples) {
			x2 = v.PX(s.samples[i+1].X)
		}
		if x2 < v.X || x1 > end {
			continue
		}
		slant := math.Min(4, (x2-x1)/2)
		p.Reset()
		p.Polygon(PointF{x1, cy}, PointF{x1 + slant, hi}, PointF{x2 - slant, hi}, PointF{x2, cy}, PointF{x2 - slant, lo}, PointF{x1 + slant, lo})
		fill := col
		fill.A = 50
		if !sm.HasValue() {
			fill = v.Theme.BadHatch
		}
		cnv.FillPath(p, fill)
		cnv.StrokePath(p, StrokeStyle{Width: 1}, col)
		if sm.HasValue() {
			text := s.valueText(sm.Last)
			if chartTextWidth(cnv, text)+2*slant+4 <= x2-x1 {
				chartDrawLabel(cnv, text, (math.Max(x1, v.X)+math.Min(x2, end))/2, cy, HAlignCenter, VAlignCenter, v.Theme.Title)
			}
		}
	}
}

// Nearest implements ChartSeries
func (s *DigitalSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	row := s.pane.rowAxis().Index(s.name)
	cy := v.PY(float64(row))
	band := math.Abs(v.PY(1) - v.PY(0))
	if !byX && math.Abs(py-cy) > band/2 {
		return ChartHit{}, false
	}
	x := v.DataX(px)
	i := sort.Search(len(s.samples), func(i int) bool { return s.samples[i].X > x }) - 1
	if i < 0 {
		return ChartHit{}, false
	}
	sm := s.samples[i]
	yl := "no data"
	if sm.HasValue() {
		yl = s.valueText(sm.Last)
	}
	return ChartHit{Series: s, Index: i, X: sm.X, Y: sm.Last, PX: px, PY: cy, dist: math.Abs(py - cy),
		XLabel: v.XAxis.scale.Format(x), YLabel: yl, Name: s.name, Color: s.Color(v.Theme), Sample: sm}, true
}

// Legend implements ChartSeries: the row names the signal
func (s *DigitalSeries) Legend(th *ChartTheme) []ChartLegendItem { return nil }

// ---------------------------------------------------------------------------
// Events and alarms

// ChartSeverity is how serious an event is
type ChartSeverity int

const (
	ChartSeverityInfo ChartSeverity = iota
	ChartSeverityWarning
	ChartSeverityCritical
)

// ChartEvent is an event of an EventSeries: when, from where, how serious
type ChartEvent struct {
	X        float64
	Source   string
	Severity ChartSeverity
	Text     string
}

// EventSeries draws events and alarms as marks in a row per source, colored
// by their severity; marks too close to tell apart are drawn as one with
// their count. A click on a mark comes as OnPointClicked: the application
// acknowledges the alarm.
//
//	ev := chart.Pane(1).AddEvents("Alarms")
//	ev.Add(ui.ChartEvent{X: t, Source: "Pump 2", Severity: ui.ChartSeverityCritical, Text: "Overheat"})
type EventSeries struct {
	ChartSeriesBase
	events []ChartEvent

	marks []eventMark
}

type eventMark struct {
	x, y    float64
	indexes []int
}

// AddEvents adds events to the pane, its vertical axis becoming the rows of
// the sources
func (p *ChartPane) AddEvents(name string) *EventSeries {
	p.rowAxis()
	s := &EventSeries{}
	s.name = name
	p.AddSeries(s)
	return s
}

// Add adds an event; its source is added to the rows
func (s *EventSeries) Add(e ChartEvent) *EventSeries {
	i := sort.Search(len(s.events), func(i int) bool { return s.events[i].X > e.X })
	s.events = append(s.events, ChartEvent{})
	copy(s.events[i+1:], s.events[i:])
	s.events[i] = e
	if s.pane != nil {
		s.pane.rowIndex(e.Source)
	}
	s.update()
	return s
}

// Events returns the events, sorted by x
func (s *EventSeries) Events() []ChartEvent { return s.events }

// Clear removes the events
func (s *EventSeries) Clear() *EventSeries {
	s.events = nil
	s.update()
	return s
}

func severityColor(sev ChartSeverity, th *ChartTheme) color.RGBA {
	switch sev {
	case ChartSeverityCritical:
		return th.Critical
	case ChartSeverityWarning:
		return th.Warning
	}
	return th.SeriesColor(0)
}

// Range implements ChartSeries
func (s *EventSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if len(s.events) == 0 {
		return 0, 0, false
	}
	if !axis.Horizontal() {
		return -0.5, float64(len(s.pane.rowAxis().categories)) - 0.5, true
	}
	return s.events[0].X, s.events[len(s.events)-1].X, true
}

func (s *EventSeries) chartXMargin() float64 { return 0.02 }

// Prepare implements ChartSeries
func (s *EventSeries) Prepare(v *ChartViewport) {}

// Draw implements ChartSeries
func (s *EventSeries) Draw(cnv *Canvas, v *ChartViewport) {
	s.marks = s.marks[:0]
	cs := s.pane.rowAxis()
	// The marks of each row, merged when closer than 10 pixels
	last := map[int]int{}
	for i, e := range s.events {
		row := cs.Index(e.Source)
		x := v.PX(e.X)
		if row < 0 || x < v.X-10 || x > v.X+v.W+10 {
			continue
		}
		if k, ok := last[row]; ok && x-s.marks[k].x < 10 {
			s.marks[k].indexes = append(s.marks[k].indexes, i)
			continue
		}
		s.marks = append(s.marks, eventMark{x: x, y: v.PY(float64(row)), indexes: []int{i}})
		last[row] = len(s.marks) - 1
	}
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.75)
	for _, m := range s.marks {
		worst := ChartSeverityInfo
		for _, i := range m.indexes {
			if s.events[i].Severity > worst {
				worst = s.events[i].Severity
			}
		}
		col := severityColor(worst, v.Theme)
		shape := MarkerCircle
		switch worst {
		case ChartSeverityWarning:
			shape = MarkerTriangleUp
		case ChartSeverityCritical:
			shape = MarkerDiamond
		}
		if len(m.indexes) == 1 {
			cnv.DrawMarker(shape, m.x, m.y, 10, col, v.Theme.PlotBackground)
			continue
		}
		// A badge of how many
		text := strconv.Itoa(len(m.indexes))
		tw, th := cnv.MeasureTextRotated(text, 0)
		w := math.Max(tw+8, th+2)
		p := pathPool.Get().(*Path)
		p.Reset()
		p.RoundedRect(m.x-w/2, m.y-th/2-1, w, th+2, (th+2)/2)
		cnv.FillPath(p, col)
		pathPool.Put(p)
		chartDrawLabel(cnv, text, m.x, m.y, HAlignCenter, VAlignCenter, chartContrastText(col))
	}
}

// Nearest implements ChartSeries: the mark under the pointer
func (s *EventSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	best, bestD := -1, math.Inf(1)
	for k, m := range s.marks {
		d := math.Hypot(m.x-px, m.y-py)
		if d < bestD {
			best, bestD = k, d
		}
	}
	if best < 0 || bestD > 12 {
		return ChartHit{}, false
	}
	m := s.marks[best]
	e := s.events[m.indexes[len(m.indexes)-1]]
	yl := e.Text
	if len(m.indexes) > 1 {
		yl = strconv.Itoa(len(m.indexes)) + " events, the last: " + e.Text
	}
	return ChartHit{Series: s, Index: m.indexes[len(m.indexes)-1], X: e.X, PX: m.x, PY: m.y, dist: bestD,
		XLabel: v.XAxis.scale.Format(e.X), YLabel: yl, Name: e.Source, Color: severityColor(e.Severity, v.Theme)}, true
}

// Legend implements ChartSeries
func (s *EventSeries) Legend(th *ChartTheme) []ChartLegendItem { return nil }
