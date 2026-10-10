package monitor

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/ipoluianov/nui/ui"
)

const (
	sampleInterval = time.Second
	historyLength  = time.Hour
	maWindow       = 30  // samples in the moving average, i.e. 30 s
	lossWindow     = 300 // samples the Loss % column is computed over, 5 min
	maxReplies     = int(historyLength / sampleInterval)
)

// profile is a kind of link with its typical behavior. The dialog scales it
// to the base latency the user enters.
type profile struct {
	name       string
	base       float64 // typical latency, ms
	drift      float64 // amplitude of the slow drift, ms
	noise      float64 // standard deviation of the per-sample noise, ms
	spikeRate  int     // one spike per spikeRate samples on average
	spikeMax   float64 // largest spike on top of the latency, ms
	outageRate int     // one outage per outageRate samples on average
	bandwidth  float64 // average traffic, Mbit/s
}

var profiles = []profile{
	{name: "LAN", base: 2, drift: 0.3, noise: 0.3, spikeRate: 200, spikeMax: 20, outageRate: 5000, bandwidth: 400},
	{name: "Europe", base: 25, drift: 4, noise: 1.5, spikeRate: 100, spikeMax: 200, outageRate: 3000, bandwidth: 80},
	{name: "Overseas", base: 120, drift: 10, noise: 5, spikeRate: 40, spikeMax: 400, outageRate: 1500, bandwidth: 25},
}

// host is one monitored server: its simulated link and the data the charts
// show. The sources keep the whole history, also while the host is not shown.
type host struct {
	name string
	profile
	phase float64 // shifts the drift so hosts of a profile differ

	outageLeft int // samples left in the current outage

	lastOK   bool
	lastPing float64
	prevPing float64 // previous reply, for the jitter; 0 - none
	replies  []bool  // whether each of the recent samples got a reply

	ping, avg, tcp, jitter, traffic *ui.TimeChartMemorySource

	maValues []float64
	maSum    float64

	// Recent non-peak samples for the peak detector.
	window []float64

	markers []ui.TimeChartMarker
	area    *ui.TimeChartArea // latency area while the host is shown, for new markers

	minute      ui.TimeChartPoint // traffic of the current minute, not added yet
	trafficFrom time.Time         // the first full minute after a pause
}

// sampleResult tells the monitor what happened to the host on a sample.
type sampleResult struct {
	wentDown  bool
	recovered bool
	spike     bool // a peak over alertThreshold
	downFor   time.Duration
}

func newHost(name string, base float64, p profile) *host {
	// Drift, noise and spikes grow with the latency.
	k := base / p.base
	p.base = base
	p.drift *= k
	p.noise *= k
	p.spikeMax *= k
	return &host{
		name:    name,
		profile: p,
		phase:   rand.Float64() * 2 * math.Pi,
		lastOK:  true,
		ping:    ui.NewTimeChartMemorySource(),
		avg:     ui.NewTimeChartMemorySource(),
		tcp:     ui.NewTimeChartMemorySource(),
		jitter:  ui.NewTimeChartMemorySource(),
		traffic: ui.NewTimeChartMemorySource(),
	}
}

// slowThreshold is the latency the host is shown as slow above.
func (h *host) slowThreshold() float64 { return 1.5*h.base + 10 }

// alertThreshold is the size of a peak that raises an alert.
func (h *host) alertThreshold() float64 { return 2*h.base + 150 }

// sample takes one measurement at t and adds it to the sources.
func (h *host) sample(t time.Time) sampleResult {
	var res sampleResult
	h.addTraffic(t)
	h.tcp.AddPoint(ui.NewTimeChartValue(t, h.simulateTCP(t)))

	v, ok := h.simulate(t)
	if ok != h.lastOK {
		res.wentDown = !ok
		res.recovered = ok
		if ok {
			res.downFor = time.Duration(h.downSamples()) * sampleInterval
		}
	}
	h.lastOK = ok
	h.replies = append(h.replies, ok)
	if len(h.replies) > maxReplies {
		h.replies = h.replies[1:]
	}
	if !ok {
		// No reply: bad quality for the ping, its average and the jitter.
		h.ping.AddPoint(ui.NewTimeChartBad(t))
		h.avg.AddPoint(ui.NewTimeChartBad(t))
		h.jitter.AddPoint(ui.NewTimeChartBad(t))
		h.prevPing = 0
		return res
	}

	h.lastPing = v
	h.ping.AddPoint(ui.NewTimeChartValue(t, v))
	h.avg.AddPoint(ui.NewTimeChartValue(t, h.movingAverage(v)))
	if h.prevPing > 0 {
		h.jitter.AddPoint(ui.NewTimeChartValue(t, math.Abs(v-h.prevPing)))
	}
	h.prevPing = v

	if h.detectPeak(v) {
		m := ui.TimeChartMarker{DT: t, Value: v, Text: fmt.Sprintf("peak %.0f ms", v)}
		h.markers = append(h.markers, m)
		if h.area != nil {
			h.area.AddMarker(m)
		}
		res.spike = v > h.alertThreshold()
	}
	return res
}

