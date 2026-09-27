package ui

import (
	"image/color"
	"math"
	"strconv"
	"time"
)

// TimeChartPoint is one sample (or one aggregated bucket) of a time series.
// For a raw sample all four values are equal. For a downsampled bucket
// First/Last are the first and last values in the bucket and High/Low are its
// extremes, so a signal that jitters inside a bucket is still drawn as a
// vertical band instead of collapsing into a flat average. The same shape is
// an OHLC bar, so trading charts use it directly.
//
// HasGood and HasBad describe data quality: HasBad means the point (or some
// sample in the bucket) is bad - e.g. the sensor was disconnected - and
// HasGood means at least one sample carries a valid value. First/Last/High/
// Low only describe the good samples. A point without HasGood has no value:
// the line breaks there. Bad spans are drawn as a light reddish hatch.
// A point with neither flag is a gap - no data at all, e.g. the
// acquisition was stopped: the line breaks without the hatch.
type TimeChartPoint struct {
	DT      time.Time
	First   float64
	Last    float64
	High    float64
	Low     float64
	HasGood bool
	HasBad  bool
}

// HasValue reports whether First/Last/High/Low are meaningful.
func (p TimeChartPoint) HasValue() bool {
	return p.HasGood
}

// IsGap reports whether the point marks a break in the data.
func (p TimeChartPoint) IsGap() bool {
	return !p.HasGood && !p.HasBad
}

func NewTimeChartValue(dt time.Time, value float64) TimeChartPoint {
	return TimeChartPoint{DT: dt, First: value, Last: value, High: value, Low: value, HasGood: true}
}

// NewTimeChartBad returns a point with no value, marking bad quality at dt.
func NewTimeChartBad(dt time.Time) TimeChartPoint {
	return TimeChartPoint{DT: dt, HasBad: true}
}

// NewTimeChartGap returns a point that breaks the line at dt without
// marking bad quality - e.g. where data acquisition was stopped or started.
func NewTimeChartGap(dt time.Time) TimeChartPoint {
	return TimeChartPoint{DT: dt}
}

// TimeChartDataSource supplies the points of a series.
//
// GetData is called on every paint for the visible range [from, to].
// groupDuration is the time covered by one screen pixel: the source should
// aggregate its data into buckets of that size (see TimeChartDownsample) so
// that no more than about one point per pixel is returned. Points must be
// sorted by DT. Returning the nearest point on each side outside the range
// lets the line continue to the edges of the plot.
//
// A source that loads data asynchronously should return what it has now and
// call Form().Update() on the chart when more data arrives.
type TimeChartDataSource interface {
	GetData(from, to time.Time, groupDuration time.Duration) []TimeChartPoint
}

type TimeChartSeriesType int

const (
	TimeChartSeriesLine TimeChartSeriesType = iota
	TimeChartSeriesCandles
)

type TimeChartSeries struct {
	chart  *TimeChart
	name   string
	source TimeChartDataSource
	// color is set by SetColor; nil means paletteIndex into the current
	// theme's palette, so the color follows the light/dark theme.
	color        color.Color
	paletteIndex int
	seriesType   TimeChartSeriesType
}

func (c *TimeChartSeries) Name() string {
	return c.name
}

func (c *TimeChartSeries) Source() TimeChartDataSource {
	return c.source
}

// SetColor fixes the series color regardless of the theme.
func (c *TimeChartSeries) SetColor(col color.Color) {
	c.color = col
	c.chart.form.Update()
}

// SetPaletteColor uses color number index of the chart palette, which has
// a dark and a light variant, so the series stays readable in both themes.
func (c *TimeChartSeries) SetPaletteColor(index int) {
	c.color = nil
	c.paletteIndex = index
	c.chart.form.Update()
}

func (c *TimeChartSeries) themeColor(th *timeChartTheme) color.Color {
	if c.color != nil {
		return c.color
	}
	return th.palette[c.paletteIndex%len(th.palette)]
}

func (c *TimeChartSeries) SetType(seriesType TimeChartSeriesType) {
	c.seriesType = seriesType
	c.chart.form.Update()
}

// TimeChartArea is one horizontal band of the chart with its own Y axis.
// All areas share the time axis.
type TimeChartArea struct {
	chart  *TimeChart
	series []*TimeChartSeries

	markers       []TimeChartMarker
	markersSorted bool

	// Y range set by SetYRange or a Ctrl + drag zoom; auto-scaled otherwise.
	yManual bool
	yMin    float64
	yMax    float64

	// Geometry and Y range from the last paint, used to map mouse Y to a
	// value.
	top       int
	height    int
	drawnYMin float64
	drawnYMax float64
}

func (c *TimeChartArea) AddSeries(name string, source TimeChartDataSource) *TimeChartSeries {
	s := &TimeChartSeries{
		chart:        c.chart,
		name:         name,
		source:       source,
		paletteIndex: c.chart.seriesCount,
	}
	c.chart.seriesCount++
	c.series = append(c.series, s)
	c.chart.form.Update()
	return s
}

func (c *TimeChartArea) Series() []*TimeChartSeries {
	return c.series
}

func (c *TimeChartArea) SetYRange(yMin, yMax float64) {
	if yMax < yMin {
		yMin, yMax = yMax, yMin
	}
	if yMax-yMin < 1e-12 {
		return
	}
	c.yManual = true
	c.yMin = yMin
	c.yMax = yMax
	c.chart.form.Update()
}

// YRange returns the manual Y range; ok is false while the area auto-scales.
func (c *TimeChartArea) YRange() (yMin, yMax float64, ok bool) {
	return c.yMin, c.yMax, c.yManual
}

// ResetYRange returns the area to auto-scaling over the visible data.
func (c *TimeChartArea) ResetYRange() {
	c.yManual = false
	c.chart.form.Update()
}

func (c *TimeChartArea) valueAtY(y int) float64 {
	if c.height < 2 {
		return c.drawnYMin
	}
	k := float64(y-c.top) / float64(c.height-1)
	return c.drawnYMax - k*(c.drawnYMax-c.drawnYMin)
}

type timeChartDragMode int

const (
	timeChartDragNone timeChartDragMode = iota
	timeChartDragPan
	timeChartDragZoom
	timeChartDragZoomRect
	timeChartDragSelect
	timeChartDragSelMove
	timeChartDragSelResizeLeft
	timeChartDragSelResizeRight
)

