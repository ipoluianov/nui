package ui

import (
	"image/color"
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

// timeChartSource adapts a TimeChartDataSource to a ChartSource of a
// TimeScale: seconds since the Unix epoch
type timeChartSource struct {
	src TimeChartDataSource
}

func (s timeChartSource) GetData(from, to, group float64) []ChartSample {
	points := s.src.GetData(ChartTimeToTime(from, nil), ChartTimeToTime(to, nil), time.Duration(group*float64(time.Second)))
	res := make([]ChartSample, len(points))
	for i, p := range points {
		res[i] = ChartSample{X: ChartTime(p.DT), First: p.First, Last: p.Last, High: p.High, Low: p.Low, HasGood: p.HasGood, HasBad: p.HasBad}
	}
	return res
}

type TimeChartSeriesType int

const (
	TimeChartSeriesLine TimeChartSeriesType = iota
	TimeChartSeriesCandles
)

// TimeChartSeries is a series of an area: a line or candles drawn from a
// TimeChartDataSource
type TimeChartSeries struct {
	area       *TimeChartArea
	name       string
	source     TimeChartDataSource
	seriesType TimeChartSeriesType
	// cs is the series of the chart's pane that draws it
	cs ChartSeries
}

func (c *TimeChartSeries) Name() string {
	return c.name
}

func (c *TimeChartSeries) Source() TimeChartDataSource {
	return c.source
}

// SetColor fixes the series color regardless of the theme.
func (c *TimeChartSeries) SetColor(col color.Color) {
	c.cs.base().SetColor(colorToRGBA(col))
}

// SetPaletteColor uses color number index of the chart palette, which has
// a dark and a light variant, so the series stays readable in both themes.
func (c *TimeChartSeries) SetPaletteColor(index int) {
	c.cs.base().SetPaletteColor(index)
}

// SetType draws the series as a line or as candles
func (c *TimeChartSeries) SetType(seriesType TimeChartSeriesType) {
	if seriesType == c.seriesType {
		return
	}
	c.seriesType = seriesType
	old := c.cs.base()
	c.cs = c.area.makeSeries(c.name, c.source, seriesType)
	b := c.cs.base()
	b.col, b.palette, b.paletteSet, b.pane = old.col, old.palette, true, old.pane
	pane := c.area.pane
	for i, s := range pane.series {
		if s.base() == old {
			pane.series[i] = c.cs
		}
	}
	c.area.chart.Update()
}

// Series returns the series of the chart that draws this one, for the
// settings TimeChartSeries has no methods for (a *LineSeries or a
// *CandleSeries)
func (c *TimeChartSeries) Series() ChartSeries { return c.cs }

// TimeChartArea is one horizontal band of the chart with its own Y axis.
// All areas share the time axis.
type TimeChartArea struct {
	chart   *TimeChart
	pane    *ChartPane
	series  []*TimeChartSeries
	markers []TimeChartMarker
}

// Pane returns the pane of the chart the area is: for its axes, layers and
// annotations
func (c *TimeChartArea) Pane() *ChartPane { return c.pane }

func (c *TimeChartArea) makeSeries(name string, source TimeChartDataSource, t TimeChartSeriesType) ChartSeries {
	src := timeChartSource{source}
	if t == TimeChartSeriesCandles {
		cs := NewCandleSeries(name)
		cs.source, cs.memory = src, nil
		return cs
	}
	ls := NewLineSeries(name)
	ls.source, ls.memory = src, nil
	ls.width = 1
	ls.markers = ChartMarkersNever
	return ls
}

func (c *TimeChartArea) AddSeries(name string, source TimeChartDataSource) *TimeChartSeries {
	s := &TimeChartSeries{area: c, name: name, source: source}
	s.cs = c.makeSeries(name, source, TimeChartSeriesLine)
	c.pane.AddSeries(s.cs)
	c.series = append(c.series, s)
	return s
}

func (c *TimeChartArea) Series() []*TimeChartSeries {
	return c.series
}

// SetYRange fixes the Y range of the area (also set by Ctrl + right drag)
func (c *TimeChartArea) SetYRange(yMin, yMax float64) {
	if yMax < yMin {
		yMin, yMax = yMax, yMin
	}
	if yMax-yMin < 1e-12 {
		return
	}
	c.pane.YAxis().SetRange(yMin, yMax)
}

// YRange returns the manual Y range; ok is false while the area auto-scales.
func (c *TimeChartArea) YRange() (yMin, yMax float64, ok bool) {
	a := c.pane.YAxis()
	return a.manMin, a.manMax, a.manual
}

// ResetYRange returns the area to auto-scaling over the visible data.
func (c *TimeChartArea) ResetYRange() {
	c.pane.YAxis().ResetRange()
}

const (
	timeChartMinSpan = 10 * time.Millisecond
	timeChartMaxSpan = 200 * 365 * 24 * time.Hour
)

// TimeChart draws time series in one or more stacked areas: a ChartView of
// a time axis shared by the areas, with the gestures of
// ChartProfileTimeSeries.
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
// wheel zooms time around the pointer. The gestures can be changed through
// Interaction, as for any ChartView.
type TimeChart struct {
	ChartView

	areas []*TimeChartArea

	defaultFrom time.Time
	defaultTo   time.Time

	onTimeRangeChanged func()
	onSelectionChanged func()

	// The range and the default view as last reported to onTimeRangeChanged
	lastFrom, lastTo float64
	lastDefault      bool
}

func NewTimeChart() *TimeChart {
	c := &TimeChart{}
	c.initChartView(NewTimeScale(time.Unix(0, 0), time.Unix(3600, 0)))
	c.SetTypeName("TimeChart")
	c.SetMinSize(DefaultButtonMinWidth, DefaultUiLineHeight)
	c.legend = ChartLegendTop
	c.in.UseProfile(ChartProfileTimeSeries)
	c.in.SetLimits(ChartLimits{MinSpanX: timeChartMinSpan.Seconds(), MaxSpanX: timeChartMaxSpan.Seconds()})

	to := time.Now()
	from := to.Add(-time.Hour)
	c.defaultFrom, c.defaultTo = from, to
	c.xAxis.SetDefaultRange(ChartTime(from), ChartTime(to))
	c.lastFrom, c.lastTo, c.lastDefault = ChartTime(from), ChartTime(to), true

	c.OnViewChanged.Connect(func(*ChartView) { c.reportTimeRange() })
	c.OnSelectionChanged.Connect(func(*ChartView) {
		if c.onSelectionChanged != nil {
			c.onSelectionChanged()
		}
	})
	return c
}

// reportTimeRange calls the handler of SetOnTimeRangeChanged when the
// visible range or the default view changed since the last call
func (c *TimeChart) reportTimeRange() {
	from, to := c.XRange()
	def := c.IsDefaultView()
	if from == c.lastFrom && to == c.lastTo && def == c.lastDefault {
		return
	}
	c.lastFrom, c.lastTo, c.lastDefault = from, to, def
	if c.onTimeRangeChanged != nil {
		c.onTimeRangeChanged()
	}
}

func (c *TimeChart) AddArea() *TimeChartArea {
	var pane *ChartPane
	if len(c.areas) == 0 {
		pane = c.panes[0]
	} else {
		pane = c.AddPane()
	}
	a := &TimeChartArea{chart: c, pane: pane}
	c.areas = append(c.areas, a)
	c.Update()
	return a
}

func (c *TimeChart) Areas() []*TimeChartArea {
	return c.areas
}

// RemoveAllAreas removes the areas with their series and markers
func (c *TimeChart) RemoveAllAreas() {
	c.areas = nil
	p := c.panes[0]
	p.series = nil
	p.under, p.over = nil, nil
	p.annotations = nil
	p.YAxis().manual = false
	c.panes = c.panes[:1]
	c.nextPalette = 0
	c.Update()
}

func (c *TimeChart) TimeRange() (from, to time.Time) {
	f, t := c.XRange()
	return ChartTimeToTime(f, nil), ChartTimeToTime(t, nil)
}

// SetTimeRange shows the given range. The chart leaves the default view
// until ResetTimeRange, ResetZoom or zooming back to a level that was the
// default view. The span is kept within 10 ms and 200 years.
func (c *TimeChart) SetTimeRange(from, to time.Time) {
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
	c.SetXRange(ChartTime(from), ChartTime(to))
}

// SetDefaultTimeRange sets the default range. The application owns it and
// may call this as often as it likes (for example on every new sample of a
// live feed): while the chart is in the default view the new range is shown
// at once; after the user has panned or zoomed it is only remembered, and
// returning to the default view (ResetZoom, ResetTimeRange or zooming all the
// way back) shows the default range as it is at that moment.
func (c *TimeChart) SetDefaultTimeRange(from, to time.Time) {
	c.defaultFrom, c.defaultTo = from, to
	c.xAxis.SetDefaultRange(ChartTime(from), ChartTime(to))
	if c.IsDefaultView() {
		c.reportTimeRange()
	}
}

func (c *TimeChart) DefaultTimeRange() (from, to time.Time) {
	return c.defaultFrom, c.defaultTo
}

// ResetTimeRange switches to the default view, keeping the Y ranges.
func (c *TimeChart) ResetTimeRange() {
	c.xAxis.manual = false
	c.viewChanged()
}

func (c *TimeChart) SetOnTimeRangeChanged(f func()) {
	c.onTimeRangeChanged = f
}

// Selection returns the selected time range; ok is false when nothing is
// selected
func (c *TimeChart) Selection() (from, to time.Time, ok bool) {
	f, t, ok := c.ChartView.Selection()
	return ChartTimeToTime(f, nil), ChartTimeToTime(t, nil), ok
}

func (c *TimeChart) SetSelection(from, to time.Time) {
	c.ChartView.SetSelection(ChartTime(from), ChartTime(to))
}

func (c *TimeChart) SetOnSelectionChanged(f func()) {
	c.onSelectionChanged = f
}

// timeChartFormatDuration formats the length of a selection of a time axis,
// as precise as its size needs
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

// timeChartPixel returns the time a pixel of the plot takes, as of the
// last paint
func (c *TimeChart) timeChartPixel() time.Duration {
	if c.lastPlotW < 1 {
		return 0
	}
	from, to := c.XRange()
	return time.Duration((to - from) / c.lastPlotW * float64(time.Second))
}

// draw lays out and draws the chart on the canvas, as a paint does
func (c *TimeChart) draw(cnv *Canvas) {
	c.paint(cnv)
}
