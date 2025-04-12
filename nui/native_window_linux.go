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

	"github.com/ipoluianov/nui/nuikey"
	"github.com/ipoluianov/nui/nuimouse"
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

type windowId C.Window

type nativeWindowPlatform struct {
	display *C.Display
	window  C.Window
	screen  C.int

	dtLastUpdateCalled time.Time
	needUpdateInTimer  bool
}

/*type NativeWindow struct {
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
}*/

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

var hwnds map[windowId]*nativeWindow

func init() {
	hwnds = make(map[windowId]*nativeWindow)
}

func GetNativeWindowByHandle(hwnd C.Window) *nativeWindow {
	if w, ok := hwnds[windowId(hwnd)]; ok {
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

func createWindow(title string, width int, height int, center bool) *nativeWindow {
	var c nativeWindow
	initCanvasBufferBackground(color.RGBA{0, 50, 0, 255})

	c.platform.display = C.XOpenDisplay(nil)
	if c.platform.display == nil {
		panic("Unable to open X display")
	}
	//defer C.XCloseDisplay(c.display)

	c.platform.screen = C.XDefaultScreen(c.platform.display)

	attrs := C.XSetWindowAttributes{}
	attrs.background_pixmap = C.None

	mask := C.CWBackPixmap

	c.platform.window = C.XCreateWindow(
		c.platform.display,
		C.XRootWindow(c.platform.display, c.platform.screen),
		100, 100, // x, y
		C.uint(width), C.uint(height), // width, height
		1,                // border width
		C.CopyFromParent, // depth
		C.InputOutput,    // class
		nil,              // visual
		C.ulong(mask),    // valuemask
		&attrs,           // attributes pointer (не значение!)
	)

	C.XSelectInput(c.platform.display, c.platform.window, C.ExposureMask|C.PropertyChangeMask|C.StructureNotifyMask|C.KeyPressMask|C.KeyReleaseMask|C.EnterWindowMask|C.LeaveWindowMask|C.ButtonPressMask|C.ButtonReleaseMask|C.PointerMotionMask)

	C.XMapWindow(c.platform.display, c.platform.window)

	var getAttr C.XWindowAttributes
	C.XGetWindowAttributes(c.platform.display, c.platform.window, &getAttr)
	c.windowWidth, c.windowHeight = int(getAttr.width), int(getAttr.height)

	// Store the window handle
	hwnds[windowId(c.platform.window)] = &c

	// Set default icon
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	c.SetAppIcon(icon)

	c.SetTitle(title)

	return &c
}

func (c *nativeWindow) Show() {
}

func (c *nativeWindow) Hide() {
}

func (c *nativeWindow) Update() {
	if time.Since(c.platform.dtLastUpdateCalled) < 40*time.Millisecond {
		c.platform.needUpdateInTimer = true
		return
	}
	c.platform.dtLastUpdateCalled = time.Now()

	C.XClearArea(
		c.platform.display,
		c.platform.window,
		0, 0,
		0, 0,
		1, // last parameter is `exposures`: if True — generate Expose event
	)
	C.XFlush(c.platform.display)
	//C.XClearWindow(c.display, c.window)
}

func eventType(event C.XEvent) int {
	return int(*(*C.int)(unsafe.Pointer(&event)))
}

/*var posX C.uint
var posY C.uint
var width C.uint
var height C.uint*/

func (c *nativeWindow) EventLoop() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	dtLastPaint := time.Now()

	for {
		for C.XPending(c.platform.display) > 0 {
			var event C.XEvent
			C.XNextEvent(c.platform.display, &event)

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

						if c.onPaint != nil {
							c.onPaint(img)
						}

						c.drawImageRGBA(c.platform.display, c.platform.window, img)
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
					if c.onMove != nil {
						c.onMove(c.windowPosX, c.windowPosY)
					}
				}

				if configureEvent.send_event == 0 && (c.windowWidth != int(configureEvent.width) || c.windowHeight != int(configureEvent.height)) {
					c.windowWidth = int(configureEvent.width)
					c.windowHeight = int(configureEvent.height)
					if c.onResize != nil {
						c.onResize(c.windowWidth, c.windowHeight)
					}
				}

				c.Update()

			case C.KeyPress:
				keyEvent := (*C.XKeyEvent)(unsafe.Pointer(&event))
				keySym := C.XLookupKeysym((*C.XKeyEvent)(unsafe.Pointer(&event)), 0)
				fmt.Printf("Key pressed: KeySym = %d, KeyCode = 0x%x\n", keySym, keyEvent.keycode)
				key := ConvertLinuxKeyToNuiKey(int(keyEvent.keycode))
				if c.onKeyDown != nil {
					c.onKeyDown(key, c.keyModifiers)
				}
				if key == nuikey.KeyShift {
					c.keyModifiers.Shift = true
				}
				if key == nuikey.KeyCtrl {
					c.keyModifiers.Ctrl = true
				}
				if key == nuikey.KeyAlt {
					c.keyModifiers.Alt = true
				}

			case C.KeyRelease:
				keyEvent := (*C.XKeyEvent)(unsafe.Pointer(&event))
				keySym := C.XLookupKeysym(keyEvent, 0)
				fmt.Printf("Key released: KeySym = %d, KeyCode = 0x%x\n", keySym, keyEvent.keycode)
				key := ConvertLinuxKeyToNuiKey(int(keyEvent.keycode))
				if key == nuikey.KeyShift {
					c.keyModifiers.Shift = false
				}
				if key == nuikey.KeyCtrl {
					c.keyModifiers.Ctrl = false
				}
				if key == nuikey.KeyAlt {
					c.keyModifiers.Alt = false
				}
				if c.onKeyUp != nil {
					c.onKeyUp(key, c.keyModifiers)
				}

			case C.EnterNotify:
				if c.onMouseEnter != nil {
					c.onMouseEnter()
				}

			case C.LeaveNotify:
				if c.onMouseLeave != nil {
					c.onMouseLeave()
				}

			case C.MotionNotify:
				motionEvent := (*C.XMotionEvent)(unsafe.Pointer(&event))
				if c.onMouseMove != nil {
					c.onMouseMove(int(motionEvent.x), int(motionEvent.y))
				}

			case C.ButtonPress:
				buttonEvent := (*C.XButtonEvent)(unsafe.Pointer(&event))
				fmt.Printf("Mouse button %d pressed at (%d, %d)\n", buttonEvent.button, buttonEvent.x, buttonEvent.y)

				x := int(buttonEvent.x)
				y := int(buttonEvent.y)

				switch buttonEvent.button {
				case 1:
					if c.onMouseButtonDown != nil {
						c.onMouseButtonDown(nuimouse.MouseButtonLeft, x, y)
					}
				case 2:
					if c.onMouseButtonDown != nil {
						c.onMouseButtonDown(nuimouse.MouseButtonMiddle, x, y)
					}
				case 3:
					if c.onMouseButtonDown != nil {
						c.onMouseButtonDown(nuimouse.MouseButtonRight, x, y)
					}
				case 4:
					if c.onMouseWheel != nil {
						c.onMouseWheel(1, 0)
					}
				case 5:
					if c.onMouseWheel != nil {
						c.onMouseWheel(-1, 0)
					}
				case 6:
					if c.onMouseWheel != nil {
						c.onMouseWheel(0, 1)
					}
				case 7:
					if c.onMouseWheel != nil {
						c.onMouseWheel(0, -1)
					}
				}

			case C.ButtonRelease:
				buttonEvent := (*C.XButtonEvent)(unsafe.Pointer(&event))
				fmt.Printf("Mouse button %d released at (%d, %d)\n", buttonEvent.button, buttonEvent.x, buttonEvent.y)

				x := int(buttonEvent.x)
				y := int(buttonEvent.y)

				switch buttonEvent.button {
				case 1:
					if c.onMouseButtonUp != nil {
						c.onMouseButtonUp(nuimouse.MouseButtonLeft, x, y)
					}
				case 2:
					if c.onMouseButtonUp != nil {
						c.onMouseButtonUp(nuimouse.MouseButtonMiddle, x, y)
					}
				case 3:
					if c.onMouseButtonUp != nil {
						c.onMouseButtonUp(nuimouse.MouseButtonRight, x, y)
					}
				}

			}
		}

		select {
		case <-ticker.C:
			{
				//fmt.Println("Timer event: 10ms tick")
				if c.platform.needUpdateInTimer {
					c.Update()
					c.platform.needUpdateInTimer = false
				}
				if c.onTimer != nil {
					c.onTimer()
					c.Update()
				}
			}
		default:
		}
	}
}

