package platforms

import (
	"image"
	"image/color"
	"sync"
	"syscall"
	"unsafe"
)

var (
	procClientToScreen  = user32.NewProc("ClientToScreen")
	procGetClientRect   = user32.NewProc("GetClientRect")
	procMonitorFromRect = user32.NewProc("MonitorFromRect")
	procGetMonitorInfoW = user32.NewProc("GetMonitorInfoW")
)

const (
	c_WS_POPUP = 0x80000000

	c_WS_EX_TOPMOST    = 0x00000008
	c_WS_EX_TOOLWINDOW = 0x00000080
	c_WS_EX_NOACTIVATE = 0x08000000

	c_CS_DROPSHADOW = 0x00020000

	c_SWP_SHOWWINDOW = 0x0040
	c_SWP_HIDEWINDOW = 0x0080

	c_WM_ERASEBKGND    = 0x0014
	c_WM_NCHITTEST     = 0x0084
	c_WM_MOUSEACTIVATE = 0x0021
	c_WM_APP           = 0x8000

	c_WM_NUI_CREATE_POPUP = c_WM_APP + 1

	c_HTTRANSPARENT            = ^uintptr(0) // -1
	c_MA_NOACTIVATE            = 3
	c_WM_SETCURSOR             = 0x0020
	c_HTCLIENT                 = 1
	c_WM_ACTIVATE              = 0x0006
	c_WA_INACTIVE              = 0
	c_HWND_TOPMOST             = ^uintptr(0) // -1
	c_MONITOR_DEFAULTTONEAREST = 2
)

type t_MONITORINFO struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

type popupWindow struct {
	popupCallbacks

	hwnd        syscall.Handle
	owner       *nativeWindow
	interactive bool
	mouseInside bool    // owner's thread only
	hCursor     uintptr // the cursor set by SetMouseCursor, 0 for the class cursor

	canvasBuffer []byte
	pixBuffer    []byte
}

var (
	popupsMu sync.Mutex
	popups   = make(map[syscall.Handle]*popupWindow)

	popupClassOnce sync.Once
	popupClassName *uint16
)

// createPopupWindow asks the owner's thread to create the popup HWND, so the
// popup shares the owner's message loop: no extra thread, and HTTRANSPARENT
// (mouse pass-through) only works between windows of the same thread.
func createPopupWindow(owner Window, interactive bool) PopupWindow {
	o, ok := owner.(*nativeWindow)
	if !ok || o == nil {
		return nil
	}
	popupClassOnce.Do(registerPopupClass)

	hwnd, _, _ := procSendMessageW.Call(uintptr(o.hwnd), c_WM_NUI_CREATE_POPUP, 0, 0)
	if hwnd == 0 {
		return nil
	}

	p := &popupWindow{hwnd: syscall.Handle(hwnd), owner: o, interactive: interactive}
	popupsMu.Lock()
	popups[p.hwnd] = p
	popupsMu.Unlock()
	return p
}

func registerPopupClass() {
	popupClassName, _ = syscall.UTF16PtrFromString("NUIPopupWindow")
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	arrow, _, _ := procLoadCursorW.Call(0, c_IDC_ARROW)
	wndClass := t_WNDCLASSEXW{
		cbSize:        uint32(unsafe.Sizeof(t_WNDCLASSEXW{})),
		style:         c_CS_DROPSHADOW,
		lpfnWndProc:   syscall.NewCallback(popupWndProc),
		hInstance:     syscall.Handle(hInstance),
		hCursor:       syscall.Handle(arrow),
		lpszClassName: popupClassName,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))
}

// createPopupHwnd runs on the owner's thread (from its wndProc). The popup
// is created hidden, so no WM_PAINT arrives before createPopupWindow
// registers it.
func createPopupHwnd(owner syscall.Handle) uintptr {
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, _ := procCreateWindowExW.Call(
		c_WS_EX_TOOLWINDOW|c_WS_EX_TOPMOST|c_WS_EX_NOACTIVATE,
		uintptr(unsafe.Pointer(popupClassName)),
		0,
		c_WS_POPUP,
		0, 0, 1, 1,
		uintptr(owner),
		0,
		hInstance,
		0,
	)
	return hwnd
}

func getPopupByHandle(hwnd syscall.Handle) *popupWindow {
	popupsMu.Lock()
	defer popupsMu.Unlock()
	return popups[hwnd]
}

func popupWndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	p := getPopupByHandle(hwnd)
	x := int(int16(lParam & 0xFFFF))
	y := int(int16((lParam >> 16) & 0xFFFF))
	if p != nil {
		// The mouse position in logical pixels
		x, y = scaleToLogical(x, p.Scale()), scaleToLogical(y, p.Scale())
	}

	switch msg {
	case c_WM_NCHITTEST:
		if p == nil || !p.interactive {
			return c_HTTRANSPARENT
		}
		return c_HTCLIENT

	case c_WM_SETCURSOR:
		// The class cursor (an arrow) unless SetMouseCursor chose another
		if p != nil && p.hCursor != 0 && lParam&0xFFFF == c_HTCLIENT {
			procSetCursor.Call(p.hCursor)
			return 1
		}

	case c_WM_MOUSEMOVE:
		if p != nil {
			if !p.mouseInside {
				p.mouseInside = true
				tme := t_TRACKMOUSEEVENT{
					cbSize:    uint32(unsafe.Sizeof(t_TRACKMOUSEEVENT{})),
					dwFlags:   c_TME_LEAVE,
					hwndTrack: hwnd,
				}
				procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			}
			p.mouseMove(x, y)
		}
		return 0

	case c_WM_MOUSELEAVE:
		if p != nil {
			p.mouseInside = false
			p.mouseLeave()
		}
		return 0

	case c_WM_MOUSEWHEEL:
		// Usually goes to the focused window (the owner) instead, depending
		// on the "scroll inactive windows" setting
		if p != nil {
			p.mouseWheel(0, int(int16((wParam>>16)&0xFFFF)/120))
		}
		return 0

	case c_WM_LBUTTONDOWN, c_WM_RBUTTONDOWN, c_WM_MBUTTONDOWN:
		if p != nil {
			p.mouseButtonDown(popupMouseButton(msg), x, y)
		}
		return 0

	case c_WM_LBUTTONUP, c_WM_RBUTTONUP, c_WM_MBUTTONUP:
		if p != nil {
			p.mouseButtonUp(popupMouseButton(msg), x, y)
		}
		return 0

	case c_WM_MOUSEACTIVATE:
		return c_MA_NOACTIVATE

	case c_WM_ERASEBKGND:
		return 1

	case c_WM_PAINT:
		var ps t_PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		if p != nil {
			p.paint(hdc)
		}
		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		return 0

	case c_WM_DESTROY:
		// Also sent when the owner is destroyed. No PostQuitMessage: the
		// message loop belongs to the owner.
		popupsMu.Lock()
		delete(popups, hwnd)
		popupsMu.Unlock()
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

// Scale is the owner's: the popup shows over it, and its size and position
// are converted to pixels with that scale too
func (p *popupWindow) Scale() float64 {
	return p.owner.Scale()
}

func (p *popupWindow) paint(hdc uintptr) {
	var r rect
	procGetClientRect.Call(uintptr(p.hwnd), uintptr(unsafe.Pointer(&r)))
	width, height := r.right-r.left, r.bottom-r.top
	if width <= 0 || height <= 0 {
		return
	}

	buf := growBuffer(&p.canvasBuffer, int(width*height*4))
	fillCanvasBuffer(buf, color.RGBA{0, 0, 0, 255})
	img := &image.RGBA{
		Pix:    buf,
		Stride: int(width) * 4,
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	if p.onPaint != nil {
		p.onPaint(img)
	}

	drawRGBAToHDC(img, hdc, width, height, &p.pixBuffer)
}

func popupMouseButton(msg uint32) MouseButton {
	switch msg {
	case c_WM_RBUTTONDOWN, c_WM_RBUTTONUP:
		return MouseButtonRight
	case c_WM_MBUTTONDOWN, c_WM_MBUTTONUP:
		return MouseButtonMiddle
	}
	return MouseButtonLeft
}

func (p *popupWindow) ShowAt(x, y, width, height int) {
	procSetWindowPos.Call(
		uintptr(p.hwnd),
		c_HWND_TOPMOST,
		uintptr(scaleToPhysical(x, p.Scale())), uintptr(scaleToPhysical(y, p.Scale())),
		uintptr(scaleToPhysical(width, p.Scale())), uintptr(scaleToPhysical(height, p.Scale())),
		c_SWP_NOACTIVATE|c_SWP_SHOWWINDOW,
	)
	procInvalidateRect.Call(uintptr(p.hwnd), 0, 0)
}

func (p *popupWindow) Hide() {
	procSetWindowPos.Call(
		uintptr(p.hwnd),
		0,
		0, 0, 0, 0,
		c_SWP_NOMOVE|c_SWP_NOSIZE|c_SWP_NOZORDER|c_SWP_NOACTIVATE|c_SWP_HIDEWINDOW,
	)
}

// SetMouseCursor is called on the owner's thread, which is also the popup's,
// so SetCursor can apply it right away when the mouse is over the popup.
func (p *popupWindow) SetMouseCursor(cursor MouseCursor) {
	p.hCursor = loadMouseCursor(cursor)
	if p.mouseInside && p.hCursor != 0 {
		procSetCursor.Call(p.hCursor)
	}
}

func (p *popupWindow) Update() {
	procInvalidateRect.Call(uintptr(p.hwnd), 0, 0)
}

// Close posts WM_CLOSE instead of calling DestroyWindow, which only works on
// the thread that created the window.
func (p *popupWindow) Close() {
	procPostMessageW.Call(uintptr(p.hwnd), c_WM_CLOSE, 0, 0)
}
