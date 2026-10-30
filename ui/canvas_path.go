package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	"golang.org/x/image/vector"
)

// PointF is a point with fractional logical coordinates
type PointF struct {
	X, Y float64
}

// Path is a shape made of lines, curves and arcs, drawn by Canvas.FillPath
// and Canvas.StrokePath. Its coordinates are logical and fractional, and
// they are the edges of the pixels, as in SVG: the pixel (2, 3) is the
// square from (2, 3) to (3, 4), so a line one wide along y = 3.5 covers a
// row of whole pixels, and a filled rectangle from (2, 3) is sharp.
//
//	p := ui.NewPath()
//	p.MoveTo(10, 10)
//	p.LineTo(60, 40)
//	p.QuadTo(80, 0, 100, 40)
//	cnv.StrokePath(p, ui.StrokeStyle{Width: 2, Join: ui.LineJoinRound}, col)
//
// A path is a list of subpaths: MoveTo starts one, Close closes it. A
// path may be reused: Reset empties it, keeping its memory.
type Path struct {
	ops []pathOp
	// The current point and the start of the current subpath
	cur, start PointF
	hasCur     bool
	fillRule   FillRule
}

// FillRule is how Canvas.FillPath decides what is inside a path whose
// subpaths overlap or which crosses itself
type FillRule int

const (
	// FillRuleNonZero fills a point that the path winds around more times
	// one way than the other: a subpath drawn the other way round inside
	// another one makes a hole, one drawn the same way does not
	FillRuleNonZero FillRule = iota
	// FillRuleEvenOdd fills a point that a ray from it crosses the path an
	// odd number of times: every subpath inside another makes a hole,
	// whatever its direction (SVG fill-rule="evenodd")
	FillRuleEvenOdd
)

// SetFillRule sets how the path is filled where it overlaps itself
func (p *Path) SetFillRule(rule FillRule) { p.fillRule = rule }

// FillRule returns how the path is filled where it overlaps itself
func (p *Path) FillRule() FillRule { return p.fillRule }

type pathOpKind uint8

const (
	pathMoveTo pathOpKind = iota
	pathLineTo
	pathQuadTo
	pathCubicTo
	pathClose
	// pathArc is an arc of an ellipse: the points p[0] + p[1]*cos(t) +
	// p[2]*sin(t) for t from a[0] to a[0]+a[1]. p[1] and p[2] are two
	// conjugate radii, so a transformed arc is an arc too, exact.
	pathArc
)

type pathOp struct {
	kind pathOpKind
	p    [3]PointF
	a    [2]float64 // pathArc: the start and the sweep, radians
}

// NewPath makes an empty path
func NewPath() *Path {
	return &Path{}
}

// Reset empties the path, keeping its memory for the next shape; the fill
// rule goes back to FillRuleNonZero
func (p *Path) Reset() {
	p.ops = p.ops[:0]
	p.hasCur = false
	p.fillRule = FillRuleNonZero
}

// Empty reports whether the path has nothing to draw
func (p *Path) Empty() bool {
	for _, op := range p.ops {
		if op.kind != pathMoveTo && op.kind != pathClose {
			return false
		}
	}
	return true
}

// MoveTo starts a new subpath at (x, y)
func (p *Path) MoveTo(x, y float64) {
	pt := PointF{x, y}
	p.ops = append(p.ops, pathOp{kind: pathMoveTo, p: [3]PointF{pt}})
	p.cur, p.start, p.hasCur = pt, pt, true
}

// ensureCur starts a subpath at pt when there is no current point: a line
// or a curve added first starts where it starts, as in SVG
func (p *Path) ensureCur(pt PointF) {
	if !p.hasCur {
		p.MoveTo(pt.X, pt.Y)
	}
}

// LineTo adds a straight line from the current point to (x, y)
func (p *Path) LineTo(x, y float64) {
	pt := PointF{x, y}
	p.ensureCur(pt)
	p.ops = append(p.ops, pathOp{kind: pathLineTo, p: [3]PointF{pt}})
	p.cur = pt
}

// QuadTo adds a quadratic Bézier curve from the current point to (x, y)
// with the control point (cx, cy)
func (p *Path) QuadTo(cx, cy, x, y float64) {
	p.ensureCur(PointF{cx, cy})
	pt := PointF{x, y}
	p.ops = append(p.ops, pathOp{kind: pathQuadTo, p: [3]PointF{{cx, cy}, pt}})
	p.cur = pt
}

// CubicTo adds a cubic Bézier curve from the current point to (x, y) with
// the control points (c1x, c1y) and (c2x, c2y)
func (p *Path) CubicTo(c1x, c1y, c2x, c2y, x, y float64) {
	p.ensureCur(PointF{c1x, c1y})
	pt := PointF{x, y}
	p.ops = append(p.ops, pathOp{kind: pathCubicTo, p: [3]PointF{{c1x, c1y}, {c2x, c2y}, pt}})
	p.cur = pt
}

// Close closes the current subpath with a line back to its start; the
// next line starts there too
func (p *Path) Close() {
	if !p.hasCur {
		return
	}
	p.ops = append(p.ops, pathOp{kind: pathClose})
	p.cur = p.start
}

// Arc adds an arc of the circle centered at (cx, cy) from the angle start
// to the angle end, in radians: 0 points right, and the angles grow
// clockwise on the screen (y goes down), so from 0 to math.Pi/2 the arc
// goes from the right to the bottom. With end below start it goes the other
// way. The arc is joined to the current point by a line, if there is one.
func (p *Path) Arc(cx, cy, r, start, end float64) {
	p.EllipticalArc(cx, cy, r, r, start, end)
}

// EllipticalArc adds an arc of the ellipse centered at (cx, cy) with the
// radii rx and ry, as Arc
func (p *Path) EllipticalArc(cx, cy, rx, ry, start, end float64) {
	x0, y0 := cx+rx*math.Cos(start), cy+ry*math.Sin(start)
	if p.hasCur {
		p.LineTo(x0, y0)
	} else {
		p.MoveTo(x0, y0)
	}
	sweep := end - start
	if sweep == 0 || math.IsNaN(sweep) {
		return
	}
	// More than a few turns draw nothing more
	sweep = max(-maxArcSweep, min(sweep, maxArcSweep))
	// Kept as an arc, not as curves near it: it is exact at any zoom - the
	// arcs of a drawing zoomed far in stay on their centers and tangents
	end = start + sweep
	pt := PointF{cx + rx*math.Cos(end), cy + ry*math.Sin(end)}
	p.ops = append(p.ops, pathOp{kind: pathArc, p: [3]PointF{{cx, cy}, {rx, 0}, {0, ry}}, a: [2]float64{start, sweep}})
	p.cur = pt
}

// maxArcSweep is the largest sweep of an arc: four turns
const maxArcSweep = 8 * math.Pi

// arcPoint returns the point of the arc op at the angle t
func arcPoint(c, u, v PointF, t float64) PointF {
	cos, sin := math.Cos(t), math.Sin(t)
	return PointF{c.X + u.X*cos + v.X*sin, c.Y + u.Y*cos + v.Y*sin}
}

// arcCubics calls f with cubic curves near the arc op, each of up to an
// eighth of a turn (within 0.0004% of its radius): for SVG
func arcCubics(op pathOp, f func(c1, c2, e PointF)) {
	c, u, v := op.p[0], op.p[1], op.p[2]
	start, sweep := op.a[0], op.a[1]
	n := max(1, int(math.Ceil(math.Abs(sweep)/(math.Pi/4))))
	step := sweep / float64(n)
	k := 4.0 / 3.0 * math.Tan(step/4)
	a := start
	for i := 0; i < n; i++ {
		b := a + step
		p0, p1 := arcPoint(c, u, v, a), arcPoint(c, u, v, b)
		// The tangents: the derivatives at a and b
		d0 := PointF{-u.X*math.Sin(a) + v.X*math.Cos(a), -u.Y*math.Sin(a) + v.Y*math.Cos(a)}
		d1 := PointF{-u.X*math.Sin(b) + v.X*math.Cos(b), -u.Y*math.Sin(b) + v.Y*math.Cos(b)}
		f(PointF{p0.X + k*d0.X, p0.Y + k*d0.Y}, PointF{p1.X - k*d1.X, p1.Y - k*d1.Y}, p1)
		a = b
	}
}

