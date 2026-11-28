package ui

import (
	"math"
	"testing"
	"time"
)

func TestAuditLineNaNAutoscale(t *testing.T) {
	ch := NewLineChart()
	ch.AddValues(100, 101, math.NaN(), 102, 103)
	newTestForm(t, ch)
	ch.Render(400, 300, 1, nil)
	ch.laidOut = true
	lo, hi, _ := ch.Pane(0).YAxis().Range()
	t.Logf("y range with a NaN: %v..%v", lo, hi)
	if lo > 100 || hi < 103 {
		t.Errorf("y axis does not show the data 100..103: %v..%v", lo, hi)
	}
}

func TestAuditLineInfAutoscale(t *testing.T) {
	ch := NewLineChart()
	ch.AddValues(100, 101, math.Inf(1), 102, 103)
	newTestForm(t, ch)
	ch.Render(400, 300, 1, nil)
	ch.laidOut = true
	lo, hi, _ := ch.Pane(0).YAxis().Range()
	t.Logf("y range with an Inf: %v..%v", lo, hi)
	if lo > 100 || hi < 103 {
		t.Errorf("y axis does not show the data 100..103: %v..%v", lo, hi)
	}
}

func TestAuditLineFarNeighbourHang(t *testing.T) {
	for _, far := range []float64{1e7, 1e8, 1e11} {
		ch := NewLineChart()
		s := ch.Series("s")
		for i := 0; i <= 1000; i++ {
			s.AddXY(float64(i), math.Sin(float64(i)))
		}
		s.AddXY(far, 0)
		ch.SetXRange(0, 1000)
		start := time.Now()
		auditTry(t, "render", func() { ch.Render(400, 300, 1, nil) })
		t.Logf("far=%g render took %v n=%d last=%v", far, time.Since(start), len(s.samples), s.samples[len(s.samples)-1])
	}
}
