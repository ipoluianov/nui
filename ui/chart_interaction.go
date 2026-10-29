package ui

import (
	"fmt"
	"runtime"
	"strings"
)

// ChartAxes says along which axes an action works
type ChartAxes int

const (
	ChartAxesX  ChartAxes = 1
	ChartAxesY  ChartAxes = 2
	ChartAxesXY ChartAxes = ChartAxesX | ChartAxesY
)

type chartActionKind int

const (
	chartActNone chartActionKind = iota
	chartActPan
	chartActZoomWheel
	chartActZoomBox
	chartActZoomBack
	chartActZoomReset
	chartActAxisDrag
	chartActAxisAutoscale
	chartActSelectRange
	chartActLegendToggle
	chartActLegendSolo
	chartActMenu
	chartActPointClick
	chartActPanStep
	chartActZoomStep
	chartActCustom
)

var chartActionNames = map[chartActionKind]string{
	chartActNone: "none", chartActPan: "pan", chartActZoomWheel: "zoom.wheel", chartActZoomBox: "zoom.box",
	chartActZoomBack: "zoom.back", chartActZoomReset: "zoom.reset", chartActAxisDrag: "axis.drag",
	chartActAxisAutoscale: "axis.autoscale", chartActSelectRange: "select.range",
	chartActLegendToggle: "legend.toggle", chartActLegendSolo: "legend.solo", chartActMenu: "menu",
	chartActPointClick: "point.click", chartActPanStep: "pan.step", chartActZoomStep: "zoom.step",
	chartActCustom: "custom",
}

// ChartAction is what a chart does on a gesture of the user: pan, zoom,
// select... The actions are the ChartAction... values; some take
// parameters (OnAxes, AnchoredRight). Bind them to gestures with
// ChartInteraction.Bind.
type ChartAction struct {
	kind        chartActionKind
	axes        ChartAxes
	anchorRight bool
	dx, dy      float64
	factor      float64
	custom      *ChartGestureFunc
}

var (
	// ChartActionNone does nothing: binding it to a gesture of a profile
	// turns the gesture off
	ChartActionNone = ChartAction{kind: chartActNone}
	// ChartActionPan moves the view with the drag, along the x axis by
	// default
	ChartActionPan = ChartAction{kind: chartActPan, axes: ChartAxesX}
	// ChartActionZoomWheel zooms around the pointer with the wheel
	ChartActionZoomWheel = ChartAction{kind: chartActZoomWheel, axes: ChartAxesX}
	// ChartActionZoomBox zooms into the range dragged over; dragged to the
	// left it zooms back one level. Each zoom saves the view on a stack.
	ChartActionZoomBox = ChartAction{kind: chartActZoomBox, axes: ChartAxesX}
	// ChartActionZoomBack goes back one zoom level
	ChartActionZoomBack = ChartAction{kind: chartActZoomBack}
	// ChartActionZoomReset clears the zoom stack: the default view, the
	// axes fit their data. On a key, only when there is something to reset,
	// so Esc still closes a dialog.
	ChartActionZoomReset = ChartAction{kind: chartActZoomReset}
	// ChartActionAxisDrag stretches the axis under the pointer as it is
	// dragged along it
	ChartActionAxisDrag = ChartAction{kind: chartActAxisDrag}
	// ChartActionAxisAutoscale makes the axis under the pointer (all of
	// them on the plot) fit the data again
	ChartActionAxisAutoscale = ChartAction{kind: chartActAxisAutoscale}
	// ChartActionSelectRange selects a range of the x axis: dragged on the
	// plot it makes a new one, on the header of the selection moves it, on
	// its edges resizes it
	ChartActionSelectRange = ChartAction{kind: chartActSelectRange, axes: ChartAxesX}
	// ChartActionLegendToggle hides or shows the series of the legend entry
	ChartActionLegendToggle = ChartAction{kind: chartActLegendToggle}
	// ChartActionLegendSolo shows only the series of the legend entry, or
	// all of them when it is the only one shown
	ChartActionLegendSolo = ChartAction{kind: chartActLegendSolo}
	// ChartActionMenu opens the context menu of the chart
	ChartActionMenu = ChartAction{kind: chartActMenu}
	// ChartActionPointClick emits OnPointClicked with the element under the
	// pointer
	ChartActionPointClick = ChartAction{kind: chartActPointClick}
)

// ChartActionPanStep moves the view by the parts of its range: dx 0.1 a
// tenth to the right (for the arrow keys)
func ChartActionPanStep(dx, dy float64) ChartAction {
	return ChartAction{kind: chartActPanStep, dx: dx, dy: dy, axes: ChartAxesXY}
}

