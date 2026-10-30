package ui

import "math"

// Vec3 is a point or a direction in 3D. The 3D of nui is right-handed: x
// to the right, y up, z towards the viewer, as in OpenGL and most model
// files.
type Vec3 struct {
	X, Y, Z float64
}

// V3 returns the vector (x, y, z)
func V3(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3             { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3             { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(k float64) Vec3        { return Vec3{a.X * k, a.Y * k, a.Z * k} }
func (a Vec3) Mul(b Vec3) Vec3             { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }
func (a Vec3) Neg() Vec3                   { return Vec3{-a.X, -a.Y, -a.Z} }
func (a Vec3) Dot(b Vec3) float64          { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Length() float64             { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Distance(b Vec3) float64     { return a.Sub(b).Length() }
func (a Vec3) Lerp(b Vec3, t float64) Vec3 { return a.Add(b.Sub(a).Scale(t)) }

// Cross returns the cross product: perpendicular to both, by the right
// hand rule
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

// Normalize returns the vector of length 1 in the same direction; the zero
// vector stays zero
func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l == 0 {
		return a
	}
	return a.Scale(1 / l)
}

// Min and Max return the smaller and the larger components of both
func (a Vec3) Min(b Vec3) Vec3 {
	return Vec3{math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z)}
}
func (a Vec3) Max(b Vec3) Vec3 {
	return Vec3{math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z)}
}

// Mat4 is a 4x4 matrix of a 3D transform, row by row: M[r*4+c]. A point
// is a column on the right of it, so a.Mul(b) applies b first, then a, as
// in OpenGL and in the formulas of the textbooks.
type Mat4 [16]float64

// Identity4 returns the transform that leaves the points as they are
func Identity4() Mat4 {
	return Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// Translate4 returns the transform that moves by v
func Translate4(v Vec3) Mat4 {
	return Mat4{1, 0, 0, v.X, 0, 1, 0, v.Y, 0, 0, 1, v.Z, 0, 0, 0, 1}
}

// Scale4 returns the transform that stretches by v along the axes
func Scale4(v Vec3) Mat4 {
	return Mat4{v.X, 0, 0, 0, 0, v.Y, 0, 0, 0, 0, v.Z, 0, 0, 0, 0, 1}
}

// RotateX4, RotateY4 and RotateZ4 return the turns around the axes by the
// angle in radians, counterclockwise looking from the end of the axis
func RotateX4(a float64) Mat4 {
	s, c := math.Sincos(a)
	return Mat4{1, 0, 0, 0, 0, c, -s, 0, 0, s, c, 0, 0, 0, 0, 1}
}

func RotateY4(a float64) Mat4 {
	s, c := math.Sincos(a)
	return Mat4{c, 0, s, 0, 0, 1, 0, 0, -s, 0, c, 0, 0, 0, 0, 1}
}

func RotateZ4(a float64) Mat4 {
	s, c := math.Sincos(a)
	return Mat4{c, -s, 0, 0, s, c, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// Rotate4 returns the turn around the axis by the angle in radians
func Rotate4(axis Vec3, a float64) Mat4 {
	return QuatAxisAngle(axis, a).Mat4()
}

// Perspective4 returns the projection of a camera with the vertical field
// of view fovY in radians, the width to height aspect of the picture and
// the distances of the near and the far planes (OpenGL's clip space)
func Perspective4(fovY, aspect, near, far float64) Mat4 {
	f := 1 / math.Tan(fovY/2)
	return Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), 2 * far * near / (near - far),
		0, 0, -1, 0,
	}
}

// Orthographic4 returns the parallel projection of the box from left,
// bottom, -near to right, top, -far onto the cube of the clip space
func Orthographic4(left, right, bottom, top, near, far float64) Mat4 {
	return Mat4{
		2 / (right - left), 0, 0, -(right + left) / (right - left),
		0, 2 / (top - bottom), 0, -(top + bottom) / (top - bottom),
		0, 0, -2 / (far - near), -(far + near) / (far - near),
		0, 0, 0, 1,
	}
}

// LookAt4 returns the view transform of a camera at eye looking at target,
// with up showing which way is up on the screen
func LookAt4(eye, target, up Vec3) Mat4 {
	f := target.Sub(eye).Normalize()
	s := f.Cross(up).Normalize()
	if s.Length() == 0 {
		// Looking along up: any side will do
		s = f.Cross(Vec3{1, 0, 0}).Normalize()
		if s.Length() == 0 {
			s = f.Cross(Vec3{0, 0, 1}).Normalize()
		}
	}
	u := s.Cross(f)
	return Mat4{
		s.X, s.Y, s.Z, -s.Dot(eye),
		u.X, u.Y, u.Z, -u.Dot(eye),
		-f.X, -f.Y, -f.Z, f.Dot(eye),
		0, 0, 0, 1,
	}
}

// Mul returns a times b: the transform that applies b, then a
func (a Mat4) Mul(b Mat4) Mat4 {
	var m Mat4
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			m[r*4+c] = a[r*4]*b[c] + a[r*4+1]*b[4+c] + a[r*4+2]*b[8+c] + a[r*4+3]*b[12+c]
		}
	}
	return m
}

// MulPoint returns the point transformed, divided by its w (a projection)
func (a Mat4) MulPoint(p Vec3) Vec3 {
	x := a[0]*p.X + a[1]*p.Y + a[2]*p.Z + a[3]
	y := a[4]*p.X + a[5]*p.Y + a[6]*p.Z + a[7]
	z := a[8]*p.X + a[9]*p.Y + a[10]*p.Z + a[11]
	w := a[12]*p.X + a[13]*p.Y + a[14]*p.Z + a[15]
	if w != 1 && w != 0 {
		return Vec3{x / w, y / w, z / w}
	}
	return Vec3{x, y, z}
}

// MulDir returns the direction transformed: without the move
func (a Mat4) MulDir(d Vec3) Vec3 {
	return Vec3{
		a[0]*d.X + a[1]*d.Y + a[2]*d.Z,
		a[4]*d.X + a[5]*d.Y + a[6]*d.Z,
		a[8]*d.X + a[9]*d.Y + a[10]*d.Z,
	}
}

// mul4 returns the homogeneous point (x, y, z, 1) transformed
func (a *Mat4) mul4(p Vec3) (x, y, z, w float64) {
	x = a[0]*p.X + a[1]*p.Y + a[2]*p.Z + a[3]
	y = a[4]*p.X + a[5]*p.Y + a[6]*p.Z + a[7]
	z = a[8]*p.X + a[9]*p.Y + a[10]*p.Z + a[11]
	w = a[12]*p.X + a[13]*p.Y + a[14]*p.Z + a[15]
	return
}

// Transpose returns the matrix flipped over its diagonal
func (a Mat4) Transpose() Mat4 {
	var m Mat4
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			m[c*4+r] = a[r*4+c]
		}
	}
	return m
}

