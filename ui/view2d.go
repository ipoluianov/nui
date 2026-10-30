package ui

import (
	"image"
	"image/color"
	"math"
)

// View2D is a widget that shows a drawing of any size in its own
// coordinates - millimeters, meters, the units of a map - and lets the user
// move around it, as the views of CAD programs do:
//
//   - the wheel zooms towards the pointer: what is under it stays there;
//   - the middle button drags the drawing (the left one too, with
//     SetPanWithLeftButton); a double click of the middle button shows all
//     of it (ZoomExtents);
//   - a drag of the left button on what the application does not take
//     selects a box (SetBoxSelection): to the right it selects what is
//     inside, to the left what it crosses;
//   - the arrows move, + and - zoom, Home shows everything, Esc ends a box.
//
// The application draws in SetPainter, in the drawing's coordinates: the
// canvas is transformed to them (y goes up by default, SetYUp). Lines are
// drawn with StrokeStyle.NonScaling to stay as thick at any zoom, text with
// AddText. The picture is kept and drawn again while the view does not
// change: the pointer, the rubber band of a tool, the highlight of what is
// under the mouse go in SetOverlay, drawn over it on every paint; after a
// change of the drawing call Invalidate. While the drawing is dragged, only
// the strip that comes into view is painted.
//
//	view := ui.NewView2D()
//	view.SetPainter(func(p *ui.View2DPaint) {
//		line := ui.StrokeStyle{Width: 1, NonScaling: true}
//		index.Search(p.Visible, func(s *Shape) bool {
//			p.Canvas.StrokePath(s.Path, line, s.Color)
//			return true
//		})
//	})
//	view.SetExtents(drawingBounds)
//	view.ZoomExtents()
type View2D struct {
	Widget

	// The view: the point of the drawing at the center of the widget and
	// the pixels a unit takes
	cx, cy, scale      float64
	yUp                bool
	minScale, maxScale float64
	extents            RectF
	hasExtents         bool

	painter, overlay func(p *View2DPaint)
	background       color.RGBA
	grid             View2DGrid
	axes             bool
	crosshair        Crosshair2D
	pickBox          int
	boxSelect        bool
	panLeft          bool
	controlsOff      bool

	// The pointer, in the widget; in is false when it is out of it
	pointerX, pointerY float64
	pointerIn          bool

	drag       view2DDrag
	zoomWindow bool

	// The picture of the drawing of the last paint, and what it was of
	cache    *image.RGBA
	cacheKey view2DCacheKey
	cacheGen uint64
	gen      uint64

	// OnViewChanged: the view moved or zoomed; the part of the drawing seen
	OnViewChanged Signal[RectF]
	// OnPointerMoved: the pointer moved over the view; the point of the
	// drawing under it
	OnPointerMoved Signal[PointF]
	// OnBoxSelected: a box was dragged with SetBoxSelection on
	OnBoxSelected Signal[BoxSelection]
}

// View2DPaint is what the painter and the overlay of a View2D draw with
type View2DPaint struct {
	// Canvas is transformed to the drawing's coordinates
	Canvas *Canvas
	View   *View2D
	// Visible is the part of the drawing to draw: what is out of it is
	// not seen (it may be only a strip of the view, see View2D)
	Visible RectF
	// Scale is the pixels a unit of the drawing takes
	Scale float64

	screenXf Matrix
}

// PixelSize returns the length of the drawing a pixel takes: to size what
// is to look the same at any zoom (a marker, a gap, a tolerance)
func (p *View2DPaint) PixelSize() float64 { return 1 / p.Scale }

// ToScreen converts a point of the drawing to the widget's coordinates
func (p *View2DPaint) ToScreen(pt PointF) PointF { return p.View.ToScreen(pt) }

// InScreen calls f with the canvas in the widget's coordinates, without the
// transform of the drawing: to draw what keeps its size in pixels (snap
// markers, handles, the labels of the overlay)
func (p *View2DPaint) InScreen(f func(cnv *Canvas)) {
	p.Canvas.Save()
	p.Canvas.SetTransform(p.screenXf)
	f(p.Canvas)
	p.Canvas.Restore()
}

// screenXf is the transform of the canvas to the widget, set for InScreen
func (p *View2DPaint) setScreen(m Matrix) { p.screenXf = m }

