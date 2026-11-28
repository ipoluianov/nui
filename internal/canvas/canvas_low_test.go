package canvas

import (
	"image"
	"image/color"
	"testing"
)

// Restore brings back the state Save kept: Save used to push the same
// pointer, so the changes after it changed the saved state too
func TestSaveRestoreKeepsState(t *testing.T) {
	c := NewCanvas(image.NewRGBA(image.Rect(0, 0, 4, 4)))
	c.SetColor(color.RGBA{255, 0, 0, 255})
	c.Save()
	c.SetColor(color.RGBA{0, 255, 0, 255})
	c.Restore()
	if got := c.CurrentState().col; got != (color.RGBA{255, 0, 0, 255}) {
		t.Errorf("after Save/SetColor/Restore the color is %v, want red", got)
	}
}

// BlendPixel takes the color as premultiplied, as color.Color is: 50%
// white over black is 50% gray, not 25%
func TestBlendPixelPremultiplied(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Pix[3], img.Pix[7] = 255, 255
	c := NewCanvas(img)
	c.BlendPixel(0, 0, color.RGBA{128, 128, 128, 128})
	if got := img.RGBAAt(0, 0); got != (color.RGBA{128, 128, 128, 255}) {
		t.Errorf("50%% white over black = %v, want {128 128 128 255}", got)
	}
	// FillRect's alpha scales the whole premultiplied color
	c.SetColor(color.RGBA{255, 255, 255, 255})
	c.FillRect(1, 0, 1, 1, 0.5)
	if got := img.RGBAAt(1, 0); got != (color.RGBA{128, 128, 128, 255}) {
		t.Errorf("FillRect white at 0.5 over black = %v, want {128 128 128 255}", got)
	}
	// Opaque colors replace the pixel, as before
	c.BlendPixel(0, 0, color.RGBA{0, 128, 255, 255})
	if got := img.RGBAAt(0, 0); got != (color.RGBA{0, 128, 255, 255}) {
		t.Errorf("opaque blend = %v", got)
	}
}

// Clear takes any color and fills only the pixels of the image, a
// sub-image too
func TestClearAnyColorSubImage(t *testing.T) {
	c := NewCanvas(image.NewRGBA(image.Rect(0, 0, 2, 2)))
	c.Clear(color.White)
	if got := c.RGBA().RGBAAt(1, 1); got != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("Clear(color.White) gave %v", got)
	}

	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	sub := base.SubImage(image.Rect(5, 5, 10, 10)).(*image.RGBA)
	NewCanvas(sub).Clear(color.RGBA{255, 0, 0, 255})
	if base.RGBAAt(9, 9).R != 255 || base.RGBAAt(5, 5).R != 255 {
		t.Errorf("the sub-image is not cleared whole")
	}
	if base.RGBAAt(4, 4).R != 0 || base.RGBAAt(9, 4).R != 0 || base.RGBAAt(0, 0).R != 0 {
		t.Errorf("Clear of the sub-image wrote outside it")
	}
}

// The coordinates of the canvas start at the top-left corner of its
// image, a sub-image's Rect.Min too
func TestSubImageDrawing(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	sub := base.SubImage(image.Rect(5, 5, 10, 10)).(*image.RGBA)
	c := NewCanvas(sub)
	c.SetColor(color.RGBA{255, 255, 255, 255})
	c.FillRect(0, 0, 5, 5, 1)
	for _, p := range []image.Point{{5, 5}, {9, 9}} {
		if base.RGBAAt(p.X, p.Y).R != 255 {
			t.Errorf("pixel %v of the sub-image is not filled", p)
		}
	}
	c.FillRect(-5, -5, 20, 20, 1)
	for _, p := range []image.Point{{4, 4}, {0, 0}, {4, 9}} {
		if base.RGBAAt(p.X, p.Y).R != 0 {
			t.Errorf("pixel %v outside the sub-image is filled", p)
		}
	}
}
