package ui

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"testing"
	"time"
)

type auditData struct {
	name string
	v    []float64
}

func auditDatasets() []auditData {
	nan, inf := math.NaN(), math.Inf(1)
	many := make([]float64, 70000)
	for i := range many {
		many[i] = math.Sin(float64(i) / 100)
	}
	return []auditData{
		{"empty", nil},
		{"single", []float64{5}},
		{"zero", []float64{0}},
		{"two-equal", []float64{3, 3}},
		{"all-equal", []float64{7, 7, 7, 7}},
		{"all-zero", []float64{0, 0, 0, 0}},
		{"negative", []float64{-5, -3, -10, -1}},
		{"mixed", []float64{-5, 3, -10, 8}},
		{"nan", []float64{1, nan, 3}},
		{"all-nan", []float64{nan, nan}},
		{"inf", []float64{1, inf, 3}},
		{"-inf", []float64{1, -inf, 3}},
		{"huge", []float64{1e308, -1e308, 1.7e308}},
		{"maxf", []float64{math.MaxFloat64, math.MaxFloat64}},
		{"tiny", []float64{1e-320, 2e-320}},
		{"many", many},
	}
}

func auditPaint(c *ChartView, w, h int) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < 2; i++ {
		cnv := NewCanvas(img)
		cnv.SetDirectTranslateAndClip(0, 0, w, h)
		c.paintFunc(cnv)
	}
}

func auditInteract(c *ChartView, w, h int) {
	auditPaint(c, w, h)
	for y := 0; y < h; y += 23 {
		for x := 0; x < w; x += 29 {
			c.onMouseMove(x, y, KeyModifiers{})
			c.updateHits()
		}
	}
	auditPaint(c, w, h)
	for _, b := range []MouseButton{MouseButtonLeft, MouseButtonRight, MouseButtonMiddle} {
		for _, mods := range []KeyModifiers{{}, {Shift: true}, {Ctrl: true}, {Alt: true}} {
			for _, pt := range [][4]int{{w / 4, h / 3, w * 3 / 4, h * 2 / 3}, {w * 3 / 4, h / 2, w / 4, h / 2}, {w / 2, h / 2, w / 2, h - 5}, {10, h / 2, 10, h / 4}, {w / 2, h - 10, w / 4, h - 10}} {
				if !c.laidOut {
					auditPaint(c, w, h)
				}
				c.onMouseDown(b, pt[0], pt[1], mods)
				for k := 1; k <= 4; k++ {
					c.onMouseMove(pt[0]+(pt[2]-pt[0])*k/4, pt[1]+(pt[3]-pt[1])*k/4, mods)
				}
				c.ProcessMouseUp(b, pt[2], pt[3], mods, "")
				auditPaint(c, w, h)
				c.onMouseDblClick(b, pt[0], pt[1], mods)
				auditPaint(c, w, h)
			}
		}
	}
	for _, d := range []int{1, -1, 5, -5, 40, -40} {
		for _, mods := range []KeyModifiers{{}, {Shift: true}, {Ctrl: true}} {
			c.onWheel(&WheelEvent{DY: d, X: w / 2, Y: h / 2, Mods: mods})
			c.onWheel(&WheelEvent{DX: d, X: w / 2, Y: h / 2, Mods: mods})
			c.onWheel(&WheelEvent{DY: d, X: 10, Y: h / 2, Mods: mods})
			c.onWheel(&WheelEvent{DY: d, X: w / 2, Y: h - 8, Mods: mods})
			auditPaint(c, w, h)
		}
	}
	for _, k := range []Key{KeyArrowLeft, KeyArrowRight, KeyArrowUp, KeyArrowDown, KeyEqual, KeyMinus, KeyHome, KeyEnd, KeyEsc, KeyBackspace} {
		c.onKeyDown(k, KeyModifiers{})
		auditPaint(c, w, h)
	}
	c.SetSelection(0, 1)
	auditPaint(c, w, h)
	st := c.SaveState()
	if err := c.RestoreState(st); err != nil {
		panic(fmt.Sprintf("restore state: %v (%q)", err, st))
	}
	var b bytes.Buffer
	c.WriteCSV(&b, true)
	c.WriteCSV(&b, false)
	c.WriteSVG(&b, w, h, nil)
	c.Render(w, h, 1, nil)
	c.ResetZoom()
	auditPaint(c, w, h)
}

type auditBuilder struct {
	name  string
	build func(d []float64) *ChartView
}

func xs(n int) []float64 {
	r := make([]float64, n)
	for i := range r {
		r[i] = float64(i)
	}
	return r
}

