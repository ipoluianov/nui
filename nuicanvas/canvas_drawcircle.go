package nuicanvas

func (c *Canvas) DrawCircle(x0, y0 int, r int) {
	x := 0
	y := r
	xsq := 0
	rsq := r * r
	ysq := rsq
	col := c.CurrentState().col
	for x <= y {
		c.BlendPixel(x0+x, y0+y, col)
		c.BlendPixel(x0+y, y0+x, col)

		c.BlendPixel(x0-x, y0+y, col)
		c.BlendPixel(x0+y, y0-x, col)

		c.BlendPixel(x0-x, y0-y, col)
		c.BlendPixel(x0-y, y0-x, col)

		c.BlendPixel(x0+x, y0-y, col)
		c.BlendPixel(x0-y, y0+x, col)

		xsq = xsq + 2*x + 1
		x++
		y1sq := ysq - 2*y + 1
		a := xsq + ysq
		b := xsq + y1sq
		if a-rsq >= rsq-b {
			y--
			ysq = y1sq
		}
	}
}

func (c *Canvas) FillCircle(x, y, radius int) {
	col := c.CurrentState().col
	beginX := x - radius
	beginY := y - radius
	endX := x + radius
	endY := y + radius

	for i := beginX; i <= endX; i++ {
		for j := beginY; j <= endY; j++ {
			if (i-x)*(i-x)+(j-y)*(j-y) <= radius*radius {
				c.BlendPixel(i, j, col)
			}
		}
	}
}
