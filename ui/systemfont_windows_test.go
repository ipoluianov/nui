package ui

import (
	"image"
	"image/color"
	"testing"
)

// On Windows the built-in fonts are drawn by GDI with ClearType
func TestBuiltinFontDrawnNatively(t *testing.T) {
	if systemFontFor(FontFamilySans, 14, "Hello") == nil {
		t.Fatal("Noto Sans isn't drawn by GDI")
	}
	if systemFontFor(FontFamilyMono, 14, "Hello") == nil {
		t.Fatal("JetBrains Mono isn't drawn by GDI")
	}
	// Characters the font lacks go to nui's rasterizer and its fallbacks
	if systemFontFor(FontFamilySans, 14, "中文") != nil {
		t.Error("Chinese text is drawn by GDI with Noto Sans")
	}

	opaque := image.NewRGBA(image.Rect(0, 0, 80, 30))
	for i := range opaque.Pix {
		opaque.Pix[i] = 255
	}
	DrawText(opaque, "Hello", color.RGBA{0, 0, 0, 255}, FontFamilySans, 14, 2, 2, 0, 0, 80, 30)
	dark, colored := 0, 0
	for i := 0; i < len(opaque.Pix); i += 4 {
		p := opaque.Pix[i : i+3]
		if p[0] < 200 || p[1] < 200 || p[2] < 200 {
			dark++
			if p[0] != p[1] || p[1] != p[2] {
				colored++
			}
		}
	}
	if dark < 30 || colored == 0 {
		t.Errorf("on white: %d dark pixels, %d ClearType-colored", dark, colored)
	}

	// On a transparent image the text is drawn in grayscale into the alpha
	clear := image.NewRGBA(image.Rect(0, 0, 80, 30))
	DrawText(clear, "Hello", color.RGBA{0, 0, 0, 255}, FontFamilySans, 14, 2, 2, 0, 0, 80, 30)
	visible := 0
	for i := 3; i < len(clear.Pix); i += 4 {
		if clear.Pix[i] > 128 {
			visible++
		}
	}
	if visible < 20 {
		t.Errorf("on transparent: %d visible pixels", visible)
	}

	// Lines with fallback characters are as high as the others
	_, h1, _ := MeasureText(FontFamilySans, 14, "Ag")
	_, h2, _ := MeasureText(FontFamilySans, 14, "Ag\u0378")
	if h1 != h2 {
		t.Errorf("line height %d, with a missing character %d", h1, h2)
	}
}
