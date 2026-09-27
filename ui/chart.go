package ui

import (
	"math"
	"strconv"
)

type ChartPoint struct {
	X float64
	Y float64
}

type Chart struct {
	Widget
	points []ChartPoint
}

type chartLayout struct {
	ok                                       bool
	w, h                                     int
	outer                                    int
	lineH                                    int
	plotL, plotT, plotR, plotB, plotW, plotH int
	vX0, vX1, vY0, vY1                       float64
	spanX, spanY                             float64
	xl, yl                                   string
	nTicks, tickDiv                          int
}

func NewChart() *Chart {
	var c Chart
	c.InitWidget()
	c.SetTypeName("Chart")
	c.SetMinSize(DefaultButtonMinWidth, DefaultUiLineHeight)
	c.SetMouseCursor(MouseCursorArrow)
	c.SetCanBeFocused(false)
	c.SetElevation(1)

	c.SetOnPaint(c.draw)

	c.SetProp("padding", 6)
	c.SetProp("tickcount", 5)

	c.SetXExpandable(true)
	c.SetYExpandable(true)

	return &c
}

func (c *Chart) Data() []ChartPoint {
	return c.points
}

func (c *Chart) SetData(points []ChartPoint) {
	c.points = points
	c.form.Update()
}

func formatChartTick(v, span float64) string {
	if span < 1e-12 {
		return strconv.FormatFloat(v, 'f', 1, 64)
	}
	mag := span
	var prec int
	switch {
	case mag >= 100:
		prec = 0
	case mag >= 10:
		prec = 1
	case mag >= 1:
		prec = 2
	default:
		prec = 3
	}
	return strconv.FormatFloat(v, 'f', prec, 64)
}

func (c *Chart) buildLayout() chartLayout {
	var ly chartLayout
	ly.w, ly.h = c.Width(), c.Height()
	ly.xl = c.GetPropString("xlabel", "")
	ly.yl = c.GetPropString("ylabel", "")
	ly.nTicks = c.GetPropInt("tickcount", 5)
	if ly.nTicks < 2 {
		ly.nTicks = 2
	}
	if ly.nTicks > 12 {
		ly.nTicks = 12
	}
	ly.tickDiv = ly.nTicks - 1
	ly.outer = c.GetPropInt("padding", 6)

	if len(c.points) < 1 {
		return ly
	}

	fontFamily := c.FontFamily()
	fontSize := c.FontSize()
	_, lineH, err := MeasureText(fontFamily, fontSize, "0")
	if err != nil {
		return ly
	}
	ly.lineH = lineH

	ly.vX0 = c.points[0].X
	ly.vX1 = c.points[0].X
	ly.vY0 = c.points[0].Y
	ly.vY1 = c.points[0].Y
	for i := 1; i < len(c.points); i++ {
		p := c.points[i]
		ly.vX0 = math.Min(ly.vX0, p.X)
		ly.vX1 = math.Max(ly.vX1, p.X)
		ly.vY0 = math.Min(ly.vY0, p.Y)
		ly.vY1 = math.Max(ly.vY1, p.Y)
	}
	ly.spanX = ly.vX1 - ly.vX0
	ly.spanY = ly.vY1 - ly.vY0
	if ly.spanX < 1e-9 {
		ly.spanX = 1
	}
	if ly.spanY < 1e-9 {
		ly.spanY = 1
	}

	maxYTickW := 0
	for j := 0; j < ly.nTicks; j++ {
		yv := ly.vY0 + float64(j)/float64(ly.tickDiv)*ly.spanY
		s := formatChartTick(yv, ly.spanY)
		tw, _, e := MeasureText(fontFamily, fontSize, s)
		if e == nil && tw > maxYTickW {
			maxYTickW = tw
		}
	}
	mLeft := 8 + maxYTickW
	if mLeft < 28 {
		mLeft = 28
	}

	mTop := ly.outer + 2
	if ly.yl != "" {
		mTop += ly.lineH + 4
	}
	mRight := ly.outer + 8
	mBottom := ly.outer + 4 + ly.lineH + 4
	if ly.xl != "" {
		mBottom += ly.lineH + 4
	}

	ly.plotL = ly.outer + mLeft
	ly.plotT = mTop
	ly.plotR = ly.w - 1 - mRight
	ly.plotB = ly.h - 1 - mBottom
	ly.plotW = ly.plotR - ly.plotL + 1
	ly.plotH = ly.plotB - ly.plotT + 1
	if ly.plotW < 1 || ly.plotH < 1 {
		return ly
	}
	ly.ok = true
	return ly
}

