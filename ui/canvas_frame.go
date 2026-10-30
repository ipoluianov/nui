package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"slices"
	"sync"

	"golang.org/x/image/vector"
)

// FillRoundedRectAA fills a rounded rectangle with antialiased corners, of
// the color given (FillRoundedRect fills with the current color).
func (c *Canvas) FillRoundedRectAA(x, y, width, height, radius int, col color.RGBA) {
	if width <= 0 || height <= 0 {
		return
	}
	if c.state.hasXf {
		c.fillRoundedXf(float64(x), float64(y), float64(width), float64(height), float64(radius), col)
		return
	}
	x, y, width, height = c.pixelRect(x, y, width, height)
	c.fillRoundedRectAAPixels(x, y, width, height, c.scaled(radius), col)
}

// fillRoundedRectAAPixels is FillRoundedRectAA in pixels of the image
func (c *Canvas) fillRoundedRectAAPixels(x, y, width, height, radius int, col color.RGBA) {
	if width <= 0 || height <= 0 {
		return
	}
	radius = max(0, min(radius, width/2, height/2))
	if radius == 0 {
		c.fillRectPixels(x, y, width, height, col)
		return
	}

	// The middle band and the parts of the top and bottom bands between the corners
	c.fillRectPixels(x, y+radius, width, height-radius*2, col)
	c.fillRectPixels(x+radius, y, width-radius*2, radius, col)
	c.fillRectPixels(x+radius, y+height-radius, width-radius*2, radius, col)

	// The corners: each pixel covered as much as the quarter circle covers it
	cover := cornerCoverages(radius)
	for cy := 0; cy < radius; cy++ {
		for cx := 0; cx < radius; cx++ {
			coverage := cover[cy*radius+cx]
			if coverage == 0 {
				continue
			}
			pixel := col
			pixel.A = uint8(float64(col.A)*coverage + 0.5)
			c.blendPixel(x+cx, y+cy, pixel)
			c.blendPixel(x+width-1-cx, y+cy, pixel)
			c.blendPixel(x+cx, y+height-1-cy, pixel)
			c.blendPixel(x+width-1-cx, y+height-1-cy, pixel)
		}
	}
}

// FillFrame draws a control's frame: a rounded rectangle filled with fill
// and outlined with a 1px border. fill must be opaque.
func (c *Canvas) FillFrame(x, y, width, height, radius int, fill, border color.Color) {
	c.FillRoundedRectAA(x, y, width, height, radius, colorToRGBA(border))
	c.FillRoundedRectAA(x+1, y+1, width-2, height-2, radius-1, colorToRGBA(fill))
}

// cornerCoverage returns how much of the pixel (cx, cy) of a top-left
// corner square is inside the quarter circle of the radius centered at
// (radius, radius), sampling it 4x4.
func cornerCoverage(cx, cy, radius int) float64 {
	const samples = 4
	r := float64(radius)
	inside := 0
	for sy := 0; sy < samples; sy++ {
		for sx := 0; sx < samples; sx++ {
			dx := float64(cx) + (float64(sx)+0.5)/samples - r
			dy := float64(cy) + (float64(sy)+0.5)/samples - r
			if dx*dx+dy*dy <= r*r {
				inside++
			}
		}
	}
	return float64(inside) / (samples * samples)
}

// cornerCache keeps the coverages of the corners by their radius: every
// frame of every control is drawn with one of a few radii
var cornerCache sync.Map // int -> []float64

// cornerCoverages returns cornerCoverage of every pixel of the corner
// square of the radius, row by row
func cornerCoverages(radius int) []float64 {
	if v, ok := cornerCache.Load(radius); ok {
		return v.([]float64)
	}
	cover := make([]float64, radius*radius)
	for cy := 0; cy < radius; cy++ {
		for cx := 0; cx < radius; cx++ {
			cover[cy*radius+cx] = cornerCoverage(cx, cy, radius)
		}
	}
	if radius <= 256 {
		cornerCache.Store(radius, cover)
	}
	return cover
}

