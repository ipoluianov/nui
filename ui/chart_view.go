package ui

import (
	"image/color"
	"math"
	"strconv"
	"time"
)

// ChartLegendMode is where a chart shows its legend
type ChartLegendMode int

const (
	// ChartLegendAuto shows it above each pane when the chart has two
	// series or more, and on the right of a pie
	ChartLegendAuto ChartLegendMode = iota
	ChartLegendTop
	ChartLegendRight
	ChartLegendNone
)

// ChartHoverMode is what a chart shows under the pointer
type ChartHoverMode int

const (
	// ChartHoverX shows the values of all the series of the pane at the
	// pointer's x, with a crosshair
	ChartHoverX ChartHoverMode = iota
	// ChartHoverNearest shows the element nearest to the pointer
	ChartHoverNearest
	// ChartHoverNone shows nothing
	ChartHoverNone
)

// ChartView is a chart: one or more panes stacked top to bottom, each with
// its axes and series, sharing the horizontal axis. It draws the axes, the
// grid, the legend and the tooltip, and lets the user pan, zoom and select
// with the mouse and the keyboard as its Interaction says.
//
// The ready-made charts (NewLineChart, NewBarChart, NewPieChart,
// NewXYChart) are a ChartView set up for their kind; all of them can be
// changed and added to through the same objects.
//
//	ch := ui.NewXYChart()
//	ch.Pane(0).AddLine("Signal").AddValues(3, 5, 4, 7, 6)
//	form.Panel().AddWidget(0, 0, ch)
type ChartView struct {
	Widget

	panes       []*ChartPane
	xAxis       *ChartAxis
	title       string
	legend      ChartLegendMode
	hoverMode   ChartHoverMode
	crosshair   bool
	fixedTheme  *ChartTheme
	theme       *ChartTheme
	nextPalette int
	in          *ChartInteraction
	tooltipFunc func(hits []ChartHit) []ChartTooltipRow
	// frameless draws no border around the widget and the plots
	frameless bool
	// paintFunc paints the chart: ChartView.paint, or a preset's own
	paintFunc func(cnv *Canvas)
	// renderW and renderH: the size of the picture Render paints, 0 - the widget's
	renderW, renderH int
	// The sync groups' callbacks of the pointer, and the x of the pointer
	// over another chart of a group
	syncPointer []func(x float64, in bool)
	hasSyncX    bool
	syncX       float64
	menuFunc    func(menu *ContextMenu, hits []ChartHit)

	// The pointer, the hits under it of the last paint
	mouseIn        bool
	mouseX, mouseY float64
	hits           []ChartHit
	hoverPane      *ChartPane

	// The selection of a range of the horizontal axis
	hasSel         bool
	selFrom, selTo float64

	zoomStack []chartViewState

	drag chartDrag

	// The layout of the last paint
	plotLeft, plotRight float64
	lastPlotW           float64
	laidOut             bool
	selHeader, selClose chartRect
	selCloseHover       bool
	legendRight         []legendHit
	rightLegendX        float64

	// OnViewChanged: the visible range of an axis changed, by the user or
	// by the code (pan, zoom, SetRange, ResetZoom)
	OnViewChanged Signal[*ChartView]
	// OnSelectionChanged: the selection was made, moved, resized or cleared
	OnSelectionChanged Signal[*ChartView]
	// OnHover: the element under the pointer changed; a hit with no series
	// when the pointer left the elements
	OnHover Signal[ChartHit]
	// OnPointClicked: the user clicked an element of a series (see
	// ChartActionPointClick)
	OnPointClicked Signal[ChartHit]
	// OnSeriesToggled: a series was hidden or shown from the legend
	OnSeriesToggled Signal[ChartSeries]
	// OnBeforeAction: the user is about to pan, zoom, select... A handler may
	// cancel it, or change its range (snap a selection to whole minutes)
	OnBeforeAction Signal[*ChartActionEvent]
}

// chartOutside is a series that draws by its plot, on the right of the
// axes: the color bar of a heat map
type chartOutside interface {
	outsideWidth(cnv *Canvas) float64
	drawOutside(cnv *Canvas, v *ChartViewport)
}

// chartRect is a rectangle in logical pixels of the widget
type chartRect struct{ x, y, w, h float64 }

