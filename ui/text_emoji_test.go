package ui

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// haveColorEmoji skips the test without a font of color emoji
func haveColorEmoji(t *testing.T) {
	t.Helper()
	ensureFallbackFonts("😀")
	l := NewTextLayout("😀", TextStyle{Size: 20})
	l.layout()
	if len(l.lines) == 0 || len(l.lines[0].runs) == 0 {
		t.Skip("no layout")
	}
	r := l.lines[0].runs[0]
	if colorKindOf(r.out.Face, r.out.Glyphs[0].GlyphID) == colorNone {
		t.Skip("no font of color emoji on this system")
	}
}

// testFont returns the path of a font of go-text's test files, if the
// module is downloaded
func testFont(t *testing.T, name string) string {
	t.Helper()
	cache := os.Getenv("GOMODCACHE")
	if cache == "" {
		home, _ := os.UserHomeDir()
		cache = filepath.Join(home, "go", "pkg", "mod")
	}
	paths, _ := filepath.Glob(filepath.Join(cache, "github.com", "go-text", "typesetting-utils@*", "opentype", "color", name))
	if len(paths) == 0 {
		t.Skipf("no %s (go-text/typesetting-utils is not downloaded)", name)
	}
	return paths[len(paths)-1]
}

// colorful counts the pixels of the picture whose components differ
func colorful(img *image.RGBA) int {
	n := 0
	for i := 0; i < len(img.Pix); i += 4 {
		p := img.Pix[i : i+3]
		if max(p[0], p[1], p[2])-min(p[0], p[1], p[2]) > 60 {
			n++
		}
	}
	return n
}

func TestEmojiInColor(t *testing.T) {
	haveColorEmoji(t)
	if !needsShaping("😀") || !needsShaping("a️") || needsShaping("★✓") {
		t.Fatal("which text is an emoji")
	}
	for _, scale := range []float64{1, 2} {
		img := drawLine("😀 🎉", 20, scale)
		if n := colorful(img); n < int(40*scale*scale) {
			t.Errorf("scale %v: %d colored pixels", scale, n)
		}
	}
	// A sequence is one picture: a family, a flag, a skin tone
	for _, seq := range []string{"👨‍👩‍👧", "🇩🇪", "👍🏽"} {
		l := NewTextLayout(seq, TextStyle{Size: 20})
		l.layout()
		if n := len(l.lines[0].clusters); n != 1 {
			t.Errorf("%q in %d clusters", seq, n)
		}
		if b := graphemeBounds(seq); len(b) != 2 || b[1] != utf8.RuneCountInString(seq) {
			t.Errorf("%q: graphemes %v", seq, b)
		}
	}
	// The space after an emoji is of the text's font, not as wide as one
	one, _, _ := MeasureText(FontFamilySans, 20, "😀")
	two, _, _ := MeasureText(FontFamilySans, 20, "😀 😀")
	space, _, _ := MeasureText(FontFamilySans, 20, "a a")
	a, _, _ := MeasureText(FontFamilySans, 20, "aa")
	if gap := two - 2*one; abs(gap-(space-a)) > 2 {
		t.Fatalf("the space between emoji is %d wide, of the text %d", gap, space-a)
	}
	// Translucent text: the emoji fades with it
	cnv, img := newWhiteCanvas(60, 40, 1)
	cnv.SetFontSize(24)
	cnv.SetColor(color.RGBA{0, 0, 0, 60})
	cnv.DrawText(4, 4, 56, 36, "😀")
	darkest := 255
	for i := 0; i < len(img.Pix); i += 4 {
		darkest = min(darkest, int(img.Pix[i]), int(img.Pix[i+1]), int(img.Pix[i+2]))
	}
	if darkest < 150 {
		t.Fatalf("a faded emoji has a pixel of %d", darkest)
	}
}

func TestEmojiTurnedAndLarge(t *testing.T) {
	haveColorEmoji(t)
	for _, draw := range []func(cnv *Canvas){
		func(cnv *Canvas) { cnv.Translate(50, 50); cnv.Rotate(0.5); cnv.DrawTextF(-15, -15, "🦊") },
		func(cnv *Canvas) { cnv.SetFontSize(200); cnv.DrawTextF(0, 0, "🦊") },
	} {
		cnv, img := newWhiteCanvas(100, 100, 1)
		cnv.SetFontSize(24)
		draw(cnv)
		if n := colorful(img); n < 100 {
			t.Errorf("%d colored pixels", n)
		}
	}
}

