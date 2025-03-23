package nuicanvas

import (
	"image"
	"image/color"
)

type Canvas struct {
	rgba *image.RGBA
}

func NewCanvas(rgba *image.RGBA) *Canvas {
	var c Canvas
	c.rgba = rgba
	return &c
}

func (c *Canvas) RGBA() *image.RGBA {
	return c.rgba
}

func (c *Canvas) SetPixel(x, y int, col color.Color) {
	if !(image.Pt(x, y).In(c.rgba.Bounds())) {
		return
	}
	c.rgba.Set(x, y, col)
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

func (c *Canvas) DrawRect(x, y, w, h int, col color.Color) {
	for i := 0; i < w; i++ {
		c.SetPixel(x+i, y, col)
		c.SetPixel(x+i, y+h-1, col)
	}
	for j := 0; j < h; j++ {
		c.SetPixel(x, y+j, col)
		c.SetPixel(x+w-1, y+j, col)
	}
}

func (c *Canvas) FillRect(x, y, w, h int, col color.Color) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c.SetPixel(x+i, y+j, col)
		}
	}
}

func (c *Canvas) DrawLine(x0, y0, x1, y1 int, col color.Color) {
	// Bresenham's line algorithm
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy

	for {
		c.SetPixel(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (c *Canvas) DrawFixedString(x, y int, str string, pixelSize int, col color.Color) {
	for i, ch := range str {
		c.DrawFixedChar(x+i*6*pixelSize, y, byte(ch), pixelSize, col)
	}
}

func (c *Canvas) DrawFixedChar(x, y int, ch byte, pixelSize int, col color.Color) {
	charMask := GetChar(ch)
	if len(charMask) != 35 {
		return
	}

	for yi := 0; yi < 7; yi++ {
		for xi := 0; xi < 5; xi++ {
			if charMask[yi*5+xi] == 1 {
				c.FillRect(x+xi*pixelSize, y+yi*pixelSize, pixelSize, pixelSize, col)
			}
			//c.DrawRect(x+xi*pixelSize, y+yi*pixelSize, pixelSize, pixelSize, color.RGBA{50, 50, 50, 255})
		}
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
