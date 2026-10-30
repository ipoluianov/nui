package ui

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Mesh is a 3D model made of triangles: the corners in Positions, three
// numbers of them in Indices per triangle, counterclockwise when seen
// from the outside (the side drawn; the back faces are left out unless
// the material is DoubleSided). Without Indices, every three Positions
// are a triangle.
//
// Normals, UVs and Colors are optional, one per position: the directions
// the surface faces (made by ComputeNormals when missing), the points of
// the material's texture (U to the right, V up, 0..1), the colors of the
// corners, mixed across the triangles.
type Mesh struct {
	Positions []Vec3
	Normals   []Vec3
	UVs       []PointF
	Colors    []color.RGBA
	Indices   []int

	// The edges of the wireframe and of the creases, made when first drawn
	edges     []int
	creases   []int
	edgesFrom int // len(Indices) they were made of
	// The bounds, made when first needed
	boundsOK bool
	lo, hi   Vec3
	// version counts the changes, for the views that keep their pictures
	version uint64
}

// Changed tells the mesh that its slices were changed in place: what it
// made of them (the edges, the bounds) is made again, and the views that
// show it draw it again
func (m *Mesh) Changed() {
	m.edges = nil
	m.boundsOK = false
	m.version++
}

// TriangleCount returns how many triangles the mesh has
func (m *Mesh) TriangleCount() int {
	if len(m.Indices) > 0 {
		return len(m.Indices) / 3
	}
	return len(m.Positions) / 3
}

// triangle returns the numbers of the corners of the triangle i
func (m *Mesh) triangle(i int) (int, int, int) {
	if len(m.Indices) > 0 {
		return m.Indices[i*3], m.Indices[i*3+1], m.Indices[i*3+2]
	}
	return i * 3, i*3 + 1, i*3 + 2
}

// Bounds returns the box that holds the mesh
func (m *Mesh) Bounds() (lo, hi Vec3) {
	if m.boundsOK {
		return m.lo, m.hi
	}
	if len(m.Positions) == 0 {
		return Vec3{}, Vec3{}
	}
	lo, hi = m.Positions[0], m.Positions[0]
	for _, p := range m.Positions[1:] {
		lo, hi = lo.Min(p), hi.Max(p)
	}
	m.lo, m.hi, m.boundsOK = lo, hi, true
	return lo, hi
}

// ComputeNormals makes the normals of the corners: the average of the
// normals of the triangles around each one, weighted by their areas, so a
// curved surface looks smooth. Corners not shared by the triangles keep
// the edges sharp: a box has its corners repeated, one per side.
func (m *Mesh) ComputeNormals() {
	m.Normals = make([]Vec3, len(m.Positions))
	for t := 0; t < m.TriangleCount(); t++ {
		a, b, c := m.triangle(t)
		if a >= len(m.Positions) || b >= len(m.Positions) || c >= len(m.Positions) {
			continue
		}
		n := m.Positions[b].Sub(m.Positions[a]).Cross(m.Positions[c].Sub(m.Positions[a]))
		m.Normals[a] = m.Normals[a].Add(n)
		m.Normals[b] = m.Normals[b].Add(n)
		m.Normals[c] = m.Normals[c].Add(n)
	}
	for i := range m.Normals {
		m.Normals[i] = m.Normals[i].Normalize()
	}
	m.version++
}

// Transform transforms the mesh in place: its positions and normals
func (m *Mesh) Transform(t Mat4) {
	nm := t.normalMatrix()
	for i, p := range m.Positions {
		m.Positions[i] = t.MulPoint(p)
	}
	for i, n := range m.Normals {
		m.Normals[i] = nm.MulDir(n).Normalize()
	}
	m.Changed()
}

// Center moves the mesh so the middle of its bounds is at 0, 0, 0 and
// stretches it so the largest side of the bounds is size: a model of any
// file fits the same view
func (m *Mesh) Center(size float64) {
	lo, hi := m.Bounds()
	ext := hi.Sub(lo)
	k := size / math.Max(ext.X, math.Max(ext.Y, ext.Z))
	if math.IsInf(k, 0) || math.IsNaN(k) {
		k = 1
	}
	mid := lo.Add(hi).Scale(0.5)
	m.Transform(Scale4(Vec3{k, k, k}).Mul(Translate4(mid.Neg())))
}

