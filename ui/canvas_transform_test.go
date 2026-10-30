package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

func near(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestMatrixAlgebra(t *testing.T) {
	m := IdentityMatrix().Translate(10, 20).Rotate(math.Pi/2).Scale(2, 3)
	// Scale first, then rotate, then translate
	x, y := m.Apply(1, 0)
	if !near(x, 10, 1e-9) || !near(y, 22, 1e-9) {
		t.Fatalf("Apply(1, 0) = %v, %v, want 10, 22", x, y)
	}
	inv, ok := m.Invert()
	if !ok {
		t.Fatal("not invertible")
	}
	id := m.Then(inv)
	for _, v := range [][2]float64{{0, 0}, {3, -7}, {1e3, 5}} {
		x, y := id.Apply(v[0], v[1])
		if !near(x, v[0], 1e-9) || !near(y, v[1], 1e-9) {
			t.Fatalf("m then inverse moved %v to %v, %v", v, x, y)
		}
	}
	if _, ok := ScalingMatrix(0, 1).Invert(); ok {
		t.Fatal("a flat transform inverted")
	}
	if !IdentityMatrix().IsIdentity() || (Matrix{}).IsIdentity() {
		t.Fatal("IsIdentity")
	}
	if got := TranslationMatrix(1, 2).Then(ScalingMatrix(2, 2)); got != (Matrix{A: 2, D: 2, E: 2, F: 4}) {
		t.Fatalf("Then = %+v", got)
	}
}

// A shape moved by the transform is the shape moved by its coordinates
func TestTransformTranslatePixelExact(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		a, imgA := newScaledTestCanvas(60, 60, scale)
		b, imgB := newScaledTestCanvas(60, 60, scale)
		p := NewPath()
		p.Circle(20, 20, 9)
		p.Rect(5, 30, 13.5, 7)
		a.Translate(7, 4)
		a.FillPath(p, red)
		a.FillRect(30, 2, 10, 6, color.RGBA{0, 255, 0, 255})
		q := p.Clone()
		q.Transform(TranslationMatrix(7, 4))
		b.FillPath(q, red)
		b.FillRect(37, 6, 10, 6, color.RGBA{0, 255, 0, 255})
		for i := range imgA.Pix {
			if d := int(imgA.Pix[i]) - int(imgB.Pix[i]); d < -1 || d > 1 {
				t.Fatalf("scale %v: pixel %d differs: %d vs %d", scale, i/4, imgA.Pix[i], imgB.Pix[i])
			}
		}
	}
}

func TestTransformRotateQuarter(t *testing.T) {
	cnv, img := newTestCanvas(60, 60)
	cnv.RotateAround(math.Pi/2, 30, 30)
	cnv.FillRect(10, 25, 40, 10, red) // a horizontal bar, turned upright
	b := inkBounds(img)
	if !near(float64(b.Dx()), 10, 1) || !near(float64(b.Dy()), 40, 1) {
		t.Fatalf("turned bar covers %v, want 10 x 40", b)
	}
}

// Save and Restore keep the transform, as the clip
func TestTransformSaveRestore(t *testing.T) {
	cnv, _ := newTestCanvas(10, 10)
	cnv.Translate(5, 5)
	cnv.Save()
	cnv.Rotate(1)
	cnv.Restore()
	if got := cnv.Transform(); got != TranslationMatrix(5, 5) {
		t.Fatalf("transform after Restore = %+v", got)
	}
	x, y := cnv.FromCanvas(cnv.ToCanvas(3, 4))
	if !near(x, 3, 1e-9) || !near(y, 4, 1e-9) {
		t.Fatalf("FromCanvas(ToCanvas) = %v, %v", x, y)
	}
	cnv.ResetTransform()
	if !cnv.Transform().IsIdentity() {
		t.Fatal("ResetTransform")
	}
}

// A stroke stretched twice is a stroke twice as wide
func TestTransformStrokeWidth(t *testing.T) {
	a, imgA := newTestCanvas(100, 100)
	b, imgB := newTestCanvas(100, 100)
	p := NewPath()
	p.MoveTo(5, 5)
	p.LineTo(40, 30)
	a.ScaleBy(2, 2)
	a.StrokePath(p, StrokeStyle{Width: 3, Join: LineJoinRound}, red)
	q := p.Clone()
	q.Transform(ScalingMatrix(2, 2))
	b.StrokePath(q, StrokeStyle{Width: 6, Join: LineJoinRound}, red)
	if sa, sb := inkSum(imgA), inkSum(imgB); !near(sa, sb, sb*0.01) {
		t.Fatalf("ink %v, want %v", sa, sb)
	}
}

