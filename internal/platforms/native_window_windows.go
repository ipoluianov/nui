package platforms

import (
	"image"
	"image/color"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type windowId syscall.Handle

// Guards app.windows. The windows all belong to the UI thread, but the map is
// also read from other goroutines (e.g. Close of a window's popup).
var appWindowsMu sync.Mutex

type nativeWindowPlatform struct {
	// Per-window paint surface and background color, sized to the window's
	// current dimensions. These used to be process-wide globals, which broke
	// as soon as a second window (e.g. a modal dialog) was created: every
	// createWindow() call reset the shared background to the default color,
	// clobbering whatever the first window had set via SetBackgroundColor.
	canvasBuffer []byte
	bgColor      color.RGBA

	// back is the DIB the frames are copied to the window from (see
	// backBuffer); repaintAt, if set, is when a frame that didn't get to the
	// window is painted again
	back      backBuffer
	repaintAt time.Time

	// icon is the HICON of SetAppIcon, destroyed when replaced
	icon uintptr

	// Every window is created on the UI thread (the main OS thread), so one
	// message loop there serves them all (see runLoopUntil) and every
	// callback runs on that thread. closed is set by WM_DESTROY; done is
	// closed then too, for Exec calls from other goroutines.
	closed bool
	done   chan struct{}

	// modalOwner is the window ShowModal disabled, enabled again when this
	// window closes (see releaseModalOwner)
	modalOwner uintptr

	// shownOnce is set by the first Show; a later Show brings a hidden window back
	shownOnce bool
}

// ///////////////////////////////////////////////////
// Window creation and management

// windowClassName is the class of all the windows; they all have the same
// window procedure, style and brush. (A class per window, with a unique
// name, used to be registered and never unregistered: a long-running
// application opening dialogs ran out of class atoms.)
const windowClassName = "NUIWindow"

var registerWindowClassOnce sync.Once

// registerWindowClass registers the class of the windows once
func registerWindowClass() (hInstance uintptr, className *uint16) {
	hInstance, _, _ = procGetModuleHandleW.Call(0)
	className, _ = syscall.UTF16PtrFromString(windowClassName)
	registerWindowClassOnce.Do(func() {
		wndClass := t_WNDCLASSEXW{
			cbSize:        uint32(unsafe.Sizeof(t_WNDCLASSEXW{})),
			style:         c_CS_OWNDC, /*| c_CS_DBLCLKS*/
			lpfnWndProc:   syscall.NewCallback(wndProc),
			hInstance:     syscall.Handle(hInstance),
			hCursor:       0,
			hbrBackground: 5,
			lpszClassName: className,
		}
		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))
	})
	return hInstance, className
}

// createWindow creates the window on the UI thread (see CreateWindow), which
// makes the window belong to it: the UI thread's message loop gets its
// messages and calls wndProc.
func createWindow(title string, posX int, posY int, width int, height int, center bool, maximized bool) *nativeWindow {
	var c nativeWindow
	c.dblClickTime = 300 * time.Millisecond
	c.showMaximized = maximized
	c.platform.done = make(chan struct{})

	c.platform.bgColor = color.RGBA{0x1F, 0x1F, 0x1F, 255}

	// Set default window title
	windowTitle, _ := syscall.UTF16PtrFromString(title)

	// Set default cursor
	c.currentCursor = MouseCursorArrow

	hInstance, className := registerWindowClass()

	windowFlags := uint32(c_WS_OVERLAPPEDWINDOW)
	if c.showMaximized {
		windowFlags |= c_WS_MAXIMIZE
	}

	// The size is logical: in pixels of the screen it opens on (the
	// primary one), corrected below if it opens on another
	c.scale = systemScale()

	// Create the window
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		uintptr(windowFlags),
		c_CW_USEDEFAULT,
		c_CW_USEDEFAULT,
		uintptr(c.toPhysical(width)),
		uintptr(c.toPhysical(height)),
		0,
		0,
		hInstance,
		0,
	)

	c.windowWidth = width
	c.windowHeight = height

	// Store the window handle
	c.hwnd = windowId(syscall.Handle(hwnd))
	if s := hwndScale(hwnd); s != c.scale {
		c.scale = s
		if !maximized {
			c.Resize(width, height)
		}
	}
	appWindowsMu.Lock()
	app.windows[c.hwnd] = &c
	appWindowsMu.Unlock()

	// Set default icon
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	c.SetAppIcon(icon)

	if center && !maximized {
		c.MoveToCenterOfScreen()
	}

	setDarkMode(hwnd, true)
	c.enableFileDrop()

	procSetTimer.Call(uintptr(c.hwnd), timerID1ms, 1, 0)
	registerSessionNotification(uintptr(c.hwnd))

	return &c
}

