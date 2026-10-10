package platforms

import (
	"image"
	"image/color"
)

// Window is a native window. All the windows live on the UI thread (the main
// OS thread): its event loop serves them all and calls every callback there,
// so callbacks of different windows never run concurrently. Call the methods
// on the UI thread too - from the callbacks, or via Post/RunOnUIThread from
// other goroutines. Update, Close and Exec may be called from any goroutine.
type Window interface {
	// Change window
	Show()                   // Shows the window, non-modal; returns immediately. Brings a hidden window back.
	Hide()                   // Hides the window until the next Show; it stays open
	ShowModal(parent Window) // Shows the window as a modal dialog owned by parent; returns immediately (blocks on macOS)
	Update()                 // Updates the window content
	Close() bool             // Closes the window
	Exec()                   // Waits for the window to close, running the event loop on the UI thread

	SystemHandle() any

	// Keyboard events
	OnKeyDown(func(keyCode Key, mods KeyModifiers) bool)
	OnKeyUp(func(keyCode Key, mods KeyModifiers))
	OnChar(func(char rune))

	// Mouse events
	OnMouseEnter(func())
	OnMouseLeave(func())
	OnMouseMove(func(x, y int))
	OnMouseButtonDown(func(btn MouseButton, x int, y int))
	OnMouseButtonUp(func(btn MouseButton, x int, y int))
	OnMouseButtonDblClick(func(btn MouseButton, x int, y int))
	OnMouseWheel(func(deltaX int, deltaY int))

	// Window events
	OnCreated(func())
	OnPaint(func(rgba *image.RGBA))
	OnMove(func(x, y int))
	OnResize(func(width, height int))
	OnCloseRequest(func() bool)
	OnTimer(func())
	// OnDeactivate is called when the window loses activation (keyboard
	// focus): the user switched to another window or application
	OnDeactivate(func())
	// OnActivate is called when the window gets activation (keyboard focus);
	// it may come more than once in a row
	OnActivate(func())
	// OnFilesDropped is called when files dragged from the system (e.g. from
	// the file manager) are dropped on the window, at (x, y) in client
	// coordinates
	OnFilesDropped(func(files []string, x, y int))

	// Window appearance
	SetTitle(title string)
	SetAppIcon(icon *image.RGBA)
	SetBackgroundColor(color color.RGBA)
	SetMouseCursor(cursor MouseCursor)
	// SetDarkMode switches the window frame (title bar) to its dark or light
	// look, to match the application's theme
	SetDarkMode(dark bool)

	Move(width int, height int)
	MoveToCenterOfScreen()
	Resize(width int, height int)
	// SetMinSize keeps the user from making the client area smaller than
	// width x height (logical pixels)
	SetMinSize(width int, height int)
	MinimizeWindow()
	MaximizeWindow()
	// RestoreWindow returns a maximized window to its normal size
	RestoreWindow()
	IsMaximized() bool
	// SetAlwaysOnTop keeps the window above the other windows
	SetAlwaysOnTop(onTop bool)
	// RequestAttention marks the window in the taskbar/dock until the user looks at it
	RequestAttention()
	// Beep plays the system alert sound
	Beep()

	SetAllowMinimize(allow bool)
	SetAllowMaximize(allow bool)

	// Get window information
	Size() (width, height int)
	Pos() (x, y int)
	PosX() int
	PosY() int
	Width() int
	Height() int
	KeyModifiers() KeyModifiers
	DrawTimeUs() int64
	// Scale returns how many pixels of the image given to OnPaint a logical
	// pixel takes: 1, or e.g. 2 on a Retina screen. The sizes, positions and
	// mouse coordinates are all logical.
	Scale() float64

	// ClientToScreen converts a point in the window's client area to screen coordinates
	ClientToScreen(x, y int) (screenX, screenY int)
	// ScreenWorkArea returns the usable area (without taskbar/dock) of the
	// monitor containing the screen point (x, y)
	ScreenWorkArea(x, y int) (areaX, areaY, areaWidth, areaHeight int)
}
