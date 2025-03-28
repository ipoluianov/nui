package nuicanvas

import (
	"image"
	"image/color"
)

type Canvas struct {
	rgba *image.RGBA

	statesStack []*State
}

func NewCanvas(rgba *image.RGBA) *Canvas {
	var c Canvas
	c.rgba = rgba
	c.statesStack = make([]*State, 0)
	c.statesStack = append(c.statesStack, NewState())
	return &c
}

func (c *Canvas) RGBA() *image.RGBA {
	return c.rgba
}

func (c *Canvas) Width() int {
	return c.rgba.Bounds().Dx()
}

func (c *Canvas) Height() int {
	return c.rgba.Bounds().Dy()
}

func (c *Canvas) Clear(col color.Color) {
	dataSize := c.rgba.Bounds().Dx() * c.rgba.Bounds().Dy() * 4
	for i := 0; i < dataSize; i += 4 {
		c.rgba.Pix[i] = col.(color.RGBA).R
		c.rgba.Pix[i+1] = col.(color.RGBA).G
		c.rgba.Pix[i+2] = col.(color.RGBA).B
		c.rgba.Pix[i+3] = col.(color.RGBA).A
	}
}
