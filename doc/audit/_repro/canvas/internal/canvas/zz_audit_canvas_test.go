package canvas

import (
	"image"
	"image/color"
	"testing"
)

func TestAuditSaveRestore(t *testing.T) {
	c := NewCanvas(image.NewRGBA(image.Rect(0, 0, 4, 4)))
	c.SetColor(color.RGBA{255, 0, 0, 255})
	c.Save()
	c.SetColor(color.RGBA{0, 255, 0, 255})
	c.Restore()
	if got := c.CurrentState().col; got != (color.RGBA{255, 0, 0, 255}) {
		t.Errorf("after Save/SetColor/Restore the color is %v, want red", got)
	}
}

func TestAuditBlendPixelTranslucent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	c := NewCanvas(img)
	// premultiplied 50% white over black: expected ~128
	c.BlendPixel(0, 0, color.RGBA{128, 128, 128, 128})
	t.Logf("50%% white over black -> %v (want ~128)", img.RGBAAt(0, 0))
	if v := img.RGBAAt(0, 0).R; v < 120 {
		t.Errorf("translucent blend too dark: %d, want ~128", v)
	}
}

func TestAuditClearNonRGBA(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Clear(color.White) panics: %v", r)
		}
	}()
	c := NewCanvas(image.NewRGBA(image.Rect(0, 0, 2, 2)))
	c.Clear(color.White)
}

func TestAuditSubImage(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	sub := base.SubImage(image.Rect(5, 5, 10, 10)).(*image.RGBA)
	c := NewCanvas(sub)
	c.SetColor(color.RGBA{255, 255, 255, 255})
	c.FillRect(0, 0, 5, 5, 1) // whole sub image
	if base.RGBAAt(9, 9).R != 255 {
		t.Errorf("FillRect on a sub-image left pixel (9,9) = %v; BlendPixel takes x,y from 0, not Rect.Min", base.RGBAAt(9, 9))
	}
}