// addGap breaks the lines where sampling was paused.
func (h *host) addGap(t time.Time) {
	h.flushMinute()
	for _, src := range []*ui.TimeChartMemorySource{h.ping, h.avg, h.tcp, h.jitter, h.traffic} {
		src.AddPoint(ui.NewTimeChartGap(t))
	}
	h.trafficFrom = t.Truncate(time.Minute).Add(time.Minute)
	h.maValues, h.maSum, h.prevPing = nil, 0, 0
}

// downSamples counts the failed samples at the end of the history before the
// current one.
func (h *host) downSamples() int {
	n := 0
	for i := len(h.replies) - 1; i >= 0 && !h.replies[i]; i-- {
		n++
	}
	return n
}

// lossPercent is the share of samples without a reply among the last n.
func (h *host) lossPercent(n int) float64 {
	if n > len(h.replies) {
		n = len(h.replies)
	}
	if n == 0 {
		return 0
	}
	lost := 0
	for _, ok := range h.replies[len(h.replies)-n:] {
		if !ok {
			lost++
		}
	}
	return 100 * float64(lost) / float64(n)
}

// average is the current 30 s moving average, 0 without data.
func (h *host) average() float64 {
	if len(h.maValues) == 0 {
		return 0
	}
	return h.maSum / float64(len(h.maValues))
}

func (h *host) movingAverage(v float64) float64 {
	h.maValues = append(h.maValues, v)
	h.maSum += v
	if len(h.maValues) > maWindow {
		h.maSum -= h.maValues[0]
		h.maValues = h.maValues[1:]
	}
	return h.average()
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
func (h *host) detectPeak(v float64) bool {
	isPeak := false
	if len(h.window) >= peakMinWindow {
		mean, std := meanStd(h.window)
		isPeak = v-mean > math.Max(peakSigmas*std, peakMinDelta)
	}
	if !isPeak {
		h.window = append(h.window, v)
		if len(h.window) > peakWindow {
			h.window = h.window[1:]
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

func (h *host) driftAt(t time.Time) float64 {
	return h.drift * math.Sin(float64(t.Unix())/600+h.phase)
}

// simulate returns a latency around the host's base that drifts slowly,
// with noise and a rare spike. Now and then the host stops answering for
// 10-90 s; ok is false for those samples.
func (h *host) simulate(t time.Time) (v float64, ok bool) {
	if h.outageLeft == 0 && rand.Intn(h.outageRate) == 0 {
		h.outageLeft = 10 + rand.Intn(81)
	}
	if h.outageLeft > 0 {
		h.outageLeft--
		return 0, false
	}
	v = h.base + h.driftAt(t) + rand.NormFloat64()*h.noise
	if rand.Intn(h.spikeRate) == 0 {
		v += h.spikeMax * (0.3 + 0.7*rand.Float64())
	}
	return math.Max(0.1, v), true
}

// simulateTCP returns a TCP connect time: a handshake costs about one and a
// half round trips, with its own noise and no spikes or outages.
func (h *host) simulateTCP(t time.Time) float64 {
	v := 1.5*(h.base+h.driftAt(t)) + math.Abs(rand.NormFloat64())*h.noise*2
	return math.Max(0.1, v)
}

// addTraffic adds the bandwidth of one second to the bar of its minute; a
// finished minute goes to the source as one OHLC point.
func (h *host) addTraffic(t time.Time) {
	minute := t.Truncate(time.Minute)
	if minute.Before(h.trafficFrom) {
		return
	}
	if !h.minute.DT.Equal(minute) {
		h.flushMinute()
		h.minute = ui.TimeChartPoint{DT: minute}
	}
	v := 0.0
	if h.outageLeft == 0 {
		// Busy during the "day" of a 30 min cycle, with bursts.
		load := 0.6 + 0.4*math.Sin(float64(t.Unix())/300+h.phase)
		v = h.bandwidth * load * (0.8 + 0.4*rand.Float64())
		if rand.Intn(120) == 0 {
			v *= 2.5
		}
	}
	p := &h.minute
	if !p.HasGood {
		p.First, p.High, p.Low, p.HasGood = v, v, v, true
	}
	p.High = math.Max(p.High, v)
	p.Low = math.Min(p.Low, v)
	p.Last = v
}

func (h *host) flushMinute() {
	if h.minute.HasGood {
		h.traffic.AddPoint(h.minute)
	}
	h.minute = ui.TimeChartPoint{}
}
