package ui

import (
	"image"
	"image/color"
	"testing"
)

// newTestCanvas is a black w x h canvas clipped to itself
func newTestCanvas(w, h int) (*Canvas, *image.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	cnv := NewCanvas(img)
	cnv.SetDirectTranslateAndClip(0, 0, w, h)
	return cnv, img
}

var white = color.RGBA{255, 255, 255, 255}

// The line with integer coordinates covers whole pixels only
func TestDrawLineFInteger(t *testing.T) {
	cnv, img := newTestCanvas(20, 10)
	cnv.DrawLineF(2, 5, 12, 5, white)
	for x := 0; x < 20; x++ {
		for y := 0; y < 10; y++ {
			want := uint8(0)
			if y == 5 && x >= 2 && x <= 12 {
				want = 254 // the blending loses a unit
			}
			if got := img.RGBAAt(x, y).R; got < want || (want == 0 && got != 0) {
				t.Fatalf("pixel %d,%d = %d, want %d", x, y, got, want)
			}
		}
	}
}

// The joint of two segments is as bright as the rest of the line
func TestDrawLineFJoint(t *testing.T) {
	cnv, img := newTestCanvas(20, 20)
	cnv.DrawLineF(2, 2, 8, 8, white)
	cnv.DrawLineF(8, 8, 14, 2, white)
	if got := img.RGBAAt(8, 8).R; got < 254 {
		t.Fatalf("joint = %d, want full", got)
	}
}

// A vertical line half a pixel off is split between the two columns
func TestDrawLineFHalfPixel(t *testing.T) {
	cnv, img := newTestCanvas(10, 10)
	cnv.DrawLineF(4.5, 2, 4.5, 7, white)
	a, b := int(img.RGBAAt(4, 4).R), int(img.RGBAAt(5, 4).R)
	if a < 120 || a > 135 || b < 120 || b > 135 {
		t.Fatalf("columns = %d, %d, want about half each", a, b)
	}
}

// A line far outside or crossing the canvas draws only inside it
func TestDrawLineFClip(t *testing.T) {
	cnv, img := newTestCanvas(10, 10)
	cnv.DrawLineF(-1e6, 5, 1e6, 5, white)
	cnv.DrawLineF(-50, -50, -40, -40, white)
	if got := img.RGBAAt(0, 5).R; got < 254 {
		t.Fatalf("left edge = %d, want full", got)
	}
	if got := img.RGBAAt(9, 5).R; got < 254 {
		t.Fatalf("right edge = %d, want full", got)
	}
}

// The triangle is filled inside, left alone outside and cut by the clip
func TestFillTriangle(t *testing.T) {
	cnv, img := newTestCanvas(40, 30)
	cnv.TranslateAndClip(5, 5, 20, 20) // the triangle goes past the clip on the right
	cnv.FillTriangle(0, 0, 30, 0, 0, 18, white)
	cases := []struct {
		x, y int
		want uint8
	}{
		{7, 7, 255},  // inside
		{24, 6, 255}, // inside, at the edge of the clip
		{26, 6, 0},   // inside the triangle, outside the clip
		{20, 18, 0},  // below the slanted edge
		{3, 3, 0},    // outside the translation
	}
	for _, tc := range cases {
		if got := img.RGBAAt(tc.x, tc.y).R; got != tc.want {
			t.Errorf("pixel %d,%d = %d, want %d", tc.x, tc.y, got, tc.want)
		}
	}
}

// On a HiDPI canvas the logical triangle takes the scaled pixels
func TestFillTriangleScaled(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 40))
	cnv := NewCanvasScaled(img, 2)
	cnv.SetDirectTranslateAndClip(0, 0, 20, 20)
	cnv.FillTriangle(0, 0, 10, 0, 0, 10, white)
	if img.RGBAAt(3, 3).A != 255 || img.RGBAAt(14, 3).A != 255 || img.RGBAAt(17, 17).A != 0 {
		t.Fatalf("scaled: %v %v %v", img.RGBAAt(3, 3), img.RGBAAt(14, 3), img.RGBAAt(17, 17))
	}
}
