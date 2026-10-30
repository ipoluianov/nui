package ui

import (
	"image"
	"image/color"
	"math"
	"sync"
)

// MipImage is an image that keeps copies of itself each half the size of
// the one before (mipmaps). Drawn smaller than half its size, smoothly, it
// is sampled from the copy of the nearest size: a photo, a map or a page
// seen small stays smooth instead of grainy, and costs no more than a
// small image. Draw it as any image:
//
//	photo := ui.NewMipImage(img)
//	cnv.DrawImageRect(0, 0, 160, 120, photo, photo.Bounds(), true)
//
// The copies are made when first needed. After the pixels of the image
// change, call Invalidate.
type MipImage struct {
	image.Image
	mu     sync.Mutex
	levels []*image.RGBA // levels[i] is 2^(i+1) times smaller
}

// NewMipImage returns the image with its mipmaps
func NewMipImage(img image.Image) *MipImage {
	if m, ok := img.(*MipImage); ok {
		return m
	}
	return &MipImage{Image: img}
}

// Invalidate drops the copies: they are made again from the image
func (m *MipImage) Invalidate() {
	m.mu.Lock()
	m.levels = nil
	m.mu.Unlock()
}

// level returns the copy n times halved (n >= 1), making it if needed
func (m *MipImage) level(n int) *image.RGBA {
	m.mu.Lock()
	defer m.mu.Unlock()
	for len(m.levels) < n {
		var from *image.RGBA
		if len(m.levels) > 0 {
			from = m.levels[len(m.levels)-1]
		} else {
			from = toRGBA(m.Image)
		}
		m.levels = append(m.levels, halve(from))
	}
	return m.levels[n-1]
}

// halve returns the image half the size, each pixel the mean of the two by
// two it comes from (of the one or two at an odd edge)
func halve(src *image.RGBA) *image.RGBA {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	dw, dh := max(1, (w+1)/2), max(1, (h+1)/2)
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		y0, y1 := min(2*y, h-1), min(2*y+1, h-1)
		r0 := src.Pix[src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y+y0):]
		r1 := src.Pix[src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y+y1):]
		out := dst.Pix[y*dst.Stride:]
		for x := 0; x < dw; x++ {
			x0, x1 := min(2*x, w-1)*4, min(2*x+1, w-1)*4
			for i := 0; i < 4; i++ {
				out[x*4+i] = uint8((uint32(r0[x0+i]) + uint32(r0[x1+i]) + uint32(r1[x0+i]) + uint32(r1[x1+i]) + 2) / 4)
			}
		}
	}
	return dst
}

// sourceImage returns the image to sample for the part src of img drawn at
// the scale k (pixels of the canvas a pixel of the image), smooth: a
// halved copy of a MipImage drawn small, with the part src in its pixels
// and the scale of the copy to the image (1/2^n); the image itself
// otherwise
func sourceImage(img image.Image, src image.Rectangle, k float64, smooth bool) (image.Image, image.Rectangle, float64) {
	m, ok := img.(*MipImage)
	if !ok {
		return img, src, 1
	}
	if !smooth || !(k < 0.5) || k <= 0 {
		return m.Image, src, 1
	}
	// The copy that is drawn at a scale of 0.5..1
	n := int(math.Floor(math.Log2(1 / k)))
	b := m.Image.Bounds()
	n = min(n, int(math.Ceil(math.Log2(float64(max(b.Dx(), b.Dy()))))))
	if n < 1 {
		return m.Image, src, 1
	}
	lv := m.level(n)
	f := 1 / float64(int(1)<<n)
	s := src.Sub(b.Min)
	part := image.Rect(int(math.Floor(float64(s.Min.X)*f)), int(math.Floor(float64(s.Min.Y)*f)),
		int(math.Ceil(float64(s.Max.X)*f)), int(math.Ceil(float64(s.Max.Y)*f))).Intersect(lv.Rect)
	return lv, part, f
}

// ColorModel, Bounds and At are those of the image

func (m *MipImage) ColorModel() color.Model { return m.Image.ColorModel() }
func (m *MipImage) Bounds() image.Rectangle { return m.Image.Bounds() }
func (m *MipImage) At(x, y int) color.Color { return m.Image.At(x, y) }
