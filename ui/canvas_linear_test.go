package ui

import (
	"image/color"
	"math"
	"strings"
	"testing"
)

// linearMid is the exact middle in light of the components a and b
func linearMid(a, b uint8, f float64) float64 {
	lin := func(c uint8) float64 {
		v := float64(c) / 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	l := lin(a)*(1-f) + lin(b)*f
	if l <= 0.0031308 {
		return 255 * l * 12.92
	}
	return 255 * (1.055*math.Pow(l, 1/2.4) - 0.055)
}

func TestLinearBlending(t *testing.T) {
	if v := mixLinear(0, 255, 128); math.Abs(float64(v)-linearMid(0, 255, 128.0/255)) > 1 {
		t.Fatalf("mixLinear %d", v)
	}
	gray := func(linear bool, draw func(cnv *Canvas)) uint8 {
		cnv, img := newScaledTestCanvas(20, 20, 1)
		cnv.Save()
		cnv.SetLinearBlending(linear)
		draw(cnv)
		cnv.Restore()
		if cnv.LinearBlending() {
			t.Fatal("not restored")
		}
		return img.RGBAAt(5, 5).R
	}
	half := color.RGBA{255, 255, 255, 128}
	for name, draw := range map[string]func(cnv *Canvas){
		"rect": func(cnv *Canvas) { cnv.FillRect(0, 0, 20, 20, half) },
		"path": func(cnv *Canvas) { cnv.FillPath(pathOf(func(p *Path) { p.Rect(0, 0, 20, 20) }), half) },
		// An edge that covers half of a pixel, opaque
		"edge": func(cnv *Canvas) { cnv.FillPath(pathOf(func(p *Path) { p.Rect(0, 0, 5.5, 20) }), white) },
	} {
		s, l := gray(false, draw), gray(true, draw)
		if math.Abs(float64(s)-128) > 2 || math.Abs(float64(l)-188) > 3 {
			t.Errorf("%s: sRGB %d (want 128), linear %d (want 188)", name, s, l)
		}
	}
}

func TestGradientLinearRGB(t *testing.T) {
	stops := []GradientStop{{0, color.RGBA{255, 0, 0, 255}}, {1, color.RGBA{0, 255, 0, 255}}}
	g := LinearGradient{X2: 100, Stops: stops, LinearRGB: true}
	worst := 0.0
	for x := 0.0; x <= 100; x += 1.7 {
		c := g.ColorAt(x, 0)
		worst = max(worst, math.Abs(float64(c.R)-linearMid(255, 0, x/100)), math.Abs(float64(c.G)-linearMid(0, 255, x/100)))
	}
	if worst > 1.5 {
		t.Fatalf("off the linear mix by %v", worst)
	}
	if c := (LinearGradient{X2: 100, Stops: stops}).ColorAt(50, 0); c.R != 128 {
		t.Fatalf("sRGB by default: %v", c)
	}
	// The fill as ColorAt
	cnv, img := newScaledTestCanvas(100, 4, 1)
	cnv.FillPathPaint(pathOf(func(p *Path) { p.Rect(0, 0, 100, 4) }), g)
	if c, want := img.RGBAAt(50, 2), g.ColorAt(50.5, 2.5); math.Abs(float64(c.R)-float64(want.R)) > 1 || math.Abs(float64(c.G)-float64(want.G)) > 1 {
		t.Fatalf("filled %v, ColorAt %v", c, want)
	}
	for _, paint := range []Paint{RadialGradient{R: 10, Stops: stops, LinearRGB: true}, ConicGradient{Stops: stops, LinearRGB: true}} {
		f := fastPaint(paint)
		if a, b := f(3, 4), paint.ColorAt(3, 4); math.Abs(float64(a.G)-float64(b.G)) > 1 {
			t.Errorf("%T: fast %v, ColorAt %v", paint, a, b)
		}
	}
	// SVG: the stops of linear light
	cnv, _ = newScaledTestCanvas(100, 4, 1)
	rec := NewSVGRecorder(100, 4)
	cnv.SetRecorder(rec)
	cnv.FillPathPaint(pathOf(func(p *Path) { p.Rect(0, 0, 100, 4) }), g)
	if n := strings.Count(rec.String(), "<stop "); n != linearStopsPerSpan+1 {
		t.Fatalf("%d stops in SVG", n)
	}
	// The stops of SVG mixed in sRGB, as a viewer does: near the linear mix
	svgStops := gradientStops(stops, true)
	worst = 0
	for f := 0.0; f <= 1; f += 0.003 {
		c := gradientColor(svgStops, f)
		worst = max(worst, math.Abs(float64(c.R)-linearMid(255, 0, f)), math.Abs(float64(c.G)-linearMid(0, 255, f)))
	}
	if worst > 2.5 {
		t.Fatalf("the stops of SVG off the linear mix by %v", worst)
	}
}

func TestLinearBlendingKeepsText(t *testing.T) {
	draw := func(linear bool) string {
		cnv, img := newWhiteCanvas(200, 30, 1)
		cnv.SetLinearBlending(linear)
		cnv.SetFontSize(14)
		cnv.DrawText(0, 0, 200, 30, "Text weight")
		cnv.DrawTextF(100, 0, "kept")
		cnv.Rotate(0.2)
		cnv.DrawTextF(0, 15, "turned")
		return string(img.Pix)
	}
	if draw(false) != draw(true) {
		t.Fatal("text is mixed in linear light")
	}
}