func (c *nativeWindow) Show() {
	if c.platform.shownOnce {
		procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOW)
		procSetForegroundWindow.Call(uintptr(c.hwnd))
		return
	}
	c.platform.shownOnce = true
	if c.showMaximized {
		procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWMAXIMIZED)
	} else {
		procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWDEFAULT)
	}
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	procUpdateWindow.Call(uintptr(c.hwnd))
}

// Hide hides the window and its taskbar button until the next Show. It
// stays open meanwhile.
func (c *nativeWindow) Hide() {
	procShowWindow.Call(uintptr(c.hwnd), c_SW_HIDE)
}

// Update paints the window. On the UI thread it paints at once; from another
// thread it only invalidates the window, and the UI thread paints it.
// (UpdateWindow there sends WM_PAINT to the UI thread and waits for it: a
// UI callback waiting for that goroutine meanwhile was a deadlock.)
func (c *nativeWindow) Update() {
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	if isUIThread() {
		procUpdateWindow.Call(uintptr(c.hwnd))
	}
}

// Exec waits until the window is closed. On the UI thread it runs the
// message loop meanwhile, serving all the windows; on another goroutine it
// just waits.
func (c *nativeWindow) Exec() {
	if !isUIThread() {
		<-c.platform.done
		return
	}
	runLoopUntil(func() bool { return c.platform.closed })
}

// Close destroys the native window via the real DestroyWindow API (not just
// a posted WM_DESTROY). DestroyWindow is what actually hides/frees the HWND
// and - critically for ShowModal - hands activation back to the owner window
// as part of its normal teardown. Posting WM_DESTROY by itself only ran our
// own cleanup handler for that message without ever destroying the window,
// so a "closed" modal left a dead but still-visible/owned HWND behind and
// the later SetForegroundWindow(owner) call was silently ignored by
// Windows' foreground-lock rules, dropping the whole app to the background.
//
// DestroyWindow only works on the thread that created the window, so a Close
// from any other goroutine is handed over to the UI thread (c_WM_NUI_CLOSE)
// instead of failing silently and leaving the window open.
func (c *nativeWindow) Close() bool {
	if !isUIThread() {
		ok, _, _ := procPostMessageW.Call(uintptr(c.hwnd), c_WM_NUI_CLOSE, 0, 0)
		return ok != 0
	}
	c.destroy()
	return true
}

// destroy destroys the window. UI thread only.
func (c *nativeWindow) destroy() {
	c.releaseModalOwner()
	procDestroyWindow.Call(uintptr(c.hwnd))
}

// releaseModalOwner enables the window ShowModal disabled and brings it to
// the front. Done before the dialog is destroyed, so Windows hands the
// activation over to the owner rather than to another application.
func (c *nativeWindow) releaseModalOwner() {
	owner := c.platform.modalOwner
	if owner == 0 {
		return
	}
	c.platform.modalOwner = 0
	procEnableWindow.Call(owner, 1)
	procSetForegroundWindow.Call(owner)
}

// ShowModal marks this window as owned by parent (GWLP_HWNDPARENT, so it
// stacks above parent, minimizes with it, and gets no separate taskbar
// button) and disables parent's HWND for the duration, the classic Win32
// technique dialogs use internally. It returns at once: the message loop
// serves the dialog like any other window, so parent keeps painting and
// running its timer - matching the Linux contract documented on
// Window.ShowModal. Closing the dialog enables parent again.
func (c *nativeWindow) ShowModal(parent Window) {
	var hwndOwner uintptr
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		hwndOwner = uintptr(p.hwnd)
	}

	if hwndOwner != 0 {
		procSetWindowLongPtrW.Call(uintptr(c.hwnd), gwlHwndParentIndex(), hwndOwner)
		procEnableWindow.Call(hwndOwner, 0)
		c.platform.modalOwner = hwndOwner
	}

	c.Show()
}

