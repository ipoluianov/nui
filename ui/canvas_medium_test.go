package ui

import (
	"encoding/xml"
	"io"
	"math"
	"strings"
	"testing"
)

// DrawShadow ignores a non-finite place, size, radius or blur, and a blur
// or a place far too large: the ints made of them used to wrap around and
// index out of the mask, or overflow the size check
func TestDrawShadowInvalidInput(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	cases := []struct {
		name                     string
		x, y, w, h, radius, blur float64
	}{
		{"blur +Inf", 0, 0, 10, 10, 2, inf},
		{"blur NaN", 0, 0, 10, 10, 2, nan},
		{"blur 1e12", 0, 0, 10, 10, 2, 1e12},
		{"x +Inf", inf, 0, 10, 10, 2, 3},
		{"x NaN", nan, 0, 10, 10, 2, 3},
		{"x -1e300", -1e300, 0, 10, 10, 2, 3},
		{"y 1e300", 0, 1e300, 10, 10, 2, 3},
		{"w +Inf", 0, 0, inf, 10, 2, 3},
		{"w 1e300", 0, 0, 1e300, 10, 2, 3},
		{"h NaN", 0, 0, 10, nan, 2, 3},
		{"radius NaN", 0, 0, 10, 10, nan, 3},
		{"radius +Inf", 0, 0, 10, 10, inf, 3},
	}
	for _, tc := range cases {
		noPanicNoHang(t, tc.name, func() {
			cnv, _ := newTestCanvas(40, 40)
			cnv.DrawShadow(tc.x, tc.y, tc.w, tc.h, tc.radius, tc.blur, white)
		})
	}
	// A NaN radius is a square corner: the shadow is still drawn
	cnv, img := newTestCanvas(40, 40)
	cnv.DrawShadow(10, 10, 20, 20, nan, 0, white)
	if got := img.RGBAAt(20, 20).R; got != 255 {
		t.Errorf("shadow of NaN radius: center = %d, want 255", got)
	}
}

// The box blur costs the same at any radius, and its sum does not overflow
// at a radius where 255*(2r+1) passes 2^32
func TestBoxBlurLargeRadius(t *testing.T) {
	// All 255: every average is 255, whatever the radius
	for _, r := range []int{1, 1e7, 1e9} {
		buf := []uint32{255, 255, 255}
		boxBlurH(buf, 3, 1, 1, r)
		boxBlurV(buf, 1, 3, 1, r)
		for i, v := range buf {
			if v != 255 {
				t.Errorf("r=%d: pixel %d = %d, want 255", r, i, v)
			}
		}
	}
	// The counted copies of the edges match adding them one by one
	row := []uint32{10, 200, 30, 255, 0, 90, 7}
	for _, r := range []int{1, 2, 3, 6, 7, 20} {
		buf := append([]uint32(nil), row...)
		boxBlurH(buf, len(row), 1, 1, r)
		for x := range row {
			var sum uint64
			for i := x - r; i <= x+r; i++ {
				sum += uint64(row[max(0, min(i, len(row)-1))])
			}
			want := uint32((sum + uint64(r)) / uint64(2*r+1))
			if buf[x] != want {
				t.Errorf("r=%d: pixel %d = %d, want %d", r, x, buf[x], want)
			}
		}
	}
	// A huge radius returns at once, a NaN one blurs nothing
	for _, radius := range []float64{1e7, 1e12, math.Inf(1), math.NaN()} {
		noPanicNoHang(t, "Blur", func() {
			cnv, img := newTestCanvas(40, 40)
			cnv.FillRect(0, 0, 40, 40, white)
			cnv.Blur(0, 0, 40, 40, radius)
			if got := img.RGBAAt(20, 20).R; got != 255 {
				t.Errorf("radius %v: an even picture blurred to %d, want 255", radius, got)
			}
		})
	}
}

