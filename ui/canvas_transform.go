package ui

import (
	"image"
	"image/color"
	"math"
)

// Matrix is a 2D affine transform: a point (x, y) becomes
//
//	x' = A*x + C*y + E
//	y' = B*x + D*y + F
//
// as the matrix of SVG and of the HTML canvas. The zero value is not the
// identity: use IdentityMatrix.
//
//	m := ui.IdentityMatrix().Translate(100, 50).Rotate(math.Pi / 6)
//	x, y := m.Apply(10, 0)
type Matrix struct {
	A, B, C, D, E, F float64
}

// IdentityMatrix returns the transform that leaves the points as they are
func IdentityMatrix() Matrix {
	return Matrix{A: 1, D: 1}
}

// TranslationMatrix returns the transform that moves by (tx, ty)
func TranslationMatrix(tx, ty float64) Matrix {
	return Matrix{A: 1, D: 1, E: tx, F: ty}
}

// ScalingMatrix returns the transform that stretches by sx and sy from 0, 0
func ScalingMatrix(sx, sy float64) Matrix {
	return Matrix{A: sx, D: sy}
}

// RotationMatrix returns the transform that turns by the angle in radians
// around 0, 0: clockwise on the screen, where y goes down
func RotationMatrix(angle float64) Matrix {
	s, c := math.Sincos(angle)
	return Matrix{A: c, B: s, C: -s, D: c}
}

// SkewMatrix returns the transform that slants by the angles in radians:
// ax along x (x' = x + tan(ax)*y), ay along y
func SkewMatrix(ax, ay float64) Matrix {
	return Matrix{A: 1, B: math.Tan(ay), C: math.Tan(ax), D: 1}
}

// Then returns the transform that applies m, then n
func (m Matrix) Then(n Matrix) Matrix {
	return Matrix{
		A: n.A*m.A + n.C*m.B,
		B: n.B*m.A + n.D*m.B,
		C: n.A*m.C + n.C*m.D,
		D: n.B*m.C + n.D*m.D,
		E: n.A*m.E + n.C*m.F + n.E,
		F: n.B*m.E + n.D*m.F + n.F,
	}
}

// Translate returns m with a move by (tx, ty) applied before it, in its
// own coordinates, as Canvas.Translate does
func (m Matrix) Translate(tx, ty float64) Matrix { return TranslationMatrix(tx, ty).Then(m) }

// Scale returns m with a stretch applied before it, as Canvas.ScaleBy does
func (m Matrix) Scale(sx, sy float64) Matrix { return ScalingMatrix(sx, sy).Then(m) }

// Rotate returns m with a turn applied before it, as Canvas.Rotate does
func (m Matrix) Rotate(angle float64) Matrix { return RotationMatrix(angle).Then(m) }

// Apply returns the point (x, y) transformed
func (m Matrix) Apply(x, y float64) (float64, float64) {
	return m.A*x + m.C*y + m.E, m.B*x + m.D*y + m.F
}

// ApplyPoint returns the point transformed
func (m Matrix) ApplyPoint(p PointF) PointF {
	x, y := m.Apply(p.X, p.Y)
	return PointF{x, y}
}

// ApplyVector returns the vector (dx, dy) transformed, without the move
func (m Matrix) ApplyVector(dx, dy float64) (float64, float64) {
	return m.A*dx + m.C*dy, m.B*dx + m.D*dy
}

// Det returns the determinant: how many times the transform changes the
// areas, negative if it mirrors them
func (m Matrix) Det() float64 { return m.A*m.D - m.B*m.C }

// Invert returns the transform that undoes m; ok is false when m flattens
// the plane to a line or a point and cannot be undone
func (m Matrix) Invert() (inv Matrix, ok bool) {
	det := m.Det()
	if det == 0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return Matrix{}, false
	}
	k := 1 / det
	return Matrix{
		A: m.D * k,
		B: -m.B * k,
		C: -m.C * k,
		D: m.A * k,
		E: (m.C*m.F - m.D*m.E) * k,
		F: (m.B*m.E - m.A*m.F) * k,
	}, true
}

// IsIdentity reports whether m leaves the points as they are
func (m Matrix) IsIdentity() bool { return m == IdentityMatrix() }

