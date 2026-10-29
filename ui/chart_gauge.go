package ui

import (
	"image/color"
	"math"
	"strconv"
)

// ChartZone is a range of values with its meaning: normal, warning,
// critical. Gauges and level indicators color their scale with the zones
// and their value with the zone it is in.
type ChartZone struct {
	From, To float64
	Color    color.RGBA
	Label    string
}

// gaugeBase is what a gauge, a level indicator and a KPI tile have: a
// value within a range, zones, a unit and a format
type gaugeBase struct {
	ChartSeriesBase
	value, min, max float64
	zones           []ChartZone
	unit            string
	decimals        int
	label           string
}

func (g *gaugeBase) chartNoAxes() {}

func (g *gaugeBase) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	return 0, 0, false
}

func (g *gaugeBase) Prepare(v *ChartViewport) {}

func (g *gaugeBase) Legend(th *ChartTheme) []ChartLegendItem { return nil }

// text returns the value as written: its decimals and its unit
func (g *gaugeBase) text(v float64) string {
	d := g.decimals
	if d < 0 {
		d = stepDecimals(niceStepAtLeast((g.max - g.min) / 200))
	}
	return strconv.FormatFloat(v, 'f', d, 64) + g.unit
}

// colorOf returns the color of the zone of the value, or the series' color
func (g *gaugeBase) colorOf(v float64, th *ChartTheme) color.RGBA {
	for _, z := range g.zones {
		if v >= math.Min(z.From, z.To) && v <= math.Max(z.From, z.To) {
			return z.Color
		}
	}
	return g.Color(th)
}

// part returns where the value is in the range, 0..1
func (g *gaugeBase) part(v float64) float64 {
	if g.max <= g.min {
		return 0
	}
	return math.Max(0, math.Min(1, (v-g.min)/(g.max-g.min)))
}

// Nearest: a gauge shows its value itself, no tooltip
func (g *gaugeBase) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	return ChartHit{}, false
}

// GaugeChart is a dial: an arc of the range with its zones, the value as an
// arc of its zone's color and as a number in the middle. For a dashboard, a
// panel of a machine.
//
//	g := ui.NewGauge(0, 120)
//	g.SetZones(ui.GaugeZonesOf(nil, 0, 80, 100, 120)).SetUnit(" °C").SetValue(72)
type GaugeChart struct {
	ChartView
	g *gaugeSeries
}

type gaugeSeries struct {
	gaugeBase
	needle bool
}

// NewGauge makes a dial of the range
func NewGauge(min, max float64) *GaugeChart {
	c := &GaugeChart{}
	c.initChartView(NewLinearScale(0, 1))
	c.SetTypeName("Gauge")
	c.SetMinSize(120, 100)
	c.legend = ChartLegendNone
	c.hoverMode = ChartHoverNone
	c.in.UseProfile(ChartProfileStatic)
	c.g = &gaugeSeries{}
	c.g.min, c.g.max, c.g.decimals = min, max, -1
	c.panes[0].AddSeries(c.g)
	return c
}

// SetValue sets the value shown
func (c *GaugeChart) SetValue(v float64) *GaugeChart {
	c.g.value = v
	c.Update()
	return c
}

// Value returns the value shown
func (c *GaugeChart) Value() float64 { return c.g.value }

// SetRange sets the range of the dial
func (c *GaugeChart) SetRange(min, max float64) *GaugeChart {
	c.g.min, c.g.max = min, max
	c.Update()
	return c
}

// SetZones sets the zones of the scale
func (c *GaugeChart) SetZones(zones []ChartZone) *GaugeChart {
	c.g.zones = zones
	c.Update()
	return c
}

// SetUnit sets the unit written after the value
func (c *GaugeChart) SetUnit(unit string) *GaugeChart {
	c.g.unit = unit
	c.Update()
	return c
}

// SetDecimals sets the decimals of the value; -1 - by the range
func (c *GaugeChart) SetDecimals(d int) *GaugeChart {
	c.g.decimals = d
	c.Update()
	return c
}

// SetLabel sets the text under the value: what it measures
func (c *GaugeChart) SetLabel(label string) *GaugeChart {
	c.g.label = label
	c.Update()
	return c
}

// SetNeedle draws a needle pointing at the value
func (c *GaugeChart) SetNeedle(on bool) *GaugeChart {
	c.g.needle = on
	c.Update()
	return c
}