// timeChartEdgeGrip is how close (in pixels) to a selection edge the pointer
// must be to resize it.
const timeChartEdgeGrip = 3

const (
	timeChartMinSpan = 10 * time.Millisecond
	timeChartMaxSpan = 200 * 365 * 24 * time.Hour
)

// timeChartZoomState is a view saved on the zoom stack.
type timeChartZoomState struct {
	from          time.Time
	to            time.Time
	isDefaultView bool
	areaY         []timeChartAreaYState
}

type timeChartAreaYState struct {
	manual bool
	yMin   float64
	yMax   float64
}

const timeChartMaxZoomDepth = 100

// TimeChart draws time series in one or more stacked areas.
//
// Mouse: left drag pans horizontally, right drag to the right zooms into the
// dragged time range, Ctrl + right drag to the right zooms into the dragged
// rectangle (time and the Y range of that area). Zooms stack: each one saves
// the current view and right drag to the left (with or without Ctrl) goes
// back one level, or to the default view when there is nothing to go back
// to (see ZoomBack, SetDefaultTimeRange and ResetZoom). A double click also
// goes back one level, Esc resets the zoom and clears the stack.
// Shift + left drag selects a range; dragging its header moves it, dragging
// its edges resizes it and the close button in the header clears it. The
// wheel zooms time around the pointer.
type TimeChart struct {
	Widget

	areas       []*TimeChartArea
	seriesCount int

	from time.Time
	to   time.Time

	defaultFrom   time.Time
	defaultTo     time.Time
	isDefaultView bool

	zoomStack []timeChartZoomState

	hasSelection bool
	selFrom      time.Time
	selTo        time.Time

	// Selection header and its close button from the last paint.
	selHeaderVisible bool
	selHeaderX       int
	selHeaderY       int
	selHeaderW       int
	selHeaderH       int
	selCloseVisible  bool
	selCloseX        int
	selCloseY        int
	selCloseSize     int
	selCloseHover    bool

	onTimeRangeChanged func()
	onSelectionChanged func()

	dragMode      timeChartDragMode
	dragButton    MouseButton
	dragStartX    int
	dragCurX      int
	dragStartY    int
	dragCurY      int
	dragArea      *TimeChartArea
	dragStartFrom time.Time
	dragStartTo   time.Time
	dragSelFrom   time.Time
	dragSelTo     time.Time

	// Theme of the current paint.
	theme *timeChartTheme

	// Plot geometry from the last paint, used to map mouse X to time.
	plotL      int
	plotW      int
	plotTop    int
	plotBottom int
}

type timeChartAreaLayout struct {
	area   *TimeChartArea
	data   [][]TimeChartPoint
	top    int
	height int
	yMin   float64
	yMax   float64
	ticks  []float64
	labels []string
}

func NewTimeChart() *TimeChart {
	var c TimeChart
	c.InitWidget()
	c.SetTypeName("TimeChart")
	c.SetMinSize(DefaultButtonMinWidth, DefaultUiLineHeight)
	c.SetMouseCursor(MouseCursorArrow)
	// Focusable so that Esc reaches the chart after a click on it.
	c.SetCanBeFocused(true)
	c.SetElevation(1)

	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.onMouseDownHandler)
	c.SetOnMouseMove(c.onMouseMoveHandler)
	c.SetOnMouseWheel(c.onMouseWheelHandler)
	c.SetOnMouseDblClick(c.onMouseDblClickHandler)
	c.SetOnKeyDown(c.onKeyDownHandler)
	c.SetOnMouseLeave(func() { c.selCloseHover = false })

	c.SetProp("padding", 6)

	c.SetXExpandable(true)
	c.SetYExpandable(true)

	c.theme = currentTimeChartTheme()
	c.to = time.Now()
	c.from = c.to.Add(-time.Hour)
	c.defaultFrom = c.from
	c.defaultTo = c.to
	c.isDefaultView = true

	return &c
}

func (c *TimeChart) AddArea() *TimeChartArea {
	a := &TimeChartArea{chart: c}
	c.areas = append(c.areas, a)
	c.form.Update()
	return a
}

func (c *TimeChart) Areas() []*TimeChartArea {
	return c.areas
}

func (c *TimeChart) RemoveAllAreas() {
	c.areas = nil
	c.form.Update()
}

func (c *TimeChart) TimeRange() (from, to time.Time) {
	return c.from, c.to
}

// SetTimeRange shows the given range. The chart leaves the default view
// until ResetTimeRange, ResetZoom or zooming back to a level that was the
// default view.
func (c *TimeChart) SetTimeRange(from, to time.Time) {
	c.setRange(from, to, false)
}

// IsDefaultView reports whether the chart shows the default range and
// follows SetDefaultTimeRange.
func (c *TimeChart) IsDefaultView() bool {
	return c.isDefaultView
}

func (c *TimeChart) setRange(from, to time.Time, isDefaultView bool) {
	if to.Before(from) {
		from, to = to, from
	}
	span := to.Sub(from)
	if span < timeChartMinSpan {
		center := from.Add(span / 2)
		from = center.Add(-timeChartMinSpan / 2)
		to = from.Add(timeChartMinSpan)
	}
	if span > timeChartMaxSpan {
		center := from.Add(span / 2)
		from = center.Add(-timeChartMaxSpan / 2)
		to = from.Add(timeChartMaxSpan)
	}
	if from.Equal(c.from) && to.Equal(c.to) && isDefaultView == c.isDefaultView {
		return
	}
	c.from = from
	c.to = to
	c.isDefaultView = isDefaultView
	if c.onTimeRangeChanged != nil {
		c.onTimeRangeChanged()
	}
	c.form.Update()
}

// SetDefaultTimeRange sets the default range. The application owns it and
// may call this as often as it likes (for example on every new sample of a
// live feed): while the chart is in the default view the new range is shown
// at once; after the user has panned or zoomed it is only remembered, and
// returning to the default view (ResetZoom, ResetTimeRange or zooming all the
// way back) shows the default range as it is at that moment.
func (c *TimeChart) SetDefaultTimeRange(from, to time.Time) {
	c.defaultFrom = from
	c.defaultTo = to
	if c.isDefaultView {
		c.setRange(from, to, true)
	}
}

