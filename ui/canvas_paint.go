package ui

import (
	"image"
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
	// Spread is what is past the ends: the end colors, or the gradient
	// repeated or mirrored
	Spread GradientSpread
	// LinearRGB mixes the colors of the stops in linear light (SVG's
	// color-interpolation="linearRGB"): brighter between saturated colors
	LinearRGB bool
}

// GradientSpread is how a gradient goes on past its ends
type GradientSpread int

const (
	// SpreadPad keeps the color of the end stop (SVG spreadMethod="pad")
	SpreadPad GradientSpread = iota
	// SpreadRepeat starts the gradient again: stripes
	SpreadRepeat
	// SpreadReflect goes back and forth along it
	SpreadReflect
)

// spreadT returns where in 0..1 the point t along the gradient takes its
// color
func spreadT(t float64, spread GradientSpread) float64 {
	switch spread {
	case SpreadRepeat:
		return t - math.Floor(t)
	case SpreadReflect:
		t = math.Mod(math.Abs(t), 2)
		if t > 1 {
			t = 2 - t
		}
		return t
	}
	return t
}

// ColorAt implements Paint
func (g LinearGradient) ColorAt(x, y float64) color.RGBA {
	dx, dy := g.X2-g.X1, g.Y2-g.Y1
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = ((x-g.X1)*dx + (y-g.Y1)*dy) / l2
	}
	return gradientColorIn(g.Stops, g.LinearRGB, spreadT(t, g.Spread))
}

// RadialGradient is a Paint whose color changes from the center (CX, CY)
// to the circle of the radius R around it
type RadialGradient struct {
	CX, CY, R float64
	Stops     []GradientStop
	// Spread is what is past the circle, as of LinearGradient
	Spread GradientSpread
	// LinearRGB mixes in linear light, as of LinearGradient
	LinearRGB bool
}

// ColorAt implements Paint
func (g RadialGradient) ColorAt(x, y float64) color.RGBA {
	t := 0.0
	if g.R > 0 {
		t = math.Hypot(x-g.CX, y-g.CY) / g.R
	}
	return gradientColorIn(g.Stops, g.LinearRGB, spreadT(t, g.Spread))
}

// ConicGradient is a Paint whose color changes around the center (CX, CY):
// the stop at 0 at the angle Angle in radians (0 - to the right), going
// clockwise on the screen to 1 a turn later. For color wheels, gauges,
// the sweep of a radar.
type ConicGradient struct {
	CX, CY, Angle float64
	Stops         []GradientStop
	// LinearRGB mixes in linear light, as of LinearGradient
	LinearRGB bool
}

// ColorAt implements Paint
func (g ConicGradient) ColorAt(x, y float64) color.RGBA {
	return gradientColorIn(g.Stops, g.LinearRGB, conicT(x-g.CX, y-g.CY, g.Angle))
}

func conicT(dx, dy, angle float64) float64 {
	t := (math.Atan2(dy, dx) - angle) / (2 * math.Pi)
	return t - math.Floor(t)
}

// ImagePattern is a Paint of an image: its pixel (i, j) (from its
// Bounds().Min) at the logical point (X + i*Scale, Y + j*Scale), repeated
// in both directions when Repeat is set, transparent around it when not.
// Smooth interpolates the pixels.
type ImagePattern struct {
	Image  image.Image
	X, Y   float64
	Scale  float64 // 0 is 1
	Repeat bool
	Smooth bool
}

// ColorAt implements Paint
func (p ImagePattern) ColorAt(x, y float64) color.RGBA {
	return p.sampler()(x, y)
}

// sampler returns ColorAt with what it needs made once
func (p ImagePattern) sampler() func(x, y float64) color.RGBA {
	if p.Image == nil {
		return func(float64, float64) color.RGBA { return color.RGBA{} }
	}
	b := p.Image.Bounds()
	if b.Empty() {
		return func(float64, float64) color.RGBA { return color.RGBA{} }
	}
	scale := p.Scale
	if scale <= 0 {
		scale = 1
	}
	sample := imageSampler(p.Image, b, p.Smooth)
	w, h := float64(b.Dx()), float64(b.Dy())
	return func(x, y float64) color.RGBA {
		u, v := (x-p.X)/scale, (y-p.Y)/scale
		if p.Repeat {
			u -= math.Floor(u/w) * w
			v -= math.Floor(v/h) * h
		} else if u < 0 || v < 0 || u >= w || v >= h {
			return color.RGBA{}
		}
		r, g, bl, a := sample(u+float64(b.Min.X), v+float64(b.Min.Y))
		return color.RGBA{uint8(r), uint8(g), uint8(bl), uint8(a)}
	}
}

