package nui

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"time"
	"unsafe"
)

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/Xatom.h>
#include <stdlib.h>
#include <string.h>
#include "ximage_helper.h"
*/
import "C"

func init() {
}

type NativeWindow struct {
	display *C.Display
	window  C.Window
	screen  C.int

	currentCursor MouseCursor
	lastSetCursor MouseCursor

	keyModifiers KeyModifiers

	dtLastUpdateCalled time.Time
	needUpdateInTimer  bool

	windowPosX   int
	windowPosY   int
	windowWidth  int
	windowHeight int

	drawTimes      [32]int64
	drawTimesIndex int

	// Keyboard events
	OnKeyDown func(keyCode Key, mods KeyModifiers)
	OnKeyUp   func(keyCode Key, mods KeyModifiers)
	OnChar    func(char rune)

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

var mouseInside bool = false

type rect struct {
	left, top, right, bottom int32
}

func loadPngFromBytes(bs []byte) (*image.RGBA, error) {
	img, err := png.Decode(bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}

	rgba := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	return rgba, nil
}

var hwnds map[C.Window]*NativeWindow

func init() {
	hwnds = make(map[C.Window]*NativeWindow)
}

func GetNativeWindowByHandle(hwnd C.Window) *NativeWindow {
	if w, ok := hwnds[hwnd]; ok {
		return w
	}
	return nil
}

func getHDCSize(hdc uintptr) (width int32, height int32) {
	var r rect
	return r.right - r.left, r.bottom - r.top
}

const chunkHeight = 100
const maxWidth = 10000

var pixBuffer = make([]byte, 4*chunkHeight*maxWidth)

const maxCanvasWidth = 10000
const maxCanvasHeight = 5000

var canvasBuffer = make([]byte, maxCanvasWidth*maxCanvasHeight*4)
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

///////////////////////////////////////////////////////////////////

func CreateWindow() *NativeWindow {
	var c NativeWindow
	initCanvasBufferBackground(color.RGBA{0, 50, 0, 255})

	c.display = C.XOpenDisplay(nil)
	if c.display == nil {
		panic("Unable to open X display")
	}
	//defer C.XCloseDisplay(c.display)

	c.screen = C.XDefaultScreen(c.display)

	attrs := C.XSetWindowAttributes{}
	attrs.background_pixmap = C.None

	mask := C.CWBackPixmap

	c.window = C.XCreateWindow(
		c.display,
		C.XRootWindow(c.display, c.screen),
		100, 100, // x, y
		800, 600, // width, height
		1,                // border width
		C.CopyFromParent, // depth
		C.InputOutput,    // class
		nil,              // visual
		C.ulong(mask),    // valuemask
		&attrs,           // attributes pointer (не значение!)
	)

	C.XSelectInput(c.display, c.window, C.ExposureMask|C.PropertyChangeMask|C.StructureNotifyMask|C.KeyPressMask|C.KeyReleaseMask|C.EnterWindowMask|C.LeaveWindowMask|C.ButtonPressMask|C.ButtonReleaseMask|C.PointerMotionMask)

	C.XMapWindow(c.display, c.window)

	var getAttr C.XWindowAttributes
	C.XGetWindowAttributes(c.display, c.window, &getAttr)
	c.windowWidth, c.windowHeight = int(getAttr.width), int(getAttr.height)

	// Store the window handle
	hwnds[c.window] = &c

	// Set default icon
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	c.SetAppIcon(icon)

	c.SetTitle(DefaultWindowTitle)

	return &c
}

func (c *NativeWindow) Show() {
}

func (c *NativeWindow) Hide() {
}

func (c *NativeWindow) Update() {
	if time.Since(c.dtLastUpdateCalled) < 40*time.Millisecond {
		c.needUpdateInTimer = true
		return
	}
	c.dtLastUpdateCalled = time.Now()

	C.XClearArea(
		c.display,
		c.window,
		0, 0,
		0, 0,
		1, // last parameter is `exposures`: if True — generate Expose event
	)
	C.XFlush(c.display)
	//C.XClearWindow(c.display, c.window)
}

func eventType(event C.XEvent) int {
	return int(*(*C.int)(unsafe.Pointer(&event)))
}

/*var posX C.uint
var posY C.uint
var width C.uint
var height C.uint*/

func (c *NativeWindow) EventLoop() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	dtLastPaint := time.Now()

	for {
		for C.XPending(c.display) > 0 {
			var event C.XEvent
			C.XNextEvent(c.display, &event)

			switch eventType(event) {

			case C.Expose:
				{
					{
						dtBeginPaint := time.Now()
						dtLastPaint = time.Now()
						hdcWidth, hdcHeight := c.windowWidth, c.windowHeight
						if hdcWidth > maxCanvasWidth {
							hdcWidth = maxCanvasWidth
						}

						if hdcHeight > maxCanvasHeight {
							hdcHeight = maxCanvasHeight
						}

						img := &image.RGBA{
							Pix:    canvasBuffer,
							Stride: int(hdcWidth) * 4,
							Rect:   image.Rect(0, 0, int(hdcWidth), int(hdcHeight)),
						}

						// Clear the canvas
						canvasDataBufferSize := int(hdcWidth * hdcHeight * 4)
						copy(canvasBuffer[:canvasDataBufferSize], canvasBufferBackground)

						if c.OnPaint != nil {
							c.OnPaint(img)
						}

						c.drawImageRGBA(c.display, c.window, img)
						paintTime := time.Since(dtLastPaint)
						_ = paintTime
						fmt.Println("PaintTime:", paintTime.Microseconds())

						c.drawTimes[c.drawTimesIndex] = time.Since(dtBeginPaint).Microseconds()
						c.drawTimesIndex++
						if c.drawTimesIndex >= len(c.drawTimes) {
							c.drawTimesIndex = 0
						}

					}

				}
			case C.MapNotify:
				mapEvent := (*C.XMapEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window became visible. Window ID: %d\n", mapEvent.window)

			case C.UnmapNotify:
				unmapEvent := (*C.XUnmapEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window was hidden. Window ID: %d\n", unmapEvent.window)

			case C.DestroyNotify:
				destroyEvent := (*C.XDestroyWindowEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window was destroyed. Window ID: %d\n", destroyEvent.window)

			case C.ReparentNotify:
				reparentEvent := (*C.XReparentEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window changed parent. Window ID: %d, New Parent ID: %d\n", reparentEvent.window, reparentEvent.parent)
			case C.ResizeRequest:
				resizeEvent := (*C.XResizeRequestEvent)(unsafe.Pointer(&event))
				fmt.Printf("Resize request received: Width=%d, Height=%d\n", resizeEvent.width, resizeEvent.height)

				c.windowWidth = int(resizeEvent.width)
				c.windowHeight = int(resizeEvent.height)

				//c.Update()

			case C.ConfigureNotify:
				configureEvent := (*C.XConfigureEvent)(unsafe.Pointer(&event))

				if configureEvent.send_event == 1 && (c.windowPosX != int(configureEvent.x) || c.windowPosY != int(configureEvent.y)) {
					c.windowPosX = int(configureEvent.x)
					c.windowPosY = int(configureEvent.y)
					if c.OnMove != nil {
						c.OnMove(c.windowPosX, c.windowPosY)
					}
				}

				if configureEvent.send_event == 0 && (c.windowWidth != int(configureEvent.width) || c.windowHeight != int(configureEvent.height)) {
					c.windowWidth = int(configureEvent.width)
					c.windowHeight = int(configureEvent.height)
					if c.OnResize != nil {
						c.OnResize(c.windowWidth, c.windowHeight)
					}
				}

				c.Update()

			case C.KeyPress:
				keyEvent := (*C.XKeyEvent)(unsafe.Pointer(&event))
				keySym := C.XLookupKeysym((*C.XKeyEvent)(unsafe.Pointer(&event)), 0)
				fmt.Printf("Key pressed: KeySym = %d, KeyCode = 0x%x\n", keySym, keyEvent.keycode)
				key := ConvertLinuxKeyToNuiKey(int(keyEvent.keycode))
				if c.OnKeyDown != nil {
					c.OnKeyDown(key, c.keyModifiers)
				}
				if key == KeyShift {
					c.keyModifiers.Shift = true
				}
				if key == KeyCtrl {
					c.keyModifiers.Ctrl = true
				}
				if key == KeyAlt {
					c.keyModifiers.Alt = true
				}

			case C.KeyRelease:
				keyEvent := (*C.XKeyEvent)(unsafe.Pointer(&event))
				keySym := C.XLookupKeysym(keyEvent, 0)
				fmt.Printf("Key released: KeySym = %d, KeyCode = 0x%x\n", keySym, keyEvent.keycode)
				key := ConvertLinuxKeyToNuiKey(int(keyEvent.keycode))
				if key == KeyShift {
					c.keyModifiers.Shift = false
				}
				if key == KeyCtrl {
					c.keyModifiers.Ctrl = false
				}
				if key == KeyAlt {
					c.keyModifiers.Alt = false
				}
				if c.OnKeyUp != nil {
					c.OnKeyUp(key, c.keyModifiers)
				}

			case C.EnterNotify:
				if c.OnMouseEnter != nil {
					c.OnMouseEnter()
				}

			case C.LeaveNotify:
				if c.OnMouseLeave != nil {
					c.OnMouseLeave()
				}

			case C.MotionNotify:
				motionEvent := (*C.XMotionEvent)(unsafe.Pointer(&event))
				if c.OnMouseMove != nil {
					c.OnMouseMove(int(motionEvent.x), int(motionEvent.y))
				}

			case C.ButtonPress:
				buttonEvent := (*C.XButtonEvent)(unsafe.Pointer(&event))
				fmt.Printf("Mouse button %d pressed at (%d, %d)\n", buttonEvent.button, buttonEvent.x, buttonEvent.y)

				x := int(buttonEvent.x)
				y := int(buttonEvent.y)

				switch buttonEvent.button {
				case 1:
					if c.OnMouseButtonDown != nil {
						c.OnMouseButtonDown(MouseButtonLeft, x, y)
					}
				case 2:
					if c.OnMouseButtonDown != nil {
						c.OnMouseButtonDown(MouseButtonMiddle, x, y)
					}
				case 3:
					if c.OnMouseButtonDown != nil {
						c.OnMouseButtonDown(MouseButtonRight, x, y)
					}
				case 4:
					if c.OnMouseWheel != nil {
						c.OnMouseWheel(1, 0)
					}
				case 5:
					if c.OnMouseWheel != nil {
						c.OnMouseWheel(-1, 0)
					}
				case 6:
					if c.OnMouseWheel != nil {
						c.OnMouseWheel(0, 1)
					}
				case 7:
					if c.OnMouseWheel != nil {
						c.OnMouseWheel(0, -1)
					}
				}

			case C.ButtonRelease:
				buttonEvent := (*C.XButtonEvent)(unsafe.Pointer(&event))
				fmt.Printf("Mouse button %d released at (%d, %d)\n", buttonEvent.button, buttonEvent.x, buttonEvent.y)

				x := int(buttonEvent.x)
				y := int(buttonEvent.y)

				switch buttonEvent.button {
				case 1:
					if c.OnMouseButtonUp != nil {
						c.OnMouseButtonUp(MouseButtonLeft, x, y)
					}
				case 2:
					if c.OnMouseButtonUp != nil {
						c.OnMouseButtonUp(MouseButtonMiddle, x, y)
					}
				case 3:
					if c.OnMouseButtonUp != nil {
						c.OnMouseButtonUp(MouseButtonRight, x, y)
					}
				}

			}
		}

		select {
		case <-ticker.C:
			{
				//fmt.Println("Timer event: 10ms tick")
				if c.needUpdateInTimer {
					c.Update()
					c.needUpdateInTimer = false
				}
				if c.OnTimer != nil {
					c.OnTimer()
					c.Update()
				}
			}
		default:
		}
	}
}

func (c *NativeWindow) Close() {
	C.XDestroyWindow(c.display, c.window)
	C.XCloseDisplay(c.display)
}

func (c *NativeWindow) SetTitle(title string) {
	cstr := C.CString(title)
	defer C.free(unsafe.Pointer(cstr))
	C.XStoreName(c.display, c.window, cstr)
}

func (c *NativeWindow) Move(x, y int) {
	C.XMoveWindow(c.display, c.window, C.int(x), C.int(y))
}

func getScreenSize() (width, height int) {
	display := C.XOpenDisplay(nil)
	screen := C.XDefaultScreen(display)
	width = int(C.XDisplayWidth(display, screen))
	height = int(C.XDisplayHeight(display, screen))
	C.XCloseDisplay(display)
	return
}

func (c *NativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := getScreenSize()
	windowWidth, windowHeight := c.Size()
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	c.Move(int(x), int(y))
}

func (c *NativeWindow) Resize(width, height int) {
	C.XResizeWindow(c.display, c.window, C.uint(width), C.uint(height))
}

func (c *NativeWindow) PosX() int {
	return c.windowPosX
}

func (c *NativeWindow) PosY() int {
	return c.windowPosY
}

func (c *NativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
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

func (c *NativeWindow) SetMouseCursor(cursor MouseCursor) {
	if c.currentCursor == cursor {
		return
	}
	c.currentCursor = cursor
	c.changeMouseCursor(cursor)
}

func (c *NativeWindow) changeMouseCursor(mouseCursor MouseCursor) bool {
	var cursorShape uint

	const (
		CursorArrow = 132
		CursorCross = 34
		CursorWait  = 150
		CursorIBeam = 152
		CursorHand  = 58
		CursorBlank = 0

		CursorResizeVertical   = 116 // XC_sb_v_double_arrow
		CursorResizeHorizontal = 108 // XC_sb_h_double_arrow
	)

	switch mouseCursor {
	case MouseCursorNotDefined:
	case MouseCursorArrow:
		cursorShape = CursorArrow
	case MouseCursorPointer:
		cursorShape = CursorHand
	case MouseCursorResizeHor:
		cursorShape = CursorResizeHorizontal
	case MouseCursorResizeVer:
		cursorShape = CursorResizeVertical
	case MouseCursorIBeam:
		cursorShape = CursorIBeam
	}

	cursor := C.XCreateFontCursor(c.display, C.uint(cursorShape))
	C.XDefineCursor(c.display, c.window, cursor)
	C.XFlush(c.display)
	return true
}

func (c *NativeWindow) MinimizeWindow() {
	C.minimizeWindow(c.display, c.window)
}

func (c *NativeWindow) MaximizeWindow() {
	C.maximizeWindow(c.display, c.window)
}

func (c *NativeWindow) SetAppIcon(icon *image.RGBA) {
	width := icon.Bounds().Dx()
	height := icon.Bounds().Dy()

	// _NET_WM_ICON: [width, height, pixels...]
	dataLen := 2 + width*height
	data := make([]C.ulong, dataLen)
	data[0] = C.ulong(width)
	data[1] = C.ulong(height)

	// Конвертировать RGBA в ARGB
	i := 2
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := icon.PixOffset(x, y)
			r := icon.Pix[offset]
			g := icon.Pix[offset+1]
			b := icon.Pix[offset+2]
			a := icon.Pix[offset+3]

			argb := (uint32(a) << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
			data[i] = C.ulong(argb)
			i++
		}
	}

	atom := C.XInternAtom(c.display, C.CString("_NET_WM_ICON"), C.False)
	typ := C.Atom(C.XA_CARDINAL)
	format := 32

	C.XChangeProperty(
		c.display,
		c.window,
		atom,
		typ,
		C.int(format),
		C.PropModeReplace,
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.int(len(data)),
	)
}

func (c *NativeWindow) drawImageRGBA(display *C.Display, window C.Window, img image.Image) {
	width := c.windowWidth
	height := c.windowHeight

	dataSize := width * height * 4

	// RGBA->BGRA
	pixelsCount := width * height
	for i := 0; i < pixelsCount; i++ {
		canvasBuffer[i*4], canvasBuffer[i*4+2] = canvasBuffer[i*4+2], canvasBuffer[i*4]
	}

	cBuffer := C.malloc(C.size_t(dataSize))
	C.memcpy(cBuffer, unsafe.Pointer(&canvasBuffer[0]), C.size_t(dataSize))

	ximage := C.XCreateImage(
		display,
		C.XDefaultVisual(display, C.XDefaultScreen(display)),
		24,
		C.ZPixmap,
		0,
		(*C.char)(cBuffer),
		C.uint(width),
		C.uint(height),
		32,
		0,
	)

	//C.DestroyXImage(ximage) // TODO:

	gc := C.XCreateGC(display, C.Drawable(window), 0, nil)
	defer C.XFreeGC(display, gc) // TODO:

	C.XPutImage(display, C.Drawable(window), gc, ximage, 0, 0, 0, 0, C.uint(width), C.uint(height))

	C.destroy_ximage(ximage)
}

/*func drawBlue(display *C.Display, window C.Window, screen C.int) {
	gc := C.XCreateGC(display, C.Drawable(window), 0, nil)
	defer C.XFreeGC(display, gc)
	colorName := C.CString("blue")
	defer C.free(unsafe.Pointer(colorName))

	var exactColor, screenColor C.XColor
	C.XAllocNamedColor(display, C.XDefaultColormap(display, screen), colorName, &screenColor, &exactColor)

	C.XSetForeground(display, gc, screenColor.pixel)

	C.XFillRectangle(display, C.Drawable(window), gc, 0, 0, width/2, height/2)
}
*/
