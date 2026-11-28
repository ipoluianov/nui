package canvas

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

// Clear fills the whole image with col
func (c *Canvas) Clear(col color.Color) {
	v := color.RGBAModel.Convert(col).(color.RGBA)
	r := c.rgba.Rect
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := c.rgba.Pix[c.rgba.PixOffset(r.Min.X, y):c.rgba.PixOffset(r.Max.X, y)]
		for i := 0; i < len(row); i += 4 {
			row[i], row[i+1], row[i+2], row[i+3] = v.R, v.G, v.B, v.A
		}
	}
}
