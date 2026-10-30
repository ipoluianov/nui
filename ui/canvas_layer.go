package ui

import (
	"image"
	"math"
)

// BlendMode is how a layer is put on what is under it (LayerStyle.Blend):
// the blend modes of CSS and SVG, and a few Porter-Duff operators
type BlendMode int

const (
	// BlendNormal draws the layer over what is under it
	BlendNormal BlendMode = iota
	// BlendMultiply darkens: white leaves the pixels under it as they are
	BlendMultiply
	// BlendScreen lightens: black leaves the pixels under it as they are
	BlendScreen
	// BlendOverlay multiplies the dark and screens the light parts of what
	// is under it
	BlendOverlay
	// BlendDarken keeps the darker of the two, color by color
	BlendDarken
	// BlendLighten keeps the lighter of the two, color by color
	BlendLighten
	// BlendColorDodge brightens what is under it by the layer
	BlendColorDodge
	// BlendColorBurn darkens what is under it by the layer
	BlendColorBurn
	// BlendHardLight is BlendOverlay with the two swapped
	BlendHardLight
	// BlendSoftLight is a softer BlendHardLight
	BlendSoftLight
	// BlendDifference subtracts the darker from the lighter
	BlendDifference
	// BlendExclusion is a softer BlendDifference
	BlendExclusion
	// BlendHue takes the hue of the layer, the saturation and the
	// luminosity of what is under it
	BlendHue
	// BlendSaturation takes the saturation of the layer
	BlendSaturation
	// BlendColor takes the hue and the saturation of the layer: tints
	BlendColor
	// BlendLuminosity takes the luminosity of the layer
	BlendLuminosity
	// BlendAdd adds the colors: glows, light
	BlendAdd
	// BlendErase clears what is under the layer as much as the layer is
	// opaque, whatever its colors: an eraser
	BlendErase
	// BlendKeepInside keeps what is under the layer only where the layer
	// is drawn, as much as it is opaque; the rest of the area of the
	// layer is cleared
	BlendKeepInside
	// BlendAtop draws the layer only over what is drawn under it, as much
	// as that is opaque: a texture over the shape under it
	BlendAtop
)

// LayerStyle is how a layer of BeginLayerStyle is put on the canvas. The
// zero value draws it as it is.
type LayerStyle struct {
	// Opacity of the layer 0..1; 0 is 1
	Opacity float64
	// Blend is how it is mixed with what is under it
	Blend BlendMode
	// Mask, if set, is the opacity of every point: the alpha of the color
	// the paint gives there, in the coordinates of the drawing at the
	// time of BeginLayerStyle. A LinearGradient from opaque to
	// transparent fades the layer out at an edge.
	Mask Paint
	// Isolated draws the layer on a transparent picture of its own, as a
	// layer of another blend is: what is erased in it (BlendErase) or
	// blended in it is of the layer alone - the layer of a drawing
	// program, with the paper under it kept
	Isolated bool
}

// canvasLayer is what BeginLayer keeps
type canvasLayer struct {
	rect    image.Rectangle
	pix     []uint8 // the pixels of rect before the layer
	opacity float64
	blend   BlendMode
	// mask gives the alpha at a pixel of the image, if there is a mask
	mask func(px, py int) uint32
	// base is the image the layer goes on, when the layer is drawn on a
	// picture of its own (a blend other than BlendNormal)
	base *image.RGBA
	// The count of the clips by shapes before the layer
	clips int
	// rec is the SVG recorder of the canvas, put back at EndLayer, when the
	// layer is of a blend SVG has not and is not recorded
	rec *SVGRecorder
}

// BeginLayer starts a group of drawings that EndLayer puts on the canvas
// at the opacity 0..1 as one: where shapes of the group overlap, they are
// not seen through each other, only the group is seen through. Layers
// nest. The group covers the clip at the time of BeginLayer.
//
//	cnv.BeginLayer(0.5)
//	cnv.FillCircle(40, 40, 20, red)
//	cnv.FillCircle(60, 40, 20, red) // no darker overlap
//	cnv.EndLayer()
func (c *Canvas) BeginLayer(opacity float64) {
	if opacity <= 0 {
		// Drawn and taken back: the old meaning of 0
		opacity = 1e-9
	}
	c.BeginLayerStyle(LayerStyle{Opacity: opacity})
}

