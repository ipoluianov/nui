package ui

import (
	"math"
	"strings"
)

// chartDrag is a gesture in progress: a drag, or a press that may still
// become a click
type chartDrag struct {
	active bool
	button MouseButton
	mods   KeyModifiers
	action ChartAction
	// click is the action of a release without a move
	click    ChartAction
	hasClick bool
	pane     *ChartPane
	axis     *ChartAxis
	zone     chartZone

	startX, startY, curX, curY float64
	moved                      bool
	// band: a band (or a rectangle) is drawn as it is dragged
	band, rect bool
	// The ranges of the axes when the drag started
	ranges map[*ChartAxis][2]float64
	// The selection being moved or resized
	selMode                int
	selFrom, selTo         float64
	startDataX, startDataY float64
}

const (
	chartSelNew = iota
	chartSelMove
	chartSelLeft
	chartSelRight
)

// chartEdgeGrip is how close to an edge of the selection the pointer must be
// to resize it
const chartEdgeGrip = 4

// chartDragThreshold is how far the pointer moves before a press is a drag
const chartDragThreshold = 3

// chartAxisState is the range of an axis saved on the zoom stack
type chartAxisState struct {
	axis     *ChartAxis
	manual   bool
	min, max float64
}

type chartViewState []chartAxisState

const chartMaxZoomDepth = 100

// allAxes returns the horizontal axes and the axes of all the panes
func (c *ChartView) allAxes() []*ChartAxis {
	res := c.xAxes()
	for _, p := range c.panes {
		res = append(res, p.axes...)
	}
	return res
}

func (c *ChartView) saveState() chartViewState {
	var st chartViewState
	for _, a := range c.allAxes() {
		min, max := a.scale.Domain()
		if a.manual {
			min, max = a.manMin, a.manMax
		}
		st = append(st, chartAxisState{axis: a, manual: a.manual, min: min, max: max})
	}
	return st
}

func (c *ChartView) restoreState(st chartViewState) {
	for _, s := range st {
		s.axis.manual = s.manual
		if s.manual {
			s.axis.setManual(s.min, s.max)
		}
	}
}

// pushZoom saves the view on the zoom stack
func (c *ChartView) pushZoom() {
	c.zoomStack = append(c.zoomStack, c.saveState())
	if len(c.zoomStack) > chartMaxZoomDepth {
		c.zoomStack = c.zoomStack[1:]
	}
}

// ZoomBack goes back to the view saved by the last zoom; with nothing saved,
// to the default view of the horizontal axes
func (c *ChartView) ZoomBack() {
	if n := len(c.zoomStack); n > 0 {
		c.restoreState(c.zoomStack[n-1])
		c.zoomStack = c.zoomStack[:n-1]
	} else {
		for _, a := range c.xAxes() {
			a.manual = false
		}
	}
	c.viewChanged()
}

// ZoomDepth returns how many levels ZoomBack can go back
func (c *ChartView) ZoomDepth() int { return len(c.zoomStack) }

// ResetZoom clears the zoom stack and lets every axis show its default range
// or fit its data
func (c *ChartView) ResetZoom() {
	c.zoomStack = nil
	for _, a := range c.allAxes() {
		a.manual = false
	}
	c.viewChanged()
}

// IsZoomed reports whether there is anything for ResetZoom to undo
func (c *ChartView) IsZoomed() bool {
	if len(c.zoomStack) > 0 {
		return true
	}
	for _, a := range c.allAxes() {
		if a.manual {
			return true
		}
	}
	return false
}

// IsDefaultView reports whether the horizontal axis shows its default range
// (or fits the data): not panned or zoomed
func (c *ChartView) IsDefaultView() bool { return !c.xAxis.manual }

// SetXRange shows the range of the horizontal axis, leaving the default view
func (c *ChartView) SetXRange(min, max float64) {
	c.xAxis.setManual(min, max)
	c.viewChanged()
}

// XRange returns the visible range of the horizontal axis
func (c *ChartView) XRange() (min, max float64) {
	c.refit()
	return c.xAxis.scale.Domain()
}

