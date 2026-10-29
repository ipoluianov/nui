package ui

import (
	"strconv"
	"strings"
)

// NewXYChart makes a chart of one pane with linear axes, zoomed with a
// drag of a rectangle and the wheel (ChartProfileScientific)
func NewXYChart() *ChartView {
	return NewChartView()
}

// LineChart is a chart of lines with methods to add the points straight to
// it: the series are made as they are named.
//
//	ch := ui.NewLineChart()
//	ch.AddValues(3, 5, 4, 7, 6, 9, 8, 10, 9, 12)
//	form.Panel().AddWidget(0, 0, ch)
//
//	ch.Series("Plan").AddXY(1, 10).AddXY(2, 12)
//	ch.Series("Fact").AddPoints([]ui.ChartPoint{{1, 9}, {2, 13}})
type LineChart struct {
	ChartView
}

// NewLineChart makes an empty line chart
func NewLineChart() *LineChart {
	c := &LineChart{}
	c.initChartView(NewLinearScale(0, 1))
	c.SetTypeName("LineChart")
	return c
}

// Series returns the line of the name, adding it when there is none
func (c *LineChart) Series(name string) *LineSeries {
	for _, p := range c.panes {
		for _, s := range p.series {
			if ls, ok := s.(*LineSeries); ok && ls.name == name {
				return ls
			}
		}
	}
	return c.panes[0].AddLine(name)
}

// defaultSeries returns the first line, made when there is none
func (c *LineChart) defaultSeries() *LineSeries {
	for _, s := range c.panes[0].series {
		if ls, ok := s.(*LineSeries); ok {
			return ls
		}
	}
	return c.panes[0].AddLine("")
}

// AddValues adds the values to the first line at x 0, 1, 2... after the last
func (c *LineChart) AddValues(values ...float64) *LineChart {
	c.defaultSeries().AddValues(values...)
	return c
}

// SetValues replaces the data of the first line with the values
func (c *LineChart) SetValues(values []float64) *LineChart {
	c.defaultSeries().SetValues(values)
	return c
}

// AddXY adds the point to the first line
func (c *LineChart) AddXY(x, y float64) *LineChart {
	c.defaultSeries().AddXY(x, y)
	return c
}

// AddPoints adds the points to the first line
func (c *LineChart) AddPoints(points []ChartPoint) *LineChart {
	c.defaultSeries().AddPoints(points)
	return c
}

// SetPoints replaces the data of the first line with the points
func (c *LineChart) SetPoints(points []ChartPoint) *LineChart {
	c.defaultSeries().SetPoints(points)
	return c
}

// Clear removes the points of all the lines
func (c *LineChart) Clear() *LineChart {
	for _, p := range c.panes {
		for _, s := range p.series {
			if ls, ok := s.(*LineSeries); ok && ls.memory != nil {
				ls.memory.Clear()
			}
		}
	}
	return c
}

// SetMaxPoints keeps the last n points of every line: for a live feed
func (c *LineChart) SetMaxPoints(n int) *LineChart {
	c.defaultSeries().SetMaxPoints(n)
	return c
}

// ProcessPropChange takes the props of a line chart: "values" ("3,5,4"),
// "title", "xtitle", "ytitle"
func (c *LineChart) ProcessPropChange(key string, value interface{}) {
	s, _ := value.(string)
	switch key {
	case "values":
		c.SetValues(parseChartNumbers(s))
	default:
		c.chartProp(key, s)
	}
}

// chartProp takes the props every chart has
func (c *ChartView) chartProp(key, value string) {
	switch key {
	case "title":
		c.SetTitle(value)
	case "xtitle":
		c.xAxis.SetTitle(value)
	case "ytitle":
		c.panes[0].YAxis().SetTitle(value)
	}
}

// parseChartNumbers parses "3, 5.5; 4|7" into numbers, skipping the rest
func parseChartNumbers(s string) []float64 {
	var res []float64
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == '|' || r == ' ' }) {
		if v, err := strconv.ParseFloat(f, 64); err == nil {
			res = append(res, v)
		}
	}
	return res
}

