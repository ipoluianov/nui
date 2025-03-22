package nui

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"syscall"
	"unsafe"
)

type NativeWindow struct {
	hwnd syscall.Handle

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
	OnMouseWheel                   func(delta int)
	OnMouseDoubleClickLeftButton   func(x, y int)
	OnMouseDoubleClickRightButton  func(x, y int)
	OnMouseDoubleClickMiddleButton func(x, y int)

	// Window events
	OnCreated func()
	OnPaint   func(width int, height int) *image.RGBA
	OnMove    func(x, y int)
	OnResize  func(width, height int)
	OnClosing func()
}

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
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	CW_USEDEFAULT       = 0x80000000
	SW_SHOWDEFAULT      = 10

	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010

	IDC_ARROW  = uintptr(32512)
	IDC_HAND   = uintptr(32649)
	IDC_SIZEWE = uintptr(32644)
	IDC_SIZENS = uintptr(32645)
	IDC_IBEAM  = uintptr(32513)

	CS_DBLCLKS = 0x0008
	CS_OWNDC   = 0x0020

	WM_SIZE = 0x0005

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
	Colors [1]RGBQUAD
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

var mouseInside bool = false

var (
	procGetDeviceCaps = gdi32.NewProc("GetDeviceCaps")
	procGetObjectType = gdi32.NewProc("GetObjectType")
	procGetClipBox    = gdi32.NewProc("GetClipBox")
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

func convertRGBAToBytes(rgba *image.RGBA) []byte {
	pixels := make([]byte, 0)
	for y := rgba.Bounds().Min.Y; y < rgba.Bounds().Max.Y; y++ {
		for x := rgba.Bounds().Min.X; x < rgba.Bounds().Max.X; x++ {
			r, g, b, a := rgba.At(x, y).RGBA()
			pixels = append(pixels, byte(b>>8), byte(g>>8), byte(r>>8), byte(a>>8))
		}
	}

	return pixels
}

var imageBytes []byte
var hwnds map[syscall.Handle]*NativeWindow

func init() {
	hwnds = make(map[syscall.Handle]*NativeWindow)
}

func GetNativeWindowByHandle(hwnd syscall.Handle) *NativeWindow {
	if w, ok := hwnds[hwnd]; ok {
		return w
	}
	return nil
}

/*func getImageBytes() (bs []byte, width int32, height int32) {
	if imageBytes != nil {
		return imageBytes, 800, 600
	}
	rgba, err := loadPngFromBytes(TestPng)
	if err != nil {
		panic(err)
	}
	bs = convertRGBAToBytes(rgba)
	imageBytes = bs
	width = int32(rgba.Bounds().Max.X)
	height = int32(rgba.Bounds().Max.Y)
	return
}*/

func GetRGBATestImage() *image.RGBA {
	rgba, err := loadPngFromBytes(TestPng)
	if err != nil {
		panic(err)
	}
	return rgba
}

func getHDCSize(hdc uintptr) (width int32, height int32) {
	w, _, _ := procGetDeviceCaps.Call(uintptr(hdc), HORZRES)
	h, _, _ := procGetDeviceCaps.Call(uintptr(hdc), VERTRES)
	return int32(w), int32(h)
}

var pixBuffer = make([]byte, 1920*1080*4*8)

func drawImageToHDC(img *image.RGBA, hdc uintptr, width, height int32) {
	height = 100
	bi := BITMAPINFO{
		Header: BITMAPINFOHEADER{
			Size:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			Width:       width,
			Height:      -height,
			Planes:      1,
			BitCount:    32,
			Compression: 0,
		},
	}

	pixels := convertRGBAToBytes(img)

	copy(pixBuffer, pixels)

	pointerToBuffer := uintptr(unsafe.Pointer(&pixBuffer[0]))

	procSetDIBitsToDevice.Call(
		hdc,
		0, 0,
		uintptr(width), uintptr(height),
		0, 0,
		0,
		uintptr(height),
		pointerToBuffer,
		uintptr(unsafe.Pointer(&bi)),
		0,
	)
}

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	//fmt.Println("Message:", native.MessageName(msg))

	win := GetNativeWindowByHandle(hwnd)

	switch msg {
	case WM_PAINT:

		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		hdcWidth, hdcHeight := getHDCSize(hdc)

		img := image.NewRGBA(image.Rect(0, 0, int(hdcWidth), int(hdcHeight)))

		if win != nil && win.OnPaint != nil {
			rgba := win.OnPaint(int(hdcWidth), int(hdcHeight))
			draw.Draw(img, img.Bounds(), rgba, image.Point{0, 0}, draw.Src)
		}

		drawImageToHDC(img, hdc, hdcWidth, hdcHeight)

		/*bi := BITMAPINFO{
			Header: BITMAPINFOHEADER{
				Size:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
				Width:       hdcWidth,
				Height:      -hdcHeight,
				Planes:      1,
				BitCount:    32,
				Compression: 0,
			},
		}

		if win != nil && win.OnPaint != nil {
			rgba := win.OnPaint(int(hdcWidth), int(hdcHeight))
			pixels := convertRGBAToBytes(rgba)
			copy(pixBuffer, pixels)
		}

		imageWidthAsUintPtr := uintptr(hdcWidth)
		imageHeightAsUintPtr := uintptr(hdcHeight)

		pointerToBuffer := uintptr(unsafe.Pointer(&pixBuffer[0]))

		procSetDIBitsToDevice.Call(
			hdc,
			0, 0,
			imageWidthAsUintPtr, imageHeightAsUintPtr,
			0, 0,
			0,
			imageHeightAsUintPtr,
			pointerToBuffer,
			uintptr(unsafe.Pointer(&bi)),
			0,
		)*/

		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		return 0

	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0

	case WM_KEYDOWN:
		scanCode := (lParam >> 16) & 0xFF
		if win != nil && win.OnKeyDown != nil {
			win.OnKeyDown(Key(scanCode))
		}
		return 0

	case WM_KEYUP:
		scanCode := (lParam >> 16) & 0xFF
		if win != nil && win.OnKeyUp != nil {
			win.OnKeyUp(Key(scanCode))
		}
		return 0

	case WM_SYSKEYDOWN:
		println("SysKey down:", wParam)
		return 0

	case WM_SYSKEYUP:
		println("SysKey up:", wParam)
		return 0

	case WM_SYSCHAR:
		println("SysChar typed:", rune(wParam), "=", string(rune(wParam)))
		return 0

	case WM_CHAR:
		println("Char typed:", rune(wParam), "=", string(rune(wParam)))
		if win != nil && win.OnChar != nil {
			win.OnChar(rune(wParam))
		}
		return 0

	case WM_MOUSEMOVE:
		//x := int16(lParam & 0xFFFF)
		//y := int16((lParam >> 16) & 0xFFFF)
		// println("Mouse move:", x, y)

		if !mouseInside {
			mouseInside = true
			println("Mouse entered window")

			tme := TRACKMOUSEEVENT{
				cbSize:    uint32(unsafe.Sizeof(TRACKMOUSEEVENT{})),
				dwFlags:   TME_LEAVE,
				hwndTrack: hwnd,
			}
			procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
		}
		return 0

	case WM_LBUTTONDOWN:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		println("Left button down at:", x, y)
		procInvalidateRect.Call(uintptr(hwnd), 0, 0)
		return 0

	case WM_LBUTTONUP:
		println("Left button up")
		return 0

	case WM_RBUTTONDOWN:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		println("Right button down at:", x, y)
		return 0

	case WM_RBUTTONUP:
		println("Right button up")
		return 0

	case WM_MBUTTONDOWN:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		println("Middle button down at:", x, y)
		return 0

	case WM_MBUTTONUP:
		println("Middle button up")
		return 0

	case WM_MOUSEWHEEL:
		delta := int16((wParam >> 16) & 0xFFFF)
		println("Mouse wheel delta:", delta)
		return 0

	case WM_LBUTTONDBLCLK:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		println("Left double click at:", x, y)
		return 0

	case WM_RBUTTONDBLCLK:
		println("Right double click")
		return 0

	case WM_MBUTTONDBLCLK:
		println("Middle double click")
		return 0

	case WM_MOUSELEAVE:
		mouseInside = false
		println("Mouse left window")
		return 0

	case WM_SIZE:
		procInvalidateRect.Call(uintptr(hwnd), 0, 0)
		return 0

	default:
		ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	}
}

