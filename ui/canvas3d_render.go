package ui

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Renderer3D draws 3D scenes on a canvas: a software rasterizer with a
// depth buffer, perspective-correct textures and colors, per pixel
// lighting (Blinn-Phong), translucency, lines and points, smoothed edges
// (supersampling) and all the processors of the computer. It keeps its
// buffers from frame to frame: keep one per view. Canvas.Draw3D uses a
// shared one.
type Renderer3D struct {
	// Antialias is how many samples a pixel takes along each side: 1 - one
	// (fast, jagged edges), 2 - four (smooth), 3 - nine. Zero is 2.
	Antialias int
	// Workers is how many goroutines draw; zero is one per processor
	Workers int

	w, h  int // the size of the buffers, in samples
	color []uint8
	depth []float32
	fill  [4]uint8 // the background, premultiplied: the buffers are cleared to it

	verts  []rVertex
	tris   []rTri
	trans  []rTri
	points []rPoint
	mats   []rMat
	lights []rLight
	clipV  []clipVert
	proj   []int32
	tex    map[image.Image]*rTexture

	// The cutting planes of the object being added, in its coordinates:
	// a point p is kept where plane[0..2]·p + plane[3] >= 0
	cuts    [][4]float64
	persp   bool
	ambient float32
	lineMi  int32 // the material of the lines of the frame, -1 - none yet
	stats   Render3DStats
}

// Render3DStats is what the last frame of a Renderer3D drew
type Render3DStats struct {
	Triangles int // drawn after the back sides and what is out of view are left out
	Lines     int
	Points    int
	Samples   int // of the buffers: pixels times Antialias squared
	Duration  time.Duration
}

// NewRenderer3D returns a renderer
func NewRenderer3D() *Renderer3D {
	return &Renderer3D{}
}

// Stats returns what the last frame drew and how long it took
func (r *Renderer3D) Stats() Render3DStats { return r.stats }

// The attributes of a corner, interpolated across a triangle: the normal
// and the position in the camera's space, the texture point, the color
const (
	attrN   = 0
	attrPos = 3
	attrUV  = 6
	attrCol = 8
	nAttrs  = 12
)

// clipVert is a corner in the clip space, before the division by w
type clipVert struct {
	x, y, z, w float64
	a          [nAttrs]float32
	// cut is how far it is on the kept side of each cutting plane
	cut [maxClipPlanes]float64
}

// maxClipPlanes is the most cutting planes of a scene that are used
const maxClipPlanes = 6

// rVertex is a corner on the buffer: x, y in samples, the depth key (less
// is nearer), 1/w and the attributes times 1/w, for the perspective
type rVertex struct {
	x, y, key, iw float32
	a             [nAttrs]float32
}

type rTri struct {
	v    [3]int32
	mat  int32
	flip bool // the back side seen: the normal turned
	// The samples it covers
	x0, y0, x1, y1 int32
	sortKey        float32
}

type rPoint struct {
	x, y, key, r float32
	col          [4]float32
}

type rMat struct {
	base                           [4]float32
	tex                            *rTexture
	spec, shin                     float32
	unlit, needVC, needUV, needPos bool
}

type rLight struct {
	point bool
	v     [3]float32 // the direction to the light, or its position, in the camera's space
	col   [3]float32
}

type rTexture struct {
	w, h int
	pix  []uint8 // straight RGBA
}

func toTexture(img image.Image) *rTexture {
	b := img.Bounds()
	t := &rTexture{w: b.Dx(), h: b.Dy(), pix: make([]uint8, b.Dx()*b.Dy()*4)}
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			t.pix[i], t.pix[i+1], t.pix[i+2], t.pix[i+3] = c.R, c.G, c.B, c.A
			i += 4
		}
	}
	return t
}

// sample returns the color of the texture at u, v (0..1, repeated; v up),
// bilinear, 0..1
func (t *rTexture) sample(u, v float32) (r, g, b, a float32) {
	if t.w == 0 || t.h == 0 {
		return 1, 1, 1, 1
	}
	x := u*float32(t.w) - 0.5
	y := (1-v)*float32(t.h) - 0.5
	x0 := float32(math.Floor(float64(x)))
	y0 := float32(math.Floor(float64(y)))
	fx, fy := x-x0, y-y0
	ix := (int(x0)%t.w + t.w) % t.w
	iy := (int(y0)%t.h + t.h) % t.h
	ix1, iy1 := (ix+1)%t.w, (iy+1)%t.h
	p00 := t.pix[(iy*t.w+ix)*4:]
	p10 := t.pix[(iy*t.w+ix1)*4:]
	p01 := t.pix[(iy1*t.w+ix)*4:]
	p11 := t.pix[(iy1*t.w+ix1)*4:]
	w00, w10, w01, w11 := (1-fx)*(1-fy), fx*(1-fy), (1-fx)*fy, fx*fy
	const k = 1.0 / 255
	r = (float32(p00[0])*w00 + float32(p10[0])*w10 + float32(p01[0])*w01 + float32(p11[0])*w11) * k
	g = (float32(p00[1])*w00 + float32(p10[1])*w10 + float32(p01[1])*w01 + float32(p11[1])*w11) * k
	b = (float32(p00[2])*w00 + float32(p10[2])*w10 + float32(p01[2])*w01 + float32(p11[2])*w11) * k
	a = (float32(p00[3])*w00 + float32(p10[3])*w10 + float32(p01[3])*w01 + float32(p11[3])*w11) * k
	return
}

