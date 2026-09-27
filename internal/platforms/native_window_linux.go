//go:build linux
// +build linux

package platforms

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"
)

type windowId uintptr

type nativeWindowPlatform struct {
	display uintptr
	window  uintptr
	screen  int32

	closed bool
	// Set by Close() from any goroutine; only the owning Exec goroutine
	// acts on it and actually tears down the Display, avoiding a use-after-free
	// if Close() is called from another window's goroutine.
	closeRequested int32

	lastMouseDownX      int
	lastMouseDownY      int
	lastMouseDownButton MouseButton
	lastMouseDownTime   time.Time

	// The window is painted once all the pending events are handled, so a
	// burst of Expose events or Update calls results in one paint. Atomic:
	// Update may be called from another goroutine.
	needPaint atomic.Bool

	// The size of the last ConfigureNotify, applied (laid out) once all the
	// pending events are handled: a window being resized gets many of them
	pendingResize               bool
	pendingWidth, pendingHeight int

	wmProtocols    uintptr
	wmDeleteWindow uintptr

	prevSetPosX int
	prevSetPosY int

	netWMState              uintptr
	netWMStateMaximizedHorz uintptr
	netWMStateMaximizedVert uintptr

	// Per-window paint surface, sized to the window's current dimensions.
	canvasBuffer []byte
	bgColor      color.RGBA

	// setWindowDecorations() sets the minimize/maximize _MOTIF_WM_HINTS bits
	// together as one property, so SetAllowMinimize/SetAllowMaximize each
	// need the other's last-set value on hand to avoid clobbering it.
	allowMinimize bool
	allowMaximize bool

	// Show() maps the window and spawns pumpEvents() on its own goroutine
	// exactly once (pumpStarted guards that), so callers never have to know
	// this needs its own goroutine at all - Exec() just waits on
	// pumpDone. Xlib itself has no thread-affinity requirement like Win32's
	// (XInitThreads(), called at package init, is enough to let another
	// goroutine safely drive this Display later), so unlike the Windows
	// backend this doesn't need a dedicated locked OS thread from creation.
	pumpStarted bool
	pumpDone    chan struct{}

	// modalChildCount counts this window's currently-open ShowModal children.
	// While > 0, pumpEvents drops keyboard/mouse callbacks for this window:
	// kwin_x11 (and apparently other Linux WMs) accepts _NET_WM_STATE_MODAL
	// and withholds keyboard focus from the parent, but still happily
	// delivers pointer button/motion events straight to it, so nui has to
	// enforce the block itself. Accessed with atomics since ShowModal/doClose
	// can run on a different goroutine than this window's own pump loop.
	modalChildCount int32

	// modalParent is the window this dialog was shown modally over (set by
	// ShowModal), kept so doClose can decrement its modalChildCount and
	// un-block it again once this dialog closes.
	modalParent *nativeWindow
}

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
var hwndsMu sync.Mutex

func init() {
	hwnds = make(map[windowId]*nativeWindow)
}

func GetNativeWindowByHandle(hwnd uintptr) *nativeWindow {
	hwndsMu.Lock()
	defer hwndsMu.Unlock()
	if w, ok := hwnds[windowId(hwnd)]; ok {
		return w
	}
	return nil
}

func getHDCSize(hdc uintptr) (width int32, height int32) {
	var r rect
	return r.right - r.left, r.bottom - r.top
}

// Sanity caps on a single window's paintable area, not a shared buffer size.
const maxCanvasWidth = 10000
const maxCanvasHeight = 5000

// ensureCanvasBuffer grows this window's own paint buffer to fit size bytes, if needed.
func (c *nativeWindow) ensureCanvasBuffer(size int) []byte {
	if cap(c.platform.canvasBuffer) < size {
		c.platform.canvasBuffer = make([]byte, size)
	} else {
		c.platform.canvasBuffer = c.platform.canvasBuffer[:size]
	}
	return c.platform.canvasBuffer
}

// fillCanvasBuffer paints buf with this window's solid background color.
func fillCanvasBuffer(buf []byte, col color.RGBA) {
	if len(buf) < 4 {
		return
	}
	buf[0], buf[1], buf[2], buf[3] = col.B, col.G, col.R, col.A
	// Doubling the filled part: a few large copies instead of a loop over the pixels
	for filled := 4; filled < len(buf); filled *= 2 {
		copy(buf[filled:], buf[:filled])
	}
}

///////////////////////////////////////////////////////////////////

