package ui

import (
	"image"
	"image/color"
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

type DrawScript struct {
	horLines []DrawScriptHorLine
	points   map[int64]float64
	Bounds   image.Rectangle
}

func NewDrawScript() *DrawScript {
	var c DrawScript
	c.horLines = make([]DrawScriptHorLine, 0)
	c.points = make(map[int64]float64)
	return &c
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
	code := c.codeByPoint(x, y)
	if _, ok := c.points[code]; ok {
		if c.points[code] < 1 {
			c.points[code] += col
		}
	} else {
		c.points[code] = col
	}
}

func (c *DrawScript) append(script *DrawScript) {
	c.horLines = append(c.horLines, script.horLines...)
	for k, v := range script.points {
		c.points[k] += v
	}
}

func (c *DrawScript) hasPixel(x int, y int) bool {
	if _, ok := c.points[c.codeByPoint(x, y)]; ok {
		return true
	} else {
		return false
	}
}

func (c *DrawScript) DrawToRGBA(img *image.RGBA, col color.Color) {
	for _, line := range c.horLines {
		for x := line.X1; x <= line.X2; x++ {
			value := line.C
			if value > 1 {
				value = 1
			}
			c.MixPixel(img, x, line.Y, col, uint32(value*255))
		}
	}

	for key, value := range c.points {
		x, y := c.pointByCode(key)
		if value > 1 {
			value = 1
		}
		c.MixPixel(img, x, y, col, uint32(value*255))
	}
}

func (c *DrawScript) MixPixel(img *image.RGBA, x int, y int, rgba color.Color, intensity uint32) {

	if x < c.Bounds.Min.X || x > c.Bounds.Max.X {
		return
	}
	if y < c.Bounds.Min.Y || y > c.Bounds.Max.Y {
		return
	}

	if intensity < 1 {
		return
	}

	cOld := img.At(x, y)
	oR, oG, oB, _ := cOld.RGBA()
	cR, cG, cB, _ := rgba.RGBA()
	cR = cR >> 8
	cG = cG >> 8
	cB = cB >> 8

	oR = oR >> 8
	oG = oG >> 8
	oB = oB >> 8

	alpha := uint32(intensity)
	antialpha := 255 - alpha

	if intensity > 0 && intensity < 255 {
		intensity += 1
	}

	//alpha, antialpha = antialpha, alpha
	nR := uint8(((uint32(oR) * antialpha) >> 8) + ((cR * alpha) >> 8))
	nG := uint8(((uint32(oG) * antialpha) >> 8) + ((cG * alpha) >> 8))
	nB := uint8(((uint32(oB) * antialpha) >> 8) + ((cB * alpha) >> 8))

	img.SetRGBA(x, y, color.RGBA{nR, nG, nB, 255})
}
