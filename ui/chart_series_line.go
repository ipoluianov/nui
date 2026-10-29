package ui

import (
	"image/color"
	"math"
	"sort"
)

// ChartLineShape is how a line goes from one point to the next
type ChartLineShape int

const (
	// ChartLineStraight joins the points by straight lines
	ChartLineStraight ChartLineShape = iota
	// ChartLineStepAfter keeps the value until the next point, then steps
	ChartLineStepAfter
	// ChartLineStepBefore steps to the next value at once
	ChartLineStepBefore
	// ChartLineStepMiddle steps half way to the next point
	ChartLineStepMiddle
	// ChartLineSpline is a smooth curve through the points that does not
	// overshoot them (a monotone cubic)
	ChartLineSpline
)

// ChartMarkerMode is when a line draws a marker at each point
type ChartMarkerMode int

const (
	// ChartMarkersAuto draws them while the points are at least 8 pixels
	// apart and there are no more than 200 of them
	ChartMarkersAuto ChartMarkerMode = iota
	ChartMarkersAlways
	ChartMarkersNever
)

// ChartGapMode is what a line does at a sample without a value
type ChartGapMode int

const (
	// ChartGapBreak breaks the line
	ChartGapBreak ChartGapMode = iota
	// ChartGapConnect goes on to the next value
	ChartGapConnect
	// ChartGapZero goes down to zero
	ChartGapZero
)

// LineSeries draws a line through its samples, optionally with markers and
// with the area under it filled: a line chart, an area chart, a stacked
// area chart, a sparkline. Where the data is merged into buckets (a sample
// per pixel), the range of each bucket is drawn as a vertical band, so the
// peaks stay visible however far the chart is zoomed out.
//
//	s := chart.AddLine("Temperature")
//	s.AddXY(0, 21.5).AddXY(1, 22.1).AddXY(2, 21.8)
//
// The samples come from a ChartSource; AddValues, AddXY and the other
// methods of the kind write to the series' own source in memory.
type LineSeries struct {
	ChartSeriesBase
	source ChartSource
	memory *ChartMemorySource
	// watched: the memory source repaints the chart
	watched bool

	shape      ChartLineShape
	width      float64
	dash       []float64
	markers    ChartMarkerMode
	markerSize float64
	marker     MarkerShape
	gaps       ChartGapMode
	// line: the line is drawn (an area without a line has it false)
	noLine bool

	fill         bool
	fillOpacity  float64
	fillGradient bool
	fillBase     float64
	stack        string

	// The data of the last Prepare: the samples and, for a stacked line,
	// the base under each of them and the factor of a stack in percent
	// (ChartPane.computeStacks)
	samples []ChartSample
	bases   []float64
	factors []float64
}

// NewLineSeries makes a line with the name; add it to a pane with
// ChartPane.AddSeries, or make it with ChartPane.AddLine
func NewLineSeries(name string) *LineSeries {
	s := &LineSeries{width: 1.5, markerSize: 6, fillOpacity: 0.25}
	s.name = name
	s.memory = NewChartMemorySource()
	s.source = s.memory
	return s
}

// SetSource draws the series from the source instead of its own memory
func (s *LineSeries) SetSource(src ChartSource) *LineSeries {
	s.source = src
	if m, ok := src.(*ChartMemorySource); ok {
		s.memory = m
		s.watched = false
		s.watchMemory()
	} else {
		s.memory = nil
	}
	s.update()
	return s
}

// Source returns the source of the samples
func (s *LineSeries) Source() ChartSource { return s.source }

// mem returns the series' memory source, making one in place of another
// source: the simple methods write to it
func (s *LineSeries) mem() *ChartMemorySource {
	if s.memory == nil {
		s.memory = NewChartMemorySource()
		s.source = s.memory
		s.watched = false
	}
	s.watchMemory()
	return s.memory
}

// watchMemory makes a change of the memory source repaint the chart: from
// any goroutine
func (s *LineSeries) watchMemory() {
	if s.watched || s.memory == nil || s.pane == nil || s.pane.view == nil {
		return
	}
	s.watched = true
	view := s.pane.view
	s.memory.watch(func() { view.RequestRepaint() })
}

