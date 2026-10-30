// Package cad is a small CAD program: a drawing of a part, its views,
// sections and dimensions, drawn with lines, arcs and text that stay sharp
// at any zoom, and the part in 3D, cut by a plane. It shows what nui gives
// the programs that draw: View2D (zoom, pan, grid, crosshair, box
// selection, a picture kept while the overlay moves), SpatialIndex (what is
// seen, what is under the pointer), lines that do not thicken with the zoom,
// text as outlines, exact arcs; and in 3D the extrusion of a sketch, solids
// of revolution, section planes, outlines of curved sides and the axes
// gizmo.
//
// Only the drawing is here: the geometry a real CAD program edits, its
// files and its commands are not.
package cad

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// NewForm returns the CAD window
func NewForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("CAD")
	form.SetSize(1280, 820)
	a := newApp(form)
	form.Panel().AddWidget(0, 0, a)
	form.SetMenuBar(a.buildMenu())
	return form
}

// tool is what the left button does in the drawing
type tool int

const (
	toolSelect tool = iota
	toolLine
	toolCircle
	toolRect
	toolDimension
)

var toolNames = []string{"Select", "Line", "Circle", "Rectangle", "Dimension"}

// The prompts of the tools, for each point taken so far
var toolPrompts = map[tool][]string{
	toolSelect:    {"Click a shape, or drag a box: to the right - inside, to the left - crossing. Shift adds."},
	toolLine:      {"Line: the first point", "Line: the next point (right click to end)"},
	toolCircle:    {"Circle: the center", "Circle: a point on it"},
	toolRect:      {"Rectangle: a corner", "Rectangle: the opposite corner"},
	toolDimension: {"Dimension: the first point", "Dimension: the second point", "Dimension: where the line goes"},
}

type app struct {
	ui.Widget
	form *ui.Form

	doc  *drawing
	view *ui.View2D
	tabs *ui.TabWidget

	tool     tool
	pts      []ui.PointF // the points the tool took
	snapOn   bool
	ortho    bool
	snap     snapPoint
	hover    *shape
	selected map[*shape]bool
	dark     bool

	// The paths of the layers, reused from frame to frame
	strokes, fills []*ui.Path
	hatches        []*shape
	lastPaint      time.Duration

	toolActions             []*ui.Action
	snapAction, orthoAction *ui.Action
	gridAction              *ui.Action
	layerChecks             []*ui.Checkbox

	status                       *ui.StatusBar
	coords, modes, scale, counts *ui.Label

	// The 3D model
	view3d          *ui.View3D
	scene           *ui.Scene3D
	section         *ui.Checkbox
	sectionAt       *ui.Slider
	edges3d, ortho3 *ui.Checkbox
	picked          *ui.Object3D
}

func newApp(form *ui.Form) *app {
	a := &app{form: form, doc: newDrawing(), snapOn: true, dark: true, selected: map[*shape]bool{}}
	a.InitWidget()
	a.SetPanelPadding(0)
	a.SetCellPadding(0)
	for range a.doc.layers {
		a.strokes = append(a.strokes, ui.NewPath())
		a.fills = append(a.fills, ui.NewPath())
	}

	a.AddWidget(0, 0, a.buildToolbar())
	a.tabs = ui.NewTabWidget()
	a.tabs.AddPage("Drawing", a.buildDrawingPage())
	a.tabs.AddPage("3D model", a.build3DPage())
	a.AddWidget(1, 0, a.tabs)

	a.status = ui.NewStatusBar()
	a.coords = a.status.AddLabel("")
	a.modes = a.status.AddLabel("")
	a.scale = a.status.AddLabel("")
	a.counts = a.status.AddLabel("")
	a.AddWidget(2, 0, a.status)

	a.doc.addSample()
	a.view.SetExtents(a.doc.extents())
	a.view.SetView(ui.PointF{X: 210, Y: 148.5}, 2.4)
	a.setTool(toolSelect)
	a.applyTheme()
	a.updateStatus()
	// The time of the last paint, after it is painted
	a.AddTimer(500, a.updateStatus)
	// The view's size is known once laid out: then show the whole sheet
	form.Invoke(func() { a.view.ZoomExtents() })
	return a
}