// GaugeZonesOf returns the usual zones of a range from min to max: normal
// up to warn, warning up to crit, critical beyond, in the theme's colors
// (nil - the current one)
func GaugeZonesOf(th *ChartTheme, min, warn, crit, max float64) []ChartZone {
	if th == nil {
		th = CurrentChartTheme()
	}
	return []ChartZone{
		{From: min, To: warn, Color: th.Good},
		{From: warn, To: crit, Color: th.Warning},
		{From: crit, To: max, Color: th.Critical},
	}
}

// Draw draws the dial: 240° of a circle, open at the bottom
func (g *gaugeSeries) Draw(cnv *Canvas, v *ChartViewport) {
	th := v.Theme
	const sweep = 4 * math.Pi / 3
	start := math.Pi/2 + (2*math.Pi-sweep)/2
	r := math.Min(v.W/2, v.H/(1+math.Sin((2*math.Pi-sweep)/2)*0.9)/1.05) - 4
	if r < 10 {
		return
	}
	cx, cy := v.X+v.W/2, v.Y+r+4
	thick := math.Max(6, r*0.16)
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	// The track, the zones on its outer edge, the value
	p.Reset()
	p.Sector(cx, cy, r-thick, r, start, start+sweep)
	cnv.FillPath(p, th.Grid)
	for _, z := range g.zones {
		a0, a1 := start+sweep*g.part(z.From), start+sweep*g.part(z.To)
		p.Reset()
		p.Sector(cx, cy, r+2, r+5, math.Min(a0, a1), math.Max(a0, a1))
		cnv.FillPath(p, z.Color)
	}
	col := g.colorOf(g.value, th)
	p.Reset()
	p.Sector(cx, cy, r-thick, r, start, start+sweep*g.part(g.value))
	cnv.FillPath(p, col)
	if g.needle {
		a := start + sweep*g.part(g.value)
		p.Reset()
		p.MoveTo(cx+math.Cos(a+math.Pi/2)*3, cy+math.Sin(a+math.Pi/2)*3)
		p.LineTo(cx+math.Cos(a)*(r-thick/2), cy+math.Sin(a)*(r-thick/2))
		p.LineTo(cx+math.Cos(a-math.Pi/2)*3, cy+math.Sin(a-math.Pi/2)*3)
		p.Close()
		cnv.FillPath(p, th.Title)
		cnv.FillCircle(cx, cy, 5, th.Title)
	}
	// The value, what it is, the ends of the range
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(math.Max(10, r*0.32))
	cnv.SetFontBold(true)
	ty := cy
	if g.needle {
		ty = cy + r*0.38
	}
	chartDrawLabel(cnv, g.text(g.value), cx, ty, HAlignCenter, VAlignCenter, th.Title)
	cnv.SetFontBold(false)
	cnv.SetFontSize(math.Max(8, r*0.12))
	_, lh := cnv.MeasureTextRotated("0", 0)
	if g.label != "" {
		chartDrawLabel(cnv, g.label, cx, ty+r*0.2+lh/2, HAlignCenter, VAlignCenter, th.Text)
	}
	for _, e := range []struct {
		val float64
		a   float64
	}{{g.min, start}, {g.max, start + sweep}} {
		x, y := cx+math.Cos(e.a)*(r-thick/2), cy+math.Sin(e.a)*(r-thick/2)+thick/2+lh/2+2
		chartDrawLabel(cnv, trimChartZeros(strconv.FormatFloat(e.val, 'f', 2, 64)), x, y, HAlignCenter, VAlignCenter, th.Text)
	}
}

// LevelChart is a vertical (or horizontal) scale filled to the value: the
// level in a tank, a temperature, a load, with its set points marked. For
// the mimic diagrams of a SCADA.
//
//	l := ui.NewLevelIndicator(0, 10)
//	l.SetZones(ui.GaugeZonesOf(nil, 0, 7, 9, 10)).AddSetPoint(8, "High").SetValue(6.4)
type LevelChart struct {
	ChartView
	l *levelSeries
}

type levelSeries struct {
	gaugeBase
	horizontal bool
	tank       bool
	setPoints  []ChartZone // From is the value, Label the name
}

// NewLevelIndicator makes a vertical level indicator of the range
func NewLevelIndicator(min, max float64) *LevelChart {
	c := &LevelChart{}
	c.initChartView(NewLinearScale(0, 1))
	c.SetTypeName("LevelIndicator")
	c.SetMinSize(60, 120)
	c.legend = ChartLegendNone
	c.hoverMode = ChartHoverNone
	c.in.UseProfile(ChartProfileStatic)
	c.l = &levelSeries{tank: true}
	c.l.min, c.l.max, c.l.decimals = min, max, -1
	c.panes[0].AddSeries(c.l)
	return c
}

