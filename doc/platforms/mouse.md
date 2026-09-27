# Mouse (`platforms`)

## Types

```go
type MouseButton int
const (
	MouseButtonLeft   MouseButton = 0
	MouseButtonMiddle MouseButton = 1
	MouseButtonRight  MouseButton = 2
)
btn.String() string // "Left", "Middle", "Right"

type MouseCursor int
const (
	MouseCursorNotDefined MouseCursor = 0
	MouseCursorArrow      MouseCursor = 1
	MouseCursorPointer    MouseCursor = 2
	MouseCursorResizeHor  MouseCursor = 3
	MouseCursorResizeVer  MouseCursor = 4
	MouseCursorIBeam      MouseCursor = 5
)
```

## Usage

```go
win.OnMouseMove(func(x, y int) {})
win.OnMouseButtonDown(func(btn platforms.MouseButton, x, y int) {})
win.OnMouseButtonUp(func(btn platforms.MouseButton, x, y int) {})
win.OnMouseButtonDblClick(func(btn platforms.MouseButton, x, y int) {})
win.OnMouseWheel(func(deltaX, deltaY int) {})
win.OnMouseEnter(func() {})
win.OnMouseLeave(func() {})

win.SetMouseCursor(platforms.MouseCursorPointer)
```

`x, y` are client-area pixel coordinates (origin top-left). Double-click detection is built in (time + distance threshold) — you get both `OnMouseButtonDown` and `OnMouseButtonDblClick` on the second click.