// The drawing page: the view, and the layers beside it

func (a *app) buildDrawingPage() ui.Widgeter {
	page := ui.NewPanel()
	page.SetPanelPadding(0)
	a.view = ui.NewView2D()
	a.view.SetCrosshair(ui.CrosshairFull)
	a.view.SetBoxSelection(true)
	a.view.SetPainter(a.paint)
	a.view.SetOverlay(a.overlay)
	a.view.OnPointerMoved.Connect(a.pointerMoved)
	a.view.OnMouseDown.Connect(a.mouseDown)
	a.view.OnBoxSelected.Connect(a.boxSelected)
	a.view.OnViewChanged.Connect(func(ui.RectF) { a.updateStatus() })
	a.view.SetOnMouseLeave(func() {
		a.hover, a.snap = nil, snapPoint{}
		a.view.Update()
	})
	page.AddWidget(0, 0, a.view)

	side := page.AddPanel(0, 1)
	side.SetMinWidth(210)
	side.SetMaxWidth(210)
	layers := ui.NewGroupBox("Layers")
	side.AddWidget(0, 0, layers)
	for i, l := range a.doc.layers {
		c := ui.NewCheckbox(l.name)
		c.SetChecked(l.visible)
		c.SetOnStateChanged(func() {
			l.visible = c.Checked()
			a.view.Invalidate()
		})
		layers.AddWidget(i, 0, c)
		a.layerChecks = append(a.layerChecks, c)
	}
	side.AddLabel(1, 0, "Wheel: zoom   Middle: pan\nMiddle double click: all\nL C R D: tools   Esc: cancel\nF3 snap   F7 grid   F8 ortho\nDel: delete the selection")
	side.AddVSpacer(2, 0)
	return page
}

// The tools, as toolbar buttons and menu items

func (a *app) buildToolbar() ui.Widgeter {
	tb := ui.NewToolBar()
	shortcuts := []string{"Escape", "L", "C", "R", "D"}
	for i, name := range toolNames {
		t := tool(i)
		act := ui.NewAction(name, func() { a.setTool(t) }).
			SetIcon(toolIcon(t)).SetShortcut(shortcuts[i]).SetCheckable(true)
		a.toolActions = append(a.toolActions, act)
		a.form.AddAction(act)
		tb.AddAction(act)
	}
	tb.AddSeparator()
	extents := ui.NewAction("Zoom to all", func() { a.view.ZoomExtents() }).SetIcon(viewIcon(0)).SetShortcut("Home")
	window := ui.NewAction("Zoom to a box", func() { a.view.StartZoomWindow() }).SetIcon(viewIcon(1)).SetShortcut("Z")
	a.form.AddAction(window)
	tb.AddAction(extents)
	tb.AddAction(window)
	tb.AddSeparator()
	a.snapAction = ui.NewAction("Snap", func() { a.snapOn = a.snapAction.Checked(); a.updateStatus() }).
		SetShortcut("F3").SetCheckable(true).SetChecked(true)
	a.gridAction = ui.NewAction("Grid", func() {
		g := a.view.Grid()
		g.Visible = a.gridAction.Checked()
		a.view.SetGrid(g)
		a.updateStatus()
	}).SetShortcut("F7").SetCheckable(true).SetChecked(true)
	a.orthoAction = ui.NewAction("Ortho", func() { a.ortho = a.orthoAction.Checked(); a.updateStatus() }).
		SetShortcut("F8").SetCheckable(true)
	for _, act := range []*ui.Action{a.snapAction, a.gridAction, a.orthoAction} {
		a.form.AddAction(act)
		tb.AddAction(act)
	}
	del := ui.NewAction("Delete", a.deleteSelected).SetShortcut("Delete")
	a.form.AddAction(del)
	return tb
}