// refit fits the axes that are not zoomed to their data now, not on the
// next paint: what XRange and ChartAxis.Range return is up to date. The
// vertical axes are fitted to the data of the plots of the last paint.
func (c *ChartView) refit() {
	c.fitXAxes()
	if !c.laidOut {
		return
	}
	for _, p := range c.panes {
		for _, s := range p.series {
			if s.Visible() {
				s.Prepare(p.viewport(p.seriesAxes(s)))
			}
		}
		p.computeStacks()
		for _, a := range p.axes {
			if a.manual || a.hasDefault {
				a.fit(0, 0, false)
				continue
			}
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, s := range p.series {
				if !s.Visible() {
					continue
				}
				x, y := p.seriesAxes(s)
				if y != a {
					continue
				}
				vp := p.viewport(x, y)
				if mn, mx, ok := s.Range(a, vp); ok {
					lo, hi = math.Min(lo, mn), math.Max(hi, mx)
				}
			}
			if lo <= hi {
				lo, hi = p.annotationRange(a, lo, hi)
			}
			a.fit(lo, hi, lo <= hi)
		}
	}
}

func (c *ChartView) viewChanged() {
	c.OnViewChanged.Emit(c)
	c.Update()
}

// Selection returns the selected range of the horizontal axis; ok is false
// when nothing is selected
func (c *ChartView) Selection() (from, to float64, ok bool) {
	return c.selFrom, c.selTo, c.hasSel
}

// SetSelection selects the range of the horizontal axis
func (c *ChartView) SetSelection(from, to float64) {
	if from > to {
		from, to = to, from
	}
	if c.hasSel && c.selFrom == from && c.selTo == to {
		return
	}
	c.hasSel, c.selFrom, c.selTo = true, from, to
	c.OnSelectionChanged.Emit(c)
	c.Update()
}

// ClearSelection removes the selection
func (c *ChartView) ClearSelection() {
	if !c.hasSel {
		return
	}
	c.hasSel = false
	c.selCloseHover = false
	c.OnSelectionChanged.Emit(c)
	c.Update()
}

// zoneAt returns what is at the point: a legend entry, an axis, a plot
func (c *ChartView) zoneAt(x, y float64) (zone chartZone, pane *ChartPane, axis *ChartAxis, entry *legendHit) {
	for i := range c.legendRight {
		h := &c.legendRight[i]
		if x >= h.x && x < h.x+h.w && y >= h.y && y < h.y+h.h {
			return chartZoneLegend, h.series.base().pane, nil, h
		}
	}
	for _, p := range c.panes {
		for i := range p.legendItems {
			h := &p.legendItems[i]
			if x >= h.x && x < h.x+h.w && y >= h.y && y < h.y+h.h {
				return chartZoneLegend, p, nil, h
			}
		}
	}
	if p := c.inPlot(x, y); p != nil {
		return chartZonePlot, p, nil, nil
	}
	for _, p := range c.panes {
		if p.noAxes() {
			continue
		}
		for _, a := range p.axes {
			if a.visible && x >= a.bandX && x < a.bandX+a.bandW && y >= a.bandY && y < a.bandY+a.bandH {
				return chartZoneYAxis, p, a, nil
			}
		}
	}
	for _, a := range c.xAxes() {
		if a.visible && x >= a.bandX && x < a.bandX+a.bandW && y >= a.bandY && y < a.bandY+a.bandH {
			for _, p := range c.panes {
				if p.xAxis == a {
					return chartZoneXAxis, p, a, nil
				}
			}
		}
	}
	return chartZoneAny, c.paneAt(x, y), nil, nil
}

// selectionPart returns which part of the selection is at the point: an
// edge, the header, or none (chartSelNew)
func (c *ChartView) selectionPart(x, y float64) int {
	if !c.hasSel {
		return chartSelNew
	}
	x1, x2 := c.xAxis.ToPixel(c.selFrom), c.xAxis.ToPixel(c.selTo)
	if p := c.inPlot(x, y); p != nil && p.xAxis == c.xAxis {
		switch {
		case math.Abs(x-x1) <= chartEdgeGrip:
			return chartSelLeft
		case math.Abs(x-x2) <= chartEdgeGrip:
			return chartSelRight
		}
	}
	if c.selHeader.contains(x, y) {
		return chartSelMove
	}
	return chartSelNew
}