func createWindow(title string, posX int, posY int, width int, height int, center bool, maximized bool) *nativeWindow {
	var c nativeWindow
	c.showMaximized = maximized
	c.platform.bgColor = color.RGBA{0, 50, 0, 255}
	c.platform.allowMinimize = true
	c.platform.allowMaximize = true
	c.platform.prevSetPosX = -1
	c.platform.prevSetPosY = -1

	c.platform.display = xOpenDisplay(0)
	if c.platform.display == 0 {
		panic("Unable to open X display")
	}

	c.platform.screen = xDefaultScreen(c.platform.display)

	// While resizing, the X server keeps the old content in the top-left
	// corner and fills only the new area with the background color until
	// the window is painted again, instead of discarding it all
	attrs := xSetWindowAttributes{}
	attrs.BackgroundPixel = backgroundPixel(c.platform.bgColor)
	attrs.BitGravity = xNorthWestGravity

	c.platform.window = xCreateWindow(
		c.platform.display,
		xRootWindow(c.platform.display, c.platform.screen),
		100, 100, // x, y
		uint32(width), uint32(height), // width, height
		1,                          // border width
		xCopyFromParent,            // depth
		xInputOutput,               // class
		0,                          // visual
		xCWBackPixel|xCWBitGravity, // valuemask
		unsafe.Pointer(&attrs),
	)

	xSelectInput(c.platform.display, c.platform.window, xExposureMask|xPropertyChangeMask|xStructureNotifyMask|xKeyPressMask|xKeyReleaseMask|xEnterWindowMask|xLeaveWindowMask|xButtonPressMask|xButtonReleaseMask|xPointerMotionMask|xFocusChangeMask)

	var getAttr xWindowAttributes
	xGetWindowAttributes(c.platform.display, c.platform.window, unsafe.Pointer(&getAttr))
	c.windowWidth, c.windowHeight = int(getAttr.Width), int(getAttr.Height)

	// Store the window handle
	hwndsMu.Lock()
	hwnds[windowId(c.platform.window)] = &c
	hwndsMu.Unlock()

	// Set default icon
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	c.SetAppIcon(icon)

	c.SetTitle(title)

	c.initCloseProtocol()
	c.initWindowStateAtoms()

	return &c
}

func (c *nativeWindow) initCloseProtocol() {
	display := c.platform.display
	window := c.platform.window

	c.platform.wmProtocols = xInternAtom(display, "WM_PROTOCOLS", xFalse)
	c.platform.wmDeleteWindow = xInternAtom(display, "WM_DELETE_WINDOW", xFalse)

	xSetWMProtocols(display, window, unsafe.Pointer(&c.platform.wmDeleteWindow), 1)
}

func (c *nativeWindow) initWindowStateAtoms() {
	display := c.platform.display

	c.platform.netWMState = xInternAtom(display, "_NET_WM_STATE", xFalse)
	c.platform.netWMStateMaximizedHorz = xInternAtom(display, "_NET_WM_STATE_MAXIMIZED_HORZ", xFalse)
	c.platform.netWMStateMaximizedVert = xInternAtom(display, "_NET_WM_STATE_MAXIMIZED_VERT", xFalse)
}

// Show maps the window and, the first time it's called, starts this
// window's own event pump on a new goroutine - callers never need to spawn
// one themselves (compare ShowModal, which has always hidden this the same
// way). Safe to call more than once; only the first call does anything.
func (c *nativeWindow) Show() {
	if c.platform.pumpStarted {
		return
	}
	c.platform.pumpStarted = true
	c.platform.pumpDone = make(chan struct{})

	xMapWindow(c.platform.display, c.platform.window)
	xFlush(c.platform.display)

	go func() {
		c.pumpEvents()
		close(c.platform.pumpDone)
	}()
}

func (c *nativeWindow) Hide() {
}

// Update asks to paint the window: pumpEvents paints it once the pending
// events are handled, within a timer tick
func (c *nativeWindow) Update() {
	c.platform.needPaint.Store(true)
}

// Exec blocks the calling goroutine until this window closes. The actual
// X11 event pump runs on the goroutine Show() started (idempotent, so
// calling it here covers a bare Exec() call with no prior Show()).
func (c *nativeWindow) Exec() {
	c.Show()
	<-c.platform.pumpDone
}

