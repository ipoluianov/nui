package ui

import (
	"math"
	"strconv"
)

// polarFrame is the circle the radar and polar series of a pane share: its
// middle, its radius, the value at the rim
type polarFrame struct {
	cx, cy, r float64
	max       float64
	rings     []float64
}

// polarSeries is a series drawn on the polar frame of its pane
type polarSeries interface {
	ChartSeries
	polarMax() float64
}

// polarFrameOf returns the frame of the pane's plot: the radius fits the
// biggest value of its polar series, rounded up
func (p *ChartPane) polarFrameOf(v *ChartViewport, cnv *Canvas) polarFrame {
	top := 0.0
	for _, s := range p.series {
		if ps, ok := s.(polarSeries); ok && s.Visible() {
			top = math.Max(top, ps.polarMax())
		}
	}
	if top <= 0 {
		top = 1
	}
	step := niceStepAtLeast(top / 4)
	rimMax := math.Ceil(top/step-1e-9) * step
	var rings []float64
	for r := step; r <= rimMax+1e-9; r += step {
		rings = append(rings, r)
	}
	lh := v.FontSize * 1.2
	labelW := chartTextWidth(cnv, "330°")
	for _, name := range p.radarCats {
		labelW = math.Max(labelW, chartTextWidth(cnv, name))
	}
	labelW = math.Min(math.Max(labelW, 24), v.W/4)
	r := math.Min(v.W/2-labelW-6, v.H/2-lh-4)
	return polarFrame{cx: v.X + v.W/2, cy: v.Y + v.H/2, r: math.Max(r, 10), max: rimMax, rings: rings}
}

// isFirstPolar reports whether the series is the first visible polar series
// of its pane: it draws the frame for all of them
func (p *ChartPane) isFirstPolar(s ChartSeries) bool {
	for _, o := range p.series {
		if _, ok := o.(polarSeries); ok && o.Visible() {
			return o == s
		}
	}
	return false
}

// drawPolarFrame draws the rings with their values and the spokes: of the
// radar's categories, or every 30 degrees
func (p *ChartPane) drawPolarFrame(cnv *Canvas, v *ChartViewport, f polarFrame, polygon bool) {
	th := v.Theme
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	n := len(p.radarCats)
	spokes := n
	if !polygon || n < 3 {
		spokes = 12
	}
	angle := func(i int) float64 { return -math.Pi/2 + 2*math.Pi*float64(i)/float64(spokes) }
	path.Reset()
	for _, ring := range f.rings {
		rr := f.r * ring / f.max
		if polygon && n >= 3 {
			for i := 0; i < spokes; i++ {
				x, y := f.cx+rr*math.Cos(angle(i)), f.cy+rr*math.Sin(angle(i))
				if i == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			path.Close()
		} else {
			path.Circle(f.cx, f.cy, rr)
		}
	}
	for i := 0; i < spokes; i++ {
		path.MoveTo(f.cx, f.cy)
		path.LineTo(f.cx+f.r*math.Cos(angle(i)), f.cy+f.r*math.Sin(angle(i)))
	}
	cnv.StrokePath(path, StrokeStyle{Width: 1}, th.Grid)
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.8)
	for _, ring := range f.rings {
		chartDrawLabel(cnv, trimChartZeros(strconv.FormatFloat(ring, 'f', 3, 64)), f.cx+3, f.cy-f.r*ring/f.max, HAlignLeft, VAlignBottom, th.Text)
	}
	cnv.SetFontSize(v.FontSize * 0.85)
	for i := 0; i < spokes; i++ {
		var text string
		if polygon && n >= 3 {
			text = p.radarCats[i]
		} else {
			text = strconv.Itoa(i*360/spokes) + "°"
		}
		a := angle(i)
		x, y := f.cx+(f.r+6)*math.Cos(a), f.cy+(f.r+6)*math.Sin(a)
		h := HAlignCenter
		if math.Cos(a) > 0.2 {
			h = HAlignLeft
		} else if math.Cos(a) < -0.2 {
			h = HAlignRight
		}
		va := VAlignCenter
		if math.Sin(a) < -0.7 {
			va = VAlignBottom
		} else if math.Sin(a) > 0.7 {
			va = VAlignTop
		}
		chartDrawLabel(cnv, text, x, y, h, va, th.Text)
	}
}

// RadarSeries draws values of the same categories as a polygon around a
// center, a spoke per category: the profile of a product, the skills of a
// team, the scores of the options compared. The series of a pane share
// the categories and the scale.
//
//	r := chart.Pane(0).AddRadar("Model A")
//	r.Set("Speed", 8).Set("Comfort", 6).Set("Price", 4)
type RadarSeries struct {
	ChartSeriesBase
	values map[string]float64
	fill   bool

	points []PointF
}

