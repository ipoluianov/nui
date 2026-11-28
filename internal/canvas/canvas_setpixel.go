package canvas

import "image/color"

func (c *Canvas) SetPixel(x, y float64, a float64) {
	state := c.CurrentState()
	col := state.col

	ix := int(x)
	iy := int(y)

	c.BlendPixel(ix, iy, col)
}

// BlendPixel draws col over the pixel (x, y), counted from the top-left
// corner of the image (its Rect.Min, which a sub-image does not have at 0).
func (c *Canvas) BlendPixel(x, y int, col color.Color) {
	if x < 0 || y < 0 || x >= c.rgba.Rect.Dx() || y >= c.rgba.Rect.Dy() {
		return
	}
	x += c.rgba.Rect.Min.X
	y += c.rgba.Rect.Min.Y

	dst := c.rgba.RGBAAt(x, y)
	// RGBA() is premultiplied by alpha: source over is d*(1-a) + s
	r, g, b, a := col.RGBA()
	k := 1 - float64(a)/65535.0
	blend := func(d uint8, s uint32) uint8 {
		v := float64(d)*k + float64(s)/257.0
		if v >= 255 {
			return 255
		}
		return uint8(v + 0.5)
	}
	c.rgba.SetRGBA(x, y, color.RGBA{
		R: blend(dst.R, r),
		G: blend(dst.G, g),
		B: blend(dst.B, b),
		A: blend(dst.A, a),
	})
}
