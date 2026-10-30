package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestClipBounds(t *testing.T) {
	cnv, _ := newScaledTestCanvas(200, 100, 2)
	if b := cnv.ClipBounds(); b != (RectF{0, 0, 200, 100}) {
		t.Fatalf("%v", b)
	}
	cnv.Translate(100, 50)
	cnv.ScaleBy(10, 10)
	b := cnv.ClipBounds()
	if math.Abs(b.MinX+10) > 1e-9 || math.Abs(b.MaxX-10) > 1e-9 || math.Abs(b.MinY+5) > 1e-9 || math.Abs(b.MaxY-5) > 1e-9 {
		t.Fatalf("transformed %v", b)
	}
	cnv.ResetTransform()
	cnv.TranslateAndClip(50, 20, 10, 10)
	if b := cnv.ClipBounds(); b != (RectF{0, 0, 10, 10}) {
		t.Fatalf("translated %v", b)
	}
}

func TestPaintCache(t *testing.T) {
	cache := NewPaintCache()
	calls, painted := 0, RectF{}
	value := red
	paint := func(cnv *Canvas) {
		calls++
		painted = cnv.ClipBounds()
		cnv.FillRect(0, 0, 100, 60, value)
		cnv.FillPath(pathOf(func(p *Path) { p.Circle(50, 30, 20) }), color.RGBA{0, 0, 255, 128})
	}
	draw := func() *image.RGBA {
		cnv, img := newScaledTestCanvas(120, 80, 1.5)
		cache.Draw(cnv, 10, 10, 100, 60, paint)
		return img
	}
	direct := func() *image.RGBA {
		cnv, img := newScaledTestCanvas(120, 80, 1.5)
		cnv.Save()
		cnv.TranslateAndClip(10, 10, 100, 60)
		paint(cnv)
		cnv.Restore()
		return img
	}
	first := draw()
	if calls != 1 {
		t.Fatalf("%d paints", calls)
	}
	again := draw()
	if calls != 1 || string(first.Pix) != string(again.Pix) {
		t.Fatalf("painted again: %d paints", calls)
	}
	want := direct()
	for i := range want.Pix {
		if d := int(want.Pix[i]) - int(first.Pix[i]); d > 1 || d < -1 {
			t.Fatalf("pixel %d: %d, painted directly %d", i/4, first.Pix[i], want.Pix[i])
		}
	}

	// A part invalidated: only it is painted, and the picture is as new
	value = white
	cache.InvalidateRect(RectF{MinX: 10, MinY: 10, MaxX: 30, MaxY: 20})
	cache.InvalidateRect(RectF{MinX: 25, MinY: 15, MaxX: 40, MaxY: 25}) // joined
	calls = 0
	part := draw()
	if calls != 1 || !painted.ContainsRect(RectF{10, 10, 40, 25}) || !(RectF{9, 9, 41, 26}).ContainsRect(painted) {
		t.Fatalf("%d paints of %v", calls, painted)
	}
	if c := at(part, int(1.5*(10+20)), int(1.5*(10+12))); c != white {
		t.Fatalf("the invalid part %v", c)
	}
	if c := at(part, int(1.5*(10+80)), int(1.5*(10+50))); c != red {
		t.Fatalf("the valid part %v", c)
	}

	// Another size: painted all again
	calls = 0
	cnv, _ := newScaledTestCanvas(120, 80, 1.5)
	cache.Draw(cnv, 0, 0, 90, 60, paint)
	if calls != 1 || painted != (RectF{0, 0, 90, 60}) {
		t.Fatalf("resized: %d paints of %v", calls, painted)
	}
	cache.Invalidate()
	if cache.Valid() {
		t.Fatal("valid after Invalidate")
	}
}

func BenchmarkPaintCacheKept(b *testing.B) {
	cache := NewPaintCache()
	cnv, _ := newScaledTestCanvas(800, 600, 1)
	paint := func(cnv *Canvas) { cnv.FillRect(0, 0, 800, 600, red) }
	for b.Loop() {
		cache.Draw(cnv, 0, 0, 800, 600, paint)
	}
}