// Circle adds a closed circle centered at (cx, cy)
func (p *Path) Circle(cx, cy, r float64) {
	p.Ellipse(cx, cy, r, r)
}

// Ellipse adds a closed ellipse centered at (cx, cy) with the radii rx, ry
func (p *Path) Ellipse(cx, cy, rx, ry float64) {
	p.MoveTo(cx+rx, cy)
	p.EllipticalArc(cx, cy, rx, ry, 0, 2*math.Pi)
	p.Close()
}

// Rect adds a closed rectangle
func (p *Path) Rect(x, y, w, h float64) {
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
}

// RoundedRect adds a closed rectangle with corners of the radius, kept
// within half of the shorter side
func (p *Path) RoundedRect(x, y, w, h, r float64) {
	r = max(0, min(r, math.Abs(w)/2, math.Abs(h)/2))
	if r == 0 {
		p.Rect(x, y, w, h)
		return
	}
	p.MoveTo(x+r, y)
	p.Arc(x+w-r, y+r, r, -math.Pi/2, 0)
	p.Arc(x+w-r, y+h-r, r, 0, math.Pi/2)
	p.Arc(x+r, y+h-r, r, math.Pi/2, math.Pi)
	p.Arc(x+r, y+r, r, math.Pi, 3*math.Pi/2)
	p.Close()
}

// Polygon adds a closed polygon through the points
func (p *Path) Polygon(pts ...PointF) {
	if len(pts) == 0 {
		return
	}
	p.Polyline(pts...)
	p.Close()
}

// Polyline adds an open line through the points, starting a new subpath
func (p *Path) Polyline(pts ...PointF) {
	if len(pts) == 0 {
		return
	}
	p.MoveTo(pts[0].X, pts[0].Y)
	for _, pt := range pts[1:] {
		p.LineTo(pt.X, pt.Y)
	}
}

// Sector adds a closed sector of a ring centered at (cx, cy) between the
// radii inner and outer and the angles start and end (as Arc): a slice of
// a pie with inner 0, of a donut or a gauge with inner above 0
func (p *Path) Sector(cx, cy, inner, outer, start, end float64) {
	p.MoveTo(cx+outer*math.Cos(start), cy+outer*math.Sin(start))
	p.Arc(cx, cy, outer, start, end)
	if inner > 0 {
		p.Arc(cx, cy, inner, end, start)
	} else {
		p.LineTo(cx, cy)
	}
	p.Close()
}

// Bounds returns the rectangle that holds the points of the path, the
// control points of the curves included; ok is false for an empty path
func (p *Path) Bounds() (minX, minY, maxX, maxY float64, ok bool) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, op := range p.ops {
		n := 0
		switch op.kind {
		case pathMoveTo, pathLineTo:
			n = 1
		case pathQuadTo:
			n = 2
		case pathCubicTo:
			n = 3
		case pathArc:
			x0, y0, x1, y1 := arcBounds(op)
			minX, minY = math.Min(minX, x0), math.Min(minY, y0)
			maxX, maxY = math.Max(maxX, x1), math.Max(maxY, y1)
		}
		for _, pt := range op.p[:n] {
			minX, minY = math.Min(minX, pt.X), math.Min(minY, pt.Y)
			maxX, maxY = math.Max(maxX, pt.X), math.Max(maxY, pt.Y)
		}
	}
	return minX, minY, maxX, maxY, minX <= maxX
}

// arcBounds returns the box of the arc op: its ends and the extremes it
// passes
func arcBounds(op pathOp) (minX, minY, maxX, maxY float64) {
	c, u, v := op.p[0], op.p[1], op.p[2]
	start, sweep := op.a[0], op.a[1]
	from, to := start, start+sweep
	if sweep < 0 {
		from, to = to, from
	}
	p := arcPoint(c, u, v, from)
	minX, minY, maxX, maxY = p.X, p.Y, p.X, p.Y
	add := func(t float64) {
		p := arcPoint(c, u, v, t)
		minX, minY = math.Min(minX, p.X), math.Min(minY, p.Y)
		maxX, maxY = math.Max(maxX, p.X), math.Max(maxY, p.Y)
	}
	add(to)
	// x and y are extreme where their derivatives are 0
	for _, t0 := range []float64{math.Atan2(v.X, u.X), math.Atan2(v.Y, u.Y)} {
		for _, t := range []float64{t0, t0 + math.Pi} {
			// The first angle of t + 2*pi*k at or after from
			t += 2 * math.Pi * math.Ceil((from-t)/(2*math.Pi))
			for ; t <= to; t += 2 * math.Pi {
				add(t)
			}
		}
	}
	return minX, minY, maxX, maxY
}

// flatPath is a path flattened to polylines in pixels of the image
type flatPath struct {
	pts []PointF
	// The subpaths: pts[subs[i].from:subs[i].to]
	subs []flatSub
	// culling: the curves out of cull (minX, minY, maxX, maxY, in pixels)
	// are lines, and the long ones are split until what is in it is
	// accurate. Set before a flatten, which clears it.
	culling bool
	cull    [4]float64
}

type flatSub struct {
	from, to int
	closed   bool
}

// flattenTolerance is how far, in pixels, the lines a curve is flattened
// to may be from it
const flattenTolerance = 0.05

