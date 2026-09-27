//go:build linux
// +build linux

package platforms

// Pure-Go bindings for the small slice of Xlib/libc used by native_window_linux.go,
// loaded at runtime via purego (dlopen/dlsym) instead of cgo. This lets the
// package cross-compile with CGO_ENABLED=0 (no C compiler or X11 headers
// needed at build time); libX11.so still has to be present on the machine
// that actually runs the resulting binary, same as before.
//
// Struct layouts below mirror /usr/include/X11/Xlib.h and X.h field-for-field
// (including field order, so Go's default alignment matches the C ABI on
// amd64/arm64). Only functions and struct fields native_window_linux.go
// actually touches are declared.
//
// Parameter types follow purego's documented conventions: opaque X11
// handles (Display*, Window, Atom, GC, Cursor, Visual*, XImage*, KeySym,
// Colormap, Pixmap, Drawable - all "unsigned long" or a pointer in C, both
// word-sized on every Linux ABI) are plain uintptr; C strings we always
// supply are Go `string` params (purego copies them for the call, see its
// func.go doc); pointers to Go-allocated structs/buffers we read or write
// are `unsafe.Pointer` so purego's own argument boxing keeps them alive for
// the call, per https://pkg.go.dev/github.com/ebitengine/purego#RegisterFunc.

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	xNone           = 0
	xCopyFromParent = 0
	xInputOutput    = 1
	xCWBackPixmap   = 1 << 0
	xCWBackPixel    = 1 << 1
	xCWBitGravity   = 1 << 4

	xNorthWestGravity   = 1
	xCWOverrideRedirect = 1 << 9
	xCWSaveUnder        = 1 << 10

	xKeyPressMask        = 1 << 0
	xKeyReleaseMask      = 1 << 1
	xButtonPressMask     = 1 << 2
	xButtonReleaseMask   = 1 << 3
	xEnterWindowMask     = 1 << 4
	xLeaveWindowMask     = 1 << 5
	xPointerMotionMask   = 1 << 6
	xExposureMask        = 1 << 15
	xFocusChangeMask     = 1 << 21
	xStructureNotifyMask = 1 << 17
	xPropertyChangeMask  = 1 << 22

	xSubstructureNotifyMask   = 1 << 19
	xSubstructureRedirectMask = 1 << 20

	xKeyPress        = 2
	xKeyRelease      = 3
	xButtonPress     = 4
	xButtonRelease   = 5
	xMotionNotify    = 6
	xEnterNotify     = 7
	xLeaveNotify     = 8
	xFocusOut        = 10
	xExpose          = 12
	xDestroyNotify   = 17
	xUnmapNotify     = 18
	xMapNotify       = 19
	xReparentNotify  = 21
	xConfigureNotify = 22
	xResizeRequest   = 25
	xPropertyNotify  = 28
	xClientMessage   = 33

	xSuccess = 0
	xZPixmap = 2

	xShiftMask   = 1 << 0
	xControlMask = 1 << 2
	xMod1Mask    = 1 << 3

	xPropModeReplace = 0

	xXAAtom     = 4
	xXACardinal = 6

	xFalse = 0
	xTrue  = 1

	xLCAll = 6 // glibc LC_ALL
)

// XSetWindowAttributes, field order per Xlib.h.
type xSetWindowAttributes struct {
	BackgroundPixmap   uintptr
	BackgroundPixel    uintptr
	BorderPixmap       uintptr
	BorderPixel        uintptr
	BitGravity         int32
	WinGravity         int32
	BackingStore       int32
	BackingPlanes      uintptr
	BackingPixel       uintptr
	SaveUnder          int32
	EventMask          int64
	DoNotPropagateMask int64
	OverrideRedirect   int32
	Colormap           uintptr
	Cursor             uintptr
}

// XWindowAttributes, field order per Xlib.h.
type xWindowAttributes struct {
	X, Y               int32
	Width, Height      int32
	BorderWidth        int32
	Depth              int32
	Visual             uintptr
	Root               uintptr
	Class              int32
	BitGravity         int32
	WinGravity         int32
	BackingStore       int32
	BackingPlanes      uintptr
	BackingPixel       uintptr
	SaveUnder          int32
	Colormap           uintptr
	MapInstalled       int32
	MapState           int32
	AllEventMasks      int64
	YourEventMask      int64
	DoNotPropagateMask int64
	OverrideRedirect   int32
	Screen             uintptr
}

