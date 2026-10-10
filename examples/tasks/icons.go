package tasks

import (
	"image"
	"image/color"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/icons"
)

// The table scales the cell images to the row height: they are drawn larger
const cellIconSize = 32

var (
	checkedIcon   = drawCheck(cellIconSize, true)
	uncheckedIcon = drawCheck(cellIconSize, false)
)

// checkIcon is the check box of the Done column
func checkIcon(done bool) image.Image {
	if done {
		return checkedIcon
	}
	return uncheckedIcon
}

func drawCheck(size int, done bool) image.Image {
	return icons.Draw(size, func(dc *gg.Context) {
		dc.DrawRoundedRectangle(2, 2, 12, 12, 2.5)
		if !done {
			dc.SetHexColor("#90A4AE")
			dc.SetLineWidth(1.2)
			dc.Stroke()
			return
		}
		dc.SetHexColor("#43A047")
		dc.Fill()
		dc.SetHexColor("#FFFFFF")
		dc.SetLineWidth(1.8)
		dc.SetLineCapRound()
		dc.MoveTo(4.8, 8.2)
		dc.LineTo(7, 10.5)
		dc.LineTo(11.2, 5.5)
		dc.Stroke()
	})
}

// labelIcon is the color mark next to the title
func labelIcon(c color.RGBA) image.Image {
	return icons.Draw(cellIconSize, func(dc *gg.Context) {
		dc.SetColor(c)
		dc.DrawCircle(8, 8, 4.5)
		dc.Fill()
	})
}

func pencilIcon() image.Image { return icons.Pencil(icons.Size) }
func crossIcon() image.Image  { return icons.Cross(icons.Size) }

// appIcon is the icon of the window and the tray: a check mark on blue
func appIcon(size int) image.Image {
	return icons.App(size, "#1E88E5", func(dc *gg.Context) {
		dc.SetLineWidth(2)
		dc.MoveTo(4.5, 8.5)
		dc.LineTo(7, 11)
		dc.LineTo(11.5, 5.5)
		dc.Stroke()
	})
}
