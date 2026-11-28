package canvas

func (c *Canvas) DrawRect(x, y, w, h float64) {
	for i := float64(1); i < w-1; i += 1 {
		c.SetPixel(x+float64(i), y, 1)
		c.SetPixel(x+float64(i), y+h-1, 1)
	}
	for j := float64(0); j < h; j += 1 {
		c.SetPixel(x, y+float64(j), 1)
		c.SetPixel(x+w-1, y+float64(j), 1)
	}
}

func (c *Canvas) FillRect(x, y, w, h int, alpha float64) {
	// The color is premultiplied: alpha scales all of its channels
	if !(alpha > 0) {
		return
	}
	alpha = min(alpha, 1)
	col := c.CurrentState().col
	col.R = uint8(float64(col.R)*alpha + 0.5)
	col.G = uint8(float64(col.G)*alpha + 0.5)
	col.B = uint8(float64(col.B)*alpha + 0.5)
	col.A = uint8(float64(col.A)*alpha + 0.5)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c.BlendPixel(x+i, y+j, col)
		}
	}
}
