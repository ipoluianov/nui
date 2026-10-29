package ui

import (
	"math"
	"strconv"
)

// WaterfallSeries draws how a total is made of the changes to it: each
// step a bar from the running total before it to the one after it, up in
// the up color, down in the down one; a total a bar from zero. For the
// bridges of finance: from the plan to the fact, from the revenue to the
// profit.
//
//	w := chart.Pane(0).AddWaterfall("Profit")
//	w.AddStep("Revenue", 420).AddStep("Costs", -260).AddStep("Taxes", -40).AddTotal("Profit")
type WaterfallSeries struct {
	ChartSeriesBase
	steps  []waterfallStep
	labels bool

	rects []barRect
}

type waterfallStep struct {
	name  string
	value float64
	total bool
}

// NewWaterfallSeries makes a waterfall of the name
func NewWaterfallSeries(name string) *WaterfallSeries {
	s := &WaterfallSeries{labels: true}
	s.name = name
	return s
}

// AddWaterfall adds a waterfall to the pane; its x axis becomes a
// CategoryScale of the steps
func (p *ChartPane) AddWaterfall(name string) *WaterfallSeries {
	if _, ok := p.xAxis.scale.(*CategoryScale); !ok {
		p.xAxis.SetScale(NewCategoryScale())
		p.xAxis.SetGrid(false, false)
	}
	s := NewWaterfallSeries(name)
	p.AddSeries(s)
	return s
}

// AddStep adds a change of the total
func (s *WaterfallSeries) AddStep(name string, delta float64) *WaterfallSeries {
	s.steps = append(s.steps, waterfallStep{name: name, value: delta})
	s.syncCategories()
	return s
}

// AddTotal adds a bar of the running total, from zero
func (s *WaterfallSeries) AddTotal(name string) *WaterfallSeries {
	s.steps = append(s.steps, waterfallStep{name: name, total: true})
	s.syncCategories()
	return s
}

// SetLabels writes the change (or the total) on each bar
func (s *WaterfallSeries) SetLabels(show bool) *WaterfallSeries {
	s.labels = show
	s.update()
	return s
}

func (s *WaterfallSeries) syncCategories() {
	if s.pane != nil {
		x, _ := s.pane.seriesAxes(s)
		if cs, ok := x.scale.(*CategoryScale); ok {
			names := make([]string, len(s.steps))
			for i, st := range s.steps {
				names[i] = st.name
			}
			cs.SetCategories(names)
		}
	}
	s.update()
}

// bars returns from where to where each bar goes
func (s *WaterfallSeries) bars() [][2]float64 {
	res := make([][2]float64, len(s.steps))
	sum := 0.0
	for i, st := range s.steps {
		if st.total {
			res[i] = [2]float64{0, sum}
			continue
		}
		res[i] = [2]float64{sum, sum + st.value}
		sum += st.value
	}
	return res
}

// Range implements ChartSeries
func (s *WaterfallSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if len(s.steps) == 0 {
		return 0, 0, false
	}
	if axis.Horizontal() {
		return -0.5, float64(len(s.steps)) - 0.5, true
	}
	lo, hi := 0.0, 0.0
	for _, b := range s.bars() {
		lo, hi = math.Min(lo, math.Min(b[0], b[1])), math.Max(hi, math.Max(b[0], b[1]))
	}
	return lo, hi, true
}

// Prepare implements ChartSeries
func (s *WaterfallSeries) Prepare(v *ChartViewport) {}

