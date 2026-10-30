package ui

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

func nearV(a, b Vec3, eps float64) bool { return a.Distance(b) <= eps }

func TestMat4(t *testing.T) {
	m := Translate4(V3(1, 2, 3)).Mul(RotateY4(0.7)).Mul(Scale4(V3(2, 3, 4)))
	inv, ok := m.Inverse()
	if !ok {
		t.Fatal("not invertible")
	}
	for _, p := range []Vec3{{}, {1, -2, 5}, {10, 3, -7}} {
		if q := inv.MulPoint(m.MulPoint(p)); !nearV(q, p, 1e-9) {
			t.Fatalf("inverse of %v gave %v", p, q)
		}
	}
	// A turn around y by a quarter takes +x to -z
	if q := RotateY4(math.Pi / 2).MulPoint(V3(1, 0, 0)); !nearV(q, V3(0, 0, -1), 1e-12) {
		t.Fatalf("RotateY4: %v", q)
	}
	if q := Rotate4(V3(0, 1, 0), math.Pi/2).MulPoint(V3(1, 0, 0)); !nearV(q, V3(0, 0, -1), 1e-12) {
		t.Fatalf("Rotate4: %v", q)
	}
	// The camera looks down -z of its own space
	v := LookAt4(V3(0, 0, 5), V3(0, 0, 0), V3(0, 1, 0))
	if q := v.MulPoint(V3(0, 0, 0)); !nearV(q, V3(0, 0, -5), 1e-12) {
		t.Fatalf("LookAt4: %v", q)
	}
	// The near and the far planes go to -1 and 1
	p := Perspective4(math.Pi/2, 1, 1, 10)
	if q := p.MulPoint(V3(0, 0, -1)); !near(q.Z, -1, 1e-12) {
		t.Fatalf("near plane at %v", q.Z)
	}
	if q := p.MulPoint(V3(0, 0, -10)); !near(q.Z, 1, 1e-12) {
		t.Fatalf("far plane at %v", q.Z)
	}
	q := QuatAxisAngle(V3(0, 0, 1), math.Pi/2)
	if r := q.Rotate(V3(1, 0, 0)); !nearV(r, V3(0, 1, 0), 1e-12) {
		t.Fatalf("Quat.Rotate: %v", r)
	}
	if r := q.Mat4().MulPoint(V3(1, 0, 0)); !nearV(r, V3(0, 1, 0), 1e-12) {
		t.Fatalf("Quat.Mat4: %v", r)
	}
	half := IdentityQuat().Slerp(q, 0.5)
	if r := half.Rotate(V3(1, 0, 0)); !nearV(r, V3(math.Sqrt2/2, math.Sqrt2/2, 0), 1e-9) {
		t.Fatalf("Slerp: %v", r)
	}
}

func TestRayIntersect(t *testing.T) {
	r := Ray3{Origin: V3(0.2, 0.2, 5), Dir: V3(0, 0, -1)}
	d, u, v, ok := r.IntersectTriangle(V3(0, 0, 0), V3(1, 0, 0), V3(0, 1, 0))
	if !ok || !near(d, 5, 1e-12) || !near(u, 0.2, 1e-12) || !near(v, 0.2, 1e-12) {
		t.Fatalf("hit %v %v %v %v", d, u, v, ok)
	}
	if _, _, _, ok := (Ray3{Origin: V3(2, 2, 5), Dir: V3(0, 0, -1)}).IntersectTriangle(V3(0, 0, 0), V3(1, 0, 0), V3(0, 1, 0)); ok {
		t.Fatal("miss reported as a hit")
	}
	if d, ok := r.IntersectPlane(V3(0, 0, 1), V3(0, 0, 1)); !ok || !near(d, 4, 1e-12) {
		t.Fatalf("plane %v %v", d, ok)
	}
}

