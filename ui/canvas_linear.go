package ui

import (
	"image/color"
	"math"
)

// Colors mixed in linear light. sRGB, the space of the colors of nui and of
// the screens, is not linear in light: the middle between black and white
// by its numbers (128) is darker than the middle in light (188). Browsers
// and nui mix in sRGB by default; mixing in linear light - as light mixes -
// gives brighter gradients between saturated colors and antialiased edges
// that keep the weight of thin lines at any angle.

// srgbToLinear is a component 0..255 as linear light 0..65535
var srgbToLinear = func() (t [256]uint32) {
	for i := range t {
		c := float64(i) / 255
		if c <= 0.04045 {
			c /= 12.92
		} else {
			c = math.Pow((c+0.055)/1.055, 2.4)
		}
		t[i] = uint32(math.Round(c * 65535))
	}
	return
}()

// linearToSRGB is linear light 0..65535, by 16, as a component 0..255
var linearToSRGB = func() (t [4096]uint8) {
	for i := range t {
		c := float64(i) / 4095
		if c <= 0.0031308 {
			c *= 12.92
		} else {
			c = 1.055*math.Pow(c, 1/2.4) - 0.055
		}
		t[i] = uint8(math.Round(c * 255))
	}
	return
}()

// mixLinear mixes the component s over d by alpha 0..255 in linear light
func mixLinear(d, s, alpha uint32) uint8 {
	l := (srgbToLinear[s]*alpha + srgbToLinear[d]*(255-alpha) + 127) / 255
	return linearToSRGB[l>>4]
}

// SetLinearBlending mixes the colors of what is drawn after it in linear
// light, kept by Save and Restore: the antialiased edges and the
// translucent fills of paths, rectangles, transformed images. Text is
// mixed as the system mixes it (in sRGB), so it keeps its weight; layers,
// blur, DrawImage and DrawImageRect are mixed in sRGB. It applies over
// opaque pixels, which is what a window has.
func (c *Canvas) SetLinearBlending(on bool) {
	c.state.linear = on
}

// LinearBlending tells whether the colors are mixed in linear light
func (c *Canvas) LinearBlending() bool {
	return c.state.linear
}

// linearStopsPerSpan is how many stops a span of a gradient is split into
// in SVG, to mix its colors in linear light where SVG viewers mix them in
// sRGB between the stops
const linearStopsPerSpan = 48

// gradientStops returns the stops of a gradient for SVG: for linear light,
// a span split in stops close at its ends, where sRGB changes fast with
// the light, so that mixing them in sRGB is within a level or two of
// mixing the stops in linear light
func gradientStops(stops []GradientStop, linear bool) []GradientStop {
	if !linear || len(stops) < 2 {
		return stops
	}
	out := make([]GradientStop, 0, (len(stops)-1)*linearStopsPerSpan+1)
	out = append(out, stops[0])
	for i := 1; i < len(stops); i++ {
		a, b := stops[i-1], stops[i]
		if b.Offset <= a.Offset {
			out = append(out, b)
			continue
		}
		for k := 1; k <= linearStopsPerSpan; k++ {
			// Closer at the ends: the steps follow 1-cos
			f := (1 - math.Cos(math.Pi*float64(k)/linearStopsPerSpan)) / 2
			out = append(out, GradientStop{Offset: a.Offset + (b.Offset-a.Offset)*f, Color: mixRGBALinear(a.Color, b.Color, f)})
		}
	}
	return out
}

// mixRGBALinear mixes the straight colors in linear light, f of b and 1-f
// of a, weighted by their alphas as mixRGBA does
func mixRGBALinear(a, b color.RGBA, f float64) color.RGBA {
	f = max(0, min(1, f))
	aa, ba := float64(a.A)*(1-f), float64(b.A)*f
	alpha := aa + ba
	if alpha == 0 {
		return color.RGBA{}
	}
	mix := func(x, y uint8) uint8 {
		l := (float64(srgbToLinear[x])*aa + float64(srgbToLinear[y])*ba) / alpha
		return linearToSRGB[min(4095, int(math.Round(l/16)))]
	}
	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), uint8(math.Round(alpha))}
}