func (a *app) buildMenu() *ui.MenuBar {
	bar := ui.NewMenuBar()
	file := bar.AddMenu("&File")
	file.AddItem("&New drawing", func() {
		a.doc.clear()
		a.clearSelection()
		a.view.Invalidate()
	})
	file.AddItem("&Sample drawing", func() {
		a.doc.addSample()
		a.clearSelection()
		a.view.SetExtents(a.doc.extents())
		a.view.ZoomExtents()
		a.view.Invalidate()
	})
	file.AddSeparator()
	file.AddItem("Export &PNG...", func() { a.export("png") })
	file.AddItem("Export S&VG...", func() { a.export("svg") })
	edit := bar.AddMenu("&Edit")
	edit.AddItem("Select &all", a.selectAll).SetShortcut("Mod+A")
	edit.AddItem("&Delete", a.deleteSelected)
	draw := bar.AddMenu("&Draw")
	for _, act := range a.toolActions {
		draw.AddAction(act)
	}
	draw.AddSeparator()
	draw.AddItem("Add 10 000 shapes", func() { a.addMany(10000) })
	draw.AddItem("Add 100 000 shapes", func() { a.addMany(100000) })
	view := bar.AddMenu("&View")
	view.AddItem("Zoom to &all", a.view.ZoomExtents).SetShortcut("Home")
	view.AddItem("Zoom to a &box", a.view.StartZoomWindow)
	view.AddAction(a.gridAction)
	view.AddSeparator()
	view.AddItem("&Dark sheet", func() { a.dark = true; a.applyTheme() })
	view.AddItem("&Light sheet", func() { a.dark = false; a.applyTheme() })
	return bar
}

func (a *app) setTool(t tool) {
	if a.tool == toolSelect && t == toolSelect && len(a.pts) == 0 {
		a.clearSelection()
	}
	a.tool = t
	a.pts = nil
	for i, act := range a.toolActions {
		act.SetChecked(tool(i) == t)
	}
	a.hover = nil
	a.updateStatus()
	a.view.Update()
}

// applyTheme sets the colors of the sheet: dark as in most CAD programs,
// or white as paper
func (a *app) applyTheme() {
	if a.dark {
		a.view.SetBackground(color.RGBA{30, 34, 40, 255})
	} else {
		a.view.SetBackground(color.RGBA{255, 255, 255, 255})
	}
	a.view.Invalidate()
}

// ink is the color of the layers without one: white on the dark sheet,
// black on the light one
func (a *app) ink() color.RGBA {
	if a.dark {
		return color.RGBA{235, 235, 235, 255}
	}
	return color.RGBA{20, 20, 20, 255}
}

func (a *app) layerColor(l *layer) color.RGBA {
	if l.col.A == 0 {
		return a.ink()
	}
	return l.col
}

// The drawing

// paint draws the shapes of the part of the drawing to draw: those the
// index finds there, a path a layer, the lines as thick at any zoom
func (a *app) paint(p *ui.View2DPaint) {
	start := time.Now()
	for i := range a.strokes {
		a.strokes[i].Reset()
		a.fills[i].Reset()
	}
	a.hatches = a.hatches[:0]
	// Letters smaller than a pixel and a half are boxes, as CAD programs
	// draw them: their outlines would cost and show nothing
	minText := 1.5 * p.PixelSize()
	a.doc.index.Search(p.Visible, func(s *shape) bool {
		if !a.doc.layers[s.layer].visible {
			return true
		}
		fill := a.fills[s.layer]
		switch s.kind {
		case kText:
			if s.r < minText {
				b := s.bounds()
				fill.Rect(b.MinX, b.MinY, b.Width(), b.Height())
				return true
			}
			a.view.AddText(fill, s.p[0], s.text, ui.FontFamilySans, s.r, 0, ui.HAlignLeft, ui.VAlignBottom)
		case kDimension:
			g := s.dimGeometry()
			s.outlinePath(a.strokes[s.layer])
			for _, ar := range g.arrows {
				fill.MoveTo(ar[0].X, ar[0].Y)
				fill.LineTo(ar[1].X, ar[1].Y)
				fill.LineTo(ar[2].X, ar[2].Y)
				fill.Close()
			}
			if s.r >= minText {
				a.view.AddText(fill, g.textAt, g.label, ui.FontFamilySans, s.r, g.angle, ui.HAlignCenter, ui.VAlignBottom)
			}
		case kHatch:
			a.hatches = append(a.hatches, s)
		default:
			// A shape of a pixel or two is a dot: its lines would cost
			// the same and show nothing more
			if b := s.bounds(); math.Max(b.Width(), b.Height()) < 2*p.PixelSize() {
				c := b.Center()
				px := p.PixelSize()
				fill.Rect(c.X-px, c.Y-px, 2*px, 2*px)
				return true
			}
			s.outlinePath(a.strokes[s.layer])
		}
		return true
	})
	cnv := p.Canvas
	for _, s := range a.hatches {
		col := a.layerColor(a.doc.layers[s.layer])
		// The hatch is of the drawing: its lines 3 mm apart, at any zoom
		cnv.FillPathPaint(s.path, ui.HatchPaint{Color: col, Angle: math.Pi / 4, Spacing: 3, Width: math.Max(0.2, p.PixelSize())})
	}
	for i, l := range a.doc.layers {
		col := a.layerColor(l)
		cnv.StrokePath(a.strokes[i], ui.StrokeStyle{Width: l.width, Dash: l.dash, NonScaling: true}, col)
		cnv.FillPath(a.fills[i], col)
	}
	a.paintSelection(p)
	a.lastPaint = time.Since(start)
}

