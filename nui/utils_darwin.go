package nui

/*
#include "window.h"
*/
import "C"
import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"time"
	"unicode"
	"unsafe"
)

//export go_on_paint
func go_on_paint(hwnd C.int, ptr unsafe.Pointer, width C.int, height C.int) {
	img := &image.RGBA{
		Pix:    unsafe.Slice((*uint8)(ptr), int(width*height*4)),
		Stride: int(width) * 4,
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowPaint(img)
	}
}

//export go_on_resize
func go_on_resize(windowId C.int, width C.int, height C.int) {
	if win, ok := hwnds[int(windowId)]; ok {
		win.windowResized(int(width), int(height))
	}
}

//export go_on_key_down
func go_on_key_down(hwnd C.int, code C.int) {
	fmt.Println("Key down", strconv.FormatInt(int64(code), 16))
	key := Key(ConvertMacOSKeyToNuiKey(int(code)))
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowKeyDown(key)
	}
}

//export go_on_key_up
func go_on_key_up(hwnd C.int, code C.int) {
	key := Key(ConvertMacOSKeyToNuiKey(int(code)))
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowKeyUp(key)
	}
}

//export go_on_modifier_change
func go_on_modifier_change(hwnd C.int, shift, ctrl, alt, cmd, caps, num, fnKey C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowKeyModifiersChanged(shift != 0, ctrl != 0, alt != 0, cmd != 0, caps != 0, num != 0, fnKey != 0)
	}
}

//export go_on_char
func go_on_char(hwnd C.int, codepoint C.int) {
	//fmt.Printf("Char typed: '%c' (U+%04X)\n", rune(codepoint), codepoint)
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowChar(rune(codepoint))
	}
}

func convertMacMouseButtons(button C.int) MouseButton {
	switch button {
	case 0:
		return MouseButtonLeft
	case 1:
		return MouseButtonRight
	case 2:
		return MouseButtonMiddle
	}
	return MouseButtonLeft
}

//export go_on_window_move
func go_on_window_move(hwnd C.int, x C.int, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowMoved(int(x), int(y))
	}
}

//export go_on_declare_draw_time
func go_on_declare_draw_time(hwnd C.int, dt C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowDeclareDrawTime(int(dt))
	}
}

//export go_on_mouse_down
func go_on_mouse_down(hwnd C.int, button, x, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDown(convertMacMouseButtons(button), int(x), int(y))
		}
	}
}

//export go_on_mouse_up
func go_on_mouse_up(hwnd C.int, button, x, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonUp(convertMacMouseButtons(button), int(x), int(y))
		}
	}
}

//export go_on_mouse_move
func go_on_mouse_move(hwnd C.int, x, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowMouseMove(int(x), int(y))
		win.macSetMouseCursor(win.currentCursor)
	}
}

//export go_on_mouse_scroll
func go_on_mouse_scroll(hwnd C.int, deltaX C.float, deltaY C.float) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowMouseWheel(float64(deltaX), float64(deltaY))
	}
}

//export go_on_mouse_enter
func go_on_mouse_enter(hwnd C.int) {
	//fmt.Println("Mouse entered")
	if win, ok := hwnds[int(hwnd)]; ok {
		if win.OnMouseEnter != nil {
			win.OnMouseEnter()
		}
		win.macSetMouseCursor(win.currentCursor)
	}
}

//export go_on_mouse_leave
func go_on_mouse_leave(hwnd C.int) {
	//fmt.Println("Mouse left")
	if win, ok := hwnds[int(hwnd)]; ok {
		if win.OnMouseLeave != nil {
			win.OnMouseLeave()
		}
		win.macSetMouseCursor(MouseCursorArrow)
	}
}

//export go_on_mouse_double_click
func go_on_mouse_double_click(hwnd C.int, button, x, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDblClick(convertMacMouseButtons(button), int(x), int(y))
		}
	}
}

var dtLastTimer = time.Now()

//export go_on_timer
func go_on_timer(hwnd C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		dtNow := time.Now()
		dtDiff := dtNow.Sub(dtLastTimer)
		if dtDiff < time.Millisecond*50 {
			return
		}
		dtLastTimer = dtNow
		if win.OnTimer != nil {
			win.OnTimer()
		}
	}
}

func GetScreenSize() (width, height int) {
	width = int(C.GetScreenWidth())
	height = int(C.GetScreenHeight())
	return
}

const maxCanvasWidth = 10000
const maxCanvasHeight = 5000

var canvasBufferBackground = make([]byte, maxCanvasWidth*maxCanvasHeight*4)

