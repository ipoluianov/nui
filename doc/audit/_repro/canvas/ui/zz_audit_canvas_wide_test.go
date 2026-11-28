package ui

import (
	"fmt"
	"testing"
)

func TestAuditWideLineHuge(t *testing.T) {
	for _, v := range []int{1 << 20, 1 << 24, 1 << 25, 1 << 26, 1 << 30} {
		v := v
		auditRun(t, fmt.Sprintf("DrawLine(-%d,0,%d,17,width 3)", v, v), func() {
			cnv, _ := newTestCanvas(40, 40)
			cnv.DrawLine(-v, 0, v, 17, 3, auditWhite)
		})
		auditRun(t, fmt.Sprintf("FillTriangle(-%d..)", v), func() {
			cnv, _ := newTestCanvas(40, 40)
			cnv.FillTriangle(-v, 0, v, 10, 0, 30, auditWhite)
		})
		auditRun(t, fmt.Sprintf("FillRoundedRect(-%d..)", v), func() {
			cnv, _ := newTestCanvas(40, 40)
			cnv.SetColor(auditWhite)
			cnv.FillRoundedRect(-v, 5, 2*v, 30, 8)
			cnv.DrawRoundedRect(-v, 5, 2*v, 30, 8)
		})
		auditRun(t, fmt.Sprintf("FillPolygon path (-%d..)", v), func() {
			cnv, _ := newTestCanvas(40, 40)
			cnv.FillPolygon([]PointF{{float64(-v), 0}, {float64(v), 10}, {0, 30}}, auditWhite)
		})
	}
}