// paintSelection draws the selected shapes highlighted, with the grips of
// their points
func (a *app) paintSelection(p *ui.View2DPaint) {
	if len(a.selected) == 0 {
		return
	}
	sel := ui.NewPath()
	var grips []ui.PointF
	for s := range a.selected {
		if !s.bounds().Intersects(p.Visible) {
			continue
		}
		switch s.kind {
		case kText:
			b := s.bounds()
			sel.Rect(b.MinX, b.MinY, b.Width(), b.Height())
		case kHatch:
			sel.AddPath(s.path)
		default:
			s.outlinePath(sel)
		}
		for _, sp := range s.snaps() {
			grips = append(grips, sp.p)
		}
	}
	hl := color.RGBA{60, 140, 255, 255}
	p.Canvas.StrokePath(sel, ui.StrokeStyle{Width: 2, Dash: []float64{6, 3}, NonScaling: true}, hl)
	p.InScreen(func(cnv *ui.Canvas) {
		for _, g := range grips {
			sp := p.ToScreen(g)
			cnv.DrawMarker(ui.MarkerSquare, sp.X, sp.Y, 8, hl, color.RGBA{255, 255, 255, 255})
		}
	})
}

// overlay draws what follows the pointer: the shape under it, the shape
// being drawn, the snap
func (a *app) overlay(p *ui.View2DPaint) {
	cnv := p.Canvas
	if a.hover != nil && !a.selected[a.hover] {
		path := ui.NewPath()
		if a.hover.kind == kText || a.hover.kind == kHatch {
			b := a.hover.bounds()
			path.Rect(b.MinX, b.MinY, b.Width(), b.Height())
		} else {
			a.hover.outlinePath(path)
		}
		cnv.StrokePath(path, ui.StrokeStyle{Width: 4, NonScaling: true, Join: ui.LineJoinRound, Cap: ui.LineCapRound}, color.RGBA{60, 140, 255, 110})
	}
	cur, ok := a.cursor()
	if ok && len(a.pts) > 0 {
		rubber := ui.StrokeStyle{Width: 1.5, NonScaling: true}
		guide := ui.StrokeStyle{Width: 1, NonScaling: true, Dash: []float64{4, 4}}
		col := a.layerColor(a.doc.layers[laySketch])
		p0 := a.pts[0]
		label := ""
		path := ui.NewPath()
		switch a.tool {
		case toolLine:
			path.MoveTo(p0.X, p0.Y)
			path.LineTo(cur.X, cur.Y)
			l := math.Hypot(cur.X-p0.X, cur.Y-p0.Y)
			label = fmt.Sprintf("%s  %s°", formatMM(l), formatMM(math.Atan2(cur.Y-p0.Y, cur.X-p0.X)*180/math.Pi))
		case toolCircle:
			r := math.Hypot(cur.X-p0.X, cur.Y-p0.Y)
			path.Circle(p0.X, p0.Y, r)
			radius := ui.NewPath()
			radius.MoveTo(p0.X, p0.Y)
			radius.LineTo(cur.X, cur.Y)
			cnv.StrokePath(radius, guide, col)
			label = "R " + formatMM(r)
		case toolRect:
			path.Rect(math.Min(p0.X, cur.X), math.Min(p0.Y, cur.Y), math.Abs(cur.X-p0.X), math.Abs(cur.Y-p0.Y))
			label = formatMM(math.Abs(cur.X-p0.X)) + " × " + formatMM(math.Abs(cur.Y-p0.Y))
		case toolDimension:
			if len(a.pts) == 1 {
				path.MoveTo(p0.X, p0.Y)
				path.LineTo(cur.X, cur.Y)
				cnv.StrokePath(path, guide, col)
				path.Reset()
			} else {
				d := dimension(a.pts[0], a.pts[1], cur, "")
				d.outlinePath(path)
				label = d.dimGeometry().label
			}
		}
		cnv.StrokePath(path, rubber, col)
		if label != "" {
			p.InScreen(func(cnv *ui.Canvas) {
				s := p.ToScreen(cur)
				tag(cnv, s.X+16, s.Y+16, label, col)
			})
		}
	}
	if a.snap.kind != snapNone && ok {
		p.InScreen(func(cnv *ui.Canvas) {
			s := p.ToScreen(a.snap.p)
			mark := color.RGBA{255, 200, 0, 255}
			shape := map[snapKind]ui.MarkerShape{
				snapEnd: ui.MarkerSquare, snapMid: ui.MarkerTriangleUp, snapCenter: ui.MarkerCircle,
				snapQuadrant: ui.MarkerDiamond, snapIntersection: ui.MarkerCross, snapGrid: ui.MarkerPlus,
			}[a.snap.kind]
			cnv.DrawMarker(shape, s.X, s.Y, 13, color.RGBA{}, mark)
			tag(cnv, s.X+12, s.Y-26, snapNames[a.snap.kind], mark)
		})
	}
}

