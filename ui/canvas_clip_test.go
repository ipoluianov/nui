package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// pathOf returns a new path f made
func pathOf(f func(p *Path)) *Path {
	p := NewPath()
	f(p)
	return p
}

// at returns the color of the pixel
func at(img *image.RGBA, x, y int) color.RGBA {
	return img.RGBAAt(x, y)
}

func TestClipPathCircle(t *testing.T) {
	cnv, img := newScaledTestCanvas(100, 100, 1)
	cnv.Save()
	cnv.ClipPath(pathOf(func(p *Path) { p.Circle(50, 50, 30) }))
	cnv.FillRect(0, 0, 100, 100, red)
	cnv.Restore()
	if at(img, 50, 50) != red || at(img, 50, 78) != red {
		t.Fatalf("inside %v %v", at(img, 50, 50), at(img, 50, 78))
	}
	if at(img, 5, 5) != black || at(img, 50, 81) != black || at(img, 75, 75) != black {
		t.Fatalf("outside %v %v %v", at(img, 5, 5), at(img, 50, 81), at(img, 75, 75))
	}
	// The edge antialiased: a pixel the circle crosses is between
	if c := at(img, 50+21, 50+21); c.R == 0 || c.R == 255 {
		t.Fatalf("edge %v", c)
	}
	// After Restore nothing is clipped
	cnv.FillRect(0, 0, 10, 10, white)
	if at(img, 5, 5) != white {
		t.Fatal("clip left after Restore")
	}
}

func TestClipPathNested(t *testing.T) {
	cnv, img := newScaledTestCanvas(100, 60, 1)
	cnv.Save()
	cnv.ClipPath(pathOf(func(p *Path) { p.Rect(10, 10, 50, 40) }))
	cnv.Save()
	cnv.ClipPath(pathOf(func(p *Path) { p.Rect(40, 0, 50, 60) }))
	cnv.FillRect(0, 0, 100, 60, red)
	cnv.Restore()
	// Back in the first clip only
	cnv.FillRect(0, 0, 20, 60, white)
	cnv.Restore()
	if at(img, 45, 30) != red || at(img, 30, 30) != black || at(img, 70, 30) != black || at(img, 45, 5) != black {
		t.Fatalf("intersection %v %v %v %v", at(img, 45, 30), at(img, 30, 30), at(img, 70, 30), at(img, 45, 5))
	}
	if at(img, 15, 30) != white || at(img, 5, 30) != black || at(img, 15, 5) != black {
		t.Fatalf("first clip %v %v %v", at(img, 15, 30), at(img, 5, 30), at(img, 15, 5))
	}
}

func TestClipPathAllDrawings(t *testing.T) {
	tile := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := range tile.Pix {
		tile.Pix[i] = 255
	}
	for _, draw := range []struct {
		name string
		f    func(cnv *Canvas)
	}{
		{"rect", func(cnv *Canvas) { cnv.FillRect(0, 0, 80, 80, white) }},
		{"image", func(cnv *Canvas) { cnv.DrawImageRect(0, 0, 80, 80, tile, tile.Rect, false) }},
		{"text", func(cnv *Canvas) {
			cnv.SetColor(white)
			cnv.SetFontSize(40)
			cnv.DrawText(0, 0, 80, 80, "███")
		}},
		{"path", func(cnv *Canvas) {
			cnv.StrokePath(pathOf(func(p *Path) { p.Rect(0, 0, 80, 80) }), StrokeStyle{Width: 100}, white)
		}},
		{"clear", func(cnv *Canvas) { cnv.Clear(white) }},
		{"blur", func(cnv *Canvas) {
			cnv.FillRect(0, 0, 80, 40, white)
			cnv.Blur(0, 0, 80, 80, 6)
		}},
		{"child", func(cnv *Canvas) {
			// A widget inside sets its own clip: still within the shape
			cnv.SetDirectTranslateAndClip(0, 0, 80, 80)
			cnv.FillRect(0, 0, 80, 80, white)
		}},
	} {
		cnv, img := newScaledTestCanvas(80, 80, 1.5)
		cnv.Save()
		cnv.ClipPath(pathOf(func(p *Path) { p.Rect(20, 20, 20, 20) }))
		draw.f(cnv)
		cnv.Restore()
		if lit := litOutside(img, image.Rect(30, 30, 60, 60)); len(lit) > 0 {
			t.Errorf("%s: %d pixels outside the clip, first %v", draw.name, len(lit), lit[0])
		}
		if at(img, 45, 32).R == 0 {
			t.Errorf("%s: nothing inside", draw.name)
		}
	}
}

