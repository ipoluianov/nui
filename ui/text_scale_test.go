package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// On a scaled screen the cursor at the end of a line stands where the drawn
// text ends: the advances rounded at the size in pixels used to add up to
// half a pixel a character more than the measured ones
func TestTextMeasuredAtScreenScale(t *testing.T) {
	defer setTestTextScale(1)
	line := "авпвпвпвыааууцуеуекуеукеуеуеуеекуеуеу11111111111111111 abc"
	for _, family := range []string{FontFamilyMono, FontFamilySans} {
		for _, scale := range []float64{1, 1.25, 1.5, 2} {
			setTestTextScale(scale)
			img := image.NewRGBA(image.Rect(0, 0, 1600, 80))
			cnv := NewCanvasScaled(img, scale)
			cnv.SetDirectTranslateAndClip(0, 0, 800, 40)
			cnv.SetFontFamily(family)
			cnv.SetFontSize(14)
			cnv.SetColor(color.Black)
			cnv.DrawText(0, 0, 800, 40, line)
			drawn := float64(inkRightEdge(img)) / scale

			positions, err := GetCharPositions(family, 14, line)
			if err != nil {
				t.Fatal(err)
			}
			width, _, _ := MeasureText(family, 14, line)
			end := positions[len(positions)-1]
			if end != width {
				t.Errorf("%s at %v: last position %d, width %d", family, scale, end, width)
			}
			// The ink of the last letter ends a little before its advance
			if d := float64(end) - drawn; d < -1 || d > 3 {
				t.Errorf("%s at %v: the text drawn to %.1f, the cursor at the end at %d", family, scale, drawn, end)
			}
		}
	}
}

// inkRightEdge returns the x after the rightmost pixel drawn on img
func inkRightEdge(img *image.RGBA) int {
	b := img.Bounds()
	for x := b.Max.X - 1; x >= b.Min.X; x-- {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if img.RGBAAt(x, y).A > 128 {
				return x + 1
			}
		}
	}
	return 0
}

// A text box filled before its window is shown (at the scale 1) measures its
// lines again for the scale of the screen: drawn clipped to the old width,
// the end of the line was cut off
func TestTextBoxRemeasuredForScale(t *testing.T) {
	defer setTestTextScale(1)
	setTestTextScale(1)
	tb := NewTextBox()
	tb.SetFontFamily("jetbrainsmono:bold:italic")
	tb.SetText("Text box in JetBrains Mono Bold Italic: click between letters")
	setTestTextScale(2)
	tb.applyThemeMetrics() // as the form does when the scale changes
	want, _, _ := MeasureText(tb.FontFamily(), tb.FontSize(), tb.Text())
	if tb.lineWidths[0] != want {
		t.Errorf("line width %d, measured at the scale %d", tb.lineWidths[0], want)
	}
}

// setTestTextScale makes the text measured for a screen of the scale, as
// Form.useTextScale does for the screen of a form
func setTestTextScale(scale float64) {
	textScaleBits.Store(math.Float64bits(scale))
}
