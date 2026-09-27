package ex06timechart

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/ipoluianov/nui/ui"
)

const (
	pingInterval = time.Second
	pingHistory  = time.Hour
	pingWindow   = 5 * time.Minute
)

// pingServer is one simulated server with its own latency profile.
type pingServer struct {
	name      string
	base      float64 // typical latency, ms
	drift     float64 // amplitude of the slow drift, ms
	noise     float64 // standard deviation of the per-sample noise, ms
	spikeRate int     // one spike per spikeRate samples on average
	spikeMax  float64 // largest spike on top of the latency, ms

	outageRate int // one outage per outageRate samples on average
	outageLeft int // samples left in the current outage

	source    *ui.TimeChartMemorySource
	area      *ui.TimeChartArea
	lastValue float64
	lastOK    bool

	// Recent non-peak samples for the peak detector.
	window []float64

	// TCP connect time: a separate probe that never drops out, so its line
	// runs straight through the ping outages.
	tcpSource *ui.TimeChartMemorySource

	// Moving average over the last maWindow samples, peaks included.
	maSource *ui.TimeChartMemorySource
	maValues []float64
	maSum    float64
}

const maWindow = 30 // samples, i.e. 30 s at one ping per second

func (c *pingServer) movingAverage(v float64) float64 {
	c.maValues = append(c.maValues, v)
	c.maSum += v
	if len(c.maValues) > maWindow {
		c.maSum -= c.maValues[0]
		c.maValues = c.maValues[1:]
	}
	return c.maSum / float64(len(c.maValues))
}

const (
	peakWindow    = 60  // samples the baseline is computed over
	peakMinWindow = 20  // samples needed before detecting anything
	peakSigmas    = 5.0 // how far above the baseline a peak must be
	peakMinDelta  = 1.0 // ms; ignore tiny deviations on very stable links
)

// detectPeak reports whether v stands out from the recent baseline: more
// than peakSigmas standard deviations (and at least peakMinDelta ms) above
// the mean of the last non-peak samples. Peaks are kept out of the window so
// a burst of them does not raise the baseline.
func (c *pingServer) detectPeak(v float64) bool {
	isPeak := false
	if len(c.window) >= peakMinWindow {
		mean, std := meanStd(c.window)
		isPeak = v-mean > math.Max(peakSigmas*std, peakMinDelta)
	}
	if !isPeak {
		c.window = append(c.window, v)
		if len(c.window) > peakWindow {
			c.window = c.window[1:]
		}
	}
	return isPeak
}

func meanStd(values []float64) (mean, std float64) {
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	for _, v := range values {
		std += (v - mean) * (v - mean)
	}
	return mean, math.Sqrt(std / float64(len(values)))
}

// PagePing simulates a ping to several servers once a second and shows each
// one live in its own area. The page moves the chart's default range to the
// last pingWindow on every new sample; the chart shows it while it is in the
// default view, and panning or zooming leaves it until the user zooms all
// the way back.
type PagePing struct {
	ui.Widget

	chart     *ui.TimeChart
	lblStatus *ui.Label
	servers   []*pingServer

	lastSample time.Time
}

func NewPagePing() *PagePing {
	var c PagePing
	c.InitWidget()

	c.lblStatus = c.AddLabel(0, 0, "")
	c.lblStatus.SetXExpandable(true)

	c.servers = []*pingServer{
		{name: "Frankfurt", base: 25, drift: 4, noise: 1.5, spikeRate: 100, spikeMax: 200, outageRate: 1200},
		{name: "New York", base: 95, drift: 8, noise: 3, spikeRate: 50, spikeMax: 300, outageRate: 900},
		{name: "Singapore", base: 180, drift: 15, noise: 8, spikeRate: 30, spikeMax: 500, outageRate: 600},
		{name: "Local gateway", base: 2, drift: 0.3, noise: 0.3, spikeRate: 200, spikeMax: 20, outageRate: 2400},
	}

	c.chart = ui.NewTimeChart()
	for _, srv := range c.servers {
		srv.source = ui.NewTimeChartMemorySource()
		srv.maSource = ui.NewTimeChartMemorySource()
		srv.area = c.chart.AddArea()
		srv.area.AddSeries("Ping to "+srv.name+", ms", srv.source).SetPaletteColor(0)
		srv.area.AddSeries("Average 30 s", srv.maSource).SetPaletteColor(1)
		srv.tcpSource = ui.NewTimeChartMemorySource()
		srv.area.AddSeries("TCP connect, ms", srv.tcpSource).SetPaletteColor(2)
	}
	c.AddWidget(1, 0, c.chart)

	now := time.Now().Truncate(pingInterval)
	c.lastSample = now.Add(-pingHistory)
	c.generateUntil(now)

	c.chart.SetOnTimeRangeChanged(c.updateStatus)
	c.chart.SetDefaultTimeRange(c.lastSample.Add(-pingWindow), c.lastSample)

	c.AddTimer(250, c.onTimer)

	return &c
}

