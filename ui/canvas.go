package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

// canvasState keeps the translation and the clip in pixels of the image;
// the coordinates given to the canvas are logical, which are as many pixels
// times the scale (see Canvas.Scale)
type canvasState struct {
	col color.Color

	translateX int
	translateY int

	// The translation in logical coordinates, for TranslatedX/Y
	logicalX int
	logicalY int

	clipX int
	clipY int
	clipW int
	clipH int

	vAlign     VAlign
	hAlign     HAlign
	fontSize   float64
	fontFamily string
	underline  bool

	// The transform of the drawing coordinates (SetTransform); hasXf is
	// false for the identity, which takes the paths of the code untouched
	xf    Matrix
	hasXf bool
}

type Canvas struct {
	rgba  *image.RGBA
	state *canvasState
	// The states saved by Save, as values: Save and Restore, called around
	// every widget, allocate nothing once the stack is deep enough
	stack []canvasState
	// scale is how many pixels of the image a logical pixel takes, e.g. 2
	// on a Retina screen
	scale float64
	// rec records the shapes as SVG too (SetRecorder)
	rec *SVGRecorder
	// The groups of BeginLayer not ended yet
	layers []canvasLayer
}

type VAlign int
type HAlign int

func (c *HAlign) String() string {
	switch *c {
	case HAlignLeft:
		return "left"
	case HAlignCenter:
		return "center"
	case HAlignRight:
		return "right"
	default:
		return "left"
	}
}

func (c *VAlign) String() string {
	switch *c {
	case VAlignTop:
		return "top"
	case VAlignCenter:
		return "center"
	case VAlignBottom:
		return "bottom"
	default:
		return "top"
	}
}

const VAlignTop VAlign = 0
const VAlignCenter VAlign = 1
const VAlignBottom VAlign = 2

const HAlignLeft HAlign = 0
const HAlignCenter HAlign = 1
const HAlignRight HAlign = 2

func NewCanvas(rgba *image.RGBA) *Canvas {
	return NewCanvasScaled(rgba, 1)
}

// NewCanvasScaled makes a canvas that draws on rgba with the scale: the
// coordinates and sizes given to it are logical, and every logical pixel
// takes scale pixels of rgba, so the drawing is sharp on a HiDPI screen
func NewCanvasScaled(rgba *image.RGBA, scale float64) *Canvas {
	var c Canvas
	c.rgba = rgba
	// Black until SetColor: the color is never nil
	c.state = &canvasState{col: color.RGBA{0, 0, 0, 255}}
	c.stack = make([]canvasState, 0, 16)
	c.scale = 1
	if scale > 0 {
		c.scale = scale
	}
	return &c
}

// Scale returns how many pixels of the image a logical pixel takes: 1 on an
// ordinary screen, e.g. 2 on a Retina one
func (c *Canvas) Scale() float64 {
	return c.scale
}

// Width returns the width of the canvas in logical pixels
func (c *Canvas) Width() int {
	return c.unscale(c.rgba.Rect.Max.X)
}

// Height returns the height of the canvas in logical pixels
func (c *Canvas) Height() int {
	return c.unscale(c.rgba.Rect.Max.Y)
}

// scaled converts a logical length (or an offset from the translation) to pixels
func (c *Canvas) scaled(v int) int {
	if c.scale == 1 {
		return v
	}
	return int(math.Round(float64(v) * c.scale))
}

// unscale converts pixels to logical pixels
func (c *Canvas) unscale(v int) int {
	if c.scale == 1 {
		return v
	}
	return int(math.Floor(float64(v) / c.scale))
}

// pixelRect converts a rectangle in the current logical coordinates to
// pixels of the image. Both edges are converted, so rectangles that touch
// still touch after rounding.
func (c *Canvas) pixelRect(x, y, width, height int) (int, int, int, int) {
	x1 := c.state.translateX + c.scaled(x)
	y1 := c.state.translateY + c.scaled(y)
	x2 := c.state.translateX + c.scaled(x+width)
	y2 := c.state.translateY + c.scaled(y+height)
	return x1, y1, x2 - x1, y2 - y1
}

// pixelCenter converts a logical point, a pixel's center, to the center of
// the pixels it takes in the image
func (c *Canvas) pixelCenter(x, y float64) (float64, float64) {
	if c.scale == 1 {
		return float64(c.state.translateX) + x, float64(c.state.translateY) + y
	}
	return float64(c.state.translateX) + (x+0.5)*c.scale - 0.5, float64(c.state.translateY) + (y+0.5)*c.scale - 0.5
}

// lineWidth returns how many pixels a line of the logical width takes
func (c *Canvas) lineWidth(width int) int {
	return max(1, c.scaled(width))
}

func (c *Canvas) RGBA() *image.RGBA {
	return c.rgba
}

func (c *Canvas) Save() {
	c.stack = append(c.stack, *c.state)
}

func (c *Canvas) Restore() {
	if len(c.stack) == 0 {
		return
	}
	*c.state = c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
}

func (c *Canvas) SetColor(col color.Color) {
	c.state.col = col
}

func (c *Canvas) SetFontFamily(fontFamily string) {
	c.state.fontFamily = fontFamily
}

// SetFontBold draws the text in the bold face of the font, or not
func (c *Canvas) SetFontBold(bold bool) {
	c.state.fontFamily = fontWithStyle(c.state.fontFamily, FontBold, bold)
}

// SetFontItalic draws the text in the italic face of the font, or not
func (c *Canvas) SetFontItalic(italic bool) {
	c.state.fontFamily = fontWithStyle(c.state.fontFamily, FontItalic, italic)
}

func (c *Canvas) SetFontSize(fontSize float64) {
	c.state.fontSize = fontSize
}