// flatten turns the path into polylines in pixels: (x, y) logical becomes
// (ox + x*scale, oy + y*scale). Curves are split by Wang's formula into
// lines within flattenTolerance of them.
func (p *Path) flatten(fp *flatPath, ox, oy, scale float64) {
	defer func() { fp.culling = false }()
	fp.pts = fp.pts[:0]
	fp.subs = fp.subs[:0]
	tr := func(pt PointF) PointF { return PointF{ox + pt.X*scale, oy + pt.Y*scale} }
	// The subpath being flattened starts at subStart; -1 - none yet. A
	// subpath of a MoveTo alone draws nothing, as in SVG.
	subStart := -1
	var cur PointF
	endSub := func(closed bool) {
		if subStart >= 0 && len(fp.pts)-subStart > 1 {
			fp.subs = append(fp.subs, flatSub{from: subStart, to: len(fp.pts), closed: closed})
		} else if subStart >= 0 {
			fp.pts = fp.pts[:subStart]
		}
		subStart = -1
	}
	// beginSeg starts the subpath at the current point before a line or a
	// curve, after a Close
	beginSeg := func() {
		if subStart < 0 {
			subStart = len(fp.pts)
			fp.pts = append(fp.pts, cur)
		}
	}
	for _, op := range p.ops {
		switch op.kind {
		case pathMoveTo:
			endSub(false)
			cur = tr(op.p[0])
			subStart = len(fp.pts)
			fp.pts = append(fp.pts, cur)
		case pathLineTo:
			beginSeg()
			cur = tr(op.p[0])
			fp.pts = append(fp.pts, cur)
		case pathQuadTo:
			beginSeg()
			c, e := tr(op.p[0]), tr(op.p[1])
			ddx, ddy := cur.X-2*c.X+e.X, cur.Y-2*c.Y+e.Y
			if fp.culling && rawCurveSteps(math.Hypot(ddx, ddy)/4) > maxCurveSteps {
				// As a cubic, split where it is seen
				fp.cubicCulled(cur, PointF{cur.X + 2*(c.X-cur.X)/3, cur.Y + 2*(c.Y-cur.Y)/3},
					PointF{e.X + 2*(c.X-e.X)/3, e.Y + 2*(c.Y-e.Y)/3}, e, 0)
				cur = e
				continue
			}
			n := curveSteps(math.Hypot(ddx, ddy) / 4)
			for i := 1; i <= n; i++ {
				t := float64(i) / float64(n)
				u := 1 - t
				fp.pts = append(fp.pts, PointF{
					u*u*cur.X + 2*u*t*c.X + t*t*e.X,
					u*u*cur.Y + 2*u*t*c.Y + t*t*e.Y,
				})
			}
			cur = e
		case pathCubicTo:
			beginSeg()
			c1, c2, e := tr(op.p[0]), tr(op.p[1]), tr(op.p[2])
			d1 := math.Hypot(cur.X-2*c1.X+c2.X, cur.Y-2*c1.Y+c2.Y)
			d2 := math.Hypot(c1.X-2*c2.X+e.X, c1.Y-2*c2.Y+e.Y)
			if fp.culling && rawCurveSteps(0.75*math.Max(d1, d2)) > maxCurveSteps {
				fp.cubicCulled(cur, c1, c2, e, 0)
				cur = e
				continue
			}
			n := curveSteps(0.75 * math.Max(d1, d2))
			for i := 1; i <= n; i++ {
				t := float64(i) / float64(n)
				u := 1 - t
				a, b, cc, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
				fp.pts = append(fp.pts, PointF{
					a*cur.X + b*c1.X + cc*c2.X + d*e.X,
					a*cur.Y + b*c1.Y + cc*c2.Y + d*e.Y,
				})
			}
			cur = e
		case pathArc:
			beginSeg()
			c := tr(op.p[0])
			u := PointF{op.p[1].X * scale, op.p[1].Y * scale}
			v := PointF{op.p[2].X * scale, op.p[2].Y * scale}
			fp.arc(c, u, v, op.a[0], op.a[1])
			cur = fp.pts[len(fp.pts)-1]
		case pathClose:
			if subStart >= 0 {
				// A line after Close goes on from the start, in a new subpath
				cur = fp.pts[subStart]
				endSub(true)
			}
		}
	}
	endSub(false)
}

// curveSteps returns how many lines a curve is split into, for the
// measure of its bend m of Wang's formula: sqrt(m / tolerance)
func curveSteps(m float64) int {
	return min(rawCurveSteps(m), maxCurveSteps)
}

// maxCurveSteps is the most lines a curve is split into: a curve longer
// than that is not accurate, unless the flattening is culled (then it is
// split where it is seen, see cubicCulled)
const maxCurveSteps = 1000

// rawCurveSteps is curveSteps without the limit
func rawCurveSteps(m float64) int {
	if math.IsNaN(m) || m <= 0 {
		return 1
	}
	return int(max(1, min(math.Ceil(math.Sqrt(m/flattenTolerance)), 1<<30)))
}

// arcSteps returns how many lines an arc of the radius (in pixels) and the
// sweep is split into, within flattenTolerance of it
func arcSteps(r, sweep float64) int {
	if !(r > flattenTolerance) {
		return max(1, int(math.Ceil(math.Abs(sweep)/(math.Pi/2))))
	}
	step := 2 * math.Acos(1-flattenTolerance/r)
	return int(max(1, min(math.Ceil(math.Abs(sweep)/step), 1<<30)))
}

// maxArcSteps is the most lines an arc is split into when the flattening
// is not culled
const maxArcSteps = 4096

// arc adds the points of the arc op (in pixels) after its start, the last
// point added. A culled long arc is split in quarters, and those in halves
// while they are long, and the pieces out of fp.cull are their chords, as
// cubicCulled does.
func (fp *flatPath) arc(c, u, v PointF, start, sweep float64) {
	// The radius is at most the root of the sum of the squares of two
	// conjugate radii
	r := math.Sqrt(u.X*u.X + u.Y*u.Y + v.X*v.X + v.Y*v.Y)
	n := arcSteps(r, sweep)
	if fp.culling && n > maxCurveSteps {
		q := max(1, int(math.Ceil(math.Abs(sweep)/(math.Pi/2))))
		step := sweep / float64(q)
		for i := 0; i < q; i++ {
			fp.arcCulled(c, u, v, r, start+step*float64(i), step, 0)
		}
		return
	}
	n = min(n, maxArcSteps)
	for i := 1; i <= n; i++ {
		fp.pts = append(fp.pts, arcPoint(c, u, v, start+sweep*float64(i)/float64(n)))
	}
}

// arcCulled adds a piece of an arc of up to a quarter of a turn
func (fp *flatPath) arcCulled(c, u, v PointF, r, start, sweep float64, depth int) {
	p0, p1 := arcPoint(c, u, v, start), arcPoint(c, u, v, start+sweep)
	// The piece is in the triangle of its ends and of where the tangents
	// at them cross
	h := sweep / 2
	tp := arcPoint(PointF{}, u, v, start+h)
	k := 1 / math.Cos(h)
	t := PointF{c.X + tp.X*k, c.Y + tp.Y*k}
	minX, minY := min(p0.X, p1.X, t.X), min(p0.Y, p1.Y, t.Y)
	maxX, maxY := max(p0.X, p1.X, t.X), max(p0.Y, p1.Y, t.Y)
	if maxX < fp.cull[0] || maxY < fp.cull[1] || minX > fp.cull[2] || minY > fp.cull[3] {
		fp.pts = append(fp.pts, p1)
		return
	}
	n := arcSteps(r, sweep)
	if n > maxCurveSteps && depth < 60 {
		fp.arcCulled(c, u, v, r, start, h, depth+1)
		fp.arcCulled(c, u, v, r, start+h, h, depth+1)
		return
	}
	n = min(n, maxCurveSteps)
	for i := 1; i <= n; i++ {
		fp.pts = append(fp.pts, arcPoint(c, u, v, start+sweep*float64(i)/float64(n)))
	}
}

// cubicCulled adds the cubic from p0 (the last point added) to e, split in
// halves while it is too long to be flattened accurately at once. A piece
// whose control points are all out of fp.cull is its chord: the chord and
// the area between it and the curve are out of it too, so neither a fill
// nor a stroke changes in it.
func (fp *flatPath) cubicCulled(p0, c1, c2, e PointF, depth int) {
	minX, minY := min(p0.X, c1.X, c2.X, e.X), min(p0.Y, c1.Y, c2.Y, e.Y)
	maxX, maxY := max(p0.X, c1.X, c2.X, e.X), max(p0.Y, c1.Y, c2.Y, e.Y)
	if maxX < fp.cull[0] || maxY < fp.cull[1] || minX > fp.cull[2] || minY > fp.cull[3] {
		fp.pts = append(fp.pts, e)
		return
	}
	d1 := math.Hypot(p0.X-2*c1.X+c2.X, p0.Y-2*c1.Y+c2.Y)
	d2 := math.Hypot(c1.X-2*c2.X+e.X, c1.Y-2*c2.Y+e.Y)
	n := rawCurveSteps(0.75 * math.Max(d1, d2))
	if n > maxCurveSteps && depth < 48 {
		// de Casteljau at the middle
		mid := func(a, b PointF) PointF { return PointF{(a.X + b.X) / 2, (a.Y + b.Y) / 2} }
		ab, bc, cd := mid(p0, c1), mid(c1, c2), mid(c2, e)
		abc, bcd := mid(ab, bc), mid(bc, cd)
		m := mid(abc, bcd)
		fp.cubicCulled(p0, ab, abc, m, depth+1)
		fp.cubicCulled(m, bcd, cd, e, depth+1)
		return
	}
	n = min(n, maxCurveSteps)
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		a, b, cc, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
		fp.pts = append(fp.pts, PointF{
			a*p0.X + b*c1.X + cc*c2.X + d*e.X,
			a*p0.Y + b*c1.Y + cc*c2.Y + d*e.Y,
		})
	}
}

