package ui

import (
	"bytes"
	"fmt"
	"math"
	"testing"
	"time"
)

// Radar with an infinite value: polarFrameOf loops forever appending rings
func TestAuditRadarInf(t *testing.T) {
	ch := NewXYChart()
	r := ch.Pane(0).AddRadar("r")
	r.Set("a", 1).Set("b", math.Inf(1)).Set("c", 2)
	auditTry(t, "radar inf", func() { ch.Render(400, 300, 1, nil) })
}

func TestAuditPolarMaxFloat(t *testing.T) {
	ch := NewXYChart()
	ch.Pane(0).AddPolar("p").Add(0, 1).Add(90, math.MaxFloat64)
	auditTry(t, "polar maxfloat", func() { ch.Render(400, 300, 1, nil) })
}

func TestAuditSymlogHang(t *testing.T) {
	auditTry(t, "symlog", func() {
		s := NewSymlogScale(-math.MaxFloat64, -math.MaxFloat64)
		lo, hi := s.Domain()
		fmt.Println("symlog domain", lo, hi)
		s.Ticks(300, meas)
	})
	auditTry(t, "symlog chart", func() {
		ch := NewLineChart()
		ch.Series("a").SetValues([]float64{1, 2, 3})
		ch.Pane(0).YAxis().SetScale(NewSymlogScale(0, 1)).SetRange(math.MaxFloat64, math.MaxFloat64)
		ch.Render(400, 300, 1, nil)
	})
}

func TestAuditLinearWidenInf(t *testing.T) {
	s := NewLinearScale(math.MaxFloat64, math.MaxFloat64)
	lo, hi := s.Domain()
	t.Logf("linear domain of MaxFloat64: %v..%v, ToUnit(MaxFloat64)=%v", lo, hi, s.ToUnit(math.MaxFloat64))
}

func TestAuditFillGradientInf(t *testing.T) {
	ch := NewLineChart()
	ch.Series("a").SetFill(true, 0.3, true).SetValues([]float64{1, math.Inf(1), 3})
	auditTry(t, "gradient fill inf", func() { ch.Render(400, 300, 1, nil) })
	ch2 := NewLineChart()
	ch2.Series("a").SetFill(true, 0.3, false).SetValues([]float64{1, math.Inf(1), 3})
	auditTry(t, "plain fill inf", func() { ch2.Render(400, 300, 1, nil) })
	ch3 := NewLineChart()
	ch3.Series("a").SetFill(true, 0.3, false).SetValues([]float64{1e308, -1e308, 1.7e308})
	auditTry(t, "plain fill huge", func() { ch3.Render(400, 300, 1, nil) })
}

func TestAuditStackPercentTiny(t *testing.T) {
	ch := NewLineChart()
	ch.Series("a").SetStack("g").SetFill(true, 0.3, false).SetValues([]float64{1e-320, 2e-320})
	ch.Series("b").SetStack("g").SetValues([]float64{1e-320, 2e-320})
	ch.Pane(0).SetStackPercent("g", true)
	auditTry(t, "percent stack of denormals", func() { ch.Render(400, 300, 1, nil) })
}

func TestAuditPieStartInf(t *testing.T) {
	ch := NewPieChart()
	ch.Add("a", 1).Add("b", 2)
	ch.Pie().SetAngles(1e12, 2*math.Pi)
	ch.SetSize(400, 300)
	auditPaint(&ch.ChartView, 400, 300)
	auditTry(t, "pie hover with start 1e12", func() {
		ch.onMouseMove(200+30, 150+30, KeyModifiers{})
		ch.updateHits()
	})
}

func TestAuditCandleInverted(t *testing.T) {
	ch := NewXYChart()
	// high below low: a feed that swapped the fields
	ch.Pane(0).AddCandles("c").AddOHLC(0, 10, 9, 11, 10)
	ch.Render(400, 300, 1, nil)
	lo, hi := ch.Pane(0).YAxis().scale.Domain()
	t.Logf("y domain of a candle with high 9, low 11: %v..%v", lo, hi)
	if lo > 9 || hi < 11 {
		t.Errorf("candle not in the y range: %v..%v", lo, hi)
	}
	ch2 := NewXYChart()
	// open above high
	ch2.Pane(0).AddCandles("c").AddOHLC(0, 100, 50, 40, 45).AddOHLC(1, 45, 50, 40, 48)
	ch2.Render(400, 300, 1, nil)
	lo, hi = ch2.Pane(0).YAxis().scale.Domain()
	t.Logf("y domain with open 100 above high 50: %v..%v", lo, hi)
	if hi < 100 {
		t.Errorf("open 100 out of the y range: %v..%v", lo, hi)
	}
}