// tag draws a small label on a dark plate
func tag(cnv *ui.Canvas, x, y float64, text string, col color.RGBA) {
	cnv.SetFontSize(11)
	w, h, _ := ui.MeasureText("", 11, text)
	ix, iy := int(x), int(y)
	cnv.SetColor(color.RGBA{20, 22, 26, 210})
	cnv.FillRoundedRect(ix, iy, w+10, h+4, 3)
	cnv.SetColor(col)
	cnv.SetHAlign(ui.HAlignLeft)
	cnv.SetVAlign(ui.VAlignCenter)
	cnv.DrawText(ix+5, iy, w+4, h+4, text)
}

// The pointer

// cursor returns the point the tools take: the pointer snapped, kept
// square to the last point with Ortho; false if the pointer is away
func (a *app) cursor() (ui.PointF, bool) {
	p, ok := a.view.Pointer()
	if !ok {
		return p, false
	}
	if a.snap.kind != snapNone {
		return a.snap.p, true
	}
	if a.ortho && len(a.pts) > 0 {
		last := a.pts[len(a.pts)-1]
		if math.Abs(p.X-last.X) > math.Abs(p.Y-last.Y) {
			p.Y = last.Y
		} else {
			p.X = last.X
		}
	}
	return p, true
}

// tolerance is the distance in the drawing the pick box takes
func (a *app) tolerance() float64 {
	return float64(a.view.PickBox()+3) / a.view.Scale()
}

func (a *app) pointerMoved(p ui.PointF) {
	a.findSnap(p)
	a.hover = nil
	if a.tool == toolSelect {
		a.hover = a.pick(p)
	}
	a.coords.SetText(fmt.Sprintf("X %s   Y %s", formatMM(p.X), formatMM(p.Y)))
	a.view.Update()
}