func auditBuilders() []auditBuilder {
	cats := func(n int) []string {
		r := make([]string, n)
		for i := range r {
			r[i] = fmt.Sprintf("c%d", i)
		}
		return r
	}
	return []auditBuilder{
		{"line", func(d []float64) *ChartView { ch := NewLineChart(); ch.Series("a").SetValues(d); ch.Series("b").SetValues(d); return &ch.ChartView }},
		{"line-spline-fill", func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.Series("a").SetShape(ChartLineSpline).SetFill(true, 0.3, true).SetValues(d)
			return &ch.ChartView
		}},
		{"line-step-stack", func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.Series("a").SetShape(ChartLineStepAfter).SetStack("g").SetFill(true, 0.3, false).SetValues(d)
			ch.Series("b").SetStack("g").SetValues(d)
			ch.Pane(0).SetStackPercent("g", true)
			return &ch.ChartView
		}},
		{"line-log", func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.Series("a").SetValues(d)
			ch.Pane(0).YAxis().SetScale(NewLogScale(1, 10))
			ch.XAxis().SetScale(NewLogScale(1, 10))
			return &ch.ChartView
		}},
		{"line-symlog", func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.Series("a").SetValues(d)
			ch.Pane(0).YAxis().SetScale(NewSymlogScale(1, 10))
			return &ch.ChartView
		}},
		{"line-time", func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.XAxis().SetScale(&TimeScale{Location: time.UTC}).SetDateRibbon(true)
			s := ch.Series("a")
			for i, v := range d {
				s.AddXY(1.7e9+float64(i)*3600, v)
			}
			return &ch.ChartView
		}},
		{"line-time-xvalues", func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.XAxis().SetScale(&TimeScale{Location: time.UTC}).SetDateRibbon(true)
			s := ch.Series("a")
			for i, v := range d {
				s.AddXY(v, float64(i))
			}
			return &ch.ChartView
		}},
		{"line-xvalues", func(d []float64) *ChartView {
			ch := NewLineChart()
			s := ch.Series("a")
			for i, v := range d {
				s.AddXY(v, float64(i))
			}
			return &ch.ChartView
		}},
		{"bars-cat", func(d []float64) *ChartView {
			ch := NewBarChart()
			ch.Series("a").SetValues(d).SetLabels(true)
			ch.Series("b").SetValues(d).SetStack("s")
			ch.SetCategories(cats(len(d))...)
			return &ch.ChartView
		}},
		{"bars-horiz-stacked", func(d []float64) *ChartView {
			ch := NewBarChart()
			for i, v := range d {
				ch.Series("a").Set(fmt.Sprint(i), v)
				ch.Series("b").Set(fmt.Sprint(i), -v)
			}
			ch.SetHorizontal(true).SetStacked(true)
			return &ch.ChartView
		}},
		{"bars-xy", func(d []float64) *ChartView {
			ch := NewXYChart()
			b := ch.Pane(0).AddBars("a")
			for i, v := range d {
				b.AddXY(float64(i), v)
			}
			return ch
		}},
		{"pie", func(d []float64) *ChartView {
			ch := NewPieChart()
			for i, v := range d {
				ch.Add(fmt.Sprint("s", i), v)
			}
			ch.Pie().SetLabels(ChartPieLabels(1))
			return &ch.ChartView
		}},
		{"donut-other", func(d []float64) *ChartView {
			ch := NewPieChart()
			for i, v := range d {
				ch.Add(fmt.Sprint("s", i), v)
			}
			ch.SetDonut(0.5)
			ch.Pie().SetOther(0.1, "other")
			return &ch.ChartView
		}},
		{"scatter", func(d []float64) *ChartView {
			ch := NewXYChart()
			s := ch.Pane(0).AddScatter("a")
			s.SetXY(d, d)
			b := ch.Pane(0).AddScatter("b").SetBubbleSizes(2, 30).SetStems(true, 0)
			b.SetColorScale(CurrentChartTheme().Sequential, 0, 0)
			for i, v := range d {
				b.AddPoint(ScatterPoint{X: float64(i), Y: v, Size: v, Value: v})
			}
			return ch
		}},
		{"candles", func(d []float64) *ChartView {
			ch := NewXYChart()
			c := ch.Pane(0).AddCandles("a")
			for i, v := range d {
				c.AddOHLC(float64(i), v, v-1, v+1, -v) // inverted high/low
			}
			return ch
		}},
		{"candles-time", func(d []float64) *ChartView {
			ch := NewXYChart()
			ch.XAxis().SetScale(&TimeScale{Location: time.UTC})
			c := ch.Pane(0).AddCandles("a")
			for i, v := range d {
				c.AddOHLC(1.7e9+float64(i)*86400, v, v+1, v-1, v)
			}
			return ch
		}},
		{"heatmap", func(d []float64) *ChartView {
			ch := NewXYChart()
			n := len(d)
			if n > 300 {
				n = 300
			}
			g := make([][]float64, n)
			for i := range g {
				g[i] = d[:n]
			}
			ch.Pane(0).AddHeatmap("h").SetGrid(g, 0, 0, 0, 0).SetLabels(true).SetColorBar(true)
			return ch
		}},
		{"heatmap-ragged", func(d []float64) *ChartView {
			ch := NewXYChart()
			g := [][]float64{d, nil, {1}}
			ch.Pane(0).AddHeatmap("h").SetGrid(g, 0, 10, 0, 10).SetColors(nil, 0, 0)
			return ch
		}},
		{"heatmap-cat", func(d []float64) *ChartView {
			ch := NewXYChart()
			ch.XAxis().SetScale(NewCategoryScale("a", "b"))
			ch.Pane(0).YAxis().SetScale(NewCategoryScale("x"))
			g := [][]float64{d, d, d}
			ch.Pane(0).AddHeatmap("h").SetGrid(g, 0, 0, 0, 0).SetLabels(true)
			return ch
		}},
		{"hist2d", func(d []float64) *ChartView {
			ch := NewXYChart()
			pts := make([]ChartPoint, len(d))
			for i, v := range d {
				pts[i] = ChartPoint{v, float64(i)}
			}
			ch.Pane(0).AddHistogram2D("h").SetPoints(pts).SetBins(0, -3)
			return ch
		}},
		{"hexbin", func(d []float64) *ChartView {
			ch := NewXYChart()
			pts := make([]ChartPoint, len(d))
			for i, v := range d {
				pts[i] = ChartPoint{v, v}
			}
			ch.Pane(0).AddHistogram2D("h").SetPoints(pts).SetHex(0)
			return ch
		}},
		{"histogram", func(d []float64) *ChartView {
			ch := NewXYChart()
			ch.Pane(0).AddHistogram("h").SetValues(d)
			return ch
		}},
		{"histogram-bins", func(d []float64) *ChartView {
			ch := NewXYChart()
			ch.Pane(0).AddHistogram("h").SetValues(d).SetBins(1000000, 0).SetMode(ChartHistogramDensity)
			return ch
		}},
		{"histogram-width", func(d []float64) *ChartView {
			ch := NewXYChart()
			ch.Pane(0).AddHistogram("h").SetValues(d).SetBins(0, 1e-9)
			return ch
		}},
		{"boxplot", func(d []float64) *ChartView {
			ch := NewXYChart()
			ch.Pane(0).AddBoxPlot("b").Set("a", d).Set("b", nil)
			ch.Pane(0).AddBoxPlot("v").SetViolin(true).SetMean(true).Set("a", d)
			return ch
		}},
		{"band", func(d []float64) *ChartView {
			ch := NewXYChart()
			b := ch.Pane(0).AddBand("b")
			e := ch.Pane(0).AddBand("e").SetErrorBars(true)
			for i, v := range d {
				b.Add(float64(i), v, v+1, v-1)
				e.Add(float64(i), v, v-1, v+1)
			}
			return ch
		}},
		{"spans", func(d []float64) *ChartView {
			ch := NewXYChart()
			s := ch.Pane(0).AddSpans("s").SetLabels(true)
			for i, v := range d {
				s.Add(ChartSpan{ID: fmt.Sprint(i), Row: fmt.Sprint("r", i%3), Start: v, End: v - 1, Label: "x", Progress: v})
			}
			s.AddDependency("0", "1").AddDependency("1", "0").AddDependency("x", "y")
			return ch
		}},
		{"digital", func(d []float64) *ChartView {
			ch := NewXYChart()
			s := ch.Pane(0).AddDigital("d")
			b := ch.Pane(0).AddDigital("bus").SetBus(true)
			for i, v := range d {
				s.AddState(float64(i), v)
				b.AddState(v, v)
			}
			return ch
		}},
		{"events", func(d []float64) *ChartView {
			ch := NewXYChart()
			s := ch.Pane(0).AddEvents("e")
			for i, v := range d {
				s.Add(ChartEvent{X: v, Source: fmt.Sprint(i % 2), Severity: ChartSeverity(i % 5), Text: "t"})
			}
			return ch
		}},
		{"waterfall", func(d []float64) *ChartView {
			ch := NewXYChart()
			s := ch.Pane(0).AddWaterfall("w").SetLabels(true)
			for i, v := range d {
				s.AddStep(fmt.Sprint(i), v)
			}
			s.AddTotal("total")
			return ch
		}},
		{"funnel", func(d []float64) *ChartView {
			ch := NewXYChart()
			s := ch.Pane(0).AddFunnel("f")
			p := ch.AddPane().AddFunnel("p").SetPyramid(true)
			for i, v := range d {
				s.Add(fmt.Sprint(i), v)
				p.Add(fmt.Sprint(i), v)
			}
			return ch
		}},
		{"radar", func(d []float64) *ChartView {
			ch := NewXYChart()
			r := ch.Pane(0).AddRadar("r").SetFill(true)
			for i, v := range d {
				r.Set(fmt.Sprint(i), v)
			}
			return ch
		}},
		{"polar", func(d []float64) *ChartView {
			ch := NewXYChart()
			for k := 0; k < 4; k++ {
				p := ch.Pane(0).AddPolar(fmt.Sprint("p", k)).SetKind(ChartPolarKind(k)).SetStack("s")
				for i, v := range d {
					p.Add(float64(i)*30, v)
				}
			}
			return ch
		}},
		{"gauge", func(d []float64) *ChartView {
			v, lo, hi := 0.0, 0.0, 0.0
			if len(d) > 0 {
				v = d[0]
				lo, hi = d[0], d[len(d)-1]
			}
			g := NewGauge(lo, hi).SetValue(v).SetZones([]ChartZone{{From: lo, To: hi}, {From: hi, To: lo}}).SetDecimals(-5)
			return &g.ChartView
		}},
		{"level", func(d []float64) *ChartView {
			v, lo, hi := 0.0, 0.0, 0.0
			if len(d) > 0 {
				v = d[0]
				lo, hi = d[0], d[len(d)-1]
			}
			g := NewLevelIndicator(lo, hi).SetValue(v).SetTank(true).AddSetPoint(v, "sp").SetZones([]ChartZone{{From: lo, To: hi}})
			return &g.ChartView
		}},
		{"level-h", func(d []float64) *ChartView {
			v, lo, hi := 0.0, 0.0, 0.0
			if len(d) > 0 {
				v = d[0]
				lo, hi = d[len(d)-1], d[0]
			}
			g := NewLevelIndicator(lo, hi).SetValue(v).SetHorizontal(true)
			return &g.ChartView
		}},
		{"kpi", func(d []float64) *ChartView {
			k := NewKPITile("k").SetTrend(d).SetDecimals(-3)
			if len(d) > 1 {
				k.SetValue(d[0], d[1])
			} else if len(d) == 1 {
				k.SetValueOnly(d[0])
			}
			return &k.ChartView
		}},
		{"pareto", func(d []float64) *ChartView {
			p := NewParetoChart()
			for i, v := range d {
				p.Add(fmt.Sprint(i), v)
			}
			return &p.ChartView
		}},
		{"control", func(d []float64) *ChartView {
			c := NewControlChart().SetValues(d)
			return &c.ChartView
		}},
		{"control-limits", func(d []float64) *ChartView {
			c := NewControlChart().SetValues(d).SetLimits(0, 0)
			return &c.ChartView
		}},
		{"chart", func(d []float64) *ChartView {
			c := NewChart()
			pts := make([]ChartPoint, len(d))
			for i, v := range d {
				pts[i] = ChartPoint{v, v}
			}
			c.SetData(pts)
			return &c.ChartView
		}},
		{"annotations", func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.Series("a").SetValues(d)
			an := ch.Pane(0).Annotations()
			for _, v := range d {
				an.AddHLine(v, "h")
				an.AddVLine(v, "v")
				an.AddXZone(v, -v, "xz")
				an.AddYZone(v, v, "yz")
				an.AddLabel(v, v, "l")
			}
			return &ch.ChartView
		}},
		{"multi-pane", func(d []float64) *ChartView {
			ch := NewLineChart()
			ch.Series("a").SetValues(d)
			p := ch.AddPane()
			p.SetWeight(0)
			p.AddLine("b").SetValues(d)
			ax := p.AddAxis("y2", ChartAxisRight, NewLogScale(1, 2))
			_ = ax
			l := p.AddLine("c")
			l.SetAxisNames("x", "y2")
			l.SetValues(d)
			p2 := ch.AddPane()
			p2.SetXAxis(p2.NewAxis("x2", ChartAxisTop, NewLinearScale(0, 1)))
			l2 := p2.AddLine("d")
			l2.SetAxisNames("x2", "y")
			l2.SetValues(d)
			return &ch.ChartView
		}},
	}
}

func TestAuditChartsSmoke(t *testing.T) {
	sizes := [][2]int{{500, 350}, {130, 85}, {3, 3}}
	for _, b := range auditBuilders() {
		for _, d := range auditDatasets() {
			for _, sz := range sizes {
				b, d, sz := b, d, sz
				if d.name == "many" && sz[0] != 500 {
					continue
				}
				runCase(t, fmt.Sprintf("%s/%s/%dx%d", b.name, d.name, sz[0], sz[1]), func() {
					c := b.build(d.v)
					c.SetSize(sz[0], sz[1])
					if d.name == "many" {
						auditPaint(c, sz[0], sz[1])
						c.Render(sz[0], sz[1], 1, nil)
						return
					}
					auditInteract(c, sz[0], sz[1])
				})
			}
		}
	}
}