func initCanvasBufferBackground(col color.Color) {
	for y := 0; y < maxCanvasHeight; y++ {
		for x := 0; x < maxCanvasWidth; x++ {
			i := (y*maxCanvasWidth + x) * 4
			r, g, b, a := col.RGBA()
			canvasBufferBackground[i+0] = byte(b)
			canvasBufferBackground[i+1] = byte(g)
			canvasBufferBackground[i+2] = byte(r)
			canvasBufferBackground[i+3] = byte(a)
		}
	}
}

var macToPCScanCode = map[int]Key{
	0x00: KeyA,
	0x01: KeyS,
	0x02: KeyD,
	0x03: KeyF,
	0x04: KeyH,
	0x05: KeyG,
	0x06: KeyZ,
	0x07: KeyX,
	0x08: KeyC,
	0x09: KeyV,
	0x0B: KeyB,
	0x0C: KeyQ,
	0x0D: KeyW,
	0x0E: KeyE,
	0x0F: KeyR,
	0x10: KeyY,
	0x11: KeyT,
	0x12: Key1,
	0x13: Key2,
	0x14: Key3,
	0x15: Key4,
	0x16: Key6,
	0x17: Key5,
	0x18: KeyEqual,
	0x19: Key9,
	0x1A: Key7,
	0x1B: KeyMinus,
	0x1C: Key8,
	0x1D: Key0,
	0x1E: KeyRightBracket,
	0x1F: KeyO,
	0x20: KeyU,
	0x21: KeyLeftBracket,
	0x22: KeyI,
	0x23: KeyP,
	0x25: KeyL,
	0x26: KeyJ,
	0x27: KeyApostrophe,
	0x28: KeyK,
	0x29: KeySemicolon,
	0x2A: KeyBackslash,
	0x2B: KeyComma,
	0x2C: KeySlash,
	0x2D: KeyN,
	0x2E: KeyM,
	0x2F: KeyDot,
	0x32: KeyGrave,
	0x41: KeyNumpadDot,
	0x43: KeyNumpadAsterisk,
	0x45: KeyNumpadPlus,
	//0x47: KeyNumpadClear,
	0x4B: KeyNumpadSlash,
	0x4C: KeyEnter,
	0x4E: KeyNumpadMinus,
	//0x51: KeyNumpadEquals,
	0x52: KeyNumpad0,
	0x53: KeyNumpad1,
	0x54: KeyNumpad2,
	0x55: KeyNumpad3,
	0x56: KeyNumpad4,
	0x57: KeyNumpad5,
	0x58: KeyNumpad6,
	0x59: KeyNumpad7,
	0x5B: KeyNumpad8,
	0x5C: KeyNumpad9,
	0x24: KeyEnter,
	0x30: KeyTab,
	0x31: KeySpace,
	0x33: KeyBackspace,
	0x35: KeyEsc,
	0x37: KeyCommand,
	0x38: KeyShift,
	0x39: KeyCapsLock,
	0x3A: KeyOption,
	0x3B: KeyCtrl,
	0x3C: KeyShift,
	0x3D: KeyOption,
	0x3E: KeyCtrl,
	0x3F: KeyFunction,
	0x40: KeyF17,
	0x4F: KeyF18,
	0x50: KeyF19,
	0x5A: KeyF20,
	0x60: KeyF5,
	0x61: KeyF6,
	0x62: KeyF7,
	0x63: KeyF3,
	0x64: KeyF8,
	0x65: KeyF9,
	0x67: KeyF11,
	0x69: KeyF13,
	0x6A: KeyF16,
	0x6B: KeyF14,
	0x6D: KeyF10,
	0x6F: KeyF12,
	0x71: KeyF15,
	0x73: KeyHome,
	0x74: KeyPageUp,
	0x75: KeyDelete,
	0x76: KeyF4,
	0x77: KeyEnd,
	0x78: KeyF2,
	0x79: KeyPageDown,
	0x7A: KeyF1,
	0x7B: KeyArrowLeft,
	0x7C: KeyArrowRight,
	0x7D: KeyArrowDown,
	0x7E: KeyArrowUp,
}

func ConvertMacOSKeyToNuiKey(macosKey int) Key {
	if key, ok := macToPCScanCode[macosKey]; ok {
		return key
	}
	return Key(0)
}

func (c *NativeWindow) startTimer(intervalMs float64) {
	C.StartTimer(C.int(c.hwnd), C.double(intervalMs))
}

func (c *NativeWindow) stopTimer() {
	C.StopTimer(C.int(c.hwnd))
}

func (c *NativeWindow) windowMouseMove(x, y int) {
	if c.OnMouseMove != nil {
		y = c.windowHeight - y
		c.OnMouseMove(x, y)
	}
	c.Update()
}

func (c *NativeWindow) windowResized(width, height int) {
	c.windowWidth = width
	c.windowHeight = height
	if c.OnResize != nil {
		c.OnResize(width, height)
	}
}