func TestClipPathTransformed(t *testing.T) {
	cnv, img := newScaledTestCanvas(100, 100, 2)
	cnv.Save()
	cnv.Translate(50, 50)
	cnv.ScaleBy(2, 2)
	cnv.ClipPath(pathOf(func(p *Path) { p.Rect(-10, -10, 20, 20) })) // 30..70 in the canvas
	cnv.ResetTransform()
	cnv.FillRect(0, 0, 100, 100, red)
	cnv.Restore()
	if at(img, 62, 62) != red || at(img, 138, 138) != red || at(img, 58, 100) != black || at(img, 142, 100) != black {
		t.Fatalf("%v %v %v %v", at(img, 62, 62), at(img, 138, 138), at(img, 58, 100), at(img, 142, 100))
	}
}

func TestClipPathEvenOddAndEmpty(t *testing.T) {
	cnv, img := newScaledTestCanvas(100, 100, 1)
	ring := pathOf(func(p *Path) { p.Circle(50, 50, 40); p.Circle(50, 50, 20) })
	ring.SetFillRule(FillRuleEvenOdd)
	cnv.Save()
	cnv.ClipPath(ring)
	cnv.FillRect(0, 0, 100, 100, red)
	cnv.Restore()
	if at(img, 50, 50) != black || at(img, 50, 20) != red {
		t.Fatalf("ring %v %v", at(img, 50, 50), at(img, 50, 20))
	}

	cnv, img = newScaledTestCanvas(20, 20, 1)
	cnv.Save()
	cnv.ClipPath(NewPath())
	cnv.SetDirectTranslateAndClip(0, 0, 20, 20)
	cnv.FillRect(0, 0, 20, 20, red)
	cnv.Restore()
	if lit := litOutside(img, image.Rectangle{}); len(lit) > 0 {
		t.Fatalf("an empty clip drew %d pixels", len(lit))
	}
}

func TestClipPathSVG(t *testing.T) {
	cnv, _ := newScaledTestCanvas(100, 100, 1)
	rec := NewSVGRecorder(100, 100)
	cnv.SetRecorder(rec)
	cnv.Save()
	cnv.ClipPath(pathOf(func(p *Path) { p.Circle(50, 50, 30) }))
	cnv.FillPath(pathOf(func(p *Path) { p.Rect(0, 0, 100, 100) }), red)
	cnv.Restore()
	cnv.FillPath(pathOf(func(p *Path) { p.Rect(0, 0, 10, 10) }), red)
	svg := rec.String()
	if !strings.Contains(svg, `<clipPath id="s1"><path d="M`) || strings.Count(svg, "<g clip-path") != 2 {
		t.Fatalf("%s", svg)
	}
	if !strings.Contains(svg, `clip-path="url(#s1)"><rect`) {
		t.Fatalf("the rectangle not clipped by the shape: %s", svg)
	}
}

func TestClipPathAllocs(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector allocates")
	}
	cnv, _ := newScaledTestCanvas(200, 200, 1)
	p := pathOf(func(p *Path) { p.RoundedRect(10, 10, 180, 180, 16) })
	f := func() {
		cnv.Save()
		cnv.ClipPath(p)
		cnv.FillRect(0, 0, 200, 200, red)
		cnv.Restore()
	}
	f()
	if n := testing.AllocsPerRun(50, f); n > 2 {
		t.Fatalf("%v allocations", n)
	}
}

func BenchmarkClipPath(b *testing.B) {
	cnv, _ := newScaledTestCanvas(300, 200, 1)
	p := pathOf(func(p *Path) { p.RoundedRect(0, 0, 300, 200, 16) })
	for b.Loop() {
		cnv.Save()
		cnv.ClipPath(p)
		cnv.Restore()
	}
}
