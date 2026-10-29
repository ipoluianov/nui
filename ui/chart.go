package ui

// ChartPoint is a point of a chart: X along the horizontal axis, Y along
// the vertical one
type ChartPoint struct {
	X float64
	Y float64
}

// Chart is a simple line chart of []ChartPoint: a ChartView with one line.
// The props "xlabel" and "ylabel" are the titles of the axes, "padding" the
// space around the plot; the ticks are picked to fit their labels.
//
// For more lines, bars, areas and the rest, see LineChart and ChartView.
type Chart struct {
	ChartView
	points []ChartPoint
	line   *LineSeries
}

func NewChart() *Chart {
	c := &Chart{}
	c.initChartView(NewLinearScale(0, 1))
	c.SetTypeName("Chart")
	c.line = c.panes[0].AddLine("")
	c.line.SetMarkers(ChartMarkersNever, MarkerCircle, 0)
	// The props are read on every paint, as they may be set before the
	// chart is in a form
	c.paintFunc = func(cnv *Canvas) {
		c.xAxis.title = c.GetPropString("xlabel", "")
		c.panes[0].YAxis().title = c.GetPropString("ylabel", "")
		c.paint(cnv)
	}
	return c
}

func (c *Chart) Data() []ChartPoint {
	return c.points
}

// SetData replaces the points; they are drawn in the order of X
func (c *Chart) SetData(points []ChartPoint) {
	c.points = points
	c.line.SetPoints(points)
	c.Update()
}
