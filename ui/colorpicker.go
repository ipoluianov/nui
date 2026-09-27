package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"
)

// ColorPicker is a field showing a color; a click (or Enter, Space, Down,
// F4) opens a panel to choose another one: a saturation/brightness square,
// a hue bar, an optional opacity bar, a hex code, a palette and the colors
// chosen recently.
//
//	picker := ui.NewColorPicker()
//	picker.SetColor(ui.ColorFromHex("#1E88E5"))
//	picker.SetOnColorChanged(func(col color.RGBA) { preview.SetBackgroundColor(col) })
//
// The color changes live while it's being chosen; Escape in the panel
// returns to the color it was opened with, a click outside keeps the new one.
//
// Colors are color.RGBA with a straight (not premultiplied) alpha, as
// ColorFromHex returns them.
type ColorPicker struct {
	Widget
	col            color.RGBA
	alphaEnabled   bool
	palette        []color.RGBA
	onColorChanged func(col color.RGBA)
	popup          *colorPickerPopup
}

// DefaultColorPickerPalette is the palette a new ColorPicker offers.
var DefaultColorPickerPalette = []color.RGBA{
	{0x00, 0x00, 0x00, 0xFF}, {0x42, 0x42, 0x42, 0xFF}, {0x75, 0x75, 0x75, 0xFF}, {0xBD, 0xBD, 0xBD, 0xFF}, {0xFF, 0xFF, 0xFF, 0xFF},
	{0x79, 0x55, 0x48, 0xFF}, {0xF4, 0x43, 0x36, 0xFF}, {0xE9, 0x1E, 0x63, 0xFF}, {0x9C, 0x27, 0xB0, 0xFF}, {0x67, 0x3A, 0xB7, 0xFF},
	{0x3F, 0x51, 0xB5, 0xFF}, {0x21, 0x96, 0xF3, 0xFF}, {0x03, 0xA9, 0xF4, 0xFF}, {0x00, 0xBC, 0xD4, 0xFF}, {0x00, 0x96, 0x88, 0xFF},
	{0x4C, 0xAF, 0x50, 0xFF}, {0x8B, 0xC3, 0x4A, 0xFF}, {0xCD, 0xDC, 0x39, 0xFF}, {0xFF, 0xEB, 0x3B, 0xFF}, {0xFF, 0x98, 0x00, 0xFF},
}

// Colors chosen recently in any ColorPicker of the application, newest first
var (
	recentColorsMu sync.Mutex
	recentColors   []color.RGBA
)

const colorPickerRecentMax = 10

func addRecentColor(col color.RGBA) {
	recentColorsMu.Lock()
	defer recentColorsMu.Unlock()
	list := []color.RGBA{col}
	for _, c := range recentColors {
		if c != col && len(list) < colorPickerRecentMax {
			list = append(list, c)
		}
	}
	recentColors = list
}

// RecentColors returns the colors chosen recently in the application's
// color pickers, newest first.
func RecentColors() []color.RGBA {
	recentColorsMu.Lock()
	defer recentColorsMu.Unlock()
	return append([]color.RGBA(nil), recentColors...)
}

func NewColorPicker() *ColorPicker {
	var c ColorPicker
	c.InitWidget()
	c.SetTypeName("ColorPicker")
	c.SetMinWidth(DefaultComboBoxMinWidth)
	c.SetMaxWidth(10000)
	c.setThemeHeight(ThemeControlHeight, true)
	c.SetCanBeFocused(true)
	c.SetMouseCursor(MouseCursorPointer)
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(func(button MouseButton, x int, y int, mods KeyModifiers) bool {
		if button == MouseButtonLeft && c.Enabled() {
			c.OpenPopup()
		}
		return true
	})
	c.col = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	c.palette = DefaultColorPickerPalette
	return &c
}

// Color returns the chosen color.
func (c *ColorPicker) Color() color.RGBA {
	return c.col
}

