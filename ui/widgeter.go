package ui

import ()

type Widgeter interface {
	Id() string
	ParentWidgetId() string
	SetParentWidgetId(id string)
	setId(id string)
	attachToForm(self Widgeter, f *Form)
	FullPath() []string
	TypeName() string
	Name() string
	X() int
	Y() int
	Width() int
	Height() int
	InnerWidth() int
	InnerHeight() int
	Form() *Form

	SetProp(key string, value interface{})
	SetPropFunction(key string, f func())
	GetProp(key string) interface{}
	GetPropString(key string, defaultValue string) string
	GetPropInt(key string, defaultValue int) int
	GetPropBool(key string, defaultValue bool) bool
	GetPropFloat64(key string, defaultValue float64) float64

	SetCloseByClickOutside(close bool)
	CloseByClickOutside() bool

	Widgets() []Widgeter
	applyThemeMetrics()
	applyLanguage()

	Elevation() int

	SetRole(role string)
	Role() string

	FindWidgetByName(name string) Widgeter

	Enabled() bool
	SetEnabled(enabled bool)

	AddTimer(duration int, callback func())

	SetName(name string)
	SetPosition(x, y int)
	SetSize(width, height int)
	//SetAnchors(left, top, right, bottom bool)

	getWidgetAt(x, y int) Widgeter
	findWidgetAt(x, y int) Widgeter

	Focus()
	ClearFocus()

	ProcessPaint(cnv *Canvas)
	ProcessMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool
	ProcessMouseUp(button MouseButton, x int, y int, mods KeyModifiers, onlyForWidgetId string) bool
	ProcessMouseMove(x int, y int, mods KeyModifiers) bool
	ProcessMouseLeave() bool
	ProcessMouseEnter() bool
	ProcessKeyDown(keyCode Key, mods KeyModifiers) bool
	ProcessKeyUp(keyCode Key, mods KeyModifiers) bool
	ProcessMouseDblClick(button MouseButton, x int, y int, mods KeyModifiers) bool
	ProcessChar(char rune, mods KeyModifiers) bool
	ProcessMouseWheel(deltaX int, deltaY int) bool
	ProcessTimer()
	ProcessFocused()
	ProcessFocusLost()
	ProcessPropChange(key string, value interface{})

	ProcessClosePopup()

	updateLayout(oldWidth, oldHeight, newWidth, newHeight int)

	RectClientAreaOnWindow() (x, y int)

	ScrollX() int
	ScrollY() int

	SetMouseCursor(cursor MouseCursor)
	MouseCursor() MouseCursor

	// Anchors() (left, top, right, bottom bool)

	AddWidget(gridX, gridY int, widget Widgeter)
	RemoveWidget(widget Widgeter)

	AllChildren() []Widgeter
	IsCanBeFocused() bool

	IsVisible() bool

	GridX() int
	GridY() int
	SetGridPosition(row, column int)

	XExpandable() bool
	YExpandable() bool

	MinWidth() int
	MinHeight() int

	MaxWidth() int
	MaxHeight() int

	SetAbsolutePositioning(absolute bool)
	SetXExpandable(expandable bool)
	SetYExpandable(expandable bool)

	ClearLayoutCache()

	nextFocus(reverse bool)
	nextFocusByDirection(dir string)

	setPreviousFocusedWidget(w Widgeter)
	getPreviousFocusedWidget() Widgeter
}