// meanScale returns how many times the transform stretches the lengths
// on average: the square root of the change of the areas
func (m Matrix) meanScale() float64 {
	return math.Sqrt(math.Abs(m.Det()))
}

// Transform transforms the points of the path in place: the curves stay
// curves, so the shape is exact at any scale
func (p *Path) Transform(m Matrix) {
	for i := range p.ops {
		op := &p.ops[i]
		if op.kind == pathArc {
			// The center moves, the radii turn and stretch
			op.p[0] = m.ApplyPoint(op.p[0])
			op.p[1].X, op.p[1].Y = m.ApplyVector(op.p[1].X, op.p[1].Y)
			op.p[2].X, op.p[2].Y = m.ApplyVector(op.p[2].X, op.p[2].Y)
			continue
		}
		for k := range op.p {
			op.p[k] = m.ApplyPoint(op.p[k])
		}
	}
	p.cur = m.ApplyPoint(p.cur)
	p.start = m.ApplyPoint(p.start)
}

// Clone returns a copy of the path
func (p *Path) Clone() *Path {
	q := *p
	q.ops = append([]pathOp(nil), p.ops...)
	return &q
}

// copyFrom makes p the same as q, keeping p's memory
func (p *Path) copyFrom(q *Path) {
	ops := append(p.ops[:0], q.ops...)
	*p = *q
	p.ops = ops
}

// The transform of the canvas. It is a part of the state that Save and
// Restore keep, as the clip is; it applies to everything drawn in logical
// coordinates after it is set.

// Transform returns the current transform of the canvas: from the
// coordinates given to the drawing methods to the logical coordinates of
// the canvas (those of TranslateAndClip)
func (c *Canvas) Transform() Matrix {
	if !c.state.hasXf {
		return IdentityMatrix()
	}
	return c.state.xf
}

// SetTransform replaces the transform of the canvas. With a transform
// other than the identity, everything drawn is moved, turned and stretched
// by it: the paths, the rectangles, the lines, the images and the text.
// The clip stays a rectangle of the canvas.
func (c *Canvas) SetTransform(m Matrix) {
	c.state.xf = m
	c.state.hasXf = !m.IsIdentity()
}

// ResetTransform sets the identity transform
func (c *Canvas) ResetTransform() {
	c.state.xf = Matrix{}
	c.state.hasXf = false
}

// Concat applies the transform m before the current one: the coordinates
// given next are transformed by m, then by what was set before
func (c *Canvas) Concat(m Matrix) {
	c.SetTransform(m.Then(c.Transform()))
}

// Translate moves the origin of what is drawn next by (dx, dy)
func (c *Canvas) Translate(dx, dy float64) {
	c.Concat(TranslationMatrix(dx, dy))
}

// ScaleBy stretches what is drawn next by sx and sy from the origin: the
// widths of the lines too. (Scale is the scale of the screen.)
func (c *Canvas) ScaleBy(sx, sy float64) {
	c.Concat(ScalingMatrix(sx, sy))
}

// Rotate turns what is drawn next by the angle in radians around the
// origin, clockwise on the screen
func (c *Canvas) Rotate(angle float64) {
	c.Concat(RotationMatrix(angle))
}

// RotateAround turns what is drawn next by the angle around (cx, cy)
func (c *Canvas) RotateAround(angle, cx, cy float64) {
	c.Translate(cx, cy)
	c.Rotate(angle)
	c.Translate(-cx, -cy)
}

// ToCanvas returns where the point given to the drawing methods lands in
// the logical coordinates of the canvas, with the transform applied
func (c *Canvas) ToCanvas(x, y float64) (float64, float64) {
	if !c.state.hasXf {
		return x, y
	}
	return c.state.xf.Apply(x, y)
}

// FromCanvas returns the point of the drawing methods' coordinates that
// lands at the logical point (x, y) of the canvas: the inverse of
// ToCanvas, e.g. to find what the mouse is over
func (c *Canvas) FromCanvas(x, y float64) (float64, float64) {
	if !c.state.hasXf {
		return x, y
	}
	inv, ok := c.state.xf.Invert()
	if !ok {
		return math.NaN(), math.NaN()
	}
	return inv.Apply(x, y)
}