func (r chartRect) contains(x, y float64) bool {
	return r.w > 0 && x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

// NewChartView makes a chart of one pane with a linear horizontal axis
func NewChartView() *ChartView {
	c := &ChartView{}
	c.initChartView(NewLinearScale(0, 1))
	return c
}

func (c *ChartView) initChartView(xScale ChartScale) {
	c.InitWidget()
	c.SetTypeName("ChartView")
	c.SetMinSize(120, 80)
	c.SetCanBeFocused(true)
	c.SetElevation(1)
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	c.SetProp("padding", 6)
	c.crosshair = true
	c.xAxis = newChartAxis(c, "x", ChartAxisBottom, xScale)
	c.panes = []*ChartPane{newChartPane(c)}
	c.in = newChartInteraction(c)
	c.in.UseProfile(ChartProfileScientific)

	c.paintFunc = c.paint
	c.SetOnPaint(func(cnv *Canvas) { c.paintFunc(cnv) })
	c.SetOnMouseDown(c.onMouseDown)
	c.SetOnMouseMove(c.onMouseMove)
	c.OnWheel.Connect(c.onWheel)
	c.SetOnMouseDblClick(c.onMouseDblClick)
	c.SetOnKeyDown(c.onKeyDown)
	c.SetOnMouseLeave(func() {
		c.notifyPointer(false)
		c.mouseIn = false
		c.selCloseHover = false
		c.setHits(nil)
		c.Update()
	})
}

// Update repaints the chart
func (c *ChartView) Update() {
	if c.form != nil {
		c.form.Update()
	}
}

// Interaction returns how the chart answers the mouse and the keyboard
func (c *ChartView) Interaction() *ChartInteraction { return c.in }

// XAxis returns the horizontal axis the panes share
func (c *ChartView) XAxis() *ChartAxis { return c.xAxis }

// Pane returns the pane number i, nil if there is none
func (c *ChartView) Pane(i int) *ChartPane {
	if i < 0 || i >= len(c.panes) {
		return nil
	}
	return c.panes[i]
}

// Panes returns the panes, top to bottom
func (c *ChartView) Panes() []*ChartPane { return c.panes }

// AddPane adds a pane under the others, sharing the horizontal axis
func (c *ChartView) AddPane() *ChartPane {
	p := newChartPane(c)
	c.panes = append(c.panes, p)
	c.Update()
	return p
}

// RemovePane removes the pane; the last one stays
func (c *ChartView) RemovePane(p *ChartPane) {
	if len(c.panes) < 2 {
		return
	}
	for i, o := range c.panes {
		if o == p {
			c.panes = append(c.panes[:i:i], c.panes[i+1:]...)
			c.Update()
			return
		}
	}
}

// SetTitle sets the title over the chart
func (c *ChartView) SetTitle(title string) *ChartView {
	c.title = title
	c.Update()
	return c
}

// SetLegend sets where the legend is shown
func (c *ChartView) SetLegend(mode ChartLegendMode) *ChartView {
	c.legend = mode
	c.Update()
	return c
}

// SetHoverMode sets what is shown under the pointer, and whether a
// crosshair follows it
func (c *ChartView) SetHoverMode(mode ChartHoverMode, crosshair bool) *ChartView {
	c.hoverMode = mode
	c.crosshair = crosshair
	c.Update()
	return c
}

// ChartTooltipRow is a row of a tooltip: a color mark (none when
// transparent), a name and a value
type ChartTooltipRow struct {
	Color color.RGBA
	Name  string
	Value string
}

// SetTooltipFunc sets the rows of the tooltip of the elements under the
// pointer; nil - the default rows: the x, then each series' value
func (c *ChartView) SetTooltipFunc(f func(hits []ChartHit) []ChartTooltipRow) *ChartView {
	c.tooltipFunc = f
	return c
}

// SetTheme fixes the colors of the chart; nil - the theme of the
// application's palette
func (c *ChartView) SetTheme(th *ChartTheme) *ChartView {
	c.fixedTheme = th
	c.Update()
	return c
}

// Hits returns the elements under the pointer
func (c *ChartView) Hits() []ChartHit {
	if c.laidOut {
		c.updateHits()
	}
	return c.hits
}

// legendMode returns where the legend of the pane is: the auto mode puts
// a pie's on the right and the others' above each pane, when there are
// two series or more
func (c *ChartView) legendMode() ChartLegendMode {
	if c.legend != ChartLegendAuto {
		return c.legend
	}
	n := 0
	for _, p := range c.panes {
		if p.noAxes() {
			return ChartLegendRight
		}
		n += len(p.series)
	}
	if n >= 2 {
		return ChartLegendTop
	}
	return ChartLegendNone
}

// xAxes returns the distinct horizontal axes of the panes
func (c *ChartView) xAxes() []*ChartAxis {
	var res []*ChartAxis
	for _, p := range c.panes {
		found := false
		for _, a := range res {
			if a == p.xAxis {
				found = true
			}
		}
		if !found {
			res = append(res, p.xAxis)
		}
	}
	return res
}

// fitXAxes fits the horizontal axes to the data of their series
func (c *ChartView) fitXAxes() {
	for _, ax := range c.xAxes() {
		lo, hi := math.Inf(1), math.Inf(-1)
		// The margin the series ask for: points are not cut at the edges
		margin := ax.margin
		for _, p := range c.panes {
			if p.xAxis != ax {
				continue
			}
			for _, s := range p.series {
				if !s.Visible() {
					continue
				}
				if x, _ := p.seriesAxes(s); x != ax {
					continue
				}
				if a, b, ok := s.Range(ax, nil); ok {
					lo, hi = math.Min(lo, a), math.Max(hi, b)
				}
				if m, ok := s.(interface{ chartXMargin() float64 }); ok {
					margin = math.Max(margin, m.chartXMargin())
				}
			}
		}
		saved := ax.margin
		ax.margin = margin
		ax.fit(lo, hi, lo <= hi)
		ax.margin = saved
	}
}

// paint lays out and draws the chart
func (c *ChartView) paint(cnv *Canvas) {
	th := c.fixedTheme
	if th == nil {
		th = CurrentChartTheme()
	}
	c.theme = th
	w, h := c.size()
	cnv.FillRect(0, 0, int(w), int(h), th.Background)
	if !c.frameless {
		cnv.SetColor(th.Border)
		cnv.DrawRect(0, 0, int(w), int(h))
	}

	fontFamily, fontSize := c.FontFamily(), c.FontSize()
	axisFont := math.Max(7, fontSize*0.85)
	cnv.SetFontFamily(fontFamily)
	cnv.SetFontSize(axisFont)
	_, lineH := cnv.MeasureTextRotated("0", 0)
	pad := float64(c.GetPropInt("padding", 6))

	top, bottom := pad, h-pad
	left, right := pad, w-pad
	if c.title != "" {
		cnv.SetFontSize(fontSize * 1.1)
		cnv.SetFontBold(true)
		_, th1 := cnv.MeasureTextRotated(c.title, 0)
		chartDrawLabel(cnv, c.title, w/2, top, HAlignCenter, VAlignTop, th.Title)
		cnv.SetFontBold(false)
		top += th1 + 4
		cnv.SetFontSize(axisFont)
	}
	legend := c.legendMode()
	c.legendRight = c.legendRight[:0]
	if legend == ChartLegendRight {
		lw := c.rightLegendWidth(cnv, th, lineH)
		right -= lw + 8
		c.rightLegendX = right + 8
	}

	// The horizontal axes: their range, and their ticks for the width of
	// the last paint, to know how thick they are
	c.fitXAxes()
	estW := c.lastPlotW
	if estW < 2 {
		estW = math.Max(2, right-left-60)
	}
	measure := func(s string) float64 { return chartTextWidth(cnv, s) }
	for _, ax := range c.xAxes() {
		c.layoutTicks(ax, cnv, estW, lineH, measure)
	}
	// Which pane draws the labels of each horizontal axis: the lowest one
	// for an axis at the bottom, the highest for one at the top
	bottomAxis, topAxis := map[*ChartAxis]*ChartPane{}, map[*ChartAxis]*ChartPane{}
	for _, p := range c.panes {
		if p.noAxes() {
			continue
		}
		if p.xAxis.position == ChartAxisTop {
			if _, ok := topAxis[p.xAxis]; !ok {
				topAxis[p.xAxis] = p
			}
		} else {
			bottomAxis[p.xAxis] = p
		}
	}

	// Vertical layout: each pane's share of the height, less the legend
	// strip above it and the horizontal axes of the panes that draw them
	const gap = 8.0
	avail := bottom - top - gap*float64(len(c.panes)-1)
	totalWeight := 0.0
	reserved := 0.0
	for _, p := range c.panes {
		totalWeight += p.weight
		reserved += c.paneReserved(p, cnv, lineH, legend, bottomAxis, topAxis)
	}
	share := avail - reserved
	if share < 20*float64(len(c.panes)) {
		c.laidOut = false
		return
	}
	y := top
	for _, p := range c.panes {
		p.legendH = 0
		if legend == ChartLegendTop && len(p.series) > 0 && !p.noAxes() {
			p.legendH = lineH + 6
		}
		p.legendY = y
		y += p.legendH
		if pp, ok := topAxis[p.xAxis]; ok && pp == p {
			y += p.xAxis.thickness(cnv, lineH)
		}
		p.y = y
		p.h = math.Floor(share * p.weight / totalWeight)
		y += p.h
		if pp, ok := bottomAxis[p.xAxis]; ok && pp == p {
			y += p.xAxis.thickness(cnv, lineH)
		}
		y += gap
	}

	// The data of the visible range, with the width of the last paint, and
	// the vertical axes fitted to it
	for _, p := range c.panes {
		p.x, p.w = left, estW
		c.xAxisGeometry(p.xAxis, left, estW)
		c.fitYAxes(p, cnv, lineH, measure)
	}

	// Horizontal layout: the axes on the left and on the right of all the
	// panes line up, so do their plots
	leftW, rightW := 0.0, 0.0
	for _, p := range c.panes {
		if p.noAxes() {
			continue
		}
		l, r := 0.0, 0.0
		for _, a := range p.axes {
			t := a.thickness(cnv, lineH)
			if a.position == ChartAxisRight {
				r += t
			} else if a.position == ChartAxisLeft {
				l += t
			}
		}
		for _, s := range p.series {
			if d, ok := s.(chartOutside); ok && s.Visible() {
				r += d.outsideWidth(cnv)
			}
		}
		leftW, rightW = math.Max(leftW, l), math.Max(rightW, r)
	}
	plotL := math.Round(left + leftW)
	plotR := math.Round(right - math.Max(rightW, 8))
	if plotR-plotL < 20 {
		c.laidOut = false
		return
	}
	c.plotLeft, c.plotRight = plotL, plotR
	plotW := plotR - plotL
	c.lastPlotW = plotW
	for _, ax := range c.xAxes() {
		c.xAxisGeometry(ax, plotL, plotW)
		c.layoutTicks(ax, cnv, plotW, lineH, measure)
	}
	for _, p := range c.panes {
		if p.noAxes() {
			p.x, p.w = left, right-left
			continue
		}
		p.x, p.w = plotL, plotW
		// The vertical axes outwards from the plot
		lx, rx := plotL, plotR
		for _, a := range p.axes {
			t := a.thickness(cnv, lineH)
			switch a.position {
			case ChartAxisLeft:
				a.bandX, a.bandW = lx-t, t
				lx -= t
			case ChartAxisRight:
				a.bandX, a.bandW = rx, t
				rx += t
			}
			a.bandY, a.bandH = p.y, p.h
		}
	}
	// The data again when the plot is much wider or narrower than guessed
	if math.Abs(plotW-estW) > estW*0.1 {
		for _, p := range c.panes {
			c.fitYAxes(p, cnv, lineH, measure)
		}
	}
	c.laidOut = true

	for _, p := range c.panes {
		c.paintPane(cnv, p, lineH)
		if p.legendH > 0 {
			c.drawTopLegend(cnv, p, lineH)
		}
	}
	for ax, p := range bottomAxis {
		ax.bandX, ax.bandW = plotL, plotW
		ax.bandY = p.y + p.h
		ax.bandH = ax.thickness(cnv, lineH)
		c.drawAxis(cnv, ax, lineH)
	}
	for ax, p := range topAxis {
		ax.bandX, ax.bandW = plotL, plotW
		ax.bandH = ax.thickness(cnv, lineH)
		ax.bandY = p.y - ax.bandH
		c.drawAxis(cnv, ax, lineH)
	}
	if legend == ChartLegendRight {
		c.drawRightLegend(cnv, th, top, lineH)
	}
	if len(c.zoomStack) > 0 && len(c.panes) > 0 {
		// At the right end of the legend strip, or in the corner of the plot
		p := c.panes[0]
		text := "Zoom level " + strconv.Itoa(len(c.zoomStack))
		tw := chartTextWidth(cnv, text)
		y := p.legendY + (p.legendH-lineH)/2
		if p.legendH == 0 {
			y = p.y + 3
			bg := pathPool.Get().(*Path)
			bg.Reset()
			bg.RoundedRect(p.x+p.w-tw-8, y-1, tw+6, lineH+2, 3)
			cnv.FillPath(bg, th.MarkerLabelBg)
			pathPool.Put(bg)
		}
		chartDrawLabel(cnv, text, p.x+p.w-5, y, HAlignRight, VAlignTop, th.Text)
	}
	c.drawSelectionHeader(cnv, lineH)
	c.updateHits()
	c.drawHover(cnv, lineH)
	c.drawSyncCrosshair(cnv)
}

// paneReserved returns the height the pane needs besides its plot
func (c *ChartView) paneReserved(p *ChartPane, cnv *Canvas, lineH float64, legend ChartLegendMode, bottomAxis, topAxis map[*ChartAxis]*ChartPane) float64 {
	r := 0.0
	if legend == ChartLegendTop && len(p.series) > 0 && !p.noAxes() {
		r += lineH + 6
	}
	if pp, ok := bottomAxis[p.xAxis]; ok && pp == p {
		r += p.xAxis.thickness(cnv, lineH)
	}
	if pp, ok := topAxis[p.xAxis]; ok && pp == p {
		r += p.xAxis.thickness(cnv, lineH)
	}
	return r
}

// xAxisGeometry places the horizontal axis along the plot
func (c *ChartView) xAxisGeometry(ax *ChartAxis, left, width float64) {
	ax.start, ax.length = left, width
}

// layoutTicks picks the ticks of the axis for its length
func (c *ChartView) layoutTicks(ax *ChartAxis, cnv *Canvas, length, lineH float64, measure func(string) float64) {
	m := measure
	if ax.labelAngle != 0 || !ax.Horizontal() {
		m = func(s string) float64 {
			along, _ := ax.labelExtent(cnv, s, lineH)
			return along
		}
	}
	if _, cat := ax.scale.(*CategoryScale); !ax.Horizontal() && ax.labelAngle == 0 && !cat {
		// The labels of a vertical axis are one line tall: spaced by two
		// lines or so, as many as an eye takes in, not as many as fit; on a
		// short axis closer, so it still has a few
		room := math.Max(lineH*1.2, math.Min(lineH*2, length/4-chartTickGap))
		m = func(string) float64 { return room }
	}
	ax.ticks = ax.scale.Ticks(length, m)
	ax.ribbonSpans = nil
	if ts, ok := ax.scale.(*TimeScale); ok && ax.ribbon && ax.Horizontal() {
		ax.ribbonSpans = ts.DateRibbon(length, m)
	}
}

// fitYAxes prepares the series of the pane for the visible range and fits
// its vertical axes to their data
func (c *ChartView) fitYAxes(p *ChartPane, cnv *Canvas, lineH float64, measure func(string) float64) {
	for _, s := range p.series {
		if !s.Visible() {
			continue
		}
		x, y := p.seriesAxes(s)
		y.start, y.length = p.y, p.h
		s.Prepare(p.viewport(x, y))
	}
	p.computeStacks()
	for _, a := range p.axes {
		a.start, a.length = p.y, p.h
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, s := range p.series {
			if !s.Visible() {
				continue
			}
			x, y := p.seriesAxes(s)
			if y != a {
				continue
			}
			if mn, mx, ok := s.Range(a, p.viewport(x, y)); ok {
				lo, hi = math.Min(lo, mn), math.Max(hi, mx)
			}
		}
		if lo <= hi {
			lo, hi = p.annotationRange(a, lo, hi)
		}
		a.fit(lo, hi, lo <= hi)
		c.layoutTicks(a, cnv, p.h, lineH, measure)
	}
}

// paintPane draws the grid, the series, the layers and the frame of the pane
func (c *ChartView) paintPane(cnv *Canvas, p *ChartPane, lineH float64) {
	th := c.theme
	px, py, pw, ph := p.x, p.y, p.w, p.h
	if th.PlotBackground != th.Background {
		cnv.FillRect(int(px), int(py), int(pw), int(ph), th.PlotBackground)
	}
	vp := p.viewport(p.xAxis, p.axes[0])
	if !p.noAxes() {
		c.drawGrid(cnv, p)
	}
	cnv.Save()
	cnv.TranslateAndClip(int(px), int(py), int(pw), int(ph))
	// The canvas is moved to the plot: the series draw in the widget's
	// coordinates, so the canvas goes back by as much without the clip
	cnv.translateBy(-int(px), -int(py))
	for _, l := range p.under {
		l.DrawLayer(cnv, vp)
	}
	for _, s := range p.series {
		if !s.Visible() {
			continue
		}
		x, y := p.seriesAxes(s)
		s.Draw(cnv, p.viewport(x, y))
	}
	for _, l := range p.over {
		l.DrawLayer(cnv, vp)
	}
	if !p.noAxes() && p.xAxis == c.xAxis {
		c.drawSelection(cnv, p)
	}
	c.drawDragBand(cnv, p)
	cnv.Restore()
	// What a series draws by the plot: the color bar of a heat map
	for _, s := range p.series {
		if d, ok := s.(chartOutside); ok && s.Visible() {
			x, y := p.seriesAxes(s)
			d.drawOutside(cnv, p.viewport(x, y))
		}
	}
	if p.noAxes() {
		return
	}
	// The frame of the plot and its vertical axes
	if !c.frameless {
		frame := pathPool.Get().(*Path)
		frame.Reset()
		frame.Rect(px-0.5, py-0.5, pw+1, ph+1)
		cnv.StrokePath(frame, StrokeStyle{Width: 1}, th.Axis)
		pathPool.Put(frame)
	}
	for _, a := range p.axes {
		if a.position == ChartAxisLeft || a.position == ChartAxisRight {
			c.drawAxis(cnv, a, lineH)
		}
	}
}

// drawGrid draws the grid lines of the axes of the pane that have them
func (c *ChartView) drawGrid(cnv *Canvas, p *ChartPane) {
	th := c.theme
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	for _, minor := range []bool{true, false} {
		path.Reset()
		for _, a := range append([]*ChartAxis{p.xAxis}, p.axes...) {
			if !a.grid || (minor && !a.minorGrid) {
				continue
			}
			for _, t := range a.ticks {
				if t.Minor != minor {
					continue
				}
				pos := a.ToPixel(t.Value)
				if !chartFinite(pos) {
					continue
				}
				if a.Horizontal() {
					x := chartCrisp(pos)
					if x < p.x || x > p.x+p.w {
						continue
					}
					path.MoveTo(x, p.y)
					path.LineTo(x, p.y+p.h)
				} else {
					y := chartCrisp(pos)
					if y < p.y || y > p.y+p.h {
						continue
					}
					path.MoveTo(p.x, y)
					path.LineTo(p.x+p.w, y)
				}
			}
		}
		col := th.Grid
		if minor {
			col = th.GridMinor
		}
		cnv.StrokePath(path, StrokeStyle{Width: 1}, col)
	}
}

// drawAxis draws the ticks, the labels, the date ribbon and the title of
// the axis in its band
func (c *ChartView) drawAxis(cnv *Canvas, a *ChartAxis, lineH float64) {
	if !a.visible {
		return
	}
	th := c.theme
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	path.Reset()
	for _, t := range a.ticks {
		pos := a.ToPixel(t.Value)
		if !chartFinite(pos) || pos < a.start-0.5 || pos > a.start+a.length+0.5 {
			continue
		}
		l := float64(chartTickLen)
		if t.Minor {
			l = 2
		}
		pos = chartCrisp(pos)
		switch a.position {
		case ChartAxisBottom:
			path.MoveTo(pos, a.bandY)
			path.LineTo(pos, a.bandY+l)
		case ChartAxisTop:
			path.MoveTo(pos, a.bandY+a.bandH)
			path.LineTo(pos, a.bandY+a.bandH-l)
		case ChartAxisLeft:
			path.MoveTo(a.bandX+a.bandW, pos)
			path.LineTo(a.bandX+a.bandW-l, pos)
		case ChartAxisRight:
			path.MoveTo(a.bandX, pos)
			path.LineTo(a.bandX+l, pos)
		}
	}
	cnv.StrokePath(path, StrokeStyle{Width: 1}, th.Axis)

	off := float64(chartTickLen + 3)
	for _, t := range a.ticks {
		if t.Minor || t.Label == "" {
			continue
		}
		pos := a.ToPixel(t.Value)
		if !chartFinite(pos) || pos < a.start-1 || pos > a.start+a.length+1 {
			continue
		}
		c.drawTickLabel(cnv, a, t.Label, pos, off, lineH)
	}
	if len(a.ribbonSpans) > 0 && a.Horizontal() {
		c.drawRibbon(cnv, a, lineH)
	}
	if a.title != "" {
		cnv.Save()
		cnv.SetColor(th.Title)
		switch a.position {
		case ChartAxisBottom:
			chartDrawLabel(cnv, a.title, a.start+a.length/2, a.bandY+a.bandH-lineH-1, HAlignCenter, VAlignTop, th.Title)
		case ChartAxisTop:
			chartDrawLabel(cnv, a.title, a.start+a.length/2, a.bandY+1, HAlignCenter, VAlignTop, th.Title)
		case ChartAxisLeft:
			cnv.SetHAlign(HAlignCenter)
			cnv.SetVAlign(VAlignTop)
			cnv.DrawTextRotated(a.bandX+1, a.start+a.length/2, -math.Pi/2, a.title)
		case ChartAxisRight:
			cnv.SetHAlign(HAlignCenter)
			cnv.SetVAlign(VAlignTop)
			cnv.DrawTextRotated(a.bandX+a.bandW-1, a.start+a.length/2, math.Pi/2, a.title)
		}
		cnv.Restore()
	}
}

// drawTickLabel writes the label of a tick at the position along the axis,
// off pixels away from the plot
func (c *ChartView) drawTickLabel(cnv *Canvas, a *ChartAxis, label string, pos, off, lineH float64) {
	col := c.theme.Text
	if a.labelAngle != 0 {
		cnv.Save()
		cnv.SetColor(col)
		switch a.position {
		case ChartAxisBottom:
			cnv.SetHAlign(HAlignRight)
			cnv.SetVAlign(VAlignTop)
			if a.labelAngle > 0 {
				cnv.SetHAlign(HAlignLeft)
			}
			cnv.DrawTextRotated(pos, a.bandY+off, a.labelAngle, label)
		case ChartAxisTop:
			cnv.SetHAlign(HAlignLeft)
			cnv.SetVAlign(VAlignBottom)
			cnv.DrawTextRotated(pos, a.bandY+a.bandH-off, a.labelAngle, label)
		case ChartAxisLeft:
			cnv.SetHAlign(HAlignRight)
			cnv.SetVAlign(VAlignCenter)
			cnv.DrawTextRotated(a.bandX+a.bandW-off, pos, a.labelAngle, label)
		case ChartAxisRight:
			cnv.SetHAlign(HAlignLeft)
			cnv.SetVAlign(VAlignCenter)
			cnv.DrawTextRotated(a.bandX+off, pos, a.labelAngle, label)
		}
		cnv.Restore()
		return
	}
	switch a.position {
	case ChartAxisBottom:
		// Kept within the axis: the first and the last labels do not stick out
		tw := chartTextWidth(cnv, label)
		x := math.Max(a.start-off, math.Min(pos, a.start+a.length+off-tw/2))
		x = math.Max(x, a.start-off+tw/2)
		chartDrawLabel(cnv, label, x, a.bandY+off-2, HAlignCenter, VAlignTop, col)
	case ChartAxisTop:
		chartDrawLabel(cnv, label, pos, a.bandY+a.bandH-off-lineH+2, HAlignCenter, VAlignTop, col)
	case ChartAxisLeft:
		chartDrawLabel(cnv, label, a.bandX+a.bandW-off, pos-lineH/2, HAlignRight, VAlignTop, col)
	case ChartAxisRight:
		chartDrawLabel(cnv, label, a.bandX+off, pos-lineH/2, HAlignLeft, VAlignTop, col)
	}
}

// drawRibbon draws the dates under the ticks of a time axis: each centered
// on the visible part of its day (month, year), the days parted by a short
// line, so a range across midnight shows both dates
func (c *ChartView) drawRibbon(cnv *Canvas, a *ChartAxis, lineH float64) {
	y := a.bandY + chartTickLen + 3 + lineH + 1
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	path.Reset()
	for i, sp := range a.ribbonSpans {
		x1, x2 := a.ToPixel(sp.Start), a.ToPixel(sp.End)
		if i > 0 {
			bx := chartCrisp(x1)
			path.MoveTo(bx, y)
			path.LineTo(bx, y+lineH)
		}
		tw := chartTextWidth(cnv, sp.Label)
		if x2-x1 < tw+8 {
			continue
		}
		x := math.Max(x1+4, math.Min((x1+x2)/2-tw/2, x2-4-tw))
		chartDrawLabel(cnv, sp.Label, x, y, HAlignLeft, VAlignTop, c.theme.Text)
	}
	cnv.StrokePath(path, StrokeStyle{Width: 1}, c.theme.Axis)
}

// chartDrawLabel writes the line of text with the point of the alignment at
// (x, y): the text as other widgets draw it, on whole pixels
func chartDrawLabel(cnv *Canvas, text string, x, y float64, h HAlign, v VAlign, col color.RGBA) {
	w, th := cnv.MeasureTextRotated(text, 0)
	switch h {
	case HAlignCenter:
		x -= w / 2
	case HAlignRight:
		x -= w
	}
	switch v {
	case VAlignCenter:
		y -= th / 2
	case VAlignBottom:
		y -= th
	}
	cnv.Save()
	cnv.SetColor(col)
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignTop)
	cnv.DrawText(int(math.Round(x)), int(math.Round(y)), int(math.Ceil(w))+2, int(math.Ceil(th))+1, text)
	cnv.Restore()
}