func rgbaF(c color.RGBA) [4]float32 {
	return [4]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, float32(c.A) / 255}
}

// Draw3D draws the scene seen by the camera in the rectangle of the
// canvas, with a renderer shared by the canvases (see Renderer3D)
func (c *Canvas) Draw3D(x, y, w, h int, scene *Scene3D, cam Camera) {
	r := sharedRenderers.Get().(*Renderer3D)
	defer sharedRenderers.Put(r)
	r.Render(c, x, y, w, h, scene, cam)
}

var sharedRenderers = sync.Pool{New: func() any { return NewRenderer3D() }}

// Render draws the scene seen by the camera in the logical rectangle of
// the canvas, within its clip. The scene's labels are drawn with the
// canvas' font. Under a transform of the canvas the picture is drawn
// transformed, as an image.
func (r *Renderer3D) Render(cnv *Canvas, x, y, w, h int, scene *Scene3D, cam Camera) {
	if w <= 0 || h <= 0 || scene == nil || cnv.clipEmpty() {
		return
	}
	if cnv.state.hasXf {
		k := cnv.scale * max(1, cnv.state.xf.meanScale())
		img := r.RenderImage(int(math.Ceil(float64(w)*k)), int(math.Ceil(float64(h)*k)), scene, cam)
		cnv.drawImageXf(float64(x), float64(y), float64(w), float64(h), img, img.Rect, true)
		return
	}
	px, py, pw, ph := cnv.pixelRect(x, y, w, h)
	if pw <= 0 || ph <= 0 {
		return
	}
	visible := image.Rect(px, py, px+pw, py+ph).Intersect(cnv.clipRect()).Intersect(cnv.rgba.Rect)
	if visible.Empty() {
		return
	}
	ss := r.Antialias
	if ss <= 0 {
		ss = 2
	}
	ss = min(ss, 4)
	r.draw(pw*ss, ph*ss, float32(ss)*float32(cnv.scale), scene, cam)
	r.resolve(cnv, px, py, ss, visible)
	r.drawLabels(cnv, x, y, w, h, scene, cam)
}

// RenderImage returns the scene seen by the camera as a picture of the
// size in pixels: to save, to print, to test
func (r *Renderer3D) RenderImage(width, height int, scene *Scene3D, cam Camera) *image.RGBA {
	cnv, img := NewImageCanvas(width, height, 1)
	r.Render(cnv, 0, 0, width, height, scene, cam)
	return img
}

// draw draws the scene into the buffers of w x h samples; px is how many
// samples a logical pixel takes, for the widths of the lines
func (r *Renderer3D) draw(w, h int, px float32, scene *Scene3D, cam Camera) {
	start := time.Now()
	r.stats = Render3DStats{Samples: w * h}
	r.w, r.h = w, h
	if cap(r.color) < w*h*4 {
		r.color = make([]uint8, w*h*4)
		r.depth = make([]float32, w*h)
	}
	r.color = r.color[:w*h*4]
	r.depth = r.depth[:w*h]
	bg := scene.Background
	a := uint32(bg.A)
	// The buffers are cleared by rasterize, each band on its worker
	r.fill = [4]uint8{uint8((uint32(bg.R)*a + 127) / 255), uint8((uint32(bg.G)*a + 127) / 255), uint8((uint32(bg.B)*a + 127) / 255), bg.A}

	r.verts, r.tris, r.trans, r.points, r.mats = r.verts[:0], r.tris[:0], r.trans[:0], r.points[:0], r.mats[:0]
	r.lineMi = -1
	if r.tex == nil {
		r.tex = map[image.Image]*rTexture{}
	}
	used := map[image.Image]bool{}

	near, far := cam.nearFar(scene)
	view := cam.ViewMatrix()
	proj := cam.ProjectionMatrix(float64(w)/float64(h), near, far)
	r.persp = !cam.Orthographic
	r.setupLights(scene, view)

	for _, o := range scene.Objects {
		if o.Hidden || o.Mesh == nil || len(o.Mesh.Positions) == 0 {
			continue
		}
		mat := o.Material
		base := mat.Color
		if base == (color.RGBA{}) {
			base = color.RGBA{200, 200, 200, 255}
		}
		rm := rMat{base: rgbaF(base), unlit: mat.Unlit, spec: float32(mat.Specular), shin: float32(mat.Shininess)}
		if rm.shin <= 0 {
			rm.shin = 32
		}
		rm.needVC = len(o.Mesh.Colors) >= len(o.Mesh.Positions)
		if mat.Texture != nil && len(o.Mesh.UVs) >= len(o.Mesh.Positions) {
			t := r.tex[mat.Texture]
			if t == nil {
				t = toTexture(mat.Texture)
				r.tex[mat.Texture] = t
			}
			used[mat.Texture] = true
			rm.tex, rm.needUV = t, true
		}
		rm.needPos = !rm.unlit && (rm.spec > 0 || r.hasPointLights())
		mi := int32(len(r.mats))
		r.mats = append(r.mats, rm)
		model := o.model()
		mv := view.Mul(model)
		mvp := proj.Mul(mv)
		r.setCuts(scene, model)
		r.transformMesh(o.Mesh, mv, mvp, rm)
		translucent := base.A < 255
		if !mat.Wireframe {
			// The back sides seen through a cut are the inside: drawn in
			// the color of the section
			back := int32(-1)
			if mat.DoubleSided {
				back = mi
			} else if len(r.cuts) > 0 && !translucent {
				sc := scene.SectionColor
				if sc.A == 0 {
					sc = color.RGBA{uint8(int(base.R) * 3 / 5), uint8(int(base.G) * 3 / 5), uint8(int(base.B) * 3 / 5), 255}
				}
				back = int32(len(r.mats))
				r.mats = append(r.mats, rMat{base: rgbaF(sc), unlit: true})
			}
			r.assembleMesh(o.Mesh, mi, back, mat, translucent, mv.normalMatrix())
		}
		edgeCol, edgeW := mat.EdgeColor, mat.EdgeWidth
		if mat.Wireframe {
			edgeCol = base
		}
		if edgeCol.A > 0 {
			if edgeW <= 0 {
				edgeW = 1
			}
			edges := o.Mesh.wireEdges(mat.Wireframe)
			col := rgbaF(edgeCol)
			for i := 0; i+1 < len(edges); i += 2 {
				r.addLine(r.clipV[edges[i]], r.clipV[edges[i+1]], col, float32(edgeW)*px)
			}
			// And the outline of the curved sides seen from here: a
			// cylinder shows its sides as a drawing does
			if !mat.Wireframe {
				if inv, ok := model.Inverse(); ok {
					eye := inv.MulPoint(cam.Position)
					dir := inv.MulDir(cam.Target.Sub(cam.Position))
					o.Mesh.outlineEdges(eye, dir, cam.Orthographic, func(a, b int) {
						r.addLine(r.clipV[a], r.clipV[b], col, float32(edgeW)*px)
					})
				}
			}
		}
	}
	// Forget the textures not drawn any more
	for k := range r.tex {
		if !used[k] {
			delete(r.tex, k)
		}
	}
	r.cuts = r.cuts[:0]
	vp := proj.Mul(view)
	for _, l := range scene.Lines {
		a, b := clipPoint(vp, l.A), clipPoint(vp, l.B)
		width := l.Width
		if width <= 0 {
			width = 1
		}
		r.addLine(a, b, rgbaF(l.Color), float32(width)*px)
	}
	for _, p := range scene.Points {
		r.addPoint(clipPoint(vp, p.P), p, px)
	}
	r.stats.Triangles = len(r.tris) + len(r.trans) - 2*r.stats.Lines
	r.stats.Points = len(r.points)
	// The translucent triangles from the far to the near
	sort.SliceStable(r.trans, func(i, j int) bool { return r.trans[i].sortKey > r.trans[j].sortKey })
	r.rasterize()
	r.stats.Duration = time.Since(start)
}

