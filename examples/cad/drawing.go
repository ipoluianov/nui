package cad

import (
	"image/color"
	"math"
	"math/rand"
	"strconv"

	"github.com/ipoluianov/nui/ui"
)

// The drawing: shapes on layers, in millimeters, y up. Only what drawing
// them needs is here - the geometry of a real CAD program is much more.

type kind int

const (
	kLine kind = iota
	kCircle
	kArc
	kText
	kDimension
	kHatch
)

// shape is one thing of the drawing
type shape struct {
	kind  kind
	layer int
	// A line: from p[0] to p[1]. A circle, an arc: the center p[0]. A
	// text: where it starts, p[0]. A dimension: the points measured p[0]
	// and p[1], and p[2] on its line.
	p [3]ui.PointF
	// The radius of a circle and an arc; the height of the letters of a
	// text and of a dimension
	r float64
	// An arc goes counterclockwise from a0 to a1, radians
	a0, a1 float64
	text   string
	// A hatch: the boundary, its holes inside it
	path *ui.Path
}

// layer is a group of shapes drawn alike
type layer struct {
	name    string
	col     color.RGBA // zero: the ink, black or white as the background
	width   float64    // pixels
	dash    []float64  // pixels
	visible bool
}

func defaultLayers() []*layer {
	return []*layer{
		{name: "Outline", width: 2, visible: true},
		{name: "Center lines", col: color.RGBA{220, 80, 70, 255}, width: 1, dash: []float64{18, 4, 3, 4}, visible: true},
		{name: "Hidden", col: color.RGBA{90, 160, 230, 255}, width: 1, dash: []float64{7, 4}, visible: true},
		{name: "Dimensions", col: color.RGBA{40, 180, 170, 255}, width: 1, visible: true},
		{name: "Hatch", col: color.RGBA{150, 150, 160, 255}, width: 1, visible: true},
		{name: "Text", width: 1, visible: true},
		{name: "Frame", width: 1, visible: true},
		{name: "Sketch", col: color.RGBA{230, 190, 60, 255}, width: 1.5, visible: true},
	}
}

// The layers of the sample, by name
const (
	layOutline = iota
	layCenter
	layHidden
	layDims
	layHatch
	layText
	layFrame
	laySketch
)

// drawing is the shapes and their index
type drawing struct {
	shapes []*shape
	layers []*layer
	index  *ui.SpatialIndex[*shape]
}

func newDrawing() *drawing {
	return &drawing{layers: defaultLayers(), index: ui.NewSpatialIndex[*shape]()}
}

func (d *drawing) add(s *shape) *shape {
	d.shapes = append(d.shapes, s)
	d.index.Insert(s, s.bounds())
	return s
}

func (d *drawing) remove(s *shape) {
	for i, x := range d.shapes {
		if x == s {
			d.shapes = append(d.shapes[:i], d.shapes[i+1:]...)
			break
		}
	}
	d.index.Remove(s)
}

func (d *drawing) clear() {
	d.shapes = nil
	d.index.Clear()
}

// extents returns the box of all the shapes
func (d *drawing) extents() ui.RectF {
	r, _ := d.index.Bounds()
	return r
}

func line(layer int, x0, y0, x1, y1 float64) *shape {
	return &shape{kind: kLine, layer: layer, p: [3]ui.PointF{{X: x0, Y: y0}, {X: x1, Y: y1}}}
}

func circle(layer int, cx, cy, r float64) *shape {
	return &shape{kind: kCircle, layer: layer, p: [3]ui.PointF{{X: cx, Y: cy}}, r: r}
}

func arc(layer int, cx, cy, r, a0, a1 float64) *shape {
	return &shape{kind: kArc, layer: layer, p: [3]ui.PointF{{X: cx, Y: cy}}, r: r, a0: a0, a1: a1}
}

func text(layer int, x, y, height float64, s string) *shape {
	return &shape{kind: kText, layer: layer, p: [3]ui.PointF{{X: x, Y: y}}, r: height, text: s}
}

// dimension measures from a to b along the line through at, parallel to
// a-b; with label empty the distance is written
func dimension(a, b, at ui.PointF, label string) *shape {
	return &shape{kind: kDimension, layer: layDims, p: [3]ui.PointF{a, b, at}, r: 3.5, text: label}
}

