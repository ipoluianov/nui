package ui

import (
	"hash/fnv"
	"image"
	"image/color"
	"math"
	"reflect"
	"unsafe"
)

// Material is how the triangles of an object look. The zero value is a
// light gray surface lit by the lights of the scene.
type Material struct {
	// Color is the color of the surface, multiplied by the colors of the
	// corners of the mesh and by the texture. Zero is light gray; an
	// alpha below 255 makes the object translucent.
	Color color.RGBA
	// Texture is the picture on the surface, placed by the UVs of the mesh
	// (repeated past 0..1)
	Texture image.Image
	// Specular is how bright the highlights are, 0..1; Shininess how small
	// they are (0 is 32; 8 - a wide dull shine, 128 - a sharp one)
	Specular, Shininess float64
	// Unlit shows the colors as they are, without light and shade: a
	// picture, a colored map, lines
	Unlit bool
	// Flat shades every triangle in one tone: a faceted look
	Flat bool
	// DoubleSided draws the back sides of the triangles too: an open
	// surface, a sheet seen from below
	DoubleSided bool
	// Wireframe draws the edges of the triangles in the Color instead of
	// filling them
	Wireframe bool
	// EdgeColor, if not transparent, draws the edges over the filled
	// triangles, EdgeWidth logical pixels wide (0 is 1)
	EdgeColor color.RGBA
	EdgeWidth float64
}

// Object3D is a mesh placed in a scene with a material
type Object3D struct {
	Name     string
	Mesh     *Mesh
	Material Material
	// Transform places the mesh in the scene: moves, turns and stretches
	// it. The zero value is the identity.
	Transform Mat4
	Hidden    bool
	// Data is the application's own, for the objects found by Pick
	Data any
}

// model returns the transform of the object
func (o *Object3D) model() Mat4 {
	if o.Transform == (Mat4{}) {
		return Identity4()
	}
	return o.Transform
}

// LightKind is the kind of a light of a scene
type LightKind int

const (
	// LightDirectional shines from far away in one direction, as the sun
	LightDirectional LightKind = iota
	// LightPoint shines from a point in all directions, as a lamp
	LightPoint
)

// Light is a light of a scene
type Light struct {
	Kind LightKind
	// Direction is where a directional light shines to: (0, -1, 0) from
	// above. In the coordinates of the scene, or of the camera when
	// FromCamera is set (x right, y up, z towards the viewer).
	Direction Vec3
	// Position is where a point light is
	Position Vec3
	// Color of the light; zero is white
	Color color.RGBA
	// Intensity 0..1 and above; zero is 1
	Intensity float64
	// FromCamera fixes the light to the camera: it turns with the view, as
	// a lamp on the viewer's head
	FromCamera bool
}

// Line3D is a line in a scene, of the width in logical pixels (0 is 1),
// hidden by what is in front of it
type Line3D struct {
	A, B  Vec3
	Color color.RGBA
	Width float64
}

// Point3D is a dot in a scene, its size in logical pixels across (0 is 3):
// a point of a cloud, a marker
type Point3D struct {
	P     Vec3
	Color color.RGBA
	Size  float64
}

// Label3D is a text at a point of a scene, drawn over it centered on the
// point
type Label3D struct {
	P     Vec3
	Text  string
	Color color.RGBA
}

// Scene3D is what a 3D view shows: objects, lines, points, labels and the
// lights. Without lights, the scene is lit by a light from the camera's
// upper right and a soft light from all around.
//
//	scene := ui.NewScene3D()
//	scene.Add(ui.NewTorusMesh(1, 0.35, 48, 24), ui.Material{Color: orange, Specular: 0.5})
//	scene.AddGrid(4, 0.5, gray)
//	view := ui.NewView3D(scene)
type Scene3D struct {
	Objects []*Object3D
	Lines   []Line3D
	Points  []Point3D
	Labels  []Label3D
	Lights  []Light
	// Ambient is the light from all around, 0..1, on the sides away from
	// the lights; with no Lights, zero is 0.25
	Ambient float64
	// Background is the color behind the scene; transparent shows what was
	// drawn under it
	Background color.RGBA
	// ClipPlanes cut the objects, a section view: of each object, only
	// what is on the side its normal points to is drawn and picked (up to
	// six planes). The lines, the points and the labels of the scene are
	// not cut.
	ClipPlanes []Plane3D
	// SectionColor colors the inside of the objects seen through the cuts;
	// zero is the color of each object, darker
	SectionColor color.RGBA
}