// xEvent is XEvent's storage: a 24-long (192-byte on LP64) union. Individual
// event struct types below are overlaid onto it with unsafe.Pointer, exactly
// like casting a C union member - same technique the old cgo code used.
type xEvent [192]byte

func (e *xEvent) eventType() int32 {
	return *(*int32)(unsafe.Pointer(e))
}

// window returns the XAnyEvent.window field: the window the event was reported for.
func (e *xEvent) window() uintptr {
	return (*xKeyEvent)(unsafe.Pointer(e)).Window
}

type xKeyEvent struct {
	Type         int32
	Serial       uintptr
	SendEvent    int32
	Display      uintptr
	Window       uintptr
	Root         uintptr
	Subwindow    uintptr
	Time         uintptr
	X, Y         int32
	XRoot, YRoot int32
	State        uint32
	Keycode      uint32
	SameScreen   int32
}

type xButtonEvent struct {
	Type         int32
	Serial       uintptr
	SendEvent    int32
	Display      uintptr
	Window       uintptr
	Root         uintptr
	Subwindow    uintptr
	Time         uintptr
	X, Y         int32
	XRoot, YRoot int32
	State        uint32
	Button       uint32
	SameScreen   int32
}

// XFocusChangeEvent, field order per Xlib.h
type xFocusChangeEvent struct {
	Type      int32
	Serial    uintptr
	SendEvent int32
	Display   uintptr
	Window    uintptr
	Mode      int32
	Detail    int32
}

const (
	xNotifyNormal       = 0
	xNotifyWhileGrabbed = 3
	xNotifyInferior     = 2
)

type xMotionEvent struct {
	Type         int32
	Serial       uintptr
	SendEvent    int32
	Display      uintptr
	Window       uintptr
	Root         uintptr
	Subwindow    uintptr
	Time         uintptr
	X, Y         int32
	XRoot, YRoot int32
	State        uint32
	IsHint       int8
	SameScreen   int32
}

type xMapEvent struct {
	Type             int32
	Serial           uintptr
	SendEvent        int32
	Display          uintptr
	Event            uintptr
	Window           uintptr
	OverrideRedirect int32
}

type xUnmapEvent struct {
	Type          int32
	Serial        uintptr
	SendEvent     int32
	Display       uintptr
	Event         uintptr
	Window        uintptr
	FromConfigure int32
}

type xDestroyWindowEvent struct {
	Type      int32
	Serial    uintptr
	SendEvent int32
	Display   uintptr
	Event     uintptr
	Window    uintptr
}

type xReparentEvent struct {
	Type             int32
	Serial           uintptr
	SendEvent        int32
	Display          uintptr
	Event            uintptr
	Window           uintptr
	Parent           uintptr
	X, Y             int32
	OverrideRedirect int32
}

type xConfigureEvent struct {
	Type             int32
	Serial           uintptr
	SendEvent        int32
	Display          uintptr
	Event            uintptr
	Window           uintptr
	X, Y             int32
	Width, Height    int32
	BorderWidth      int32
	Above            uintptr
	OverrideRedirect int32
}

type xResizeRequestEvent struct {
	Type          int32
	Serial        uintptr
	SendEvent     int32
	Display       uintptr
	Window        uintptr
	Width, Height int32
}

type xPropertyEvent struct {
	Type      int32
	Serial    uintptr
	SendEvent int32
	Display   uintptr
	Window    uintptr
	Atom      uintptr
	Time      uintptr
	State     int32
}

type xClientMessageEvent struct {
	Type        int32
	Serial      uintptr
	SendEvent   int32
	Display     uintptr
	Window      uintptr
	MessageType uintptr
	Format      int32
	// union { char b[20]; short s[10]; long l[5] }; long[5] (8*5) is the
	// largest member, and it's also what forces the union - and everything
	// after it in the enclosing struct - onto an 8-byte boundary. [5]int64
	// (rather than [40]byte) matters here specifically to get Go to insert
	// that same padding after Format: a byte array only demands 1-byte
	// alignment, which put Data 4 bytes too early and silently misread
	// every ClientMessage (including WM_DELETE_WINDOW).
	Data [5]int64
}

