package ui

import (
	"image"
	"image/color"
)

// GroupBox is a panel framed with a title, grouping related controls. Its
// children are laid out on the grid like in a Panel:
//
//	group := ui.NewGroupBox("Connection")
//	group.AddLabel(0, 0, "Host")
//	group.AddWidget(0, 1, hostBox)
type GroupBox struct {
	Widget
	title string
}

const (
	groupBoxPadding    = 8
	groupBoxTitleInset = 10
)

func NewGroupBox(title string) *GroupBox {
	var c GroupBox
	c.InitWidget()
	c.SetTypeName("GroupBox")
	c.SetPanelPadding(groupBoxPadding)
	c.SetCellPadding(4)
	c.SetOnPaint(c.draw)
	c.insetTop = c.titleHeight() - groupBoxPadding/2
	c.SetTitle(title)
	return &c
}

func (c *GroupBox) Title() string {
	return c.title
}

func (c *GroupBox) SetTitle(title string) {
	c.title = title
	textWidth, _, _ := MeasureText(c.FontFamily(), c.FontSize(), title)
	c.SetMinWidth(groupBoxTitleInset*2 + textWidth + 8)
	c.form.Update()
}

func (c *GroupBox) SetTitleFunc(f func() string) {
	c.setTextFunc(f, c.SetTitle)
}

func (c *GroupBox) titleHeight() int {
	return ThemeLineHeight()
}

// draw: a rounded frame whose top line runs behind the title, with a gap
// for it. The gap is cut out by clipping, so the frame suits any background.
func (c *GroupBox) draw(cnv *Canvas) {
	p := CurrentPalette()
	border := MixColors(p.Border, p.Window, 0.2)
	th := c.titleHeight()
	top := th / 2
	w, h := c.Width(), c.Height()

	textWidth, _, err := MeasureText(c.FontFamily(), c.FontSize(), c.title)
	if err != nil || c.title == "" {
		textWidth = 0
	}
	gapFrom, gapTo := groupBoxTitleInset-4, groupBoxTitleInset+textWidth+4
	if textWidth == 0 {
		gapFrom, gapTo = 0, 0
	}
	frame := func(clipX, clipY, clipW, clipH int) {
		strokeRoundedRectClipped(cnv, 0, top, w, h-top, themeControlRadius+1, border, clipX, clipY, clipW, clipH)
	}
	frame(0, 0, gapFrom, h)
	frame(gapTo, 0, w-gapTo, h)
	frame(gapFrom, top+2, gapTo-gapFrom, h-top-2)

	textColor := colorToRGBA(c.ForegroundColor())
	if !c.Enabled() {
		textColor = p.DisabledText
	}
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(textColor)
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.DrawText(groupBoxTitleInset, 0, w-groupBoxTitleInset*2, th, c.title)
}

// strokeRoundedRectClipped draws the 1px outline of a rounded rectangle,
// anti-aliased, only within the clip rectangle (and the canvas' clip).
func strokeRoundedRectClipped(cnv *Canvas, x, y, w, h, radius int, col color.RGBA, clipX, clipY, clipW, clipH int) {
	cx, cy, cw, ch := cnv.pixelRect(clipX, clipY, clipW, clipH)
	if cw <= 0 || ch <= 0 {
		return
	}
	px, py, pw, ph := cnv.pixelRect(x, y, w, h)
	cnv.fillRoundedPixels(px, py, pw, ph, float64(radius)*cnv.scale, cnv.lineWidth(1), col, image.Rect(cx, cy, cx+cw, cy+ch))
}

func (c *GroupBox) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.insetTop = c.titleHeight() - groupBoxPadding/2
	c.SetTitle(c.title)
}