// Draw implements ChartSeries
func (s *WaterfallSeries) Draw(cnv *Canvas, v *ChartViewport) {
	s.rects = s.rects[:0]
	bars := s.bars()
	band := math.Abs(v.PX(1) - v.PX(0))
	w := band * 0.65
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	total := s.Color(v.Theme)
	for i, b := range bars {
		col := total
		if !s.steps[i].total {
			col = v.Theme.Up
			if b[1] < b[0] {
				col = v.Theme.Down
			}
		}
		x := math.Round(v.PX(float64(i)) - w/2)
		y1, y2 := math.Round(v.PY(b[0])), math.Round(v.PY(b[1]))
		top, h := math.Min(y1, y2), math.Max(math.Abs(y2-y1), 1)
		p.Reset()
		p.Rect(x, top, math.Round(w), h)
		cnv.FillPath(p, col)
		value := s.steps[i].value
		if s.steps[i].total {
			value = b[1]
		}
		s.rects = append(s.rects, barRect{x: x, y: top, w: math.Round(w), h: h, index: i, value: value, cat: s.steps[i].name})
		// The dashed line to the next bar at the level they share
		if i+1 < len(bars) {
			y := chartCrisp(v.PY(b[1]))
			p.Reset()
			p.MoveTo(x+math.Round(w), y)
			p.LineTo(math.Round(v.PX(float64(i+1))-w/2), y)
			cnv.StrokePath(p, StrokeStyle{Width: 1, Dash: []float64{3, 2}}, v.Theme.Axis)
		}
	}
	if !s.labels {
		return
	}
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.85)
	_, lineH := cnv.MeasureTextRotated("0", 0)
	for _, r := range s.rects {
		text := chartValueLabel(v.YAxis.scale, r.value)
		if !s.steps[r.index].total && r.value > 0 {
			text = "+" + text
		}
		if chartTextWidth(cnv, text) > r.w+10 {
			continue
		}
		y, va := r.y-2, VAlignBottom
		if r.value < 0 && !s.steps[r.index].total {
			y, va = r.y+r.h+2, VAlignTop
		}
		if y-lineH < v.Y || y+lineH > v.Y+v.H {
			continue
		}
		chartDrawLabel(cnv, text, r.x+r.w/2, y, HAlignCenter, va, v.Theme.Text)
	}
}

// Nearest implements ChartSeries
func (s *WaterfallSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	for _, r := range s.rects {
		if px >= r.x-2 && px <= r.x+r.w+2 {
			yl := chartValueLabel(v.YAxis.scale, r.value)
			if !s.steps[r.index].total && r.value > 0 {
				yl = "+" + yl
			}
			return ChartHit{Series: s, Index: r.index, X: float64(r.index), Y: r.value, PX: r.x + r.w/2, PY: r.y,
				XLabel: r.cat, YLabel: yl, Name: s.name, Color: s.Color(v.Theme)}, true
		}
	}
	return ChartHit{}, false
}

// Legend implements ChartSeries
func (s *WaterfallSeries) Legend(th *ChartTheme) []ChartLegendItem {
	return s.legendItem(th, ChartLegendBox)
}

// ---------------------------------------------------------------------------
// Funnel

// FunnelSeries draws the stages of a process as bars narrowing down, each
// as wide as its count against the first stage, with the part that went on
// from the stage before: the visitors who registered, then paid. A pane of
// a funnel has no axes.
//
//	f := chart.Pane(0).AddFunnel("Sales")
//	f.Add("Visited", 12000).Add("Signed up", 3100).Add("Paid", 420)
type FunnelSeries struct {
	ChartSeriesBase
	stages  []funnelStage
	pyramid bool

	rects []barRect
}

type funnelStage struct {
	name  string
	value float64
}

// NewFunnelSeries makes a funnel of the name
func NewFunnelSeries(name string) *FunnelSeries {
	s := &FunnelSeries{}
	s.name = name
	return s
}

// AddFunnel adds a funnel to the pane
func (p *ChartPane) AddFunnel(name string) *FunnelSeries {
	s := NewFunnelSeries(name)
	p.AddSeries(s)
	return s
}

// Add adds the next stage
func (s *FunnelSeries) Add(name string, value float64) *FunnelSeries {
	s.stages = append(s.stages, funnelStage{name, value})
	s.update()
	return s
}

