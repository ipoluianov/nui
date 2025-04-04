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

/*
 Mac OS key codes
 enum {
	kVK_ANSI_A                    = 0x00,
	kVK_ANSI_S                    = 0x01,
	kVK_ANSI_D                    = 0x02,
	kVK_ANSI_F                    = 0x03,
	kVK_ANSI_H                    = 0x04,
	kVK_ANSI_G                    = 0x05,
	kVK_ANSI_Z                    = 0x06,
	kVK_ANSI_X                    = 0x07,
	kVK_ANSI_C                    = 0x08,
	kVK_ANSI_V                    = 0x09,
	kVK_ANSI_B                    = 0x0B,
	kVK_ANSI_Q                    = 0x0C,
	kVK_ANSI_W                    = 0x0D,
	kVK_ANSI_E                    = 0x0E,
	kVK_ANSI_R                    = 0x0F,
	kVK_ANSI_Y                    = 0x10,
	kVK_ANSI_T                    = 0x11,
	kVK_ANSI_1                    = 0x12,
	kVK_ANSI_2                    = 0x13,
	kVK_ANSI_3                    = 0x14,
	kVK_ANSI_4                    = 0x15,
	kVK_ANSI_6                    = 0x16,
	kVK_ANSI_5                    = 0x17,
	kVK_ANSI_Equal                = 0x18,
	kVK_ANSI_9                    = 0x19,
	kVK_ANSI_7                    = 0x1A,
	kVK_ANSI_Minus                = 0x1B,
	kVK_ANSI_8                    = 0x1C,
	kVK_ANSI_0                    = 0x1D,
	kVK_ANSI_RightBracket         = 0x1E,
	kVK_ANSI_O                    = 0x1F,
	kVK_ANSI_U                    = 0x20,
	kVK_ANSI_LeftBracket          = 0x21,
	kVK_ANSI_I                    = 0x22,
	kVK_ANSI_P                    = 0x23,
	kVK_ANSI_L                    = 0x25,
	kVK_ANSI_J                    = 0x26,
	kVK_ANSI_Quote                = 0x27,
	kVK_ANSI_K                    = 0x28,
	kVK_ANSI_Semicolon            = 0x29,
	kVK_ANSI_Backslash            = 0x2A,
	kVK_ANSI_Comma                = 0x2B,
	kVK_ANSI_Slash                = 0x2C,
	kVK_ANSI_N                    = 0x2D,
	kVK_ANSI_M                    = 0x2E,
	kVK_ANSI_Period               = 0x2F,
	kVK_ANSI_Grave                = 0x32,
	kVK_ANSI_KeypadDecimal        = 0x41,
	kVK_ANSI_KeypadMultiply       = 0x43,
	kVK_ANSI_KeypadPlus           = 0x45,
	kVK_ANSI_KeypadClear          = 0x47,
	kVK_ANSI_KeypadDivide         = 0x4B,
	kVK_ANSI_KeypadEnter          = 0x4C,
	kVK_ANSI_KeypadMinus          = 0x4E,
	kVK_ANSI_KeypadEquals         = 0x51,
	kVK_ANSI_Keypad0              = 0x52,
	kVK_ANSI_Keypad1              = 0x53,
	kVK_ANSI_Keypad2              = 0x54,
	kVK_ANSI_Keypad3              = 0x55,
	kVK_ANSI_Keypad4              = 0x56,
	kVK_ANSI_Keypad5              = 0x57,
	kVK_ANSI_Keypad6              = 0x58,
	kVK_ANSI_Keypad7              = 0x59,
	kVK_ANSI_Keypad8              = 0x5B,
	kVK_ANSI_Keypad9              = 0x5C
  };

  enum {
	kVK_Return                    = 0x24,
	kVK_Tab                       = 0x30,
	kVK_Space                     = 0x31,
	kVK_Delete                    = 0x33,
	kVK_Escape                    = 0x35,
	kVK_Command                   = 0x37,
	kVK_Shift                     = 0x38,
	kVK_CapsLock                  = 0x39,
	kVK_Option                    = 0x3A,
	kVK_Control                   = 0x3B,
	kVK_RightShift                = 0x3C,
	kVK_RightOption               = 0x3D,
	kVK_RightControl              = 0x3E,
	kVK_Function                  = 0x3F,
	kVK_F17                       = 0x40,
	kVK_VolumeUp                  = 0x48,
	kVK_VolumeDown                = 0x49,
	kVK_Mute                      = 0x4A,
	kVK_F18                       = 0x4F,
	kVK_F19                       = 0x50,
	kVK_F20                       = 0x5A,
	kVK_F5                        = 0x60,
	kVK_F6                        = 0x61,
	kVK_F7                        = 0x62,
	kVK_F3                        = 0x63,
	kVK_F8                        = 0x64,
	kVK_F9                        = 0x65,
	kVK_F11                       = 0x67,
	kVK_F13                       = 0x69,
	kVK_F16                       = 0x6A,
	kVK_F14                       = 0x6B,
	kVK_F10                       = 0x6D,
	kVK_F12                       = 0x6F,
	kVK_F15                       = 0x71,
	kVK_Help                      = 0x72,
	kVK_Home                      = 0x73,
	kVK_PageUp                    = 0x74,
	kVK_ForwardDelete             = 0x75,
	kVK_F4                        = 0x76,
	kVK_End                       = 0x77,
	kVK_F2                        = 0x78,
	kVK_PageDown                  = 0x79,
	kVK_F1                        = 0x7A,
	kVK_LeftArrow                 = 0x7B,
	kVK_RightArrow                = 0x7C,
	kVK_DownArrow                 = 0x7D,
	kVK_UpArrow                   = 0x7E
  };
*/

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
