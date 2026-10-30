package ui

import (
	"image"
	"image/color"
	"slices"
	"testing"
)

// The primitives every widget draws on every frame: a button is a frame, a
// combo box an arrow, a check box a mark

func BenchmarkFillFrame(b *testing.B) {
	cnv, _ := newTestCanvas(200, 60)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cnv.FillFrame(10, 10, 120, 30, 6, color.RGBA{240, 240, 240, 255}, color.RGBA{120, 120, 120, 255})
	}
}

func BenchmarkFillRoundedRect(b *testing.B) {
	cnv, _ := newTestCanvas(200, 60)
	cnv.SetColor(color.RGBA{40, 120, 220, 255})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cnv.FillRoundedRect(10, 10, 120, 30, 6)
	}
}

func BenchmarkDrawRoundedRect(b *testing.B) {
	cnv, _ := newTestCanvas(200, 60)
	cnv.SetColor(color.RGBA{40, 120, 220, 255})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cnv.DrawRoundedRect(10, 10, 120, 30, 6)
	}
}

func BenchmarkFillTriangle(b *testing.B) {
	cnv, _ := newTestCanvas(40, 40)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cnv.FillTriangle(10, 15, 22, 15, 16, 22, color.RGBA{40, 40, 40, 255})
	}
}

func BenchmarkDrawLineWide(b *testing.B) {
	cnv, _ := newTestCanvas(100, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cnv.DrawLine(10, 10, 80, 60, 2, color.RGBA{40, 40, 40, 255})
	}
}

func BenchmarkFillRectTranslucent(b *testing.B) {
	cnv, _ := newTestCanvas(300, 200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cnv.FillRect(0, 0, 300, 200, color.RGBA{40, 120, 220, 100})
	}
}

func BenchmarkFillPathGradient(b *testing.B) {
	cnv, _ := newTestCanvas(300, 200)
	p := NewPath()
	p.RoundedRect(10, 10, 280, 180, 12)
	g := LinearGradient{X1: 0, Y1: 10, X2: 0, Y2: 190, Stops: []GradientStop{
		{0, color.RGBA{37, 99, 235, 255}}, {1, color.RGBA{37, 99, 235, 0}}}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cnv.FillPathPaint(p, g)
	}
}

// The rounded rectangles drawn in parts are the ones rasterized whole
func TestFillRoundedPartsMatchWhole(t *testing.T) {
	for _, thickness := range []int{0, 1, 2, 3} {
		for _, radius := range []float64{0, 1, 2.5, 4, 6, 9, 15, 40} {
			for _, size := range [][2]int{{120, 30}, {20, 9}, {9, 40}, {60, 60}, {5, 5}} {
				for _, alpha := range []uint8{255, 120} {
					col := color.RGBA{40, 120, 220, alpha}
					got, img1 := newTestCanvas(140, 80)
					want, img2 := newTestCanvas(140, 80)
					clip := image.Rect(0, 3, 100, 70)
					got.fillRoundedPixels(7, 5, size[0], size[1], radius, thickness, col, clip)
					fillRoundedWhole(want, 7, 5, size[0], size[1], radius, thickness, col, clip)
					for i := range img1.Pix {
						d := int(img1.Pix[i]) - int(img2.Pix[i])
						if d < -1 || d > 1 {
							x, y := (i/4)%140, i/4/140
							t.Fatalf("t%d r%v %v a%d: pixel %d,%d = %v, want %v", thickness, radius, size, alpha,
								x, y, img1.RGBAAt(x, y), img2.RGBAAt(x, y))
						}
					}
				}
			}
		}
	}
}

// fillRoundedWhole is fillRoundedPixels rasterizing the whole shape at once
func fillRoundedWhole(c *Canvas, x, y, w, h int, radius float64, thickness int, col color.Color, clip image.Rectangle) {
	b := image.Rect(x, y, x+w, y+h).Intersect(clip).Intersect(c.clipRect())
	ras := &rasterizer{}
	z := ras.reset(b.Dx(), b.Dy())
	t := float64(thickness)
	if thickness > 0 && w > thickness*2 && h > thickness*2 {
		addPoints(z, roundedRectPoints(nil, float64(x), float64(y), float64(w), float64(h), radius+t/2), float64(b.Min.X), float64(b.Min.Y))
		inner := roundedRectPoints(nil, float64(x)+t, float64(y)+t, float64(w)-t*2, float64(h)-t*2, radius-t/2)
		slices.Reverse(inner)
		addPoints(z, inner, float64(b.Min.X), float64(b.Min.Y))
	} else {
		addPoints(z, roundedRectPoints(nil, float64(x), float64(y), float64(w), float64(h), radius), float64(b.Min.X), float64(b.Min.Y))
	}
	c.fillRasterized(ras, b, col)
}
