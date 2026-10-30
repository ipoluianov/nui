package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// NewImageCanvas makes an image of width x height logical pixels at the
// scale (1 - one pixel each) and a canvas that draws on it, clipped to
// it: to draw off the screen, a picture to save or to draw again later.
// The image is transparent; what is drawn on it keeps its alpha, so it
// can be drawn over something else (DrawImage, DrawImageTransformed).
func NewImageCanvas(width, height int, scale float64) (*Canvas, *image.RGBA) {
	if scale <= 0 {
		scale = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, int(math.Ceil(float64(width)*scale)), int(math.Ceil(float64(height)*scale))))
	cnv := NewCanvasScaled(img, scale)
	cnv.SetDirectTranslateAndClip(0, 0, width, height)
	return cnv, img
}

// Clear sets every pixel within the clip to the color, its alpha too,
// without blending: col transparent makes them transparent. The transform
// is not applied.
func (c *Canvas) Clear(col color.Color) {
	b := c.clipRect().Intersect(c.rgba.Rect)
	if b.Empty() {
		return
	}
	r, g, bl, a := col.RGBA() // premultiplied, as the image keeps it
	px := [4]uint8{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), uint8(a >> 8)}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		off := c.rgba.PixOffset(b.Min.X, y)
		row := c.rgba.Pix[off : off+b.Dx()*4]
		for i := 0; i < len(row); i += 4 {
			copy(row[i:i+4], px[:])
		}
	}
}

// PixelAt returns the color of the logical pixel (x, y), from the current
// origin, straight (not premultiplied); outside the image it is
// transparent. The transform is not applied.
func (c *Canvas) PixelAt(x, y int) color.RGBA {
	px := c.state.translateX + c.scaled(x)
	py := c.state.translateY + c.scaled(y)
	if !(image.Point{px, py}).In(c.rgba.Rect) {
		return color.RGBA{}
	}
	p := c.rgba.Pix[c.rgba.PixOffset(px, py):]
	a := uint32(p[3])
	if a == 0 {
		return color.RGBA{}
	}
	if a == 255 {
		return color.RGBA{p[0], p[1], p[2], 255}
	}
	return color.RGBA{uint8((uint32(p[0])*255 + a/2) / a), uint8((uint32(p[1])*255 + a/2) / a), uint8((uint32(p[2])*255 + a/2) / a), uint8(a)}
}

// pixelBox returns the pixels of the logical rectangle within the clip and
// the image
func (c *Canvas) pixelBox(x, y, w, h int) image.Rectangle {
	px, py, pw, ph := c.pixelRect(x, y, w, h)
	return image.Rect(px, py, px+pw, py+ph).Intersect(c.clipRect()).Intersect(c.rgba.Rect)
}

// AdjustPixels replaces the color of every pixel of the logical rectangle
// within the clip with what f returns for it: a filter of the
// application's own. The colors are straight (not premultiplied).
//
//	// Gray
//	cnv.AdjustPixels(0, 0, w, h, func(c color.RGBA) color.RGBA {
//		l := uint8((299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000)
//		return color.RGBA{l, l, l, c.A}
//	})
func (c *Canvas) AdjustPixels(x, y, w, h int, f func(color.RGBA) color.RGBA) {
	b := c.pixelBox(x, y, w, h)
	for py := b.Min.Y; py < b.Max.Y; py++ {
		off := c.rgba.PixOffset(b.Min.X, py)
		for px := b.Min.X; px < b.Max.X; px, off = px+1, off+4 {
			p := c.rgba.Pix[off : off+4 : off+4]
			in := color.RGBA{p[0], p[1], p[2], p[3]}
			if a := uint32(p[3]); a != 0 && a != 255 {
				in.R = uint8((uint32(p[0])*255 + a/2) / a)
				in.G = uint8((uint32(p[1])*255 + a/2) / a)
				in.B = uint8((uint32(p[2])*255 + a/2) / a)
			}
			out := f(in)
			a := uint32(out.A)
			p[0] = uint8((uint32(out.R)*a + 127) / 255)
			p[1] = uint8((uint32(out.G)*a + 127) / 255)
			p[2] = uint8((uint32(out.B)*a + 127) / 255)
			p[3] = out.A
		}
	}
}