func (c *Canvas) SetUnderline(underline bool) {
	c.state.underline = underline
}

func (c *Canvas) SetHAlign(hAlign HAlign) {
	c.state.hAlign = hAlign
}

func (c *Canvas) SetVAlign(vAlign VAlign) {
	c.state.vAlign = vAlign
}

func (c *Canvas) SetDirectTranslateAndClip(x, y, w, h int) {
	c.ResetTransform()
	c.state.logicalX = x
	c.state.logicalY = y
	c.state.translateX = c.scaled(x)
	c.state.translateY = c.scaled(y)
	c.state.clipX = c.state.translateX
	c.state.clipY = c.state.translateY
	c.state.clipW = c.scaled(x+w) - c.state.clipX
	c.state.clipH = c.scaled(y+h) - c.state.clipY
}

func (c *Canvas) TranslateAndClip(x, y int, w, h int) {
	clipX, clipY, clipW, clipH := c.pixelRect(x, y, w, h)
	c.state.logicalX += x
	c.state.logicalY += y
	c.state.translateX = clipX
	c.state.translateY = clipY

	if clipX < c.state.clipX {
		clipW -= c.state.clipX - clipX
		clipX = c.state.clipX
	}

	if clipY < c.state.clipY {
		clipH -= c.state.clipY - clipY
		clipY = c.state.clipY
	}

	if clipX+clipW > c.state.clipX+c.state.clipW {
		clipW -= (clipX + clipW) - (c.state.clipX + c.state.clipW)
	}

	if clipY+clipH > c.state.clipY+c.state.clipH {
		clipH -= (clipY + clipH) - (c.state.clipY + c.state.clipH)
	}

	if clipW < 0 {
		clipW = 0
	}
	if clipH < 0 {
		clipH = 0
	}

	c.state.clipX = clipX
	c.state.clipY = clipY
	c.state.clipW = clipW
	c.state.clipH = clipH
}

// translateBy moves the origin by (dx, dy) logical pixels, keeping the clip
func (c *Canvas) translateBy(dx, dy int) {
	c.state.logicalX += dx
	c.state.logicalY += dy
	c.state.translateX += c.scaled(dx)
	c.state.translateY += c.scaled(dy)
}

// SetPixel draws the logical pixel (x, y) in the current color: as many
// pixels of the image as it takes, blended as FillRect blends
func (c *Canvas) SetPixel(x, y int) {
	c.FillRect(x, y, 1, 1, c.state.col)
}

// MixPixel blends the color over the pixel (x, y) of the image: in pixels,
// with no translation, unlike the other methods
func (c *Canvas) MixPixel(x int, y int, rgba color.Color) {

	if x < c.state.clipX || x >= c.state.clipX+c.state.clipW {
		return
	}
	if y < c.state.clipY || y >= c.state.clipY+c.state.clipH {
		return
	}
	if !(image.Point{x, y}).In(c.rgba.Rect) {
		return
	}
	r, g, b, a := straightRGBA8(rgba)
	c.mixPixelAt(c.rgba.PixOffset(x, y), r, g, b, a)
}

// straightRGBA8 returns the 8-bit components of the color with its alpha
// not applied to them. A color.RGBA is taken so, as it is given throughout
// nui (color.RGBA{255, 0, 0, 128} is a half transparent red); the other
// types of colors are converted.
func straightRGBA8(col color.Color) (r, g, b, a uint32) {
	if c, ok := col.(color.RGBA); ok {
		return uint32(c.R), uint32(c.G), uint32(c.B), uint32(c.A)
	}
	c := color.NRGBAModel.Convert(col).(color.NRGBA)
	return uint32(c.R), uint32(c.G), uint32(c.B), uint32(c.A)
}

// blend8 blends the 8-bit component src with the alpha over dst, rounded
// to the nearest: alpha 255 gives src, 0 gives dst
func blend8(dst, src, alpha uint32) uint8 {
	v := src*alpha + dst*(255-alpha) + 128
	return uint8((v + v>>8) >> 8) // v / 255, rounded
}

// mixPixelAt blends the straight 8-bit color (see straightRGBA8) over the
// pixel at the offset i of the image's Pix, which is premultiplied, as
// draw.Over does: an opaque pixel stays opaque, a transparent one (of an
// image drawn off the screen) takes the alpha. Every blending of the canvas
// but of the text and of the images goes here.
func (c *Canvas) mixPixelAt(i int, r, g, b, alpha uint32) {
	if alpha == 0 {
		return
	}
	pix := c.rgba.Pix[i : i+4 : i+4]
	if alpha == 255 {
		pix[0], pix[1], pix[2], pix[3] = uint8(r), uint8(g), uint8(b), 255
		return
	}
	pix[0] = blend8(uint32(pix[0]), r, alpha)
	pix[1] = blend8(uint32(pix[1]), g, alpha)
	pix[2] = blend8(uint32(pix[2]), b, alpha)
	if pix[3] != 255 {
		pix[3] = blend8(uint32(pix[3]), 255, alpha)
	}
}