// AddRadar adds a radar of the name to the pane
func (p *ChartPane) AddRadar(name string) *RadarSeries {
	s := &RadarSeries{values: map[string]float64{}, fill: true}
	s.name = name
	p.AddSeries(s)
	return s
}

// Set sets the value of the category, adding the category as a spoke
func (s *RadarSeries) Set(category string, value float64) *RadarSeries {
	s.values[category] = value
	if s.pane != nil {
		found := false
		for _, c := range s.pane.radarCats {
			if c == category {
				found = true
			}
		}
		if !found {
			s.pane.radarCats = append(s.pane.radarCats, category)
		}
	}
	s.update()
	return s
}

// SetFill fills the polygon translucent (on by default)
func (s *RadarSeries) SetFill(fill bool) *RadarSeries {
	s.fill = fill
	s.update()
	return s
}

func (s *RadarSeries) chartNoAxes() {}

func (s *RadarSeries) polarMax() float64 {
	m := 0.0
	for _, v := range s.values {
		m = math.Max(m, v)
	}
	return m
}

// Range implements ChartSeries
func (s *RadarSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	return 0, 0, false
}

// Prepare implements ChartSeries
func (s *RadarSeries) Prepare(v *ChartViewport) {}

// Draw implements ChartSeries
func (s *RadarSeries) Draw(cnv *Canvas, v *ChartViewport) {
	f := s.pane.polarFrameOf(v, cnv)
	if s.pane.isFirstPolar(s) {
		s.pane.drawPolarFrame(cnv, v, f, true)
	}
	n := len(s.pane.radarCats)
	if n < 3 {
		return
	}
	col := s.Color(v.Theme)
	s.points = s.points[:0]
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	for i, c := range s.pane.radarCats {
		val := math.Max(0, s.values[c])
		a := -math.Pi/2 + 2*math.Pi*float64(i)/float64(n)
		rr := f.r * val / f.max
		pt := PointF{f.cx + rr*math.Cos(a), f.cy + rr*math.Sin(a)}
		s.points = append(s.points, pt)
		if i == 0 {
			p.MoveTo(pt.X, pt.Y)
		} else {
			p.LineTo(pt.X, pt.Y)
		}
	}
	p.Close()
	if s.fill {
		fill := col
		fill.A = 50
		cnv.FillPath(p, fill)
	}
	cnv.StrokePath(p, StrokeStyle{Width: 2, Join: LineJoinRound}, col)
	for _, pt := range s.points {
		cnv.DrawMarker(MarkerCircle, pt.X, pt.Y, 6, col, v.Theme.PlotBackground)
	}
}

// Nearest implements ChartSeries
func (s *RadarSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	best, bestD := -1, math.Inf(1)
	for i, pt := range s.points {
		if d := math.Hypot(pt.X-px, pt.Y-py); d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return ChartHit{}, false
	}
	c := s.pane.radarCats[best]
	return ChartHit{Series: s, Index: best, Y: s.values[c], PX: s.points[best].X, PY: s.points[best].Y, dist: bestD,
		XLabel: c, YLabel: trimChartZeros(strconv.FormatFloat(s.values[c], 'f', 3, 64)), Name: s.name, Color: s.Color(v.Theme)}, true
}

// Legend implements ChartSeries
func (s *RadarSeries) Legend(th *ChartTheme) []ChartLegendItem {
	return s.legendItem(th, ChartLegendLine)
}

// ChartPolarKind is how a PolarSeries draws its points
type ChartPolarKind int

const (
	// ChartPolarLine joins the points by a line around the center
	ChartPolarLine ChartPolarKind = iota
	// ChartPolarArea fills the shape of the line
	ChartPolarArea
	// ChartPolarBars draws a sector per point: a wind rose
	ChartPolarBars
)

// PolarSeries draws values by an angle in degrees (0 at the top, growing
// clockwise, as a compass): the directivity of an antenna, a cyclic process,
// as bars the wind rose of a station. The series of a pane share the scale.
//
//	rose := chart.Pane(0).AddPolar("Wind").SetKind(ui.ChartPolarBars)
//	rose.Add(0, 12).Add(45, 7).Add(90, 3)
type PolarSeries struct {
	ChartSeriesBase
	points []ChartPoint // X the angle, Y the value
	kind   ChartPolarKind
	stack  string

	drawn []PointF
}

// AddPolar adds a polar series of the name to the pane
func (p *ChartPane) AddPolar(name string) *PolarSeries {
	s := &PolarSeries{}
	s.name = name
	p.AddSeries(s)
	return s
}