// Plane3D is the plane through Point square to Normal
type Plane3D struct {
	Point, Normal Vec3
}

// Distance returns how far the point is from the plane, positive on the
// side the normal points to
func (p Plane3D) Distance(q Vec3) float64 {
	return q.Sub(p.Point).Dot(p.Normal.Normalize())
}

// clipped reports whether the point of the scene is cut away
func (s *Scene3D) clipped(p Vec3) bool {
	for i, pl := range s.ClipPlanes {
		if i == maxClipPlanes {
			break
		}
		if pl.Normal != (Vec3{}) && pl.Distance(p) < 0 {
			return true
		}
	}
	return false
}

// NewScene3D returns an empty scene
func NewScene3D() *Scene3D {
	return &Scene3D{}
}

// Add adds the mesh with the material and returns the object, to move or
// to name it
func (s *Scene3D) Add(mesh *Mesh, mat Material) *Object3D {
	o := &Object3D{Mesh: mesh, Material: mat, Transform: Identity4()}
	s.Objects = append(s.Objects, o)
	return o
}

// Remove removes the object from the scene
func (s *Scene3D) Remove(o *Object3D) {
	for i, x := range s.Objects {
		if x == o {
			s.Objects = append(s.Objects[:i], s.Objects[i+1:]...)
			return
		}
	}
}

// AddLine adds a line
func (s *Scene3D) AddLine(a, b Vec3, col color.RGBA, width float64) {
	s.Lines = append(s.Lines, Line3D{A: a, B: b, Color: col, Width: width})
}

// AddPolyline adds the lines through the points: a path, a trajectory
func (s *Scene3D) AddPolyline(pts []Vec3, col color.RGBA, width float64) {
	for i := 1; i < len(pts); i++ {
		s.AddLine(pts[i-1], pts[i], col, width)
	}
}

// AddPoint adds a dot
func (s *Scene3D) AddPoint(p Vec3, col color.RGBA, size float64) {
	s.Points = append(s.Points, Point3D{P: p, Color: col, Size: size})
}

// AddLabel adds a text at the point
func (s *Scene3D) AddLabel(p Vec3, text string, col color.RGBA) {
	s.Labels = append(s.Labels, Label3D{P: p, Text: text, Color: col})
}

// AddAxes adds the axes from 0, 0, 0 of the length: x red, y green, z
// blue, with their names at the ends
func (s *Scene3D) AddAxes(length float64) {
	axes := [3]struct {
		d    Vec3
		c    color.RGBA
		name string
	}{
		{Vec3{1, 0, 0}, color.RGBA{220, 50, 47, 255}, "X"},
		{Vec3{0, 1, 0}, color.RGBA{60, 170, 60, 255}, "Y"},
		{Vec3{0, 0, 1}, color.RGBA{38, 110, 220, 255}, "Z"},
	}
	for _, a := range axes {
		end := a.d.Scale(length)
		s.AddLine(Vec3{}, end, a.c, 2)
		s.AddLabel(a.d.Scale(length*1.08), a.name, a.c)
	}
}

// AddGrid adds a grid of lines on the floor (y = 0) around 0, 0, 0: size
// across, a line every step
func (s *Scene3D) AddGrid(size, step float64, col color.RGBA) {
	if step <= 0 || size <= 0 {
		return
	}
	n := int(math.Floor(size / 2 / step))
	h := float64(n) * step
	for i := -n; i <= n; i++ {
		v := float64(i) * step
		s.AddLine(Vec3{v, 0, -h}, Vec3{v, 0, h}, col, 1)
		s.AddLine(Vec3{-h, 0, v}, Vec3{h, 0, v}, col, 1)
	}
}

