package ui

import (
	"image"
	"image/color"
	"slices"
)

type DrawScriptHorLine struct {
	X1 int
	X2 int
	Y  int
	C  float64
}

type DrawScriptPoint struct {
	X int
	Y int
	C float64
}

// drawScriptOp is a plot or an append of a point, kept in the order made:
// the points are merged only when needed, which is much cheaper than a map
// updated on every pixel
type drawScriptOp struct {
	code int64
	c    float64
	// An appended point is added as is; a plotted one only while the
	// pixel is not full yet
	add bool
}

type DrawScript struct {
	horLines []DrawScriptHorLine
	ops      []drawScriptOp
	// ops are merged: one per pixel, sorted by code
	merged bool
	// The pixels, for hasPixel; nil - not built yet
	pixels map[int64]struct{}
	Bounds image.Rectangle
}

func NewDrawScript() *DrawScript {
	return &DrawScript{merged: true}
}

func (c *DrawScript) pointByCode(code int64) (int, int) {
	const MaxUint = ^uint32(0)
	//const MinUint = 0
	const MaxInt = int32(MaxUint >> 1)
	//const MinInt = -MaxInt - 1

	x := int((code >> 32) - int64(MaxInt/2))
	y := int((code & 0xFFFFFFFF) - int64(MaxInt/2))
	return y, x
}

func (c *DrawScript) codeByPoint(x int, y int) int64 {
	const MaxUint = ^uint32(0)
	//const MinUint = 0
	const MaxInt = int32(MaxUint >> 1)
	//const MinInt = -MaxInt - 1

	x64 := int64(x + int(MaxInt/2))
	y64 := int64(y + int(MaxInt/2))
	y64 <<= 32
	y64 |= x64 & 0xFFFFFFFF
	return y64
}

func (c *DrawScript) plot(x int, y int, col float64) {
	c.ops = append(c.ops, drawScriptOp{code: c.codeByPoint(x, y), c: col})
	c.merged = false
	c.pixels = nil
}

func (c *DrawScript) append(script *DrawScript) {
	c.horLines = append(c.horLines, script.horLines...)
	script.merge()
	for _, op := range script.ops {
		c.ops = append(c.ops, drawScriptOp{code: op.code, c: op.c, add: true})
	}
	c.merged = false
	c.pixels = nil
}

// merge leaves one op per pixel with its intensity
func (c *DrawScript) merge() {
	if c.merged {
		return
	}
	c.merged = true
	// Stable: the ops of a pixel are merged in the order they were made
	slices.SortStableFunc(c.ops, func(a, b drawScriptOp) int {
		switch {
		case a.code < b.code:
			return -1
		case a.code > b.code:
			return 1
		}
		return 0
	})
	merged := c.ops[:0]
	for _, op := range c.ops {
		last := len(merged) - 1
		if last < 0 || merged[last].code != op.code {
			merged = append(merged, drawScriptOp{code: op.code, c: op.c})
			continue
		}
		if op.add || merged[last].c < 1 {
			merged[last].c += op.c
		}
	}
	c.ops = merged
}

func (c *DrawScript) hasPixel(x int, y int) bool {
	if c.pixels == nil {
		c.pixels = make(map[int64]struct{}, len(c.ops))
		for _, op := range c.ops {
			c.pixels[op.code] = struct{}{}
		}
	}
	_, ok := c.pixels[c.codeByPoint(x, y)]
	return ok
}

func (c *DrawScript) DrawToRGBA(img *image.RGBA, col color.Color) {
	rgb := rgb8(col)

	for _, line := range c.horLines {
		for x := line.X1; x <= line.X2; x++ {
			value := line.C
			if value > 1 {
				value = 1
			}
			c.mixPixel(img, x, line.Y, rgb, uint32(value*255))
		}
	}

	c.merge()
	for _, op := range c.ops {
		x, y := c.pointByCode(op.code)
		value := op.c
		if value > 1 {
			value = 1
		}
		c.mixPixel(img, x, y, rgb, uint32(value*255))
	}
}

func (c *DrawScript) MixPixel(img *image.RGBA, x int, y int, rgba color.Color, intensity uint32) {
	c.mixPixel(img, x, y, rgb8(rgba), intensity)
}

// rgb8 returns the 8-bit components of the color
func rgb8(col color.Color) [3]uint32 {
	r, g, b, _ := col.RGBA()
	return [3]uint32{r >> 8, g >> 8, b >> 8}
}

// mixPixel blends the color over the pixel with the intensity 0..255
func (c *DrawScript) mixPixel(img *image.RGBA, x int, y int, rgb [3]uint32, intensity uint32) {
	if x < c.Bounds.Min.X || x > c.Bounds.Max.X {
		return
	}
	if y < c.Bounds.Min.Y || y > c.Bounds.Max.Y {
		return
	}
	if !(image.Point{X: x, Y: y}).In(img.Rect) {
		return
	}

	if intensity < 1 {
		return
	}

	alpha := intensity
	antialpha := 255 - alpha

	pix := img.Pix[img.PixOffset(x, y):]
	for i := range 3 {
		pix[i] = uint8(((uint32(pix[i]) * antialpha) >> 8) + ((rgb[i] * alpha) >> 8))
	}
	pix[3] = 255
}