// gwlHwndParentIndex returns GWLP_HWNDPARENT (-8) sign-extended to uintptr.
// Converting the negative literal straight to uintptr is a compile error
// (Go constant-conversion rules reject negative-to-unsigned even when typed);
// routing it through an int32 function parameter forces a runtime
// conversion, which correctly sign-extends instead.
func gwlHwndParentIndex() uintptr {
	return gwlIndexToUintptr(-8)
}

func gwlIndexToUintptr(n int32) uintptr {
	return uintptr(n)
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
		//fmt.Println("failed to create icon")
		return
	}

	procSendMessageW.Call(uintptr(c.hwnd), c_WM_SETICON, c_ICON_BIG, uintptr(hIcon))
	procSendMessageW.Call(uintptr(c.hwnd), c_WM_SETICON, c_ICON_SMALL, uintptr(hIcon))
	// The window doesn't use the previous icon any more
	c.destroyIcon()
	c.platform.icon = uintptr(hIcon)
}

// destroyIcon frees the icon of the last SetAppIcon
func (c *nativeWindow) destroyIcon() {
	if c.platform.icon != 0 {
		procDestroyIcon.Call(c.platform.icon)
		c.platform.icon = 0
	}
}

func (c *nativeWindow) SetBackgroundColor(color color.RGBA) {
	c.platform.bgColor = color
	c.Update()
}

func (c *nativeWindow) SetMouseCursor(cursor MouseCursor) {
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
		uintptr(c.toPhysical(x)), uintptr(c.toPhysical(y)),
		0, 0,
		uintptr(flags),
	)
}

func (c *nativeWindow) MoveToCenterOfScreen() {
	// In physical pixels, as the screen's size
	screenWidth, screenHeight := getScreenSize()
	windowWidth, windowHeight := c.toPhysical(c.windowWidth), c.toPhysical(c.windowHeight)
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	procSetWindowPos.Call(uintptr(c.hwnd), 0, uintptr(x), uintptr(y), 0, 0, c_SWP_NOSIZE|c_SWP_NOZORDER)
}

func (c *nativeWindow) Resize(width, height int) {
	flags := c_SWP_NOMOVE | c_SWP_NOZORDER

	procSetWindowPos.Call(
		uintptr(c.hwnd),
		0,
		0, 0,
		uintptr(c.toPhysical(width)),
		uintptr(c.toPhysical(height)),
		uintptr(flags),
	)
}

func (c *nativeWindow) MinimizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWMINIMIZED)
}

func (c *nativeWindow) MaximizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWMAXIMIZED)
}

func (c *nativeWindow) RestoreWindow() {
	procShowWindow.Call(uintptr(c.hwnd), c_SW_RESTORE)
}

func (c *nativeWindow) SetDarkMode(dark bool) {
	setDarkMode(uintptr(c.hwnd), dark)
	// Redraw the frame: the title bar doesn't pick up the change by itself
	// until the window is activated again
	flags := c_SWP_NOMOVE | c_SWP_NOSIZE | c_SWP_NOZORDER | c_SWP_NOACTIVATE | c_SWP_FRAMECHANGED
	procSetWindowPos.Call(uintptr(c.hwnd), 0, 0, 0, 0, 0, uintptr(flags))
}

func (c *nativeWindow) SetAlwaysOnTop(onTop bool) {
	insertAfter := ^uintptr(1) // HWND_NOTOPMOST (-2)
	if onTop {
		insertAfter = ^uintptr(0) // HWND_TOPMOST (-1)
	}
	procSetWindowPos.Call(uintptr(c.hwnd), insertAfter, 0, 0, 0, 0,
		uintptr(c_SWP_NOMOVE|c_SWP_NOSIZE|c_SWP_NOACTIVATE))
}

// RequestAttention flashes the taskbar button until the window comes to the foreground
func (c *nativeWindow) RequestAttention() {
	const flashwTray, flashwTimerNoFG = 0x2, 0xC
	info := struct {
		cbSize    uint32
		hwnd      uintptr
		dwFlags   uint32
		uCount    uint32
		dwTimeout uint32
	}{hwnd: uintptr(c.hwnd), dwFlags: flashwTray | flashwTimerNoFG}
	info.cbSize = uint32(unsafe.Sizeof(info))
	procFlashWindowEx.Call(uintptr(unsafe.Pointer(&info)))
}