func (c *NativeWindow) windowMouseWheel(deltaX, deltaY float64) {
	deltaXInt := 0
	if deltaX > 0.2 {
		deltaXInt = 1
	}
	if deltaX < -0.2 {
		deltaXInt = -1
	}

	deltaYInt := 0
	if deltaY > 0.2 {
		deltaYInt = 1
	}
	if deltaY < -0.2 {
		deltaYInt = -1
	}

	if c.OnMouseWheel != nil {
		c.OnMouseWheel(deltaXInt, deltaYInt)
	}
}

// key modifiers
func (c *NativeWindow) windowKeyModifiersChanged(shift bool, ctrl bool, alt bool, cmd bool, caps bool, num bool, _ bool) {
	// Key shift
	if c.keyModifiers.Shift && !shift {
		c.windowKeyUp(KeyShift)
	}
	if !c.keyModifiers.Shift && shift {
		c.windowKeyDown(KeyShift)
	}
	c.keyModifiers.Shift = shift

	// Key ctrl
	if c.keyModifiers.Ctrl && !ctrl {
		c.windowKeyUp(KeyCtrl)
	}
	if !c.keyModifiers.Ctrl && ctrl {
		c.windowKeyDown(KeyCtrl)
	}
	c.keyModifiers.Ctrl = ctrl

	// Key alt
	if c.keyModifiers.Alt && !alt {
		c.windowKeyUp(KeyOption)
	}
	if !c.keyModifiers.Alt && alt {
		c.windowKeyDown(KeyOption)
	}
	c.keyModifiers.Alt = alt

	// Key cmd
	if c.keyModifiers.Cmd && !cmd {
		c.windowKeyUp(KeyCommand)
	}
	if !c.keyModifiers.Cmd && cmd {
		c.windowKeyDown(KeyCommand)
	}
	c.keyModifiers.Cmd = cmd

	if caps != c.lastCapsLockState {
		if caps {
			c.windowKeyDown(KeyCapsLock)
		} else {
			c.windowKeyDown(KeyCapsLock)
		}
		c.lastCapsLockState = caps
	}

	if num != c.lastNumLockState {
		if num {
			c.windowKeyDown(KeyNumLock)
		} else {
			c.windowKeyDown(KeyNumLock)
		}
		c.lastNumLockState = num
	}
}

func (c *NativeWindow) windowKeyDown(keyCode Key) {
	if c.OnKeyDown != nil {
		c.OnKeyDown(keyCode, c.keyModifiers)
	}
}

func (c *NativeWindow) windowKeyUp(keyCode Key) {
	if c.OnKeyUp != nil {
		c.OnKeyUp(keyCode, c.keyModifiers)
	}
}

func (c *NativeWindow) windowDeclareDrawTime(dt int) {
	c.drawTimes[c.drawTimesIndex] = int64(dt)
	c.drawTimesIndex++
	if c.drawTimesIndex >= len(c.drawTimes) {
		c.drawTimesIndex = 0
	}
}

func (c *NativeWindow) windowPaint(rgba *image.RGBA) {

	imgDataSize := rgba.Rect.Dx() * rgba.Rect.Dy() * 4
	copy(rgba.Pix[:imgDataSize], canvasBufferBackground)

	if c.OnPaint != nil {
		c.OnPaint(rgba)
	}
}

func (c *NativeWindow) windowChar(char rune) {
	if !unicode.IsPrint(char) {
		return
	}

	if c.OnChar != nil {
		c.OnChar(char)
	}
}

func (c *NativeWindow) windowMouseButtonDown(button MouseButton, x, y int) {
	if c.OnMouseButtonDown != nil {
		c.OnMouseButtonDown(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *NativeWindow) windowMouseButtonUp(button MouseButton, x, y int) {
	if c.OnMouseButtonUp != nil {
		c.OnMouseButtonUp(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *NativeWindow) windowMouseButtonDblClick(button MouseButton, x, y int) {
	if c.OnMouseButtonDblClick != nil {
		c.OnMouseButtonDblClick(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *NativeWindow) windowMoved(x, y int) {
	c.windowPosX = x
	c.windowPosY = y
	if c.OnMove != nil {
		c.OnMove(x, y)
	}
}

func (c *NativeWindow) requestWindowPosition() (int, int) {
	x := int(C.GetWindowPositionX(C.int(c.hwnd)))
	y := int(C.GetWindowPositionY(C.int(c.hwnd)))
	return x, y
}

func (c *NativeWindow) requestWindowSize() (int, int) {
	w := int(C.GetWindowWidth(C.int(c.hwnd)))
	h := int(C.GetWindowHeight(C.int(c.hwnd)))
	return w, h
}