// DrawLine draws the line of the logical width from (x1, y1) to (x2, y2). A
// horizontal or a vertical line one wide covers the pixels of the FillRect
// of its row or column, the last point left out; any other one is drawn
// between the centers of its end pixels.
func (c *Canvas) DrawLine(x1 int, y1 int, x2 int, y2 int, width int, col color.Color) {
	if c.clipEmpty() {
		return
	}
	if c.state.hasXf {
		c.lineXf(float64(x1), float64(y1), float64(x2), float64(y2), float64(width), col)
		return
	}
	if c.rec != nil {
		p := NewPath()
		p.MoveTo(float64(x1)+0.5, float64(y1)+0.5)
		p.LineTo(float64(x2)+0.5, float64(y2)+0.5)
		c.rec.strokePath(c, p, StrokeStyle{Width: float64(max(width, 1))}, col)
	}
	if (x1 == x2 || y1 == y2) && width == 1 {
		if x1 == x2 {
			c.FillRect(x1, min(y1, y2), 1, max(y1, y2)-min(y1, y2), col)
		} else {
			c.FillRect(min(x1, x2), y1, max(x1, x2)-min(x1, x2), 1, col)
		}
		return
	}
	fx1, fy1 := c.pixelCenter(float64(x1), float64(y1))
	fx2, fy2 := c.pixelCenter(float64(x2), float64(y2))
	if w := c.lineWidth(width); w > 1 {
		c.drawWideLine(math.Round(fx1), math.Round(fy1), math.Round(fx2), math.Round(fy2), float64(w), col)
		return
	}
	// Clipped to a pixel past the clip: what is past it is left out by the
	// script's bounds
	px1, py1, px2, py2, visible := CohenSutherland(int(math.Round(fx1)), int(math.Round(fy1)), int(math.Round(fx2)), int(math.Round(fy2)), c.state.clipX, c.state.clipY, c.state.clipX+c.state.clipW, c.state.clipY+c.state.clipH)
	if !visible {
		return
	}
	script := c.MakeScriptLine(float64(px1), float64(py1), float64(px2), float64(py2), float64(c.lineWidth(width)))
	script.Bounds = c.scriptBounds()
	script.DrawToRGBA(c.rgba, col)
}

// clipEmpty reports whether the clip has no pixels: nothing is drawn
func (c *Canvas) clipEmpty() bool {
	return c.state.clipW <= 0 || c.state.clipH <= 0
}

// clipLast returns the last column and row of the clip, in pixels
func (c *Canvas) clipLast() image.Point {
	return image.Point{X: c.state.clipX + c.state.clipW - 1, Y: c.state.clipY + c.state.clipH - 1}
}

// scriptBounds returns the clip as DrawScript.Bounds take it, the last
// column and row included. Not made with image.Rect, which would swap the
// edges of an empty clip into a pixel wide one.
func (c *Canvas) scriptBounds() image.Rectangle {
	return image.Rectangle{Min: image.Point{X: c.state.clipX, Y: c.state.clipY}, Max: c.clipLast()}
}

// DrawLineF draws an antialiased line one pixel wide between points with
// fractional coordinates, both ends included. Integer coordinates are the
// centers of the pixels, so a line between them is as sharp as DrawLine's;
// a line between fractional ones is spread over the neighbor pixels, e.g.
// for a chart that scrolls smoothly.
func (c *Canvas) DrawLineF(x1, y1, x2, y2 float64, col color.Color) {
	if c.clipEmpty() {
		return
	}
	if c.state.hasXf {
		c.lineXf(x1, y1, x2, y2, 1, col)
		return
	}
	x1, y1 = c.pixelCenter(x1, y1)
	x2, y2 = c.pixelCenter(x2, y2)

	// Clipped with a margin of a pixel: the pixels at the edge get their share
	left, top := float64(c.state.clipX-1), float64(c.state.clipY-1)
	right, bottom := float64(c.state.clipX+c.state.clipW), float64(c.state.clipY+c.state.clipH)
	x1, y1, x2, y2, visible := clipLineF(x1, y1, x2, y2, left, top, right, bottom)
	if !visible {
		return
	}

	// A line as wide as a logical pixel: as many lines a pixel apart
	script := NewDrawScript()
	n := c.lineWidth(1)
	for i := 0; i < n; i++ {
		plotLineWuF(script, x1, y1, x2, y2, float64(i)-float64(n-1)/2)
	}
	script.Bounds = c.scriptBounds()
	script.DrawToRGBA(c.rgba, col)
}

// clipLineF clips the line to the rectangle (Liang-Barsky)
func clipLineF(x1, y1, x2, y2, left, top, right, bottom float64) (float64, float64, float64, float64, bool) {
	dx, dy := x2-x1, y2-y1
	t0, t1 := 0.0, 1.0
	for _, edge := range [4][2]float64{
		{-dx, x1 - left},
		{dx, right - x1},
		{-dy, y1 - top},
		{dy, bottom - y1},
	} {
		p, q := edge[0], edge[1]
		if p == 0 {
			if q < 0 {
				return 0, 0, 0, 0, false // parallel to the edge and outside
			}
			continue
		}
		t := q / p
		if p < 0 {
			t0 = math.Max(t0, t)
		} else {
			t1 = math.Min(t1, t)
		}
		if t0 > t1 {
			return 0, 0, 0, 0, false
		}
	}
	return x1 + t0*dx, y1 + t0*dy, x1 + t1*dx, y1 + t1*dy, true
}

// plotLineWuF plots the line by Xiaolin Wu's algorithm with fractional ends:
// along the major axis a pixel per step, split between the two pixels
// across it by the distance to their centers. Unlike the classic algorithm
// the end pixels are plotted in full, so the joints of a polyline drawn
// segment by segment are not dimmer than the rest of it. The line is moved
// by shift pixels across the major axis.
func plotLineWuF(script *DrawScript, x1, y1, x2, y2, shift float64) {
	steep := math.Abs(y2-y1) > math.Abs(x2-x1)
	if steep {
		x1, y1 = y1, x1
		x2, y2 = y2, x2
	}
	if x1 > x2 {
		x1, x2 = x2, x1
		y1, y2 = y2, y1
	}
	plot := func(x, y int, c float64) {
		if steep {
			x, y = y, x
		}
		script.plot(x, y, c)
	}

	gradient := 0.0 // a point
	if dx := x2 - x1; dx > 0 {
		gradient = (y2 - y1) / dx
	}

	// The line at the center of every pixel column from the first to the last
	xStart, xEnd := int(math.Round(x1)), int(math.Round(x2))
	for x := xStart; x <= xEnd; x++ {
		y := y1 + gradient*(float64(x)-x1) + shift
		plot(x, ipart(y), 1-fpart(y))
		plot(x, ipart(y)+1, fpart(y))
	}
}