// A gradient under a transform is in the drawing coordinates
func TestTransformPaint(t *testing.T) {
	cnv, img := newTestCanvas(40, 40)
	cnv.Translate(20, 0)
	p := NewPath()
	p.Rect(0, 0, 20, 40)
	cnv.FillPathPaint(p, LinearGradient{X1: 0, X2: 20, Stops: []GradientStop{{0, color.RGBA{0, 0, 0, 255}}, {1, color.RGBA{255, 0, 0, 255}}}})
	if r := img.RGBAAt(21, 10).R; r > 20 {
		t.Fatalf("start of the gradient %d, want dark", r)
	}
	if r := img.RGBAAt(38, 10).R; r < 230 {
		t.Fatalf("end of the gradient %d, want red", r)
	}
}

func TestFillRuleEvenOdd(t *testing.T) {
	ring := func(rule FillRule) *image.RGBA {
		cnv, img := newTestCanvas(60, 60)
		p := NewPath()
		// Both circles the same way round: only even-odd makes a hole
		p.Circle(30, 30, 25)
		p.Circle(30, 30, 12)
		p.SetFillRule(rule)
		cnv.FillPath(p, red)
		return img
	}
	nz, eo := ring(FillRuleNonZero), ring(FillRuleEvenOdd)
	if nz.RGBAAt(30, 30) != red {
		t.Fatalf("nonzero center = %v, want filled", nz.RGBAAt(30, 30))
	}
	if eo.RGBAAt(30, 30) != black {
		t.Fatalf("even-odd center = %v, want a hole", eo.RGBAAt(30, 30))
	}
	if eo.RGBAAt(30, 10) != red {
		t.Fatalf("even-odd ring = %v, want filled", eo.RGBAAt(30, 10))
	}
	// A simple shape is the same by both rules, its edges too
	simple := func(rule FillRule) float64 {
		cnv, img := newTestCanvas(60, 60)
		p := NewPath()
		p.Polygon(PointF{3.3, 5}, PointF{50, 12.7}, PointF{20, 55})
		p.SetFillRule(rule)
		cnv.FillPath(p, red)
		return inkSum(img)
	}
	if a, b := simple(FillRuleNonZero), simple(FillRuleEvenOdd); !near(a, b, a*0.002) {
		t.Fatalf("triangle ink nonzero %v, even-odd %v", a, b)
	}
	// A star crossing itself: the even-odd one has a hole in the middle
	star := NewPath()
	for i := 0; i < 5; i++ {
		a := -math.Pi/2 + float64(i)*4*math.Pi/5
		if i == 0 {
			star.MoveTo(30+25*math.Cos(a), 30+25*math.Sin(a))
		} else {
			star.LineTo(30+25*math.Cos(a), 30+25*math.Sin(a))
		}
	}
	star.Close()
	star.SetFillRule(FillRuleEvenOdd)
	cnv, img := newTestCanvas(60, 60)
	cnv.FillPath(star, red)
	if img.RGBAAt(30, 31) != black {
		t.Fatalf("even-odd star center = %v", img.RGBAAt(30, 31))
	}
}

func TestDrawImageTransformed(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	cnv, img := newTestCanvas(40, 40)
	// Turned upright and moved: 4 wide, 8 tall
	cnv.DrawImageTransformed(src, RotationMatrix(math.Pi/2).Then(TranslationMatrix(20, 10)), false)
	b := inkBounds(img)
	if b != image.Rect(16, 10, 20, 18) {
		t.Fatalf("turned image covers %v", b)
	}
	// Under a whole-pixel move it is DrawImage's image
	a, imgA := newTestCanvas(40, 40)
	a.Translate(3, 5)
	a.DrawImage(2, 2, src)
	if b := inkBounds(imgA); b != image.Rect(5, 7, 13, 11) {
		t.Fatalf("moved image covers %v", b)
	}
}

func TestDrawTextTransformed(t *testing.T) {
	cnv, img := newTestCanvas(120, 120)
	cnv.SetColor(color.RGBA{255, 255, 255, 255})
	cnv.SetFontSize(14)
	cnv.RotateAround(math.Pi/2, 60, 60)
	cnv.DrawText(10, 50, 100, 20, "Turned text")
	b := inkBounds(img)
	if b.Empty() || b.Dy() <= b.Dx() {
		t.Fatalf("text turned upright covers %v", b)
	}
}

func TestSVGRecorderTransform(t *testing.T) {
	cnv, _ := newTestCanvas(100, 100)
	rec := NewSVGRecorder(100, 100)
	cnv.SetRecorder(rec)
	cnv.Translate(10, 20)
	p := NewPath()
	p.Rect(0, 0, 5, 5)
	p.SetFillRule(FillRuleEvenOdd)
	cnv.FillPath(p, red)
	cnv.FillPathPaint(p, LinearGradient{X2: 5, Stops: []GradientStop{{0, red}, {1, black}}})
	svg := rec.String()
	for _, want := range []string{"M10 20L15 20", `fill-rule="evenodd"`, `gradientTransform="matrix(1 0 0 1 10 20)"`} {
		if !strings.Contains(svg, want) {
			t.Fatalf("SVG has no %q:\n%s", want, svg)
		}
	}
}
