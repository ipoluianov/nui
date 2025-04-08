package nui

import (
	"image"

	"github.com/ipoluianov/nui/nuikey"
	"github.com/ipoluianov/nui/nuimouse"
)

type nativeWindow struct {
	hwnd windowId

	platform nativeWindowPlatform

	currentCursor nuimouse.MouseCursor
	lastSetCursor nuimouse.MouseCursor

	mouseInside bool

	keyModifiers nuikey.KeyModifiers

	windowPosX   int
	windowPosY   int
	windowWidth  int
	windowHeight int

	drawTimes      [32]int64
	drawTimesIndex int

	onKeyDown func(keyCode nuikey.Key, mods nuikey.KeyModifiers)
	onKeyUp   func(keyCode nuikey.Key, mods nuikey.KeyModifiers)
	onChar    func(char rune)

	// Mouse events
	onMouseEnter          func()
	onMouseLeave          func()
	onMouseMove           func(x, y int)
	onMouseButtonDown     func(button nuimouse.MouseButton, x, y int)
	onMouseButtonUp       func(button nuimouse.MouseButton, x, y int)
	onMouseButtonDblClick func(button nuimouse.MouseButton, x, y int)
	onMouseWheel          func(deltaX int, deltaY int)

	// Window events
	onCreated      func()
	onPaint        func(rgba *image.RGBA)
	onMove         func(x, y int)
	onResize       func(width, height int)
	onCloseRequest func() bool
	onTimer        func()
}

func (c *nativeWindow) OnKeyDown(f func(keyCode nuikey.Key, mods nuikey.KeyModifiers)) {
	c.onKeyDown = f
}

func (c *nativeWindow) OnKeyUp(f func(keyCode nuikey.Key, mods nuikey.KeyModifiers)) {
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

func (c *nativeWindow) OnMouseButtonDown(f func(button nuimouse.MouseButton, x, y int)) {
	c.onMouseButtonDown = f
}

func (c *nativeWindow) OnMouseButtonUp(f func(button nuimouse.MouseButton, x, y int)) {
	c.onMouseButtonUp = f
}

func (c *nativeWindow) OnMouseButtonDblClick(f func(button nuimouse.MouseButton, x, y int)) {
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

func (c *nativeWindow) Exec() {
	c.Show()
	c.EventLoop()
}
