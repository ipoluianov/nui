package nui

type MouseCursor int

const (
	MouseCursorNotDefined MouseCursor = 0
	MouseCursorArrow      MouseCursor = 1
	MouseCursorPointer    MouseCursor = 2
	MouseCursorResizeHor  MouseCursor = 3
	MouseCursorResizeVer  MouseCursor = 4
	MouseCursorIBeam      MouseCursor = 5
)
