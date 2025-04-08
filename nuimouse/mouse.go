package nuimouse

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
