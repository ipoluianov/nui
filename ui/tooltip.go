package ui

import (
	"image"
	"time"

	"github.com/ipoluianov/nui/internal/platforms"
)

const (
	tooltipDelay   = 600 * time.Millisecond
	tooltipPadding = 6
)

// tooltipState tracks the tooltip of the widget under the mouse cursor.
// The tooltip appears after the mouse rests on a widget for tooltipDelay,
// and is hidden when the hovered widget changes, the mouse leaves the
// window or a mouse button is pressed (until the hover changes again).
//
// The tooltip is shown in a native popup window, so it isn't clipped by the
// form's bounds. Where the platform has no popups yet, it's drawn inside
// the form instead.
type tooltipState struct {
	hoverStart time.Time
	visible    bool
	suppressed bool
	x, y       int

	popup          platforms.PopupWindow
	popupFailed    bool
	popupVisible   bool
	popupText      string
	popupW, popupH int
}

// SetTooltip sets the text shown when the mouse rests on the widget.
// An empty string disables the tooltip.
func (c *Widget) SetTooltip(text string) {
	c.SetProp("tooltip", text)
}

func (c *Widget) Tooltip() string {
	return c.GetPropString("tooltip", "")
}

func (c *Form) tooltipText() string {
	if c.hoverWidget == nil {
		return ""
	}
	return c.hoverWidget.GetPropString("tooltip", "")
}

// tooltipHoverChanged restarts the delay when the mouse moves to another widget.
func (c *Form) tooltipHoverChanged() {
	c.tooltip.hoverStart = time.Now()
	c.tooltip.suppressed = false
	c.tooltipHide()
}

// tooltipSuppress hides the tooltip until the mouse moves to another widget.
func (c *Form) tooltipSuppress() {
	c.tooltip.suppressed = true
	c.tooltipHide()
}

func (c *Form) tooltipHide() {
	if !c.tooltip.visible {
		return
	}
	c.tooltip.visible = false
	if c.tooltip.popupVisible {
		c.tooltip.popupVisible = false
		c.tooltip.popup.Hide()
	} else {
		c.Update()
	}
}

// tooltipClose destroys the popup before the form's window goes away.
func (c *Form) tooltipClose() {
	c.tooltipHide()
	if c.tooltip.popup != nil {
		c.tooltip.popup.Close()
		c.tooltip.popup = nil
	}
}

func (c *Form) tooltipProcessTimer() {
	if c.tooltip.visible || c.tooltip.suppressed || c.mouseLeftButtonPressed {
		return
	}
	if c.tooltipText() == "" || time.Since(c.tooltip.hoverStart) < tooltipDelay {
		return
	}
	c.tooltip.visible = true
	c.tooltip.x = c.lastMouseX
	c.tooltip.y = c.lastMouseY
	if !c.tooltipShowPopup() {
		c.Update()
	}
}

func (c *Form) tooltipSize(text string) (w, h int, ok bool) {
	textWidth, textHeight, err := MeasureText(ThemeFontFamily(), ThemeFontSize(), text)
	if err != nil {
		return 0, 0, false
	}
	return textWidth + tooltipPadding*4, textHeight + tooltipPadding*2, true
}

// tooltipShowPopup shows the tooltip in a native popup window below-right
// of the cursor, kept inside the monitor's work area. Returns false when
// popups are unavailable and the tooltip must be drawn inside the form.
func (c *Form) tooltipShowPopup() bool {
	if c.wnd == nil || c.tooltip.popupFailed {
		return false
	}
	if c.tooltip.popup == nil {
		c.tooltip.popup = platforms.CreatePopupWindow(c.wnd, false)
		if c.tooltip.popup == nil {
			c.tooltip.popupFailed = true
			return false
		}
		c.tooltip.popup.OnPaint(c.tooltipPopupPaint)
	}

	text := c.tooltipText()
	w, h, ok := c.tooltipSize(text)
	if !ok {
		return true
	}

	cursorX, cursorY := c.wnd.ClientToScreen(c.tooltip.x, c.tooltip.y)
	areaX, areaY, areaW, areaH := c.wnd.ScreenWorkArea(cursorX, cursorY)

	x := cursorX + 12
	y := cursorY + 20
	if x+w > areaX+areaW {
		x = areaX + areaW - w
	}
	if y+h > areaY+areaH {
		y = cursorY - h - 4
	}
	if x < areaX {
		x = areaX
	}
	if y < areaY {
		y = areaY
	}

	c.tooltip.popupText = text
	c.tooltip.popupW = w
	c.tooltip.popupH = h
	c.tooltip.popupVisible = true
	c.tooltip.popup.ShowAt(x, y, w, h)
	return true
}

func (c *Form) tooltipPopupPaint(rgba *image.RGBA) {
	p := CurrentPalette()
	cnv := NewCanvasScaled(rgba, c.tooltip.popup.Scale())
	cnv.SetDirectTranslateAndClip(0, 0, c.tooltip.popupW, c.tooltip.popupH)
	// The popup is rectangular, so no rounded corners here
	cnv.FillFrame(0, 0, c.tooltip.popupW, c.tooltip.popupH, 0, p.ToolTipBase, p.Border)
	tooltipDrawText(cnv, 0, 0, c.tooltip.popupW, c.tooltip.popupH, c.tooltip.popupText)
}

func tooltipDrawText(cnv *Canvas, x, y, w, h int, text string) {
	cnv.SetHAlign(HAlignCenter)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(CurrentPalette().ToolTipText)
	cnv.SetFontFamily(ThemeFontFamily())
	cnv.SetFontSize(ThemeFontSize())
	cnv.DrawText(x, y, w, h, text)
}

// tooltipPaint draws the tooltip inside the form when there's no popup.
func (c *Form) tooltipPaint(cnv *Canvas) {
	if !c.tooltip.visible || c.tooltip.popupVisible {
		return
	}
	text := c.tooltipText()
	if text == "" {
		return
	}

	w, h, ok := c.tooltipSize(text)
	if !ok {
		return
	}

	// Below-right of the cursor, kept inside the window
	x := c.tooltip.x + 12
	y := c.tooltip.y + 20
	if x+w > c.width {
		x = c.width - w
	}
	if y+h > c.height {
		y = c.tooltip.y - h - 4
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	p := CurrentPalette()
	cnv.FillFrame(x, y, w, h, themeControlRadius, p.ToolTipBase, p.Border)

	tooltipDrawText(cnv, x, y, w, h, text)
}
