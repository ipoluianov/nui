package ui

import (
	"math"
)

// ChartAxisPosition is the side of the plot an axis is drawn on
type ChartAxisPosition int

const (
	ChartAxisBottom ChartAxisPosition = iota
	ChartAxisLeft
	ChartAxisRight
	ChartAxisTop
)

// ChartAxis is an axis of a chart: its scale, the side it is drawn on, its
// title, grid and visible range. The range is, in this order: the one set
// by SetRange or by the user's zoom (manual), the default one of
// SetDefaultRange (where a live chart shows the last minutes), or the one
// that fits the data of the series on it, with a margin.
type ChartAxis struct {
	view     *ChartView
	name     string
	position ChartAxisPosition
	scale    ChartScale
	title    string

	visible   bool
	grid      bool
	minorGrid bool

	manual         bool
	manMin, manMax float64
	hasDefault     bool
	defMin, defMax float64
	includeZero    bool
	// margin is the part of the data range added on each side when fitting
	margin     float64
	inverted   bool
	labelAngle float64
	// ribbon draws the dates under a time axis
	ribbon bool

	// The layout of the last paint: where the axis is along its direction
	// (start, length) and the band it is drawn in (bandX/Y/W/H), in logical
	// pixels of the widget; the ticks; whether the labels are drawn here
	start, length float64
	bandX, bandY  float64
	bandW, bandH  float64
	ticks         []ChartTick
	ribbonSpans   []ChartDateSpan
	drawLabels    bool
	laidOut       bool
}

func newChartAxis(view *ChartView, name string, pos ChartAxisPosition, scale ChartScale) *ChartAxis {
	a := &ChartAxis{view: view, name: name, position: pos, scale: scale, visible: true, grid: true, margin: 0.05}
	if pos == ChartAxisBottom || pos == ChartAxisTop {
		// The data reaches the left and the right edges
		a.margin = 0
	}
	if _, ok := scale.(*TimeScale); ok {
		a.ribbon = true
	}
	return a
}

func (a *ChartAxis) update() {
	if a.view != nil {
		a.view.Update()
	}
}

// Name returns the name the series refer to the axis by
func (a *ChartAxis) Name() string { return a.name }

// Position returns the side of the plot the axis is on
func (a *ChartAxis) Position() ChartAxisPosition { return a.position }

// SetPosition moves the axis to the side of the plot
func (a *ChartAxis) SetPosition(pos ChartAxisPosition) *ChartAxis {
	a.position = pos
	a.update()
	return a
}

// Horizontal reports whether the axis runs along the bottom or the top
func (a *ChartAxis) Horizontal() bool {
	return a.position == ChartAxisBottom || a.position == ChartAxisTop
}

// Scale returns the scale of the axis
func (a *ChartAxis) Scale() ChartScale { return a.scale }

// SetScale replaces the scale (a LogScale instead of the linear one); the
// range of the axis stays
func (a *ChartAxis) SetScale(s ChartScale) *ChartAxis {
	if s == nil {
		return a
	}
	min, max := a.scale.Domain()
	s.SetDomain(min, max)
	a.scale = s
	_, isTime := s.(*TimeScale)
	a.ribbon = isTime
	a.update()
	return a
}

// SetTitle sets the title drawn along the axis: "Pressure, kPa"
func (a *ChartAxis) SetTitle(title string) *ChartAxis {
	a.title = title
	a.update()
	return a
}

// Title returns the title of the axis
func (a *ChartAxis) Title() string { return a.title }

// SetVisible shows or hides the line, the ticks and the labels of the axis;
// its grid is drawn either way
func (a *ChartAxis) SetVisible(visible bool) *ChartAxis {
	a.visible = visible
	a.update()
	return a
}

// Visible reports whether the axis is drawn
func (a *ChartAxis) Visible() bool { return a.visible }

// SetGrid shows the grid lines of the major ticks, and of the minor ones
func (a *ChartAxis) SetGrid(major, minor bool) *ChartAxis {
	a.grid, a.minorGrid = major, minor
	a.update()
	return a
}

// SetIncludeZero keeps zero in the range that fits the data: bars and
// areas start at zero
func (a *ChartAxis) SetIncludeZero(include bool) *ChartAxis {
	a.includeZero = include
	a.update()
	return a
}

// SetMargin sets the part of the data range added on each side of a fitted
// range: 0.05 by default for value axes, 0 for the horizontal ones
func (a *ChartAxis) SetMargin(margin float64) *ChartAxis {
	a.margin = math.Max(0, margin)
	a.update()
	return a
}

// SetInverted makes the values grow the other way: down a vertical axis, to
// the left along a horizontal one (a depth, a ranking)
func (a *ChartAxis) SetInverted(inverted bool) *ChartAxis {
	a.inverted = inverted
	a.update()
	return a
}

// SetLabelAngle turns the labels by the angle in radians: -math.Pi/4 for
// the long category names under a column chart
func (a *ChartAxis) SetLabelAngle(angle float64) *ChartAxis {
	a.labelAngle = angle
	a.update()
	return a
}

// SetDateRibbon shows or hides the dates under a time axis
func (a *ChartAxis) SetDateRibbon(show bool) *ChartAxis {
	a.ribbon = show
	a.update()
	return a
}

// SetRange fixes the visible range of the axis (until ResetRange or the
// user's zoom back)
func (a *ChartAxis) SetRange(min, max float64) *ChartAxis {
	a.setManual(min, max)
	a.update()
	return a
}

