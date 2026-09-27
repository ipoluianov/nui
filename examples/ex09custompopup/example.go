package ex09custompopup

import "github.com/ipoluianov/nui/ui"

// NewExampleForm shows a popup widget of your own: a color palette that
// drops down from a button. Like every popup it's shown in its own window,
// so it extends beyond the small form, and it opens above the button when it
// doesn't fit below it (PopupFlipped).
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Custom Popup")
	form.SetSize(320, 160)

	sample := form.Panel().AddLabel(0, 0, "Sample text")

	btn := ui.NewButton("Choose color...")
	btn.SetOnClick(func() {
		palette := newPalette(func(name, hex string) {
			sample.SetText("Sample text: " + name)
			sample.SetForegroundColor(ui.ColorFromHex(hex))
		})
		palette.openBelow(btn)
	})
	form.Panel().AddWidget(1, 0, btn)
	form.Panel().AddVSpacer(2, 0)

	return form
}

var paletteColors = []struct{ name, hex string }{
	{"Red", "#E53935"}, {"Orange", "#FB8C00"}, {"Yellow", "#FDD835"},
	{"Green", "#43A047"}, {"Teal", "#00897B"}, {"Blue", "#1E88E5"},
	{"Indigo", "#3949AB"}, {"Purple", "#8E24AA"}, {"Gray", "#757575"},
}

const (
	paletteColumns = 3
	paletteCellW   = 120
	paletteCellH   = 40
)

// palette is the popup widget: a grid of color buttons.
type palette struct {
	ui.Widget

	// anchorX, anchorY, anchorW: the button the palette drops down from,
	// in the form's client coordinates
	anchorX, anchorY, anchorW int
}

func newPalette(onPick func(name, hex string)) *palette {
	var c palette
	c.InitWidget()
	c.SetAutoFillBackground(true)
	c.SetRole("popup")

	for i, pc := range paletteColors {
		btn := ui.NewButton(pc.name)
		btn.SetForegroundColor(ui.ColorFromHex(pc.hex))
		btn.SetOnClick(func() {
			onPick(pc.name, pc.hex)
			c.Form().CloseTopPopup()
		})
		c.AddWidget(i/paletteColumns, i%paletteColumns, btn)
	}
	return &c
}

// openBelow opens the palette under the button.
func (c *palette) openBelow(anchor *ui.Button) {
	c.anchorX, c.anchorY = anchor.RectClientAreaOnWindow()
	c.anchorW = anchor.Width()

	rows := (len(paletteColors) + paletteColumns - 1) / paletteColumns
	c.SetSize(paletteColumns*paletteCellW, rows*paletteCellH)
	c.SetPosition(c.anchorX, c.anchorY+anchor.Height())
	anchor.Form().OpenPopup(c)
}

// PopupFlipped opens the palette above the button when it doesn't fit
// below, and aligns it to the button's right edge when it doesn't fit to
// the right.
func (c *palette) PopupFlipped() (x, y int) {
	return c.anchorX + c.anchorW - c.Width(), c.anchorY - c.Height()
}
