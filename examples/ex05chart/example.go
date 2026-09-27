package ex05chart

import (
	"math"
	"math/rand"

	"github.com/ipoluianov/nui/ui"
)

type Content struct {
	ui.Widget

	chart *ui.Chart

	live      bool
	liveX     float64
	liveValue float64
}

func NewContent() *Content {
	var c Content
	c.InitWidget()

	buttons := c.AddPanel(0, 0)
	buttons.AddButton(0, 0, "Sine", c.showSine)
	buttons.AddButton(0, 1, "Parabola", c.showParabola)
	buttons.AddButton(0, 2, "Random Walk", c.showRandomWalk)
	buttons.AddButton(0, 3, "Live", c.startLive)
	buttons.AddButton(0, 4, "Stop", func() { c.live = false })
	buttons.AddHSpacer(0, 5)

	c.chart = ui.NewChart()
	c.chart.SetProp("xlabel", "X")
	c.chart.SetProp("ylabel", "Y")
	c.chart.SetProp("tickcount", 6)
	c.AddWidget(1, 0, c.chart)

	c.showSine()

	c.AddTimer(100, c.onLiveTick)

	return &c
}

func (c *Content) showSine() {
	c.live = false
	points := make([]ui.ChartPoint, 0, 200)
	for i := 0; i < 200; i++ {
		x := float64(i) / 10
		points = append(points, ui.ChartPoint{X: x, Y: math.Sin(x)})
	}
	c.chart.SetData(points)
}

func (c *Content) showParabola() {
	c.live = false
	points := make([]ui.ChartPoint, 0, 101)
	for i := -50; i <= 50; i++ {
		x := float64(i)
		points = append(points, ui.ChartPoint{X: x, Y: x * x})
	}
	c.chart.SetData(points)
}

func (c *Content) showRandomWalk() {
	c.live = false
	points := make([]ui.ChartPoint, 0, 500)
	y := 0.0
	for i := 0; i < 500; i++ {
		y += rand.Float64()*2 - 1
		points = append(points, ui.ChartPoint{X: float64(i), Y: y})
	}
	c.chart.SetData(points)
}

// startLive clears the chart and lets onLiveTick append a new random-walk
// point every 100ms, keeping only the last 100 points on screen.
func (c *Content) startLive() {
	c.liveX = 0
	c.liveValue = 0
	c.chart.SetData(nil)
	c.live = true
}

func (c *Content) onLiveTick() {
	if !c.live {
		return
	}
	c.liveValue += rand.Float64()*2 - 1
	points := append(c.chart.Data(), ui.ChartPoint{X: c.liveX, Y: c.liveValue})
	if len(points) > 100 {
		points = points[len(points)-100:]
	}
	c.liveX++
	c.chart.SetData(points)
}

func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Example 05 - Chart")
	form.SetSize(800, 500)
	form.Panel().AddWidget(0, 0, NewContent())
	return form
}