// SetValue sets the level
func (c *LevelChart) SetValue(v float64) *LevelChart {
	c.l.value = v
	c.Update()
	return c
}

// SetZones colors the scale and the level by the zones
func (c *LevelChart) SetZones(zones []ChartZone) *LevelChart {
	c.l.zones = zones
	c.Update()
	return c
}

// AddSetPoint marks a value on the scale: a set point, an alarm limit
func (c *LevelChart) AddSetPoint(v float64, label string) *LevelChart {
	c.l.setPoints = append(c.l.setPoints, ChartZone{From: v, Label: label})
	c.Update()
	return c
}

// SetUnit sets the unit written after the value
func (c *LevelChart) SetUnit(unit string) *LevelChart {
	c.l.unit = unit
	c.Update()
	return c
}

// SetHorizontal fills the scale from the left instead of from the bottom
func (c *LevelChart) SetHorizontal(horizontal bool) *LevelChart {
	c.l.horizontal = horizontal
	c.Update()
	return c
}

// SetTank draws a tank with rounded ends (on by default) or a plain bar
func (c *LevelChart) SetTank(tank bool) *LevelChart {
	c.l.tank = tank
	c.Update()
	return c
}

func (l *levelSeries) Draw(cnv *Canvas, v *ChartViewport) {
	th := v.Theme
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.85)
	_, lh := cnv.MeasureTextRotated("0", 0)
	labelW := 0.0
	for _, sp := range l.setPoints {
		labelW = math.Max(labelW, chartTextWidth(cnv, sp.Label+" "+l.text(sp.From)))
	}
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	col := l.colorOf(l.value, th)
	if l.horizontal {
		x, y, w, h := v.X, v.Y+lh+4, v.W, math.Min(v.H-lh*2-8, 40)
		rad := 0.0
		if l.tank {
			rad = h / 2
		}
		p.Reset()
		p.RoundedRect(x, y, w, h, rad)
		cnv.FillPath(p, th.Grid)
		p.Reset()
		p.RoundedRect(x, y, w*l.part(l.value), h, rad)
		cnv.FillPath(p, col)
		for _, sp := range l.setPoints {
			sx := chartCrisp(x + w*l.part(sp.From))
			p.Reset()
			p.MoveTo(sx, y-3)
			p.LineTo(sx, y+h+3)
			cnv.StrokePath(p, StrokeStyle{Width: 1.5}, th.Title)
			chartDrawLabel(cnv, sp.Label, sx, y-4, HAlignCenter, VAlignBottom, th.Text)
		}
		chartDrawLabel(cnv, l.text(l.value), x+w/2, y+h+4, HAlignCenter, VAlignTop, th.Title)
		return
	}
	// The value on top, the tank under it, the set points on its right
	valueH := lh + 6
	w := math.Min(v.W-labelW-12, math.Max(30, v.W*0.35))
	x, y, h := v.X+4, v.Y+valueH, v.H-valueH-2
	rad := 0.0
	if l.tank {
		rad = math.Min(w/2, 14)
	}
	p.Reset()
	p.RoundedRect(x, y, w, h, rad)
	cnv.FillPath(p, th.Grid)
	// The zones as a thin band at the right edge of the tank
	for _, z := range l.zones {
		y0, y1 := y+h*(1-l.part(z.To)), y+h*(1-l.part(z.From))
		p.Reset()
		p.Rect(x+w+2, y0, 3, y1-y0)
		cnv.FillPath(p, z.Color)
	}
	// The level: the tank's shape cut at the value
	cnv.Save()
	fill := y + h*(1-l.part(l.value))
	ix, iy := int(math.Floor(x)), int(math.Floor(fill))
	cnv.TranslateAndClip(ix, iy, int(math.Ceil(w))+1, int(math.Ceil(y+h-fill))+1)
	cnv.translateBy(-ix, -iy)
	p.Reset()
	p.RoundedRect(x, y, w, h, rad)
	cnv.FillPath(p, col)
	cnv.Restore()
	p.Reset()
	p.RoundedRect(x+0.5, y+0.5, w-1, h-1, rad)
	cnv.StrokePath(p, StrokeStyle{Width: 1}, th.Axis)
	for _, sp := range l.setPoints {
		sy := chartCrisp(y + h*(1-l.part(sp.From)))
		p.Reset()
		p.MoveTo(x-2, sy)
		p.LineTo(x+w+8, sy)
		cnv.StrokePath(p, StrokeStyle{Width: 1.5, Dash: []float64{4, 2}}, th.Title)
		chartDrawLabel(cnv, sp.Label+" "+l.text(sp.From), x+w+10, sy, HAlignLeft, VAlignCenter, th.Text)
	}
	cnv.SetFontBold(true)
	chartDrawLabel(cnv, l.text(l.value), x+w/2, v.Y, HAlignCenter, VAlignTop, th.Title)
}

