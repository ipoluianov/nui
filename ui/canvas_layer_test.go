package ui

import (
	"image/color"
	"testing"
)

// nearRGBA tells whether the colors differ by tol at most, each component
func nearRGBA(a, b color.RGBA, tol int) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= tol && int(y)-int(x) <= tol }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}

func TestBlendModes(t *testing.T) {
	gray := color.RGBA{128, 128, 128, 255}
	for _, tc := range []struct {
		mode  BlendMode
		layer color.RGBA
		want  color.RGBA
	}{
		{BlendMultiply, red, color.RGBA{128, 0, 0, 255}},
		{BlendScreen, color.RGBA{128, 0, 0, 255}, color.RGBA{192, 128, 128, 255}},
		{BlendDarken, color.RGBA{200, 50, 128, 255}, color.RGBA{128, 50, 128, 255}},
		{BlendLighten, color.RGBA{200, 50, 128, 255}, color.RGBA{200, 128, 128, 255}},
		{BlendDifference, color.RGBA{255, 0, 128, 255}, color.RGBA{127, 128, 0, 255}},
		{BlendAdd, color.RGBA{200, 10, 0, 255}, color.RGBA{255, 138, 128, 255}},
		{BlendColor, red, color.RGBA{255, 73, 73, 255}},
		{BlendLuminosity, white, white},
		{BlendNormal, red, red},
	} {
		cnv, img := newScaledTestCanvas(20, 20, 1)
		cnv.FillRect(0, 0, 20, 20, gray)
		cnv.BeginLayerStyle(LayerStyle{Blend: tc.mode})
		cnv.FillRect(0, 0, 10, 20, tc.layer)
		cnv.EndLayer()
		if c := at(img, 5, 5); !nearRGBA(c, tc.want, 2) {
			t.Errorf("mode %d: %v, want %v", tc.mode, c, tc.want)
		}
		if c := at(img, 15, 5); c != gray {
			t.Errorf("mode %d: %v where the layer is not drawn", tc.mode, c)
		}
	}
}

func TestBlendOpacity(t *testing.T) {
	cnv, img := newScaledTestCanvas(20, 20, 1)
	cnv.FillRect(0, 0, 20, 20, white)
	cnv.BeginLayerStyle(LayerStyle{Blend: BlendMultiply, Opacity: 0.5})
	cnv.FillRect(0, 0, 20, 20, black)
	cnv.EndLayer()
	if c := at(img, 5, 5); !nearRGBA(c, color.RGBA{128, 128, 128, 255}, 2) {
		t.Fatalf("%v", c)
	}
}

func TestBlendErase(t *testing.T) {
	cnv, img := NewImageCanvas(40, 40, 1)
	cnv.FillRect(0, 0, 40, 40, red)
	cnv.BeginLayerStyle(LayerStyle{Blend: BlendErase})
	cnv.FillPath(pathOf(func(p *Path) { p.Circle(20, 20, 10) }), color.RGBA{0, 255, 0, 255})
	cnv.FillRect(0, 0, 5, 5, color.RGBA{0, 0, 255, 128})
	cnv.EndLayer()
	rgba := img
	if c := at(rgba, 20, 20); c.A != 0 {
		t.Fatalf("not erased %v", c)
	}
	if c := at(rgba, 35, 20); c != red {
		t.Fatalf("erased outside %v", c)
	}
	if c := at(rgba, 2, 2); !nearRGBA(c, color.RGBA{127, 0, 0, 127}, 2) {
		t.Fatalf("half erased %v", c)
	}
}

func TestBlendKeepInsideAndAtop(t *testing.T) {
	cnv, img := NewImageCanvas(40, 40, 1)
	rgba := img
	cnv.FillRect(0, 0, 40, 40, red)
	cnv.BeginLayerStyle(LayerStyle{Blend: BlendKeepInside})
	cnv.FillRect(10, 10, 10, 10, black)
	cnv.EndLayer()
	if at(rgba, 15, 15) != red || at(rgba, 5, 5).A != 0 {
		t.Fatalf("keep inside %v %v", at(rgba, 15, 15), at(rgba, 5, 5))
	}

	cnv.BeginLayerStyle(LayerStyle{Blend: BlendAtop})
	cnv.FillRect(0, 0, 40, 40, white)
	cnv.EndLayer()
	if at(rgba, 15, 15) != white || at(rgba, 5, 5).A != 0 {
		t.Fatalf("atop %v %v", at(rgba, 15, 15), at(rgba, 5, 5))
	}
}