func (c *TimeChart) DefaultTimeRange() (from, to time.Time) {
	return c.defaultFrom, c.defaultTo
}

// ResetTimeRange switches to the default view.
func (c *TimeChart) ResetTimeRange() {
	c.setRange(c.defaultFrom, c.defaultTo, true)
}

// ResetZoom returns to the default time range, auto-scales Y in every area
// and clears the zoom stack.
func (c *TimeChart) ResetZoom() {
	c.zoomStack = nil
	for _, a := range c.areas {
		a.yManual = false
	}
	c.ResetTimeRange()
	c.form.Update()
}

// ZoomDepth returns how many zoom levels ZoomBack can go back.
func (c *TimeChart) ZoomDepth() int {
	return len(c.zoomStack)
}

// pushZoom saves the current view before a zoom-in.
func (c *TimeChart) pushZoom() {
	st := timeChartZoomState{from: c.from, to: c.to, isDefaultView: c.isDefaultView}
	for _, a := range c.areas {
		st.areaY = append(st.areaY, timeChartAreaYState{manual: a.yManual, yMin: a.yMin, yMax: a.yMax})
	}
	c.zoomStack = append(c.zoomStack, st)
	if len(c.zoomStack) > timeChartMaxZoomDepth {
		c.zoomStack = c.zoomStack[1:]
	}
}

// ZoomBack restores the view saved by the last zoom-in. With an empty stack
// it resets to the default view.
func (c *TimeChart) ZoomBack() {
	if len(c.zoomStack) == 0 {
		c.ResetZoom()
		return
	}
	st := c.zoomStack[len(c.zoomStack)-1]
	c.zoomStack = c.zoomStack[:len(c.zoomStack)-1]
	for i, a := range c.areas {
		if i < len(st.areaY) {
			a.yManual = st.areaY[i].manual
			a.yMin = st.areaY[i].yMin
			a.yMax = st.areaY[i].yMax
		} else {
			a.yManual = false
		}
	}
	if st.isDefaultView {
		// The default range has probably moved on since the zoom-in.
		c.ResetTimeRange()
	} else {
		c.SetTimeRange(st.from, st.to)
	}
	c.form.Update()
}

func (c *TimeChart) SetOnTimeRangeChanged(f func()) {
	c.onTimeRangeChanged = f
}

func (c *TimeChart) Selection() (from, to time.Time, ok bool) {
	return c.selFrom, c.selTo, c.hasSelection
}

func (c *TimeChart) SetSelection(from, to time.Time) {
	if to.Before(from) {
		from, to = to, from
	}
	c.hasSelection = true
	c.selFrom = from
	c.selTo = to
	if c.onSelectionChanged != nil {
		c.onSelectionChanged()
	}
	c.form.Update()
}

func (c *TimeChart) ClearSelection() {
	if !c.hasSelection {
		return
	}
	c.hasSelection = false
	if c.onSelectionChanged != nil {
		c.onSelectionChanged()
	}
	c.form.Update()
}

func (c *TimeChart) SetOnSelectionChanged(f func()) {
	c.onSelectionChanged = f
}

func (c *TimeChart) span() time.Duration {
	s := c.to.Sub(c.from)
	if s < 1 {
		s = 1
	}
	return s
}

func (c *TimeChart) timeAtX(x int) time.Time {
	if c.plotW < 2 {
		return c.from
	}
	k := float64(x-c.plotL) / float64(c.plotW-1)
	return c.from.Add(time.Duration(k * float64(c.span())))
}

// xOfTime returns the X coordinate relative to the left edge of the plot.
func (c *TimeChart) xOfTime(t time.Time) int {
	return int(math.Round(c.xOfTimeF(t)))
}

// xOfTimeF returns the X coordinate relative to the left edge of the plot
// with the fraction of a pixel: the lines drawn with it move smoothly when
// the time range moves by less than a pixel.
func (c *TimeChart) xOfTimeF(t time.Time) float64 {
	if c.plotW < 2 {
		return 0
	}
	x := float64(t.Sub(c.from)) / float64(c.span()) * float64(c.plotW-1)
	return math.Max(-1e6, math.Min(1e6, x))
}

func (c *TimeChart) clampToPlot(x int) int {
	if x < c.plotL {
		return c.plotL
	}
	if x > c.plotL+c.plotW-1 {
		return c.plotL + c.plotW - 1
	}
	return x
}

func (c *TimeChart) areaAtY(y int) *TimeChartArea {
	for _, a := range c.areas {
		if y >= a.top && y < a.top+a.height {
			return a
		}
	}
	return nil
}

func (c *TimeChart) clampToArea(a *TimeChartArea, y int) int {
	if y < a.top {
		return a.top
	}
	if y > a.top+a.height-1 {
		return a.top + a.height - 1
	}
	return y
}

func (c *TimeChart) isOverSelectionClose(x, y int) bool {
	return c.hasSelection && c.selCloseVisible &&
		x >= c.selCloseX && x < c.selCloseX+c.selCloseSize &&
		y >= c.selCloseY && y < c.selCloseY+c.selCloseSize
}

func (c *TimeChart) isOverSelectionHeader(x, y int) bool {
	return c.hasSelection && c.selHeaderVisible &&
		x >= c.selHeaderX && x < c.selHeaderX+c.selHeaderW &&
		y >= c.selHeaderY && y < c.selHeaderY+c.selHeaderH
}

// selectionEdgeAt returns the resize mode for the selection edge under the
// pointer, or timeChartDragNone. The nearer edge wins when both are in reach.
func (c *TimeChart) selectionEdgeAt(x, y int) timeChartDragMode {
	if !c.hasSelection || y < c.plotTop || y > c.plotBottom {
		return timeChartDragNone
	}
	dl := x - (c.plotL + c.xOfTime(c.selFrom))
	dr := x - (c.plotL + c.xOfTime(c.selTo))
	dl = max(dl, -dl)
	dr = max(dr, -dr)
	if dr <= timeChartEdgeGrip && dr <= dl {
		return timeChartDragSelResizeRight
	}
	if dl <= timeChartEdgeGrip {
		return timeChartDragSelResizeLeft
	}
	return timeChartDragNone
}