// KPITile is a number of a dashboard: what it is, its value, how it
// changed against the previous value (green up, red down - or the other
// way for SetLowerIsBetter), and its recent trend as a sparkline.
//
//	k := ui.NewKPITile("Revenue")
//	k.SetUnit(" k$").SetValue(1280, 1150).SetTrend(last30days)
type KPITile struct {
	ChartView
	k *kpiSeries
}

type kpiSeries struct {
	gaugeBase
	hasPrev     bool
	prevValue   float64
	trend       []float64
	lowerBetter bool
}

// NewKPITile makes a tile of the title
func NewKPITile(title string) *KPITile {
	c := &KPITile{}
	c.initChartView(NewLinearScale(0, 1))
	c.SetTypeName("KPITile")
	c.SetMinSize(160, 90)
	c.legend = ChartLegendNone
	c.hoverMode = ChartHoverNone
	c.in.UseProfile(ChartProfileStatic)
	c.k = &kpiSeries{}
	c.k.label, c.k.decimals = title, -2
	c.panes[0].AddSeries(c.k)
	return c
}

// SetValue sets the value and the previous one it is compared with
func (c *KPITile) SetValue(value, previous float64) *KPITile {
	c.k.value, c.k.prevValue, c.k.hasPrev = value, previous, true
	c.Update()
	return c
}

// SetValueOnly sets the value without a comparison
func (c *KPITile) SetValueOnly(value float64) *KPITile {
	c.k.value, c.k.hasPrev = value, false
	c.Update()
	return c
}

// SetTrend sets the values of the sparkline under the number
func (c *KPITile) SetTrend(values []float64) *KPITile {
	c.k.trend = append(c.k.trend[:0], values...)
	c.Update()
	return c
}

// SetUnit sets the unit written after the value
func (c *KPITile) SetUnit(unit string) *KPITile {
	c.k.unit = unit
	c.Update()
	return c
}

// SetDecimals sets the decimals of the value; -2 - three significant digits
func (c *KPITile) SetDecimals(d int) *KPITile {
	c.k.decimals = d
	c.Update()
	return c
}

// SetLowerIsBetter colors a fall green and a rise red: for costs, errors,
// response times
func (c *KPITile) SetLowerIsBetter(lower bool) *KPITile {
	c.k.lowerBetter = lower
	c.Update()
	return c
}

func (k *kpiSeries) valueText(v float64) string {
	if k.decimals >= 0 {
		return strconv.FormatFloat(v, 'f', k.decimals, 64) + k.unit
	}
	return chartValueLabel(&LinearScale{Unit: k.unit, SI: math.Abs(v) >= 1e5}, v)
}

func (k *kpiSeries) Draw(cnv *Canvas, v *ChartViewport) {
	th := v.Theme
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.9)
	_, lh := cnv.MeasureTextRotated("0", 0)
	chartDrawLabel(cnv, k.label, v.X+2, v.Y, HAlignLeft, VAlignTop, th.Text)
	big := math.Max(14, math.Min(v.H*0.32, v.FontSize*2.4))
	cnv.SetFontSize(big)
	cnv.SetFontBold(true)
	_, bh := cnv.MeasureTextRotated("0", 0)
	text := k.valueText(k.value)
	chartDrawLabel(cnv, text, v.X+2, v.Y+lh+2, HAlignLeft, VAlignTop, th.Title)
	tw := chartTextWidth(cnv, text)
	cnv.SetFontBold(false)
	cnv.SetFontSize(v.FontSize * 0.9)
	if k.hasPrev && k.prevValue != 0 {
		d := (k.value - k.prevValue) / math.Abs(k.prevValue) * 100
		up := d >= 0
		good := up != k.lowerBetter
		col := th.Good
		if !good {
			col = th.Critical
		}
		arrow := "▲ "
		if !up {
			arrow = "▼ "
		}
		chartDrawLabel(cnv, arrow+strconv.FormatFloat(math.Abs(d), 'f', 1, 64)+"%", v.X+tw+10, v.Y+lh+2+bh-lh, HAlignLeft, VAlignTop, col)
	}
	if len(k.trend) > 1 {
		ty := v.Y + lh + bh + 6
		if h := v.Y + v.H - ty; h > 8 {
			DrawSparkline(cnv, v.X, ty, v.W, h, k.trend, ChartSparkLine, th.SeriesColor(0), th)
		}
	}
}
