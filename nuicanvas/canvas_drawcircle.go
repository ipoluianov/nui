package nuicanvas

/*
func (c *Canvas) FillCircle(x, y, radius int) {
	for i := 0; i < c.Width(); i++ {
		for j := 0; j < c.Height(); j++ {
			if (i-x)*(i-x)+(j-y)*(j-y) <= radius*radius {
				c.SetPixel(i, j)
			}
		}
	}
}

func (c *Canvas) DrawCircle(x0, y0, radius int) {
	x := radius
	y := 0
	err := 0

	for x >= y {
		c.SetPixel(x0+x, y0+y)
		c.SetPixel(x0+y, y0+x)
		c.SetPixel(x0-y, y0+x)
		c.SetPixel(x0-x, y0+y)
		c.SetPixel(x0-x, y0-y)
		c.SetPixel(x0-y, y0-x)
		c.SetPixel(x0+y, y0-x)
		c.SetPixel(x0+x, y0-y)

		y++
		err += 1 + 2*y
		if 2*(err-x)+1 > 0 {
			x--
			err += 1 - 2*x
		}
	}
}

func (c *Canvas) DrawCircleAA(x0, y0, radius int) {
	steps := radius * 2

	for i := 0; i < steps; i++ {
		theta := 2 * math.Pi * float64(i) / float64(steps)
		x := float64(radius) * math.Cos(theta)
		y := float64(radius) * math.Sin(theta)

		ix := int(x)
		iy := int(y)

		fx := x - float64(ix)
		fy := y - float64(iy)

		c.SetPixelAlpha(x0+ix, y0+iy, (1-fx)*(1-fy))
		c.SetPixelAlpha(x0+ix+1, y0+iy, fx*(1-fy))
		c.SetPixelAlpha(x0+ix, y0+iy+1, (1-fx)*fy)
		c.SetPixelAlpha(x0+ix+1, y0+iy+1, fx*fy)
	}
}
*/