func (a *ChartAxis) setManual(min, max float64) {
	if min > max {
		min, max = max, min
	}
	a.manual, a.manMin, a.manMax = true, min, max
	a.scale.SetDomain(min, max)
}

// ResetRange lets the axis show its default range or fit the data again
func (a *ChartAxis) ResetRange() *ChartAxis {
	a.manual = false
	a.update()
	return a
}

// Range returns the visible range; manual is true when it was set by
// SetRange or by the user's zoom
func (a *ChartAxis) Range() (min, max float64, manual bool) {
	if a.view != nil && !a.manual {
		a.view.refit()
	}
	min, max = a.scale.Domain()
	return min, max, a.manual
}

// SetDefaultRange sets the range the axis shows when it is not zoomed: a
// live chart moves it with every new sample to show the last minutes, and
// the user, having panned away into the history, comes back to it with Esc.
// Without a default range the axis fits the data.
func (a *ChartAxis) SetDefaultRange(min, max float64) *ChartAxis {
	if min > max {
		min, max = max, min
	}
	a.hasDefault, a.defMin, a.defMax = true, min, max
	a.update()
	return a
}

// DefaultRange returns the default range; ok is false when there is none
func (a *ChartAxis) DefaultRange() (min, max float64, ok bool) {
	return a.defMin, a.defMax, a.hasDefault
}

// ClearDefaultRange makes the axis fit the data when it is not zoomed
func (a *ChartAxis) ClearDefaultRange() *ChartAxis {
	a.hasDefault = false
	a.update()
	return a
}

// ToPixel returns the position of the value along the axis, in logical
// pixels of the widget (x for a horizontal axis, y for a vertical one), as
// of the last layout
func (a *ChartAxis) ToPixel(v float64) float64 {
	u := a.scale.ToUnit(v)
	if a.inverted {
		u = 1 - u
	}
	if a.Horizontal() {
		return a.start + u*a.length
	}
	return a.start + (1-u)*a.length
}

// FromPixel returns the value at the position along the axis
func (a *ChartAxis) FromPixel(p float64) float64 {
	if a.length <= 0 {
		return a.scale.FromUnit(0)
	}
	u := (p - a.start) / a.length
	if !a.Horizontal() {
		u = 1 - u
	}
	if a.inverted {
		u = 1 - u
	}
	return a.scale.FromUnit(u)
}

// Ticks returns the ticks of the last layout
func (a *ChartAxis) Ticks() []ChartTick { return a.ticks }

// fit sets the domain to the range of the data, with the margin, zero
// included if asked; the manual and the default ranges come first
func (a *ChartAxis) fit(min, max float64, ok bool) {
	switch {
	case a.manual:
		a.scale.SetDomain(a.manMin, a.manMax)
		return
	case a.hasDefault:
		a.scale.SetDomain(a.defMin, a.defMax)
		return
	}
	if !ok {
		if cs, isCat := a.scale.(*CategoryScale); isCat {
			cs.SetCategories(cs.categories)
			return
		}
		min, max = 0, 1
	}
	if a.includeZero {
		min, max = math.Min(min, 0), math.Max(max, 0)
	}
	switch s := a.scale.(type) {
	case *CategoryScale:
		s.SetDomain(math.Min(min, -0.5), math.Max(max, float64(len(s.categories))-0.5))
		s.fitted = true
		return
	case *LogScale:
		if min > 0 && max > 0 && a.margin > 0 {
			f := math.Pow(max/min, a.margin)
			min, max = min/f, max*f
		}
		s.SetDomain(min, max)
		return
	}
	if max-min < 1e-300 {
		d := math.Max(math.Abs(min)*0.1, 1)
		min, max = min-d, max+d
	}
	m := (max - min) * a.margin
	// A range from zero is not taken below it by the margin
	lo, hi := min-m, max+m
	if a.includeZero && min == 0 {
		lo = 0
	}
	if a.includeZero && max == 0 {
		hi = 0
	}
	a.scale.SetDomain(lo, hi)
}

// labelExtent returns how long the label is along the axis and how thick
// across it, turned by the label angle
func (a *ChartAxis) labelExtent(cnv *Canvas, label string, lineH float64) (along, across float64) {
	w, h := cnv.MeasureTextRotated(label, a.labelAngle)
	if a.labelAngle == 0 {
		tw := chartTextWidth(cnv, label)
		w, h = tw, lineH
	}
	if a.Horizontal() {
		return w, h
	}
	return h, w
}

// thickness returns how much room the axis takes across its direction:
// the ticks, the widest label, the date ribbon and the title
func (a *ChartAxis) thickness(cnv *Canvas, lineH float64) float64 {
	if !a.visible {
		return 0
	}
	t := float64(chartTickLen + 3)
	widest := 0.0
	for _, tk := range a.ticks {
		if tk.Minor {
			continue
		}
		_, across := a.labelExtent(cnv, tk.Label, lineH)
		widest = math.Max(widest, across)
	}
	if widest == 0 && a.Horizontal() {
		widest = lineH
	}
	t += widest
	if a.ribbon && a.Horizontal() && len(a.ribbonSpans) > 0 {
		t += lineH + 2
	}
	if a.title != "" {
		t += lineH + 4
	}
	return math.Ceil(t)
}

// chartTickLen is the length of a tick mark outside the plot
const chartTickLen = 4

// chartTextWidth returns the width of the text in the canvas' font
func chartTextWidth(cnv *Canvas, text string) float64 {
	w, _ := cnv.MeasureTextRotated(text, 0)
	return w
}