// Append adds the triangles of o to the mesh; the attributes one of them
// lacks are made up (normals, the white color, UVs of 0)
func (m *Mesh) Append(o *Mesh) {
	base := len(m.Positions)
	hasN, hasUV, hasC := len(m.Normals) > 0 || len(o.Normals) > 0, len(m.UVs) > 0 || len(o.UVs) > 0, len(m.Colors) > 0 || len(o.Colors) > 0
	if hasN && len(m.Normals) < base {
		if len(m.Normals) == 0 && base > 0 {
			m.ComputeNormals()
		}
	}
	if len(m.Indices) == 0 {
		for i := 0; i < base; i++ {
			m.Indices = append(m.Indices, i)
		}
	}
	m.Positions = append(m.Positions, o.Positions...)
	if hasN {
		on := o.Normals
		if len(on) < len(o.Positions) {
			oc := *o
			oc.ComputeNormals()
			on = oc.Normals
		}
		m.Normals = append(m.Normals, on...)
	}
	if hasUV {
		for len(m.UVs) < base {
			m.UVs = append(m.UVs, PointF{})
		}
		m.UVs = append(m.UVs, o.UVs...)
		for len(m.UVs) < len(m.Positions) {
			m.UVs = append(m.UVs, PointF{})
		}
	}
	if hasC {
		for len(m.Colors) < base {
			m.Colors = append(m.Colors, color.RGBA{255, 255, 255, 255})
		}
		m.Colors = append(m.Colors, o.Colors...)
		for len(m.Colors) < len(m.Positions) {
			m.Colors = append(m.Colors, color.RGBA{255, 255, 255, 255})
		}
	}
	for t := 0; t < o.TriangleCount(); t++ {
		a, b, c := o.triangle(t)
		m.Indices = append(m.Indices, base+a, base+b, base+c)
	}
	m.Changed()
}

// wireEdges returns the edges of the triangles, each once: pairs of
// numbers of the corners. Corners at the same place (the sides of a box)
// are taken as one, so an edge is not drawn twice. With all false, only
// the edges of the outline and of the creases (where the surface turns by
// more than 30 degrees) are returned: not the diagonals of the flat sides.
func (m *Mesh) wireEdges(all bool) []int {
	key := len(m.Indices) + len(m.Positions)<<32
	if m.edges != nil && m.edgesFrom == key {
		if all {
			return m.edges
		}
		return m.creases
	}
	type pair struct{ a, b Vec3 }
	type edge struct {
		a, b   int
		n      Vec3
		count  int
		crease bool
	}
	found := make(map[pair]int, m.TriangleCount()*2)
	var edges []edge
	creaseCos := math.Cos(30 * math.Pi / 180)
	for t := 0; t < m.TriangleCount(); t++ {
		i := [3]int{}
		i[0], i[1], i[2] = m.triangle(t)
		if i[0] >= len(m.Positions) || i[1] >= len(m.Positions) || i[2] >= len(m.Positions) {
			continue
		}
		p0, p1, p2 := m.Positions[i[0]], m.Positions[i[1]], m.Positions[i[2]]
		fn := p1.Sub(p0).Cross(p2.Sub(p0)).Normalize()
		for k := 0; k < 3; k++ {
			a, b := i[k], i[(k+1)%3]
			pa, pb := m.Positions[a], m.Positions[b]
			if pb.X < pa.X || (pb.X == pa.X && (pb.Y < pa.Y || (pb.Y == pa.Y && pb.Z < pa.Z))) {
				pa, pb = pb, pa
			}
			if e, ok := found[pair{pa, pb}]; ok {
				edges[e].count++
				if fn.Dot(edges[e].n) < creaseCos {
					edges[e].crease = true
				}
				continue
			}
			found[pair{pa, pb}] = len(edges)
			edges = append(edges, edge{a: a, b: b, n: fn, count: 1})
		}
	}
	m.edges, m.creases = make([]int, 0, len(edges)*2), nil
	for _, e := range edges {
		m.edges = append(m.edges, e.a, e.b)
		if e.count == 1 || e.crease {
			m.creases = append(m.creases, e.a, e.b)
		}
	}
	m.edgesFrom = key
	if all {
		return m.edges
	}
	return m.creases
}

