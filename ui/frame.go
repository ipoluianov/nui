package ui

type Frame struct {
	Widget
}

func NewFrame() *Frame {
	var c Frame
	c.InitWidget()
	c.SetTypeName("Frame")
	c.SetElevation(1)
	c.SetAutoFillBackground(true)
	return &c
}
