package nuicanvas

import "image/color"

func (c *Canvas) SetPixel(x, y float64, a float64) {
	state := c.CurrentState()
	col := state.col

	ix := int(x)
	iy := int(y)

	c.BlendPixel(ix, iy, col)
}

func (c *Canvas) BlendPixel(x, y int, col color.Color) {
	if x < 0 || y < 0 || x >= c.rgba.Bounds().Dx() || y >= c.rgba.Bounds().Dy() {
		return
	}

	dst := c.rgba.RGBAAt(x, y)
	r, g, b, a := col.RGBA()

	sr := float64(r) / 65535.0
	sg := float64(g) / 65535.0
	sb := float64(b) / 65535.0
	sa := float64(a) / 65535.0

	finalAlpha := sa

	dr := float64(dst.R) / 255.0
	dg := float64(dst.G) / 255.0
	db := float64(dst.B) / 255.0

	outR := dr*(1-finalAlpha) + sr*finalAlpha
	outG := dg*(1-finalAlpha) + sg*finalAlpha
	outB := db*(1-finalAlpha) + sb*finalAlpha

	c.rgba.SetRGBA(x, y, color.RGBA{
		R: uint8(outR * 255),
		G: uint8(outG * 255),
		B: uint8(outB * 255),
		A: 255,
	})
}