func TestAuditSaveStateNonFinite(t *testing.T) {
	ch := NewLineChart()
	ch.Series("a").SetValues([]float64{1, 2, 3})
	ch.SetSelection(0, math.Inf(1))
	ch.Pane(0).YAxis().SetRange(0, 10)
	st := ch.SaveState()
	t.Logf("SaveState = %q", st)
	if len(st) == 0 {
		t.Errorf("SaveState returned nothing: the zoom of y lost too")
	}
	if err := ch.RestoreState(st); err != nil {
		t.Errorf("RestoreState(SaveState()) = %v", err)
	}
}

func TestAuditTimeDownsampleFarDates(t *testing.T) {
	for _, year := range []int{1600, 2300, 3000} {
		base := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		var pts []TimeChartPoint
		for i := 0; i < 100; i++ {
			pts = append(pts, NewTimeChartValue(base.Add(time.Duration(i)*time.Minute), float64(i)))
		}
		res := TimeChartDownsample(pts, base, base.Add(100*time.Minute), 10*time.Minute)
		if len(res) == 0 {
			t.Errorf("year %d: nothing", year)
			continue
		}
		t.Logf("year %d: first bucket at %v (want %v)", year, res[0].DT.UTC(), base)
		if res[0].DT.Year() != year {
			t.Errorf("year %d: bucket at %v", year, res[0].DT.UTC())
		}
	}
}

func TestAuditTimeChartFarDates(t *testing.T) {
	ch := NewTimeChart()
	src := NewTimeChartMemorySource()
	base := time.Date(2300, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5000; i++ {
		src.AddPoint(NewTimeChartValue(base.Add(time.Duration(i)*time.Second), math.Sin(float64(i)/100)))
	}
	ch.AddArea().AddSeries("s", src)
	ch.SetTimeRange(base, base.Add(5000*time.Second))
	ch.Render(500, 300, 1, nil)
	ls := ch.areas[0].series[0].cs.(*LineSeries)
	n := 0
	for _, s := range ls.samples {
		if s.X >= ChartTime(base) && s.X <= ChartTime(base.Add(5000*time.Second)) {
			n++
		}
	}
	t.Logf("samples drawn %d, within the range %d", len(ls.samples), n)
	if n == 0 {
		t.Errorf("no sample drawn in the visible range of year 2300")
	}
}

func TestAuditBarSetQuadratic(t *testing.T) {
	for _, n := range []int{5000, 20000} {
		ch := NewBarChart()
		s := ch.Series("a")
		start := time.Now()
		for i := 0; i < n; i++ {
			s.Set(fmt.Sprint("c", i), float64(i))
		}
		t.Logf("%d categories by Set: %v", n, time.Since(start))
	}
}

func TestAuditPaneWeightNaN(t *testing.T) {
	ch := NewLineChart()
	ch.Series("a").SetValues([]float64{1, 2, 3})
	ch.AddPane().SetWeight(math.NaN()).AddLine("b").SetValues([]float64{1, 2})
	auditTry(t, "nan weight", func() { ch.Render(400, 300, 1, nil) })
	t.Logf("pane heights %v %v", ch.panes[0].h, ch.panes[1].h)
}

func TestAuditRenderSizes(t *testing.T) {
	for _, sz := range [][3]float64{{0, 0, 1}, {-10, -10, 1}, {100, 100, math.NaN()}, {100, 100, math.Inf(1)}, {100, 100, 1e-9}} {
		sz := sz
		ch := NewLineChart()
		ch.Series("a").SetValues([]float64{1, 2, 3})
		auditTry(t, fmt.Sprint("render ", sz), func() { ch.Render(int(sz[0]), int(sz[1]), sz[2], nil) })
		auditTry(t, fmt.Sprint("svg ", sz), func() { ch.WriteSVG(&bytes.Buffer{}, int(sz[0]), int(sz[1]), nil) })
	}
}
