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

	"github.com/ipoluianov/nui/nuikey"
	"github.com/ipoluianov/nui/nuimouse"
)

//export go_on_paint
func go_on_paint(hwnd C.int, ptr unsafe.Pointer, width C.int, height C.int) {
	img := &image.RGBA{
		Pix:    unsafe.Slice((*uint8)(ptr), int(width*height*4)),
		Stride: int(width) * 4,
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowPaint(img)
	}
}

//export go_on_resize
func go_on_resize(hwnd C.int, width C.int, height C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowResized(int(width), int(height))
	}
}

//export go_on_key_down
func go_on_key_down(hwnd C.int, code C.int) {
	fmt.Println("Key down", strconv.FormatInt(int64(code), 16))
	key := nuikey.Key(ConvertMacOSKeyToNuiKey(int(code)))
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowKeyDown(key)
	}
}

//export go_on_key_up
func go_on_key_up(hwnd C.int, code C.int) {
	key := nuikey.Key(ConvertMacOSKeyToNuiKey(int(code)))
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowKeyUp(key)
	}
}

//export go_on_modifier_change
func go_on_modifier_change(hwnd C.int, shift, ctrl, alt, cmd, caps, num, fnKey C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowKeyModifiersChanged(shift != 0, ctrl != 0, alt != 0, cmd != 0, caps != 0, num != 0, fnKey != 0)
	}
}

//export go_on_char
func go_on_char(hwnd C.int, codepoint C.int) {
	//fmt.Printf("Char typed: '%c' (U+%04X)\n", rune(codepoint), codepoint)
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowChar(rune(codepoint))
	}
}

func convertMacMouseButtons(button C.int) nuimouse.MouseButton {
	switch button {
	case 0:
		return nuimouse.MouseButtonLeft
	case 1:
		return nuimouse.MouseButtonRight
	case 2:
		return nuimouse.MouseButtonMiddle
	}
	return nuimouse.MouseButtonLeft
}

//export go_on_window_move
func go_on_window_move(hwnd C.int, x C.int, y C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowMoved(int(x), int(y))
	}

}

//export go_on_declare_draw_time
func go_on_declare_draw_time(hwnd C.int, dt C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowDeclareDrawTime(int(dt))
	}
}

//export go_on_mouse_down
func go_on_mouse_down(hwnd C.int, button, x, y C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDown(convertMacMouseButtons(button), int(x), int(y))
		}
	}
}

//export go_on_mouse_up
func go_on_mouse_up(hwnd C.int, button, x, y C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonUp(convertMacMouseButtons(button), int(x), int(y))
		}
	}
}

//export go_on_mouse_move
func go_on_mouse_move(hwnd C.int, x, y C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowMouseMove(int(x), int(y))
		win.macSetMouseCursor(win.currentCursor)
	}
}

//export go_on_mouse_scroll
func go_on_mouse_scroll(hwnd C.int, deltaX C.float, deltaY C.float) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowMouseWheel(float64(deltaX), float64(deltaY))
	}
}

//export go_on_mouse_enter
func go_on_mouse_enter(hwnd C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowMouseEnter()
	}
}

//export go_on_mouse_leave
func go_on_mouse_leave(hwnd C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		win.windowMouseLeave()
	}
}

//export go_on_mouse_double_click
func go_on_mouse_double_click(hwnd C.int, button, x, y C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDblClick(convertMacMouseButtons(button), int(x), int(y))
		}
	}
}

var dtLastTimer = time.Now()

