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

	windowPosX   int
	windowPosY   int
	windowWidth  int
	windowHeight int

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
	OnMouseWheel                   func(deltaX float64, deltaY float64)
	OnMouseDoubleClickLeftButton   func(x, y int)
	OnMouseDoubleClickRightButton  func(x, y int)
	OnMouseDoubleClickMiddleButton func(x, y int)

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

func GetRGBATestImage() *image.RGBA {
	rgba, err := loadPngFromBytes(TestPng)
	if err != nil {
		panic(err)
	}
	return rgba
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
	/*cwa := C.XSetWindowAttributes{}
	cwa.background_pixmap = C.None*/

	/*c.window = C.XCreateSimpleWindow(
		c.display,
		C.XRootWindow(c.display, c.screen),
		100, 100,
		800, 600,
		1,
		C.XBlackPixel(c.display, c.screen),
		C.XWhitePixel(c.display, c.screen),
	)*/

	C.XSelectInput(c.display, c.window, C.ExposureMask|C.PropertyChangeMask|C.StructureNotifyMask|C.KeyPressMask|C.KeyReleaseMask|C.EnterWindowMask|C.LeaveWindowMask|C.ButtonPressMask|C.ButtonReleaseMask|C.PointerMotionMask)

	C.XMapWindow(c.display, c.window)

	// Store the window handle
	hwnds[c.window] = &c

	// Set default icon
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	c.SetAppIcon(icon)

	return &c
}

func (c *NativeWindow) Show() {
}

func (c *NativeWindow) Hide() {
}