// AddValues adds the values at the next x: 0, 1, 2... after the last one
func (s *LineSeries) AddValues(values ...float64) *LineSeries {
	m := s.mem()
	x := 0.0
	if last, ok := m.Last(); ok {
		x = math.Floor(last.X) + 1
	}
	samples := make([]ChartSample, len(values))
	for i, v := range values {
		samples[i] = ChartValue(x+float64(i), v)
	}
	m.Append(samples...)
	return s
}

// SetValues replaces the data with the values at x 0, 1, 2...
func (s *LineSeries) SetValues(values []float64) *LineSeries {
	samples := make([]ChartSample, len(values))
	for i, v := range values {
		samples[i] = ChartValue(float64(i), v)
	}
	s.mem().SetSamples(samples)
	return s
}

// AddXY adds the point (x, y); points are kept sorted by x
func (s *LineSeries) AddXY(x, y float64) *LineSeries {
	s.mem().Append(ChartValue(x, y))
	return s
}

// AddPoints adds the points; they are kept sorted by x
func (s *LineSeries) AddPoints(points []ChartPoint) *LineSeries {
	samples := make([]ChartSample, len(points))
	for i, p := range points {
		samples[i] = ChartValue(p.X, p.Y)
	}
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].X < samples[j].X })
	s.mem().Append(samples...)
	return s
}

// SetPoints replaces the data with the points
func (s *LineSeries) SetPoints(points []ChartPoint) *LineSeries {
	samples := make([]ChartSample, len(points))
	for i, p := range points {
		samples[i] = ChartValue(p.X, p.Y)
	}
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].X < samples[j].X })
	s.mem().SetSamples(samples)
	return s
}

// AddSamples adds samples with their quality: ChartValue, ChartBad, ChartGap
func (s *LineSeries) AddSamples(samples ...ChartSample) *LineSeries {
	s.mem().Append(samples...)
	return s
}

// Clear removes the data of the series' own source
func (s *LineSeries) Clear() *LineSeries {
	s.mem().Clear()
	return s
}

// SetMaxPoints keeps the last n points only: for a live feed
func (s *LineSeries) SetMaxPoints(n int) *LineSeries {
	s.mem().SetMaxPoints(n)
	return s
}

// SetShape sets how the line goes between the points
func (s *LineSeries) SetShape(shape ChartLineShape) *LineSeries {
	s.shape = shape
	s.update()
	return s
}

// SetLineWidth sets the width of the line in logical pixels; 0 hides the
// line, leaving the area or the markers
func (s *LineSeries) SetLineWidth(width float64) *LineSeries {
	s.width = math.Max(0, width)
	s.noLine = width <= 0
	s.update()
	return s
}

// SetDash makes the line dashed: the lengths of a dash, a gap...
func (s *LineSeries) SetDash(pattern ...float64) *LineSeries {
	s.dash = pattern
	s.update()
	return s
}

// SetMarkers sets when markers are drawn at the points, their shape and size
func (s *LineSeries) SetMarkers(mode ChartMarkerMode, shape MarkerShape, size float64) *LineSeries {
	s.markers, s.marker = mode, shape
	if size > 0 {
		s.markerSize = size
	}
	s.update()
	return s
}

// SetGapMode sets what the line does at a sample without a value
func (s *LineSeries) SetGapMode(mode ChartGapMode) *LineSeries {
	s.gaps = mode
	s.update()
	return s
}

// SetFill fills the area between the line and the base value (0), or the
// line under it in a stack; opacity is how much of the series color it
// takes (0.25 by default), gradient fades it out towards the base
func (s *LineSeries) SetFill(fill bool, opacity float64, gradient bool) *LineSeries {
	s.fill = fill
	if opacity > 0 {
		s.fillOpacity = math.Min(opacity, 1)
	}
	s.fillGradient = gradient
	s.update()
	return s
}

// SetFillBase sets the value the area is filled down (or up) to
func (s *LineSeries) SetFillBase(base float64) *LineSeries {
	s.fillBase = base
	s.update()
	return s
}

// SetStack puts the line on the lines of the same stack group added before
// it in the pane: a stacked area chart. "" - not stacked.
func (s *LineSeries) SetStack(group string) *LineSeries {
	s.stack = group
	s.update()
	return s
}