// View2DGrid is the grid of a View2D
type View2DGrid struct {
	Visible bool
	// Step is the distance between the lines in units of the drawing;
	// zero picks one of 1, 2, 5 times a power of ten as the zoom changes,
	// to keep the lines apart
	Step float64
	// Major is the count of steps between the stronger lines; zero is 5
	// (and 2 or 5 with an automatic step)
	Major int
	// Dots draws the crossings of the lines only
	Dots bool
	// The colors; zero picks them from the background
	Color, MajorColor color.RGBA
}

// Crosshair2D is how a View2D shows the pointer
type Crosshair2D int

const (
	// CrosshairNone: the system's arrow
	CrosshairNone Crosshair2D = iota
	// CrosshairSmall: a cross of the system
	CrosshairSmall
	// CrosshairFull: lines across all the view, with a pick box at the
	// pointer (SetPickBox), as AutoCAD draws it
	CrosshairFull
)

// BoxSelection is a box dragged over a View2D
type BoxSelection struct {
	// Rect is the box in the drawing's coordinates
	Rect RectF
	// Crossing is true for a box dragged to the left: what it crosses is
	// selected, not only what is inside it
	Crossing bool
	Mods     KeyModifiers
}

type view2DDrag struct {
	active, moved  bool
	pan, box       bool
	button         MouseButton
	startX, startY int
	lastX, lastY   int
	mods           KeyModifiers
}

type view2DCacheKey struct {
	cx, cy, scale float64
	yUp           bool
	rect          image.Rectangle
	bg            color.RGBA
	grid          View2DGrid
	axes          bool
}

// NewView2D returns a view at the scale 1 with the origin in the middle,
// y up
func NewView2D() *View2D {
	v := &View2D{scale: 1, yUp: true, minScale: 1e-9, maxScale: 1e9, pickBox: 5, axes: true}
	v.grid = View2DGrid{Visible: true}
	v.InitWidget()
	v.SetTypeName("View2D")
	v.SetMinSize(80, 60)
	v.SetCanBeFocused(true)
	v.SetXExpandable(true)
	v.SetYExpandable(true)
	v.SetOnPaint(v.paint)
	v.SetOnMouseDown(v.mouseDown)
	v.SetOnMouseMove(v.mouseMove)
	v.SetOnMouseDblClick(v.mouseDblClick)
	v.SetOnMouseLeave(func() {
		v.pointerIn = false
		v.Update()
	})
	v.SetOnKeyDown(v.keyDown)
	v.OnWheel.Connect(v.wheel)
	return v
}

// Update repaints the view, the picture of the drawing kept: after a change
// of the overlay
func (v *View2D) Update() {
	if v.form != nil {
		v.form.Update()
	}
}

// Invalidate paints the drawing again: after a change of what the painter
// draws
func (v *View2D) Invalidate() {
	v.gen++
	v.Update()
}

// SetPainter sets the function that draws the drawing
func (v *View2D) SetPainter(f func(p *View2DPaint)) {
	v.painter = f
	v.Invalidate()
}

// SetOverlay sets the function that draws over the drawing on every paint:
// what follows the pointer
func (v *View2D) SetOverlay(f func(p *View2DPaint)) {
	v.overlay = f
	v.Update()
}

// SetYUp makes y go up, as in mathematics and in CAD (the default), or
// down, as on the screen and in images
func (v *View2D) SetYUp(up bool) {
	v.yUp = up
	v.viewChanged()
}

// YUp reports whether y goes up
func (v *View2D) YUp() bool { return v.yUp }

// SetBackground sets the color under the drawing; transparent (the
// default) is the background of the theme
func (v *View2D) SetBackground(col color.RGBA) {
	v.background = col
	v.Update()
}

// SetGrid sets the grid under the drawing
func (v *View2D) SetGrid(g View2DGrid) {
	v.grid = g
	v.Update()
}

// Grid returns the grid
func (v *View2D) Grid() View2DGrid { return v.grid }

// SetAxesVisible shows or hides the axes through the origin
func (v *View2D) SetAxesVisible(on bool) {
	v.axes = on
	v.Update()
}

// SetCrosshair sets how the pointer is shown
func (v *View2D) SetCrosshair(c Crosshair2D) {
	v.crosshair = c
	if c == CrosshairNone {
		v.SetMouseCursor(MouseCursorNotDefined)
	} else {
		v.SetMouseCursor(MouseCursorCross)
	}
	v.Update()
}

