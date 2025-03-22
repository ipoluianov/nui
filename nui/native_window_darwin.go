package nui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics
#include "window.h"
*/
import "C"
import (
	"image"
)

type NativeWindow struct {
	hwnd int

	currentCursor MouseCursor
	lastSetCursor MouseCursor

	// Keyboard events
	OnKeyDown func(keyCode Key)
	OnKeyUp   func(keyCode Key)
	OnChar    func(char rune)

	// Mouse events
	OnMouseEnter                   func()
	OnMouseLeave                   func()
	OnMouseMove                    func(x, y int)
	OnMouseDownLeftButton          func(x, y int)
	OnMouseUpLeftButton            func(x, y int)
	OnMouseDownRightButton         func(x, y int)
	OnMouseUpRightButton           func(x, y int)
	OnMouseDownMiddleButton        func(x, y int)
	OnMouseUpMiddleButton          func(x, y int)
	OnMouseWheel                   func(delta int)
	OnMouseDoubleClickLeftButton   func(x, y int)
	OnMouseDoubleClickRightButton  func(x, y int)
	OnMouseDoubleClickMiddleButton func(x, y int)

	// Window events
	OnCreated      func()
	OnPaint        func(rgba *image.RGBA)
	OnMove         func(x, y int)
	OnResize       func(width, height int)
	OnCloseRequest func() bool
}

var hwnds map[int]*NativeWindow

func init() {
	hwnds = make(map[int]*NativeWindow)
}

func CreateWindow() *NativeWindow {
	var c NativeWindow
	c.hwnd = int(C.InitWindow())
	hwnds[c.hwnd] = &c
	return &c
}

func (c *NativeWindow) Show() {
}

func (c *NativeWindow) EventLoop() {
	C.RunEventLoop()
}

func (c *NativeWindow) Close() {
}

func (c *NativeWindow) SetTitle(title string) {
}

func (c *NativeWindow) SetMouseCursor(cursor MouseCursor) {
}

func (c *NativeWindow) MaximizeWindow() {
}

func (c *NativeWindow) MinimizeWindow() {
}

func (c *NativeWindow) Move(x, y int) {
}

func (c *NativeWindow) Resize(width, height int) {
}