//export go_on_timer
func go_on_timer(hwnd C.int) {
	if win, ok := hwnds[windowId(hwnd)]; ok {
		dtNow := time.Now()
		dtDiff := dtNow.Sub(dtLastTimer)
		if dtDiff < time.Millisecond*50 {
			return
		}
		dtLastTimer = dtNow
		if win.onTimer != nil {
			win.onTimer()
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

var macToPCScanCode = map[int]nuikey.Key{
	0x00: nuikey.KeyA,
	0x01: nuikey.KeyS,
	0x02: nuikey.KeyD,
	0x03: nuikey.KeyF,
	0x04: nuikey.KeyH,
	0x05: nuikey.KeyG,
	0x06: nuikey.KeyZ,
	0x07: nuikey.KeyX,
	0x08: nuikey.KeyC,
	0x09: nuikey.KeyV,
	0x0B: nuikey.KeyB,
	0x0C: nuikey.KeyQ,
	0x0D: nuikey.KeyW,
	0x0E: nuikey.KeyE,
	0x0F: nuikey.KeyR,
	0x10: nuikey.KeyY,
	0x11: nuikey.KeyT,
	0x12: nuikey.Key1,
	0x13: nuikey.Key2,
	0x14: nuikey.Key3,
	0x15: nuikey.Key4,
	0x16: nuikey.Key6,
	0x17: nuikey.Key5,
	0x18: nuikey.KeyEqual,
	0x19: nuikey.Key9,
	0x1A: nuikey.Key7,
	0x1B: nuikey.KeyMinus,
	0x1C: nuikey.Key8,
	0x1D: nuikey.Key0,
	0x1E: nuikey.KeyRightBracket,
	0x1F: nuikey.KeyO,
	0x20: nuikey.KeyU,
	0x21: nuikey.KeyLeftBracket,
	0x22: nuikey.KeyI,
	0x23: nuikey.KeyP,
	0x25: nuikey.KeyL,
	0x26: nuikey.KeyJ,
	0x27: nuikey.KeyApostrophe,
	0x28: nuikey.KeyK,
	0x29: nuikey.KeySemicolon,
	0x2A: nuikey.KeyBackslash,
	0x2B: nuikey.KeyComma,
	0x2C: nuikey.KeySlash,
	0x2D: nuikey.KeyN,
	0x2E: nuikey.KeyM,
	0x2F: nuikey.KeyDot,
	0x32: nuikey.KeyGrave,
	0x41: nuikey.KeyNumpadDot,
	0x43: nuikey.KeyNumpadAsterisk,
	0x45: nuikey.KeyNumpadPlus,
	//0x47: KeyNumpadClear,
	0x4B: nuikey.KeyNumpadSlash,
	0x4C: nuikey.KeyEnter,
	0x4E: nuikey.KeyNumpadMinus,
	//0x51: KeyNumpadEquals,
	0x52: nuikey.KeyNumpad0,
	0x53: nuikey.KeyNumpad1,
	0x54: nuikey.KeyNumpad2,
	0x55: nuikey.KeyNumpad3,
	0x56: nuikey.KeyNumpad4,
	0x57: nuikey.KeyNumpad5,
	0x58: nuikey.KeyNumpad6,
	0x59: nuikey.KeyNumpad7,
	0x5B: nuikey.KeyNumpad8,
	0x5C: nuikey.KeyNumpad9,
	0x24: nuikey.KeyEnter,
	0x30: nuikey.KeyTab,
	0x31: nuikey.KeySpace,
	0x33: nuikey.KeyBackspace,
	0x35: nuikey.KeyEsc,
	0x37: nuikey.KeyCommand,
	0x38: nuikey.KeyShift,
	0x39: nuikey.KeyCapsLock,
	0x3B: nuikey.KeyCtrl,
	0x3C: nuikey.KeyShift,
	0x3E: nuikey.KeyCtrl,
	0x3F: nuikey.KeyFunction,
	0x40: nuikey.KeyF17,
	0x4F: nuikey.KeyF18,
	0x50: nuikey.KeyF19,
	0x5A: nuikey.KeyF20,
	0x60: nuikey.KeyF5,
	0x61: nuikey.KeyF6,
	0x62: nuikey.KeyF7,
	0x63: nuikey.KeyF3,
	0x64: nuikey.KeyF8,
	0x65: nuikey.KeyF9,
	0x67: nuikey.KeyF11,
	0x69: nuikey.KeyF13,
	0x6A: nuikey.KeyF16,
	0x6B: nuikey.KeyF14,
	0x6D: nuikey.KeyF10,
	0x6F: nuikey.KeyF12,
	0x71: nuikey.KeyF15,
	0x73: nuikey.KeyHome,
	0x74: nuikey.KeyPageUp,
	0x75: nuikey.KeyDelete,
	0x76: nuikey.KeyF4,
	0x77: nuikey.KeyEnd,
	0x78: nuikey.KeyF2,
	0x79: nuikey.KeyPageDown,
	0x7A: nuikey.KeyF1,
	0x7B: nuikey.KeyArrowLeft,
	0x7C: nuikey.KeyArrowRight,
	0x7D: nuikey.KeyArrowDown,
	0x7E: nuikey.KeyArrowUp,
}

func ConvertMacOSKeyToNuiKey(macosKey int) nuikey.Key {
	if key, ok := macToPCScanCode[macosKey]; ok {
		return key
	}
	return nuikey.Key(0)
}

func (c *nativeWindow) startTimer(intervalMs float64) {
	C.StartTimer(C.int(c.hwnd), C.double(intervalMs))
}

func (c *nativeWindow) stopTimer() {
	C.StopTimer(C.int(c.hwnd))
}

func (c *nativeWindow) windowMouseMove(x, y int) {
	if c.onMouseMove != nil {
		y = c.windowHeight - y
		c.onMouseMove(x, y)
	}
	c.Update()
}

func (c *nativeWindow) windowResized(width, height int) {
	c.windowWidth = width
	c.windowHeight = height
	if c.onResize != nil {
		c.onResize(width, height)
	}
}

func (c *nativeWindow) windowMouseWheel(deltaX, deltaY float64) {
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

	if c.onMouseWheel != nil {
		c.onMouseWheel(deltaXInt, deltaYInt)
	}
}

func (c *nativeWindow) windowMouseEnter() {
	if c.onMouseEnter != nil {
		c.onMouseEnter()
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMouseLeave() {
	fmt.Println("Mouse leave")
	if c.onMouseLeave != nil {
		c.onMouseLeave()
	}
	c.macSetMouseCursor(nuimouse.MouseCursorArrow)
}

// key modifiers
func (c *nativeWindow) windowKeyModifiersChanged(shift bool, ctrl bool, alt bool, cmd bool, caps bool, num bool, _ bool) {
	// Key shift
	if c.keyModifiers.Shift && !shift {
		c.windowKeyUp(nuikey.KeyShift)
	}
	if !c.keyModifiers.Shift && shift {
		c.windowKeyDown(nuikey.KeyShift)
	}
	c.keyModifiers.Shift = shift

	// Key ctrl
	if c.keyModifiers.Ctrl && !ctrl {
		c.windowKeyUp(nuikey.KeyCtrl)
	}
	if !c.keyModifiers.Ctrl && ctrl {
		c.windowKeyDown(nuikey.KeyCtrl)
	}
	c.keyModifiers.Ctrl = ctrl

	// Key alt
	if c.keyModifiers.Alt && !alt {
		c.windowKeyUp(nuikey.KeyAlt)
	}
	if !c.keyModifiers.Alt && alt {
		c.windowKeyDown(nuikey.KeyAlt)
	}
	c.keyModifiers.Alt = alt

	// Key cmd
	if c.keyModifiers.Cmd && !cmd {
		c.windowKeyUp(nuikey.KeyCommand)
	}
	if !c.keyModifiers.Cmd && cmd {
		c.windowKeyDown(nuikey.KeyCommand)
	}
	c.keyModifiers.Cmd = cmd

	if caps != c.platform.lastCapsLockState {
		if caps {
			c.windowKeyDown(nuikey.KeyCapsLock)
		} else {
			c.windowKeyDown(nuikey.KeyCapsLock)
		}
		c.platform.lastCapsLockState = caps
	}

	if num != c.platform.lastNumLockState {
		if num {
			c.windowKeyDown(nuikey.KeyNumLock)
		} else {
			c.windowKeyDown(nuikey.KeyNumLock)
		}
		c.platform.lastNumLockState = num
	}
}

func (c *nativeWindow) windowKeyDown(keyCode nuikey.Key) {
	if c.onKeyDown != nil {
		c.onKeyDown(keyCode, c.keyModifiers)
	}
}

func (c *nativeWindow) windowKeyUp(keyCode nuikey.Key) {
	if c.onKeyUp != nil {
		keyModifiers := c.keyModifiers
		if keyCode == nuikey.KeyShift {
			keyModifiers.Shift = false
		}
		if keyCode == nuikey.KeyCtrl {
			keyModifiers.Ctrl = false
		}
		if keyCode == nuikey.KeyAlt {
			keyModifiers.Alt = false
		}
		if keyCode == nuikey.KeyCommand {
			keyModifiers.Cmd = false
		}
		c.onKeyUp(keyCode, keyModifiers)
	}
}

func (c *nativeWindow) windowDeclareDrawTime(dt int) {
	c.drawTimes[c.drawTimesIndex] = int64(dt)
	c.drawTimesIndex++
	if c.drawTimesIndex >= len(c.drawTimes) {
		c.drawTimesIndex = 0
	}
}

func (c *nativeWindow) windowPaint(rgba *image.RGBA) {

	imgDataSize := rgba.Rect.Dx() * rgba.Rect.Dy() * 4
	copy(rgba.Pix[:imgDataSize], canvasBufferBackground)

	if c.onPaint != nil {
		c.onPaint(rgba)
	}
}

func (c *nativeWindow) windowChar(char rune) {
	if !unicode.IsPrint(char) {
		return
	}

	if c.onChar != nil {
		c.onChar(char)
	}
}

func (c *nativeWindow) windowMouseButtonDown(button nuimouse.MouseButton, x, y int) {
	if c.onMouseButtonDown != nil {
		y = c.windowHeight - y
		c.onMouseButtonDown(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMouseButtonUp(button nuimouse.MouseButton, x, y int) {
	if c.onMouseButtonUp != nil {
		y = c.windowHeight - y
		c.onMouseButtonUp(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMouseButtonDblClick(button nuimouse.MouseButton, x, y int) {
	if c.onMouseButtonDblClick != nil {
		y = c.windowHeight - y
		c.onMouseButtonDblClick(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMoved(x, y int) {
	c.windowPosX = x
	c.windowPosY = y
	if c.onMove != nil {
		c.onMove(x, y)
	}
}

func (c *nativeWindow) requestWindowPosition() (int, int) {
	x := int(C.GetWindowPositionX(C.int(c.hwnd)))
	y := int(C.GetWindowPositionY(C.int(c.hwnd)))
	return x, y
}

func (c *nativeWindow) requestWindowSize() (int, int) {
	w := int(C.GetWindowWidth(C.int(c.hwnd)))
	h := int(C.GetWindowHeight(C.int(c.hwnd)))
	return w, h
}
