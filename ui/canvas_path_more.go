package ui

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// ArcTo adds an arc of the radius r tangent to the line from the current
// point to (x1, y1) and to the line from there to (x2, y2), joined to the
// current point by a line: a rounded corner, as arcTo of the HTML canvas.
// Without a current point it starts at (x1, y1).
func (p *Path) ArcTo(x1, y1, x2, y2, r float64) {
	if !p.hasCur {
		p.MoveTo(x1, y1)
		return
	}
	x0, y0 := p.cur.X, p.cur.Y
	// The directions from the corner to both ends
	ax, ay := x0-x1, y0-y1
	bx, by := x2-x1, y2-y1
	la, lb := math.Hypot(ax, ay), math.Hypot(bx, by)
	cross := ax*by - ay*bx
	if r <= 0 || la == 0 || lb == 0 || math.Abs(cross) < 1e-12*la*lb {
		p.LineTo(x1, y1) // no corner to round
		return
	}
	ax, ay, bx, by = ax/la, ay/la, bx/lb, by/lb
	// Half the angle between the lines; the arc touches them at d from
	// the corner
	half := math.Acos(max(-1, min(1, ax*bx+ay*by))) / 2
	d := r / math.Tan(half)
	t1x, t1y := x1+ax*d, y1+ay*d
	t2x, t2y := x1+bx*d, y1+by*d
	// The center is on the bisector, r / sin(half) from the corner
	mx, my := ax+bx, ay+by
	ml := math.Hypot(mx, my)
	cd := r / math.Sin(half)
	cx, cy := x1+mx/ml*cd, y1+my/ml*cd
	start := math.Atan2(t1y-cy, t1x-cx)
	end := math.Atan2(t2y-cy, t2x-cx)
	// The short way round, the side of the turn
	sweep := end - start
	for sweep > math.Pi {
		sweep -= 2 * math.Pi
	}
	for sweep < -math.Pi {
		sweep += 2 * math.Pi
	}
	p.LineTo(t1x, t1y)
	p.EllipticalArc(cx, cy, r, r, start, start+sweep)
}

// SVGArcTo adds an elliptical arc from the current point to (x, y) as the
// A command of an SVG path: the radii rx and ry, the ellipse turned by
// rotation radians, largeArc picks the longer of the two arcs, sweep the
// one going clockwise on the screen. Radii too small to reach are grown,
// as SVG does.
func (p *Path) SVGArcTo(rx, ry, rotation float64, largeArc, sweep bool, x, y float64) {
	if !p.hasCur {
		p.MoveTo(x, y)
		return
	}
	x0, y0 := p.cur.X, p.cur.Y
	if x0 == x && y0 == y {
		return
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		p.LineTo(x, y)
		return
	}
	// The endpoint to center conversion of the SVG specification (F.6.5)
	sinPhi, cosPhi := math.Sincos(rotation)
	dx, dy := (x0-x)/2, (y0-y)/2
	x1p := cosPhi*dx + sinPhi*dy
	y1p := -sinPhi*dx + cosPhi*dy
	if l := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry); l > 1 {
		s := math.Sqrt(l)
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	k := 0.0
	if den > 0 && num > 0 {
		k = math.Sqrt(num / den)
	}
	if largeArc == sweep {
		k = -k
	}
	cxp, cyp := k*rx*y1p/ry, -k*ry*x1p/rx
	cx := cosPhi*cxp - sinPhi*cyp + (x0+x)/2
	cy := sinPhi*cxp + cosPhi*cyp + (y0+y)/2
	angle := func(ux, uy, vx, vy float64) float64 {
		return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
	}
	theta := angle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	delta := angle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)
	if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	} else if sweep && delta < 0 {
		delta += 2 * math.Pi
	}
	// Cubic curves of up to a quarter of a turn, on the turned ellipse
	n := max(1, int(math.Ceil(math.Abs(delta)/(math.Pi/2)-1e-9)))
	step := delta / float64(n)
	kk := 4.0 / 3.0 * math.Tan(step/4)
	pt := func(a float64) (float64, float64) {
		ex, ey := rx*math.Cos(a), ry*math.Sin(a)
		return cx + cosPhi*ex - sinPhi*ey, cy + sinPhi*ex + cosPhi*ey
	}
	der := func(a float64) (float64, float64) {
		ex, ey := -rx*math.Sin(a), ry*math.Cos(a)
		return cosPhi*ex - sinPhi*ey, sinPhi*ex + cosPhi*ey
	}
	a := theta
	for i := 0; i < n; i++ {
		b := a + step
		ax, ay := pt(a)
		dax, day := der(a)
		bx, by := pt(b)
		dbx, dby := der(b)
		if i == n-1 {
			bx, by = x, y // exactly at the end
		}
		p.CubicTo(ax+kk*dax, ay+kk*day, bx-kk*dbx, by-kk*dby, bx, by)
		a = b
	}
}