// updateCursor shows what a left press at (x, y) would grab.
func (c *TimeChart) updateCursor(x, y int) {
	cursor := MouseCursorArrow
	switch {
	case c.isOverSelectionClose(x, y):
		cursor = MouseCursorPointer
	case c.selectionEdgeAt(x, y) != timeChartDragNone:
		cursor = MouseCursorResizeHor
	case c.isOverSelectionHeader(x, y):
		cursor = MouseCursorPointer
	}
	if c.MouseCursor() != cursor {
		c.SetMouseCursor(cursor)
	}
}

func (c *TimeChart) onMouseDownHandler(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if button == MouseButtonLeft {
		if c.isOverSelectionClose(x, y) {
			c.ClearSelection()
			return true
		}
		mode := c.selectionEdgeAt(x, y)
		if mode == timeChartDragNone && c.isOverSelectionHeader(x, y) {
			mode = timeChartDragSelMove
		}
		if mode != timeChartDragNone {
			c.dragMode = mode
			c.dragButton = button
			c.dragStartX = x
			c.dragSelFrom = c.selFrom
			c.dragSelTo = c.selTo
			return true
		}
	}
	switch button {
	case MouseButtonLeft:
		if mods.Shift {
			c.dragMode = timeChartDragSelect
		} else {
			c.dragMode = timeChartDragPan
		}
	case MouseButtonRight:
		if mods.Ctrl {
			c.dragArea = c.areaAtY(y)
			if c.dragArea == nil {
				return false
			}
			c.dragMode = timeChartDragZoomRect
		} else {
			c.dragMode = timeChartDragZoom
		}
	default:
		return false
	}
	c.dragButton = button
	c.dragStartX = x
	c.dragCurX = x
	c.dragStartY = y
	c.dragCurY = y
	c.dragStartFrom = c.from
	c.dragStartTo = c.to
	return true
}

func (c *TimeChart) onMouseMoveHandler(x int, y int, mods KeyModifiers) bool {
	switch c.dragMode {
	case timeChartDragPan:
		if c.plotW < 2 {
			return true
		}
		span := c.dragStartTo.Sub(c.dragStartFrom)
		shift := -time.Duration(float64(x-c.dragStartX) / float64(c.plotW-1) * float64(span))
		if shift == 0 {
			return true
		}
		c.SetTimeRange(c.dragStartFrom.Add(shift), c.dragStartTo.Add(shift))
		return true
	case timeChartDragZoom, timeChartDragZoomRect, timeChartDragSelect:
		c.dragCurX = x
		c.dragCurY = y
		c.form.Update()
		return true
	case timeChartDragSelMove:
		if c.plotW < 2 {
			return true
		}
		shift := time.Duration(float64(x-c.dragStartX) / float64(c.plotW-1) * float64(c.span()))
		c.SetSelection(c.dragSelFrom.Add(shift), c.dragSelTo.Add(shift))
		return true
	case timeChartDragSelResizeLeft:
		c.SetSelection(c.timeAtX(c.clampToPlot(x)), c.dragSelTo)
		return true
	case timeChartDragSelResizeRight:
		c.SetSelection(c.dragSelFrom, c.timeAtX(c.clampToPlot(x)))
		return true
	}
	c.updateCursor(x, y)
	if hover := c.isOverSelectionClose(x, y); hover != c.selCloseHover {
		c.selCloseHover = hover
		c.form.Update()
	}
	return false
}

// ProcessMouseUp is overridden because the form routes onMouseUp only to the
// widget pressed with the left button; a right-button zoom must end here too.
// Every widget receives ProcessMouseUp, so the release is seen even when it
// happens outside the chart.
func (c *TimeChart) ProcessMouseUp(button MouseButton, x int, y int, mods KeyModifiers, onlyForWidgetId string) bool {
	if c.dragMode != timeChartDragNone && button == c.dragButton {
		c.finishDrag(x, y)
	}
	return c.Widget.ProcessMouseUp(button, x, y, mods, onlyForWidgetId)
}

func (c *TimeChart) finishDrag(x int, y int) {
	mode := c.dragMode
	c.dragMode = timeChartDragNone

	x1 := c.clampToPlot(c.dragStartX)
	x2 := c.clampToPlot(x)
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	dragged := x2-x1 >= 3

	switch mode {
	case timeChartDragZoom:
		if dragged {
			if x < c.dragStartX {
				c.ZoomBack()
			} else {
				c.pushZoom()
				c.SetTimeRange(c.timeAtX(x1), c.timeAtX(x2))
			}
		}
	case timeChartDragZoomRect:
		a := c.dragArea
		c.dragArea = nil
		if !dragged {
			break
		}
		if x < c.dragStartX {
			c.ZoomBack()
			break
		}
		c.pushZoom()
		y1 := c.clampToArea(a, c.dragStartY)
		y2 := c.clampToArea(a, y)
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		if y2-y1 >= 3 {
			a.SetYRange(a.valueAtY(y2), a.valueAtY(y1))
		}
		c.SetTimeRange(c.timeAtX(x1), c.timeAtX(x2))
	case timeChartDragSelect:
		if dragged {
			c.SetSelection(c.timeAtX(x1), c.timeAtX(x2))
		}
	}
	c.form.Update()
}

// onMouseDblClickHandler goes one zoom level back. The selection header is
// left alone so that a quick double click on it (or on its close button)
// does not also change the zoom.
func (c *TimeChart) onMouseDblClickHandler(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft && button != MouseButtonRight {
		return false
	}
	if c.isOverSelectionHeader(x, y) {
		return true
	}
	c.ZoomBack()
	return true
}

// isZoomed reports whether there is anything for ResetZoom to undo.
func (c *TimeChart) isZoomed() bool {
	if len(c.zoomStack) > 0 || !c.isDefaultView {
		return true
	}
	for _, a := range c.areas {
		if a.yManual {
			return true
		}
	}
	return false
}

// onKeyDownHandler resets the zoom on Esc. When there is nothing to reset
// Esc is left unhandled, so it still reaches the form (e.g. to close a
// dialog).
func (c *TimeChart) onKeyDownHandler(key Key, mods KeyModifiers) bool {
	if key == KeyEsc && c.isZoomed() {
		c.ResetZoom()
		return true
	}
	return false
}

