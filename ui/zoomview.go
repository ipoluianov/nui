package ui

import (
	"image"
	"math"
)

// ZoomView shows content larger or smaller than itself, zoomed and moved
// by the user: an image, a drawing, a map, a diagram.
//
//   - the wheel scrolls (Shift+wheel sideways), Ctrl+wheel zooms around the
//     mouse: the point under it stays;
//   - the middle button drags the content (the left one too, with
//     SetPanWithLeftButton);
//   - the keys + and - zoom, 0 shows the actual size.
//
// The content is an image (SetImage) or painted by the application in the
// content's coordinates (SetPainter):
//
//	view := ui.NewZoomView()
//	view.SetImage(photo)
//	view.FitToView()
type ZoomView struct {
	Widget

	// OnZoomChanged: the zoom changed; the new zoom
	OnZoomChanged Signal[float64]

	img            image.Image
	smooth         bool
	painter        func(p *ZoomPaint)
	contentW       float64
	contentH       float64
	zoom           float64
	minZoom        float64
	maxZoom        float64
	offX, offY     float64 // the content point at the view's top-left corner
	panLeft        bool
	panning        bool
	panX, panY     int     // where the drag started, in the view
	panOffX, panOY float64 // the offset when it started
}

// ZoomPaint is the content of a ZoomView to paint, see SetPainter: the
// canvas is the view's; ToView converts the content's coordinates to it
type ZoomPaint struct {
	Canvas *Canvas
	Zoom   float64
	view   *ZoomView
}

// ToView converts a point of the content to the canvas
func (p *ZoomPaint) ToView(x, y float64) (int, int) {
	return p.view.ToView(x, y)
}

// NewZoomView makes an empty view at the zoom 1
func NewZoomView() *ZoomView {
	c := &ZoomView{zoom: 1, minZoom: 1.0 / 32, maxZoom: 64, smooth: true}
	c.InitWidget()
	c.SetTypeName("ZoomView")
	c.SetCanBeFocused(true)
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	c.SetMinSize(60, 60)
	c.SetOnPaint(c.draw)
	c.OnWheel.Connect(c.wheel)
	c.OnMouseDown.Connect(c.mouseDown)
	c.OnMouseMove.Connect(c.mouseMove)
	c.OnMouseUp.Connect(func(e *MouseEvent) { c.panning = false })
	c.OnKeyDown.Connect(c.keyDown)
	return c
}

// SetImage shows the image; the content is its size
func (c *ZoomView) SetImage(img image.Image) {
	c.img = img
	if img != nil {
		b := img.Bounds()
		c.contentW, c.contentH = float64(b.Dx()), float64(b.Dy())
	}
	c.clampOffset()
	c.form.Update()
}

// SetSmooth draws the zoomed image with interpolated pixels (a photo, the
// default) or as sharp squares (pixel art, a bitmap being edited)
func (c *ZoomView) SetSmooth(smooth bool) {
	c.smooth = smooth
	c.form.Update()
}

// SetPainter paints the content with the function, over the image if any
func (c *ZoomView) SetPainter(paint func(p *ZoomPaint)) {
	c.painter = paint
	c.form.Update()
}

// SetContentSize sets the size of the content, for a painter without an
// image: the view keeps it in reach
func (c *ZoomView) SetContentSize(width, height float64) {
	c.contentW, c.contentH = width, height
	c.clampOffset()
	c.form.Update()
}

// SetPanWithLeftButton lets the left button drag the content too
func (c *ZoomView) SetPanWithLeftButton(pan bool) {
	c.panLeft = pan
}

// SetZoomRange sets the smallest and the largest zoom
func (c *ZoomView) SetZoomRange(minZoom, maxZoom float64) {
	c.minZoom, c.maxZoom = minZoom, maxZoom
	c.SetZoom(c.zoom)
}

// Zoom returns the zoom: 2 shows the content twice as large
func (c *ZoomView) Zoom() float64 {
	return c.zoom
}

// SetZoom zooms around the center of the view
func (c *ZoomView) SetZoom(zoom float64) {
	c.ZoomAt(zoom, c.Width()/2, c.Height()/2)
}

// ZoomAt zooms keeping the content point at the view's point x, y where it is
func (c *ZoomView) ZoomAt(zoom float64, x, y int) {
	zoom = math.Max(c.minZoom, math.Min(c.maxZoom, zoom))
	cx, cy := c.ToContent(x, y)
	changed := zoom != c.zoom
	c.zoom = zoom
	c.offX = cx - float64(x)/zoom
	c.offY = cy - float64(y)/zoom
	c.clampOffset()
	c.form.Update()
	if changed {
		c.OnZoomChanged.Emit(zoom)
	}
}

