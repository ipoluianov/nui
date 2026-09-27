package platforms

// Darwin: callbacks from cocoa_darwin.go's ObjC classes plus Mac-specific
// input / size helpers. No cgo - callers are Go closures in the same binary.

import (
	"image"
	"image/color"
	"time"
	"unicode"
	"unsafe"
)

func go_on_paint(hwnd windowId, ptr unsafe.Pointer, width int, height int) {
	// width/height come from drawRect's drawable rect in cocoa_darwin.go.
	img := &image.RGBA{
		Pix:    unsafe.Slice((*uint8)(ptr), width*height*4),
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}

	if win, ok := hwnds[hwnd]; ok {
		win.windowPaint(img)
	}
}

func go_on_resize(hwnd windowId, width int, height int) {
	// width/height: NSWindow.contentLayoutRect (client area)
	if win, ok := hwnds[hwnd]; ok {
		win.windowResized(width, height)
	}
}

func go_on_close_request(hwnd windowId) bool {
	win, ok := hwnds[hwnd]
	if !ok || win.onCloseRequest == nil {
		return true
	}
	return win.onCloseRequest()
}

func go_on_window_will_close(hwnd windowId) {
	closePopupsOf(hwnd)
	delete(hwnds, hwnd)
	if mainWindowIDSet && hwnd == mainWindowID {
		quitApp()
	}
}

func go_on_window_deactivate(hwnd windowId) {
	if win, ok := hwnds[hwnd]; ok && win.onDeactivate != nil {
		win.onDeactivate()
	}
}

func go_on_key_down(hwnd windowId, code int) {
	key := Key(ConvertMacOSKeyToNuiKey(code))
	if win, ok := hwnds[hwnd]; ok {
		win.windowKeyDown(key)
	}
}

func go_on_key_up(hwnd windowId, code int) {
	key := Key(ConvertMacOSKeyToNuiKey(code))
	if win, ok := hwnds[hwnd]; ok {
		win.windowKeyUp(key)
	}
}

func go_on_modifier_change(hwnd windowId, shift, ctrl, alt, cmd, caps, num, fnKey bool) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowKeyModifiersChanged(shift, ctrl, alt, cmd, caps, num, fnKey)
	}
}

func go_on_char(hwnd windowId, codepoint int) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowChar(rune(codepoint))
	}
}

func convertMacMouseButtons(button int) MouseButton {
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

func go_on_window_move(hwnd windowId, x int, y int) {
	// x,Y: frame left and top-down Y from cocoa_darwin.go (matches Move()).
	if win, ok := hwnds[hwnd]; ok {
		win.windowMoved(x, y)
	}
}

func go_on_declare_draw_time(hwnd windowId, dt int) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowDeclareDrawTime(dt)
	}
}

func go_on_mouse_down(hwnd windowId, button, x, y int) {
	if win, ok := hwnds[hwnd]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDown(convertMacMouseButtons(button), x, y)
		}
	}
}

func go_on_mouse_up(hwnd windowId, button, x, y int) {
	if win, ok := hwnds[hwnd]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonUp(convertMacMouseButtons(button), x, y)
		}
	}
}

func go_on_mouse_move(hwnd windowId, x, y int) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowMouseMove(x, y)
		win.macSetMouseCursor(win.currentCursor)
	}
}

func go_on_mouse_scroll(hwnd windowId, deltaX float64, deltaY float64) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowMouseWheel(deltaX, deltaY)
	}
}

func go_on_mouse_enter(hwnd windowId) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowMouseEnter()
	}
}

func go_on_mouse_leave(hwnd windowId) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowMouseLeave()
	}
}

func go_on_mouse_double_click(hwnd windowId, button, x, y int) {
	if win, ok := hwnds[hwnd]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDblClick(convertMacMouseButtons(button), x, y)
		}
	}
}

func go_on_timer(hwnd windowId) {
	if win, ok := hwnds[hwnd]; ok {
		dtNow := time.Now()
		if dtNow.Sub(win.platform.lastTimerTick) < time.Millisecond*50 {
			return
		}
		win.platform.lastTimerTick = dtNow
		if win.onTimer != nil {
			win.onTimer()
		}
	}
}

// Main display frame in points (see cocoa_darwin.go getScreenWidth/Height).
func GetScreenSize() (width, height int) {
	return getScreenWidth(), getScreenHeight()
}

const maxCanvasWidth = 10000
const maxCanvasHeight = 5000

var canvasBufferBackground = make([]byte, maxCanvasWidth*maxCanvasHeight*4)
var canvasBufferBackgroundColor color.Color

