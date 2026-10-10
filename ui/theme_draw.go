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

// drawScrollBarTrack fills the room of a scroll bar: a faint shade of the
// text color, so it suits any background and shows the room is taken.
func drawScrollBarTrack(cnv *Canvas, x, y, width, height int) {
	cnv.FillRect(x, y, width, height, withAlpha(CurrentPalette().Text, 22))
}

// The directions of the scroll bar arrows
const (
	arrowUp = iota
	arrowDown
	arrowLeft
	arrowRight
)

// drawScrollBarButton draws an arrow button of a scroll bar: the arrow in
// the text color, faint when the content can't scroll that way; a shade
// under it when the mouse is over it or it is held.
func drawScrollBarButton(cnv *Canvas, x, y, width, height, dir int, hovered, pressed, enabled bool) {
	p := CurrentPalette()
	if enabled && pressed {
		cnv.FillRect(x, y, width, height, withAlpha(p.Text, 60))
	} else if enabled && hovered {
		cnv.FillRect(x, y, width, height, withAlpha(p.Text, 30))
	}
	alpha := uint8(170)
	if !enabled {
		alpha = 60
	}
	col := withAlpha(p.Text, alpha)

	// A triangle twice as wide as high, centered in the button
	s := min(width, height)
	half := max(2, s*2/7)   // half of the base
	depth := max(1, half/2) // half of the height
	cx, cy := x+width/2, y+height/2
	switch dir {
	case arrowUp:
		cnv.FillTriangle(cx-half, cy+depth, cx+half, cy+depth, cx, cy-depth, col)
	case arrowDown:
		cnv.FillTriangle(cx-half, cy-depth, cx+half, cy-depth, cx, cy+depth, col)
	case arrowLeft:
		cnv.FillTriangle(cx+depth, cy-half, cx+depth, cy+half, cx-depth, cy, col)
	case arrowRight:
		cnv.FillTriangle(cx-depth, cy-half, cx-depth, cy+half, cx+depth, cy, col)
	}
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