// Blur blurs the pixels of the logical rectangle within the clip, as by a
// Gaussian blur of the radius in logical pixels (its standard deviation
// radius/2): the frosted glass behind a dialog, a background out of focus.
// Three box blurs make it, so the cost does not grow with the radius.
func (c *Canvas) Blur(x, y, w, h int, radius float64) {
	b := c.pixelBox(x, y, w, h)
	if b.Empty() || radius <= 0 {
		return
	}
	sigma := radius * c.scale / 2
	bw, bh := b.Dx(), b.Dy()
	buf := make([]uint32, bw*bh*4)
	for yy := 0; yy < bh; yy++ {
		off := c.rgba.PixOffset(b.Min.X, b.Min.Y+yy)
		for i, v := range c.rgba.Pix[off : off+bw*4] {
			buf[yy*bw*4+i] = uint32(v)
		}
	}
	blurChannels(buf, bw, bh, 4, sigma)
	for yy := 0; yy < bh; yy++ {
		off := c.rgba.PixOffset(b.Min.X, b.Min.Y+yy)
		row := c.rgba.Pix[off : off+bw*4]
		for i := range row {
			row[i] = uint8(buf[yy*bw*4+i])
		}
	}
}

// DrawShadow draws the soft shadow of a rounded rectangle: the shape of
// x, y, w, h and the corner radius, of the color, blurred by blur logical
// pixels around it. Draw it before the card, the popup or the button it
// is the shadow of, moved a little down (y + 2) for a light from above.
// The transform is not applied.
func (c *Canvas) DrawShadow(x, y, w, h, radius, blur float64, col color.Color) {
	cr, cg, cb, ca := straightRGBA8(col)
	if ca == 0 || w <= 0 || h <= 0 || c.clipEmpty() {
		return
	}
	s := c.scale
	sigma := max(blur, 0) * s / 2
	pad := int(math.Ceil(sigma*3)) + 1
	// The shape in pixels, in a mask with room for the blur around it
	x0 := float64(c.state.translateX) + x*s
	y0 := float64(c.state.translateY) + y*s
	ox, oy := int(math.Floor(x0))-pad, int(math.Floor(y0))-pad
	mw, mh := int(math.Ceil(w*s))+pad*2+1, int(math.Ceil(h*s))+pad*2+1
	area := image.Rect(ox, oy, ox+mw, oy+mh)
	visible := area.Intersect(c.clipRect()).Intersect(c.rgba.Rect)
	if visible.Empty() || mw*mh > 1<<24 {
		return
	}
	ras := rasterizers.Get().(*rasterizer)
	defer rasterizers.Put(ras)
	z := ras.reset(mw, mh)
	var buf [4 * 33][2]float64
	addPoints(z, roundedRectPoints(buf[:0], x0, y0, w*s, h*s, radius*s), float64(ox), float64(oy))
	mask := ras.mask(mw, mh)
	z.DrawOp = draw.Src
	z.Draw(mask, mask.Rect, image.Opaque, image.Point{})
	if sigma > 0 {
		vals := make([]uint32, mw*mh)
		for i, v := range mask.Pix {
			vals[i] = uint32(v)
		}
		blurChannels(vals, mw, mh, 1, sigma)
		for i, v := range vals {
			mask.Pix[i] = uint8(v)
		}
	}
	for py := visible.Min.Y; py < visible.Max.Y; py++ {
		off := c.rgba.PixOffset(visible.Min.X, py)
		row := mask.Pix[(py-oy)*mw:]
		for px := visible.Min.X; px < visible.Max.X; px, off = px+1, off+4 {
			if m := uint32(row[px-ox]); m != 0 {
				c.mixPixelAt(off, cr, cg, cb, (ca*m+127)/255)
			}
		}
	}
}

// blurChannels blurs the image of w x h pixels of n channels in buf (0..255
// each) by three box blurs that make a Gaussian of the sigma, in pixels
func blurChannels(buf []uint32, w, h, n int, sigma float64) {
	for _, r := range boxesForGauss(sigma) {
		if r <= 0 {
			continue
		}
		boxBlurH(buf, w, h, n, r)
		boxBlurV(buf, w, h, n, r)
	}
}

// boxesForGauss returns the radii of three box blurs that together blur
// as a Gaussian of the sigma (W. Jarosz, "Fast image convolutions")
func boxesForGauss(sigma float64) [3]int {
	const n = 3
	wIdeal := math.Sqrt(12*sigma*sigma/n + 1)
	wl := int(math.Floor(wIdeal))
	if wl%2 == 0 {
		wl--
	}
	wu := wl + 2
	mIdeal := (12*sigma*sigma - float64(n*wl*wl) - float64(4*n*wl) - 3*n) / float64(-4*wl-4)
	m := int(math.Round(mIdeal))
	var r [3]int
	for i := 0; i < n; i++ {
		if i < m {
			r[i] = (wl - 1) / 2
		} else {
			r[i] = (wu - 1) / 2
		}
	}
	return r
}

