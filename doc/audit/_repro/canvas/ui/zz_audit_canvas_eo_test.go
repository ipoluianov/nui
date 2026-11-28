package ui

import (
	"math"
	"math/rand"
	"testing"
)

// A big triangle in a wide canvas: rasterized in bands 64 pixels wide
func TestAuditEvenOddBands(t *testing.T) {
	c1, img1 := newTestCanvas(400, 300)
	c2, img2 := newTestCanvas(400, 300)
	p := NewPath()
	p.Polygon(PointF{0, 0}, PointF{400, 37.3}, PointF{10, 290})
	c1.FillPath(p, auditWhite)
	p.SetFillRule(FillRuleEvenOdd)
	c2.FillPath(p, auditWhite)
	worst, wx, wy := 0, 0, 0
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			d := int(img1.RGBAAt(x, y).R) - int(img2.RGBAAt(x, y).R)
			if d < 0 {
				d = -d
			}
			if d > worst {
				worst, wx, wy = d, x, y
			}
		}
	}
	p.SetFillRule(FillRuleNonZero)
	inside := 0
	for sy := 0; sy < 64; sy++ {
		for sx := 0; sx < 64; sx++ {
			if p.Contains(float64(wx)+(float64(sx)+0.5)/64, float64(wy)+(float64(sy)+0.5)/64) {
				inside++
			}
		}
	}
	t.Logf("true coverage at worst pixel: %.0f", float64(inside)/4096*255)
	t.Logf("bands: worst diff %d at (%d,%d): nonzero %d evenodd %d", worst, wx, wy, img1.RGBAAt(wx, wy).R, img2.RGBAAt(wx, wy).R)
	if worst > 8 {
		t.Errorf("even-odd differs from nonzero by %d at (%d,%d)", worst, wx, wy)
	}
}

// The edge of a widget's clip: a shape crossing the left edge of the clip
func TestAuditEvenOddClipEdge(t *testing.T) {
	var cols [2][40]uint8
	for k, eo := range []bool{false, true} {
		c, img := newTestCanvas(40, 40)
		c.SetDirectTranslateAndClip(10, 0, 30, 40)
		p := NewPath()
		// a flat slanted edge crossing x=10 (pixel column 10 of the image)
		p.Polygon(PointF{-10, 0}, PointF{20.5, 0}, PointF{-19.5, 40}, PointF{-10, 40})
		if eo {
			p.SetFillRule(FillRuleEvenOdd)
		}
		c.FillPath(p, auditWhite)
		for y := 0; y < 40; y++ {
			cols[k][y] = img.RGBAAt(10, y).R
		}
	}
	t.Logf("column 10, nonzero: %v", cols[0])
	t.Logf("column 10, evenodd: %v", cols[1])
	if cols[0] != cols[1] {
		for y := 0; y < 40; y++ {
			if d := int(cols[0][y]) - int(cols[1][y]); d > 8 || d < -8 {
				t.Errorf("row %d: nonzero %d, evenodd %d", y, cols[0][y], cols[1][y])
			}
		}
	}
}

// A simple (star-shaped, not self-intersecting) polygon fills the same by
// both rules: the even-odd rasterizer should match the nonzero one
func TestAuditEvenOddVsNonZero(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	worst := 0
	var worstPts []PointF
	worstX, worstY := 0, 0
	for i := 0; i < 3000; i++ {
		n := 3 + rng.Intn(12)
		cx, cy := rng.Float64()*60-10, rng.Float64()*60-10
		pts := make([]PointF, n)
		for k := 0; k < n; k++ {
			a := 2 * math.Pi * (float64(k) + rng.Float64()*0.8) / float64(n)
			r := 1 + rng.Float64()*40
			pts[k] = PointF{cx + r*math.Cos(a), cy + r*math.Sin(a)}
		}
		c1, img1 := newTestCanvas(48, 48)
		c2, img2 := newTestCanvas(48, 48)
		p := NewPath()
		p.Polygon(pts...)
		c1.FillPath(p, auditWhite)
		p.SetFillRule(FillRuleEvenOdd)
		c2.FillPath(p, auditWhite)
		for j := 0; j < len(img1.Pix); j += 4 {
			d := int(img1.Pix[j]) - int(img2.Pix[j])
			if d < 0 {
				d = -d
			}
			if d > worst {
				worst, worstPts = d, pts
				worstX, worstY = (j/4)%48, (j/4)/48
			}
		}
	}
	t.Logf("worst per-pixel difference nonzero vs evenodd: %d at (%d,%d), polygon %v", worst, worstX, worstY, worstPts)
	// Ground truth by supersampling the pixel
	p := NewPath()
	p.Polygon(worstPts...)
	inside := 0
	const ss = 64
	for sy := 0; sy < ss; sy++ {
		for sx := 0; sx < ss; sx++ {
			if p.Contains(float64(worstX)+(float64(sx)+0.5)/ss, float64(worstY)+(float64(sy)+0.5)/ss) {
				inside++
			}
		}
	}
	c1, img1 := newTestCanvas(48, 48)
	c1.FillPath(p, auditWhite)
	p.SetFillRule(FillRuleEvenOdd)
	c2, img2 := newTestCanvas(48, 48)
	c2.FillPath(p, auditWhite)
	t.Logf("pixel (%d,%d): true coverage %.0f, nonzero %d, evenodd %d", worstX, worstY,
		float64(inside)/ss/ss*255, img1.RGBAAt(worstX, worstY).R, img2.RGBAAt(worstX, worstY).R)
	if worst > 8 {
		t.Errorf("even-odd coverage differs from nonzero by %d on a simple polygon", worst)
	}
}
