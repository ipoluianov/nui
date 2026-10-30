package ui

import (
	"image"
	"image/color"
	"math"
	"time"
)

// View3D is a widget that shows a 3D scene and lets the user look around
// it with the mouse and the keyboard, as the viewers of CAD programs do:
//
//   - the left button turns the view around the target (orbit)
//   - the right or the middle button, or Shift with the left one, moves it
//     (pan)
//   - the wheel brings it nearer and farther, towards the pointer
//   - a click picks the object under the pointer (OnPick)
//   - a double click centers the view on the point under the pointer, or
//     shows the whole scene when there is nothing under it
//   - the arrows turn the view, with Shift move it; + and - zoom; Home or
//     R resets it; 1, 3 and 7 look from the front, the right and the top;
//     5 or O switches between perspective and orthographic
//
// While the view moves it is drawn without smoothing, to keep up with the
// mouse; it is smoothed when it stops.
//
//	scene := ui.NewScene3D()
//	scene.Add(ui.NewTorusMesh(1, 0.35, 48, 24), ui.Material{Color: orange, Specular: 0.5})
//	view := ui.NewView3D(scene)
//	view.OnPick.Connect(func(h ui.Hit3D) { fmt.Println(h.Object.Name) })
type View3D struct {
	Widget

	scene    *Scene3D
	renderer *Renderer3D

	// The orbit: the camera looks at target from distance, turned by yaw
	// around the vertical and raised by pitch
	target               Vec3
	distance, yaw, pitch float64
	fov                  float64
	ortho                bool
	home                 view3DOrbit
	homeSet              bool
	antialias            int
	interacting          bool
	idle                 *Timer
	autoRotate           float64
	autoTimer            *Timer
	lastTick             time.Time
	overlay              func(cnv *Canvas)
	background           color.RGBA
	controlsOff          bool
	drag                 view3DDrag
	pickOnHover          bool
	hover                *Object3D
	gizmo                bool
	gizmoEnds            [3]PointF // where the ends of the axes of the gizmo were drawn

	// The picture of the last paint, drawn again while nothing it shows
	// changed: the window repaints for the other widgets too
	cache    *image.RGBA
	cacheKey view3DCacheKey
	cacheSig uint64

	// OnPick: a click on an object of the scene (a click on nothing sends
	// nothing)
	OnPick Signal[Hit3D]
	// OnHover: the object under the pointer changed (nil - none), when
	// SetHoverPicking is on
	OnHover Signal[*Object3D]
	// OnViewChanged: the camera moved
	OnViewChanged Signal[Camera]
}

type view3DCacheKey struct {
	cam        Camera
	rect       image.Rectangle
	aa         int
	bg         color.RGBA
	fontSize   float64
	fontFamily string
}

type view3DOrbit struct {
	target               Vec3
	distance, yaw, pitch float64
}

type view3DDrag struct {
	active, moved  bool
	pan            bool
	button         MouseButton
	startX, startY int
	lastX, lastY   int
}

// NewView3D returns a view of the scene, which may be nil (SetScene);
// the view is fitted to the scene
func NewView3D(scene *Scene3D) *View3D {
	v := &View3D{}
	v.InitWidget()
	v.SetTypeName("View3D")
	v.SetMinSize(80, 60)
	v.SetCanBeFocused(true)
	v.SetXExpandable(true)
	v.SetYExpandable(true)
	v.renderer = NewRenderer3D()
	v.fov = math.Pi / 4
	v.distance, v.yaw, v.pitch = 5, math.Pi/6, math.Pi/7
	v.idle = v.NewTimer(180*time.Millisecond, func() {
		v.interacting = false
		v.Update()
	})
	v.idle.SetSingleShot(true)
	v.SetOnPaint(v.paint)
	v.SetOnMouseDown(v.mouseDown)
	v.SetOnMouseMove(v.mouseMove)
	v.SetOnMouseDblClick(v.mouseDblClick)
	v.SetOnKeyDown(v.keyDown)
	v.OnWheel.Connect(v.wheel)
	v.SetOnMouseLeave(func() {
		if v.pickOnHover && v.hover != nil {
			v.hover = nil
			v.OnHover.Emit(nil)
		}
	})
	v.SetScene(scene)
	return v
}

