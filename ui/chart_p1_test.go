package ui_test

import (
	"math"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// noise returns n deterministic values of a normal-like distribution
func noise(n int, mean, sd float64, seed int) []float64 {
	res := make([]float64, n)
	for i := range res {
		// The sum of three uniform values from a Weyl sequence
		u := 0.0
		for k := 0; k < 3; k++ {
			x := float64(i*3+k+seed*7919) * 0.6180339887498949
			u += x - math.Floor(x)
		}
		res[i] = mean + sd*(u-1.5)*2
	}
	return res
}

// grid2 shows the charts in a grid of two columns
func grid2(t *testing.T, w, h int, charts ...ui.Widgeter) *uitest.Window {
	t.Helper()
	form := ui.NewForm()
	form.SetSize(w, h)
	for i, c := range charts {
		form.Panel().AddWidget(i/2, i%2, c)
	}
	return uitest.Show(t, form)
}

func TestComputeBoxStats(t *testing.T) {
	b := ui.ComputeBoxStats([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 100, math.NaN()})
	if b.N != 10 || b.Median != 5.5 || b.Min != 1 || b.Max != 100 {
		t.Fatalf("%+v", b)
	}
	if len(b.Outliers) != 1 || b.Outliers[0] != 100 || b.HighWhisker != 9 {
		t.Fatalf("outliers %v, whisker %v", b.Outliers, b.HighWhisker)
	}
}

func TestHistogramBins(t *testing.T) {
	ch := ui.NewXYChart()
	h := ch.Pane(0).AddHistogram("h").SetValues(noise(1000, 50, 10, 1))
	showChart(t, ch, 400, 260)
	edges, counts := h.Bins()
	total := 0.0
	for _, c := range counts {
		total += c
	}
	if total != 1000 || len(edges) != len(counts)+1 || len(counts) < 8 || len(counts) > 60 {
		t.Fatalf("%d bins, total %v", len(counts), total)
	}
	h.SetMode(ui.ChartHistogramDensity)
	ch.Pane(0).YAxis().Range()
	_, counts = h.Bins()
	area := 0.0
	for i, c := range counts {
		area += c * (edges[i+1] - edges[i])
	}
	if math.Abs(area-1) > 1e-9 {
		t.Fatalf("density area %v", area)
	}
}

func TestStatisticalCharts(t *testing.T) {
	hist := ui.NewXYChart()
	hist.SetTitle("Histogram")
	hist.Pane(0).AddHistogram("Latency").SetValues(append(noise(800, 40, 8, 1), noise(300, 70, 5, 2)...))

	box := ui.NewXYChart()
	box.SetTitle("Box plot")
	b := box.Pane(0).AddBoxPlot("2025")
	b2 := box.Pane(0).AddBoxPlot("2026")
	for i, region := range []string{"North", "South", "East"} {
		b.Set(region, append(noise(60, 50+float64(i)*8, 10, i), 110))
		b2.Set(region, noise(60, 55+float64(i)*5, 8, i+5))
	}

	violin := ui.NewXYChart()
	violin.SetTitle("Violin")
	vs := violin.Pane(0).AddBoxPlot("Score").SetViolin(true)
	vs.Set("A", append(noise(200, 40, 8, 3), noise(100, 70, 4, 4)...))
	vs.Set("B", noise(300, 55, 12, 6))

	band := ui.NewXYChart()
	band.SetTitle("Forecast and errors")
	fc := band.Pane(0).AddBand("Forecast")
	eb := band.Pane(0).AddBand("Measured").SetErrorBars(true)
	for i := 0; i <= 20; i++ {
		x := float64(i)
		y := 20 + 8*math.Sin(x/4)
		fc.Add(x, y, y-1-x*0.2, y+1+x*0.2)
		if i%4 == 0 {
			eb.Add(x, y+math.Sin(x), y+math.Sin(x)-1.5, y+math.Sin(x)+1.5)
		}
	}
	w := grid2(t, 800, 520, hist, box, violin, band)
	w.MatchGolden("chart-statistics")
}

func TestHeatmaps(t *testing.T) {
	hm := ui.NewXYChart()
	hm.SetTitle("Load by hour and day")
	hm.XAxis().SetScale(ui.NewCategoryScale("0", "3", "6", "9", "12", "15", "18", "21"))
	hm.Pane(0).YAxis().SetScale(ui.NewCategoryScale("Mon", "Tue", "Wed", "Thu", "Fri"))
	grid := make([][]float64, 5)
	for d := range grid {
		grid[d] = make([]float64, 8)
		for h := range grid[d] {
			grid[d][h] = math.Round(50 + 40*math.Sin(float64(h)/2.5+float64(d)))
		}
	}
	hm.Pane(0).AddHeatmap("Load").SetGrid(grid, 0, 0, 0, 0).SetLabels(true)

	field := ui.NewXYChart()
	field.SetTitle("Field 200×200")
	big := make([][]float64, 200)
	for r := range big {
		big[r] = make([]float64, 200)
		for c := range big[r] {
			x, y := float64(c)/40-2.5, float64(r)/40-2.5
			big[r][c] = math.Sin(x*x+y) * math.Cos(y*1.5)
		}
	}
	field.Pane(0).AddHeatmap("f").SetGrid(big, -2.5, 2.5, -2.5, 2.5).SetColors(ui.CurrentChartTheme().Diverging, -1, 1)

	h2 := ui.NewXYChart()
	h2.SetTitle("2D histogram")
	hex := ui.NewXYChart()
	hex.SetTitle("Hexbin")
	var pts []ui.ChartPoint
	xs, ys := noise(4000, 0, 1, 7), noise(4000, 0, 1, 11)
	for i := range xs {
		pts = append(pts, ui.ChartPoint{X: xs[i] + ys[i]*0.5, Y: ys[i]})
	}
	h2.Pane(0).AddHistogram2D("n").SetBins(30, 20).SetPoints(pts)
	hex.Pane(0).AddHistogram2D("n").SetHex(9).SetPoints(pts)
	w := grid2(t, 800, 520, hm, field, h2, hex)
	w.MatchGolden("chart-heatmaps")
}

func TestBusinessCharts(t *testing.T) {
	wf := ui.NewXYChart()
	wf.SetTitle("Profit bridge")
	wf.Pane(0).AddWaterfall("Profit").AddStep("Revenue", 420).AddStep("Costs", -260).
		AddStep("Other", 35).AddStep("Taxes", -45).AddTotal("Profit")

	fun := ui.NewXYChart()
	fun.SetTitle("Funnel")
	fun.Pane(0).AddFunnel("Sales").Add("Visited", 12000).Add("Signed up", 3100).Add("Trial", 1400).Add("Paid", 420)

	par := ui.NewParetoChart()
	par.SetTitle("Defects")
	par.Add("Dents", 18).Add("Scratches", 42).Add("Color", 7).Add("Cracks", 11).Add("Other", 4)

	cc := ui.NewControlChart()
	cc.SetTitle("Shaft diameter")
	vals := noise(30, 10, 0.05, 9)
	vals[12] = 10.2
	for i := 20; i < 29; i++ {
		vals[i] += 0.03
	}
	cc.SetValues(vals)
	w := grid2(t, 800, 520, wf, fun, par, cc)
	w.MatchGolden("chart-business")

	rules := map[int]bool{}
	for _, v := range cc.Violations() {
		rules[v.Rule] = true
	}
	if !rules[1] {
		t.Fatalf("the point beyond 3σ is not found: %+v", cc.Violations())
	}
}

func TestIndicators(t *testing.T) {
	g := ui.NewGauge(0, 120)
	g.SetZones(ui.GaugeZonesOf(nil, 0, 80, 100, 120)).SetUnit(" °C").SetLabel("Oil").SetValue(86)
	n := ui.NewGauge(0, 100)
	n.SetZones(ui.GaugeZonesOf(nil, 0, 60, 85, 100)).SetNeedle(true).SetDecimals(0).SetUnit("%").SetValue(42)
	lvl := ui.NewLevelIndicator(0, 10)
	lvl.SetZones(ui.GaugeZonesOf(nil, 0, 7, 9, 10)).AddSetPoint(8, "High").AddSetPoint(2, "Low").SetUnit(" m").SetValue(6.4)
	kpi := ui.NewKPITile("Revenue")
	kpi.SetUnit(" k$").SetValue(1280, 1150).SetTrend(noise(30, 100, 20, 3))
	w := grid2(t, 640, 400, g, n, lvl, kpi)
	w.MatchGolden("chart-indicators")
}

func TestTimelines(t *testing.T) {
	gantt := ui.NewChartView()
	gantt.SetTitle("Release plan")
	d := func(day int) float64 { return ui.ChartTime(time.Date(2026, 3, day, 0, 0, 0, 0, time.UTC)) }
	gantt.XAxis().SetScale(ui.NewTimeScale(time.Unix(0, 0), time.Unix(1, 0)))
	gantt.XAxis().Scale().(*ui.TimeScale).Location = time.UTC
	g := gantt.Pane(0).AddSpans("Plan")
	g.Add(ui.ChartSpan{ID: "d", Row: "Design", Start: d(2), End: d(6), Label: "Design", Progress: 1})
	g.Add(ui.ChartSpan{ID: "b", Row: "Build", Start: d(6), End: d(15), Label: "Build", Progress: 0.4})
	g.Add(ui.ChartSpan{ID: "t", Row: "Test", Start: d(13), End: d(19), Label: "Test"})
	g.Add(ui.ChartSpan{ID: "r", Row: "Release", Start: d(20), End: d(20), Label: "Release"})
	g.AddDependency("d", "b").AddDependency("b", "t").AddDependency("t", "r")

	states := ui.NewChartView()
	states.SetTitle("Machines")
	states.XAxis().SetScale(ui.NewLinearScale(0, 1))
	st := states.Pane(0).AddSpans("State")
	th := ui.CurrentChartTheme()
	st.SetStateColor("Run", th.Good).SetStateColor("Idle", th.Warning).SetStateColor("Down", th.Critical)
	for m, name := range []string{"Press 1", "Press 2", "Lathe"} {
		x := 0.0
		for k := 0; k < 6; k++ {
			l := 1 + float64((m*3+k*5)%4)
			state := []string{"Run", "Idle", "Run", "Down", "Run", "Idle"}[(k+m)%6]
			st.Add(ui.ChartSpan{Row: name, Start: x, End: x + l, State: state})
			x += l
		}
	}

	sig := ui.NewChartView()
	sig.SetTitle("Signals and alarms")
	valve := sig.Pane(0).AddDigital("Valve open")
	pump := sig.Pane(0).AddDigital("Pump mode").SetBus(true)
	pump.SetStateName(0, "Off").SetStateName(1, "Low").SetStateName(2, "High")
	ev := sig.Pane(0).AddEvents("Alarms")
	for i := 0; i < 20; i++ {
		valve.AddState(float64(i*5), float64(i%2))
		pump.AddState(float64(i*5), float64((i/2)%3))
	}
	for _, e := range []ui.ChartEvent{
		{X: 12, Source: "Pump 2", Severity: ui.ChartSeverityWarning, Text: "Pressure high"},
		{X: 40, Source: "Pump 2", Severity: ui.ChartSeverityCritical, Text: "Overheat"},
		{X: 41, Source: "Pump 2", Severity: ui.ChartSeverityWarning, Text: "Vibration"},
		{X: 70, Source: "Tank", Severity: ui.ChartSeverityInfo, Text: "Refilled"},
	} {
		ev.Add(e)
	}

	stems := ui.NewXYChart()
	stems.SetTitle("Stems")
	s := stems.Pane(0).AddScatter("x[n]").SetStems(true, 0)
	for i := 0; i < 24; i++ {
		s.Add(float64(i), math.Sin(float64(i)/3)*math.Exp(-float64(i)/15))
	}
	w := grid2(t, 860, 560, gantt, states, sig, stems)
	w.MatchGolden("chart-timelines")
}

func TestPolarCharts(t *testing.T) {
	radar := ui.NewXYChart()
	radar.SetTitle("Radar")
	a := radar.Pane(0).AddRadar("Model A")
	b := radar.Pane(0).AddRadar("Model B")
	for i, c := range []string{"Speed", "Comfort", "Safety", "Price", "Range", "Style"} {
		a.Set(c, 4+float64((i*3)%6))
		b.Set(c, 3+float64((i*5+2)%7))
	}
	rose := ui.NewXYChart()
	rose.SetTitle("Wind rose")
	calm := rose.Pane(0).AddPolar("< 5 m/s").SetKind(ui.ChartPolarBars).SetStack("w")
	strong := rose.Pane(0).AddPolar("≥ 5 m/s").SetKind(ui.ChartPolarBars).SetStack("w")
	for k := 0; k < 16; k++ {
		ang := float64(k) * 22.5
		calm.Add(ang, 3+2*math.Cos(ang*math.Pi/180-0.6))
		strong.Add(ang, 1+1.5*math.Cos(ang*math.Pi/180-0.9)+1.5)
	}
	w := grid2(t, 760, 340, radar, rose)
	w.MatchGolden("chart-polar")
}

// Charts of a sync group share their range and their selection
func TestChartSyncGroup(t *testing.T) {
	a, b := ui.NewLineChart(), ui.NewLineChart()
	for i := 0; i <= 100; i++ {
		a.AddXY(float64(i), math.Sin(float64(i)/10))
		b.AddXY(float64(i), math.Cos(float64(i)/10))
	}
	ui.NewChartSyncGroup().Add(&a.ChartView, &b.ChartView)
	grid2(t, 800, 300, a, b)
	a.SetXRange(20, 40)
	if lo, hi := b.XRange(); lo != 20 || hi != 40 {
		t.Fatalf("b at %v..%v", lo, hi)
	}
	b.SetSelection(25, 30)
	if from, to, ok := a.Selection(); !ok || from != 25 || to != 30 {
		t.Fatalf("a's selection %v %v %v", ok, from, to)
	}
	a.ResetZoom()
	if lo, hi := b.XRange(); lo != 0 || hi != 100 {
		t.Fatalf("b after reset %v..%v", lo, hi)
	}
}

// The view is saved and restored: the zoom, the hidden series, the selection
func TestChartSaveRestoreState(t *testing.T) {
	ch := ui.NewLineChart()
	ch.Series("A").AddValues(1, 2, 3, 4)
	ch.Series("B").AddValues(4, 3, 2, 1)
	showChart(t, &ch.ChartView, 400, 260)
	ch.SetXRange(1, 2)
	ch.Series("B").SetVisible(false)
	ch.SetSelection(1.2, 1.5)
	state := ch.SaveState()

	other := ui.NewLineChart()
	other.Series("A").AddValues(1, 2, 3, 4)
	other.Series("B").AddValues(4, 3, 2, 1)
	if err := other.RestoreState(state); err != nil {
		t.Fatal(err)
	}
	if lo, hi := other.XRange(); lo != 1 || hi != 2 {
		t.Fatalf("range %v..%v", lo, hi)
	}
	if other.Series("B").Visible() || !other.Series("A").Visible() {
		t.Fatal("hidden series not restored")
	}
	if from, to, ok := other.Selection(); !ok || from != 1.2 || to != 1.5 {
		t.Fatal("selection not restored")
	}
	if other.RestoreState([]byte("{bad")) == nil {
		t.Fatal("bad JSON accepted")
	}
}
