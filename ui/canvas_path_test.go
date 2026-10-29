package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"
)

// inkBounds returns the rectangle of the pixels brighter than black
func inkBounds(img *image.RGBA) image.Rectangle {
	var r image.Rectangle
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.R|c.G|c.B != 0 {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return r
}

// inkSum returns the brightness of the red component added up, in pixels
func inkSum(img *image.RGBA) float64 {
	sum := 0.0
	for i := 0; i < len(img.Pix); i += 4 {
		sum += float64(img.Pix[i]) / 255
	}
	return sum
}

// A rectangle on whole coordinates fills whole pixels: sharp, nothing around
func TestFillPathRectSharp(t *testing.T) {
	cnv, img := newTestCanvas(20, 20)
	p := NewPath()
	p.Rect(3, 4, 10, 5)
	cnv.FillPath(p, white)
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			want := uint8(0)
			if x >= 3 && x < 13 && y >= 4 && y < 9 {
				want = 255
			}
			if got := img.RGBAAt(x, y).R; got != want {
				t.Fatalf("pixel %d,%d = %d, want %d", x, y, got, want)
			}
		}
	}
}

// A line one wide through the middle of a row covers that row only
func TestStrokePathCrispLine(t *testing.T) {
	cnv, img := newTestCanvas(20, 10)
	p := NewPath()
	p.MoveTo(2, 5.5)
	p.LineTo(15, 5.5)
	cnv.StrokePath(p, StrokeStyle{}, white)
	for x := 2; x < 15; x++ {
		if got := img.RGBAAt(x, 5).R; got != 255 {
			t.Fatalf("pixel %d,5 = %d, want 255", x, got)
		}
		if img.RGBAAt(x, 4).R != 0 || img.RGBAAt(x, 6).R != 0 {
			t.Fatalf("rows around the line drawn at x %d", x)
		}
	}
	if img.RGBAAt(1, 5).R != 0 || img.RGBAAt(15, 5).R != 0 {
		t.Fatal("butt ends go past the end points")
	}
}

// A translucent line that crosses itself is blended once where it crosses
func TestStrokePathTranslucentOverlapOnce(t *testing.T) {
	cnv, img := newTestCanvas(30, 30)
	p := NewPath()
	p.MoveTo(5, 15.5)
	p.LineTo(25, 15.5)
	p.LineTo(25, 25)
	p.LineTo(15.5, 25)
	p.LineTo(15.5, 5)
	cnv.StrokePath(p, StrokeStyle{Width: 3}, color.RGBA{255, 255, 255, 128})
	cross, plain := img.RGBAAt(15, 15).R, img.RGBAAt(8, 15).R
	if cross != plain {
		t.Fatalf("crossing = %d, the line elsewhere = %d", cross, plain)
	}
}

// The dashes of the pattern alternate with the gaps
func TestStrokePathDash(t *testing.T) {
	cnv, img := newTestCanvas(40, 5)
	p := NewPath()
	p.MoveTo(0, 2.5)
	p.LineTo(40, 2.5)
	cnv.StrokePath(p, StrokeStyle{Dash: []float64{4, 4}}, white)
	for x := 0; x < 40; x++ {
		want := uint8(0)
		if (x/4)%2 == 0 {
			want = 255
		}
		if got := img.RGBAAt(x, 2).R; got != want {
			t.Fatalf("pixel %d = %d, want %d", x, got, want)
		}
	}
	// The offset shifts the pattern
	cnv2, img2 := newTestCanvas(40, 5)
	cnv2.StrokePath(p, StrokeStyle{Dash: []float64{4, 4}, DashOffset: 4}, white)
	if img2.RGBAAt(1, 2).R != 0 || img2.RGBAAt(5, 2).R != 255 {
		t.Fatalf("offset: %d %d", img2.RGBAAt(1, 2).R, img2.RGBAAt(5, 2).R)
	}
}