// pick returns the shape nearest the point within the pick box
func (a *app) pick(p ui.PointF) *shape {
	tol := a.tolerance()
	var best *shape
	bestD := tol
	for _, s := range a.doc.index.Nearest(p, tol, 64) {
		if !a.doc.layers[s.layer].visible {
			continue
		}
		d := s.distance(p)
		// A hatch or a text takes the point only when no line does
		if s.kind == kHatch || s.kind == kText {
			d = tol * 0.99
		}
		if d < bestD {
			best, bestD = s, d
		}
	}
	return best
}

// findSnap finds the snap point near the pointer: the ends, the middles,
// the centers, the quadrants and the crossings of the shapes near it, the
// nearest of the most important kind
func (a *app) findSnap(p ui.PointF) {
	a.snap = snapPoint{}
	if !a.snapOn || a.tool == toolSelect {
		return
	}
	tol := a.tolerance() * 1.5
	near := a.doc.index.Nearest(p, tol, 32)
	best := math.Inf(1)
	consider := func(sp snapPoint) {
		d := math.Hypot(sp.p.X-p.X, sp.p.Y-p.Y)
		if d > tol {
			return
		}
		// The ends and the crossings win over the others nearby
		if sp.kind == snapEnd || sp.kind == snapIntersection {
			d *= 0.5
		}
		if d < best {
			best, a.snap = d, sp
		}
	}
	for i, s := range near {
		if !a.doc.layers[s.layer].visible {
			continue
		}
		for _, sp := range s.snaps() {
			consider(sp)
		}
		for _, t := range near[i+1:] {
			for _, x := range intersections(s, t) {
				consider(snapPoint{x, snapIntersection})
			}
		}
	}
}

func (a *app) mouseDown(e *ui.MouseEvent) {
	p, ok := a.cursor()
	if !ok {
		return
	}
	switch {
	case e.Button == ui.MouseButtonRight:
		// Ends the tool's shape, or the tool
		if len(a.pts) > 0 {
			a.pts = nil
		} else {
			a.setTool(toolSelect)
		}
		a.updateStatus()
		a.view.Update()
		e.Handled = true
	case e.Button != ui.MouseButtonLeft:
	case a.tool == toolSelect:
		hit := a.pick(p)
		if hit == nil {
			// The view drags a box
			if !e.Mods.Shift {
				a.clearSelection()
			}
			return
		}
		if !e.Mods.Shift {
			clear(a.selected)
		}
		if a.selected[hit] {
			delete(a.selected, hit)
		} else {
			a.selected[hit] = true
		}
		a.view.Invalidate()
		a.updateStatus()
		e.Handled = true
	default:
		a.addPoint(p)
		e.Handled = true
	}
}

// addPoint gives the tool a point; with all it needs, it adds the shape
func (a *app) addPoint(p ui.PointF) {
	a.pts = append(a.pts, p)
	p0 := a.pts[0]
	var added []*shape
	switch a.tool {
	case toolLine:
		if len(a.pts) == 2 {
			added = append(added, line(laySketch, p0.X, p0.Y, p.X, p.Y))
			// Go on from the end
			a.pts = []ui.PointF{p}
		}
	case toolCircle:
		if len(a.pts) == 2 {
			if r := math.Hypot(p.X-p0.X, p.Y-p0.Y); r > 0 {
				added = append(added, circle(laySketch, p0.X, p0.Y, r))
			}
			a.pts = nil
		}
	case toolRect:
		if len(a.pts) == 2 {
			for _, l := range [][4]float64{{p0.X, p0.Y, p.X, p0.Y}, {p.X, p0.Y, p.X, p.Y}, {p.X, p.Y, p0.X, p.Y}, {p0.X, p.Y, p0.X, p0.Y}} {
				added = append(added, line(laySketch, l[0], l[1], l[2], l[3]))
			}
			a.pts = nil
		}
	case toolDimension:
		if len(a.pts) == 3 {
			added = append(added, dimension(a.pts[0], a.pts[1], p, ""))
			a.pts = nil
		}
	}
	for _, s := range added {
		a.doc.add(s)
	}
	if len(added) > 0 {
		a.view.SetExtents(a.doc.extents())
		a.view.Invalidate()
	}
	a.updateStatus()
}

// The selection