// SetColor sets the color without calling SetOnColorChanged's function.
// A color.RGBA is taken as it is (straight alpha); other colors are
// converted. Without SetAlphaEnabled the color is made opaque.
func (c *ColorPicker) SetColor(col color.Color) {
	c.col = c.normalize(toStraightRGBA(col))
	if c.popup != nil {
		c.popup.setColor(c.col, true)
	}
	c.form.Update()
}

// SetOnColorChanged sets the function called whenever the user changes the
// color, also while dragging in the panel.
func (c *ColorPicker) SetOnColorChanged(f func(col color.RGBA)) {
	c.onColorChanged = f
}

// SetAlphaEnabled lets the user choose the opacity too; the hex code then
// has 8 digits.
func (c *ColorPicker) SetAlphaEnabled(enabled bool) {
	c.alphaEnabled = enabled
	c.col = c.normalize(c.col)
	c.form.Update()
}

func (c *ColorPicker) AlphaEnabled() bool {
	return c.alphaEnabled
}

// SetPalette sets the colors offered in the panel; nil hides the palette.
func (c *ColorPicker) SetPalette(colors []color.RGBA) {
	c.palette = colors
}

func (c *ColorPicker) Palette() []color.RGBA {
	return c.palette
}

// IsPopupOpen reports whether the panel is open.
func (c *ColorPicker) IsPopupOpen() bool {
	return c.popup != nil
}

func (c *ColorPicker) normalize(col color.RGBA) color.RGBA {
	if !c.alphaEnabled {
		col.A = 0xFF
	}
	return col
}

// changeColor is a change made by the user in the panel.
func (c *ColorPicker) changeColor(col color.RGBA) {
	col = c.normalize(col)
	if col == c.col {
		return
	}
	c.col = col
	c.form.Update()
	if c.onColorChanged != nil {
		c.onColorChanged(col)
	}
}

// OpenPopup opens the panel below the field (above it when it doesn't fit).
func (c *ColorPicker) OpenPopup() {
	if c.form == nil || c.popup != nil {
		return
	}
	popup := newColorPickerPopup(c)
	c.popup = popup
	x, y := c.RectClientAreaOnWindow()
	popup.anchorX, popup.anchorY, popup.anchorW = x, y, c.Width()
	popup.SetPosition(x, y+c.Height())
	c.form.OpenPopup(popup)
}

// ClosePopup closes the panel keeping the chosen color.
func (c *ColorPicker) ClosePopup() {
	if c.popup != nil && c.form.TopPopupWidget() == Widgeter(c.popup) {
		c.form.CloseTopPopup()
	}
}

func (c *ColorPicker) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	if c.onKeyDown != nil && c.onKeyDown(key, mods) {
		return true
	}
	switch key {
	case KeyEnter, KeySpace, KeyArrowDown, KeyF4:
		if c.Enabled() {
			c.OpenPopup()
			if c.popup != nil {
				c.popup.hexBox.Focus() // so the code can be typed at once
			}
		}
		return true
	}
	return false
}

func (c *ColorPicker) draw(cnv *Canvas) {
	p := CurrentPalette()
	fill, border := p.Button, p.Border
	foreColor := colorToRGBA(c.ForegroundColor())
	switch {
	case !c.Enabled():
		foreColor = p.DisabledText
	case c.IsHovered() || c.popup != nil:
		fill = hoverColor(fill, foreColor)
	}
	if (c.IsFocused() || c.popup != nil) && c.Enabled() {
		border = p.Highlight
	}
	cnv.FillFrame(0, 0, c.Width(), c.Height(), themeControlRadius, fill, border)

	const inset = 4
	swatchH := c.Height() - inset*2
	swatchW := swatchH * 3 / 2
	drawColorSwatch(cnv, inset, inset, swatchW, swatchH, c.col, fill)
	cnv.SetColor(p.Border)
	cnv.DrawRect(inset, inset, swatchW, swatchH)

	textX := inset + swatchW + themeTextInset
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(foreColor)
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	textW := c.Width() - textX - comboBoxArrowPadding*2 - comboBoxArrowWidth
	cnv.DrawText(textX, 0, textW, c.Height(), colorToHexString(c.col, c.alphaEnabled))

	x := c.Width() - comboBoxArrowPadding - comboBoxArrowWidth
	y := (c.Height() - comboBoxArrowHeight) / 2
	cnv.FillTriangle(x, y, x+comboBoxArrowWidth, y, x+comboBoxArrowWidth/2, y+comboBoxArrowHeight, foreColor)
}

