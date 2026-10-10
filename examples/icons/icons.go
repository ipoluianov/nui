// Package icons draws the small icons the example applications use for
// their menus and toolbars, so the examples need no image files; an
// application would load its own (e.g. PNGs with image.Decode).
package icons

import (
	"image"

	"github.com/fogleman/gg"
)

// Size is the size of the menu icons. Toolbar icons are drawn with Draw at
// any size.
const Size = 16

// Draw draws an icon of the given size; the drawing is in a 16x16
// coordinate space, scaled to the size.
func Draw(size int, draw func(dc *gg.Context)) image.Image {
	dc := gg.NewContext(size, size)
	dc.Scale(float64(size)/Size, float64(size)/Size)
	draw(dc)
	return dc.Image()
}

// New is a sheet of paper with a folded corner.
func New(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.MoveTo(3, 1.5)
		dc.LineTo(10, 1.5)
		dc.LineTo(13, 4.5)
		dc.LineTo(13, 14.5)
		dc.LineTo(3, 14.5)
		dc.ClosePath()
		dc.SetHexColor("#F5F5F5")
		dc.FillPreserve()
		dc.SetHexColor("#757575")
		dc.SetLineWidth(1)
		dc.Stroke()
		dc.MoveTo(10, 1.5)
		dc.LineTo(10, 4.5)
		dc.LineTo(13, 4.5)
		dc.Stroke()
	})
}

// Folder is a yellow folder.
func Folder(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#E0A526")
		dc.DrawRoundedRectangle(1, 3, 6, 3, 1)
		dc.Fill()
		dc.SetHexColor("#FFC940")
		dc.DrawRoundedRectangle(1, 5, 14, 9, 1.5)
		dc.Fill()
	})
}

// File is a sheet of paper with lines of text.
func File(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.DrawRectangle(3, 1.5, 10, 13)
		dc.SetHexColor("#F5F5F5")
		dc.FillPreserve()
		dc.SetHexColor("#757575")
		dc.SetLineWidth(1)
		dc.Stroke()
		dc.SetHexColor("#90A4AE")
		for y := 4.5; y < 13; y += 2.5 {
			dc.DrawLine(5, y, 11, y)
		}
		dc.Stroke()
	})
}

// Save is a floppy disk.
func Save(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#1D6FB5")
		dc.DrawRoundedRectangle(1.5, 1.5, 13, 13, 1.5)
		dc.Fill()
		dc.SetHexColor("#FFFFFF")
		dc.DrawRectangle(4, 2, 8, 5)
		dc.Fill()
		dc.SetHexColor("#CFD8DC")
		dc.DrawRectangle(4, 9.5, 8, 4)
		dc.Fill()
	})
}

// Printer is a printer with a sheet of paper.
func Printer(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#F5F5F5")
		dc.DrawRectangle(4.5, 1.5, 7, 5)
		dc.FillPreserve()
		dc.SetHexColor("#757575")
		dc.SetLineWidth(1)
		dc.Stroke()
		dc.SetHexColor("#607D8B")
		dc.DrawRoundedRectangle(1, 6, 14, 6, 1.5)
		dc.Fill()
		dc.SetHexColor("#F5F5F5")
		dc.DrawRectangle(4.5, 10, 7, 4.5)
		dc.Fill()
	})
}

// Cross is a red cross.
func Cross(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#E53935")
		dc.SetLineWidth(2.5)
		dc.SetLineCapRound()
		dc.DrawLine(4, 4, 12, 12)
		dc.DrawLine(12, 4, 4, 12)
		dc.Stroke()
	})
}

// Plus is a green plus.
func Plus(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#43A047")
		dc.SetLineWidth(2.5)
		dc.SetLineCapRound()
		dc.DrawLine(8, 3, 8, 13)
		dc.DrawLine(3, 8, 13, 8)
		dc.Stroke()
	})
}

// Pencil is a pencil for "edit".
func Pencil(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.RotateAbout(gg.Radians(45), 8, 8)
		dc.SetHexColor("#FFB300")
		dc.DrawRectangle(6, 0.5, 4, 11)
		dc.Fill()
		dc.SetHexColor("#F48FB1")
		dc.DrawRectangle(6, 0.5, 4, 2)
		dc.Fill()
		dc.SetHexColor("#5D4037")
		dc.MoveTo(6, 11.5)
		dc.LineTo(10, 11.5)
		dc.LineTo(8, 15)
		dc.ClosePath()
		dc.Fill()
	})
}

