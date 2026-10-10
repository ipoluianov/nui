package platforms

import (
	"image"
	"image/color"
	"image/draw"
	"sync/atomic"
	"time"

	"github.com/ebitengine/purego/objc"
)

// Cocoa has one shared NSApplication run loop for the whole process; only the
// first window to reach Exec() may start it.
var eventLoopStarted int32

// Darwin/Cocoa implementation; window chrome and bridges live in cocoa_darwin.go.

type windowId int
type nativeWindowPlatform struct {
	lastCapsLockState bool
	lastNumLockState  bool

	keyModifiers KeyModifiers

	// Throttles go_on_timer per window; must not be shared across windows.
	lastTimerTick time.Time

	// bgColor clears the frame before onPaint
	bgColor color.RGBA
}

/*type NativeWindow struct {
	hwnd int

	currentCursor MouseCursor
	lastSetCursor MouseCursor

	windowPosX   int
	windowPosY   int
	windowWidth  int
	windowHeight int

	keyModifiers KeyModifiers

	lastCapsLockState bool
	lastNumLockState  bool

	// Keyboard events
	OnKeyDown func(keyCode Key, modifiers KeyModifiers)
	OnKeyUp   func(keyCode Key, modifiers KeyModifiers)
	OnChar    func(char rune)

	drawTimes      [32]int64
	drawTimesIndex int

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
}*/

var hwnds map[windowId]*nativeWindow

// The first window created is treated as the app's main window: closing it quits the
// whole app immediately, instead of macOS's default of waiting for every window to close.
var mainWindowID windowId
var mainWindowIDSet bool

func init() {
	hwnds = make(map[windowId]*nativeWindow)
}

/////////////////////////////////////////////////////
// Window creation and management

// width, height: client/content size (setContentSize), same units as OnResize on this platform.
func createWindow(title string, posX int, posY int, width int, height int, center bool, maximized bool) *nativeWindow {
	var c nativeWindow

	c.showMaximized = maximized

	c.platform.bgColor = color.RGBA{0, 50, 0, 255}

	c.hwnd = initWindow()
	// Register before Resize so ObjC-triggered go_on_resize reaches Go with a populated hwnds map.
	hwnds[c.hwnd] = &c
	if !mainWindowIDSet {
		mainWindowIDSet = true
		mainWindowID = c.hwnd
	}

	c.Resize(width, height)
	c.windowWidth = int(width)
	c.windowHeight = int(height)
	// Re-read in case AppKit adjusted the content rect by a pixel.
	if w, h := c.requestWindowSize(); w >= 0 && h >= 0 {
		c.windowWidth, c.windowHeight = w, h
	}

	c.SetTitle(title)

	if c.showMaximized {
		c.MaximizeWindow()
	} else if center {
		c.MoveToCenterOfScreen()
	} else {
		c.Move(posX, posY)
	}

	c.windowPosX, c.windowPosY = c.requestWindowPosition()
	c.startTimer(1)
	return &c
}

func (c *nativeWindow) Show() {
	showWindow(c.hwnd)
}

// Hide orders the window out until the next Show. It stays open meanwhile.
func (c *nativeWindow) Hide() {
	hideWindow(c.hwnd)
}

// Update and Close may be called from any goroutine (see Window): AppKit
// and the window maps belong to the main thread, so from another one they
// run there, without waiting (a wait could deadlock with a main thread
// waiting for the caller).
func (c *nativeWindow) Update() {
	onMainThread(func() { updateWindow(c.hwnd) })
}

// Exec runs the ONE shared NSApplication run loop for the whole process, not
// just this window's: the first caller starts and blocks on it for as long
// as the app runs; later callers just return immediately since that shared
// loop already services their window too.
func (c *nativeWindow) Exec() {
	if atomic.CompareAndSwapInt32(&eventLoopStarted, 0, 1) {
		runEventLoop()
	}
}

func (c *nativeWindow) Close() bool {
	onMainThread(func() { closeWindowById(c.hwnd) })
	return true
}

// onMainThread runs f at once on the main thread, later from another one
func onMainThread(f func()) {
	if isMainThread() {
		f()
		return
	}
	dispatchAsyncMain(f)
}

// ShowModal shows the window as an app-modal dialog. Unlike Linux/Windows this call
// blocks until the dialog closes (required for reliable modal-on-modal nesting on
// Cocoa); parent's timers/repaint keep running since they share the same run loop.
func (c *nativeWindow) ShowModal(parent Window) {
	parentID := windowId(-1)
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		parentID = p.hwnd
	}
	showModalWindow(c.hwnd, parentID)
}

///////////////////////////////////////////////////
// Window appearance

func (c *nativeWindow) SetTitle(title string) {
	setWindowTitle(c.hwnd, title)
}

