package platforms

import (
	"image"
)

// PopupWindow is a borderless window for tooltips, menus and similar popups
// that may extend beyond the owner window. It never takes activation or
// keyboard focus, has no taskbar button and stays above its owner.
//
// A non-interactive popup (a tooltip) lets the mouse pass through to the
// window below it. An interactive one (a menu) gets mouse events itself;
// keyboard input stays with the owner.
type PopupWindow interface {
	OnPaint(func(rgba *image.RGBA))

	// Mouse events of an interactive popup, in popup coordinates
	OnMouseMove(func(x, y int))
	OnMouseButtonDown(func(btn MouseButton, x, y int))
	OnMouseButtonUp(func(btn MouseButton, x, y int))
	OnMouseLeave(func())
	OnMouseWheel(func(deltaX, deltaY int))

	// SetMouseCursor sets the cursor shown over an interactive popup
	SetMouseCursor(cursor MouseCursor)

	// ShowAt moves the popup to (x, y) in screen coordinates, resizes it and
	// shows it without activating it
	ShowAt(x, y, width, height int)
	Hide()
	Update()
	Close()
}

// CreatePopupWindow creates a hidden popup owned by owner. Returns nil when
// the platform doesn't support popups - callers should fall back to drawing
// inside the owner window.
func CreatePopupWindow(owner Window, interactive bool) PopupWindow {
	return createPopupWindow(owner, interactive)
}

// popupCallbacks holds the callbacks common to all platform popups.
type popupCallbacks struct {
	onPaint           func(rgba *image.RGBA)
	onMouseMove       func(x, y int)
	onMouseButtonDown func(btn MouseButton, x, y int)
	onMouseButtonUp   func(btn MouseButton, x, y int)
	onMouseLeave      func()
	onMouseWheel      func(deltaX, deltaY int)
}

func (c *popupCallbacks) OnPaint(f func(rgba *image.RGBA)) {
	c.onPaint = f
}

func (c *popupCallbacks) OnMouseMove(f func(x, y int)) {
	c.onMouseMove = f
}

func (c *popupCallbacks) OnMouseButtonDown(f func(btn MouseButton, x, y int)) {
	c.onMouseButtonDown = f
}

func (c *popupCallbacks) OnMouseButtonUp(f func(btn MouseButton, x, y int)) {
	c.onMouseButtonUp = f
}

func (c *popupCallbacks) OnMouseLeave(f func()) {
	c.onMouseLeave = f
}

func (c *popupCallbacks) OnMouseWheel(f func(deltaX, deltaY int)) {
	c.onMouseWheel = f
}

func (c *popupCallbacks) mouseMove(x, y int) {
	if c.onMouseMove != nil {
		c.onMouseMove(x, y)
	}
}

func (c *popupCallbacks) mouseButtonDown(btn MouseButton, x, y int) {
	if c.onMouseButtonDown != nil {
		c.onMouseButtonDown(btn, x, y)
	}
}

func (c *popupCallbacks) mouseButtonUp(btn MouseButton, x, y int) {
	if c.onMouseButtonUp != nil {
		c.onMouseButtonUp(btn, x, y)
	}
}

func (c *popupCallbacks) mouseLeave() {
	if c.onMouseLeave != nil {
		c.onMouseLeave()
	}
}

func (c *popupCallbacks) mouseWheel(deltaX, deltaY int) {
	if c.onMouseWheel != nil {
		c.onMouseWheel(deltaX, deltaY)
	}
}
