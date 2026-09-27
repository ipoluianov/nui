package ui

import (
	"image/color"
	"sync"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// "AVAWAT Ta" is kerned in Noto Sans
func TestCharPositionsMatchMeasureText(t *testing.T) {
	for _, family := range []string{FontFamilySans, FontFamilyMono} {
		for _, text := range []string{"", "a", "Hello, world", "Привет, мир", "AVAWAT Ta"} {
			positions, err := GetCharPositions(family, 14, text)
			if err != nil {
				t.Fatal(err)
			}
			width, _, err := MeasureText(family, 14, text)
			if err != nil {
				t.Fatal(err)
			}
			if last := positions[len(positions)-1]; last != width {
				t.Errorf("%s %q: last position %d, width %d", family, text, last, width)
			}
		}
	}
}

func TestUnknownFamilyUsesDefaultFont(t *testing.T) {
	w1, h1, err := MeasureText("no such font", 14, "abc")
	if err != nil {
		t.Fatal(err)
	}
	w2, h2, _ := MeasureText(FontFamilySans, 14, "abc")
	if w1 != w2 || h1 != h2 {
		t.Errorf("unknown family %dx%d, default %dx%d", w1, h1, w2, h2)
	}
}

// A character missing in the requested font is drawn with the fallback
// font: Go Regular as the requested font, JetBrains Mono as the fallback.
func TestFallbackFont(t *testing.T) {
	if err := RegisterFont("test-goregular", goregular.TTF); err != nil {
		t.Fatal(err)
	}

	goFace := newTestFace(t, goregular.TTF)
	monoFace := newTestFace(t, fontJetBrainsMono)
	var missing rune = -1
	for r := rune(0x2000); r < 0x10000; r++ {
		_, inGo := goFace.GlyphAdvance(r)
		_, inMono := monoFace.GlyphAdvance(r)
		if !inGo && inMono {
			missing = r
			break
		}
	}
	if missing < 0 {
		t.Skip("no character that only JetBrains Mono has")
	}

	before, _, _ := MeasureText("test-goregular", 14, string(missing))
	AddFallbackFont(FontFamilyMono)
	t.Cleanup(func() {
		fontsMu.Lock()
		fallbackFonts = nil
		faceCache = make(map[faceKey]*faceEntry)
		fontsMu.Unlock()
	})
	after, _, _ := MeasureText("test-goregular", 14, string(missing))

	want, _ := monoFace.GlyphAdvance(missing)
	if after != want.Ceil() {
		t.Errorf("%U: width %d with the fallback (%d without), JetBrains Mono advance %d", missing, after, before, want.Ceil())
	}
}

// Forms paint on their own goroutines: run with -race
func TestConcurrentTextUse(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				size := float64(12 + (i+j)%4)
				MeasureText(FontFamilyMono, size, "concurrent text")
				GetCharPositions(FontFamilyMono, size, "concurrent")
				renderText("render", color.RGBA{A: 255}, FontFamilyMono, size)
			}
		}(i)
	}
	wg.Wait()
}

func newTestFace(t *testing.T, data []byte) font.Face {
	f, err := opentype.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 14, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		t.Fatal(err)
	}
	return face
}

func BenchmarkMeasureText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MeasureText(FontFamilyMono, 14, "Button text")
	}
}

// The previous implementation, for comparison: the font file was parsed
// and a face created on every call
func BenchmarkMeasureTextParseEachTime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f, _ := opentype.Parse(fontJetBrainsMono)
		face, _ := opentype.NewFace(f, &opentype.FaceOptions{Size: 14, DPI: 72, Hinting: font.HintingFull})
		font.MeasureString(face, "Button text")
		face.Close()
	}
}