// LineJoin is the shape of the corners of a stroked line
type LineJoin int

const (
	// LineJoinMiter makes sharp corners, cut off as LineJoinBevel where
	// they would stick out more than StrokeStyle.MiterLimit
	LineJoinMiter LineJoin = iota
	// LineJoinRound rounds the corners
	LineJoinRound
	// LineJoinBevel cuts the corners off
	LineJoinBevel
)

// LineCap is the shape of the ends of a stroked line
type LineCap int

const (
	// LineCapButt ends the line at its end point
	LineCapButt LineCap = iota
	// LineCapRound ends the line with a half circle around its end point
	LineCapRound
	// LineCapSquare ends the line half its width past its end point
	LineCapSquare
)

// StrokeStyle is how Canvas.StrokePath draws the outline of a path. The
// zero value is a solid line a logical pixel wide with sharp corners and
// butt ends.
type StrokeStyle struct {
	// Width is the width of the line in logical pixels; 0 is 1
	Width float64
	Join  LineJoin
	Cap   LineCap
	// MiterLimit is how many times the half width a sharp corner may stick
	// out before it is cut off; 0 is 4
	MiterLimit float64
	// Dash is the pattern of a dashed line, in logical pixels: the lengths
	// of a dash, a gap, a dash, a gap... repeated along the line. An odd
	// count is repeated twice, as in SVG. Empty is a solid line.
	Dash []float64
	// DashOffset is where in the pattern the line starts, in logical pixels
	DashOffset float64
	// NonScaling keeps the width and the dashes in logical pixels of the
	// canvas under its transform: the path is transformed, its line is
	// not (SVG's vector-effect="non-scaling-stroke"). The lines of a
	// drawing stay as thick at any zoom, as in CAD programs.
	NonScaling bool
}

// FillPath fills the inside of the path with the color, anti-aliased, within
// the clip. Open subpaths are closed by a straight line. Where the
// subpaths overlap, the winding rule decides (nonzero): a subpath drawn the
// other way round inside another one makes a hole in it.
func (c *Canvas) FillPath(p *Path, col color.Color) {
	r, g, b, a := straightRGBA8(col)
	if a == 0 {
		return
	}
	if c.state.hasXf {
		c.withPathXf(p, func(tp *Path) { c.FillPath(tp, col) })
		return
	}
	if c.rec != nil {
		c.rec.fillPath(c, p, col)
	}
	c.fillPathWith(p, func(off int, cover []uint8, _, _ int) { c.blendSolid(off, cover, r, g, b, a) })
}

// Paint gives the color of each point of a shape filled by
// Canvas.FillPathPaint: a gradient, a hatch or a pattern of the
// application's own. The point is in logical coordinates of the canvas;
// the color is straight, not premultiplied (as color.RGBA is given
// throughout nui).
type Paint interface {
	ColorAt(x, y float64) color.RGBA
}

// FillPathPaint fills the inside of the path, as FillPath, with the colors
// the paint gives: a LinearGradient, a RadialGradient, a HatchPaint
func (c *Canvas) FillPathPaint(p *Path, paint Paint) {
	if paint == nil {
		return
	}
	colorAt := fastPaint(paint)
	if c.state.hasXf {
		// The paint is in the drawing coordinates: each pixel takes the
		// color of the point it comes from
		xf := c.state.xf
		inv, ok := xf.Invert()
		if !ok {
			return
		}
		at := colorAt
		colorAt = func(x, y float64) color.RGBA { return at(inv.Apply(x, y)) }
		c.withPathXf(p, func(tp *Path) {
			if c.rec != nil {
				c.rec.paintXf = &xf
				c.rec.fillPathPaint(c, tp, paint)
				c.rec.paintXf = nil
			}
			c.fillPathColors(tp, colorAt)
		})
		return
	}
	if c.rec != nil {
		c.rec.fillPathPaint(c, p, paint)
	}
	c.fillPathColors(p, colorAt)
}

// fillPathColors fills the path with the colors of the logical points of
// the canvas colorAt gives
func (c *Canvas) fillPathColors(p *Path, colorAt func(x, y float64) color.RGBA) {
	in := pathInk{at: colorAt}
	c.fillPathWith(p, func(off int, cover []uint8, px, py int) { c.blendInk(off, cover, px, py, &in) })
}

// fillPathWith rasterizes the path and calls blend for the runs of the
// pixels it covers (see coverRow)
func (c *Canvas) fillPathWith(p *Path, blend coverRow) {
	if c.clipEmpty() || len(p.ops) == 0 {
		return
	}
	fp := flatPaths.Get().(*flatPath)
	defer flatPaths.Put(fp)
	// Culled by the clip of the whole frame, see shapeClip
	clip := c.shapeClip().Intersect(c.rgba.Rect)
	fp.culling = true
	fp.cull = [4]float64{float64(clip.Min.X) - 2, float64(clip.Min.Y) - 2, float64(clip.Max.X) + 2, float64(clip.Max.Y) + 2}
	p.flatten(fp, float64(c.state.translateX), float64(c.state.translateY), c.scale)

	pb := polyBuilders.Get().(*polyBuilder)
	defer polyBuilders.Put(pb)
	pb.reset()
	pb.evenOdd = p.fillRule == FillRuleEvenOdd
	for _, sub := range fp.subs {
		pb.addRaw(fp.pts[sub.from:sub.to])
	}
	c.rasterizePolys(pb, blend)
}

// StrokePath draws the outline of the path with the style and the color,
// anti-aliased, within the clip. Where the line overlaps itself, a
// translucent color is blended once, as SVG does.
func (c *Canvas) StrokePath(p *Path, style StrokeStyle, col color.Color) {
	r, g, b, a := straightRGBA8(col)
	if a == 0 {
		return
	}
	c.stroke(p, style, pathInk{r: r, g: g, b: b, a: a}, col, nil)
}

// StrokePathPaint draws the outline of the path, as StrokePath, with the
// colors the paint gives: a line along a gradient, a track colored by a
// value of the application's own
func (c *Canvas) StrokePathPaint(p *Path, style StrokeStyle, paint Paint) {
	if paint == nil {
		return
	}
	c.stroke(p, style, pathInk{at: fastPaint(paint)}, nil, paint)
}

// pathInk is what a shape is filled with: a straight color, or the colors of
// a paint at the logical points of the canvas
type pathInk struct {
	r, g, b, a uint32
	at         func(x, y float64) color.RGBA
}

// blendInk blends the ink over the run of pixels from the offset off, at
// (px, py), by their coverage
func (c *Canvas) blendInk(off int, cover []uint8, px, py int, in *pathInk) {
	if in.at == nil {
		c.blendSolid(off, cover, in.r, in.g, in.b, in.a)
		return
	}
	tx, ty, s := float64(c.state.translateX), float64(c.state.translateY), c.scale
	for x, m := range cover {
		if m != 0 {
			// The logical point at the center of the pixel
			col := in.at((float64(px+x)+0.5-tx)/s, (float64(py)+0.5-ty)/s)
			c.mixPixelAt(off+x*4, uint32(col.R), uint32(col.G), uint32(col.B), (uint32(col.A)*uint32(m)+127)/255)
		}
	}
}

