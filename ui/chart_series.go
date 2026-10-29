package ui

import (
	"image/color"
	"math"
)

// ChartSeries is what a pane of a chart draws: a line, bars, candles, a pie.
// The chart calls its methods on every paint, in this order: Range for the
// horizontal axes that fit the data, Prepare with the visible range, Range
// for the vertical axes, Draw. Nearest finds the element under the pointer
// for the tooltip and the clicks.
//
// A series of an application's own embeds ChartSeriesBase, which gives it
// the name, the visibility, the axes and the color, and implements the rest.
type ChartSeries interface {
	Name() string
	Visible() bool
	SetVisible(visible bool)
	// AxisNames returns the names of the axes the series is drawn on, "x"
	// and "y" by default
	AxisNames() (x, y string)
	// Range returns the range of the data along the axis: of all of it for
	// the horizontal axis, of the prepared part for the vertical one
	Range(axis *ChartAxis, v *ChartViewport) (min, max float64, ok bool)
	// Prepare gets the data of the visible range from the source
	Prepare(v *ChartViewport)
	// Draw draws the series within the plot of the viewport
	Draw(cnv *Canvas, v *ChartViewport)
	// Nearest returns the element nearest to the point (px, py), in logical
	// pixels of the widget: only by x when byX is set (a tooltip of all the
	// series at the pointer's x); ok is false when there is none near
	Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool)
	// Legend returns the entries of the series in the legend
	Legend(th *ChartTheme) []ChartLegendItem

	base() *ChartSeriesBase
}

// ChartHit is an element of a series under the pointer: a point of a line,
// a bar, a slice of a pie
type ChartHit struct {
	Series ChartSeries
	// Index of the element in the data the series drew (a category, a
	// slice); -1 when it has none
	Index int
	// The element's values and where it is, in logical pixels of the widget
	X, Y   float64
	PX, PY float64
	// XLabel and YLabel are the values as the tooltip shows them; Name is
	// the series' or the slice's name
	XLabel, YLabel string
	Name           string
	Color          color.RGBA
	// Sample is the sample of a line or a candle
	Sample ChartSample
	// dist is the distance to the pointer, to pick the nearest series
	dist float64
}

// ChartLegendKind is how an entry of the legend shows its series
type ChartLegendKind int

const (
	ChartLegendLine ChartLegendKind = iota
	ChartLegendBox
	ChartLegendMarker
)

// ChartLegendItem is an entry of the legend
type ChartLegendItem struct {
	Name   string
	Color  color.RGBA
	Kind   ChartLegendKind
	Marker MarkerShape
	// Hidden draws the entry dimmed: its series or slice is hidden
	Hidden bool
	// Index is the slice of a pie the entry is for; -1 - the whole series
	Index int
}

// chartLegendToggler is a series whose legend entries are parts of it, the
// slices of a pie: a click on an entry hides the part, not the series
type chartLegendToggler interface {
	toggleLegendItem(index int)
}

// ChartViewport is the plot a series draws in on one paint: its axes, the
// rectangle inside them in logical pixels of the widget, the theme
type ChartViewport struct {
	Pane         *ChartPane
	XAxis, YAxis *ChartAxis
	// The plot: X, Y of the top left corner, W and H
	X, Y, W, H float64
	Theme      *ChartTheme
	FontFamily string
	FontSize   float64
}

// PX returns the x in pixels of the value of the horizontal axis
func (v *ChartViewport) PX(x float64) float64 { return v.XAxis.ToPixel(x) }

// PY returns the y in pixels of the value of the vertical axis
func (v *ChartViewport) PY(y float64) float64 { return v.YAxis.ToPixel(y) }

// DataX returns the value of the horizontal axis at the x in pixels
func (v *ChartViewport) DataX(px float64) float64 { return v.XAxis.FromPixel(px) }

// DataY returns the value of the vertical axis at the y in pixels
func (v *ChartViewport) DataY(py float64) float64 { return v.YAxis.FromPixel(py) }