func (r *Renderer3D) hasPointLights() bool {
	for _, l := range r.lights {
		if l.point {
			return true
		}
	}
	return false
}

// setupLights puts the lights of the scene into the camera's space
func (r *Renderer3D) setupLights(scene *Scene3D, view Mat4) {
	r.lights = r.lights[:0]
	r.ambient = float32(scene.Ambient)
	if len(scene.Lights) == 0 {
		if r.ambient == 0 {
			r.ambient = 0.25
		}
		d := Vec3{0.35, 0.5, 1}.Normalize()
		r.lights = append(r.lights, rLight{v: [3]float32{float32(d.X), float32(d.Y), float32(d.Z)}, col: [3]float32{0.8, 0.8, 0.8}})
		return
	}
	for _, l := range scene.Lights {
		col := l.Color
		if col == (color.RGBA{}) {
			col = color.RGBA{255, 255, 255, 255}
		}
		k := l.Intensity
		if k == 0 {
			k = 1
		}
		rl := rLight{point: l.Kind == LightPoint, col: [3]float32{float32(col.R) / 255 * float32(k), float32(col.G) / 255 * float32(k), float32(col.B) / 255 * float32(k)}}
		var v Vec3
		if rl.point {
			v = l.Position
			if !l.FromCamera {
				v = view.MulPoint(v)
			}
		} else {
			v = l.Direction.Neg()
			if !l.FromCamera {
				v = view.MulDir(v)
			}
			v = v.Normalize()
		}
		rl.v = [3]float32{float32(v.X), float32(v.Y), float32(v.Z)}
		r.lights = append(r.lights, rl)
	}
}

// setCuts sets the cutting planes of the scene for the object of the
// model matrix, in its coordinates
func (r *Renderer3D) setCuts(scene *Scene3D, model Mat4) {
	r.cuts = r.cuts[:0]
	for _, pl := range scene.ClipPlanes {
		if len(r.cuts) == maxClipPlanes {
			break
		}
		n := pl.Normal.Normalize()
		if n == (Vec3{}) {
			continue
		}
		// n·(L p + t) - n·q = (Lᵀ n)·p + n·(t - q)
		t := Vec3{model[3], model[7], model[11]}
		ln := Vec3{
			model[0]*n.X + model[4]*n.Y + model[8]*n.Z,
			model[1]*n.X + model[5]*n.Y + model[9]*n.Z,
			model[2]*n.X + model[6]*n.Y + model[10]*n.Z,
		}
		r.cuts = append(r.cuts, [4]float64{ln.X, ln.Y, ln.Z, n.Dot(t.Sub(pl.Point))})
	}
}