func (c *nativeWindow) Close() {
	C.XDestroyWindow(c.platform.display, c.platform.window)
	C.XCloseDisplay(c.platform.display)
}

func (c *nativeWindow) SetTitle(title string) {
	cstr := C.CString(title)
	defer C.free(unsafe.Pointer(cstr))
	C.XStoreName(c.platform.display, c.platform.window, cstr)
}

func (c *nativeWindow) Move(x, y int) {
	C.XMoveWindow(c.platform.display, c.platform.window, C.int(x), C.int(y))
}

func getScreenSize() (width, height int) {
	display := C.XOpenDisplay(nil)
	screen := C.XDefaultScreen(display)
	width = int(C.XDisplayWidth(display, screen))
	height = int(C.XDisplayHeight(display, screen))
	C.XCloseDisplay(display)
	return
}

func (c *nativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := getScreenSize()
	windowWidth, windowHeight := c.Size()
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	c.Move(int(x), int(y))
}

func (c *nativeWindow) Resize(width, height int) {
	C.XResizeWindow(c.platform.display, c.platform.window, C.uint(width), C.uint(height))
}

func (c *nativeWindow) PosX() int {
	return c.windowPosX
}

func (c *nativeWindow) PosY() int {
	return c.windowPosY
}

func (c *nativeWindow) Pos() (x, y int) {
	return c.windowPosX, c.windowPosY
}