// boxBlurH averages every pixel with r on each side of it along the rows,
// the edge pixels repeated past the edges
func boxBlurH(buf []uint32, w, h, n, r int) {
	tmp := make([]uint32, w*n)
	div := uint32(2*r + 1)
	for y := 0; y < h; y++ {
		row := buf[y*w*n : (y+1)*w*n]
		for ch := 0; ch < n; ch++ {
			at := func(x int) uint32 { return row[max(0, min(x, w-1))*n+ch] }
			var sum uint32
			for x := -r; x <= r; x++ {
				sum += at(x)
			}
			for x := 0; x < w; x++ {
				tmp[x*n+ch] = (sum + div/2) / div
				sum += at(x+r+1) - at(x-r)
			}
		}
		copy(row, tmp)
	}
}

// boxBlurV is boxBlurH along the columns
func boxBlurV(buf []uint32, w, h, n, r int) {
	tmp := make([]uint32, h)
	div := uint32(2*r + 1)
	for x := 0; x < w; x++ {
		for ch := 0; ch < n; ch++ {
			at := func(y int) uint32 { return buf[(max(0, min(y, h-1))*w+x)*n+ch] }
			var sum uint32
			for y := -r; y <= r; y++ {
				sum += at(y)
			}
			for y := 0; y < h; y++ {
				tmp[y] = (sum + div/2) / div
				sum += at(y+r+1) - at(y-r)
			}
			for y := 0; y < h; y++ {
				buf[(y*w+x)*n+ch] = tmp[y]
			}
		}
	}
}

// FloodFill fills the area of the pixels around the logical pixel (x, y)
// that have its color, within tolerance (0..255, the largest difference
// of a component), with the color: the bucket of a paint program. Only the
// clip is filled; the transform is not applied.
func (c *Canvas) FloodFill(x, y int, col color.Color, tolerance int) {
	b := c.clipRect().Intersect(c.rgba.Rect)
	sx := c.state.translateX + c.scaled(x)
	sy := c.state.translateY + c.scaled(y)
	if !(image.Point{sx, sy}).In(b) {
		return
	}
	cr, cg, cb, ca := straightRGBA8(col)
	seedOff := c.rgba.PixOffset(sx, sy)
	var seed [4]uint8
	copy(seed[:], c.rgba.Pix[seedOff:seedOff+4])
	tol := max(0, tolerance)
	match := func(off int) bool {
		p := c.rgba.Pix[off : off+4 : off+4]
		for i := 0; i < 4; i++ {
			d := int(p[i]) - int(seed[i])
			if d < -tol || d > tol {
				return false
			}
		}
		return true
	}
	w := b.Dx()
	done := make([]bool, w*b.Dy())
	type span struct{ x, y int }
	stack := []span{{sx, sy}}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if done[(s.y-b.Min.Y)*w+s.x-b.Min.X] || !match(c.rgba.PixOffset(s.x, s.y)) {
			continue
		}
		// The run of matching pixels of the row through the point
		l, r := s.x, s.x
		for l > b.Min.X && !done[(s.y-b.Min.Y)*w+l-1-b.Min.X] && match(c.rgba.PixOffset(l-1, s.y)) {
			l--
		}
		for r < b.Max.X-1 && !done[(s.y-b.Min.Y)*w+r+1-b.Min.X] && match(c.rgba.PixOffset(r+1, s.y)) {
			r++
		}
		for xx := l; xx <= r; xx++ {
			done[(s.y-b.Min.Y)*w+xx-b.Min.X] = true
		}
		// The rows above and below: a seed at every start of a run
		for _, ny := range [2]int{s.y - 1, s.y + 1} {
			if ny < b.Min.Y || ny >= b.Max.Y {
				continue
			}
			in := false
			for xx := l; xx <= r; xx++ {
				ok := !done[(ny-b.Min.Y)*w+xx-b.Min.X] && match(c.rgba.PixOffset(xx, ny))
				if ok && !in {
					stack = append(stack, span{xx, ny})
				}
				in = ok
			}
		}
	}
	// Painted after the search: the matching compares the old colors
	for yy := b.Min.Y; yy < b.Max.Y; yy++ {
		off := c.rgba.PixOffset(b.Min.X, yy)
		for xx := 0; xx < w; xx, off = xx+1, off+4 {
			if done[(yy-b.Min.Y)*w+xx] {
				c.mixPixelAt(off, cr, cg, cb, ca)
			}
		}
	}
}
