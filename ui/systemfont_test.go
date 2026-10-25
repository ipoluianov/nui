package ui

import (
	"errors"
	"image"
	"image/color"
	"runtime"
	"testing"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/ipoluianov/nui/internal/platforms"
)

func TestSystemFont(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("checked on Linux, where FreeType draws the system fonts")
	}
	if err := RegisterSystemFont("nosuch", "No Such Font Family 123"); !errors.Is(err, ErrSystemFontNotFound) {
		t.Fatalf("unknown font: %v", err)
	}
	if err := RegisterSystemFont("testsys", "DejaVu Sans"); err != nil {
		t.Skip("DejaVu Sans: ", err)
	}
	// Measured and drawn by the system font, not the default one
	sysW, sysH, _ := MeasureText("testsys", 14, "Hello world")
	defW, _, _ := MeasureText(FontFamilySans, 14, "Hello world")
	if sysW == defW || sysW == 0 || sysH == 0 {
		t.Errorf("system font width %d (default %d), height %d", sysW, defW, sysH)
	}
	pos, _ := GetCharPositions("testsys", 14, "Hello")
	if len(pos) != 6 || pos[5] == 0 || pos[1] >= pos[2] {
		t.Errorf("positions %v", pos)
	}
	// Other sizes open on demand
	big, _, _ := MeasureText("testsys", 28, "Hello world")
	if big < sysW*3/2 {
		t.Errorf("28 px width %d, 14 px %d", big, sysW)
	}

	img := image.NewRGBA(image.Rect(0, 0, 120, 30))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	DrawText(img, "Hello", color.RGBA{0, 0, 0, 255}, "testsys", 14, 2, 2, 0, 0, 60, 30)
	dark, colored, outside := 0, 0, 0
	for y := 0; y < 30; y++ {
		for x := 0; x < 120; x++ {
			p := img.RGBAAt(x, y)
			if p.R < 200 || p.G < 200 || p.B < 200 {
				if x >= 60 {
					outside++
				}
				dark++
				if p.R != p.G || p.G != p.B {
					colored++
				}
			}
		}
	}
	if dark < 50 || outside != 0 {
		t.Errorf("drawn: %d dark pixels, %d outside the clip", dark, outside)
	}
	t.Logf("subpixel-colored pixels: %d of %d", colored, dark)
}

// The bold and italic faces of a system font: DejaVu Sans has its own
func TestSystemFontStyles(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("checked on Linux, where FreeType draws the system fonts")
	}
	if err := RegisterSystemFont("testsysstyle", "DejaVu Sans"); err != nil {
		t.Skip("DejaVu Sans: ", err)
	}
	regular, _, _ := MeasureText("testsysstyle", 14, "Hello world")
	bold, _, _ := MeasureText("testsysstyle:bold", 14, "Hello world")
	italic, _, _ := MeasureText("testsysstyle:italic", 14, "Hello world")
	if bold <= regular || italic == 0 {
		t.Errorf("widths: regular %d, bold %d, italic %d", regular, bold, italic)
	}
	draw := func(family string) []byte {
		img := image.NewRGBA(image.Rect(0, 0, 100, 24))
		for i := range img.Pix {
			img.Pix[i] = 255
		}
		DrawText(img, "Hello", color.RGBA{0, 0, 0, 255}, family, 14, 0, 0, 0, 0, 100, 24)
		return img.Pix
	}
	if string(draw("testsysstyle")) == string(draw("testsysstyle:italic")) {
		t.Error("italic is drawn as regular")
	}
}

// A family nui doesn't know is looked for among the system fonts; a font
// file the application registers for it later takes its place
func TestUnknownFamilyFromSystem(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("checked on Linux")
	}
	name := "DejaVu Sans Mono"
	if _, err := platforms.OpenSystemFont(name, 14, false, false); err != nil {
		t.Skip(name, ": ", err)
	}
	sys, _, _ := MeasureText(name, 14, "Hello world")
	def, _, _ := MeasureText(FontFamilySans, 14, "Hello world")
	if !isSystemFamily("dejavu sans mono") || sys == def {
		t.Fatalf("not the system font: width %d, default %d", sys, def)
	}
	if err := RegisterFont(name, goregular.TTF); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		fontsMu.Lock()
		delete(fonts, "dejavu sans mono")
		faceCache = make(map[faceKey]*faceEntry)
		fontsMu.Unlock()
	})
	if isSystemFamily("dejavu sans mono") {
		t.Error("the registered file doesn't replace the system font")
	}
}
