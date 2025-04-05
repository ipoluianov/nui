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

func CreateWindow() *NativeWindow {
	var c NativeWindow

	initCanvasBufferBackground(color.RGBA{0, 50, 0, 255})

	c.hwnd = int(C.InitWindow())

	x, y := c.requestWindowPosition()
	c.windowPosX = int(x)
	c.windowPosY = int(y)

	hwnds[c.hwnd] = &c
	c.startTimer(1)
	return &c
}

func (c *NativeWindow) Show() {
	C.ShowWindow(C.int(c.hwnd))
}

func (c *NativeWindow) EventLoop() {
	C.RunEventLoop()
}

func (c *NativeWindow) Close() {
	C.CloseWindowById(C.int(c.hwnd))
}

func (c *NativeWindow) SetTitle(title string) {
	C.SetWindowTitle(C.int(c.hwnd), C.CString(title))
}

func (c *NativeWindow) SetMouseCursor(cursor MouseCursor) {
}

func (c *NativeWindow) MaximizeWindow() {
	C.MaximizeWindow(C.int(c.hwnd))
}

func (c *NativeWindow) MinimizeWindow() {
	C.MinimizeWindow(C.int(c.hwnd))
}

func (c *NativeWindow) Move(x, y int) {
	C.SetWindowPosition(C.int(c.hwnd), C.int(x), C.int(y))
}

func (c *NativeWindow) Resize(width, height int) {
	C.SetWindowSize(C.int(c.hwnd), C.int(width), C.int(height))
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

func (c *NativeWindow) Update() {
	C.UpdateWindow(C.int(c.hwnd))
}

func (c *NativeWindow) startTimer(intervalMs float64) {
	C.StartTimer(C.int(c.hwnd), C.double(intervalMs))
}

func (c *NativeWindow) stopTimer() {
	C.StopTimer(C.int(c.hwnd))
}

func (c *NativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
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
func (c *NativeWindow) windowKeyModifiersChanged(shift bool, ctrl bool, alt bool, cmd bool) {
	c.keyModifiers.Shift = shift
	c.keyModifiers.Ctrl = ctrl
	c.keyModifiers.Alt = alt
	c.keyModifiers.Cmd = cmd
}

func (c *NativeWindow) KeyModifiers() KeyModifiers {
	return c.keyModifiers
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
	if c.OnChar != nil {
		c.OnChar(char)
	}
}

func (c *NativeWindow) windowMouseButtonDown(button MouseButton, x, y int) {
	if c.OnMouseButtonDown != nil {
		c.OnMouseButtonDown(button, x, y)
	}
}

func (c *NativeWindow) windowMouseButtonUp(button MouseButton, x, y int) {
	if c.OnMouseButtonUp != nil {
		c.OnMouseButtonUp(button, x, y)
	}
}

func (c *NativeWindow) windowMouseButtonDblClick(button MouseButton, x, y int) {
	if c.OnMouseButtonDblClick != nil {
		c.OnMouseButtonDblClick(button, x, y)
	}
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