func (c *TimeChart) onMouseWheelHandler(deltaX, deltaY int) bool {
	if deltaY == 0 {
		return false
	}
	k := math.Pow(0.8, float64(deltaY))
	anchor := c.timeAtX(c.clampToPlot(c.lastMouseX))
	newFrom := anchor.Add(-time.Duration(float64(anchor.Sub(c.from)) * k))
	newTo := anchor.Add(time.Duration(float64(c.to.Sub(anchor)) * k))
	c.SetTimeRange(newFrom, newTo)
	return true
}

func timeChartNiceStep(span float64, maxTicks int) float64 {
	raw := span / float64(maxTicks)
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 5, 10} {
		if m*mag >= raw {
			return m * mag
		}
	}
	return 10 * mag
}

func (c *TimeChart) buildAreaLayout(al *timeChartAreaLayout, groupDuration time.Duration, lineH int) {
	first := true
	for i, s := range al.area.series {
		if s.source == nil {
			continue
		}
		points := s.source.GetData(c.from, c.to, groupDuration)
		al.data[i] = points
		for _, p := range points {
			if p.DT.Before(c.from) || p.DT.After(c.to) || !p.HasValue() {
				continue
			}
			if first {
				al.yMin, al.yMax = p.Low, p.High
				first = false
				continue
			}
			al.yMin = math.Min(al.yMin, p.Low)
			al.yMax = math.Max(al.yMax, p.High)
		}
	}
	if al.area.yManual {
		al.yMin, al.yMax = al.area.yMin, al.area.yMax
	} else {
		if first {
			al.yMin, al.yMax = 0, 1
		}
		if al.yMax-al.yMin < 1e-12 {
			al.yMin -= 1
			al.yMax += 1
		}
		margin := (al.yMax - al.yMin) * 0.05
		al.yMin -= margin
		al.yMax += margin
	}
	al.area.top = al.top
	al.area.height = al.height
	al.area.drawnYMin = al.yMin
	al.area.drawnYMax = al.yMax

	maxTicks := al.height / (lineH * 2)
	if maxTicks < 2 {
		maxTicks = 2
	}
	step := timeChartNiceStep(al.yMax-al.yMin, maxTicks)
	decimals := 0
	if step < 1 {
		decimals = int(math.Ceil(-math.Log10(step) - 1e-9))
	}
	for v := math.Ceil(al.yMin/step) * step; v <= al.yMax; v += step {
		al.ticks = append(al.ticks, v)
		al.labels = append(al.labels, strconv.FormatFloat(v, 'f', decimals, 64))
	}
}

func (al *timeChartAreaLayout) yOf(v float64) int {
	k := (v - al.yMin) / (al.yMax - al.yMin)
	return int(math.Round((1 - k) * float64(al.height-1)))
}

func (c *TimeChart) draw(cnv *Canvas) {
	w, h := c.Width(), c.Height()
	th := currentTimeChartTheme()
	c.theme = th
	cnv.FillRect(0, 0, w, h, th.background)
	cnv.SetColor(th.border)
	cnv.DrawRect(0, 0, w, h)

	fontFamily := c.FontFamily()
	fontSize := c.FontSize()
	cnv.SetFontFamily(fontFamily)
	cnv.SetFontSize(fontSize)
	_, lineH, err := MeasureText(fontFamily, fontSize, "0")
	if err != nil {
		return
	}

	axis := th.axis
	muted := th.text
	grid := th.grid

	outer := c.GetPropInt("padding", 6)
	const areaGap = 8

	// The time axis uses a smaller font: a row of times and, below it, a
	// ribbon of dates.
	axisFontSize := math.Max(7, fontSize*0.85)
	_, axisLineH, err := MeasureText(fontFamily, axisFontSize, "0")
	if err != nil {
		return
	}

	plotT := outer
	plotB := h - 1 - outer - 5 - axisLineH - 2 - axisLineH - 2
	legendH := axisLineH + 4
	if plotB-plotT < 10 {
		return
	}

	// The time per pixel depends on the plot width, which depends on the
	// Y labels, which depend on the data - so the data is fetched with the
	// width from the previous paint (or an estimate on the first one).
	estPlotW := c.plotW
	if estPlotW < 2 {
		estPlotW = w - 2*outer - 60
	}
	if estPlotW < 2 {
		return
	}
	groupDuration := c.span() / time.Duration(estPlotW)
	if groupDuration < 1 {
		groupDuration = 1
	}

	areas := make([]timeChartAreaLayout, len(c.areas))
	if len(areas) > 0 {
		// Each area is a legend strip followed by the plot, so the legend
		// never covers data.
		slotH := (plotB - plotT + 1 - areaGap*(len(areas)-1)) / len(areas)
		areaH := slotH - legendH
		if areaH < 10 {
			return
		}
		for i, a := range c.areas {
			al := &areas[i]
			al.area = a
			al.data = make([][]TimeChartPoint, len(a.series))
			al.top = plotT + i*(slotH+areaGap) + legendH
			al.height = areaH
			c.buildAreaLayout(al, groupDuration, lineH)
		}
	}

	maxLabelW := 20
	for _, al := range areas {
		for _, s := range al.labels {
			if tw, _, e := MeasureText(fontFamily, fontSize, s); e == nil && tw > maxLabelW {
				maxLabelW = tw
			}
		}
	}
	c.plotL = outer + maxLabelW + 8
	plotR := w - 1 - outer - 8
	c.plotW = plotR - c.plotL + 1
	if c.plotW < 2 {
		return
	}

	ticks, tickStep := c.timeTicks(fontFamily, axisFontSize)

	for _, al := range areas {
		cnv.Save()
		cnv.TranslateAndClip(c.plotL, al.top, c.plotW, al.height)

		for _, t := range ticks {
			x := c.xOfTime(t)
			cnv.DrawLine(x, 0, x, al.height, 1, grid)
		}
		for _, v := range al.ticks {
			y := al.yOf(v)
			cnv.DrawLine(0, y, c.plotW, y, 1, grid)
		}

		for i, s := range al.area.series {
			switch s.seriesType {
			case TimeChartSeriesCandles:
				c.drawCandles(cnv, &al, al.data[i], s.themeColor(th))
			default:
				c.drawLine(cnv, &al, al.data[i], s.themeColor(th))
			}
		}

		c.drawMarkers(cnv, &al, axisFontSize)
		cnv.SetFontSize(fontSize)
		c.drawSelectionOverlay(cnv, al.area, al.height)
		cnv.Restore()

		c.drawLegend(cnv, al.area, al.top-legendH, legendH, axisFontSize, muted)
		cnv.SetFontSize(fontSize)

		cnv.SetColor(axis)
		cnv.DrawRect(c.plotL-1, al.top-1, c.plotW+2, al.height+2)

		cnv.SetColor(muted)
		cnv.SetHAlign(HAlignRight)
		cnv.SetVAlign(VAlignCenter)
		for j, v := range al.ticks {
			y := al.top + al.yOf(v)
			cnv.DrawText(outer, y-lineH/2-1, c.plotL-outer-6, lineH+2, al.labels[j])
		}
	}

	if len(areas) > 0 {
		c.plotTop = areas[0].top
		c.plotBottom = areas[len(areas)-1].top + areas[len(areas)-1].height - 1
	}
	if depth := len(c.zoomStack); depth > 0 && len(areas) > 0 {
		// Right end of the first area's legend strip.
		cnv.SetFontSize(axisFontSize)
		cnv.SetColor(muted)
		cnv.SetHAlign(HAlignRight)
		cnv.SetVAlign(VAlignCenter)
		cnv.DrawText(c.plotL, areas[0].top-legendH, c.plotW, legendH, "Zoom level "+strconv.Itoa(depth))
		cnv.SetHAlign(HAlignLeft)
		cnv.SetVAlign(VAlignTop)
		cnv.SetFontSize(fontSize)
	}
	if len(areas) > 0 {
		c.drawSelectionHeader(cnv, areas[0].top, lineH)
	}

	cnv.SetFontSize(axisFontSize)
	cnv.SetColor(muted)
	cnv.SetVAlign(VAlignTop)
	cnv.SetHAlign(HAlignCenter)
	for _, t := range ticks {
		x := c.plotL + c.xOfTime(t)
		cnv.DrawLine(x, plotB+1, x, plotB+5, 1, axis)
		label := t.Format(tickStep.format)
		tw, _, e := MeasureText(fontFamily, axisFontSize, label)
		if e != nil {
			continue
		}
		cnv.DrawText(x-tw/2-2, plotB+6, tw+4, axisLineH+2, label)
	}
	cnv.SetHAlign(HAlignLeft)
	c.drawDateRibbon(cnv, tickStep.ribbon(), plotB+6+axisLineH+2, axisLineH, axisFontSize, axis)
	cnv.SetFontSize(fontSize)
	cnv.SetVAlign(VAlignTop)
}