// RegularPolygon adds a closed polygon of n equal sides inscribed in the
// circle of the radius r around (cx, cy), its first corner at the angle
// rotation (0 - to the right, -math.Pi/2 - up)
func (p *Path) RegularPolygon(cx, cy, r float64, n int, rotation float64) {
	if n < 3 {
		return
	}
	for i := 0; i < n; i++ {
		a := rotation + 2*math.Pi*float64(i)/float64(n)
		if i == 0 {
			p.MoveTo(cx+r*math.Cos(a), cy+r*math.Sin(a))
		} else {
			p.LineTo(cx+r*math.Cos(a), cy+r*math.Sin(a))
		}
	}
	p.Close()
}

// Star adds a closed star of the points around (cx, cy): its tips at the
// radius outer, the corners between them at inner, the first tip up
func (p *Path) Star(cx, cy, outer, inner float64, points int) {
	if points < 2 {
		return
	}
	for i := 0; i < points*2; i++ {
		r := outer
		if i%2 == 1 {
			r = inner
		}
		a := -math.Pi/2 + math.Pi*float64(i)/float64(points)
		if i == 0 {
			p.MoveTo(cx+r*math.Cos(a), cy+r*math.Sin(a))
		} else {
			p.LineTo(cx+r*math.Cos(a), cy+r*math.Sin(a))
		}
	}
	p.Close()
}

// Spline adds a smooth curve through the points (a Catmull-Rom spline made
// of cubic curves), starting a new subpath; closed joins the last point to
// the first smoothly and closes it. tension 0 is the usual curve, 1 makes
// straight lines between the points, negative values bend it more.
func (p *Path) Spline(pts []PointF, closed bool, tension float64) {
	n := len(pts)
	if n == 0 {
		return
	}
	p.MoveTo(pts[0].X, pts[0].Y)
	if n == 1 {
		return
	}
	at := func(i int) PointF {
		if closed {
			return pts[((i%n)+n)%n]
		}
		return pts[max(0, min(i, n-1))]
	}
	k := (1 - tension) / 6
	segs := n - 1
	if closed {
		segs = n
	}
	for i := 0; i < segs; i++ {
		p0, p1, p2, p3 := at(i-1), at(i), at(i+1), at(i+2)
		p.CubicTo(
			p1.X+(p2.X-p0.X)*k, p1.Y+(p2.Y-p0.Y)*k,
			p2.X-(p3.X-p1.X)*k, p2.Y-(p3.Y-p1.Y)*k,
			p2.X, p2.Y,
		)
	}
	if closed {
		p.Close()
	}
}

// AddPath adds the subpaths of q to p
func (p *Path) AddPath(q *Path) {
	for _, op := range q.ops {
		switch op.kind {
		case pathMoveTo:
			p.MoveTo(op.p[0].X, op.p[0].Y)
		case pathLineTo:
			p.LineTo(op.p[0].X, op.p[0].Y)
		case pathQuadTo:
			p.QuadTo(op.p[0].X, op.p[0].Y, op.p[1].X, op.p[1].Y)
		case pathCubicTo:
			p.CubicTo(op.p[0].X, op.p[0].Y, op.p[1].X, op.p[1].Y, op.p[2].X, op.p[2].Y)
		case pathArc:
			// Its start is the current point, as when it was added
			p.ops = append(p.ops, op)
			p.cur = arcPoint(op.p[0], op.p[1], op.p[2], op.a[0]+op.a[1])
		case pathClose:
			p.Close()
		}
	}
}

// CurrentPoint returns where the next line starts; ok is false on an
// empty path
func (p *Path) CurrentPoint() (pt PointF, ok bool) {
	return p.cur, p.hasCur
}

// flattenFine flattens the path in its own coordinates, for measures and
// hit tests, within a 1/20 of a unit
func (p *Path) flattenFine() *flatPath {
	fp := &flatPath{}
	p.flatten(fp, 0, 0, 1)
	return fp
}

// Length returns the length of the path: of its lines and curves, the
// closing lines included
func (p *Path) Length() float64 {
	fp := p.flattenFine()
	total := 0.0
	for _, sub := range fp.subs {
		pts := fp.pts[sub.from:sub.to]
		for i := 1; i < len(pts); i++ {
			total += math.Hypot(pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y)
		}
		if sub.closed {
			total += math.Hypot(pts[0].X-pts[len(pts)-1].X, pts[0].Y-pts[len(pts)-1].Y)
		}
	}
	return total
}

