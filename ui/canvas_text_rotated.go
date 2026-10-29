package ui

import (
	"image"
	"image/color"
	"math"
	"sync"
)

// DrawTextRotated draws a line of text turned by the angle in radians
// around the anchor (x, y), in the current font and color: math.Pi/2 turns
// it clockwise to read downwards, -math.Pi/2 to read upwards (the title of
// a vertical axis). The anchor is the point of the text that the current
// alignment names: SetHAlign picks its left, center or right, SetVAlign its
// top, middle or bottom. The text is drawn at the size it takes in pixels;
// a turn by a quarter is exact, other angles are smoothed.
//
//	cnv.SetHAlign(ui.HAlignCenter)
//	cnv.SetVAlign(ui.VAlignCenter)
//	cnv.DrawTextRotated(12, height/2, -math.Pi/2, "Pressure, kPa")
func (c *Canvas) DrawTextRotated(x, y float64, angle float64, text string) {
	if text == "" || c.clipEmpty() {
		return
	}
	fontSize := c.state.fontSize * c.scale
	w, h, err := measureText(c.state.fontFamily, fontSize, text)
	if err != nil || w <= 0 || h <= 0 {
		return
	}
	// The text as a mask: white on black, its coverage in the components
	buf := textMasks.Get().(*image.RGBA)
	defer textMasks.Put(buf)
	if cap(buf.Pix) < w*h*4 {
		buf.Pix = make([]uint8, w*h*4)
	}
	buf.Pix = buf.Pix[:w*h*4]
	buf.Stride = w * 4
	buf.Rect = image.Rect(0, 0, w, h)
	for i := 0; i < len(buf.Pix); i += 4 {
		buf.Pix[i], buf.Pix[i+1], buf.Pix[i+2], buf.Pix[i+3] = 0, 0, 0, 255
	}
	DrawText(buf, text, color.RGBA{255, 255, 255, 255}, c.state.fontFamily, fontSize, 0, 0, 0, 0, w, h)
	coverage := func(sx, sy int) float64 {
		if sx < 0 || sy < 0 || sx >= w || sy >= h {
			return 0
		}
		i := sy*buf.Stride + sx*4
		// A system font may draw with subpixel antialiasing: the mean of the components
		return (float64(buf.Pix[i]) + float64(buf.Pix[i+1]) + float64(buf.Pix[i+2])) / (3 * 255)
	}

	// The anchor in the text's own pixels
	var ax, ay float64
	switch c.state.hAlign {
	case HAlignCenter:
		ax = float64(w) / 2
	case HAlignRight:
		ax = float64(w)
	}
	switch c.state.vAlign {
	case VAlignCenter:
		ay = float64(h) / 2
	case VAlignBottom:
		ay = float64(h)
	}
	// The anchor on whole pixels of the text and of the image: a text
	// turned by a quarter is as sharp as unturned text
	ax, ay = math.Round(ax), math.Round(ay)
	px := math.Round(float64(c.state.translateX) + x*c.scale)
	py := math.Round(float64(c.state.translateY) + y*c.scale)
	if c.rec != nil {
		c.rec.text(c, text, x-ax/c.scale, y-ay/c.scale, float64(h)/c.scale, angle, x, y)
	}
	cos, sin := math.Cos(angle), math.Sin(angle)
	// A quarter turn: exact, without smoothing
	quarter := -1
	for q := 0; q < 4; q++ {
		if math.Abs(math.Remainder(angle-float64(q)*math.Pi/2, 2*math.Pi)) < 1e-9 {
			quarter = q
			cos, sin = [4]float64{1, 0, -1, 0}[q], [4]float64{0, 1, 0, -1}[q]
		}
	}
	// The bounds of the turned text around the anchor
	corners := [4][2]float64{{-ax, -ay}, {float64(w) - ax, -ay}, {float64(w) - ax, float64(h) - ay}, {-ax, float64(h) - ay}}
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, cr := range corners {
		rx := cr[0]*cos - cr[1]*sin
		ry := cr[0]*sin + cr[1]*cos
		minX, maxX = math.Min(minX, rx), math.Max(maxX, rx)
		minY, maxY = math.Min(minY, ry), math.Max(maxY, ry)
	}
	b := image.Rect(int(math.Floor(px+minX))-1, int(math.Floor(py+minY))-1, int(math.Ceil(px+maxX))+1, int(math.Ceil(py+maxY))+1).
		Intersect(c.clipRect()).Intersect(c.rgba.Rect)
	if b.Empty() {
		return
	}
	r, g, bl, a := straightRGBA8(c.state.col)
	for yy := b.Min.Y; yy < b.Max.Y; yy++ {
		offset := c.rgba.PixOffset(b.Min.X, yy)
		for xx := b.Min.X; xx < b.Max.X; xx++ {
			// The center of the pixel turned back into the text
			dx, dy := float64(xx)+0.5-px, float64(yy)+0.5-py
			u := dx*cos + dy*sin + ax
			v := -dx*sin + dy*cos + ay
			var cov float64
			if quarter >= 0 {
				cov = coverage(int(math.Floor(u)), int(math.Floor(v)))
			} else {
				// Bilinear between the four text pixels around the point
				fu, fv := u-0.5, v-0.5
				iu, iv := int(math.Floor(fu)), int(math.Floor(fv))
				tu, tv := fu-float64(iu), fv-float64(iv)
				cov = coverage(iu, iv)*(1-tu)*(1-tv) + coverage(iu+1, iv)*tu*(1-tv) +
					coverage(iu, iv+1)*(1-tu)*tv + coverage(iu+1, iv+1)*tu*tv
			}
			if cov <= 0 {
				continue
			}
			c.mixPixelAt(offset+(xx-b.Min.X)*4, r, g, bl, uint32(float64(a)*cov+0.5))
		}
	}
}

var textMasks = sync.Pool{New: func() any { return &image.RGBA{} }}

// MeasureTextRotated returns the logical width and height of the box that
// holds the line of text turned by the angle, as DrawTextRotated draws it
func (c *Canvas) MeasureTextRotated(text string, angle float64) (width, height float64) {
	w, h, err := measureText(c.state.fontFamily, c.state.fontSize*c.scale, text)
	if err != nil {
		return 0, 0
	}
	cos, sin := math.Abs(math.Cos(angle)), math.Abs(math.Sin(angle))
	fw, fh := float64(w)/c.scale, float64(h)/c.scale
	return fw*cos + fh*sin, fw*sin + fh*cos
}