// A circle's ink adds up to its area
func TestFillCircleArea(t *testing.T) {
	cnv, img := newTestCanvas(40, 40)
	cnv.FillCircle(20, 20, 10, white)
	if got, want := inkSum(img), math.Pi*100; math.Abs(got-want) > want*0.01 {
		t.Fatalf("area = %.1f, want %.1f", got, want)
	}
}

// A subpath drawn the other way round inside another one is a hole
func TestFillPathHole(t *testing.T) {
	cnv, img := newTestCanvas(30, 30)
	p := NewPath()
	p.Rect(0, 0, 30, 30)
	// Counterclockwise
	p.MoveTo(10, 10)
	p.LineTo(10, 20)
	p.LineTo(20, 20)
	p.LineTo(20, 10)
	p.Close()
	cnv.FillPath(p, white)
	if img.RGBAAt(15, 15).R != 0 || img.RGBAAt(5, 5).R != 255 {
		t.Fatalf("hole %d, ring %d", img.RGBAAt(15, 15).R, img.RGBAAt(5, 5).R)
	}
}

// Points a billion pixels away are clipped, quickly
func TestStrokePathHugeCoordinates(t *testing.T) {
	cnv, img := newTestCanvas(50, 50)
	p := NewPath()
	p.MoveTo(-1e9, 25.5)
	p.LineTo(1e9, 25.5)
	p.MoveTo(25.5, -1e9)
	p.LineTo(25.5, 1e9)
	start := time.Now()
	cnv.StrokePath(p, StrokeStyle{Width: 1}, white)
	cnv.FillPath(p, white)
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Fatalf("took %v", d)
	}
	if img.RGBAAt(0, 25).R != 255 || img.RGBAAt(49, 25).R != 255 || img.RGBAAt(25, 0).R != 255 {
		t.Fatal("the lines are not drawn across the canvas")
	}
}

// A sharp corner sticks out with a miter join and is cut with a bevel
func TestStrokePathJoins(t *testing.T) {
	corner := func(join LineJoin) uint8 {
		cnv, img := newTestCanvas(40, 40)
		p := NewPath()
		p.MoveTo(5, 30)
		p.LineTo(20, 10)
		p.LineTo(35, 30)
		cnv.StrokePath(p, StrokeStyle{Width: 6, Join: join}, white)
		// Above the vertex, where only the miter reaches
		return img.RGBAAt(20, 6).R
	}
	if m, b := corner(LineJoinMiter), corner(LineJoinBevel); m < 200 || b > 50 {
		t.Fatalf("miter %d, bevel %d", m, b)
	}
	if r := corner(LineJoinRound); r > corner(LineJoinMiter) {
		t.Fatalf("round %d reaches further than miter", r)
	}
}

// The caps: butt ends at the point, square and round go half the width past
func TestStrokePathCaps(t *testing.T) {
	past := func(capStyle LineCap) uint8 {
		cnv, img := newTestCanvas(40, 20)
		p := NewPath()
		p.MoveTo(10, 10)
		p.LineTo(30, 10)
		cnv.StrokePath(p, StrokeStyle{Width: 8, Cap: capStyle}, white)
		return img.RGBAAt(32, 10).R
	}
	if past(LineCapButt) != 0 || past(LineCapSquare) != 255 || past(LineCapRound) < 200 {
		t.Fatalf("butt %d square %d round %d", past(LineCapButt), past(LineCapSquare), past(LineCapRound))
	}
}

// On a HiDPI canvas the logical rectangle takes twice the pixels
func TestFillPathScaled(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 40))
	cnv := NewCanvasScaled(img, 2)
	cnv.SetDirectTranslateAndClip(0, 0, 20, 20)
	p := NewPath()
	p.Rect(2, 2, 5, 5)
	cnv.FillPath(p, white)
	if got := inkBounds(img); got != image.Rect(4, 4, 14, 14) {
		t.Fatalf("ink %v, want (4,4)-(14,14)", got)
	}
}