// PointAtLength returns the point at the distance d along the path from
// its start and the direction of the path there, in radians (0 - to the
// right): to move a thing along a path, to put labels on it. A distance
// past the ends gives the ends; ok is false for an empty path.
func (p *Path) PointAtLength(d float64) (pt PointF, angle float64, ok bool) {
	fp := p.flattenFine()
	var last PointF
	lastAngle := 0.0
	for _, sub := range fp.subs {
		pts := fp.pts[sub.from:sub.to]
		if sub.closed {
			pts = append(pts[:len(pts):len(pts)], pts[0])
		}
		for i := 1; i < len(pts); i++ {
			a, b := pts[i-1], pts[i]
			l := math.Hypot(b.X-a.X, b.Y-a.Y)
			if l == 0 {
				continue
			}
			ang := math.Atan2(b.Y-a.Y, b.X-a.X)
			if !ok && d <= 0 {
				return a, ang, true
			}
			ok = true
			if d <= l {
				t := d / l
				return PointF{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}, ang, true
			}
			d -= l
			last, lastAngle = b, ang
		}
	}
	return last, lastAngle, ok
}

// Contains reports whether the point is inside the path by its fill rule,
// as FillPath would fill it: to find the shape under the mouse
func (p *Path) Contains(x, y float64) bool {
	fp := p.flattenFine()
	winding, crossings := 0, 0
	for _, sub := range fp.subs {
		pts := fp.pts[sub.from:sub.to]
		n := len(pts)
		for i := 0; i < n; i++ {
			a, b := pts[i], pts[(i+1)%n]
			if a.Y <= y {
				if b.Y > y && cross2(a, b, x, y) > 0 {
					winding++
					crossings++
				}
			} else if b.Y <= y && cross2(a, b, x, y) < 0 {
				winding--
				crossings++
			}
		}
	}
	if p.fillRule == FillRuleEvenOdd {
		return crossings%2 == 1
	}
	return winding != 0
}

// cross2 is the side of the line a-b the point (x, y) is on
func cross2(a, b PointF, x, y float64) float64 {
	return (b.X-a.X)*(y-a.Y) - (x-a.X)*(b.Y-a.Y)
}

// StrokeContains reports whether the point is within width/2 of the
// lines of the path: on its stroke of the width, the joins rounded
func (p *Path) StrokeContains(x, y, width float64) bool {
	fp := p.flattenFine()
	hw := width / 2
	for _, sub := range fp.subs {
		pts := fp.pts[sub.from:sub.to]
		n := len(pts)
		segs := n - 1
		if sub.closed {
			segs = n
		}
		for i := 0; i < segs; i++ {
			if distToSegment(x, y, pts[i], pts[(i+1)%n]) <= hw {
				return true
			}
		}
	}
	return false
}

func distToSegment(x, y float64, a, b PointF) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = max(0, min(1, ((x-a.X)*dx+(y-a.Y)*dy)/l2))
	}
	return math.Hypot(x-(a.X+t*dx), y-(a.Y+t*dy))
}

// ErrSVGPath is the error of ParseSVGPath for data it cannot read
var ErrSVGPath = errors.New("bad SVG path data")