func (c *NativeWindow) Update() {
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

//go:embed test.png
var pngContent []byte

func loadImageFromEmbed() (image.Image, error) {
	img, err := png.Decode(bytes.NewReader(pngContent))
	return img, err
}

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
						//fmt.Println("PaintTime:", paintTime.Microseconds())
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

				needEventMove := false
				if c.windowPosX != int(configureEvent.x) || c.windowPosY != int(configureEvent.y) {
					needEventMove = true
				}

				needEventResize := false
				if c.windowWidth != int(configureEvent.width) || c.windowHeight != int(configureEvent.height) {
					needEventResize = true
				}

				c.windowPosX = int(configureEvent.x)
				c.windowPosY = int(configureEvent.y)
				c.windowWidth = int(configureEvent.width)
				c.windowHeight = int(configureEvent.height)

				if needEventMove {
					// TODO: event window move
				}

				if needEventResize {
					if c.OnResize != nil {
						c.OnResize(c.windowWidth, c.windowHeight)
					}
				}

				c.Update()

			case C.KeyPress:
				keyEvent := (*C.XKeyEvent)(unsafe.Pointer(&event))
				keySym := C.XLookupKeysym((*C.XKeyEvent)(unsafe.Pointer(&event)), 0)
				fmt.Printf("Key pressed: KeySym = %d, KeyCode = %d\n", keySym, keyEvent.keycode)
				if c.OnKeyDown != nil {
					c.OnKeyDown(Key(0))
				}

				//resizeWindow(display, window, 600, 200)
				//moveWindow(display, window, 100, 100)
				//setWindowTitle(display, window, "HELLO")
			case C.KeyRelease:
				keyEvent := (*C.XKeyEvent)(unsafe.Pointer(&event))
				keySym := C.XLookupKeysym(keyEvent, 0)
				fmt.Printf("Key released: KeySym = %d, KeyCode = %d\n", keySym, keyEvent.keycode)

			case C.EnterNotify:
				//enterEvent := (*C.XCrossingEvent)(unsafe.Pointer(&event))
				//fmt.Printf("Cursor entered window at (%d, %d)\n", enterEvent.x, enterEvent.y)
				if c.OnMouseEnter != nil {
					c.OnMouseEnter()
				}

			case C.LeaveNotify:
				//leaveEvent := (*C.XCrossingEvent)(unsafe.Pointer(&event))
				//fmt.Printf("Cursor left window at (%d, %d)\n", leaveEvent.x, leaveEvent.y)
				if c.OnMouseLeave != nil {
					c.OnMouseLeave()
				}

			case C.MotionNotify:
				motionEvent := (*C.XMotionEvent)(unsafe.Pointer(&event))
				if c.OnMouseMove != nil {
					c.OnMouseMove(int(motionEvent.x), int(motionEvent.y))
				}

			case C.ButtonPress:
				/*buttonEvent := (*C.XButtonEvent)(unsafe.Pointer(&event))
				fmt.Printf("Mouse button %d pressed at (%d, %d)\n", buttonEvent.button, buttonEvent.x, buttonEvent.y)*/

				buttonEvent := (*C.XButtonEvent)(unsafe.Pointer(&event))
				fmt.Printf("Mouse button %d pressed at (%d, %d)\n", buttonEvent.button, buttonEvent.x, buttonEvent.y)

				x := int(buttonEvent.x)
				y := int(buttonEvent.y)

				switch buttonEvent.button {
				case 1:
					if c.OnMouseDownLeftButton != nil {
						c.OnMouseDownLeftButton(x, y)
					}
				case 2:
					if c.OnMouseDownMiddleButton != nil {
						c.OnMouseDownMiddleButton(x, y)
					}
				case 3:
					if c.OnMouseDownRightButton != nil {
						c.OnMouseDownRightButton(x, y)
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
					if c.OnMouseUpLeftButton != nil {
						c.OnMouseUpLeftButton(x, y)
					}
				case 2:
					if c.OnMouseUpMiddleButton != nil {
						c.OnMouseUpMiddleButton(x, y)
					}
				case 3:
					if c.OnMouseUpRightButton != nil {
						c.OnMouseUpRightButton(x, y)
					}
				}

			}
		}

		select {
		case <-ticker.C:
			{
				//fmt.Println("Timer event: 10ms tick")
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

func GetScreenSize() (width, height int) {
	display := C.XOpenDisplay(nil)
	screen := C.XDefaultScreen(display)
	width = int(C.XDisplayWidth(display, screen))
	height = int(C.XDisplayHeight(display, screen))
	C.XCloseDisplay(display)
	return
}

func (c *NativeWindow) MoveToCenterOfScreen() {
}

func (c *NativeWindow) Resize(width, height int) {
	C.XResizeWindow(c.display, c.window, C.uint(width), C.uint(height))
}

func (c *NativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
}

func (c *NativeWindow) SetMouseCursor(cursor MouseCursor) {
	if c.currentCursor == cursor {
		return
	}
	c.currentCursor = cursor
	c.changeMouseCursor(cursor)
}

func (c *NativeWindow) changeMouseCursor(cursor MouseCursor) bool {
	return false
}

func (c *NativeWindow) MinimizeWindow() {
	/*wmState := C.XInternAtom(c.display, C.CString("_NET_WM_STATE"), C.False)
	  wmHidden := C.XInternAtom(c.display, C.CString("_NET_WM_STATE_HIDDEN"), C.False)

	  var xev C.XEvent
	  xev._type = C.ClientMessage
	  xev.xclient.type = C.ClientMessage
	  xev.xclient.window = c.window
	  xev.xclient.message_type = wmState
	  xev.xclient.format = 32
	  xev.xclient.data.set(0, 1)
	  xev.xclient.data.set(1, wmHidden)

	  root := C.XDefaultRootWindow(c.display)
	  C.XSendEvent(c.display, root, C.False,
	      C.SubstructureNotifyMask|C.SubstructureRedirectMask,
	      &xev)	*/
}

func (c *NativeWindow) MaximizeWindow() {
}

func (c *NativeWindow) RestoreWindow() {
}

func (c *NativeWindow) SetAppIcon(icon *image.RGBA) {
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
