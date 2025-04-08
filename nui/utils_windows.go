package nui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
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

	procLoadCursorW = user32.NewProc("LoadCursorW")
	procSetCursor   = user32.NewProc("SetCursor")

	procSetTimer  = user32.NewProc("SetTimer")
	procKillTimer = user32.NewProc("KillTimer")

	procSendMessageW = user32.NewProc("SendMessageW")
	procCreateIcon   = user32.NewProc("CreateIcon")

	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	CW_USEDEFAULT       = 0x80000000
	SW_SHOWDEFAULT      = 10

	SW_HIDE          = 0
	SW_SHOWNORMAL    = 1
	SW_SHOWMINIMIZED = 2
	SW_SHOWMAXIMIZED = 3
	SW_RESTORE       = 9

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010

	WM_SETICON      = 0x0080
	ICON_SMALL      = 0
	ICON_BIG        = 1
	IMAGE_ICON      = 1
	LR_DEFAULTCOLOR = 0x0000

	IDC_ARROW  = uintptr(32512)
	IDC_HAND   = uintptr(32649)
	IDC_SIZEWE = uintptr(32644)
	IDC_SIZENS = uintptr(32645)
	IDC_IBEAM  = uintptr(32513)

	CS_DBLCLKS = 0x0008
	CS_OWNDC   = 0x0020

	WM_MOVE = 0x0003
	WM_SIZE = 0x0005

	WM_CLOSE   = 0x0010
	WM_DESTROY = 0x0002

	WM_KEYDOWN = 0x0100
	WM_KEYUP   = 0x0101
	WM_CHAR    = 0x0102

	WM_SYSKEYDOWN = 0x0104
	WM_SYSKEYUP   = 0x0105
	WM_SYSCHAR    = 0x0106

	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_MOUSEMOVE   = 0x0200
	WM_RBUTTONDOWN = 0x0204
	WM_RBUTTONUP   = 0x0205
	WM_MBUTTONDOWN = 0x0207
	WM_MBUTTONUP   = 0x0208
	WM_MOUSEWHEEL  = 0x020A // Dec: 522
	WM_XBUTTONDOWN = 0x020B
	WM_XBUTTONUP   = 0x020C

	// dec 132 to hex is 0x84

	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONDBLCLK = 0x0206
	WM_MBUTTONDBLCLK = 0x0209

	WM_MOUSELEAVE = 0x02A3

	TME_LEAVE = 0x00000002

	WM_TIMER   = 0x0113
	timerID1ms = 1 // любой уникальный ID
)