// Inverse returns the transform that undoes a; ok is false when there is
// none (a flattens the space)
func (a Mat4) Inverse() (inv Mat4, ok bool) {
	m := a
	inv[0] = m[5]*m[10]*m[15] - m[5]*m[11]*m[14] - m[9]*m[6]*m[15] + m[9]*m[7]*m[14] + m[13]*m[6]*m[11] - m[13]*m[7]*m[10]
	inv[4] = -m[4]*m[10]*m[15] + m[4]*m[11]*m[14] + m[8]*m[6]*m[15] - m[8]*m[7]*m[14] - m[12]*m[6]*m[11] + m[12]*m[7]*m[10]
	inv[8] = m[4]*m[9]*m[15] - m[4]*m[11]*m[13] - m[8]*m[5]*m[15] + m[8]*m[7]*m[13] + m[12]*m[5]*m[11] - m[12]*m[7]*m[9]
	inv[12] = -m[4]*m[9]*m[14] + m[4]*m[10]*m[13] + m[8]*m[5]*m[14] - m[8]*m[6]*m[13] - m[12]*m[5]*m[10] + m[12]*m[6]*m[9]
	inv[1] = -m[1]*m[10]*m[15] + m[1]*m[11]*m[14] + m[9]*m[2]*m[15] - m[9]*m[3]*m[14] - m[13]*m[2]*m[11] + m[13]*m[3]*m[10]
	inv[5] = m[0]*m[10]*m[15] - m[0]*m[11]*m[14] - m[8]*m[2]*m[15] + m[8]*m[3]*m[14] + m[12]*m[2]*m[11] - m[12]*m[3]*m[10]
	inv[9] = -m[0]*m[9]*m[15] + m[0]*m[11]*m[13] + m[8]*m[1]*m[15] - m[8]*m[3]*m[13] - m[12]*m[1]*m[11] + m[12]*m[3]*m[9]
	inv[13] = m[0]*m[9]*m[14] - m[0]*m[10]*m[13] - m[8]*m[1]*m[14] + m[8]*m[2]*m[13] + m[12]*m[1]*m[10] - m[12]*m[2]*m[9]
	inv[2] = m[1]*m[6]*m[15] - m[1]*m[7]*m[14] - m[5]*m[2]*m[15] + m[5]*m[3]*m[14] + m[13]*m[2]*m[7] - m[13]*m[3]*m[6]
	inv[6] = -m[0]*m[6]*m[15] + m[0]*m[7]*m[14] + m[4]*m[2]*m[15] - m[4]*m[3]*m[14] - m[12]*m[2]*m[7] + m[12]*m[3]*m[6]
	inv[10] = m[0]*m[5]*m[15] - m[0]*m[7]*m[13] - m[4]*m[1]*m[15] + m[4]*m[3]*m[13] + m[12]*m[1]*m[7] - m[12]*m[3]*m[5]
	inv[14] = -m[0]*m[5]*m[14] + m[0]*m[6]*m[13] + m[4]*m[1]*m[14] - m[4]*m[2]*m[13] - m[12]*m[1]*m[6] + m[12]*m[2]*m[5]
	inv[3] = -m[1]*m[6]*m[11] + m[1]*m[7]*m[10] + m[5]*m[2]*m[11] - m[5]*m[3]*m[10] - m[9]*m[2]*m[7] + m[9]*m[3]*m[6]
	inv[7] = m[0]*m[6]*m[11] - m[0]*m[7]*m[10] - m[4]*m[2]*m[11] + m[4]*m[3]*m[10] + m[8]*m[2]*m[7] - m[8]*m[3]*m[6]
	inv[11] = -m[0]*m[5]*m[11] + m[0]*m[7]*m[9] + m[4]*m[1]*m[11] - m[4]*m[3]*m[9] - m[8]*m[1]*m[7] + m[8]*m[3]*m[5]
	inv[15] = m[0]*m[5]*m[10] - m[0]*m[6]*m[9] - m[4]*m[1]*m[10] + m[4]*m[2]*m[9] + m[8]*m[1]*m[6] - m[8]*m[2]*m[5]
	det := m[0]*inv[0] + m[1]*inv[4] + m[2]*inv[8] + m[3]*inv[12]
	if det == 0 || math.IsNaN(det) {
		return Mat4{}, false
	}
	k := 1 / det
	for i := range inv {
		inv[i] *= k
	}
	return inv, true
}