// drawTopLegend draws the entries of the pane's series in the strip above it
func (c *ChartView) drawTopLegend(cnv *Canvas, p *ChartPane, lineH float64) {
	p.legendItems = p.legendItems[:0]
	x := p.x
	y := p.legendY + (p.legendH-lineH)/2
	for _, s := range p.series {
		for _, it := range s.Legend(c.theme) {
			tw := chartTextWidth(cnv, it.Name)
			if x+tw+14 > p.x+p.w && x > p.x {
				return
			}
			c.drawLegendEntry(cnv, it, x, y, lineH)
			p.legendItems = append(p.legendItems, legendHit{x: x, y: y, w: tw + 14, h: lineH, series: s, index: it.Index})
			x += tw + 26
		}
	}
}

// drawLegendEntry draws the mark and the name of an entry at (x, y)
func (c *ChartView) drawLegendEntry(cnv *Canvas, it ChartLegendItem, x, y, lineH float64) {
	col := it.Color
	text := c.theme.Text
	if it.Hidden {
		col = MixColors(c.theme.Background, col, 0.3)
		text = MixColors(c.theme.Background, text, 0.45)
	}
	cy := y + lineH/2
	switch it.Kind {
	case ChartLegendLine:
		p := pathPool.Get().(*Path)
		p.Reset()
		p.MoveTo(x, cy)
		p.LineTo(x+10, cy)
		cnv.StrokePath(p, StrokeStyle{Width: 2, Cap: LineCapRound}, col)
		pathPool.Put(p)
	case ChartLegendMarker:
		cnv.DrawMarker(it.Marker, x+5, cy, 8, col, nil)
	default:
		p := pathPool.Get().(*Path)
		p.Reset()
		p.RoundedRect(x+1, cy-4, 8, 8, 2)
		cnv.FillPath(p, col)
		pathPool.Put(p)
	}
	chartDrawLabel(cnv, it.Name, x+14, y, HAlignLeft, VAlignTop, text)
}