// Clear removes everything from the scene but the lights
func (s *Scene3D) Clear() {
	s.Objects, s.Lines, s.Points, s.Labels = nil, nil, nil, nil
}

// Bounds returns the box that holds what the scene shows; ok is false for
// an empty scene
func (s *Scene3D) Bounds() (lo, hi Vec3, ok bool) {
	return s.bounds(true)
}

// fitBounds returns the bounds of the objects and the points, without the
// lines (a grid, the axes) when there are any: what a view is fitted to
func (s *Scene3D) fitBounds() (lo, hi Vec3, ok bool) {
	if lo, hi, ok = s.bounds(false); ok {
		return lo, hi, ok
	}
	return s.bounds(true)
}

func (s *Scene3D) bounds(lines bool) (lo, hi Vec3, ok bool) {
	add := func(p Vec3) {
		if !ok {
			lo, hi, ok = p, p, true
			return
		}
		lo, hi = lo.Min(p), hi.Max(p)
	}
	for _, o := range s.Objects {
		if o.Hidden || o.Mesh == nil || len(o.Mesh.Positions) == 0 {
			continue
		}
		mlo, mhi := o.Mesh.Bounds()
		m := o.model()
		for i := 0; i < 8; i++ {
			c := Vec3{mlo.X, mlo.Y, mlo.Z}
			if i&1 != 0 {
				c.X = mhi.X
			}
			if i&2 != 0 {
				c.Y = mhi.Y
			}
			if i&4 != 0 {
				c.Z = mhi.Z
			}
			add(m.MulPoint(c))
		}
	}
	if lines {
		for _, l := range s.Lines {
			add(l.A)
			add(l.B)
		}
	}
	for _, p := range s.Points {
		add(p.P)
	}
	return lo, hi, ok
}

// Hit3D is what Pick found under a ray
type Hit3D struct {
	Object *Object3D
	// Triangle is the number of the triangle of the object's mesh
	Triangle int
	// Point is where the ray hit it, Normal the side it faces, in the
	// coordinates of the scene
	Point, Normal Vec3
	Distance      float64
}

// Pick returns the nearest triangle of a visible object the ray hits:
// what is under the mouse (Camera.Ray, View3D.PickAt)
func (s *Scene3D) Pick(ray Ray3) (hit Hit3D, ok bool) {
	ray.Dir = ray.Dir.Normalize()
	best := math.Inf(1)
	for _, o := range s.Objects {
		if o.Hidden || o.Mesh == nil {
			continue
		}
		m := o.model()
		inv, invOK := m.Inverse()
		if !invOK {
			continue
		}
		// The ray in the object's own coordinates; the distances are
		// compared in the scene's
		local := Ray3{Origin: inv.MulPoint(ray.Origin), Dir: inv.MulDir(ray.Dir)}
		lo, hi := o.Mesh.Bounds()
		if _, in := local.intersectBox(lo, hi); !in {
			continue
		}
		mesh := o.Mesh
		for t := 0; t < mesh.TriangleCount(); t++ {
			a, b, c := mesh.triangle(t)
			if a >= len(mesh.Positions) || b >= len(mesh.Positions) || c >= len(mesh.Positions) {
				continue
			}
			pa, pb, pc := mesh.Positions[a], mesh.Positions[b], mesh.Positions[c]
			lt, _, _, hitOK := local.IntersectTriangle(pa, pb, pc)
			if !hitOK {
				continue
			}
			p := m.MulPoint(local.At(lt))
			d := p.Sub(ray.Origin).Dot(ray.Dir)
			if d <= 0 || d >= best || s.clipped(p) {
				continue
			}
			n := m.normalMatrix().MulDir(pb.Sub(pa).Cross(pc.Sub(pa))).Normalize()
			if n.Dot(ray.Dir) > 0 {
				n = n.Neg()
			}
			best = d
			hit = Hit3D{Object: o, Triangle: t, Point: p, Normal: n, Distance: d}
			ok = true
		}
	}
	return hit, ok
}

