package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"sync"

	gtfont "github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
)

// Color glyphs - emoji - of the two kinds of color fonts:
//
//   - pictures (CBDT, sbix: Noto Color Emoji, Apple Color Emoji): a PNG of
//     the glyph, scaled to the box of its metrics;
//   - layers of shapes (COLR, versions 0 and 1: Segoe UI Emoji, the COLRv1
//     Noto Color Emoji): outlines filled with the colors of the palette and
//     gradients, transformed and composed, drawn by the canvas.
//
// A color glyph keeps its colors; only the palette entry "the text color"
// takes the color of the text. Its pictures are made once for a size and
// kept, as the masks of the other glyphs.

// colorKind is what draws a glyph
type colorKind uint8

const (
	colorNone   colorKind = iota + 1 // an outline in the color of the text
	colorLayers                      // COLR
	colorBitmap                      // CBDT, sbix
)

var colorKinds sync.Map // shapedGlyphKey -> colorKind

// colorKindOf tells how the glyph of the face is drawn
func colorKindOf(face *gtfont.Face, gid gtfont.GID) colorKind {
	key := shapedGlyphKey{face.Font, gid}
	if k, ok := colorKinds.Load(key); ok {
		return k.(colorKind)
	}
	k := colorNone
	shapeMu.Lock()
	if face.COLR != nil && len(face.CPAL) > 0 {
		if _, ok := face.COLR.Search(tables.GlyphID(gid)); ok {
			k = colorLayers
		}
	}
	if k == colorNone {
		if b, ok := face.GlyphDataBitmap(gid); ok && (b.Format == gtfont.PNG || b.Format == gtfont.JPG) {
			k = colorBitmap
		}
	}
	shapeMu.Unlock()
	colorKinds.Store(key, k)
	return k
}

// colorImage is the picture of a color glyph at a size: its pixel (0, 0)
// is (x, y) from the origin of the glyph on the baseline
type colorImage struct {
	img  *image.RGBA
	x, y int
}

type colorImageKey struct {
	shapedGlyphKey
	size  float32
	phase uint8
	fg    color.RGBA
}

var (
	colorImagesMu sync.Mutex
	colorImages   = map[colorImageKey]*colorImage{}
	// colorImagesBytes is the memory of the pictures, kept under a budget
	colorImagesBytes int
)

const colorImagesBudget = 32 << 20

// colorGlyphImage returns the picture of the color glyph at the size in
// pixels, its origin a quarter of a pixel right by the phase; fg is the
// color of the text
func colorGlyphImage(face *gtfont.Face, gid gtfont.GID, kind colorKind, size float64, phase int, fg color.RGBA) *colorImage {
	key := colorImageKey{shapedGlyphKey{face.Font, gid}, float32(size), uint8(phase), fg}
	colorImagesMu.Lock()
	ci, ok := colorImages[key]
	colorImagesMu.Unlock()
	if ok {
		return ci
	}
	dx := float64(phase) / 4
	if kind == colorBitmap {
		ci = renderBitmapGlyph(face, gid, size, dx)
	} else {
		ci = renderLayerGlyph(face, gid, size, dx, fg)
	}
	colorImagesMu.Lock()
	if ci.img != nil {
		colorImagesBytes += len(ci.img.Pix)
	}
	if colorImagesBytes > colorImagesBudget {
		clear(colorImages)
		colorImagesBytes = 0
	}
	colorImages[key] = ci
	colorImagesMu.Unlock()
	return ci
}

// The decoded pictures of the bitmap glyphs
var glyphBitmaps sync.Map // shapedGlyphKey -> image.Image (nil if it cannot be decoded)

