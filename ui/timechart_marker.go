package ui

import (
	"image/color"
	"math"
	"sort"
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
func (c *TimeChartArea) AddMarker(m TimeChartMarker) {
	c.markers = append(c.markers, m)
	c.markersSorted = len(c.markers) < 2 || (c.markersSorted && !m.DT.Before(c.markers[len(c.markers)-2].DT))
	c.chart.form.Update()
}

// SetMarkers replaces all markers of the area.
func (c *TimeChartArea) SetMarkers(markers []TimeChartMarker) {
	c.markers = append([]TimeChartMarker(nil), markers...)
	c.markersSorted = false
	c.chart.form.Update()
}

func (c *TimeChartArea) Markers() []TimeChartMarker {
	return c.markers
}

func (c *TimeChartArea) ClearMarkers() {
	c.markers = nil
	c.chart.form.Update()
}

// visibleMarkers returns the markers inside [from, to], sorted by DT.
func (c *TimeChartArea) visibleMarkers(from, to time.Time) []TimeChartMarker {
	if !c.markersSorted {
		sort.SliceStable(c.markers, func(i, j int) bool { return c.markers[i].DT.Before(c.markers[j].DT) })
		c.markersSorted = true
	}
	i0 := sort.Search(len(c.markers), func(i int) bool { return !c.markers[i].DT.Before(from) })
	i1 := sort.Search(len(c.markers), func(i int) bool { return c.markers[i].DT.After(to) })
	return c.markers[i0:i1]
}

// drawMarkers draws the markers of an area in its plot coordinates: a small
// triangle pointing at the value and the text above it (below it when there
// is no room). Texts that would overlap an already drawn one are skipped, so
// a zoomed-out view stays readable; zooming in reveals them.
func (c *TimeChart) drawMarkers(cnv *Canvas, al *timeChartAreaLayout, fontSize float64) {
	markers := al.area.visibleMarkers(c.from, c.to)
	if len(markers) == 0 {
		return
	}
	fontFamily := c.FontFamily()
	_, lineH, err := MeasureText(fontFamily, fontSize, "0")
	if err != nil {
		return
	}
	cnv.SetFontSize(fontSize)
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)

	const (
		triH   = 6
		triW   = 4
		gap    = 2
		padX   = 3
		minGap = 6
	)
	labelBg := c.theme.markerLabelBg
	lastRight := math.MinInt
	for _, m := range markers {
		col := m.Color
		if col == nil {
			col = c.theme.marker
		}
		x := c.xOfTime(m.DT)
		y := al.yOf(m.Value)

		boxH := lineH + 2
		above := y-gap-triH-boxH >= 0
		var boxY int
		if above {
			fillMarkerTriangle(cnv, x, y-gap, -1, triW, triH, col)
			boxY = y - gap - triH - boxH
		} else {
			fillMarkerTriangle(cnv, x, y+gap, 1, triW, triH, col)
			boxY = y + gap + triH
		}

		if m.Text == "" {
			continue
		}
		tw, _, err := MeasureText(fontFamily, fontSize, m.Text)
		if err != nil {
			continue
		}
		boxW := tw + 2*padX
		boxX := x - boxW/2
		if boxX < lastRight+minGap {
			continue
		}
		lastRight = boxX + boxW
		cnv.FillRect(boxX, boxY, boxW, boxH, labelBg)
		cnv.SetColor(col)
		cnv.DrawText(boxX+padX, boxY, tw+2, boxH, m.Text)
	}
	cnv.SetVAlign(VAlignTop)
}

// fillMarkerTriangle fills a triangle with its tip at (x, tipY), growing in
// direction dir (-1 up, 1 down) to halfW on each side over height rows.
// Canvas.FillTriangle is not used because it builds a full-canvas clip mask
// on every call, which is too slow for many markers.
func fillMarkerTriangle(cnv *Canvas, x, tipY, dir, halfW, height int, col color.Color) {
	for i := 0; i < height; i++ {
		hw := halfW * i / (height - 1)
		cnv.FillRect(x-hw, tipY+dir*i, 2*hw+1, 1, col)
	}
}