func (c *ChartView) onMouseDown(button MouseButton, xi, yi int, mods KeyModifiers) bool {
	if !c.laidOut {
		return false
	}
	x, y := float64(xi), float64(yi)
	c.Focus()
	if button == MouseButtonLeft && c.selClose.contains(x, y) {
		c.ClearSelection()
		return true
	}
	zone, pane, axis, _ := c.zoneAt(x, y)
	d := chartDrag{active: true, button: button, mods: mods, pane: pane, axis: axis, zone: zone,
		startX: x, startY: y, curX: x, curY: y}
	if pane != nil && !pane.noAxes() {
		d.startDataX = pane.xAxis.FromPixel(x)
		d.startDataY = pane.axes[0].FromPixel(y)
	}
	// The selection's header and edges move and resize it
	if button == MouseButtonLeft {
		if part := c.selectionPart(x, y); part != chartSelNew {
			d.action, d.selMode = ChartActionSelectRange, part
			d.selFrom, d.selTo = c.selFrom, c.selTo
			c.drag = d
			return true
		}
	}
	action, hasDrag := c.in.match(chartGestDrag, button, 0, mods, zone)
	click, hasClick := c.in.match(chartGestClick, button, 0, mods, zone)
	if !hasDrag && !hasClick {
		return button == MouseButtonLeft
	}
	d.action, d.click, d.hasClick = action, click, hasClick
	if !hasDrag {
		d.action = ChartActionNone
	}
	d.ranges = map[*ChartAxis][2]float64{}
	for _, a := range c.allAxes() {
		min, max := a.scale.Domain()
		d.ranges[a] = [2]float64{min, max}
	}
	c.drag = d
	if d.action.kind == chartActCustom && d.action.custom.Begin != nil {
		if !d.action.custom.Begin(c.gestureEvent()) {
			c.drag.action = ChartActionNone
		}
	}
	return true
}

// gestureEvent returns the state of the gesture for a ChartGestureFunc
func (c *ChartView) gestureEvent() ChartGestureEvent {
	d := &c.drag
	e := ChartGestureEvent{View: c, Pane: d.pane, X: d.curX, Y: d.curY, StartX: d.startX, StartY: d.startY,
		StartDataX: d.startDataX, StartDataY: d.startDataY, Mods: d.mods, Hits: c.hits}
	if d.pane != nil && !d.pane.noAxes() {
		e.DataX = d.pane.xAxis.FromPixel(d.curX)
		e.DataY = d.pane.axes[0].FromPixel(d.curY)
	}
	return e
}

func (c *ChartView) onMouseMove(xi, yi int, mods KeyModifiers) bool {
	x, y := float64(xi), float64(yi)
	c.mouseIn, c.mouseX, c.mouseY = true, x, y
	d := &c.drag
	c.notifyPointer(true)
	if !d.active {
		c.updateCursor(x, y)
		if hover := c.selClose.contains(x, y); hover != c.selCloseHover {
			c.selCloseHover = hover
		}
		c.Update()
		return false
	}
	d.curX, d.curY = x, y
	if !d.moved && math.Hypot(x-d.startX, y-d.startY) < chartDragThreshold {
		return true
	}
	d.moved = true
	switch d.action.kind {
	case chartActPan:
		c.panBy(x-d.startX, y-d.startY, d.action.axes)
	case chartActZoomBox:
		d.band = true
		d.rect = d.action.axes&ChartAxesY != 0
	case chartActSelectRange:
		c.dragSelection()
	case chartActAxisDrag:
		c.stretchAxis()
	case chartActCustom:
		if d.action.custom.Move != nil {
			d.action.custom.Move(c.gestureEvent())
		}
	}
	c.Update()
	return true
}

// ProcessMouseUp ends a drag of any button: the form sends the release of
// the left button only to the widget it was pressed on, so a right drag
// ends here, wherever it is released
func (c *ChartView) ProcessMouseUp(button MouseButton, x int, y int, mods KeyModifiers, onlyForWidgetId string) bool {
	if c.drag.active && button == c.drag.button {
		c.finishDrag(float64(x), float64(y))
	}
	return c.Widget.ProcessMouseUp(button, x, y, mods, onlyForWidgetId)
}

