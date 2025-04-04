package nuicanvas

func (c *Canvas) DrawLine(x0, y0, x1, y1 int, alpha float64) {
	col := c.CurrentState().col
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy

	for {
		c.BlendPixel(x0, y0, col, alpha) // Рисуем текущую точку

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err

		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