// ---------------------------------------------------------------- Panel

// Layout of the panel
const (
	cpPad        = 10
	cpGap        = 8
	cpSVWidth    = 220
	cpSVHeight   = 160
	cpBarSize    = 18
	cpSwatchGap  = 4
	cpPaletteCol = 10
	cpMarkerSize = 12
)

type cpArea int

const (
	cpAreaNone cpArea = iota
	cpAreaSV
	cpAreaHue
	cpAreaAlpha
)

type colorPickerPopup struct {
	Widget
	picker *ColorPicker

	// HSV kept separately from the color: the hue survives grays and black
	h, s, v float64
	a       uint8

	original  color.RGBA
	cancelled bool

	hexBox      *TextBox
	settingHex  bool
	dragArea    cpArea
	recent      []color.RGBA
	anchorX     int
	anchorY     int
	anchorW     int
	svImage     *image.RGBA
	svImageHue  float64
	hueImage    *image.RGBA
	alphaImage  *image.RGBA
	alphaForRGB [3]uint8
}

func newColorPickerPopup(picker *ColorPicker) *colorPickerPopup {
	var c colorPickerPopup
	c.InitWidget()
	c.SetTypeName("ColorPickerPopup")
	c.SetAbsolutePositioning(true)
	c.SetRole("popup")
	c.SetAutoFillBackground(true)
	c.SetOnPaint(c.draw)
	c.SetOnPostPaint(func(cnv *Canvas) {
		cnv.SetColor(CurrentPalette().Border)
		cnv.DrawRect(0, 0, c.Width(), c.Height())
	})
	c.SetOnMouseDown(c.mouseDown)
	c.SetOnMouseMove(c.mouseMove)
	c.SetOnMouseUp(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		c.dragArea = cpAreaNone
		return true
	})
	c.picker = picker
	c.original = picker.col
	c.recent = RecentColors()
	c.svImageHue = -1

	c.hexBox = NewTextBox()
	c.hexBox.SetOnTextChanged(c.hexChanged)
	c.hexBox.SetOnTextBoxKeyDown(func() {
		ev := CurrentEvent().Parameter.(*EventTextboxKeyDown)
		if ev.Key == KeyEnter {
			c.hexChanged()
			c.picker.ClosePopup()
			ev.Processed = true
		}
	})
	c.AddWidget(0, 0, c.hexBox)

	c.setColor(picker.col, true)
	c.layout()
	return &c
}

// Geometry, in the panel's coordinates

func (c *colorPickerPopup) contentWidth() int {
	return cpSVWidth + cpGap + cpBarSize
}

func (c *colorPickerPopup) svRect() (int, int, int, int) {
	return cpPad, cpPad, cpSVWidth, cpSVHeight
}

func (c *colorPickerPopup) hueRect() (int, int, int, int) {
	return cpPad + cpSVWidth + cpGap, cpPad, cpBarSize, cpSVHeight
}

func (c *colorPickerPopup) alphaRect() (int, int, int, int) {
	return cpPad, cpPad + cpSVHeight + cpGap, c.contentWidth(), cpBarSize
}

// previewRect is the old/new color comparison, left of the hex field.
func (c *colorPickerPopup) previewRect() (int, int, int, int) {
	y := cpPad + cpSVHeight + cpGap
	if c.picker.alphaEnabled {
		y += cpBarSize + cpGap
	}
	return cpPad, y, 80, ThemeControlHeight()
}

func (c *colorPickerPopup) swatchSize() int {
	return (c.contentWidth() - (cpPaletteCol-1)*cpSwatchGap) / cpPaletteCol
}