// SetPickBox sets the half size in pixels of the box at the pointer of
// CrosshairFull: what the box touches is what a click picks
func (v *View2D) SetPickBox(halfSize int) {
	v.pickBox = halfSize
	v.Update()
}

// PickBox returns the half size of the pick box in pixels
func (v *View2D) PickBox() int { return v.pickBox }

// SetBoxSelection lets the left button drag a selection box (OnBoxSelected)
// where the application does not take the press
func (v *View2D) SetBoxSelection(on bool) { v.boxSelect = on }

// SetPanWithLeftButton lets the left button drag the drawing too
func (v *View2D) SetPanWithLeftButton(on bool) { v.panLeft = on }

// SetControlsEnabled turns the moving and the zooming by the user on and
// off
func (v *View2D) SetControlsEnabled(on bool) { v.controlsOff = !on }

// SetScaleRange sets the smallest and the largest zoom, in pixels a unit
func (v *View2D) SetScaleRange(minScale, maxScale float64) {
	v.minScale, v.maxScale = minScale, maxScale
	v.SetScale(v.scale)
}

// Scale returns the pixels a unit of the drawing takes
func (v *View2D) Scale() float64 { return v.scale }

// SetScale zooms around the center of the view
func (v *View2D) SetScale(scale float64) {
	v.scale = v.clampScale(scale)
	v.viewChanged()
}

func (v *View2D) clampScale(s float64) float64 {
	if math.IsNaN(s) || s <= 0 {
		return v.scale
	}
	return math.Max(v.minScale, math.Min(v.maxScale, s))
}

// Center returns the point of the drawing at the center of the view
func (v *View2D) Center() PointF { return PointF{v.cx, v.cy} }

// SetCenter moves the point of the drawing to the center of the view
func (v *View2D) SetCenter(p PointF) {
	v.cx, v.cy = p.X, p.Y
	v.viewChanged()
}

// SetView sets the center and the scale at once
func (v *View2D) SetView(center PointF, scale float64) {
	v.cx, v.cy = center.X, center.Y
	v.scale = v.clampScale(scale)
	v.viewChanged()
}

// ZoomAt zooms by the factor keeping the point of the widget where it is
func (v *View2D) ZoomAt(factor, x, y float64) {
	before := v.ToWorld(x, y)
	v.scale = v.clampScale(v.scale * factor)
	after := v.ToWorld(x, y)
	v.cx += before.X - after.X
	v.cy += before.Y - after.Y
	v.viewChanged()
}

// ZoomToRect shows the rectangle of the drawing as large as it fits, with
// a margin of 5% around
func (v *View2D) ZoomToRect(r RectF) {
	if r.Empty() {
		return
	}
	w, h := float64(max(v.Width(), 1)), float64(max(v.Height(), 1))
	rw, rh := math.Max(r.Width(), 1e-12), math.Max(r.Height(), 1e-12)
	c := r.Center()
	v.cx, v.cy = c.X, c.Y
	v.scale = v.clampScale(math.Min(w/rw, h/rh) / 1.1)
	v.viewChanged()
}

// SetExtents sets the box of all the drawing, which ZoomExtents shows
func (v *View2D) SetExtents(r RectF) {
	v.extents, v.hasExtents = r, !r.Empty()
}

// ZoomExtents shows all the drawing (SetExtents)
func (v *View2D) ZoomExtents() {
	if v.hasExtents {
		v.ZoomToRect(v.extents)
	}
}

// StartZoomWindow makes the next drag of the left button a box to zoom to
func (v *View2D) StartZoomWindow() {
	v.zoomWindow = true
	v.Focus()
}

// WorldMatrix returns the transform from the drawing to the widget
func (v *View2D) WorldMatrix() Matrix {
	return v.frame().matrix()
}

// ToScreen converts a point of the drawing to the widget's coordinates
func (v *View2D) ToScreen(p PointF) PointF {
	return v.frame().toScreen(p)
}

// ToWorld converts a point of the widget to the drawing's coordinates
func (v *View2D) ToWorld(x, y float64) PointF {
	return v.frame().toWorld(x, y)
}

