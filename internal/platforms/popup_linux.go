package platforms

import (
	"image"
	"image/color"
	"sync"
	"unsafe"
)

const (
	xShapeInput    = 2
	xShapeSet      = 0
	xShapeUnsorted = 0
)

// popupWindow is an override-redirect X11 window: the window manager doesn't
// decorate, place or focus it. It lives on the shared Display, so the event
// loop delivers its events (see processEvent) like the ones of the windows.
type popupWindow struct {
	popupCallbacks

	owner       *nativeWindow
	display     uintptr
	window      uintptr
	interactive bool

	// Guarded by popupsMu
	closed        bool
	width, height int

	canvasBuffer []byte
}

var (
	popupsMu sync.Mutex
	popups   = make(map[uintptr]*popupWindow)
)

func createPopupWindow(owner Window, interactive bool) PopupWindow {
	o, ok := owner.(*nativeWindow)
	if !ok || o == nil || o.platform.closed {
		return nil
	}
	display := o.platform.display

	attrs := xSetWindowAttributes{}
	attrs.BackgroundPixmap = xNone
	attrs.OverrideRedirect = xTrue
	attrs.SaveUnder = xTrue

	window := xCreateWindow(
		display,
		xRootWindow(display, o.platform.screen),
		0, 0, 1, 1,
		0,
		xCopyFromParent,
		xInputOutput,
		0,
		xCWBackPixmap|xCWOverrideRedirect|xCWSaveUnder,
		unsafe.Pointer(&attrs),
	)
	if window == 0 {
		return nil
	}

	windowType := "_NET_WM_WINDOW_TYPE_TOOLTIP"
	if interactive {
		windowType = "_NET_WM_WINDOW_TYPE_POPUP_MENU"
		xSelectInput(display, window, xExposureMask|xButtonPressMask|xButtonReleaseMask|xPointerMotionMask|xLeaveWindowMask)
	} else {
		xSelectInput(display, window, xExposureMask)
		// Empty input shape: the mouse passes through to the window below,
		// so the owner keeps its hover state while the cursor is over the popup
		if xShapeAvailable {
			xShapeCombineRectangles(display, window, xShapeInput, 0, 0, nil, 0, xShapeSet, xShapeUnsorted)
		}
	}

	// Hints for compositors (shadows, animations) and stacking
	wmWindowType := xInternAtom(display, "_NET_WM_WINDOW_TYPE", xFalse)
	wmWindowTypeValue := xInternAtom(display, windowType, xFalse)
	xChangeProperty(display, window, wmWindowType, xXAAtom, 32, xPropModeReplace, unsafe.Pointer(&wmWindowTypeValue), 1)
	xSetTransientForHint(display, window, o.platform.window)

	xFlush(display)

	p := &popupWindow{owner: o, display: display, window: window, interactive: interactive}
	popupsMu.Lock()
	popups[window] = p
	popupsMu.Unlock()
	return p
}

func getPopupByWindow(window uintptr) *popupWindow {
	popupsMu.Lock()
	defer popupsMu.Unlock()
	return popups[window]
}

// closePopupsOf destroys the owner's popups right before the owner is closed
func closePopupsOf(owner *nativeWindow) {
	popupsMu.Lock()
	defer popupsMu.Unlock()
	for window, p := range popups {
		if p.owner == owner {
			p.closed = true
			delete(popups, window)
			xDestroyWindow(p.display, window)
		}
	}
}

// processEvent runs on the UI thread, from the event loop.
func (p *popupWindow) processEvent(event *xEvent) {
	switch event.eventType() {
	case xExpose:
		p.paint()
	case xMotionNotify:
		motionEvent := (*xMotionEvent)(unsafe.Pointer(event))
		p.mouseMove(int(motionEvent.X), int(motionEvent.Y))
	case xButtonPress, xButtonRelease:
		buttonEvent := (*xButtonEvent)(unsafe.Pointer(event))
		var btn MouseButton
		switch buttonEvent.Button {
		case 1:
			btn = MouseButtonLeft
		case 2:
			btn = MouseButtonMiddle
		case 3:
			btn = MouseButtonRight
		default:
			// Wheel: buttons 4-7 press and release at once, one step each
			if event.eventType() == xButtonPress {
				switch buttonEvent.Button {
				case 4:
					p.mouseWheel(0, 1)
				case 5:
					p.mouseWheel(0, -1)
				case 6:
					p.mouseWheel(1, 0)
				case 7:
					p.mouseWheel(-1, 0)
				}
			}
			return
		}
		if event.eventType() == xButtonPress {
			p.mouseButtonDown(btn, int(buttonEvent.X), int(buttonEvent.Y))
		} else {
			p.mouseButtonUp(btn, int(buttonEvent.X), int(buttonEvent.Y))
		}
	case xLeaveNotify:
		p.mouseLeave()
	}
}

func (p *popupWindow) paint() {

	popupsMu.Lock()
	closed, width, height := p.closed, p.width, p.height
	popupsMu.Unlock()
	if closed || width <= 0 || height <= 0 {
		return
	}
	width = min(width, maxCanvasWidth)
	height = min(height, maxCanvasHeight)

	buf := growBuffer(&p.canvasBuffer, width*height*4)
	fillCanvasBuffer(buf, color.RGBA{0, 0, 0, 255})
	img := &image.RGBA{
		Pix:    buf,
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}

	if p.onPaint != nil {
		p.onPaint(img)
	}

	putImageRGBA(p.display, p.window, img, width, height)
}

func (p *popupWindow) ShowAt(x, y, width, height int) {
	popupsMu.Lock()
	defer popupsMu.Unlock()
	if p.closed || width <= 0 || height <= 0 {
		return
	}
	p.width, p.height = width, height

	xMoveWindow(p.display, p.window, int32(x), int32(y))
	xResizeWindow(p.display, p.window, uint32(width), uint32(height))
	xMapRaised(p.display, p.window)
	// Already mapped with the same size: no Expose would come on its own
	xClearArea(p.display, p.window, 0, 0, 0, 0, xTrue)
	xFlush(p.display)
}

func (p *popupWindow) Hide() {
	popupsMu.Lock()
	defer popupsMu.Unlock()
	if p.closed {
		return
	}
	xUnmapWindow(p.display, p.window)
	xFlush(p.display)
}

func (p *popupWindow) SetMouseCursor(cursor MouseCursor) {
	popupsMu.Lock()
	defer popupsMu.Unlock()
	if p.closed {
		return
	}
	xDefineCursor(p.display, p.window, xCreateFontCursor(p.display, xCursorShape(cursor)))
	xFlush(p.display)
}

func (p *popupWindow) Update() {
	popupsMu.Lock()
	defer popupsMu.Unlock()
	if p.closed {
		return
	}
	xClearArea(p.display, p.window, 0, 0, 0, 0, xTrue)
	xFlush(p.display)
}

func (p *popupWindow) Close() {
	popupsMu.Lock()
	defer popupsMu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	delete(popups, p.window)
	xDestroyWindow(p.display, p.window)
	xFlush(p.display)
}
