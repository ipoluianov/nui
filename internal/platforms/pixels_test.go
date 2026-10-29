//go:build linux || windows

package platforms

import (
	"image"
	"testing"
)

// TestCopyRGBAToBGRA: the conversion gives what converting a pixel at a
// time gives, for even and odd widths, rows not 8-byte aligned in the
// source or in the destination, and frames large enough to go in bands
func TestCopyRGBAToBGRA(t *testing.T) {
	cases := []struct {
		name                    string
		imgWidth, width, height int
		offsetX, dstStride      int
	}{
		{"even", 64, 64, 8, 0, 64},
		{"odd width", 63, 63, 9, 0, 63},
		{"part of the image", 100, 37, 5, 0, 40},
		{"source rows unaligned", 101, 50, 6, 1, 50},
		{"destination rows unaligned", 64, 64, 7, 0, 65},
		{"one pixel", 1, 1, 1, 0, 1},
		{"in bands", 1920, 1920, 1080, 0, 1920},
		{"in bands, odd", 1001, 999, 777, 1, 1003},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			full := image.NewRGBA(image.Rect(0, 0, tc.imgWidth+tc.offsetX, tc.height))
			for i := range full.Pix {
				full.Pix[i] = byte(i*7 + i/13)
			}
			img := full.SubImage(image.Rect(tc.offsetX, 0, tc.imgWidth+tc.offsetX, tc.height)).(*image.RGBA)

			dst := make([]uint32, tc.dstStride*tc.height+1)
			const untouched = 0xDEADBEEF
			for i := range dst {
				dst[i] = untouched
			}
			copyRGBAToBGRA(dst, tc.dstStride, img, tc.width, tc.height)

			for y := 0; y < tc.height; y++ {
				for x := 0; x < tc.dstStride; x++ {
					got := dst[y*tc.dstStride+x]
					if x >= tc.width {
						if got != untouched {
							t.Fatalf("pixel %d,%d past the width written", x, y)
						}
						continue
					}
					p := img.Pix[y*img.Stride+x*4:]
					want := uint32(p[2]) | uint32(p[1])<<8 | uint32(p[0])<<16 | uint32(p[3])<<24
					if got != want {
						t.Fatalf("pixel %d,%d = %08x, want %08x", x, y, got, want)
					}
				}
			}
			if dst[len(dst)-1] != untouched {
				t.Fatal("written past the end")
			}
		})
	}
}