// parseChartItems parses "Linux:60|Windows:30" into names and values
func parseChartItems(s string) (names []string, values []float64) {
	for _, item := range splitXMLItems(s) {
		name, num, ok := strings.Cut(item, ":")
		if !ok {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
		if err != nil {
			continue
		}
		names = append(names, strings.TrimSpace(name))
		values = append(values, v)
	}
	return names, values
}

// BarChart is a chart of bars by category, with methods to add the values
// straight to it.
//
//	bar := ui.NewBarChart()
//	bar.Add("Jan", 120).Add("Feb", 95).Add("Mar", 140)
//	bar.Series("Plan").Set("Jan", 100).Set("Feb", 110)
type BarChart struct {
	ChartView
	horizontal bool
	stacked    bool
}

// NewBarChart makes an empty bar chart of vertical columns
func NewBarChart() *BarChart {
	c := &BarChart{}
	c.initChartView(NewCategoryScale())
	c.SetTypeName("BarChart")
	c.in.UseProfile(ChartProfileDashboard)
	c.xAxis.SetGrid(false, false)
	c.panes[0].YAxis().SetIncludeZero(true)
	return c
}

// categories returns the scale of the categories
func (c *BarChart) categories() *CategoryScale {
	if c.horizontal {
		cs, _ := c.panes[0].YAxis().scale.(*CategoryScale)
		return cs
	}
	cs, _ := c.xAxis.scale.(*CategoryScale)
	return cs
}

// Series returns the bars of the name, adding them when there are none
func (c *BarChart) Series(name string) *BarSeries {
	for _, s := range c.panes[0].series {
		if bs, ok := s.(*BarSeries); ok && bs.name == name {
			return bs
		}
	}
	s := c.panes[0].AddBars(name)
	s.horizontal = c.horizontal
	if c.stacked {
		s.stack = "stack"
	}
	return s
}

func (c *BarChart) defaultSeries() *BarSeries {
	for _, s := range c.panes[0].series {
		if bs, ok := s.(*BarSeries); ok {
			return bs
		}
	}
	return c.Series("")
}

// Add sets the value of the category of the first series, adding the
// category when there is none of the name
func (c *BarChart) Add(category string, value float64) *BarChart {
	c.defaultSeries().Set(category, value)
	return c
}

// SetCategories sets the categories, in their order
func (c *BarChart) SetCategories(categories ...string) *BarChart {
	if cs := c.categories(); cs != nil {
		cs.SetCategories(categories)
		c.Update()
	}
	return c
}

// SetHorizontal lays the bars on their side: the categories go down the
// left axis, the first one at the top
func (c *BarChart) SetHorizontal(horizontal bool) *BarChart {
	if horizontal == c.horizontal {
		return c
	}
	cats := c.categories()
	c.horizontal = horizontal
	y := c.panes[0].YAxis()
	if horizontal {
		c.xAxis.scale = NewLinearScale(0, 1)
		c.xAxis.SetIncludeZero(true).SetGrid(true, false)
		c.xAxis.margin = 0.05
		y.scale = cats
		y.SetInverted(true).SetGrid(false, false).SetIncludeZero(false)
		y.margin = 0
	} else {
		c.xAxis.scale = cats
		c.xAxis.SetIncludeZero(false).SetGrid(false, false)
		c.xAxis.margin = 0
		y.scale = NewLinearScale(0, 1)
		y.SetInverted(false).SetGrid(true, false).SetIncludeZero(true)
		y.margin = 0.05
	}
	for _, s := range c.panes[0].series {
		if bs, ok := s.(*BarSeries); ok {
			bs.horizontal = horizontal
		}
	}
	c.Update()
	return c
}

// SetStacked stands the bars of all the series on each other
func (c *BarChart) SetStacked(stacked bool) *BarChart {
	c.stacked = stacked
	for _, s := range c.panes[0].series {
		if bs, ok := s.(*BarSeries); ok {
			bs.stack = ""
			if stacked {
				bs.stack = "stack"
			}
		}
	}
	c.Update()
	return c
}

// ProcessPropChange takes the props of a bar chart: "items"
// ("Jan:120|Feb:95"), "horizontal", "stacked", "title", "xtitle", "ytitle"
func (c *BarChart) ProcessPropChange(key string, value interface{}) {
	s, _ := value.(string)
	switch key {
	case "items":
		names, values := parseChartItems(s)
		for i, n := range names {
			c.Add(n, values[i])
		}
	case "horizontal":
		c.SetHorizontal(s == "true" || s == "1")
	case "stacked":
		c.SetStacked(s == "true" || s == "1")
	default:
		c.chartProp(key, s)
	}
}

// PieChart is a pie (or a donut) with a method to add the slices straight
// to it.
//
//	pie := ui.NewPieChart()
//	pie.Add("Linux", 60).Add("Windows", 30).Add("macOS", 10)
type PieChart struct {
	ChartView
	pie *PieSeries
}

// NewPieChart makes an empty pie with the legend on the right
func NewPieChart() *PieChart {
	c := &PieChart{}
	c.initChartView(NewLinearScale(0, 1))
	c.SetTypeName("PieChart")
	c.in.UseProfile(ChartProfileDashboard)
	c.hoverMode = ChartHoverNearest
	c.pie = c.panes[0].AddPie("")
	return c
}

// Pie returns the pie series, for its settings
func (c *PieChart) Pie() *PieSeries { return c.pie }

// Add adds a slice of the value
func (c *PieChart) Add(name string, value float64) *PieChart {
	c.pie.Add(name, value)
	return c
}

// SetDonut makes a donut: the hole is the part of the radius, 0 - a pie
func (c *PieChart) SetDonut(hole float64) *PieChart {
	c.pie.SetInnerRadius(hole)
	return c
}

// Clear removes the slices
func (c *PieChart) Clear() *PieChart {
	c.pie.Clear()
	return c
}

// ProcessPropChange takes the props of a pie: "items"
// ("Linux:60|Windows:30"), "donut" (the hole: "0.5"), "title"
func (c *PieChart) ProcessPropChange(key string, value interface{}) {
	s, _ := value.(string)
	switch key {
	case "items":
		names, values := parseChartItems(s)
		for i, n := range names {
			c.Add(n, values[i])
		}
	case "donut":
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			c.SetDonut(v)
		}
	default:
		c.chartProp(key, s)
	}
}