// The path is drawn within the clip and the translation
func TestFillPathClipAndTranslate(t *testing.T) {
	cnv, img := newTestCanvas(40, 40)
	cnv.TranslateAndClip(10, 10, 10, 10)
	p := NewPath()
	p.Rect(-5, -5, 50, 50)
	cnv.FillPath(p, white)
	if got := inkBounds(img); got != image.Rect(10, 10, 20, 20) {
		t.Fatalf("ink %v, want the clip", got)
	}
}

// A sector of a quarter is in its quarter only
func TestPathSector(t *testing.T) {
	cnv, img := newTestCanvas(40, 40)
	p := NewPath()
	p.Sector(20, 20, 0, 15, 0, math.Pi/2) // from the right to the bottom
	cnv.FillPath(p, white)
	if img.RGBAAt(26, 26).R != 255 || img.RGBAAt(14, 26).R != 0 || img.RGBAAt(26, 14).R != 0 {
		t.Fatalf("bottom right %d, bottom left %d, top right %d", img.RGBAAt(26, 26).R, img.RGBAAt(14, 26).R, img.RGBAAt(26, 14).R)
	}
	// A ring's sector leaves the middle empty
	cnv2, img2 := newTestCanvas(40, 40)
	p.Reset()
	p.Sector(20, 20, 8, 15, 0, math.Pi*2-0.01)
	cnv2.FillPath(p, white)
	if img2.RGBAAt(20, 20).R != 0 || img2.RGBAAt(31, 20).R != 255 {
		t.Fatalf("donut: middle %d, ring %d", img2.RGBAAt(20, 20).R, img2.RGBAAt(31, 20).R)
	}
}

// A curve is flattened close to it: the circle of four cubic arcs
func TestPathCurvesClose(t *testing.T) {
	cnv, img := newTestCanvas(100, 100)
	p := NewPath()
	p.MoveTo(10, 90)
	p.QuadTo(50, -70, 90, 90) // the top of the curve at y 10
	cnv.StrokePath(p, StrokeStyle{Width: 2}, white)
	top := inkBounds(img).Min.Y
	if top < 8 || top > 10 {
		t.Fatalf("the top of the curve at %d, want about 9", top)
	}
}

// A lone MoveTo draws nothing, not even round caps
func TestStrokePathLoneMoveTo(t *testing.T) {
	cnv, img := newTestCanvas(20, 20)
	p := NewPath()
	p.MoveTo(5, 5)
	p.MoveTo(10, 10)
	cnv.StrokePath(p, StrokeStyle{Width: 4, Cap: LineCapRound}, white)
	if !inkBounds(img).Empty() {
		t.Fatal("a lone MoveTo drew something")
	}
}

// The gradient goes from the first color to the last along its line
func TestLinearGradient(t *testing.T) {
	cnv, img := newTestCanvas(100, 10)
	p := NewPath()
	p.Rect(0, 0, 100, 10)
	cnv.FillPathPaint(p, LinearGradient{X1: 0, Y1: 0, X2: 100, Y2: 0, Stops: []GradientStop{
		{0, color.RGBA{255, 0, 0, 255}}, {1, color.RGBA{0, 0, 255, 255}},
	}})
	left, mid, right := img.RGBAAt(0, 5), img.RGBAAt(50, 5), img.RGBAAt(99, 5)
	if left.R < 250 || right.B < 250 || mid.R < 110 || mid.R > 145 || mid.B < 110 || mid.B > 145 {
		t.Fatalf("left %v mid %v right %v", left, mid, right)
	}
}

// A stop to transparent fades the color out, not to black
func TestGradientToTransparent(t *testing.T) {
	c := mixRGBA(color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 0, 0}, 0.5)
	if c.R != 255 || c.A < 126 || c.A > 129 {
		t.Fatalf("mixed %v, want red half transparent", c)
	}
}

