package nui

import (
	_ "embed"
	"runtime"
)

func init() {
	// Lock the OS thread to prevent it from being moved to another thread
	// This is important for GUI applications to ensure that the GUI
	// is always run on the same thread
	runtime.LockOSThread()
}

/*
// Keyboard events
OnKeyDown func(keyCode Key)
OnKeyUp   func(keyCode Key)
OnChar    func(char rune)

// Mouse events
OnMouseEnter                   func()
OnMouseLeave                   func()
OnMouseMove                    func(x, y int)
OnMouseButtonDown          	   func(btn MouseButton, x int, y int)
OnMouseButtonUp            	   func(btn MouseButton, x int, y int)
OnMouseDoubleClick             func(btn MouseButton, x int, y int)
OnMouseWheel                   func(deltaX float64, deltaY float64)

// Window events
OnCreated      func()
OnPaint        func(rgba *image.RGBA)
OnMove         func(x, y int)
OnResize       func(width, height int)
OnCloseRequest func() bool
OnTimer        func()

// Change window
CreateWindow()
Show()
Update()
EventLoop()
Close()

// Window appearance
SetTitle(title string)
SetAppIcon(icon *image.RGBA)
SetMouseCursor(cursor MouseCursor)

Move(width int, height int)
MoveToCenterOfScreen()
Resize(width int, height int)
MinimizeWindow()
MaximizeWindow()

// Get window information
Size() (width, height int)
Pos() (x, y int)
PosX() int
PosY() int
Width() int
Height() int
DrawTimeUs() int64

*/

const (
	DefaultWindowTitle = "NUI Window"
)

type MouseCursor int

const (
	MouseCursorNotDefined MouseCursor = 0
	MouseCursorArrow      MouseCursor = 1
	MouseCursorPointer    MouseCursor = 2
	MouseCursorResizeHor  MouseCursor = 3
	MouseCursorResizeVer  MouseCursor = 4
	MouseCursorIBeam      MouseCursor = 5
)

type MouseButton int

const (
	MouseButtonLeft   MouseButton = 0
	MouseButtonMiddle MouseButton = 1
	MouseButtonRight  MouseButton = 2
)

func (m MouseButton) String() string {
	switch m {
	case MouseButtonLeft:
		return "Left"
	case MouseButtonMiddle:
		return "Middle"
	case MouseButtonRight:
		return "Right"
	}
	return "Unknown"
}
