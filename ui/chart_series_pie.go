package ui

import (
	"image/color"
	"math"
	"strconv"
)

// PieSeries draws the parts of a whole as the slices of a circle: a pie, or
// a donut with a hole (SetInnerRadius), or a half circle. A pane with only
// pies has no axes. Each slice has its entry in the legend; a click on it
// hides the slice.
//
//	pie := chart.AddPie("OS")
//	pie.Add("Linux", 60).Add("Windows", 30).Add("macOS", 10)
type PieSeries struct {
	ChartSeriesBase
	slices []pieSlice

	inner      float64 // the hole, a part of the radius
	start      float64 // the angle of the first slice, radians
	sweep      float64 // the angle of the whole, 2π for a full circle
	labels     ChartPieLabels
	otherBelow float64 // slices below this part go to "Other"
	otherName  string
	hover      int // the slice under the pointer, pulled out; -1 - none
	// The colors fixed by SetSliceColor, by the names of the slices
	colors map[string]color.RGBA

	// The geometry and the slices of the last paint
	cx, cy, r float64
	drawn     []pieDrawn
}

type pieSlice struct {
	name   string
	value  float64
	hidden bool
}

type pieDrawn struct {
	name       string
	value      float64
	part       float64
	a0, a1     float64
	col        color.RGBA
	index      int // in slices; -1 - the "Other" slice
	labelPoint PointF
}

// ChartPieLabels is what is written on the slices
type ChartPieLabels int

const (
	ChartPieLabelPercent ChartPieLabels = iota
	ChartPieLabelValue
	ChartPieLabelName
	ChartPieLabelNamePercent
	ChartPieLabelNone
)

// NewPieSeries makes a pie of the name
func NewPieSeries(name string) *PieSeries {
	s := &PieSeries{start: -math.Pi / 2, sweep: 2 * math.Pi, otherName: "Other", hover: -1}
	s.name = name
	return s
}

// Add adds a slice of the value; a slice of a name it has already is
// given the value
func (s *PieSeries) Add(name string, value float64) *PieSeries {
	for i := range s.slices {
		if s.slices[i].name == name {
			s.slices[i].value = value
			s.update()
			return s
		}
	}
	s.slices = append(s.slices, pieSlice{name: name, value: value})
	s.update()
	return s
}

// Set is Add
func (s *PieSeries) Set(name string, value float64) *PieSeries { return s.Add(name, value) }

// Clear removes all the slices
func (s *PieSeries) Clear() *PieSeries {
	s.slices = nil
	s.update()
	return s
}

// SetSliceColor fixes the color of the slice of the name, added before or
// after
func (s *PieSeries) SetSliceColor(name string, col color.RGBA) *PieSeries {
	if s.colors == nil {
		s.colors = map[string]color.RGBA{}
	}
	s.colors[name] = col
	s.update()
	return s
}

// SetInnerRadius makes a donut: the hole is the part of the radius (0..0.95)
func (s *PieSeries) SetInnerRadius(part float64) *PieSeries {
	s.inner = math.Max(0, math.Min(part, 0.95))
	s.update()
	return s
}

// SetAngles sets where the first slice starts (radians, 0 - to the right,
// growing clockwise; -π/2, the top, by default) and the angle of the whole:
// π for a half circle
func (s *PieSeries) SetAngles(start, sweep float64) *PieSeries {
	s.start = start
	if sweep > 0 {
		s.sweep = math.Min(sweep, 2*math.Pi)
	}
	s.update()
	return s
}

// SetLabels sets what is written on the slices
func (s *PieSeries) SetLabels(labels ChartPieLabels) *PieSeries {
	s.labels = labels
	s.update()
	return s
}

// SetOther puts the slices smaller than the part of the whole (0.03 - 3%)
// together into one slice of the name; 0 - never
func (s *PieSeries) SetOther(below float64, name string) *PieSeries {
	s.otherBelow = math.Max(0, below)
	if name != "" {
		s.otherName = name
	}
	s.update()
	return s
}

// chartNoAxes marks a series drawn without axes: a pane of only such series
// has none
func (s *PieSeries) chartNoAxes() {}

// Range implements ChartSeries: a pie has no range along the axes
func (s *PieSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	return 0, 0, false
}

// Prepare implements ChartSeries
func (s *PieSeries) Prepare(v *ChartViewport) {}

// layout returns the slices to draw with their angles
func (s *PieSeries) layout(th *ChartTheme) []pieDrawn {
	total := 0.0
	for _, sl := range s.slices {
		if !sl.hidden && sl.value > 0 && chartFinite(sl.value) {
			total += sl.value
		}
	}
	if total <= 0 {
		return nil
	}
	var res []pieDrawn
	other := 0.0
	for i, sl := range s.slices {
		if sl.hidden || !(sl.value > 0) || !chartFinite(sl.value) {
			continue
		}
		part := sl.value / total
		if part < s.otherBelow {
			other += sl.value
			continue
		}
		col := s.sliceColor(i, th)
		res = append(res, pieDrawn{name: sl.name, value: sl.value, part: part, col: col, index: i})
	}
	if other > 0 {
		res = append(res, pieDrawn{name: s.otherName, value: other, part: other / total, col: th.SeriesColor(len(th.Palette) - 1), index: -1})
	}
	a := s.start
	for i := range res {
		res[i].a0 = a
		a += res[i].part * s.sweep
		res[i].a1 = a
	}
	return res
}

func (s *PieSeries) sliceColor(i int, th *ChartTheme) color.RGBA {
	if c, ok := s.colors[s.slices[i].name]; ok {
		return c
	}
	return th.SeriesColor(i)
}

