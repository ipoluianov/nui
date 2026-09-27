package ui

import (
	"image"

	"github.com/ipoluianov/nui/internal/platforms"
)

// PopupPlacer is implemented by popup widgets (see Form.OpenPopup) that know
// a better position than being pushed back onto the screen when they don't
// fit on it.
type PopupPlacer interface {
	// PopupFlipped returns, in the form's client coordinates, the X to use
	// when the popup doesn't fit to the right and the Y to use when it
	// doesn't fit below
	PopupFlipped() (x, y int)
}

// PopupCanceler is implemented by popup widgets that undo what was done in
// them when Escape closes them (a click outside keeps it), e.g. a color
// picker's panel returning to the color it was opened with.
type PopupCanceler interface {
	CancelPopup()
}

// popupHost is the native window of an open popup widget. The widget keeps
// its position in the form's client coordinates, which may be outside the
// client area. Mouse events of the window are translated to those
// coordinates, and the form sends them to the popup widget (see mouseTarget).
type popupHost struct {
	widget        Widgeter
	wnd           platforms.PopupWindow
	width, height int
	// cursor is the last cursor set on the window
	cursor MouseCursor
}

// syncPopupWindows shows windows for newly opened popup widgets and hides
// the windows of closed ones. Called whenever the popup list changes.
func (c *Form) syncPopupWindows() {
	if c.wnd == nil {
		return
	}

	open := c.popupHosts[:0]
	for _, h := range c.popupHosts {
		if c.isPopupWidgetOpen(h.widget) {
			open = append(open, h)
			continue
		}
		h.wnd.Hide()
		c.freePopupWindows = append(c.freePopupWindows, h.wnd)
		if c.popupUnderMouse == h {
			c.popupUnderMouse = nil
		}
	}
	c.popupHosts = open

	for _, w := range c.topWidget.PopupWidgets {
		if c.popupHostOf(w) != nil {
			continue
		}
		wnd := c.takePopupWindow()
		if wnd == nil {
			continue // the platform failed to create a window: the popup stays invisible
		}
		h := &popupHost{widget: w, wnd: wnd}
		c.bindPopupWindow(h)
		c.popupHosts = append(c.popupHosts, h)
		c.placePopupWindow(h)
	}
}

func (c *Form) isPopupWidgetOpen(w Widgeter) bool {
	for _, popupWidget := range c.topWidget.PopupWidgets {
		if popupWidget == w {
			return true
		}
	}
	return false
}

func (c *Form) popupHostOf(w Widgeter) *popupHost {
	for _, h := range c.popupHosts {
		if h.widget == w {
			return h
		}
	}
	return nil
}

func (c *Form) isPopupHostOpen(h *popupHost) bool {
	for _, openHost := range c.popupHosts {
		if openHost == h {
			return true
		}
	}
	return false
}

// takePopupWindow reuses a hidden popup window or creates a new one.
func (c *Form) takePopupWindow() platforms.PopupWindow {
	if n := len(c.freePopupWindows); n > 0 {
		wnd := c.freePopupWindows[n-1]
		c.freePopupWindows = c.freePopupWindows[:n-1]
		return wnd
	}
	if c.popupWindowsUnavailable {
		return nil
	}
	wnd := platforms.CreatePopupWindow(c.wnd, true)
	if wnd == nil {
		c.popupWindowsUnavailable = true
	}
	return wnd
}

func (c *Form) bindPopupWindow(h *popupHost) {
	h.wnd.OnPaint(func(rgba *image.RGBA) {
		cnv := NewCanvas(rgba)
		cnv.SetDirectTranslateAndClip(0, 0, h.widget.Width(), h.widget.Height())
		h.widget.ProcessPaint(cnv)
	})
	h.wnd.OnMouseMove(func(x, y int) {
		if !c.isPopupHostOpen(h) {
			return
		}
		c.popupUnderMouse = h
		c.processMouseMove(x+h.widget.X(), y+h.widget.Y())
	})
	h.wnd.OnMouseButtonDown(func(btn MouseButton, x, y int) {
		if !c.isPopupHostOpen(h) {
			return
		}
		c.popupUnderMouse = h
		c.processMouseDown(btn, x+h.widget.X(), y+h.widget.Y())
		if !c.isPopupHostOpen(h) {
			// The click closed the popup (e.g. chose a menu item): its
			// window is hidden, so the button release may never come
			c.mouseLeftButtonPressed = false
			c.mouseLeftButtonPressedWidget = nil
			c.mouseDownPopup = nil
		}
	})
	h.wnd.OnMouseButtonUp(func(btn MouseButton, x, y int) {
		if !c.isPopupHostOpen(h) {
			return
		}
		c.processMouseUp(btn, x+h.widget.X(), y+h.widget.Y())
	})
	h.wnd.OnMouseWheel(func(deltaX, deltaY int) {
		if !c.isPopupHostOpen(h) {
			return
		}
		c.popupUnderMouse = h
		c.processMouseWheel(deltaX, deltaY)
	})
	h.wnd.OnMouseLeave(func() {
		if c.popupUnderMouse == h {
			c.popupUnderMouse = nil
			c.processMouseLeave()
		}
	})
}