// pixelMatrix returns the transform from the drawing methods' coordinates
// to the pixels of the image
func (c *Canvas) pixelMatrix() Matrix {
	return c.Transform().Then(Matrix{A: c.scale, D: c.scale, E: float64(c.state.translateX), F: float64(c.state.translateY)})
}

// withPathXf calls f with the path transformed by the canvas' transform,
// the transform taken off for the time of f
func (c *Canvas) withPathXf(p *Path, f func(p *Path)) {
	tp := pathPool.Get().(*Path)
	defer pathPool.Put(tp)
	tp.copyFrom(p)
	tp.Transform(c.state.xf)
	xf := c.state.xf
	c.state.hasXf = false
	defer func() { c.state.hasXf = true; c.state.xf = xf }()
	f(tp)
}

// withoutXf calls f with the transform taken off, put back after
func (c *Canvas) withoutXf(f func()) {
	xf, has := c.state.xf, c.state.hasXf
	c.state.hasXf = false
	defer func() { c.state.xf, c.state.hasXf = xf, has }()
	f()
}

// The integer methods under a transform: their shapes drawn as paths

func (c *Canvas) fillRectXf(x, y, w, h float64, col color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	p.Rect(x, y, w, h)
	c.FillPath(p, col)
}

func (c *Canvas) strokeRectXf(x, y, w, h, r float64, col color.Color) {
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	// The middle of the one pixel wide outline
	p.RoundedRect(x+0.5, y+0.5, w-1, h-1, r)
	c.StrokePath(p, StrokeStyle{Width: 1}, col)
}

func (c *Canvas) fillRoundedXf(x, y, w, h, r float64, col color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	p.RoundedRect(x, y, w, h, r)
	c.FillPath(p, col)
}

// lineXf draws the line between the centers of the pixels, of the width
func (c *Canvas) lineXf(x1, y1, x2, y2, width float64, col color.Color) {
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	p.MoveTo(x1+0.5, y1+0.5)
	p.LineTo(x2+0.5, y2+0.5)
	c.StrokePath(p, StrokeStyle{Width: max(width, 1)}, col)
}

// drawImageXf draws the part src of the image stretched to the rectangle
// x, y, w, h of the drawing coordinates, with the transform
func (c *Canvas) drawImageXf(x, y, w, h float64, img image.Image, src image.Rectangle, smooth bool) {
	if src.Empty() || w <= 0 || h <= 0 {
		return
	}
	m := Matrix{A: w / float64(src.Dx()), D: h / float64(src.Dy()),
		E: x - float64(src.Min.X)*w/float64(src.Dx()), F: y - float64(src.Min.Y)*h/float64(src.Dy())}
	c.drawImageMatrix(img, src, m.Then(c.pixelMatrix()), smooth)
}

// DrawImageTransformed draws the image with the transform m applied to it
// before the canvas' own: the pixel (i, j) of the image (from its
// Bounds().Min) lands at m.Apply(i, j). smooth interpolates the pixels; it
// is for photos, sharp squares for pixel art.
//
//	// An icon turned by 30 degrees around its center at (100, 80)
//	m := ui.TranslationMatrix(-16, -16).Then(ui.RotationMatrix(math.Pi / 6)).Then(ui.TranslationMatrix(100, 80))
//	cnv.DrawImageTransformed(icon, m, true)
func (c *Canvas) DrawImageTransformed(img image.Image, m Matrix, smooth bool) {
	b := img.Bounds()
	if b.Empty() {
		return
	}
	c.drawImageMatrix(img, b, TranslationMatrix(-float64(b.Min.X), -float64(b.Min.Y)).Then(m).Then(c.pixelMatrix()), smooth)
}