// The shapes. All are centered at 0, 0, 0, with normals and UVs.

// NewBoxMesh returns a box of the sizes along x, y and z: six sides of
// four corners each, so its edges are sharp
func NewBoxMesh(sx, sy, sz float64) *Mesh {
	m := &Mesh{}
	h := Vec3{sx / 2, sy / 2, sz / 2}
	// Each side: its normal and the two directions along it
	sides := [6][3]Vec3{
		{{1, 0, 0}, {0, 0, -1}, {0, 1, 0}},
		{{-1, 0, 0}, {0, 0, 1}, {0, 1, 0}},
		{{0, 1, 0}, {1, 0, 0}, {0, 0, -1}},
		{{0, -1, 0}, {1, 0, 0}, {0, 0, 1}},
		{{0, 0, 1}, {1, 0, 0}, {0, 1, 0}},
		{{0, 0, -1}, {-1, 0, 0}, {0, 1, 0}},
	}
	for _, s := range sides {
		n, u, v := s[0], s[1], s[2]
		base := len(m.Positions)
		for _, c := range [4][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			p := n.Add(u.Scale(c[0])).Add(v.Scale(c[1])).Mul(h)
			m.Positions = append(m.Positions, p)
			m.Normals = append(m.Normals, n)
			m.UVs = append(m.UVs, PointF{(c[0] + 1) / 2, (c[1] + 1) / 2})
		}
		m.Indices = append(m.Indices, base, base+1, base+2, base, base+2, base+3)
	}
	return m
}

// NewSphereMesh returns a sphere of the radius made of segments around and
// rings from the top to the bottom (24 and 16 look round)
func NewSphereMesh(radius float64, segments, rings int) *Mesh {
	segments, rings = max(segments, 3), max(rings, 2)
	m := &Mesh{}
	for r := 0; r <= rings; r++ {
		phi := math.Pi * float64(r) / float64(rings)
		for s := 0; s <= segments; s++ {
			th := 2 * math.Pi * float64(s) / float64(segments)
			n := Vec3{math.Sin(phi) * math.Cos(th), math.Cos(phi), -math.Sin(phi) * math.Sin(th)}
			m.Positions = append(m.Positions, n.Scale(radius))
			m.Normals = append(m.Normals, n)
			m.UVs = append(m.UVs, PointF{float64(s) / float64(segments), 1 - float64(r)/float64(rings)})
		}
	}
	m.Indices = gridIndices(segments, rings, true)
	return m
}

// gridIndices returns the triangles of a grid of (cols+1) x (rows+1)
// corners, row by row; flip turns them the other way round
func gridIndices(cols, rows int, flip bool) []int {
	var idx []int
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			a := r*(cols+1) + c
			b, d, e := a+1, a+cols+1, a+cols+2
			if flip {
				idx = append(idx, a, d, b, b, d, e)
			} else {
				idx = append(idx, a, b, d, b, e, d)
			}
		}
	}
	return idx
}