// drawDateRibbon draws one label per calendar unit (day, month or year)
// under the tick row. Each label is centered on the visible part of its unit
// and kept inside it, and units are separated by a short line at their
// boundary, so when the range crosses midnight both dates are shown.
func (c *TimeChart) drawDateRibbon(cnv *Canvas, unit timeChartRibbon, y int, lineH int, fontSize float64, sepColor color.Color) {
	if unit == timeChartRibbonNone {
		return
	}
	t := unit.start(c.from)
	for i := 0; !t.After(c.to) && i < 1000; i++ {
		next := unit.next(t)
		bx := c.xOfTime(t)
		if bx > 0 && bx < c.plotW {
			cnv.DrawLine(c.plotL+bx, y, c.plotL+bx, y+lineH, 1, sepColor)
		}
		sx1 := max(bx, 0)
		sx2 := min(c.xOfTime(next)-1, c.plotW-1)
		label := unit.format(t)
		tw, _, err := MeasureText(c.FontFamily(), fontSize, label)
		if err == nil && sx2-sx1+1 >= tw+8 {
			x := (sx1+sx2)/2 - tw/2
			x = max(x, sx1+4)
			x = min(x, sx2-4-tw)
			cnv.DrawText(c.plotL+x, y, tw+2, lineH+1, label)
		}
		t = next
	}
}

func (c *TimeChart) drawLine(cnv *Canvas, al *timeChartAreaLayout, points []TimeChartPoint, col color.Color) {
	c.drawBadHatch(cnv, al, points)
	// X with the fraction of a pixel, so the line does not jump by whole
	// pixels as the chart scrolls; Y in whole pixels, so the flat parts of
	// the line stay sharp
	havePrev := false
	var prevX, prevY float64
	for _, p := range points {
		if !p.HasValue() {
			havePrev = false
			continue
		}
		x := c.xOfTimeF(p.DT)
		yHigh := float64(al.yOf(p.High))
		yLow := float64(al.yOf(p.Low))
		if yLow > yHigh || !havePrev {
			cnv.DrawLineF(x, yHigh, x, yLow, col)
		}
		if havePrev {
			cnv.DrawLineF(prevX, prevY, x, float64(al.yOf(p.First)), col)
		}
		prevX, prevY = x, float64(al.yOf(p.Last))
		havePrev = true
	}
}

// drawBadHatch hatches, over the full height of the area, every span of
// points with HasBad. A span runs from its first bad point to the next point
// without HasBad; a span at the end of the data is one point interval wide.
func (c *TimeChart) drawBadHatch(cnv *Canvas, al *timeChartAreaLayout, points []TimeChartPoint) {
	hatch := c.theme.badHatch
	for i := 0; i < len(points); {
		if !points[i].HasBad {
			i++
			continue
		}
		j := i
		for j < len(points) && points[j].HasBad {
			j++
		}
		x1 := c.xOfTime(points[i].DT)
		var x2 int
		switch {
		case j < len(points):
			x2 = c.xOfTime(points[j].DT)
		case j >= 2:
			x2 = 2*c.xOfTime(points[j-1].DT) - c.xOfTime(points[j-2].DT)
		default:
			x2 = x1 + 1
		}
		c.hatchRect(cnv, max(x1, 0), min(max(x2, x1+1), c.plotW), al.height, hatch)
		i = j
	}
}

// hatchRect draws diagonal lines every 6 px over columns [x1, x2) of the
// current plot. The pattern is anchored to the plot, so adjacent spans join
// up seamlessly.
func (c *TimeChart) hatchRect(cnv *Canvas, x1, x2, height int, col color.Color) {
	const period = 6
	tx, ty := cnv.TranslatedX(), cnv.TranslatedY()
	for x := x1; x < x2; x++ {
		for y := (period - x%period) % period; y < height; y += period {
			cnv.MixPixel(tx+x, ty+y, col)
		}
	}
}

