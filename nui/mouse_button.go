package nui

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