// NewCylinderMesh returns a cylinder along y of the radii at the bottom
// and the top (a cone with one of them 0) and the height, closed by caps
// when caps is set
func NewCylinderMesh(bottomRadius, topRadius, height float64, segments int, caps bool) *Mesh {
	segments = max(segments, 3)
	m := &Mesh{}
	h := height / 2
	slope := (bottomRadius - topRadius) / height
	for i := 0; i <= 1; i++ {
		y, r := -h, bottomRadius
		if i == 1 {
			y, r = h, topRadius
		}
		for s := 0; s <= segments; s++ {
			th := 2 * math.Pi * float64(s) / float64(segments)
			cs, sn := math.Cos(th), -math.Sin(th)
			m.Positions = append(m.Positions, Vec3{r * cs, y, r * sn})
			m.Normals = append(m.Normals, Vec3{cs, slope, sn}.Normalize())
			m.UVs = append(m.UVs, PointF{float64(s) / float64(segments), float64(i)})
		}
	}
	m.Indices = gridIndices(segments, 1, false)
	if caps {
		for _, top := range []bool{false, true} {
			y, r, ny := -h, bottomRadius, -1.0
			if top {
				y, r, ny = h, topRadius, 1
			}
			if r <= 0 {
				continue
			}
			center := len(m.Positions)
			m.Positions = append(m.Positions, Vec3{0, y, 0})
			m.Normals = append(m.Normals, Vec3{0, ny, 0})
			m.UVs = append(m.UVs, PointF{0.5, 0.5})
			for s := 0; s <= segments; s++ {
				th := 2 * math.Pi * float64(s) / float64(segments)
				cs, sn := math.Cos(th), -math.Sin(th)
				m.Positions = append(m.Positions, Vec3{r * cs, y, r * sn})
				m.Normals = append(m.Normals, Vec3{0, ny, 0})
				m.UVs = append(m.UVs, PointF{0.5 + cs/2, 0.5 - sn/2})
			}
			for s := 0; s < segments; s++ {
				a, b := center+1+s, center+2+s
				if top {
					m.Indices = append(m.Indices, center, a, b)
				} else {
					m.Indices = append(m.Indices, center, b, a)
				}
			}
		}
	}
	return m
}

// NewConeMesh returns a cone along y with the tip up
func NewConeMesh(radius, height float64, segments int) *Mesh {
	return NewCylinderMesh(radius, 0, height, segments, true)
}

// NewTorusMesh returns a ring around y: the radius of the ring and of its
// tube, made of segments around the ring and sides around the tube
func NewTorusMesh(radius, tube float64, segments, sides int) *Mesh {
	segments, sides = max(segments, 3), max(sides, 3)
	m := &Mesh{}
	for s := 0; s <= sides; s++ {
		ph := 2 * math.Pi * float64(s) / float64(sides)
		for g := 0; g <= segments; g++ {
			th := 2 * math.Pi * float64(g) / float64(segments)
			dir := Vec3{math.Cos(th), 0, -math.Sin(th)}
			n := dir.Scale(math.Cos(ph)).Add(Vec3{0, math.Sin(ph), 0})
			m.Positions = append(m.Positions, dir.Scale(radius).Add(n.Scale(tube)))
			m.Normals = append(m.Normals, n)
			m.UVs = append(m.UVs, PointF{float64(g) / float64(segments), float64(s) / float64(sides)})
		}
	}
	m.Indices = gridIndices(segments, sides, false)
	return m
}

// NewPlaneMesh returns a flat rectangle on the floor (y = 0), facing up,
// of the sizes along x and z, split into nx x nz squares
func NewPlaneMesh(sx, sz float64, nx, nz int) *Mesh {
	return NewSurfaceMesh(-sx/2, sx/2, -sz/2, sz/2, nx, nz, func(x, z float64) float64 { return 0 })
}

// NewSurfaceMesh returns the surface y = f(x, z) over the rectangle from
// x0, z0 to x1, z1, sampled at nx x nz squares: the graph of a function of
// two variables, a terrain. Points where f is NaN or infinite leave holes.
func NewSurfaceMesh(x0, x1, z0, z1 float64, nx, nz int, f func(x, z float64) float64) *Mesh {
	nx, nz = max(nx, 1), max(nz, 1)
	m := &Mesh{}
	valid := make([]bool, 0, (nx+1)*(nz+1))
	for j := 0; j <= nz; j++ {
		z := z1 + (z0-z1)*float64(j)/float64(nz)
		for i := 0; i <= nx; i++ {
			x := x0 + (x1-x0)*float64(i)/float64(nx)
			y := f(x, z)
			ok := !math.IsNaN(y) && !math.IsInf(y, 0)
			if !ok {
				y = 0
			}
			valid = append(valid, ok)
			m.Positions = append(m.Positions, Vec3{x, y, z})
			m.UVs = append(m.UVs, PointF{float64(i) / float64(nx), float64(j) / float64(nz)})
		}
	}
	// The rows go to -z: the triangles face up
	m.Indices = gridIndices(nx, nz, false)
	// Without the triangles of the holes
	idx := m.Indices[:0]
	for t := 0; t+2 < len(m.Indices); t += 3 {
		a, b, c := m.Indices[t], m.Indices[t+1], m.Indices[t+2]
		if valid[a] && valid[b] && valid[c] {
			idx = append(idx, a, b, c)
		}
	}
	m.Indices = idx
	m.ComputeNormals()
	return m
}