func (c *ChartView) finishDrag(x, y float64) {
	d := c.drag
	c.drag.active = false
	c.drag.band = false
	defer c.Update()
	if !d.moved {
		if d.hasClick {
			c.runClick(d.click, d)
		}
		if d.action.kind == chartActCustom && d.action.custom.End != nil {
			c.drag = d
			d.action.custom.End(c.gestureEvent())
			c.drag.active = false
		}
		return
	}
	switch d.action.kind {
	case chartActZoomBox:
		if x < d.startX {
			c.ZoomBack()
			return
		}
		p := d.pane
		if p == nil || p.noAxes() {
			return
		}
		x1, x2 := c.clampX(p, math.Min(d.startX, x)), c.clampX(p, math.Max(d.startX, x))
		if x2-x1 < chartDragThreshold {
			return
		}
		from, to := p.xAxis.FromPixel(x1), p.xAxis.FromPixel(x2)
		e := &ChartActionEvent{Action: d.action, From: math.Min(from, to), To: math.Max(from, to)}
		if c.OnBeforeAction.Emit(e); e.Cancel {
			return
		}
		c.pushZoom()
		if d.rect && !c.in.limits.LockY {
			y1 := math.Max(math.Min(d.startY, y), p.y)
			y2 := math.Min(math.Max(d.startY, y), p.y+p.h)
			if y2-y1 >= chartDragThreshold {
				for _, a := range p.axes {
					lo, hi := a.FromPixel(y2), a.FromPixel(y1)
					a.setManual(math.Min(lo, hi), math.Max(lo, hi))
				}
			}
		}
		c.setAxisRange(p.xAxis, e.From, e.To)
		c.viewChanged()
	case chartActSelectRange:
		c.dragSelection()
	case chartActCustom:
		if d.action.custom.End != nil {
			c.drag = d
			d.action.custom.End(c.gestureEvent())
			c.drag.active = false
		}
	}
}

// clampX keeps the x within the plot of the pane
func (c *ChartView) clampX(p *ChartPane, x float64) float64 {
	return math.Max(p.x, math.Min(x, p.x+p.w))
}

// runClick runs the action of a click
func (c *ChartView) runClick(a ChartAction, d chartDrag) {
	switch a.kind {
	case chartActLegendToggle, chartActLegendSolo:
		_, _, _, entry := c.zoneAt(d.startX, d.startY)
		if entry != nil {
			c.toggleLegend(entry, a.kind == chartActLegendSolo)
		}
	case chartActPointClick:
		c.mouseIn, c.mouseX, c.mouseY = true, d.startX, d.startY
		c.updateHits()
		if len(c.hits) > 0 {
			c.OnPointClicked.Emit(c.hits[0])
		}
	case chartActMenu:
		c.showMenu(d.startX, d.startY)
	case chartActZoomBack:
		c.ZoomBack()
	case chartActZoomReset:
		c.ResetZoom()
	case chartActAxisAutoscale:
		c.autoscale(d.axis, d.pane)
	case chartActCustom:
		if a.custom.Begin != nil {
			c.drag = d
			if a.custom.Begin(c.gestureEvent()) && a.custom.End != nil {
				a.custom.End(c.gestureEvent())
			}
			c.drag.active = false
		}
	}
}

// toggleLegend hides or shows the series (or the slice) of the entry; solo
// shows it alone, or all of them when it is the only one shown
func (c *ChartView) toggleLegend(entry *legendHit, solo bool) {
	s := entry.series
	if t, ok := s.(chartLegendToggler); ok && entry.index >= 0 {
		t.toggleLegendItem(entry.index)
		c.OnSeriesToggled.Emit(s)
		return
	}
	if !solo {
		s.SetVisible(!s.Visible())
		c.OnSeriesToggled.Emit(s)
		return
	}
	alone := s.Visible()
	for _, p := range c.panes {
		for _, o := range p.series {
			if o != s && o.Visible() {
				alone = false
			}
		}
	}
	for _, p := range c.panes {
		for _, o := range p.series {
			o.SetVisible(alone || o == s)
		}
	}
	c.OnSeriesToggled.Emit(s)
}

func (c *ChartView) onMouseDblClick(button MouseButton, xi, yi int, mods KeyModifiers) bool {
	x, y := float64(xi), float64(yi)
	if c.selHeader.contains(x, y) {
		return true
	}
	zone, pane, axis, _ := c.zoneAt(x, y)
	a, ok := c.in.match(chartGestDblClick, button, 0, mods, zone)
	if !ok {
		return false
	}
	c.runClick(a, chartDrag{startX: x, startY: y, pane: pane, axis: axis, zone: zone})
	c.Update()
	return true
}