// BeginLayerStyle starts a group of drawings, as BeginLayer, that EndLayer
// puts on the canvas with the style: an opacity, a blend mode, a mask.
//
//	cnv.BeginLayerStyle(ui.LayerStyle{Blend: ui.BlendErase})
//	cnv.StrokePath(stroke, ui.StrokeStyle{Width: 20, Cap: ui.LineCapRound}, black) // rubs out
//	cnv.EndLayer()
//
// A layer of BlendNormal is drawn on the canvas, the pixels under it kept
// to mix at EndLayer. A layer of another blend, or Isolated, is drawn on a
// transparent picture of its own: what reads the canvas inside it (Blur,
// PixelAt) reads the layer alone.
func (c *Canvas) BeginLayerStyle(st LayerStyle) {
	b := c.clipRect().Intersect(c.rgba.Rect)
	opacity := st.Opacity
	if opacity <= 0 {
		opacity = 1
	}
	l := canvasLayer{rect: b, opacity: min(1, opacity), blend: st.Blend, clips: len(c.clips)}
	if st.Mask != nil {
		l.mask = c.maskSampler(st.Mask)
	}
	if c.rec != nil && !c.rec.beginLayer(c, st) {
		l.rec, c.rec = c.rec, nil
	}
	l.pix = make([]uint8, b.Dx()*b.Dy()*4)
	if st.Blend == BlendNormal && !st.Isolated {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			off := c.rgba.PixOffset(b.Min.X, y)
			copy(l.pix[(y-b.Min.Y)*b.Dx()*4:], c.rgba.Pix[off:off+b.Dx()*4])
		}
	} else {
		// A transparent picture over the clip: the drawings go there
		l.base = c.rgba
		c.rgba = &image.RGBA{Pix: l.pix, Stride: b.Dx() * 4, Rect: b}
	}
	c.layers = append(c.layers, l)
}

// maskSampler returns the alpha of the paint at the pixels of the image,
// in the coordinates of the drawing now
func (c *Canvas) maskSampler(paint Paint) func(px, py int) uint32 {
	colorAt := fastPaint(paint)
	tx, ty, s := float64(c.state.translateX), float64(c.state.translateY), c.scale
	inv, ok := c.state.xf.Invert()
	hasXf := c.state.hasXf && ok
	return func(px, py int) uint32 {
		x, y := (float64(px)+0.5-tx)/s, (float64(py)+0.5-ty)/s
		if hasXf {
			x, y = inv.Apply(x, y)
		}
		return uint32(colorAt(x, y).A)
	}
}

// EndLayer ends the group of the last BeginLayer, putting what was drawn
// in it on what was under it by its style
func (c *Canvas) EndLayer() {
	if len(c.layers) == 0 {
		return
	}
	l := c.layers[len(c.layers)-1]
	c.layers = c.layers[:len(c.layers)-1]
	// The clips by shapes begun in the layer end with it
	c.endClips(l.clips)
	if l.rec != nil {
		c.rec = l.rec
	} else if c.rec != nil {
		c.rec.endLayer()
	}
	if l.base != nil {
		c.rgba = l.base
		c.compositeLayer(l)
		return
	}
	if l.opacity >= 1 && l.mask == nil {
		return
	}
	// The group drawn over what was under it, mixed with what was under
	// it: the group at the opacity over it, as the pixels are blended over
	k := uint32(math.Round(l.opacity * 256))
	b := l.rect
	for y := b.Min.Y; y < b.Max.Y; y++ {
		off := c.rgba.PixOffset(b.Min.X, y)
		cur := c.rgba.Pix[off : off+b.Dx()*4]
		old := l.pix[(y-b.Min.Y)*b.Dx()*4:]
		if l.mask == nil {
			for i, v := range cur {
				o := uint32(old[i])
				cur[i] = uint8((o*(256-k) + uint32(v)*k + 128) >> 8)
			}
			continue
		}
		for x := 0; x < b.Dx(); x++ {
			km := (k*l.mask(b.Min.X+x, y) + 127) / 255
			for i := x * 4; i < x*4+4; i++ {
				o := uint32(old[i])
				cur[i] = uint8((o*(256-km) + uint32(cur[i])*km + 128) >> 8)
			}
		}
	}
}