// generateUntil adds a sample for every interval up to t. The page timer
// stops while another tab is open, so this also fills in what was missed.
func (c *PagePing) generateUntil(t time.Time) {
	for next := c.lastSample.Add(pingInterval); !next.After(t); next = next.Add(pingInterval) {
		for _, srv := range c.servers {
			srv.tcpSource.AddPoint(ui.NewTimeChartValue(next, srv.simulateTCP(next)))
			v, ok := srv.simulate(next)
			srv.lastOK = ok
			if !ok {
				// No reply: bad quality for the ping and for its average.
				srv.source.AddPoint(ui.NewTimeChartBad(next))
				srv.maSource.AddPoint(ui.NewTimeChartBad(next))
				continue
			}
			srv.lastValue = v
			srv.source.AddPoint(ui.NewTimeChartValue(next, srv.lastValue))
			srv.maSource.AddPoint(ui.NewTimeChartValue(next, srv.movingAverage(srv.lastValue)))
			if srv.detectPeak(srv.lastValue) {
				srv.area.AddMarker(ui.TimeChartMarker{
					DT:    next,
					Value: srv.lastValue,
					Text:  fmt.Sprintf("peak %.0f ms", srv.lastValue),
				})
			}
		}
		c.lastSample = next
	}
}

// simulateTCP returns a TCP connect time: a handshake costs about one and a
// half round trips, with its own noise and no spikes or outages.
func (c *pingServer) simulateTCP(t time.Time) float64 {
	v := 1.5*(c.base+c.drift*math.Sin(float64(t.Unix())/600)) + math.Abs(rand.NormFloat64())*c.noise*2
	return math.Max(0.1, v)
}

// simulate returns a latency around the server's base that drifts slowly,
// with noise and a rare spike. Now and then the server stops answering for
// 10-90 s; ok is false for those samples.
func (c *pingServer) simulate(t time.Time) (v float64, ok bool) {
	if c.outageLeft == 0 && rand.Intn(c.outageRate) == 0 {
		c.outageLeft = 10 + rand.Intn(81)
	}
	if c.outageLeft > 0 {
		c.outageLeft--
		return 0, false
	}
	v = c.base + c.drift*math.Sin(float64(t.Unix())/600) + rand.NormFloat64()*c.noise
	if rand.Intn(c.spikeRate) == 0 {
		v += c.spikeMax * (0.3 + 0.7*rand.Float64())
	}
	return math.Max(0.1, v), true
}

func (c *PagePing) onTimer() {
	before := c.lastSample
	c.generateUntil(time.Now())
	if c.lastSample.Equal(before) {
		return
	}
	c.chart.SetDefaultTimeRange(c.lastSample.Add(-pingWindow), c.lastSample)
	c.updateStatus()
	c.Form().Update()
}

func (c *PagePing) updateStatus() {
	text := "Last ping:"
	for _, srv := range c.servers {
		if srv.lastOK {
			text += fmt.Sprintf(" %s %.1f ms,", srv.name, srv.lastValue)
		} else {
			text += fmt.Sprintf(" %s timeout,", srv.name)
		}
	}
	text = strings.TrimSuffix(text, ",") + "   "
	if c.chart.IsDefaultView() {
		text += "Live"
	} else {
		text += "Paused - zoom back (right drag left or double click) to return to live"
	}
	c.lblStatus.SetText(text)
}