// glyphBitmap returns the picture of a bitmap glyph and its box in font
// units, y up: (x0, y0) the bottom left, (x1, y1) the top right
func glyphBitmap(face *gtfont.Face, gid gtfont.GID) (image.Image, [4]float64) {
	shapeMu.Lock()
	data, ok := face.GlyphDataBitmap(gid)
	ext, extOK := face.GlyphExtents(gid)
	adv := face.HorizontalAdvance(gid)
	upem := float64(face.Upem())
	shapeMu.Unlock()
	box := [4]float64{float64(ext.XBearing), float64(ext.YBearing + ext.Height), float64(ext.XBearing + ext.Width), float64(ext.YBearing)}
	if !extOK || ext.Width <= 0 || ext.Height >= 0 {
		// No metrics: the advance wide, as tall as an em above the baseline
		box = [4]float64{0, -0.2 * upem, float64(adv), 0.8 * upem}
	}
	key := shapedGlyphKey{face.Font, gid}
	if img, ok := glyphBitmaps.Load(key); ok {
		img, _ := img.(image.Image)
		return img, box
	}
	var img image.Image
	if ok {
		switch data.Format {
		case gtfont.PNG:
			img, _ = png.Decode(bytes.NewReader(data.Data))
		case gtfont.JPG:
			img, _ = jpeg.Decode(bytes.NewReader(data.Data))
		}
	}
	if img != nil {
		// Drawn much smaller than it is: its mipmaps, made once
		img = NewMipImage(img)
	}
	glyphBitmaps.Store(key, img)
	return img, box
}

// renderBitmapGlyph scales the picture of the glyph to the box of its
// metrics at the size
func renderBitmapGlyph(face *gtfont.Face, gid gtfont.GID, size, dx float64) *colorImage {
	img, box := glyphBitmap(face, gid)
	if img == nil {
		return &colorImage{}
	}
	k := size / float64(face.Upem())
	x0, x1 := box[0]*k+dx, box[2]*k+dx
	top, bottom := -box[3]*k, -box[1]*k
	r := image.Rect(int(math.Floor(x0)), int(math.Floor(top)), int(math.Ceil(x1)), int(math.Ceil(bottom)))
	if r.Empty() {
		return &colorImage{}
	}
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	cnv := NewCanvasScaled(out, 1)
	cnv.SetDirectTranslateAndClip(0, 0, r.Dx(), r.Dy())
	b := img.Bounds()
	m := TranslationMatrix(-float64(b.Min.X), -float64(b.Min.Y)).
		Then(ScalingMatrix((x1-x0)/float64(b.Dx()), (bottom-top)/float64(b.Dy()))).
		Then(TranslationMatrix(x0-float64(r.Min.X), top-float64(r.Min.Y)))
	cnv.drawImageMatrix(img, b, m, true)
	return &colorImage{img: out, x: r.Min.X, y: r.Min.Y}
}

// renderLayerGlyph draws the layers of a COLR glyph at the size
func renderLayerGlyph(face *gtfont.Face, gid gtfont.GID, size, dx float64, fg color.RGBA) *colorImage {
	k := size / float64(face.Upem())
	box := colrBox(face, gid)
	r := image.Rect(int(math.Floor(box[0]*k+dx))-1, int(math.Floor(-box[3]*k))-1, int(math.Ceil(box[2]*k+dx))+1, int(math.Ceil(-box[1]*k))+1)
	if r.Empty() || r.Dx()*r.Dy() > 4096*4096 {
		return &colorImage{}
	}
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	cnv := NewCanvasScaled(out, 1)
	cnv.SetDirectTranslateAndClip(0, 0, r.Dx(), r.Dy())
	// Font units, y up, to the pixels of the picture
	cnv.SetTransform(Matrix{A: k, D: -k, E: dx - float64(r.Min.X), F: -float64(r.Min.Y)})
	drawColrGlyph(cnv, face, gid, fg)
	return &colorImage{img: out, x: r.Min.X, y: r.Min.Y}
}

// colrBox returns the box of a COLR glyph in font units, y up: its clip
// box, or the advance and the line of the font
func colrBox(face *gtfont.Face, gid gtfont.GID) [4]float64 {
	shapeMu.Lock()
	defer shapeMu.Unlock()
	if cb, ok := face.COLR.ClipList.Search(tables.GlyphID(gid)); ok {
		switch c := cb.(type) {
		case tables.ClipBoxFormat1:
			return [4]float64{float64(c.XMin), float64(c.YMin), float64(c.XMax), float64(c.YMax)}
		case tables.ClipBoxFormat2:
			return [4]float64{float64(c.XMin), float64(c.YMin), float64(c.XMax), float64(c.YMax)}
		}
	}
	upem := float64(face.Upem())
	asc, desc := 0.9*upem, -0.25*upem
	if e, ok := face.FontHExtents(); ok {
		asc, desc = float64(e.Ascender), float64(e.Descender)
	}
	// The extent of the layers may pass the advance: an em around it
	adv := float64(face.HorizontalAdvance(gid))
	return [4]float64{-0.25 * upem, desc, max(adv, upem) + 0.25*upem, asc}
}

