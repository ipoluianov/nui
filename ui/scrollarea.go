package ui

type ScrollArea struct {
	Widget
}

func NewScrollArea() *ScrollArea {
	var c ScrollArea
	c.InitWidget()
	c.SetTypeName("ScrollArea")
	c.SetElevation(1)
	c.SetAutoFillBackground(true)
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	c.SetAllowScroll(true, true)
	return &c
}