///////////////////////////////////////////////////////////////////

func CreateWindow() *NativeWindow {
	var c NativeWindow
	className, _ := syscall.UTF16PtrFromString("MyWindowClass")
	windowTitle, _ := syscall.UTF16PtrFromString("1234567")

	hInstance, _, _ := procGetModuleHandleW.Call(0)

	wndClass := WNDCLASSEXW{
		cbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		style:         CS_OWNDC | CS_DBLCLKS,
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     syscall.Handle(hInstance),
		hCursor:       0,
		hbrBackground: 5,
		lpszClassName: className,
	}

	if err := procTextOutW.Find(); err != nil {
		panic("TextOutW not found: " + err.Error())
	}

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		CW_USEDEFAULT,
		CW_USEDEFAULT,
		640,
		480,
		0,
		0,
		hInstance,
		0,
	)

	c.hwnd = syscall.Handle(hwnd)

	hwnds[c.hwnd] = &c

	procShowWindow.Call(hwnd, SW_SHOWDEFAULT)
	procInvalidateRect.Call(uintptr(hwnd), 0, 0)
	procUpdateWindow.Call(hwnd)

	return &c
}

func (c *NativeWindow) EventLoop() {
	var msg MSG
	for {
		ret, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		e := err.(syscall.Errno)
		if e != 0 {
			panic(e)
		}

		if ret == 0 {
			fmt.Println("Exiting...")
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (c *NativeWindow) Close() {
	procPostMessageW.Call(uintptr(c.hwnd), WM_DESTROY, 0, 0)
}

func (c *NativeWindow) SetTitle(title string) {
	strPtr, _ := syscall.UTF16PtrFromString(title)
	procSetWindowTextW.Call(
		uintptr(c.hwnd),
		uintptr(unsafe.Pointer(strPtr)),
	)
}

func (c *NativeWindow) Move(x, y int) {
	flags := SWP_NOSIZE | SWP_NOZORDER

	procSetWindowPos.Call(
		uintptr(c.hwnd),
		0,
		uintptr(x), uintptr(y),
		0, 0,
		uintptr(flags),
	)
}

func (c *NativeWindow) Resize(width, height int) {
	flags := SWP_NOMOVE | SWP_NOZORDER

	procSetWindowPos.Call(
		uintptr(c.hwnd),
		0,
		0, 0,
		uintptr(width),
		uintptr(height),
		uintptr(flags),
	)
}

func (c *NativeWindow) SetMouseCursor(cursor MouseCursor) bool {
	var cursorID uintptr

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

	ret, _, _ := procSetCursor.Call(hCursor)
	return ret != 0
}