// VisibleRect returns the part of the drawing in the view
func (v *View2D) VisibleRect() RectF {
	return v.frame().worldRect(0, 0, float64(v.Width()), float64(v.Height()))
}

// worldRect returns the rectangle of the drawing under the one of the widget
func (v *View2D) worldRect(x0, y0, x1, y1 float64) RectF {
	return v.frame().worldRect(x0, y0, x1, y1)
}

// view2DFrame is the view in a picture of a size: of the widget, or of
// RenderImage
type view2DFrame struct {
	cx, cy, scale, w, h float64
	yUp                 bool
}

func (v *View2D) frame() view2DFrame {
	return v.frameSized(v.Width(), v.Height())
}

func (v *View2D) frameSized(w, h int) view2DFrame {
	return view2DFrame{v.cx, v.cy, v.scale, float64(w), float64(h), v.yUp}
}

func (f view2DFrame) matrix() Matrix {
	s := f.scale
	if f.yUp {
		return Matrix{A: s, D: -s, E: f.w/2 - f.cx*s, F: f.h/2 + f.cy*s}
	}
	return Matrix{A: s, D: s, E: f.w/2 - f.cx*s, F: f.h/2 - f.cy*s}
}

func (f view2DFrame) toScreen(p PointF) PointF {
	x := f.w/2 + (p.X-f.cx)*f.scale
	if f.yUp {
		return PointF{x, f.h/2 - (p.Y-f.cy)*f.scale}
	}
	return PointF{x, f.h/2 + (p.Y-f.cy)*f.scale}
}

func (f view2DFrame) toWorld(x, y float64) PointF {
	wx := f.cx + (x-f.w/2)/f.scale
	if f.yUp {
		return PointF{wx, f.cy - (y-f.h/2)/f.scale}
	}
	return PointF{wx, f.cy + (y-f.h/2)/f.scale}
}

func (f view2DFrame) worldRect(x0, y0, x1, y1 float64) RectF {
	return RectFromPoints(f.toWorld(x0, y0), f.toWorld(x1, y1))
}

// Pointer returns the point of the drawing under the pointer; false if the
// pointer is not over the view
func (v *View2D) Pointer() (PointF, bool) {
	return v.ToWorld(v.pointerX, v.pointerY), v.pointerIn
}

// PointerScreen returns the pointer in the widget's coordinates
func (v *View2D) PointerScreen() (PointF, bool) {
	return PointF{v.pointerX, v.pointerY}, v.pointerIn
}

// AddText adds the outlines of the text to the path at the point of the
// drawing, letters height units tall, turned by angle (radians,
// counterclockwise with y up), aligned to the point as Path.TextAligned:
// the text of the drawing, which reads upright whichever way y goes
func (v *View2D) AddText(p *Path, at PointF, text, fontFamily string, height, angle float64, hAlign HAlign, vAlign VAlign) {
	q := pathPool.Get().(*Path)
	defer pathPool.Put(q)
	q.Reset()
	q.TextAligned(0, 0, text, fontFamily, height, hAlign, vAlign)
	m := IdentityMatrix()
	if v.yUp {
		m = ScalingMatrix(1, -1)
	}
	q.Transform(m.Then(RotationMatrix(angle)).Then(TranslationMatrix(at.X, at.Y)))
	p.AddPath(q)
}

// RenderImage paints the drawing on its background into a picture of the
// size, around the center of the view at its scale, at the density of the
// pixels (2: twice as dense, for a print or a sharp copy). The grid, the
// axes and the overlay are left out.
func (v *View2D) RenderImage(width, height int, pixelScale float64) *image.RGBA {
	cnv, img := NewImageCanvas(width, height, pixelScale)
	v.paintContent(cnv, width, height, image.Rectangle{}, false)
	return img
}

// RenderSVG paints the drawing as RenderImage does, written as SVG: the
// lines, the curves and the text (of AddText) as curves, sharp at any size
func (v *View2D) RenderSVG(width, height int) string {
	cnv, _ := NewImageCanvas(width, height, 1)
	rec := NewSVGRecorder(float64(width), float64(height))
	cnv.SetRecorder(rec)
	v.paintContent(cnv, width, height, image.Rectangle{}, false)
	return rec.String()
}

func (v *View2D) viewChanged() {
	v.OnViewChanged.Emit(v.VisibleRect())
	v.Update()
}