// supersampled returns how much of the pixel (px, py) of the logical
// coordinates the path covers, 0..255, by 32x32 samples
func supersampled(p *Path, px, py float64) float64 {
	const n = 32
	in := 0
	for sy := 0; sy < n; sy++ {
		for sx := 0; sx < n; sx++ {
			if p.Contains(px+(float64(sx)+0.5)/n, py+(float64(sy)+0.5)/n) {
				in++
			}
		}
	}
	return float64(in) / (n * n) * 255
}

// The even-odd rasterizer gives the true coverage where a line crosses the
// left edge of the rectangle rasterized (the edge of a clip, of a band): the
// part of the line left of it used to be spread over the part within
func TestEvenOddCoverageAtRasterEdge(t *testing.T) {
	cases := []struct {
		name   string
		pts    []PointF
		size   int
		clipX  int // the left edge of the clip, in pixels
		region [4]int
	}{
		// A shape across the left edge of a widget's clip: the pixel (10, 20)
		// is 1/8 covered, the rasterizer gave twice that
		{"clip edge", []PointF{{-10, 0}, {20.5, 0}, {-19.5, 40}, {-10, 40}}, 40, 10, [4]int{10, 14, 20, 26}},
		// A simple polygon of a random test, its worst pixel (0, 18)
		{"random", []PointF{{27.614565353310567, 9.691168769701726}, {15.309674963022372, 24.877116875079686},
			{-6.714352974723685, 15.589699570194007}, {-24.890331709257502, 26.232232117700203},
			{-32.118599681855144, 12.60619293527441}, {-8.876332252193464, -2.3122110493568955},
			{-41.30051245003882, -15.224767398049309}, {-13.609576586693043, -23.734721326286706},
			{-3.0233432962031124, -25.12924941301089}, {-4.971231359451437, -7.933085699684432},
			{-3.701196770722205, -4.355452728636898}}, 48, 0, [4]int{0, 0, 8, 30}},
		// A wide triangle: rasterized in bands 64 pixels wide, its edges
		// cross the bands' left edges
		{"band 64", []PointF{{0, 0}, {400, 37.3}, {10, 290}}, 400, 0, [4]int{62, 0, 67, 30}},
		{"band 128", []PointF{{0, 0}, {400, 37.3}, {10, 290}}, 400, 0, [4]int{126, 0, 131, 30}},
	}
	for _, tc := range cases {
		cnv, img := newTestCanvas(tc.size, max(tc.size, 300))
		if tc.clipX != 0 {
			cnv.SetDirectTranslateAndClip(tc.clipX, 0, tc.size-tc.clipX, tc.size)
		}
		p := NewPath()
		p.Polygon(tc.pts...)
		p.SetFillRule(FillRuleEvenOdd)
		cnv.FillPath(p, white)
		worst, wx, wy, wwant := 0.0, 0, 0, 0.0
		for y := tc.region[1]; y < tc.region[3]; y++ {
			for x := tc.region[0]; x < tc.region[2]; x++ {
				want := supersampled(p, float64(x-tc.clipX), float64(y))
				if d := math.Abs(float64(img.RGBAAt(x, y).R) - want); d > worst {
					worst, wx, wy, wwant = d, x, y, want
				}
			}
		}
		if worst > 6 {
			t.Errorf("%s: pixel (%d,%d) = %d, want %.0f", tc.name, wx, wy, img.RGBAAt(wx, wy).R, wwant)
		}
	}
}

// A triangle and a wide line with points far out of the picture (2^25 and
// more) draw their visible part: the points went unclipped into the
// fixed-point rasterizer, which divided by zero
func TestFillPolygonPixelsHugeCoordinates(t *testing.T) {
	noPanicNoHang(t, "FillTriangle", func() {
		cnv, img := newTestCanvas(40, 40)
		cnv.FillTriangle(-1<<25, 0, 1<<25, 10, 0, 30, white)
		if got := img.RGBAAt(20, 15).R; got != 255 {
			t.Errorf("FillTriangle: pixel (20,15) = %d, want 255", got)
		}
	})
	noPanicNoHang(t, "DrawLine", func() {
		cnv, img := newTestCanvas(40, 40)
		cnv.DrawLine(-1<<26, 0, 1<<26, 17, 3, white)
		if got := img.RGBAAt(20, 8).R; got == 0 {
			t.Errorf("DrawLine: pixel (20,8) not drawn")
		}
	})
	noPanicNoHang(t, "fillPolygonPixels non-finite", func() {
		cnv, img := newTestCanvas(40, 40)
		cnv.fillPolygonPixels([][2]float64{{math.Inf(-1), 0}, {40, 0}, {40, 30}}, white)
		cnv.fillPolygonPixels([][2]float64{{math.NaN(), 0}, {40, 0}, {40, 30}}, white)
		_ = img
	})
}