// rightLegendWidth returns the width of the legend on the right
func (c *ChartView) rightLegendWidth(cnv *Canvas, th *ChartTheme, lineH float64) float64 {
	w := 0.0
	for _, p := range c.panes {
		for _, s := range p.series {
			for _, it := range s.Legend(th) {
				w = math.Max(w, chartTextWidth(cnv, it.Name)+14)
			}
		}
	}
	cw, _ := c.size()
	return math.Min(w, cw/3)
}

// drawRightLegend draws the entries of all the series in a column on the right
func (c *ChartView) drawRightLegend(cnv *Canvas, th *ChartTheme, top, lineH float64) {
	y := top + 4
	for _, p := range c.panes {
		for _, s := range p.series {
			for _, it := range s.Legend(th) {
				if _, ch := c.size(); y+lineH > ch-4 {
					return
				}
				c.drawLegendEntry(cnv, it, c.rightLegendX, y, lineH)
				c.legendRight = append(c.legendRight, legendHit{x: c.rightLegendX, y: y, w: chartTextWidth(cnv, it.Name) + 14, h: lineH, series: s, index: it.Index})
				y += lineH + 4
			}
		}
	}
}

// drawSelection draws the selected range over the plot of the pane
func (c *ChartView) drawSelection(cnv *Canvas, p *ChartPane) {
	if !c.hasSel {
		return
	}
	th := c.theme
	x1, x2 := c.xAxis.ToPixel(c.selFrom), c.xAxis.ToPixel(c.selTo)
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	path.Reset()
	path.Rect(x1, p.y, x2-x1, p.h)
	cnv.FillPath(path, th.Selection)
	path.Reset()
	for _, x := range []float64{x1, x2} {
		path.MoveTo(chartCrisp(x), p.y)
		path.LineTo(chartCrisp(x), p.y+p.h)
	}
	cnv.StrokePath(path, StrokeStyle{Width: 1}, th.SelectionEdge)
}