// ChartActionZoomStep zooms the view around its middle: 0.8 shows 80% of
// the range, 1.25 shows more (for the + and - keys)
func ChartActionZoomStep(factor float64) ChartAction {
	return ChartAction{kind: chartActZoomStep, factor: factor, axes: ChartAxesX}
}

// OnAxes returns the action working along the axes
func (a ChartAction) OnAxes(axes ChartAxes) ChartAction {
	a.axes = axes
	return a
}

// AnchoredRight returns the zoom keeping the right edge of the range in
// place, not the point under the pointer: the last bar of a trading chart
// stays in view
func (a ChartAction) AnchoredRight() ChartAction {
	a.anchorRight = true
	return a
}

// Name returns the name of the action: "pan", "zoom.box"...
func (a ChartAction) Name() string { return chartActionNames[a.kind] }

// ChartGestureFunc is an action of an application's own: Begin is called
// when the gesture starts (returning false lets it go to nothing), Move as
// a drag goes on, End when it ends. A click calls Begin and End.
type ChartGestureFunc struct {
	Begin func(e ChartGestureEvent) bool
	Move  func(e ChartGestureEvent)
	End   func(e ChartGestureEvent)
}

// ChartGestureEvent is the state of a gesture for a ChartGestureFunc
type ChartGestureEvent struct {
	View *ChartView
	Pane *ChartPane
	// The pointer and where the gesture started, in logical pixels of the
	// widget and as values of the pane's x axis and first y axis
	X, Y, StartX, StartY                 float64
	DataX, DataY, StartDataX, StartDataY float64
	Mods                                 KeyModifiers
	// Hits are the elements under the pointer
	Hits []ChartHit
}

// ChartActionEvent is an action about to change the view, for
// ChartView.OnBeforeAction: a handler may set Cancel, or change From and
// To - the new range of the x axis (pan, zoom) or of the selection
type ChartActionEvent struct {
	Action   ChartAction
	From, To float64
	Cancel   bool
}

// ChartTool is what a drag with the left button does on the plot, when the
// application has a toolbar for it (ChartInteraction.SetTool)
type ChartTool int

const (
	// ChartToolDefault leaves the left drag as the profile binds it
	ChartToolDefault ChartTool = iota
	ChartToolPan
	ChartToolZoom
	ChartToolSelect
	ChartToolNone
)

// ChartPanBounds limits how far the view may be panned along x
type ChartPanBounds int

const (
	// ChartPanFree pans anywhere
	ChartPanFree ChartPanBounds = iota
	// ChartPanData keeps the view on the data of the series
	ChartPanData
	// ChartPanRange keeps the view within ChartLimits.Min and Max
	ChartPanRange
)

// ChartLimits are the bounds of the user's pan and zoom; 0 - no limit
type ChartLimits struct {
	MinSpanX, MaxSpanX float64
	MinSpanY, MaxSpanY float64
	PanBounds          ChartPanBounds
	Min, Max           float64
	// LockY keeps the y axes fitting the data whatever the gesture
	LockY bool
}

type chartZone int

const (
	chartZonePlot chartZone = iota
	chartZoneAxis
	chartZoneXAxis
	chartZoneYAxis
	chartZoneLegend
	chartZoneAny
)

type chartGestureKind int

const (
	chartGestDrag chartGestureKind = iota
	chartGestClick
	chartGestDblClick
	chartGestWheel
	chartGestWheelX
	chartGestKey
)

type chartGesture struct {
	kind                  chartGestureKind
	button                MouseButton
	key                   Key
	ctrl, shift, alt, cmd bool
	zone                  chartZone
}

type chartBinding struct {
	gesture chartGesture
	text    string
	action  ChartAction
}

// ChartProfile is a set of bindings of gestures to actions for a kind of
// chart (ChartInteraction.UseProfile)
type ChartProfile int

const (
	// ChartProfileScientific: left drag zooms into a rectangle, right drag
	// pans, the wheel zooms both axes, a double click resets
	ChartProfileScientific ChartProfile = iota
	// ChartProfileTimeSeries: as TimeChart - left drag pans along x, Shift
	// selects a range, right drag zooms into a range of x (Ctrl into a
	// rectangle), the wheel zooms x, a double click goes back a level
	ChartProfileTimeSeries
	// ChartProfileFinance: as trading terminals - left drag pans both
	// axes, the wheel zooms x keeping the last bar, dragging an axis
	// stretches it, a double click fits the axes to the data
	ChartProfileFinance
	// ChartProfileDashboard: only the tooltip and the legend
	ChartProfileDashboard
	// ChartProfileStatic: nothing, for printing and thumbnails
	ChartProfileStatic
)

