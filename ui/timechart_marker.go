package ui

import (
	"image/color"
	"time"
)

// TimeChartMarker is a text note pinned to a point of an area: DT on the
// time axis and Value on the area's Y axis (for example the top of a peak).
// A nil Color uses the theme's marker color.
type TimeChartMarker struct {
	DT    time.Time
	Value float64
	Text  string
	Color color.Color
}

// AddMarker adds a marker to the area. Markers may be added in any order.
// Texts that would overlap one drawn before are skipped, so a zoomed-out
// view stays readable; zooming in reveals them.
func (c *TimeChartArea) AddMarker(m TimeChartMarker) {
	c.markers = append(c.markers, m)
	c.addLabel(m)
}

func (c *TimeChartArea) addLabel(m TimeChartMarker) {
	a := c.pane.Annotations().AddLabel(ChartTime(m.DT), m.Value, m.Text)
	if m.Color != nil {
		a.SetColor(colorToRGBA(m.Color))
	}
}

// SetMarkers replaces all markers of the area.
func (c *TimeChartArea) SetMarkers(markers []TimeChartMarker) {
	c.markers = append([]TimeChartMarker(nil), markers...)
	c.pane.Annotations().RemoveKind(ChartAnnotationLabel)
	for _, m := range c.markers {
		c.addLabel(m)
	}
}

func (c *TimeChartArea) Markers() []TimeChartMarker {
	return c.markers
}

func (c *TimeChartArea) ClearMarkers() {
	c.markers = nil
	c.pane.Annotations().RemoveKind(ChartAnnotationLabel)
}