// Filling the 200MB background buffer is expensive; every createWindow() call used to redo it
// with the same default color and stall the main thread. Skip it when the color didn't change.
func initCanvasBufferBackground(col color.Color) {
	if canvasBufferBackgroundColor == col {
		return
	}
	canvasBufferBackgroundColor = col

	r, g, b, a := col.RGBA()
	rb, gb, bb, ab := byte(b), byte(g), byte(r), byte(a)
	for i := 0; i < len(canvasBufferBackground); i += 4 {
		canvasBufferBackground[i+0] = rb
		canvasBufferBackground[i+1] = gb
		canvasBufferBackground[i+2] = bb
		canvasBufferBackground[i+3] = ab
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
	0x3B: KeyCtrl,
	0x3C: KeyShift,
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

func (c *nativeWindow) startTimer(intervalMs float64) {
	startTimer(c.hwnd, intervalMs)
}

func (c *nativeWindow) stopTimer() {
	stopTimer(c.hwnd)
}

func (c *nativeWindow) windowMouseMove(x, y int) {
	if c.onMouseMove != nil {
		// NSView coords: origin bottom-left; nui uses top-left Y.
		_, areaH := c.requestClientAreaSize()
		y = areaH - y
		c.onMouseMove(x, y)
	}
	c.Update()
}

func (c *nativeWindow) windowResized(width, height int) {
	// Client-area width/height from window.m (contentLayoutRect).
	c.windowWidth = width
	c.windowHeight = height
	if c.onResize != nil {
		c.onResize(width, height)
	}
}

func (c *nativeWindow) windowMouseWheel(deltaX, deltaY float64) {
	if c.onMouseWheel != nil {
		c.onMouseWheel(wheelStep(deltaX), wheelStep(deltaY))
	}
}

// wheelStep turns a scroll delta into one wheel step: -1, 0 or 1.
func wheelStep(delta float64) int {
	if delta > 0.2 {
		return 1
	}
	if delta < -0.2 {
		return -1
	}
	return 0
}

func (c *nativeWindow) windowMouseEnter() {
	if c.onMouseEnter != nil {
		c.onMouseEnter()
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMouseLeave() {
	if c.onMouseLeave != nil {
		c.onMouseLeave()
	}
	c.macSetMouseCursor(MouseCursorArrow)
}

// key modifiers
func (c *nativeWindow) windowKeyModifiersChanged(shift bool, ctrl bool, alt bool, cmd bool, caps bool, num bool, _ bool) {
	// Key shift
	if c.platform.keyModifiers.Shift && !shift {
		c.windowKeyUp(KeyShift)
	}
	if !c.platform.keyModifiers.Shift && shift {
		c.windowKeyDown(KeyShift)
	}
	c.platform.keyModifiers.Shift = shift

	// Key ctrl
	if c.platform.keyModifiers.Ctrl && !ctrl {
		c.windowKeyUp(KeyCtrl)
	}
	if !c.platform.keyModifiers.Ctrl && ctrl {
		c.windowKeyDown(KeyCtrl)
	}
	c.platform.keyModifiers.Ctrl = ctrl

	// Key alt
	if c.platform.keyModifiers.Alt && !alt {
		c.windowKeyUp(KeyAlt)
	}
	if !c.platform.keyModifiers.Alt && alt {
		c.windowKeyDown(KeyAlt)
	}
	c.platform.keyModifiers.Alt = alt

	// Key cmd
	if c.platform.keyModifiers.Cmd && !cmd {
		c.windowKeyUp(KeyCommand)
	}
	if !c.platform.keyModifiers.Cmd && cmd {
		c.windowKeyDown(KeyCommand)
	}
	c.platform.keyModifiers.Cmd = cmd

	if caps != c.platform.lastCapsLockState {
		if caps {
			c.windowKeyDown(KeyCapsLock)
		} else {
			c.windowKeyDown(KeyCapsLock)
		}
		c.platform.lastCapsLockState = caps
	}

	if num != c.platform.lastNumLockState {
		if num {
			c.windowKeyDown(KeyNumLock)
		} else {
			c.windowKeyDown(KeyNumLock)
		}
		c.platform.lastNumLockState = num
	}
}

func (c *nativeWindow) windowKeyDown(keyCode Key) {
	if c.onKeyDown != nil {
		c.onKeyDown(keyCode, c.platform.keyModifiers)
	}
}

func (c *nativeWindow) windowKeyUp(keyCode Key) {
	if c.onKeyUp != nil {
		keyModifiers := c.platform.keyModifiers
		if keyCode == KeyShift {
			keyModifiers.Shift = false
		}
		if keyCode == KeyCtrl {
			keyModifiers.Ctrl = false
		}
		if keyCode == KeyAlt {
			keyModifiers.Alt = false
		}
		if keyCode == KeyCommand {
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

func (c *nativeWindow) windowMouseButtonDown(button MouseButton, x, y int) {
	if c.onMouseButtonDown != nil {
		// Flip Y: see windowMouseMove.
		_, areaH := c.requestClientAreaSize()
		y = areaH - y
		c.onMouseButtonDown(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMouseButtonUp(button MouseButton, x, y int) {
	if c.onMouseButtonUp != nil {
		// Flip Y: see windowMouseMove.
		_, areaH := c.requestClientAreaSize()
		y = areaH - y
		c.onMouseButtonUp(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMouseButtonDblClick(button MouseButton, x, y int) {
	if c.onMouseButtonDblClick != nil {
		// Flip Y: see windowMouseMove.
		_, areaH := c.requestClientAreaSize()
		y = areaH - y
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

// Frame position: X is Cocoa left edge; Y is top-down distance to window top (cocoa_darwin.go helpers).
func (c *nativeWindow) requestWindowPosition() (int, int) {
	return getWindowPositionX(c.hwnd), getWindowPositionY(c.hwnd)
}

// Client-area size via contentLayoutRect (same getters as getWindowWidth/Height in cocoa_darwin.go).
func (c *nativeWindow) requestWindowSize() (int, int) {
	return getWindowWidth(c.hwnd), getWindowHeight(c.hwnd)
}

// On Darwin, identical dimensions to requestWindowSize (thin bridge to ObjC symmetry).
func (c *nativeWindow) requestClientAreaSize() (int, int) {
	return getClientAreaWidth(c.hwnd), getClientAreaHeight(c.hwnd)
}
