package ui

import (
	"image"
	"image/color"
)

const (
	// ToolButtonDefaultSize is the default width and height of a ToolButton
	ToolButtonDefaultSize = 48

	toolButtonCheckedLift    = 3
	toolButtonCheckMarkWidth = 3
	// Every button has a lighter bottom edge so it stands out from the toolbar
	toolButtonEdgeWidth = 2
	toolButtonEdgeLift  = 8

	// A flat button's background is a rounded rectangle this far inside its edges
	toolButtonFlatInset  = 3
	toolButtonFlatRadius = 6
)

// ToolButton is a fixed-size image button for toolbars.
//
// On top of ButtonImage it has:
//   - a disabled state: a grayed-out copy of the image is shown and clicks are ignored;
//   - a checked state for toggles: a lighter background and a bright bar along the bottom;
//   - a lighter bottom edge, so the buttons stand out from the toolbar;
//   - a flat look (SetFlat): just the image, without the box.
type ToolButton struct {
	ButtonImage

	img         image.Image
	imgDisabled image.Image
	checked     bool
	flat        bool
	highlight   color.Color
	onClick     func()
}

// NewToolButton creates a ToolButtonDefaultSize square button with the image
// (drawn centered, as is) and the tooltip. onClick may be nil.
func NewToolButton(img image.Image, tooltip string, onClick func()) *ToolButton {
	var c ToolButton
	c.initButtonImage(img)
	c.SetTypeName("ToolButton")
	c.onClick = onClick
	c.SetImage(img)
	c.SetButtonSize(ToolButtonDefaultSize, ToolButtonDefaultSize)
	c.SetTooltip(tooltip)
	c.SetOnPostPaint(c.drawBottomEdge)
	c.SetOnButtonClick(func(btn *ButtonImage) {
		if c.Enabled() && c.onClick != nil {
			c.onClick()
		}
	})
	return &c
}

// SetButtonSize fixes the size of the button, e.g. to make the main action wider
func (c *ToolButton) SetButtonSize(width, height int) {
	c.SetMinSize(width, height)
	c.SetMaxSize(width, height)
}

// SetOnClick sets the function called when the enabled button is clicked
func (c *ToolButton) SetOnClick(onClick func()) {
	c.onClick = onClick
}

// SetImage sets the image; the disabled look is derived from it
func (c *ToolButton) SetImage(img image.Image) {
	c.img = img
	c.imgDisabled = toolButtonDisabledImage(img)
	c.showImage()
}

func (c *ToolButton) SetEnabled(enabled bool) {
	if c.Enabled() == enabled {
		return
	}
	c.ButtonImage.SetEnabled(enabled)
	c.showImage()
}

func (c *ToolButton) showImage() {
	if c.Enabled() {
		c.ButtonImage.SetImage(c.img)
	} else {
		c.ButtonImage.SetImage(c.imgDisabled)
	}
}

func (c *ToolButton) IsChecked() bool {
	return c.checked
}

// SetChecked shows the button as toggled on: a lighter background
// and a bar along the bottom edge. The button does not toggle itself on click.
func (c *ToolButton) SetChecked(checked bool) {
	if c.checked == checked {
		return
	}
	c.checked = checked
	if checked {
		c.SetElevation(toolButtonCheckedLift)
	} else {
		c.SetElevation(0)
	}
	c.form.Update()
}

// SetFlat draws the button without its box: just the image, on a soft
// rounded background while the mouse is over it, it is pressed or checked.
// A row of flat buttons makes a light toolbar.
func (c *ToolButton) SetFlat(flat bool) {
	if c.flat == flat {
		return
	}
	c.flat = flat
	if flat {
		c.SetOnPaint(c.drawFlat)
	} else {
		c.SetOnPaint(c.draw)
	}
	c.form.Update()
}

func (c *ToolButton) IsFlat() bool {
	return c.flat
}

func (c *ToolButton) drawFlat(cnv *Canvas) {
	backColor := colorToRGBA(c.BackgroundColor())
	textColor := colorToRGBA(c.ForegroundColor())
	var fill color.Color
	switch {
	case c.pressed && c.Enabled():
		fill = pressedColor(backColor, textColor)
	case c.IsHovered() && c.Enabled():
		fill = hoverColor(backColor, textColor)
	case c.checked:
		fill = c.BackgroundColorWithAddElevation(toolButtonCheckedLift)
	}
	if fill != nil {
		cnv.SetColor(fill)
		cnv.FillRoundedRect(toolButtonFlatInset, toolButtonFlatInset,
			c.Width()-toolButtonFlatInset*2, c.Height()-toolButtonFlatInset*2, toolButtonFlatRadius)
	}
	if c.ButtonImage.img != nil {
		x, y := c.imagePosition()
		cnv.DrawImage(x, y, c.ButtonImage.img)
	}
}

// SetHighlight draws the bottom bar in the color to draw attention to the button,
// e.g. the action to start with. nil returns the usual look.
func (c *ToolButton) SetHighlight(col color.Color) {
	if c.highlight == col {
		return
	}
	c.highlight = col
	c.form.Update()
}

// drawBottomEdge draws the lighter bottom edge, or a bright bar when the button is checked or highlighted
func (c *ToolButton) drawBottomEdge(cnv *Canvas) {
	if c.flat {
		c.drawFlatMark(cnv)
		return
	}
	if c.checked {
		cnv.FillRect(0, c.Height()-toolButtonCheckMarkWidth, c.Width(), toolButtonCheckMarkWidth, CurrentPalette().Highlight)
		return
	}
	if c.highlight != nil {
		cnv.FillRect(0, c.Height()-toolButtonCheckMarkWidth, c.Width(), toolButtonCheckMarkWidth, c.highlight)
		return
	}
	cnv.FillRect(0, c.Height()-toolButtonEdgeWidth, c.Width(), toolButtonEdgeWidth, c.BackgroundColorWithAddElevation(toolButtonEdgeLift))
}

// drawFlatMark draws a short bar under the image of a flat button that is
// checked or highlighted; an ordinary flat button has no edge
func (c *ToolButton) drawFlatMark(cnv *Canvas) {
	var col color.Color
	switch {
	case c.checked:
		col = CurrentPalette().Highlight
	case c.highlight != nil:
		col = c.highlight
	default:
		return
	}
	w := c.Width() / 3
	cnv.FillRect((c.Width()-w)/2, c.Height()-toolButtonFlatInset-toolButtonCheckMarkWidth, w, toolButtonCheckMarkWidth, col)
}

// toolButtonDisabledImage returns a grayscale, semi-transparent copy of img
func toolButtonDisabledImage(img image.Image) image.Image {
	if img == nil {
		return nil
	}
	b := img.Bounds()
	res := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			gray := uint8((299*uint32(c.R) + 587*uint32(c.G) + 114*uint32(c.B)) / 1000)
			res.SetNRGBA(x, y, color.NRGBA{R: gray, G: gray, B: gray, A: c.A * 2 / 5})
		}
	}
	return res
}
