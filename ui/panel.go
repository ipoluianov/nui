package ui

type Panel struct {
	Widget
}

func NewPanel() *Panel {
	var c Panel
	c.InitWidget()
	c.SetTypeName("Panel")
	return &c
}