// ChartInteraction is how a chart answers the mouse and the keyboard: a
// table of gestures and the actions bound to them. A profile fills it for a
// kind of chart; Bind and Unbind change single gestures.
//
//	in := chart.Interaction()
//	in.UseProfile(ui.ChartProfileFinance)
//	in.Bind("Shift+LeftDrag", ui.ChartActionSelectRange)
//	in.Bind("Ctrl+Wheel", ui.ChartActionZoomWheel.OnAxes(ui.ChartAxesY))
//	in.Unbind("RightDrag")
//
// A gesture is written as a shortcut: modifiers (Ctrl, Shift, Alt, Cmd,
// Mod) and one of LeftDrag, RightDrag, MiddleDrag, LeftClick, RightClick,
// MiddleClick, LeftDblClick, RightDblClick, Wheel, WheelX (the sideways
// scroll of a touchpad) or a key (Esc, Home, Left, Plus, A...). A mouse
// gesture may be limited to a zone: @plot (the default), @axis, @xaxis,
// @yaxis, @legend, @any.
type ChartInteraction struct {
	view     *ChartView
	bindings []chartBinding
	limits   ChartLimits
	tool     ChartTool
	profile  ChartProfile
	// The actions of the tools for a toolbar, checked by SetTool
	toolActions map[ChartTool]*Action
}

func newChartInteraction(view *ChartView) *ChartInteraction {
	return &ChartInteraction{view: view}
}

// Profile returns the profile used last
func (in *ChartInteraction) Profile() ChartProfile { return in.profile }

// UseProfile replaces the bindings with the profile's
func (in *ChartInteraction) UseProfile(p ChartProfile) {
	in.profile = p
	in.bindings = nil
	common := func() {
		in.mustBind("LeftClick@legend", ChartActionLegendToggle)
		in.mustBind("LeftDblClick@legend", ChartActionLegendSolo)
	}
	keys := func() {
		in.mustBind("Esc", ChartActionZoomReset)
		in.mustBind("Home", ChartActionZoomReset)
		in.mustBind("Left", ChartActionPanStep(-0.1, 0))
		in.mustBind("Right", ChartActionPanStep(0.1, 0))
		in.mustBind("Plus", ChartActionZoomStep(0.8))
		in.mustBind("Minus", ChartActionZoomStep(1.25))
		in.mustBind("NumPlus", ChartActionZoomStep(0.8))
		in.mustBind("NumMinus", ChartActionZoomStep(1.25))
		in.mustBind("Backspace", ChartActionZoomBack)
	}
	switch p {
	case ChartProfileScientific:
		in.mustBind("LeftDrag", ChartActionZoomBox.OnAxes(ChartAxesXY))
		in.mustBind("Shift+LeftDrag", ChartActionSelectRange)
		in.mustBind("RightDrag", ChartActionPan.OnAxes(ChartAxesXY))
		in.mustBind("MiddleDrag", ChartActionPan.OnAxes(ChartAxesXY))
		in.mustBind("Wheel", ChartActionZoomWheel.OnAxes(ChartAxesXY))
		in.mustBind("Ctrl+Wheel", ChartActionZoomWheel.OnAxes(ChartAxesY))
		in.mustBind("Shift+Wheel", ChartActionZoomWheel.OnAxes(ChartAxesX))
		in.mustBind("LeftDrag@axis", ChartActionAxisDrag)
		in.mustBind("LeftDblClick@axis", ChartActionAxisAutoscale)
		in.mustBind("LeftDblClick", ChartActionZoomReset)
		in.mustBind("LeftClick", ChartActionPointClick)
		common()
		keys()
		in.mustBind("Up", ChartActionPanStep(0, 0.1))
		in.mustBind("Down", ChartActionPanStep(0, -0.1))
	case ChartProfileTimeSeries:
		in.mustBind("LeftDrag", ChartActionPan)
		in.mustBind("Shift+LeftDrag", ChartActionSelectRange)
		in.mustBind("RightDrag", ChartActionZoomBox)
		in.mustBind("Ctrl+RightDrag", ChartActionZoomBox.OnAxes(ChartAxesXY))
		in.mustBind("Wheel", ChartActionZoomWheel)
		in.mustBind("WheelX", ChartActionPan)
		in.mustBind("LeftDblClick", ChartActionZoomBack)
		in.mustBind("RightDblClick", ChartActionZoomBack)
		in.mustBind("LeftClick", ChartActionPointClick)
		common()
		keys()
	case ChartProfileFinance:
		in.mustBind("LeftDrag", ChartActionPan.OnAxes(ChartAxesXY))
		in.mustBind("Shift+LeftDrag", ChartActionSelectRange)
		in.mustBind("Wheel", ChartActionZoomWheel.AnchoredRight())
		in.mustBind("WheelX", ChartActionPan)
		in.mustBind("LeftDrag@axis", ChartActionAxisDrag)
		in.mustBind("LeftDblClick@axis", ChartActionAxisAutoscale)
		in.mustBind("LeftDblClick", ChartActionAxisAutoscale)
		in.mustBind("LeftClick", ChartActionPointClick)
		common()
		keys()
	case ChartProfileDashboard:
		in.mustBind("LeftClick", ChartActionPointClick)
		common()
	case ChartProfileStatic:
	}
	in.view.Update()
}

