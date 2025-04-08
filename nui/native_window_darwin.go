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
)

type NativeWindow struct {
	hwnd int

	currentCursor MouseCursor
	lastSetCursor MouseCursor

	windowPosX   int
	windowPosY   int
	windowWidth  int
	windowHeight int

	keyModifiers KeyModifiers

	lastCapsLockState bool
	lastNumLockState  bool

	// Keyboard events
	OnKeyDown func(keyCode Key, modifiers KeyModifiers)
	OnKeyUp   func(keyCode Key, modifiers KeyModifiers)
	OnChar    func(char rune)

	drawTimes      [32]int64
	drawTimesIndex int

	// Mouse events
	OnMouseEnter          func()
	OnMouseLeave          func()
	OnMouseMove           func(x, y int)
	OnMouseButtonDown     func(button MouseButton, x, y int)
	OnMouseButtonUp       func(button MouseButton, x, y int)
	OnMouseButtonDblClick func(button MouseButton, x, y int)
	OnMouseWheel          func(deltaX int, deltaY int)

	// Window events
	OnCreated      func()
	OnPaint        func(rgba *image.RGBA)
	OnMove         func(x, y int)
	OnResize       func(width, height int)
	OnCloseRequest func() bool
	OnTimer        func()
}

var hwnds map[int]*NativeWindow

func init() {
	hwnds = make(map[int]*NativeWindow)
}

/////////////////////////////////////////////////////
// Window creation and management

func CreateWindow() *NativeWindow {
	var c NativeWindow

	initCanvasBufferBackground(color.RGBA{0, 50, 0, 255})

	c.hwnd = int(C.InitWindow())

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

func (c *NativeWindow) Show() {
	C.ShowWindow(C.int(c.hwnd))
}

func (c *NativeWindow) Update() {
	C.UpdateWindow(C.int(c.hwnd))
}

func (c *NativeWindow) EventLoop() {
	C.RunEventLoop()
}

func (c *NativeWindow) Close() {
	C.CloseWindowById(C.int(c.hwnd))
}

///////////////////////////////////////////////////
// Window appearance

func (c *NativeWindow) SetTitle(title string) {
	C.SetWindowTitle(C.int(c.hwnd), C.CString(title))
}

func (c *NativeWindow) SetAppIcon(img image.Image) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	C.SetAppIconFromRGBA(
		(*C.char)(unsafe.Pointer(&rgba.Pix[0])),
		C.int(width),
		C.int(height),
	)
}

func (c *NativeWindow) SetBackgroundColor(color color.RGBA) {
	initCanvasBufferBackground(color)
	c.Update()
}

func (c *NativeWindow) SetMouseCursor(cursor MouseCursor) {
	c.currentCursor = cursor
	c.macSetMouseCursor(c.currentCursor)
}

func (c *NativeWindow) macSetMouseCursor(cursor MouseCursor) {
	if c.lastSetCursor == cursor {
		return
	}
	c.lastSetCursor = cursor
	var macCursor C.int
	macCursor = 0
	switch c.currentCursor {
	case MouseCursorArrow:
		macCursor = 1
	case MouseCursorPointer:
		macCursor = 2
	case MouseCursorResizeHor:
		macCursor = 3
	case MouseCursorResizeVer:
		macCursor = 4
	case MouseCursorIBeam:
		macCursor = 5
	}
	C.SetMacCursor(macCursor)
}

/////////////////////////////////////////////////////
// Window position and size

func (c *NativeWindow) Move(x, y int) {
	C.SetWindowPosition(C.int(c.hwnd), C.int(x), C.int(y))
}

func (c *NativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := GetScreenSize()
	windowWidth, windowHeight := c.Size()
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	c.Move(int(x), int(y))
}

func (c *NativeWindow) Resize(width, height int) {
	C.SetWindowSize(C.int(c.hwnd), C.int(width), C.int(height))
}

func (c *NativeWindow) MinimizeWindow() {
	C.MinimizeWindow(C.int(c.hwnd))
}

func (c *NativeWindow) MaximizeWindow() {
	C.MaximizeWindow(C.int(c.hwnd))
}

//////////////////////////////////////////////////
// Window information

func (c *NativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
}

func (c *NativeWindow) Pos() (x, y int) {
	return c.windowPosX, c.windowPosY
}

func (c *NativeWindow) PosX() int {
	return c.windowPosX
}

func (c *NativeWindow) PosY() int {
	return c.windowPosY
}

func (c *NativeWindow) Width() int {
	return c.windowWidth
}

func (c *NativeWindow) Height() int {
	return c.windowHeight
}

func (c *NativeWindow) KeyModifiers() KeyModifiers {
	return c.keyModifiers
}

func (c *NativeWindow) DrawTimeUs() int64 {
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
