package paint

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/fogleman/gg"
)

// newDocument returns a white image of the size.
func newDocument(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	return img
}

// cloneImage returns a copy of the image.
func cloneImage(img *image.RGBA) *image.RGBA {
	c := image.NewRGBA(img.Bounds())
	copy(c.Pix, img.Pix)
	return c
}

// toRGBA converts a decoded image to *image.RGBA starting at 0,0.
func toRGBA(src image.Image) *image.RGBA {
	b := src.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(img, img.Bounds(), src, b.Min, draw.Src)
	return img
}

// drawPixelLine draws a one-pixel line without antialiasing (Bresenham),
// as the pencil does.
func drawPixelLine(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	e := dx + dy
	for {
		if image.Pt(x0, y0).In(img.Bounds()) {
			img.SetRGBA(x0, y0, col)
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		if e2 := 2 * e; e2 >= dy {
			e += dy
			x0 += sx
		} else {
			e += dx
			y0 += sy
		}
	}
}

// drawStroke draws a thick line segment with round ends: a brush stroke.
func drawStroke(img *image.RGBA, x0, y0, x1, y1 int, width float64, col color.RGBA) {
	dc := gg.NewContextForRGBA(img)
	dc.SetColor(col)
	dc.SetLineWidth(width)
	dc.SetLineCapRound()
	dc.DrawLine(float64(x0)+0.5, float64(y0)+0.5, float64(x1)+0.5, float64(y1)+0.5)
	dc.Stroke()
}

// shapeKind is a figure drawn by dragging from one corner to the other.
type shapeKind int

const (
	shapeLine shapeKind = iota
	shapeRect
	shapeEllipse
)

// drawShape draws a line, rectangle or ellipse between two points; a
// rectangle or ellipse is filled when fill is true.
func drawShape(img *image.RGBA, kind shapeKind, p0, p1 image.Point, width float64, col color.RGBA, fill bool) {
	dc := gg.NewContextForRGBA(img)
	dc.SetColor(col)
	dc.SetLineWidth(width)
	dc.SetLineCapRound()
	x0, y0 := float64(p0.X)+0.5, float64(p0.Y)+0.5
	x1, y1 := float64(p1.X)+0.5, float64(p1.Y)+0.5
	switch kind {
	case shapeLine:
		dc.DrawLine(x0, y0, x1, y1)
		dc.Stroke()
		return
	case shapeRect:
		dc.DrawRectangle(math.Min(x0, x1), math.Min(y0, y1), math.Abs(x1-x0), math.Abs(y1-y0))
	case shapeEllipse:
		dc.DrawEllipse((x0+x1)/2, (y0+y1)/2, math.Abs(x1-x0)/2, math.Abs(y1-y0)/2)
	}
	if fill {
		dc.FillPreserve()
	}
	dc.Stroke()
}

// constrain moves p so that the shape from p0 to p is a square or circle,
// or, for a line, horizontal, vertical or at 45 degrees.
func constrain(kind shapeKind, p0, p image.Point) image.Point {
	dx, dy := p.X-p0.X, p.Y-p0.Y
	if kind == shapeLine {
		switch {
		case abs(dx) > 2*abs(dy):
			return image.Pt(p.X, p0.Y)
		case abs(dy) > 2*abs(dx):
			return image.Pt(p0.X, p.Y)
		}
	}
	d := max(abs(dx), abs(dy))
	sx, sy := sign(dx), sign(dy)
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	return image.Pt(p0.X+d*sx, p0.Y+d*sy)
}

// floodFill fills the area of similar colors around (x, y) with col: the
// neighbors whose channels differ from the start pixel by at most
// tolerance. It returns the number of pixels filled.
func floodFill(img *image.RGBA, x, y int, col color.RGBA, tolerance uint8) int {
	b := img.Bounds()
	if !image.Pt(x, y).In(b) {
		return 0
	}
	target := img.RGBAAt(x, y)
	if target == col {
		return 0
	}
	similar := func(c color.RGBA) bool {
		return near(c.R, target.R, tolerance) && near(c.G, target.G, tolerance) &&
			near(c.B, target.B, tolerance) && near(c.A, target.A, tolerance)
	}

	visited := make([]bool, b.Dx()*b.Dy())
	stack := []image.Point{{x, y}}
	filled := 0
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		i := (p.Y-b.Min.Y)*b.Dx() + p.X - b.Min.X
		if visited[i] || !similar(img.RGBAAt(p.X, p.Y)) {
			continue
		}
		visited[i] = true
		img.SetRGBA(p.X, p.Y, col)
		filled++
		for _, n := range []image.Point{{p.X - 1, p.Y}, {p.X + 1, p.Y}, {p.X, p.Y - 1}, {p.X, p.Y + 1}} {
			if n.In(b) && !visited[(n.Y-b.Min.Y)*b.Dx()+n.X-b.Min.X] {
				stack = append(stack, n)
			}
		}
	}
	return filled
}

// invertColors inverts the colors of the image, keeping the opacity.
func invertColors(img *image.RGBA) {
	for i := 0; i < len(img.Pix); i += 4 {
		a := img.Pix[i+3]
		// The pixels are premultiplied: inverted, a channel is a-c
		img.Pix[i] = a - img.Pix[i]
		img.Pix[i+1] = a - img.Pix[i+1]
		img.Pix[i+2] = a - img.Pix[i+2]
	}
}

// flipHorizontal mirrors the image left to right.
func flipHorizontal(img *image.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for l, r := b.Min.X, b.Max.X-1; l < r; l, r = l+1, r-1 {
			cl, cr := img.RGBAAt(l, y), img.RGBAAt(r, y)
			img.SetRGBA(l, y, cr)
			img.SetRGBA(r, y, cl)
		}
	}
}

func near(a, b, tolerance uint8) bool {
	if a > b {
		return a-b <= tolerance
	}
	return b-a <= tolerance
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}
