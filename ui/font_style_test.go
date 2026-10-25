package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestParseFontFamily(t *testing.T) {
	for _, tc := range []struct {
		in     string
		family string
		style  FontStyle
	}{
		{"notosans", "notosans", FontRegular},
		{"Consolas:Bold", "Consolas", FontBold},
		{"jetbrainsmono:italic:bold", "jetbrainsmono", FontBold | FontItalic},
		{"x:oblique", "x", FontItalic},
		{"x:regular", "x", FontRegular},
		{"", "", FontRegular},
	} {
		family, style := parseFontFamily(tc.in)
		if family != tc.family || style != tc.style {
			t.Errorf("%q: %q %v, want %q %v", tc.in, family, style, tc.family, tc.style)
		}
	}
	if got := StyledFont("notosans:italic", FontBold); got != "notosans:bold:italic" {
		t.Errorf("StyledFont = %q", got)
	}
	if got := fontWithStyle("notosans:bold:italic", FontBold, false); got != "notosans:italic" {
		t.Errorf("fontWithStyle = %q", got)
	}
}

// The built-in faces are real ones: JetBrains Mono Bold is as wide as the
// regular face (a synthesized bold would be wider), Noto Sans Bold is wider
func TestBuiltinFaces(t *testing.T) {
	text := "Hello, Привет 0123"
	for _, style := range []FontStyle{FontBold, FontItalic, FontBold | FontItalic} {
		regular, _, _ := MeasureText(FontFamilyMono, 14, text)
		styled, _, _ := MeasureText(StyledFont(FontFamilyMono, style), 14, text)
		if styled != regular {
			t.Errorf("mono %v: width %d, regular %d", style, styled, regular)
		}
		if !isRegisteredFace(faceName(FontFamilyMono, style)) {
			t.Errorf("mono %v is not loaded", style)
		}
	}
	regular, _, _ := MeasureText(FontFamilySans, 14, text)
	bold, _, _ := MeasureText(StyledFont(FontFamilySans, FontBold), 14, text)
	if bold <= regular {
		t.Errorf("sans bold width %d, regular %d", bold, regular)
	}
	if !isRegisteredFace(FontFamilySans + ":bold") {
		t.Error("sans bold is not loaded")
	}

	img1 := renderText("Aa", color.Black, FontFamilySans, 20)
	img2 := renderText("Aa", color.Black, FontFamilySans+":italic", 20)
	if bytes.Equal(img1.Pix, img2.Pix) {
		t.Error("italic is drawn as regular")
	}
}

func isRegisteredFace(name string) bool {
	fontsMu.RLock()
	defer fontsMu.RUnlock()
	_, ok := fonts[name]
	return ok
}

// A family with the regular face only gets the other styles synthesized
func TestSynthesizedStyles(t *testing.T) {
	if err := RegisterFont("test-synth", goregular.TTF); err != nil {
		t.Fatal(err)
	}
	text := "Hello"
	regular, h1, _ := MeasureText("test-synth", 20, text)
	bold, h2, _ := MeasureText("test-synth:bold", 20, text)
	italic, _, _ := MeasureText("test-synth:italic", 20, text)
	if bold != regular+len(text) { // one pixel per glyph at 20 px
		t.Errorf("bold width %d, regular %d", bold, regular)
	}
	if italic != regular || h1 != h2 {
		t.Errorf("italic width %d, regular %d; heights %d %d", italic, regular, h1, h2)
	}
	positions, _ := GetCharPositions("test-synth:bold", 20, text)
	if positions[len(positions)-1] != bold {
		t.Errorf("positions end at %d, width %d", positions[len(positions)-1], bold)
	}

	r := renderText("l", color.Black, "test-synth", 20)
	b := renderText("l", color.Black, "test-synth:bold", 20)
	i := renderText("l", color.Black, "test-synth:italic", 20)
	if ink(b) <= ink(r) {
		t.Errorf("bold ink %d, regular %d", ink(b), ink(r))
	}
	// The top of an italic "l" leans right of its bottom
	top, bottom := inkCenter(i, 0, i.Rect.Dy()/3), inkCenter(i, i.Rect.Dy()*2/3, i.Rect.Dy()*5/6)
	if top <= bottom {
		t.Errorf("italic l: top at x %.1f, bottom at x %.1f", top, bottom)
	}

	// The top of the last italic letter stands out of its advance: not cut off
	advance, _, _ := MeasureText("test-synth:italic", 20, "I")
	if img := renderText("I", color.Black, "test-synth:italic", 20); img.Rect.Max.X <= advance {
		t.Errorf("italic I: image to x %d, advance %d", img.Rect.Max.X, advance)
	}

	// A face registered later replaces the synthesized one
	if err := RegisterFont("test-synth:bold", goregular.TTF); err != nil {
		t.Fatal(err)
	}
	if w, _, _ := MeasureText("test-synth:bold", 20, text); w != regular {
		t.Errorf("registered bold width %d, want %d", w, regular)
	}
}

// ink sums the alpha of the image
func ink(img *image.RGBA) int {
	sum := 0
	for i := 3; i < len(img.Pix); i += 4 {
		sum += int(img.Pix[i])
	}
	return sum
}

// inkCenter returns the mean x of the ink in the rows y0 to y1
func inkCenter(img *image.RGBA, y0, y1 int) float64 {
	var sum, weight float64
	for y := y0; y < y1; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			a := float64(img.Pix[img.PixOffset(x, y)+3])
			sum += a * float64(x)
			weight += a
		}
	}
	if weight == 0 {
		return 0
	}
	return sum / weight
}