// drawSelectionHeader draws a strip over the top of the selection with its
// length and a button that clears it
func (c *ChartView) drawSelectionHeader(cnv *Canvas, lineH float64) {
	c.selHeader, c.selClose = chartRect{}, chartRect{}
	if !c.hasSel || len(c.panes) == 0 {
		return
	}
	var p *ChartPane
	for _, q := range c.panes {
		if q.xAxis == c.xAxis && !q.noAxes() {
			p = q
			break
		}
	}
	if p == nil {
		return
	}
	th := c.theme
	x1, x2 := c.xAxis.ToPixel(c.selFrom), c.xAxis.ToPixel(c.selTo)
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if x2 < p.x || x1 > p.x+p.w {
		return
	}
	x1, x2 = math.Max(x1, p.x), math.Min(x2, p.x+p.w)
	hh := lineH + 6
	c.selHeader = chartRect{x1, p.y, x2 - x1, hh}
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	path.Reset()
	path.Rect(x1, p.y, x2-x1, hh)
	cnv.FillPath(path, th.SelectionHeader)
	size := hh - 4
	closeX := x2 - size - 2
	if closeX < x1+2 {
		return
	}
	c.selClose = chartRect{closeX, p.y + 2, size, size}
	if c.selCloseHover {
		path.Reset()
		path.Rect(closeX, p.y+2, size, size)
		cnv.FillPath(path, th.CloseHover)
	}
	pad := size / 4
	path.Reset()
	path.MoveTo(closeX+pad, p.y+2+pad)
	path.LineTo(closeX+size-pad, p.y+2+size-pad)
	path.MoveTo(closeX+size-pad, p.y+2+pad)
	path.LineTo(closeX+pad, p.y+2+size-pad)
	cnv.StrokePath(path, StrokeStyle{Width: 1.2}, th.HeaderText)
	text := c.selectionText()
	if tw := chartTextWidth(cnv, text); tw+8 <= closeX-x1 {
		chartDrawLabel(cnv, text, x1+4, p.y+(hh-lineH)/2, HAlignLeft, VAlignTop, th.HeaderText)
	}
}