// Update repaints the view
func (v *View3D) Update() {
	v.Widget.Update()
}

// Scene returns the scene shown
func (v *View3D) Scene() *Scene3D { return v.scene }

// SetScene shows the scene and fits the view to it
func (v *View3D) SetScene(scene *Scene3D) {
	v.scene = scene
	v.homeSet = false
	v.FitToScene()
}

// Renderer returns the renderer of the view, e.g. for its Stats
func (v *View3D) Renderer() *Renderer3D { return v.renderer }

// SetAntialias sets the samples per pixel side of the still view (see
// Renderer3D.Antialias); zero is 2
func (v *View3D) SetAntialias(n int) {
	v.antialias = n
	v.Update()
}

// SetOrthographic switches to the parallel projection, or back to the
// perspective
func (v *View3D) SetOrthographic(ortho bool) {
	v.ortho = ortho
	v.viewChanged()
}

// Orthographic reports whether the projection is parallel
func (v *View3D) Orthographic() bool { return v.ortho }

// SetFieldOfView sets the vertical angle of view in radians
func (v *View3D) SetFieldOfView(fov float64) {
	if fov > 0 && fov < math.Pi {
		v.fov = fov
		v.viewChanged()
	}
}

// SetBackground sets the color behind the scene when the scene has none;
// transparent (the default) is the background of the theme
func (v *View3D) SetBackground(col color.RGBA) {
	v.background = col
	v.Update()
}

// SetControlsEnabled turns the mouse and the keyboard controls on and off;
// the view still picks on click
func (v *View3D) SetControlsEnabled(on bool) { v.controlsOff = !on }

// SetHoverPicking makes the view find the object under the pointer as it
// moves, and tell OnHover
func (v *View3D) SetHoverPicking(on bool) { v.pickOnHover = on }

// Hovered returns the object under the pointer, with SetHoverPicking on
func (v *View3D) Hovered() *Object3D { return v.hover }

// SetOverlay sets a function drawing over the scene in the coordinates of
// the widget: a legend, a scale, the name of what is selected
func (v *View3D) SetOverlay(f func(cnv *Canvas)) {
	v.overlay = f
	v.Update()
}

// SetAxesGizmo shows the axes of the scene in the lower left corner, turned
// as the view is: x red, y green, z blue. A click on the end of an axis
// looks along it; a second click, from the other side.
func (v *View3D) SetAxesGizmo(on bool) {
	v.gizmo = on
	v.Update()
}

// gizmoGeometry returns the center of the gizmo and the length of its axes
func (v *View3D) gizmoGeometry() (cx, cy, length float64) {
	return 42, float64(v.Height()) - 42, 28
}

func (v *View3D) paintGizmo(cnv *Canvas) {
	cx, cy, length := v.gizmoGeometry()
	view := v.Camera().ViewMatrix()
	type axis struct {
		dir   Vec3
		col   color.RGBA
		name  string
		depth float64
		end   PointF
	}
	axes := []axis{
		{dir: Vec3{1, 0, 0}, col: color.RGBA{220, 60, 60, 255}, name: "X"},
		{dir: Vec3{0, 1, 0}, col: color.RGBA{70, 170, 70, 255}, name: "Y"},
		{dir: Vec3{0, 0, 1}, col: color.RGBA{60, 110, 230, 255}, name: "Z"},
	}
	for i := range axes {
		d := view.MulDir(axes[i].dir)
		axes[i].end = PointF{cx + d.X*length, cy - d.Y*length}
		axes[i].depth = d.Z
		v.gizmoEnds[i] = axes[i].end
	}
	// The far ones first
	order := []int{0, 1, 2}
	for i := 1; i < 3; i++ {
		for k := i; k > 0 && axes[order[k]].depth < axes[order[k-1]].depth; k-- {
			order[k], order[k-1] = order[k-1], order[k]
		}
	}
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(10)
	cnv.SetFontBold(true)
	cnv.SetHAlign(HAlignCenter)
	cnv.SetVAlign(VAlignCenter)
	for _, i := range order {
		a := axes[i]
		line := NewPath()
		line.MoveTo(cx, cy)
		line.LineTo(a.end.X, a.end.Y)
		cnv.StrokePath(line, StrokeStyle{Width: 2, Cap: LineCapRound}, a.col)
		cnv.FillCircle(a.end.X, a.end.Y, 8, a.col)
		cnv.SetColor(color.RGBA{255, 255, 255, 255})
		cnv.DrawText(int(math.Round(a.end.X))-8, int(math.Round(a.end.Y))-8, 16, 16, a.name)
	}
}

