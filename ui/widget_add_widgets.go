package ui

func (c *Widget) AddPanel(row int, col int) *Panel {
	panel := NewPanel()
	c.AddWidget(row, col, panel)
	return panel
}

func (c *Widget) AddLabel(row int, col int, text string) *Label {
	lbl := NewLabel(text)
	c.AddWidget(row, col, lbl)
	return lbl
}

func (c *Widget) AddButton(row int, col int, text string, onPush func()) *Button {
	btn := NewButton(text)
	btn.SetOnClick(onPush)
	c.AddWidget(row, col, btn)
	return btn
}

func (c *Widget) AddHSpacer(row int, col int) *HSpacer {
	hspacer := NewHSpacer()
	c.AddWidget(row, col, hspacer)
	return hspacer
}

func (c *Widget) AddVSpacer(row int, col int) *VSpacer {
	vspacer := NewVSpacer()
	c.AddWidget(row, col, vspacer)
	return vspacer
}
