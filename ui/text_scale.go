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
// the form the UI thread works on now (see Form.useTextScale). Measured
// from another goroutine, the text is measured for whichever that is.
func textScale() float64 {
	if b := textScaleBits.Load(); b != 0 {
		return math.Float64frombits(b)
	}
	return 1
}

// useTextScale makes the text measured for the screen of the form's window,
// before the form lays out, paints or handles an event. Each form has its
// own scale: with windows on screens of different scales, a scale for all
// of them made each paint lay the others out again, for ever, and their
// texts jumped. Only the form whose window moved to a screen of another
// scale is laid out again.
func (c *Form) useTextScale() {
	if c == nil || c.wnd == nil {
		return
	}
	scale := c.wnd.Scale()
	if scale <= 0 {
		return
	}
	textScaleBits.Store(math.Float64bits(scale))
	if c.textScale == scale {
		return
	}
	laidOut := c.textScale != 0
	c.textScale = scale
	if laidOut {
		// Its texts have other sizes now
		c.applyFontSize()
	}
}

// toLogicalText converts a size of text in the pixels of the screen to
// logical pixels
func toLogicalText(v int, scale float64) int {
	return int(math.Round(float64(v) / scale))
}
