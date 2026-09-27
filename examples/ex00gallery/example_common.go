package ex00gallery

import "github.com/ipoluianov/nui/ui"

// addSectionHeader adds an underlined header label for a demo section
// and returns the row where the section's content should start.
func addSectionHeader(w *ui.Widget, row int, text string) int {
	header := w.AddLabel(row, 0, text)
	header.SetUnderline(true)
	header.SetFontSize(ui.DefaultFontSize * 1.1)
	return row + 1
}

// addSectionGap adds a fixed-height empty row so demo sections don't visually merge together.
func addSectionGap(w *ui.Widget, row int) int {
	gap := ui.NewSpace()
	gap.SetSize(1, 20)
	w.AddWidget(row, 0, gap)
	return row + 1
}