// swatchesTop returns where the palette starts and where the recent colors
// start.
func (c *colorPickerPopup) swatchesTop() (paletteY, recentY int) {
	_, y, _, h := c.previewRect()
	paletteY = y + h + cpGap
	rows := (len(c.picker.palette) + cpPaletteCol - 1) / cpPaletteCol
	recentY = paletteY + rows*(c.swatchSize()+cpSwatchGap)
	if rows > 0 {
		recentY += cpGap - cpSwatchGap
	}
	return
}

func (c *colorPickerPopup) layout() {
	px, py, pw, ph := c.previewRect()
	hexX := px + pw + cpGap
	c.hexBox.SetPosition(hexX, py)
	c.hexBox.SetSize(cpPad+c.contentWidth()-hexX, ph)

	_, recentY := c.swatchesTop()
	height := recentY
	if len(c.recent) > 0 {
		height += c.swatchSize()
	} else {
		height -= cpGap
	}
	c.SetSize(cpPad*2+c.contentWidth(), height+cpPad)
}

// swatchAt returns the palette or recent color at (x, y).
func (c *colorPickerPopup) swatchAt(x, y int) (color.RGBA, bool) {
	size := c.swatchSize()
	paletteY, recentY := c.swatchesTop()
	find := func(colors []color.RGBA, top int) (color.RGBA, bool) {
		if y < top || x < cpPad {
			return color.RGBA{}, false
		}
		col := (x - cpPad) / (size + cpSwatchGap)
		row := (y - top) / (size + cpSwatchGap)
		inX := (x-cpPad)%(size+cpSwatchGap) < size
		inY := (y-top)%(size+cpSwatchGap) < size
		i := row*cpPaletteCol + col
		if col >= cpPaletteCol || !inX || !inY || i >= len(colors) {
			return color.RGBA{}, false
		}
		return colors[i], true
	}
	if y >= recentY && len(c.recent) > 0 {
		return find(c.recent, recentY)
	}
	return find(c.picker.palette, paletteY)
}

// PopupFlipped opens the panel above the field when it doesn't fit below.
func (c *colorPickerPopup) PopupFlipped() (int, int) {
	return c.anchorX + c.anchorW - c.Width(), c.anchorY - c.Height()
}

// CancelPopup is called on Escape: the color goes back to what it was.
func (c *colorPickerPopup) CancelPopup() {
	c.cancelled = true
	c.picker.changeColor(c.original)
}

func (c *colorPickerPopup) ProcessClosePopup() {
	if !c.cancelled && c.picker.col != c.original {
		addRecentColor(c.picker.col)
	}
	c.picker.popup = nil
	c.picker.form.Update()
}

// ---------------------------------------------------------------- Color state

// setColor takes a color from outside the HSV controls (hex, swatch, the
// picker); updateHex also rewrites the hex field.
func (c *colorPickerPopup) setColor(col color.RGBA, updateHex bool) {
	h, s, v := rgbToHSV(col.R, col.G, col.B)
	if s == 0 || v == 0 {
		h = c.h // a gray has no hue: keep the one shown
	}
	if v == 0 {
		s = c.s
	}
	c.h, c.s, c.v, c.a = h, s, v, col.A
	if updateHex {
		c.updateHex()
	}
	c.form.Update()
}

func (c *colorPickerPopup) currentColor() color.RGBA {
	r, g, b := hsvToRGB(c.h, c.s, c.v)
	return color.RGBA{r, g, b, c.a}
}

// hsvChanged applies a change made with the square or the bars.
func (c *colorPickerPopup) hsvChanged() {
	c.picker.changeColor(c.currentColor())
	c.updateHex()
	c.form.Update()
}

func (c *colorPickerPopup) updateHex() {
	c.settingHex = true
	c.hexBox.SetText(colorToHexString(c.currentColor(), c.picker.alphaEnabled))
	c.settingHex = false
}