// selectionText returns the length of the selection as the header shows it
func (c *ChartView) selectionText() string {
	d := math.Abs(c.selTo - c.selFrom)
	if _, ok := c.xAxis.scale.(*TimeScale); ok {
		return timeChartFormatDuration(time.Duration(d * float64(time.Second)))
	}
	min, max := c.xAxis.scale.Domain()
	return "Δ " + FormatChartNumber(d, niceStepAtLeast((max-min)/1000), false)
}

// drawDragBand draws the band (or the rectangle) being dragged to zoom or
// to select, over the plot of the pane
func (c *ChartView) drawDragBand(cnv *Canvas, p *ChartPane) {
	d := &c.drag
	if !d.active || !d.band {
		return
	}
	th := c.theme
	col := th.ZoomBand
	if d.action.kind == chartActSelectRange {
		col = th.SelectionDrag
	}
	x1, x2 := math.Min(d.startX, d.curX), math.Max(d.startX, d.curX)
	x1, x2 = math.Max(x1, p.x), math.Min(x2, p.x+p.w)
	y1, y2 := p.y, p.y+p.h
	if d.rect {
		if d.pane != p {
			return
		}
		y1, y2 = math.Max(math.Min(d.startY, d.curY), p.y), math.Min(math.Max(d.startY, d.curY), p.y+p.h)
	} else if p.xAxis != d.pane.xAxis {
		return
	}
	if x2 <= x1 {
		return
	}
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	path.Reset()
	path.Rect(x1, y1, x2-x1, y2-y1)
	cnv.FillPath(path, col)
}

