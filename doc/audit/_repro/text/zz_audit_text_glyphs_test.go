package ui

import (
	"image"
	"runtime/debug"
	"strings"
	"testing"
)

func TestAuditAllGlyphs(t *testing.T) {
	old := TextHinting
	defer func() { TextHinting = old }()
	TextHinting = true
	var b strings.Builder
	for r := rune(0x20); r < 0x3000; r++ {
		if r == 0x85 || r == 0x2029 {
			continue
		}
		b.WriteRune(r)
	}
	all := []rune(b.String())
	for _, fam := range []string{FontFamilySans, FontFamilyMono, FontFamilySans + ":bold", FontFamilyMono + ":italic"} {
		for _, size := range []float64{0.3, 1, 3, 7, 11, 13.5, 24, 61} {
			for i := 0; i < len(all); i += 64 {
				chunk := string(all[i:min(i+64, len(all))])
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Fatalf("%s %v %q: %v\n%s", fam, size, chunk, r, debug.Stack())
						}
					}()
					cnv := NewCanvasScaled(image.NewRGBA(image.Rect(0, 0, 300, 80)), 1)
					cnv.SetFontFamily(fam)
					cnv.SetFontSize(size)
					cnv.DrawText(0, 0, 300, 80, chunk)
					cnv.DrawTextF(0.3, 0.7, chunk)
					p := NewPath()
					p.Text(1, 1, chunk, fam, size)
				}()
			}
		}
	}
}