// MakeScriptLine makes the pixels of a line of the width between the pixels
// (x1, y1) and (x2, y2) of the image.
//
// Deprecated: the former way DrawLine drew wide lines, exported by
// accident; it will be removed. Draw lines with DrawLine and DrawLineF.
func (c *Canvas) MakeScriptLine(x1, y1, x2, y2, width float64) *DrawScript {
	x1 = math.Round(x1)
	y1 = math.Round(y1)
	x2 = math.Round(x2)
	y2 = math.Round(y2)
	width = math.Round(width)

	result := NewDrawScript()

	// A line of no length has no direction: nothing is drawn, as by
	// MakeScriptLineWu (a wide one would go on with NaN corners)
	if x1 == x2 && y1 == y2 {
		return result
	}

	{
		minX := math.Min(x1, x2)
		maxX := math.Max(x1, x2)
		minY := math.Min(y1, y2)
		maxY := math.Max(y1, y2)

		if int(minX) > c.state.clipX+c.state.clipW {
			return result
		}
		if int(maxX) < c.state.clipX {
			return result
		}
		if int(minY) > c.state.clipY+c.state.clipH {
			return result
		}
		if int(maxY) < c.state.clipY {
			return result
		}
	}

	if width > 1.1 {

		width -= 1
		width /= 2

		deltaX := x2 - x1
		deltaY := y2 - y1

		tnA := deltaY / deltaX

		katY := width * math.Cos(math.Atan(tnA))
		katX := math.Sqrt(math.Abs(width*width - katY*katY))

		if deltaX <= 0 && deltaY <= 0 {
			katY = -katY
		}

		if deltaX <= 0 && deltaY > 0 {
			katX = -katX
			katY = -katY
		}

		if deltaX > 0 && deltaY > 0 {
			katX = -katX
		}

		a1_x := x1 - katX
		a1_y := y1 - katY
		a2_x := x1 + katX
		a2_y := y1 + katY

		b1_x := x2 - katX
		b1_y := y2 - katY
		b2_x := x2 + katX
		b2_y := y2 + katY

		var ps []image.Point
		ps = make([]image.Point, 0)
		ps = append(ps, image.Pt(int(a1_x), int(a1_y)))
		ps = append(ps, image.Pt(int(a2_x), int(a2_y)))
		ps = append(ps, image.Pt(int(b2_x), int(b2_y)))
		ps = append(ps, image.Pt(int(b1_x), int(b1_y)))
		result.append(c.fillEx(ps))

		result.append(c.MakeScriptLineWu(a1_x, a1_y, a2_x, a2_y))
		result.append(c.MakeScriptLineWu(b1_x, b1_y, b2_x, b2_y))
		result.append(c.MakeScriptLineWu(a1_x, a1_y, b1_x, b1_y))
		result.append(c.MakeScriptLineWu(a2_x, a2_y, b2_x, b2_y))
	} else {
		result.append(c.MakeScriptLineWu(x1, y1, x2, y2))
	}

	return result
}

// MakeScriptLineBresenham makes the pixels of a line without antialiasing.
//
// Deprecated: a part of MakeScriptLine, exported by accident; it will be
// removed. Draw lines with DrawLine and DrawLineF.
func (c *Canvas) MakeScriptLineBresenham(x1 float64, y1 float64, x2 float64, y2 float64) *DrawScript {
	x1 = math.Round(x1)
	y1 = math.Round(y1)
	x2 = math.Round(x2)
	y2 = math.Round(y2)

	result := NewDrawScript()

	stepByX := true
	if math.Abs(x2-x1) < math.Abs(y2-y1) {
		stepByX = false
	}

	if stepByX {
		if x2 < x1 { // Swap
			xo := x1
			x1 = x2
			x2 = xo
			yo := y1
			y1 = y2
			y2 = yo
		}

		beginX := int(x1)
		endX := int(x2)

		deltaX := math.Abs(x2 - x1)
		deltaY := math.Abs(y2 - y1)

		y := int(y1)
		var err float64 = 0
		errDelta := deltaY / deltaX
		var yDir = 1
		if (y2 - y1) < 0 {
			yDir = -1
		}

		for x := beginX; x <= endX; x++ {
			result.plot(x, y, 1)
			err += errDelta
			if err > 0.5 {
				y += yDir
				err -= 1.0
			}
		}
	} else {
		if y2 < y1 {
			xo := x1
			x1 = x2
			x2 = xo
			yo := y1
			y1 = y2
			y2 = yo
		}

		beginY := int(y1)
		endY := int(y2)

		deltaX := math.Abs(x2 - x1)
		deltaY := math.Abs(y2 - y1)

		x := int(x1)
		err := 0.0
		errDelta := deltaX / deltaY
		xDir := 1
		if (x2 - x1) < 0 {
			xDir = -1
		}

		for y := beginY; y <= endY; y++ {
			result.plot(x, y, 1)
			err += errDelta
			if err > 0.5 {
				x += xDir
				err -= 1.0
			}
		}
	}

	return result
}