// paneAt returns the pane whose plot is at y (or the nearest one in y)
func (c *ChartView) paneAt(x, y float64) *ChartPane {
	var best *ChartPane
	bestD := math.Inf(1)
	for _, p := range c.panes {
		d := 0.0
		if y < p.y {
			d = p.y - y
		} else if y > p.y+p.h {
			d = y - p.y - p.h
		}
		if d < bestD {
			best, bestD = p, d
		}
	}
	return best
}

// inPlot reports whether the point is on the plot of a pane
func (c *ChartView) inPlot(x, y float64) *ChartPane {
	for _, p := range c.panes {
		if x >= p.x && x < p.x+p.w && y >= p.y && y < p.y+p.h {
			return p
		}
	}
	return nil
}

// updateHits finds the elements under the pointer
func (c *ChartView) updateHits() {
	if !c.mouseIn || c.hoverMode == ChartHoverNone || c.drag.active && c.drag.band {
		c.setHits(nil)
		return
	}
	p := c.inPlot(c.mouseX, c.mouseY)
	if p == nil {
		c.setHits(nil)
		return
	}
	c.hoverPane = p
	byX := c.hoverMode == ChartHoverX && !p.noAxes()
	var hits []ChartHit
	for _, s := range p.series {
		if !s.Visible() {
			continue
		}
		x, y := p.seriesAxes(s)
		h, ok := s.Nearest(p.viewport(x, y), c.mouseX, c.mouseY, byX)
		if !ok {
			continue
		}
		hits = append(hits, h)
	}
	if !byX {
		// The nearest one, within 24 pixels
		best := -1
		for i, h := range hits {
			if h.dist <= 24 && (best < 0 || h.dist < hits[best].dist) {
				best = i
			}
		}
		if best < 0 {
			hits = nil
		} else {
			hits = hits[best : best+1]
		}
	} else {
		// The series whose nearest sample is far from the pointer's x
		// (no data there) are left out
		keep := hits[:0]
		for _, h := range hits {
			if h.dist <= 40 {
				keep = append(keep, h)
			}
		}
		hits = keep
	}
	c.setHits(hits)
}