// drawColrGlyph draws the COLR glyph on the canvas, whose transform takes
// font units (y up) to the place of the glyph
func drawColrGlyph(cnv *Canvas, face *gtfont.Face, gid gtfont.GID, fg color.RGBA) {
	shapeMu.Lock()
	defer shapeMu.Unlock()
	paint, ok := face.COLR.Search(tables.GlyphID(gid))
	if !ok {
		return
	}
	d := colrDrawer{cnv: cnv, face: face, fg: fg}
	d.paint(paint, 0)
}

// colrDrawer draws the paints of a COLR glyph; shapeMu is held
type colrDrawer struct {
	cnv  *Canvas
	face *gtfont.Face
	fg   color.RGBA
}

// maxColrDepth limits the nesting of paints: a font that loops is cut off
const maxColrDepth = 64

func f214(v tables.Fixed214) float64 { return float64(v) / 16384 }

// color returns the color of the palette entry at the alpha
func (d *colrDrawer) color(index uint16, alpha float64) color.RGBA {
	c := d.fg
	if index != 0xFFFF && int(index) < len(d.face.CPAL[0]) {
		e := d.face.CPAL[0][index]
		c = color.RGBA{e.Red, e.Green, e.Blue, e.Alpha}
	}
	c.A = uint8(math.Round(float64(c.A) * max(0, min(1, alpha))))
	return c
}

// outline returns the outline of a glyph of the font in font units, y up
func (d *colrDrawer) outline(gid tables.GlyphID) *Path {
	p := NewPath()
	o, ok := d.face.GlyphDataOutline(gtfont.GID(gid))
	if !ok {
		return p
	}
	open := false
	for _, s := range o.Segments {
		a := s.Args
		switch s.Op {
		case ot.SegmentOpMoveTo:
			if open {
				p.Close()
			}
			p.MoveTo(float64(a[0].X), float64(a[0].Y))
			open = true
		case ot.SegmentOpLineTo:
			p.LineTo(float64(a[0].X), float64(a[0].Y))
		case ot.SegmentOpQuadTo:
			p.QuadTo(float64(a[0].X), float64(a[0].Y), float64(a[1].X), float64(a[1].Y))
		case ot.SegmentOpCubeTo:
			p.CubicTo(float64(a[0].X), float64(a[0].Y), float64(a[1].X), float64(a[1].Y), float64(a[2].X), float64(a[2].Y))
		}
	}
	if open {
		p.Close()
	}
	return p
}

// fill returns the Paint of a paint table that fills (a color or a
// gradient), nil for any other
func (d *colrDrawer) fill(paint tables.PaintTable) Paint {
	switch p := paint.(type) {
	case tables.PaintSolid:
		return solidPaint(d.color(p.PaletteIndex, f214(p.Alpha)))
	case tables.PaintVarSolid:
		return solidPaint(d.color(p.PaletteIndex, f214(p.Alpha)))
	case tables.PaintLinearGradient:
		return d.linear(d.stops(p.ColorLine.ColorStops), p.ColorLine.Extend, p.X0, p.Y0, p.X1, p.Y1, p.X2, p.Y2)
	case tables.PaintVarLinearGradient:
		return d.linear(d.varStops(p.ColorLine.ColorStops), p.ColorLine.Extend, p.X0, p.Y0, p.X1, p.Y1, p.X2, p.Y2)
	case tables.PaintRadialGradient:
		return d.radial(d.stops(p.ColorLine.ColorStops), p.ColorLine.Extend, p.X0, p.Y0, float64(p.Radius0), p.X1, p.Y1, float64(p.Radius1))
	case tables.PaintVarRadialGradient:
		return d.radial(d.varStops(p.ColorLine.ColorStops), p.ColorLine.Extend, p.X0, p.Y0, float64(p.Radius0), p.X1, p.Y1, float64(p.Radius1))
	case tables.PaintSweepGradient:
		return d.sweep(d.stops(p.ColorLine.ColorStops), p.ColorLine.Extend, p.CenterX, p.CenterY, p.StartAngle, p.EndAngle)
	case tables.PaintVarSweepGradient:
		return d.sweep(d.varStops(p.ColorLine.ColorStops), p.ColorLine.Extend, p.CenterX, p.CenterY, p.StartAngle, p.EndAngle)
	}
	return nil
}