// normalMatrix returns the transform of the normals of a: the inverse of
// its transpose, so they stay perpendicular to the surfaces under a
// stretch
func (a Mat4) normalMatrix() Mat4 {
	inv, ok := a.Inverse()
	if !ok {
		return a
	}
	return inv.Transpose()
}

// Quat is a rotation as a unit quaternion: smooth to combine and to
// interpolate, as the turns of a trackball are
type Quat struct {
	X, Y, Z, W float64
}

// IdentityQuat returns the rotation that does not turn
func IdentityQuat() Quat { return Quat{W: 1} }

// QuatAxisAngle returns the turn by the angle in radians around the axis
func QuatAxisAngle(axis Vec3, a float64) Quat {
	axis = axis.Normalize()
	s, c := math.Sincos(a / 2)
	return Quat{axis.X * s, axis.Y * s, axis.Z * s, c}
}

// Mul returns the rotation q after r
func (q Quat) Mul(r Quat) Quat {
	return Quat{
		q.W*r.X + q.X*r.W + q.Y*r.Z - q.Z*r.Y,
		q.W*r.Y - q.X*r.Z + q.Y*r.W + q.Z*r.X,
		q.W*r.Z + q.X*r.Y - q.Y*r.X + q.Z*r.W,
		q.W*r.W - q.X*r.X - q.Y*r.Y - q.Z*r.Z,
	}
}

// Normalize returns q of length 1
func (q Quat) Normalize() Quat {
	l := math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)
	if l == 0 {
		return IdentityQuat()
	}
	return Quat{q.X / l, q.Y / l, q.Z / l, q.W / l}
}

// Rotate returns the vector turned by q
func (q Quat) Rotate(v Vec3) Vec3 {
	u := Vec3{q.X, q.Y, q.Z}
	t := u.Cross(v).Scale(2)
	return v.Add(t.Scale(q.W)).Add(u.Cross(t))
}

