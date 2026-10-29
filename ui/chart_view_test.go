package ui_test

import (
	"math"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// showChart shows the chart alone in a form of the size
func showChart(t *testing.T, chart ui.Widgeter, w, h int) *uitest.Window {
	t.Helper()
	form := ui.NewForm()
	form.SetSize(w, h)
	form.Panel().AddWidget(0, 0, chart)
	return uitest.Show(t, form)
}

// A chart of ten points on a form: three lines of code
func TestLineChartTenPoints(t *testing.T) {
	ch := ui.NewLineChart()
	ch.AddValues(3, 5, 4, 7, 6, 9, 8, 10, 9, 12)
	w := showChart(t, ch, 480, 300)
	w.MatchGolden("chart-line-ten-points")
	lo, hi := ch.XRange()
	if lo != 0 || hi != 9 {
		t.Fatalf("x range %v..%v, want the data 0..9", lo, hi)
	}
}

func TestLineChartSeriesAndArea(t *testing.T) {
	ch := ui.NewLineChart()
	ch.SetTitle("Temperature")
	plan := ch.Series("Plan")
	plan.SetShape(ui.ChartLineSpline)
	fact := ch.Series("Fact").SetFill(true, 0.3, true)
	for i := 0; i < 24; i++ {
		x := float64(i)
		plan.AddXY(x, 20+5*math.Sin(x/4))
		fact.AddXY(x, 19+5*math.Sin(x/4+0.4)+math.Sin(x*1.7))
	}
	ch.XAxis().SetTitle("Hour")
	ch.Pane(0).YAxis().SetTitle("°C")
	ch.Pane(0).Annotations().AddHLine(24, "Limit").SetDash(4, 3)
	w := showChart(t, ch, 560, 340)
	w.MatchGolden("chart-line-series-area")
}

func TestBarChart(t *testing.T) {
	bar := ui.NewBarChart()
	bar.Series("2025").Set("Jan", 120).Set("Feb", 95).Set("Mar", 140).Set("Apr", 80)
	bar.Series("2026").Set("Jan", 130).Set("Feb", 110).Set("Mar", 125).Set("Apr", -20).SetLabels(true)
	w := showChart(t, bar, 480, 300)
	w.MatchGolden("chart-bar-grouped")

	bar.SetHorizontal(true)
	w.MatchGolden("chart-bar-horizontal")

	bar.SetHorizontal(false)
	bar.SetStacked(true)
	w.MatchGolden("chart-bar-stacked")
}

func TestPieChart(t *testing.T) {
	pie := ui.NewPieChart()
	pie.Add("Linux", 60).Add("Windows", 30).Add("macOS", 10).Add("Other", 4)
	w := showChart(t, pie, 420, 280)
	w.MatchGolden("chart-pie")
	pie.SetDonut(0.55)
	w.MatchGolden("chart-donut")
}

// A trading chart: candles over the volume, sharing a time axis
func TestCandlesAndVolume(t *testing.T) {
	ch := ui.NewChartView()
	ch.XAxis().SetScale(ui.NewTimeScale(time.Unix(0, 0), time.Unix(1, 0)))
	ts := ch.XAxis().Scale().(*ui.TimeScale)
	ts.Location = time.UTC
	price := ch.Pane(0)
	price.SetWeight(3)
	candles := price.AddCandles("BTC")
	volPane := ch.AddPane()
	vol := volPane.AddBars("Volume")
	vol.SetUpDownColors(true)
	start := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	p := 100.0
	for i := 0; i < 60; i++ {
		x := ui.ChartTime(start.Add(time.Duration(i) * time.Hour))
		open := p
		p += 3 * math.Sin(float64(i)*1.3)
		hi, lo := math.Max(open, p)+1.5, math.Min(open, p)-1.2
		candles.AddOHLC(x, open, hi, lo, p)
		vol.AddXY(x, 0)
	}
	// The volume bars colored by the candles: the same samples
	vol.SetSource(candlesVolume(candles))
	w := showChart(t, ch, 640, 400)
	w.MatchGolden("chart-candles-volume")
}

// candlesVolume is a source of a volume per candle, colored as it
func candlesVolume(c *ui.CandleSeries) ui.ChartSource {
	return ui.ChartSourceFunc(func(from, to, group float64) []ui.ChartSample {
		var res []ui.ChartSample
		for _, s := range c.Source().(*ui.ChartMemorySource).Samples() {
			v := math.Abs(s.Last-s.First)*10 + 5
			res = append(res, ui.ChartSample{X: s.X, First: s.First, Last: s.Last, High: v, Low: v, HasGood: true})
			res[len(res)-1].Last = s.Last
			res[len(res)-1].First = s.First
			// The bar's value is Last: keep the direction, give the height
			if s.Last >= s.First {
				res[len(res)-1].First, res[len(res)-1].Last = 0, v
			} else {
				res[len(res)-1].First, res[len(res)-1].Last = v+1, v
			}
		}
		return res
	})
}

// A TimeChart as the network monitor uses it: areas of lines with peaks
// marked, a span of bad data, a gap, candles in another area
func TestTimeChartMonitorLike(t *testing.T) {
	ch := ui.NewTimeChart()
	// The ticks and the dates of the zone of UTC, the same on any machine
	ch.XAxis().Scale().(*ui.TimeScale).Location = time.UTC
	base := time.Date(2026, 3, 1, 23, 50, 0, 0, time.UTC)
	ping, avg, traffic := ui.NewTimeChartMemorySource(), ui.NewTimeChartMemorySource(), ui.NewTimeChartMemorySource()
	for s := 0; s < 1200; s++ {
		dt := base.Add(time.Duration(s) * time.Second)
		switch {
		case s >= 400 && s < 460:
			ping.AddPoint(ui.NewTimeChartBad(dt))
			avg.AddPoint(ui.NewTimeChartBad(dt))
			continue
		case s == 800:
			ping.AddPoint(ui.NewTimeChartGap(dt))
			avg.AddPoint(ui.NewTimeChartGap(dt))
			continue
		}
		v := 20 + 8*math.Sin(float64(s)/60) + 6*math.Sin(float64(s)*1.37)
		if s%300 == 150 {
			v += 30
		}
		ping.AddPoint(ui.NewTimeChartValue(dt, v))
		avg.AddPoint(ui.NewTimeChartValue(dt, 20+8*math.Sin(float64(s)/60)))
	}
	for m := 0; m < 20; m++ {
		dt := base.Add(time.Duration(m) * time.Minute)
		o := 50 + 10*math.Sin(float64(m)/3)
		c := o + 6*math.Sin(float64(m)*2.1)
		traffic.AddPoint(ui.TimeChartPoint{DT: dt, First: o, Last: c, High: math.Max(o, c) + 3, Low: math.Min(o, c) - 3, HasGood: true})
	}
	area := ch.AddArea()
	area.AddSeries("host: ping, ms", ping).SetPaletteColor(0)
	area.AddSeries("Average 30 s", avg).SetPaletteColor(1)
	for s := 150; s < 1200; s += 300 {
		area.AddMarker(ui.TimeChartMarker{DT: base.Add(time.Duration(s) * time.Second), Value: 50, Text: "peak"})
	}
	ch.AddArea().AddSeries("inbound, Mbit/s", traffic).SetType(ui.TimeChartSeriesCandles)
	ch.SetDefaultTimeRange(base, base.Add(20*time.Minute))
	ch.SetSelection(base.Add(5*time.Minute), base.Add(8*time.Minute))
	w := showChart(t, ch, 700, 460)
	w.MatchGolden("chart-timechart-monitor")
	from, to := ch.TimeRange()
	if !from.Equal(base) || to.Sub(base) != 20*time.Minute {
		t.Fatalf("time range %v..%v", from, to)
	}
}
