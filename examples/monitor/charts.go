package monitor

import (
	"time"

	"github.com/ipoluianov/nui/ui"
)

// trafficMinWindow is the shortest range of the traffic chart: its bars are
// one minute long, so a 1 min window would show just one.
const trafficMinWindow = 30 * time.Minute

// charts are the time charts of the tabs; each selected host gets an area in
// every chart.
type charts struct {
	latency *ui.TimeChart
	jitter  *ui.TimeChart
	traffic *ui.TimeChart
}

func newCharts() *charts {
	return &charts{
		latency: ui.NewTimeChart(),
		jitter:  ui.NewTimeChart(),
		traffic: ui.NewTimeChart(),
	}
}

// show rebuilds the areas for the selected hosts.
func (c *charts) show(all, selected []*host) {
	c.latency.RemoveAllAreas()
	c.jitter.RemoveAllAreas()
	c.traffic.RemoveAllAreas()
	for _, h := range all {
		h.area = nil
	}

	for _, h := range selected {
		h.area = c.latency.AddArea()
		h.area.AddSeries(h.name+": ping, ms", h.ping).SetPaletteColor(0)
		h.area.AddSeries("Average 30 s", h.avg).SetPaletteColor(1)
		h.area.AddSeries("TCP connect, ms", h.tcp).SetPaletteColor(2)
		h.area.SetMarkers(h.markers)

		c.jitter.AddArea().AddSeries(h.name+": jitter, ms", h.jitter).SetPaletteColor(3)

		c.traffic.AddArea().AddSeries(h.name+": inbound per minute, Mbit/s", h.traffic).
			SetType(ui.TimeChartSeriesCandles)
	}
}

// setRange moves the default ranges to end at the last sample. A chart the
// user has panned or zoomed stays where it is until zoomed back.
func (c *charts) setRange(last time.Time, window time.Duration) {
	c.latency.SetDefaultTimeRange(last.Add(-window), last)
	c.jitter.SetDefaultTimeRange(last.Add(-window), last)
	c.traffic.SetDefaultTimeRange(last.Add(-max(window, trafficMinWindow)), last)
}