// compositeLayer puts the picture of the layer l on the canvas by its blend
func (c *Canvas) compositeLayer(l canvasLayer) {
	b := l.rect
	w := b.Dx()
	k := uint32(math.Round(l.opacity * 255))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		off := c.rgba.PixOffset(b.Min.X, y)
		dst := c.rgba.Pix[off : off+w*4 : off+w*4]
		src := l.pix[(y-b.Min.Y)*w*4 : (y-b.Min.Y+1)*w*4]
		if k == 255 && l.mask == nil && blendRow(l.blend, src, dst) {
			continue
		}
		for x := 0; x < w; x++ {
			s := src[x*4 : x*4+4 : x*4+4]
			if s[3] == 0 && l.blend != BlendKeepInside {
				continue
			}
			kk := k
			if l.mask != nil {
				kk = (k*l.mask(b.Min.X+x, y) + 127) / 255
			}
			blendFloat(l.blend, s, dst[x*4:x*4+4:x*4+4], kk)
		}
	}
}

// blendFloat puts the pixel s at the opacity k 0..255 on d by the mode
func blendFloat(mode BlendMode, s, d []uint8, k uint32) {
	var sp, dp [4]float32
	kf := float32(k) / 255
	for i := range 4 {
		sp[i] = unit8[s[i]] * kf
		dp[i] = unit8[d[i]]
	}
	r := blendPixel(mode, sp, dp)
	for i := range 4 {
		d[i] = uint8(max(0, min(255, r[i]*255+0.5)))
	}
}

// unit8 is a component 0..255 as 0..1
var unit8 = func() (t [256]float32) {
	for i := range t {
		t[i] = float32(i) / 255
	}
	return
}()

// mul8 is a*b/255 rounded, for a and b 0..255
func mul8(a, b uint32) uint32 {
	v := a*b + 128
	return (v + v>>8) >> 8
}

// blendRow puts the row src of an opaque layer on the row dst by the mode
// in integers, where it can: the Porter-Duff modes, and the simple ones
// where both pixels are opaque, the common case. It returns false if it
// does not know the mode; the pixels it cannot do it does by blendFloat.
func blendRow(mode BlendMode, src, dst []uint8) bool {
	switch mode {
	case BlendErase, BlendKeepInside:
		for i := 0; i+3 < len(src); i += 4 {
			k := uint32(src[i+3])
			if mode == BlendErase {
				if k == 0 {
					continue
				}
				k = 255 - k
			} else if k == 255 {
				continue
			}
			d := dst[i : i+4 : i+4]
			if k == 0 {
				d[0], d[1], d[2], d[3] = 0, 0, 0, 0
				continue
			}
			d[0], d[1], d[2], d[3] = uint8(mul8(uint32(d[0]), k)), uint8(mul8(uint32(d[1]), k)), uint8(mul8(uint32(d[2]), k)), uint8(mul8(uint32(d[3]), k))
		}
		return true
	case BlendAdd:
		for i, v := range src {
			dst[i] = uint8(min(255, uint32(v)+uint32(dst[i])))
		}
		return true
	case BlendNormal:
		// Premultiplied over
		for i := 0; i+3 < len(src); i += 4 {
			switch ia := 255 - uint32(src[i+3]); ia {
			case 255:
			case 0:
				copy(dst[i:i+4], src[i:i+4])
			default:
				d := dst[i : i+4 : i+4]
				for j := range 4 {
					d[j] = uint8(min(255, uint32(src[i+j])+mul8(uint32(d[j]), ia)))
				}
			}
		}
		return true
	case BlendMultiply, BlendScreen, BlendDarken, BlendLighten, BlendDifference, BlendExclusion, BlendAtop:
	default:
		return false
	}
	for i := 0; i+3 < len(src); i += 4 {
		s, d := src[i:i+4:i+4], dst[i:i+4:i+4]
		if s[3] == 0 {
			continue
		}
		if s[3] != 255 || d[3] != 255 {
			blendFloat(mode, s, d, 255)
			continue
		}
		for j := range 3 {
			b, f := uint32(d[j]), uint32(s[j])
			var r uint32
			switch mode {
			case BlendMultiply:
				r = mul8(b, f)
			case BlendScreen:
				r = b + f - mul8(b, f)
			case BlendDarken:
				r = min(b, f)
			case BlendLighten:
				r = max(b, f)
			case BlendDifference:
				r = max(b, f) - min(b, f)
			case BlendExclusion:
				r = b + f - 2*mul8(b, f)
			case BlendAtop:
				r = f
			}
			d[j] = uint8(r)
		}
	}
	return true
}