func clipPoint(m Mat4, p Vec3) clipVert {
	x, y, z, w := m.mul4(p)
	return clipVert{x: x, y: y, z: z, w: w}
}

// transformMesh puts the corners of the mesh into clip space, with their
// attributes, into r.clipV; big meshes on all the processors
func (r *Renderer3D) transformMesh(m *Mesh, mv, mvp Mat4, rm rMat) {
	n := len(m.Positions)
	if cap(r.clipV) < n {
		r.clipV = make([]clipVert, n)
	}
	r.clipV = r.clipV[:n]
	nm := mv.normalMatrix()
	hasN := len(m.Normals) >= n
	if !hasN && !rm.unlit {
		m.ComputeNormals()
		hasN = true
	}
	do := func(from, to int) {
		for i := from; i < to; i++ {
			p := m.Positions[i]
			cv := &r.clipV[i]
			cv.x, cv.y, cv.z, cv.w = mvp.mul4(p)
			for k, c := range r.cuts {
				cv.cut[k] = c[0]*p.X + c[1]*p.Y + c[2]*p.Z + c[3]
			}
			if hasN && !rm.unlit {
				nn := nm.MulDir(m.Normals[i])
				cv.a[attrN], cv.a[attrN+1], cv.a[attrN+2] = float32(nn.X), float32(nn.Y), float32(nn.Z)
			}
			if rm.needPos {
				vp := mv.MulPoint(p)
				cv.a[attrPos], cv.a[attrPos+1], cv.a[attrPos+2] = float32(vp.X), float32(vp.Y), float32(vp.Z)
			}
			if rm.needUV {
				uv := m.UVs[i]
				cv.a[attrUV], cv.a[attrUV+1] = float32(uv.X), float32(uv.Y)
			}
			if rm.needVC {
				c := m.Colors[i]
				cv.a[attrCol], cv.a[attrCol+1], cv.a[attrCol+2], cv.a[attrCol+3] = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
			}
		}
	}
	r.parallel(n, 8192, do)
}

// parallel calls do on the chunks of 0..n, at least min long, on the
// workers
func (r *Renderer3D) parallel(n, minChunk int, do func(from, to int)) {
	workers := r.workers()
	if n < minChunk*2 || workers == 1 {
		do(0, n)
		return
	}
	chunk := max(minChunk, (n+workers-1)/workers)
	var wg sync.WaitGroup
	for from := 0; from < n; from += chunk {
		to := min(from+chunk, n)
		wg.Add(1)
		go func() {
			defer wg.Done()
			do(from, to)
		}()
	}
	wg.Wait()
}

func (r *Renderer3D) workers() int {
	if r.Workers > 0 {
		return r.Workers
	}
	return runtime.GOMAXPROCS(0)
}

// project puts the corner on the buffer
func (r *Renderer3D) project(cv *clipVert) rVertex {
	iw := 1 / cv.w
	v := rVertex{
		x:  float32((cv.x*iw + 1) / 2 * float64(r.w)),
		y:  float32((1 - cv.y*iw) / 2 * float64(r.h)),
		iw: 1,
	}
	if r.persp {
		v.key = float32(-iw)
		v.iw = float32(iw)
	} else {
		v.key = float32(cv.z * iw)
	}
	for k := range v.a {
		v.a[k] = cv.a[k] * v.iw
	}
	return v
}

// outside reports whether the triangle is entirely beyond a side of the
// view
func outside(a, b, c *clipVert) bool {
	return (a.x > a.w && b.x > b.w && c.x > c.w) || (a.x < -a.w && b.x < -b.w && c.x < -c.w) ||
		(a.y > a.w && b.y > b.w && c.y > c.w) || (a.y < -a.w && b.y < -b.w && c.y < -c.w) ||
		(a.z > a.w && b.z > b.w && c.z > c.w)
}

// nearDist is how far the corner is in front of the near plane
func nearDist(v *clipVert) float64 { return v.z + v.w }

// assembleMesh makes the triangles of the mesh, its corners already in
// r.clipV
func (r *Renderer3D) assembleMesh(m *Mesh, mi, back int32, mat Material, translucent bool, nm Mat4) {
	n := len(m.Positions)
	if cap(r.proj) < n {
		r.proj = make([]int32, n)
	}
	r.proj = r.proj[:n]
	for i := range r.proj {
		r.proj[i] = -1
	}
	vert := func(i int) int32 {
		if r.proj[i] < 0 {
			r.proj[i] = int32(len(r.verts))
			r.verts = append(r.verts, r.project(&r.clipV[i]))
		}
		return r.proj[i]
	}
tris:
	for t := 0; t < m.TriangleCount(); t++ {
		ia, ib, ic := m.triangle(t)
		if ia >= n || ib >= n || ic >= n {
			continue
		}
		a, b, c := &r.clipV[ia], &r.clipV[ib], &r.clipV[ic]
		if outside(a, b, c) {
			continue
		}
		// Cut away all, or in part
		cut := false
		for k := range r.cuts {
			da, db, dc := a.cut[k], b.cut[k], c.cut[k]
			if da < 0 && db < 0 && dc < 0 {
				continue tris
			}
			if da < 0 || db < 0 || dc < 0 {
				cut = true
			}
		}
		if cut || mat.Flat || nearDist(a) < 0 || nearDist(b) < 0 || nearDist(c) < 0 {
			cv := [3]clipVert{*a, *b, *c}
			if mat.Flat && !mat.Unlit {
				// The normal of the face, in the camera's space
				pa, pb, pc := m.Positions[ia], m.Positions[ib], m.Positions[ic]
				fn := nm.MulDir(pb.Sub(pa).Cross(pc.Sub(pa))).Normalize()
				for k := range cv {
					cv[k].a[attrN], cv[k].a[attrN+1], cv[k].a[attrN+2] = float32(fn.X), float32(fn.Y), float32(fn.Z)
				}
			}
			r.addClipped(cv, mi, back, translucent)
			continue
		}
		r.addTri([3]int32{vert(ia), vert(ib), vert(ic)}, mi, back, translucent)
	}
}

