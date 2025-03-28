package nuicanvas

import "math"

func (c *Canvas) DrawLineSDF(x0, y0, x1, y1 float64, thickness float64) {
	length := math.Hypot(x1-x0, y1-y0)
	if length == 0 {
		return
	}

	dx := x1 - x0
	dy := y1 - y0

	minX := int(math.Floor(min(x0, x1) - thickness))
	maxX := int(math.Ceil(max(x0, x1) + thickness))
	minY := int(math.Floor(min(y0, y1) - thickness))
	maxY := int(math.Ceil(max(y0, y1) + thickness))

	if minX < 0 {
		minX = 0
	}
	if maxX > c.Width() {
		maxX = c.Width()
	}
	if minY < 0 {
		minY = 0
	}
	if maxY > c.Height() {
		maxY = c.Height()
	}

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5

			t := clamp(((px-x0)*dx+(py-y0)*dy)/(length*length), 0, 1)
			nx := x0 + t*dx
			ny := y0 + t*dy

			dist := math.Hypot(px-nx, py-ny)
			alpha := 1.0 - smoothstep(thickness-1, thickness, dist)
			if alpha > 0 {
				c.SetPixel(float64(x), float64(y), alpha)
			}
		}
	}
}

func clamp(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}