// Camera is where a 3D scene is seen from
type Camera struct {
	Position, Target Vec3
	// Up is the direction that is up on the screen; zero is +y
	Up Vec3
	// FOV is the vertical field of view in radians; zero is 45 degrees
	FOV float64
	// Orthographic projects without perspective: the far is as large as
	// the near, as in a drawing. OrthoHeight is how tall the view is in
	// the units of the scene; zero shows at the target what the
	// perspective camera would.
	Orthographic bool
	OrthoHeight  float64
	// Near and Far are the nearest and the farthest distances seen; zero
	// takes them from the scene
	Near, Far float64
}

// NewCamera returns a perspective camera at the position looking at the
// target
func NewCamera(position, target Vec3) Camera {
	return Camera{Position: position, Target: target, Up: Vec3{0, 1, 0}, FOV: math.Pi / 4}
}

func (c Camera) up() Vec3 {
	if c.Up == (Vec3{}) {
		return Vec3{0, 1, 0}
	}
	return c.Up
}

func (c Camera) fov() float64 {
	if c.FOV <= 0 || c.FOV >= math.Pi {
		return math.Pi / 4
	}
	return c.FOV
}

// ViewMatrix returns the transform from the scene to the camera
func (c Camera) ViewMatrix() Mat4 {
	return LookAt4(c.Position, c.Target, c.up())
}

// orthoHeight returns the height of the orthographic view
func (c Camera) orthoHeight() float64 {
	if c.OrthoHeight > 0 {
		return c.OrthoHeight
	}
	d := c.Position.Distance(c.Target)
	if d == 0 {
		d = 1
	}
	return 2 * d * math.Tan(c.fov()/2)
}

// ProjectionMatrix returns the projection of the camera onto a picture of
// the width to height aspect, between the near and the far distances
func (c Camera) ProjectionMatrix(aspect, near, far float64) Mat4 {
	if c.Orthographic {
		h := c.orthoHeight() / 2
		return Orthographic4(-h*aspect, h*aspect, -h, h, near, far)
	}
	return Perspective4(c.fov(), aspect, near, far)
}

// nearFar returns the near and the far distances of the camera: its own,
// or those that hold the bounds of the scene
func (c Camera) nearFar(s *Scene3D) (near, far float64) {
	near, far = c.Near, c.Far
	if near > 0 && far > near {
		return near, far
	}
	dist := math.Max(c.Position.Distance(c.Target), 1e-6)
	radius := dist
	center := c.Target
	if s != nil {
		if lo, hi, ok := s.Bounds(); ok {
			center = lo.Add(hi).Scale(0.5)
			radius = math.Max(hi.Sub(lo).Length()/2, 1e-6)
		}
	}
	forward := c.Target.Sub(c.Position).Normalize()
	d := center.Sub(c.Position).Dot(forward)
	autoFar := d + radius*1.5
	autoNear := d - radius*1.5
	autoFar = math.Max(autoFar, dist*2)
	if c.Orthographic {
		autoNear = math.Min(autoNear, -radius) // what is behind the camera shows too
	} else {
		autoNear = math.Max(autoNear, autoFar*1e-4)
	}
	if near <= 0 {
		near = autoNear
	}
	if far <= 0 || far <= near {
		far = math.Max(autoFar, near+radius)
	}
	return near, far
}

// Project returns where the point of the scene is in a picture of the
// width and height (0, 0 - its top left corner) and how far it is from
// the camera; ok is false for a point behind the camera
func (c Camera) Project(p Vec3, width, height float64) (x, y, depth float64, ok bool) {
	v := c.ViewMatrix()
	vp := v.MulPoint(p)
	depth = -vp.Z
	if !c.Orthographic && depth <= 1e-9 {
		return 0, 0, depth, false
	}
	proj := c.ProjectionMatrix(width/math.Max(height, 1), 1, 2)
	px, py, _, w := proj.mul4(vp)
	if w == 0 {
		return 0, 0, depth, false
	}
	x = (px/w + 1) / 2 * width
	y = (1 - py/w) / 2 * height
	return x, y, depth, true
}

