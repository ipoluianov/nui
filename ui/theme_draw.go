package ui

import "image/color"

// indicatorColors returns the colors of a check box or radio button
// indicator: the fill, the border and the check mark.
func indicatorColors(w *Widget, checked bool) (fill, border, mark color.RGBA) {
	p := CurrentPalette()
	enabled := w.Enabled()

	if checked {
		fill, border, mark = p.Highlight, p.Highlight, p.HighlightedText
		if !enabled {
			fill = MixColors(p.Highlight, p.Window, 0.55)
			border = fill
		} else if w.IsHovered() {
			fill = MixColors(p.Highlight, p.HighlightedText, 0.12)
			border = fill
		}
		return
	}

	fill, border = p.Base, p.Border
	switch {
	case !enabled:
		fill, border = p.Window, MixColors(p.Border, p.Window, 0.5)
	case w.IsFocused():
		border = p.Highlight
	case w.IsHovered():
		border = MixColors(p.Border, p.Highlight, 0.6)
	}
	return
}

// indicatorTextColor is the text color of a check box or radio button.
func indicatorTextColor(w *Widget) color.Color {
	if !w.Enabled() {
		return CurrentPalette().DisabledText
	}
	return w.ForegroundColor()
}

// inputFrameColors returns the fill and the border of an input field.
func inputFrameColors(w *Widget) (fill, border color.RGBA) {
	p := CurrentPalette()
	fill, border = p.Base, p.Border
	if w.backgroundColor != nil {
		fill = colorToRGBA(w.backgroundColor)
	}
	switch {
	case !w.Enabled():
		fill = p.Window
	case w.IsFocused():
		border = p.Highlight
	case w.IsHovered():
		border = MixColors(p.Border, p.WindowText, 0.25)
	}
	return
}

// drawScrollBarThumb draws a scroll bar thumb in the rectangle: a rounded
// bar of the text color, translucent so it suits any background, inset so
// it doesn't touch the edges.
func drawScrollBarThumb(cnv *Canvas, x, y, width, height int, hovered bool) {
	const inset = 2
	alpha := uint8(80)
	if hovered {
		alpha = 140
	}
	w, h := width-inset*2, height-inset*2
	cnv.FillRoundedRectAA(x+inset, y+inset, w, h, min(w, h)/2, withAlpha(CurrentPalette().Text, alpha))
}
