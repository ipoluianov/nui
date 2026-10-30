package ui

import (
	"math"
	"testing"
)

func circlePts(cx, cy, r float64, n int, ccw bool) []PointF {
	pts := make([]PointF, n)
	for i := range pts {
		a := 2 * math.Pi * float64(i) / float64(n)
		if !ccw {
			a = -a
		}
		pts[i] = PointF{cx + r*math.Cos(a), cy + r*math.Sin(a)}
	}
	return pts
}

// The triangles of a polygon with holes cover its area, all of them
// counterclockwise
func TestTriangulatePolygon(t *testing.T) {
	star := make([]PointF, 0, 20)
	for i := 0; i < 20; i++ {
		r := 10.0
		if i%2 == 1 {
			r = 6
		}
		a := math.Pi * float64(i) / 10
		star = append(star, PointF{r * math.Cos(a), r * math.Sin(a)})
	}
	cases := []struct {
		name    string
		outline []PointF
		holes   [][]PointF
	}{
		{"square", []PointF{{0, 0}, {4, 0}, {4, 4}, {0, 4}}, nil},
		{"square, clockwise, a hole", []PointF{{0, 0}, {0, 4}, {4, 4}, {4, 0}}, [][]PointF{{{1, 1}, {2, 1}, {2, 2}, {1, 2}}}},
		{"star, holes", star, [][]PointF{circlePts(0, 0, 2, 24, true), circlePts(3.5, 0, 1, 16, false), circlePts(-3.5, 0, 1, 16, true), circlePts(0, 3.5, 1, 12, true)}},
		{"plate with a slot", []PointF{{0, 0}, {10, 0}, {10, 6}, {0, 6}}, [][]PointF{{{2, 2}, {8, 2}, {8, 3}, {2, 3}}, circlePts(5, 4.5, 0.6, 20, true)}},
	}
	for _, c := range cases {
		all := append([]PointF(nil), c.outline...)
		want := math.Abs(signedArea(c.outline))
		for _, h := range c.holes {
			all = append(all, h...)
			want -= math.Abs(signedArea(h))
		}
		tris := TriangulatePolygon(c.outline, c.holes)
		got := 0.0
		for i := 0; i < len(tris); i += 3 {
			a := signedArea([]PointF{all[tris[i]], all[tris[i+1]], all[tris[i+2]]})
			if a < -1e-12 {
				t.Fatalf("%s: triangle %d is clockwise", c.name, i/3)
			}
			got += a
		}
		if math.Abs(got-want) > 1e-9*math.Max(1, want) {
			t.Errorf("%s: the triangles cover %v, the polygon %v", c.name, got, want)
		}
	}
}

// closedSolid checks that every edge of the mesh is shared by two
// triangles going opposite ways, the normals agree with the windings, and
// returns its volume
func closedSolid(t *testing.T, name string, m *Mesh) float64 {
	t.Helper()
	type edge struct{ a, b Vec3 }
	count := map[edge]int{}
	vol := 0.0
	for i := 0; i < m.TriangleCount(); i++ {
		a, b, c := m.triangle(i)
		pa, pb, pc := m.Positions[a], m.Positions[b], m.Positions[c]
		fn := pb.Sub(pa).Cross(pc.Sub(pa))
		if fn.Length() < 1e-12 {
			continue
		}
		if avg := m.Normals[a].Add(m.Normals[b]).Add(m.Normals[c]); fn.Dot(avg) <= 0 {
			t.Fatalf("%s: triangle %d winds against its normals", name, i)
		}
		count[edge{pa, pb}]++
		count[edge{pb, pc}]++
		count[edge{pc, pa}]++
		vol += pa.Dot(pb.Cross(pc)) / 6
	}
	for e, n := range count {
		if back := count[edge{e.b, e.a}]; back != n {
			t.Fatalf("%s: the edge %v-%v is not closed (%d and %d)", name, e.a, e.b, n, back)
		}
	}
	return vol
}

func TestExtrudeMesh(t *testing.T) {
	outline := []PointF{{0, 0}, {10, 0}, {10, 6}, {0, 6}}
	holes := [][]PointF{circlePts(3, 3, 1, 32, true), circlePts(7, 3, 1.5, 32, false)}
	m := NewExtrudeMesh(outline, holes, 2)
	area := 60 - math.Abs(signedArea(holes[0])) - math.Abs(signedArea(holes[1]))
	if v := closedSolid(t, "plate", m); math.Abs(v-area*2) > 1e-9 {
		t.Fatalf("the volume of the plate is %v, want %v", v, area*2)
	}
	// The sides of the holes are smooth, the corners of the plate sharp
	creases := len(m.wireEdges(false)) / 2
	// 4 corners, the 2+2 rings of the holes, the 4+4 edges of the outline
	if want := 4 + 2*32 + 2*32 + 8; creases != want {
		t.Errorf("%d creases, want %d", creases, want)
	}
}

func TestLatheMesh(t *testing.T) {
	// A cylinder of radius 1 and height 2 with a groove
	profile := []PointF{{0, 0}, {1, 0}, {1, 0.8}, {0.8, 0.9}, {1, 1}, {1, 2}, {0, 2}}
	m := NewLatheMesh(profile, 64)
	vol := closedSolid(t, "shaft", m)
	// Pappus: the area of the profile times the way of its centroid; the
	// groove is a triangle of area 0.02 at the radius 2.8/3. A circle of
	// 64 sides holds sin(2pi/64)*64/(2pi) of the round one.
	k := math.Sin(2*math.Pi/64) * 64 / (2 * math.Pi)
	want := (2*math.Pi*0.5*2 - 2*math.Pi*2.8/3*0.02) * k
	if math.Abs(vol-want) > 1e-9 {
		t.Fatalf("the volume of the shaft is %v, about %v", vol, want)
	}
}