// addTri adds the triangle of the projected corners; one that shows its
// back is drawn with the material back, or left out if it is -1
func (r *Renderer3D) addTri(v [3]int32, mi, back int32, translucent bool) {
	a, b, c := &r.verts[v[0]], &r.verts[v[1]], &r.verts[v[2]]
	area := (b.x-a.x)*(c.y-a.y) - (b.y-a.y)*(c.x-a.x)
	if area == 0 || math.IsNaN(float64(area)) {
		return
	}
	// y goes down the buffer: the front, counterclockwise in the scene,
	// is clockwise here
	flip := area > 0
	if flip {
		if back < 0 {
			return
		}
		mi = back
	}
	t := rTri{v: v, mat: mi, flip: flip}
	x0 := min(a.x, b.x, c.x)
	x1 := max(a.x, b.x, c.x)
	y0 := min(a.y, b.y, c.y)
	y1 := max(a.y, b.y, c.y)
	if x1 < 0 || y1 < 0 || x0 > float32(r.w) || y0 > float32(r.h) {
		return
	}
	t.x0 = int32(max(0, math.Floor(float64(x0))))
	t.y0 = int32(max(0, math.Floor(float64(y0))))
	t.x1 = int32(min(float64(r.w-1), math.Ceil(float64(x1))))
	t.y1 = int32(min(float64(r.h-1), math.Ceil(float64(y1))))
	if translucent {
		t.sortKey = (a.key + b.key + c.key) / 3
		r.trans = append(r.trans, t)
	} else {
		r.tris = append(r.tris, t)
	}
}

// addClipped adds the triangle cut by the near plane and by the cutting
// planes: what is in front of them, a fan of triangles
func (r *Renderer3D) addClipped(cv [3]clipVert, mi, back int32, translucent bool) {
	// Each plane adds a corner at most: 3 + 1 + maxClipPlanes
	var bufA, bufB [3 + 1 + maxClipPlanes]clipVert
	poly := bufA[:0]
	poly = append(poly, cv[:]...)
	cutBy := func(dist func(*clipVert) float64) {
		out := bufB[:0]
		if &poly[0] == &bufB[0] {
			out = bufA[:0]
		}
		for i := range poly {
			p, q := &poly[i], &poly[(i+1)%len(poly)]
			dp, dq := dist(p), dist(q)
			if dp >= 0 {
				out = append(out, *p)
			}
			if (dp >= 0) != (dq >= 0) {
				out = append(out, lerpClip(p, q, dp/(dp-dq)))
			}
		}
		poly = out
	}
	cutBy(nearDist)
	for k := range r.cuts {
		if len(poly) < 3 {
			return
		}
		cutBy(func(v *clipVert) float64 { return v.cut[k] })
	}
	n := len(poly)
	if n < 3 {
		return
	}
	base := int32(len(r.verts))
	for i := 0; i < n; i++ {
		r.verts = append(r.verts, r.project(&poly[i]))
	}
	for i := int32(1); i+1 < int32(n); i++ {
		r.addTri([3]int32{base, base + i, base + i + 1}, mi, back, translucent)
	}
}

func lerpClip(p, q *clipVert, t float64) clipVert {
	v := clipVert{
		x: p.x + (q.x-p.x)*t,
		y: p.y + (q.y-p.y)*t,
		z: p.z + (q.z-p.z)*t,
		w: p.w + (q.w-p.w)*t,
	}
	tf := float32(t)
	for k := range v.a {
		v.a[k] = p.a[k] + (q.a[k]-p.a[k])*tf
	}
	for k := range v.cut {
		v.cut[k] = p.cut[k] + (q.cut[k]-p.cut[k])*t
	}
	return v
}

// lineMat returns the material of the lines: their own colors, unlit
func (r *Renderer3D) lineMat() int32 {
	if r.lineMi < 0 {
		r.lineMi = int32(len(r.mats))
		r.mats = append(r.mats, rMat{base: [4]float32{1, 1, 1, 1}, unlit: true, needVC: true})
	}
	return r.lineMi
}

