package ui

type Space struct {
	Widget
}

func NewSpace() *Space {
	var c Space
	c.InitWidget()
	c.SetPanelPadding(0)
	c.SetCellPadding(0)
	c.SetTypeName("Space")
	return &c
}

func (c *Space) SetSize(width, height int) {
	// Fixed size
	c.SetProp("width", width)
	c.SetProp("height", height)
	c.SetMinSize(width, height)
	c.SetMaxSize(width, height)
	c.SetXExpandable(false)
	c.SetYExpandable(false)
	// The layout engine also calls SetSize (via the Widgeter interface) on
	// every layout pass to apply the computed on-screen size. Without this,
	// c.w/c.h are never touched and stay at the InitWidget() defaults
	// (300x180), so the widget's actual clickable/paintable area never
	// shrinks to the declared fixed size - it silently overlaps whatever
	// comes after it in the layout.
	c.Widget.SetSize(width, height)
}

func (c *Space) MinWidth() int {
	return c.GetPropInt("width", 0)
}

func (c *Space) MinHeight() int {
	return c.GetPropInt("height", 0)
}

func (c *Space) MaxWidth() int {
	return c.GetPropInt("width", 0)
}

func (c *Space) MaxHeight() int {
	return c.GetPropInt("height", 0)
}