// gradientColor returns the color of the stops at t, mixed between the two
// stops around it. The stops are taken in their order; ones out of order
// are skipped over, as in CSS.
func gradientColor(stops []GradientStop, t float64) color.RGBA {
	return gradientColorIn(stops, false, t)
}

// gradientColorIn is gradientColor, the colors mixed in linear light if
// linear
func gradientColorIn(stops []GradientStop, linear bool, t float64) color.RGBA {
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
			if linear {
				return mixRGBALinear(a.Color, b.Color, (t-a.Offset)/span)
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

// gradientLUTSize is how many colors a gradient is computed at before a
// fill: a step is a 1/1024 of the gradient, under a level of a color
const gradientLUTSize = 1024

// gradientLUT is the colors of the stops at gradientLUTSize points from the
// first stop's offset to the last one's
type gradientLUT struct {
	lo, hi      float64
	k           float64 // gradientLUTSize / (hi - lo)
	first, last color.RGBA
	colors      [gradientLUTSize + 1]color.RGBA
}

func newGradientLUT(stops []GradientStop, linear bool) *gradientLUT {
	l := &gradientLUT{}
	if len(stops) == 0 {
		return l
	}
	l.lo, l.hi = stops[0].Offset, stops[0].Offset
	for _, st := range stops {
		l.hi = max(l.hi, st.Offset)
	}
	if l.hi > l.lo {
		l.k = gradientLUTSize / (l.hi - l.lo)
	}
	l.first = gradientColorIn(stops, linear, l.lo)
	l.last = gradientColorIn(stops, linear, math.Inf(1))
	for i := range l.colors {
		l.colors[i] = gradientColorIn(stops, linear, l.lo+(l.hi-l.lo)*float64(i)/gradientLUTSize)
	}
	return l
}

func (l *gradientLUT) at(t float64) color.RGBA {
	if !(t > l.lo) { // NaN too
		return l.first
	}
	if t >= l.hi {
		return l.last
	}
	return l.colors[min(int((t-l.lo)*l.k+0.5), gradientLUTSize)]
}

// fastPaint returns the paint's ColorAt; of the gradients of nui, made
// with their colors computed once, not mixed again at every pixel
func fastPaint(paint Paint) func(x, y float64) color.RGBA {
	switch g := paint.(type) {
	case LinearGradient:
		if len(g.Stops) < 2 {
			break
		}
		lut := newGradientLUT(g.Stops, g.LinearRGB)
		dx, dy := g.X2-g.X1, g.Y2-g.Y1
		l2 := dx*dx + dy*dy
		if l2 <= 0 {
			break
		}
		kx, ky := dx/l2, dy/l2
		if g.Spread != SpreadPad {
			return func(x, y float64) color.RGBA {
				return lut.at(spreadT((x-g.X1)*kx+(y-g.Y1)*ky, g.Spread))
			}
		}
		return func(x, y float64) color.RGBA {
			return lut.at((x-g.X1)*kx + (y-g.Y1)*ky)
		}
	case RadialGradient:
		if len(g.Stops) < 2 || g.R <= 0 {
			break
		}
		lut := newGradientLUT(g.Stops, g.LinearRGB)
		k := 1 / g.R
		return func(x, y float64) color.RGBA {
			dx, dy := x-g.CX, y-g.CY
			return lut.at(spreadT(math.Sqrt(dx*dx+dy*dy)*k, g.Spread))
		}
	case ConicGradient:
		if len(g.Stops) < 2 {
			break
		}
		lut := newGradientLUT(g.Stops, g.LinearRGB)
		return func(x, y float64) color.RGBA {
			return lut.at(conicT(x-g.CX, y-g.CY, g.Angle))
		}
	case ImagePattern:
		return g.sampler()
	}
	return paint.ColorAt
}