func (c *nativeWindow) Beep() {
	const mbIconExclamation = 0x30
	procMessageBeep.Call(mbIconExclamation)
}

// SetAllowMinimize shows or hides the titlebar's minimize button by toggling
// WS_MINIMIZEBOX, e.g. for dialog-style windows that shouldn't offer it.
func (c *nativeWindow) SetAllowMinimize(allow bool) {
	c.toggleWindowStyleBit(c_WS_MINIMIZEBOX, allow)
}

// SetAllowMaximize shows or hides the titlebar's maximize button by toggling
// WS_MAXIMIZEBOX, e.g. for dialog-style windows that shouldn't offer it.
func (c *nativeWindow) SetAllowMaximize(allow bool) {
	c.toggleWindowStyleBit(c_WS_MAXIMIZEBOX, allow)
}

// toggleWindowStyleBit flips a GWL_STYLE bit and asks the non-client frame
// (titlebar/buttons) to redraw so the change is visible immediately.
func (c *nativeWindow) toggleWindowStyleBit(bit uint32, set bool) {
	style, _, _ := procGetWindowLongPtrW.Call(uintptr(c.hwnd), gwlStyleIndex())
	newStyle := uint32(style)
	if set {
		newStyle |= bit
	} else {
		newStyle &^= bit
	}
	procSetWindowLongPtrW.Call(uintptr(c.hwnd), gwlStyleIndex(), uintptr(newStyle))

	flags := c_SWP_NOMOVE | c_SWP_NOSIZE | c_SWP_NOZORDER | c_SWP_NOACTIVATE | c_SWP_FRAMECHANGED
	procSetWindowPos.Call(uintptr(c.hwnd), 0, 0, 0, 0, 0, uintptr(flags))
}

// gwlStyleIndex returns GWL_STYLE (-16) sign-extended to uintptr; see
// gwlHwndParentIndex for why this needs the int32-parameter indirection.
func gwlStyleIndex() uintptr {
	return gwlIndexToUintptr(-16)
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

func (c *nativeWindow) IsMaximized() bool {
	ret, _, _ := procIsZoomed.Call(uintptr(c.hwnd))
	return ret != 0
}

func (c *nativeWindow) KeyModifiers() KeyModifiers {
	// GetAsyncKeyState reports the keys held in any application
	if active, _, _ := procGetForegroundWindow.Call(); active != uintptr(c.hwnd) {
		return KeyModifiers{}
	}
	return currentModifierState()
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

func (c *nativeWindow) SystemHandle() any {
	return syscall.Handle(c.hwnd)
}

// ClientToScreen: the screen coordinates are logical too, in the window's
// scale (see dpi_windows.go)
func (c *nativeWindow) ClientToScreen(x, y int) (int, int) {
	pt := struct{ x, y int32 }{int32(c.toPhysical(x)), int32(c.toPhysical(y))}
	procClientToScreen.Call(uintptr(c.hwnd), uintptr(unsafe.Pointer(&pt)))
	return c.toLogicalScreen(int(pt.x)), c.toLogicalScreen(int(pt.y))
}

func (c *nativeWindow) ScreenWorkArea(x, y int) (int, int, int, int) {
	areaX, areaY, areaW, areaH := c.screenWorkAreaPhysical(c.toPhysical(x), c.toPhysical(y))
	return c.toLogicalScreen(areaX), c.toLogicalScreen(areaY), c.toLogical(areaW), c.toLogical(areaH)
}

func (c *nativeWindow) screenWorkAreaPhysical(x, y int) (int, int, int, int) {
	// MonitorFromRect instead of MonitorFromPoint: POINT is passed by value
	// there, which doesn't map onto syscall's uintptr arguments portably.
	r := rect{int32(x), int32(y), int32(x + 1), int32(y + 1)}
	hMonitor, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&r)), c_MONITOR_DEFAULTTONEAREST)

	mi := t_MONITORINFO{cbSize: uint32(unsafe.Sizeof(t_MONITORINFO{}))}
	ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
	if ret == 0 {
		w, h := getScreenSize()
		return 0, 0, w, h
	}
	work := mi.rcWork
	return int(work.left), int(work.top), int(work.right - work.left), int(work.bottom - work.top)
}
