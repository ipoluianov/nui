package ui

import (
	"image"
	"sync"
	"fmt"
	"runtime/debug"
	"strings"
	"image/color"
	"math"
	"testing"
	"time"
)

func auditRun(t *testing.T, name string, f func()) {
	t.Helper()
	done := make(chan any, 1)
	go func() {
		defer func() {
			r := recover()
			if r != nil {
				r = fmt.Sprintf("%v\n%s", r, auditStackHead(debug.Stack()))
			}
			done <- r
		}()
		f()
	}()
	select {
	case r := <-done:
		if r != nil {
			t.Errorf("%s: PANIC: %v", name, r)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("%s: HANG (>5s)", name)
	}
}

var auditWhite = color.RGBA{255, 255, 255, 255}

func TestAuditArcBoundsInf(t *testing.T) {
	auditRun(t, "Arc start=Inf Bounds", func() {
		p := NewPath()
		p.Arc(0, 0, 10, math.Inf(1), 0)
		p.Bounds()
	})
	auditRun(t, "Arc start=1e17 Bounds", func() {
		p := NewPath()
		p.Arc(0, 0, 10, 1e17, 1e17+16)
		p.Bounds()
	})
}

func TestAuditFillInfNaN(t *testing.T) {
	inf := math.Inf(1)
	nan := math.NaN()
	for _, eo := range []bool{false, true} {
		for _, pt := range []PointF{{inf, 10}, {-inf, 10}, {nan, 10}, {10, nan}, {10, inf}, {1e300, 10}, {1e30, 1e30}} {
			eo, pt := eo, pt
			auditRun(t, fmt.Sprintf("fill eo=%v pt=%v", eo, pt), func() {
				cnv, _ := newTestCanvas(50, 50)
				p := NewPath()
				p.Polygon(PointF{0, 0}, pt, PointF{0, 40}, PointF{30, 30})
				if eo {
					p.SetFillRule(FillRuleEvenOdd)
				}
				cnv.FillPath(p, auditWhite)
				cnv.StrokePath(p, StrokeStyle{Width: 3, Dash: []float64{2, 1}}, auditWhite)
				cnv.StrokePath(p, StrokeStyle{Width: 3, Join: LineJoinRound, Cap: LineCapRound}, auditWhite)
			})
		}
	}
}

func TestAuditBlur(t *testing.T) {
	for _, r := range []float64{math.Inf(1), math.NaN(), 1e7, 1e12, 1e300} {
		r := r
		auditRun(t, fmt.Sprintf("Blur r=%v", r), func() {
			cnv, _ := newTestCanvas(40, 40)
			cnv.FillRect(10, 10, 10, 10, auditWhite)
			cnv.Blur(0, 0, 40, 40, r)
		})
	}
}

func TestAuditShadow(t *testing.T) {
	inf, nan := math.Inf(1), math.NaN()
	cases := [][6]float64{
		{0, 0, 10, 10, 2, inf}, {0, 0, 10, 10, 2, nan}, {0, 0, 10, 10, 2, 1e12},
		{0, 0, inf, 10, 2, 3}, {0, 0, 1e300, 10, 2, 3}, {0, 0, 10, 1e300, 2, 3},
		{inf, 0, 10, 10, 2, 3}, {nan, 0, 10, 10, 2, 3}, {-1e300, 0, 10, 10, 2, 3},
		{0, 0, 10, 10, inf, 3}, {0, 0, 10, 10, nan, 3}, {0, 0, 10, 10, -5, 3},
		{0, 0, nan, 10, 2, 3},
	}
	for _, cs := range cases {
		cs := cs
		auditRun(t, fmt.Sprintf("Shadow %v", cs), func() {
			cnv, _ := newTestCanvas(40, 40)
			cnv.DrawShadow(cs[0], cs[1], cs[2], cs[3], cs[4], cs[5], auditWhite)
		})
	}
}

func TestAuditDrawLineHugeInt(t *testing.T) {
	auditRun(t, "DrawLine huge", func() {
		cnv, img := newTestCanvas(40, 40)
		cnv.DrawLine(-3000000000, -3000000000, 3000000000, 3000000000, 1, auditWhite)
		t.Logf("pixel(10,10)=%v pixel(10,30)=%v ink=%v", img.RGBAAt(10, 10), img.RGBAAt(10, 30), inkBounds(img))
	})
	auditRun(t, "CohenSutherland", func() {
		x1, y1, x2, y2, ok := CohenSutherland(-3000000000, -3000000000, 3000000000, 3000000000, 0, 0, 39, 39)
		t.Logf("CS -> %d %d %d %d %v", x1, y1, x2, y2, ok)
	})
}

func TestAuditTextXfHugeZoom(t *testing.T) {
	auditRun(t, "DrawText under ScaleBy(2^32)", func() {
		cnv, _ := newTestCanvas(40, 40)
		cnv.ScaleBy(1<<32, 1<<32)
		cnv.DrawText(0, 0, 1, 1, "A")
	})
}

func TestAuditClipPathHuge(t *testing.T) {
	for _, big := range []float64{1e300, math.Inf(1)} {
		big := big
		auditRun(t, fmt.Sprintf("ClipPath right=%v", big), func() {
			cnv, img := newTestCanvas(40, 40)
			cnv.Save()
			p := NewPath()
			p.Rect(0, 0, big, 30) // covers x>=0, y<30
			cnv.ClipPath(p)
			cnv.FillRect(0, 0, 40, 40, auditWhite)
			cnv.Restore()
			t.Logf("right=%v: pixel(5,5)=%v (want white) ink=%v", big, img.RGBAAt(5, 5), inkBounds(img))
		})
		auditRun(t, fmt.Sprintf("ClipPath left=-%v", big), func() {
			cnv, img := newTestCanvas(40, 40)
			cnv.Save()
			p := NewPath()
			p.Rect(-big, 0, big+30, 30)
			cnv.ClipPath(p)
			cnv.FillRect(0, 0, 40, 40, auditWhite)
			cnv.Restore()
			t.Logf("left=-%v: pixel(5,5)=%v (want white) ink=%v", big, img.RGBAAt(5, 5), inkBounds(img))
		})
		auditRun(t, fmt.Sprintf("FillPath right=%v", big), func() {
			cnv, img := newTestCanvas(40, 40)
			p := NewPath()
			p.Rect(0, 0, big, 30)
			cnv.FillPath(p, auditWhite)
			t.Logf("FillPath right=%v: pixel(5,5)=%v ink=%v", big, img.RGBAAt(5, 5), inkBounds(img))
		})
	}
}

func TestAuditPaintCacheInf(t *testing.T) {
	for _, r := range []RectF{{math.Inf(-1), math.Inf(-1), math.Inf(1), math.Inf(1)}, {-1e300, -1e300, 1e300, 1e300}, {0, 0, 40, 40}} {
		r := r
		auditRun(t, fmt.Sprintf("PaintCache %v", r), func() {
			cnv, _ := newTestCanvas(40, 40)
			pc := NewPaintCache()
			calls := 0
			paint := func(c *Canvas) { calls++ }
			pc.Draw(cnv, 0, 0, 40, 40, paint)
			pc.InvalidateRect(r)
			pc.Draw(cnv, 0, 0, 40, 40, paint)
			t.Logf("InvalidateRect(%v): paint calls=%d (want 2)", r, calls)
		})
	}
}

func TestAuditSVGInjection(t *testing.T) {
	rec := NewSVGRecorder(100, 100)
	cnv, _ := newTestCanvas(100, 100)
	cnv.SetRecorder(rec)
	cnv.SetFontFamily(`Evil" onload="alert(1)`)
	cnv.SetColor(auditWhite)
	cnv.DrawText(0, 0, 100, 20, "hi")
	s := rec.String()
	t.Logf("svg: %s", s)
	if err := xmlCheck(s); err != nil {
		t.Errorf("font family: invalid XML: %v", err)
	}
	rec2 := NewSVGRecorder(100, 100)
	cnv2, _ := newTestCanvas(100, 100)
	cnv2.SetRecorder(rec2)
	cnv2.SetColor(auditWhite)
	cnv2.DrawText(0, 0, 100, 20, "a\x01b\x00c\x1b")
	if err := xmlCheck(rec2.String()); err != nil {
		t.Errorf("control chars: invalid XML: %v", err)
	}
}

func TestAuditDashCorner(t *testing.T) {
	// A 20x20 square dashed 4 on, 4 off: a dash ends exactly at each corner
	for _, dash := range [][]float64{{4, 4}, {5, 5}, {10, 10}, {0, 4}} {
		for _, cp := range []LineCap{LineCapButt, LineCapRound} {
			cnv, img := newTestCanvas(40, 40)
			p := NewPath()
			p.Rect(10, 10, 20, 20)
			cnv.StrokePath(p, StrokeStyle{Width: 2, Dash: dash, Cap: cp}, auditWhite)
			t.Logf("dash=%v cap=%v: ink=%v sum=%.1f", dash, cp, inkBounds(img), inkSum(img))
		}
	}
	// open polyline, dash ending at the vertex
	cnv, img := newTestCanvas(40, 40)
	cnv.DrawPolyline([]PointF{{0, 10}, {10, 10}, {10, 30}}, StrokeStyle{Width: 2, Dash: []float64{10, 5}}, auditWhite)
	t.Logf("polyline dash {10,5}: ink=%v sum=%.1f", inkBounds(img), inkSum(img))
	cnv, img = newTestCanvas(40, 40)
	cnv.DrawPolyline([]PointF{{0, 10}, {10, 10}, {10, 30}}, StrokeStyle{Width: 2, Dash: []float64{9, 5}}, auditWhite)
	t.Logf("polyline dash {9,5}: ink=%v sum=%.1f", inkBounds(img), inkSum(img))
}

func TestAuditDashCornerRootCause(t *testing.T) {
	pts := []PointF{{0, 10}, {10, 10}, {10, 30}}
	dashPolyline(pts, []float64{10, 5}, 0, func(d []PointF) {
		t.Logf("dash: %v", d)
		pb := &polyBuilder{}
		strokePolyline(pb, d, false, StrokeStyle{Width: 2}, 1)
		for _, p := range pb.pts {
			if math.IsNaN(p.X) || math.IsNaN(p.Y) {
				t.Logf("  NaN point in stroke polygons: %v", pb.pts)
				break
			}
		}
	})
}

func TestAuditParallelImageCanvas(t *testing.T) {
	mip := NewMipImage(func() *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 64, 64))
		for i := range img.Pix {
			img.Pix[i] = uint8(i)
		}
		return img
	}())
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				cnv, _ := NewImageCanvas(64, 64, 1+float64(g%2))
				cnv.SetColor(auditWhite)
				cnv.FillRoundedRect(2, 2, 40, 40, 6)
				cnv.DrawRoundedRect(2, 2, 40, 40, 6)
				p := NewPath()
				p.Circle(30, 30, 20)
				cnv.FillPath(p, auditWhite)
				cnv.StrokePath(p, StrokeStyle{Width: 2, Dash: []float64{3, 2}}, auditWhite)
				cnv.FillPathPaint(p, LinearGradient{X1: 0, Y1: 0, X2: 60, Y2: 60, Stops: []GradientStop{{0, auditWhite}, {1, color.RGBA{0, 0, 255, 255}}}})
				cnv.BeginLayerStyle(LayerStyle{Blend: BlendMultiply, Opacity: 0.5})
				cnv.FillCircle(20, 20, 10, auditWhite)
				cnv.EndLayer()
				cnv.Save()
				cnv.ClipPath(p)
				cnv.DrawImageRect(0, 0, 16, 16, mip, mip.Bounds(), true)
				cnv.Restore()
				cnv.Blur(0, 0, 64, 64, 3)
				cnv.DrawShadow(5, 5, 30, 30, 4, 6, auditWhite)
				cnv.DrawText(0, 0, 64, 20, "Hello")
				cnv.Rotate(0.2)
				cnv.DrawText(0, 0, 64, 20, "Hello")
				cnv.DrawImageTransformed(mip, ScalingMatrix(0.1, 0.1), true)
				if i%5 == 0 {
					mip.Invalidate()
				}
			}
		}(g)
	}
	wg.Wait()
}

func auditStackHead(b []byte) string {
	lines := strings.Split(string(b), "\n")
	var out []string
	for _, l := range lines {
		if strings.Contains(l, "nui/ui.") && !strings.Contains(l, "auditRun") {
			out = append(out, strings.TrimSpace(l))
		}
		if len(out) >= 6 {
			break
		}
	}
	return strings.Join(out, " | ")
}