// Draw implements ChartSeries
func (s *PieSeries) Draw(cnv *Canvas, v *ChartViewport) {
	s.drawn = s.layout(v.Theme)
	// The circle in the middle of the plot; a half circle stands on its
	// diameter, so it may be twice as wide
	pad := 6.0
	if s.labels != ChartPieLabelNone {
		pad = 8
	}
	r := math.Min(v.W, v.H)/2 - pad
	cy := v.Y + v.H/2
	if s.sweep <= math.Pi+1e-9 {
		r = math.Min(v.W/2, v.H) - pad
		cy = v.Y + v.H - pad - (v.H-2*pad-r)/2
	}
	s.cx, s.cy, s.r = v.X+v.W/2, cy, math.Max(r, 4)
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	for i, d := range s.drawn {
		cx, cy := s.cx, s.cy
		if i == s.hover {
			// Pulled out along its middle
			mid := (d.a0 + d.a1) / 2
			cx, cy = cx+4*math.Cos(mid), cy+4*math.Sin(mid)
		}
		p.Reset()
		p.Sector(cx, cy, s.r*s.inner, s.r, d.a0, d.a1)
		cnv.FillPath(p, d.col)
		// A thin line of the background between the slices
		if len(s.drawn) > 1 {
			cnv.StrokePath(p, StrokeStyle{Width: 1, Join: LineJoinBevel}, v.Theme.PlotBackground)
		}
	}
	s.drawLabels(cnv, v)
}

// drawLabels writes the labels in the middle of the slices big enough for them
func (s *PieSeries) drawLabels(cnv *Canvas, v *ChartViewport) {
	if s.labels == ChartPieLabelNone {
		return
	}
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.85)
	cnv.SetHAlign(HAlignCenter)
	cnv.SetVAlign(VAlignCenter)
	_, lineH := cnv.MeasureTextRotated("0", 0)
	rr := s.r * (1 + s.inner) / 2
	if s.inner == 0 {
		rr = s.r * 0.62
	}
	for i, d := range s.drawn {
		var text string
		pct := strconv.FormatFloat(d.part*100, 'f', 0, 64) + "%"
		switch s.labels {
		case ChartPieLabelPercent:
			text = pct
		case ChartPieLabelValue:
			text = FormatChartNumber(d.value, logLabelStep(math.Abs(d.value))/10, false)
		case ChartPieLabelName:
			text = d.name
		case ChartPieLabelNamePercent:
			text = d.name + " " + pct
		}
		mid := (d.a0 + d.a1) / 2
		x, y := s.cx+rr*math.Cos(mid), s.cy+rr*math.Sin(mid)
		s.drawn[i].labelPoint = PointF{x, y}
		// Only where it fits within the slice: its arc at the label's
		// radius is long enough
		arc := (d.a1 - d.a0) * rr
		if arc < math.Max(chartTextWidth(cnv, text)*0.6, lineH*1.2) {
			continue
		}
		cnv.SetColor(chartContrastText(d.col))
		cnv.DrawTextRotated(x, y, 0, text)
	}
}

// chartContrastText returns black or white, whichever reads better on the color
func chartContrastText(bg color.RGBA) color.RGBA {
	l := 0.299*float64(bg.R) + 0.587*float64(bg.G) + 0.114*float64(bg.B)
	if l > 150 {
		return color.RGBA{20, 20, 20, 255}
	}
	return color.RGBA{255, 255, 255, 255}
}

// sliceAt returns the drawn slice at the point, -1 if none
func (s *PieSeries) sliceAt(px, py float64) int {
	dx, dy := px-s.cx, py-s.cy
	d := math.Hypot(dx, dy)
	if d > s.r+4 || d < s.r*s.inner {
		return -1
	}
	a := math.Atan2(dy, dx)
	for i, sl := range s.drawn {
		// The angle brought into the slice's turn
		x := a
		for x < sl.a0 {
			x += 2 * math.Pi
		}
		for x >= sl.a0+2*math.Pi {
			x -= 2 * math.Pi
		}
		if x < sl.a1 {
			return i
		}
	}
	return -1
}

// Nearest implements ChartSeries: the slice under the pointer
func (s *PieSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	i := s.sliceAt(px, py)
	if i < 0 {
		return ChartHit{}, false
	}
	d := s.drawn[i]
	mid := (d.a0 + d.a1) / 2
	return ChartHit{
		Series: s, Index: d.index, Y: d.value, Name: d.name, Color: d.col,
		XLabel: d.name, YLabel: FormatChartNumber(d.value, logLabelStep(math.Abs(d.value))/10, false) +
			" (" + strconv.FormatFloat(d.part*100, 'f', 1, 64) + "%)",
		PX: s.cx + s.r*0.7*math.Cos(mid), PY: s.cy + s.r*0.7*math.Sin(mid),
	}, true
}

// setHover pulls the slice of the index (in the drawn ones) out; -1 - none
func (s *PieSeries) setHover(i int) bool {
	if s.hover == i {
		return false
	}
	s.hover = i
	return true
}

// Legend implements ChartSeries: an entry per slice
func (s *PieSeries) Legend(th *ChartTheme) []ChartLegendItem {
	items := make([]ChartLegendItem, 0, len(s.slices))
	for i, sl := range s.slices {
		items = append(items, ChartLegendItem{Name: sl.name, Color: s.sliceColor(i, th), Kind: ChartLegendBox, Hidden: sl.hidden, Index: i})
	}
	return items
}

// toggleLegendItem hides or shows the slice
func (s *PieSeries) toggleLegendItem(index int) {
	if index >= 0 && index < len(s.slices) {
		s.slices[index].hidden = !s.slices[index].hidden
		s.update()
	}
}