// Arrow is a green arrow up or down.
func Arrow(size int, up bool) image.Image {
	return Draw(size, func(dc *gg.Context) {
		if !up {
			dc.RotateAbout(gg.Radians(180), 8, 8)
		}
		dc.SetHexColor("#43A047")
		dc.MoveTo(8, 1.5)
		dc.LineTo(14, 8)
		dc.LineTo(10.5, 8)
		dc.LineTo(10.5, 14.5)
		dc.LineTo(5.5, 14.5)
		dc.LineTo(5.5, 8)
		dc.LineTo(2, 8)
		dc.ClosePath()
		dc.Fill()
	})
}

// Back is an arrow to the left.
func Back(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#1E88E5")
		dc.MoveTo(1.5, 8)
		dc.LineTo(8, 2)
		dc.LineTo(8, 5.5)
		dc.LineTo(14.5, 5.5)
		dc.LineTo(14.5, 10.5)
		dc.LineTo(8, 10.5)
		dc.LineTo(8, 14)
		dc.ClosePath()
		dc.Fill()
	})
}

// Undo is a curved arrow to the left; Redo the same to the right.
func Undo(size int) image.Image { return curvedArrow(size, false) }

// Redo is a curved arrow to the right.
func Redo(size int) image.Image { return curvedArrow(size, true) }

func curvedArrow(size int, right bool) image.Image {
	return Draw(size, func(dc *gg.Context) {
		if right {
			dc.Translate(16, 0)
			dc.Scale(-1, 1)
		}
		dc.SetHexColor("#1E88E5")
		dc.SetLineWidth(2)
		dc.DrawArc(9, 10, 5, gg.Radians(180), gg.Radians(360))
		dc.Stroke()
		dc.MoveTo(1, 9)
		dc.LineTo(7, 9)
		dc.LineTo(4, 14)
		dc.ClosePath()
		dc.Fill()
	})
}

// Search is a magnifying glass.
func Search(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#607D8B")
		dc.SetLineWidth(2)
		dc.DrawCircle(6.5, 6.5, 4.5)
		dc.Stroke()
		dc.SetLineWidth(2.5)
		dc.SetLineCapRound()
		dc.DrawLine(10, 10, 14, 14)
		dc.Stroke()
	})
}

// Sun is a sun with rays.
func Sun(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#FFB300")
		dc.DrawCircle(8, 8, 3.5)
		dc.Fill()
		dc.SetLineWidth(1.5)
		dc.SetLineCapRound()
		for i := 0; i < 8; i++ {
			dc.Push()
			dc.RotateAbout(gg.Radians(float64(i)*45), 8, 8)
			dc.DrawLine(8, 1.5, 8, 2.8)
			dc.Stroke()
			dc.Pop()
		}
	})
}

// Moon is a crescent moon.
func Moon(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		// A circle without the part a shifted circle covers
		dc.DrawCircle(11, 5, 5.5)
		dc.Clip()
		dc.InvertMask()
		dc.SetHexColor("#7E57C2")
		dc.DrawCircle(8, 8, 6.5)
		dc.Fill()
	})
}

// Info is a blue circle with an "i".
func Info(size int) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#1D6FB5")
		dc.DrawCircle(8, 8, 7)
		dc.Fill()
		dc.SetHexColor("#FFFFFF")
		dc.DrawCircle(8, 4.5, 1.3)
		dc.Fill()
		dc.DrawRoundedRectangle(7, 6.8, 2, 6, 0.8)
		dc.Fill()
	})
}

// App is a rounded square with a letter-like mark, the icon of an
// application in the launcher: color is its background.
func App(size int, color string, mark func(dc *gg.Context)) image.Image {
	return Draw(size, func(dc *gg.Context) {
		dc.SetHexColor(color)
		dc.DrawRoundedRectangle(0.5, 0.5, 15, 15, 3.5)
		dc.Fill()
		dc.SetHexColor("#FFFFFF")
		dc.SetLineWidth(1.3)
		dc.SetLineCapRound()
		mark(dc)
	})
}