func (e *xClientMessageEvent) setDataLong(i int, v int64) {
	e.Data[i] = v
}

func (e *xClientMessageEvent) dataLong(i int) int64 {
	return e.Data[i]
}

var (
	xOpenDisplay          func(displayName uintptr) uintptr
	xCloseDisplay         func(display uintptr) int32
	xDefaultScreen        func(display uintptr) int32
	xDefaultRootWindow    func(display uintptr) uintptr
	xRootWindow           func(display uintptr, screen int32) uintptr
	xCreateWindow         func(display, parent uintptr, x, y int32, width, height uint32, borderWidth uint32, depth int32, class uint32, visual uintptr, valuemask uintptr, attrs unsafe.Pointer) uintptr
	xSelectInput          func(display, window uintptr, eventMask int64) int32
	xGetWindowAttributes  func(display, window uintptr, attrs unsafe.Pointer) int32
	xMapWindow            func(display, window uintptr) int32
	xFlush                func(display uintptr) int32
	xConnectionNumber     func(display uintptr) int32
	xSetWindowBackground  func(display, window uintptr, pixel uintptr) int32
	xClearArea            func(display, window uintptr, x, y int32, width, height uint32, exposures int32) int32
	xPending              func(display uintptr) int32
	xNextEvent            func(display uintptr, event unsafe.Pointer) int32
	xDestroyWindow        func(display, window uintptr) int32
	xStoreName            func(display, window uintptr, name string) int32
	xInternAtom           func(display uintptr, name string, onlyIfExists int32) uintptr
	xChangeProperty       func(display, window, property, typ uintptr, format, mode int32, data unsafe.Pointer, nelements int32) int32
	xSetWMProtocols       func(display, window uintptr, protocols unsafe.Pointer, count int32) int32
	xMoveWindow           func(display, window uintptr, x, y int32) int32
	xResizeWindow         func(display, window uintptr, width, height uint32) int32
	xDisplayWidth         func(display uintptr, screen int32) int32
	xDisplayHeight        func(display uintptr, screen int32) int32
	xGetWindowProperty    func(display, window, property uintptr, longOffset, longLength int64, delete int32, reqType uintptr, actualTypeReturn, actualFormatReturn, nitemsReturn, bytesAfterReturn, propReturn unsafe.Pointer) int32
	xFree                 func(data unsafe.Pointer) int32
	xCreateFontCursor     func(display uintptr, shape uint32) uintptr
	xDefineCursor         func(display, window, cursor uintptr) int32
	xQueryPointer         func(display, window uintptr, rootReturn, childReturn, rootXReturn, rootYReturn, winXReturn, winYReturn, maskReturn unsafe.Pointer) int32
	xLookupKeysym         func(event unsafe.Pointer, index int32) uintptr
	xLookupString         func(event unsafe.Pointer, bufferReturn unsafe.Pointer, bytesBuffer int32, keysymReturn unsafe.Pointer, status uintptr) int32
	xTranslateCoordinates func(display, srcW, destW uintptr, srcX, srcY int32, destXReturn, destYReturn, childReturn unsafe.Pointer) int32
	xCreateGC             func(display, drawable uintptr, valuemask uintptr, values uintptr) uintptr
	xFreeGC               func(display, gc uintptr) int32
	xCreateImage          func(display, visual uintptr, depth uint32, format, offset int32, data unsafe.Pointer, width, height uint32, bitmapPad, bytesPerLine int32) uintptr
	xPutImage             func(display, drawable, gc, image uintptr, srcX, srcY, destX, destY int32, width, height uint32) int32
	xDestroyImage         func(image uintptr) int32
	xDefaultVisual        func(display uintptr, screen int32) uintptr
	xSendEvent            func(display, window uintptr, propagate int32, eventMask int64, event unsafe.Pointer) int32
	xSetTransientForHint  func(display, window, propWindow uintptr) int32
	xIconifyWindow        func(display, window uintptr, screenNumber int32) int32
	xBell                 func(display uintptr, percent int32) int32
	xInitThreads          func() int32
	xMapRaised            func(display, window uintptr) int32
	xUnmapWindow          func(display, window uintptr) int32

	libcSetlocale func(category int32, locale string) uintptr
	libcMalloc    func(size uintptr) unsafe.Pointer
	libcFree      func(ptr uintptr)

	// Xinerama is optional: it's present on virtually every X11 desktop but
	// isn't a hard dependency the way libX11/libc are, so its absence just
	// falls back to whole-screen centering (see monitorRectForWindow)
	// instead of panicking.
	xineramaAvailable    bool
	xineramaIsActive     func(display uintptr) int32
	xineramaQueryScreens func(display uintptr, numberReturn unsafe.Pointer) unsafe.Pointer

	// XShape (libXext) is optional too: it only makes popups transparent to
	// the mouse (see createPopupWindow).
	xShapeAvailable         bool
	xShapeCombineRectangles func(display, window uintptr, destKind, xOff, yOff int32, rectangles unsafe.Pointer, nRects, op, ordering int32)
)