func (c *nativeWindow) SetAppIcon(icon *image.RGBA) {
	bounds := icon.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, icon, bounds.Min, draw.Src)

	setAppIconFromRGBA(rgba.Pix, width, height)
}

func (c *nativeWindow) SetBackgroundColor(color color.RGBA) {
	c.platform.bgColor = color
	c.Update()
}

func (c *nativeWindow) SetMouseCursor(cursor MouseCursor) {
	c.currentCursor = cursor
	c.macSetMouseCursor(c.currentCursor)
}

// Maps MouseCursor kinds to IDs consumed by cocoa_darwin.go's setMacCursor.
func (c *nativeWindow) macSetMouseCursor(cursor MouseCursor) {
	if c.lastSetCursor == cursor {
		return
	}
	c.lastSetCursor = cursor
	setMacCursor(macCursorType(c.currentCursor))
}

// macCursorType maps the cursor kind to setMacCursor's cursor type.
func macCursorType(cursor MouseCursor) int {
	switch cursor {
	case MouseCursorArrow:
		return 1
	case MouseCursorPointer:
		return 2
	case MouseCursorResizeHor:
		return 3
	case MouseCursorResizeVer:
		return 4
	case MouseCursorIBeam:
		return 5
	}
	return 0
}

/////////////////////////////////////////////////////
// Window position and size

func (c *nativeWindow) Move(x, y int) {
	setWindowPosition(c.hwnd, x, y)
}

// Uses Size() client dimensions vs main screen frame (aligned with Windows centering heuristic).
func (c *nativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := GetScreenSize()
	windowWidth, windowHeight := c.Size()
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	c.Move(int(x), int(y))
}

func (c *nativeWindow) Resize(width, height int) {
	setWindowSize(c.hwnd, width, height) // NSWindow setContentSize
}

// SetMinSize keeps the user from making the content of the window smaller
// than width x height (NSWindow setContentMinSize)
func (c *nativeWindow) SetMinSize(width, height int) {
	setWindowMinSize(c.hwnd, width, height)
}

func (c *nativeWindow) MinimizeWindow() {
	minimizeWindow(c.hwnd)
}

func (c *nativeWindow) MaximizeWindow() {
	maximizeWindow(c.hwnd)
}

func (c *nativeWindow) RestoreWindow() {
	restoreWindow(c.hwnd)
}

func (c *nativeWindow) SetDarkMode(dark bool) {
	setWindowDarkMode(c.hwnd, dark)
}

func (c *nativeWindow) SetAlwaysOnTop(onTop bool) {
	setWindowAlwaysOnTop(c.hwnd, onTop)
}

func (c *nativeWindow) RequestAttention() {
	requestUserAttention()
}

func (c *nativeWindow) Beep() {
	systemBeep()
}

// SetAllowMinimize shows or hides the titlebar's miniaturize button, e.g. for
// dialog-style windows that shouldn't offer it.
func (c *nativeWindow) SetAllowMinimize(allow bool) {
	setWindowAllowMinimize(c.hwnd, allow)
}

// SetAllowMaximize shows or hides the titlebar's zoom button, e.g. for
// dialog-style windows that shouldn't offer it.
func (c *nativeWindow) SetAllowMaximize(allow bool) {
	setWindowAllowMaximize(c.hwnd, allow)
}

//////////////////////////////////////////////////
// Window information

// Client-area size; updated from go_on_resize (contentLayoutRect).
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
	return isWindowZoomed(c.hwnd)
}

// KeyModifiers is the state of the keys right now: [NSEvent modifierFlags]
func (c *nativeWindow) KeyModifiers() KeyModifiers {
	// modifierFlags reports the keys held in any application
	if !isKeyWindow(c.hwnd) {
		return KeyModifiers{}
	}
	flags := objc.Send[uint64](objc.ID(objc.GetClass("NSEvent")), objc.RegisterName("modifierFlags"))
	return KeyModifiers{
		Shift: flags&nsEventModifierFlagShift != 0,
		Ctrl:  flags&nsEventModifierFlagControl != 0,
		Alt:   flags&nsEventModifierFlagOption != 0,
		Cmd:   flags&nsEventModifierFlagCommand != 0,
	}
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
	return nil
}

func (c *nativeWindow) ClientToScreen(x, y int) (int, int) {
	var originX, originY int
	runOnMainSync(func() {
		originX, originY = getClientOrigin(c.hwnd)
	})
	return originX + x, originY + y
}

func (c *nativeWindow) ScreenWorkArea(x, y int) (areaX, areaY, areaW, areaH int) {
	runOnMainSync(func() {
		areaX, areaY, areaW, areaH = getScreenWorkArea(x, y)
	})
	return
}
