package ui

import (
	"math"
	"sort"
)

// Solids made of 2D profiles, as CAD programs make parts of sketches: a
// plate with holes extruded from its outline, a shaft revolved from its
// half section.

// TriangulatePolygon splits the polygon with the holes into triangles and
// returns their corners, three per triangle, counterclockwise (with y up):
// indices into the points of the outline followed by those of each hole in
// turn. The outline and the holes may go either way round; the holes must
// be inside the outline and not touch it nor each other. Nil if the
// polygon has no area.
func TriangulatePolygon(outline []PointF, holes [][]PointF) []int {
	// All the points, and the rings as indices into them
	pts := append([]PointF(nil), outline...)
	ring := func(from, n int, ccw bool) []int {
		r := make([]int, n)
		for i := range r {
			r[i] = from + i
		}
		if (signedArea(pts[from:from+n]) > 0) != ccw {
			for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
				r[i], r[j] = r[j], r[i]
			}
		}
		return r
	}
	outer := cleanRing(pts, ring(0, len(outline), true))
	if len(outer) < 3 {
		return nil
	}
	type hole struct {
		idx  []int
		maxX float64
	}
	var hs []hole
	for _, h := range holes {
		from := len(pts)
		pts = append(pts, h...)
		idx := cleanRing(pts, ring(from, len(h), false))
		if len(idx) < 3 {
			continue
		}
		mx := math.Inf(-1)
		for _, i := range idx {
			mx = math.Max(mx, pts[i].X)
		}
		hs = append(hs, hole{idx, mx})
	}
	// The holes joined to the outline by bridges, the rightmost first
	// (Eberly's method)
	sort.Slice(hs, func(i, j int) bool { return hs[i].maxX > hs[j].maxX })
	for _, h := range hs {
		outer = bridgeHole(pts, outer, h.idx)
	}
	return earClip(pts, outer)
}

// signedArea returns the area of the ring, positive counterclockwise (y up)
func signedArea(pts []PointF) float64 {
	a := 0.0
	for i := range pts {
		p, q := pts[i], pts[(i+1)%len(pts)]
		a += p.X*q.Y - q.X*p.Y
	}
	return a / 2
}

// cleanRing drops the repeated points of the ring, the last one equal to
// the first too
func cleanRing(pts []PointF, r []int) []int {
	out := r[:0:0]
	for _, i := range r {
		if len(out) > 0 && pts[out[len(out)-1]] == pts[i] {
			continue
		}
		out = append(out, i)
	}
	for len(out) > 1 && pts[out[0]] == pts[out[len(out)-1]] {
		out = out[:len(out)-1]
	}
	return out
}

// bridgeHole joins the hole (clockwise) to the outer ring (counterclockwise)
// by a cut from its rightmost point to a point of the ring it sees
func bridgeHole(pts []PointF, outer, hole []int) []int {
	// The rightmost point of the hole
	hi := 0
	for k, i := range hole {
		if pts[i].X > pts[hole[hi]].X || pts[i].X == pts[hole[hi]].X && pts[i].Y < pts[hole[hi]].Y {
			hi = k
		}
	}
	m := pts[hole[hi]]
	// The nearest crossing of the ray to the right with an edge of the ring
	best, bestX := -1, math.Inf(1)
	for k := range outer {
		a, b := pts[outer[k]], pts[outer[(k+1)%len(outer)]]
		if (a.Y > m.Y) == (b.Y > m.Y) && !(a.Y == m.Y || b.Y == m.Y) {
			continue
		}
		if a.Y == b.Y {
			continue
		}
		t := (m.Y - a.Y) / (b.Y - a.Y)
		if t < 0 || t > 1 {
			continue
		}
		x := a.X + t*(b.X-a.X)
		if x >= m.X && x < bestX {
			bestX, best = x, k
		}
	}
	if best < 0 {
		return outer
	}
	// The end of that edge farther right is the candidate; a reflex point
	// of the ring inside the triangle (m, crossing, candidate) blocks it:
	// then the one of those at the smallest angle to the ray is taken
	a, b := outer[best], outer[(best+1)%len(outer)]
	cand := best
	if pts[b].X > pts[a].X {
		cand = (best + 1) % len(outer)
	}
	ip := PointF{bestX, m.Y}
	cp := pts[outer[cand]]
	if ip != cp {
		bestAngle := math.Inf(1)
		for k := range outer {
			p := pts[outer[k]]
			if k == cand || p.X < m.X {
				continue
			}
			prev, next := pts[outer[(k+len(outer)-1)%len(outer)]], pts[outer[(k+1)%len(outer)]]
			if cross(prev, p, next) > 0 { // convex: can't block
				continue
			}
			if !pointInTriangle(p, m, ip, cp) {
				continue
			}
			angle := math.Abs(math.Atan2(p.Y-m.Y, p.X-m.X))
			if angle < bestAngle {
				bestAngle, cand = angle, k
			}
		}
	}
	// outer[..cand] hole[hi..] hole[..hi] hole[hi] outer[cand..]
	out := make([]int, 0, len(outer)+len(hole)+2)
	out = append(out, outer[:cand+1]...)
	for k := 0; k <= len(hole); k++ {
		out = append(out, hole[(hi+k)%len(hole)])
	}
	out = append(out, outer[cand:]...)
	return out
}