// hexChanged applies the hex code once it's a valid one.
func (c *colorPickerPopup) hexChanged() {
	if c.settingHex {
		return
	}
	col, ok := ParseHexColor(c.hexBox.Text())
	if !ok {
		return
	}
	if !c.picker.alphaEnabled {
		col.A = 0xFF
	}
	c.setColor(col, false)
	c.picker.changeColor(col)
}

// ---------------------------------------------------------------- Mouse

func (c *colorPickerPopup) mouseDown(button MouseButton, x, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft {
		return true
	}
	inRect := func(rx, ry, rw, rh int) bool { return x >= rx && x < rx+rw && y >= ry && y < ry+rh }
	switch {
	case inRect(c.svRect()):
		c.dragArea = cpAreaSV
	case inRect(c.hueRect()):
		c.dragArea = cpAreaHue
	case c.picker.alphaEnabled && inRect(c.alphaRect()):
		c.dragArea = cpAreaAlpha
	default:
		if px, py, pw, ph := c.previewRect(); inRect(px, py, pw/2, ph) {
			c.setColor(c.original, true) // the old color
			c.picker.changeColor(c.original)
			return true
		}
		if col, ok := c.swatchAt(x, y); ok {
			c.setColor(col, true)
			c.picker.changeColor(c.picker.normalize(col))
			c.picker.ClosePopup()
		}
		return true
	}
	c.dragTo(x, y)
	return true
}

func (c *colorPickerPopup) mouseMove(x, y int, mods KeyModifiers) bool {
	if c.dragArea != cpAreaNone && c.form.mouseLeftButtonPressed {
		c.dragTo(x, y)
	}
	return true
}

func (c *colorPickerPopup) dragTo(x, y int) {
	clamp := func(v float64) float64 { return math.Max(0, math.Min(1, v)) }
	switch c.dragArea {
	case cpAreaSV:
		rx, ry, rw, rh := c.svRect()
		c.s = clamp(float64(x-rx) / float64(rw-1))
		c.v = 1 - clamp(float64(y-ry)/float64(rh-1))
	case cpAreaHue:
		_, ry, _, rh := c.hueRect()
		c.h = clamp(float64(y-ry)/float64(rh-1)) * 360
		if c.h >= 360 {
			c.h = 359.999
		}
	case cpAreaAlpha:
		rx, _, rw, _ := c.alphaRect()
		c.a = uint8(math.Round(clamp(float64(x-rx)/float64(rw-1)) * 255))
	default:
		return
	}
	c.hsvChanged()
}

// ---------------------------------------------------------------- Painting

func (c *colorPickerPopup) draw(cnv *Canvas) {
	p := CurrentPalette()
	cur := c.currentColor()

	// Saturation/brightness square with its marker
	sx, sy, sw, sh := c.svRect()
	cnv.DrawImage(sx, sy, c.svSquare(sw, sh))
	cnv.SetColor(p.Border)
	cnv.DrawRect(sx, sy, sw, sh)
	// The marker may stick out of the square: the padding has room for it
	mx := sx + int(math.Round(c.s*float64(sw-1)))
	my := sy + int(math.Round((1-c.v)*float64(sh-1)))
	opaque := cur
	opaque.A = 0xFF
	drawMarkerRing(cnv, mx, my, opaque)

	// Hue bar
	hx, hy, hw, hh := c.hueRect()
	cnv.DrawImage(hx, hy, c.hueBar(hw, hh))
	cnv.SetColor(p.Border)
	cnv.DrawRect(hx, hy, hw, hh)
	drawBarMarker(cnv, hx-2, hy+int(math.Round(c.h/360*float64(hh-1)))-2, hw+4, 5)

	// Opacity bar
	if c.picker.alphaEnabled {
		ax, ay, aw, ah := c.alphaRect()
		cnv.DrawImage(ax, ay, c.alphaBar(aw, ah, cur))
		cnv.SetColor(p.Border)
		cnv.DrawRect(ax, ay, aw, ah)
		drawBarMarker(cnv, ax+int(math.Round(float64(c.a)/255*float64(aw-1)))-2, ay-2, 5, ah+4)
	}

	// Old | new
	px, py, pw, ph := c.previewRect()
	back := p.PopupBase
	drawColorSwatch(cnv, px, py, pw/2, ph, c.original, back)
	drawColorSwatch(cnv, px+pw/2, py, pw-pw/2, ph, cur, back)
	cnv.SetColor(p.Border)
	cnv.DrawRect(px, py, pw, ph)

	// Palette and recent colors
	size := c.swatchSize()
	paletteY, recentY := c.swatchesTop()
	drawSwatches := func(colors []color.RGBA, top int) {
		for i, col := range colors {
			x := cpPad + (i%cpPaletteCol)*(size+cpSwatchGap)
			y := top + (i/cpPaletteCol)*(size+cpSwatchGap)
			drawColorSwatch(cnv, x, y, size, size, col, back)
			border := MixColors(p.Border, p.Text, 0.2)
			if c.picker.normalize(col) == c.picker.col {
				border = p.Highlight
				cnv.SetColor(border)
				cnv.DrawRect(x-1, y-1, size+2, size+2)
			}
			cnv.SetColor(border)
			cnv.DrawRect(x, y, size, size)
		}
	}
	drawSwatches(c.picker.palette, paletteY)
	if len(c.recent) > 0 {
		cnv.FillRect(cpPad, recentY-cpGap/2-1, c.contentWidth(), 1, p.Divider)
		drawSwatches(c.recent, recentY)
	}
}