func (c *Canvas) fillEx(points []image.Point) *DrawScript {
	result := NewDrawScript()
	if len(points) < 3 {
		return result
	}

	// Borders
	lastPoint := points[0]
	for i := 1; i < len(points); i++ {
		result.append(c.MakeScriptLineBresenham(float64(lastPoint.X), float64(lastPoint.Y), float64(points[i].X), float64(points[i].Y)))
		lastPoint = points[i]
	}
	result.append(c.MakeScriptLineBresenham(float64(lastPoint.X), float64(lastPoint.Y), float64(points[0].X), float64(points[0].Y)))

	minX := int(math.MaxInt32)
	maxX := int(math.MinInt32)
	minY := int(math.MaxInt32)
	maxY := int(math.MinInt32)

	for _, pp := range points {
		if pp.X > maxX {
			maxX = pp.X
		}
		if pp.X < minX {
			minX = pp.X
		}
		if pp.Y > maxY {
			maxY = pp.Y
		}
		if pp.Y < minY {
			minY = pp.Y
		}
	}

	beginX := 0

	for y := minY + 1; y <= maxY-1; y++ {
		countOfTriggers := 0
		flag := false

		for x := minX; x <= maxX+1; x++ {
			if result.hasPixel(x, y) && !result.hasPixel(x+1, y) {
				countOfTriggers++
			}
		}

		if countOfTriggers > 1 {
			for x := minX; x <= maxX; x++ {
				if result.hasPixel(x, y) && !result.hasPixel(x+1, y) {
					if flag {
						var line DrawScriptHorLine
						line.X1 = beginX
						line.X2 = x
						line.Y = y
						line.C = 1
						result.horLines = append(result.horLines, line)
					} else {
						beginX = x
					}

					flag = !flag
				}
			}
		}
	}

	return result
}

func ipart(x float64) int {
	rn := math.Floor(x)
	return int(rn)
}

func round(x float64) int {
	iprt := ipart(x + 0.5)
	return iprt
}

func fpart(x float64) float64 {
	return x - float64(ipart(x))
}

// MakeScriptLineWu makes the pixels of an antialiased line one pixel wide.
//
// Deprecated: an internal of DrawLine, exported by accident; it will be
// unexported. Draw lines with DrawLine and DrawLineF.
func (c *Canvas) MakeScriptLineWu(x1, y1, x2, y2 float64) *DrawScript {

	result := NewDrawScript()

	x1 = math.Round(x1)
	y1 = math.Round(y1)
	x2 = math.Round(x2)
	y2 = math.Round(y2)

	if math.Abs(x1-x2) < 0.1 && math.Abs(y1-y2) < 0.1 {
		return result
	}

	stepByX := true
	if math.Abs(x2-x1) < math.Abs(y2-y1) {
		stepByX = false
	}

	if stepByX {
		if x2 < x1 {
			xo := x1
			x1 = x2
			x2 = xo
			yo := y1
			y1 = y2
			y2 = yo
		}

		dx := x2 - x1
		dy := y2 - y1
		gradient := dy / dx
		offset := y1 + gradient*(float64(round(x1))-x1)

		workFrom := int(x1)
		workTo := int(x2)

		xgap := 1 - fpart(x1+0.5)
		result.plot(workFrom, int(math.Floor(y1)), (1-fpart(offset))*xgap)
		result.plot(workFrom, int(math.Floor(y1))+1, fpart(offset)*xgap)

		xgap = fpart(x1 + 0.5)
		result.plot(workTo, int(math.Floor(y2)), (1-fpart(offset))*xgap)
		result.plot(workTo, int(math.Floor(y2))+1, fpart(offset)*xgap)

		offset += gradient

		for workIndex := workFrom + 1; workIndex <= workTo-1; workIndex++ {
			err := fpart(offset)
			if err > 0 {
				c1 := 1 - err
				c2 := err
				result.plot(workIndex, ipart(offset), c1)
				result.plot(workIndex, ipart(offset)+1, c2)
			} else {
				c1 := 1 - math.Abs(err)
				c2 := math.Abs(err)
				result.plot(workIndex, ipart(offset), c1)
				result.plot(workIndex, ipart(offset)-1, c2)
			}
			offset = offset + gradient
		}
	} else {
		if y2 < y1 {
			xo := x1
			x1 = x2
			x2 = xo
			yo := y1
			y1 = y2
			y2 = yo
		}

		dx := x2 - x1
		dy := y2 - y1

		gradient := dx / dy

		offset := x1 + gradient*(float64(round(y1))-y1)

		workFrom := int(y1)
		workTo := int(y2)

		xgap := 1 - fpart(y1+0.5)
		result.plot(int(math.Floor(x1)), workFrom, (1-fpart(offset))*xgap)
		result.plot(int(math.Floor(x1))+1, workFrom, fpart(offset)*xgap)

		xgap = fpart(y1 + 0.5)
		result.plot(int(math.Floor(x2)), workTo, (1-fpart(offset))*xgap)
		result.plot(int(math.Floor(x2))+1, workTo, fpart(offset)*xgap)

		offset += gradient

		for workIndex := workFrom + 1; workIndex <= workTo-1; workIndex++ {
			err := fpart(offset)
			if err > 0 {
				c1 := 1 - err
				c2 := err
				result.plot(ipart(offset), workIndex, c1)
				result.plot(ipart(offset)+1, workIndex, c2)
			} else {
				c1 := 1 - math.Abs(err)
				c2 := math.Abs(err)
				result.plot(ipart(offset), workIndex, c1)
				result.plot(ipart(offset)-1, workIndex, c2)
			}
			offset = offset + gradient
		}
	}

	return result
}

