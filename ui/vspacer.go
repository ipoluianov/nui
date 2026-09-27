package ui

type VSpacer struct {
	Widget
}

func NewVSpacer() *VSpacer {
	var c VSpacer
	c.InitWidget()
	c.SetTypeName("VSpacer")
	c.SetYExpandable(true)
	return &c
}