// XineramaScreenInfo, field order per X11/extensions/Xinerama.h.
type xineramaScreenInfo struct {
	ScreenNumber  int32
	XOrg, YOrg    int16
	Width, Height int16
}

func dlopenFirst(names ...string) (uintptr, error) {
	var lastErr error
	for _, name := range names {
		h, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err == nil {
			return h, nil
		}
		lastErr = err
	}
	return 0, fmt.Errorf("nui: unable to load any of %v: %w", names, lastErr)
}

func init() {
	libX11, err := dlopenFirst("libX11.so.6", "libX11.so")
	if err != nil {
		panic(err)
	}

	libc, err := dlopenFirst("libc.so.6", "libc.so")
	if err != nil {
		panic(err)
	}

	purego.RegisterLibFunc(&xOpenDisplay, libX11, "XOpenDisplay")
	purego.RegisterLibFunc(&xCloseDisplay, libX11, "XCloseDisplay")
	purego.RegisterLibFunc(&xDefaultScreen, libX11, "XDefaultScreen")
	purego.RegisterLibFunc(&xDefaultRootWindow, libX11, "XDefaultRootWindow")
	purego.RegisterLibFunc(&xRootWindow, libX11, "XRootWindow")
	purego.RegisterLibFunc(&xCreateWindow, libX11, "XCreateWindow")
	purego.RegisterLibFunc(&xSelectInput, libX11, "XSelectInput")
	purego.RegisterLibFunc(&xGetWindowAttributes, libX11, "XGetWindowAttributes")
	purego.RegisterLibFunc(&xMapWindow, libX11, "XMapWindow")
	purego.RegisterLibFunc(&xFlush, libX11, "XFlush")
	purego.RegisterLibFunc(&xConnectionNumber, libX11, "XConnectionNumber")
	purego.RegisterLibFunc(&xSetWindowBackground, libX11, "XSetWindowBackground")
	purego.RegisterLibFunc(&xClearArea, libX11, "XClearArea")
	purego.RegisterLibFunc(&xPending, libX11, "XPending")
	purego.RegisterLibFunc(&xNextEvent, libX11, "XNextEvent")
	purego.RegisterLibFunc(&xDestroyWindow, libX11, "XDestroyWindow")
	purego.RegisterLibFunc(&xStoreName, libX11, "XStoreName")
	purego.RegisterLibFunc(&xInternAtom, libX11, "XInternAtom")
	purego.RegisterLibFunc(&xChangeProperty, libX11, "XChangeProperty")
	purego.RegisterLibFunc(&xSetWMProtocols, libX11, "XSetWMProtocols")
	purego.RegisterLibFunc(&xMoveWindow, libX11, "XMoveWindow")
	purego.RegisterLibFunc(&xResizeWindow, libX11, "XResizeWindow")
	purego.RegisterLibFunc(&xDisplayWidth, libX11, "XDisplayWidth")
	purego.RegisterLibFunc(&xDisplayHeight, libX11, "XDisplayHeight")
	purego.RegisterLibFunc(&xGetWindowProperty, libX11, "XGetWindowProperty")
	purego.RegisterLibFunc(&xFree, libX11, "XFree")
	purego.RegisterLibFunc(&xCreateFontCursor, libX11, "XCreateFontCursor")
	purego.RegisterLibFunc(&xDefineCursor, libX11, "XDefineCursor")
	purego.RegisterLibFunc(&xQueryPointer, libX11, "XQueryPointer")
	purego.RegisterLibFunc(&xLookupKeysym, libX11, "XLookupKeysym")
	purego.RegisterLibFunc(&xLookupString, libX11, "XLookupString")
	purego.RegisterLibFunc(&xTranslateCoordinates, libX11, "XTranslateCoordinates")
	purego.RegisterLibFunc(&xCreateGC, libX11, "XCreateGC")
	purego.RegisterLibFunc(&xFreeGC, libX11, "XFreeGC")
	purego.RegisterLibFunc(&xCreateImage, libX11, "XCreateImage")
	purego.RegisterLibFunc(&xPutImage, libX11, "XPutImage")
	purego.RegisterLibFunc(&xDestroyImage, libX11, "XDestroyImage")
	purego.RegisterLibFunc(&xDefaultVisual, libX11, "XDefaultVisual")
	purego.RegisterLibFunc(&xSendEvent, libX11, "XSendEvent")
	purego.RegisterLibFunc(&xSetTransientForHint, libX11, "XSetTransientForHint")
	purego.RegisterLibFunc(&xIconifyWindow, libX11, "XIconifyWindow")
	purego.RegisterLibFunc(&xBell, libX11, "XBell")
	purego.RegisterLibFunc(&xInitThreads, libX11, "XInitThreads")
	purego.RegisterLibFunc(&xMapRaised, libX11, "XMapRaised")
	purego.RegisterLibFunc(&xUnmapWindow, libX11, "XUnmapWindow")

	purego.RegisterLibFunc(&libcSetlocale, libc, "setlocale")
	purego.RegisterLibFunc(&libcMalloc, libc, "malloc")
	purego.RegisterLibFunc(&libcFree, libc, "free")

	libcSetlocale(xLCAll, "")
	// Required before any Xlib call once multiple windows run their event loops on separate goroutines.
	xInitThreads()

	if h, err := dlopenFirst("libXinerama.so.1", "libXinerama.so"); err == nil {
		purego.RegisterLibFunc(&xineramaIsActive, h, "XineramaIsActive")
		purego.RegisterLibFunc(&xineramaQueryScreens, h, "XineramaQueryScreens")
		xineramaAvailable = true
	}

	if h, err := dlopenFirst("libXext.so.6", "libXext.so"); err == nil {
		purego.RegisterLibFunc(&xShapeCombineRectangles, h, "XShapeCombineRectangles")
		xShapeAvailable = true
	}
}

