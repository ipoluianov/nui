package ui

import (
	"math"
	"os"
	"runtime"
	"testing"
	"time"
)

// auditStackAfter dumps the goroutines after d and exits: for hangs
func auditStackAfter(d time.Duration) {
	go func() {
		time.Sleep(d)
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		os.Stderr.Write(buf[:n])
		os.Exit(3)
	}()
}

// Dense line with one far neighbour sample outside the view: drawColumns
// iterates once per pixel of the distance to it
func TestAuditHangFar(t *testing.T) {
	auditStackAfter(3 * time.Second)
	ch := NewLineChart()
	s := ch.Series("s")
	for i := 0; i <= 1000; i++ {
		s.AddXY(float64(i), float64(i%7))
	}
	s.AddXY(1e11, 0)
	ch.SetXRange(0, 1000)
	ch.Render(400, 300, 1, nil)
}

// Radar with an infinite value: the rings loop never ends and its slice
// grows without bound (run under ulimit -v to see the OOM)
func TestAuditRadarInfOOM(t *testing.T) {
	auditStackAfter(4 * time.Second)
	ch := NewXYChart()
	ch.Pane(0).AddRadar("r").Set("a", 1).Set("b", math.Inf(1)).Set("c", 2)
	ch.Render(400, 300, 1, nil)
}

// Symlog axis with a manual range at the float limit: the domain becomes
// infinite and Ticks loops forever appending ticks
func TestAuditSymlogOOM(t *testing.T) {
	auditStackAfter(4 * time.Second)
	ch := NewLineChart()
	ch.Series("a").SetValues([]float64{1, 2, 3})
	ch.Pane(0).YAxis().SetScale(NewSymlogScale(0, 1)).SetRange(math.MaxFloat64, math.MaxFloat64)
	ch.Render(400, 300, 1, nil)
}