// drawImageMatrix draws the part src of the image with the transform m from
// its coordinates to the pixels of the canvas' image, within the clip.
// Every pixel of the destination is sampled from the source at the point
// it comes from: nearest, or bilinear when smooth, the edges antialiased.
func (c *Canvas) drawImageMatrix(img image.Image, src image.Rectangle, m Matrix, smooth bool) {
	src = src.Intersect(img.Bounds())
	if src.Empty() || c.clipEmpty() {
		return
	}
	// A MipImage drawn small: its copy of the nearest size, whose point q
	// is the point Min + q/f of the image
	b := img.Bounds()
	if lv, part, f := sourceImage(img, src, math.Sqrt(math.Abs(m.A*m.D-m.B*m.C)), smooth); f != 1 {
		m = ScalingMatrix(1/f, 1/f).Then(TranslationMatrix(float64(b.Min.X), float64(b.Min.Y))).Then(m)
		img, src = lv, part
	} else {
		img = lv
	}
	if src.Empty() {
		return
	}
	inv, ok := m.Invert()
	if !ok {
		return
	}
	// The pixels the transformed source covers
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, q := range [4][2]float64{{float64(src.Min.X), float64(src.Min.Y)}, {float64(src.Max.X), float64(src.Min.Y)},
		{float64(src.Max.X), float64(src.Max.Y)}, {float64(src.Min.X), float64(src.Max.Y)}} {
		x, y := m.Apply(q[0], q[1])
		minX, minY, maxX, maxY = math.Min(minX, x), math.Min(minY, y), math.Max(maxX, x), math.Max(maxY, y)
	}
	// The source points are stepped from the left of the clip of the whole
	// frame (the sums round as there), the pixels written in clip: see
	// shapeClip
	b = boxPixels([4]float64{minX, minY, maxX, maxY}).Intersect(c.shapeClip()).Intersect(c.rgba.Rect)
	clip := b.Intersect(c.clipRect())
	if clip.Empty() {
		return
	}
	sample := imageSampler(img, src, smooth)
	opacity := float64(255-c.state.imageFade) / 255
	sx0, sy0 := float64(src.Min.X), float64(src.Min.Y)
	sx1, sy1 := float64(src.Max.X), float64(src.Max.Y)
	for py := clip.Min.Y; py < clip.Max.Y; py++ {
		off := c.rgba.PixOffset(b.Min.X, py)
		// The source point of the pixel's center moves by (A, B) of inv per pixel
		u, v := inv.Apply(float64(b.Min.X)+0.5, float64(py)+0.5)
		for px := b.Min.X; px < clip.Max.X; px, off, u, v = px+1, off+4, u+inv.A, v+inv.B {
			if px < clip.Min.X {
				continue
			}
			// How much of the pixel is inside the source: the edges are
			// soft over a pixel of the destination
			cov := edgeCoverage(u, sx0, sx1, inv.A, inv.C) * edgeCoverage(v, sy0, sy1, inv.B, inv.D) * opacity
			if cov <= 0 {
				continue
			}
			r, g, bl, a := sample(u, v)
			if a == 0 {
				continue
			}
			a = uint32(float64(a)*cov + 0.5)
			c.mixPixelAt(off, r, g, bl, a)
		}
	}
}

// edgeCoverage returns how much of a destination pixel, at the source
// coordinate t that changes by da and db per pixel across x and y, falls
// within [lo, hi): 1 inside, fading to 0 over a pixel at the edges
func edgeCoverage(t, lo, hi, da, db float64) float64 {
	half := (math.Abs(da) + math.Abs(db)) / 2
	if half < 1e-9 {
		if t >= lo && t < hi {
			return 1
		}
		return 0
	}
	in := min(t+half, hi) - max(t-half, lo)
	return max(0, min(1, in/(2*half)))
}

