package paint

import (
	"image"
	"image/color"

	"github.com/ipoluianov/nui/ui"
)

// maxUndo is how many steps Undo goes back.
const maxUndo = 30

// fillTolerance: the fill spreads over colors this close to the clicked
// one, so it also covers the antialiased edges of the shapes.
const fillTolerance = 48

// paintCanvas is the drawing area: a custom widget showing the document
// image at its real size and drawing on it with the mouse.
type paintCanvas struct {
	ui.Widget

	doc *image.RGBA
	// preview is the document with the shape being dragged, shown instead
	// of doc until the mouse is released
	preview *image.RGBA

	tool               tool
	primary, secondary color.RGBA
	size               int
	fillShapes         bool

	drawing     bool
	start, last image.Point

	undo, redo [][]byte
	modified   bool

	onChanged    func()                             // the document changed
	onMouseMoved func(p image.Point, in bool)       // the cursor moved over the image
	onPicked     func(col color.RGBA, primary bool) // the eyedropper took a color
}

func newPaintCanvas(w, h int) *paintCanvas {
	var c paintCanvas
	c.InitWidget()
	c.SetTypeName("PaintCanvas")
	c.primary = color.RGBA{0, 0, 0, 255}
	c.secondary = color.RGBA{255, 255, 255, 255}
	c.size = 5
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.mouseDown)
	c.SetOnMouseMove(c.mouseMove)
	c.SetOnMouseUp(c.mouseUp)
	c.SetOnMouseLeave(func() {
		if c.onMouseMoved != nil {
			c.onMouseMoved(image.Point{}, false)
		}
	})
	c.setDocument(newDocument(w, h))
	return &c
}

// setDocument replaces the image, e.g. with an opened file; the undo
// history is cleared.
func (c *paintCanvas) setDocument(img *image.RGBA) {
	c.doc = img
	c.preview = nil
	c.drawing = false
	c.undo, c.redo = nil, nil
	c.modified = false
	b := img.Bounds()
	c.SetMinSize(b.Dx(), b.Dy())
	c.SetMaxSize(b.Dx(), b.Dy())
	if f := c.Form(); f != nil {
		f.UpdateLayout()
	}
	c.changed()
}

func (c *paintCanvas) draw(cnv *ui.Canvas) {
	if c.preview != nil {
		cnv.DrawImage(0, 0, c.preview)
	} else {
		cnv.DrawImage(0, 0, c.doc)
	}
}

// changed repaints the canvas and tells the window
func (c *paintCanvas) changed() {
	if f := c.Form(); f != nil {
		f.Update()
	}
	if c.onChanged != nil {
		c.onChanged()
	}
}

// edit runs a change of the document as one undo step.
func (c *paintCanvas) edit(f func(img *image.RGBA)) {
	c.saveUndo()
	f(c.doc)
	c.modified = true
	c.changed()
}

func (c *paintCanvas) saveUndo() {
	c.undo = append(c.undo, append([]byte(nil), c.doc.Pix...))
	if len(c.undo) > maxUndo {
		c.undo = c.undo[1:]
	}
	c.redo = nil
}

func (c *paintCanvas) Undo() {
	if len(c.undo) == 0 || c.drawing {
		return
	}
	c.redo = append(c.redo, c.doc.Pix)
	c.doc.Pix = c.undo[len(c.undo)-1]
	c.undo = c.undo[:len(c.undo)-1]
	c.modified = true
	c.changed()
}

func (c *paintCanvas) Redo() {
	if len(c.redo) == 0 || c.drawing {
		return
	}
	c.undo = append(c.undo, c.doc.Pix)
	c.doc.Pix = c.redo[len(c.redo)-1]
	c.redo = c.redo[:len(c.redo)-1]
	c.modified = true
	c.changed()
}

func (c *paintCanvas) mouseDown(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
	p := image.Pt(x, y)
	if !p.In(c.doc.Bounds()) {
		return false
	}
	// The right button takes the secondary color, but only with the tools
	// that act on a click: the window sends the moves and the release of
	// the left button only
	primary := button == ui.MouseButtonLeft
	col := c.primary
	if !primary {
		col = c.secondary
	}
	switch c.tool {
	case toolFill:
		c.saveUndo()
		if floodFill(c.doc, x, y, col, fillTolerance) == 0 {
			c.undo = c.undo[:len(c.undo)-1]
			return true
		}
		c.modified = true
		c.changed()
		return true
	case toolPicker:
		if c.onPicked != nil {
			c.onPicked(c.doc.RGBAAt(x, y), primary)
		}
		return true
	}
	if !primary {
		return true
	}

	c.saveUndo()
	c.drawing = true
	c.start, c.last = p, p
	c.drawFreehand(p)
	c.drawPreview(p, mods)
	c.changed()
	return true
}

func (c *paintCanvas) mouseMove(x, y int, mods ui.KeyModifiers) bool {
	p := image.Pt(x, y)
	if c.onMouseMoved != nil {
		c.onMouseMoved(p, p.In(c.doc.Bounds()))
	}
	if !c.drawing {
		return true
	}
	c.drawFreehand(p)
	c.drawPreview(p, mods)
	c.last = p
	c.Form().Update()
	return true
}

func (c *paintCanvas) mouseUp(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
	if !c.drawing || button != ui.MouseButtonLeft {
		return false
	}
	c.drawing = false
	if kind, ok := c.tool.shape(); ok {
		c.drawShapeTo(c.doc, kind, image.Pt(x, y), mods)
		c.preview = nil
	}
	c.modified = true
	c.changed()
	return true
}

// drawFreehand draws the pencil, brush or eraser from the last point to p.
func (c *paintCanvas) drawFreehand(p image.Point) {
	switch c.tool {
	case toolPencil:
		drawPixelLine(c.doc, c.last.X, c.last.Y, p.X, p.Y, c.primary)
	case toolBrush:
		drawStroke(c.doc, c.last.X, c.last.Y, p.X, p.Y, float64(c.size), c.primary)
	case toolEraser:
		drawStroke(c.doc, c.last.X, c.last.Y, p.X, p.Y, float64(c.size*2), c.secondary)
	}
}

// drawPreview shows the shape being dragged on a copy of the document.
func (c *paintCanvas) drawPreview(p image.Point, mods ui.KeyModifiers) {
	kind, ok := c.tool.shape()
	if !ok {
		return
	}
	if c.preview == nil {
		c.preview = image.NewRGBA(c.doc.Bounds())
	}
	copy(c.preview.Pix, c.doc.Pix)
	c.drawShapeTo(c.preview, kind, p, mods)
}

func (c *paintCanvas) drawShapeTo(img *image.RGBA, kind shapeKind, p image.Point, mods ui.KeyModifiers) {
	if mods.Shift {
		p = constrain(kind, c.start, p)
	}
	drawShape(img, kind, c.start, p, float64(c.size), c.primary, c.fillShapes)
}
