package platforms

import (
	"image"
	"time"
)

type nativeWindow struct {
	hwnd windowId

	platform nativeWindowPlatform

	currentCursor MouseCursor
	lastSetCursor MouseCursor

	showMaximized bool

	mouseInside bool

	windowPosX   int
	windowPosY   int
	windowWidth  int
	windowHeight int

	lastMouseDownX    int
	lastMouseDownY    int
	lastMouseButton   MouseButton
	lastMouseDownTime time.Time
	dblClickTime      time.Duration

	drawTimes      [32]int64
	drawTimesIndex int

	timerLastDT time.Time

	onKeyDown func(keyCode Key, mods KeyModifiers) bool
	onKeyUp   func(keyCode Key, mods KeyModifiers)
	onChar    func(char rune)

	// Mouse events
	onMouseEnter          func()
	onMouseLeave          func()
	onMouseMove           func(x, y int)
	onMouseButtonDown     func(button MouseButton, x, y int)
	onMouseButtonUp       func(button MouseButton, x, y int)
	onMouseButtonDblClick func(button MouseButton, x, y int)
	onMouseWheel          func(deltaX int, deltaY int)

	// Window events
	onCreated      func()
	onPaint        func(rgba *image.RGBA)
	onMove         func(x, y int)
	onResize       func(width, height int)
	onCloseRequest func() bool
	onTimer        func()
	onDeactivate   func()
}

func (c *nativeWindow) OnKeyDown(f func(keyCode Key, mods KeyModifiers) bool) {
	c.onKeyDown = f
}

func (c *nativeWindow) OnKeyUp(f func(keyCode Key, mods KeyModifiers)) {
	c.onKeyUp = f
}

func (c *nativeWindow) OnChar(f func(char rune)) {
	c.onChar = f
}

func (c *nativeWindow) OnMouseEnter(f func()) {
	c.onMouseEnter = f
}

func (c *nativeWindow) OnMouseLeave(f func()) {
	c.onMouseLeave = f
}

func (c *nativeWindow) OnMouseMove(f func(x, y int)) {
	c.onMouseMove = f
}

func (c *nativeWindow) OnMouseButtonDown(f func(button MouseButton, x, y int)) {
	c.onMouseButtonDown = f
}

func (c *nativeWindow) OnMouseButtonUp(f func(button MouseButton, x, y int)) {
	c.onMouseButtonUp = f
}

func (c *nativeWindow) OnMouseButtonDblClick(f func(button MouseButton, x, y int)) {
	c.onMouseButtonDblClick = f
}

func (c *nativeWindow) OnMouseWheel(f func(deltaX int, deltaY int)) {
	c.onMouseWheel = f
}

func (c *nativeWindow) OnCreated(f func()) {
	c.onCreated = f
}

func (c *nativeWindow) OnPaint(f func(rgba *image.RGBA)) {
	c.onPaint = f
}

func (c *nativeWindow) OnMove(f func(x, y int)) {
	c.onMove = f
}

func (c *nativeWindow) OnResize(f func(width, height int)) {
	c.onResize = f
}

func (c *nativeWindow) OnCloseRequest(f func() bool) {
	c.onCloseRequest = f
}

func (c *nativeWindow) OnTimer(f func()) {
	c.onTimer = f
}

func (c *nativeWindow) OnDeactivate(f func()) {
	c.onDeactivate = f
}

// growBuffer grows *buf to fit size bytes, if needed. Each window (and
// popup) owns its buffers, so concurrent paints on different windows never
// share memory.
func growBuffer(buf *[]byte, size int) []byte {
	if cap(*buf) < size {
		*buf = make([]byte, size)
	} else {
		*buf = (*buf)[:size]
	}
	return *buf
}