// Add adds the value at the angle in degrees
func (s *PolarSeries) Add(angle, value float64) *PolarSeries {
	s.points = append(s.points, ChartPoint{X: angle, Y: value})
	s.update()
	return s
}

// SetKind sets how the points are drawn
func (s *PolarSeries) SetKind(kind ChartPolarKind) *PolarSeries {
	s.kind = kind
	s.update()
	return s
}

// SetStack stands the bars on those of the series of the group added
// before: the speeds of a wind rose
func (s *PolarSeries) SetStack(group string) *PolarSeries {
	s.stack = group
	s.update()
	return s
}

func (s *PolarSeries) chartNoAxes() {}

// base returns the value the bar at the angle stands on
func (s *PolarSeries) stackBase(angle float64) float64 {
	if s.stack == "" || s.pane == nil {
		return 0
	}
	base := 0.0
	for _, o := range s.pane.series {
		if o == ChartSeries(s) {
			break
		}
		if ps, ok := o.(*PolarSeries); ok && ps.stack == s.stack && ps.Visible() {
			for _, p := range ps.points {
				if p.X == angle {
					base += p.Y
				}
			}
		}
	}
	return base
}

func (s *PolarSeries) polarMax() float64 {
	m := 0.0
	for _, p := range s.points {
		m = math.Max(m, p.Y+s.stackBase(p.X))
	}
	return m
}

// Range implements ChartSeries
func (s *PolarSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	return 0, 0, false
}

// Prepare implements ChartSeries
func (s *PolarSeries) Prepare(v *ChartViewport) {}

// Draw implements ChartSeries
func (s *PolarSeries) Draw(cnv *Canvas, v *ChartViewport) {
	f := s.pane.polarFrameOf(v, cnv)
	if s.pane.isFirstPolar(s) {
		s.pane.drawPolarFrame(cnv, v, f, false)
	}
	col := s.Color(v.Theme)
	at := func(angle, val float64) PointF {
		a := angle*math.Pi/180 - math.Pi/2
		rr := f.r * math.Max(0, val) / f.max
		return PointF{f.cx + rr*math.Cos(a), f.cy + rr*math.Sin(a)}
	}
	s.drawn = s.drawn[:0]
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	if s.kind == ChartPolarBars {
		// The width of a sector: the step between the angles
		step := 360.0
		for i := 1; i < len(s.points); i++ {
			if d := math.Abs(s.points[i].X - s.points[i-1].X); d > 0 && d < step {
				step = d
			}
		}
		if len(s.points) > 1 && step == 360 {
			step = 360 / float64(len(s.points))
		}
		w := step * 0.8 * math.Pi / 180
		for _, pt := range s.points {
			base := s.stackBase(pt.X)
			a := pt.X*math.Pi/180 - math.Pi/2
			p.Reset()
			p.Sector(f.cx, f.cy, f.r*base/f.max, f.r*(base+pt.Y)/f.max, a-w/2, a+w/2)
			cnv.FillPath(p, col)
			s.drawn = append(s.drawn, at(pt.X, base+pt.Y))
		}
		return
	}
	p.Reset()
	for i, pt := range s.points {
		q := at(pt.X, pt.Y)
		s.drawn = append(s.drawn, q)
		if i == 0 {
			p.MoveTo(q.X, q.Y)
		} else {
			p.LineTo(q.X, q.Y)
		}
	}
	// A full turn of points closes the line
	if len(s.points) > 2 {
		p.Close()
	}
	if s.kind == ChartPolarArea {
		fill := col
		fill.A = 60
		cnv.FillPath(p, fill)
	}
	cnv.StrokePath(p, StrokeStyle{Width: 1.75, Join: LineJoinRound}, col)
}

// Nearest implements ChartSeries
func (s *PolarSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	best, bestD := -1, math.Inf(1)
	for i, q := range s.drawn {
		if d := math.Hypot(q.X-px, q.Y-py); d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return ChartHit{}, false
	}
	pt := s.points[best]
	return ChartHit{Series: s, Index: best, X: pt.X, Y: pt.Y, PX: s.drawn[best].X, PY: s.drawn[best].Y, dist: bestD,
		XLabel: trimChartZeros(strconv.FormatFloat(pt.X, 'f', 1, 64)) + "°", YLabel: trimChartZeros(strconv.FormatFloat(pt.Y, 'f', 3, 64)),
		Name: s.name, Color: s.Color(v.Theme)}, true
}

// Legend implements ChartSeries
func (s *PolarSeries) Legend(th *ChartTheme) []ChartLegendItem {
	if s.kind == ChartPolarLine {
		return s.legendItem(th, ChartLegendLine)
	}
	return s.legendItem(th, ChartLegendBox)
}