func (v *View2D) backgroundColor() color.RGBA {
	if v.background.A != 0 {
		return v.background
	}
	return colorToRGBA(v.BackgroundColor())
}

func (v *View2D) paint(cnv *Canvas) {
	w, h := v.Width(), v.Height()
	if w <= 0 || h <= 0 {
		return
	}
	px, py, pw, ph := cnv.pixelRect(0, 0, w, h)
	rect := image.Rect(px, py, px+pw, py+ph)
	key := view2DCacheKey{cx: v.cx, cy: v.cy, scale: v.scale, yUp: v.yUp, rect: rect, bg: v.backgroundColor(), grid: v.grid, axes: v.axes}
	cacheable := !rect.Empty() && !cnv.state.hasXf && cnv.rec == nil && rect.In(cnv.clipRect().Intersect(cnv.rgba.Rect)) && key.bg.A == 255
	switch {
	case cacheable && v.cache != nil && v.cacheGen == v.gen && key == v.cacheKey:
		copyPixels(cnv.rgba, rect, v.cache)
	case cacheable && v.cache != nil && v.cacheGen == v.gen && v.scrolled(key, cnv):
		// Moved by whole pixels: the picture moved, the strips that came
		// into view painted
	default:
		v.paintContent(cnv, w, h, image.Rectangle{}, true)
		if cacheable {
			if v.cache == nil || v.cache.Rect.Size() != rect.Size() {
				v.cache = image.NewRGBA(image.Rect(0, 0, pw, ph))
			}
			copyPixelsOut(v.cache, cnv.rgba, rect)
			v.cacheKey, v.cacheGen = key, v.gen
		} else {
			v.cache = nil
		}
	}
	v.paintOverlay(cnv, w, h)
}

// scrolled paints the view from the picture of the last paint moved, when
// only the center changed, by whole pixels; false if it can't
func (v *View2D) scrolled(key view2DCacheKey, cnv *Canvas) bool {
	old := v.cacheKey
	if old.scale != key.scale || old.yUp != key.yUp || old.rect != key.rect || old.bg != key.bg || old.grid != key.grid || old.axes != key.axes {
		return false
	}
	// The move in pixels of the image
	k := key.scale * cnv.scale
	dx := (old.cx - key.cx) * k
	dy := (key.cy - old.cy) * k
	if !key.yUp {
		dy = -dy
	}
	idx, idy := math.Round(dx), math.Round(dy)
	if math.Abs(dx-idx) > 1e-6 || math.Abs(dy-idy) > 1e-6 {
		return false
	}
	ix, iy := int(idx), int(idy)
	rect := key.rect
	pw, ph := rect.Dx(), rect.Dy()
	if abs(ix) >= pw/2 || abs(iy) >= ph/2 {
		return false
	}
	// Move the picture in place
	scrollPixels(v.cache, ix, iy)
	copyPixels(cnv.rgba, rect, v.cache)
	// The strips: in logical pixels, a pixel wider for the rounding
	s := cnv.scale
	lw, lh := v.Width(), v.Height()
	var strips []image.Rectangle
	if ix > 0 {
		strips = append(strips, image.Rect(0, 0, int(math.Ceil(float64(ix)/s))+1, lh))
	} else if ix < 0 {
		strips = append(strips, image.Rect(lw-int(math.Ceil(float64(-ix)/s))-1, 0, lw, lh))
	}
	if iy > 0 {
		strips = append(strips, image.Rect(0, 0, lw, int(math.Ceil(float64(iy)/s))+1))
	} else if iy < 0 {
		strips = append(strips, image.Rect(0, lh-int(math.Ceil(float64(-iy)/s))-1, lw, lh))
	}
	for _, st := range strips {
		v.paintContent(cnv, lw, lh, st, true)
	}
	copyPixelsOut(v.cache, cnv.rgba, rect)
	v.cacheKey = key
	return true
}

// scrollPixels moves the picture by (dx, dy) pixels; what comes in is left
// as it was
func scrollPixels(img *image.RGBA, dx, dy int) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	row := func(y int) []byte { return img.Pix[y*img.Stride : y*img.Stride+w*4] }
	move := func(y int) {
		src := row(y - dy)
		dst := row(y)
		if dx >= 0 {
			copy(dst[dx*4:], src[:(w-dx)*4])
		} else {
			copy(dst[:(w+dx)*4], src[-dx*4:])
		}
	}
	if dy > 0 {
		for y := h - 1; y >= dy; y-- {
			move(y)
		}
	} else {
		for y := 0; y < h+dy; y++ {
			move(y)
		}
	}
}

