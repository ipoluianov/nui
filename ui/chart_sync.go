package ui

import (
	"encoding/json"
	"math"
	"strconv"
)

// ChartSyncGroup ties charts together: the same range of the horizontal
// axis as one is panned and zoomed, the crosshair at the same x in all of
// them, the same selection. For a dashboard of several charts over the same
// period.
//
//	sync := ui.NewChartSyncGroup()
//	sync.Add(pressure, temperature, flow)
type ChartSyncGroup struct {
	views     []*ChartView
	x         bool
	crosshair bool
	selection bool
	busy      bool
}

// NewChartSyncGroup makes a group that ties the ranges, the crosshairs and
// the selections of its charts
func NewChartSyncGroup() *ChartSyncGroup {
	return &ChartSyncGroup{x: true, crosshair: true, selection: true}
}

// SetSync sets what the charts share
func (g *ChartSyncGroup) SetSync(xRange, crosshair, selection bool) *ChartSyncGroup {
	g.x, g.crosshair, g.selection = xRange, crosshair, selection
	return g
}

// Add adds charts to the group
func (g *ChartSyncGroup) Add(views ...*ChartView) *ChartSyncGroup {
	for _, v := range views {
		v := v
		g.views = append(g.views, v)
		v.OnViewChanged.Connect(func(*ChartView) { g.viewChanged(v) })
		v.OnSelectionChanged.Connect(func(*ChartView) { g.selectionChanged(v) })
		v.syncPointer = append(v.syncPointer, func(x float64, in bool) { g.pointer(v, x, in) })
	}
	return g
}

// Views returns the charts of the group
func (g *ChartSyncGroup) Views() []*ChartView { return g.views }

func (g *ChartSyncGroup) viewChanged(src *ChartView) {
	if g.busy || !g.x {
		return
	}
	g.busy = true
	defer func() { g.busy = false }()
	min, max := src.XRange()
	manual := src.xAxis.manual
	for _, v := range g.views {
		if v == src {
			continue
		}
		if manual {
			v.xAxis.setManual(min, max)
		} else {
			v.xAxis.manual = false
		}
		v.viewChanged()
	}
}

func (g *ChartSyncGroup) selectionChanged(src *ChartView) {
	if g.busy || !g.selection {
		return
	}
	g.busy = true
	defer func() { g.busy = false }()
	from, to, ok := src.Selection()
	for _, v := range g.views {
		if v == src {
			continue
		}
		if ok {
			v.SetSelection(from, to)
		} else {
			v.ClearSelection()
		}
	}
}

func (g *ChartSyncGroup) pointer(src *ChartView, x float64, in bool) {
	if !g.crosshair {
		return
	}
	for _, v := range g.views {
		if v == src {
			continue
		}
		v.hasSyncX, v.syncX = in, x
		v.Update()
	}
}

// notifyPointer tells the sync groups of the chart where its pointer is
func (c *ChartView) notifyPointer(in bool) {
	if len(c.syncPointer) == 0 {
		return
	}
	x := 0.0
	if in && c.laidOut {
		x = c.xAxis.FromPixel(c.mouseX)
	}
	for _, f := range c.syncPointer {
		f(x, in)
	}
}

// drawSyncCrosshair draws the line at the x of the pointer over another
// chart of the group
func (c *ChartView) drawSyncCrosshair(cnv *Canvas) {
	if !c.hasSyncX || c.mouseIn {
		return
	}
	x := c.xAxis.ToPixel(c.syncX)
	if !chartFinite(x) {
		return
	}
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	path.Reset()
	for _, p := range c.panes {
		if p.xAxis == c.xAxis && !p.noAxes() && x >= p.x && x <= p.x+p.w {
			path.MoveTo(chartCrisp(x), p.y)
			path.LineTo(chartCrisp(x), p.y+p.h)
		}
	}
	cnv.StrokePath(path, StrokeStyle{Width: 1, Dash: []float64{3, 3}}, c.theme.Crosshair)
}

// chartAxisJSON is the saved range of an axis
type chartAxisJSON struct {
	Manual bool    `json:"manual"`
	Min    float64 `json:"min,omitempty"`
	Max    float64 `json:"max,omitempty"`
}

// chartStateJSON is the saved view of a chart
type chartStateJSON struct {
	Axes      map[string]chartAxisJSON `json:"axes"`
	Hidden    []string                 `json:"hidden,omitempty"`
	Selection []float64                `json:"selection,omitempty"`
}

// axisKey names the axis for the saved state: "x", "0.y", "1.y2"...
func (c *ChartView) axisKeys() map[string]*ChartAxis {
	keys := map[string]*ChartAxis{}
	for i, a := range c.xAxes() {
		if i == 0 {
			keys["x"] = a
		} else {
			keys["x"+strconv.Itoa(i)] = a
		}
	}
	for i, p := range c.panes {
		for _, a := range p.axes {
			keys[strconv.Itoa(i)+"."+a.name] = a
		}
	}
	return keys
}

// SaveState returns the view of the chart as JSON: the zoomed ranges of the
// axes, the hidden series, the selection. Keep it in the settings and give
// it back to RestoreState on the next start.
func (c *ChartView) SaveState() []byte {
	st := chartStateJSON{Axes: map[string]chartAxisJSON{}}
	for k, a := range c.axisKeys() {
		if a.manual {
			st.Axes[k] = chartAxisJSON{Manual: true, Min: a.manMin, Max: a.manMax}
		}
	}
	for _, p := range c.panes {
		for _, s := range p.series {
			if !s.Visible() && s.Name() != "" {
				st.Hidden = append(st.Hidden, s.Name())
			}
		}
	}
	if c.hasSel {
		st.Selection = []float64{c.selFrom, c.selTo}
	}
	data, _ := json.Marshal(st)
	return data
}

// RestoreState shows the view saved by SaveState; the axes and the series
// the chart no longer has are skipped
func (c *ChartView) RestoreState(data []byte) error {
	var st chartStateJSON
	if err := json.Unmarshal(data, &st); err != nil {
		return err
	}
	keys := c.axisKeys()
	for k, a := range keys {
		if s, ok := st.Axes[k]; ok && s.Manual && s.Max > s.Min && !math.IsInf(s.Max-s.Min, 0) {
			a.setManual(s.Min, s.Max)
		} else {
			a.manual = false
		}
	}
	hidden := map[string]bool{}
	for _, n := range st.Hidden {
		hidden[n] = true
	}
	for _, p := range c.panes {
		for _, s := range p.series {
			if s.Name() != "" {
				s.SetVisible(!hidden[s.Name()])
			}
		}
	}
	if len(st.Selection) == 2 {
		c.SetSelection(st.Selection[0], st.Selection[1])
	} else {
		c.ClearSelection()
	}
	c.zoomStack = nil
	c.viewChanged()
	return nil
}
