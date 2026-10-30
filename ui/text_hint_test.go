package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"golang.org/x/image/font/sfnt"
)

// sansGlyph returns the font of Noto Sans and the glyph of the character
func sansGlyph(t *testing.T, r rune) (*sfnt.Font, sfnt.GlyphIndex) {
	t.Helper()
	resolveFont(FontFamilySans)
	fontsMu.RLock()
	f, _, ok := lookupFont(FontFamilySans, FontRegular)
	fontsMu.RUnlock()
	if !ok {
		t.Fatal("no Noto Sans")
	}
	var buf sfnt.Buffer
	g, err := f.GlyphIndex(&buf, r)
	if err != nil || g == 0 {
		t.Fatalf("no glyph of %q", r)
	}
	return f, g
}

// rowInk returns the coverage of each row of the mask at its middle columns
func rowInk(m *glyphMask) map[int]int {
	rows := map[int]int{}
	for y := 0; y < m.h; y++ {
		v := 0
		for x := m.w / 3; x < m.w*2/3; x++ {
			v = max(v, int(m.pix[y*m.w+x]))
		}
		rows[m.y+y] = v
	}
	return rows
}

func TestHintSnapsToPixels(t *testing.T) {
	f, g := sansGlyph(t, 'z')
	for _, size := range []float64{11, 12, 13, 14, 15, 16, 18} {
		TextHinting = true
		hinted := rowInk(sfntGlyphMask(f, g, size, 0, 0, 0))
		// The top and the bottom bars of z: whole rows of ink, the rows
		// past them clear; the bottom on the baseline
		full := 0
		for _, v := range hinted {
			if v >= 250 {
				full++
			}
		}
		if hinted[0] > 10 {
			t.Errorf("size %v: ink under the baseline: %d", size, hinted[0])
		}
		if hinted[-1] < 250 {
			t.Errorf("size %v: the bottom bar is not on the baseline: %d", size, hinted[-1])
		}
		// The top of z at the x-height, a whole number of pixels
		top := 0
		for y := -1; y > -int(size); y-- {
			if hinted[y] > 10 {
				top = y
			}
		}
		if hinted[top] < 250 {
			t.Errorf("size %v: the top bar's top row is %d, not whole", size, hinted[top])
		}
		if full < 2 {
			t.Errorf("size %v: %d whole rows", size, full)
		}
	}
	// Without hinting, the rows at the edges are partly covered
	TextHinting = false
	defer func() { TextHinting = true }()
	plain := rowInk(sfntGlyphMask(f, g, 13, 0, 0, 0))
	soft := 0
	for _, v := range plain {
		if v > 20 && v < 235 {
			soft++
		}
	}
	if soft == 0 {
		t.Fatal("the outline not hinted is as sharp: the test proves nothing")
	}
}

func TestHintStems(t *testing.T) {
	// The bar of H: a whole count of rows, at least one
	f, g := sansGlyph(t, 'H')
	for _, size := range []float64{10, 12, 13, 17, 24} {
		m := sfntGlyphMask(f, g, size, 0, 0, 0)
		rows := rowInk(m)
		soft := 0
		for y := -int(size); y < 0; y++ {
			if v := rows[y]; v > 30 && v < 225 {
				soft++
			}
		}
		if soft > 0 {
			t.Errorf("size %v: %d rows of the bar of H partly covered", size, soft)
		}
	}
}

func TestHintKeepsWidths(t *testing.T) {
	// The advances are those of the measuring: the ink is where it was
	text := "The quick brown fox"
	draw := func(hint bool) *image.RGBA {
		TextHinting = hint
		defer func() { TextHinting = true }()
		cnv, img := newWhiteCanvas(200, 30, 1)
		cnv.SetFontSize(14)
		cnv.DrawText(0, 0, 200, 30, text)
		return img
	}
	_, cx0 := inkOf(draw(false), image.Rect(0, 0, 200, 30))
	_, cx1 := inkOf(draw(true), image.Rect(0, 0, 200, 30))
	if math.Abs(cx0-cx1) > 0.3 {
		t.Fatalf("the ink moved from %v to %v", cx0, cx1)
	}
}

func TestSubpixelText(t *testing.T) {
	f, g := sansGlyph(t, 'l')
	m := sfntGlyphMask(f, g, 14, 0, 0, 1)
	if !m.lcd || len(m.pix) != m.w*m.h*3 {
		t.Fatalf("not a mask of stripes: %d bytes for %d×%d", len(m.pix), m.w, m.h)
	}
	// The stem of l: the stripes of its edge pixels differ
	colored := false
	for i := 0; i+2 < len(m.pix); i += 3 {
		if int(m.pix[i])-int(m.pix[i+2]) > 40 || int(m.pix[i+2])-int(m.pix[i]) > 40 {
			colored = true
		}
	}
	if !colored {
		t.Fatal("no pixel has stripes of different coverage")
	}

	// On an opaque picture the colors are of the stripes; on a transparent
	// one gray
	cnv, img := newWhiteCanvas(40, 30, 1)
	cnv.lcd = 1
	cnv.SetFontSize(14)
	cnv.DrawText(5, 5, 30, 25, "l")
	tinted := false
	for i := 0; i < len(img.Pix); i += 4 {
		if p := img.Pix[i : i+3]; int(p[0])-int(p[2]) > 30 || int(p[2])-int(p[0]) > 30 {
			tinted = true
		}
	}
	if !tinted {
		t.Fatal("no colored fringe on an opaque picture")
	}
	cnv, rgba := NewImageCanvas(40, 30, 1)
	cnv.lcd = 1
	cnv.SetColor(color.RGBA{0, 0, 0, 255})
	cnv.SetFontSize(14)
	cnv.DrawText(5, 5, 30, 25, "l")
	for i := 0; i < len(rgba.Pix); i += 4 {
		if p := rgba.Pix[i : i+4]; p[3] > 0 && (p[0] != p[1] || p[1] != p[2]) {
			t.Fatalf("a colored pixel %v on a transparent picture", p)
		}
	}
}

func TestTextSubpixelSetting(t *testing.T) {
	defer func() { TextSubpixel = SubpixelAuto }()
	for _, tc := range []struct {
		set    Subpixel
		system int
		want   int
	}{{SubpixelAuto, 2, 2}, {SubpixelAuto, 0, 0}, {SubpixelNone, 1, 0}, {SubpixelRGB, 0, 1}, {SubpixelBGR, 1, 2}} {
		TextSubpixel = tc.set
		if got := lcdOf(tc.system); got != tc.want {
			t.Errorf("%v with the system %d: %d, want %d", tc.set, tc.system, got, tc.want)
		}
	}
}