// outlinePath adds the shape's lines and curves to the path; texts and
// hatches have none
func (s *shape) outlinePath(p *ui.Path) {
	switch s.kind {
	case kLine:
		p.MoveTo(s.p[0].X, s.p[0].Y)
		p.LineTo(s.p[1].X, s.p[1].Y)
	case kCircle:
		p.Circle(s.p[0].X, s.p[0].Y, s.r)
	case kArc:
		c := s.p[0]
		p.MoveTo(c.X+s.r*math.Cos(s.a0), c.Y+s.r*math.Sin(s.a0))
		p.Arc(c.X, c.Y, s.r, s.a0, s.a1)
	case kDimension:
		g := s.dimGeometry()
		for _, l := range g.lines {
			p.MoveTo(l[0].X, l[0].Y)
			p.LineTo(l[1].X, l[1].Y)
		}
	}
}

// dimGeometry is a dimension worked out: its lines, the arrows and the text
type dimGeom struct {
	lines  [][2]ui.PointF
	arrows [][3]ui.PointF
	textAt ui.PointF
	angle  float64
	label  string
}

func (s *shape) dimGeometry() dimGeom {
	a, b, at := s.p[0], s.p[1], s.p[2]
	dx, dy := b.X-a.X, b.Y-a.Y
	l := math.Hypot(dx, dy)
	if l == 0 {
		return dimGeom{}
	}
	ux, uy := dx/l, dy/l
	nx, ny := -uy, ux
	// The offset of the dimension line from a-b
	off := (at.X-a.X)*nx + (at.Y-a.Y)*ny
	sign := 1.0
	if off < 0 {
		sign = -1
	}
	da := ui.PointF{X: a.X + nx*off, Y: a.Y + ny*off}
	db := ui.PointF{X: b.X + nx*off, Y: b.Y + ny*off}
	const gap, over, arrowLen, arrowW = 1.5, 2.0, 3.5, 0.9
	g := dimGeom{label: s.text}
	if g.label == "" {
		g.label = formatMM(l)
	}
	// The extension lines: from a little off the part to a little past
	// the dimension line
	ext := func(p, d ui.PointF) [2]ui.PointF {
		return [2]ui.PointF{{X: p.X + nx*sign*gap, Y: p.Y + ny*sign*gap}, {X: d.X + nx*sign*over, Y: d.Y + ny*sign*over}}
	}
	g.lines = append(g.lines, ext(a, da), ext(b, db), [2]ui.PointF{da, db})
	arrow := func(tip ui.PointF, dir float64) [3]ui.PointF {
		bx, by := tip.X+ux*arrowLen*dir, tip.Y+uy*arrowLen*dir
		return [3]ui.PointF{tip, {X: bx + nx*arrowW, Y: by + ny*arrowW}, {X: bx - nx*arrowW, Y: by - ny*arrowW}}
	}
	g.arrows = append(g.arrows, arrow(da, 1), arrow(db, -1))
	g.angle = math.Atan2(uy, ux)
	// Read from below or from the right
	if g.angle > math.Pi/2+1e-9 || g.angle <= -math.Pi/2+1e-9 {
		g.angle += math.Pi
	}
	mid := ui.PointF{X: (da.X + db.X) / 2, Y: (da.Y + db.Y) / 2}
	// The text above the line (on the side away from the part)
	tx, ty := -math.Sin(g.angle), math.Cos(g.angle)
	g.textAt = ui.PointF{X: mid.X + tx*1, Y: mid.Y + ty*1}
	return g
}