// CohenSutherland clips the line to the rectangle, its edges included, and
// reports whether any of it is left.
//
// Deprecated: an internal of DrawLine, exported by accident; it will be
// unexported. Draw lines with DrawLine and DrawLineF.
func CohenSutherland(x1, y1, x2, y2, left, top, right, bottom int) (int, int, int, int, bool) {

	invalid := false

	for {
		// Left-Right-Top-Bottom
		code1 := 0
		if x1 < left {
			code1 |= 8
		}
		if x1 > right {
			code1 |= 4
		}
		if y1 < top {
			code1 |= 2
		}
		if y1 > bottom {
			code1 |= 1
		}

		code2 := 0
		if x2 < left {
			code2 |= 8
		}
		if x2 > right {
			code2 |= 4
		}
		if y2 < top {
			code2 |= 2
		}
		if y2 > bottom {
			code2 |= 1
		}

		if code1 == 0 && code2 == 0 {
			break
		}

		if (code1 & code2) != 0 {
			invalid = true
			break // Outside
		}

		code := code1
		if code == 0 {
			code = code2
		}

		x := 0
		y := 0

		if (code & 2) != 0 {
			x = x1 + (x2-x1)*(top-y1)/(y2-y1)
			y = top
		} else {
			if (code & 1) != 0 {
				x = x1 + (x2-x1)*(bottom-y1)/(y2-y1)
				y = bottom
			} else {
				if (code & 4) != 0 {
					y = y1 + (y2-y1)*(right-x1)/(x2-x1)
					x = right
				} else {
					if (code & 8) != 0 {
						y = y1 + (y2-y1)*(left-x1)/(x2-x1)
						x = left
					}
				}
			}
		}

		if code1 != 0 {
			x1 = x
			y1 = y
		} else {
			x2 = x
			y2 = y
		}
	}

	return x1, y1, x2, y2, !invalid
}

func (c *Canvas) FillRect(x int, y int, width int, height int, colr color.Color) {
	if c.state.hasXf {
		c.fillRectXf(float64(x), float64(y), float64(width), float64(height), colr)
		return
	}
	if c.rec != nil {
		c.rec.rect(c, float64(x), float64(y), float64(width), float64(height), colr, false)
	}
	x, y, width, height = c.pixelRect(x, y, width, height)
	c.fillRectPixels(x, y, width, height, colr)
}

// fillRectPixels fills the rectangle in pixels of the image, within the clip
func (c *Canvas) fillRectPixels(x int, y int, width int, height int, colr color.Color) {

	if x < 0 {
		width += x
		x = 0
	}

	if y < 0 {
		height += y
		y = 0
	}

	if x+width > c.rgba.Rect.Max.X {
		width = c.rgba.Rect.Max.X - x
	}

	if y+height > c.rgba.Rect.Max.Y {
		height = c.rgba.Rect.Max.Y - y
	}

	if width < 0 {
		return
	}

	if height < 0 {
		return
	}

	if x < c.state.clipX {
		width -= c.state.clipX - x
		x = c.state.clipX
	}

	if y < c.state.clipY {
		height -= c.state.clipY - y
		y = c.state.clipY
	}

	if x+width > c.state.clipX+c.state.clipW {
		width = c.state.clipX + c.state.clipW - x
	}

	if y+height > c.state.clipY+c.state.clipH {
		height = c.state.clipY + c.state.clipH - y
	}

	if width < 1 {
		return
	}

	if height < 1 {
		return
	}

	if _, _, _, a := colr.RGBA(); a == 65535 {
		cc := color.RGBAModel.Convert(colr).(color.RGBA)
		// The first row filled by doubling its filled part, the others
		// copied from it: no pixel loop, nothing allocated
		first := y*c.rgba.Stride + x*4
		line := c.rgba.Pix[first : first+width*4]
		line[0], line[1], line[2], line[3] = cc.R, cc.G, cc.B, cc.A
		for filled := 4; filled < len(line); filled *= 2 {
			copy(line[filled:], line[:filled])
		}
		for yy := y + 1; yy < y+height; yy++ {
			offset := yy*c.rgba.Stride + x*4
			copy(c.rgba.Pix[offset:offset+width*4], line)
		}
	} else {
		// Blended as by MixPixel, pixel by pixel; the rectangle is within
		// the clip and the image already
		r, g, b, alpha := straightRGBA8(colr)
		if alpha == 0 {
			return
		}
		// blend8 with the source's part computed once
		sr, sg, sb, ia := r*alpha+128, g*alpha+128, b*alpha+128, 255-alpha
		for yy := y; yy < y+height; yy++ {
			offset := c.rgba.PixOffset(x, yy)
			row := c.rgba.Pix[offset : offset+width*4 : offset+width*4]
			for i := 0; i < len(row); i += 4 {
				px := row[i : i+4 : i+4]
				v := sr + uint32(px[0])*ia
				px[0] = uint8((v + v>>8) >> 8)
				v = sg + uint32(px[1])*ia
				px[1] = uint8((v + v>>8) >> 8)
				v = sb + uint32(px[2])*ia
				px[2] = uint8((v + v>>8) >> 8)
				if px[3] != 255 {
					px[3] = blend8(uint32(px[3]), 255, alpha)
				}
			}
		}
	}
}

func (c *Canvas) DrawRect(x int, y int, width int, height int) {
	if c.state.hasXf {
		c.strokeRectXf(float64(x), float64(y), float64(width), float64(height), 0, c.state.col)
		return
	}
	if c.rec != nil {
		c.rec.rect(c, float64(x), float64(y), float64(width), float64(height), c.state.col, true)
	}
	// The sides, a logical pixel wide, without overlapping in the corners: a
	// translucent color is blended once on each pixel
	px, py, pw, ph := c.pixelRect(x, y, width, height)
	lw := min(c.lineWidth(1), pw)
	lh := min(c.lineWidth(1), ph)
	c.fillRectPixels(px, py, pw, lh, c.state.col)
	if ph > lh {
		c.fillRectPixels(px, py+ph-lh, pw, lh, c.state.col)
	}
	c.fillRectPixels(px, py+lh, lw, ph-lh*2, c.state.col)
	if pw > lw {
		c.fillRectPixels(px+pw-lw, py+lh, lw, ph-lh*2, c.state.col)
	}
}