// Group returns the part of the horizontal axis a pixel takes: the size of
// the buckets a source merges its data into; 0 when the axis is not linear
func (v *ChartViewport) Group() float64 {
	if v.XAxis == nil || v.W <= 0 {
		return 0
	}
	switch v.XAxis.scale.(type) {
	case *LinearScale, *TimeScale:
		min, max := v.XAxis.scale.Domain()
		return (max - min) / v.W
	}
	return 0
}

// XRange returns the visible range of the horizontal axis
func (v *ChartViewport) XRange() (min, max float64) {
	return v.XAxis.scale.Domain()
}

// ChartSeriesBase is the part every series has: the name, the visibility,
// the axes and the color. A series of an application's own embeds it.
type ChartSeriesBase struct {
	pane    *ChartPane
	name    string
	hidden  bool
	xAxis   string
	yAxis   string
	col     *color.RGBA
	palette int
	// paletteSet: the palette color was chosen, not given in the order of
	// adding
	paletteSet bool
}

func (b *ChartSeriesBase) base() *ChartSeriesBase { return b }

// Name returns the name of the series, shown in the legend and the tooltip
func (b *ChartSeriesBase) Name() string { return b.name }

// SetName renames the series
func (b *ChartSeriesBase) SetName(name string) {
	b.name = name
	b.update()
}

// Visible reports whether the series is drawn
func (b *ChartSeriesBase) Visible() bool { return !b.hidden }

// SetVisible shows or hides the series (a click on its legend entry does it)
func (b *ChartSeriesBase) SetVisible(visible bool) {
	b.hidden = !visible
	b.update()
}

// AxisNames returns the names of the axes of the series
func (b *ChartSeriesBase) AxisNames() (string, string) {
	x, y := b.xAxis, b.yAxis
	if x == "" {
		x = "x"
	}
	if y == "" {
		y = "y"
	}
	return x, y
}

// SetAxisNames draws the series on the axes of the names: "y2" for a second
// vertical axis on the right (ChartPane.AddAxis)
func (b *ChartSeriesBase) SetAxisNames(x, y string) {
	b.xAxis, b.yAxis = x, y
	b.update()
}

// SetColor fixes the color of the series, whatever the theme
func (b *ChartSeriesBase) SetColor(col color.RGBA) {
	b.col = &col
	b.update()
}

// SetPaletteColor colors the series with the color number i of the theme's
// palette: it has a light and a dark variant, so the series stays readable
// in both themes. Without either call the series get the palette's colors
// in the order they are added.
func (b *ChartSeriesBase) SetPaletteColor(i int) {
	b.col = nil
	b.palette = i
	b.paletteSet = true
	b.update()
}

// Color returns the color of the series in the theme
func (b *ChartSeriesBase) Color(th *ChartTheme) color.RGBA {
	if b.col != nil {
		return *b.col
	}
	return th.SeriesColor(b.palette)
}

// Pane returns the pane the series is in; nil before it is added to one
func (b *ChartSeriesBase) Pane() *ChartPane { return b.pane }

func (b *ChartSeriesBase) update() {
	if b.pane != nil && b.pane.view != nil {
		b.pane.view.Update()
	}
}

// Legend returns one entry of the series' color
func (b *ChartSeriesBase) legendItem(th *ChartTheme, kind ChartLegendKind) []ChartLegendItem {
	return []ChartLegendItem{{Name: b.name, Color: b.Color(th), Kind: kind, Hidden: b.hidden, Index: -1}}
}

// chartCrisp returns the coordinate on the middle of its pixel: a line a
// pixel wide there covers whole pixels
func chartCrisp(v float64) float64 {
	return math.Floor(v) + 0.5
}

// chartFinite reports whether the value is a number that can be drawn
func chartFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// nearestSampleIndex returns the index of the sample of the sorted slice
// whose X is nearest to x; -1 for an empty one
func nearestSampleIndex(samples []ChartSample, x float64) int {
	n := len(samples)
	if n == 0 {
		return -1
	}
	lo, hi := 0, n
	for lo < hi {
		m := (lo + hi) / 2
		if samples[m].X < x {
			lo = m + 1
		} else {
			hi = m
		}
	}
	switch {
	case lo == 0:
		return 0
	case lo == n:
		return n - 1
	}
	if x-samples[lo-1].X <= samples[lo].X-x {
		return lo - 1
	}
	return lo
}
