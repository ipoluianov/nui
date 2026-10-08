package platforms

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procShowWindow       = user32.NewProc("ShowWindow")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procDestroyWindow    = user32.NewProc("DestroyWindow")

	procSetCapture     = user32.NewProc("SetCapture")
	procReleaseCapture = user32.NewProc("ReleaseCapture")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")

	procBeginPaint        = user32.NewProc("BeginPaint")
	procEndPaint          = user32.NewProc("EndPaint")
	procTextOutW          = gdi32.NewProc("TextOutW")
	procSetDIBitsToDevice = gdi32.NewProc("SetDIBitsToDevice")

	procTrackMouseEvent = user32.NewProc("TrackMouseEvent")

	procInvalidateRect = user32.NewProc("InvalidateRect")

	procPostMessageW   = user32.NewProc("PostMessageW")
	procSetWindowTextW = user32.NewProc("SetWindowTextW")
	procSetWindowPos   = user32.NewProc("SetWindowPos")
	procFlashWindowEx  = user32.NewProc("FlashWindowEx")
	procMessageBeep    = user32.NewProc("MessageBeep")

	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetCurrentThreadId       = kernel32.NewProc("GetCurrentThreadId")

	procLoadCursorW = user32.NewProc("LoadCursorW")
	procSetCursor   = user32.NewProc("SetCursor")

	procSetTimer  = user32.NewProc("SetTimer")
	procKillTimer = user32.NewProc("KillTimer")

	procSendMessageW = user32.NewProc("SendMessageW")
	procCreateIcon   = user32.NewProc("CreateIcon")

	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")

	procIsZoomed            = user32.NewProc("IsZoomed")
	procGetKeyState         = user32.NewProc("GetKeyState")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")

	procEnableWindow        = user32.NewProc("EnableWindow")
	procGetWindowLongPtrW   = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW   = user32.NewProc("SetWindowLongPtrW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")

	modDwmapi                 = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute = modDwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	c_WS_OVERLAPPEDWINDOW = 0x00CF0000
	c_WS_MAXIMIZE         = 0x01000000
	c_WS_VISIBLE          = 0x10000000
	c_CW_USEDEFAULT       = 0x80000000

	c_SW_HIDE          = 0
	c_SW_SHOW          = 5
	c_SW_SHOWNORMAL    = 1
	c_SW_SHOWMINIMIZED = 2
	c_SW_SHOWMAXIMIZED = 3
	c_SW_RESTORE       = 9
	c_SW_SHOWDEFAULT   = 10

	c_SM_CXSCREEN = 0
	c_SM_CYSCREEN = 1

	c_SWP_NOSIZE       = 0x0001
	c_SWP_NOMOVE       = 0x0002
	c_SWP_NOZORDER     = 0x0004
	c_SWP_NOACTIVATE   = 0x0010
	c_SWP_FRAMECHANGED = 0x0020

	c_WS_MINIMIZEBOX = 0x00020000
	c_WS_MAXIMIZEBOX = 0x00010000

	c_WM_SETICON      = 0x0080
	c_ICON_SMALL      = 0
	c_ICON_BIG        = 1
	c_IMAGE_ICON      = 1
	c_LR_DEFAULTCOLOR = 0x0000

	c_IDC_ARROW  = uintptr(32512)
	c_IDC_HAND   = uintptr(32649)
	c_IDC_SIZEWE = uintptr(32644)
	c_IDC_SIZENS = uintptr(32645)
	c_IDC_IBEAM  = uintptr(32513)

	c_CS_DBLCLKS = 0x0008
	c_CS_OWNDC   = 0x0020

	c_WM_MOVE = 0x0003
	c_WM_SIZE = 0x0005

	c_WM_CLOSE   = 0x0010
	c_WM_DESTROY = 0x0002

	// c_WM_NUI_CLOSE asks the UI thread to destroy the window (see nativeWindow.Close)
	c_WM_NUI_CLOSE = c_WM_APP + 2

	c_WM_KEYDOWN = 0x0100
	c_WM_KEYUP   = 0x0101
	c_WM_CHAR    = 0x0102

	c_WM_SYSKEYDOWN = 0x0104
	c_WM_SYSKEYUP   = 0x0105
	c_WM_SYSCHAR    = 0x0106

	c_WM_LBUTTONDOWN = 0x0201
	c_WM_LBUTTONUP   = 0x0202
	c_WM_MOUSEMOVE   = 0x0200
	c_WM_RBUTTONDOWN = 0x0204
	c_WM_RBUTTONUP   = 0x0205
	c_WM_MBUTTONDOWN = 0x0207
	c_WM_MBUTTONUP   = 0x0208
	c_WM_MOUSEWHEEL  = 0x020A // Dec: 522
	c_WM_XBUTTONDOWN = 0x020B
	c_WM_XBUTTONUP   = 0x020C

	// dec 132 to hex is 0x84

	c_WM_LBUTTONDBLCLK = 0x0203
	c_WM_RBUTTONDBLCLK = 0x0206
	c_WM_MBUTTONDBLCLK = 0x0209

	c_WM_MOUSELEAVE = 0x02A3

	c_TME_LEAVE = 0x00000002

	c_WM_TIMER = 0x0113
	timerID1ms = 1 // any unique ID

	c_DWMWA_USE_IMMERSIVE_DARK_MODE_OLD = 19
	c_DWMWA_USE_IMMERSIVE_DARK_MODE     = 20
)