func TestEmojiCOLR(t *testing.T) {
	for _, name := range []string{"NotoColorEmoji-Regular.ttf", "CoralPixels-Regular.ttf"} {
		data, err := os.ReadFile(testFont(t, name))
		if err != nil {
			t.Fatal(err)
		}
		family := "test-colr-" + strings.ToLower(name[:5])
		if err := RegisterFont(family, data); err != nil {
			t.Fatal(err)
		}
		text := "🦊🌈"
		if name == "CoralPixels-Regular.ttf" {
			text = "AB"
		}
		l := NewTextLayout(text, TextStyle{Size: 40, Font: family})
		l.layout()
		r := l.lines[0].runs[0]
		if k := colorKindOf(r.out.Face, r.out.Glyphs[0].GlyphID); k != colorLayers {
			t.Fatalf("%s: the glyph is of the kind %d, not of layers", name, k)
		}
		for _, turned := range []bool{false, true} {
			cnv, img := newWhiteCanvas(200, 80, 1)
			if turned {
				cnv.Rotate(0.05)
			}
			cnv.DrawTextLayout(l, 10, 10)
			if n := colorful(img); n < 200 {
				t.Errorf("%s turned %v: %d colored pixels", name, turned, n)
			}
		}
		// Recorded as SVG: shapes in several colors
		cnv, _ := newWhiteCanvas(200, 80, 1)
		rec := NewSVGRecorder(200, 80)
		cnv.SetRecorder(rec)
		cnv.DrawTextLayout(l, 10, 10)
		svg := rec.String()
		fills := map[string]bool{}
		for _, part := range strings.Split(svg, `fill="`)[1:] {
			fills[part[:strings.IndexByte(part, '"')]] = true
		}
		if strings.Count(svg, "<path") < 5 || len(fills) < 3 {
			t.Errorf("%s: SVG of %d paths in %d fills", name, strings.Count(svg, "<path"), len(fills))
		}
	}
}

// The picture of a COLR glyph and its shapes drawn on the canvas agree
func TestEmojiCOLRPictureAsShapes(t *testing.T) {
	data, err := os.ReadFile(testFont(t, "NotoColorEmoji-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	RegisterFont("test-colr-match", data)
	l := NewTextLayout("🦊", TextStyle{Size: 48, Font: "test-colr-match"})
	a, ia := newWhiteCanvas(80, 80, 1)
	a.DrawTextLayout(l, 10, 10)
	b, ib := newWhiteCanvas(80, 80, 1)
	b.Rotate(1e-9) // the shapes, not the kept picture
	b.DrawTextLayout(l, 10, 10)
	diff := 0.0
	for i := range ia.Pix {
		diff += math.Abs(float64(ia.Pix[i]) - float64(ib.Pix[i]))
	}
	if mean := diff / float64(len(ia.Pix)); mean > 2 {
		t.Fatalf("the picture and the shapes differ by %v a component", mean)
	}
}

func TestEmojiConcurrent(t *testing.T) {
	haveColorEmoji(t)
	done := make(chan bool)
	for g := 0; g < 4; g++ {
		go func() {
			for i := 0; i < 10; i++ {
				cnv, _ := newWhiteCanvas(120, 30, 1)
				cnv.SetFontSize(10 + float64(i))
				cnv.DrawText(0, 0, 120, 30, "a 😀 🎉 👍🏽 b")
			}
			done <- true
		}()
	}
	for g := 0; g < 4; g++ {
		<-done
	}
}

// A flag is in color even when a font of another script, loaded before,
// has the letters of the regions in black
func TestEmojiOverBlackFonts(t *testing.T) {
	ensureFallbackFonts("שלום नमस्ते ★ ⚑")
	haveColorEmoji(t)
	img := drawLine("🇩🇪 🇫🇷", 24, 1)
	if n := colorful(img); n < 100 {
		t.Fatalf("the flags in %d colored pixels", n)
	}
}
