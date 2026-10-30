package ui

import "math"

// RectF is a rectangle of fractional coordinates: an area of a drawing, of
// a map, the box of a shape. Min is the corner with the smaller
// coordinates, whichever way y goes.
type RectF struct {
	MinX, MinY, MaxX, MaxY float64
}

// RectFromPoints returns the rectangle with the two corners, in any order
func RectFromPoints(a, b PointF) RectF {
	return RectF{math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Max(a.X, b.X), math.Max(a.Y, b.Y)}
}

// EmptyRectF returns a rectangle that holds nothing, to grow by Union and
// AddPoint
func EmptyRectF() RectF {
	return RectF{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
}

// Empty reports whether the rectangle holds no point
func (r RectF) Empty() bool { return !(r.MinX <= r.MaxX && r.MinY <= r.MaxY) }

func (r RectF) Width() float64  { return r.MaxX - r.MinX }
func (r RectF) Height() float64 { return r.MaxY - r.MinY }

// Center returns the middle of the rectangle
func (r RectF) Center() PointF { return PointF{(r.MinX + r.MaxX) / 2, (r.MinY + r.MaxY) / 2} }

// Contains reports whether the point is in the rectangle, on its edges too
func (r RectF) Contains(p PointF) bool {
	return p.X >= r.MinX && p.X <= r.MaxX && p.Y >= r.MinY && p.Y <= r.MaxY
}

// ContainsRect reports whether all of q is in the rectangle
func (r RectF) ContainsRect(q RectF) bool {
	return q.MinX >= r.MinX && q.MaxX <= r.MaxX && q.MinY >= r.MinY && q.MaxY <= r.MaxY
}

// Intersects reports whether the rectangles share a point
func (r RectF) Intersects(q RectF) bool {
	return r.MinX <= q.MaxX && q.MinX <= r.MaxX && r.MinY <= q.MaxY && q.MinY <= r.MaxY
}

// Union returns the rectangle that holds both
func (r RectF) Union(q RectF) RectF {
	return RectF{math.Min(r.MinX, q.MinX), math.Min(r.MinY, q.MinY), math.Max(r.MaxX, q.MaxX), math.Max(r.MaxY, q.MaxY)}
}

// AddPoint returns the rectangle grown to hold the point
func (r RectF) AddPoint(p PointF) RectF {
	return RectF{math.Min(r.MinX, p.X), math.Min(r.MinY, p.Y), math.Max(r.MaxX, p.X), math.Max(r.MaxY, p.Y)}
}

// Inset returns the rectangle with d taken off every side; a negative d
// grows it
func (r RectF) Inset(d float64) RectF {
	return RectF{r.MinX + d, r.MinY + d, r.MaxX - d, r.MaxY - d}
}

// SpatialIndex finds the items in an area quickly among many: the shapes of
// a drawing to draw in the part of it seen, the ones under the mouse, the
// ones in a selection box. Each item is kept with its box; a search tests
// only the items in the parts of the plane the area reaches.
//
//	index := ui.NewSpatialIndex[*Shape]()
//	index.Insert(shape, shape.Bounds())
//	index.Search(view.VisibleRect(), func(s *Shape) bool {
//		s.Draw(cnv)
//		return true // go on
//	})
//
// It is a quadtree: an item is kept in the smallest square of the tree it
// fits in. It grows to hold items anywhere. It is not safe for concurrent
// changes; searches may run together.
type SpatialIndex[T comparable] struct {
	root  *spatialNode[T]
	where map[T]spatialPlace[T]
}

type spatialNode[T comparable] struct {
	cx, cy, half float64 // the square: center and half its side
	items        []spatialItem[T]
	kids         *[4]*spatialNode[T]
	count        int // the items here and below
}

type spatialItem[T comparable] struct {
	item T
	box  RectF
}

type spatialPlace[T comparable] struct {
	box RectF
}

// spatialSplit is how many items a square keeps before it is split
const spatialSplit = 16

// spatialMinHalf: squares smaller than this are not split (points on top
// of each other)
const spatialMinHalf = 1e-9

// NewSpatialIndex returns an empty index
func NewSpatialIndex[T comparable]() *SpatialIndex[T] {
	return &SpatialIndex[T]{where: map[T]spatialPlace[T]{}}
}

// Len returns the count of the items
func (s *SpatialIndex[T]) Len() int { return len(s.where) }

// Clear removes all the items
func (s *SpatialIndex[T]) Clear() {
	s.root = nil
	clear(s.where)
}

// Insert adds the item with its box, or moves it there if it is in the
// index already. A box with NaN coordinates is not kept.
func (s *SpatialIndex[T]) Insert(item T, box RectF) {
	if _, ok := s.where[item]; ok {
		s.Remove(item)
	}
	if box.Empty() || math.IsInf(box.MinX, 0) || math.IsInf(box.MinY, 0) || math.IsInf(box.MaxX, 0) || math.IsInf(box.MaxY, 0) {
		return
	}
	if s.root == nil {
		c := box.Center()
		half := math.Max(math.Max(box.Width(), box.Height()), 1)
		s.root = &spatialNode[T]{cx: c.X, cy: c.Y, half: half}
	}
	// Grow the root until the box is in it
	for !s.root.holds(box) {
		old := s.root
		c := box.Center()
		// The new root is twice as big, old in one of its quarters, towards
		// the box
		dx, dy := 1.0, 1.0
		if c.X < old.cx {
			dx = -1
		}
		if c.Y < old.cy {
			dy = -1
		}
		root := &spatialNode[T]{cx: old.cx + dx*old.half, cy: old.cy + dy*old.half, half: old.half * 2}
		if old.count > 0 {
			root.kids = &[4]*spatialNode[T]{}
			root.kids[root.quarter(old.cx, old.cy)] = old
			root.count = old.count
		}
		s.root = root
	}
	s.root.insert(spatialItem[T]{item, box})
	s.where[item] = spatialPlace[T]{box}
}

// Remove takes the item out; false if it was not there
func (s *SpatialIndex[T]) Remove(item T) bool {
	place, ok := s.where[item]
	if !ok {
		return false
	}
	delete(s.where, item)
	s.root.remove(item, place.box)
	return true
}

// Box returns the box the item was inserted with
func (s *SpatialIndex[T]) Box(item T) (RectF, bool) {
	p, ok := s.where[item]
	return p.box, ok
}

// Search calls f with every item whose box meets the area, until f
// returns false
func (s *SpatialIndex[T]) Search(area RectF, f func(item T) bool) {
	if s.root != nil && !area.Empty() {
		s.root.search(area, f)
	}
}

// All calls f with every item, until f returns false
func (s *SpatialIndex[T]) All(f func(item T) bool) {
	for item := range s.where {
		if !f(item) {
			return
		}
	}
}

// Bounds returns the box of all the items; false if there are none
func (s *SpatialIndex[T]) Bounds() (RectF, bool) {
	r := EmptyRectF()
	for _, p := range s.where {
		r = r.Union(p.box)
	}
	return r, !r.Empty()
}

// Nearest returns the items whose boxes are within the distance of the
// point, nearest box first: the candidates of a pick, to be tested by
// their real shapes
func (s *SpatialIndex[T]) Nearest(p PointF, distance float64, limit int) []T {
	type cand struct {
		item T
		d    float64
	}
	var cands []cand
	s.Search(RectF{p.X - distance, p.Y - distance, p.X + distance, p.Y + distance}, func(item T) bool {
		b := s.where[item].box
		dx := math.Max(math.Max(b.MinX-p.X, 0), p.X-b.MaxX)
		dy := math.Max(math.Max(b.MinY-p.Y, 0), p.Y-b.MaxY)
		if d := math.Hypot(dx, dy); d <= distance {
			cands = append(cands, cand{item, d})
		}
		return true
	})
	// Insertion sort: the candidates are few
	for i := 1; i < len(cands); i++ {
		for k := i; k > 0 && cands[k].d < cands[k-1].d; k-- {
			cands[k], cands[k-1] = cands[k-1], cands[k]
		}
	}
	if limit > 0 && len(cands) > limit {
		cands = cands[:limit]
	}
	out := make([]T, len(cands))
	for i, c := range cands {
		out[i] = c.item
	}
	return out
}

func (n *spatialNode[T]) holds(b RectF) bool {
	return b.MinX >= n.cx-n.half && b.MaxX <= n.cx+n.half && b.MinY >= n.cy-n.half && b.MaxY <= n.cy+n.half
}

// quarter returns the quarter of the node the point is in
func (n *spatialNode[T]) quarter(x, y float64) int {
	q := 0
	if x >= n.cx {
		q |= 1
	}
	if y >= n.cy {
		q |= 2
	}
	return q
}

// kidFor returns the quarter the box fits in, -1 if it crosses the middle
func (n *spatialNode[T]) kidFor(b RectF) int {
	if b.MaxX < n.cx {
		if b.MaxY < n.cy {
			return 0
		}
		if b.MinY >= n.cy {
			return 2
		}
	} else if b.MinX >= n.cx {
		if b.MaxY < n.cy {
			return 1
		}
		if b.MinY >= n.cy {
			return 3
		}
	}
	return -1
}

func (n *spatialNode[T]) kid(q int) *spatialNode[T] {
	if n.kids == nil {
		n.kids = &[4]*spatialNode[T]{}
	}
	if n.kids[q] == nil {
		h := n.half / 2
		dx, dy := -h, -h
		if q&1 != 0 {
			dx = h
		}
		if q&2 != 0 {
			dy = h
		}
		n.kids[q] = &spatialNode[T]{cx: n.cx + dx, cy: n.cy + dy, half: h}
	}
	return n.kids[q]
}

// insert adds the item here or below
func (n *spatialNode[T]) insert(it spatialItem[T]) {
	n.count++
	if n.kids != nil && n.half > spatialMinHalf {
		if q := n.kidFor(it.box); q >= 0 {
			n.kid(q).insert(it)
			return
		}
	}
	n.items = append(n.items, it)
	if n.kids == nil && len(n.items) > spatialSplit && n.half > spatialMinHalf {
		// Split: the items that fit in a quarter go down
		items := n.items
		n.items = nil
		n.kids = &[4]*spatialNode[T]{}
		for _, x := range items {
			n.count-- // counted again by insert
			n.insert(x)
		}
	}
}

func (n *spatialNode[T]) remove(item T, box RectF) bool {
	for i, x := range n.items {
		if x.item == item {
			last := len(n.items) - 1
			n.items[i] = n.items[last]
			var zero spatialItem[T]
			n.items[last] = zero
			n.items = n.items[:last]
			n.count--
			return true
		}
	}
	if n.kids == nil {
		return false
	}
	q := n.kidFor(box)
	if q < 0 || n.kids[q] == nil {
		return false
	}
	if n.kids[q].remove(item, box) {
		n.count--
		if n.kids[q].count == 0 {
			n.kids[q] = nil
		}
		return true
	}
	return false
}

func (n *spatialNode[T]) search(area RectF, f func(T) bool) bool {
	for _, x := range n.items {
		if x.box.Intersects(area) && !f(x.item) {
			return false
		}
	}
	if n.kids == nil {
		return true
	}
	for _, k := range n.kids {
		if k == nil || k.count == 0 {
			continue
		}
		if area.MaxX < k.cx-k.half || area.MinX > k.cx+k.half || area.MaxY < k.cy-k.half || area.MinY > k.cy+k.half {
			continue
		}
		if !k.search(area, f) {
			return false
		}
	}
	return true
}