// stroke strokes the path with the ink; col or paint is the ink for the
// SVG recorder
func (c *Canvas) stroke(p *Path, style StrokeStyle, in pathInk, col color.Color, paint Paint) {
	if c.clipEmpty() || len(p.ops) == 0 {
		return
	}
	if c.state.hasXf {
		xf := c.state.xf
		if in.at != nil {
			// The paint is in the drawing coordinates
			inv, ok := xf.Invert()
			if !ok {
				return
			}
			at := in.at
			in.at = func(x, y float64) color.RGBA { return at(inv.Apply(x, y)) }
		}
		if style.NonScaling {
			c.withPathXf(p, func(tp *Path) {
				if c.rec != nil && paint != nil {
					c.rec.paintXf = &xf
					defer func() { c.rec.paintXf = nil }()
				}
				c.stroke(tp, style, in, col, paint)
			})
			return
		}
		c.strokePathXf(p, style, in, col, paint)
		return
	}
	if c.rec != nil {
		c.rec.strokePath(c, p, style, col, paint)
	}
	// What lies more than the width past the clip draws nothing there
	width := style.Width
	if width <= 0 {
		width = 1
	}
	hw := width * c.scale / 2
	margin := hw*max(style.MiterLimit, 4) + 2
	// Culled by the clip of the whole frame, see shapeClip
	clip := c.shapeClip().Intersect(c.rgba.Rect)
	cull := [4]float64{float64(clip.Min.X) - margin, float64(clip.Min.Y) - margin, float64(clip.Max.X) + margin, float64(clip.Max.Y) + margin}

	fp := flatPaths.Get().(*flatPath)
	defer flatPaths.Put(fp)
	fp.culling, fp.cull = true, cull
	p.flatten(fp, float64(c.state.translateX), float64(c.state.translateY), c.scale)

	pb := polyBuilders.Get().(*polyBuilder)
	defer polyBuilders.Put(pb)
	pb.reset()
	pb.cull = cull
	pb.culling = true

	st := strokeStyleInPixels(style, c.scale)
	for _, sub := range fp.subs {
		strokeSubpathCulled(pb, fp.pts[sub.from:sub.to], sub.closed, st, cull)
	}
	c.rasterizePolys(pb, func(off int, cover []uint8, px, py int) { c.blendInk(off, cover, px, py, &in) })
}

// strokeStyleInPixels returns the style with its lengths in pixels
func strokeStyleInPixels(style StrokeStyle, scale float64) StrokeStyle {
	st := style
	if st.Width <= 0 {
		st.Width = 1
	}
	st.Width *= scale
	if st.MiterLimit <= 0 {
		st.MiterLimit = 4
	}
	if len(style.Dash) > 0 {
		dash := make([]float64, 0, len(style.Dash)*2)
		total := 0.0
		for _, d := range style.Dash {
			d = max(0, d*scale)
			dash = append(dash, d)
			total += d
		}
		if len(dash)%2 == 1 {
			dash = append(dash, dash...)
			total *= 2
		}
		if total <= 0 {
			dash = nil // nothing but gaps of no length: a solid line
		}
		st.Dash = dash
		st.DashOffset = style.DashOffset * scale
	}
	return st
}

// polyBuilder collects the polygons of a shape in pixels of the image
type polyBuilder struct {
	pts []PointF
	// The polygons: pts[ends[i-1]:ends[i]]
	ends []int
	// culling drops a polygon that lies entirely out of cull (minX, minY,
	// maxX, maxY)
	culling bool
	cull    [4]float64
	// evenOdd fills by FillRuleEvenOdd
	evenOdd bool
	// The scratch for dashes and for the clipping of polygons and lines
	dash, clipA, clipB, run []PointF
	// The bounds of the polygons (minX, minY, maxX, maxY), the polygons of
	// the bands and their heights, for rasterizePolys
	boxes  [][4]float64
	bands  [][]int
	bandYs [][2]float64
}

var polyBuilders = sync.Pool{New: func() any { return &polyBuilder{} }}
var flatPaths = sync.Pool{New: func() any { return &flatPath{} }}

func (pb *polyBuilder) reset() {
	pb.pts = pb.pts[:0]
	pb.ends = pb.ends[:0]
	pb.culling = false
	pb.evenOdd = false
}

// addRaw adds the polygon as it goes: a filled path keeps the direction of
// its subpaths, which makes holes
func (pb *polyBuilder) addRaw(pts []PointF) {
	if len(pts) < 2 {
		return
	}
	pb.pts = append(pb.pts, pts...)
	pb.ends = append(pb.ends, len(pb.pts))
}

// add adds the polygon turned clockwise on the screen, whatever its
// direction: the pieces of a stroke all wind the same way, so where they
// overlap they make one shape, not holes
func (pb *polyBuilder) add(pts ...PointF) {
	if len(pts) < 3 {
		return
	}
	if pb.culling {
		minX, minY, maxX, maxY := pts[0].X, pts[0].Y, pts[0].X, pts[0].Y
		for _, pt := range pts[1:] {
			// Not math.Min: a NaN cannot get here, and this is the hot loop
			if pt.X < minX {
				minX = pt.X
			} else if pt.X > maxX {
				maxX = pt.X
			}
			if pt.Y < minY {
				minY = pt.Y
			} else if pt.Y > maxY {
				maxY = pt.Y
			}
		}
		if maxX < pb.cull[0] || maxY < pb.cull[1] || minX > pb.cull[2] || minY > pb.cull[3] {
			return
		}
	}
	area := 0.0
	for i := range pts {
		j := (i + 1) % len(pts)
		area += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
	}
	start := len(pb.pts)
	pb.pts = append(pb.pts, pts...)
	if area < 0 {
		added := pb.pts[start:]
		for i, j := 0, len(added)-1; i < j; i, j = i+1, j-1 {
			added[i], added[j] = added[j], added[i]
		}
	}
	pb.ends = append(pb.ends, len(pb.pts))
}

// addCircle adds a circle of the radius around (x, y), clockwise
func (pb *polyBuilder) addCircle(x, y, r float64) {
	n := circleSteps(r)
	var buf [130]PointF
	pts := buf[:0]
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / float64(n)
		pts = append(pts, PointF{x + r*math.Cos(a), y + r*math.Sin(a)})
	}
	pb.add(pts...)
}

// circleSteps returns how many lines a circle of the radius in pixels is
// made of, within flattenTolerance of it
func circleSteps(r float64) int {
	if r <= flattenTolerance {
		return 8
	}
	n := int(math.Ceil(math.Pi / math.Acos(1-flattenTolerance/r)))
	return max(8, min(n, 128))
}