// paintContent paints the background, the grid and the axes (with decor)
// and the drawing of the view of the size into the canvas, in the
// rectangle of the widget (all of it when empty)
func (v *View2D) paintContent(cnv *Canvas, w, h int, part image.Rectangle, decor bool) {
	if part.Empty() {
		part = image.Rect(0, 0, w, h)
	}
	cnv.Save()
	defer cnv.Restore()
	// The clip on the part, the origin kept: a strip is painted exactly as
	// the same pixels of the whole
	cnv.intersectClip(part.Min.X, part.Min.Y, part.Dx(), part.Dy())
	f := v.frameSized(w, h)
	bg := v.backgroundColor()
	cnv.FillRect(0, 0, w, h, bg)
	visible := f.worldRect(float64(part.Min.X), float64(part.Min.Y), float64(part.Max.X), float64(part.Max.Y))
	if v.grid.Visible && decor {
		v.paintGrid(cnv, f, visible, bg)
	}
	if v.axes && decor {
		v.paintAxes(cnv, f, bg)
	}
	if v.painter != nil {
		cnv.SetTransform(f.matrix())
		p := &View2DPaint{Canvas: cnv, View: v, Scale: v.scale, Visible: visible}
		p.setScreen(IdentityMatrix())
		v.painter(p)
	}
}

// intersectClip narrows the clip to the logical rectangle, the origin kept
func (c *Canvas) intersectClip(x, y, w, h int) {
	px, py, pw, ph := c.pixelRect(x, y, w, h)
	r := image.Rect(px, py, px+pw, py+ph).Intersect(image.Rect(c.state.clipX, c.state.clipY, c.state.clipX+c.state.clipW, c.state.clipY+c.state.clipH))
	c.state.clipX, c.state.clipY, c.state.clipW, c.state.clipH = r.Min.X, r.Min.Y, r.Dx(), r.Dy()
}

// gridStep returns the step of the grid and the count of steps between
// the major lines
func (v *View2D) gridStep() (float64, int) {
	step, major := v.grid.Step, v.grid.Major
	if step > 0 {
		if major <= 0 {
			major = 5
		}
		return step, major
	}
	// 1, 2 or 5 times a power of ten, at least 12 pixels apart
	minStep := 12 / v.scale
	p := math.Pow(10, math.Floor(math.Log10(minStep)))
	for _, m := range []float64{1, 2, 5, 10} {
		if m*p >= minStep {
			step = m * p
			if major <= 0 {
				if m == 5 {
					major = 2
				} else {
					major = 5
				}
			}
			break
		}
	}
	return step, major
}

func (v *View2D) paintGrid(cnv *Canvas, f view2DFrame, r RectF, bg color.RGBA) {
	w, h := int(f.w), int(f.h)
	step, major := v.gridStep()
	if !(step > 0) || step*v.scale < 4 {
		return
	}
	minor, strong := v.grid.Color, v.grid.MajorColor
	if minor.A == 0 {
		minor = mixRGBA(bg, contrastInk(bg), 0.07)
	}
	if strong.A == 0 {
		strong = mixRGBA(bg, contrastInk(bg), 0.16)
	}
	i0, i1 := math.Floor(r.MinX/step), math.Ceil(r.MaxX/step)
	j0, j1 := math.Floor(r.MinY/step), math.Ceil(r.MaxY/step)
	if (i1-i0)*(j1-j0) > 1e6 && v.grid.Dots || i1-i0 > 1e4 || j1-j0 > 1e4 {
		return
	}
	isMajor := func(i float64) bool { return math.Mod(math.Abs(i), float64(major)) == 0 }
	if v.grid.Dots {
		for i := i0; i <= i1; i++ {
			for j := j0; j <= j1; j++ {
				col := minor
				if isMajor(i) && isMajor(j) {
					col = strong
				}
				s := f.toScreen(PointF{i * step, j * step})
				cnv.FillRect(snapLine(s.X), snapLine(s.Y), 1, 1, col)
			}
		}
		return
	}
	for pass := 0; pass < 2; pass++ {
		for i := i0; i <= i1; i++ {
			if isMajor(i) != (pass == 1) {
				continue
			}
			col := minor
			if pass == 1 {
				col = strong
			}
			x := f.toScreen(PointF{i * step, 0}).X
			cnv.FillRect(snapLine(x), 0, 1, h, col)
		}
		for j := j0; j <= j1; j++ {
			if isMajor(j) != (pass == 1) {
				continue
			}
			col := minor
			if pass == 1 {
				col = strong
			}
			y := f.toScreen(PointF{0, j * step}).Y
			cnv.FillRect(0, snapLine(y), w, 1, col)
		}
	}
}