// onWheel zooms or pans with the wheel, as bound
func (c *ChartView) onWheel(e *WheelEvent) {
	if !c.laidOut {
		return
	}
	x, y := float64(e.X), float64(e.Y)
	zone, pane, axis, _ := c.zoneAt(x, y)
	kind, delta := chartGestWheel, e.DY
	if e.DY == 0 && e.DX != 0 {
		kind, delta = chartGestWheelX, e.DX
	}
	if delta == 0 {
		return
	}
	a, ok := c.in.match(kind, 0, 0, e.Mods, zone)
	if !ok && zone != chartZonePlot {
		a, ok = c.in.match(kind, 0, 0, e.Mods, chartZonePlot)
	}
	if !ok || pane == nil || pane.noAxes() {
		return
	}
	e.Handled = true
	switch a.kind {
	case chartActZoomWheel:
		k := math.Pow(0.8, float64(delta))
		if axis != nil {
			c.zoomAxis(axis, k, axis.unitAt(x, y), false)
		} else {
			c.zoomAt(pane, a, k, x, y)
		}
	case chartActPan:
		c.panBy(float64(delta)*30, 0, ChartAxesX)
		c.drag.ranges = nil
	}
	c.viewChanged()
}

func (c *ChartView) onKeyDown(key Key, mods KeyModifiers) bool {
	if !c.laidOut {
		return false
	}
	a, ok := c.in.match(chartGestKey, 0, key, mods, chartZonePlot)
	if !ok {
		return false
	}
	switch a.kind {
	case chartActZoomReset:
		if !c.IsZoomed() {
			return false // Esc goes on to the form: it closes a dialog
		}
		c.ResetZoom()
	case chartActZoomBack:
		if !c.IsZoomed() {
			return false
		}
		c.ZoomBack()
	case chartActPanStep:
		p := c.panes[0]
		c.drag.ranges = nil
		if a.dx != 0 {
			c.panBy(-a.dx*p.w, 0, ChartAxesX)
		}
		if a.dy != 0 && !c.in.limits.LockY {
			c.panBy(0, a.dy*p.h, ChartAxesY)
		}
	case chartActZoomStep:
		for _, ax := range c.xAxes() {
			c.zoomAxis(ax, a.factor, 0.5, true)
		}
	case chartActMenu:
		c.showMenu(c.mouseX, c.mouseY)
	case chartActCustom:
		if a.custom.Begin != nil {
			c.drag = chartDrag{startX: c.mouseX, startY: c.mouseY, curX: c.mouseX, curY: c.mouseY, pane: c.paneAt(c.mouseX, c.mouseY)}
			a.custom.Begin(c.gestureEvent())
		}
	default:
		return false
	}
	c.viewChanged()
	return true
}

// unitAt returns where the point is along the axis, 0..1 of its range
func (a *ChartAxis) unitAt(x, y float64) float64 {
	p := y
	if a.Horizontal() {
		p = x
	}
	return a.scale.ToUnit(a.FromPixel(p))
}

// pixelUnits returns the part of the axis' range a move of d pixels along
// it is, in the direction the values grow
func (a *ChartAxis) pixelUnits(d float64) float64 {
	if a.length <= 0 {
		return 0
	}
	u := d / a.length
	if !a.Horizontal() {
		u = -u
	}
	if a.inverted {
		u = -u
	}
	return u
}

// panBy moves the view by the drag (dx, dy) from its start: the data
// follows the pointer. Without a drag in progress, from the current view.
func (c *ChartView) panBy(dx, dy float64, axes ChartAxes) {
	d := &c.drag
	start := func(a *ChartAxis) (float64, float64) {
		if r, ok := d.ranges[a]; ok && d.active {
			return r[0], r[1]
		}
		return a.scale.Domain()
	}
	pane := d.pane
	if pane == nil || !d.active {
		pane = c.paneAt(c.mouseX, c.mouseY)
	}
	if axes&ChartAxesX != 0 && dx != 0 {
		for _, a := range c.xAxes() {
			if pane != nil && a != pane.xAxis && len(c.xAxes()) > 1 {
				continue
			}
			min, max := start(a)
			a.scale.SetDomain(min, max)
			u := a.pixelUnits(dx)
			from, to := a.scale.FromUnit(-u), a.scale.FromUnit(1-u)
			e := &ChartActionEvent{Action: ChartActionPan, From: from, To: to}
			if c.OnBeforeAction.Emit(e); e.Cancel {
				a.scale.SetDomain(min, max)
				continue
			}
			c.setAxisRange(a, e.From, e.To)
		}
	}
	if axes&ChartAxesY != 0 && dy != 0 && pane != nil && !c.in.limits.LockY {
		for _, a := range pane.axes {
			min, max := start(a)
			a.scale.SetDomain(min, max)
			u := a.pixelUnits(dy)
			a.setManual(a.scale.FromUnit(-u), a.scale.FromUnit(1-u))
		}
	}
	c.OnViewChanged.Emit(c)
}

