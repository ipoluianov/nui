package paint

import (
	"image"
	"image/color"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

type tool int

const (
	toolPencil tool = iota
	toolBrush
	toolLine
	toolRect
	toolEllipse
	toolFill
	toolEraser
	toolPicker
)

// toolInfo describes a tool's button.
type toolInfo struct {
	name     string
	shortcut string
	icon     func(size int) image.Image
}

// drawn makes an icon of a drawing in the 16x16 space of icons.Draw.
func drawn(draw func(dc *gg.Context)) func(size int) image.Image {
	return func(size int) image.Image { return icons.Draw(size, draw) }
}

var tools = []toolInfo{
	toolPencil:  {"Pencil", "P", icons.Pencil},
	toolBrush:   {"Brush", "B", drawn(drawBrushIcon)},
	toolLine:    {"Line", "L", drawn(drawLineIcon)},
	toolRect:    {"Rectangle", "R", drawn(drawRectIcon)},
	toolEllipse: {"Ellipse", "E", drawn(drawEllipseIcon)},
	toolFill:    {"Fill", "F", drawn(drawFillIcon)},
	toolEraser:  {"Eraser", "X", drawn(drawEraserIcon)},
	toolPicker:  {"Color Picker", "I", drawn(drawPickerIcon)},
}

func (t tool) String() string { return tools[t].name }

// shape returns the figure the tool draws by dragging.
func (t tool) shape() (shapeKind, bool) {
	switch t {
	case toolLine:
		return shapeLine, true
	case toolRect:
		return shapeRect, true
	case toolEllipse:
		return shapeEllipse, true
	}
	return 0, false
}

// The tool icons, drawn in the 16x16 space of icons.Draw.

const toolIconSize = 24

func drawBrushIcon(dc *gg.Context) {
	dc.SetHexColor("#8D6E63")
	dc.SetLineWidth(2.5)
	dc.SetLineCapRound()
	dc.DrawLine(14, 2, 8, 8)
	dc.Stroke()
	dc.SetHexColor("#1E88E5")
	dc.MoveTo(8.5, 7.5)
	dc.CubicTo(4, 7, 4, 12, 1.5, 14.5)
	dc.CubicTo(6, 14.5, 9.5, 12, 8.5, 7.5)
	dc.Fill()
}

func drawLineIcon(dc *gg.Context) {
	dc.SetHexColor("#90A4AE")
	dc.SetLineWidth(2)
	dc.SetLineCapRound()
	dc.DrawLine(2.5, 13.5, 13.5, 2.5)
	dc.Stroke()
}

func drawRectIcon(dc *gg.Context) {
	dc.SetHexColor("#90A4AE")
	dc.SetLineWidth(1.5)
	dc.DrawRectangle(2, 3.5, 12, 9)
	dc.Stroke()
}

func drawEllipseIcon(dc *gg.Context) {
	dc.SetHexColor("#90A4AE")
	dc.SetLineWidth(1.5)
	dc.DrawEllipse(8, 8, 6.5, 4.5)
	dc.Stroke()
}

func drawFillIcon(dc *gg.Context) {
	// A tilted bucket with paint pouring from it
	dc.Push()
	dc.RotateAbout(gg.Radians(-30), 7, 8)
	dc.SetHexColor("#B0BEC5")
	dc.DrawRectangle(3, 5, 8, 8)
	dc.Fill()
	dc.SetHexColor("#78909C")
	dc.SetLineWidth(1)
	dc.DrawLine(3, 5, 11, 5)
	dc.Stroke()
	dc.Pop()
	dc.SetHexColor("#E53935")
	dc.DrawEllipse(13.5, 12, 1.8, 2.8)
	dc.Fill()
}

func drawEraserIcon(dc *gg.Context) {
	dc.RotateAbout(gg.Radians(-35), 8, 8)
	dc.SetHexColor("#F48FB1")
	dc.DrawRoundedRectangle(1, 5, 8, 6, 1)
	dc.Fill()
	dc.SetHexColor("#ECEFF1")
	dc.DrawRoundedRectangle(8, 5, 7, 6, 1)
	dc.Fill()
	dc.SetHexColor("#90A4AE")
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(1, 5, 14, 6, 1)
	dc.Stroke()
}

func drawPickerIcon(dc *gg.Context) {
	dc.SetLineCapRound()
	dc.SetHexColor("#90A4AE")
	dc.SetLineWidth(2)
	dc.DrawLine(3, 13, 10, 6)
	dc.Stroke()
	dc.SetHexColor("#546E7A")
	dc.SetLineWidth(4)
	dc.DrawLine(10.5, 5.5, 13, 3)
	dc.Stroke()
	dc.SetHexColor("#1E88E5")
	dc.DrawCircle(2.5, 13.5, 1.5)
	dc.Fill()
}

// The quick colors of the palette popup.
var paletteColors = []string{
	"#000000", "#616161", "#9E9E9E", "#FFFFFF",
	"#B71C1C", "#E53935", "#FB8C00", "#FDD835",
	"#43A047", "#1B5E20", "#00897B", "#00ACC1",
	"#1E88E5", "#3949AB", "#8E24AA", "#D81B60",
	"#6D4C41", "#FFCCBC", "#C5E1A5", "#B3E5FC",
}

const (
	paletteColumns = 4
	paletteCell    = 30
	palettePadding = 6
)

// palette is a popup with a grid of color swatches that drops down from a
// button: a click with the left button chooses the primary color, with the
// right button the secondary one.
type palette struct {
	ui.Widget

	onPick func(col color.RGBA, primary bool)
	hover  int

	// anchorX, anchorY, anchorW: the button the palette drops down from,
	// in the form's client coordinates
	anchorX, anchorY, anchorW int
}

func newPalette(onPick func(col color.RGBA, primary bool)) *palette {
	c := &palette{onPick: onPick, hover: -1}
	c.InitWidget()
	c.SetAutoFillBackground(true)
	c.SetRole("popup")
	c.SetOnPaint(c.draw)
	c.SetOnMouseMove(func(x, y int, mods ui.KeyModifiers) bool {
		c.hover = c.cellAt(x, y)
		c.Form().Update()
		return true
	})
	c.SetOnMouseDown(func(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
		if i := c.cellAt(x, y); i >= 0 {
			c.onPick(ui.ColorFromHex(paletteColors[i]), button != ui.MouseButtonRight)
			c.Form().CloseTopPopup()
		}
		return true
	})
	c.SetTooltip("Left click: primary color, right click: secondary color")
	return c
}

func (c *palette) cellRect(i int) image.Rectangle {
	x := palettePadding + i%paletteColumns*paletteCell
	y := palettePadding + i/paletteColumns*paletteCell
	return image.Rect(x, y, x+paletteCell, y+paletteCell)
}

func (c *palette) cellAt(x, y int) int {
	for i := range paletteColors {
		if image.Pt(x, y).In(c.cellRect(i)) {
			return i
		}
	}
	return -1
}

func (c *palette) draw(cnv *ui.Canvas) {
	cnv.SetColor(color.RGBA{128, 128, 128, 255})
	cnv.DrawRect(0, 0, c.Width(), c.Height())
	for i, hex := range paletteColors {
		r := c.cellRect(i).Inset(3)
		border := color.Color(color.RGBA{128, 128, 128, 255})
		if i == c.hover {
			border = c.ForegroundColor()
		}
		cnv.FillFrame(r.Min.X, r.Min.Y, r.Dx(), r.Dy(), 3, ui.ColorFromHex(hex), border)
	}
}

// openBelow opens the palette under the button.
func (c *palette) openBelow(anchor *ui.Button) {
	c.anchorX, c.anchorY = anchor.RectClientAreaOnWindow()
	c.anchorW = anchor.Width()

	rows := (len(paletteColors) + paletteColumns - 1) / paletteColumns
	c.SetSize(paletteColumns*paletteCell+2*palettePadding, rows*paletteCell+2*palettePadding)
	c.SetPosition(c.anchorX, c.anchorY+anchor.Height())
	anchor.Form().OpenPopup(c)
}

// PopupFlipped opens the palette above the button when it doesn't fit
// below, and aligns it to the button's right edge when it doesn't fit to
// the right.
func (c *palette) PopupFlipped() (x, y int) {
	return c.anchorX + c.anchorW - c.Width(), c.anchorY - c.Height()
}