// rasterizePolys fills the polygons, anti-aliased, and calls blend for
// every pixel within the clip they cover
func (c *Canvas) rasterizePolys(pb *polyBuilder, blend coverRow) {
	if len(pb.ends) == 0 {
		return
	}
	// The bounds of every polygon, and of all of them
	pb.boxes = pb.boxes[:0]
	all := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	from := 0
	for _, to := range pb.ends {
		box := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
		for _, pt := range pb.pts[from:to] {
			box[0], box[1] = math.Min(box[0], pt.X), math.Min(box[1], pt.Y)
			box[2], box[3] = math.Max(box[2], pt.X), math.Max(box[3], pt.Y)
		}
		from = to
		pb.boxes = append(pb.boxes, box)
		all[0], all[1] = math.Min(all[0], box[0]), math.Min(all[1], box[1])
		all[2], all[3] = math.Max(all[2], box[2]), math.Max(all[3], box[3])
	}
	// The pieces rasterized are cut in the clip of the whole frame (where a
	// piece starts changes the rounding), the pixels blended in clip: see
	// shapeClip
	b := boxPixels(all).Intersect(c.shapeClip()).Intersect(c.rgba.Rect)
	clip := b.Intersect(c.clipRect())
	if clip.Empty() {
		return
	}
	// The rasterizer's work is the area it covers: a thin line across a
	// wide chart is rasterized in bands, each as tall as the line is there
	const band = 64
	if b.Dx() <= band*2 || b.Dx()*b.Dy() <= 256*256 {
		c.rasterizeRect(pb, b, nil, clip, blend)
		return
	}
	// The polygons of every band, by their bounds (a polygon on the edge of
	// two is in both), and how tall the band has to be
	bands := (b.Dx() + band - 1) / band
	for len(pb.bands) < bands {
		pb.bands = append(pb.bands, nil)
	}
	ys := pb.bandYs[:0]
	for i := 0; i < bands; i++ {
		pb.bands[i] = pb.bands[i][:0]
		ys = append(ys, [2]float64{math.Inf(1), math.Inf(-1)})
	}
	for i, box := range pb.boxes {
		first := int(math.Floor((box[0] - 1 - float64(b.Min.X)) / band))
		last := int(math.Floor((box[2] + 1 - float64(b.Min.X)) / band))
		for k := max(first, 0); k <= min(last, bands-1); k++ {
			pb.bands[k] = append(pb.bands[k], i)
			ys[k][0], ys[k][1] = math.Min(ys[k][0], box[1]), math.Max(ys[k][1], box[3])
		}
	}
	pb.bandYs = ys
	for k := 0; k < bands; k++ {
		if len(pb.bands[k]) == 0 {
			continue
		}
		x := b.Min.X + k*band
		strip := image.Rect(x, b.Min.Y, min(x+band, b.Max.X), b.Max.Y)
		r := boxPixels([4]float64{float64(strip.Min.X), ys[k][0], float64(strip.Max.X), ys[k][1]}).Intersect(strip)
		if r.Overlaps(clip) {
			c.rasterizeRect(pb, r, pb.bands[k], clip, blend)
		}
	}
}

// boxPixels returns the pixels that hold the box (minX, minY, maxX, maxY),
// kept within a billion pixels: farther ones are clipped anyway
func boxPixels(box [4]float64) image.Rectangle {
	if !(box[0] <= box[2] && box[1] <= box[3]) { // empty or NaN
		return image.Rectangle{}
	}
	return image.Rect(int(math.Floor(math.Max(box[0], -1e9))), int(math.Floor(math.Max(box[1], -1e9))),
		int(math.Ceil(math.Min(box[2], 1e9))), int(math.Ceil(math.Min(box[3], 1e9))))
}

// rasterizeRect rasterizes the polygons of pb that reach into b (of those
// numbered in only, if given), clipped to it, and blends the pixels of b
// they cover within clip
func (c *Canvas) rasterizeRect(pb *polyBuilder, b image.Rectangle, only []int, clip image.Rectangle, blend coverRow) {
	ras := rasterizers.Get().(*rasterizer)
	defer rasterizers.Put(ras)
	var z pathSink = ras.reset(b.Dx(), b.Dy())
	if pb.evenOdd {
		z = ras.eo.reset(b.Dx(), b.Dy())
	}
	ox, oy := float64(b.Min.X), float64(b.Min.Y)
	// Clipped to the rectangle with a margin: the rasterizer walks every row
	// between the ends of a line, those far above it too
	edges := [4]float64{-1, -1, float64(b.Dx()) + 1, float64(b.Dy()) + 1}
	drawn := false
	count := len(pb.ends)
	if only != nil {
		count = len(only)
	}
	for k := 0; k < count; k++ {
		i := k
		if only != nil {
			i = only[k]
		}
		from := 0
		if i > 0 {
			from = pb.ends[i-1]
		}
		poly := pb.pts[from:pb.ends[i]]
		box := pb.boxes[i]
		if box[2] < float64(b.Min.X)-1 || box[0] > float64(b.Max.X)+1 || box[3] < float64(b.Min.Y)-1 || box[1] > float64(b.Max.Y)+1 {
			continue
		}
		pts := clipPolygon(poly, ox, oy, edges, &pb.clipA, &pb.clipB)
		if len(pts) < 3 {
			continue
		}
		z.MoveTo(float32(pts[0].X), float32(pts[0].Y))
		for _, pt := range pts[1:] {
			z.LineTo(float32(pt.X), float32(pt.Y))
		}
		z.ClosePath()
		drawn = true
	}
	if !drawn {
		return
	}
	mask := ras.mask(b.Dx(), b.Dy())
	if pb.evenOdd {
		ras.eo.accumulate(mask)
	} else {
		ras.z.DrawOp = draw.Src
		ras.z.Draw(mask, mask.Rect, image.Opaque, image.Point{})
	}
	// The rows and the columns of b within the clip
	clip = clip.Intersect(b)
	for yy := clip.Min.Y - b.Min.Y; yy < clip.Max.Y-b.Min.Y; yy++ {
		row := mask.Pix[yy*mask.Stride : yy*mask.Stride+b.Dx()]
		// The row from its first covered pixel to its last
		first, last := clip.Min.X-b.Min.X, clip.Max.X-b.Min.X
		for first < last && row[first] == 0 {
			first++
		}
		for last > first && row[last-1] == 0 {
			last--
		}
		if first < last {
			blend(c.rgba.PixOffset(b.Min.X+first, b.Min.Y+yy), row[first:last], b.Min.X+first, b.Min.Y+yy)
		}
	}
}

// coverRow blends a run of pixels of a row: off is the offset in Pix of
// the first, at (px, py); cover is the coverage 0..255 of each
type coverRow func(off int, cover []uint8, px, py int)

// blendSolid blends the straight color (r, g, b, a) over the run of
// pixels from the offset off by their coverage
func (c *Canvas) blendSolid(off int, cover []uint8, r, g, b, a uint32) {
	pix := c.rgba.Pix[off : off+len(cover)*4 : off+len(cover)*4]
	if a != 255 {
		for x, m := range cover {
			if m != 0 {
				c.mixPixelAt(off+x*4, r, g, b, (a*uint32(m)+127)/255)
			}
		}
		return
	}
	if c.state.linear {
		for x, m := range cover {
			if m != 0 {
				c.mixPixelAt(off+x*4, r, g, b, uint32(m))
			}
		}
		return
	}
	// An opaque color: the coverage is its alpha, as mixPixelAt blends it
	for x, m := range cover {
		switch m {
		case 0:
		case 255:
			p := pix[x*4 : x*4+4 : x*4+4]
			p[0], p[1], p[2], p[3] = uint8(r), uint8(g), uint8(b), 255
		default:
			p := pix[x*4 : x*4+4 : x*4+4]
			k, ik := uint32(m), 255-uint32(m)
			v := r*k + uint32(p[0])*ik + 128
			p[0] = uint8((v + v>>8) >> 8)
			v = g*k + uint32(p[1])*ik + 128
			p[1] = uint8((v + v>>8) >> 8)
			v = b*k + uint32(p[2])*ik + 128
			p[2] = uint8((v + v>>8) >> 8)
			if p[3] != 255 {
				v = 255*k + uint32(p[3])*ik + 128
				p[3] = uint8((v + v>>8) >> 8)
			}
		}
	}
}

