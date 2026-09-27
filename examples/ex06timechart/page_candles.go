package ex06timechart

import (
	"math"
	"math/rand"
	"time"

	"github.com/ipoluianov/nui/ui"
)

type PageCandles struct {
	ui.Widget

	chart *ui.TimeChart
}

func NewPageCandles() *PageCandles {
	var c PageCandles
	c.InitWidget()

	now := time.Now().Truncate(time.Minute)
	begin := now.Add(-48 * time.Hour)

	c.chart = ui.NewTimeChart()
	c.AddWidget(0, 0, c.chart)
	c.chart.AddArea().AddSeries("Price", priceSource(begin, now)).SetType(ui.TimeChartSeriesCandles)

	c.chart.SetDefaultTimeRange(now.Add(-2*time.Hour), now)

	return &c
}

// priceSource builds one-minute OHLC bars from a random walk; zooming out
// merges them into longer bars through the same downsampling.
func priceSource(begin, end time.Time) *ui.TimeChartMemorySource {
	var points []ui.TimeChartPoint
	price := 100.0
	for t := begin; !t.After(end); t = t.Add(time.Minute) {
		p := ui.TimeChartPoint{DT: t, First: price, High: price, Low: price, HasGood: true}
		for j := 0; j < 12; j++ {
			price += rand.Float64()*0.2 - 0.1
			p.High = math.Max(p.High, price)
			p.Low = math.Min(p.Low, price)
		}
		p.Last = price
		points = append(points, p)
	}
	src := ui.NewTimeChartMemorySource()
	src.SetPoints(points)
	return src
}
