package ui

import (
	"math/rand"
	"sort"
	"testing"
)

// The translucent triangles come from the far to the near, those as far in
// the order they were added: the same as a stable sort, over several frames
// (the buffers are reused)
func TestSortTranslucent(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	r := NewRenderer3D()
	for _, n := range []int{0, 1, 2, 17, 5000, 300} {
		r.trans = r.trans[:0]
		for i := 0; i < n; i++ {
			// Few depths, so many are as far
			r.trans = append(r.trans, rTri{v: [3]int32{int32(i)}, sortKey: float32(rnd.Intn(20))})
		}
		want := append([]rTri(nil), r.trans...)
		sort.SliceStable(want, func(i, j int) bool { return want[i].sortKey > want[j].sortKey })
		r.sortTranslucent()
		if len(r.trans) != n {
			t.Fatalf("n = %d: %d triangles after sorting", n, len(r.trans))
		}
		for i := range want {
			if r.trans[i] != want[i] {
				t.Fatalf("n = %d: at %d triangle %d (key %g), want %d (key %g)", n, i,
					r.trans[i].v[0], r.trans[i].sortKey, want[i].v[0], want[i].sortKey)
			}
		}
	}
}

// BenchmarkSortTranslucent: the triangles of 100 000 translucent lines
func BenchmarkSortTranslucent(b *testing.B) {
	rnd := rand.New(rand.NewSource(1))
	src := make([]rTri, 200000)
	for i := range src {
		src[i].sortKey = rnd.Float32()
	}
	r := NewRenderer3D()
	for b.Loop() {
		r.trans = append(r.trans[:0], src...)
		r.sortTranslucent()
	}
}