// setAxisRange shows the range on a horizontal axis within the limits
func (c *ChartView) setAxisRange(a *ChartAxis, from, to float64) {
	l := c.in.limits
	span := to - from
	if a.Horizontal() {
		if l.MinSpanX > 0 && span < l.MinSpanX {
			mid := (from + to) / 2
			from, to = mid-l.MinSpanX/2, mid+l.MinSpanX/2
		}
		if l.MaxSpanX > 0 && span > l.MaxSpanX {
			mid := (from + to) / 2
			from, to = mid-l.MaxSpanX/2, mid+l.MaxSpanX/2
		}
		lo, hi, bounded := 0.0, 0.0, false
		switch l.PanBounds {
		case ChartPanData:
			lo, hi, bounded = c.dataXExtent(a)
		case ChartPanRange:
			lo, hi, bounded = l.Min, l.Max, l.Max > l.Min
		}
		if bounded {
			span := to - from
			if span > hi-lo {
				from, to = lo, hi
			} else if from < lo {
				from, to = lo, lo+span
			} else if to > hi {
				from, to = hi-span, hi
			}
		}
	} else {
		if l.MinSpanY > 0 && span < l.MinSpanY {
			mid := (from + to) / 2
			from, to = mid-l.MinSpanY/2, mid+l.MinSpanY/2
		}
		if l.MaxSpanY > 0 && span > l.MaxSpanY {
			mid := (from + to) / 2
			from, to = mid-l.MaxSpanY/2, mid+l.MaxSpanY/2
		}
	}
	a.setManual(from, to)
}

// dataXExtent returns the range of the data of the series on the axis
func (c *ChartView) dataXExtent(a *ChartAxis) (float64, float64, bool) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range c.panes {
		if p.xAxis != a {
			continue
		}
		for _, s := range p.series {
			if mn, mx, ok := s.Range(a, nil); ok {
				lo, hi = math.Min(lo, mn), math.Max(hi, mx)
			}
		}
	}
	return lo, hi, lo < hi
}

// zoomAt zooms the axes of the action in the pane by k around the point
func (c *ChartView) zoomAt(p *ChartPane, a ChartAction, k, x, y float64) {
	if a.axes&ChartAxesX != 0 {
		u := p.xAxis.unitAt(x, y)
		if a.anchorRight {
			u = 1
			if p.xAxis.inverted {
				u = 0
			}
		}
		c.zoomAxis(p.xAxis, k, u, false)
	}
	if a.axes&ChartAxesY != 0 && !c.in.limits.LockY {
		for _, ax := range p.axes {
			c.zoomAxis(ax, k, ax.unitAt(x, y), false)
		}
	}
}

// zoomAxis zooms the axis by k around the point u of its range (0..1); a
// horizontal axis asks OnBeforeAction first
func (c *ChartView) zoomAxis(a *ChartAxis, k, u float64, _ bool) {
	if !chartFinite(u) {
		u = 0.5
	}
	from, to := a.scale.FromUnit(u-u*k), a.scale.FromUnit(u+(1-u)*k)
	if a.Horizontal() {
		e := &ChartActionEvent{Action: ChartActionZoomWheel, From: from, To: to}
		if c.OnBeforeAction.Emit(e); e.Cancel {
			return
		}
		c.setAxisRange(a, e.From, e.To)
		return
	}
	c.setAxisRange(a, from, to)
}

// stretchAxis stretches the dragged axis around the middle of its range
func (c *ChartView) stretchAxis() {
	d := &c.drag
	a := d.axis
	if a == nil {
		return
	}
	r := d.ranges[a]
	a.scale.SetDomain(r[0], r[1])
	delta := d.curX - d.startX
	if !a.Horizontal() {
		delta = d.curY - d.startY
	} else {
		delta = -delta
	}
	k := math.Exp(delta / 150)
	from, to := a.scale.FromUnit(0.5-0.5*k), a.scale.FromUnit(0.5+0.5*k)
	c.setAxisRange(a, from, to)
	c.OnViewChanged.Emit(c)
}

