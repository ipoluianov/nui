package ui

import (
	"image"
	"image/color"
)

// FillRoundedRectAA fills a rounded rectangle with antialiased corners.
// Unlike FillRoundedRect it doesn't go through gg, which builds a clip mask
// of the whole canvas on every call, so it's cheap enough for every control.
func (c *Canvas) FillRoundedRectAA(x, y, width, height, radius int, col color.RGBA) {
	if width <= 0 || height <= 0 {
		return
	}
	radius = max(0, min(radius, width/2, height/2))
	if radius == 0 {
		c.FillRect(x, y, width, height, col)
		return
	}

	// The middle band and the parts of the top and bottom bands between the corners
	c.FillRect(x, y+radius, width, height-radius*2, col)
	c.FillRect(x+radius, y, width-radius*2, radius, col)
	c.FillRect(x+radius, y+height-radius, width-radius*2, radius, col)

	// The corners: each pixel covered as much as the quarter circle covers it
	for cy := 0; cy < radius; cy++ {
		for cx := 0; cx < radius; cx++ {
			coverage := cornerCoverage(cx, cy, radius)
			if coverage == 0 {
				continue
			}
			pixel := col
			pixel.A = uint8(float64(col.A)*coverage + 0.5)
			c.blendPixel(x+cx, y+cy, pixel)
			c.blendPixel(x+width-1-cx, y+cy, pixel)
			c.blendPixel(x+cx, y+height-1-cy, pixel)
			c.blendPixel(x+width-1-cx, y+height-1-cy, pixel)
		}
	}
}

// FillFrame draws a control's frame: a rounded rectangle filled with fill
// and outlined with a 1px border. fill must be opaque.
func (c *Canvas) FillFrame(x, y, width, height, radius int, fill, border color.Color) {
	c.FillRoundedRectAA(x, y, width, height, radius, colorToRGBA(border))
	c.FillRoundedRectAA(x+1, y+1, width-2, height-2, radius-1, colorToRGBA(fill))
}

// cornerCoverage returns how much of the pixel (cx, cy) of a top-left
// corner square is inside the quarter circle of the radius centered at
// (radius, radius), sampling it 4x4.
func cornerCoverage(cx, cy, radius int) float64 {
	const samples = 4
	r := float64(radius)
	inside := 0
	for sy := 0; sy < samples; sy++ {
		for sx := 0; sx < samples; sx++ {
			dx := float64(cx) + (float64(sx)+0.5)/samples - r
			dy := float64(cy) + (float64(sy)+0.5)/samples - r
			if dx*dx+dy*dy <= r*r {
				inside++
			}
		}
	}
	return float64(inside) / (samples * samples)
}

// blendPixel blends the straight (not premultiplied) color into the pixel
// at (x, y) in the current coordinates, inside the clip rectangle only.
func (c *Canvas) blendPixel(x, y int, col color.RGBA) {
	x += c.state.translateX
	y += c.state.translateY
	if x < c.state.clipX || x >= c.state.clipX+c.state.clipW || y < c.state.clipY || y >= c.state.clipY+c.state.clipH {
		return
	}
	if !(image.Point{x, y}).In(c.rgba.Rect) {
		return
	}
	if col.A == 255 {
		c.rgba.SetRGBA(x, y, col)
		return
	}
	c.MixPixel(x, y, col)
}
