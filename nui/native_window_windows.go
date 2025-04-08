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

	"github.com/ipoluianov/nui/nuikey"
	"github.com/ipoluianov/nui/nuimouse"
)

type windowId syscall.Handle

type nativeWindowPlatform struct {
}

// ///////////////////////////////////////////////////
// Window creation and management

func createWindow(title string, width int, height int, center bool) *nativeWindow {
	var c nativeWindow

	// Create a unique class name
	dt := time.Now().Format("2006-01-02-15-04-05")
	randomNumber := rand.Intn(1024 * 1024)
	tempClassName := "WCL" + dt + strconv.Itoa(randomNumber)
	className, _ := syscall.UTF16PtrFromString(tempClassName)

	initCanvasBufferBackground(color.RGBA{0x1F, 0x1F, 0x1F, 255})

	// Set default window title
	windowTitle, _ := syscall.UTF16PtrFromString(title)

	// Set default cursor
	c.currentCursor = nuimouse.MouseCursorArrow

	// Get the instance handle
	hInstance, _, _ := procGetModuleHandleW.Call(0)

	// Register the window class
	wndClass := t_WNDCLASSEXW{
		cbSize:        uint32(unsafe.Sizeof(t_WNDCLASSEXW{})),
		style:         c_CS_OWNDC | c_CS_DBLCLKS,
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
		c_WS_OVERLAPPEDWINDOW,
		c_CW_USEDEFAULT,
		c_CW_USEDEFAULT,
		uintptr(width),
		uintptr(height),
		0,
		0,
		hInstance,
		0,
	)

	c.windowWidth = width
	c.windowHeight = height

	// Store the window handle
	c.hwnd = windowId(syscall.Handle(hwnd))
	app.windows[c.hwnd] = &c

	// Set default icon
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	c.SetAppIcon(icon)

	if center {
		c.MoveToCenterOfScreen()
	}

	setDarkMode(hwnd, true)

	return &c
}

func (c *nativeWindow) Show() {
	// Show the window
	procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWDEFAULT)
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	procUpdateWindow.Call(uintptr(c.hwnd))
}

func (c *nativeWindow) Update() {
	// Update the window
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	procUpdateWindow.Call(uintptr(c.hwnd))
}

func (c *nativeWindow) EventLoop() {
	var msg t_MSG

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

func (c *nativeWindow) Close() {
	procPostMessageW.Call(uintptr(c.hwnd), c_WM_DESTROY, 0, 0)
}

///////////////////////////////////////////////////
// Window appearance

func (c *nativeWindow) SetTitle(title string) {
	strPtr, _ := syscall.UTF16PtrFromString(title)
	procSetWindowTextW.Call(
		uintptr(c.hwnd),
		uintptr(unsafe.Pointer(strPtr)),
	)
}

func (c *nativeWindow) SetAppIcon(icon *image.RGBA) {
	hIcon := createHICONFromRGBA(icon)
	if hIcon == 0 {
		fmt.Println("failed to create icon")
		return
	}

	procSendMessageW.Call(uintptr(c.hwnd), c_WM_SETICON, c_ICON_BIG, uintptr(hIcon))
	procSendMessageW.Call(uintptr(c.hwnd), c_WM_SETICON, c_ICON_SMALL, uintptr(hIcon))
}

func (c *nativeWindow) SetBackgroundColor(color color.RGBA) {
	initCanvasBufferBackground(color)
	c.Update()
}

func (c *nativeWindow) SetMouseCursor(cursor nuimouse.MouseCursor) {
	if c.currentCursor == cursor {
		return
	}
	c.currentCursor = cursor
	c.changeMouseCursor(cursor)
}

/////////////////////////////////////////////////////
// Window position and size

func (c *nativeWindow) Move(x, y int) {
	flags := c_SWP_NOSIZE | c_SWP_NOZORDER

	procSetWindowPos.Call(
		uintptr(c.hwnd),
		0,
		uintptr(x), uintptr(y),
		0, 0,
		uintptr(flags),
	)
}

func (c *nativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := getScreenSize()
	windowWidth, windowHeight := c.Size()
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	c.Move(int(x), int(y))
}

func (c *nativeWindow) Resize(width, height int) {
	flags := c_SWP_NOMOVE | c_SWP_NOZORDER

	procSetWindowPos.Call(
		uintptr(c.hwnd),
		0,
		0, 0,
		uintptr(width),
		uintptr(height),
		uintptr(flags),
	)
}

func (c *nativeWindow) MinimizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWMINIMIZED)
}

func (c *nativeWindow) MaximizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWMAXIMIZED)
}

//////////////////////////////////////////////////
// Window information

func (c *nativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
}

func (c *nativeWindow) Pos() (x, y int) {
	return c.windowPosX, c.windowPosY
}

func (c *nativeWindow) PosX() int {
	return c.windowPosX
}

func (c *nativeWindow) PosY() int {
	return c.windowPosY
}

func (c *nativeWindow) Width() int {
	return c.windowWidth
}

func (c *nativeWindow) Height() int {
	return c.windowHeight
}

func (c *nativeWindow) KeyModifiers() nuikey.KeyModifiers {
	return c.keyModifiers
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