// colorLine is the stops of a gradient: offsets in their own range
type colorLine struct {
	offsets []float64
	colors  []color.RGBA
}

func (d *colrDrawer) stops(s []tables.ColorStop) colorLine {
	var l colorLine
	for _, st := range s {
		l.offsets = append(l.offsets, f214(st.StopOffset))
		l.colors = append(l.colors, d.color(st.PaletteIndex, f214(st.Alpha)))
	}
	return l
}

func (d *colrDrawer) varStops(s []tables.VarColorStop) colorLine {
	var l colorLine
	for _, st := range s {
		l.offsets = append(l.offsets, f214(st.StopOffset))
		l.colors = append(l.colors, d.color(st.PaletteIndex, f214(st.Alpha)))
	}
	return l
}

// normalized returns the stops sorted and moved to 0..1, and the range
// they had: the geometry of the gradient is moved to it
func (l colorLine) normalized() ([]GradientStop, float64, float64) {
	n := len(l.offsets)
	if n == 0 {
		return nil, 0, 1
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	// Stable by offset: stops at one offset keep their order
	for i := 1; i < n; i++ {
		for j := i; j > 0 && l.offsets[idx[j]] < l.offsets[idx[j-1]]; j-- {
			idx[j], idx[j-1] = idx[j-1], idx[j]
		}
	}
	lo, hi := l.offsets[idx[0]], l.offsets[idx[n-1]]
	span := hi - lo
	out := make([]GradientStop, n)
	for i, k := range idx {
		o := 0.0
		if span > 0 {
			o = (l.offsets[k] - lo) / span
		}
		out[i] = GradientStop{Offset: o, Color: l.colors[k]}
	}
	if span <= 0 {
		// One offset: the color of the last stop all over
		out = []GradientStop{{0, out[n-1].Color}, {1, out[n-1].Color}}
		hi = lo + 1
	}
	return out, lo, hi
}

func spreadOf(e tables.Extend) GradientSpread {
	switch e {
	case tables.ExtendRepeat:
		return SpreadRepeat
	case tables.ExtendReflect:
		return SpreadReflect
	}
	return SpreadPad
}

// linear returns the gradient along p0..p3, p3 the foot of p1 on the line
// through p0 square to p0p2, as COLR defines it
func (d *colrDrawer) linear(l colorLine, e tables.Extend, x0, y0, x1, y1, x2, y2 int16) Paint {
	stops, lo, hi := l.normalized()
	p0 := PointF{float64(x0), float64(y0)}
	p1 := PointF{float64(x1), float64(y1)}
	// The direction square to p0p2
	nx, ny := -(float64(y2) - p0.Y), float64(x2)-p0.X
	n2 := nx*nx + ny*ny
	p3 := p1
	if n2 > 0 {
		f := ((p1.X-p0.X)*nx + (p1.Y-p0.Y)*ny) / n2
		p3 = PointF{p0.X + f*nx, p0.Y + f*ny}
	}
	at := func(t float64) PointF { return PointF{p0.X + (p3.X-p0.X)*t, p0.Y + (p3.Y-p0.Y)*t} }
	a, b := at(lo), at(hi)
	return LinearGradient{X1: a.X, Y1: a.Y, X2: b.X, Y2: b.Y, Stops: stops, Spread: spreadOf(e)}
}

// radial returns the gradient between two circles
func (d *colrDrawer) radial(l colorLine, e tables.Extend, x0, y0 int16, r0 float64, x1, y1 int16, r1 float64) Paint {
	stops, lo, hi := l.normalized()
	c := func(t float64) (float64, float64, float64) {
		return float64(x0) + (float64(x1)-float64(x0))*t, float64(y0) + (float64(y1)-float64(y0))*t, r0 + (r1-r0)*t
	}
	ax, ay, ar := c(lo)
	bx, by, br := c(hi)
	return twoCircleGradient{x0: ax, y0: ay, r0: ar, x1: bx, y1: by, r1: br, stops: stops, spread: spreadOf(e)}
}

// sweep returns the gradient around a center, counterclockwise in font
// units from the start angle to the end one
func (d *colrDrawer) sweep(l colorLine, e tables.Extend, cx, cy int16, start, end tables.Fixed214) Paint {
	stops, lo, hi := l.normalized()
	a0 := (f214(start) + 1) * math.Pi
	a1 := (f214(end) + 1) * math.Pi
	return sweepGradient{cx: float64(cx), cy: float64(cy), a0: a0 + (a1-a0)*lo, a1: a0 + (a1-a0)*hi, stops: stops, spread: spreadOf(e)}
}

// paint draws a paint table within the clip of the canvas
func (d *colrDrawer) paint(paint tables.PaintTable, depth int) {
	if depth > maxColrDepth {
		return
	}
	cnv := d.cnv
	switch p := paint.(type) {
	case tables.PaintColrLayersResolved:
		// COLR version 0: glyphs in colors, one over another
		for _, layer := range p {
			cnv.FillPath(d.outline(layer.GlyphID), d.color(layer.PaletteIndex, 1))
		}
	case tables.PaintColrLayers:
		layers, err := d.face.COLR.LayerList.Resolve(p)
		if err != nil {
			return
		}
		for _, layer := range layers {
			d.paint(layer, depth+1)
		}
	case tables.PaintGlyph:
		shape := d.outline(p.GlyphID)
		if f := d.fill(p.Paint); f != nil {
			cnv.FillPathPaint(shape, f)
			return
		}
		cnv.Save()
		cnv.ClipPath(shape)
		d.paint(p.Paint, depth+1)
		cnv.Restore()
	case tables.PaintColrGlyph:
		if inner, ok := d.face.COLR.Search(p.GlyphID); ok {
			d.paint(inner, depth+1)
		}
	case tables.PaintTransform:
		t := p.Transform
		d.transformed(Matrix{A: float64(t.Xx), B: float64(t.Yx), C: float64(t.Xy), D: float64(t.Yy), E: float64(t.Dx), F: float64(t.Dy)}, p.Paint, depth)
	case tables.PaintVarTransform:
		t := p.Transform
		d.transformed(Matrix{A: float64(t.Xx), B: float64(t.Yx), C: float64(t.Xy), D: float64(t.Yy), E: float64(t.Dx), F: float64(t.Dy)}, p.Paint, depth)
	case tables.PaintTranslate:
		d.transformed(TranslationMatrix(float64(p.Dx), float64(p.Dy)), p.Paint, depth)
	case tables.PaintVarTranslate:
		d.transformed(TranslationMatrix(float64(p.Dx), float64(p.Dy)), p.Paint, depth)
	case tables.PaintScale:
		d.transformed(ScalingMatrix(f214(p.ScaleX), f214(p.ScaleY)), p.Paint, depth)
	case tables.PaintVarScale:
		d.transformed(ScalingMatrix(f214(p.ScaleX), f214(p.ScaleY)), p.Paint, depth)
	case tables.PaintScaleAroundCenter:
		d.transformed(around(ScalingMatrix(f214(p.ScaleX), f214(p.ScaleY)), p.CenterX, p.CenterY), p.Paint, depth)
	case tables.PaintVarScaleAroundCenter:
		d.transformed(around(ScalingMatrix(f214(p.ScaleX), f214(p.ScaleY)), p.CenterX, p.CenterY), p.Paint, depth)
	case tables.PaintScaleUniform:
		d.transformed(ScalingMatrix(f214(p.Scale), f214(p.Scale)), p.Paint, depth)
	case tables.PaintVarScaleUniform:
		d.transformed(ScalingMatrix(f214(p.Scale), f214(p.Scale)), p.Paint, depth)
	case tables.PaintScaleUniformAroundCenter:
		d.transformed(around(ScalingMatrix(f214(p.Scale), f214(p.Scale)), p.CenterX, p.CenterY), p.Paint, depth)
	case tables.PaintVarScaleUniformAroundCenter:
		d.transformed(around(ScalingMatrix(f214(p.Scale), f214(p.Scale)), p.CenterX, p.CenterY), p.Paint, depth)
	case tables.PaintRotate:
		d.transformed(RotationMatrix(f214(p.Angle)*math.Pi), p.Paint, depth)
	case tables.PaintVarRotate:
		d.transformed(RotationMatrix(f214(p.Angle)*math.Pi), p.Paint, depth)
	case tables.PaintRotateAroundCenter:
		d.transformed(around(RotationMatrix(f214(p.Angle)*math.Pi), p.CenterX, p.CenterY), p.Paint, depth)
	case tables.PaintVarRotateAroundCenter:
		d.transformed(around(RotationMatrix(f214(p.Angle)*math.Pi), p.CenterX, p.CenterY), p.Paint, depth)
	case tables.PaintSkew:
		d.transformed(colrSkew(p.XSkewAngle, p.YSkewAngle), p.Paint, depth)
	case tables.PaintVarSkew:
		d.transformed(colrSkew(p.XSkewAngle, p.YSkewAngle), p.Paint, depth)
	case tables.PaintSkewAroundCenter:
		d.transformed(around(colrSkew(p.XSkewAngle, p.YSkewAngle), p.CenterX, p.CenterY), p.Paint, depth)
	case tables.PaintVarSkewAroundCenter:
		d.transformed(around(colrSkew(p.XSkewAngle, p.YSkewAngle), p.CenterX, p.CenterY), p.Paint, depth)
	case tables.PaintComposite:
		// The backdrop and the source on a picture of their own, the source
		// put on the backdrop by the mode
		cnv.BeginLayerStyle(LayerStyle{Isolated: true})
		d.paint(p.BackdropPaint, depth+1)
		cnv.BeginLayerStyle(LayerStyle{Blend: compositeBlend(p.CompositeMode), Isolated: true})
		d.paint(p.SourcePaint, depth+1)
		cnv.EndLayer()
		cnv.EndLayer()
	default:
		// A fill alone: all of the clip
		if f := d.fill(paint); f != nil {
			b := cnv.ClipBounds()
			area := NewPath()
			area.Rect(b.MinX, b.MinY, b.Width(), b.Height())
			cnv.FillPathPaint(area, f)
		}
	}
}

// transformed draws the paint with the transform m before the canvas'
func (d *colrDrawer) transformed(m Matrix, paint tables.PaintTable, depth int) {
	d.cnv.Save()
	d.cnv.Concat(m)
	d.paint(paint, depth+1)
	d.cnv.Restore()
}

// around returns the transform m about the center (cx, cy)
func around(m Matrix, cx, cy int16) Matrix {
	return TranslationMatrix(-float64(cx), -float64(cy)).Then(m).Then(TranslationMatrix(float64(cx), float64(cy)))
}

// colrSkew returns the skew of COLR: the angles counterclockwise, 180° a
// unit, the y axis leaning by x, the x axis by y
func colrSkew(xAngle, yAngle tables.Fixed214) Matrix {
	return Matrix{A: 1, B: math.Tan(f214(yAngle) * math.Pi), C: -math.Tan(f214(xAngle) * math.Pi), D: 1}
}

// compositeBlend returns the blend of a composite mode of COLR; the
// Porter-Duff modes nui has not are drawn over
func compositeBlend(m tables.CompositeMode) BlendMode {
	switch m {
	case tables.CompositeDestOut:
		return BlendErase
	case tables.CompositeDestIn:
		return BlendKeepInside
	case tables.CompositeSrcAtop:
		return BlendAtop
	case tables.CompositePlus:
		return BlendAdd
	}
	if b, ok := map[tables.CompositeMode]BlendMode{
		tables.CompositeScreen: BlendScreen, tables.CompositeOverlay: BlendOverlay, tables.CompositeDarken: BlendDarken,
		tables.CompositeLighten: BlendLighten, tables.CompositeColorDodge: BlendColorDodge, tables.CompositeColorBurn: BlendColorBurn,
		tables.CompositeHardLight: BlendHardLight, tables.CompositeSoftLight: BlendSoftLight, tables.CompositeDifference: BlendDifference,
		tables.CompositeExclusion: BlendExclusion, tables.CompositeMultiply: BlendMultiply, tables.CompositeHslHue: BlendHue,
		tables.CompositeHslSaturation: BlendSaturation, tables.CompositeHslColor: BlendColor, tables.CompositeHslLuminosity: BlendLuminosity,
	}[m]; ok {
		return b
	}
	return BlendNormal
}

// solidPaint is a Paint of one color
type solidPaint color.RGBA

func (s solidPaint) ColorAt(x, y float64) color.RGBA { return color.RGBA(s) }

// twoCircleGradient is the radial gradient of COLR and CSS: the circles
// between (x0, y0, r0) at 0 and (x1, y1, r1) at 1
type twoCircleGradient struct {
	x0, y0, r0, x1, y1, r1 float64
	stops                  []GradientStop
	spread                 GradientSpread
}

// ColorAt implements Paint: the color of the largest t whose circle passes
// through the point
func (g twoCircleGradient) ColorAt(x, y float64) color.RGBA {
	cdx, cdy, dr := g.x1-g.x0, g.y1-g.y0, g.r1-g.r0
	pdx, pdy := x-g.x0, y-g.y0
	a := cdx*cdx + cdy*cdy - dr*dr
	b := pdx*cdx + pdy*cdy + g.r0*dr
	c := pdx*pdx + pdy*pdy - g.r0*g.r0
	t := math.NaN()
	if math.Abs(a) < 1e-9 {
		if b != 0 {
			t = c / (2 * b)
		}
	} else if disc := b*b - a*c; disc >= 0 {
		s := math.Sqrt(disc)
		t1, t2 := (b+s)/a, (b-s)/a
		if t1 < t2 {
			t1, t2 = t2, t1
		}
		switch {
		case g.r0+t1*dr >= 0:
			t = t1
		case g.r0+t2*dr >= 0:
			t = t2
		}
	}
	if math.IsNaN(t) || g.r0+t*dr < 0 {
		return color.RGBA{}
	}
	return gradientColor(g.stops, spreadT(t, g.spread))
}

// sweepGradient is the sweep gradient of COLR: counterclockwise in font
// units (y up) from the angle a0 at 0 to a1 at 1
type sweepGradient struct {
	cx, cy, a0, a1 float64
	stops          []GradientStop
	spread         GradientSpread
}

// ColorAt implements Paint
func (g sweepGradient) ColorAt(x, y float64) color.RGBA {
	a := math.Atan2(y-g.cy, x-g.cx)
	if a < 0 {
		a += 2 * math.Pi
	}
	if g.a1 == g.a0 {
		return gradientColor(g.stops, 0)
	}
	return gradientColor(g.stops, spreadT((a-g.a0)/(g.a1-g.a0), g.spread))
}

// blendColorImage puts the picture of a color glyph on dst, its origin at
// the pixel (x, y), within the clip, at the opacity alpha 0..255
func blendColorImage(dst *image.RGBA, clip image.Rectangle, ci *colorImage, x, y int, alpha uint32) {
	if ci == nil || ci.img == nil {
		return
	}
	box := ci.img.Rect.Add(image.Pt(x+ci.x, y+ci.y))
	vis := box.Intersect(clip).Intersect(dst.Rect)
	if vis.Empty() {
		return
	}
	for py := vis.Min.Y; py < vis.Max.Y; py++ {
		d := dst.Pix[dst.PixOffset(vis.Min.X, py):]
		s := ci.img.Pix[ci.img.PixOffset(vis.Min.X-box.Min.X, py-box.Min.Y):]
		for i := 0; i < vis.Dx()*4; i += 4 {
			sa := uint32(s[i+3])
			if sa == 0 {
				continue
			}
			if alpha != 255 {
				sa = mul8(sa, alpha)
			}
			ia := 255 - sa
			for c := 0; c < 3; c++ {
				sc := uint32(s[i+c])
				if alpha != 255 {
					sc = mul8(sc, alpha)
				}
				d[i+c] = uint8(min(255, sc+mul8(uint32(d[i+c]), ia)))
			}
			d[i+3] = uint8(min(255, sa+mul8(uint32(d[i+3]), ia)))
		}
	}
}

var colorFaces sync.Map // *gtfont.Font -> bool

// isColorFace tells whether the face is of color emoji: its smiling face
// is a color glyph. Called with shapeMu held.
func isColorFace(face *gtfont.Face) bool {
	if v, ok := colorFaces.Load(face.Font); ok {
		return v.(bool)
	}
	color := false
	if gid, ok := face.NominalGlyph(0x1F600); ok {
		if face.COLR != nil {
			_, color = face.COLR.Search(tables.GlyphID(gid))
		}
		if !color {
			b, ok := face.GlyphDataBitmap(gid)
			color = ok && (b.Format == gtfont.PNG || b.Format == gtfont.JPG)
		}
	}
	colorFaces.Store(face.Font, color)
	return color
}