// gizmoClick looks along the axis whose end of the gizmo is at the point;
// false if none is
func (v *View3D) gizmoClick(x, y int) bool {
	if !v.gizmo {
		return false
	}
	for i, e := range v.gizmoEnds {
		if math.Hypot(float64(x)+0.5-e.X, float64(y)+0.5-e.Y) > 9 {
			continue
		}
		yaw, pitch := v.yaw, v.pitch
		switch i {
		case 0:
			yaw, pitch = math.Pi/2, 0
		case 1:
			yaw, pitch = 0, clampPitch(math.Pi/2)
		case 2:
			yaw, pitch = 0, 0
		}
		// Already looking along it: from the other side
		if math.Abs(math.Remainder(v.yaw-yaw, 2*math.Pi)) < 1e-3 && math.Abs(v.pitch-pitch) < 1e-3 {
			if i == 1 {
				pitch = -pitch
			} else {
				yaw += math.Pi
			}
		}
		v.yaw, v.pitch = yaw, pitch
		v.viewChanged()
		return true
	}
	return false
}

// SetAutoRotate turns the view around the target by speed radians a
// second; zero stops it
func (v *View3D) SetAutoRotate(speed float64) {
	v.autoRotate = speed
	if speed == 0 {
		if v.autoTimer != nil {
			v.autoTimer.Stop()
		}
		return
	}
	if v.autoTimer == nil {
		v.autoTimer = v.NewTimer(16*time.Millisecond, v.autoStep)
	}
	v.lastTick = time.Now()
	v.autoTimer.Start()
}

func (v *View3D) autoStep() {
	now := time.Now()
	dt := now.Sub(v.lastTick).Seconds()
	v.lastTick = now
	if v.drag.active || dt > 0.5 {
		return
	}
	v.yaw += v.autoRotate * dt
	v.Update()
}

// Camera returns the camera of the view as it is
func (v *View3D) Camera() Camera {
	cp := math.Cos(v.pitch)
	dir := Vec3{cp * math.Sin(v.yaw), math.Sin(v.pitch), cp * math.Cos(v.yaw)}
	cam := NewCamera(v.target.Add(dir.Scale(v.distance)), v.target)
	cam.FOV = v.fov
	cam.Orthographic = v.ortho
	return cam
}

// SetView places the camera: looking at the target from the distance,
// turned by yaw around the vertical (0 - from +z, the front) and raised by
// pitch (math.Pi/2 - from the top), in radians
func (v *View3D) SetView(target Vec3, distance, yaw, pitch float64) {
	v.target = target
	v.distance = math.Max(distance, 1e-9)
	v.yaw = yaw
	v.pitch = clampPitch(pitch)
	v.viewChanged()
}

// View returns the target, the distance, the yaw and the pitch of the camera
func (v *View3D) View() (target Vec3, distance, yaw, pitch float64) {
	return v.target, v.distance, v.yaw, v.pitch
}

func clampPitch(p float64) float64 {
	const lim = math.Pi/2 - 1e-3
	return math.Max(-lim, math.Min(lim, p))
}