// formatMM writes the length to a hundredth, without the zeros at the end
func formatMM(v float64) string {
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

// bounds returns the box of the shape
func (s *shape) bounds() ui.RectF {
	switch s.kind {
	case kLine:
		return ui.RectFromPoints(s.p[0], s.p[1])
	case kCircle:
		c := s.p[0]
		return ui.RectF{MinX: c.X - s.r, MinY: c.Y - s.r, MaxX: c.X + s.r, MaxY: c.Y + s.r}
	case kText:
		// p[0] is the lower left corner of the text's box
		w, asc, desc := ui.MeasureTextOutline(s.text, ui.FontFamilySans, s.r)
		return ui.RectF{MinX: s.p[0].X, MinY: s.p[0].Y, MaxX: s.p[0].X + w, MaxY: s.p[0].Y + asc + desc}
	case kHatch:
		x0, y0, x1, y1, _ := s.path.Bounds()
		return ui.RectF{MinX: x0, MinY: y0, MaxX: x1, MaxY: y1}
	}
	p := ui.NewPath()
	s.outlinePath(p)
	x0, y0, x1, y1, ok := p.Bounds()
	if !ok {
		return ui.EmptyRectF()
	}
	r := ui.RectF{MinX: x0, MinY: y0, MaxX: x1, MaxY: y1}
	if s.kind == kDimension {
		// The text above the line
		r = r.Inset(-s.r * 1.5)
	}
	return r
}

// distance returns how far the point is from the lines of the shape
func (s *shape) distance(q ui.PointF) float64 {
	switch s.kind {
	case kLine:
		return segmentDistance(q, s.p[0], s.p[1])
	case kCircle:
		return math.Abs(math.Hypot(q.X-s.p[0].X, q.Y-s.p[0].Y) - s.r)
	case kArc:
		c := s.p[0]
		a := math.Atan2(q.Y-c.Y, q.X-c.X)
		if angleIn(a, s.a0, s.a1) {
			return math.Abs(math.Hypot(q.X-c.X, q.Y-c.Y) - s.r)
		}
		e0, e1 := s.arcEnds()
		return math.Min(math.Hypot(q.X-e0.X, q.Y-e0.Y), math.Hypot(q.X-e1.X, q.Y-e1.Y))
	case kText:
		if s.bounds().Contains(q) {
			return 0
		}
		return math.Inf(1)
	case kDimension:
		g := s.dimGeometry()
		d := math.Inf(1)
		for _, l := range g.lines {
			d = math.Min(d, segmentDistance(q, l[0], l[1]))
		}
		return d
	case kHatch:
		if s.path.Contains(q.X, q.Y) {
			return 0
		}
	}
	return math.Inf(1)
}

func (s *shape) arcEnds() (ui.PointF, ui.PointF) {
	c := s.p[0]
	return ui.PointF{X: c.X + s.r*math.Cos(s.a0), Y: c.Y + s.r*math.Sin(s.a0)},
		ui.PointF{X: c.X + s.r*math.Cos(s.a1), Y: c.Y + s.r*math.Sin(s.a1)}
}

// angleIn reports whether the angle is on the arc from a0 counterclockwise
// to a1
func angleIn(a, a0, a1 float64) bool {
	sweep := math.Mod(a1-a0, 2*math.Pi)
	if sweep < 0 {
		sweep += 2 * math.Pi
	}
	d := math.Mod(a-a0, 2*math.Pi)
	if d < 0 {
		d += 2 * math.Pi
	}
	return d <= sweep+1e-12
}

func segmentDistance(q, a, b ui.PointF) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = math.Max(0, math.Min(1, ((q.X-a.X)*dx+(q.Y-a.Y)*dy)/l2))
	}
	return math.Hypot(q.X-a.X-t*dx, q.Y-a.Y-t*dy)
}

// snapKind is what a snap point of a shape is
type snapKind int

const (
	snapNone snapKind = iota
	snapEnd
	snapMid
	snapCenter
	snapQuadrant
	snapIntersection
	snapGrid
)

var snapNames = map[snapKind]string{
	snapEnd: "Endpoint", snapMid: "Midpoint", snapCenter: "Center",
	snapQuadrant: "Quadrant", snapIntersection: "Intersection", snapGrid: "Grid",
}

type snapPoint struct {
	p    ui.PointF
	kind snapKind
}

// snaps returns the points of the shape the pointer snaps to
func (s *shape) snaps() []snapPoint {
	switch s.kind {
	case kLine:
		a, b := s.p[0], s.p[1]
		return []snapPoint{{a, snapEnd}, {b, snapEnd}, {ui.PointF{X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2}, snapMid}}
	case kCircle:
		c := s.p[0]
		return []snapPoint{{c, snapCenter},
			{ui.PointF{X: c.X + s.r, Y: c.Y}, snapQuadrant}, {ui.PointF{X: c.X, Y: c.Y + s.r}, snapQuadrant},
			{ui.PointF{X: c.X - s.r, Y: c.Y}, snapQuadrant}, {ui.PointF{X: c.X, Y: c.Y - s.r}, snapQuadrant}}
	case kArc:
		e0, e1 := s.arcEnds()
		m := (s.a0 + s.a1) / 2
		if s.a1 < s.a0 {
			m += math.Pi
		}
		c := s.p[0]
		return []snapPoint{{c, snapCenter}, {e0, snapEnd}, {e1, snapEnd},
			{ui.PointF{X: c.X + s.r*math.Cos(m), Y: c.Y + s.r*math.Sin(m)}, snapMid}}
	}
	return nil
}