// The triangles of the shapes face outwards and their normals agree
func TestMeshShapesOutward(t *testing.T) {
	for name, m := range map[string]*Mesh{
		"box":      NewBoxMesh(1, 2, 3),
		"sphere":   NewSphereMesh(1, 16, 8),
		"cylinder": NewCylinderMesh(1, 0.5, 2, 16, true),
		"cone":     NewConeMesh(1, 2, 16),
		"torus":    NewTorusMesh(2, 0.5, 16, 8),
	} {
		lo, hi := m.Bounds()
		center := lo.Add(hi).Scale(0.5)
		for i := 0; i < m.TriangleCount(); i++ {
			a, b, c := m.triangle(i)
			pa, pb, pc := m.Positions[a], m.Positions[b], m.Positions[c]
			fn := pb.Sub(pa).Cross(pc.Sub(pa))
			if fn.Length() < 1e-12 {
				continue // the poles of a sphere
			}
			avg := m.Normals[a].Add(m.Normals[b]).Add(m.Normals[c])
			if fn.Dot(avg) <= 0 {
				t.Fatalf("%s: triangle %d winds against its normals", name, i)
			}
			if name != "torus" && fn.Dot(pa.Add(pb).Add(pc).Scale(1.0/3).Sub(center)) < -1e-9 {
				t.Fatalf("%s: triangle %d faces inwards", name, i)
			}
		}
	}
	s := NewSurfaceMesh(-1, 1, -1, 1, 4, 4, func(x, z float64) float64 { return 0 })
	for _, n := range s.Normals {
		if !nearV(n, V3(0, 1, 0), 1e-9) {
			t.Fatalf("flat surface normal %v", n)
		}
	}
	holes := NewSurfaceMesh(-1, 1, -1, 1, 4, 4, func(x, z float64) float64 {
		if x > 0 {
			return math.NaN()
		}
		return 0
	})
	if holes.TriangleCount() >= s.TriangleCount() {
		t.Fatal("NaN made no hole")
	}
}

func TestMeshCenterAppend(t *testing.T) {
	m := NewBoxMesh(2, 4, 8)
	m.Transform(Translate4(V3(10, 0, 0)))
	m.Center(1)
	lo, hi := m.Bounds()
	if !nearV(lo, V3(-0.125, -0.25, -0.5), 1e-9) || !nearV(hi, V3(0.125, 0.25, 0.5), 1e-9) {
		t.Fatalf("centered %v %v", lo, hi)
	}
	n := m.TriangleCount()
	m.Append(NewSphereMesh(1, 8, 4))
	if m.TriangleCount() != n+8*4*2 || len(m.Normals) != len(m.Positions) {
		t.Fatalf("appended: %d triangles, %d normals of %d", m.TriangleCount(), len(m.Normals), len(m.Positions))
	}
	// A box has 12 edges for its outline, 18 with the diagonals
	b := NewBoxMesh(1, 1, 1)
	if got := len(b.wireEdges(false)) / 2; got != 12 {
		t.Fatalf("box creases %d", got)
	}
	if got := len(b.wireEdges(true)) / 2; got != 18 {
		t.Fatalf("box edges %d", got)
	}
}

func TestLoadOBJ(t *testing.T) {
	obj := `# a square of two triangles and a quad
v 0 0 0 1 0 0
v 1 0 0 0 1 0
v 1 1 0 0 0 1
v 0 1 0 1 1 1
vt 0 0
vt 1 0
vt 1 1
vt 0 1
vn 0 0 1
f 1/1/1 2/2/1 3/3/1
f -4/1/1 -2/3/1 -1/4/1
f 1 2 3 4
`
	m, err := LoadOBJ(strings.NewReader(obj))
	if err != nil {
		t.Fatal(err)
	}
	if m.TriangleCount() != 4 {
		t.Fatalf("%d triangles", m.TriangleCount())
	}
	if len(m.Colors) != len(m.Positions) || m.Colors[0] != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("colors %v", m.Colors)
	}
	for _, n := range m.Normals {
		if !nearV(n, V3(0, 0, 1), 1e-9) {
			t.Fatalf("normal %v", n)
		}
	}
	// Written and read again
	var buf bytes.Buffer
	if err := m.WriteOBJ(&buf); err != nil {
		t.Fatal(err)
	}
	m2, err := LoadOBJ(&buf)
	if err != nil || m2.TriangleCount() != 4 {
		t.Fatalf("again: %v, %d", err, m2.TriangleCount())
	}
	for _, bad := range []string{"f 1 2 3", "v 1 2", "v 0 0 0\nv 1 0 0\nf 1 2", ""} {
		if _, err := LoadOBJ(strings.NewReader(bad)); err == nil {
			t.Fatalf("%q: no error", bad)
		}
	}
}

func TestLoadSTL(t *testing.T) {
	ascii := `solid t
facet normal 0 0 1
 outer loop
  vertex 0 0 0
  vertex 1 0 0
  vertex 0 1 0
 endloop
endfacet
endsolid t`
	m, err := LoadSTL(strings.NewReader(ascii))
	if err != nil || m.TriangleCount() != 1 {
		t.Fatalf("text: %v", err)
	}
	// The same triangle in binary
	var buf bytes.Buffer
	buf.Write(make([]byte, 80))
	buf.Write([]byte{1, 0, 0, 0})
	f := func(v float32) {
		b := math.Float32bits(v)
		buf.Write([]byte{byte(b), byte(b >> 8), byte(b >> 16), byte(b >> 24)})
	}
	for _, v := range []float32{0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0} {
		f(v)
	}
	buf.Write([]byte{0, 0})
	m, err = LoadSTL(&buf)
	if err != nil || m.TriangleCount() != 1 || !nearV(m.Normals[0], V3(0, 0, 1), 1e-9) {
		t.Fatalf("binary: %v %+v", err, m)
	}
	if _, err := LoadSTL(strings.NewReader("hello")); err == nil {
		t.Fatal("garbage read")
	}
}

