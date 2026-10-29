package ui

import (
	"math"
	"sort"
	"strconv"
)

// ParetoChart shows the causes of a problem by how much each one causes:
// the bars sorted from the biggest, and the line of their running total in
// percent on the right axis, with the 80% mark, so the few causes that make
// most of it stand out.
//
//	p := ui.NewParetoChart()
//	p.Add("Scratches", 42).Add("Dents", 18).Add("Wrong color", 7)
type ParetoChart struct {
	ChartView
	items []paretoItem
	bars  *BarSeries
	cum   *LineSeries
}

type paretoItem struct {
	name  string
	value float64
}

// NewParetoChart makes an empty Pareto chart
func NewParetoChart() *ParetoChart {
	c := &ParetoChart{}
	c.initChartView(NewCategoryScale())
	c.SetTypeName("ParetoChart")
	c.in.UseProfile(ChartProfileDashboard)
	c.xAxis.SetGrid(false, false)
	p := c.panes[0]
	p.YAxis().SetIncludeZero(true)
	pct := p.AddAxis("pct", ChartAxisRight, &LinearScale{Unit: "%"})
	pct.SetRange(0, 105)
	c.bars = p.AddBars("Count")
	c.cum = p.AddLine("Cumulative")
	c.cum.SetAxisNames("x", "pct")
	c.cum.SetMarkers(ChartMarkersAlways, MarkerCircle, 6)
	p.Annotations().AddHLine(80, "80%").SetAxis("pct").SetDash(4, 3)
	return c
}

// Add adds a cause and how much it causes; the bars are sorted anew
func (c *ParetoChart) Add(name string, value float64) *ParetoChart {
	for i := range c.items {
		if c.items[i].name == name {
			c.items[i].value = value
			c.rebuild()
			return c
		}
	}
	c.items = append(c.items, paretoItem{name, value})
	c.rebuild()
	return c
}

// rebuild sorts the causes and fills the bars and the running total
func (c *ParetoChart) rebuild() {
	sort.SliceStable(c.items, func(i, j int) bool { return c.items[i].value > c.items[j].value })
	names := make([]string, len(c.items))
	values := make([]float64, len(c.items))
	total := 0.0
	for i, it := range c.items {
		names[i], values[i] = it.name, it.value
		total += math.Max(it.value, 0)
	}
	c.xAxis.scale.(*CategoryScale).SetCategories(names)
	c.bars.SetValues(values)
	pts := make([]ChartPoint, len(values))
	run := 0.0
	for i, v := range values {
		run += math.Max(v, 0)
		pct := 0.0
		if total > 0 {
			pct = run / total * 100
		}
		pts[i] = ChartPoint{X: float64(i), Y: pct}
	}
	c.cum.SetPoints(pts)
	c.Update()
}

// Bars returns the bars, Line the running total, for their settings
func (c *ParetoChart) Bars() *BarSeries  { return c.bars }
func (c *ParetoChart) Line() *LineSeries { return c.cum }

// ControlViolation is a point of a control chart out of control: its index
// and the Western Electric rule it breaks (1: beyond 3σ; 2: two of three
// beyond 2σ on a side; 3: four of five beyond 1σ on a side; 4: eight in a
// row on a side of the center)
type ControlViolation struct {
	Index int
	Rule  int
}

// ControlChart is a Shewhart control chart of individual values: the
// center line at the mean, the control limits at 3σ (σ by the moving
// range), the zones of 1σ and 2σ, and the points that break the Western
// Electric rules marked. For the statistical control of a process.
//
//	cc := ui.NewControlChart()
//	cc.SetValues(measurements)
//	for _, v := range cc.Violations() { ... }
type ControlChart struct {
	ChartView
	values     []float64
	line       *LineSeries
	bad        *ScatterSeries
	fixed      bool
	cl, sigma  float64
	violations []ControlViolation
}

// NewControlChart makes an empty control chart
func NewControlChart() *ControlChart {
	c := &ControlChart{}
	c.initChartView(NewLinearScale(0, 1))
	c.SetTypeName("ControlChart")
	c.legend = ChartLegendNone
	p := c.panes[0]
	c.line = p.AddLine("Value")
	c.line.SetMarkers(ChartMarkersAlways, MarkerCircle, 5)
	c.bad = p.AddScatter("Out of control")
	c.bad.SetMarker(MarkerCircle, 9)
	return c
}

// SetValues replaces the values and computes the limits and the violations
func (c *ControlChart) SetValues(values []float64) *ControlChart {
	c.values = append(c.values[:0], values...)
	c.compute()
	return c
}

