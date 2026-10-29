package ui

import (
	"image/color"
	"math"
)

// MarkerShape is the shape of a marker drawn by Canvas.DrawMarker: the
// points of a scatter chart, the samples of a line, the entries of a
// legend, the status dots of a table
type MarkerShape int

const (
	MarkerCircle MarkerShape = iota
	MarkerSquare
	MarkerDiamond
	MarkerTriangleUp
	MarkerTriangleDown
	// MarkerCross (×) and MarkerPlus (+) are lines: stroked only
	MarkerCross
	MarkerPlus
	MarkerStar
)

// DrawMarker draws a marker of the shape and the size (the logical width
// of the shape) centered at (x, y): filled with fill and outlined, a
// logical pixel wide, with stroke. Either color may be nil: not filled or
// not outlined. The line shapes (MarkerCross, MarkerPlus) are stroked with
// stroke, or with fill when stroke is nil, 1.5 logical pixels wide.
func (c *Canvas) DrawMarker(shape MarkerShape, x, y, size float64, fill, stroke color.Color) {
	if size <= 0 {
		return
	}
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	r := size / 2
	switch shape {
	case MarkerCross, MarkerPlus:
		col := stroke
		if col == nil {
			col = fill
		}
		if col == nil {
			return
		}
		if shape == MarkerCross {
			d := r * math.Sqrt2 / 2 * 1.1
			p.MoveTo(x-d, y-d)
			p.LineTo(x+d, y+d)
			p.MoveTo(x-d, y+d)
			p.LineTo(x+d, y-d)
		} else {
			p.MoveTo(x-r, y)
			p.LineTo(x+r, y)
			p.MoveTo(x, y-r)
			p.LineTo(x, y+r)
		}
		c.StrokePath(p, StrokeStyle{Width: 1.5}, col)
		return
	case MarkerCircle:
		p.Circle(x, y, r)
	case MarkerSquare:
		// A square looks bigger than a circle of the same width: a bit smaller
		s := r * 0.9
		p.Rect(x-s, y-s, s*2, s*2)
	case MarkerDiamond:
		s := r * 1.15
		p.Polygon(PointF{x, y - s}, PointF{x + s, y}, PointF{x, y + s}, PointF{x - s, y})
	case MarkerTriangleUp, MarkerTriangleDown:
		// An equilateral triangle centered on its centroid
		// s is half the side; the height of the triangle is s*sqrt(3)
		s := r * 1.1
		h := s * math.Sqrt(3)
		dir := 1.0
		if shape == MarkerTriangleDown {
			dir = -1
		}
		p.Polygon(
			PointF{x, y - dir*h*2/3},
			PointF{x + s, y + dir*h/3},
			PointF{x - s, y + dir*h/3},
		)
	case MarkerStar:
		outer, inner := r*1.15, r*0.5
		for i := 0; i < 10; i++ {
			rr := outer
			if i%2 == 1 {
				rr = inner
			}
			a := -math.Pi/2 + math.Pi*float64(i)/5
			if i == 0 {
				p.MoveTo(x+rr*math.Cos(a), y+rr*math.Sin(a))
			} else {
				p.LineTo(x+rr*math.Cos(a), y+rr*math.Sin(a))
			}
		}
		p.Close()
	default:
		return
	}
	if fill != nil {
		c.FillPath(p, fill)
	}
	if stroke != nil {
		c.StrokePath(p, StrokeStyle{Width: 1}, stroke)
	}
}
