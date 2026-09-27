package ex06timechart

import (
	"math"
	"time"

	"github.com/ipoluianov/nui/ui"
)

type PageJitter struct {
	ui.Widget

	chart     *ui.TimeChart
	lblStatus *ui.Label
}

func NewPageJitter() *PageJitter {
	var c PageJitter
	c.InitWidget()

	c.lblStatus = c.AddLabel(0, 0, "")
	c.lblStatus.SetXExpandable(true)

	now := time.Now().Truncate(time.Second)
	begin := now.Add(-48 * time.Hour)

	c.chart = ui.NewTimeChart()
	c.AddWidget(1, 0, c.chart)

	area := c.chart.AddArea()
	area.AddSeries("Jitter (mean = sine)", jitterSource(begin, now))
	area.AddSeries("Sine", sineSource(begin, now))

	c.chart.SetOnTimeRangeChanged(c.updateStatus)
	c.chart.SetOnSelectionChanged(c.updateStatus)
	c.chart.SetDefaultTimeRange(now.Add(-2*time.Hour), now)

	return &c
}

func (c *PageJitter) updateStatus() {
	from, to := c.chart.TimeRange()
	text := "Range: " + from.Format("2006-01-02 15:04:05") + " - " + to.Format("2006-01-02 15:04:05")
	if selFrom, selTo, ok := c.chart.Selection(); ok {
		text += "   Selection: " + selFrom.Format("15:04:05") + " - " + selTo.Format("15:04:05")
	}
	c.lblStatus.SetText(text)
}

// jitterSource flips between +1 and -1 around a slow sine on every sample.
// Averaging it per pixel would draw a clean sine; with High/Low the chart
// shows the band the signal actually jumps in.
func jitterSource(begin, end time.Time) *ui.TimeChartMemorySource {
	var points []ui.TimeChartPoint
	i := 0
	for t := begin; !t.After(end); t = t.Add(time.Second) {
		v := 5 * math.Sin(float64(t.Unix())/1800)
		if i%2 == 0 {
			v += 1
		} else {
			v -= 1
		}
		points = append(points, ui.NewTimeChartValue(t, v))
		i++
	}
	src := ui.NewTimeChartMemorySource()
	src.SetPoints(points)
	return src
}

func sineSource(begin, end time.Time) *ui.TimeChartMemorySource {
	var points []ui.TimeChartPoint
	for t := begin; !t.After(end); t = t.Add(10 * time.Second) {
		points = append(points, ui.NewTimeChartValue(t, 5*math.Sin(float64(t.Unix())/1800)-8))
	}
	src := ui.NewTimeChartMemorySource()
	src.SetPoints(points)
	return src
}