// addLine adds the line between the corners as a band of two triangles
// facing the viewer, width samples wide, a little nearer than it is: the
// edges of a surface and the lines on it do not sink into it
func (r *Renderer3D) addLine(a, b clipVert, col [4]float32, width float32) {
	for k := -1; k < len(r.cuts); k++ {
		da, db := nearDist(&a), nearDist(&b)
		if k >= 0 {
			da, db = a.cut[k], b.cut[k]
		}
		if da < 0 && db < 0 {
			return
		}
		if da < 0 {
			a = lerpClip(&a, &b, da/(da-db))
		} else if db < 0 {
			b = lerpClip(&b, &a, db/(db-da))
		}
	}
	pa, pb := r.project(&a), r.project(&b)
	dx, dy := pb.x-pa.x, pb.y-pa.y
	l := float32(math.Hypot(float64(dx), float64(dy)))
	if l < 1e-6 {
		dx, dy, l = 1, 0, 1
	}
	hw := max(width, 1) / 2
	nx, ny := -dy/l*hw, dx/l*hw
	// The ends lengthened by half the width: the lines of a wireframe meet
	ex, ey := dx/l*hw, dy/l*hw
	mi := r.lineMat()
	base := int32(len(r.verts))
	bias := func(v rVertex) rVertex {
		if r.persp {
			v.key *= 1 + 2e-3
		} else {
			v.key -= 2e-3
		}
		v.a = [nAttrs]float32{}
		for k := 0; k < 4; k++ {
			v.a[attrCol+k] = col[k] * v.iw
		}
		return v
	}
	for _, c := range [4][3]float32{{-1, 1, 0}, {-1, -1, 0}, {1, -1, 1}, {1, 1, 1}} {
		v := pa
		if c[2] == 1 {
			v = pb
		}
		v.x += nx*c[1] + ex*c[0]
		v.y += ny*c[1] + ey*c[0]
		r.verts = append(r.verts, bias(v))
	}
	translucent := col[3] < 1
	r.addTri([3]int32{base, base + 1, base + 2}, mi, mi, translucent)
	r.addTri([3]int32{base, base + 2, base + 3}, mi, mi, translucent)
	r.stats.Lines++
}

func (r *Renderer3D) addPoint(cv clipVert, p Point3D, px float32) {
	if nearDist(&cv) < 0 || cv.w <= 0 {
		return
	}
	v := r.project(&cv)
	size := p.Size
	if size <= 0 {
		size = 3
	}
	rad := float32(size) * px / 2
	if v.x+rad < 0 || v.y+rad < 0 || v.x-rad > float32(r.w) || v.y-rad > float32(r.h) {
		return
	}
	key := v.key
	if r.persp {
		key *= 1 + 1e-3
	} else {
		key -= 1e-3
	}
	r.points = append(r.points, rPoint{x: v.x, y: v.y, key: key, r: rad, col: rgbaF(p.Color)})
}

// rasterize draws the triangles and the points into the buffers: the rows
// in bands, the bands on the workers
func (r *Renderer3D) rasterize() {
	bandH := max(16, r.h/(r.workers()*4))
	bands := (r.h + bandH - 1) / bandH
	var next int32 = -1
	var wg sync.WaitGroup
	for w := 0; w < min(r.workers(), bands); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				b := int(atomic.AddInt32(&next, 1))
				if b >= bands {
					return
				}
				y0, y1 := int32(b*bandH), int32(min((b+1)*bandH, r.h)-1)
				r.clearRows(int(y0), int(y1))
				// The depth of the opaque surfaces first, then each pixel is
				// lit once, by the surface seen there: the hidden ones cost
				// no light. The textured ones may have holes, so they are
				// drawn as they come.
				for i := range r.tris {
					if t := &r.tris[i]; t.y1 >= y0 && t.y0 <= y1 && r.mats[t.mat].tex == nil {
						r.rasterTri(t, y0, y1, rasterDepth)
					}
				}
				for i := range r.tris {
					if t := &r.tris[i]; t.y1 >= y0 && t.y0 <= y1 {
						mode := rasterSeen
						if r.mats[t.mat].tex != nil {
							mode = rasterOpaque
						}
						r.rasterTri(t, y0, y1, mode)
					}
				}
				for i := range r.points {
					r.rasterPoint(&r.points[i], y0, y1)
				}
				for i := range r.trans {
					if t := &r.trans[i]; t.y1 >= y0 && t.y0 <= y1 {
						r.rasterTri(t, y0, y1, rasterBlend)
					}
				}
			}
		}()
	}
	wg.Wait()
}

// clearRows clears the rows y0..y1 of the buffers: the background, the
// depth infinite
func (r *Renderer3D) clearRows(y0, y1 int) {
	col := r.color[y0*r.w*4 : (y1+1)*r.w*4]
	copy(col, r.fill[:])
	for n := 4; n < len(col); n *= 2 {
		copy(col[n:], col[:n])
	}
	depth := r.depth[y0*r.w : (y1+1)*r.w]
	depth[0] = float32(math.Inf(1))
	for n := 1; n < len(depth); n *= 2 {
		copy(depth[n:], depth[:n])
	}
}

// The passes of rasterTri
const (
	// rasterDepth writes the depth only
	rasterDepth = iota
	// rasterSeen colors the samples whose depth is the triangle's: those
	// where it was found the nearest by rasterDepth
	rasterSeen
	// rasterOpaque tests the depth and writes the color and the depth
	rasterOpaque
	// rasterBlend blends the color over the samples it is in front of,
	// keeping their depth: a translucent surface
	rasterBlend
)

