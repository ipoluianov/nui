package nui

import "runtime"

func init() {
	runtime.LockOSThread()
}

/*
----------------------------------------------------

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
Hide()
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