// testScene3D is a red box in front of a blue one, seen from +z
func testScene3D() (*Scene3D, Camera) {
	s := NewScene3D()
	s.Background = color.RGBA{0, 0, 0, 255}
	near := s.Add(NewBoxMesh(1, 1, 1), Material{Color: color.RGBA{255, 0, 0, 255}, Unlit: true})
	near.Name = "near"
	far := s.Add(NewBoxMesh(4, 4, 1), Material{Color: color.RGBA{0, 0, 255, 255}, Unlit: true})
	far.Name = "far"
	far.Transform = Translate4(V3(0, 0, -3))
	return s, NewCamera(V3(0, 0, 6), V3(0, 0, 0))
}

func TestRender3DDepth(t *testing.T) {
	s, cam := testScene3D()
	for _, aa := range []int{1, 2} {
		r := NewRenderer3D()
		r.Antialias = aa
		img := r.RenderImage(100, 100, s, cam)
		if c := img.RGBAAt(50, 50); c != (color.RGBA{255, 0, 0, 255}) {
			t.Fatalf("aa %d: the near box at the center is %v", aa, c)
		}
		if c := img.RGBAAt(50, 30); c != (color.RGBA{0, 0, 255, 255}) {
			t.Fatalf("aa %d: the far box around it is %v", aa, c)
		}
		if c := img.RGBAAt(2, 2); c != (color.RGBA{0, 0, 0, 255}) {
			t.Fatalf("aa %d: the background %v", aa, c)
		}
		if st := r.Stats(); st.Triangles <= 0 || st.Triangles > 24 {
			t.Fatalf("aa %d: %d triangles drawn, the back sides left out", aa, st.Triangles)
		}
	}
	// The same from behind: the blue box hides the red one
	cam.Position = V3(0, 0, -9)
	img := NewRenderer3D().RenderImage(100, 100, s, cam)
	if c := img.RGBAAt(50, 50); c != (color.RGBA{0, 0, 255, 255}) {
		t.Fatalf("from behind %v", c)
	}
}

func TestRender3DOrthoAndInside(t *testing.T) {
	s, cam := testScene3D()
	cam.Orthographic = true
	img := NewRenderer3D().RenderImage(100, 100, s, cam)
	if c := img.RGBAAt(50, 50); c != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("orthographic center %v", c)
	}
	// The camera inside the box: its triangles cut by the near plane,
	// nothing breaks; the inside of a box is its back sides
	s.Objects[0].Material.DoubleSided = true
	cam = NewCamera(V3(0, 0, 0.2), V3(0, 0, -1))
	img = NewRenderer3D().RenderImage(60, 60, s, cam)
	if c := img.RGBAAt(30, 30); c != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("inside the double sided box %v", c)
	}
}

func TestRender3DTranslucentAndLight(t *testing.T) {
	s := NewScene3D()
	s.Background = color.RGBA{0, 0, 0, 255}
	s.Add(NewBoxMesh(1, 1, 1), Material{Color: color.RGBA{255, 255, 255, 128}, Unlit: true})
	cam := NewCamera(V3(0, 0, 5), V3(0, 0, 0))
	img := NewRenderer3D().RenderImage(50, 50, s, cam)
	if c := img.RGBAAt(25, 25); !near(float64(c.R), 128, 3) {
		t.Fatalf("half translucent white over black %v", c)
	}
	// Lit from the front, the side facing the light is brighter than the
	// one facing away
	s = NewScene3D()
	s.Background = color.RGBA{0, 0, 0, 255}
	s.Lights = []Light{{Direction: V3(-1, 0, 0)}} // from +x
	s.Add(NewSphereMesh(1, 32, 16), Material{Color: color.RGBA{255, 255, 255, 255}})
	img = NewRenderer3D().RenderImage(100, 100, s, cam)
	right, left := img.RGBAAt(68, 50).R, img.RGBAAt(32, 50).R
	if right < 150 || left > 20 {
		t.Fatalf("lit side %d, dark side %d", right, left)
	}
}