// FitToScene aims at the middle of the scene from far enough to see all of
// its objects and points (the lines too when there is nothing else),
// keeping the direction; the first fit is where ResetView goes back to
func (v *View3D) FitToScene() {
	if v.scene != nil {
		if lo, hi, ok := v.scene.fitBounds(); ok {
			v.target = lo.Add(hi).Scale(0.5)
			radius := math.Max(hi.Sub(lo).Length()/2, 1e-6)
			v.distance = radius / math.Sin(v.fov/2) * 1.05
		}
	}
	if !v.homeSet {
		v.home = view3DOrbit{v.target, v.distance, v.yaw, v.pitch}
		v.homeSet = true
	}
	v.viewChanged()
}

// ResetView goes back to the view of the first fit
func (v *View3D) ResetView() {
	if v.homeSet {
		v.target, v.distance, v.yaw, v.pitch = v.home.target, v.home.distance, v.home.yaw, v.home.pitch
	}
	v.viewChanged()
}

func (v *View3D) viewChanged() {
	v.OnViewChanged.Emit(v.Camera())
	v.Update()
}

// moving draws without smoothing until the view stops
func (v *View3D) moving() {
	v.interacting = true
	v.idle.Start()
}

// PickAt returns the object under the point of the widget
func (v *View3D) PickAt(x, y int) (Hit3D, bool) {
	if v.scene == nil {
		return Hit3D{}, false
	}
	ray := v.Camera().Ray(float64(x)+0.5, float64(y)+0.5, float64(v.Width()), float64(v.Height()))
	return v.scene.Pick(ray)
}

// Project returns where the point of the scene is in the widget
func (v *View3D) Project(p Vec3) (x, y float64, ok bool) {
	x, y, _, ok = v.Camera().Project(p, float64(v.Width()), float64(v.Height()))
	return
}

// RenderImage returns the view as a picture of the size, smoothed: to
// save or to copy
func (v *View3D) RenderImage(width, height int) *image.RGBA {
	if v.scene == nil {
		return image.NewRGBA(image.Rect(0, 0, width, height))
	}
	r := NewRenderer3D()
	r.Antialias = max(v.antialias, 2)
	return r.RenderImage(width, height, v.scene, v.Camera())
}

func (v *View3D) paint(cnv *Canvas) {
	w, h := v.Width(), v.Height()
	if v.scene == nil {
		return
	}
	scene := v.scene
	var bg color.RGBA
	if scene.Background.A == 0 {
		bg = v.background
		if bg.A == 0 {
			bg = colorToRGBA(v.BackgroundColor())
		}
	}
	aa := v.antialias
	if aa <= 0 {
		aa = 2
	}
	if v.interacting {
		aa = 1
	}
	v.renderer.Antialias = aa
	cam := v.Camera()
	// The picture of the last paint is right while the camera, the size,
	// the background, the font of the labels and the scene are the same,
	// and the view is not clipped or transformed; all its pixels are then
	// the view's own
	px, py, pw, ph := cnv.pixelRect(0, 0, w, h)
	rect := image.Rect(px, py, px+pw, py+ph)
	key := view3DCacheKey{cam: cam, rect: rect, aa: aa, bg: bg, fontSize: cnv.state.fontSize, fontFamily: cnv.state.fontFamily}
	sig, cacheable := scene.signature()
	cacheable = cacheable && !rect.Empty() && !cnv.state.hasXf && cnv.rec == nil &&
		rect.In(cnv.shapeClip().Intersect(cnv.rgba.Rect)) &&
		(scene.Background.A == 255 || scene.Background.A == 0 && bg.A == 255)
	if cacheable && v.cache != nil && key == v.cacheKey && sig == v.cacheSig {
		copyPixelsPart(cnv.rgba, rect, v.cache, cnv.clipRect())
	} else {
		if scene.Background.A == 0 {
			cnv.FillRect(0, 0, w, h, bg)
		}
		v.renderer.Render(cnv, 0, 0, w, h, scene, cam)
		switch {
		case !cacheable:
			v.cache = nil
		case rect.In(cnv.clipRect()):
			if v.cache == nil || v.cache.Rect.Size() != rect.Size() {
				v.cache = image.NewRGBA(image.Rect(0, 0, pw, ph))
			}
			copyPixelsOut(v.cache, cnv.rgba, rect)
			v.cacheKey, v.cacheSig = key, sig
		}
		// A frame that repaints only a part of the view leaves the cache:
		// the rest of the view on the canvas is the last frame
	}
	if v.overlay != nil {
		cnv.Save()
		v.overlay(cnv)
		cnv.Restore()
	}
	if v.gizmo {
		v.paintGizmo(cnv)
	}
}