// cross returns the turn at b from a to c: positive counterclockwise
func cross(a, b, c PointF) float64 {
	return (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
}

func pointInTriangle(p, a, b, c PointF) bool {
	d1 := (p.X-b.X)*(a.Y-b.Y) - (a.X-b.X)*(p.Y-b.Y)
	d2 := (p.X-c.X)*(b.Y-c.Y) - (b.X-c.X)*(p.Y-c.Y)
	d3 := (p.X-a.X)*(c.Y-a.Y) - (c.X-a.X)*(p.Y-a.Y)
	neg := d1 < 0 || d2 < 0 || d3 < 0
	pos := d1 > 0 || d2 > 0 || d3 > 0
	return !(neg && pos)
}

// earClip cuts the ears of the counterclockwise ring off one by one
func earClip(pts []PointF, ring []int) []int {
	n := len(ring)
	if n < 3 {
		return nil
	}
	prev, next := make([]int, n), make([]int, n)
	for i := range ring {
		prev[i], next[i] = (i+n-1)%n, (i+1)%n
	}
	isEar := func(i int) bool {
		a, b, c := pts[ring[prev[i]]], pts[ring[i]], pts[ring[next[i]]]
		if cross(a, b, c) <= 0 {
			return false
		}
		// No other point of the ring inside it (the twins of the bridges,
		// at the same places as its corners, don't count)
		for k := next[next[i]]; k != prev[i]; k = next[k] {
			p := pts[ring[k]]
			if p == a || p == b || p == c {
				continue
			}
			if pointInTriangle(p, a, b, c) {
				return false
			}
		}
		return true
	}
	tris := make([]int, 0, (n-2)*3)
	left, i, stall := n, 0, 0
	for left > 3 {
		if isEar(i) {
			tris = append(tris, ring[prev[i]], ring[i], ring[next[i]])
			next[prev[i]], prev[next[i]] = next[i], prev[i]
			left--
			i, stall = prev[i], 0
			continue
		}
		i = next[i]
		stall++
		if stall > left {
			// Nothing is an ear (points in a line, a bad polygon): cut the
			// rest as a fan rather than loop
			for k := next[i]; next[k] != i; k = next[k] {
				tris = append(tris, ring[i], ring[k], ring[next[k]])
			}
			return tris
		}
	}
	tris = append(tris, ring[prev[i]], ring[i], ring[next[i]])
	return tris
}

// creaseAngle is the turn of a profile above which its sides meet at a
// sharp edge; below it they are shaded as one smooth side
const creaseAngle = 30 * math.Pi / 180

// NewExtrudeMesh returns the solid of the polygon with the holes (as for
// TriangulatePolygon) in the plane z = 0 pushed up to z = height: a plate,
// a profile, a part of a sketch. The sides turning by less than 30 degrees
// are shaded smooth (the arcs of the outline), the others meet at sharp
// edges.
func NewExtrudeMesh(outline []PointF, holes [][]PointF, height float64) *Mesh {
	m := &Mesh{}
	tris := TriangulatePolygon(outline, holes)
	if tris == nil {
		return m
	}
	all := append([]PointF(nil), outline...)
	for _, h := range holes {
		all = append(all, h...)
	}
	// The caps: the bottom faces down, the top up
	for _, top := range []bool{false, true} {
		z, nz := 0.0, -1.0
		if top {
			z, nz = height, 1
		}
		base := len(m.Positions)
		for _, p := range all {
			m.Positions = append(m.Positions, Vec3{p.X, p.Y, z})
			m.Normals = append(m.Normals, Vec3{0, 0, nz})
			m.UVs = append(m.UVs, p)
		}
		for i := 0; i+2 < len(tris); i += 3 {
			a, b, c := base+tris[i], base+tris[i+1], base+tris[i+2]
			if top {
				m.Indices = append(m.Indices, a, b, c)
			} else {
				m.Indices = append(m.Indices, a, c, b)
			}
		}
	}
	// The sides: the outline counterclockwise, the holes clockwise, so the
	// outside is to the right of each edge
	side := func(r []PointF, ccw bool) {
		r = cleanPoints(r)
		if len(r) < 3 {
			return
		}
		if (signedArea(r) > 0) != ccw {
			r = reversed(r)
		}
		n := len(r)
		// The outward normal of each edge
		en := make([]PointF, n)
		for i := range r {
			d := PointF{r[(i+1)%n].X - r[i].X, r[(i+1)%n].Y - r[i].Y}
			l := math.Hypot(d.X, d.Y)
			en[i] = PointF{d.Y / l, -d.X / l}
		}
		// The normal of the edge i at its corner k (i or i+1): smooth if
		// the turn there is small
		cornerN := func(i, k int) PointF {
			other := (i + 1) % n
			if k == i {
				other = (i + n - 1) % n
			}
			a, b := en[i], en[other]
			if math.Acos(math.Max(-1, math.Min(1, a.X*b.X+a.Y*b.Y))) < creaseAngle {
				s := PointF{a.X + b.X, a.Y + b.Y}
				l := math.Hypot(s.X, s.Y)
				return PointF{s.X / l, s.Y / l}
			}
			return a
		}
		for i := 0; i < n; i++ {
			j := (i + 1) % n
			base := len(m.Positions)
			ni, nj := cornerN(i, i), cornerN(i, j)
			for _, v := range [4]struct {
				p PointF
				n PointF
				z float64
			}{{r[i], ni, 0}, {r[j], nj, 0}, {r[j], nj, height}, {r[i], ni, height}} {
				m.Positions = append(m.Positions, Vec3{v.p.X, v.p.Y, v.z})
				m.Normals = append(m.Normals, Vec3{v.n.X, v.n.Y, 0})
				m.UVs = append(m.UVs, PointF{float64(i) / float64(n), v.z})
			}
			m.Indices = append(m.Indices, base, base+1, base+2, base, base+2, base+3)
		}
	}
	side(outline, true)
	for _, h := range holes {
		side(h, false)
	}
	return m
}

// NewLatheMesh returns the solid made by turning the profile around the y
// axis: the points are (radius, height), from the bottom up the outside,
// as the half section of a shaft, a flange, a vase is drawn. A point of
// radius 0 closes the solid there. The profile's turns of less than 30
// degrees are shaded smooth, the others are sharp edges.
func NewLatheMesh(profile []PointF, segments int) *Mesh {
	segments = max(segments, 3)
	m := &Mesh{}
	prof := cleanPoints(profile)
	n := len(prof)
	if n < 2 {
		return m
	}
	// The normal of each segment of the profile, outward: (dy, -dr) turned
	// to point away from the axis for a profile going up
	sn := make([]PointF, n-1)
	for i := 0; i+1 < n; i++ {
		d := PointF{prof[i+1].X - prof[i].X, prof[i+1].Y - prof[i].Y}
		l := math.Hypot(d.X, d.Y)
		sn[i] = PointF{d.Y / l, -d.X / l}
	}
	cornerN := func(i, k int) PointF {
		other := i - 1
		if k == i+1 {
			other = i + 1
		}
		if other < 0 || other >= n-1 {
			return sn[i]
		}
		a, b := sn[i], sn[other]
		if math.Acos(math.Max(-1, math.Min(1, a.X*b.X+a.Y*b.Y))) < creaseAngle {
			s := PointF{a.X + b.X, a.Y + b.Y}
			l := math.Hypot(s.X, s.Y)
			return PointF{s.X / l, s.Y / l}
		}
		return a
	}
	for i := 0; i+1 < n; i++ {
		base := len(m.Positions)
		for k, end := range []int{i, i + 1} {
			p, nr := prof[end], cornerN(i, end)
			for s := 0; s <= segments; s++ {
				// The last at the first exactly: the seam is closed
				th := 2 * math.Pi * float64(s%segments) / float64(segments)
				cs, sn := math.Cos(th), -math.Sin(th)
				m.Positions = append(m.Positions, Vec3{p.X * cs, p.Y, p.X * sn})
				m.Normals = append(m.Normals, Vec3{nr.X * cs, nr.Y, nr.X * sn}.Normalize())
				m.UVs = append(m.UVs, PointF{float64(s) / float64(segments), float64(i+k) / float64(n-1)})
			}
		}
		row := segments + 1
		for s := 0; s < segments; s++ {
			a, b := base+s, base+s+1
			c, d := base+row+s, base+row+s+1
			// Leave out the triangles of no area at the axis
			if prof[i].X != 0 {
				m.Indices = append(m.Indices, a, b, d)
			}
			if prof[i+1].X != 0 {
				m.Indices = append(m.Indices, a, d, c)
			}
		}
	}
	return m
}

// cleanPoints drops the repeated points, the last one equal to the first
// too
func cleanPoints(pts []PointF) []PointF {
	out := make([]PointF, 0, len(pts))
	for _, p := range pts {
		if len(out) > 0 && out[len(out)-1] == p {
			continue
		}
		out = append(out, p)
	}
	for len(out) > 1 && out[0] == out[len(out)-1] {
		out = out[:len(out)-1]
	}
	return out
}

func reversed(pts []PointF) []PointF {
	out := make([]PointF, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}