// ColorByHeight colors the corners of the mesh by their y with the colors
// of the stops, from the lowest (0) to the highest (1): the map of a
// terrain, the height of a function
func (m *Mesh) ColorByHeight(stops []GradientStop) {
	lo, hi := m.Bounds()
	span := hi.Y - lo.Y
	m.Colors = make([]color.RGBA, len(m.Positions))
	for i, p := range m.Positions {
		t := 0.0
		if span > 0 {
			t = (p.Y - lo.Y) / span
		}
		m.Colors[i] = gradientColor(stops, t)
	}
}

// ErrModel is the error of LoadOBJ and LoadSTL for a file they cannot read
var ErrModel = errors.New("bad 3D model")

// LoadOBJ reads a Wavefront OBJ model: the corners (v, with the colors of
// "v x y z r g b" too), the texture points (vt), the normals (vn) and the
// faces (f, of any count of corners, as triangles). The materials and the
// groups are not read.
func LoadOBJ(r io.Reader) (*Mesh, error) {
	var pos, nrm []Vec3
	var uvs []PointF
	var cols []color.RGBA
	hasColors := false
	m := &Mesh{}
	// A corner of a face is the combination of the numbers of its
	// position, texture point and normal: made once
	type corner struct{ v, t, n int }
	corners := map[corner]int{}
	hasUV, hasN := false, false
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	line := 0
	parse := func(fields []string, n int) ([]float64, error) {
		if len(fields) < n {
			return nil, fmt.Errorf("%w: line %d: %d numbers expected", ErrModel, line, n)
		}
		out := make([]float64, len(fields))
		for i, f := range fields {
			v, err := strconv.ParseFloat(f, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: line %d: %v", ErrModel, line, err)
			}
			out[i] = v
		}
		return out, nil
	}
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if i := strings.IndexByte(text, '#'); i >= 0 {
			text = text[:i]
		}
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "v":
			v, err := parse(fields[1:], 3)
			if err != nil {
				return m, err
			}
			pos = append(pos, Vec3{v[0], v[1], v[2]})
			c := color.RGBA{255, 255, 255, 255}
			if len(v) >= 6 {
				hasColors = true
				k := 255.0
				if v[3] > 1 || v[4] > 1 || v[5] > 1 {
					k = 1
				}
				c = color.RGBA{uint8(clamp255(v[3] * k)), uint8(clamp255(v[4] * k)), uint8(clamp255(v[5] * k)), 255}
			}
			cols = append(cols, c)
		case "vt":
			v, err := parse(fields[1:], 1)
			if err != nil {
				return m, err
			}
			uv := PointF{X: v[0]}
			if len(v) > 1 {
				uv.Y = v[1]
			}
			uvs = append(uvs, uv)
		case "vn":
			v, err := parse(fields[1:], 3)
			if err != nil {
				return m, err
			}
			nrm = append(nrm, Vec3{v[0], v[1], v[2]}.Normalize())
		case "f":
			if len(fields) < 4 {
				return m, fmt.Errorf("%w: line %d: a face of less than 3 corners", ErrModel, line)
			}
			face := make([]int, 0, len(fields)-1)
			for _, f := range fields[1:] {
				parts := strings.Split(f, "/")
				var c corner
				var err error
				ref := func(s string, n int) (int, error) {
					if s == "" {
						return -1, nil
					}
					i, err := strconv.Atoi(s)
					if err != nil {
						return 0, fmt.Errorf("%w: line %d: %v", ErrModel, line, err)
					}
					if i < 0 {
						i += n // from the end
					} else {
						i--
					}
					if i < 0 || i >= n {
						return 0, fmt.Errorf("%w: line %d: no element %s", ErrModel, line, s)
					}
					return i, nil
				}
				if c.v, err = ref(parts[0], len(pos)); err != nil {
					return m, err
				}
				if c.v < 0 {
					return m, fmt.Errorf("%w: line %d: a corner without a position", ErrModel, line)
				}
				c.t, c.n = -1, -1
				if len(parts) > 1 {
					if c.t, err = ref(parts[1], len(uvs)); err != nil {
						return m, err
					}
				}
				if len(parts) > 2 {
					if c.n, err = ref(parts[2], len(nrm)); err != nil {
						return m, err
					}
				}
				idx, ok := corners[c]
				if !ok {
					idx = len(m.Positions)
					corners[c] = idx
					m.Positions = append(m.Positions, pos[c.v])
					m.Colors = append(m.Colors, cols[c.v])
					var uv PointF
					if c.t >= 0 {
						uv, hasUV = uvs[c.t], true
					}
					m.UVs = append(m.UVs, uv)
					var n Vec3
					if c.n >= 0 {
						n, hasN = nrm[c.n], true
					}
					m.Normals = append(m.Normals, n)
				}
				face = append(face, idx)
			}
			// A polygon as a fan of triangles
			for i := 1; i+1 < len(face); i++ {
				m.Indices = append(m.Indices, face[0], face[i], face[i+1])
			}
		}
	}
	if err := sc.Err(); err != nil {
		return m, err
	}
	if len(m.Indices) == 0 {
		return m, fmt.Errorf("%w: no faces", ErrModel)
	}
	if !hasColors {
		m.Colors = nil
	}
	if !hasUV {
		m.UVs = nil
	}
	if !hasN {
		m.ComputeNormals()
	} else {
		// The corners without a normal of their own get the computed one
		missing := false
		for _, n := range m.Normals {
			if n == (Vec3{}) {
				missing = true
				break
			}
		}
		if missing {
			given := m.Normals
			m.ComputeNormals()
			for i, n := range given {
				if n != (Vec3{}) {
					m.Normals[i] = n
				}
			}
		}
	}
	return m, nil
}

