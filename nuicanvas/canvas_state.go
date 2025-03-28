package nuicanvas

import "image/color"

type State struct {
	trasformX float64
	trasformY float64
	clipX     float64
	clipY     float64
	clipW     float64
	clipH     float64

	col       color.RGBA
	lineWidth float64
}

func NewState() *State {
	c := State{}
	c.trasformX = 0
	c.trasformY = 0
	c.clipX = 0
	c.clipY = 0
	c.clipW = 2147483647
	c.clipH = 2147483647
	return &c
}

func (c *Canvas) CurrentState() *State {
	if len(c.statesStack) == 0 {
		return NewState()
	}
	return c.statesStack[len(c.statesStack)-1]
}

func (c *Canvas) Transform(x float64, y float64) {
	c.CurrentState().trasformX += x
	c.CurrentState().trasformY += y
}

func (c *Canvas) Clip(x float64, y float64, w float64, h float64) {
	c.CurrentState().clipX = x
	c.CurrentState().clipY = y
	c.CurrentState().clipW = w
	c.CurrentState().clipH = h
}

func (c *Canvas) SetColor(col color.RGBA) {
	c.CurrentState().col = col
}

func (c *Canvas) SetLineWidth(w float64) {
	c.CurrentState().lineWidth = w
}

func (c *Canvas) Save() {
	c.statesStack = append(c.statesStack, c.CurrentState())
}

func (c *Canvas) Restore() {
	if len(c.statesStack) > 1 {
		c.statesStack = c.statesStack[:len(c.statesStack)-1]
	}
}

func (c *Canvas) checkClip(x, y float64) bool {
	state := c.CurrentState()
	if x < state.clipX || x >= state.clipX+state.clipW {
		return false
	}
	if y < state.clipY || y >= state.clipY+state.clipH {
		return false
	}
	return true
}