func (c *nativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
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

func (c *nativeWindow) SetBackgroundColor(color color.RGBA) {
	initCanvasBufferBackground(color)
	c.Update()
}

func (c *nativeWindow) SetMouseCursor(cursor nuimouse.MouseCursor) {
	if c.currentCursor == cursor {
		return
	}
	c.currentCursor = cursor
	c.changeMouseCursor(cursor)
}

func (c *nativeWindow) changeMouseCursor(mouseCursor nuimouse.MouseCursor) bool {
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
	case nuimouse.MouseCursorNotDefined:
	case nuimouse.MouseCursorArrow:
		cursorShape = CursorArrow
	case nuimouse.MouseCursorPointer:
		cursorShape = CursorHand
	case nuimouse.MouseCursorResizeHor:
		cursorShape = CursorResizeHorizontal
	case nuimouse.MouseCursorResizeVer:
		cursorShape = CursorResizeVertical
	case nuimouse.MouseCursorIBeam:
		cursorShape = CursorIBeam
	}

	cursor := C.XCreateFontCursor(c.platform.display, C.uint(cursorShape))
	C.XDefineCursor(c.platform.display, c.platform.window, cursor)
	C.XFlush(c.platform.display)
	return true
}

func (c *nativeWindow) MinimizeWindow() {
	C.minimizeWindow(c.platform.display, c.platform.window)
}

func (c *nativeWindow) MaximizeWindow() {
	C.maximizeWindow(c.platform.display, c.platform.window)
}

func (c *nativeWindow) SetAppIcon(icon *image.RGBA) {
	width := icon.Bounds().Dx()
	height := icon.Bounds().Dy()

	// _NET_WM_ICON: [width, height, pixels...]
	dataLen := 2 + width*height
	data := make([]C.ulong, dataLen)
	data[0] = C.ulong(width)
	data[1] = C.ulong(height)

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

	atom := C.XInternAtom(c.platform.display, C.CString("_NET_WM_ICON"), C.False)
	typ := C.Atom(C.XA_CARDINAL)
	format := 32

	C.XChangeProperty(
		c.platform.display,
		c.platform.window,
		atom,
		typ,
		C.int(format),
		C.PropModeReplace,
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.int(len(data)),
	)
}

func (c *nativeWindow) drawImageRGBA(display *C.Display, window C.Window, img image.Image) {
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