func TestRender3DLinesPointsTexture(t *testing.T) {
	s := NewScene3D()
	s.Background = color.RGBA{0, 0, 0, 255}
	s.AddLine(V3(-1, 0, 0), V3(1, 0, 0), color.RGBA{0, 255, 0, 255}, 3)
	s.AddPoint(V3(0, 0.5, 0), color.RGBA{255, 255, 0, 255}, 8)
	cam := NewCamera(V3(0, 0, 3), V3(0, 0, 0))
	img := NewRenderer3D().RenderImage(100, 100, s, cam)
	if c := img.RGBAAt(50, 50); c.G < 200 {
		t.Fatalf("line %v", c)
	}
	_, py, _, _ := cam.Project(V3(0, 0.5, 0), 100, 100)
	if c := img.RGBAAt(50, int(py)); c.R < 200 || c.G < 200 {
		t.Fatalf("point at %v: %v", py, c)
	}
	// A checker texture on a quad facing the camera
	tex := image.NewRGBA(image.Rect(0, 0, 2, 2))
	tex.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	tex.SetRGBA(1, 0, color.RGBA{0, 255, 0, 255})
	tex.SetRGBA(0, 1, color.RGBA{0, 0, 255, 255})
	tex.SetRGBA(1, 1, color.RGBA{255, 255, 255, 255})
	quad := &Mesh{
		Positions: []Vec3{{-1, -1, 0}, {1, -1, 0}, {1, 1, 0}, {-1, 1, 0}},
		UVs:       []PointF{{0, 0}, {1, 0}, {1, 1}, {0, 1}},
		Indices:   []int{0, 1, 2, 0, 2, 3},
	}
	s = NewScene3D()
	s.Add(quad, Material{Texture: tex, Unlit: true, Color: color.RGBA{255, 255, 255, 255}})
	cam = NewCamera(V3(0, 0, 2.2), V3(0, 0, 0))
	cam.Orthographic, cam.OrthoHeight = true, 2
	img = NewRenderer3D().RenderImage(100, 100, s, cam)
	// V goes up: the top row of the picture at the top
	if c := img.RGBAAt(25, 25); c.R < 200 || c.G > 60 {
		t.Fatalf("top left texel %v", c)
	}
	if c := img.RGBAAt(75, 75); c.R < 200 || c.G < 200 || c.B < 200 {
		t.Fatalf("bottom right texel %v", c)
	}
}

func TestPickAndProject(t *testing.T) {
	s, cam := testScene3D()
	ray := cam.Ray(50, 50, 100, 100)
	hit, ok := s.Pick(ray)
	if !ok || hit.Object.Name != "near" || !near(hit.Point.Z, 0.5, 1e-9) || !nearV(hit.Normal, V3(0, 0, 1), 1e-9) {
		t.Fatalf("pick %+v %v", hit, ok)
	}
	hit, ok = s.Pick(cam.Ray(50, 35, 100, 100))
	if !ok || hit.Object.Name != "far" {
		t.Fatalf("pick around %+v %v", hit, ok)
	}
	if _, ok := s.Pick(cam.Ray(1, 1, 100, 100)); ok {
		t.Fatal("picked nothing")
	}
	// Project and Ray agree
	for _, ortho := range []bool{false, true} {
		cam.Orthographic = ortho
		p := V3(0.3, -0.2, 0.5)
		x, y, _, ok := cam.Project(p, 200, 100)
		if !ok {
			t.Fatal("not projected")
		}
		r := cam.Ray(x, y, 200, 100)
		// The point is on the ray
		d := p.Sub(r.Origin)
		if off := d.Sub(r.Dir.Scale(d.Dot(r.Dir))).Length(); off > 1e-9 {
			t.Fatalf("ortho %v: the ray misses the point by %v", ortho, off)
		}
	}
}

func BenchmarkRender3DSphere(b *testing.B) {
	s := NewScene3D()
	s.Add(NewSphereMesh(1, 128, 64), Material{Color: color.RGBA{60, 140, 230, 255}, Specular: 0.5})
	cam := NewCamera(V3(0, 1, 3), V3(0, 0, 0))
	cnv, _ := NewImageCanvas(800, 600, 1)
	r := NewRenderer3D()
	r.Antialias = 1
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Render(cnv, 0, 0, 800, 600, s, cam)
	}
	b.ReportMetric(float64(r.Stats().Triangles), "tris")
}

// A shiny surface filling a big view, smoothed: the cost of the pixels
func BenchmarkRender3DFill(b *testing.B) {
	s := NewScene3D()
	s.Add(NewTorusMesh(1, 0.4, 64, 32), Material{Color: color.RGBA{230, 120, 40, 255}, Specular: 0.5, Shininess: 48})
	cam := NewCamera(V3(0, 1.2, 1.8), V3(0, 0, 0))
	cnv, _ := NewImageCanvas(800, 700, 1)
	r := NewRenderer3D()
	r.Antialias = 2
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Render(cnv, 0, 0, 800, 700, s, cam)
	}
}