// intersections returns where the lines and the circles of the two shapes
// cross
func intersections(a, b *shape) []ui.PointF {
	if a.kind > b.kind {
		a, b = b, a
	}
	var out []ui.PointF
	switch {
	case a.kind == kLine && b.kind == kLine:
		if p, ok := segmentCross(a.p[0], a.p[1], b.p[0], b.p[1]); ok {
			out = append(out, p)
		}
	case a.kind == kLine && (b.kind == kCircle || b.kind == kArc):
		for _, p := range lineCircle(a.p[0], a.p[1], b.p[0], b.r) {
			if b.kind == kCircle || angleIn(math.Atan2(p.Y-b.p[0].Y, p.X-b.p[0].X), b.a0, b.a1) {
				out = append(out, p)
			}
		}
	case (a.kind == kCircle || a.kind == kArc) && (b.kind == kCircle || b.kind == kArc):
		for _, p := range circleCircle(a.p[0], a.r, b.p[0], b.r) {
			ok := true
			for _, s := range []*shape{a, b} {
				if s.kind == kArc && !angleIn(math.Atan2(p.Y-s.p[0].Y, p.X-s.p[0].X), s.a0, s.a1) {
					ok = false
				}
			}
			if ok {
				out = append(out, p)
			}
		}
	}
	return out
}

func segmentCross(a, b, c, d ui.PointF) (ui.PointF, bool) {
	r := ui.PointF{X: b.X - a.X, Y: b.Y - a.Y}
	s := ui.PointF{X: d.X - c.X, Y: d.Y - c.Y}
	den := r.X*s.Y - r.Y*s.X
	if math.Abs(den) < 1e-12 {
		return ui.PointF{}, false
	}
	t := ((c.X-a.X)*s.Y - (c.Y-a.Y)*s.X) / den
	u := ((c.X-a.X)*r.Y - (c.Y-a.Y)*r.X) / den
	if t < -1e-9 || t > 1+1e-9 || u < -1e-9 || u > 1+1e-9 {
		return ui.PointF{}, false
	}
	return ui.PointF{X: a.X + t*r.X, Y: a.Y + t*r.Y}, true
}

func lineCircle(a, b, c ui.PointF, r float64) []ui.PointF {
	dx, dy := b.X-a.X, b.Y-a.Y
	fx, fy := a.X-c.X, a.Y-c.Y
	A := dx*dx + dy*dy
	B := 2 * (fx*dx + fy*dy)
	C := fx*fx + fy*fy - r*r
	disc := B*B - 4*A*C
	if A == 0 || disc < 0 {
		return nil
	}
	sq := math.Sqrt(disc)
	var out []ui.PointF
	for _, t := range []float64{(-B - sq) / (2 * A), (-B + sq) / (2 * A)} {
		if t >= -1e-9 && t <= 1+1e-9 {
			out = append(out, ui.PointF{X: a.X + t*dx, Y: a.Y + t*dy})
		}
	}
	return out
}

func circleCircle(c0 ui.PointF, r0 float64, c1 ui.PointF, r1 float64) []ui.PointF {
	d := math.Hypot(c1.X-c0.X, c1.Y-c0.Y)
	if d == 0 || d > r0+r1 || d < math.Abs(r0-r1) {
		return nil
	}
	a := (r0*r0 - r1*r1 + d*d) / (2 * d)
	h := math.Sqrt(math.Max(0, r0*r0-a*a))
	mx, my := c0.X+a*(c1.X-c0.X)/d, c0.Y+a*(c1.Y-c0.Y)/d
	ox, oy := -(c1.Y-c0.Y)/d*h, (c1.X-c0.X)/d*h
	return []ui.PointF{{X: mx + ox, Y: my + oy}, {X: mx - ox, Y: my - oy}}
}

// The sample: a flange plate, its front view and its section, the
// dimensions, on an A3 sheet with a title block

// The plate: its size, the radius of its corners, the center hole and the
// four holes for the bolts
const (
	plateW, plateH, plateR = 160.0, 100.0, 10.0
	plateT                 = 12.0
	boreR                  = 20.0
	boltR, boltX, boltY    = 6.0, 60.0, 30.0
)

