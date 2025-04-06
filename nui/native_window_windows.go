package nui

import (
	"fmt"
	"image"
	"image/color"
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

	mouseInside bool

	keyModifiers KeyModifiers

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
}

var hwnds map[syscall.Handle]*NativeWindow

func init() {
	hwnds = make(map[syscall.Handle]*NativeWindow)
}

/////////////////////////////////////////////////////
// Window creation and management

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

///////////////////////////////////////////////////
// Window appearance

func (c *NativeWindow) SetTitle(title string) {
	strPtr, _ := syscall.UTF16PtrFromString(title)
	procSetWindowTextW.Call(
		uintptr(c.hwnd),
		uintptr(unsafe.Pointer(strPtr)),
	)
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

func (c *NativeWindow) SetMouseCursor(cursor MouseCursor) {
	if c.currentCursor == cursor {
		return
	}
	c.currentCursor = cursor
	c.changeMouseCursor(cursor)
}

/////////////////////////////////////////////////////
// Window position and size

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

func (c *NativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := getScreenSize()
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

func (c *NativeWindow) MinimizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), SW_SHOWMINIMIZED)
}

func (c *NativeWindow) MaximizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), SW_SHOWMAXIMIZED)
}

//////////////////////////////////////////////////
// Window information

func (c *NativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
}

func (c *NativeWindow) Pos() (x, y int) {
	return c.windowPosX, c.windowPosY
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

func (c *NativeWindow) KeyModifiers() KeyModifiers {
	return c.keyModifiers
}

func (c *NativeWindow) DrawTimeUs() int64 {
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