// rasterTri fills the samples of the triangle within the rows y0..y1 in
// the pass mode
func (r *Renderer3D) rasterTri(t *rTri, y0, y1 int32, mode int) {
	blend := mode == rasterBlend
	a, b, c := &r.verts[t.v[0]], &r.verts[t.v[1]], &r.verts[t.v[2]]
	area := (b.x-a.x)*(c.y-a.y) - (b.y-a.y)*(c.x-a.x)
	if area == 0 {
		return
	}
	inv := 1 / area
	m := &r.mats[t.mat]
	ys, ye := max(t.y0, y0), min(t.y1, y1)
	// The edge functions, the barycentric coordinates, positive inside: w0
	// of the edge b-c, w1 of c-a, w2 of a-b; their steps along x
	e0dx := -(c.y - b.y) * inv
	e1dx := -(a.y - c.y) * inv
	e2dx := -(b.y - a.y) * inv
	fx0 := float32(t.x0) + 0.5
	var attrs [nAttrs]float32
	for y := ys; y <= ye; y++ {
		fy := float32(y) + 0.5
		w0 := ((c.x-b.x)*(fy-b.y) - (c.y-b.y)*(fx0-b.x)) * inv
		w1 := ((a.x-c.x)*(fy-c.y) - (a.y-c.y)*(fx0-c.x)) * inv
		w2 := ((b.x-a.x)*(fy-a.y) - (b.y-a.y)*(fx0-a.x)) * inv
		row := int(y) * r.w
		inside := false
		for x := t.x0; x <= t.x1; x, w0, w1, w2 = x+1, w0+e0dx, w1+e1dx, w2+e2dx {
			if w0 < 0 || w1 < 0 || w2 < 0 {
				if inside {
					break // a triangle is convex: past it on this row
				}
				continue
			}
			inside = true
			key := w0*a.key + w1*b.key + w2*c.key
			idx := row + int(x)
			if mode == rasterSeen {
				if key != r.depth[idx] {
					continue
				}
			} else if key >= r.depth[idx] {
				continue
			}
			if !r.persp && (key < -1 || key > 1) {
				continue
			}
			if mode == rasterDepth {
				r.depth[idx] = key
				continue
			}
			iw := w0*a.iw + w1*b.iw + w2*c.iw
			k := 1 / iw
			cr, cg, cb, ca := m.base[0], m.base[1], m.base[2], m.base[3]
			if m.needVC {
				for j := attrCol; j < attrCol+4; j++ {
					attrs[j] = (w0*a.a[j] + w1*b.a[j] + w2*c.a[j]) * k
				}
				cr, cg, cb, ca = cr*attrs[attrCol], cg*attrs[attrCol+1], cb*attrs[attrCol+2], ca*attrs[attrCol+3]
			}
			if m.tex != nil {
				u := (w0*a.a[attrUV] + w1*b.a[attrUV] + w2*c.a[attrUV]) * k
				v := (w0*a.a[attrUV+1] + w1*b.a[attrUV+1] + w2*c.a[attrUV+1]) * k
				tr, tg, tb, ta := m.tex.sample(u, v)
				cr, cg, cb, ca = cr*tr, cg*tg, cb*tb, ca*ta
			}
			if ca < 1.0/255 {
				continue
			}
			if !m.unlit {
				for j := attrN; j < attrN+3; j++ {
					attrs[j] = (w0*a.a[j] + w1*b.a[j] + w2*c.a[j]) * k
				}
				if m.needPos {
					for j := attrPos; j < attrPos+3; j++ {
						attrs[j] = (w0*a.a[j] + w1*b.a[j] + w2*c.a[j]) * k
					}
				}
				cr, cg, cb = r.shade(m, &attrs, t.flip, cr, cg, cb)
			}
			p := r.color[idx*4 : idx*4+4 : idx*4+4]
			if ca >= 1 && !blend {
				p[0], p[1], p[2], p[3] = to8(cr), to8(cg), to8(cb), 255
				r.depth[idx] = key
				continue
			}
			// Over what is there; a translucent surface hides nothing
			ia := 1 - ca
			p[0] = to8(cr*ca + float32(p[0])/255*ia)
			p[1] = to8(cg*ca + float32(p[1])/255*ia)
			p[2] = to8(cb*ca + float32(p[2])/255*ia)
			p[3] = to8(ca + float32(p[3])/255*ia)
			if !blend {
				r.depth[idx] = key
			}
		}
	}
}

