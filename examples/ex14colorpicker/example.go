package ex14colorpicker

import (
	"image/color"

	"github.com/ipoluianov/nui/ui"
)

// NewExampleForm shows color pickers: the sample text takes the chosen
// colors live, while they're being picked. Escape in the panel returns to
// the color it was opened with; the colors chosen are offered as recent in
// every picker.
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Color Picker")
	form.SetSize(460, 260)
	panel := form.Panel()

	sample := ui.NewLabel("Sample text")
	sample.SetTextAlign(ui.HAlignCenter)
	sample.SetAutoFillBackground(true)
	sample.SetYExpandable(true)

	fields := panel.AddPanel(0, 0)
	fields.SetCellPadding(6)

	fields.AddLabel(0, 0, "Text color")
	textColor := ui.NewColorPicker()
	textColor.SetColor(ui.ColorFromHex("#FFFFFF"))
	textColor.SetOnColorChanged(func(col color.RGBA) { sample.SetForegroundColor(col) })
	fields.AddWidget(0, 1, textColor)

	fields.AddLabel(1, 0, "Background")
	background := ui.NewColorPicker()
	background.SetColor(ui.ColorFromHex("#1E88E5"))
	background.SetOnColorChanged(func(col color.RGBA) { sample.SetBackgroundColor(col) })
	fields.AddWidget(1, 1, background)

	fields.AddLabel(2, 0, "With opacity")
	overlay := ui.NewColorPicker()
	overlay.SetAlphaEnabled(true)
	overlay.SetColor(ui.ColorFromHex("#FF980080"))
	fields.AddWidget(2, 1, overlay)

	sample.SetForegroundColor(textColor.Color())
	sample.SetBackgroundColor(background.Color())
	panel.AddWidget(1, 0, sample)

	return form
}
