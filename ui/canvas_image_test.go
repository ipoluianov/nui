package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

func TestStrokePathPaint(t *testing.T) {
	blue := color.RGBA{0, 0, 255, 255}
	grad := LinearGradient{X1: 10, X2: 90, Stops: []GradientStop{{0, red}, {1, blue}}}
	line := pathOf(func(p *Path) { p.MoveTo(10, 10); p.LineTo(90, 10) })

	cnv, img := newScaledTestCanvas(100, 20, 1)
	cnv.StrokePathPaint(line, StrokeStyle{Width: 4}, grad)
	if c := at(img, 11, 10); c.R < 240 || c.B > 15 {
		t.Fatalf("start %v", c)
	}
	if c := at(img, 88, 10); c.B < 240 || c.R > 15 {
		t.Fatalf("end %v", c)
	}
	if at(img, 50, 5) != black {
		t.Fatal("drawn off the line")
	}

	// Under a transform the paint goes with the drawing
	cnv, img = newScaledTestCanvas(100, 40, 2)
	cnv.Translate(0, 10)
	cnv.ScaleBy(1, 2)
	cnv.StrokePathPaint(line, StrokeStyle{Width: 2}, grad)
	cnv.StrokePathPaint(line, StrokeStyle{Width: 2, NonScaling: true}, grad)
	if c := at(img, 22, 60); c.R < 230 {
		t.Fatalf("transformed start %v", c)
	}
	if c := at(img, 176, 60); c.B < 230 {
		t.Fatalf("transformed end %v", c)
	}

	cnv, _ = newScaledTestCanvas(100, 20, 1)
	rec := NewSVGRecorder(100, 20)
	cnv.SetRecorder(rec)
	cnv.StrokePathPaint(line, StrokeStyle{Width: 4}, grad)
	if svg := rec.String(); !strings.Contains(svg, `<linearGradient`) || !strings.Contains(svg, `stroke="url(#g`) {
		t.Fatalf("%s", svg)
	}
}

func TestImageOpacity(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	half := color.RGBA{128, 128, 128, 255}
	for _, draw := range []struct {
		name  string
		scale float64
		f     func(cnv *Canvas)
	}{
		{"image", 1, func(cnv *Canvas) { cnv.DrawImage(0, 0, src) }},
		{"scaled", 2, func(cnv *Canvas) { cnv.DrawImage(0, 0, src) }},
		{"rect", 1, func(cnv *Canvas) { cnv.DrawImageRect(0, 0, 20, 20, src, src.Rect, true) }},
		{"transformed", 1, func(cnv *Canvas) { cnv.DrawImageTransformed(src, ScalingMatrix(2, 2), true) }},
	} {
		cnv, img := newScaledTestCanvas(20, 20, draw.scale)
		cnv.Save()
		cnv.SetImageOpacity(0.5)
		draw.f(cnv)
		cnv.Restore()
		if c := at(img, 4, 4); !nearRGBA(c, half, 2) {
			t.Errorf("%s: %v", draw.name, c)
		}
		if cnv.ImageOpacity() != 1 {
			t.Errorf("%s: the opacity not restored", draw.name)
		}
	}
	// Text under a transform is not an image
	cnv, img := newScaledTestCanvas(60, 30, 1)
	cnv.SetImageOpacity(0.2)
	cnv.Rotate(0.001)
	cnv.SetColor(white)
	cnv.SetFontSize(24)
	cnv.DrawText(0, 0, 60, 30, "██")
	if c := at(img, 8, 15); c.R < 240 {
		t.Fatalf("text faded %v", c)
	}
}

// checker returns an image of black and white squares of the cell
func checker(n, cell int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			v := uint8(0)
			if (x/cell+y/cell)%2 == 0 {
				v = 255
			}
			img.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return img
}

// spread returns how far the pixels of the image are from a gray of 128
func spread(img *image.RGBA, r image.Rectangle) int {
	worst := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			worst = max(worst, int(math.Abs(float64(img.RGBAAt(x, y).R)-128)))
		}
	}
	return worst
}

func TestMipImage(t *testing.T) {
	src := checker(512, 4)
	mip := NewMipImage(src)
	inner := image.Rect(2, 2, 35, 35)

	cnv, img := newScaledTestCanvas(37, 37, 1)
	cnv.DrawImageRect(0, 0, 37, 37, src, src.Rect, true)
	plain := spread(img, inner)

	cnv, img = newScaledTestCanvas(37, 37, 1)
	cnv.DrawImageRect(0, 0, 37, 37, mip, mip.Bounds(), true)
	if s := spread(img, inner); s > 4 || plain < 64 {
		t.Fatalf("the mipmapped picture %d from gray, the plain one %d", s, plain)
	}

	cnv, img = newScaledTestCanvas(32, 32, 1)
	cnv.Rotate(0.3)
	cnv.DrawImageTransformed(mip, ScalingMatrix(0.1, 0.1), true)
	if s := spread(img, image.Rect(12, 12, 20, 20)); s > 6 {
		t.Fatalf("transformed %d from gray", s)
	}

	// At its size, as the image itself
	a, ia := newScaledTestCanvas(40, 40, 1)
	b, ib := newScaledTestCanvas(40, 40, 1)
	small := checker(40, 1)
	a.DrawImageRect(0, 0, 40, 40, small, small.Rect, true)
	b.DrawImageRect(0, 0, 40, 40, NewMipImage(small), small.Rect, true)
	a.DrawImage(0, 0, small)
	b.DrawImage(0, 0, NewMipImage(small))
	if string(ia.Pix) != string(ib.Pix) {
		t.Fatal("a MipImage at its size drawn otherwise")
	}

	// Changed and invalidated: the copies made again
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	mip.Invalidate()
	cnv, img = newScaledTestCanvas(32, 32, 1)
	cnv.DrawImageRect(0, 0, 32, 32, mip, mip.Bounds(), true)
	if c := at(img, 16, 16); c != white {
		t.Fatalf("after Invalidate %v", c)
	}
}

func BenchmarkMipImageSmall(b *testing.B) {
	src := checker(2048, 4)
	mip := NewMipImage(src)
	cnv, _ := newScaledTestCanvas(200, 200, 1)
	for b.Loop() {
		cnv.DrawImageRect(0, 0, 200, 200, mip, mip.Bounds(), true)
	}
}
