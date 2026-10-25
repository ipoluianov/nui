package platforms

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func TestDIBRoundTrip(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.NRGBA{255, 0, 0, 255})
	img.Set(2, 0, color.NRGBA{0, 0, 255, 128})
	img.Set(1, 1, color.NRGBA{10, 20, 30, 255})
	got, err := imageFromDIB(dibFromImage(img))
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			if got.At(x, y) != img.At(x, y) {
				t.Errorf("%d,%d: %v, want %v", x, y, got.At(x, y), img.At(x, y))
			}
		}
	}
}

func TestDIB24TopDownNoAlpha(t *testing.T) {
	// 2x2, 24 bits, top-down, rows padded to 8 bytes
	dib := make([]byte, 40+16)
	le := binary.LittleEndian
	le.PutUint32(dib[0:], 40)
	le.PutUint32(dib[4:], 2)
	le.PutUint32(dib[8:], uint32(0xFFFFFFFE)) // -2
	le.PutUint16(dib[12:], 1)
	le.PutUint16(dib[14:], 24)
	copy(dib[40:], []byte{0, 0, 255, 0, 255, 0, 0, 0}) // red, green
	copy(dib[48:], []byte{255, 0, 0, 1, 2, 3, 0, 0})   // blue, (3,2,1)
	img, err := imageFromDIB(dib)
	if err != nil {
		t.Fatal(err)
	}
	want := map[image.Point]color.NRGBA{{0, 0}: {255, 0, 0, 255}, {1, 0}: {0, 255, 0, 255}, {0, 1}: {0, 0, 255, 255}, {1, 1}: {3, 2, 1, 255}}
	for p, c := range want {
		if got := img.At(p.X, p.Y); got != c {
			t.Errorf("%v: %v, want %v", p, got, c)
		}
	}

	// 32 bits with an empty alpha: opaque
	img32, _ := imageFromDIB(dibFromImage(image.NewNRGBA(image.Rect(0, 0, 1, 1))))
	if _, _, _, a := img32.At(0, 0).RGBA(); a != 0xFFFF {
		t.Errorf("alpha %d, want opaque", a)
	}
	if _, err := imageFromDIB(dib[:30]); err == nil {
		t.Error("a short DIB is read")
	}
}
