package ui

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func auditPaint(t *testing.T, f *Form) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	f.paintScaled(img, 1)
}

func TestAuditSliderNaN(t *testing.T) {
	s := NewSlider()
	f := newTestForm(t, s)
	s.SetRange(0, 10)
	s.SetValue(math.NaN())
	t.Logf("value after SetValue(NaN) = %v", s.Value())
	auditPaint(t, f)
	s.ProcessKeyDown(KeyArrowRight, KeyModifiers{})
	t.Logf("after right arrow = %v", s.Value())
	s.SetRange(math.NaN(), 10)
	t.Logf("range NaN: min %v max %v val %v", s.Min(), s.Max(), s.Value())
	auditPaint(t, f)
	s.SetRange(0, 10)
	s.SetStep(math.NaN())
	t.Logf("step NaN: step %v val %v", s.Step(), s.Value())
	s.SetValue(5)
	t.Logf("value %v", s.Value())
	s.SetStep(math.Inf(1))
	s.SetValue(5)
	t.Logf("step inf: value %v", s.Value())
	s.SetStep(0)
	s.SetRange(-math.MaxFloat64, math.MaxFloat64)
	s.SetValue(1)
	auditPaint(t, f)
	s.ProcessKeyDown(KeyArrowRight, KeyModifiers{})
	t.Logf("huge range key: %v", s.Value())
	s.SetTickInterval(1e-300)
	auditPaint(t, f)
}

func TestAuditSpatialHuge(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		s := NewSpatialIndex[int]()
		s.Insert(1, RectF{0, 0, 1, 1})
		s.Insert(2, RectF{1.7e308, 1.7e308, 1.75e308, 1.75e308})
		n := 0
		s.Search(RectF{1.6e308, 1.6e308, 1.79e308, 1.79e308}, func(int) bool { n++; return true })
		t.Logf("found %d", n)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SpatialIndex.Insert hangs on a box at 1.7e308")
	}
}

func TestAuditSpatialWide(t *testing.T) {
	s := NewSpatialIndex[int]()
	s.Insert(1, RectF{-1.7e308, -1.7e308, 1.7e308, 1.7e308})
	for i := 0; i < 40; i++ {
		s.Insert(i+2, RectF{float64(i), float64(i), float64(i) + 1, float64(i) + 1})
	}
	n := 0
	s.Search(RectF{5, 5, 6, 6}, func(int) bool { n++; return true })
	t.Logf("found %d (want 4)", n)
	for i := 0; i < 40; i++ {
		if !s.Remove(i + 2) {
			t.Errorf("remove %d failed", i+2)
		}
	}
}

func TestAuditMeshNegativeIndex(t *testing.T) {
	m := NewBoxMesh(1, 1, 1)
	m.Indices = append(m.Indices, -1, 0, 1)
	sc := NewScene3D()
	sc.Add(m, Material{})
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic: %v", r)
		}
	}()
	NewRenderer3D().RenderImage(64, 64, sc, NewCamera(V3(3, 3, 3), V3(0, 0, 0)))
}

func TestAuditMeshNegativeIndexNormals(t *testing.T) {
	m := &Mesh{Positions: []Vec3{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}, Indices: []int{0, 1, -2}}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ComputeNormals panic: %v", r)
		}
	}()
	m.ComputeNormals()
}

func TestAuditTriangulateCollinear(t *testing.T) {
	tris := TriangulatePolygon([]PointF{{0, 0}, {1, 0}, {2, 0}}, nil)
	t.Logf("collinear 3: %v", tris)
	tris = TriangulatePolygon([]PointF{{0, 0}, {1, 0}, {2, 0}, {3, 0}}, nil)
	t.Logf("collinear 4: %v", tris)
}

func TestAuditTriangulateRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for iter := 0; iter < 20000; iter++ {
		n := rng.Intn(12)
		pts := make([]PointF, n)
		for i := range pts {
			pts[i] = PointF{float64(rng.Intn(6)), float64(rng.Intn(6))}
		}
		var holes [][]PointF
		if rng.Intn(2) == 0 {
			h := make([]PointF, rng.Intn(6))
			for i := range h {
				h[i] = PointF{float64(rng.Intn(6)), float64(rng.Intn(6))}
			}
			holes = append(holes, h)
		}
		total := n
		for _, h := range holes {
			total += len(h)
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic %v on %v %v", r, pts, holes)
				}
			}()
			tris := TriangulatePolygon(pts, holes)
			for _, i := range tris {
				if i < 0 || i >= total {
					t.Fatalf("index %d out of %d for %v %v", i, total, pts, holes)
				}
			}
			m := NewExtrudeMesh(pts, holes, 1)
			for _, nn := range m.Normals {
				if math.IsNaN(nn.X) || math.IsNaN(nn.Y) {
					t.Fatalf("NaN normal for %v %v", pts, holes)
				}
			}
		}()
	}
}