// Mat4 returns the rotation as a matrix
func (q Quat) Mat4() Mat4 {
	x, y, z, w := q.X, q.Y, q.Z, q.W
	return Mat4{
		1 - 2*(y*y+z*z), 2 * (x*y - z*w), 2 * (x*z + y*w), 0,
		2 * (x*y + z*w), 1 - 2*(x*x+z*z), 2 * (y*z - x*w), 0,
		2 * (x*z - y*w), 2 * (y*z + x*w), 1 - 2*(x*x+y*y), 0,
		0, 0, 0, 1,
	}
}

// Slerp returns the rotation the part t of the way from q to r, along the
// shortest arc
func (q Quat) Slerp(r Quat, t float64) Quat {
	d := q.X*r.X + q.Y*r.Y + q.Z*r.Z + q.W*r.W
	if d < 0 {
		r, d = Quat{-r.X, -r.Y, -r.Z, -r.W}, -d
	}
	if d > 0.9995 {
		return Quat{q.X + (r.X-q.X)*t, q.Y + (r.Y-q.Y)*t, q.Z + (r.Z-q.Z)*t, q.W + (r.W-q.W)*t}.Normalize()
	}
	th := math.Acos(d)
	s := math.Sin(th)
	a, b := math.Sin((1-t)*th)/s, math.Sin(t*th)/s
	return Quat{q.X*a + r.X*b, q.Y*a + r.Y*b, q.Z*a + r.Z*b, q.W*a + r.W*b}
}

// Ray3 is a half line from Origin in the direction Dir: what is under the
// mouse in a 3D view (Camera.Ray)
type Ray3 struct {
	Origin, Dir Vec3
}

// At returns the point at the distance t along the ray (Dir of length 1)
func (r Ray3) At(t float64) Vec3 { return r.Origin.Add(r.Dir.Scale(t)) }

// IntersectTriangle returns the distance along the ray to the triangle and
// the barycentric coordinates of the hit (Möller–Trumbore); ok is false
// when the ray misses it. Both sides of the triangle are hit.
func (r Ray3) IntersectTriangle(a, b, c Vec3) (t, u, v float64, ok bool) {
	e1, e2 := b.Sub(a), c.Sub(a)
	p := r.Dir.Cross(e2)
	det := e1.Dot(p)
	if math.Abs(det) < 1e-12 {
		return 0, 0, 0, false
	}
	inv := 1 / det
	s := r.Origin.Sub(a)
	u = s.Dot(p) * inv
	if u < 0 || u > 1 {
		return 0, 0, 0, false
	}
	q := s.Cross(e1)
	v = r.Dir.Dot(q) * inv
	if v < 0 || u+v > 1 {
		return 0, 0, 0, false
	}
	t = e2.Dot(q) * inv
	return t, u, v, t > 1e-9
}

// IntersectPlane returns the distance along the ray to the plane through
// the point with the normal; ok is false when it is parallel to it or
// behind
func (r Ray3) IntersectPlane(point, normal Vec3) (t float64, ok bool) {
	d := r.Dir.Dot(normal)
	if math.Abs(d) < 1e-12 {
		return 0, false
	}
	t = point.Sub(r.Origin).Dot(normal) / d
	return t, t >= 0
}

// intersectBox returns whether the ray meets the box of min, max, and the
// distance to it
func (r Ray3) intersectBox(lo, hi Vec3) (float64, bool) {
	t0, t1 := 0.0, math.Inf(1)
	for i := 0; i < 3; i++ {
		var o, d, l, h float64
		switch i {
		case 0:
			o, d, l, h = r.Origin.X, r.Dir.X, lo.X, hi.X
		case 1:
			o, d, l, h = r.Origin.Y, r.Dir.Y, lo.Y, hi.Y
		default:
			o, d, l, h = r.Origin.Z, r.Dir.Z, lo.Z, hi.Z
		}
		if math.Abs(d) < 1e-15 {
			if o < l || o > h {
				return 0, false
			}
			continue
		}
		a, b := (l-o)/d, (h-o)/d
		if a > b {
			a, b = b, a
		}
		t0, t1 = math.Max(t0, a), math.Min(t1, b)
		if t0 > t1 {
			return 0, false
		}
	}
	return t0, true
}