// The hatch draws stripes with gaps between them
func TestHatchPaint(t *testing.T) {
	cnv, img := newTestCanvas(30, 30)
	p := NewPath()
	p.Rect(0, 0, 30, 30)
	cnv.FillPathPaint(p, HatchPaint{Color: white, Spacing: 6, Width: 2})
	covered := inkSum(img) / (30 * 30)
	if covered < 0.25 || covered > 0.42 {
		t.Fatalf("a third of the area expected under the stripes, got %.2f", covered)
	}
}

// Every marker draws within its size around the center
func TestDrawMarker(t *testing.T) {
	for shape := MarkerCircle; shape <= MarkerStar; shape++ {
		cnv, img := newTestCanvas(40, 40)
		cnv.DrawMarker(shape, 20, 20, 10, white, nil)
		b := inkBounds(img)
		if b.Empty() {
			t.Errorf("marker %d drew nothing", shape)
			continue
		}
		if b.Min.X < 12 || b.Max.X > 28 || b.Min.Y < 12 || b.Max.Y > 28 {
			t.Errorf("marker %d out of its size: %v", shape, b)
		}
	}
}

// The text turned upwards is taller than wide, around its anchor
func TestDrawTextRotated(t *testing.T) {
	cnv, img := newTestCanvas(60, 120)
	cnv.SetColor(white)
	cnv.SetFontSize(14)
	cnv.SetHAlign(HAlignCenter)
	cnv.SetVAlign(VAlignCenter)
	cnv.DrawTextRotated(30, 60, -math.Pi/2, "Pressure")
	b := inkBounds(img)
	if b.Empty() || b.Dy() < b.Dx()*2 {
		t.Fatalf("ink %v: not turned", b)
	}
	if cx, cy := (b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2; math.Abs(float64(cx-30)) > 4 || math.Abs(float64(cy-60)) > 6 {
		t.Fatalf("ink centered at %d,%d, want about 30,60", cx, cy)
	}
	// Measured as drawn
	w, h := cnv.MeasureTextRotated("Pressure", -math.Pi/2)
	if math.Abs(h-float64(b.Dy())) > 6 || w < float64(b.Dx()) {
		t.Fatalf("measured %.0fx%.0f, ink %v", w, h, b)
	}
}

// Text turned by any angle is drawn too, smoothed
func TestDrawTextRotatedAngle(t *testing.T) {
	cnv, img := newTestCanvas(100, 100)
	cnv.SetColor(white)
	cnv.SetFontSize(14)
	cnv.DrawTextRotated(20, 20, math.Pi/4, "Diagonal")
	b := inkBounds(img)
	if b.Dx() < 30 || b.Dy() < 30 {
		t.Fatalf("ink %v: not along the diagonal", b)
	}
}

func BenchmarkStrokePolyline(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 1920, 400))
	cnv := NewCanvas(img)
	cnv.SetDirectTranslateAndClip(0, 0, 1920, 400)
	pts := make([]PointF, 1920)
	for i := range pts {
		pts[i] = PointF{float64(i), 200 + 150*math.Sin(float64(i)/30)}
	}
	b.ReportAllocs()
	for b.Loop() {
		cnv.DrawPolyline(pts, StrokeStyle{Width: 1.5, Join: LineJoinRound}, white)
	}
}

func BenchmarkFillCircleSmall(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	cnv := NewCanvas(img)
	cnv.SetDirectTranslateAndClip(0, 0, 200, 200)
	b.ReportAllocs()
	for b.Loop() {
		cnv.FillCircle(100, 100, 4, white)
	}
}

// A typical chart line: a noisy signal a pixel per point within a band
func BenchmarkStrokePolylineNoisy(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 1920, 400))
	cnv := NewCanvas(img)
	cnv.SetDirectTranslateAndClip(0, 0, 1920, 400)
	pts := make([]PointF, 1920)
	v := 200.0
	for i := range pts {
		v += math.Sin(float64(i)*12.9898)*4 + math.Sin(float64(i)/200)*0.5
		pts[i] = PointF{float64(i), v}
	}
	b.ReportAllocs()
	for b.Loop() {
		cnv.DrawPolyline(pts, StrokeStyle{Width: 1.5}, white)
	}
}