// setHits keeps the hits under the pointer, pulls a pie's slice out and
// emits OnHover when the first hit changed
func (c *ChartView) setHits(hits []ChartHit) {
	same := len(hits) == len(c.hits)
	if same && len(hits) > 0 {
		same = hits[0].Series == c.hits[0].Series && hits[0].Index == c.hits[0].Index
	}
	c.hits = hits
	// The pie's slice under the pointer pulled out
	for _, p := range c.panes {
		for _, s := range p.series {
			if pie, ok := s.(*PieSeries); ok {
				idx := -1
				if len(hits) > 0 && hits[0].Series == ChartSeries(pie) {
					idx = pie.sliceAt(hits[0].PX, hits[0].PY)
				}
				if pie.setHover(idx) {
					c.Update()
				}
			}
		}
	}
	if !same {
		if len(hits) > 0 {
			c.OnHover.Emit(hits[0])
		} else {
			c.OnHover.Emit(ChartHit{Index: -1})
		}
	}
}

// drawHover draws the crosshair and the tooltip of the elements under the
// pointer
func (c *ChartView) drawHover(cnv *Canvas, lineH float64) {
	if len(c.hits) == 0 || c.hoverPane == nil {
		return
	}
	th := c.theme
	p := c.hoverPane
	if c.crosshair && !p.noAxes() {
		x := chartCrisp(c.hits[0].PX)
		path := pathPool.Get().(*Path)
		path.Reset()
		for _, q := range c.panes {
			if q.xAxis == p.xAxis && !q.noAxes() {
				path.MoveTo(x, q.y)
				path.LineTo(x, q.y+q.h)
			}
		}
		y := chartCrisp(c.mouseY)
		path.MoveTo(p.x, y)
		path.LineTo(p.x+p.w, y)
		cnv.StrokePath(path, StrokeStyle{Width: 1, Dash: []float64{3, 3}}, th.Crosshair)
		pathPool.Put(path)
		for _, h := range c.hits {
			// Only those within the plot: a point of another axis may be off it
			if h.PY >= p.y-1 && h.PY <= p.y+p.h+1 {
				cnv.DrawMarker(MarkerCircle, h.PX, h.PY, 7, h.Color, th.PlotBackground)
			}
		}
		// The value under the pointer, at the left edge of the plot
		if a := p.axes[0]; a.visible {
			c.drawAxisFlag(cnv, a.scale.Format(a.FromPixel(c.mouseY)), p.x+1, c.mouseY, HAlignLeft, lineH)
		}
	}
	rows := c.tooltipRows()
	if len(rows) == 0 {
		return
	}
	// The box: the widest row, near the pointer, kept inside the widget
	nameW, valW := 0.0, 0.0
	for _, r := range rows {
		nameW = math.Max(nameW, chartTextWidth(cnv, r.Name))
		valW = math.Max(valW, chartTextWidth(cnv, r.Value))
	}
	const padX, padY, mark = 8.0, 5.0, 12.0
	rowH := lineH + 3
	bw := padX*2 + mark + nameW + 12 + valW
	bh := padY*2 + rowH*float64(len(rows))
	bx, by := c.mouseX+16, c.mouseY+16
	cw, ch := c.size()
	if bx+bw > cw-4 {
		bx = c.mouseX - 16 - bw
	}
	if by+bh > ch-4 {
		by = c.mouseY - 16 - bh
	}
	bx, by = math.Max(4, bx), math.Max(4, by)
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	path.Reset()
	path.RoundedRect(bx+0.5, by+0.5, bw, bh, 4)
	cnv.FillPath(path, th.TooltipBg)
	cnv.StrokePath(path, StrokeStyle{Width: 1}, th.TooltipBorder)
	for i, r := range rows {
		y := by + padY + float64(i)*rowH
		if r.Color.A > 0 {
			path.Reset()
			path.RoundedRect(bx+padX, y+rowH/2-4, 8, 8, 2)
			cnv.FillPath(path, r.Color)
		}
		chartDrawLabel(cnv, r.Name, bx+padX+mark, y+1, HAlignLeft, VAlignTop, th.TooltipText)
		chartDrawLabel(cnv, r.Value, bx+bw-padX, y+1, HAlignRight, VAlignTop, th.TooltipText)
	}
}

// drawAxisFlag draws a value in a box on an axis, at the pointer
func (c *ChartView) drawAxisFlag(cnv *Canvas, text string, x, y float64, h HAlign, lineH float64) {
	tw := chartTextWidth(cnv, text)
	bx := x - tw - 6
	if h == HAlignLeft {
		bx = x
	}
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	path.Reset()
	path.RoundedRect(bx, y-lineH/2-1, tw+6, lineH+2, 2)
	cnv.FillPath(path, c.theme.Crosshair)
	chartDrawLabel(cnv, text, bx+3, y-lineH/2, HAlignLeft, VAlignTop, c.theme.Background)
}

// tooltipRows returns the rows of the tooltip of the hits
func (c *ChartView) tooltipRows() []ChartTooltipRow {
	if c.tooltipFunc != nil {
		return c.tooltipFunc(c.hits)
	}
	var rows []ChartTooltipRow
	first := c.hits[0]
	if _, pie := first.Series.(*PieSeries); pie {
		return []ChartTooltipRow{{Color: first.Color, Name: first.Name, Value: first.YLabel}}
	}
	rows = append(rows, ChartTooltipRow{Name: first.XLabel})
	for _, h := range c.hits {
		name := h.Name
		if name == "" {
			name = "Value"
		}
		rows = append(rows, ChartTooltipRow{Color: h.Color, Name: name, Value: h.YLabel})
	}
	return rows
}
