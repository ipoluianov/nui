package paint

import (
	"image"
	"image/color"
	"testing"
)

var (
	black = color.RGBA{0, 0, 0, 255}
	red   = color.RGBA{255, 0, 0, 255}
	white = color.RGBA{255, 255, 255, 255}
)

func TestFloodFillStopsAtBorder(t *testing.T) {
	img := newDocument(20, 20)
	// A closed square outline from 5,5 to 15,15
	for i := 5; i <= 15; i++ {
		img.SetRGBA(i, 5, black)
		img.SetRGBA(i, 15, black)
		img.SetRGBA(5, i, black)
		img.SetRGBA(15, i, black)
	}

	if n := floodFill(img, 10, 10, red, 0); n != 9*9 {
		t.Errorf("filled %d pixels inside, want %d", n, 9*9)
	}
	if got := img.RGBAAt(6, 6); got != red {
		t.Errorf("inside = %v, want red", got)
	}
	if got := img.RGBAAt(5, 5); got != black {
		t.Errorf("border = %v, want black", got)
	}
	if got := img.RGBAAt(2, 2); got != white {
		t.Errorf("outside = %v, want white", got)
	}

	// Outside: everything but the square
	if n := floodFill(img, 0, 0, red, 0); n != 400-11*11 {
		t.Errorf("filled %d pixels outside, want %d", n, 400-11*11)
	}
	// The same color again: nothing to do
	if n := floodFill(img, 0, 0, red, 0); n != 0 {
		t.Errorf("refill filled %d pixels, want 0", n)
	}
}

func TestFloodFillTolerance(t *testing.T) {
	img := newDocument(10, 1)
	img.SetRGBA(5, 0, color.RGBA{240, 240, 240, 255})
	if n := floodFill(img, 0, 0, red, 0); n != 5 {
		t.Errorf("without tolerance filled %d, want 5", n)
	}
	img = newDocument(10, 1)
	img.SetRGBA(5, 0, color.RGBA{240, 240, 240, 255})
	if n := floodFill(img, 0, 0, red, 20); n != 10 {
		t.Errorf("with tolerance filled %d, want 10", n)
	}
}

func TestDrawShapes(t *testing.T) {
	img := newDocument(100, 100)
	drawShape(img, shapeRect, image.Pt(10, 10), image.Pt(50, 40), 2, black, false)
	if got := img.RGBAAt(10, 25); got.R > 100 {
		t.Errorf("rectangle edge = %v, want dark", got)
	}
	if got := img.RGBAAt(30, 25); got != white {
		t.Errorf("inside of an outline = %v, want white", got)
	}

	drawShape(img, shapeEllipse, image.Pt(60, 60), image.Pt(90, 90), 1, red, true)
	if got := img.RGBAAt(75, 75); got != red {
		t.Errorf("center of a filled ellipse = %v, want red", got)
	}
	if got := img.RGBAAt(61, 61); got != white {
		t.Errorf("corner outside the ellipse = %v, want white", got)
	}
}

func TestConstrain(t *testing.T) {
	p0 := image.Pt(10, 10)
	if got := constrain(shapeRect, p0, image.Pt(30, 15)); got != image.Pt(30, 30) {
		t.Errorf("square = %v", got)
	}
	if got := constrain(shapeEllipse, p0, image.Pt(5, 30)); got != image.Pt(-10, 30) {
		t.Errorf("circle = %v", got)
	}
	if got := constrain(shapeLine, p0, image.Pt(40, 12)); got != image.Pt(40, 10) {
		t.Errorf("horizontal line = %v", got)
	}
}

func TestInvertAndFlip(t *testing.T) {
	img := newDocument(3, 1)
	img.SetRGBA(0, 0, red)
	flipHorizontal(img)
	if img.RGBAAt(2, 0) != red || img.RGBAAt(0, 0) != white {
		t.Errorf("flip: %v", img.Pix)
	}
	invertColors(img)
	if got := img.RGBAAt(2, 0); got != (color.RGBA{0, 255, 255, 255}) {
		t.Errorf("inverted red = %v, want cyan", got)
	}
	if got := img.RGBAAt(0, 0); got != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("inverted white = %v, want black", got)
	}
}

func TestUndoRedo(t *testing.T) {
	c := newPaintCanvas(10, 10)
	c.edit(invertColors)
	if c.doc.RGBAAt(0, 0) != black {
		t.Fatal("edit didn't apply")
	}
	c.Undo()
	if c.doc.RGBAAt(0, 0) != white {
		t.Error("undo didn't restore")
	}
	c.Redo()
	if c.doc.RGBAAt(0, 0) != black {
		t.Error("redo didn't reapply")
	}
	for i := 0; i < maxUndo+10; i++ {
		c.edit(invertColors)
	}
	if len(c.undo) != maxUndo {
		t.Errorf("undo steps = %d, want %d", len(c.undo), maxUndo)
	}
}