func clamp255(v float64) float64 { return math.Max(0, math.Min(255, math.Round(v))) }

// LoadSTL reads an STL model, binary or text. Its triangles have no
// shared corners; the normals are made per triangle, so the model looks
// faceted, as CAD parts are.
func LoadSTL(r io.Reader) (*Mesh, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	m := &Mesh{}
	// A binary file: an 80 byte header, the count, 50 bytes a triangle.
	// Some binary files start with "solid" too, so the size decides.
	if len(data) >= 84 {
		n := int(binary.LittleEndian.Uint32(data[80:84]))
		if 84+n*50 == len(data) {
			f := func(off int) float64 {
				return float64(math.Float32frombits(binary.LittleEndian.Uint32(data[off:])))
			}
			for i := 0; i < n; i++ {
				off := 84 + i*50
				nrm := Vec3{f(off), f(off + 4), f(off + 8)}
				var tri [3]Vec3
				for k := 0; k < 3; k++ {
					o := off + 12 + k*12
					tri[k] = Vec3{f(o), f(o + 4), f(o + 8)}
				}
				addSTLTriangle(m, tri, nrm)
			}
			return m, nil
		}
	}
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("solid")) {
		return nil, fmt.Errorf("%w: neither a binary nor a text STL", ErrModel)
	}
	var tri [3]Vec3
	var nrm Vec3
	k := 0
	for i, l := range strings.Split(string(data), "\n") {
		fields := strings.Fields(l)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "facet":
			k = 0
			if len(fields) >= 5 {
				nrm = parseVec3(fields[2:5])
			}
		case "vertex":
			if len(fields) < 4 || k > 2 {
				return m, fmt.Errorf("%w: line %d", ErrModel, i+1)
			}
			tri[k] = parseVec3(fields[1:4])
			k++
		case "endfacet":
			if k == 3 {
				addSTLTriangle(m, tri, nrm)
			}
		}
	}
	if len(m.Positions) == 0 {
		return m, fmt.Errorf("%w: no triangles", ErrModel)
	}
	return m, nil
}