// AddValue adds a value
func (c *ControlChart) AddValue(v float64) *ControlChart {
	c.values = append(c.values, v)
	c.compute()
	return c
}

// SetLimits fixes the center and σ (from a reference period) instead of
// computing them from the values; sigma 0 computes them again
func (c *ControlChart) SetLimits(center, sigma float64) *ControlChart {
	c.fixed = sigma > 0
	c.cl, c.sigma = center, sigma
	c.compute()
	return c
}

// Limits returns the center line and the control limits
func (c *ControlChart) Limits() (center, lcl, ucl float64) {
	return c.cl, c.cl - 3*c.sigma, c.cl + 3*c.sigma
}

// Violations returns the points out of control
func (c *ControlChart) Violations() []ControlViolation { return c.violations }

func (c *ControlChart) compute() {
	vals := c.values
	if !c.fixed && len(vals) > 0 {
		sum, mr := 0.0, 0.0
		for i, v := range vals {
			sum += v
			if i > 0 {
				mr += math.Abs(v - vals[i-1])
			}
		}
		c.cl = sum / float64(len(vals))
		c.sigma = 0
		if len(vals) > 1 {
			// d2 of a moving range of two
			c.sigma = mr / float64(len(vals)-1) / 1.128
		}
	}
	c.line.SetValues(vals)
	c.violations = controlViolations(vals, c.cl, c.sigma)
	var bad []ScatterPoint
	seen := map[int]bool{}
	for _, v := range c.violations {
		if !seen[v.Index] {
			seen[v.Index] = true
			bad = append(bad, ScatterPoint{X: float64(v.Index), Y: vals[v.Index], Value: math.NaN(), Label: "rule " + strconv.Itoa(v.Rule)})
		}
	}
	c.bad.SetPoints(bad)
	th := c.fixedTheme
	if th == nil {
		th = CurrentChartTheme()
	}
	c.bad.SetColor(th.Critical)
	ann := c.panes[0].Annotations()
	ann.Clear()
	if c.sigma > 0 {
		f := func(v float64) string { return trimChartZeros(strconv.FormatFloat(v, 'f', 3, 64)) }
		ann.AddYZone(c.cl-c.sigma, c.cl+c.sigma, "").SetColor(th.Good)
		ann.AddHLine(c.cl, "CL "+f(c.cl)).SetColor(th.Good)
		ann.AddHLine(c.cl+3*c.sigma, "UCL "+f(c.cl+3*c.sigma)).SetColor(th.Critical).SetDash(5, 3)
		ann.AddHLine(c.cl-3*c.sigma, "LCL "+f(c.cl-3*c.sigma)).SetColor(th.Critical).SetDash(5, 3)
		ann.AddHLine(c.cl+2*c.sigma, "").SetColor(th.Warning).SetDash(2, 3)
		ann.AddHLine(c.cl-2*c.sigma, "").SetColor(th.Warning).SetDash(2, 3)
	}
	c.Update()
}

// controlViolations finds the points breaking the Western Electric rules
func controlViolations(vals []float64, cl, sigma float64) []ControlViolation {
	var res []ControlViolation
	if sigma <= 0 {
		return nil
	}
	z := make([]float64, len(vals))
	for i, v := range vals {
		z[i] = (v - cl) / sigma
	}
	side := func(x float64) int {
		switch {
		case x > 0:
			return 1
		case x < 0:
			return -1
		}
		return 0
	}
	for i := range z {
		if math.Abs(z[i]) > 3 {
			res = append(res, ControlViolation{i, 1})
		}
		// k of the last n beyond the limit on the side of the point
		kOfN := func(k, n int, limit float64) bool {
			if i < n-1 || math.Abs(z[i]) <= limit {
				return false
			}
			s, count := side(z[i]), 0
			for j := i - n + 1; j <= i; j++ {
				if side(z[j]) == s && math.Abs(z[j]) > limit {
					count++
				}
			}
			return count >= k
		}
		if kOfN(2, 3, 2) {
			res = append(res, ControlViolation{i, 2})
		}
		if kOfN(4, 5, 1) {
			res = append(res, ControlViolation{i, 3})
		}
		if i >= 7 {
			s, run := side(z[i]), true
			for j := i - 7; j <= i; j++ {
				if side(z[j]) != s || s == 0 {
					run = false
					break
				}
			}
			if run {
				res = append(res, ControlViolation{i, 4})
			}
		}
	}
	return res
}