// DrawText under a zoom of 2^32 draws the outlines: the size of the picture
// of the text, an int product, wrapped around to 0 and passed the check
func TestDrawTextHugeZoom(t *testing.T) {
	for _, k := range []float64{1 << 32, 1e300, math.Inf(1), math.NaN()} {
		noPanicNoHang(t, "DrawText", func() {
			cnv, _ := newTestCanvas(40, 40)
			cnv.ScaleBy(k, k)
			cnv.DrawText(0, 0, 1, 1, "A")
		})
	}
}

// ClipPath of a shape with huge or infinite bounds clips to the shape: the
// bounds converted to ints wrapped around and clipped everything away
func TestClipPathHugeBounds(t *testing.T) {
	for _, big := range []float64{1e300, math.Inf(1)} {
		for _, pts := range [][]PointF{
			{{0, 0}, {big, 0}, {big, 30}, {0, 30}},
			{{-big, 0}, {30, 0}, {30, 30}, {-big, 30}},
		} {
			cnv, img := newTestCanvas(40, 40)
			cnv.Save()
			p := NewPath()
			p.Polygon(pts...)
			cnv.ClipPath(p)
			cnv.FillRect(0, 0, 40, 40, white)
			cnv.Restore()
			if got := img.RGBAAt(5, 5).R; got != 255 {
				t.Errorf("%v: inside pixel = %d, want 255", pts, got)
			}
			if got := img.RGBAAt(5, 35).R; got != 0 {
				t.Errorf("%v: outside pixel = %d, want 0", pts, got)
			}
		}
	}
}

// The font family in an exported SVG is escaped: a quote in it used to end
// the attribute, and the rest of the name became attributes of the text
func TestSVGFontFamilyEscaped(t *testing.T) {
	for _, family := range []string{
		`Evil" onload="alert(1)`,
		`Evil"/><script>alert(1)</script><text a="`,
		`A & B <c> 'd' \e` + "\n",
	} {
		rec := NewSVGRecorder(100, 100)
		cnv, _ := newTestCanvas(100, 100)
		cnv.SetRecorder(rec)
		cnv.SetFontFamily(family)
		cnv.DrawText(0, 0, 100, 20, "hi")
		svg := rec.String()
		dec := xml.NewDecoder(strings.NewReader(svg))
		texts := 0
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("family %q: invalid XML: %v\n%s", family, err, svg)
			}
			el, ok := tok.(xml.StartElement)
			if !ok {
				continue
			}
			if el.Name.Local == "script" {
				t.Errorf("family %q: a script element injected:\n%s", family, svg)
			}
			if el.Name.Local != "text" {
				continue
			}
			texts++
			for _, a := range el.Attr {
				switch a.Name.Local {
				case "x", "y", "font-family", "font-size", "fill", "fill-opacity", "font-weight", "font-style":
				default:
					t.Errorf("family %q: attribute %s injected:\n%s", family, a.Name.Local, svg)
				}
				if a.Name.Local == "font-family" && (!strings.HasPrefix(a.Value, "'Evil") && !strings.HasPrefix(a.Value, "'A & B")) {
					t.Errorf("family %q: font-family = %q", family, a.Value)
				}
			}
		}
		if texts != 1 {
			t.Errorf("family %q: %d text elements, want 1:\n%s", family, texts, svg)
		}
	}
}
