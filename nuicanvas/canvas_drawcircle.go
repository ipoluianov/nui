package nuicanvas

import "image/color"

func (c *Canvas) DrawCircle(x0, y0, radius int, alpha float64) {
	x := radius
	y := 0
	err := 0

	col := c.CurrentState().col

	for x >= y {
		c.plotCirclePoints(x0, y0, x, y, col, alpha)
		y++
		if err <= 0 {
			err += 2*y + 1
		}
		if err > 0 {
			x--
			err -= 2*x + 1
		}
	}
}

func (c *Canvas) plotCirclePoints(cx, cy, x, y int, col color.Color, alpha float64) {
	c.BlendPixel(cx+x, cy+y, col, alpha)
	c.BlendPixel(cx-x, cy+y, col, alpha)
	c.BlendPixel(cx+x, cy-y, col, alpha)
	c.BlendPixel(cx-x, cy-y, col, alpha)
	c.BlendPixel(cx+y, cy+x, col, alpha)
	c.BlendPixel(cx-y, cy+x, col, alpha)
	c.BlendPixel(cx+y, cy-x, col, alpha)
	c.BlendPixel(cx-y, cy-x, col, alpha)
}

func (c *Canvas) FillCircle(x0, y0, radius int, alpha float64) {
	x := radius
	y := 0
	err := 0

	col := c.CurrentState().col

	for x >= y {
		c.drawHLine(x0-x, x0+x, y0+y, col, alpha)
		c.drawHLine(x0-x, x0+x, y0-y, col, alpha)
		c.drawHLine(x0-y, x0+y, y0+x, col, alpha)
		c.drawHLine(x0-y, x0+y, y0-x, col, alpha)

		y++
		if err <= 0 {
			err += 2*y + 1
		}
		if err > 0 {
			x--
			err -= 2*x + 1
		}
	}
}

func (c *Canvas) drawHLine(x1, x2, y int, col color.Color, alpha float64) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if x1 < 0 {
		x1 = 0
	}
	for x := x1; x <= x2; x++ {
		c.BlendPixel(x, y, col, alpha)
	}
}
