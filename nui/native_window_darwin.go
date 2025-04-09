package nui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics
#include "window.h"
*/
import "C"
import (
	"image"
	"image/color"
	"image/draw"
	"unsafe"

	"github.com/ipoluianov/nui/nuikey"
	"github.com/ipoluianov/nui/nuimouse"
)

type windowId int
type nativeWindowPlatform struct {
	lastCapsLockState bool
	lastNumLockState  bool
}

/*type NativeWindow struct {
	hwnd int

	currentCursor nuimouse.MouseCursor
	lastSetCursor nuimouse.MouseCursor

	windowPosX   int
	windowPosY   int
	windowWidth  int
	windowHeight int

	keyModifiers nuikey.KeyModifiers

	lastCapsLockState bool
	lastNumLockState  bool

	// Keyboard events
	OnKeyDown func(keyCode nuikey.Key, modifiers nuikey.KeyModifiers)
	OnKeyUp   func(keyCode nuikey.Key, modifiers nuikey.KeyModifiers)
	OnChar    func(char rune)

	drawTimes      [32]int64
	drawTimesIndex int

	// Mouse events
	OnMouseEnter          func()
	OnMouseLeave          func()
	OnMouseMove           func(x, y int)
	OnMouseButtonDown     func(button nuimouse.MouseButton, x, y int)
	OnMouseButtonUp       func(button nuimouse.MouseButton, x, y int)
	OnMouseButtonDblClick func(button nuimouse.MouseButton, x, y int)
	OnMouseWheel          func(deltaX int, deltaY int)

	// Window events
	OnCreated      func()
	OnPaint        func(rgba *image.RGBA)
	OnMove         func(x, y int)
	OnResize       func(width, height int)
	OnCloseRequest func() bool
	OnTimer        func()
}*/

var hwnds map[windowId]*nativeWindow

func init() {
	hwnds = make(map[windowId]*nativeWindow)
}

/////////////////////////////////////////////////////
// Window creation and management

func createWindow(title string, width int, height int, center bool) *nativeWindow {
	var c nativeWindow

	initCanvasBufferBackground(color.RGBA{0, 50, 0, 255})

	c.hwnd = windowId(C.InitWindow())

	x, y := c.requestWindowPosition()
	c.windowPosX = int(x)
	c.windowPosY = int(y)

	w, h := c.requestWindowSize()
	c.windowWidth = int(w)
	c.windowHeight = int(h)

	hwnds[c.hwnd] = &c
	c.startTimer(1)
	return &c
}

func (c *nativeWindow) Show() {
	C.ShowWindow(C.int(c.hwnd))
}

func (c *nativeWindow) Update() {
	C.UpdateWindow(C.int(c.hwnd))
}

func (c *nativeWindow) EventLoop() {
	C.RunEventLoop()
}

func (c *nativeWindow) Close() {
	C.CloseWindowById(C.int(c.hwnd))
}

///////////////////////////////////////////////////
// Window appearance

func (c *nativeWindow) SetTitle(title string) {
	C.SetWindowTitle(C.int(c.hwnd), C.CString(title))
}

func (c *nativeWindow) SetAppIcon(icon *image.RGBA) {
	bounds := icon.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, icon, bounds.Min, draw.Src)

	C.SetAppIconFromRGBA(
		(*C.char)(unsafe.Pointer(&rgba.Pix[0])),
		C.int(width),
		C.int(height),
	)
}

func (c *nativeWindow) SetBackgroundColor(color color.RGBA) {
	initCanvasBufferBackground(color)
	c.Update()
}

func (c *nativeWindow) SetMouseCursor(cursor nuimouse.MouseCursor) {
	c.currentCursor = cursor
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) macSetMouseCursor(cursor nuimouse.MouseCursor) {
	if c.lastSetCursor == cursor {
		return
	}
	c.lastSetCursor = cursor
	var macCursor C.int
	macCursor = 0
	switch c.currentCursor {
	case nuimouse.MouseCursorArrow:
		macCursor = 1
	case nuimouse.MouseCursorPointer:
		macCursor = 2
	case nuimouse.MouseCursorResizeHor:
		macCursor = 3
	case nuimouse.MouseCursorResizeVer:
		macCursor = 4
	case nuimouse.MouseCursorIBeam:
		macCursor = 5
	}
	C.SetMacCursor(macCursor)
}

/////////////////////////////////////////////////////
// Window position and size

func (c *nativeWindow) Move(x, y int) {
	C.SetWindowPosition(C.int(c.hwnd), C.int(x), C.int(y))
}

func (c *nativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := GetScreenSize()
	windowWidth, windowHeight := c.Size()
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	c.Move(int(x), int(y))
}

func (c *nativeWindow) Resize(width, height int) {
	C.SetWindowSize(C.int(c.hwnd), C.int(width), C.int(height))
}

func (c *nativeWindow) MinimizeWindow() {
	C.MinimizeWindow(C.int(c.hwnd))
}

func (c *nativeWindow) MaximizeWindow() {
	C.MaximizeWindow(C.int(c.hwnd))
}

//////////////////////////////////////////////////
// Window information

func (c *nativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
}

func (c *nativeWindow) Pos() (x, y int) {
	return c.windowPosX, c.windowPosY
}

func (c *nativeWindow) PosX() int {
	return c.windowPosX
}

func (c *nativeWindow) PosY() int {
	return c.windowPosY
}

func (c *nativeWindow) Width() int {
	return c.windowWidth
}

func (c *nativeWindow) Height() int {
	return c.windowHeight
}

func (c *nativeWindow) KeyModifiers() nuikey.KeyModifiers {
	return c.keyModifiers
}

func (c *nativeWindow) DrawTimeUs() int64 {
	drawTimeAvg := int64(0)
	count := 0
	for _, t := range c.drawTimes {
		if t == 0 {
			continue
		}
		drawTimeAvg += t
		count++
	}
	if count == 0 {
		return 0
	}
	drawTimeAvg = drawTimeAvg / int64(count)
	return drawTimeAvg
}