// The places of the views on the sheet
var (
	frontAt   = ui.PointF{X: 130, Y: 170}
	sectionAt = ui.PointF{X: 270, Y: 170}
)

func (d *drawing) addSample() {
	d.clear()
	// The sheet: A3, a frame 10 mm in, the title block
	d.addRect(layFrame, 0, 0, 420, 297)
	d.addRect(layFrame, 10, 10, 400, 277)
	d.addRect(layFrame, 230, 10, 180, 40)
	d.add(line(layFrame, 230, 30, 410, 30))
	d.add(line(layFrame, 330, 10, 330, 30))
	d.add(text(layText, 236, 36, 7, "FLANGE PLATE"))
	d.add(text(layText, 236, 18, 3.5, "Material: steel S235"))
	d.add(text(layText, 336, 18, 3.5, "Scale 1:1   Sheet 1/1"))
	d.add(text(layText, 336, 23, 2.5, "nui CAD example"))

	// The front view: the outline with rounded corners and the holes
	x0, y0 := frontAt.X-plateW/2, frontAt.Y-plateH/2
	x1, y1 := frontAt.X+plateW/2, frontAt.Y+plateH/2
	r := plateR
	d.add(line(layOutline, x0+r, y0, x1-r, y0))
	d.add(line(layOutline, x1, y0+r, x1, y1-r))
	d.add(line(layOutline, x1-r, y1, x0+r, y1))
	d.add(line(layOutline, x0, y1-r, x0, y0+r))
	d.add(arc(layOutline, x1-r, y0+r, r, -math.Pi/2, 0))
	d.add(arc(layOutline, x1-r, y1-r, r, 0, math.Pi/2))
	d.add(arc(layOutline, x0+r, y1-r, r, math.Pi/2, math.Pi))
	d.add(arc(layOutline, x0+r, y0+r, r, math.Pi, 3*math.Pi/2))
	d.add(circle(layOutline, frontAt.X, frontAt.Y, boreR))
	for _, sx := range []float64{-1, 1} {
		for _, sy := range []float64{-1, 1} {
			cx, cy := frontAt.X+sx*boltX, frontAt.Y+sy*boltY
			d.add(circle(layOutline, cx, cy, boltR))
			d.add(line(layCenter, cx-boltR-3, cy, cx+boltR+3, cy))
			d.add(line(layCenter, cx, cy-boltR-3, cx, cy+boltR+3))
		}
	}
	d.add(line(layCenter, x0-6, frontAt.Y, x1+6, frontAt.Y))
	d.add(line(layCenter, frontAt.X, y0-6, frontAt.X, y1+6))
	// The circle of the bolts, a center line too
	d.add(circle(layCenter, frontAt.X, frontAt.Y, math.Hypot(boltX, boltY)))

	// The section A-A through the middle: the plate cut, the holes empty
	sx0, sx1 := sectionAt.X-plateT/2, sectionAt.X+plateT/2
	sy0, sy1 := sectionAt.Y-plateH/2, sectionAt.Y+plateH/2
	d.addRect(layOutline, sx0, sy0, plateT, plateH)
	for _, cy := range []float64{-boltY, boltY} {
		d.add(line(layOutline, sx0, sectionAt.Y+cy-boltR, sx1, sectionAt.Y+cy-boltR))
		d.add(line(layOutline, sx0, sectionAt.Y+cy+boltR, sx1, sectionAt.Y+cy+boltR))
		d.add(line(layCenter, sx0-4, sectionAt.Y+cy, sx1+4, sectionAt.Y+cy))
	}
	d.add(line(layOutline, sx0, sectionAt.Y-boreR, sx1, sectionAt.Y-boreR))
	d.add(line(layOutline, sx0, sectionAt.Y+boreR, sx1, sectionAt.Y+boreR))
	d.add(line(layCenter, sx0-6, sectionAt.Y, sx1+6, sectionAt.Y))
	// The cut material: the plate less the holes
	hatch := ui.NewPath()
	hatch.Rect(sx0, sy0, plateT, plateH)
	for _, band := range [][2]float64{{-boltY - boltR, -boltY + boltR}, {-boreR, boreR}, {boltY - boltR, boltY + boltR}} {
		hatch.Rect(sx0, sectionAt.Y+band[0], plateT, band[1]-band[0])
	}
	hatch.SetFillRule(ui.FillRuleEvenOdd)
	d.add(&shape{kind: kHatch, layer: layHatch, path: hatch})
	d.add(text(layText, sectionAt.X-6, sy1+8, 5, "A-A"))
	// The cutting line on the front view
	d.add(line(layCenter, frontAt.X+20, y1+10, frontAt.X+20, y1+4))
	d.add(text(layText, frontAt.X+22, y1+6, 4, "A"))

	// The hidden edges of the holes, seen from the side, on the front view's
	// right: no - on the section they are cut. A top view would show them:
	// a strip under the front view
	tx0, ty := x0, frontAt.Y-plateH/2-40
	d.addRect(layOutline, tx0, ty, plateW, plateT)
	for _, hx := range []float64{-boltX, boltX} {
		d.add(line(layHidden, frontAt.X+hx-boltR, ty, frontAt.X+hx-boltR, ty+plateT))
		d.add(line(layHidden, frontAt.X+hx+boltR, ty, frontAt.X+hx+boltR, ty+plateT))
	}
	d.add(line(layHidden, frontAt.X-boreR, ty, frontAt.X-boreR, ty+plateT))
	d.add(line(layHidden, frontAt.X+boreR, ty, frontAt.X+boreR, ty+plateT))
	d.add(line(layCenter, frontAt.X, ty-5, frontAt.X, ty+plateT+5))

	// The dimensions
	d.add(dimension(ui.PointF{X: x0, Y: y1}, ui.PointF{X: x1, Y: y1}, ui.PointF{X: x0, Y: y1 + 18}, ""))
	d.add(dimension(ui.PointF{X: frontAt.X - boltX, Y: frontAt.Y + boltY}, ui.PointF{X: frontAt.X + boltX, Y: frontAt.Y + boltY},
		ui.PointF{X: x0, Y: y1 + 10}, ""))
	d.add(dimension(ui.PointF{X: x0, Y: y0}, ui.PointF{X: x0, Y: y1}, ui.PointF{X: x0 - 16, Y: y0}, ""))
	d.add(dimension(ui.PointF{X: frontAt.X - boltX, Y: frontAt.Y - boltY}, ui.PointF{X: frontAt.X - boltX, Y: frontAt.Y + boltY},
		ui.PointF{X: x0 - 8, Y: y0}, ""))
	d.add(dimension(ui.PointF{X: sx0, Y: sy0}, ui.PointF{X: sx1, Y: sy0}, ui.PointF{X: sx0, Y: sy0 - 8}, ""))
	d.add(dimension(ui.PointF{X: sx1, Y: sectionAt.Y - boreR}, ui.PointF{X: sx1, Y: sectionAt.Y + boreR},
		ui.PointF{X: sx1 + 12, Y: 0}, "Ø40"))
	d.add(text(layDims, frontAt.X+boltX+8, frontAt.Y+boltY+8, 3.5, "4× Ø12"))
	d.add(text(layDims, x1-4, y0-7, 3.5, "R10"))
}

// addRect adds the four lines of a rectangle
func (d *drawing) addRect(layer int, x, y, w, h float64) {
	d.add(line(layer, x, y, x+w, y))
	d.add(line(layer, x+w, y, x+w, y+h))
	d.add(line(layer, x+w, y+h, x, y+h))
	d.add(line(layer, x, y+h, x, y))
}

// addMany adds n shapes at random over a large area: a test of speed
func (d *drawing) addMany(n int) {
	rng := rand.New(rand.NewSource(int64(len(d.shapes))))
	size := math.Sqrt(float64(n)) * 12
	for i := 0; i < n; i++ {
		x, y := 500+rng.Float64()*size, rng.Float64()*size
		switch rng.Intn(4) {
		case 0, 1:
			a := rng.Float64() * 2 * math.Pi
			l := 2 + rng.Float64()*10
			d.add(line(laySketch, x, y, x+l*math.Cos(a), y+l*math.Sin(a)))
		case 2:
			d.add(circle(layOutline, x, y, 1+rng.Float64()*4))
		default:
			a := rng.Float64() * 2 * math.Pi
			d.add(arc(layHidden, x, y, 2+rng.Float64()*5, a, a+1+rng.Float64()*3))
		}
	}
}
