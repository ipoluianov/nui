package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestGradientSpread(t *testing.T) {
	stops := []GradientStop{{0, black}, {1, white}}
	g := LinearGradient{X1: 0, X2: 10, Stops: stops}
	if c := g.ColorAt(15, 0); c != white {
		t.Fatalf("pad past the end %v", c)
	}
	g.Spread = SpreadRepeat
	if c := g.ColorAt(15, 0); !near(float64(c.R), 128, 2) {
		t.Fatalf("repeat %v", c)
	}
	g.Spread = SpreadReflect
	if c := g.ColorAt(12, 0); !near(float64(c.R), 204, 2) {
		t.Fatalf("reflect %v", c)
	}
	// The fast fill gives the colors of ColorAt
	for _, spread := range []GradientSpread{SpreadPad, SpreadRepeat, SpreadReflect} {
		for _, paint := range []Paint{
			LinearGradient{X1: 3, Y1: 2, X2: 17, Y2: 9, Stops: stops, Spread: spread},
			RadialGradient{CX: 20, CY: 20, R: 7, Stops: stops, Spread: spread},
			ConicGradient{CX: 20, CY: 20, Angle: 0.3, Stops: stops},
		} {
			f := fastPaint(paint)
			for x := 0.5; x < 40; x += 3.7 {
				for y := 0.5; y < 40; y += 2.9 {
					a, b := f(x, y), paint.ColorAt(x, y)
					if !near(float64(a.R), float64(b.R), 1) || a.A != b.A {
						t.Fatalf("%T at %v,%v: fast %v, ColorAt %v", paint, x, y, a, b)
					}
				}
			}
		}
	}
}

func TestConicGradient(t *testing.T) {
	g := ConicGradient{CX: 0, CY: 0, Stops: []GradientStop{{0, black}, {1, white}}}
	// A quarter turn clockwise from the right: down the screen
	if c := g.ColorAt(0, 10); !near(float64(c.R), 64, 2) {
		t.Fatalf("a quarter %v", c)
	}
	if c := g.ColorAt(-10, 0.0001); !near(float64(c.R), 128, 2) {
		t.Fatalf("a half %v", c)
	}
}

func TestImagePattern(t *testing.T) {
	tile := image.NewRGBA(image.Rect(0, 0, 2, 2))
	tile.SetRGBA(0, 0, red)
	tile.SetRGBA(1, 1, red)
	p := ImagePattern{Image: tile, Repeat: true}
	if p.ColorAt(4.5, 6.5) != red || p.ColorAt(5.5, 6.5) != (color.RGBA{}) {
		t.Fatal("repeated checker")
	}
	p.Repeat = false
	if p.ColorAt(4.5, 4.5) != (color.RGBA{}) {
		t.Fatal("past the image")
	}
	cnv, img := newTestCanvas(10, 10)
	path := NewPath()
	path.Rect(0, 0, 10, 10)
	cnv.FillPathPaint(path, ImagePattern{Image: tile, Repeat: true, Scale: 5})
	if img.RGBAAt(2, 2) != red || img.RGBAAt(7, 2) != black {
		t.Fatalf("scaled pattern %v %v", img.RGBAAt(2, 2), img.RGBAAt(7, 2))
	}
}

// Drawn off the screen, a translucent shape keeps its alpha
func TestImageCanvasAlpha(t *testing.T) {
	cnv, img := NewImageCanvas(10, 10, 1)
	cnv.FillRect(0, 0, 5, 5, color.RGBA{255, 0, 0, 128})
	if got := img.RGBAAt(1, 1); got != (color.RGBA{128, 0, 0, 128}) {
		t.Fatalf("premultiplied pixel %v", got)
	}
	if got := cnv.PixelAt(1, 1); !near(float64(got.R), 255, 1) || got.A != 128 {
		t.Fatalf("PixelAt %v", got)
	}
	cnv.FillCircle(5, 5, 4, color.RGBA{0, 0, 255, 255})
	if got := img.RGBAAt(5, 5); got != (color.RGBA{0, 0, 255, 255}) {
		t.Fatalf("opaque over %v", got)
	}
	cnv.Clear(color.RGBA{})
	if got := img.RGBAAt(5, 5); got != (color.RGBA{}) {
		t.Fatalf("cleared %v", got)
	}
}

func TestLayerOpacity(t *testing.T) {
	cnv, img := newTestCanvas(40, 20)
	cnv.BeginLayer(0.5)
	cnv.FillRect(0, 0, 25, 20, red)
	cnv.FillRect(15, 0, 25, 20, red)
	cnv.EndLayer()
	// The overlap is not darker than the rest
	a, b := img.RGBAAt(5, 5), img.RGBAAt(20, 5)
	if a != b || !near(float64(a.R), 128, 1) {
		t.Fatalf("layer %v, overlap %v", a, b)
	}
}

func TestBlur(t *testing.T) {
	cnv, img := newTestCanvas(40, 40)
	cnv.FillRect(20, 0, 20, 40, white)
	before := inkSum(img)
	cnv.Blur(0, 0, 40, 40, 6)
	if after := inkSum(img); !near(after, before, before*0.02) {
		t.Fatalf("blur changed the ink %v -> %v", before, after)
	}
	if r := img.RGBAAt(20, 20).R; r < 100 || r > 160 {
		t.Fatalf("the edge after the blur %d", r)
	}
	if r := img.RGBAAt(2, 20).R; r != 0 {
		t.Fatalf("far from the edge %d", r)
	}
}

func TestDrawShadow(t *testing.T) {
	cnv, img := newTestCanvas(60, 60)
	cnv.FillRect(0, 0, 60, 60, white)
	cnv.DrawShadow(15, 15, 30, 30, 4, 8, color.RGBA{0, 0, 0, 200})
	if c := img.RGBAAt(30, 30); c.R > 80 {
		t.Fatalf("under the shape %v", c)
	}
	if c := img.RGBAAt(12, 30); c.R == 255 || c.R < 100 {
		t.Fatalf("soft edge %v", c)
	}
	if c := img.RGBAAt(1, 1); c != white {
		t.Fatalf("far away %v", c)
	}
}

func TestFloodFill(t *testing.T) {
	cnv, img := newTestCanvas(30, 30)
	cnv.DrawCircle(15, 15, 10, 2, white)
	cnv.FloodFill(15, 15, red, 40)
	if img.RGBAAt(15, 15) != red || img.RGBAAt(1, 1) != black {
		t.Fatal("fill inside the circle only")
	}
	cnv.AdjustPixels(0, 0, 30, 30, func(c color.RGBA) color.RGBA { return color.RGBA{c.G, c.R, c.B, c.A} })
	if img.RGBAAt(15, 15) != (color.RGBA{0, 255, 0, 255}) {
		t.Fatalf("AdjustPixels %v", img.RGBAAt(15, 15))
	}
	_ = math.Pi
}