// placePopupWindow shows the window over the widget's position, moved onto
// the screen if needed. The widget is moved along, so the client
// coordinates of the mouse events match what is on the screen.
func (c *Form) placePopupWindow(h *popupHost) {
	w := h.widget
	width, height := w.Width(), w.Height()
	originX, originY := c.wnd.ClientToScreen(0, 0)
	x, y := originX+w.X(), originY+w.Y()
	areaX, areaY, areaW, areaH := c.wnd.ScreenWorkArea(x, y)

	if placer, ok := w.(PopupPlacer); ok {
		flippedX, flippedY := placer.PopupFlipped()
		if x+width > areaX+areaW {
			x = originX + flippedX
		}
		if y+height > areaY+areaH {
			y = originY + flippedY
		}
	}
	x = max(areaX, min(x, areaX+areaW-width))
	y = max(areaY, min(y, areaY+areaH-height))

	w.SetPosition(x-originX, y-originY)
	h.width, h.height = width, height
	h.wnd.ShowAt(x, y, width, height)
}

// updatePopupWindows repaints the popup windows along with the form and
// follows the size changes of their widgets.
func (c *Form) updatePopupWindows() {
	for _, h := range c.popupHosts {
		if h.widget.Width() != h.width || h.widget.Height() != h.height {
			c.placePopupWindow(h)
		}
		h.wnd.Update()
	}
}

// mouseTarget returns the widget that gets a mouse event at the client point
// (x, y), and the point in its coordinates: the popup widget of the window
// the event came from, or, for the form's own window, the menu bar or the
// form's top widget below it.
func (c *Form) mouseTarget(host *popupHost, x, y int) (Widgeter, int, int) {
	if host != nil {
		return host.widget, x - host.widget.X(), y - host.widget.Y()
	}
	if c.menuBar != nil && y < c.topWidget.Y() {
		return c.menuBar, x - c.menuBar.X(), y - c.menuBar.Y()
	}
	return c.topWidget, x - c.topWidget.X(), y - c.topWidget.Y()
}

// widgetUnderMouse finds the widget at the client point (x, y) in the window
// the mouse is over.
func (c *Form) widgetUnderMouse(x, y int) Widgeter {
	target, targetX, targetY := c.mouseTarget(c.popupUnderMouse, x, y)
	if w := target.findWidgetAt(targetX, targetY); w != nil {
		return w
	}
	return target
}

// closePopupsByClickOutside closes the popups when the form itself is
// clicked (a click outside all popup windows), topmost first, up to one
// that stays open on such clicks. Returns true when the click was outside
// the popups, so it must not reach the form's widgets.
func (c *Form) closePopupsByClickOutside() bool {
	if c.popupUnderMouse != nil || len(c.topWidget.PopupWidgets) == 0 {
		return false
	}
	for len(c.topWidget.PopupWidgets) > 0 {
		topPopup := c.topWidget.PopupWidgets[len(c.topWidget.PopupWidgets)-1]
		if !topPopup.CloseByClickOutside() {
			break
		}
		c.topWidget.CloseTopPopup()
	}
	return true
}

// closePopups closes all popup widgets, e.g. when the form loses activation:
// their windows would otherwise stay above other applications.
func (c *Form) closePopups() {
	if len(c.topWidget.PopupWidgets) > 0 {
		c.topWidget.CloseAllPopup()
	}
}

// destroyPopupWindows runs before the form's window goes away.
func (c *Form) destroyPopupWindows() {
	for _, h := range c.popupHosts {
		h.wnd.Close()
	}
	for _, wnd := range c.freePopupWindows {
		wnd.Close()
	}
	c.popupHosts = nil
	c.freePopupWindows = nil
	c.popupUnderMouse = nil
}

func (c *Form) processDeactivate() {
	c.tooltipHide()
	c.closePopups()
}
