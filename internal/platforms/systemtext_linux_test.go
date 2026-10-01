//go:build linux

package platforms

import (
	"image"
	"image/color"
	"os"
	"testing"
	"unsafe"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// The FreeType struct offsets are right when what is read through them
// matches the font: its family name, and metrics and advances that agree
// with Go's own rasterizer of the same file.
func TestSystemFontFreeType(t *testing.T) {
	fc, ok := fcMatch("Noto Sans")
	if !ok || fc.families[0] != "Noto Sans" {
		t.Skip("no Noto Sans in fontconfig")
	}
	sf, err := OpenSystemFont("Noto Sans", 14)
	if err != nil {
		t.Fatal(err)
	}
	f := sf.(*linuxFont)
	name := (*byte)(*(*unsafe.Pointer)(unsafe.Add(f.primary.face, 40)))
	got := ""
	for p := unsafe.Pointer(name); *(*byte)(p) != 0; p = unsafe.Add(p, 1) {
		got += string(rune(*(*byte)(p)))
	}
	if got != "Noto Sans" {
		t.Fatalf("family_name = %q", got)
	}

	data, err := os.ReadFile(fc.file)
	if err != nil {
		t.Skip(err)
	}
	otf, _ := opentype.Parse(data)
	face, _ := opentype.NewFace(otf, &opentype.FaceOptions{Size: 14, DPI: 72, Hinting: font.HintingNone})
	m := face.Metrics()
	ascent, descent := sf.Metrics()
	t.Logf("FreeType ascent %d descent %d; Go ascent %v descent %v", ascent, descent, m.Ascent, m.Descent)
	if ascent != m.Ascent.Ceil() || descent != m.Descent.Ceil() {
		t.Errorf("metrics %d/%d, Go %d/%d", ascent, descent, m.Ascent.Ceil(), m.Descent.Ceil())
	}
	text := "Hello, Привет AVATAR"
	pos := sf.Positions(text)
	goWidth := font.MeasureString(face, text).Round()
	t.Logf("width FreeType %d, Go %d; positions %v", pos[len(pos)-1], goWidth, pos)
	// Hinting rounds the advances to whole pixels, as on the desktop: the
	// width differs a little from the unhinted one
	if d := pos[len(pos)-1] - goWidth; d < -goWidth/20 || d > goWidth/20 {
		t.Errorf("width %d, Go %d", pos[len(pos)-1], goWidth)
	}

	g := f.glyph('A')
	t.Logf("A: %dx%d left %d top %d subpixel %v (rgba %d)", g.w, g.h, g.left, g.top, g.subpixel, fc.rgba)
	if g.h < 8 || g.h > 14 || g.w < 7 || g.top < 8 {
		t.Errorf("glyph A bitmap %dx%d top %d", g.w, g.h, g.top)
	}

	img := image.NewRGBA(image.Rect(0, 0, 200, 30))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	sf.Draw(img, text, color.RGBA{0, 0, 0, 255}, 2, 2, img.Rect)
	dark := 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] < 128 {
			dark++
		}
	}
	if dark < 100 {
		t.Errorf("drawn text has %d dark pixels", dark)
	}
}