func to8(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

// shade returns the color of the surface of the base color lit by the
// lights at the point of the attributes
func (r *Renderer3D) shade(m *rMat, at *[nAttrs]float32, flip bool, br, bg, bb float32) (float32, float32, float32) {
	nx, ny, nz := at[attrN], at[attrN+1], at[attrN+2]
	l := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if l > 0 {
		nx, ny, nz = nx/l, ny/l, nz/l
	}
	if flip {
		nx, ny, nz = -nx, -ny, -nz
	}
	amb := r.ambient
	cr, cg, cb := br*amb, bg*amb, bb*amb
	px, py, pz := at[attrPos], at[attrPos+1], at[attrPos+2]
	for i := range r.lights {
		lt := &r.lights[i]
		lx, ly, lz := lt.v[0], lt.v[1], lt.v[2]
		if lt.point {
			lx, ly, lz = lx-px, ly-py, lz-pz
			ll := float32(math.Sqrt(float64(lx*lx + ly*ly + lz*lz)))
			if ll > 0 {
				lx, ly, lz = lx/ll, ly/ll, lz/ll
			}
		}
		ndl := nx*lx + ny*ly + nz*lz
		if ndl <= 0 {
			continue
		}
		cr += br * lt.col[0] * ndl
		cg += bg * lt.col[1] * ndl
		cb += bb * lt.col[2] * ndl
		if m.spec > 0 {
			// Blinn-Phong: the half way between the light and the eye
			vx, vy, vz := float32(0), float32(0), float32(1)
			if r.persp {
				vl := float32(math.Sqrt(float64(px*px + py*py + pz*pz)))
				if vl > 0 {
					vx, vy, vz = -px/vl, -py/vl, -pz/vl
				}
			}
			hx, hy, hz := lx+vx, ly+vy, lz+vz
			hl := float32(math.Sqrt(float64(hx*hx + hy*hy + hz*hz)))
			if hl > 0 {
				ndh := (nx*hx + ny*hy + nz*hz) / hl
				if ndh > 0 {
					// Schlick's approximation of ndh^shin: as bright and as
					// wide, without a power
					s := m.spec * ndh / (m.shin - m.shin*ndh + ndh)
					cr += lt.col[0] * s
					cg += lt.col[1] * s
					cb += lt.col[2] * s
				}
			}
		}
	}
	return cr, cg, cb
}

// rasterPoint fills the samples of the round dot within the rows y0..y1
func (r *Renderer3D) rasterPoint(p *rPoint, y0, y1 int32) {
	ys := max(int32(math.Floor(float64(p.y-p.r))), y0, 0)
	ye := min(int32(math.Ceil(float64(p.y+p.r))), y1, int32(r.h-1))
	xs := max(int32(math.Floor(float64(p.x-p.r))), 0)
	xe := min(int32(math.Ceil(float64(p.x+p.r))), int32(r.w-1))
	r2 := p.r * p.r
	ca := p.col[3]
	for y := ys; y <= ye; y++ {
		dy := float32(y) + 0.5 - p.y
		for x := xs; x <= xe; x++ {
			dx := float32(x) + 0.5 - p.x
			if dx*dx+dy*dy > r2 {
				continue
			}
			idx := int(y)*r.w + int(x)
			if p.key >= r.depth[idx] {
				continue
			}
			px := r.color[idx*4 : idx*4+4 : idx*4+4]
			if ca >= 1 {
				px[0], px[1], px[2], px[3] = to8(p.col[0]), to8(p.col[1]), to8(p.col[2]), 255
				r.depth[idx] = p.key
				continue
			}
			ia := 1 - ca
			px[0] = to8(p.col[0]*ca + float32(px[0])/255*ia)
			px[1] = to8(p.col[1]*ca + float32(px[1])/255*ia)
			px[2] = to8(p.col[2]*ca + float32(px[2])/255*ia)
			px[3] = to8(ca + float32(px[3])/255*ia)
		}
	}
}

// resolve puts the buffers on the canvas: the samples of every pixel
// averaged, blended over what is there, within visible
func (r *Renderer3D) resolve(cnv *Canvas, px, py, ss int, visible image.Rectangle) {
	n := uint32(ss * ss)
	rows := visible.Dy()
	r.parallel(rows, 32, func(from, to int) {
		for yy := visible.Min.Y + from; yy < visible.Min.Y+to; yy++ {
			off := cnv.rgba.PixOffset(visible.Min.X, yy)
			sy := (yy - py) * ss
			for xx := visible.Min.X; xx < visible.Max.X; xx, off = xx+1, off+4 {
				sx := (xx - px) * ss
				var sr, sg, sb, sa uint32
				for j := 0; j < ss; j++ {
					i := ((sy+j)*r.w + sx) * 4
					for k := 0; k < ss; k++ {
						p := r.color[i : i+4 : i+4]
						sr += uint32(p[0])
						sg += uint32(p[1])
						sb += uint32(p[2])
						sa += uint32(p[3])
						i += 4
					}
				}
				if sa == 0 {
					continue
				}
				d := cnv.rgba.Pix[off : off+4 : off+4]
				if sa == 255*n {
					d[0], d[1], d[2], d[3] = uint8((sr+n/2)/n), uint8((sg+n/2)/n), uint8((sb+n/2)/n), 255
					continue
				}
				// Premultiplied sums over the pixel, as draw.Over
				a := (sa + n/2) / n
				ia := 255 - a
				d[0] = uint8(min(255, (sr+n/2)/n+(uint32(d[0])*ia+127)/255))
				d[1] = uint8(min(255, (sg+n/2)/n+(uint32(d[1])*ia+127)/255))
				d[2] = uint8(min(255, (sb+n/2)/n+(uint32(d[2])*ia+127)/255))
				d[3] = uint8(min(255, a+(uint32(d[3])*ia+127)/255))
			}
		}
	})
}

// drawLabels draws the texts of the scene at their points
func (r *Renderer3D) drawLabels(cnv *Canvas, x, y, w, h int, scene *Scene3D, cam Camera) {
	if len(scene.Labels) == 0 {
		return
	}
	cnv.Save()
	defer cnv.Restore()
	cnv.TranslateAndClip(x, y, w, h)
	if cnv.state.fontSize <= 0 {
		cnv.SetFontSize(12)
	}
	cnv.SetHAlign(HAlignCenter)
	cnv.SetVAlign(VAlignCenter)
	for _, l := range scene.Labels {
		lx, ly, _, ok := cam.Project(l.P, float64(w), float64(h))
		if !ok {
			continue
		}
		col := l.Color
		if col == (color.RGBA{}) {
			col = color.RGBA{128, 128, 128, 255}
		}
		cnv.SetColor(col)
		cnv.DrawText(int(math.Round(lx))-150, int(math.Round(ly))-15, 300, 30, l.Text)
	}
}