// ParseSVGPath reads the d attribute of an SVG path element: the commands
// M, L, H, V, C, S, Q, T, A and Z, absolute (upper case) and relative
// (lower case). An icon of an SVG file is drawn so:
//
//	p, err := ui.ParseSVGPath("M12 2L2 22h20z")
//	cnv.Translate(x, y)
//	cnv.FillPath(p, col)
//
// The path read up to an error is returned with it.
func ParseSVGPath(d string) (*Path, error) {
	p := NewPath()
	sc := svgScanner{s: d}
	var cmd byte
	// The control point of the last curve, for S and T
	var lastCtrl PointF
	var lastKind byte
	for {
		sc.skipSpace()
		if sc.done() {
			return p, nil
		}
		if c := sc.s[sc.i]; isSVGCommand(c) {
			cmd = c
			sc.i++
		} else if cmd == 0 || cmd&^0x20 == 'Z' {
			return p, fmt.Errorf("%w: %q at %d", ErrSVGPath, c, sc.i)
		}
		rel := cmd >= 'a'
		cur, _ := p.CurrentPoint()
		ox, oy := 0.0, 0.0
		if rel {
			ox, oy = cur.X, cur.Y
		}
		var nums [7]float64
		read := func(n int) error {
			for k := 0; k < n; k++ {
				v, err := sc.number()
				if err != nil {
					return err
				}
				nums[k] = v
			}
			return nil
		}
		upper := cmd &^ 0x20
		kind := upper
		var err error
		switch upper {
		case 'Z':
			p.Close()
		case 'M':
			if err = read(2); err == nil {
				p.MoveTo(ox+nums[0], oy+nums[1])
				// The numbers after the first pair are lines
				if rel {
					cmd = 'l'
				} else {
					cmd = 'L'
				}
			}
		case 'L':
			if err = read(2); err == nil {
				p.LineTo(ox+nums[0], oy+nums[1])
			}
		case 'H':
			if err = read(1); err == nil {
				x := nums[0]
				if rel {
					x += cur.X
				}
				p.LineTo(x, cur.Y)
			}
		case 'V':
			if err = read(1); err == nil {
				y := nums[0]
				if rel {
					y += cur.Y
				}
				p.LineTo(cur.X, y)
			}
		case 'C':
			if err = read(6); err == nil {
				lastCtrl = PointF{ox + nums[2], oy + nums[3]}
				p.CubicTo(ox+nums[0], oy+nums[1], lastCtrl.X, lastCtrl.Y, ox+nums[4], oy+nums[5])
			}
		case 'S':
			if err = read(4); err == nil {
				c1 := cur
				if lastKind == 'C' || lastKind == 'S' {
					c1 = PointF{2*cur.X - lastCtrl.X, 2*cur.Y - lastCtrl.Y}
				}
				lastCtrl = PointF{ox + nums[0], oy + nums[1]}
				p.CubicTo(c1.X, c1.Y, lastCtrl.X, lastCtrl.Y, ox+nums[2], oy+nums[3])
			}
		case 'Q':
			if err = read(4); err == nil {
				lastCtrl = PointF{ox + nums[0], oy + nums[1]}
				p.QuadTo(lastCtrl.X, lastCtrl.Y, ox+nums[2], oy+nums[3])
			}
		case 'T':
			if err = read(2); err == nil {
				c := cur
				if lastKind == 'Q' || lastKind == 'T' {
					c = PointF{2*cur.X - lastCtrl.X, 2*cur.Y - lastCtrl.Y}
				}
				lastCtrl = c
				p.QuadTo(c.X, c.Y, ox+nums[0], oy+nums[1])
			}
		case 'A':
			if err = read(3); err == nil {
				rx, ry, rot := nums[0], nums[1], nums[2]
				var large, sweep bool
				if large, err = sc.flag(); err == nil {
					if sweep, err = sc.flag(); err == nil {
						if err = read(2); err == nil {
							p.SVGArcTo(rx, ry, rot*math.Pi/180, large, sweep, ox+nums[0], oy+nums[1])
						}
					}
				}
			}
		default:
			return p, fmt.Errorf("%w: command %q", ErrSVGPath, cmd)
		}
		if err != nil {
			return p, err
		}
		lastKind = kind
	}
}

func isSVGCommand(c byte) bool {
	switch c &^ 0x20 {
	case 'M', 'L', 'H', 'V', 'C', 'S', 'Q', 'T', 'A', 'Z':
		return true
	}
	return false
}

// svgScanner reads the numbers of path data
type svgScanner struct {
	s string
	i int
}

func (sc *svgScanner) done() bool { return sc.i >= len(sc.s) }

func (sc *svgScanner) skipSpace() {
	for sc.i < len(sc.s) {
		switch sc.s[sc.i] {
		case ' ', '\t', '\n', '\r', '\f', ',':
			sc.i++
		default:
			return
		}
	}
}

// number reads a number: 1, -2.5, .5, 1e-3; "1.5.5" is two of them
func (sc *svgScanner) number() (float64, error) {
	sc.skipSpace()
	start := sc.i
	if sc.i < len(sc.s) && (sc.s[sc.i] == '+' || sc.s[sc.i] == '-') {
		sc.i++
	}
	digits, dot := 0, false
	for sc.i < len(sc.s) {
		c := sc.s[sc.i]
		if c >= '0' && c <= '9' {
			digits++
		} else if c == '.' && !dot {
			dot = true
		} else {
			break
		}
		sc.i++
	}
	if digits == 0 {
		sc.i = start
		return 0, fmt.Errorf("%w: a number expected at %d", ErrSVGPath, start)
	}
	if sc.i < len(sc.s) && (sc.s[sc.i] == 'e' || sc.s[sc.i] == 'E') {
		j := sc.i + 1
		if j < len(sc.s) && (sc.s[j] == '+' || sc.s[j] == '-') {
			j++
		}
		if j < len(sc.s) && sc.s[j] >= '0' && sc.s[j] <= '9' {
			for j < len(sc.s) && sc.s[j] >= '0' && sc.s[j] <= '9' {
				j++
			}
			sc.i = j
		}
	}
	return strconv.ParseFloat(sc.s[start:sc.i], 64)
}

// flag reads a flag of an arc: 0 or 1, which may be written without a
// space after it ("a1 1 0 00 1 1")
func (sc *svgScanner) flag() (bool, error) {
	sc.skipSpace()
	if sc.i < len(sc.s) && (sc.s[sc.i] == '0' || sc.s[sc.i] == '1') {
		sc.i++
		return sc.s[sc.i-1] == '1', nil
	}
	return false, fmt.Errorf("%w: a flag expected at %d", ErrSVGPath, sc.i)
}