func TestLayerMask(t *testing.T) {
	for _, blend := range []BlendMode{BlendNormal, BlendLighten} {
		cnv, img := newScaledTestCanvas(100, 10, 2)
		fade := LinearGradient{X1: 0, X2: 100, Stops: []GradientStop{{0, white}, {1, color.RGBA{}}}}
		cnv.BeginLayerStyle(LayerStyle{Blend: blend, Mask: fade})
		cnv.FillRect(0, 0, 100, 10, white)
		cnv.EndLayer()
		if c := at(img, 2, 5); c.R < 250 {
			t.Errorf("blend %d: the start %v", blend, c)
		}
		if c := at(img, 100, 5); !near(float64(c.R), 128, 4) {
			t.Errorf("blend %d: the middle %v", blend, c)
		}
		if c := at(img, 197, 5); c.R > 5 {
			t.Errorf("blend %d: the end %v", blend, c)
		}
	}
	// The mask in the coordinates of the drawing
	cnv, img := newScaledTestCanvas(100, 10, 1)
	cnv.Translate(50, 0)
	cnv.BeginLayerStyle(LayerStyle{Mask: LinearGradient{X1: 0, X2: 1, Stops: []GradientStop{{0, color.RGBA{}}, {1, white}}}})
	cnv.ResetTransform()
	cnv.FillRect(0, 0, 100, 10, white)
	cnv.EndLayer()
	if at(img, 40, 5).R != 0 || at(img, 60, 5).R != 255 {
		t.Fatalf("transformed mask %v %v", at(img, 40, 5), at(img, 60, 5))
	}
}

func TestLayerOwnPicture(t *testing.T) {
	// Text, images, paths drawn in a layer of its own as on the canvas
	draw := func(cnv *Canvas) {
		cnv.SetColor(white)
		cnv.SetFontSize(14)
		cnv.DrawText(5, 5, 100, 20, "Layers")
		cnv.StrokePath(pathOf(func(p *Path) { p.Circle(60, 40, 15) }), StrokeStyle{Width: 3}, white)
		cnv.FillRect(0, 60, 30, 10, white)
	}
	direct, want := newScaledTestCanvas(120, 80, 1.5)
	draw(direct)
	cnv, img := newScaledTestCanvas(120, 80, 1.5)
	cnv.Save()
	cnv.TranslateAndClip(0, 0, 120, 80)
	cnv.BeginLayerStyle(LayerStyle{Blend: BlendLighten})
	if cnv.Width() != 120 || cnv.Height() != 80 {
		t.Fatalf("the size in a layer %d×%d", cnv.Width(), cnv.Height())
	}
	cnv.BeginLayerStyle(LayerStyle{Blend: BlendScreen})
	draw(cnv)
	cnv.EndLayer()
	cnv.EndLayer()
	cnv.Restore()
	for i := range img.Pix {
		if d := int(img.Pix[i]) - int(want.Pix[i]); d > 2 || d < -2 {
			t.Fatalf("pixel %d: %d, want %d", i/4, img.Pix[i], want.Pix[i])
		}
	}
}

func TestLayerEndsClips(t *testing.T) {
	cnv, img := newScaledTestCanvas(40, 40, 1)
	cnv.Save()
	cnv.BeginLayerStyle(LayerStyle{Blend: BlendLighten})
	cnv.ClipPath(pathOf(func(p *Path) { p.Rect(0, 0, 10, 10) }))
	cnv.FillRect(0, 0, 40, 40, red)
	cnv.EndLayer() // the clip ended with the layer
	cnv.Restore()
	if at(img, 5, 5) != red || at(img, 20, 20) != black {
		t.Fatalf("%v %v", at(img, 5, 5), at(img, 20, 20))
	}
}

func BenchmarkLayerMultiply(b *testing.B) {
	cnv, _ := newScaledTestCanvas(300, 200, 1)
	for b.Loop() {
		cnv.BeginLayerStyle(LayerStyle{Blend: BlendMultiply})
		cnv.FillRect(0, 0, 300, 200, red)
		cnv.EndLayer()
	}
}

func BenchmarkLayerErase(b *testing.B) {
	cnv, _ := newScaledTestCanvas(300, 200, 1)
	for b.Loop() {
		cnv.BeginLayerStyle(LayerStyle{Blend: BlendErase})
		cnv.FillRect(0, 0, 300, 200, red)
		cnv.EndLayer()
	}
}

func TestLayerIsolated(t *testing.T) {
	cnv, img := newScaledTestCanvas(40, 20, 1)
	cnv.FillRect(0, 0, 40, 20, white)
	cnv.BeginLayerStyle(LayerStyle{Isolated: true, Opacity: 1})
	cnv.FillRect(0, 0, 40, 20, red)
	cnv.FillRect(0, 0, 20, 20, color.RGBA{0, 0, 255, 128})
	cnv.BeginLayerStyle(LayerStyle{Blend: BlendErase})
	cnv.FillRect(30, 0, 10, 20, black)
	cnv.EndLayer()
	cnv.EndLayer()
	// Erased in the layer: the paper under it is there
	if c := at(img, 35, 10); c != white {
		t.Fatalf("erased %v", c)
	}
	if c := at(img, 25, 10); c != red {
		t.Fatalf("the layer %v", c)
	}
	if c := at(img, 10, 10); !nearRGBA(c, color.RGBA{127, 0, 128, 255}, 2) {
		t.Fatalf("translucent in the layer %v", c)
	}
}