// Samples returns the samples drawn on the last paint
func (s *LineSeries) Samples() []ChartSample { return s.samples }

// Range implements ChartSeries
func (s *LineSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if axis.Horizontal() {
		if e, ok := s.source.(ChartSourceExtent); ok {
			return e.Extent()
		}
		return 0, 0, false
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	xmin, xmax := v.XRange()
	for i, p := range s.samples {
		if !p.HasValue() || p.X < xmin || p.X > xmax {
			continue
		}
		l, h := s.stacked(i, p.Low), s.stacked(i, p.High)
		lo, hi = math.Min(lo, math.Min(l, h)), math.Max(hi, math.Max(l, h))
		if s.fill || s.bases != nil {
			fb := s.fillBase
			if s.bases != nil {
				fb = s.bases[i]
			}
			lo, hi = math.Min(lo, fb), math.Max(hi, fb)
		}
	}
	return lo, hi, lo <= hi
}

// Prepare implements ChartSeries; the stack is computed by the pane
// once all its series are prepared
func (s *LineSeries) Prepare(v *ChartViewport) {
	s.samples, s.bases, s.factors = nil, nil, nil
	if s.source == nil {
		return
	}
	xmin, xmax := v.XRange()
	s.samples = s.source.GetData(xmin, xmax, v.Group())
}

// stacked returns the value of the sample as drawn: in percent of the
// stack's total, on the lines under it
func (s *LineSeries) stacked(i int, v float64) float64 {
	if s.factors != nil {
		v *= s.factors[i]
	}
	if s.bases != nil {
		return v + s.bases[i]
	}
	return v
}

// Draw implements ChartSeries
func (s *LineSeries) Draw(cnv *Canvas, v *ChartViewport) {
	if len(s.samples) == 0 {
		return
	}
	col := s.Color(v.Theme)
	drawBadSpans(cnv, v, s.samples)

	// The runs of samples with values, each a piece of the line
	runs := s.runs()
	if s.fill {
		s.drawFill(cnv, v, runs, col)
	}
	if !s.noLine && s.width > 0 && s.dense(v) {
		s.drawColumns(cnv, v, runs, col)
	} else if !s.noLine && s.width > 0 {
		p := pathPool.Get().(*Path)
		defer pathPool.Put(p)
		p.Reset()
		for _, r := range runs {
			s.addRun(p, v, r)
		}
		style := StrokeStyle{Width: s.width, Join: LineJoinRound, Dash: s.dash}
		if len(s.dash) == 0 {
			style.Cap = LineCapRound
		}
		cnv.StrokePath(p, style, col)
	}
	if s.showMarkers(v) {
		for i, p := range s.samples {
			if !p.HasValue() {
				continue
			}
			x, y := v.PX(p.X), v.PY(s.stacked(i, p.Last))
			if x < v.X-s.markerSize || x > v.X+v.W+s.markerSize {
				continue
			}
			cnv.DrawMarker(s.marker, x, y, s.markerSize, col, v.Theme.PlotBackground)
		}
	}
}

// dense reports whether the line has a sample on every pixel or so: a thin
// straight line is then drawn as columns, quicker and as exact
func (s *LineSeries) dense(v *ChartViewport) bool {
	if s.width > 1.6 || len(s.dash) > 0 || s.shape != ChartLineStraight {
		return false
	}
	xmin, xmax := v.XRange()
	n := 0
	for _, p := range s.samples {
		if p.X >= xmin && p.X <= xmax {
			n++
		}
	}
	return float64(n) >= v.W*0.5
}

// drawColumns draws the line as a column per sample: from the value the
// line comes in at (the last of the sample before) through the range of the
// sample, as wide as the line; the columns of fractional x share their
// pixels, so the line scrolls smoothly
func (s *LineSeries) drawColumns(cnv *Canvas, v *ChartViewport, runs []lineRun, col color.RGBA) {
	w := s.width
	half := w / 2
	for _, r := range runs {
		prevY, prevX, has := 0.0, 0.0, false
		for i := r.from; i < r.to; i++ {
			first, last, high, low, ok := s.sampleValues(i)
			if !ok {
				has = false
				continue
			}
			x := v.PX(s.samples[i].X)
			yf, yl, yh, ylo := v.PY(first), v.PY(last), v.PY(high), v.PY(low)
			top := math.Min(math.Min(yf, yl), math.Min(yh, ylo))
			bottom := math.Max(math.Max(yf, yl), math.Max(yh, ylo))
			if has {
				// The step from the sample before: a column of its own when
				// the samples are more than a pixel apart, a slanted run of
				// columns then
				if x-prevX > 1.5 {
					// A column a pixel or so apart, each from where the one
					// before ended; the sample's own goes on from the last
					steps := int(math.Ceil(x - prevX))
					yAt := func(k int) float64 { return prevY + (yf-prevY)*float64(k)/float64(steps) }
					for k := 1; k < steps; k++ {
						cx := prevX + (x-prevX)*float64(k)/float64(steps)
						cy0, cy1 := yAt(k-1), yAt(k)
						cnv.fillRectF(cx-half, math.Min(cy0, cy1)-half, w, math.Abs(cy1-cy0)+w, col)
					}
					top, bottom = math.Min(top, yAt(steps-1)), math.Max(bottom, yAt(steps-1))
				} else {
					top, bottom = math.Min(top, prevY), math.Max(bottom, prevY)
				}
			}
			cnv.fillRectF(x-half, top-half, w, bottom-top+w, col)
			prevX, prevY, has = x, yl, true
		}
	}
}

// lineRun is a piece of the line between breaks: the samples from to to
type lineRun struct{ from, to int }

// runs returns the pieces of the line: a sample without a value breaks it,
// unless the gaps are connected or go down to zero
func (s *LineSeries) runs() []lineRun {
	var runs []lineRun
	start := -1
	for i, p := range s.samples {
		if p.HasValue() || s.gaps == ChartGapZero {
			if start < 0 {
				start = i
			}
			continue
		}
		if s.gaps == ChartGapConnect {
			continue
		}
		if start >= 0 {
			runs = append(runs, lineRun{start, i})
			start = -1
		}
	}
	if start >= 0 {
		runs = append(runs, lineRun{start, len(s.samples)})
	}
	return runs
}

// sampleValues returns the values of the sample as drawn: zero for a gap
// going down to zero
func (s *LineSeries) sampleValues(i int) (first, last, high, low float64, ok bool) {
	p := s.samples[i]
	if !p.HasValue() {
		if s.gaps == ChartGapZero {
			z := s.stacked(i, 0)
			return z, z, z, z, true
		}
		return 0, 0, 0, 0, false
	}
	return s.stacked(i, p.First), s.stacked(i, p.Last), s.stacked(i, p.High), s.stacked(i, p.Low), true
}

// addRun adds the piece of the line to the path, with the bands of the
// merged buckets as subpaths of their own
func (s *LineSeries) addRun(p *Path, v *ChartViewport, r lineRun) {
	// Flat lines a pixel wide stay sharp on the middle of a pixel row
	snap := func(y float64) float64 {
		if s.width <= 1.01 {
			return chartCrisp(y)
		}
		return y
	}
	type pt struct{ x, first, last float64 }
	pts := make([]pt, 0, r.to-r.from)
	for i := r.from; i < r.to; i++ {
		first, last, _, _, ok := s.sampleValues(i)
		if !ok {
			continue
		}
		pts = append(pts, pt{v.PX(s.samples[i].X), snap(v.PY(first)), snap(v.PY(last))})
	}
	if len(pts) == 0 {
		return
	}
	smooth := s.shape == ChartLineSpline && len(pts) > 2
	if smooth {
		// A spline through buckets would invent a curve inside each: only
		// through single values
		for _, q := range pts {
			if q.first != q.last {
				smooth = false
				break
			}
		}
	}
	if smooth {
		xs := make([]float64, len(pts))
		ys := make([]float64, len(pts))
		for i, q := range pts {
			xs[i], ys[i] = q.x, q.last
		}
		addMonotoneSpline(p, xs, ys)
	} else {
		p.MoveTo(pts[0].x, pts[0].first)
		if pts[0].last != pts[0].first {
			p.LineTo(pts[0].x, pts[0].last)
		}
		for i := 1; i < len(pts); i++ {
			prev, q := pts[i-1], pts[i]
			switch s.shape {
			case ChartLineStepAfter:
				p.LineTo(q.x, prev.last)
			case ChartLineStepBefore:
				p.LineTo(prev.x, q.first)
			case ChartLineStepMiddle:
				mid := (prev.x + q.x) / 2
				p.LineTo(mid, prev.last)
				p.LineTo(mid, q.first)
			}
			p.LineTo(q.x, q.first)
			if q.last != q.first {
				p.LineTo(q.x, q.last)
			}
		}
	}
	// The range of each merged bucket
	for i := r.from; i < r.to; i++ {
		first, last, high, low, ok := s.sampleValues(i)
		if !ok || high == low {
			continue
		}
		top, bottom := math.Max(first, last), math.Min(first, last)
		if high <= top && low >= bottom {
			continue
		}
		x := v.PX(s.samples[i].X)
		p.MoveTo(x, v.PY(high))
		p.LineTo(x, v.PY(low))
	}
}

// drawFill fills the area between each piece of the line and its base
func (s *LineSeries) drawFill(cnv *Canvas, v *ChartViewport, runs []lineRun, col color.RGBA) {
	fillCol := col
	fillCol.A = uint8(float64(col.A) * s.fillOpacity)
	baseY := func(i int) float64 {
		if s.bases != nil {
			return v.PY(s.bases[i])
		}
		y := v.PY(s.fillBase)
		return math.Max(v.Y, math.Min(v.Y+v.H, y))
	}
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	for _, r := range runs {
		var idx []int
		for i := r.from; i < r.to; i++ {
			if _, _, _, _, ok := s.sampleValues(i); ok {
				idx = append(idx, i)
			}
		}
		if len(idx) == 0 {
			continue
		}
		// Along the line, then back along the base
		for k, i := range idx {
			_, last, _, _, _ := s.sampleValues(i)
			x, y := v.PX(s.samples[i].X), v.PY(last)
			if k == 0 {
				p.MoveTo(x, y)
			} else {
				if s.shape == ChartLineStepAfter {
					_, prevLast, _, _, _ := s.sampleValues(idx[k-1])
					p.LineTo(x, v.PY(prevLast))
				}
				p.LineTo(x, y)
			}
		}
		for k := len(idx) - 1; k >= 0; k-- {
			i := idx[k]
			x := v.PX(s.samples[i].X)
			if s.bases != nil && s.shape == ChartLineStepAfter && k < len(idx)-1 {
				p.LineTo(v.PX(s.samples[idx[k+1]].X), baseY(i))
			}
			p.LineTo(x, baseY(i))
		}
		p.Close()
	}
	if s.fillGradient {
		// From the top of the plot, fading out towards its bottom
		fade := fillCol
		fade.A = 0
		cnv.FillPathPaint(p, LinearGradient{X1: 0, Y1: v.Y, X2: 0, Y2: v.Y + v.H, Stops: []GradientStop{{0, fillCol}, {1, fade}}})
		return
	}
	cnv.FillPath(p, fillCol)
}

// showMarkers reports whether the markers are drawn at the points
func (s *LineSeries) showMarkers(v *ChartViewport) bool {
	switch s.markers {
	case ChartMarkersAlways:
		return true
	case ChartMarkersNever:
		return false
	}
	n := 0
	xmin, xmax := v.XRange()
	for _, p := range s.samples {
		if p.HasValue() && p.X >= xmin && p.X <= xmax {
			n++
		}
	}
	return n > 0 && n <= 200 && v.W/float64(n) >= 8
}

// Nearest implements ChartSeries
func (s *LineSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	if len(s.samples) == 0 {
		return ChartHit{}, false
	}
	x := v.DataX(px)
	i := nearestSampleIndex(s.samples, x)
	// The nearest sample with a value, looking both ways
	best := -1
	bestD := math.Inf(1)
	for d := 0; d < len(s.samples); d++ {
		found := false
		for _, j := range [2]int{i - d, i + d} {
			if j < 0 || j >= len(s.samples) || !s.samples[j].HasValue() {
				continue
			}
			dx := math.Abs(v.PX(s.samples[j].X) - px)
			if dx < bestD {
				best, bestD = j, dx
			}
			found = true
		}
		if found && d > 0 {
			break
		}
		if d > 0 && v.PX(s.samples[max(i-d, 0)].X) < px-40 && v.PX(s.samples[min(i+d, len(s.samples)-1)].X) > px+40 {
			break
		}
	}
	if best < 0 {
		return ChartHit{}, false
	}
	p := s.samples[best]
	hx, hy := v.PX(p.X), v.PY(s.stacked(best, p.Last))
	dist := math.Abs(hx - px)
	if !byX {
		dist = math.Hypot(hx-px, hy-py)
	}
	yl := v.YAxis.scale.Format(p.Last)
	if p.High != p.Low {
		yl = v.YAxis.scale.Format(p.Low) + " … " + v.YAxis.scale.Format(p.High)
	}
	return ChartHit{
		Series: s, Index: best, X: p.X, Y: p.Last, PX: hx, PY: hy,
		XLabel: v.XAxis.scale.Format(p.X), YLabel: yl, Name: s.name,
		Color: s.Color(v.Theme), Sample: p, dist: dist,
	}, true
}

// Legend implements ChartSeries
func (s *LineSeries) Legend(th *ChartTheme) []ChartLegendItem {
	kind := ChartLegendLine
	if s.fill && s.noLine {
		kind = ChartLegendBox
	}
	items := s.legendItem(th, kind)
	if s.markers == ChartMarkersAlways {
		items[0].Kind = ChartLegendMarker
		items[0].Marker = s.marker
	}
	return items
}

// drawBadSpans hatches, over the height of the plot, every span of samples
// with HasBad: from its first bad sample to the next good one; a span at
// the end of the data is as wide as the step between the last samples
func drawBadSpans(cnv *Canvas, v *ChartViewport, samples []ChartSample) {
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	for i := 0; i < len(samples); {
		if !samples[i].HasBad {
			i++
			continue
		}
		j := i
		for j < len(samples) && samples[j].HasBad {
			j++
		}
		x1 := v.PX(samples[i].X)
		var x2 float64
		switch {
		case j < len(samples):
			x2 = v.PX(samples[j].X)
		case j >= 2:
			x2 = 2*v.PX(samples[j-1].X) - v.PX(samples[j-2].X)
		default:
			x2 = x1 + 1
		}
		x1 = math.Max(math.Floor(x1), v.X)
		x2 = math.Min(math.Max(math.Ceil(x2), x1+1), v.X+v.W)
		if x2 > x1 {
			p.Rect(x1, v.Y, x2-x1, v.H)
		}
		i = j
	}
	if !p.Empty() {
		cnv.FillPathPaint(p, HatchPaint{Color: v.Theme.BadHatch, Angle: math.Pi / 4, Spacing: 6, Width: 1})
	}
}

// addMonotoneSpline adds a smooth curve through the points to the path that
// does not overshoot them (Fritsch–Carlson): between two equal values it is
// flat, at a peak it turns
func addMonotoneSpline(p *Path, xs, ys []float64) {
	n := len(xs)
	d := make([]float64, n-1) // the slopes of the segments
	for i := 0; i < n-1; i++ {
		dx := xs[i+1] - xs[i]
		if dx == 0 {
			d[i] = 0
		} else {
			d[i] = (ys[i+1] - ys[i]) / dx
		}
	}
	m := make([]float64, n) // the tangents at the points
	m[0], m[n-1] = d[0], d[n-2]
	for i := 1; i < n-1; i++ {
		if d[i-1]*d[i] <= 0 {
			m[i] = 0
		} else {
			m[i] = (d[i-1] + d[i]) / 2
		}
	}
	for i := 0; i < n-1; i++ {
		if d[i] == 0 {
			m[i], m[i+1] = 0, 0
			continue
		}
		a, b := m[i]/d[i], m[i+1]/d[i]
		if s := a*a + b*b; s > 9 {
			t := 3 / math.Sqrt(s)
			m[i], m[i+1] = t*a*d[i], t*b*d[i]
		}
	}
	p.MoveTo(xs[0], ys[0])
	for i := 0; i < n-1; i++ {
		h := (xs[i+1] - xs[i]) / 3
		p.CubicTo(xs[i]+h, ys[i]+m[i]*h, xs[i+1]-h, ys[i+1]-m[i+1]*h, xs[i+1], ys[i+1])
	}
}
