package platforms

import (
	"image"
	"image/color"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type windowId syscall.Handle

// Guards app.windows: multiple windows each run their own goroutine/OS
// thread and message loop (see Exec), so creating one window while
// another's wndProc looks up its handle is a real concurrent map access.
var appWindowsMu sync.Mutex

type nativeWindowPlatform struct {
	// Per-window paint surface and background color, sized to the window's
	// current dimensions. These used to be process-wide globals, which broke
	// as soon as a second window (e.g. a modal dialog) was created: every
	// createWindow() call reset the shared background to the default color,
	// clobbering whatever the first window had set via SetBackgroundColor.
	canvasBuffer []byte
	bgColor      color.RGBA

	// Scratch buffer for drawImageToHDC's RGBA->BGRA conversion, sized to
	// this window's own paint chunk. Used to be a process-wide global (see
	// its removal), which raced the same way canvasBuffer used to: each
	// window pumps its own message loop on its own locked OS thread (see
	// pumpMessages), so two windows' WM_PAINT handlers can run this
	// conversion concurrently and corrupt each other's pixels - visible as
	// one window's content bleeding into another's, most noticeably when a
	// second (e.g. modal) window appears.
	pixBuffer []byte

	// GetMessage/PeekMessage only ever deliver a window's messages to the OS
	// thread that created it (and PostQuitMessage only quits that same
	// thread's queue), so CreateWindowExW and the whole message loop must run
	// on one dedicated, locked OS thread for the window's entire life.
	// createWindow() spawns that thread and starts the pump on it right away
	// (see pumpMessages); Exec(), whichever goroutine calls it (Run,
	// ShowModal's own goroutine, etc.), just waits on pumpDone. Without this,
	// a second window's own PostQuitMessage could land on another window's
	// (e.g. the main window's) thread and close the whole app instead of
	// just itself.
	//
	// The pump must start immediately rather than wait for an explicit
	// signal from Exec(): Win32 cross-thread calls (SendMessageW,
	// SetWindowText, ShowWindow, ...) are only ever delivered while the
	// owning thread is blocked inside GetMessage/PeekMessage - a thread
	// merely parked on a Go channel receive is invisible to that mechanism.
	// SetAppIcon (called right after createWindow() returns, from the
	// caller's own goroutine) would otherwise deadlock forever waiting for a
	// pump that never starts.
	pumpDone chan struct{}
}

// ///////////////////////////////////////////////////
// Window creation and management

func createWindow(title string, posX int, posY int, width int, height int, center bool, maximized bool) *nativeWindow {
	created := make(chan *nativeWindow, 1)

	go func() {
		runtime.LockOSThread()

		var c nativeWindow
		c.dblClickTime = 300 * time.Millisecond
		c.showMaximized = maximized
		c.platform.pumpDone = make(chan struct{})

		// Create a unique class name
		dt := time.Now().Format("2006-01-02-15-04-05")
		randomNumber := rand.Intn(1024 * 1024)
		tempClassName := "WCL" + dt + strconv.Itoa(randomNumber)
		className, _ := syscall.UTF16PtrFromString(tempClassName)

		c.platform.bgColor = color.RGBA{0x1F, 0x1F, 0x1F, 255}

		// Set default window title
		windowTitle, _ := syscall.UTF16PtrFromString(title)

		// Set default cursor
		c.currentCursor = MouseCursorArrow

		// Get the instance handle
		hInstance, _, _ := procGetModuleHandleW.Call(0)

		// Register the window class
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

		windowFlags := uint32(c_WS_OVERLAPPEDWINDOW)
		if c.showMaximized {
			windowFlags |= c_WS_MAXIMIZE
		}

		// Create the window
		hwnd, _, _ := procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(windowTitle)),
			uintptr(windowFlags),
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

		created <- &c

		c.pumpMessages()
		close(c.platform.pumpDone)
	}()

	return <-created
}

func (c *nativeWindow) Show() {
	if c.showMaximized {
		procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWMAXIMIZED)
	} else {
		procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWDEFAULT)
	}
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	procUpdateWindow.Call(uintptr(c.hwnd))
}

func (c *nativeWindow) Update() {
	// Update the window
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	procUpdateWindow.Call(uintptr(c.hwnd))
}

// Exec blocks the calling goroutine until the window is closed. The actual
// GetMessage/DispatchMessage pump always runs on the dedicated OS thread
// that created c.hwnd (see nativeWindowPlatform) and is already running by
// the time this is called - so this just waits for it to finish.
func (c *nativeWindow) Exec() {
	<-c.platform.pumpDone
}

// pumpMessages runs the real GetMessage/DispatchMessage loop. It must only
// ever be called from the dedicated OS thread that created c.hwnd.
func (c *nativeWindow) pumpMessages() {
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
			//fmt.Println("Error:", e)
		}

		if ret == 0 {
			//fmt.Println("Exiting...")
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
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
// from any other goroutine is handed over to that thread (c_WM_NUI_CLOSE)
// instead of failing silently and leaving the window open.
func (c *nativeWindow) Close() bool {
	windowThread, _, _ := procGetWindowThreadProcessId.Call(uintptr(c.hwnd), 0)
	currentThread, _, _ := procGetCurrentThreadId.Call()
	if windowThread != currentThread {
		ok, _, _ := procPostMessageW.Call(uintptr(c.hwnd), c_WM_NUI_CLOSE, 0, 0)
		return ok != 0
	}
	procDestroyWindow.Call(uintptr(c.hwnd))
	return true
}

// ShowModal marks this window as owned by parent (GWLP_HWNDPARENT, so it
// stacks above parent, minimizes with it, and gets no separate taskbar
// button) and disables parent's HWND for the duration, the classic Win32
// technique dialogs use internally. It runs on its own goroutine/thread like
// a non-modal window, so parent's own event loop (repaint, timers) keeps
// running - matching the Linux contract documented on Window.ShowModal.
func (c *nativeWindow) ShowModal(parent Window) {
	var hwndOwner uintptr
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		hwndOwner = uintptr(p.hwnd)
	}

	if hwndOwner != 0 {
		procSetWindowLongPtrW.Call(uintptr(c.hwnd), gwlHwndParentIndex(), hwndOwner)
		procEnableWindow.Call(hwndOwner, 0)
	}

	c.Show()

	go func() {
		c.Exec()
		if hwndOwner != 0 {
			procEnableWindow.Call(hwndOwner, 1)
			procSetForegroundWindow.Call(hwndOwner)
		}
	}()
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
	return getModifierState()
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

func (c *nativeWindow) ClientToScreen(x, y int) (int, int) {
	pt := struct{ x, y int32 }{int32(x), int32(y)}
	procClientToScreen.Call(uintptr(c.hwnd), uintptr(unsafe.Pointer(&pt)))
	return int(pt.x), int(pt.y)
}

func (c *nativeWindow) ScreenWorkArea(x, y int) (int, int, int, int) {
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