// blendPixel blends the straight (not premultiplied) color into the pixel
// (x, y) of the image, inside the clip rectangle only.
func (c *Canvas) blendPixel(x, y int, col color.RGBA) {
	if x < c.state.clipX || x >= c.state.clipX+c.state.clipW || y < c.state.clipY || y >= c.state.clipY+c.state.clipH {
		return
	}
	if !(image.Point{x, y}).In(c.rgba.Rect) {
		return
	}
	// As MixPixel, without boxing the color into an interface
	c.mixPixelAt(c.rgba.PixOffset(x, y), uint32(col.R), uint32(col.G), uint32(col.B), uint32(col.A))
}

// fillRoundedPixels fills, anti-aliased, the rectangle x, y, w, h in pixels
// of the image with corners of the radius. With a thickness above 0 only
// its outline is filled, that many pixels wide; the radius is then that of
// the middle of the outline, as a stroke along it. Only the part within
// clip (and the canvas' clip) is drawn; each pixel is blended once, by how
// much of it the shape covers.
func (c *Canvas) fillRoundedPixels(x, y, w, h int, radius float64, thickness int, col color.Color, clip image.Rectangle) {
	if w <= 0 || h <= 0 {
		return
	}
	b := image.Rect(x, y, x+w, y+h).Intersect(clip).Intersect(c.clipRect()).Intersect(c.rgba.Rect)
	if b.Empty() {
		return
	}
	outline := thickness > 0 && w > thickness*2 && h > thickness*2
	t := float64(thickness)
	// The outline of the shape, and of the hole of an outline, made once
	var outerBuf, innerBuf [4 * 33][2]float64
	outerPts := roundedRectPoints(outerBuf[:0], float64(x), float64(y), float64(w), float64(h), radius+t/2*b2f(outline))
	var innerPts [][2]float64
	if outline {
		innerPts = roundedRectPoints(innerBuf[:0], float64(x)+t, float64(y)+t, float64(w)-t*2, float64(h)-t*2, radius-t/2)
		slices.Reverse(innerPts) // the other way round: it is taken out
	}
	part := func(r image.Rectangle) {
		r = r.Intersect(b)
		if r.Empty() {
			return
		}
		ras := rasterizers.Get().(*rasterizer)
		defer rasterizers.Put(ras)
		z := ras.reset(r.Dx(), r.Dy())
		addPoints(z, outerPts, float64(r.Min.X), float64(r.Min.Y))
		addPoints(z, innerPts, float64(r.Min.X), float64(r.Min.Y))
		c.fillRasterized(ras, r, col)
	}
	fill := func(r image.Rectangle) {
		if r = r.Intersect(b); !r.Empty() {
			c.fillRectPixels(r.Min.X, r.Min.Y, r.Dx(), r.Dy(), col)
		}
	}
	// Only the corners are rasterized: between them the shape is whole
	// pixels, rectangles filled directly, as a button is drawn on every
	// frame
	outer := max(0, min(radius+t/2*b2f(outline), float64(w)/2, float64(h)/2))
	band := max(int(math.Ceil(outer))+1, thickness+1)
	if w <= band*2 || h <= band*2 {
		part(image.Rect(x, y, x+w, y+h))
		return
	}
	x2, y2 := x+w, y+h
	part(image.Rect(x, y, x+band, y+band))
	part(image.Rect(x2-band, y, x2, y+band))
	part(image.Rect(x, y2-band, x+band, y2))
	part(image.Rect(x2-band, y2-band, x2, y2))
	if outline {
		fill(image.Rect(x+band, y, x2-band, y+thickness))
		fill(image.Rect(x+band, y2-thickness, x2-band, y2))
		fill(image.Rect(x, y+band, x+thickness, y2-band))
		fill(image.Rect(x2-thickness, y+band, x2, y2-band))
		return
	}
	fill(image.Rect(x+band, y, x2-band, y2))
	fill(image.Rect(x, y+band, x+band, y2-band))
	fill(image.Rect(x2-band, y+band, x2, y2-band))
}

