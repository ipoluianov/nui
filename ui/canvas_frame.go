package ui

import (
	"image"
	"image/color"
	"math"
	"slices"

	"golang.org/x/image/vector"
)

// FillRoundedRectAA fills a rounded rectangle with antialiased corners, of
// the color given (FillRoundedRect fills with the current color).
func (c *Canvas) FillRoundedRectAA(x, y, width, height, radius int, col color.RGBA) {
	if width <= 0 || height <= 0 {
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
	for cy := 0; cy < radius; cy++ {
		for cx := 0; cx < radius; cx++ {
			coverage := cornerCoverage(cx, cy, radius)
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

// blendPixel blends the straight (not premultiplied) color into the pixel
// (x, y) of the image, inside the clip rectangle only.
func (c *Canvas) blendPixel(x, y int, col color.RGBA) {
	if x < c.state.clipX || x >= c.state.clipX+c.state.clipW || y < c.state.clipY || y >= c.state.clipY+c.state.clipH {
		return
	}
	if !(image.Point{x, y}).In(c.rgba.Rect) {
		return
	}
	if col.A == 255 {
		c.rgba.SetRGBA(x, y, col)
		return
	}
	c.MixPixel(x, y, col)
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
	z := vector.NewRasterizer(b.Dx(), b.Dy())
	ox, oy := float64(x-b.Min.X), float64(y-b.Min.Y)
	t := float64(thickness)
	if thickness > 0 && w > thickness*2 && h > thickness*2 {
		roundedRectPath(z, ox, oy, float64(w), float64(h), radius+t/2, false)
		// The inner part the other way round: it is taken out
		roundedRectPath(z, ox+t, oy+t, float64(w)-t*2, float64(h)-t*2, radius-t/2, true)
	} else {
		roundedRectPath(z, ox, oy, float64(w), float64(h), radius, false)
	}
	c.fillRasterized(z, b, col)
}

// fillRasterized blends the color over the pixels of b by how much the
// shape added to z covers each: z is b's size, its origin at b.Min
func (c *Canvas) fillRasterized(z *vector.Rasterizer, b image.Rectangle, col color.Color) {
	mask := image.NewAlpha(image.Rect(0, 0, b.Dx(), b.Dy()))
	z.Draw(mask, mask.Rect, image.Opaque, image.Point{})

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
	z := vector.NewRasterizer(b.Dx(), b.Dy())
	ox, oy := float64(b.Min.X), float64(b.Min.Y)
	z.MoveTo(float32(pts[0][0]-ox), float32(pts[0][1]-oy))
	for _, p := range pts[1:] {
		z.LineTo(float32(p[0]-ox), float32(p[1]-oy))
	}
	z.ClosePath()
	c.fillRasterized(z, b, col)
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
	pts := make([][2]float64, 0, (steps+1)*2)
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

// roundedRectPath adds the rectangle with rounded corners to the path of
// the rasterizer, clockwise or, reversed, counterclockwise. The radius is
// kept within half of the sides; the arcs are made of short lines.
func roundedRectPath(z *vector.Rasterizer, x, y, w, h, radius float64, reversed bool) {
	radius = max(0, min(radius, w/2, h/2))
	steps := min(4+int(radius)/2, 32)
	var pts [][2]float64
	corner := func(cx, cy, from float64) {
		if radius == 0 {
			pts = append(pts, [2]float64{cx, cy})
			return
		}
		for i := 0; i <= steps; i++ {
			angle := (from + 90*float64(i)/float64(steps)) * math.Pi / 180
			pts = append(pts, [2]float64{cx + radius*math.Cos(angle), cy + radius*math.Sin(angle)})
		}
	}
	// Clockwise on the screen (y down): from the top left corner's end
	corner(x+w-radius, y+radius, -90) // top right
	corner(x+w-radius, y+h-radius, 0) // bottom right
	corner(x+radius, y+h-radius, 90)  // bottom left
	corner(x+radius, y+radius, 180)   // top left
	if reversed {
		slices.Reverse(pts)
	}
	z.MoveTo(float32(pts[0][0]), float32(pts[0][1]))
	for _, p := range pts[1:] {
		z.LineTo(float32(p[0]), float32(p[1]))
	}
	z.ClosePath()
}

// clipRect returns the clip in pixels of the image
func (c *Canvas) clipRect() image.Rectangle {
	return image.Rect(c.state.clipX, c.state.clipY, c.state.clipX+c.state.clipW, c.state.clipY+c.state.clipH)
}