// SetPyramid joins the stages into a smooth shape instead of bars
func (s *FunnelSeries) SetPyramid(on bool) *FunnelSeries {
	s.pyramid = on
	s.update()
	return s
}

func (s *FunnelSeries) chartNoAxes() {}

// Range implements ChartSeries
func (s *FunnelSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	return 0, 0, false
}

// Prepare implements ChartSeries
func (s *FunnelSeries) Prepare(v *ChartViewport) {}

// Draw implements ChartSeries
func (s *FunnelSeries) Draw(cnv *Canvas, v *ChartViewport) {
	s.rects = s.rects[:0]
	n := len(s.stages)
	if n == 0 {
		return
	}
	top := 0.0
	for _, st := range s.stages {
		top = math.Max(top, st.value)
	}
	if top <= 0 {
		return
	}
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.9)
	_, lineH := cnv.MeasureTextRotated("0", 0)
	gap := lineH + 6 // the conversion between the stages
	bh := (v.H - gap*float64(n-1)) / float64(n)
	if bh < 4 {
		gap, bh = 2, (v.H-2*float64(n-1))/float64(n)
	}
	cx := v.X + v.W/2
	maxW := v.W * 0.9
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	for i, st := range s.stages {
		w := math.Max(2, maxW*st.value/top)
		y := v.Y + float64(i)*(bh+gap)
		col := v.Theme.SeriesColor(i)
		if b := s.base(); b.col != nil {
			col = *b.col
			col.A = uint8(255 - min(i*30, 150))
		}
		p.Reset()
		if s.pyramid && i+1 < n {
			nw := math.Max(2, maxW*s.stages[i+1].value/top)
			p.Polygon(PointF{cx - w/2, y}, PointF{cx + w/2, y}, PointF{cx + nw/2, y + bh + gap}, PointF{cx - nw/2, y + bh + gap})
		} else {
			p.RoundedRect(cx-w/2, y, w, bh, 3)
		}
		cnv.FillPath(p, col)
		s.rects = append(s.rects, barRect{x: cx - w/2, y: y, w: w, h: bh, index: i, value: st.value, cat: st.name})
		text := st.name + "  " + chartValueLabel(NewLinearScale(0, top), st.value)
		tc := chartContrastText(col)
		if chartTextWidth(cnv, text)+8 > w {
			tc = v.Theme.Text
			chartDrawLabel(cnv, text, cx+w/2+6, y+bh/2, HAlignLeft, VAlignCenter, tc)
		} else {
			chartDrawLabel(cnv, text, cx, y+bh/2, HAlignCenter, VAlignCenter, tc)
		}
		if i > 0 && !s.pyramid && gap >= lineH {
			prev := s.stages[i-1].value
			if prev > 0 {
				pct := strconv.FormatFloat(st.value/prev*100, 'f', 1, 64) + "%"
				chartDrawLabel(cnv, "↓ "+pct, cx, y-gap/2, HAlignCenter, VAlignCenter, v.Theme.Text)
			}
		}
	}
}

// Nearest implements ChartSeries
func (s *FunnelSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	for _, r := range s.rects {
		if py >= r.y && py <= r.y+r.h && px >= r.x && px <= r.x+r.w {
			yl := chartValueLabel(NewLinearScale(0, 1), r.value)
			if r.index > 0 && s.stages[0].value > 0 {
				yl += " (" + strconv.FormatFloat(r.value/s.stages[0].value*100, 'f', 1, 64) + "% of the first)"
			}
			return ChartHit{Series: s, Index: r.index, Y: r.value, PX: r.x + r.w/2, PY: r.y + r.h/2,
				XLabel: r.cat, YLabel: yl, Name: r.cat, Color: v.Theme.SeriesColor(r.index)}, true
		}
	}
	return ChartHit{}, false
}

// Legend implements ChartSeries: a funnel names its stages itself
func (s *FunnelSeries) Legend(th *ChartTheme) []ChartLegendItem { return nil }
