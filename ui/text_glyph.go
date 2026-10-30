package ui

import (
	"image"
	"image/color"
	"math"
	"sync"

	gtfont "github.com/go-text/typesetting/font"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// The glyphs of nui's text, as the system draws its own: fitted to the
// pixels by the hinter (see text_hint.go), and on an LCD screen drawn with
// its red, green and blue stripes - three times as many columns as pixels,
// filtered as FreeType does, so a stem has no colored fringes. The masks
// are made once for a font, a size and a place within a pixel, and kept.

// TextHinting fits the glyphs to the pixels vertically (see text_hint.go);
// false draws their outlines as they are. Set it before the text is drawn.
var TextHinting = true

// Subpixel is the order of the colored stripes of the pixels of a screen,
// which text is drawn with
type Subpixel int

const (
	// SubpixelAuto takes the order of the screen from the system
	SubpixelAuto Subpixel = iota
	// SubpixelNone draws text in shades of gray
	SubpixelNone
	// SubpixelRGB is red on the left, then green, then blue
	SubpixelRGB
	// SubpixelBGR is blue on the left
	SubpixelBGR
)

// TextSubpixel is how the text of the windows is drawn on LCD screens:
// SubpixelAuto takes the system's setting (fontconfig on Linux, ClearType
// on Windows; macOS draws in gray). Pictures (NewImageCanvas, export,
// printing) are always drawn in gray, and so is text on what is not
// opaque.
var TextSubpixel = SubpixelAuto

// lcdOf returns the subpixel order the text of a window is drawn with:
// 0 gray, 1 RGB, 2 BGR
func lcdOf(system int) int {
	switch TextSubpixel {
	case SubpixelNone:
		return 0
	case SubpixelRGB:
		return 1
	case SubpixelBGR:
		return 2
	}
	return system
}

// glyphMaskKey is a glyph drawn at a size in pixels, its origin a quarter
// of a pixel right of a whole one
type glyphMaskKey struct {
	font  any
	gid   uint32
	size  float32
	phase uint8
	synth FontStyle
	lcd   bool
	hint  bool
}

// glyphMask is the coverage of a glyph, its origin at (-x, -y) of it: a
// byte a pixel, or three (red, green, blue) for an LCD screen
type glyphMask struct {
	x, y, w, h int
	lcd        bool
	pix        []uint8
}

var (
	glyphMasksMu sync.Mutex
	glyphMasks   = map[glyphMaskKey]*glyphMask{}
)

// maxGlyphMasks is how many masks are kept: when there are more, they are
// all made again as needed
const maxGlyphMasks = 16384

// glyphMaskFor returns the kept mask of the key, made by make the first
// time
func glyphMaskFor(key glyphMaskKey, make func() *glyphMask) *glyphMask {
	glyphMasksMu.Lock()
	m, ok := glyphMasks[key]
	glyphMasksMu.Unlock()
	if ok {
		return m
	}
	m = make()
	glyphMasksMu.Lock()
	if len(glyphMasks) >= maxGlyphMasks {
		clear(glyphMasks)
	}
	glyphMasks[key] = m
	glyphMasksMu.Unlock()
	return m
}

// layoutGlyphMask returns the mask of a glyph of a TextLayout
func layoutGlyphMask(face *gtfont.Face, gid gtfont.GID, size float64, phase int, synth FontStyle, lcd int) *glyphMask {
	key := glyphMaskKey{face.Font, uint32(gid), float32(size), uint8(phase), synth, lcd != 0, TextHinting}
	return glyphMaskFor(key, func() *glyphMask {
		var z *hintZones
		if key.hint {
			z = zonesOf(face.Font, func(r rune) *glyphShape {
				shapeMu.Lock()
				g, ok := face.NominalGlyph(r)
				shapeMu.Unlock()
				if !ok {
					return nil
				}
				return shapeOf(face, g)
			})
		}
		bold := 0.0
		if synth&FontBold != 0 {
			bold = size / 24
		}
		return rasterGlyph(shapeOf(face, gid), z, key.hint, size, float64(phase)/4, synth&FontItalic != 0, bold, key.lcd)
	})
}

// sfntShapes are the outlines of the glyphs of DrawText's fonts, in ems
var sfntShapes sync.Map // glyphKey -> *glyphShape

// sfntShape returns the outline of the glyph in ems, y down
func sfntShape(f *sfnt.Font, g sfnt.GlyphIndex) *glyphShape {
	key := glyphKey{f, g}
	if s, ok := sfntShapes.Load(key); ok {
		return s.(*glyphShape)
	}
	buf := sfntBuffers.Get().(*sfnt.Buffer)
	o := glyphOutline(buf, f, g)
	sfntBuffers.Put(buf)
	s := &glyphShape{}
	if o != nil {
		k := 1 / float64(o.unitsPerEm)
		for _, seg := range o.segs {
			op := glyphOp{}
			switch seg.op {
			case sfnt.SegmentOpMoveTo:
				op.kind = 'M'
			case sfnt.SegmentOpLineTo:
				op.kind = 'L'
			case sfnt.SegmentOpQuadTo:
				op.kind = 'Q'
			case sfnt.SegmentOpCubeTo:
				op.kind = 'C'
			}
			for i := 0; i < op.n(); i++ {
				op.p[i] = PointF{seg.p[i].X * k, seg.p[i].Y * k}
			}
			s.ops = append(s.ops, op)
		}
	}
	sfntShapes.Store(key, s)
	return s
}

// sfntGlyphMask returns the mask of a glyph of DrawText; bold is the
// widening of a synthesized bold in pixels
func sfntGlyphMask(f *sfnt.Font, g sfnt.GlyphIndex, size float64, synth FontStyle, bold int, lcd int) *glyphMask {
	key := glyphMaskKey{f, uint32(g), float32(size), 0, synth, lcd != 0, TextHinting}
	return glyphMaskFor(key, func() *glyphMask {
		var z *hintZones
		if key.hint {
			z = zonesOf(f, func(r rune) *glyphShape {
				buf := sfntBuffers.Get().(*sfnt.Buffer)
				i, err := f.GlyphIndex(buf, r)
				sfntBuffers.Put(buf)
				if err != nil || i == 0 {
					return nil
				}
				return sfntShape(f, i)
			})
		}
		return rasterGlyph(sfntShape(f, g), z, key.hint, size, 0, synth&FontItalic != 0, float64(bold), key.lcd)
	})
}

// lcdFilter spreads the coverage of a stripe over its neighbors, as
// FreeType's default LCD filter: the colors of the fringes cancel out
var lcdFilter = [5]uint32{8, 77, 86, 77, 8}

// rasterGlyph makes the mask of the outline in ems at the size in pixels,
// hinted if hint, moved right by dx, slanted if italic, widened right by
// bold pixels, in the stripes of an LCD screen if lcd
func rasterGlyph(s *glyphShape, z *hintZones, hint bool, size, dx float64, italic bool, bold float64, lcd bool) *glyphMask {
	var ops []glyphOp
	if hint && s != nil {
		ops = hintGlyph(s, z, size)
	} else if s != nil {
		ops = make([]glyphOp, len(s.ops))
		for i, op := range s.ops {
			ops[i].kind = op.kind
			for k := 0; k < op.n(); k++ {
				ops[i].p[k] = PointF{op.p[k].X * size, op.p[k].Y * size}
			}
		}
	}
	if len(ops) == 0 {
		return &glyphMask{lcd: lcd}
	}
	shear := 0.0
	if italic {
		shear = italicShear
	}
	sx := 1.0
	if lcd {
		sx = 3
	}
	// The copies of a synthesized bold: none more than a pixel apart
	copies := []float64{0}
	if bold > 0 {
		n := int(math.Ceil(bold))
		for i := 1; i <= n; i++ {
			copies = append(copies, bold*float64(i)/float64(n))
		}
	}
	p := NewPath()
	for _, off := range copies {
		at := func(q PointF) (float64, float64) {
			return (q.X - q.Y*shear + dx + off) * sx, q.Y
		}
		open := false
		for _, op := range ops {
			switch op.kind {
			case 'M':
				if open {
					p.Close()
				}
				p.MoveTo(at(op.p[0]))
				open = true
			case 'L':
				p.LineTo(at(op.p[0]))
			case 'Q':
				x1, y1 := at(op.p[0])
				x2, y2 := at(op.p[1])
				p.QuadTo(x1, y1, x2, y2)
			case 'C':
				x1, y1 := at(op.p[0])
				x2, y2 := at(op.p[1])
				x3, y3 := at(op.p[2])
				p.CubicTo(x1, y1, x2, y2, x3, y3)
			}
		}
		if open {
			p.Close()
		}
	}
	minX, minY, maxX, maxY, ok := p.Bounds()
	if !ok {
		return &glyphMask{lcd: lcd}
	}
	// The box in pixels, a pixel around: the filter spreads past the ink
	px0, px1 := int(math.Floor(minX/sx))-1, int(math.Ceil(maxX/sx))+1
	r := image.Rect(int(px0*int(sx)), int(math.Floor(minY))-1, int(px1*int(sx)), int(math.Ceil(maxY))+1)
	cover := make([]uint8, r.Dx()*r.Dy())
	// A canvas of the box, only to rasterize: the coverage goes to cover
	mc := &Canvas{rgba: &image.RGBA{Rect: r, Stride: r.Dx() * 4}, scale: 1,
		state: &canvasState{clipX: r.Min.X, clipY: r.Min.Y, clipW: r.Dx(), clipH: r.Dy()}}
	mc.fillPathWith(p, func(_ int, c []uint8, x, y int) {
		copy(cover[(y-r.Min.Y)*r.Dx()+x-r.Min.X:], c)
	})
	m := &glyphMask{x: px0, y: r.Min.Y, w: px1 - px0, h: r.Dy(), lcd: lcd}
	if !lcd {
		m.pix = cover
		return m
	}
	// The stripes filtered: each takes of its two neighbors on each side
	m.pix = make([]uint8, m.w*m.h*3)
	n := r.Dx()
	for y := 0; y < m.h; y++ {
		row := cover[y*n : (y+1)*n]
		out := m.pix[y*n : (y+1)*n]
		for i := range out {
			var v uint32
			for k, f := range lcdFilter {
				if j := i + k - 2; j >= 0 && j < n {
					v += uint32(row[j]) * f
				}
			}
			out[i] = uint8(min(255, (v+128)>>8))
		}
	}
	return m
}

// blendGlyphMask blends the straight color (r, g, b, a) over the pixels of
// dst the mask covers, its origin at the pixel (x, y), within the clip. An
// LCD mask blends each color by its own stripe (lcd 2: blue on the left)
// where the pixel is opaque, and by their mean where it is not: the
// stripes of a screen are only under an opaque picture.
func blendGlyphMask(dst *image.RGBA, clip image.Rectangle, m *glyphMask, x, y int, r, g, b, a uint32, lcd int) {
	box := image.Rect(x+m.x, y+m.y, x+m.x+m.w, y+m.y+m.h)
	vis := box.Intersect(clip).Intersect(dst.Rect)
	if vis.Empty() || a == 0 {
		return
	}
	w := vis.Dx()
	for py := vis.Min.Y; py < vis.Max.Y; py++ {
		off := dst.PixOffset(vis.Min.X, py)
		px := dst.Pix[off : off+w*4 : off+w*4]
		mi := (py-box.Min.Y)*m.w + vis.Min.X - box.Min.X
		if !m.lcd {
			blendGrayRow(px, m.pix[mi:mi+w], r, g, b, a)
			continue
		}
		blendLCDRow(px, m.pix[mi*3:(mi+w)*3], r, g, b, a, lcd)
	}
}

// mix8 is d*(255-k) + s*k, divided by 255 and rounded
func mix8(d, s, k uint32) uint8 {
	v := s*k + d*(255-k) + 128
	return uint8((v + v>>8) >> 8)
}

// blendGrayRow blends the color over the pixels by their coverage
func blendGrayRow(px, cover []uint8, r, g, b, a uint32) {
	for i, c := range cover {
		if c == 0 {
			continue
		}
		k := uint32(c)
		if a != 255 {
			k = (k*a + 127) / 255
		}
		d := px[i*4 : i*4+4 : i*4+4]
		if k == 255 {
			d[0], d[1], d[2], d[3] = uint8(r), uint8(g), uint8(b), 255
			continue
		}
		d[0], d[1], d[2] = mix8(uint32(d[0]), r, k), mix8(uint32(d[1]), g, k), mix8(uint32(d[2]), b, k)
		if d[3] != 255 {
			d[3] = mix8(uint32(d[3]), 255, k)
		}
	}
}

// blendLCDRow blends the color over the pixels, each color by its stripe
// where the pixel is opaque, by the mean of the stripes where it is not
func blendLCDRow(px, cover []uint8, r, g, b, a uint32, lcd int) {
	for i := 0; i < len(cover)/3; i++ {
		cr, cg, cb := uint32(cover[i*3]), uint32(cover[i*3+1]), uint32(cover[i*3+2])
		if cr|cg|cb == 0 {
			continue
		}
		d := px[i*4 : i*4+4 : i*4+4]
		if lcd == 2 {
			cr, cb = cb, cr
		}
		if d[3] != 255 || lcd == 0 {
			cr = (cr + cg + cb + 1) / 3
			cg, cb = cr, cr
		}
		if a != 255 {
			cr, cg, cb = (cr*a+127)/255, (cg*a+127)/255, (cb*a+127)/255
		}
		d[0], d[1], d[2] = mix8(uint32(d[0]), r, cr), mix8(uint32(d[1]), g, cg), mix8(uint32(d[2]), b, cb)
		if d[3] != 255 {
			d[3] = mix8(uint32(d[3]), 255, max(cr, cg, cb))
		}
	}
}

// placedGlyph is a character of a line of DrawText at its place
type placedGlyph struct {
	font  *sfnt.Font
	synth FontStyle
	r     rune
	x     int
}

var placedGlyphPool = sync.Pool{New: func() any { return new([]placedGlyph) }}

// drawTextGlyphs draws a line of text as DrawText does - at the same
// places, kerned and advanced as it measures - from nui's own glyphs
func drawTextGlyphs(dst *image.RGBA, text string, col color.Color, family string, size float64, x, y int, clip image.Rectangle, lcd int) {
	r, g, b, a := straightRGBA8(col)
	if a == 0 || text == "" {
		return
	}
	ensureFallbackFonts(text)
	gp := placedGlyphPool.Get().(*[]placedGlyph)
	glyphs := (*gp)[:0]
	defer func() { *gp = glyphs; placedGlyphPool.Put(gp) }()
	baseline := y
	withFace(family, size, func(face font.Face) {
		fb, ok := face.(*fallbackFace)
		if !ok {
			return
		}
		m := fb.Metrics()
		baseline = y + (m.Ascent + m.Descent).Ceil() - m.Descent.Ceil()
		dot := fixed.Int26_6(0)
		prev := rune(-1)
		for _, ch := range text {
			if prev >= 0 {
				dot += fb.Kern(prev, ch)
			}
			i := fb.faceIndexFor(ch)
			glyphs = append(glyphs, placedGlyph{fb.fonts[i], fb.synths[i], ch, dot.Round()})
			adv, _ := fb.GlyphAdvance(ch)
			dot += adv
			prev = ch
		}
	})
	// On the baseline of the system font the other text is drawn with
	if ascent, _, ok := nativeMetrics(family, size); ok {
		baseline = y + ascent
	}
	bold := max(1, int(math.Round(size/24)))
	buf := sfntBuffers.Get().(*sfnt.Buffer)
	defer sfntBuffers.Put(buf)
	for _, gl := range glyphs {
		gi, err := gl.font.GlyphIndex(buf, gl.r)
		if err != nil {
			continue
		}
		bw := 0
		if gl.synth&FontBold != 0 {
			bw = bold
		}
		m := sfntGlyphMask(gl.font, gi, size, gl.synth, bw, lcd)
		blendGlyphMask(dst, clip, m, x+gl.x, baseline, r, g, b, a, lcd)
	}
}
