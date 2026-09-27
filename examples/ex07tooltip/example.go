package ex07tooltip

import (
	"image"
	"image/color"

	"github.com/ipoluianov/nui/ui"
)

// NewExampleForm shows tooltips on different widgets. The form is small on
// purpose: long tooltips and the ones near the right edge extend beyond the
// window instead of being clipped by it.
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Tooltips")
	form.SetSize(360, 240)

	panel := form.Panel()

	toolbar := panel.AddPanel(0, 0)
	toolbar.AddWidget(0, 0, ui.NewToolButton(toolIcon(color.RGBA{0x4C, 0xAF, 0x50, 0xFF}), "New", nil))
	toolbar.AddWidget(0, 1, ui.NewToolButton(toolIcon(color.RGBA{0x21, 0x96, 0xF3, 0xFF}), "Open file", nil))
	toolbar.AddWidget(0, 2, ui.NewToolButton(toolIcon(color.RGBA{0xFF, 0x98, 0x00, 0xFF}), "Save all open documents to disk", nil))
	toolbar.AddHSpacer(0, 3)
	toolbar.AddWidget(0, 4, ui.NewToolButton(toolIcon(color.RGBA{0xF4, 0x43, 0x36, 0xFF}), "This button is at the right edge, so its tooltip goes beyond the window", nil))

	label := ui.NewLabel("Rest the mouse on a widget")
	label.SetTooltip("Labels can have tooltips too")
	panel.AddWidget(1, 0, label)

	txt := ui.NewTextBox()
	txt.SetTooltip("Type anything here. The tooltip disappears when you press a key")
	panel.AddWidget(2, 0, txt)

	chk := ui.NewCheckbox("Checkbox")
	chk.SetTooltip("A checkbox with a tooltip")
	panel.AddWidget(3, 0, chk)

	btn := ui.NewButton("Button with a long tooltip")
	btn.SetTooltip("A very long tooltip that does not fit into this small window at all, " +
		"so it is shown in a separate popup window")
	panel.AddWidget(4, 0, btn)

	panel.AddButton(5, 0, "No tooltip", nil)
	panel.AddVSpacer(6, 0)

	return form
}

// toolIcon draws a simple colored square with a lighter border.
func toolIcon(col color.RGBA) image.Image {
	const size = 28
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	border := color.RGBA{
		R: uint8(min(int(col.R)+60, 255)),
		G: uint8(min(int(col.G)+60, 255)),
		B: uint8(min(int(col.B)+60, 255)),
		A: 0xFF,
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if x < 2 || y < 2 || x >= size-2 || y >= size-2 {
				img.Set(x, y, border)
			} else {
				img.Set(x, y, col)
			}
		}
	}
	return img
}