// svSquare is the square for the current hue, cached until it changes.
func (c *colorPickerPopup) svSquare(w, h int) *image.RGBA {
	if c.svImage != nil && c.svImageHue == c.h {
		return c.svImage
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		v := 1 - float64(y)/float64(h-1)
		for x := 0; x < w; x++ {
			r, g, b := hsvToRGB(c.h, float64(x)/float64(w-1), v)
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, 0xFF
		}
	}
	c.svImage, c.svImageHue = img, c.h
	return img
}

func (c *colorPickerPopup) hueBar(w, h int) *image.RGBA {
	if c.hueImage != nil {
		return c.hueImage
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		r, g, b := hsvToRGB(float64(y)/float64(h-1)*360, 1, 1)
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, 0xFF
		}
	}
	c.hueImage = img
	return img
}

// alphaBar goes from transparent to the opaque color over a checkerboard.
func (c *colorPickerPopup) alphaBar(w, h int, col color.RGBA) *image.RGBA {
	rgb := [3]uint8{col.R, col.G, col.B}
	if c.alphaImage != nil && c.alphaForRGB == rgb {
		return c.alphaImage
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		a := float64(x) / float64(w-1)
		for y := 0; y < h; y++ {
			bg := checkerColor(x, y)
			r, g, b := blendStraight(col.R, col.G, col.B, a, bg)
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, 0xFF
		}
	}
	c.alphaImage, c.alphaForRGB = img, rgb
	return img
}

// drawMarkerRing draws the round marker of the square: a white ring with a
// dark outline, visible on any color, filled with the color.
func drawMarkerRing(cnv *Canvas, x, y int, fill color.RGBA) {
	const r = cpMarkerSize / 2
	cnv.FillRoundedRectAA(x-r-1, y-r-1, (r+1)*2, (r+1)*2, r+1, color.RGBA{0, 0, 0, 0xA0})
	cnv.FillRoundedRectAA(x-r, y-r, r*2, r*2, r, color.RGBA{0xFF, 0xFF, 0xFF, 0xFF})
	cnv.FillRoundedRectAA(x-r+2, y-r+2, r*2-4, r*2-4, r-2, fill)
}

// drawBarMarker draws the marker of a bar: a white rectangle with a dark
// outline.
func drawBarMarker(cnv *Canvas, x, y, w, h int) {
	cnv.SetColor(color.RGBA{0, 0, 0, 0xFF})
	cnv.DrawRect(x, y, w, h)
	cnv.SetColor(color.RGBA{0xFF, 0xFF, 0xFF, 0xFF})
	cnv.DrawRect(x+1, y+1, w-2, h-2)
}