func TestAuditRenderNaNInf(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1), 1e300, -1e300} {
		m := &Mesh{Positions: []Vec3{{0, 0, 0}, {1, 0, 0}, {v, 1, 0}, {0, 0, 1}}, Indices: []int{0, 1, 2, 0, 2, 3}}
		sc := NewScene3D()
		sc.Add(m, Material{EdgeColor: color.RGBA{0, 0, 0, 255}})
		sc.AddLine(V3(0, 0, 0), V3(v, v, 0), color.RGBA{0, 0, 0, 255}, 2)
		sc.AddPoint(V3(v, 0, 0), color.RGBA{0, 0, 0, 255}, 3)
		sc.AddLabel(V3(v, 0, 0), "x", color.RGBA{0, 0, 0, 255})
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("v=%v panic: %v", v, r)
				}
			}()
			NewRenderer3D().RenderImage(64, 64, sc, NewCamera(V3(3, 3, 3), V3(0, 0, 0)))
		}()
	}
}

func TestAuditCameraAtTarget(t *testing.T) {
	sc := NewScene3D()
	sc.Add(NewBoxMesh(1, 1, 1), Material{})
	for _, cam := range []Camera{
		NewCamera(V3(0, 0, 0), V3(0, 0, 0)),
		{Position: V3(0, 0, 5), Target: V3(0, 0, 0), Near: 1, Far: 1},
		{Position: V3(0, 0, 5), Target: V3(0, 0, 0), Near: math.NaN()},
		{Position: V3(0, 0, 5), Target: V3(0, 0, 0), Orthographic: true},
		{Position: V3(0, 0, 0), Target: V3(0, 0, 0), Orthographic: true},
		{Position: V3(0, 5, 0), Target: V3(0, 0, 0), Up: V3(0, 1, 0)},
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("cam %+v panic: %v", cam, r)
				}
			}()
			img := NewRenderer3D().RenderImage(32, 32, sc, cam)
			_ = img
			r := cam.Ray(16, 16, 32, 32)
			sc.Pick(r)
			cam.Project(V3(0, 0, 0), 32, 32)
		}()
	}
}

func TestAuditLoadSTLOBJ(t *testing.T) {
	// binary STL count huge with matching length impossible; check count*50 overflow on 32-bit is not relevant
	var buf bytes.Buffer
	buf.Write(make([]byte, 80))
	binary.Write(&buf, binary.LittleEndian, uint32(1))
	buf.Write(make([]byte, 50))
	m, err := LoadSTL(&buf)
	t.Logf("zero triangle STL: %v %v", len(m.Positions), err)
	for _, s := range []string{
		"solid\nfacet normal 0 0 1\nvertex 0 0 0\nvertex 1 0 0\nvertex 0 1 0\nendfacet\n",
		"solid\nfacet normal\nvertex 0 0\n",
		"solid\nvertex 1 2 3\nvertex 1 2 3\nvertex 1 2 3\nendfacet\nendfacet\n",
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("STL %q panic: %v", s, r)
				}
			}()
			m, err := LoadSTL(strings.NewReader(s))
			if m != nil {
				t.Logf("STL %q: %d pos %v", s, len(m.Positions), err)
			}
		}()
	}
	for _, s := range []string{
		"v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n",
		"v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1/1 2 3\n",
		"v 0 0 0\nv 1 0 0\nv 0 1 0\nvt 0\nf 1/1/ 2 3\n",
		"v 0 0 0\nv 1 0 0\nv 0 1 0\nf -1 -2 -3\n",
		"v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 -4\n",
		"v 0 0 0 1 1 1\nv 1 0 0\nv 0 1 0\nf 1 2 3\n",
		"v nan nan nan\nv 1 0 0\nv 0 1 0\nf 1 2 3\n",
		"v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1//1 2 3\n",
		"v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1/2/3/4 2 3\n",
		"v 0 0 0\nv 1 0 0\nv 0 1 0\nf 9223372036854775807 2 3\n",
		"v 0 0 0\nv 1 0 0\nv 0 1 0\nf -9223372036854775808 2 3\n",
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("OBJ %q panic: %v", s, r)
				}
			}()
			m, err := LoadOBJ(strings.NewReader(s))
			t.Logf("OBJ %q: %d pos %v", s, len(m.Positions), err)
		}()
	}
}