// DrawRoundedRect draws the outline of a rounded rectangle in the current
// color, a logical pixel wide, anti-aliased, within the clip
func (c *Canvas) DrawRoundedRect(x int, y int, width int, height int, radius int) {
	if c.state.hasXf {
		c.strokeRectXf(float64(x), float64(y), float64(width), float64(height), float64(radius)-0.5, c.state.col)
		return
	}
	px, py, pw, ph := c.pixelRect(x, y, width, height)
	c.fillRoundedPixels(px, py, pw, ph, float64(radius)*c.scale, c.lineWidth(1), c.state.col, c.clipRect())
}

// FillRoundedRect fills a rounded rectangle in the current color,
// anti-aliased, within the clip
func (c *Canvas) FillRoundedRect(x int, y int, width int, height int, radius int) {
	if c.state.hasXf {
		c.fillRoundedXf(float64(x), float64(y), float64(width), float64(height), float64(radius), c.state.col)
		return
	}
	px, py, pw, ph := c.pixelRect(x, y, width, height)
	c.fillRoundedPixels(px, py, pw, ph, float64(radius)*c.scale, 0, c.state.col, c.clipRect())
}

// FillTriangle fills the triangle with corners (x1,y1), (x2,y2), (x3,y3).
func (c *Canvas) FillTriangle(x1, y1, x2, y2, x3, y3 int, colr color.Color) {
	c.fillPolygon([][2]float64{{float64(x1), float64(y1)}, {float64(x2), float64(y2)}, {float64(x3), float64(y3)}}, colr)
}

// fillPolygon fills the polygon of logical points with antialiased edges.
// The coverage is computed in a mask of the polygon's size within the clip,
// not of the whole image: the arrows of the combo boxes and the num boxes
// are drawn on every frame.
func (c *Canvas) fillPolygon(pts [][2]float64, colr color.Color) {
	if c.state.hasXf {
		poly := make([]PointF, len(pts))
		for i, p := range pts {
			poly[i] = PointF{p[0], p[1]}
		}
		c.FillPolygon(poly, colr)
		return
	}
	px := make([][2]float64, len(pts))
	for i, p := range pts {
		px[i] = [2]float64{float64(c.state.translateX) + p[0]*c.scale, float64(c.state.translateY) + p[1]*c.scale}
	}
	c.fillPolygonPixels(px, colr)
}

func (c *Canvas) DrawImage(x int, y int, img image.Image) {
	if c.state.hasXf {
		b := img.Bounds()
		c.drawImageXf(float64(x), float64(y), float64(b.Dx()), float64(b.Dy()), img, b, true)
		return
	}
	if c.scale != 1 {
		c.drawImageScaled(x, y, img)
		return
	}
	bInner := image.Rectangle{}
	bInner.Min.X = x + c.state.translateX
	bInner.Min.Y = y + c.state.translateY
	// The image's own bounds may not start at 0, 0 (a SubImage)
	bInner.Max.X = bInner.Min.X + img.Bounds().Dx()
	bInner.Max.Y = bInner.Min.Y + img.Bounds().Dy()

	xOffset := 0
	yOffset := 0

	if bInner.Min.X < c.state.clipX {
		xOffset = c.state.clipX - bInner.Min.X
		bInner.Min.X = c.state.clipX
	}

	if bInner.Min.Y < c.state.clipY {
		yOffset = c.state.clipY - bInner.Min.Y
		bInner.Min.Y = c.state.clipY
	}

	if bInner.Max.X > c.state.clipX+c.state.clipW {
		bInner.Max.X = c.state.clipX + c.state.clipW
	}

	if bInner.Max.Y > c.state.clipY+c.state.clipH {
		bInner.Max.Y = c.state.clipY + c.state.clipH
	}

	if bInner.Min.X >= bInner.Max.X {
		return
	}

	if bInner.Min.Y >= bInner.Max.Y {
		return
	}

	draw.Draw(c.rgba, bInner, img, img.Bounds().Min.Add(image.Point{xOffset, yOffset}), draw.Over)
}

// drawImageScaled draws the image, its pixels taken as logical ones,
// stretched to the pixels they take
func (c *Canvas) drawImageScaled(x int, y int, img image.Image) {
	b := img.Bounds()
	px, py, pw, ph := c.pixelRect(x, y, b.Dx(), b.Dy())
	dst := image.Rect(px, py, px+pw, py+ph)
	clip := image.Rect(c.state.clipX, c.state.clipY, c.state.clipX+c.state.clipW, c.state.clipY+c.state.clipH).Intersect(c.rgba.Rect)
	if dst.Intersect(clip).Empty() {
		return
	}
	target := c.rgba.SubImage(clip).(*image.RGBA)
	// Catmull-Rom keeps small images such as icons sharp; it costs too much
	// for big ones
	var scaler xdraw.Scaler = xdraw.CatmullRom
	if b.Dx()*b.Dy() > 256*256 {
		scaler = xdraw.ApproxBiLinear
	}
	scaler.Scale(target, dst, img, b, xdraw.Over, nil)
}

// TranslatedX returns the logical x of the current origin
func (c *Canvas) TranslatedX() int {
	return c.state.logicalX
}

// TranslatedY returns the logical y of the current origin
func (c *Canvas) TranslatedY() int {
	return c.state.logicalY
}

// ClipX, ClipY, ClipW and ClipH return the clip rectangle in pixels of the image
func (c *Canvas) ClipX() int {
	return c.state.clipX
}