type t_WNDCLASSEXW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

type t_PAINTSTRUCT struct {
	hdc         syscall.Handle
	fErase      int32
	rcPaint     struct{ left, top, right, bottom int32 }
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type t_MSG struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

type t_BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type t_RGBQUAD struct {
	Blue     byte
	Green    byte
	Red      byte
	Reserved byte
}

type t_BITMAPINFO struct {
	Header t_BITMAPINFOHEADER
	Colors [3]t_RGBQUAD
}

type t_TRACKMOUSEEVENT struct {
	cbSize      uint32
	dwFlags     uint32
	hwndTrack   syscall.Handle
	dwHoverTime uint32
}

const (
	c_WM_PAINT = 0x000F
)

var (
	procGetClipBox = gdi32.NewProc("GetClipBox")
)

const (
	c_HORZRES   = 8
	c_VERTRES   = 10
	c_BITSPIXEL = 12
	c_PLANES    = 14

	c_OBJ_DC        = 1
	c_OBJ_MEMDC     = 10
	c_OBJ_ENHMETADC = 12
)

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

func getNativeWindowByHandle(hwnd windowId) *nativeWindow {
	appWindowsMu.Lock()
	defer appWindowsMu.Unlock()
	if w, ok := app.windows[hwnd]; ok {
		return w
	}
	return nil
}

func getHDCSize(hdc uintptr) (width int32, height int32) {
	var r rect
	procGetClipBox.Call(hdc, uintptr(unsafe.Pointer(&r)))
	return r.right - r.left, r.bottom - r.top
}

const chunkHeight = 100

// ensurePixBuffer grows this window's own RGBA->BGRA scratch buffer.
func (c *nativeWindow) ensurePixBuffer(size int) []byte {
	return growBuffer(&c.platform.pixBuffer, size)
}

func (c *nativeWindow) drawImageToHDC(img *image.RGBA, hdc uintptr, width, height int32) {
	drawRGBAToHDC(img, hdc, width, height, &c.platform.pixBuffer)
}

// drawRGBAToHDC blits img to hdc, converting RGBA->BGRA in chunks through
// the caller's scratch buffer.
func drawRGBAToHDC(img *image.RGBA, hdc uintptr, width, height int32, scratch *[]byte) {
	imgStride := img.Stride
	totalHeight := int(height)

	pixBuffer := growBuffer(scratch, int(width)*4*chunkHeight)

	for y := 0; y < totalHeight; y += chunkHeight {
		h := chunkHeight
		if y+h > totalHeight {
			h = totalHeight - y
		}

		bi := t_BITMAPINFO{
			Header: t_BITMAPINFOHEADER{
				Size:        uint32(unsafe.Sizeof(t_BITMAPINFOHEADER{})),
				Width:       width,
				Height:      -int32(h),
				Planes:      1,
				BitCount:    32,
				Compression: 0,
			},
		}

		srcOffset := y * imgStride
		dataSize := int(width) * 4 * h

		_ = srcOffset
		_ = dataSize
		copy(pixBuffer[:dataSize], img.Pix[srcOffset:srcOffset+dataSize])

		// Convert RGBA to BGRA
		//RgbaToBgraSIMD(pixBuffer[:dataSize])
		for i := 0; i < dataSize; i += 4 {
			b := pixBuffer[i+0]
			g := pixBuffer[i+1]
			r := pixBuffer[i+2]
			a := pixBuffer[i+3]
			pixBuffer[i+0] = r
			pixBuffer[i+1] = g
			pixBuffer[i+2] = b
			pixBuffer[i+3] = a
		}

		ptr := uintptr(unsafe.Pointer(&pixBuffer[0]))

		_ = ptr
		_ = bi

		procSetDIBitsToDevice.Call(
			hdc,
			0, uintptr(y), // xDest, yDest
			uintptr(width), uintptr(h), // w, h
			0, 0, // xSrc, ySrc
			0, uintptr(h), // Start scan line, number of scan lines
			ptr,
			uintptr(unsafe.Pointer(&bi)),
			0,
		)
	}
}

// Sanity caps on a single window's paintable area, not a shared buffer size.
const maxCanvasWidth = 6000
const maxCanvasHeight = 4000

// ensureCanvasBuffer grows this window's own paint buffer to fit size bytes, if needed.
func (c *nativeWindow) ensureCanvasBuffer(size int) []byte {
	return growBuffer(&c.platform.canvasBuffer, size)
}

// fillCanvasBuffer paints buf with this window's solid background color.
func fillCanvasBuffer(buf []byte, col color.RGBA) {
	for i := 0; i+3 < len(buf); i += 4 {
		buf[i+0] = col.R
		buf[i+1] = col.G
		buf[i+2] = col.B
		buf[i+3] = col.A
	}
}

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	//fmt.Println("Message:", native.MessageName(msg))

	/*ctrl, alt, shift := getModifierState()
	fmt.Println("Keys State:", "Ctrl:", ctrl, "Alt:", alt, "Shift:", shift)*/

	win := getNativeWindowByHandle(windowId(hwnd))

	switch msg {
	case c_WM_PAINT:

		dtBegin := time.Now()

		var ps t_PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		if win == nil {
			procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
			return 0
		}

		hdcWidth, hdcHeight := getHDCSize(hdc)
		if hdcWidth > maxCanvasWidth {
			hdcWidth = maxCanvasWidth
		}

		if hdcHeight > maxCanvasHeight {
			hdcHeight = maxCanvasHeight
		}

		// Clear the canvas to this window's own background color.
		canvasDataBufferSize := int(hdcWidth * hdcHeight * 4)
		buf := win.ensureCanvasBuffer(canvasDataBufferSize)
		fillCanvasBuffer(buf, win.platform.bgColor)

		img := &image.RGBA{
			Pix:    buf,
			Stride: int(hdcWidth) * 4,
			Rect:   image.Rect(0, 0, int(hdcWidth), int(hdcHeight)),
		}

		// The buffer is in physical pixels; the scale tells the canvas so
		if win.onPaint != nil {
			win.onPaint(img)
		}

		win.drawImageToHDC(img, hdc, hdcWidth, hdcHeight)

		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		win.drawTimes[win.drawTimesIndex] = time.Since(dtBegin).Microseconds()
		win.drawTimesIndex++
		if win.drawTimesIndex >= len(win.drawTimes) {
			win.drawTimesIndex = 0
		}

		return 0

	case c_WM_ACTIVATE:
		if wParam&0xFFFF == c_WA_INACTIVE && win != nil && win.onDeactivate != nil {
			win.onDeactivate()
		}
		// No return: DefWindowProc sets the keyboard focus on activation

	case c_WM_DROPFILES:
		if win != nil {
			win.processDropFiles(wParam)
		} else {
			procDragFinish.Call(wParam)
		}
		return 0

	case c_WM_NUI_CREATE_POPUP:
		// Popups are created on their owner's thread (see createPopupWindow)
		return createPopupHwnd(hwnd)

	case c_WM_DESTROY:
		procKillTimer.Call(uintptr(hwnd), timerID1ms)
		appWindowsMu.Lock()
		delete(app.windows, windowId(hwnd))
		appWindowsMu.Unlock()
		// The message loop is shared by all the windows and keeps running;
		// only the loops and goroutines waiting for this window end (see Exec)
		if win != nil && !win.platform.closed {
			win.releaseModalOwner()
			win.platform.closed = true
			close(win.platform.done)
		}
		return 0

	case c_WM_KEYDOWN:
		scanCode := uint32(wParam)

		k := Key(scanCode)
		if scanCode == 0x5B || scanCode == 0x5C {
			k = KeyWin
		}

		if win != nil && win.onKeyDown != nil {
			win.onKeyDown(k, getModifierState())
		}
		return 0

	case c_WM_KEYUP:
		scanCode := uint32(wParam)

		k := Key(scanCode)
		if scanCode == 0x5B || scanCode == 0x5C {
			k = KeyWin
		}

		if win != nil && win.onKeyUp != nil {
			win.onKeyUp(k, getModifierState())
		}
		return 0

	case c_WM_SYSKEYDOWN:
		scanCode := uint32(wParam)

		k := Key(scanCode)
		if scanCode == 0x5B || scanCode == 0x5C {
			k = KeyWin
		}

		if k == KeyF4 && getModifierState().Alt {
			break
		}

		/*fmt.Println("SYSKEY_DOWN", k.String())

		if k == KeyEnter {
			fmt.Println("Enter key pressed with Alt")
		}*/

		/*if k != KeyShift &&
			k != KeyCtrl &&
			k != KeyAlt &&
			k != KeyF10 {
			break
		}*/

		if win != nil && win.onKeyDown != nil {
			win.onKeyDown(k, getModifierState())
		}
		return 0

	case c_WM_SYSKEYUP:
		scanCode := uint32(wParam)

		k := Key(scanCode)
		if scanCode == 0x5B || scanCode == 0x5C {
			k = KeyWin
		}

		if win != nil && win.onKeyUp != nil {
			win.onKeyUp(k, getModifierState())
		}
		return 0

	case c_WM_SYSCHAR:
		// Alt+letter: the menu mnemonics and shortcuts were handled with
		// WM_SYSKEYDOWN; DefWindowProc would only beep for a letter the window
		// menu doesn't have. Alt+Space still opens the window menu.
		if wParam != ' ' {
			return 0
		}

	case c_WM_CHAR:
		//println("Char typed:", rune(wParam), "=", string(rune(wParam)))

		if win != nil && win.onChar != nil && wParam >= 32 {
			win.onChar(rune(wParam))
		}
		return 0

	case c_WM_MOUSEMOVE:
		if win != nil && win.onMouseMove != nil {
			x, y := win.mousePos(lParam)
			win.onMouseMove(x, y)
		}

		if !win.mouseInside {
			win.mouseInside = true
			if win != nil {
				win.lastSetCursor = MouseCursorNotDefined
			}
			if win != nil && win.onMouseEnter != nil {
				win.onMouseEnter()
			}

			tme := t_TRACKMOUSEEVENT{
				cbSize:    uint32(unsafe.Sizeof(t_TRACKMOUSEEVENT{})),
				dwFlags:   c_TME_LEAVE,
				hwndTrack: hwnd,
			}
			procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
		}

		win.changeMouseCursor(win.currentCursor)
		return 0

	case c_WM_LBUTTONDOWN:
		procSetCapture.Call(uintptr(hwnd))
		if win != nil && win.onMouseButtonDown != nil {
			x, y := win.mousePos(lParam)
			win.onMouseButtonDown(MouseButtonLeft, int(x), int(y))
			doubleClickDetected := false
			if win.lastMouseButton == MouseButtonLeft {
				if time.Since(win.lastMouseDownTime) < win.dblClickTime &&
					math.Abs(float64(win.lastMouseDownX-int(x))) < 3 && math.Abs(float64(win.lastMouseDownY-int(y))) < 3 {
					if win.onMouseButtonDblClick != nil {
						win.onMouseButtonDblClick(MouseButtonLeft, int(x), int(y))
					}
					doubleClickDetected = true
				}
			}
			win.lastMouseDownX = int(x)
			win.lastMouseDownY = int(y)
			win.lastMouseButton = MouseButtonLeft
			if doubleClickDetected {
				win.lastMouseDownTime = time.Now().Add(-win.dblClickTime * 2)
			} else {
				win.lastMouseDownTime = time.Now()
			}
		}
		return 0

	case c_WM_LBUTTONUP:
		procReleaseCapture.Call()
		if win != nil && win.onMouseButtonUp != nil {
			x, y := win.mousePos(lParam)
			win.onMouseButtonUp(MouseButtonLeft, int(x), int(y))
		}
		return 0

	case c_WM_RBUTTONDOWN:
		if win != nil && win.onMouseButtonDown != nil {
			x, y := win.mousePos(lParam)
			win.onMouseButtonDown(MouseButtonRight, int(x), int(y))
			doubleClickDetected := false
			if win.lastMouseButton == MouseButtonRight {
				if time.Since(win.lastMouseDownTime) < win.dblClickTime &&
					math.Abs(float64(win.lastMouseDownX-int(x))) < 3 && math.Abs(float64(win.lastMouseDownY-int(y))) < 3 {
					if win.onMouseButtonDblClick != nil {
						win.onMouseButtonDblClick(MouseButtonRight, int(x), int(y))
					}
					doubleClickDetected = true
				}
			}
			win.lastMouseDownX = int(x)
			win.lastMouseDownY = int(y)
			win.lastMouseButton = MouseButtonRight
			if doubleClickDetected {
				win.lastMouseDownTime = time.Now().Add(-win.dblClickTime * 2)
			} else {
				win.lastMouseDownTime = time.Now()
			}
		}
		return 0

	case c_WM_RBUTTONUP:
		if win != nil && win.onMouseButtonUp != nil {
			x, y := win.mousePos(lParam)
			win.onMouseButtonUp(MouseButtonRight, int(x), int(y))
		}
		return 0

	case c_WM_MBUTTONDOWN:
		if win != nil && win.onMouseButtonDown != nil {
			x, y := win.mousePos(lParam)

			win.onMouseButtonDown(MouseButtonMiddle, int(x), int(y))
			doubleClickDetected := false
			if win.lastMouseButton == MouseButtonMiddle {
				if time.Since(win.lastMouseDownTime) < win.dblClickTime &&
					math.Abs(float64(win.lastMouseDownX-int(x))) < 3 && math.Abs(float64(win.lastMouseDownY-int(y))) < 3 {
					if win.onMouseButtonDblClick != nil {
						win.onMouseButtonDblClick(MouseButtonMiddle, int(x), int(y))
					}
					doubleClickDetected = true
				}
			}
			win.lastMouseDownX = int(x)
			win.lastMouseDownY = int(y)
			win.lastMouseButton = MouseButtonMiddle
			if doubleClickDetected {
				win.lastMouseDownTime = time.Now().Add(-win.dblClickTime * 2)
			} else {
				win.lastMouseDownTime = time.Now()
			}
		}
		return 0

	case c_WM_MBUTTONUP:
		if win != nil && win.onMouseButtonUp != nil {
			x, y := win.mousePos(lParam)
			win.onMouseButtonUp(MouseButtonMiddle, int(x), int(y))
		}
		return 0

	case c_WM_MOUSEWHEEL:
		deltaY := int16((wParam >> 16) & 0xFFFF)
		if win != nil && win.onMouseWheel != nil {
			win.onMouseWheel(0, int(deltaY/120))
		}
		return 0

	/*case c_WM_LBUTTONDBLCLK:
		if win != nil && win.onMouseButtonDblClick != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.onMouseButtonDblClick(MouseButtonLeft, int(x), int(y))
		}
		return 0

	case c_WM_RBUTTONDBLCLK:
		if win != nil && win.onMouseButtonDblClick != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.onMouseButtonDblClick(MouseButtonRight, int(x), int(y))
		}
		return 0

	case c_WM_MBUTTONDBLCLK:
		if win != nil && win.onMouseButtonDblClick != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.onMouseButtonDblClick(MouseButtonMiddle, int(x), int(y))
		}
		return 0*/

	case c_WM_MOUSELEAVE:
		win.mouseInside = false
		if win != nil && win.onMouseLeave != nil {
			win.onMouseLeave()
		}
		return 0

	case c_WM_SIZE:
		if win != nil {
			// The client area in logical pixels
			width := win.toLogicalSize(int(uint16(lParam & 0xFFFF)))
			height := win.toLogicalSize(int(uint16((lParam >> 16) & 0xFFFF)))
			if win.onResize != nil {
				win.onResize(width, height)
			}
			win.windowWidth = width
			win.windowHeight = height
		}
		procInvalidateRect.Call(uintptr(hwnd), 0, 0)
		return 0

	case c_WM_MOVE:
		if win != nil {
			x := win.toLogicalScreen(int(int16(lParam & 0xFFFF)))
			y := win.toLogicalScreen(int(int16((lParam >> 16) & 0xFFFF)))
			win.windowPosX = x
			win.windowPosY = y
			if win.onMove != nil {
				win.onMove(x, y)
			}
		}
		return 0

	case c_WM_DPICHANGED:
		if win != nil {
			win.dpiChanged(wParam, lParam)
		}
		return 0

	case c_WM_NUI_CLOSE:
		if win != nil {
			win.destroy()
		} else {
			procDestroyWindow.Call(uintptr(hwnd))
		}
		return 0

	case c_WM_CLOSE:
		if win != nil && win.onCloseRequest != nil {
			allow := win.onCloseRequest()
			if !allow {
				return 0
			}
		}
		if win != nil {
			win.releaseModalOwner()
		}
		procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return 0

	case c_WM_TIMER:
		if wParam == timerID1ms {
			if win != nil && win.onTimer != nil {
				if time.Since(win.timerLastDT) > time.Millisecond*10 {
					win.onTimer()
					win.timerLastDT = time.Now()
				}
			}
		}
		return 0

	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

// mousePos returns the mouse position of the message in logical pixels
func (c *nativeWindow) mousePos(lParam uintptr) (int, int) {
	x := int(int16(lParam & 0xFFFF))
	y := int(int16((lParam >> 16) & 0xFFFF))
	return c.toLogical(x), c.toLogical(y)
}

func (c *nativeWindow) changeMouseCursor(cursor MouseCursor) bool {
	if c.lastSetCursor == cursor && c.lastSetCursor != MouseCursorNotDefined {
		return true
	}

	hCursor := loadMouseCursor(cursor)
	if hCursor == 0 {
		return false
	}

	c.lastSetCursor = cursor
	//fmt.Println("Setting cursor to:", cursor)

	ret, _, _ := procSetCursor.Call(hCursor)
	return ret != 0
}

// loadMouseCursor returns the system cursor for the cursor kind, 0 if none.
func loadMouseCursor(cursor MouseCursor) uintptr {
	var cursorID uintptr
	switch cursor {
	case MouseCursorArrow:
		cursorID = c_IDC_ARROW
	case MouseCursorPointer:
		cursorID = c_IDC_HAND
	case MouseCursorResizeHor:
		cursorID = c_IDC_SIZEWE
	case MouseCursorResizeVer:
		cursorID = c_IDC_SIZENS
	case MouseCursorIBeam:
		cursorID = c_IDC_IBEAM
	default:
		return 0
	}
	hCursor, _, _ := procLoadCursorW.Call(0, cursorID)
	return hCursor
}

func createHICONFromRGBA(img *image.RGBA) syscall.Handle {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	pixels := make([]byte, 0, width*height*4)

	for y := 0; y < height; y++ {
		rowStart := y * img.Stride
		for x := 0; x < width; x++ {
			i := rowStart + x*4
			r := img.Pix[i]
			g := img.Pix[i+1]
			b := img.Pix[i+2]
			a := img.Pix[i+3]

			pixels = append(pixels, b, g, r, a)
		}
	}

	/*for y := height - 1; y >= 0; y-- {
		rowStart := y * img.Stride
		for x := 0; x < width; x++ {
			i := rowStart + x*4
			r := img.Pix[i]
			g := img.Pix[i+1]
			b := img.Pix[i+2]
			a := img.Pix[i+3]

			// Windows ожидает BGRA
			pixels = append(pixels, b, g, r, a)
		}
	}*/

	hIcon, _, _ := procCreateIcon.Call(
		0, // hInstance (0 = current)
		uintptr(width),
		uintptr(height),
		1,  // Planes
		32, // BitsPerPixel
		0,  // XOR mask (set to 0 — not used)
		uintptr(unsafe.Pointer(&pixels[0])),
	)

	return syscall.Handle(hIcon)
}

func getScreenSize() (width, height int) {
	w, _, _ := procGetSystemMetrics.Call(c_SM_CXSCREEN)
	h, _, _ := procGetSystemMetrics.Call(c_SM_CYSCREEN)
	return int(w), int(h)
}

func setDarkMode(hwnd uintptr, enable bool) {
	var useDark uint32
	if enable {
		useDark = 1
	}

	// Сначала пробуем с 20
	ret, _, _ := procDwmSetWindowAttribute.Call(
		hwnd,
		uintptr(c_DWMWA_USE_IMMERSIVE_DARK_MODE),
		uintptr(unsafe.Pointer(&useDark)),
		unsafe.Sizeof(useDark),
	)

	// Если не сработало — пробуем 19
	if ret != 0 {
		procDwmSetWindowAttribute.Call(
			hwnd,
			uintptr(c_DWMWA_USE_IMMERSIVE_DARK_MODE_OLD),
			uintptr(unsafe.Pointer(&useDark)),
			unsafe.Sizeof(useDark),
		)
	}
}

const (
	VK_SHIFT   = 0x10
	VK_CONTROL = 0x11
	VK_MENU    = 0x12 // Это клавиша ALT
)

// currentModifierState is the state of the keys right now (GetAsyncKeyState),
// not as of the last message like getModifierState
func currentModifierState() KeyModifiers {
	isPressed := func(vk int) bool {
		ret, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
		return (uint16(ret) & 0x8000) != 0
	}
	return KeyModifiers{
		Ctrl:  isPressed(VK_CONTROL),
		Alt:   isPressed(VK_MENU),
		Shift: isPressed(VK_SHIFT),
	}
}

func getModifierState() KeyModifiers {
	// Highest bit (0x8000) indicates the key is pressed
	isPressed := func(vk int) bool {
		ret, _, _ := procGetKeyState.Call(uintptr(vk))
		return (uint16(ret) & 0x8000) != 0
	}

	ctrl := isPressed(VK_CONTROL)
	alt := isPressed(VK_MENU)
	shift := isPressed(VK_SHIFT)

	return KeyModifiers{
		Ctrl:  ctrl,
		Alt:   alt,
		Shift: shift,
	}
}
