package ui

import (
	"math/rand"
	"sort"
	"testing"
)

// The index finds what a search of all the items finds, through inserts,
// moves and removes, near the origin and far from it
func TestSpatialIndexMatchesBruteForce(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	idx := NewSpatialIndex[int]()
	boxes := map[int]RectF{}
	randBox := func() RectF {
		// Mostly small items, some long ones, a few far away
		x, y := rng.Float64()*1000, rng.Float64()*1000
		if rng.Intn(50) == 0 {
			x, y = x*1e6, -y*1e6
		}
		w, h := rng.Float64()*5, rng.Float64()*5
		if rng.Intn(20) == 0 {
			w = rng.Float64() * 800
		}
		return RectF{x, y, x + w, y + h}
	}
	for i := 0; i < 5000; i++ {
		b := randBox()
		idx.Insert(i, b)
		boxes[i] = b
	}
	for i := 0; i < 1500; i++ {
		k := rng.Intn(5000)
		if rng.Intn(2) == 0 {
			if _, ok := boxes[k]; ok != idx.Remove(k) {
				t.Fatalf("remove %d: %v", k, ok)
			}
			delete(boxes, k)
		} else {
			b := randBox()
			idx.Insert(k, b)
			boxes[k] = b
		}
	}
	if idx.Len() != len(boxes) {
		t.Fatalf("len %d, want %d", idx.Len(), len(boxes))
	}
	for q := 0; q < 300; q++ {
		x, y := rng.Float64()*1100-50, rng.Float64()*1100-50
		area := RectF{x, y, x + rng.Float64()*200, y + rng.Float64()*200}
		if q%50 == 0 {
			area = RectF{-1e10, -1e10, 1e10, 1e10}
		}
		var got, want []int
		idx.Search(area, func(i int) bool { got = append(got, i); return true })
		for i, b := range boxes {
			if b.Intersects(area) {
				want = append(want, i)
			}
		}
		sort.Ints(got)
		sort.Ints(want)
		if len(got) != len(want) {
			t.Fatalf("area %v: %d found, want %d", area, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("area %v: found %v, want %v", area, got[i], want[i])
			}
		}
	}
	near := idx.Nearest(PointF{500, 500}, 20, 5)
	for i := 1; i < len(near); i++ {
		if boxDistance(boxes[near[i-1]], PointF{500, 500}) > boxDistance(boxes[near[i]], PointF{500, 500}) {
			t.Fatal("Nearest not sorted")
		}
	}
}

func boxDistance(b RectF, p PointF) float64 {
	dx := max(b.MinX-p.X, 0, p.X-b.MaxX)
	dy := max(b.MinY-p.Y, 0, p.Y-b.MaxY)
	return dx*dx + dy*dy
}

func BenchmarkSpatialSearch(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	idx := NewSpatialIndex[int]()
	for i := 0; i < 100000; i++ {
		x, y := rng.Float64()*10000, rng.Float64()*10000
		idx.Insert(i, RectF{x, y, x + 10, y + 10})
	}
	b.ResetTimer()
	n := 0
	for i := 0; i < b.N; i++ {
		x, y := rng.Float64()*9000, rng.Float64()*9000
		idx.Search(RectF{x, y, x + 1000, y + 600}, func(int) bool { n++; return true })
	}
}