// drawColorSwatch fills the rectangle with the color; a translucent color is
// shown over a checkerboard. back is what's behind a fully transparent part.
func drawColorSwatch(cnv *Canvas, x, y, w, h int, col color.RGBA, back color.RGBA) {
	if col.A == 0xFF {
		cnv.FillRect(x, y, w, h, color.RGBA{col.R, col.G, col.B, 0xFF})
		return
	}
	const cell = 5
	a := float64(col.A) / 255
	for cy := 0; cy < h; cy += cell {
		for cx := 0; cx < w; cx += cell {
			bg := checkerColor(cx, cy)
			r, g, b := blendStraight(col.R, col.G, col.B, a, bg)
			cnv.FillRect(x+cx, y+cy, min(cell, w-cx), min(cell, h-cy), color.RGBA{r, g, b, 0xFF})
		}
	}
}

func checkerColor(x, y int) uint8 {
	if (x/5+y/5)%2 == 0 {
		return 0xFF
	}
	return 0xCC
}

func blendStraight(r, g, b uint8, a float64, bg uint8) (uint8, uint8, uint8) {
	mix := func(v uint8) uint8 { return uint8(float64(v)*a + float64(bg)*(1-a) + 0.5) }
	return mix(r), mix(g), mix(b)
}

// ---------------------------------------------------------------- Conversions

// ParseHexColor parses "#RGB", "#RRGGBB" or "#RRGGBBAA", with or without
// the "#", in any case.
func ParseHexColor(s string) (color.RGBA, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 && len(s) != 8 {
		return color.RGBA{}, false
	}
	var v [4]uint8
	v[3] = 0xFF
	for i := 0; i < len(s)/2; i++ {
		var b uint8
		if _, err := fmt.Sscanf(s[i*2:i*2+2], "%02x", &b); err != nil {
			return color.RGBA{}, false
		}
		v[i] = b
	}
	return color.RGBA{v[0], v[1], v[2], v[3]}, true
}

func colorToHexString(col color.RGBA, withAlpha bool) string {
	if withAlpha {
		return fmt.Sprintf("#%02X%02X%02X%02X", col.R, col.G, col.B, col.A)
	}
	return fmt.Sprintf("#%02X%02X%02X", col.R, col.G, col.B)
}

// toStraightRGBA takes a color.RGBA as it is (the library's convention),
// other colors through color.NRGBA.
func toStraightRGBA(col color.Color) color.RGBA {
	if rgba, ok := col.(color.RGBA); ok {
		return rgba
	}
	n := color.NRGBAModel.Convert(col).(color.NRGBA)
	return color.RGBA{n.R, n.G, n.B, n.A}
}

// rgbToHSV returns the hue in degrees [0, 360), saturation and value in [0, 1].
func rgbToHSV(r, g, b uint8) (h, s, v float64) {
	rf, gf, bf := float64(r)/255, float64(g)/255, float64(b)/255
	maxC := math.Max(rf, math.Max(gf, bf))
	minC := math.Min(rf, math.Min(gf, bf))
	d := maxC - minC
	v = maxC
	if maxC > 0 {
		s = d / maxC
	}
	if d == 0 {
		return 0, s, v
	}
	switch maxC {
	case rf:
		h = math.Mod((gf-bf)/d, 6)
	case gf:
		h = (bf-rf)/d + 2
	default:
		h = (rf-gf)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return
}

func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	c := v * s
	hp := math.Mod(h, 360) / 60
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	var r, g, b float64
	switch {
	case hp < 1:
		r, g, b = c, x, 0
	case hp < 2:
		r, g, b = x, c, 0
	case hp < 3:
		r, g, b = 0, c, x
	case hp < 4:
		r, g, b = 0, x, c
	case hp < 5:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	m := v - c
	to8 := func(f float64) uint8 { return uint8(math.Round((f + m) * 255)) }
	return to8(r), to8(g), to8(b)
}
