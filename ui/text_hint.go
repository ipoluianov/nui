package ui

import (
	"math"
	"sort"
	"sync"
)

// A light vertical hinter, as FreeType's light autohinter: the glyphs are
// fitted to the pixels up and down, not across, so the text keeps its
// widths and its shapes and only gets sharp.
//
//   - The height of the small letters (x-height) is scaled to whole pixels:
//     the tops of a, e, n, x... lie on a row of pixels instead of across two.
//   - The lines the letters stand on and reach to - the baseline, the
//     x-height, the height of the capitals, of the ascenders, the
//     descenders - are put on the edges of the pixels; the overshoot of
//     round letters (the top of o past the top of x) is dropped while it is
//     less than half a pixel.
//   - The horizontal strokes - the bars of e, H, t, the serifs - get a
//     whole count of pixels, at least one, and their edges are put on the
//     edges of the pixels.
//   - Every other point of the outline is moved in proportion between the
//     edges around it.

// hintZones are the heights of the font that its letters line up on, in
// ems, y down (above the baseline is negative); 0 where the font has no
// letter to measure
type hintZones struct {
	xHeight, xOver       float64 // the top of x; of o
	capHeight, capOver   float64 // H; O
	ascender, descender  float64 // d; p
	baseOver             float64 // the bottom of o
	hasX, hasCap, hasAsc bool
}

var hintZoneCache sync.Map // font -> *hintZones

// zonesOf returns the zones of the font, measured on its letters by shape
// (nil for a letter it lacks)
func zonesOf(font any, shape func(r rune) *glyphShape) *hintZones {
	if z, ok := hintZoneCache.Load(font); ok {
		return z.(*hintZones)
	}
	z := &hintZones{}
	top := func(r rune) (float64, float64, bool) {
		s := shape(r)
		if s == nil || len(s.ops) == 0 {
			return 0, 0, false
		}
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, op := range s.ops {
			// The points on the outline: TrueType and CFF fonts have them
			// at the extremes
			p := op.p[op.n()-1]
			lo, hi = min(lo, p.Y), max(hi, p.Y)
		}
		return lo, hi, true
	}
	if t, _, ok := top('x'); ok && t < 0 {
		z.xHeight, z.hasX = t, true
		z.xOver = t
		if o, b, ok := top('o'); ok {
			z.xOver = min(t, o)
			z.baseOver = max(0, b)
		}
	}
	if t, _, ok := top('H'); ok && t < 0 {
		z.capHeight, z.hasCap = t, true
		z.capOver = t
		if o, _, ok := top('O'); ok {
			z.capOver = min(t, o)
		}
	}
	if t, _, ok := top('d'); ok && t < 0 {
		z.ascender, z.hasAsc = t, true
	}
	if _, b, ok := top('p'); ok && b > 0 {
		z.descender = b
	}
	hintZoneCache.Store(font, z)
	return z
}

// n returns the count of the points of the op
func (op glyphOp) n() int {
	switch op.kind {
	case 'Q':
		return 2
	case 'C':
		return 3
	}
	return 1
}

// hintEdge is a horizontal edge of a glyph: the top of a stroke (the ink
// under it) or its bottom
type hintEdge struct {
	y      float64 // pixels, y down
	x0, x1 float64
	top    bool // the ink is under it
	target float64
	fixed  bool
}

// hintGlyph returns the outline of the glyph at the size in pixels, y down
// from the baseline, fitted to the pixels vertically; italic slants it
// after
func hintGlyph(s *glyphShape, z *hintZones, size float64) []glyphOp {
	ops := make([]glyphOp, len(s.ops))
	// The x-height scaled to whole pixels, as FreeType does: up from
	// three eighths of a pixel
	sy := size
	if z != nil && z.hasX {
		xh := -z.xHeight * size
		if fitted := math.Floor(xh + 40.0/64); fitted >= 1 && xh > 0 {
			sy = size * fitted / xh
		}
	}
	for i, op := range s.ops {
		ops[i].kind = op.kind
		for k := 0; k < op.n(); k++ {
			ops[i].p[k] = PointF{op.p[k].X * size, op.p[k].Y * sy}
		}
	}
	edges := findEdges(ops, size)
	if len(edges) == 0 {
		return ops
	}
	zone := func(e *hintEdge) {
		if z == nil {
			return
		}
		snap := func(flat, over float64) bool {
			flat, over = flat*sy, over*sy
			lo, hi := min(flat, over), max(flat, over)
			fuzz := max(0.3, size/40)
			if e.y < lo-fuzz || e.y > hi+fuzz {
				return false
			}
			// The overshoot kept only from half a pixel
			d := e.y - flat
			if math.Abs(d) < 0.5 {
				d = 0
			}
			e.target, e.fixed = math.Round(flat)+math.Round(d), true
			return true
		}
		if e.top {
			if z.hasX && snap(z.xHeight, z.xOver) {
				return
			}
			if z.hasCap && snap(z.capHeight, z.capOver) {
				return
			}
			if z.hasAsc && snap(z.ascender, z.ascender) {
				return
			}
		} else {
			if snap(0, z.baseOver) {
				return
			}
			if z.descender > 0 {
				snap(z.descender, z.descender)
			}
		}
	}
	for i := range edges {
		zone(&edges[i])
	}
	// The strokes: a top over the nearest bottom under it that overlaps it
	maxStem := 0.3 * size
	for i := range edges {
		t := &edges[i]
		if !t.top {
			continue
		}
		best := -1
		for j := range edges {
			b := &edges[j]
			if b.top || b.y <= t.y || b.y-t.y > maxStem || b.x1 < t.x0 || b.x0 > t.x1 {
				continue
			}
			if best < 0 || b.y < edges[best].y {
				best = j
			}
		}
		if best < 0 {
			continue
		}
		b := &edges[best]
		w := b.y - t.y
		fw := math.Max(1, math.Round(w))
		switch {
		case t.fixed && b.fixed:
		case t.fixed:
			b.target, b.fixed = t.target+fw, true
		case b.fixed:
			t.target, t.fixed = b.target-fw, true
		default:
			// The middle of the stroke kept as near as can be
			t.target = math.Round((t.y + b.y - fw) / 2)
			b.target = t.target + fw
			t.fixed, b.fixed = true, true
		}
	}
	for i := range edges {
		if !edges[i].fixed {
			edges[i].target, edges[i].fixed = math.Round(edges[i].y), true
		}
	}
	// From the original heights to the fitted ones, in order: an edge out
	// of order with the ones before is left out
	sort.Slice(edges, func(a, b int) bool { return edges[a].y < edges[b].y })
	var from, to []float64
	for _, e := range edges {
		if n := len(from); n > 0 {
			if e.y-from[n-1] < 1e-6 {
				continue
			}
			if e.target < to[n-1] {
				continue
			}
		}
		from, to = append(from, e.y), append(to, e.target)
	}
	move := func(y float64) float64 {
		n := len(from)
		switch {
		case y <= from[0]:
			return y + to[0] - from[0]
		case y >= from[n-1]:
			return y + to[n-1] - from[n-1]
		}
		i := sort.SearchFloat64s(from, y)
		a, b := i-1, i
		f := (y - from[a]) / (from[b] - from[a])
		return to[a] + f*(to[b]-to[a])
	}
	for i := range ops {
		for k := 0; k < ops[i].n(); k++ {
			ops[i].p[k].Y = move(ops[i].p[k].Y)
		}
	}
	return ops
}