// clipPolygon moves the polygon by (-ox, -oy) and clips it to the
// rectangle of edges (minX, minY, maxX, maxY) by Sutherland–Hodgman: what is
// left covers the same pixels within it. a and b are the scratch.
func clipPolygon(pts []PointF, ox, oy float64, edges [4]float64, a, b *[]PointF) []PointF {
	in := (*a)[:0]
	inside := true
	for _, pt := range pts {
		q := PointF{pt.X - ox, pt.Y - oy}
		if q.X < edges[0] || q.Y < edges[1] || q.X > edges[2] || q.Y > edges[3] {
			inside = false
		}
		in = append(in, q)
	}
	*a = in
	if inside {
		return in
	}
	src, dst := a, b
	for side := 0; side < 4; side++ {
		out := (*dst)[:0]
		poly := *src
		if len(poly) == 0 {
			break
		}
		isIn := func(q PointF) bool {
			switch side {
			case 0:
				return q.X >= edges[0]
			case 1:
				return q.Y >= edges[1]
			case 2:
				return q.X <= edges[2]
			default:
				return q.Y <= edges[3]
			}
		}
		cross := func(p1, p2 PointF) PointF {
			var t float64
			switch side {
			case 0:
				t = (edges[0] - p1.X) / (p2.X - p1.X)
			case 1:
				t = (edges[1] - p1.Y) / (p2.Y - p1.Y)
			case 2:
				t = (edges[2] - p1.X) / (p2.X - p1.X)
			default:
				t = (edges[3] - p1.Y) / (p2.Y - p1.Y)
			}
			return PointF{p1.X + (p2.X-p1.X)*t, p1.Y + (p2.Y-p1.Y)*t}
		}
		prev := poly[len(poly)-1]
		prevIn := isIn(prev)
		for _, cur := range poly {
			curIn := isIn(cur)
			if curIn {
				if !prevIn {
					out = append(out, cross(prev, cur))
				}
				out = append(out, cur)
			} else if prevIn {
				out = append(out, cross(prev, cur))
			}
			prev, prevIn = cur, curIn
		}
		*dst = out
		src, dst = dst, src
	}
	return *src
}

// rasterizer keeps the rasterizer and the mask between the shapes: a chart
// fills and strokes many of them in every frame
type rasterizer struct {
	z     vector.Rasterizer
	eo    evenOddRasterizer
	alpha image.Alpha
}

// pathSink takes the lines of a polygon: the rasterizer of x/image/vector
// or the even-odd one
type pathSink interface {
	MoveTo(x, y float32)
	LineTo(x, y float32)
	ClosePath()
}

var rasterizers = sync.Pool{New: func() any { return &rasterizer{} }}

func (r *rasterizer) reset(w, h int) *vector.Rasterizer {
	r.z.Reset(w, h)
	return &r.z
}

// mask returns an alpha image of the size; its pixels are overwritten by
// the rasterizer (draw.Src)
func (r *rasterizer) mask(w, h int) *image.Alpha {
	if cap(r.alpha.Pix) < w*h {
		r.alpha.Pix = make([]uint8, w*h)
	}
	r.alpha.Pix = r.alpha.Pix[:w*h]
	r.alpha.Stride = w
	r.alpha.Rect = image.Rect(0, 0, w, h)
	return &r.alpha
}

// strokeSubpath adds the polygons of the stroke of the polyline: a
// quadrilateral per segment, the joins between them and the caps at the
// ends. All of them wind the same way (polyBuilder.add), so the stroke is
// their union.
func strokeSubpath(pb *polyBuilder, pts []PointF, closed bool, st StrokeStyle) {
	// The points without repeats: a segment of no length has no direction
	clean := pb.dash[:0]
	for _, pt := range pts {
		if math.IsNaN(pt.X) || math.IsNaN(pt.Y) || math.IsInf(pt.X, 0) || math.IsInf(pt.Y, 0) {
			continue
		}
		if len(clean) > 0 && math.Abs(clean[len(clean)-1].X-pt.X) < 1e-9 && math.Abs(clean[len(clean)-1].Y-pt.Y) < 1e-9 {
			continue
		}
		clean = append(clean, pt)
	}
	if closed && len(clean) > 2 && math.Abs(clean[0].X-clean[len(clean)-1].X) < 1e-9 && math.Abs(clean[0].Y-clean[len(clean)-1].Y) < 1e-9 {
		clean = clean[:len(clean)-1]
	}
	pb.dash = clean
	hw := st.Width / 2
	if len(clean) == 0 {
		return
	}
	if len(clean) == 1 {
		// A point: drawn only by its caps, as in SVG
		pt := clean[0]
		switch st.Cap {
		case LineCapRound:
			pb.addCircle(pt.X, pt.Y, hw)
		case LineCapSquare:
			pb.add(PointF{pt.X - hw, pt.Y - hw}, PointF{pt.X + hw, pt.Y - hw}, PointF{pt.X + hw, pt.Y + hw}, PointF{pt.X - hw, pt.Y + hw})
		}
		return
	}
	if len(st.Dash) > 0 {
		// Each dash is an open line of its own, with its own caps
		pts := clean
		if closed {
			pts = append(pts, pts[0])
		}
		dashPolyline(pts, st.Dash, st.DashOffset, func(dash []PointF) {
			strokePolyline(pb, dash, false, st, hw)
		})
		return
	}
	strokePolyline(pb, clean, closed && len(clean) > 2, st, hw)
}

// strokePolyline adds the stroke of the polyline without repeated points
func strokePolyline(pb *polyBuilder, pts []PointF, closed bool, st StrokeStyle, hw float64) {
	n := len(pts)
	if n < 2 {
		return
	}
	segs := n - 1
	if closed {
		segs = n
	}
	dir := func(i int) (dx, dy float64) {
		a, b := pts[i], pts[(i+1)%n]
		l := math.Hypot(b.X-a.X, b.Y-a.Y)
		return (b.X - a.X) / l, (b.Y - a.Y) / l
	}
	for i := 0; i < segs; i++ {
		a, b := pts[i], pts[(i+1)%n]
		dx, dy := dir(i)
		nx, ny := -dy*hw, dx*hw
		// The square caps lengthen the first and the last segments
		if !closed && st.Cap == LineCapSquare {
			if i == 0 {
				a = PointF{a.X - dx*hw, a.Y - dy*hw}
			}
			if i == segs-1 {
				b = PointF{b.X + dx*hw, b.Y + dy*hw}
			}
		}
		pb.add(PointF{a.X + nx, a.Y + ny}, PointF{b.X + nx, b.Y + ny}, PointF{b.X - nx, b.Y - ny}, PointF{a.X - nx, a.Y - ny})
	}
	// The joins at the inner points, and at the start of a closed line
	first, last := 1, n-1
	if closed {
		first, last = 0, n
	}
	for i := first; i < last; i++ {
		prev := (i - 1 + n) % n
		d0x, d0y := dir(prev)
		d1x, d1y := dir(i % n)
		addJoin(pb, pts[i%n], d0x, d0y, d1x, d1y, st, hw)
	}
	if !closed && st.Cap == LineCapRound {
		pb.addCircle(pts[0].X, pts[0].Y, hw)
		pb.addCircle(pts[n-1].X, pts[n-1].Y, hw)
	}
}

