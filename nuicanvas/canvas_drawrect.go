package nuicanvas

func (c *Canvas) DrawRect(x, y, w, h float64) {
	for i := float64(1); i < w-1; i += 1 {
		c.SetPixel(x+float64(i), y, 1)
		c.SetPixel(x+float64(i), y+h-1, 1)
	}
	for j := float64(0); j < h; j += 1 {
		c.SetPixel(x, y+float64(j), 1)
		c.SetPixel(x+w-1, y+float64(j), 1)
	}
}

func (c *Canvas) FillRect(x, y, w, h float64) {
	for j := float64(0); j < h; j++ {
		for i := float64(0); i < w; i++ {
			c.SetPixel(x+i, y+j, 1)
		}
	}
}