func (a *app) boxSelected(b ui.BoxSelection) {
	if !b.Mods.Shift {
		clear(a.selected)
	}
	a.doc.index.Search(b.Rect, func(s *shape) bool {
		if !a.doc.layers[s.layer].visible {
			return true
		}
		if b.Rect.ContainsRect(s.bounds()) || b.Crossing && crosses(s, b.Rect) {
			a.selected[s] = true
		}
		return true
	})
	a.view.Invalidate()
	a.updateStatus()
}

// crosses reports whether the lines of the shape pass through the box
func crosses(s *shape, r ui.RectF) bool {
	switch s.kind {
	case kText, kHatch:
		return s.bounds().Intersects(r)
	case kLine:
		return segmentInRect(s.p[0], s.p[1], r)
	}
	// Along the curves, in short pieces
	p := ui.NewPath()
	s.outlinePath(p)
	n := 96
	length := p.Length()
	prev, _, _ := p.PointAtLength(0)
	for i := 1; i <= n; i++ {
		q, _, _ := p.PointAtLength(length * float64(i) / float64(n))
		if segmentInRect(prev, q, r) {
			return true
		}
		prev = q
	}
	return false
}

func segmentInRect(a, b ui.PointF, r ui.RectF) bool {
	if r.Contains(a) || r.Contains(b) {
		return true
	}
	corners := [4]ui.PointF{{X: r.MinX, Y: r.MinY}, {X: r.MaxX, Y: r.MinY}, {X: r.MaxX, Y: r.MaxY}, {X: r.MinX, Y: r.MaxY}}
	for i := range corners {
		if _, ok := segmentCross(a, b, corners[i], corners[(i+1)%4]); ok {
			return true
		}
	}
	return false
}

func (a *app) clearSelection() {
	if len(a.selected) > 0 {
		clear(a.selected)
		a.view.Invalidate()
	}
	a.updateStatus()
}

func (a *app) selectAll() {
	for _, s := range a.doc.shapes {
		if a.doc.layers[s.layer].visible {
			a.selected[s] = true
		}
	}
	a.view.Invalidate()
	a.updateStatus()
}

func (a *app) deleteSelected() {
	for s := range a.selected {
		a.doc.remove(s)
	}
	clear(a.selected)
	a.hover = nil
	a.view.Invalidate()
	a.updateStatus()
}

func (a *app) addMany(n int) {
	start := time.Now()
	a.doc.addMany(n)
	a.view.SetExtents(a.doc.extents())
	a.view.ZoomExtents()
	a.view.Invalidate()
	a.form.ShowToast(fmt.Sprintf("%d shapes added in %d ms", n, time.Since(start).Milliseconds()), ui.ToastInfo)
	a.updateStatus()
}

func (a *app) updateStatus() {
	if a.status == nil {
		return
	}
	prompts := toolPrompts[a.tool]
	a.status.SetText(prompts[min(len(a.pts), len(prompts)-1)])
	var modes []string
	for _, m := range []struct {
		on   bool
		name string
	}{{a.snapOn, "SNAP"}, {a.view.Grid().Visible, "GRID"}, {a.ortho, "ORTHO"}} {
		if m.on {
			modes = append(modes, m.name)
		}
	}
	a.modes.SetText(strings.Join(modes, " "))
	a.scale.SetText(fmt.Sprintf("%s px/mm", formatMM(a.view.Scale())))
	text := fmt.Sprintf("%d shapes", len(a.doc.shapes))
	if len(a.selected) > 0 {
		text += fmt.Sprintf(", %d selected", len(a.selected))
	}
	a.counts.SetText(fmt.Sprintf("%s   drawn in %.1f ms", text, float64(a.lastPaint.Microseconds())/1000))
}

// export writes the part of the drawing in the view to a file: a PNG twice
// as dense as the screen, or an SVG
func (a *app) export(format string) {
	opts := ui.SaveFileDialogOptions{
		Title:           "Export the drawing",
		DefaultFileName: "drawing." + format,
		Filters:         []ui.FileDialogFilter{{DisplayName: strings.ToUpper(format), Patterns: []string{"*." + format}}},
	}
	a.form.ShowSaveFileDialog(opts, func(path string, err error) {
		if errors.Is(err, ui.ErrNoFileDialog) || err != nil || path == "" {
			return
		}
		if err := a.writeExport(path, format); err != nil {
			ui.ShowMessageBox(a, "Export", err.Error())
		}
	})
}