// blendPixel returns the premultiplied pixel s put on d by the mode
func blendPixel(mode BlendMode, s, d [4]float32) [4]float32 {
	sa, da := s[3], d[3]
	switch mode {
	case BlendErase:
		return [4]float32{d[0] * (1 - sa), d[1] * (1 - sa), d[2] * (1 - sa), da * (1 - sa)}
	case BlendKeepInside:
		return [4]float32{d[0] * sa, d[1] * sa, d[2] * sa, da * sa}
	case BlendAtop:
		return [4]float32{s[0]*da + d[0]*(1-sa), s[1]*da + d[1]*(1-sa), s[2]*da + d[2]*(1-sa), da}
	case BlendAdd:
		return [4]float32{min(1, s[0]+d[0]), min(1, s[1]+d[1]), min(1, s[2]+d[2]), min(1, sa+da)}
	}
	// The colors straight, mixed by the mode, put on as W3C's compositing
	// says: where one of the two is seen through, the other one shows
	var cs, cb [3]float32
	for i := range 3 {
		if sa > 0 {
			cs[i] = s[i] / sa
		}
		if da > 0 {
			cb[i] = d[i] / da
		}
	}
	var m [3]float32
	switch mode {
	case BlendHue:
		m = setLum(setSat(cs, sat(cb)), lum(cb))
	case BlendSaturation:
		m = setLum(setSat(cb, sat(cs)), lum(cb))
	case BlendColor:
		m = setLum(cs, lum(cb))
	case BlendLuminosity:
		m = setLum(cb, lum(cs))
	default:
		for i := range 3 {
			m[i] = blendChannel(mode, cb[i], cs[i])
		}
	}
	var r [4]float32
	for i := range 3 {
		r[i] = s[i]*(1-da) + d[i]*(1-sa) + sa*da*m[i]
	}
	r[3] = sa + da - sa*da
	return r
}

// blendChannel mixes a straight color b under s by a separable mode
func blendChannel(mode BlendMode, b, s float32) float32 {
	switch mode {
	case BlendMultiply:
		return b * s
	case BlendScreen:
		return b + s - b*s
	case BlendOverlay:
		return blendChannel(BlendHardLight, s, b)
	case BlendDarken:
		return min(b, s)
	case BlendLighten:
		return max(b, s)
	case BlendColorDodge:
		switch {
		case b == 0:
			return 0
		case s >= 1:
			return 1
		}
		return min(1, b/(1-s))
	case BlendColorBurn:
		switch {
		case b >= 1:
			return 1
		case s <= 0:
			return 0
		}
		return 1 - min(1, (1-b)/s)
	case BlendHardLight:
		if s <= 0.5 {
			return b * 2 * s
		}
		return blendChannel(BlendScreen, b, 2*s-1)
	case BlendSoftLight:
		if s <= 0.5 {
			return b - (1-2*s)*b*(1-b)
		}
		var d float32
		if b <= 0.25 {
			d = ((16*b-12)*b + 4) * b
		} else {
			d = float32(math.Sqrt(float64(b)))
		}
		return b + (2*s-1)*(d-b)
	case BlendDifference:
		if b > s {
			return b - s
		}
		return s - b
	case BlendExclusion:
		return b + s - 2*b*s
	}
	return s
}

// The helpers of the non-separable modes, as W3C's compositing defines them

func lum(c [3]float32) float32 { return 0.3*c[0] + 0.59*c[1] + 0.11*c[2] }

func clipColor(c [3]float32) [3]float32 {
	l := lum(c)
	n := min(c[0], c[1], c[2])
	x := max(c[0], c[1], c[2])
	for i := range 3 {
		if n < 0 && l != n {
			c[i] = l + (c[i]-l)*l/(l-n)
		}
		if x > 1 && x != l {
			c[i] = l + (c[i]-l)*(1-l)/(x-l)
		}
	}
	return c
}

func setLum(c [3]float32, l float32) [3]float32 {
	d := l - lum(c)
	return clipColor([3]float32{c[0] + d, c[1] + d, c[2] + d})
}

func sat(c [3]float32) float32 { return max(c[0], c[1], c[2]) - min(c[0], c[1], c[2]) }

func setSat(c [3]float32, s float32) [3]float32 {
	hi, lo := max(c[0], c[1], c[2]), min(c[0], c[1], c[2])
	var r [3]float32
	if hi > lo {
		for i := range 3 {
			r[i] = (c[i] - lo) * s / (hi - lo)
		}
	}
	return r
}