// snapLine returns the pixel of a line of the grid or an axis at the
// coordinate. The coordinate of a line that should fall on a pixel edge
// comes out a hair below it or not by the rounding (fused multiply-add on
// arm64), and the line would jump a pixel between the paints the scroll
// joins
func snapLine(v float64) int { return int(math.Floor(v + 1e-6)) }

// paintAxes draws the axes through the origin: x red, y green, faint
func (v *View2D) paintAxes(cnv *Canvas, f view2DFrame, bg color.RGBA) {
	o := f.toScreen(PointF{})
	if o.Y >= -1 && o.Y <= f.h+1 {
		cnv.FillRect(0, snapLine(o.Y), int(f.w), 1, mixRGBA(bg, color.RGBA{220, 50, 47, 255}, 0.55))
	}
	if o.X >= -1 && o.X <= f.w+1 {
		cnv.FillRect(snapLine(o.X), 0, 1, int(f.h), mixRGBA(bg, color.RGBA{60, 170, 60, 255}, 0.55))
	}
}

func (v *View2D) paintOverlay(cnv *Canvas, w, h int) {
	cnv.Save()
	defer cnv.Restore()
	if v.overlay != nil {
		cnv.Save()
		cnv.SetTransform(v.WorldMatrix())
		p := &View2DPaint{Canvas: cnv, View: v, Scale: v.scale, Visible: v.VisibleRect()}
		p.setScreen(IdentityMatrix())
		v.overlay(p)
		cnv.Restore()
	}
	ink := contrastInk(v.backgroundColor())
	// The box being dragged: blue and solid to the right (inside), green
	// and dashed to the left (crossing)
	if d := v.drag; d.active && d.box && d.moved {
		x0, y0 := float64(min(d.startX, d.lastX)), float64(min(d.startY, d.lastY))
		x1, y1 := float64(max(d.startX, d.lastX)), float64(max(d.startY, d.lastY))
		box := pathPool.Get().(*Path)
		box.Reset()
		box.Rect(x0+0.5, y0+0.5, x1-x0, y1-y0)
		fill, line := color.RGBA{60, 120, 220, 40}, color.RGBA{60, 120, 220, 230}
		style := StrokeStyle{Width: 1}
		if d.lastX < d.startX && !v.zoomWindow {
			fill, line = color.RGBA{60, 180, 90, 40}, color.RGBA{60, 180, 90, 230}
			style.Dash = []float64{5, 3}
		}
		cnv.FillPath(box, fill)
		cnv.StrokePath(box, style, line)
		pathPool.Put(box)
	}
	if v.crosshair == CrosshairFull && v.pointerIn && !v.drag.pan {
		x, y := int(math.Floor(v.pointerX)), int(math.Floor(v.pointerY))
		col := mixRGBA(v.backgroundColor(), ink, 0.75)
		b := v.pickBox
		cnv.FillRect(0, y, max(0, x-b), 1, col)
		cnv.FillRect(x+b+1, y, max(0, w-x-b-1), 1, col)
		cnv.FillRect(x, 0, 1, max(0, y-b), col)
		cnv.FillRect(x, y+b+1, 1, max(0, h-y-b-1), col)
		if b > 0 {
			cnv.SetColor(col)
			cnv.DrawRect(x-b, y-b, 2*b+1, 2*b+1)
		}
	}
}

// contrastInk returns black on a light color, white on a dark one
func contrastInk(bg color.RGBA) color.RGBA {
	if int(bg.R)*299+int(bg.G)*587+int(bg.B)*114 > 128000 {
		return color.RGBA{0, 0, 0, 255}
	}
	return color.RGBA{255, 255, 255, 255}
}

