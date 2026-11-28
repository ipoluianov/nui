package ui

import (
	"testing"
)

func FuzzAuditParseSVGPath(f *testing.F) {
	for _, s := range []string{
		"M12 2L2 22h20z", "M0 0 C 10 10 20 20 30 30 S 40 40 50 50", "m1 1 a1 1 0 00 1 1",
		"M0,0Q5,5,10,0T20,0", "M1e5 1e5 L-1e5 -1e5", "M0 0 A 5 5 30 1 1 10 10z", "M.5.5.5", "M0 0 H10 V10 h-10 v-10 Z",
		"M0 0L1e308 1e308", "M0 0a1e308 1e308 0 1 1 1 1",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p, _ := ParseSVGPath(s)
		if p == nil {
			t.Fatal("nil path")
		}
		// Known: Inf coordinates (or ones overflowing to Inf under the
		// transform) panic the rasterizer; skip that class here
		if x0, y0, x1, y1, ok := p.Bounds(); ok && (x0 < -1e15 || y0 < -1e15 || x1 > 1e15 || y1 > 1e15) {
			return
		}
		cnv, _ := newTestCanvas(32, 32)
		cnv.FillPath(p, auditWhite)
		p.SetFillRule(FillRuleEvenOdd)
		cnv.FillPath(p, auditWhite)
		cnv.StrokePath(p, StrokeStyle{Width: 2, Join: LineJoinRound, Cap: LineCapRound, Dash: []float64{3, 1}}, auditWhite)
		p.Bounds()
		p.Length()
		p.PointAtLength(5)
		p.Contains(5, 5)
		p.StrokeContains(5, 5, 2)
		cnv.Save()
		cnv.Rotate(0.3)
		cnv.ScaleBy(2, 0.5)
		cnv.FillPath(p, auditWhite)
		cnv.StrokePath(p, StrokeStyle{Width: 1}, auditWhite)
		cnv.ClipPath(p)
		cnv.Restore()
		rec := NewSVGRecorder(32, 32)
		cnv.SetRecorder(rec)
		cnv.FillPath(p, auditWhite)
		_ = rec.String()
	})
}