func (c *TimeChart) drawCandles(cnv *Canvas, al *timeChartAreaLayout, points []TimeChartPoint, seriesColor color.Color) {
	c.drawBadHatch(cnv, al, points)

	up := c.theme.candleUp
	down := c.theme.candleDown

	minDx := 30
	for i := 1; i < len(points); i++ {
		dx := c.xOfTime(points[i].DT) - c.xOfTime(points[i-1].DT)
		if dx > 0 && dx < minDx {
			minDx = dx
		}
	}
	bodyW := minDx * 7 / 10
	if bodyW < 1 {
		bodyW = 1
	}

	for _, p := range points {
		if !p.HasValue() {
			continue
		}
		col := up
		if p.Last < p.First {
			col = down
		}
		x := c.xOfTime(p.DT)
		cnv.DrawLine(x, al.yOf(p.High), x, al.yOf(p.Low)+1, 1, col)
		yTop := al.yOf(math.Max(p.First, p.Last))
		yBot := al.yOf(math.Min(p.First, p.Last))
		cnv.FillRect(x-bodyW/2, yTop, bodyW, yBot-yTop+1, col)
	}
}

// drawSelectionOverlay draws the committed selection and the band (or the
// rectangle) being dragged, in the plot coordinates of the area.
func (c *TimeChart) drawSelectionOverlay(cnv *Canvas, area *TimeChartArea, height int) {
	if c.hasSelection {
		x1 := c.xOfTime(c.selFrom)
		x2 := c.xOfTime(c.selTo)
		cnv.FillRect(x1, 0, x2-x1+1, height, c.theme.selection)
		c.drawSelectionEdges(cnv, x1, x2, height)
	}
	// A leftward zoom drag means "zoom back", so it shows no band.
	zoomingIn := c.dragMode == timeChartDragZoom && c.dragCurX > c.dragStartX
	if zoomingIn || c.dragMode == timeChartDragSelect {
		x1 := c.clampToPlot(c.dragStartX) - c.plotL
		x2 := c.clampToPlot(c.dragCurX) - c.plotL
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		col := c.theme.zoomBand
		if c.dragMode == timeChartDragSelect {
			col = c.theme.selectionDrag
		}
		cnv.FillRect(x1, 0, x2-x1+1, height, col)
		if c.dragMode == timeChartDragSelect {
			c.drawSelectionEdges(cnv, x1, x2, height)
		}
	}
	if c.dragMode == timeChartDragZoomRect && c.dragArea == area && c.dragCurX > c.dragStartX {
		x1 := c.clampToPlot(c.dragStartX) - c.plotL
		x2 := c.clampToPlot(c.dragCurX) - c.plotL
		y1 := c.clampToArea(area, c.dragStartY) - area.top
		y2 := c.clampToArea(area, c.dragCurY) - area.top
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		cnv.FillRect(x1, y1, x2-x1+1, y2-y1+1, c.theme.zoomBand)
	}
}

// drawSelectionEdges outlines the left and right edges of a selection so
// they stay visible over busy data.
func (c *TimeChart) drawSelectionEdges(cnv *Canvas, x1, x2, height int) {
	edge := c.theme.selectionEdge
	cnv.DrawLine(x1, 0, x1, height, 1, edge)
	cnv.DrawLine(x2, 0, x2, height, 1, edge)
}

// drawSelectionHeader draws a translucent strip over the top of the
// selection with its duration and a close button that clears it.
func (c *TimeChart) drawSelectionHeader(cnv *Canvas, top int, lineH int) {
	c.selHeaderVisible = false
	c.selCloseVisible = false
	if !c.hasSelection || len(c.areas) == 0 {
		return
	}
	x1 := c.xOfTime(c.selFrom)
	x2 := c.xOfTime(c.selTo)
	if x2 < 0 || x1 > c.plotW-1 {
		return
	}
	x1 = max(x1, 0)
	x2 = min(x2, c.plotW-1)
	headerW := x2 - x1 + 1
	headerH := lineH + 6
	c.selHeaderVisible = true
	c.selHeaderX = c.plotL + x1
	c.selHeaderY = top
	c.selHeaderW = headerW
	c.selHeaderH = headerH

	cnv.Save()
	cnv.TranslateAndClip(c.plotL+x1, top, headerW, headerH)
	cnv.FillRect(0, 0, headerW, headerH, c.theme.selectionHeader)

	size := headerH - 4
	closeX := headerW - size - 2
	if closeX >= 2 {
		c.selCloseVisible = true
		c.selCloseX = c.plotL + x1 + closeX
		c.selCloseY = top + 2
		c.selCloseSize = size
		if c.selCloseHover {
			cnv.FillRect(closeX, 2, size, size, c.theme.closeHover)
		}
		pad := size / 4
		white := c.theme.headerText
		cnv.DrawLine(closeX+pad, 2+pad, closeX+size-pad, 2+size-pad, 1, white)
		cnv.DrawLine(closeX+size-pad, 2+pad, closeX+pad, 2+size-pad, 1, white)

		text := timeChartFormatDuration(c.selTo.Sub(c.selFrom))
		if tw, _, err := MeasureText(c.FontFamily(), c.FontSize(), text); err == nil && tw+8 <= closeX {
			cnv.SetColor(white)
			cnv.SetHAlign(HAlignLeft)
			cnv.SetVAlign(VAlignCenter)
			cnv.DrawText(4, 0, tw+2, headerH, text)
		}
	}
	cnv.Restore()
}

func timeChartFormatDuration(d time.Duration) string {
	switch {
	case d >= time.Hour:
		d = d.Round(time.Minute)
	case d >= time.Minute:
		d = d.Round(time.Second)
	case d >= time.Second:
		d = d.Round(time.Millisecond)
	default:
		d = d.Round(time.Microsecond)
	}
	return d.String()
}

// drawLegend draws the series names in the strip of the given height at y,
// above an area, clipped to the plot width.
func (c *TimeChart) drawLegend(cnv *Canvas, area *TimeChartArea, y int, height int, fontSize float64, textColor color.Color) {
	cnv.Save()
	cnv.TranslateAndClip(c.plotL, y, c.plotW, height)
	cnv.SetFontSize(fontSize)
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	const box = 8
	x := 0
	for _, s := range area.series {
		tw, _, err := MeasureText(c.FontFamily(), fontSize, s.name)
		if err != nil {
			continue
		}
		cnv.FillRect(x, (height-box)/2, box, box, s.themeColor(c.theme))
		x += box + 4
		cnv.SetColor(textColor)
		cnv.DrawText(x, 0, tw+2, height, s.name)
		x += tw + 14
	}
	cnv.Restore()
}