// Ray returns the ray from the camera through the point (x, y) of a
// picture of the width and height: to find what is under the mouse
func (c Camera) Ray(x, y, width, height float64) Ray3 {
	aspect := width / math.Max(height, 1)
	nx := 2*x/width - 1
	ny := 1 - 2*y/height
	inv, _ := c.ViewMatrix().Inverse()
	if c.Orthographic {
		h := c.orthoHeight() / 2
		origin := inv.MulPoint(Vec3{nx * h * aspect, ny * h, 0})
		return Ray3{Origin: origin, Dir: c.Target.Sub(c.Position).Normalize()}
	}
	t := math.Tan(c.fov() / 2)
	dir := inv.MulDir(Vec3{nx * t * aspect, ny * t, -1}).Normalize()
	return Ray3{Origin: c.Position, Dir: dir}
}

// signature returns a hash of what the scene shows, to know whether a
// picture of it is still right; ok is false when it cannot be told (a
// texture that is not a pointer)
func (s *Scene3D) signature() (sig uint64, ok bool) {
	h := fnv.New64a()
	var buf [8]byte
	u := func(v uint64) {
		for i := range buf {
			buf[i] = byte(v >> (8 * i))
		}
		h.Write(buf[:])
	}
	f := func(v float64) { u(math.Float64bits(v)) }
	vec := func(v Vec3) { f(v.X); f(v.Y); f(v.Z) }
	col := func(c color.RGBA) { u(uint64(c.R) | uint64(c.G)<<8 | uint64(c.B)<<16 | uint64(c.A)<<24) }
	ptr := func(p unsafe.Pointer) { u(uint64(uintptr(p))) }
	b := func(v bool) {
		if v {
			u(1)
		} else {
			u(0)
		}
	}
	u(uint64(len(s.Objects)))
	for _, o := range s.Objects {
		ptr(unsafe.Pointer(o))
		b(o.Hidden)
		for _, v := range o.Transform {
			f(v)
		}
		m := o.Material
		col(m.Color)
		col(m.EdgeColor)
		f(m.Specular)
		f(m.Shininess)
		f(m.EdgeWidth)
		b(m.Unlit)
		b(m.Flat)
		b(m.DoubleSided)
		b(m.Wireframe)
		if m.Texture != nil {
			rv := reflect.ValueOf(m.Texture)
			if rv.Kind() != reflect.Pointer {
				return 0, false
			}
			u(uint64(rv.Pointer()))
		}
		if me := o.Mesh; me != nil {
			ptr(unsafe.Pointer(me))
			u(me.version)
			u(uint64(len(me.Positions)))
			u(uint64(len(me.Indices)))
			u(uint64(len(me.Normals)))
			u(uint64(len(me.Colors)))
			u(uint64(len(me.UVs)))
			ptr(unsafe.Pointer(unsafe.SliceData(me.Positions)))
			ptr(unsafe.Pointer(unsafe.SliceData(me.Indices)))
			ptr(unsafe.Pointer(unsafe.SliceData(me.Normals)))
			ptr(unsafe.Pointer(unsafe.SliceData(me.Colors)))
			ptr(unsafe.Pointer(unsafe.SliceData(me.UVs)))
		}
	}
	u(uint64(len(s.Lines)))
	for _, l := range s.Lines {
		vec(l.A)
		vec(l.B)
		col(l.Color)
		f(l.Width)
	}
	u(uint64(len(s.Points)))
	for _, p := range s.Points {
		vec(p.P)
		col(p.Color)
		f(p.Size)
	}
	u(uint64(len(s.Labels)))
	for _, l := range s.Labels {
		vec(l.P)
		col(l.Color)
		h.Write([]byte(l.Text))
		u(0)
	}
	u(uint64(len(s.Lights)))
	for _, l := range s.Lights {
		u(uint64(l.Kind))
		vec(l.Direction)
		vec(l.Position)
		col(l.Color)
		f(l.Intensity)
		b(l.FromCamera)
	}
	f(s.Ambient)
	col(s.Background)
	u(uint64(len(s.ClipPlanes)))
	for _, p := range s.ClipPlanes {
		vec(p.Point)
		vec(p.Normal)
	}
	col(s.SectionColor)
	return h.Sum64(), true
}