// imageSampler returns the straight 8-bit color of the image at the
// fractional point (x, y) of its coordinates, kept within src
func imageSampler(img image.Image, src image.Rectangle, smooth bool) func(x, y float64) (r, g, b, a uint32) {
	at := func(x, y int) (r, g, b, a uint32) {
		x = max(src.Min.X, min(x, src.Max.X-1))
		y = max(src.Min.Y, min(y, src.Max.Y-1))
		if rgba, ok := img.(*image.RGBA); ok {
			// Premultiplied: straightened
			i := rgba.PixOffset(x, y)
			p := rgba.Pix[i : i+4 : i+4]
			a = uint32(p[3])
			if a == 0 {
				return 0, 0, 0, 0
			}
			if a == 255 {
				return uint32(p[0]), uint32(p[1]), uint32(p[2]), 255
			}
			return (uint32(p[0])*255 + a/2) / a, (uint32(p[1])*255 + a/2) / a, (uint32(p[2])*255 + a/2) / a, a
		}
		if n, ok := img.(*image.NRGBA); ok {
			i := n.PixOffset(x, y)
			p := n.Pix[i : i+4 : i+4]
			return uint32(p[0]), uint32(p[1]), uint32(p[2]), uint32(p[3])
		}
		cc := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
		return uint32(cc.R), uint32(cc.G), uint32(cc.B), uint32(cc.A)
	}
	if !smooth {
		return func(x, y float64) (uint32, uint32, uint32, uint32) {
			return at(int(math.Floor(x)), int(math.Floor(y)))
		}
	}
	return func(x, y float64) (uint32, uint32, uint32, uint32) {
		// Bilinear between the four pixel centers around the point, the
		// colors weighted by their alphas
		x -= 0.5
		y -= 0.5
		x0, y0 := math.Floor(x), math.Floor(y)
		fx, fy := x-x0, y-y0
		ix, iy := int(x0), int(y0)
		var sr, sg, sb, sa float64
		for k, w := range [4]float64{(1 - fx) * (1 - fy), fx * (1 - fy), (1 - fx) * fy, fx * fy} {
			if w == 0 {
				continue
			}
			r, g, b, a := at(ix+k%2, iy+k/2)
			wa := w * float64(a)
			sr += float64(r) * wa
			sg += float64(g) * wa
			sb += float64(b) * wa
			sa += wa
		}
		if sa <= 0 {
			return 0, 0, 0, 0
		}
		return uint32(sr/sa + 0.5), uint32(sg/sa + 0.5), uint32(sb/sa + 0.5), uint32(sa + 0.5)
	}
}

// drawTextOutlines draws the text of DrawText in its box, aligned in it,
// as the outlines of its letters (see TextLayout)
func (c *Canvas) drawTextOutlines(x, y, width, height int, text string) {
	l := NewTextLayout(text, TextStyle{Font: c.state.fontFamily, Size: c.state.fontSize, Underline: c.state.underline})
	l.SetAlign(c.state.hAlign)
	lw, lh := l.Size()
	fx := float64(x)
	switch c.state.hAlign {
	case HAlignCenter:
		fx += (float64(width) - lw) / 2
	case HAlignRight:
		fx += float64(width) - lw
	}
	fy := float64(y)
	switch c.state.vAlign {
	case VAlignCenter:
		fy += (float64(height) - lh) / 2
	case VAlignBottom:
		fy += float64(height) - lh
	}
	c.DrawTextLayout(l, fx, fy)
}

// drawTextXf draws the text of DrawText under the transform: drawn into a
// picture at the canvas' resolution, which is then drawn transformed
func (c *Canvas) drawTextXf(x, y, width, height int, text string) {
	if width <= 0 || height <= 0 {
		return
	}
	// The resolution of the picture follows the stretch of the transform,
	// so the text is as sharp as it can be
	k := max(1, c.state.xf.meanScale()) * c.scale
	pw, ph := int(math.Ceil(float64(width)*k)), int(math.Ceil(float64(height)*k))
	if pw <= 0 || ph <= 0 {
		return
	}
	if pw*ph > 1<<20 {
		// Too large a picture: the outlines of the letters, sharp at any
		// zoom
		c.drawTextOutlines(x, y, width, height, text)
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, pw, ph))
	sub := NewCanvasScaled(img, k)
	st := *c.state
	sub.state.col, sub.state.fontFamily, sub.state.fontSize = st.col, st.fontFamily, st.fontSize
	sub.state.hAlign, sub.state.vAlign, sub.state.underline = st.hAlign, st.vAlign, st.underline
	sub.SetDirectTranslateAndClip(0, 0, int(math.Ceil(float64(pw)/k)), int(math.Ceil(float64(ph)/k)))
	sub.drawTextLines(0, 0, width, height, text, false)
	// The text is not an image: drawn opaque whatever SetImageOpacity says
	fade := c.state.imageFade
	c.state.imageFade = 0
	c.drawImageMatrix(img, img.Rect, ScalingMatrix(1/k, 1/k).Then(TranslationMatrix(float64(x), float64(y))).Then(c.pixelMatrix()), true)
	c.state.imageFade = fade
}