// destroyXImage frees an XImage* created by xCreateImage, including its
// malloc'd pixel buffer (see (*nativeWindow).drawImageRGBA).
func destroyXImage(image uintptr) {
	xDestroyImage(image)
}

// maximizeWindowX/restoreWindowX/minimizeWindowX/setWindowModalX/setWindowDecorationsX
// port nui/ximage_helper.c's helpers (kept there for reference/the old cgo
// build) to pure Go, calling the same Xlib entry points.

func setNetWMStateMaximized(display, window uintptr, add bool) {
	wmState := xInternAtom(display, "_NET_WM_STATE", xFalse)
	maxH := xInternAtom(display, "_NET_WM_STATE_MAXIMIZED_HORZ", xFalse)
	maxV := xInternAtom(display, "_NET_WM_STATE_MAXIMIZED_VERT", xFalse)

	var ev xClientMessageEvent
	ev.Type = xClientMessage
	ev.SendEvent = xTrue
	ev.Window = window
	ev.MessageType = wmState
	ev.Format = 32
	if add {
		ev.setDataLong(0, 1) // _NET_WM_STATE_ADD
	}
	ev.setDataLong(1, int64(maxH))
	ev.setDataLong(2, int64(maxV))

	root := xDefaultRootWindow(display)
	xSendEvent(display, root, xFalse, xSubstructureRedirectMask|xSubstructureNotifyMask, unsafe.Pointer(&ev))
}

