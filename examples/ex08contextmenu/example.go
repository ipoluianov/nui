package ex08contextmenu

import (
	"image"
	"image/color"

	"github.com/ipoluianov/nui/ui"
)

// NewExampleForm shows context menus. The form is small on purpose: the
// menus are shown in their own windows, so they extend beyond the form, and
// near the edge of the screen they open to the left and upwards.
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Context Menus")
	form.SetSize(380, 240)

	panel := form.Panel()
	status := panel.AddLabel(0, 0, "Right-click the areas below")
	setStatus := func(text string) { status.SetText(text) }

	// Icons, separators and nested submenus
	area := newArea("Icons, separators, submenus")
	panel.AddWidget(1, 0, area)

	menu := ui.NewContextMenu(area)
	menu.AddItem("Cut", func() { setStatus("Cut") }).SetImage(icon(color.RGBA{0xF4, 0x43, 0x36, 0xFF}))
	menu.AddItem("Copy", func() { setStatus("Copy") }).SetImage(icon(color.RGBA{0x21, 0x96, 0xF3, 0xFF}))
	menu.AddItem("Paste", func() { setStatus("Paste") }).SetImage(icon(color.RGBA{0x4C, 0xAF, 0x50, 0xFF}))
	menu.AddSeparator()

	colors := ui.NewContextMenu(area)
	colors.AddItem("Red", func() { setStatus("Color: Red") })
	colors.AddItem("Green", func() { setStatus("Color: Green") })
	colors.AddItem("Blue", func() { setStatus("Color: Blue") })

	shades := ui.NewContextMenu(area)
	shades.AddItem("Light gray", func() { setStatus("Color: Light gray") })
	shades.AddItem("Dark gray", func() { setStatus("Color: Dark gray") })
	colors.AddSeparator()
	colors.AddItemWithSubmenu("Grays", shades)

	menu.AddItemWithSubmenu("Color", colors)
	menu.AddSeparator()
	menu.AddItem("Properties", func() { setStatus("Properties") })
	area.SetContextMenu(menu)

	// A long menu that doesn't fit into the form
	areaLong := newArea("A long menu")
	panel.AddWidget(2, 0, areaLong)

	menuLong := ui.NewContextMenu(areaLong)
	for _, name := range []string{"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"} {
		menuLong.AddItem(name, func() { setStatus("Month: " + name) })
	}
	menuLong.AddSeparator()
	menuLong.AddItem("A long item that makes the menu wide", func() {
		setStatus("The long item")
	})
	areaLong.SetContextMenu(menuLong)

	return form
}

// newArea is a label to right-click, stretched to fill its row.
func newArea(hint string) *ui.Label {
	area := ui.NewLabel(hint)
	area.SetTextAlign(ui.HAlignCenter)
	area.SetAutoFillBackground(true)
	area.SetElevation(1)
	area.SetYExpandable(true)
	return area
}

// icon draws a small filled circle.
func icon(col color.RGBA) image.Image {
	const size = 16
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	r := float64(size)/2 - 1
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - float64(size)/2 + 0.5
			dy := float64(y) - float64(size)/2 + 0.5
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, col)
			}
		}
	}
	return img
}
