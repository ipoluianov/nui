package nui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

type NativeWindow struct {
	hwnd syscall.Handle

	currentCursor MouseCursor
	lastSetCursor MouseCursor

	keyModifiers KeyModifiers

	windowPosX   int
	windowPosY   int
	windowWidth  int
	windowHeight int

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
	//procGetDeviceCaps = gdi32.NewProc("GetDeviceCaps")
	//procGetObjectType = gdi32.NewProc("GetObjectType")
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

func GetRGBATestImage() *image.RGBA {
	rgba, err := loadPngFromBytes(TestPng)
	if err != nil {
		panic(err)
	}
	return rgba
}

func getHDCSize(hdc uintptr) (width int32, height int32) {
	var r rect
	procGetClipBox.Call(hdc, uintptr(unsafe.Pointer(&r)))
	return r.right - r.left, r.bottom - r.top
}

const chunkHeight = 100
const maxWidth = 10000

var pixBuffer = make([]byte, 4*chunkHeight*maxWidth)

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

		for row := 0; row < h; row++ {
			srcOffset := (y + row) * imgStride
			dstOffset := row * int(width) * 4

			for col := 0; col < int(width); col++ {
				r := img.Pix[srcOffset+col*4+0]
				g := img.Pix[srcOffset+col*4+1]
				b := img.Pix[srcOffset+col*4+2]
				//a := img.Pix[srcOffset+col*4+3]

				pixBuffer[dstOffset+col*4+0] = b // Blue
				pixBuffer[dstOffset+col*4+1] = g // Green
				pixBuffer[dstOffset+col*4+2] = r // Red
				pixBuffer[dstOffset+col*4+3] = 255
			}
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

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	//fmt.Println("Message:", native.MessageName(msg))

	win := GetNativeWindowByHandle(hwnd)

	switch msg {
	case WM_PAINT:

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

		if !mouseInside {
			mouseInside = true
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
		mouseInside = false
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

///////////////////////////////////////////////////////////////////

func CreateWindow() *NativeWindow {
	var c NativeWindow

	// Create a unique class name
	dt := time.Now().Format("2006-01-02-15-04-05")
	randomNumber := rand.Intn(1024 * 1024)
	tempClassName := "WCL" + dt + strconv.Itoa(randomNumber)
	className, _ := syscall.UTF16PtrFromString(tempClassName)

	initCanvasBufferBackground(color.RGBA{0, 50, 0, 255})

	// Set default window title
	windowTitle, _ := syscall.UTF16PtrFromString(DefaultWindowTitle)

	// Set default cursor
	c.currentCursor = MouseCursorArrow

	// Get the instance handle
	hInstance, _, _ := procGetModuleHandleW.Call(0)

	// Register the window class
	wndClass := WNDCLASSEXW{
		cbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		style:         CS_OWNDC | CS_DBLCLKS,
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     syscall.Handle(hInstance),
		hCursor:       0,
		hbrBackground: 5,
		lpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))

	// Create the window
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		WS_OVERLAPPEDWINDOW,
		CW_USEDEFAULT,
		CW_USEDEFAULT,
		640,
		480,
		0,
		0,
		hInstance,
		0,
	)

	// Store the window handle
	c.hwnd = syscall.Handle(hwnd)
	hwnds[c.hwnd] = &c

	// Set default icon
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	c.SetAppIcon(icon)

	return &c
}

func (c *NativeWindow) Show() {
	// Show the window
	procShowWindow.Call(uintptr(c.hwnd), SW_SHOWDEFAULT)
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	procUpdateWindow.Call(uintptr(c.hwnd))
}

func (c *NativeWindow) Hide() {
	// Hide the window
	procShowWindow.Call(uintptr(c.hwnd), SW_HIDE)
}

func (c *NativeWindow) Update() {
	// Update the window
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	procUpdateWindow.Call(uintptr(c.hwnd))
}

func (c *NativeWindow) EventLoop() {
	var msg MSG

	procSetTimer.Call(
		uintptr(c.hwnd),
		timerID1ms,
		1,
		0,
	)

	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	for {
		ret, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		e := err.(syscall.Errno)
		if e != 0 {
			fmt.Println("Error:", e)
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

func GetScreenSize() (width, height int) {
	w, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	h, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	return int(w), int(h)
}

func (c *NativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := GetScreenSize()
	windowWidth, windowHeight := c.Size()
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	c.Move(int(x), int(y))
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

func (c *NativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
}

func (c *NativeWindow) Position() (x, y int) {
	return c.windowPosX, c.windowPosY
}

func (c *NativeWindow) SetMouseCursor(cursor MouseCursor) {
	if c.currentCursor == cursor {
		return
	}
	c.currentCursor = cursor
	c.changeMouseCursor(cursor)
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

func (c *NativeWindow) MinimizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), SW_SHOWMINIMIZED)
}

func (c *NativeWindow) MaximizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), SW_SHOWMAXIMIZED)
}

func (c *NativeWindow) RestoreWindow() {
	procShowWindow.Call(uintptr(c.hwnd), SW_RESTORE)
}

func (c *NativeWindow) PosX() int {
	return c.windowPosX
}

func (c *NativeWindow) PosY() int {
	return c.windowPosY
}

func (c *NativeWindow) Width() int {
	return c.windowWidth
}

func (c *NativeWindow) Height() int {
	return c.windowHeight
}

func createHICONFromRGBA(img *image.RGBA) syscall.Handle {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	// В Windows иконки идут снизу вверх — инвертируем
	pixels := make([]byte, 0, width*height*4)
	for y := height - 1; y >= 0; y-- {
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
	}

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

func (c *NativeWindow) SetAppIcon(icon *image.RGBA) {
	hIcon := createHICONFromRGBA(icon)
	if hIcon == 0 {
		fmt.Println("failed to create icon")
		return
	}

	procSendMessageW.Call(uintptr(c.hwnd), WM_SETICON, ICON_BIG, uintptr(hIcon))
	procSendMessageW.Call(uintptr(c.hwnd), WM_SETICON, ICON_SMALL, uintptr(hIcon))
}

func (c *NativeWindow) KeyModifiers() KeyModifiers {
	return c.keyModifiers
}