// autoscale makes the axis fit its data again; with no axis, all the axes
// of the pane and the horizontal one
func (c *ChartView) autoscale(a *ChartAxis, p *ChartPane) {
	if a != nil {
		a.manual = false
	} else if p != nil {
		p.xAxis.manual = false
		for _, ax := range p.axes {
			ax.manual = false
		}
	}
	c.viewChanged()
}

// dragSelection makes, moves or resizes the selection as dragged
func (c *ChartView) dragSelection() {
	d := &c.drag
	p := d.pane
	if p == nil || p.noAxes() {
		return
	}
	ax := c.xAxis
	var from, to float64
	switch d.selMode {
	case chartSelNew:
		d.band = true
		x1, x2 := c.clampX(p, math.Min(d.startX, d.curX)), c.clampX(p, math.Max(d.startX, d.curX))
		if c.drag.active {
			return // drawn as a band until released
		}
		if x2-x1 < chartDragThreshold {
			return
		}
		from, to = ax.FromPixel(x1), ax.FromPixel(x2)
	case chartSelMove:
		shift := ax.FromPixel(d.curX) - ax.FromPixel(d.startX)
		from, to = d.selFrom+shift, d.selTo+shift
	case chartSelLeft:
		from, to = ax.FromPixel(c.clampX(p, d.curX)), d.selTo
	case chartSelRight:
		from, to = d.selFrom, ax.FromPixel(c.clampX(p, d.curX))
	}
	e := &ChartActionEvent{Action: ChartActionSelectRange, From: math.Min(from, to), To: math.Max(from, to)}
	if c.OnBeforeAction.Emit(e); e.Cancel {
		return
	}
	c.SetSelection(e.From, e.To)
}

// updateCursor shows the resize cursor over the edges of the selection and
// a hand over its header and the legend
func (c *ChartView) updateCursor(x, y float64) {
	cursor := MouseCursorArrow
	switch c.selectionPart(x, y) {
	case chartSelLeft, chartSelRight:
		cursor = MouseCursorResizeHor
	case chartSelMove:
		cursor = MouseCursorPointer
	default:
		if zone, _, _, _ := c.zoneAt(x, y); zone == chartZoneLegend {
			cursor = MouseCursorPointer
		}
	}
	if c.MouseCursor() != cursor {
		c.SetMouseCursor(cursor)
	}
}

// showMenu opens the context menu of the chart at the point: zoom, the
// series to show, and what the application adds (SetContextMenuFunc)
func (c *ChartView) showMenu(x, y float64) {
	menu := NewContextMenu(c)
	menu.AddItem("Reset zoom", func() { c.ResetZoom() })
	back := menu.AddItem("Zoom back", func() { c.ZoomBack() })
	back.SetEnabled(len(c.zoomStack) > 0)
	if c.hasSel {
		menu.AddItem("Clear selection", func() { c.ClearSelection() })
	}
	first := true
	for _, p := range c.panes {
		for _, s := range p.series {
			if s.Name() == "" {
				continue
			}
			if first {
				menu.AddSeparator()
				first = false
			}
			s := s
			it := menu.AddItem(s.Name(), func() {
				s.SetVisible(!s.Visible())
				c.OnSeriesToggled.Emit(s)
			})
			it.SetCheckable(true)
			it.SetChecked(s.Visible())
		}
	}
	menu.AddSeparator()
	menu.AddItem("Copy image", func() { c.CopyImage() })
	menu.AddItem("Copy data (CSV)", func() {
		var b strings.Builder
		if c.WriteCSV(&b, false) == nil {
			ClipboardSetText(b.String())
		}
	})
	if c.menuFunc != nil {
		c.mouseIn, c.mouseX, c.mouseY = true, x, y
		c.updateHits()
		c.menuFunc(menu, c.hits)
	}
	wx, wy := c.RectClientAreaOnWindow()
	menu.ShowMenu(wx+int(x), wy+int(y))
}

// SetContextMenuFunc lets the application add its items to the context menu
// of the chart: f gets the menu and the elements under the pointer
func (c *ChartView) SetContextMenuFunc(f func(menu *ContextMenu, hits []ChartHit)) *ChartView {
	c.menuFunc = f
	return c
}