func (in *ChartInteraction) mustBind(gesture string, a ChartAction) {
	if err := in.Bind(gesture, a); err != nil {
		panic(err)
	}
}

// Bind binds the gesture to the action, in place of the action bound to it
// before
func (in *ChartInteraction) Bind(gesture string, action ChartAction) error {
	g, err := parseChartGesture(gesture)
	if err != nil {
		return err
	}
	for i, b := range in.bindings {
		if b.gesture == g {
			in.bindings[i].action = action
			in.bindings[i].text = gesture
			return nil
		}
	}
	in.bindings = append(in.bindings, chartBinding{gesture: g, text: gesture, action: action})
	return nil
}

// BindFunc binds the gesture to an action of the application's own
func (in *ChartInteraction) BindFunc(gesture string, f ChartGestureFunc) error {
	return in.Bind(gesture, ChartAction{kind: chartActCustom, custom: &f})
}

// Unbind removes the binding of the gesture
func (in *ChartInteraction) Unbind(gesture string) error {
	g, err := parseChartGesture(gesture)
	if err != nil {
		return err
	}
	for i, b := range in.bindings {
		if b.gesture == g {
			in.bindings = append(in.bindings[:i:i], in.bindings[i+1:]...)
			return nil
		}
	}
	return nil
}

// Binding returns the action bound to the gesture
func (in *ChartInteraction) Binding(gesture string) (ChartAction, bool) {
	g, err := parseChartGesture(gesture)
	if err != nil {
		return ChartAction{}, false
	}
	for _, b := range in.bindings {
		if b.gesture == g {
			return b.action, true
		}
	}
	return ChartAction{}, false
}

// Bindings returns the gestures and the names of their actions, for a help
// screen or a settings dialog
func (in *ChartInteraction) Bindings() map[string]string {
	m := make(map[string]string, len(in.bindings))
	for _, b := range in.bindings {
		m[b.text] = b.action.Name()
	}
	return m
}

// SetLimits sets the bounds of the user's pan and zoom
func (in *ChartInteraction) SetLimits(l ChartLimits) { in.limits = l }

// Limits returns the bounds of the user's pan and zoom
func (in *ChartInteraction) Limits() ChartLimits { return in.limits }

// SetTool makes the left drag on the plot pan, zoom or select, whatever the
// profile binds to it: for the buttons of a toolbar (see ToolAction)
func (in *ChartInteraction) SetTool(t ChartTool) {
	in.tool = t
	for tool, a := range in.toolActions {
		a.SetChecked(tool == t)
	}
}

// Tool returns the tool of the left drag
func (in *ChartInteraction) Tool() ChartTool { return in.tool }

// ToolAction returns a checkable Action that sets the tool: put the actions
// of the tools on a toolbar, and they show which one is on
func (in *ChartInteraction) ToolAction(t ChartTool, text string) *Action {
	if a, ok := in.toolActions[t]; ok {
		return a
	}
	a := NewAction(text, func() { in.SetTool(t) })
	a.SetCheckable(true)
	a.SetChecked(in.tool == t)
	if in.toolActions == nil {
		in.toolActions = map[ChartTool]*Action{}
	}
	in.toolActions[t] = a
	return a
}

