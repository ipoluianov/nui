package nuicanvas

func (c *Canvas) DrawFixedString(x, y float64, str string, pixelSize float64) {
	for i, ch := range str {
		c.DrawFixedChar(x+float64(i)*6*pixelSize, y, byte(ch), pixelSize)
	}
}

func (c *Canvas) DrawFixedChar(x, y float64, ch byte, pixelSize float64) {
	charMask := GetChar(ch)
	if len(charMask) != 35 {
		return
	}

	for yi := 0; yi < 7; yi++ {
		for xi := 0; xi < 5; xi++ {
			if charMask[yi*5+xi] == 1 {
				//c.FillRect(x+float64(xi)*pixelSize, y+float64(yi)*pixelSize, pixelSize, pixelSize)
				c.BlendPixel(int(x+float64(xi)*pixelSize), int(y+float64(yi)*pixelSize), c.CurrentState().col)
			}
		}
	}
}
