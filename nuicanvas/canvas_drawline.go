package nuicanvas

import "math"

type LineCapStyle int

const (
	LineCapRound LineCapStyle = iota
	LineCapSquare
)

func (c *Canvas) DrawLineSDF(x0, y0, x1, y1, thickness float64, capStyle LineCapStyle) {
	if thickness <= 0 {
		return
	}

	dx := x1 - x0
	dy := y1 - y0
	length := math.Hypot(dx, dy) // length of the line
	if length == 0 {
		return
	}

	// normalize the direction vector
	nx := dx / length
	ny := dy / length

	// adjust the start and end points if the cap style is square
	if capStyle == LineCapSquare {
		hx := nx * (thickness / 2)
		hy := ny * (thickness / 2)
		x0 -= hx
		y0 -= hy
		x1 += hx
		y1 += hy
		dx = x1 - x0
		dy = y1 - y0
		length = math.Hypot(dx, dy)
	}

	// calculate the bounding box of the line
	minX := int(math.Floor(min(x0, x1) - thickness))
	maxX := int(math.Ceil(max(x0, x1) + thickness))
	minY := int(math.Floor(min(y0, y1) - thickness))
	maxY := int(math.Ceil(max(y0, y1) + thickness))

	// iterate over the bounding box
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {

			// calculate the distance between the current pixel and the line
			px := float64(x) + 0.5
			py := float64(y) + 0.5
			t := ((px-x0)*dx + (py-y0)*dy) / (length * length)
			if t < 0 || t > 1 {
				// the projection of the point is outside the line segment,
				continue
			}

			// calculate the nearest point on the line
			nx := x0 + t*dx
			ny := y0 + t*dy

			// calculate the distance between the current pixel and the line
			dist := math.Hypot(px-nx, py-ny)
			blur := 1.0
			alpha := 1.0 - smoothstep(thickness, thickness+blur, dist)
			if alpha > 0.1 {
				c.SetPixel(math.Round(px), math.Round(py), alpha)
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
