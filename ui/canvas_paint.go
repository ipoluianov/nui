package ui

import (
	"image/color"
	"math"
)

// GradientStop is a color of a gradient at the offset 0..1 along it
type GradientStop struct {
	Offset float64
	Color  color.RGBA
}

// LinearGradient is a Paint whose color changes along the line from
// (X1, Y1) to (X2, Y2), in logical coordinates of the canvas: before the
// start it is the first stop's color, past the end the last one's.
//
//	cnv.FillPathPaint(area, ui.LinearGradient{X1: 0, Y1: top, X2: 0, Y2: bottom,
//		Stops: []ui.GradientStop{{0, blue}, {1, transparent}}})
type LinearGradient struct {
	X1, Y1, X2, Y2 float64
	Stops          []GradientStop
}

// ColorAt implements Paint
func (g LinearGradient) ColorAt(x, y float64) color.RGBA {
	dx, dy := g.X2-g.X1, g.Y2-g.Y1
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = ((x-g.X1)*dx + (y-g.Y1)*dy) / l2
	}
	return gradientColor(g.Stops, t)
}

// RadialGradient is a Paint whose color changes from the center (CX, CY)
// to the circle of the radius R around it
type RadialGradient struct {
	CX, CY, R float64
	Stops     []GradientStop
}

// ColorAt implements Paint
func (g RadialGradient) ColorAt(x, y float64) color.RGBA {
	t := 0.0
	if g.R > 0 {
		t = math.Hypot(x-g.CX, y-g.CY) / g.R
	}
	return gradientColor(g.Stops, t)
}

// gradientColor returns the color of the stops at t, mixed between the two
// stops around it. The stops are taken in their order; ones out of order
// are skipped over, as in CSS.
func gradientColor(stops []GradientStop, t float64) color.RGBA {
	switch len(stops) {
	case 0:
		return color.RGBA{}
	case 1:
		return stops[0].Color
	}
	if math.IsNaN(t) || t <= stops[0].Offset {
		return stops[0].Color
	}
	for i := 1; i < len(stops); i++ {
		a, b := stops[i-1], stops[i]
		if t <= b.Offset {
			span := b.Offset - a.Offset
			if span <= 0 {
				return b.Color
			}
			return mixRGBA(a.Color, b.Color, (t-a.Offset)/span)
		}
	}
	return stops[len(stops)-1].Color
}

// mixRGBA mixes the straight colors, f of b and 1-f of a. The color
// components are weighted by the alphas, so a stop to transparent fades
// the color out instead of darkening it to the black of its components.
func mixRGBA(a, b color.RGBA, f float64) color.RGBA {
	f = max(0, min(1, f))
	aa, ba := float64(a.A)*(1-f), float64(b.A)*f
	alpha := aa + ba
	if alpha == 0 {
		return color.RGBA{}
	}
	mix := func(x, y uint8) uint8 {
		return uint8(math.Round((float64(x)*aa + float64(y)*ba) / alpha))
	}
	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), uint8(math.Round(alpha))}
}

// HatchPaint is a Paint of parallel stripes: of the Color, Width wide,
// every Spacing logical pixels, at the Angle in radians (0 - horizontal
// stripes, math.Pi/4 - from the bottom left to the top right), over the
// Background (transparent - over what is under the shape). For the spans
// of bad data in a chart, the areas that are not available, placeholders.
type HatchPaint struct {
	Color, Background color.RGBA
	Angle             float64
	// Spacing is the distance between the stripes, 0 is 6; Width is the
	// width of a stripe, 0 is 1
	Spacing, Width float64
}

// ColorAt implements Paint
func (h HatchPaint) ColorAt(x, y float64) color.RGBA {
	spacing := h.Spacing
	if spacing <= 0 {
		spacing = 6
	}
	width := h.Width
	if width <= 0 {
		width = 1
	}
	// The distance across the stripes; y grows down the screen, so the
	// stripes at a positive angle rise to the right
	d := x*math.Sin(h.Angle) + y*math.Cos(h.Angle)
	d = math.Mod(d, spacing)
	if d < 0 {
		d += spacing
	}
	d = math.Min(d, spacing-d)
	// How much of a logical pixel across the stripe it covers: the edges
	// antialiased over a pixel
	coverage := max(0, min(1, width/2-d+0.5))
	if coverage == 0 {
		return h.Background
	}
	return mixRGBA(h.Background, h.Color, coverage)
}