// FitToView zooms the whole content into the view and centers it
func (c *ZoomView) FitToView() {
	if c.contentW <= 0 || c.contentH <= 0 || c.Width() <= 0 || c.Height() <= 0 {
		return
	}
	c.SetZoom(math.Min(float64(c.Width())/c.contentW, float64(c.Height())/c.contentH))
	c.CenterOn(c.contentW/2, c.contentH/2)
}

// ActualSize shows the content at the zoom 1, around the center of the view
func (c *ZoomView) ActualSize() {
	c.SetZoom(1)
}

// CenterOn moves the content point to the center of the view
func (c *ZoomView) CenterOn(x, y float64) {
	c.offX = x - float64(c.Width())/2/c.zoom
	c.offY = y - float64(c.Height())/2/c.zoom
	c.clampOffset()
	c.form.Update()
}

// ToContent converts a point of the view to the content's coordinates
func (c *ZoomView) ToContent(x, y int) (float64, float64) {
	return c.offX + float64(x)/c.zoom, c.offY + float64(y)/c.zoom
}

// ToView converts a point of the content to the view's coordinates
func (c *ZoomView) ToView(x, y float64) (int, int) {
	return int(math.Round((x - c.offX) * c.zoom)), int(math.Round((y - c.offY) * c.zoom))
}

// clampOffset keeps the content in reach: a content smaller than the view
// is centered, a larger one can't leave the view
func (c *ZoomView) clampOffset() {
	clamp := func(off, content, view float64) float64 {
		if content*c.zoom <= view {
			return -(view/c.zoom - content) / 2
		}
		return math.Max(0, math.Min(off, content-view/c.zoom))
	}
	if c.contentW > 0 {
		c.offX = clamp(c.offX, c.contentW, float64(c.Width()))
	}
	if c.contentH > 0 {
		c.offY = clamp(c.offY, c.contentH, float64(c.Height()))
	}
}

func (c *ZoomView) draw(cnv *Canvas) {
	c.clampOffset()
	if c.img != nil {
		x, y := c.ToView(0, 0)
		cnv.DrawImageRect(x, y, int(math.Round(c.contentW*c.zoom)), int(math.Round(c.contentH*c.zoom)), c.img, c.img.Bounds(), c.smooth)
	}
	if c.painter != nil {
		c.painter(&ZoomPaint{Canvas: cnv, Zoom: c.zoom, view: c})
	}
}

// zoomStep is the factor of a step of the wheel or a key
const zoomStep = 1.25

func (c *ZoomView) wheel(e *WheelEvent) {
	e.Handled = true
	if e.Mods.Ctrl || e.Mods.Cmd {
		c.ZoomAt(c.zoom*math.Pow(zoomStep, float64(e.DY)), e.X, e.Y)
		return
	}
	step := 40 / c.zoom
	if e.Mods.Shift {
		c.offX -= float64(e.DY) * step
	} else {
		c.offX -= float64(e.DX) * step
		c.offY -= float64(e.DY) * step
	}
	c.clampOffset()
	c.form.Update()
}

func (c *ZoomView) mouseDown(e *MouseEvent) {
	c.Focus()
	if e.Button == MouseButtonMiddle || e.Button == MouseButtonLeft && c.panLeft {
		c.panning = true
		c.panX, c.panY = e.X, e.Y
		c.panOffX, c.panOY = c.offX, c.offY
		e.Handled = true
	}
}

func (c *ZoomView) mouseMove(e *MouseEvent) {
	if !c.panning {
		return
	}
	c.offX = c.panOffX - float64(e.X-c.panX)/c.zoom
	c.offY = c.panOY - float64(e.Y-c.panY)/c.zoom
	c.clampOffset()
	c.form.Update()
	e.Handled = true
}

func (c *ZoomView) keyDown(e *KeyEvent) {
	switch e.Key {
	case KeyEqual, KeyNumpadPlus:
		c.SetZoom(c.zoom * zoomStep)
	case KeyMinus, KeyNumpadMinus:
		c.SetZoom(c.zoom / zoomStep)
	case Key0:
		c.ActualSize()
	default:
		return
	}
	e.Handled = true
}