func (c *Canvas) ClipY() int {
	return c.state.clipY
}

func (c *Canvas) ClipW() int {
	return c.state.clipW
}

func (c *Canvas) ClipH() int {
	return c.state.clipH
}

func (c *Canvas) DrawText(x int, y int, width int, height int, text string) {
	if c.state.hasXf {
		c.drawTextXf(x, y, width, height, text)
		return
	}
	c.drawTextLines(x, y, width, height, text, true)
}

// drawTextLines is DrawText without a transform; record writes the text to
// the SVG recorder, if there is one
func (c *Canvas) drawTextLines(x int, y int, width int, height int, text string, record bool) {
	lines := textLines(text)

	fontFamily := c.state.fontFamily
	fontSize := c.state.fontSize
	colr := c.state.col
	underline := c.state.underline
	vAlign := c.state.vAlign
	hAlign := c.state.hAlign

	yOffset := 0

	_, textHeight, err := MeasureText(fontFamily, fontSize, "Йg")
	if err != nil {
		return
	}

	//textHeight := 20

	fulltextHeight := textHeight * len(lines)

	switch vAlign {
	case VAlignTop:
		yOffset = 0
	case VAlignCenter:
		yOffset = height/2 - fulltextHeight/2
	case VAlignBottom:
		yOffset = height - fulltextHeight
	}

	c.Save()
	c.TranslateAndClip(x, y, width, height)
	//x = 0
	//y = 0

	for _, str := range lines {
		xx := 0
		// The width only aligns and underlines the line
		textWidth := 0
		if hAlign != HAlignLeft || underline {
			w, _, err := MeasureText(fontFamily, fontSize, str)
			if err != nil {
				break // restored below
			}
			textWidth = w
		}

		if hAlign != HAlignLeft {

			switch hAlign {
			case HAlignLeft:
				xx = 0
			case HAlignCenter:
				xx = (width / 2) - (textWidth / 2)
			case HAlignRight:
				xx = width - textWidth
			}
		}

		//c.DrawText(xx, yOffset+y, str, fontFamily, fontSize, colr, underline)
		//c.DrawText(xx, yOffset+y, str, fontFamily, fontSize, colr, underline)

		if c.rec != nil && record {
			tx := xx
			if hAlign != HAlignLeft && textWidth == 0 {
				tx = 0
			}
			c.rec.text(c, str, float64(tx), float64(yOffset), float64(textHeight), 0, 0, 0)
		}

		// The text is drawn at the size it takes in pixels, not stretched
		textX := c.state.translateX + c.scaled(xx)
		textY := c.state.translateY + c.scaled(yOffset)

		DrawText(c.rgba, str, colr, fontFamily, fontSize*c.scale, textX, textY, c.state.clipX, c.state.clipY, c.state.clipW, c.state.clipH)

		// Where and as thick as drawMnemonicUnderline underlines a letter
		if underline && textWidth > 0 {
			lineY := yOffset + textHeight - max(textHeight/6, 2)
			c.FillRect(xx, lineY, textWidth, max(int(fontSize/14), 1), colr)
		}

		yOffset += textHeight
	}

	c.Restore()
}

// DrawImageRect draws the part src of the image (in its own coordinates)
// stretched to the rectangle of x, y, width, height: zoomed in or out.
// smooth interpolates the pixels (a photo); without it each image pixel is
// a sharp square (pixel art, a zoomed-in bitmap being edited). Only the part
// within the clip is computed.
func (c *Canvas) DrawImageRect(x, y, width, height int, img image.Image, src image.Rectangle, smooth bool) {
	src = src.Intersect(img.Bounds())
	if width <= 0 || height <= 0 || src.Empty() {
		return
	}
	if c.state.hasXf {
		c.drawImageXf(float64(x), float64(y), float64(width), float64(height), img, src, smooth)
		return
	}
	px, py, pw, ph := c.pixelRect(x, y, width, height)
	dst := image.Rect(px, py, px+pw, py+ph)
	clip := image.Rect(c.state.clipX, c.state.clipY, c.state.clipX+c.state.clipW, c.state.clipY+c.state.clipH).Intersect(c.rgba.Rect)
	visible := dst.Intersect(clip)
	if visible.Empty() {
		return
	}
	// The part of src that lands in the visible part of dst
	fx := float64(src.Dx()) / float64(dst.Dx())
	fy := float64(src.Dy()) / float64(dst.Dy())
	srcVisible := image.Rect(
		src.Min.X+int(math.Floor(float64(visible.Min.X-dst.Min.X)*fx)),
		src.Min.Y+int(math.Floor(float64(visible.Min.Y-dst.Min.Y)*fy)),
		src.Min.X+int(math.Ceil(float64(visible.Max.X-dst.Min.X)*fx)),
		src.Min.Y+int(math.Ceil(float64(visible.Max.Y-dst.Min.Y)*fy)),
	).Intersect(src)
	// The destination of srcVisible, aligned to its whole source pixels
	dstVisible := image.Rect(
		dst.Min.X+int(math.Round(float64(srcVisible.Min.X-src.Min.X)/fx)),
		dst.Min.Y+int(math.Round(float64(srcVisible.Min.Y-src.Min.Y)/fy)),
		dst.Min.X+int(math.Round(float64(srcVisible.Max.X-src.Min.X)/fx)),
		dst.Min.Y+int(math.Round(float64(srcVisible.Max.Y-src.Min.Y)/fy)),
	)
	target := c.rgba.SubImage(clip).(*image.RGBA)
	var scaler xdraw.Scaler = xdraw.NearestNeighbor
	if smooth {
		scaler = xdraw.ApproxBiLinear
	}
	scaler.Scale(target, dstVisible, img, srcVisible, xdraw.Over, nil)
}
