package ui

import (
	"image"
	"math"
)

// ClipBounds returns the part of the drawing coordinates the clip shows:
// the box of the clip taken back through the transform. A painter draws
// only what reaches into it - the rest would be clipped anyway.
func (c *Canvas) ClipBounds() RectF {
	r := c.clipRect().Intersect(c.rgba.Rect)
	if c.clipEmpty() || r.Empty() {
		return RectF{}
	}
	inv, ok := c.pixelMatrix().Invert()
	if !ok {
		return RectF{}
	}
	b := EmptyRectF()
	for _, q := range [4][2]float64{{float64(r.Min.X), float64(r.Min.Y)}, {float64(r.Max.X), float64(r.Min.Y)},
		{float64(r.Min.X), float64(r.Max.Y)}, {float64(r.Max.X), float64(r.Max.Y)}} {
		x, y := inv.Apply(q[0], q[1])
		b = b.AddPoint(PointF{X: x, Y: y})
	}
	return b
}

// PaintCache keeps the picture of an expensive drawing - a chart of a
// million points, a map, a page, the static part of a game - and puts it
// on the canvas again until it is invalidated: the widget's other
// drawings, the cursor, the selection, are painted over it every frame
// without painting it again. A part of it can be invalidated alone: only
// that part is painted again.
//
//	cache := ui.NewPaintCache()
//	w.SetOnPaint(func(cnv *ui.Canvas) {
//		cache.Draw(cnv, 0, 0, w.Width(), w.Height(), paintMap)
//		paintCursor(cnv) // every frame
//	})
//	...
//	cache.InvalidateRect(changed) // a marker moved: its box painted again
//	w.Update()
//
// The picture is transparent where nothing is drawn. It is made again
// when the size or the scale of the screen changes. Under a transform of
// the canvas, or with an SVG recorder, the drawing is painted directly,
// not kept.
type PaintCache struct {
	img   *image.RGBA
	w, h  int
	scale float64
	valid bool
	dirty []RectF
}

// NewPaintCache returns an empty cache: the first Draw paints
func NewPaintCache() *PaintCache {
	return &PaintCache{}
}

// Invalidate makes the next Draw paint all of the picture again
func (p *PaintCache) Invalidate() {
	p.valid = false
	p.dirty = p.dirty[:0]
}

// InvalidateRect makes the next Draw paint the rectangle again, in the
// logical coordinates of the picture (its top left corner is 0, 0)
func (p *PaintCache) InvalidateRect(r RectF) {
	if !p.valid || r.Width() <= 0 || r.Height() <= 0 {
		return
	}
	// Rectangles that overlap are painted as one
	for i, d := range p.dirty {
		if d.Intersects(r) {
			p.dirty = append(p.dirty[:i], p.dirty[i+1:]...)
			p.InvalidateRect(d.Union(r))
			return
		}
	}
	p.dirty = append(p.dirty, r)
}

// Valid tells whether the picture is kept whole: Draw will not paint
func (p *PaintCache) Valid() bool {
	return p.valid && len(p.dirty) == 0
}

// Draw puts the picture of paint, w × h logical pixels, at (x, y) of the
// canvas; paint is called to paint the picture, or its invalid parts,
// first. paint draws from 0, 0 of the picture and may skip what is out of
// cnv.ClipBounds().
func (p *PaintCache) Draw(cnv *Canvas, x, y, w, h int, paint func(cnv *Canvas)) {
	if w <= 0 || h <= 0 {
		return
	}
	if cnv.state.hasXf || cnv.rec != nil {
		// Turned or stretched, or recorded as vectors: painted directly
		cnv.Save()
		cnv.Translate(float64(x), float64(y))
		paint(cnv)
		cnv.Restore()
		return
	}
	pw, ph := int(math.Ceil(float64(w)*cnv.scale)), int(math.Ceil(float64(h)*cnv.scale))
	if p.img == nil || p.w != w || p.h != h || p.scale != cnv.scale || p.img.Rect.Dx() != pw || p.img.Rect.Dy() != ph {
		p.img = image.NewRGBA(image.Rect(0, 0, pw, ph))
		p.w, p.h, p.scale = w, h, cnv.scale
		p.valid = false
	}
	if !p.valid {
		clear(p.img.Pix)
		pc := NewCanvasScaled(p.img, cnv.scale)
		pc.SetDirectTranslateAndClip(0, 0, w, h)
		paint(pc)
		p.valid = true
		p.dirty = p.dirty[:0]
	}
	for _, d := range p.dirty {
		pc := NewCanvasScaled(p.img, cnv.scale)
		pc.SetDirectTranslateAndClip(0, 0, w, h)
		// The whole pixels the rectangle touches
		x0, y0 := int(math.Floor(d.MinX)), int(math.Floor(d.MinY))
		pc.intersectClip(x0, y0, int(math.Ceil(d.MaxX))-x0, int(math.Ceil(d.MaxY))-y0)
		if pc.clipEmpty() {
			continue
		}
		pc.Clear(image.Transparent)
		paint(pc)
	}
	p.dirty = p.dirty[:0]
	// The picture over the canvas, pixel for pixel
	px, py := cnv.state.translateX+cnv.scaled(x), cnv.state.translateY+cnv.scaled(y)
	dst := image.Rect(px, py, px+pw, py+ph)
	vis := dst.Intersect(cnv.clipRect()).Intersect(cnv.rgba.Rect)
	if vis.Empty() {
		return
	}
	drawOver(cnv.rgba, vis, p.img, vis.Min.Sub(dst.Min))
}