// pumpEvents is the real XPending/XNextEvent loop, run on the goroutine
// Show() spawns for the life of the window.
func (c *nativeWindow) pumpEvents() {
	const tickInterval = 10 * time.Millisecond
	nextTick := time.Now().Add(tickInterval)

	for !c.platform.closed {
		// A close requested while a modal dialog of this window is open waits
		// until the dialog is gone (see Close)
		if atomic.LoadInt32(&c.platform.closeRequested) != 0 && !c.inputBlocked() {
			c.doClose()
			break
		}

		for !c.platform.closed && xPending(c.platform.display) > 0 {
			var event xEvent
			xNextEvent(c.platform.display, unsafe.Pointer(&event))

			if c.platform.closed {
				break
			}

			// Popups share this window's Display, so their events arrive here
			if w := event.window(); w != c.platform.window {
				if p := getPopupByWindow(w); p != nil {
					p.processEvent(&event)
					continue
				}
			}

			// A Move made before the WM framed the window is repeated once
			// the frame is known. Checked only while one is pending: it asks
			// the X server and waits for the answer.
			if c.platform.prevSetPosX >= 0 && c.platform.prevSetPosY >= 0 {
				if _, _, _, _, ok := c.getFrameExtents(); ok {
					c.Move(c.platform.prevSetPosX, c.platform.prevSetPosY)
					c.platform.prevSetPosX = -1
					c.platform.prevSetPosY = -1
				}
			}

			// A modal dialog owned by this window is open: kwin_x11 (and
			// apparently other Linux WMs) withholds keyboard focus from us
			// but still delivers pointer/keyboard events straight to this
			// window, so drop them here ourselves instead of dispatching to
			// app callbacks. Once the dialog closes, doClose() drops
			// modalChildCount back to 0 and these events flow again.
			if c.inputBlocked() {
				switch event.eventType() {
				case xKeyPress, xKeyRelease, xButtonPress, xButtonRelease, xMotionNotify, xEnterNotify, xLeaveNotify:
					continue
				}
			}

			switch event.eventType() {

			case xExpose:
				// The whole window is painted anyway: once, after the queue
				c.platform.needPaint.Store(true)
			case xMapNotify:
				mapEvent := (*xMapEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window became visible. Window ID: %d\n", mapEvent.Window)

			case xUnmapNotify:
				unmapEvent := (*xUnmapEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window was hidden. Window ID: %d\n", unmapEvent.Window)

				// The WM just iconified us (titlebar button, window menu,
				// keyboard shortcut - ICCCM has the client unmap itself to go
				// Iconic regardless of which one triggered it), but
				// SetAllowMinimize(false) says this window shouldn't be
				// minimizable. Since kwin's decoration doesn't reliably honor
				// the _MOTIF_WM_HINTS decorations bits for button visibility
				// on every theme, undo the effect directly: re-map right
				// away instead of trying to prevent the click itself.
				if !c.platform.allowMinimize && !c.platform.closed && atomic.LoadInt32(&c.platform.closeRequested) == 0 {
					xMapWindow(c.platform.display, c.platform.window)
					xFlush(c.platform.display)
				}

			case xDestroyNotify:
				destroyEvent := (*xDestroyWindowEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window was destroyed. Window ID: %d\n", destroyEvent.Window)

			case xReparentNotify:
				reparentEvent := (*xReparentEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window changed parent. Window ID: %d, New Parent ID: %d\n", reparentEvent.Window, reparentEvent.Parent)
			case xResizeRequest:
				resizeEvent := (*xResizeRequestEvent)(unsafe.Pointer(&event))
				fmt.Printf("Resize request received: Width=%d, Height=%d\n", resizeEvent.Width, resizeEvent.Height)

				c.windowWidth = int(resizeEvent.Width)
				c.windowHeight = int(resizeEvent.Height)

			case xConfigureNotify:
				configureEvent := (*xConfigureEvent)(unsafe.Pointer(&event))

				prevWindowPosX := c.windowPosX
				prevWindowPosY := c.windowPosY

				c.updateWindowPos()

				if configureEvent.SendEvent == 1 && (c.windowPosX != prevWindowPosX || c.windowPosY != prevWindowPosY) {
					if c.onMove != nil {
						c.onMove(c.windowPosX, c.windowPosY)
					}
				}

				if configureEvent.SendEvent == 0 {
					c.platform.pendingResize = true
					c.platform.pendingWidth = int(configureEvent.Width)
					c.platform.pendingHeight = int(configureEvent.Height)
				}

				c.Update()

			case xKeyPress:
				keyEvent := (*xKeyEvent)(unsafe.Pointer(&event))
				keySym := xLookupKeysym(unsafe.Pointer(&event), 0)
				fmt.Printf("Key pressed: KeySym = %d, KeyCode = 0x%x\n", keySym, keyEvent.Keycode)
				key := ConvertLinuxKeyToNuiKey(int(keyEvent.Keycode))
				processed := false
				if c.onKeyDown != nil {
					processed = c.onKeyDown(key, c.getModifierState())
				}

				if processed {
					break
				}

				if c.platform.closed {
					break
				}

				var buf [32]byte
				var sym uintptr

				n := xLookupString(
					unsafe.Pointer(&event),
					unsafe.Pointer(&buf[0]),
					int32(len(buf)),
					unsafe.Pointer(&sym),
					0,
				)

				if n > 0 {
					text := string(buf[:n])
					fmt.Printf("Text input: %s\n", text)

					firstRune, _ := utf8.DecodeRuneInString(text)
					if firstRune > 0 && firstRune != 127 {
						if c.onChar != nil {
							c.onChar(firstRune)
						}
					}
				}

			case xKeyRelease:
				keyEvent := (*xKeyEvent)(unsafe.Pointer(&event))
				keySym := xLookupKeysym(unsafe.Pointer(&event), 0)
				fmt.Printf("Key released: KeySym = %d, KeyCode = 0x%x\n", keySym, keyEvent.Keycode)
				key := ConvertLinuxKeyToNuiKey(int(keyEvent.Keycode))
				if c.onKeyUp != nil {
					c.onKeyUp(key, c.getModifierState())
				}

			case xFocusOut:
				focusEvent := (*xFocusChangeEvent)(unsafe.Pointer(&event))
				// Skip the temporary focus changes of keyboard grabs (e.g. the
				// WM's own shortcuts) and focus moving into our own subwindows
				if (focusEvent.Mode == xNotifyNormal || focusEvent.Mode == xNotifyWhileGrabbed) &&
					focusEvent.Detail != xNotifyInferior && c.onDeactivate != nil {
					c.onDeactivate()
				}

			case xEnterNotify:
				if c.onMouseEnter != nil {
					c.onMouseEnter()
				}

			case xLeaveNotify:
				if c.onMouseLeave != nil {
					c.onMouseLeave()
				}

			case xMotionNotify:
				motionEvent := (*xMotionEvent)(unsafe.Pointer(&event))
				if c.onMouseMove != nil {
					c.onMouseMove(int(motionEvent.X), int(motionEvent.Y))
				}

			case xButtonPress:
				buttonEvent := (*xButtonEvent)(unsafe.Pointer(&event))

				x := int(buttonEvent.X)
				y := int(buttonEvent.Y)

				switch buttonEvent.Button {
				case 1:
					if c.onMouseButtonDown != nil {
						c.onMouseButtonDown(MouseButtonLeft, x, y)
					}
				case 2:
					if c.onMouseButtonDown != nil {
						c.onMouseButtonDown(MouseButtonMiddle, x, y)
					}
				case 3:
					if c.onMouseButtonDown != nil {
						c.onMouseButtonDown(MouseButtonRight, x, y)
					}
				case 4:
					if c.onMouseWheel != nil {
						c.onMouseWheel(0, 1)
					}
				case 5:
					if c.onMouseWheel != nil {
						c.onMouseWheel(0, -1)
					}
				case 6:
					if c.onMouseWheel != nil {
						c.onMouseWheel(1, 0)
					}
				case 7:
					if c.onMouseWheel != nil {
						c.onMouseWheel(-1, 0)
					}
				}

				dblClickDetected := false
				// Double click detection
				if buttonEvent.Button == 1 || buttonEvent.Button == 2 || buttonEvent.Button == 3 {
					if c.lastMouseButton == MouseButton(buttonEvent.Button) {
						timeSinceLastClick := time.Since(c.lastMouseDownTime)
						distanceX := int(buttonEvent.X) - c.lastMouseDownX
						distanceY := int(buttonEvent.Y) - c.lastMouseDownY
						distanceSquared := distanceX*distanceX + distanceY*distanceY
						if timeSinceLastClick < 500*time.Millisecond && distanceSquared < 25 {
							// Detected double click
							if c.onMouseButtonDblClick != nil {
								var btn MouseButton
								switch buttonEvent.Button {
								case 1:
									btn = MouseButtonLeft
								case 2:
									btn = MouseButtonMiddle
								case 3:
									btn = MouseButtonRight
								}
								c.onMouseButtonDblClick(btn, x, y)
							}
							dblClickDetected = true
						}
					}
				}

				if !dblClickDetected {
					// Update last mouse down info
					c.lastMouseDownX = int(buttonEvent.X)
					c.lastMouseDownY = int(buttonEvent.Y)
					c.lastMouseButton = MouseButton(buttonEvent.Button)
					c.lastMouseDownTime = time.Now()
				} else {
					// Reset last mouse down info to avoid triple click detection
					c.lastMouseDownX = 0
					c.lastMouseDownY = 0
					c.lastMouseButton = MouseButton(0)
					c.lastMouseDownTime = time.Time{}
				}

			case xButtonRelease:
				buttonEvent := (*xButtonEvent)(unsafe.Pointer(&event))

				x := int(buttonEvent.X)
				y := int(buttonEvent.Y)

				switch buttonEvent.Button {
				case 1:
					if c.onMouseButtonUp != nil {
						c.onMouseButtonUp(MouseButtonLeft, x, y)
					}
				case 2:
					if c.onMouseButtonUp != nil {
						c.onMouseButtonUp(MouseButtonMiddle, x, y)
					}
				case 3:
					if c.onMouseButtonUp != nil {
						c.onMouseButtonUp(MouseButtonRight, x, y)
					}
				}

			case xClientMessage:
				xclient := (*xClientMessageEvent)(unsafe.Pointer(&event))
				data0 := xclient.dataLong(0)

				if xclient.MessageType == c.platform.wmProtocols &&
					uintptr(data0) == c.platform.wmDeleteWindow {

					if c.inputBlocked() {
						// A modal dialog owned by this window is open - refuse
						// the close request outright, same as native modal
						// dialogs do, without even asking onCloseRequest.
						break
					}

					allowClose := true
					if c.onCloseRequest != nil {
						allowClose = c.onCloseRequest()
					}

					if allowClose {
						// Close (not a bare XDestroyWindow) so the request is
						// actually flushed before this event loop stops pumping.
						c.Close()
					}
				}

			case xPropertyNotify:
				propEvent := (*xPropertyEvent)(unsafe.Pointer(&event))
				if propEvent.Atom == c.platform.netWMState && !c.platform.allowMaximize && c.IsMaximized() {
					// Same idea as the UnmapNotify case above: the WM just
					// maximized us (button, window menu, double-click on the
					// titlebar, drag-to-edge, ...) despite
					// SetAllowMaximize(false), so ask it to un-maximize
					// again right away rather than relying on it to have
					// refused the click in the first place.
					restoreWindowX(c.platform.display, c.platform.window)
				}
			}

		}

		if c.platform.closed {
			break
		}

		// All the queued events are handled: lay out for the latest size once
		if c.platform.pendingResize {
			c.platform.pendingResize = false
			if c.windowWidth != c.platform.pendingWidth || c.windowHeight != c.platform.pendingHeight {
				c.windowWidth = c.platform.pendingWidth
				c.windowHeight = c.platform.pendingHeight
				if c.onResize != nil {
					c.onResize(c.windowWidth, c.windowHeight)
				}
			}
		}

		if now := time.Now(); !now.Before(nextTick) {
			nextTick = now.Add(tickInterval)
			if c.onTimer != nil {
				c.onTimer()
			}
		}

		if c.platform.needPaint.Swap(false) {
			c.paint()
		}

		// Sleep until an event comes or the next tick, instead of spinning
		if xPending(c.platform.display) == 0 {
			waitForFd(int(xConnectionNumber(c.platform.display)), time.Until(nextTick))
		}
	}
}

// paint draws the whole window
func (c *nativeWindow) paint() {
	dtBeginPaint := time.Now()
	hdcWidth, hdcHeight := c.windowWidth, c.windowHeight
	if hdcWidth > maxCanvasWidth {
		hdcWidth = maxCanvasWidth
	}
	if hdcHeight > maxCanvasHeight {
		hdcHeight = maxCanvasHeight
	}
	if hdcWidth <= 0 || hdcHeight <= 0 {
		return
	}

	buf := c.ensureCanvasBuffer(hdcWidth * hdcHeight * 4)
	fillCanvasBuffer(buf, c.platform.bgColor)

	img := &image.RGBA{
		Pix:    buf,
		Stride: hdcWidth * 4,
		Rect:   image.Rect(0, 0, hdcWidth, hdcHeight),
	}

	if c.onPaint != nil {
		c.onPaint(img)
	}

	putImageRGBA(c.platform.display, c.platform.window, img, hdcWidth, hdcHeight)

	c.drawTimes[c.drawTimesIndex] = time.Since(dtBeginPaint).Microseconds()
	c.drawTimesIndex++
	if c.drawTimesIndex >= len(c.drawTimes) {
		c.drawTimesIndex = 0
	}
}

// waitForFd waits until fd has data to read or the timeout passes
func waitForFd(fd int, timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	var readFds syscall.FdSet
	// The words of the set are 32 or 64 bits, depending on the architecture
	bitsPerWord := int(unsafe.Sizeof(readFds.Bits[0])) * 8
	readFds.Bits[fd/bitsPerWord] |= 1 << (fd % bitsPerWord)
	tv := syscall.NsecToTimeval(timeout.Nanoseconds())
	// EINTR (e.g. the Go runtime's preemption signals) only ends the wait early
	_, _ = syscall.Select(fd+1, &readFds, nil, nil, &tv)
}

// Close requests the window to close and is safe to call from any goroutine
// (e.g. a parent window closing a dialog it owns). The actual XDestroyWindow/
// XCloseDisplay teardown always runs on the window's own Exec goroutine
// (see doClose), since closing the Display while that goroutine might still
// be mid-call on it (XPending/XNextEvent) would be a use-after-free.
//
// A window that still has a modal dialog open on top of it (inputBlocked)
// is closed as soon as that dialog closes: tearing it down earlier would
// leave the dialog's modalParent dangling. This is the usual case of a dialog
// result handler closing the parent while the dialog itself is still closing.
// (The titlebar close button is refused outright in that state, see the
// WM_DELETE_WINDOW handler in pumpEvents.)
func (c *nativeWindow) Close() bool {
	atomic.StoreInt32(&c.platform.closeRequested, 1)
	return true
}

// doClose performs the actual Xlib teardown. Must only be called from the
// window's own Exec goroutine.
func (c *nativeWindow) doClose() {
	// Popups live on this window's Display, which is about to close
	closePopupsOf(c)
	xDestroyWindow(c.platform.display, c.platform.window)
	xCloseDisplay(c.platform.display)
	c.platform.closed = true

	if c.platform.modalParent != nil {
		atomic.AddInt32(&c.platform.modalParent.platform.modalChildCount, -1)
		c.platform.modalParent = nil
	}

	hwndsMu.Lock()
	delete(hwnds, windowId(c.platform.window))
	hwndsMu.Unlock()
}

func (c *nativeWindow) SetTitle(title string) {
	xStoreName(c.platform.display, c.platform.window, title)

	// XStoreName sets WM_NAME as a Latin-1 STRING property, which garbles any
	// non-ASCII title. Also set _NET_WM_NAME as UTF8_STRING so EWMH-compliant
	// window managers and desktop environments display Unicode titles correctly.
	utf8StringAtom := xInternAtom(c.platform.display, "UTF8_STRING", xFalse)
	netWmNameAtom := xInternAtom(c.platform.display, "_NET_WM_NAME", xFalse)

	titleBytes := []byte(title)
	var dataPtr unsafe.Pointer
	if len(titleBytes) > 0 {
		dataPtr = unsafe.Pointer(&titleBytes[0])
	}

	xChangeProperty(
		c.platform.display,
		c.platform.window,
		netWmNameAtom,
		utf8StringAtom,
		8,
		xPropModeReplace,
		dataPtr,
		int32(len(titleBytes)),
	)
}

func (c *nativeWindow) Move(x, y int) {
	c.platform.prevSetPosX = x
	c.platform.prevSetPosY = y
	left, _, top, _, ok := c.getFrameExtents()
	if ok {
		x -= left
		y -= top
	}

	xMoveWindow(c.platform.display, c.platform.window, int32(x), int32(y))
}

// MoveToCenterOfScreen centers the window on whichever monitor currently
// holds the largest portion of it, not on the combined virtual desktop
// spanning every monitor (nor always the primary one) - so on a multi-
// monitor setup it lands in the middle of the screen it's actually on.
func (c *nativeWindow) MoveToCenterOfScreen() {
	monX, monY, monWidth, monHeight := monitorRectForWindow(c.platform.display, c.platform.screen, c.windowPosX, c.windowPosY, c.windowWidth, c.windowHeight)
	windowWidth, windowHeight := c.Size()
	x := monX + (monWidth-windowWidth)/2
	y := monY + (monHeight-windowHeight)/2
	c.Move(x, y)
}

func (c *nativeWindow) Resize(width, height int) {
	xResizeWindow(c.platform.display, c.platform.window, uint32(width), uint32(height))
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

func (c *nativeWindow) IsMaximized() bool {
	display := c.platform.display
	window := c.platform.window

	var actualType uintptr
	var actualFormat int32
	var nitems uintptr
	var bytesAfter uintptr
	var prop unsafe.Pointer

	if c.platform.closed {
		return false
	}

	status := xGetWindowProperty(
		display,
		window,
		c.platform.netWMState,
		0,
		1024,
		xFalse,
		xXAAtom,
		unsafe.Pointer(&actualType),
		unsafe.Pointer(&actualFormat),
		unsafe.Pointer(&nitems),
		unsafe.Pointer(&bytesAfter),
		unsafe.Pointer(&prop),
	)

	if status != xSuccess || prop == nil {
		return false
	}
	defer xFree(prop)

	if actualType != xXAAtom || actualFormat != 32 || nitems == 0 {
		return false
	}

	atoms := unsafe.Slice((*uintptr)(prop), int(nitems))

	hasHorz := false
	hasVert := false

	for _, atom := range atoms {
		if atom == c.platform.netWMStateMaximizedHorz {
			hasHorz = true
		}
		if atom == c.platform.netWMStateMaximizedVert {
			hasVert = true
		}
	}

	return hasHorz && hasVert
}

func (c *nativeWindow) KeyModifiers() KeyModifiers {
	return c.getModifierState()
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
	c.platform.bgColor = color
	xSetWindowBackground(c.platform.display, c.platform.window, backgroundPixel(color))
	c.Update()
}

// backgroundPixel is the color as a pixel of the default 24-bit TrueColor visual
func backgroundPixel(col color.RGBA) uintptr {
	return uintptr(col.R)<<16 | uintptr(col.G)<<8 | uintptr(col.B)
}

func (c *nativeWindow) SetMouseCursor(cursor MouseCursor) {
	if c.currentCursor == cursor {
		return
	}
	c.currentCursor = cursor
	c.changeMouseCursor(cursor)
}

func (c *nativeWindow) changeMouseCursor(mouseCursor MouseCursor) bool {
	cursor := xCreateFontCursor(c.platform.display, xCursorShape(mouseCursor))
	xDefineCursor(c.platform.display, c.platform.window, cursor)
	xFlush(c.platform.display)
	return true
}

// xCursorShape returns the X cursor font shape for the cursor kind.
func xCursorShape(mouseCursor MouseCursor) uint32 {
	var cursorShape uint32

	const (
		CursorArrow = 132
		CursorCross = 34
		CursorWait  = 150
		CursorIBeam = 152
		CursorHand  = 60
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
	return cursorShape
}

func (c *nativeWindow) MinimizeWindow() {
	minimizeWindowX(c.platform.display, c.platform.window)
}

func (c *nativeWindow) MaximizeWindow() {
	maximizeWindowX(c.platform.display, c.platform.window)
}

func (c *nativeWindow) RestoreWindow() {
	restoreWindowX(c.platform.display, c.platform.window)
}

// SetDarkMode sets _GTK_THEME_VARIANT, which GNOME and KDE use to draw the
// window decorations dark or light
func (c *nativeWindow) SetDarkMode(dark bool) {
	variant := []byte("light")
	if dark {
		variant = []byte("dark")
	}
	utf8StringAtom := xInternAtom(c.platform.display, "UTF8_STRING", xFalse)
	variantAtom := xInternAtom(c.platform.display, "_GTK_THEME_VARIANT", xFalse)
	xChangeProperty(c.platform.display, c.platform.window, variantAtom, utf8StringAtom, 8, xPropModeReplace, unsafe.Pointer(&variant[0]), int32(len(variant)))
	xFlush(c.platform.display)
}

func (c *nativeWindow) SetAlwaysOnTop(onTop bool) {
	setNetWMState(c.platform.display, c.platform.window, onTop, "_NET_WM_STATE_ABOVE")
}

// RequestAttention sets _NET_WM_STATE_DEMANDS_ATTENTION; the window manager
// clears it when the window gets focus
func (c *nativeWindow) RequestAttention() {
	setNetWMState(c.platform.display, c.platform.window, true, "_NET_WM_STATE_DEMANDS_ATTENTION")
}

// Beep plays the bell of the desktop sound theme. The X11 bell (XBell) is silent
// on most systems today - it needs a PC speaker or a sound server module that
// is usually not loaded - so it is only the fallback when no sound can be played.
func (c *nativeWindow) Beep() {
	if player, args, ok := bellSoundCommand(); ok {
		go exec.Command(player, args...).Run()
		return
	}
	xBell(c.platform.display, 0)
	xFlush(c.platform.display)
}

const bellSoundFile = "/usr/share/sounds/freedesktop/stereo/bell.oga"

// bellSoundCommand finds a way to play the bell sound: canberra-gtk-play uses the
// desktop sound theme (and its mute setting), pw-play and paplay play the file
func bellSoundCommand() (player string, args []string, ok bool) {
	if p, err := exec.LookPath("canberra-gtk-play"); err == nil {
		return p, []string{"--id=bell"}, true
	}
	if _, err := os.Stat(bellSoundFile); err != nil {
		return "", nil, false
	}
	for _, name := range []string{"pw-play", "paplay"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, []string{bellSoundFile}, true
		}
	}
	return "", nil, false
}

// SetAllowMinimize shows or hides the titlebar's minimize button via
// _MOTIF_WM_HINTS, e.g. for dialog-style windows that shouldn't offer it.
func (c *nativeWindow) SetAllowMinimize(allow bool) {
	c.platform.allowMinimize = allow
	c.applyWindowDecorations()
}

// SetAllowMaximize shows or hides the titlebar's maximize button via
// _MOTIF_WM_HINTS, e.g. for dialog-style windows that shouldn't offer it.
func (c *nativeWindow) SetAllowMaximize(allow bool) {
	c.platform.allowMaximize = allow
	c.applyWindowDecorations()
}

func (c *nativeWindow) applyWindowDecorations() {
	setWindowDecorationsX(c.platform.display, c.platform.window, c.platform.allowMinimize, c.platform.allowMaximize)
}

// ShowModal marks the window as a modal dialog owned by parent (WM_TRANSIENT_FOR +
// _NET_WM_STATE_MODAL) and shows it. Show() already runs the event pump on its
// own goroutine, so parent's event loop/timer keep running. The WM (verified
// against kwin_x11/KDE) only honors this for keyboard focus, not pointer
// input, so parent's own pump loop also drops keyboard/mouse callbacks for
// as long as this dialog - tracked via parent's modalChildCount - stays open.
func (c *nativeWindow) ShowModal(parent Window) {
	// Map first: setWindowModal's _NET_WM_STATE_MODAL request is a
	// ClientMessage sent to root, which a WM only honors for a window it
	// already manages (i.e. already mapped). Sending it before Show() maps
	// the window means the WM silently drops it - the window still opens,
	// just non-modal, since it's asking about a window it doesn't know yet.
	c.Show()
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		c.platform.modalParent = p
		atomic.AddInt32(&p.platform.modalChildCount, 1)
		setWindowModalX(c.platform.display, c.platform.window, p.platform.window)
	}
}

// inputBlocked reports whether c should drop keyboard/mouse callbacks
// because a modal dialog it owns is currently open (see modalChildCount).
func (c *nativeWindow) inputBlocked() bool {
	return atomic.LoadInt32(&c.platform.modalChildCount) > 0
}

func (c *nativeWindow) SetAppIcon(icon *image.RGBA) {
	width := icon.Bounds().Dx()
	height := icon.Bounds().Dy()

	// _NET_WM_ICON: [width, height, pixels...]
	dataLen := 2 + width*height
	data := make([]uintptr, dataLen)
	data[0] = uintptr(width)
	data[1] = uintptr(height)

	i := 2
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := icon.PixOffset(x, y)
			r := icon.Pix[offset]
			g := icon.Pix[offset+1]
			b := icon.Pix[offset+2]
			a := icon.Pix[offset+3]

			argb := (uint32(a) << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
			data[i] = uintptr(argb)
			i++
		}
	}

	atom := xInternAtom(c.platform.display, "_NET_WM_ICON", xFalse)

	xChangeProperty(
		c.platform.display,
		c.platform.window,
		atom,
		xXACardinal,
		32,
		xPropModeReplace,
		unsafe.Pointer(&data[0]),
		int32(len(data)),
	)
}

func (c *nativeWindow) drawImageRGBA(display uintptr, window uintptr, img image.Image) {
	putImageRGBA(display, window, img.(*image.RGBA), c.windowWidth, c.windowHeight)
}

// putImageRGBA copies the top-left width x height pixels of img to window.
func putImageRGBA(display uintptr, window uintptr, img *image.RGBA, width, height int) {
	dataSize := width * height * 4

	// XDestroyImage (via destroyXImage below) frees this buffer through
	// libX11's own free(), so it has to come from the same libc malloc, not
	// Go's allocator.
	cBuffer := libcMalloc(uintptr(dataSize))

	// Copied converting RGBA to the BGRA of the X image in one pass, a
	// pixel at a time as a little-endian uint32 (all the Linux targets are)
	dst := unsafe.Slice((*uint32)(cBuffer), width*height)
	for y := 0; y < height; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+width*4]
		src := unsafe.Slice((*uint32)(unsafe.Pointer(&row[0])), width)
		out := dst[y*width : (y+1)*width]
		for i, v := range src {
			out[i] = v&0xFF00FF00 | (v&0xFF)<<16 | (v>>16)&0xFF
		}
	}

	ximage := xCreateImage(
		display,
		xDefaultVisual(display, xDefaultScreen(display)),
		24,
		xZPixmap,
		0,
		cBuffer,
		uint32(width),
		uint32(height),
		32,
		0,
	)

	gc := xCreateGC(display, window, 0, 0)
	defer xFreeGC(display, gc)

	xPutImage(display, window, gc, ximage, 0, 0, 0, 0, uint32(width), uint32(height))

	destroyXImage(ximage)
}

func (c *nativeWindow) SystemHandle() any {
	return nil
}

func (c *nativeWindow) getModifierState() KeyModifiers {
	display := c.platform.display
	window := c.platform.window

	var rootRet, childRet uintptr
	var rootX, rootY, winX, winY int32
	var mask uint32

	xQueryPointer(
		display,
		window,
		unsafe.Pointer(&rootRet),
		unsafe.Pointer(&childRet),
		unsafe.Pointer(&rootX), unsafe.Pointer(&rootY),
		unsafe.Pointer(&winX), unsafe.Pointer(&winY),
		unsafe.Pointer(&mask),
	)

	return KeyModifiers{
		Shift: (mask & xShiftMask) != 0,
		Ctrl:  (mask & xControlMask) != 0,
		Alt:   (mask & xMod1Mask) != 0,
	}
}

func (c *nativeWindow) getFrameExtents() (left, right, top, bottom int, ok bool) {
	display := c.platform.display
	window := c.platform.window

	atom := xInternAtom(display, "_NET_FRAME_EXTENTS", xFalse)

	var actualType uintptr
	var actualFormat int32
	var nitems uintptr
	var bytesAfter uintptr
	var prop unsafe.Pointer

	if c.platform.closed {
		return 0, 0, 0, 0, false
	}

	status := xGetWindowProperty(
		display,
		window,
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

	data := (*[4]uintptr)(prop)

	left = int(data[0])
	right = int(data[1])
	top = int(data[2])
	bottom = int(data[3])

	return left, right, top, bottom, true
}

func (c *nativeWindow) updateWindowPos() {
	display := c.platform.display
	window := c.platform.window
	root := xRootWindow(display, c.platform.screen)

	var x, y int32
	var child uintptr

	if xTranslateCoordinates(
		display,
		window,
		root,
		0, 0,
		unsafe.Pointer(&x), unsafe.Pointer(&y),
		unsafe.Pointer(&child),
	) == 0 {
		return
	}

	c.windowPosX = int(x)
	c.windowPosY = int(y)
}

func (c *nativeWindow) ClientToScreen(x, y int) (int, int) {
	display := c.platform.display
	root := xRootWindow(display, c.platform.screen)

	var sx, sy int32
	var child uintptr

	if xTranslateCoordinates(
		display,
		c.platform.window,
		root,
		int32(x), int32(y),
		unsafe.Pointer(&sx), unsafe.Pointer(&sy),
		unsafe.Pointer(&child),
	) == 0 {
		return c.windowPosX + x, c.windowPosY + y
	}
	return int(sx), int(sy)
}

// ScreenWorkArea intersects the monitor with _NET_WORKAREA. The latter is a
// single rect over the whole virtual desktop, so this cuts off panels along
// the outer edges of the desktop, but not ones between monitors.
func (c *nativeWindow) ScreenWorkArea(x, y int) (int, int, int, int) {
	mx, my, mw, mh := monitorRectForWindow(c.platform.display, c.platform.screen, x, y, 1, 1)
	wx, wy, ww, wh, ok := netWorkAreaX(c.platform.display, c.platform.screen)
	if !ok {
		return mx, my, mw, mh
	}
	left, top := max(mx, wx), max(my, wy)
	right, bottom := min(mx+mw, wx+ww), min(my+mh, wy+wh)
	if right <= left || bottom <= top {
		return mx, my, mw, mh
	}
	return left, top, right - left, bottom - top
}