// setNetWMState asks the window manager to add or remove a _NET_WM_STATE_* state
func setNetWMState(display, window uintptr, add bool, state string) {
	var ev xClientMessageEvent
	ev.Type = xClientMessage
	ev.SendEvent = xTrue
	ev.Window = window
	ev.MessageType = xInternAtom(display, "_NET_WM_STATE", xFalse)
	ev.Format = 32
	if add {
		ev.setDataLong(0, 1) // _NET_WM_STATE_ADD
	}
	ev.setDataLong(1, int64(xInternAtom(display, state, xFalse)))

	root := xDefaultRootWindow(display)
	xSendEvent(display, root, xFalse, xSubstructureRedirectMask|xSubstructureNotifyMask, unsafe.Pointer(&ev))
	xFlush(display)
}

func maximizeWindowX(display, window uintptr) {
	setNetWMStateMaximized(display, window, true)
}

func restoreWindowX(display, window uintptr) {
	setNetWMStateMaximized(display, window, false)
}

func minimizeWindowX(display, window uintptr) {
	screen := xDefaultScreen(display)
	xIconifyWindow(display, window, screen)
}

// setWindowModalX marks window as a modal dialog owned by parent:
// sets WM_TRANSIENT_FOR so window managers group/stack it with its parent,
// tags it as a dialog via _NET_WM_WINDOW_TYPE, and requests
// _NET_WM_STATE_MODAL so compliant window managers (GNOME/KDE/XFCE) block
// input to the parent while it's open.
func setWindowModalX(display, window, parent uintptr) {
	xSetTransientForHint(display, window, parent)

	wmWindowType := xInternAtom(display, "_NET_WM_WINDOW_TYPE", xFalse)
	wmWindowTypeDialog := xInternAtom(display, "_NET_WM_WINDOW_TYPE_DIALOG", xFalse)
	xChangeProperty(display, window, wmWindowType, xXAAtom, 32, xPropModeReplace, unsafe.Pointer(&wmWindowTypeDialog), 1)

	wmState := xInternAtom(display, "_NET_WM_STATE", xFalse)
	stateModal := xInternAtom(display, "_NET_WM_STATE_MODAL", xFalse)

	var ev xClientMessageEvent
	ev.Type = xClientMessage
	ev.SendEvent = xTrue
	ev.Window = window
	ev.MessageType = wmState
	ev.Format = 32
	ev.setDataLong(0, 1) // _NET_WM_STATE_ADD
	ev.setDataLong(1, int64(stateModal))

	root := xDefaultRootWindow(display)
	xSendEvent(display, root, xFalse, xSubstructureRedirectMask|xSubstructureNotifyMask, unsafe.Pointer(&ev))
}

// Motif WM hints: the de-facto cross-desktop (GNOME/KDE/XFCE) convention for
// asking a window manager to add/remove specific titlebar decorations and
// functions on an otherwise plain X11 top-level window. Not part of any
// ICCCM/EWMH spec, but this exact 5-field, format-32 layout is what
// GTK/Qt/wxWidgets all write, so window managers reliably honor it.
type motifWmHints struct {
	Flags       uint64
	Functions   uint64
	Decorations uint64
	InputMode   int64
	Status      uint64
}

const (
	mwmHintsFunctions   = 1 << 0
	mwmHintsDecorations = 1 << 1

	mwmFuncAll      = 1 << 0
	mwmFuncMinimize = 1 << 3
	mwmFuncMaximize = 1 << 4

	mwmDecorAll      = 1 << 0
	mwmDecorMinimize = 1 << 5
	mwmDecorMaximize = 1 << 6
)

// setWindowDecorationsX sets/clears the minimize/maximize _MOTIF_WM_HINTS bits.
func setWindowDecorationsX(display, window uintptr, allowMinimize, allowMaximize bool) {
	motifHints := xInternAtom(display, "_MOTIF_WM_HINTS", xFalse)
	if motifHints == xNone {
		return
	}

	hints := motifWmHints{
		Flags:       mwmHintsFunctions | mwmHintsDecorations,
		Functions:   mwmFuncAll,
		Decorations: mwmDecorAll,
	}
	if !allowMinimize {
		hints.Functions |= mwmFuncMinimize
		hints.Decorations |= mwmDecorMinimize
	}
	if !allowMaximize {
		hints.Functions |= mwmFuncMaximize
		hints.Decorations |= mwmDecorMaximize
	}

	xChangeProperty(display, window, motifHints, motifHints, 32, xPropModeReplace, unsafe.Pointer(&hints), 5)
	xFlush(display)
}

