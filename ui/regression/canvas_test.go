package regression

import (
	"image"
	"image/color"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// Canvas.DrawImage draws a sub-image from its own bounds
func TestDrawImageSubImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 20, 10))
	red, blue := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if x < 10 {
				src.SetRGBA(x, y, red)
			} else {
				src.SetRGBA(x, y, blue)
			}
		}
	}
	sub := src.SubImage(image.Rect(10, 0, 20, 10)) // the blue half
	form := newForm(200, 100)
	box := paintBox(func(cnv *ui.Canvas) { cnv.DrawImage(0, 0, sub) })
	form.Panel().AddWidget(0, 0, box)
	w := uitest.Show(t, form)
	x, y := box.RectClientAreaOnWindow()
	got := w.Screenshot().RGBAAt(x+2, y+2)
	if got != blue {
		t.Errorf("the blue half is drawn as %v", got)
	}
}

// Canvas.DrawText breaks lines at "\n" too
func TestDrawTextNewline(t *testing.T) {
	inkBelow := func(text string) int {
		form := newForm(200, 120)
		box := paintBox(func(cnv *ui.Canvas) {
			cnv.SetColor(color.Black)
			cnv.SetFontSize(14)
			cnv.SetVAlign(ui.VAlignTop)
			cnv.DrawText(0, 0, 100, 60, text)
		})
		form.Panel().AddWidget(0, 0, box)
		w := uitest.Show(t, form)
		x, y := box.RectClientAreaOnWindow()
		img := w.Screenshot()
		form.Close()
		// The ink of the second line: below the first one
		dark := 0
		for py := y + 22; py < y+50; py++ {
			for px := x; px < x+100; px++ {
				if c := img.RGBAAt(px, py); c.R < 128 {
					dark++
				}
			}
		}
		return dark
	}
	crlf, lf := inkBelow("A\r\nB"), inkBelow("A\nB")
	if crlf == 0 {
		t.Fatal("no second line even with \\r\\n: the check is broken")
	}
	if lf == 0 {
		t.Error(`"\n" doesn't break the line`)
	}
}
