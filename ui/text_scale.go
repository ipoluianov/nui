package ui

import (
	"math"
	"sync/atomic"
)

// The text is drawn at the size it takes in the pixels of the screen
// (Canvas.DrawText draws at fontSize times the scale), and the advances of
// the characters are rounded at that size: at 2x a character of JetBrains
// Mono 14 px is 17 pixels, 8.5 logical ones, while at 14 px it is 8. Measured
// in logical pixels the text came out shorter than drawn by half a pixel a
// character, and the cursor, the selection and the alignment were off. So
// MeasureText and GetCharPositions measure in the pixels of the screen and
// convert the result to logical pixels: off by half a pixel at most, not
// adding up along the line.

// textScaleBits is the scale of the screen the text is measured for, as
// math.Float64bits; 0 is 1. Measuring is safe from any goroutine.
var textScaleBits atomic.Uint64

// textScale returns the scale of the screen the text is measured for: of
// the last window created or painted. One for all the forms: with screens
// of different scales the text of a window on the other one may be off by
// a little again.
func textScale() float64 {
	if b := textScaleBits.Load(); b != 0 {
		return math.Float64frombits(b)
	}
	return 1
}

// setTextScale sets the scale of the screen the text is measured for, and
// lays the open forms out again when it changed: their texts have other
// sizes now
func setTextScale(scale float64) {
	if scale <= 0 || scale == textScale() {
		return
	}
	textScaleBits.Store(math.Float64bits(scale))
	lineHeightMu.Lock()
	lineHeightValue = 0
	lineHeightMu.Unlock()
	for _, form := range openForms() {
		form.applyFontSize()
	}
}

// toLogicalText converts a size of text in the pixels of the screen to
// logical pixels
func toLogicalText(v int, scale float64) int {
	return int(math.Round(float64(v) / scale))
}