// netWorkAreaX reads the first desktop's _NET_WORKAREA from the root window:
// the virtual desktop minus panels/docks, as set by the window manager.
func netWorkAreaX(display uintptr, screen int32) (x, y, width, height int, ok bool) {
	atom := xInternAtom(display, "_NET_WORKAREA", xTrue)
	if atom == xNone {
		return 0, 0, 0, 0, false
	}

	var actualType uintptr
	var actualFormat int32
	var nitems uintptr
	var bytesAfter uintptr
	var prop unsafe.Pointer

	status := xGetWindowProperty(
		display,
		xRootWindow(display, screen),
		atom,
		0,
		4,
		xFalse,
		xXACardinal,
		unsafe.Pointer(&actualType),
		unsafe.Pointer(&actualFormat),
		unsafe.Pointer(&nitems),
		unsafe.Pointer(&bytesAfter),
		unsafe.Pointer(&prop),
	)
	if status != xSuccess || prop == nil {
		return 0, 0, 0, 0, false
	}
	defer xFree(prop)

	if actualType != xXACardinal || actualFormat != 32 || nitems < 4 {
		return 0, 0, 0, 0, false
	}

	// Format 32 properties come back as C longs
	data := (*[4]uintptr)(prop)
	return int(data[0]), int(data[1]), int(data[2]), int(data[3]), true
}

// monitorRectForWindow returns the bounds of the monitor that contains the
// largest portion of the window rect (winX, winY, winWidth, winHeight) - not
// just "the primary monitor" or the whole multi-monitor virtual desktop.
// Falls back to the full X screen (what XDisplayWidth/XDisplayHeight report)
// when Xinerama isn't available or reports no active heads, which is also
// exactly right for a single-monitor setup.
func monitorRectForWindow(display uintptr, screen int32, winX, winY, winWidth, winHeight int) (x, y, width, height int) {
	fullWidth := int(xDisplayWidth(display, screen))
	fullHeight := int(xDisplayHeight(display, screen))

	if !xineramaAvailable || xineramaIsActive(display) == 0 {
		return 0, 0, fullWidth, fullHeight
	}

	var count int32
	screens := xineramaQueryScreens(display, unsafe.Pointer(&count))
	if screens == nil || count == 0 {
		return 0, 0, fullWidth, fullHeight
	}
	defer xFree(screens)

	infos := unsafe.Slice((*xineramaScreenInfo)(screens), int(count))

	best := 0
	bestArea := -1
	for i, m := range infos {
		area := rectOverlapArea(winX, winY, winWidth, winHeight, int(m.XOrg), int(m.YOrg), int(m.Width), int(m.Height))
		if area > bestArea {
			bestArea = area
			best = i
		}
	}

	// The window doesn't currently overlap any monitor (e.g. it was moved
	// fully off-screen) - pick whichever monitor's center is closest instead
	// of defaulting to head 0, so it still lands somewhere reasonable.
	if bestArea <= 0 {
		best = nearestMonitor(infos, winX, winY, winWidth, winHeight)
	}

	m := infos[best]
	return int(m.XOrg), int(m.YOrg), int(m.Width), int(m.Height)
}

func rectOverlapArea(ax, ay, aw, ah, bx, by, bw, bh int) int {
	left := max(ax, bx)
	top := max(ay, by)
	right := min(ax+aw, bx+bw)
	bottom := min(ay+ah, by+bh)
	if right <= left || bottom <= top {
		return 0
	}
	return (right - left) * (bottom - top)
}

func nearestMonitor(infos []xineramaScreenInfo, winX, winY, winWidth, winHeight int) int {
	cx := winX + winWidth/2
	cy := winY + winHeight/2

	best := 0
	bestDist := -1
	for i, m := range infos {
		mcx := int(m.XOrg) + int(m.Width)/2
		mcy := int(m.YOrg) + int(m.Height)/2
		dx := cx - mcx
		dy := cy - mcy
		dist := dx*dx + dy*dy
		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			best = i
		}
	}
	return best
}