// copyPixelsPart copies the picture to the rectangle r of the image, only
// its pixels within part
func copyPixelsPart(dst *image.RGBA, r image.Rectangle, src *image.RGBA, part image.Rectangle) {
	p := r.Intersect(part)
	for y := p.Min.Y; y < p.Max.Y; y++ {
		off := dst.PixOffset(p.Min.X, y)
		from := src.PixOffset(p.Min.X-r.Min.X, y-r.Min.Y)
		copy(dst.Pix[off:off+p.Dx()*4], src.Pix[from:from+p.Dx()*4])
	}
}

// copyPixels copies the picture to the rectangle of the image
func copyPixels(dst *image.RGBA, r image.Rectangle, src *image.RGBA) {
	for y := 0; y < r.Dy(); y++ {
		off := dst.PixOffset(r.Min.X, r.Min.Y+y)
		copy(dst.Pix[off:off+r.Dx()*4], src.Pix[y*src.Stride:y*src.Stride+r.Dx()*4])
	}
}

// copyPixelsOut copies the rectangle of the image to the picture
func copyPixelsOut(dst *image.RGBA, src *image.RGBA, r image.Rectangle) {
	for y := 0; y < r.Dy(); y++ {
		off := src.PixOffset(r.Min.X, r.Min.Y+y)
		copy(dst.Pix[y*dst.Stride:y*dst.Stride+r.Dx()*4], src.Pix[off:off+r.Dx()*4])
	}
}

// Invalidate draws the scene again at the next paint: after a change the
// view cannot see, as a mesh changed in place without Mesh.Changed
func (v *View3D) Invalidate() {
	v.cache = nil
	v.Update()
}

func (v *View3D) mouseDown(button MouseButton, x, y int, mods KeyModifiers) bool {
	v.Focus()
	v.drag = view3DDrag{active: true, button: button, startX: x, startY: y, lastX: x, lastY: y,
		pan: button == MouseButtonRight || button == MouseButtonMiddle || mods.Shift}
	return true
}

func (v *View3D) mouseMove(x, y int, mods KeyModifiers) bool {
	d := &v.drag
	if !d.active {
		if v.pickOnHover {
			var obj *Object3D
			if hit, ok := v.PickAt(x, y); ok {
				obj = hit.Object
			}
			if obj != v.hover {
				v.hover = obj
				v.OnHover.Emit(obj)
			}
		}
		return false
	}
	if !d.moved && abs(x-d.startX)+abs(y-d.startY) < 3 {
		return true
	}
	d.moved = true
	dx, dy := float64(x-d.lastX), float64(y-d.lastY)
	d.lastX, d.lastY = x, y
	if v.controlsOff {
		return true
	}
	if d.pan {
		v.panBy(dx, dy)
	} else {
		v.yaw -= dx * 0.008
		v.pitch = clampPitch(v.pitch + dy*0.008)
	}
	v.moving()
	v.viewChanged()
	return true
}

// panBy moves the target by the pixels on the screen
func (v *View3D) panBy(dx, dy float64) {
	h := float64(max(v.Height(), 1))
	// The units of the scene a pixel takes at the target
	k := 2 * v.distance * math.Tan(v.fov/2) / h
	cam := v.Camera()
	forward := cam.Target.Sub(cam.Position).Normalize()
	right := forward.Cross(Vec3{0, 1, 0}).Normalize()
	up := right.Cross(forward)
	v.target = v.target.Add(right.Scale(-dx * k)).Add(up.Scale(dy * k))
}