func (v *View2D) mouseDown(button MouseButton, x, y int, mods KeyModifiers) bool {
	v.Focus()
	d := view2DDrag{active: true, button: button, startX: x, startY: y, lastX: x, lastY: y, mods: mods}
	switch {
	case v.controlsOff:
		return false
	case button == MouseButtonMiddle || button == MouseButtonLeft && v.panLeft && !v.zoomWindow:
		d.pan = true
	case button == MouseButtonLeft && (v.boxSelect || v.zoomWindow):
		d.box = true
	default:
		return false
	}
	v.drag = d
	return true
}

func (v *View2D) mouseMove(x, y int, mods KeyModifiers) bool {
	v.pointerX, v.pointerY, v.pointerIn = float64(x)+0.5, float64(y)+0.5, true
	v.OnPointerMoved.Emit(v.ToWorld(v.pointerX, v.pointerY))
	d := &v.drag
	if !d.active {
		v.Update()
		return false
	}
	if !d.moved && abs(x-d.startX)+abs(y-d.startY) < 3 {
		return true
	}
	if !d.moved && d.pan {
		v.SetMouseCursor(MouseCursorMove)
	}
	d.moved = true
	dx, dy := x-d.lastX, y-d.lastY
	d.lastX, d.lastY = x, y
	if d.pan {
		v.panPixels(float64(dx), float64(dy))
		return true
	}
	v.Update()
	return true
}

// panPixels moves the drawing by the pixels of the widget
func (v *View2D) panPixels(dx, dy float64) {
	v.cx -= dx / v.scale
	if v.yUp {
		v.cy += dy / v.scale
	} else {
		v.cy -= dy / v.scale
	}
	v.viewChanged()
}

// ProcessMouseUp ends a drag of any button: the form sends the release of
// the left button only to the widget it was pressed on
func (v *View2D) ProcessMouseUp(button MouseButton, x int, y int, mods KeyModifiers, onlyForWidgetId string) bool {
	if d := v.drag; d.active && button == d.button {
		v.drag.active = false
		if d.pan && d.moved {
			v.SetCrosshair(v.crosshair)
		}
		if d.box && d.moved {
			r := v.worldRect(float64(d.startX), float64(d.startY), float64(x), float64(y))
			if v.zoomWindow {
				v.zoomWindow = false
				v.ZoomToRect(r)
			} else {
				v.OnBoxSelected.Emit(BoxSelection{Rect: r, Crossing: x < d.startX, Mods: d.mods})
			}
		}
		v.Update()
	}
	return v.Widget.ProcessMouseUp(button, x, y, mods, onlyForWidgetId)
}

func (v *View2D) mouseDblClick(button MouseButton, x, y int, mods KeyModifiers) bool {
	if button != MouseButtonMiddle || v.controlsOff {
		return false
	}
	v.ZoomExtents()
	return true
}

// view2DZoomStep is the zoom of a step of the wheel or a key
const view2DZoomStep = 1.25

func (v *View2D) wheel(e *WheelEvent) {
	if v.controlsOff {
		return
	}
	e.Handled = true
	if e.DY != 0 {
		v.ZoomAt(math.Pow(view2DZoomStep, float64(e.DY)), float64(e.X)+0.5, float64(e.Y)+0.5)
	}
	if e.DX != 0 {
		v.panPixels(float64(e.DX)*40, 0)
	}
}

func (v *View2D) keyDown(key Key, mods KeyModifiers) bool {
	if key == KeyEsc && (v.drag.active || v.zoomWindow) {
		v.drag.active, v.zoomWindow = false, false
		v.Update()
		return true
	}
	if v.controlsOff {
		return false
	}
	step := float64(min(v.Width(), v.Height())) / 8
	switch key {
	case KeyArrowLeft:
		v.panPixels(step, 0)
	case KeyArrowRight:
		v.panPixels(-step, 0)
	case KeyArrowUp:
		v.panPixels(0, step)
	case KeyArrowDown:
		v.panPixels(0, -step)
	case KeyEqual, KeyNumpadPlus, KeyPageUp:
		v.SetScale(v.scale * view2DZoomStep)
	case KeyMinus, KeyNumpadMinus, KeyPageDown:
		v.SetScale(v.scale / view2DZoomStep)
	case KeyHome:
		v.ZoomExtents()
	default:
		return false
	}
	return true
}