// addJoin adds the corner at v between the segment coming in the direction
// d0 and the one going out in the direction d1
func addJoin(pb *polyBuilder, v PointF, d0x, d0y, d1x, d1y float64, st StrokeStyle, hw float64) {
	cross := d0x*d1y - d0y*d1x
	dot := d0x*d1x + d0y*d1y
	if math.Abs(cross) < 1e-9 && dot > 0 {
		return // straight on: the segments meet edge to edge
	}
	// The outer side of the turn
	k := -1.0
	if cross < 0 {
		k = 1
	}
	n0 := PointF{-d0y * hw * k, d0x * hw * k}
	n1 := PointF{-d1y * hw * k, d1x * hw * k}
	a := PointF{v.X + n0.X, v.Y + n0.Y}
	b := PointF{v.X + n1.X, v.Y + n1.Y}
	if st.Join == LineJoinRound {
		// The arc on the outer side from a to b, as many lines as a circle
		// of the radius has on that angle: none on a slight turn of a
		// smooth line, where it would differ from a bevel by less than the
		// tolerance
		turn := math.Atan2(math.Abs(cross), dot)
		steps := int(math.Ceil(turn * float64(circleSteps(hw)) / (2 * math.Pi)))
		if steps <= 1 {
			pb.add(v, a, b)
			return
		}
		var buf [130]PointF
		pts := append(buf[:0], v, a)
		a0 := math.Atan2(n0.Y, n0.X)
		// From n0 to n1 the short way round
		da := math.Atan2(n0.X*n1.Y-n0.Y*n1.X, n0.X*n1.X+n0.Y*n1.Y)
		for i := 1; i < steps; i++ {
			ang := a0 + da*float64(i)/float64(steps)
			pts = append(pts, PointF{v.X + hw*math.Cos(ang), v.Y + hw*math.Sin(ang)})
		}
		pts = append(pts, b)
		pb.add(pts...)
		return
	}
	if st.Join == LineJoinMiter {
		mx, my := n0.X+n1.X, n0.Y+n1.Y
		ml := math.Hypot(mx, my)
		if ml > 1e-9 {
			// The tip of the corner: along the bisector, hw / cos(half the turn)
			cosHalf := (mx*n0.X + my*n0.Y) / (ml * hw)
			if cosHalf > 1e-9 && 1/cosHalf <= st.MiterLimit {
				l := hw / cosHalf
				tip := PointF{v.X + mx/ml*l, v.Y + my/ml*l}
				pb.add(v, a, tip, b)
				return
			}
		}
	}
	pb.add(v, a, b)
}

// dashPolyline splits the polyline into the dashes of the pattern, from
// offset into it, and calls f with each (the slice is reused)
func dashPolyline(pts []PointF, pattern []float64, offset float64, f func(dash []PointF)) {
	total := 0.0
	for _, d := range pattern {
		total += d
	}
	if total <= 0 {
		f(pts)
		return
	}
	offset = math.Mod(offset, total)
	if offset < 0 {
		offset += total
	}
	// Where in the pattern the line starts: offset is below the total, so
	// a turn of the pattern ends the loop
	idx := 0
	for offset >= pattern[idx] {
		offset -= pattern[idx]
		idx = (idx + 1) % len(pattern)
	}
	left := pattern[idx] - offset // of the current dash or gap
	on := idx%2 == 0
	var cur []PointF
	if on {
		cur = append(cur, pts[0])
	}
	// At most a million dashes: a pattern far smaller than the line, a
	// pixel's fraction long, would draw nothing visible anyway
	budget := 1 << 20
	for i := 0; i+1 < len(pts) && budget > 0; i++ {
		a, b := pts[i], pts[i+1]
		segLen := math.Hypot(b.X-a.X, b.Y-a.Y)
		pos := 0.0
		for segLen-pos > left && budget > 0 {
			budget--
			pos += left
			t := pos / segLen
			pt := PointF{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}
			if on {
				cur = append(cur, pt)
				f(cur)
				cur = cur[:0]
			} else {
				cur = append(cur[:0], pt)
			}
			on = !on
			idx = (idx + 1) % len(pattern)
			left = pattern[idx]
		}
		left -= segLen - pos
		if on {
			cur = append(cur, b)
		}
	}
	if on && len(cur) > 1 {
		f(cur)
	}
}

// FillCircle fills a circle centered at (cx, cy), anti-aliased
func (c *Canvas) FillCircle(cx, cy, r float64, col color.Color) {
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	p.Circle(cx, cy, r)
	c.FillPath(p, col)
}

// DrawCircle draws the outline of a circle centered at (cx, cy), the line
// width logical pixels wide, anti-aliased
func (c *Canvas) DrawCircle(cx, cy, r, width float64, col color.Color) {
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	p.Circle(cx, cy, r)
	c.StrokePath(p, StrokeStyle{Width: width}, col)
}

// FillPolygon fills the polygon through the points, anti-aliased
func (c *Canvas) FillPolygon(pts []PointF, col color.Color) {
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	p.Polygon(pts...)
	c.FillPath(p, col)
}

// DrawPolyline draws the line through the points with the style,
// anti-aliased: one shape, so the joints are as the style says and a
// translucent line is not darker where it overlaps itself
func (c *Canvas) DrawPolyline(pts []PointF, style StrokeStyle, col color.Color) {
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	p.Polyline(pts...)
	c.StrokePath(p, style, col)
}

var pathPool = sync.Pool{New: func() any { return NewPath() }}

// drawPixels draws the premultiplied image at the logical point (x, y),
// its pixels as pixels of the canvas' image (not scaled), within the clip:
// a picture made at the canvas' resolution, as a density map
func (c *Canvas) drawPixels(x, y float64, img *image.RGBA) {
	px := c.state.translateX + int(math.Round(x*c.scale))
	py := c.state.translateY + int(math.Round(y*c.scale))
	r := img.Rect.Add(image.Pt(px, py)).Intersect(c.clipRect()).Intersect(c.rgba.Rect)
	if r.Empty() {
		return
	}
	drawOver(c.rgba, r, img, r.Min.Sub(image.Pt(px, py)))
}

// fillRectF fills the rectangle of fractional logical coordinates,
// antialiased by how much of each pixel it covers, without a rasterizer:
// the columns of a dense chart line are drawn so, thousands a frame
func (c *Canvas) fillRectF(x, y, w, h float64, col color.RGBA) {
	if w <= 0 || h <= 0 || col.A == 0 {
		return
	}
	s := c.scale
	x0 := float64(c.state.translateX) + x*s
	y0 := float64(c.state.translateY) + y*s
	x1, y1 := x0+w*s, y0+h*s
	clip := c.clipRect().Intersect(c.rgba.Rect)
	px0, px1 := int(math.Floor(x0)), int(math.Ceil(x1))
	py0, py1 := int(math.Floor(y0)), int(math.Ceil(y1))
	if px0 < clip.Min.X {
		px0 = clip.Min.X
	}
	if py0 < clip.Min.Y {
		py0 = clip.Min.Y
	}
	if px1 > clip.Max.X {
		px1 = clip.Max.X
	}
	if py1 > clip.Max.Y {
		py1 = clip.Max.Y
	}
	if px0 >= px1 || py0 >= py1 {
		return
	}
	// cover returns how much of the pixel p the span [lo, hi) takes
	cover := func(p int, lo, hi float64) float64 {
		a, b := float64(p), float64(p+1)
		if lo > a {
			a = lo
		}
		if hi < b {
			b = hi
		}
		if b <= a {
			return 0
		}
		return b - a
	}
	// The coverage of each column, the same in every row but the first
	// and the last: a line is a few columns wide
	var colBuf [8]float64
	cols := colBuf[:0]
	if px1-px0 > len(colBuf) {
		cols = make([]float64, 0, px1-px0)
	}
	for px := px0; px < px1; px++ {
		cols = append(cols, cover(px, x0, x1))
	}
	r, g, b := uint32(col.R), uint32(col.G), uint32(col.B)
	a := float64(col.A)
	var full [8]uint32
	fullA := full[:0]
	if len(cols) > len(full) {
		fullA = make([]uint32, 0, len(cols))
	}
	for _, cv := range cols {
		fullA = append(fullA, uint32(a*cv+0.5))
	}
	for py := py0; py < py1; py++ {
		off := c.rgba.PixOffset(px0, py)
		cy := 1.0
		if py == py0 || py == py1-1 {
			cy = cover(py, y0, y1)
		}
		for i, cv := range cols {
			alpha := fullA[i]
			if cy != 1 {
				alpha = uint32(a*cv*cy + 0.5)
			}
			if alpha != 0 {
				c.mixPixelAt(off, r, g, b, alpha)
			}
			off += 4
		}
	}
}