type WNDCLASSEXW struct {
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

type PAINTSTRUCT struct {
	hdc         syscall.Handle
	fErase      int32
	rcPaint     struct{ left, top, right, bottom int32 }
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type MSG struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

type BITMAPINFOHEADER struct {
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

type RGBQUAD struct {
	Blue     byte
	Green    byte
	Red      byte
	Reserved byte
}

type BITMAPINFO struct {
	Header BITMAPINFOHEADER
	Colors [3]RGBQUAD
}

type TRACKMOUSEEVENT struct {
	cbSize      uint32
	dwFlags     uint32
	hwndTrack   syscall.Handle
	dwHoverTime uint32
}

const (
	WM_PAINT = 0x000F
)

var (
	procGetClipBox = gdi32.NewProc("GetClipBox")
)

const (
	HORZRES   = 8
	VERTRES   = 10
	BITSPIXEL = 12
	PLANES    = 14

	OBJ_DC        = 1
	OBJ_MEMDC     = 10
	OBJ_ENHMETADC = 12
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

func GetNativeWindowByHandle(hwnd syscall.Handle) *NativeWindow {
	if w, ok := hwnds[hwnd]; ok {
		return w
	}
	return nil
}

func getHDCSize(hdc uintptr) (width int32, height int32) {
	var r rect
	procGetClipBox.Call(hdc, uintptr(unsafe.Pointer(&r)))
	return r.right - r.left, r.bottom - r.top
}

const chunkHeight = 300
const maxWidth = 10000

var pixBuffer = make([]byte, 4*chunkHeight*maxWidth)

//go:noescape
func RgbaToBgraSIMD(data []byte)

func drawImageToHDC(img *image.RGBA, hdc uintptr, width, height int32) {
	imgStride := img.Stride
	totalHeight := int(height)

	for y := 0; y < totalHeight; y += chunkHeight {
		h := chunkHeight
		if y+h > totalHeight {
			h = totalHeight - y
		}

		bi := BITMAPINFO{
			Header: BITMAPINFOHEADER{
				Size:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
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
		RgbaToBgraSIMD(pixBuffer[:dataSize])

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

const maxCanvasWidth = 6000
const maxCanvasHeight = 4000

var canvasBuffer = make([]byte, maxCanvasWidth*maxCanvasHeight*4)
var canvasBufferBackground = make([]byte, maxCanvasWidth*maxCanvasHeight*4)

func initCanvasBufferBackground(col color.Color) {
	dataSize := maxCanvasWidth * maxCanvasHeight * 4
	r, g, b, a := col.RGBA()
	for i := 0; i < dataSize; i += 4 {
		canvasBufferBackground[i+0] = byte(r)
		canvasBufferBackground[i+1] = byte(g)
		canvasBufferBackground[i+2] = byte(b)
		canvasBufferBackground[i+3] = byte(a)
	}
}

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	//fmt.Println("Message:", native.MessageName(msg))

	win := GetNativeWindowByHandle(hwnd)

	switch msg {
	case WM_PAINT:

		dtBegin := time.Now()

		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		hdcWidth, hdcHeight := getHDCSize(hdc)
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

		if win != nil && win.OnPaint != nil {
			win.OnPaint(img)
		}

		drawImageToHDC(img, hdc, hdcWidth, hdcHeight)

		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		win.drawTimes[win.drawTimesIndex] = time.Since(dtBegin).Microseconds()
		win.drawTimesIndex++
		if win.drawTimesIndex >= len(win.drawTimes) {
			win.drawTimesIndex = 0
		}

		return 0

	case WM_DESTROY:
		procKillTimer.Call(uintptr(hwnd), timerID1ms)
		procPostQuitMessage.Call(0)
		return 0

	case WM_KEYDOWN:
		scanCode := uint32(wParam)

		needGenEvent := true

		k := Key(scanCode)
		if scanCode == 0x5B || scanCode == 0x5C {
			k = KeyWin
		}

		if k == KeyShift {
			if win.keyModifiers.Shift {
				needGenEvent = false
			}
			win.keyModifiers.Shift = true
		} else if k == KeyCtrl {
			if win.keyModifiers.Ctrl {
				needGenEvent = false
			}
			win.keyModifiers.Ctrl = true
		} else if k == KeyAlt {
			if win.keyModifiers.Alt {
				needGenEvent = false
			}
			win.keyModifiers.Alt = true
		} else if k == KeyCommand {
			if win.keyModifiers.Cmd {
				needGenEvent = false
			}
			win.keyModifiers.Cmd = true
		}

		if win != nil && win.OnKeyDown != nil && needGenEvent {
			win.OnKeyDown(k, win.keyModifiers)
		}
		return 0

	case WM_KEYUP:
		scanCode := uint32(wParam)

		needGenEvent := true
		k := Key(scanCode)
		if scanCode == 0x5B || scanCode == 0x5C {
			k = KeyWin
		}

		if k == KeyShift {
			if !win.keyModifiers.Shift {
				needGenEvent = false
			}
			win.keyModifiers.Shift = false
		} else if k == KeyCtrl {
			if !win.keyModifiers.Ctrl {
				needGenEvent = false
			}
			win.keyModifiers.Ctrl = false
		} else if k == KeyAlt {
			if !win.keyModifiers.Alt {
				needGenEvent = false
			}
			win.keyModifiers.Alt = false
		} else if k == KeyCommand {
			if !win.keyModifiers.Cmd {
				needGenEvent = false
			}
			win.keyModifiers.Cmd = false
		}

		if win != nil && win.OnKeyUp != nil && needGenEvent {
			win.OnKeyUp(k, win.keyModifiers)
		}
		return 0

	case WM_SYSKEYDOWN:
		scanCode := uint32(wParam)

		needGenEvent := true

		k := Key(scanCode)
		if scanCode == 0x5B || scanCode == 0x5C {
			k = KeyWin
		}

		if k == KeyShift {
			if win.keyModifiers.Shift {
				needGenEvent = false
			}
			win.keyModifiers.Shift = true
		} else if k == KeyCtrl {
			if win.keyModifiers.Ctrl {
				needGenEvent = false
			}
			win.keyModifiers.Ctrl = true
		} else if k == KeyAlt {
			if win.keyModifiers.Alt {
				needGenEvent = false
			}
			win.keyModifiers.Alt = true
		} else if k == KeyCommand {
			if win.keyModifiers.Cmd {
				needGenEvent = false
			}
			win.keyModifiers.Cmd = true
		}

		if win != nil && win.OnKeyDown != nil && needGenEvent {
			win.OnKeyDown(k, win.keyModifiers)
		}
		return 0

	case WM_SYSKEYUP:
		scanCode := uint32(wParam)

		needGenEvent := true

		k := Key(scanCode)
		if scanCode == 0x5B || scanCode == 0x5C {
			k = KeyWin
		}

		if k == KeyShift {
			if !win.keyModifiers.Shift {
				needGenEvent = false
			}
			win.keyModifiers.Shift = false
		} else if k == KeyCtrl {
			if !win.keyModifiers.Ctrl {
				needGenEvent = false
			}
			win.keyModifiers.Ctrl = false
		} else if k == KeyAlt {
			if !win.keyModifiers.Alt {
				needGenEvent = false
			}
			win.keyModifiers.Alt = false
		} else if k == KeyCommand {
			if !win.keyModifiers.Cmd {
				needGenEvent = false
			}
			win.keyModifiers.Cmd = false
		}

		if win != nil && win.OnKeyUp != nil && needGenEvent {
			win.OnKeyUp(k, win.keyModifiers)
		}
		return 0

	case WM_SYSCHAR:
		println("SysChar typed:", rune(wParam), "=", string(rune(wParam)))
		return 0

	case WM_CHAR:
		println("Char typed:", rune(wParam), "=", string(rune(wParam)))

		if win != nil && win.OnChar != nil && wParam >= 32 {
			win.OnChar(rune(wParam))
		}
		return 0

	case WM_MOUSEMOVE:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		if win != nil && win.OnMouseMove != nil {
			win.OnMouseMove(int(x), int(y))
		}

		if !win.mouseInside {
			win.mouseInside = true
			if win != nil {
				win.lastSetCursor = MouseCursorNotDefined
			}
			if win != nil && win.OnMouseEnter != nil {
				win.OnMouseEnter()
			}

			tme := TRACKMOUSEEVENT{
				cbSize:    uint32(unsafe.Sizeof(TRACKMOUSEEVENT{})),
				dwFlags:   TME_LEAVE,
				hwndTrack: hwnd,
			}
			procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
		}

		win.changeMouseCursor(win.currentCursor)
		return 0

	case WM_LBUTTONDOWN:
		if win != nil && win.OnMouseButtonDown != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.OnMouseButtonDown(MouseButtonLeft, int(x), int(y))
		}
		return 0

	case WM_LBUTTONUP:
		if win != nil && win.OnMouseButtonUp != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.OnMouseButtonUp(MouseButtonLeft, int(x), int(y))
		}
		return 0

	case WM_RBUTTONDOWN:
		if win != nil && win.OnMouseButtonDown != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.OnMouseButtonDown(MouseButtonRight, int(x), int(y))
		}
		return 0

	case WM_RBUTTONUP:
		if win != nil && win.OnMouseButtonUp != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.OnMouseButtonUp(MouseButtonRight, int(x), int(y))
		}
		return 0

	case WM_MBUTTONDOWN:
		if win != nil && win.OnMouseButtonDown != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.OnMouseButtonDown(MouseButtonMiddle, int(x), int(y))
		}
		return 0

	case WM_MBUTTONUP:
		if win != nil && win.OnMouseButtonUp != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.OnMouseButtonUp(MouseButtonMiddle, int(x), int(y))
		}
		return 0

	case WM_MOUSEWHEEL:
		deltaY := int16((wParam >> 16) & 0xFFFF)
		if win != nil && win.OnMouseWheel != nil {
			win.OnMouseWheel(0, int(deltaY/120))
		}
		return 0

	case WM_LBUTTONDBLCLK:
		if win != nil && win.OnMouseButtonDblClick != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.OnMouseButtonDblClick(MouseButtonLeft, int(x), int(y))
		}
		return 0

	case WM_RBUTTONDBLCLK:
		if win != nil && win.OnMouseButtonDblClick != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.OnMouseButtonDblClick(MouseButtonRight, int(x), int(y))
		}
		return 0

	case WM_MBUTTONDBLCLK:
		if win != nil && win.OnMouseButtonDblClick != nil {
			x := int16(lParam & 0xFFFF)
			y := int16((lParam >> 16) & 0xFFFF)
			win.OnMouseButtonDblClick(MouseButtonMiddle, int(x), int(y))
		}
		return 0

	case WM_MOUSELEAVE:
		win.mouseInside = false
		if win != nil && win.OnMouseLeave != nil {
			win.OnMouseLeave()
		}
		return 0

	case WM_SIZE:
		width := int16(lParam & 0xFFFF)
		height := int16((lParam >> 16) & 0xFFFF)
		if win != nil && win.OnResize != nil {
			win.OnResize(int(width), int(height))
		}
		win.windowWidth = int(width)
		win.windowHeight = int(height)
		procInvalidateRect.Call(uintptr(hwnd), 0, 0)
		return 0

	case WM_MOVE:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		win.windowPosX = int(x)
		win.windowPosY = int(y)
		if win != nil && win.OnMove != nil {
			win.OnMove(int(x), int(y))
		}
		return 0

	case WM_CLOSE:
		if win != nil && win.OnCloseRequest != nil {
			allow := win.OnCloseRequest()
			if !allow {
				return 0
			}
		}
		procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return 0

	case WM_TIMER:
		if wParam == timerID1ms {
			if win != nil && win.OnTimer != nil {
				win.OnTimer()
			}
		}
		return 0

	default:
		ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	}
}

func (c *NativeWindow) changeMouseCursor(cursor MouseCursor) bool {
	var cursorID uintptr

	if c.lastSetCursor == cursor && c.lastSetCursor != MouseCursorNotDefined {
		return true
	}

	switch cursor {
	case MouseCursorArrow:
		cursorID = IDC_ARROW
	case MouseCursorPointer:
		cursorID = IDC_HAND
	case MouseCursorResizeHor:
		cursorID = IDC_SIZEWE
	case MouseCursorResizeVer:
		cursorID = IDC_SIZENS
	case MouseCursorIBeam:
		cursorID = IDC_IBEAM
	default:
		return false
	}

	hCursor, _, _ := procLoadCursorW.Call(0, cursorID)
	if hCursor == 0 {
		return false
	}

	c.lastSetCursor = cursor
	fmt.Println("Setting cursor to:", cursor)

	ret, _, _ := procSetCursor.Call(hCursor)
	return ret != 0
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
	w, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	h, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	return int(w), int(h)
}
