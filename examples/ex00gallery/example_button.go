package ex00gallery

import (
	"fmt"
	"image"
	"image/color"

	"github.com/ipoluianov/nui/ui"
)

type ExamplePageButton struct {
	ui.Widget

	lblStatus  *ui.Label
	clickCount int
}

func NewExamplePageButton() *ExamplePageButton {
	var c ExamplePageButton
	c.InitWidget()

	row := 0

	c.lblStatus = c.AddLabel(row, 0, "Click a button below")
	c.lblStatus.SetUnderline(true)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Basic button")
	c.AddButton(row, 0, "Click me", func() {
		c.clickCount++
		c.setStatus(fmt.Sprintf("Basic button clicked %d time(s)", c.clickCount))
	})
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Disabled button")
	btnDisabled := c.AddButton(row, 0, "Can't click me", func() {
		c.setStatus("This should never show up")
	})
	btnDisabled.SetEnabled(false)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Button sized to its text")
	c.AddButton(row, 0, "A button with a much longer label", func() {
		c.setStatus("Long button clicked")
	})
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Multiple buttons, shared status")
	c.AddButton(row, 0, "Yes", func() { c.setStatus("You chose: Yes") })
	c.AddButton(row, 1, "No", func() { c.setStatus("You chose: No") })
	c.AddButton(row, 2, "Cancel", func() { c.setStatus("You chose: Cancel") })
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Image button (click cycles the image alignment)")
	c.addAlignCyclingImageButton(row, 0, "Red", ui.ColorFromHex("#E53935"))
	c.addAlignCyclingImageButton(row, 1, "Green", ui.ColorFromHex("#43A047"))
	c.addAlignCyclingImageButton(row, 2, "Blue", ui.ColorFromHex("#1E88E5"))
	row = addSectionGap(&c.Widget, row+1)

	c.AddVSpacer(row, 0)

	return &c
}

func (c *ExamplePageButton) setStatus(text string) {
	c.lblStatus.SetText(text)
}

// imageAlignCycle is the sequence of alignments an image button steps
// through on each click, demonstrating ButtonImage's halign/valign props.
var imageAlignCycle = []struct {
	h    ui.HAlign
	v    ui.VAlign
	name string
}{
	{ui.HAlignLeft, ui.VAlignTop, "left/top"},
	{ui.HAlignCenter, ui.VAlignTop, "center/top"},
	{ui.HAlignRight, ui.VAlignTop, "right/top"},
	{ui.HAlignRight, ui.VAlignCenter, "right/center"},
	{ui.HAlignRight, ui.VAlignBottom, "right/bottom"},
	{ui.HAlignCenter, ui.VAlignBottom, "center/bottom"},
	{ui.HAlignLeft, ui.VAlignBottom, "left/bottom"},
	{ui.HAlignLeft, ui.VAlignCenter, "left/center"},
	{ui.HAlignCenter, ui.VAlignCenter, "center/center"},
}

// addAlignCyclingImageButton adds a ButtonImage that steps to the next
// entry in imageAlignCycle each time it's clicked, so repeated clicks
// visibly move the icon around the button.
func (c *ExamplePageButton) addAlignCyclingImageButton(row, col int, name string, fg color.Color) *ui.ButtonImage {
	btn := ui.NewButtonImage(iconImage(20, fg))
	alignIndex := 0
	btn.SetOnButtonClick(func(b *ui.ButtonImage) {
		alignIndex = (alignIndex + 1) % len(imageAlignCycle)
		a := imageAlignCycle[alignIndex]
		b.SetImageHAlign(a.h)
		b.SetImageVAlign(a.v)
		c.setStatus(fmt.Sprintf("%s image button aligned: %s", name, a.name))
	})
	c.AddWidget(row, col, btn)
	return btn
}

// iconImage renders a simple filled-circle placeholder icon, since
// ButtonImage takes a plain image.Image with no built-in asset loading.
func iconImage(size int, fg color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	center := size / 2
	radius := size / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := x - center
			dy := y - center
			if dx*dx+dy*dy <= radius*radius {
				img.Set(x, y, fg)
			}
		}
	}
	return img
}