// b2f returns 1 for true, 0 for false
func b2f(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

// addPoints adds the closed polygon, moved by (-ox, -oy), to the rasterizer
func addPoints(z *vector.Rasterizer, pts [][2]float64, ox, oy float64) {
	if len(pts) == 0 {
		return
	}
	z.MoveTo(float32(pts[0][0]-ox), float32(pts[0][1]-oy))
	for _, p := range pts[1:] {
		z.LineTo(float32(p[0]-ox), float32(p[1]-oy))
	}
	z.ClosePath()
}

// fillRasterized blends the color over the pixels of b by how much the
// shape added to the rasterizer covers each: it is b's size, its origin
// at b.Min
func (c *Canvas) fillRasterized(ras *rasterizer, b image.Rectangle, col color.Color) {
	mask := ras.mask(b.Dx(), b.Dy())
	ras.z.DrawOp = draw.Src
	ras.z.Draw(mask, mask.Rect, image.Opaque, image.Point{})

	r, g, bl, a := straightRGBA8(col)
	for yy := 0; yy < b.Dy(); yy++ {
		offset := c.rgba.PixOffset(b.Min.X, b.Min.Y+yy)
		for xx, m := range mask.Pix[yy*mask.Stride : yy*mask.Stride+b.Dx()] {
			if m != 0 {
				c.mixPixelAt(offset+xx*4, r, g, bl, (a*uint32(m)+127)/255)
			}
		}
	}
}

// fillPolygonPixels fills the polygon of points in pixels of the image
// (x 0 is the left edge of the first column), anti-aliased, within the clip
func (c *Canvas) fillPolygonPixels(pts [][2]float64, col color.Color) {
	if len(pts) < 3 {
		return
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		minX, minY = math.Min(minX, p[0]), math.Min(minY, p[1])
		maxX, maxY = math.Max(maxX, p[0]), math.Max(maxY, p[1])
	}
	b := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX)), int(math.Ceil(maxY))).
		Intersect(c.clipRect()).Intersect(c.rgba.Rect)
	if b.Empty() {
		return
	}
	ras := rasterizers.Get().(*rasterizer)
	defer rasterizers.Put(ras)
	z := ras.reset(b.Dx(), b.Dy())
	ox, oy := float64(b.Min.X), float64(b.Min.Y)
	z.MoveTo(float32(pts[0][0]-ox), float32(pts[0][1]-oy))
	for _, p := range pts[1:] {
		z.LineTo(float32(p[0]-ox), float32(p[1]-oy))
	}
	z.ClosePath()
	c.fillRasterized(ras, b, col)
}

// drawMark draws a stroke of a mark (a check, a dash) of the logical width,
// which may be fractional, between the logical points: as drawWideLine, so
// a mark keeps its weight at any scale and its ends round
func (c *Canvas) drawMark(x1, y1, x2, y2 int, width float64, col color.Color) {
	fx1, fy1 := c.pixelCenter(float64(x1), float64(y1))
	fx2, fy2 := c.pixelCenter(float64(x2), float64(y2))
	c.drawWideLine(math.Round(fx1), math.Round(fy1), math.Round(fx2), math.Round(fy2), max(1, width*c.scale), col)
}