// findEdges finds the horizontal edges of the outline, in pixels: the
// runs of it that go across, near flat, long enough - the flat tops and
// bottoms of strokes and the tops and bottoms of round letters
func findEdges(ops []glyphOp, size float64) []hintEdge {
	// The outline as polylines
	var contours [][]PointF
	var cur []PointF
	var at PointF
	for _, op := range ops {
		switch op.kind {
		case 'M':
			if len(cur) > 1 {
				contours = append(contours, cur)
			}
			cur = []PointF{op.p[0]}
		case 'L':
			cur = append(cur, op.p[0])
		case 'Q':
			for k := 1; k <= 8; k++ {
				t := float64(k) / 8
				u := 1 - t
				cur = append(cur, PointF{u*u*at.X + 2*u*t*op.p[0].X + t*t*op.p[1].X, u*u*at.Y + 2*u*t*op.p[0].Y + t*t*op.p[1].Y})
			}
		case 'C':
			for k := 1; k <= 8; k++ {
				t := float64(k) / 8
				u := 1 - t
				a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
				cur = append(cur, PointF{a*at.X + b*op.p[0].X + c*op.p[1].X + d*op.p[2].X, a*at.Y + b*op.p[0].Y + c*op.p[1].Y + d*op.p[2].Y})
			}
		}
		at = op.p[op.n()-1]
	}
	if len(cur) > 1 {
		contours = append(contours, cur)
	}
	shape := NewPath()
	for _, c := range contours {
		shape.Polygon(c...)
	}
	minLen := max(0.8, size/20)
	eps := max(0.15, size/60)
	var edges []hintEdge
	for _, c := range contours {
		n := len(c)
		flat := func(i int) bool { // the piece from point i to i+1
			a, b := c[i%n], c[(i+1)%n]
			dx, dy := b.X-a.X, b.Y-a.Y
			return math.Abs(dx) > 1e-9 && math.Abs(dy) <= 0.25*math.Abs(dx)
		}
		// Start where a run cannot go on from the piece before
		start := -1
		for i := 0; i < n; i++ {
			if !flat(i) {
				start = i + 1
				break
			}
		}
		if start < 0 {
			continue // flat all round: nothing to fit
		}
		for k := 0; k < n; {
			i := (start + k) % n
			if !flat(i) {
				k++
				continue
			}
			// A run of flat pieces going the same way
			dir := c[(i+1)%n].X - c[i].X
			x0, x1 := math.Inf(1), math.Inf(-1)
			ylo, yhi := math.Inf(1), math.Inf(-1)
			j := k
			for j < n {
				ii := (start + j) % n
				d := c[(ii+1)%n].X - c[ii].X
				if !flat(ii) || d*dir <= 0 {
					break
				}
				for _, p := range []PointF{c[ii], c[(ii+1)%n]} {
					x0, x1 = min(x0, p.X), max(x1, p.X)
					ylo, yhi = min(ylo, p.Y), max(yhi, p.Y)
				}
				j++
			}
			k = j
			if x1-x0 < minLen {
				continue
			}
			mx := (x0 + x1) / 2
			mid := (ylo + yhi) / 2
			below, above := shape.Contains(mx, mid+eps), shape.Contains(mx, mid-eps)
			if below == above {
				continue
			}
			// The edge at the extreme of the run: the top of a round top
			e := hintEdge{x0: x0, x1: x1, top: below}
			if below {
				e.y = ylo
			} else {
				e.y = yhi
			}
			edges = append(edges, e)
		}
	}
	// Edges of a kind at about one height are one
	sort.Slice(edges, func(a, b int) bool { return edges[a].y < edges[b].y })
	var out []hintEdge
	for _, e := range edges {
		merged := false
		for i := range out {
			if out[i].top == e.top && math.Abs(out[i].y-e.y) < 0.25 {
				out[i].x0, out[i].x1 = min(out[i].x0, e.x0), max(out[i].x1, e.x1)
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, e)
		}
	}
	return out
}