func parseVec3(f []string) Vec3 {
	x, _ := strconv.ParseFloat(f[0], 64)
	y, _ := strconv.ParseFloat(f[1], 64)
	z, _ := strconv.ParseFloat(f[2], 64)
	return Vec3{x, y, z}
}

// addSTLTriangle adds a triangle with its normal, made from the corners
// when the file's is missing
func addSTLTriangle(m *Mesh, tri [3]Vec3, n Vec3) {
	computed := tri[1].Sub(tri[0]).Cross(tri[2].Sub(tri[0])).Normalize()
	if n.Length() < 0.5 || n.Dot(computed) < 0 {
		n = computed
	}
	for _, p := range tri {
		m.Positions = append(m.Positions, p)
		m.Normals = append(m.Normals, n.Normalize())
	}
}

// WriteOBJ writes the mesh as a Wavefront OBJ model
func (m *Mesh) WriteOBJ(w io.Writer) error {
	bw := bufio.NewWriter(w)
	num := func(v float64) string { return strconv.FormatFloat(v, 'g', 7, 64) }
	for i, p := range m.Positions {
		fmt.Fprintf(bw, "v %s %s %s", num(p.X), num(p.Y), num(p.Z))
		if i < len(m.Colors) {
			c := m.Colors[i]
			fmt.Fprintf(bw, " %s %s %s", num(float64(c.R)/255), num(float64(c.G)/255), num(float64(c.B)/255))
		}
		bw.WriteString("\n")
	}
	for _, t := range m.UVs {
		fmt.Fprintf(bw, "vt %s %s\n", num(t.X), num(t.Y))
	}
	for _, n := range m.Normals {
		fmt.Fprintf(bw, "vn %s %s %s\n", num(n.X), num(n.Y), num(n.Z))
	}
	ref := func(i int) string {
		s := strconv.Itoa(i + 1)
		switch {
		case len(m.UVs) > i && len(m.Normals) > i:
			return s + "/" + s + "/" + s
		case len(m.Normals) > i:
			return s + "//" + s
		case len(m.UVs) > i:
			return s + "/" + s
		}
		return s
	}
	for t := 0; t < m.TriangleCount(); t++ {
		a, b, c := m.triangle(t)
		fmt.Fprintf(bw, "f %s %s %s\n", ref(a), ref(b), ref(c))
	}
	return bw.Flush()
}

// LoadMeshFile reads a model file by its extension: .obj (LoadOBJ) or
// .stl (LoadSTL)
func LoadMeshFile(path string) (*Mesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	switch strings.ToLower(filepath.Ext(path)) {
	case ".obj":
		return LoadOBJ(f)
	case ".stl":
		return LoadSTL(f)
	}
	return nil, fmt.Errorf("%w: unknown file type %q", ErrModel, filepath.Ext(path))
}

// NewShapeMesh returns a shape by its name, about 2 across: "box",
// "sphere", "cylinder", "cone", "torus", "plane"; nil for another name
func NewShapeMesh(name string) *Mesh {
	switch strings.ToLower(name) {
	case "box", "cube":
		return NewBoxMesh(1.6, 1.6, 1.6)
	case "sphere":
		return NewSphereMesh(1, 48, 32)
	case "cylinder":
		return NewCylinderMesh(0.8, 0.8, 2, 48, true)
	case "cone":
		return NewConeMesh(1, 2, 48)
	case "torus":
		return NewTorusMesh(1, 0.4, 64, 32)
	case "plane":
		return NewPlaneMesh(2, 2, 1, 1)
	}
	return nil
}
