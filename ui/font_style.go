package ui

import (
	"image"
	"math"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Font styles. A font family string may name a style after colons, as in
// fontconfig: "notosans:bold", "jetbrainsmono:bold:italic". Such a string
// goes wherever a family does - Widget.SetFontFamily, Canvas.SetFontFamily,
// MeasureText, DrawText, RegisterFont:
//
//	label.SetFontBold(true)                      // the widget's font, bold
//	cnv.SetFontFamily(ui.StyledFont(ui.FontFamilyMono, ui.FontItalic))
//	w, h, _ := ui.MeasureText("notosans:bold", 14, "Total")
//
// A face the family doesn't have is synthesized from the nearest one it has:
// bold by widening the strokes, italic by slanting the letters.

// FontStyle is a set of style flags of a font
type FontStyle uint8

const (
	FontRegular FontStyle = 0
	FontBold    FontStyle = 1 << 0
	FontItalic  FontStyle = 1 << 1
)

// StyledFont returns the family with the style added to the one it may
// already name, e.g. StyledFont("notosans", FontBold) is "notosans:bold"
func StyledFont(family string, style FontStyle) string {
	base, own := parseFontFamily(family)
	return faceName(base, own|style)
}

// fontWithStyle returns the family with the style turned on or off
func fontWithStyle(family string, style FontStyle, on bool) string {
	base, own := parseFontFamily(family)
	if on {
		own |= style
	} else {
		own &^= style
	}
	return faceName(base, own)
}

// parseFontFamily splits "family:bold:italic" into the family, as written,
// and the style
func parseFontFamily(s string) (family string, style FontStyle) {
	family, rest, found := strings.Cut(s, ":")
	family = strings.TrimSpace(family)
	for found {
		var token string
		token, rest, found = strings.Cut(rest, ":")
		switch strings.ToLower(strings.TrimSpace(token)) {
		case "bold":
			style |= FontBold
		case "italic", "oblique":
			style |= FontItalic
		}
	}
	return family, style
}

// faceName is the name of a face of the family: the family with the style
// after colons, in a fixed order
func faceName(family string, style FontStyle) string {
	if style&FontBold != 0 {
		family += ":bold"
	}
	if style&FontItalic != 0 {
		family += ":italic"
	}
	return family
}

// styleCandidates returns the faces to look for, nearest first, for the
// style: the missing flags are synthesized. A slanted face looks closer to
// a real italic than a widened one to a real bold, so a real bold comes
// before a real italic.
func styleCandidates(style FontStyle) []FontStyle {
	switch style {
	case FontBold | FontItalic:
		return []FontStyle{FontBold | FontItalic, FontBold, FontItalic, FontRegular}
	case FontBold:
		return []FontStyle{FontBold, FontRegular}
	case FontItalic:
		return []FontStyle{FontItalic, FontRegular}
	}
	return []FontStyle{FontRegular}
}

// The slant of a synthesized italic: 12 degrees, as FreeType's
// FT_GlyphSlot_Oblique
var italicShear = math.Tan(12 * math.Pi / 180)

// syntheticFace draws a face with the styles it lacks: bold widens every
// glyph by a few pixels to the right, italic slants it around the baseline.
// The line metrics stay those of the face.
type syntheticFace struct {
	font.Face
	bold   int // the widening in pixels, 0 if not bold
	italic bool
	mask   *image.Alpha
}

func newSyntheticFace(face font.Face, style FontStyle, size float64) font.Face {
	if style == FontRegular {
		return face
	}
	f := &syntheticFace{Face: face, italic: style&FontItalic != 0}
	if style&FontBold != 0 {
		// About the stroke widening of FreeType's FT_GlyphSlot_Embolden
		f.bold = max(1, int(math.Round(size/24)))
	}
	return f
}

func (f *syntheticFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	dr, mask, mp, advance, ok := f.Face.Glyph(dot, r)
	advance += fixed.I(f.bold)
	if !ok || dr.Empty() {
		return dr, mask, mp, advance, ok
	}
	src, isAlpha := mask.(*image.Alpha)
	if !isAlpha {
		return dr, mask, mp, advance, ok
	}
	at := func(x, y int) int { // the source coverage at the pixel of dst
		if x < dr.Min.X || x >= dr.Max.X {
			return 0
		}
		return int(src.Pix[src.PixOffset(mp.X+x-dr.Min.X, mp.Y+y-dr.Min.Y)])
	}

	baseline := dot.Y.Round()
	shift := func(y float64) float64 { // the slant of a row whose center is at y
		if !f.italic {
			return 0
		}
		return (float64(baseline) - y) * italicShear
	}
	minShift := math.Floor(shift(float64(dr.Max.Y) - 0.5))
	maxShift := math.Ceil(shift(float64(dr.Min.Y) + 0.5))
	out := image.Rect(dr.Min.X+int(minShift), dr.Min.Y, dr.Max.X+int(maxShift)+f.bold, dr.Max.Y)
	if f.mask == nil || f.mask.Rect != out {
		f.mask = image.NewAlpha(out)
	} else {
		clear(f.mask.Pix)
	}
	for y := out.Min.Y; y < out.Max.Y; y++ {
		s := shift(float64(y) + 0.5)
		whole := math.Floor(s)
		frac := s - whole
		row := f.mask.Pix[f.mask.PixOffset(out.Min.X, y):]
		for x := out.Min.X; x < out.Max.X; x++ {
			sx := x - int(whole)
			// Slanted by a fraction of a pixel: between two source pixels
			v := 0
			for b := 0; b <= f.bold; b++ {
				c := int(math.Round(float64(at(sx-b, y))*(1-frac) + float64(at(sx-b-1, y))*frac))
				v = max(v, c)
			}
			row[x-out.Min.X] = byte(v)
		}
	}
	return out, f.mask, out.Min, advance, true
}

func (f *syntheticFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	bounds, advance, ok := f.Face.GlyphBounds(r)
	bounds.Max.X += fixed.I(f.bold)
	if f.italic {
		// Y grows down: above the baseline (negative y) leans right
		bounds.Max.X += fixed.Int26_6(float64(-bounds.Min.Y) * italicShear)
		bounds.Min.X -= fixed.Int26_6(float64(bounds.Max.Y) * italicShear)
	}
	return bounds, advance + fixed.I(f.bold), ok
}

func (f *syntheticFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	advance, ok := f.Face.GlyphAdvance(r)
	return advance + fixed.I(f.bold), ok
}