// drawWideLine draws a line of the width (in pixels) between the
// pixels (x1, y1) and (x2, y2) of the image, both ends included. A
// horizontal or a vertical one is a rectangle of whole pixels, as sharp as
// FillRect; any other one is anti-aliased, with round ends, so the segments
// of a polyline (a check mark) join smoothly. A line of no length draws
// nothing.
func (c *Canvas) drawWideLine(x1, y1, x2, y2 float64, width float64, col color.Color) {
	if x1 == x2 && y1 == y2 {
		return
	}
	if x1 == x2 || y1 == y2 {
		// The pixels of the ends and between them; across, the width
		// rounded to whole pixels around the line: one more on the lower
		// side of an even width
		w := max(1, int(math.Round(width)))
		left, right := int(math.Round(min(x1, x2))), int(math.Round(max(x1, x2)))
		top, bottom := int(math.Round(min(y1, y2))), int(math.Round(max(y1, y2)))
		before := (w - 1) / 2
		if y1 == y2 {
			c.fillRectPixels(left, top-before, right-left+1, w, col)
		} else {
			c.fillRectPixels(left-before, top, w, bottom-top+1, col)
		}
		return
	}
	// The centers of the end pixels; the outline of the line with half
	// circles around them, counterclockwise from the start's left side
	ax, ay, bx, by := x1+0.5, y1+0.5, x2+0.5, y2+0.5
	angle := math.Atan2(by-ay, bx-ax)
	r := width / 2
	steps := min(4+int(width), 16)
	var buf [34][2]float64
	pts := buf[:0]
	for i := 0; i <= steps; i++ { // around the end
		a := angle - math.Pi/2 + math.Pi*float64(i)/float64(steps)
		pts = append(pts, [2]float64{bx + r*math.Cos(a), by + r*math.Sin(a)})
	}
	for i := 0; i <= steps; i++ { // around the start
		a := angle + math.Pi/2 + math.Pi*float64(i)/float64(steps)
		pts = append(pts, [2]float64{ax + r*math.Cos(a), ay + r*math.Sin(a)})
	}
	c.fillPolygonPixels(pts, col)
}

// quarterTrigs are the cosines and sines of steps+1 angles from 0 to 90
// degrees, by the steps
var quarterTrigs [33][][2]float64
var quarterTrigsOnce sync.Once

func quarterTrig(steps int) [][2]float64 {
	quarterTrigsOnce.Do(func() {
		for n := 1; n < len(quarterTrigs); n++ {
			t := make([][2]float64, n+1)
			for i := range t {
				angle := 90 * float64(i) / float64(n) * math.Pi / 180
				t[i] = [2]float64{math.Cos(angle), math.Sin(angle)}
			}
			quarterTrigs[n] = t
		}
	})
	return quarterTrigs[steps]
}

// roundedRectPoints appends the outline of the rectangle with rounded
// corners to pts, clockwise on the screen. The radius is kept within half
// of the sides; the arcs are made of short lines.
func roundedRectPoints(pts [][2]float64, x, y, w, h, radius float64) [][2]float64 {
	radius = max(0, min(radius, w/2, h/2))
	steps := min(4+int(radius)/2, 32)
	corner := func(cx, cy, from float64) {
		if radius == 0 {
			pts = append(pts, [2]float64{cx, cy})
			return
		}
		// The quarter from the angle from, a multiple of 90 degrees: the
		// cosines and sines of the quarter from 0, turned
		trig := quarterTrig(steps)
		for i := 0; i <= steps; i++ {
			cs, sn := trig[i][0], trig[i][1]
			switch from {
			case -90:
				cs, sn = sn, -cs
			case 90:
				cs, sn = -sn, cs
			case 180:
				cs, sn = -cs, -sn
			}
			pts = append(pts, [2]float64{cx + radius*cs, cy + radius*sn})
		}
	}
	// From the top left corner's end
	corner(x+w-radius, y+radius, -90) // top right
	corner(x+w-radius, y+h-radius, 0) // bottom right
	corner(x+radius, y+h-radius, 90)  // bottom left
	corner(x+radius, y+radius, 180)   // top left
	return pts
}

// clipRect returns the clip in pixels of the image
func (c *Canvas) clipRect() image.Rectangle {
	return image.Rect(c.state.clipX, c.state.clipY, c.state.clipX+c.state.clipW, c.state.clipY+c.state.clipH)
}
