package ui

import (
	"image/color"
	"math"
)

// ChartSparklineKind is how a sparkline draws its values
type ChartSparklineKind int

const (
	// ChartSparkLine is a line with the area under it lightly filled
	ChartSparkLine ChartSparklineKind = iota
	// ChartSparkBars is a column per value
	ChartSparkBars
	// ChartSparkWinLoss is a column up for a value above zero, down for
	// below, all of the same height
	ChartSparkWinLoss
)

// Sparkline is a small chart without axes, legend or frame: the trend of a
// value in a line of a dashboard, beside a number. Its last value is marked.
//
//	sp := ui.NewSparkline()
//	sp.SetValues(cpuHistory)
//
// To draw one in a cell of a table, use DrawSparkline.
type Sparkline struct {
	ChartView
	kind   ChartSparklineKind
	values []float64
	max    int
	col    *color.RGBA
}

// NewSparkline makes an empty sparkline
func NewSparkline() *Sparkline {
	c := &Sparkline{}
	c.initChartView(NewLinearScale(0, 1))
	c.SetTypeName("Sparkline")
	c.SetMinSize(60, 20)
	c.SetCanBeFocused(false)
	c.SetProp("padding", 2)
	c.frameless = true
	c.legend = ChartLegendNone
	c.hoverMode = ChartHoverNone
	c.in.UseProfile(ChartProfileStatic)
	c.xAxis.SetVisible(false).SetGrid(false, false)
	c.panes[0].YAxis().SetVisible(false).SetGrid(false, false)
	c.paintFunc = c.paintSparkline
	return c
}

// SetValues replaces the values
func (c *Sparkline) SetValues(values []float64) *Sparkline {
	c.values = append(c.values[:0], values...)
	c.trim()
	c.Update()
	return c
}

// AddValue adds a value at the end; with SetMaxValues the first ones go
func (c *Sparkline) AddValue(v float64) *Sparkline {
	c.values = append(c.values, v)
	c.trim()
	c.Update()
	return c
}

// SetMaxValues keeps the last n values only; 0 keeps all
func (c *Sparkline) SetMaxValues(n int) *Sparkline {
	c.max = max(n, 0)
	c.trim()
	c.Update()
	return c
}

func (c *Sparkline) trim() {
	if c.max > 0 && len(c.values) > c.max {
		c.values = c.values[len(c.values)-c.max:]
	}
}

// Values returns the values
func (c *Sparkline) Values() []float64 { return c.values }

// SetKind sets how the values are drawn
func (c *Sparkline) SetKind(kind ChartSparklineKind) *Sparkline {
	c.kind = kind
	c.Update()
	return c
}

// SetColor fixes the color; without it the first color of the theme's palette
func (c *Sparkline) SetColor(col color.RGBA) *Sparkline {
	c.col = &col
	c.Update()
	return c
}

func (c *Sparkline) paintSparkline(cnv *Canvas) {
	th := c.fixedTheme
	if th == nil {
		th = CurrentChartTheme()
	}
	c.theme = th
	w, h := c.size()
	cnv.FillRect(0, 0, int(w), int(h), th.Background)
	col := th.SeriesColor(0)
	if c.col != nil {
		col = *c.col
	}
	pad := float64(c.GetPropInt("padding", 2))
	DrawSparkline(cnv, pad, pad, w-2*pad, h-2*pad, c.values, c.kind, col, th)
}

// DrawSparkline draws the values as a sparkline in the rectangle of the
// canvas: in a cell of a table, in a widget's own paint. th gives the colors
// of the win/loss columns; nil - the current theme.
func DrawSparkline(cnv *Canvas, x, y, w, h float64, values []float64, kind ChartSparklineKind, col color.RGBA, th *ChartTheme) {
	if w <= 2 || h <= 2 || len(values) == 0 {
		return
	}
	if th == nil {
		th = CurrentChartTheme()
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range values {
		if chartFinite(v) {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	if lo > hi {
		return
	}
	if kind != ChartSparkLine {
		lo, hi = math.Min(lo, 0), math.Max(hi, 0)
	}
	if hi-lo < 1e-300 {
		lo, hi = lo-1, hi+1
	}
	n := len(values)
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	switch kind {
	case ChartSparkLine:
		// A margin for the marker of the last value
		x0, x1 := x+1.5, x+w-3
		y0, y1 := y+2, y+h-2
		px := func(i int) float64 {
			if n == 1 {
				return x1
			}
			return x0 + float64(i)/float64(n-1)*(x1-x0)
		}
		py := func(v float64) float64 { return y1 - (v-lo)/(hi-lo)*(y1-y0) }
		p.Reset()
		started := false
		first, last := -1, -1
		for i, v := range values {
			if !chartFinite(v) {
				started = false
				continue
			}
			if !started {
				p.MoveTo(px(i), py(v))
				started = true
			} else {
				p.LineTo(px(i), py(v))
			}
			if first < 0 {
				first = i
			}
			last = i
		}
		if first >= 0 && last > first {
			area := pathPool.Get().(*Path)
			area.Reset()
			area.MoveTo(px(first), y1)
			for i := first; i <= last; i++ {
				if chartFinite(values[i]) {
					area.LineTo(px(i), py(values[i]))
				}
			}
			area.LineTo(px(last), y1)
			area.Close()
			fill := col
			fill.A = col.A / 6
			cnv.FillPath(area, fill)
			pathPool.Put(area)
		}
		cnv.StrokePath(p, StrokeStyle{Width: 1.25, Join: LineJoinRound, Cap: LineCapRound}, col)
		if last >= 0 {
			cnv.FillCircle(px(last), py(values[last]), 2, col)
		}
	default:
		bw := w / float64(n)
		gap := math.Min(1, bw*0.2)
		zero := y + h - (0-lo)/(hi-lo)*h
		for i, v := range values {
			if !chartFinite(v) {
				continue
			}
			c := col
			top, bottom := y+h-(v-lo)/(hi-lo)*h, zero
			if kind == ChartSparkWinLoss {
				mid := y + h/2
				c = th.Up
				top, bottom = y+1, mid-1
				if v < 0 {
					c = th.Down
					top, bottom = mid+1, y+h-1
				} else if v == 0 {
					continue
				}
			}
			if top > bottom {
				top, bottom = bottom, top
			}
			p.Reset()
			p.Rect(x+float64(i)*bw+gap/2, top, math.Max(bw-gap, 0.5), math.Max(bottom-top, 1))
			cnv.FillPath(p, c)
		}
	}
}