// match returns the action bound to the gesture in the zone: the binding of
// the zone first, then of any zone
func (in *ChartInteraction) match(kind chartGestureKind, button MouseButton, key Key, mods KeyModifiers, zone chartZone) (ChartAction, bool) {
	if in.tool != ChartToolDefault && kind == chartGestDrag && button == MouseButtonLeft && zone == chartZonePlot &&
		!mods.Shift && !mods.Ctrl && !mods.Alt && !mods.Cmd {
		switch in.tool {
		case ChartToolPan:
			return ChartActionPan.OnAxes(ChartAxesXY), true
		case ChartToolZoom:
			return ChartActionZoomBox.OnAxes(ChartAxesXY), true
		case ChartToolSelect:
			return ChartActionSelectRange, true
		case ChartToolNone:
			return ChartActionNone, true
		}
	}
	zones := []chartZone{zone}
	switch zone {
	case chartZoneXAxis, chartZoneYAxis:
		zones = append(zones, chartZoneAxis)
	}
	zones = append(zones, chartZoneAny)
	for _, z := range zones {
		for _, b := range in.bindings {
			g := b.gesture
			if g.kind != kind || g.zone != z || g.ctrl != mods.Ctrl || g.shift != mods.Shift || g.alt != mods.Alt || g.cmd != mods.Cmd {
				continue
			}
			if kind == chartGestKey && g.key != key {
				continue
			}
			if (kind == chartGestDrag || kind == chartGestClick || kind == chartGestDblClick) && g.button != button {
				continue
			}
			return b.action, true
		}
	}
	return ChartAction{}, false
}

// parseChartGesture parses a gesture: "Shift+LeftDrag@plot", "Ctrl+Wheel", "Esc"
func parseChartGesture(s string) (chartGesture, error) {
	var g chartGesture
	text := strings.TrimSpace(s)
	if at := strings.LastIndex(text, "@"); at > 0 {
		switch strings.ToLower(text[at+1:]) {
		case "plot":
			g.zone = chartZonePlot
		case "axis":
			g.zone = chartZoneAxis
		case "xaxis":
			g.zone = chartZoneXAxis
		case "yaxis":
			g.zone = chartZoneYAxis
		case "legend":
			g.zone = chartZoneLegend
		case "any":
			g.zone = chartZoneAny
		default:
			return g, fmt.Errorf("chart gesture %q: unknown zone %q", s, text[at+1:])
		}
		text = text[:at]
	}
	parts := strings.Split(text, "+")
	if strings.HasSuffix(text, "++") {
		parts = append(strings.Split(strings.TrimSuffix(text, "++"), "+"), "PLUS")
	}
	if len(parts) == 0 || text == "" {
		return g, fmt.Errorf("chart gesture %q: empty", s)
	}
	for _, mod := range parts[:len(parts)-1] {
		switch strings.ToUpper(strings.TrimSpace(mod)) {
		case "CTRL", "CONTROL":
			g.ctrl = true
		case "SHIFT":
			g.shift = true
		case "ALT", "OPTION":
			g.alt = true
		case "CMD", "COMMAND", "META":
			g.cmd = true
		case "MOD":
			if runtime.GOOS == "darwin" {
				g.cmd = true
			} else {
				g.ctrl = true
			}
		default:
			return g, fmt.Errorf("chart gesture %q: unknown modifier %q", s, mod)
		}
	}
	name := strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
	mouse := map[string]struct {
		kind   chartGestureKind
		button MouseButton
	}{
		"LEFTDRAG": {chartGestDrag, MouseButtonLeft}, "RIGHTDRAG": {chartGestDrag, MouseButtonRight},
		"MIDDLEDRAG": {chartGestDrag, MouseButtonMiddle}, "LEFTCLICK": {chartGestClick, MouseButtonLeft},
		"RIGHTCLICK": {chartGestClick, MouseButtonRight}, "MIDDLECLICK": {chartGestClick, MouseButtonMiddle},
		"LEFTDBLCLICK": {chartGestDblClick, MouseButtonLeft}, "RIGHTDBLCLICK": {chartGestDblClick, MouseButtonRight},
		"WHEEL": {chartGestWheel, 0}, "WHEELX": {chartGestWheelX, 0},
	}
	if m, ok := mouse[name]; ok {
		g.kind, g.button = m.kind, m.button
		return g, nil
	}
	if g.zone != chartZonePlot {
		return g, fmt.Errorf("chart gesture %q: a key has no zone", s)
	}
	key, ok := shortcutKeys[name]
	if !ok {
		return g, fmt.Errorf("chart gesture %q: unknown gesture %q", s, parts[len(parts)-1])
	}
	g.kind, g.key = chartGestKey, key
	return g, nil
}