// ProcessMouseUp ends a drag of any button: the form sends the release of
// the left button only to the widget it was pressed on
func (v *View3D) ProcessMouseUp(button MouseButton, x int, y int, mods KeyModifiers, onlyForWidgetId string) bool {
	if v.drag.active && button == v.drag.button {
		moved := v.drag.moved
		v.drag.active = false
		if !moved && button == MouseButtonLeft && !v.gizmoClick(x, y) {
			if hit, ok := v.PickAt(x, y); ok {
				v.OnPick.Emit(hit)
			}
		}
	}
	return v.Widget.ProcessMouseUp(button, x, y, mods, onlyForWidgetId)
}

func (v *View3D) mouseDblClick(button MouseButton, x, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft || v.controlsOff {
		return false
	}
	if hit, ok := v.PickAt(x, y); ok {
		// Centered on the point, as near as it was
		v.target = hit.Point
	} else {
		v.FitToScene()
	}
	v.viewChanged()
	return true
}

func (v *View3D) wheel(e *WheelEvent) {
	if v.controlsOff || e.DY == 0 {
		return
	}
	e.Handled = true
	k := math.Pow(0.85, float64(e.DY))
	// Towards the point under the pointer: what is under it stays there
	w, h := float64(v.Width()), float64(v.Height())
	if w > 0 && h > 0 {
		px, py := float64(e.X)-w/2, float64(e.Y)-h/2
		v.panBy(-px*(1-k), -py*(1-k))
	}
	v.distance = math.Max(v.distance*k, 1e-6)
	v.moving()
	v.viewChanged()
}

func (v *View3D) keyDown(key Key, mods KeyModifiers) bool {
	if v.controlsOff {
		return false
	}
	const step = math.Pi / 36
	switch key {
	case KeyArrowLeft, KeyArrowRight, KeyArrowUp, KeyArrowDown:
		dx, dy := 0.0, 0.0
		switch key {
		case KeyArrowLeft:
			dx = -1
		case KeyArrowRight:
			dx = 1
		case KeyArrowUp:
			dy = -1
		case KeyArrowDown:
			dy = 1
		}
		if mods.Shift {
			v.panBy(dx*20, dy*20)
		} else {
			v.yaw -= dx * step
			v.pitch = clampPitch(v.pitch - dy*step)
		}
	case KeyEqual, KeyNumpadPlus, KeyPageUp:
		v.distance *= 0.85
	case KeyMinus, KeyNumpadMinus, KeyPageDown:
		v.distance /= 0.85
	case KeyHome, KeyR:
		v.ResetView()
		return true
	case Key1, KeyNumpad1:
		v.yaw, v.pitch = 0, 0
	case Key3, KeyNumpad3:
		v.yaw, v.pitch = math.Pi/2, 0
	case Key7, KeyNumpad7:
		v.yaw, v.pitch = 0, clampPitch(math.Pi/2)
	case Key5, KeyNumpad5, KeyO:
		v.ortho = !v.ortho
	default:
		return false
	}
	v.moving()
	v.viewChanged()
	return true
}

// xmlObject returns the object the attributes of a <view3d> element set,
// made in the scene when first needed
func (v *View3D) xmlObject() *Object3D {
	if v.scene == nil {
		v.scene = NewScene3D()
	}
	for _, o := range v.scene.Objects {
		if o.Name == "view3d" {
			return o
		}
	}
	o := v.scene.Add(nil, Material{Specular: 0.3})
	o.Name = "view3d"
	return o
}

// setXMLMesh shows the mesh of a <view3d> element
func (v *View3D) setXMLMesh(m *Mesh) {
	v.xmlObject().Mesh = m
	v.homeSet = false
	v.FitToScene()
}