type timeChartTimeStep struct {
	d      time.Duration
	months int
	format string
}

// timeChartRibbon is the calendar unit of the date ribbon under the ticks:
// one unit coarser than the tick step, so tick labels plus the ribbon always
// give the full date and time.
type timeChartRibbon int

const (
	timeChartRibbonNone timeChartRibbon = iota
	timeChartRibbonDay
	timeChartRibbonMonth
	timeChartRibbonYear
)

func (st timeChartTimeStep) ribbon() timeChartRibbon {
	switch {
	case st.months >= 12:
		return timeChartRibbonNone
	case st.months > 0:
		return timeChartRibbonYear
	case st.d >= 24*time.Hour:
		return timeChartRibbonMonth
	}
	return timeChartRibbonDay
}

func (u timeChartRibbon) start(t time.Time) time.Time {
	y, m, d := t.Date()
	switch u {
	case timeChartRibbonMonth:
		return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
	case timeChartRibbonYear:
		return time.Date(y, 1, 1, 0, 0, 0, 0, t.Location())
	}
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func (u timeChartRibbon) next(t time.Time) time.Time {
	switch u {
	case timeChartRibbonMonth:
		return t.AddDate(0, 1, 0)
	case timeChartRibbonYear:
		return t.AddDate(1, 0, 0)
	}
	return t.AddDate(0, 0, 1)
}

func (u timeChartRibbon) format(t time.Time) string {
	switch u {
	case timeChartRibbonMonth:
		return t.Format("January 2006")
	case timeChartRibbonYear:
		return t.Format("2006")
	}
	return t.Format("2006-01-02")
}

var timeChartTimeSteps = []timeChartTimeStep{
	{time.Millisecond, 0, "15:04:05.000"},
	{2 * time.Millisecond, 0, "15:04:05.000"},
	{5 * time.Millisecond, 0, "15:04:05.000"},
	{10 * time.Millisecond, 0, "15:04:05.000"},
	{20 * time.Millisecond, 0, "15:04:05.000"},
	{50 * time.Millisecond, 0, "15:04:05.000"},
	{100 * time.Millisecond, 0, "15:04:05.000"},
	{200 * time.Millisecond, 0, "15:04:05.000"},
	{500 * time.Millisecond, 0, "15:04:05.000"},
	{time.Second, 0, "15:04:05"},
	{2 * time.Second, 0, "15:04:05"},
	{5 * time.Second, 0, "15:04:05"},
	{10 * time.Second, 0, "15:04:05"},
	{15 * time.Second, 0, "15:04:05"},
	{30 * time.Second, 0, "15:04:05"},
	{time.Minute, 0, "15:04"},
	{2 * time.Minute, 0, "15:04"},
	{5 * time.Minute, 0, "15:04"},
	{10 * time.Minute, 0, "15:04"},
	{15 * time.Minute, 0, "15:04"},
	{30 * time.Minute, 0, "15:04"},
	{time.Hour, 0, "15:04"},
	{2 * time.Hour, 0, "15:04"},
	{3 * time.Hour, 0, "15:04"},
	{6 * time.Hour, 0, "15:04"},
	{12 * time.Hour, 0, "15:04"},
	{24 * time.Hour, 0, "02"},
	{2 * 24 * time.Hour, 0, "02"},
	{7 * 24 * time.Hour, 0, "02"},
	{14 * 24 * time.Hour, 0, "02"},
	{0, 1, "Jan"},
	{0, 3, "Jan"},
	{0, 6, "Jan"},
	{0, 12, "2006"},
	{0, 24, "2006"},
	{0, 60, "2006"},
	{0, 120, "2006"},
	{0, 600, "2006"},
}

func (st timeChartTimeStep) approx() time.Duration {
	if st.months > 0 {
		return time.Duration(st.months) * 30 * 24 * time.Hour
	}
	return st.d
}

// timeTicks picks the smallest step whose labels fit and returns the tick
// times inside the visible range, aligned to local calendar boundaries.
func (c *TimeChart) timeTicks(fontFamily string, fontSize float64) ([]time.Time, timeChartTimeStep) {
	st := timeChartTimeSteps[len(timeChartTimeSteps)-1]
	for _, candidate := range timeChartTimeSteps {
		sample := time.Date(2000, 12, 28, 23, 59, 59, 999000000, time.Local).Format(candidate.format)
		tw, _, err := MeasureText(fontFamily, fontSize, sample)
		if err != nil {
			tw = 80
		}
		px := float64(candidate.approx()) / float64(c.span()) * float64(c.plotW)
		if px >= float64(tw+30) {
			st = candidate
			break
		}
	}

	var res []time.Time
	add := func(t time.Time) bool {
		if !t.Before(c.from) {
			res = append(res, t)
		}
		return len(res) < 500
	}

	loc := c.from.Location()
	switch {
	case st.months > 0:
		y, m, _ := c.from.Date()
		idx := y*12 + int(m) - 1
		t := time.Date(y, m, 1, 0, 0, 0, 0, loc).AddDate(0, -(idx % st.months), 0)
		for ; !t.After(c.to); t = t.AddDate(0, st.months, 0) {
			if !add(t) {
				break
			}
		}
	case st.d >= 24*time.Hour:
		days := int(st.d / (24 * time.Hour))
		y, m, d := c.from.Date()
		dayNum := int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
		t := time.Date(y, m, d, 0, 0, 0, 0, loc).AddDate(0, 0, -(((dayNum % days) + days) % days))
		for ; !t.After(c.to); t = t.AddDate(0, 0, days) {
			if !add(t) {
				break
			}
		}
	default:
		_, off := c.from.Zone()
		offD := time.Duration(off) * time.Second
		t := c.from.Add(offD).Truncate(st.d).Add(-offD)
		for ; !t.After(c.to); t = t.Add(st.d) {
			if !add(t) {
				break
			}
		}
	}
	return res, st
}