func (c *Chart) draw(cnv *Canvas) {
	// The same colors as the TimeChart
	th := currentTimeChartTheme()
	cnv.FillRect(0, 0, c.Width(), c.Height(), th.background)
	cnv.SetColor(th.border)
	cnv.DrawRect(0, 0, c.Width(), c.Height())

	ly := c.buildLayout()
	if !ly.ok {
		return
	}

	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())

	axis := th.axis
	muted := th.text
	lineC := th.palette[0]

	spanX := ly.spanX
	spanY := ly.spanY
	if spanX < 1e-9 {
		spanX = 1e-9
	}
	if spanY < 1e-9 {
		spanY = 1e-9
	}

	if ly.yl != "" {
		cnv.Save()
		cnv.SetColor(muted)
		cnv.SetHAlign(HAlignLeft)
		cnv.SetVAlign(VAlignTop)
		cnv.DrawText(ly.outer+2, ly.outer+1, c.Width()-2*ly.outer, ly.lineH+2, ly.yl)
		cnv.Restore()
	}

	cnv.DrawLine(ly.plotL, ly.plotB, ly.plotR, ly.plotB, 1, axis)
	cnv.DrawLine(ly.plotL, ly.plotT, ly.plotL, ly.plotB, 1, axis)

	cnv.SetColor(muted)
	cnv.SetHAlign(HAlignRight)
	cnv.SetVAlign(VAlignCenter)
	for j := 0; j < ly.nTicks; j++ {
		yv := ly.vY0 + float64(j)/float64(ly.tickDiv)*ly.spanY
		py := ly.plotT + int(math.Round((1-(yv-ly.vY0)/spanY)*float64(ly.plotH-1)))
		if py < ly.plotT {
			py = ly.plotT
		}
		if py > ly.plotB {
			py = ly.plotB
		}
		cnv.DrawLine(ly.plotL, py, ly.plotL+4, py, 1, axis)
		cnv.DrawText(ly.outer+2, py-ly.lineH/2, ly.plotL-2-ly.outer, ly.lineH+1, formatChartTick(yv, spanY))
	}
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignTop)

	cnv.SetHAlign(HAlignCenter)
	cnv.SetVAlign(VAlignTop)
	cnv.SetColor(muted)
	for j := 0; j < ly.nTicks; j++ {
		xv := ly.vX0 + float64(j)/float64(ly.tickDiv)*ly.spanX
		px := ly.plotL + int(math.Round((xv-ly.vX0)/spanX*float64(ly.plotW-1)))
		if px < ly.plotL {
			px = ly.plotL
		}
		if px > ly.plotR {
			px = ly.plotR
		}
		cnv.DrawLine(px, ly.plotB, px, ly.plotB+4, 1, axis)
		xs := formatChartTick(xv, spanX)
		tw, _, e := MeasureText(c.FontFamily(), c.FontSize(), xs)
		if e != nil {
			tw = 24
		}
		if tw < 12 {
			tw = 12
		}
		if tw > ly.plotW {
			tw = ly.plotW
		}
		cnv.DrawText(px-tw/2, ly.plotB+6, tw, ly.lineH+2, xs)
	}
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignTop)

	if ly.xl != "" {
		cnv.Save()
		cnv.SetColor(muted)
		cnv.SetHAlign(HAlignCenter)
		cnv.SetVAlign(VAlignTop)
		cnv.DrawText(0, c.Height()-ly.outer-ly.lineH-1, c.Width(), ly.lineH+2, ly.xl)
		cnv.Restore()
	}

	dxDen := 1.0
	if ly.plotW > 1 {
		dxDen = float64(ly.plotW - 1)
	}
	dyDen := 1.0
	if ly.plotH > 1 {
		dyDen = float64(ly.plotH - 1)
	}
	for i := 0; i < len(c.points)-1; i++ {
		a, b := c.points[i], c.points[i+1]
		x1 := ly.plotL + int(math.Round((a.X-ly.vX0)/spanX*dxDen))
		y1 := ly.plotT + int(math.Round((1-(a.Y-ly.vY0)/spanY)*dyDen))
		x2 := ly.plotL + int(math.Round((b.X-ly.vX0)/spanX*dxDen))
		y2 := ly.plotT + int(math.Round((1-(b.Y-ly.vY0)/spanY)*dyDen))
		cnv.DrawLine(x1, y1, x2, y2, 1, lineC)
	}
}