func (a *app) writeExport(path, format string) error {
	w, h := a.view.Width(), a.view.Height()
	if format == "svg" {
		return os.WriteFile(path, []byte(a.view.RenderSVG(w, h)), 0o644)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, a.view.RenderImage(w, h, 2))
}

// The icons of the tools, drawn by the canvas

func iconCanvas(draw func(cnv *ui.Canvas, ink color.RGBA)) image.Image {
	cnv, img := ui.NewImageCanvas(18, 18, 1)
	draw(cnv, color.RGBA{90, 160, 230, 255})
	return img
}

func toolIcon(t tool) image.Image {
	return iconCanvas(func(cnv *ui.Canvas, ink color.RGBA) {
		p := ui.NewPath()
		st := ui.StrokeStyle{Width: 1.6, Join: ui.LineJoinRound, Cap: ui.LineCapRound}
		switch t {
		case toolSelect:
			p.Polygon([]ui.PointF{{X: 4, Y: 2}, {X: 4, Y: 15}, {X: 7.5, Y: 11.5}, {X: 10, Y: 16}, {X: 12, Y: 15}, {X: 9.5, Y: 10.5}, {X: 14, Y: 10.5}}...)
			cnv.FillPath(p, ink)
			return
		case toolLine:
			p.MoveTo(3, 15)
			p.LineTo(15, 3)
			cnv.StrokePath(p, st, ink)
			cnv.FillCircle(3, 15, 2, ink)
			cnv.FillCircle(15, 3, 2, ink)
			return
		case toolCircle:
			p.Circle(9, 9, 6.5)
			cnv.StrokePath(p, st, ink)
			cnv.FillCircle(9, 9, 1.5, ink)
			return
		case toolRect:
			p.Rect(2.5, 4.5, 13, 9)
		case toolDimension:
			p.MoveTo(2, 4)
			p.LineTo(2, 14)
			p.MoveTo(16, 4)
			p.LineTo(16, 14)
			p.MoveTo(2, 9)
			p.LineTo(16, 9)
			cnv.StrokePath(p, ui.StrokeStyle{Width: 1.2}, ink)
			arrows := ui.NewPath()
			arrows.Polygon([]ui.PointF{{X: 2, Y: 9}, {X: 6, Y: 7}, {X: 6, Y: 11}}...)
			arrows.Polygon([]ui.PointF{{X: 16, Y: 9}, {X: 12, Y: 7}, {X: 12, Y: 11}}...)
			cnv.FillPath(arrows, ink)
			return
		}
		cnv.StrokePath(p, st, ink)
	})
}

func viewIcon(kind int) image.Image {
	return iconCanvas(func(cnv *ui.Canvas, ink color.RGBA) {
		p := ui.NewPath()
		st := ui.StrokeStyle{Width: 1.6, Cap: ui.LineCapRound}
		if kind == 0 {
			// Four corners: all of it
			for _, c := range [][3]float64{{2, 2, 1}, {16, 2, -1}} {
				p.MoveTo(c[0]+5*c[2], c[1])
				p.LineTo(c[0], c[1])
				p.LineTo(c[0], c[1]+5)
				p.MoveTo(c[0]+5*c[2], 16)
				p.LineTo(c[0], 16)
				p.LineTo(c[0], 11)
			}
			cnv.StrokePath(p, st, ink)
			return
		}
		// A box and a lens
		p.Rect(2, 2, 9, 9)
		cnv.StrokePath(p, ui.StrokeStyle{Width: 1.2, Dash: []float64{2, 1.5}}, ink)
		q := ui.NewPath()
		q.Circle(11, 11, 3.5)
		q.MoveTo(13.5, 13.5)
		q.LineTo(16.5, 16.5)
		cnv.StrokePath(q, st, ink)
	})
}
