package ui

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"math"
	"reflect"
	"strconv"
	"strings"
)

type Widget struct {
	id       string
	name     string
	typeName string

	form           *Form
	parentWidgetId string

	// position
	x int
	y int

	// size
	w int
	h int

	// inner widgets
	absolutePositioning bool
	widgets             []Widgeter

	gridX int // Grid position
	gridY int // Grid position

	minWidth  int // Minimum width
	maxWidth  int // Maximum width
	minHeight int // Minimum height
	maxHeight int // Maximum height

	// themeHeight, when set, gives the height from the theme font (see
	// setThemeHeight); setting a height explicitly turns it off
	themeHeight      func() int
	themeHeightFixed bool

	// Texts that follow the language, see setTextFunc
	textFunc    func() string
	textSetter  func(string)
	tooltipFunc func() string

	allowScrollX   bool
	allowScrollY   bool
	hideScrollbarX bool
	hideScrollbarY bool
	scrollX        int
	scrollY        int
	innerWidth     int
	innerHeight    int

	scrollBarXSize            int
	scrollingX                bool
	scrollingXInitial         int
	scrollingXInitialMousePos int

	scrollBarYSize            int
	scrollingY                bool
	scrollingYInitial         int
	scrollingYInitialMousePos int

	previousFocusedWidget Widgeter

	// enabled bool

	props          map[string]interface{}
	propsFunctions map[string]func()

	timers []*timer

	visible bool

	//autoFillBackground bool

	contextMenu *ContextMenu

	canBeFocused bool

	allowCallMouseClickCallback bool

	dontAllowOnClickIfDisabled bool

	// temp
	lastMouseX       int // After scrolling
	lastMouseY       int // After scrolling
	lastMouseAbsPosX int // Last mouse position relative to the widget
	lastMouseAbsPosY int // Last mouse position relative to the widget

	//mouseCursor MouseCursor

	foregroundColor color.Color
	backgroundColor color.Color

	PopupWidgets []Widgeter

	layoutCacheXExpandableValid bool
	layoutCacheYExpandableValid bool
	layoutCacheMinWidthValid    bool
	layoutCacheMinHeightValid   bool

	layoutCacheXExpandable bool
	layoutCacheYExpandable bool
	layoutCacheMinWidth    int
	layoutCacheMinHeight   int

	closeByClickOutside bool

	// insetTop is extra space above the grid of children, e.g. for the
	// title of a GroupBox
	insetTop int

	// callbacks
	onCustomPaint   func(cnv *Canvas)
	onPostPaint     func(cnv *Canvas)
	onMouseDown     func(button MouseButton, x int, y int, mods KeyModifiers) bool
	onMouseUp       func(button MouseButton, x int, y int, mods KeyModifiers) bool
	onMouseDblClick func(button MouseButton, x int, y int, mods KeyModifiers) bool
	onMouseMove     func(x int, y int, mods KeyModifiers) bool
	onMouseLeave    func()
	onMouseEnter    func()
	onKeyDown       func(key Key, mods KeyModifiers) bool
	onKeyUp         func(key Key, mods KeyModifiers) bool
	onChar          func(char rune, mods KeyModifiers) bool
	onMouseWheel    func(deltaX, deltaY int) bool
	//onClick         func()
	onScrollChanged func(scrollX, scrollY int)

	onFocused   func()
	onFocusLost func()
}

/*
	Properties:
	- elevation: int - The elevation level of the widget.
	- role: string - The role of the widget ["primary", "secondary", "surface"].
	- autofillbackground: bool - Whether the widget should automatically fill its background.
	- padding: int - The padding around the widget's content.
	- spacing: int - The spacing between child widgets.
	- cursor: string - The mouse cursor to use when hovering over the widget ["arrow", "pointer", "resize-hor", "resize-ver", "ibeam"].
	- xexpandable: bool - Whether the widget can expand in the horizontal direction.
	- yexpandable: bool - Whether the widget can expand in the vertical direction.

	- onclick: function - Callback function for mouse click events.
*/

// DefaultUiLineHeight is the height of a single-line control at the theme
// font size (see ThemeControlHeight); ApplyBaseFontSize updates it.
var DefaultUiLineHeight = 30

type Event struct {
	Parameter any
}

var eventsStack []*Event

func PushEvent(p any) {
	var ev Event
	ev.Parameter = p
	eventsStack = append(eventsStack, &ev)
}

func PopEvent() {
	if len(eventsStack) == 0 {
		return
	}
	eventsStack = eventsStack[:len(eventsStack)-1]
}

func CurrentEvent() *Event {
	if len(eventsStack) == 0 {
		return nil
	}
	return eventsStack[len(eventsStack)-1]
}

/*func NewWidget() *Widget {
	var c Widget
	c.InitWidget()
	return &c
}*/

type ContainerGridColumnInfo struct {
	minWidth   int
	maxWidth   int
	expandable bool
	width      int
	collapsed  bool
}

type ContainerGridRowInfo struct {
	minHeight  int
	maxHeight  int
	expandable bool
	height     int
	collapsed  bool
}

const MaxUint = ^uint(0)
const MinUint = 0

const MaxInt = int(^uint(0) >> 1)
const MinInt = -MaxInt - 1

const MAX_WIDTH = 100000
const MAX_HEIGHT = 100000

func (c *Widget) Form() *Form {
	return c.form
}

func (c *Widget) InitWidget() {
	c.id = newWidgetId()
	c.typeName = "Widget"
	c.name = "Widget-" + c.id
	c.props = make(map[string]any)
	c.propsFunctions = make(map[string]func())
	c.timers = make([]*timer, 0)
	c.x = 0
	c.y = 0
	c.w = 300
	c.h = 180
	c.minWidth = 0
	c.minHeight = 0
	c.maxWidth = MAX_WIDTH
	c.maxHeight = MAX_HEIGHT
	c.visible = true
	c.allowCallMouseClickCallback = true
	//c.panelPadding = 2
	//c.cellPadding = 6
	c.SetProp("padding", 2)
	c.SetProp("spacing", 6)
	c.scrollBarXSize = 10
	c.scrollBarYSize = 10
	c.innerWidth = 0
	/*c.anchorLeft = true
	c.anchorTop = true
	c.anchorRight = false
	c.anchorBottom = false*/
	c.widgets = make([]Widgeter, 0)
	c.closeByClickOutside = true

	c.SetProp("enabled", true)
	// c.backgroundColor = color.RGBA{R: 0, G: 0, B: 0, A: 0}
	c.PopupWidgets = make([]Widgeter, 0)
}

func (c *Widget) Id() string {
	return c.id
}

func (c *Widget) SetAutoFillBackground(autoFill bool) {
	//c.autoFillBackground = autoFill
	c.SetProp("autofillbackground", autoFill)
}

func (c *Widget) Enabled() bool {
	return c.GetPropBool("enabled", true)
}

func (c *Widget) SetEnabled(enabled bool) {
	c.SetProp("enabled", enabled)
}

func (c *Widget) FullPath() []string {
	path := make([]string, 0)
	path = append(path, c.Id())
	parentWidgetId := c.parentWidgetId
	for parentWidgetId != "" {
		parentWidget := c.form.WidgetById(parentWidgetId)
		if parentWidget == nil {
			break
		}
		path = append(path, parentWidget.Id())
		parentWidgetId = parentWidget.ParentWidgetId()
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func (c *Widget) ParentWidgetId() string {
	return c.parentWidgetId
}

func (c *Widget) SetParentWidgetId(id string) {
	c.parentWidgetId = id
	if id == "" {
		c.attachToForm(c, nil)
	}
}

func (c *Widget) SetName(name string) {
	c.name = name
}

func (c *Widget) CloseByClickOutside() bool {
	return c.closeByClickOutside
}

func (c *Widget) SetCloseByClickOutside(close bool) {
	c.closeByClickOutside = close
}

func (c *Widget) Widgets() []Widgeter {
	return c.widgets
}

func (c *Widget) SetVisible(visible bool) {
	if c.visible != visible {
		c.visible = visible
		c.updateLayout(c.w, c.h, c.w, c.h)
		c.form.Update()
	}
}

func (c *Widget) SetElevation(elevation int) {
	c.SetProp("elevation", elevation)
}

func (c *Widget) Elevation() int {
	return c.GetPropInt("elevation", 0)
}

func (c *Widget) SetRole(role string) {
	c.SetProp("role", role)
}

func (c *Widget) Role() string {
	return c.GetPropString("role", "")
}

func (c *Widget) IsRoleSurface() bool {
	return c.Role() != "primary" && c.Role() != "secondary"
}

func (c *Widget) IsRolePrimary() bool {
	return c.Role() == "primary"
}

func (c *Widget) IsRoleSecondary() bool {
	return c.Role() == "secondary"
}

func (c *Widget) IsCanBeFocused() bool {
	return c.canBeFocused
}

func (c *Widget) SetCanBeFocused(canBeFocused bool) {
	c.canBeFocused = canBeFocused
}

func (c *Widget) SetTypeName(typeName string) {
	c.typeName = typeName
}

func (c *Widget) TypeName() string {
	return c.typeName
}

func (c *Widget) IsVisible() bool {
	return c.visible
}

func (c *Widget) GridX() int {
	return c.gridX
}

func (c *Widget) GridY() int {
	return c.gridY
}

func (c *Widget) SetGridPosition(row, column int) {
	c.gridX = column
	c.gridY = row
}

func (c *Widget) MinWidth() int {
	panelPadding := c.GetPropInt("padding", 2)

	if c.layoutCacheMinWidthValid {
		return c.layoutCacheMinWidth
	}

	result := 0

	calcFromChildren := !c.allowScrollX

	if calcFromChildren {
		_, _, _, allCellPadding := c.makeColumnsInfo(c.Width())
		columnsInfo, _, _, _ := c.makeColumnsInfo(c.Width() - (panelPadding + allCellPadding + panelPadding))
		for _, columnInfo := range columnsInfo {
			result += columnInfo.minWidth
		}
		result = result + panelPadding + allCellPadding + panelPadding
	}

	c.layoutCacheMinWidthValid = true
	if c.minWidth > result {
		c.layoutCacheMinWidth = c.minWidth
		return c.minWidth
	}
	c.layoutCacheMinWidth = result

	return result
}

func (c *Widget) MinHeight() int {
	panelPadding := c.GetPropInt("padding", 2)

	if c.layoutCacheMinHeightValid {
		return c.layoutCacheMinHeight
	}

	result := 0

	calcFromChildren := !c.allowScrollY

	if calcFromChildren {
		_, _, _, allCellPadding := c.makeRowsInfo(c.Height())
		rowsInfo, _, _, _ := c.makeRowsInfo(c.Height() - (panelPadding + allCellPadding + panelPadding + c.insetTop))
		for _, rowInfo := range rowsInfo {
			result += rowInfo.minHeight
		}
		result += panelPadding + allCellPadding + panelPadding + c.insetTop
	}

	c.layoutCacheMinHeightValid = true
	if c.minHeight > result {
		c.layoutCacheMinHeight = c.minHeight
		return c.minHeight
	}
	c.layoutCacheMinHeight = result
	return result
}

func (c *Widget) MaxWidth() int {
	return c.maxWidth
}

func (c *Widget) MaxHeight() int {
	return c.maxHeight
}

func (c *Widget) AddTimer(intervalMs int, callback func()) {
	t := &timer{
		intervalMs: intervalMs,
		callback:   callback,
	}
	c.timers = append(c.timers, t)
}

func (c *Widget) SetPanelPadding(padding int) {
	//c.panelPadding = padding
	c.SetProp("padding", padding)
}

func (c *Widget) SetCellPadding(padding int) {
	//c.cellPadding = padding
	c.SetProp("spacing", padding)
}

func (c *Widget) AddWidget(gridRow int, gridColumn int, w Widgeter) {
	if _, exists := lookupWidget(w.Id()); exists {
		return
	}
	w.SetGridPosition(gridRow, gridColumn)
	c.widgets = append(c.widgets, w)
	w.SetParentWidgetId(c.Id())
	w.attachToForm(w, c.form)
	c.updateLayout(0, 0, 0, 0)
	if c.form != nil {
		c.form.Panel().updateLayout(0, 0, 0, 0) // Global Update Layout
		c.form.UpdateLayout()
	}
}

func (c *Widget) setId(id string) {
	c.id = id
}

// self must be the widget's own concrete Widgeter value (not c). c is only
// the embedded Widget when this method is reached via a promoted-method call
// through the Widgeter interface on a struct that embeds Widget (e.g. *Table),
// so registering allwidgets under c instead of self would store the base
// *Widget and silently break dispatch of any method the subtype overrides.
func (c *Widget) attachToForm(self Widgeter, form *Form) {
	c.form = form
	if form != nil {
		registerWidget(self)
		// Sizes measured before the widget had a form missed the property
		// changes made meanwhile, e.g. SetFontFamily on a label
		self.applyThemeMetrics()
	} else {
		unregisterWidget(c.id)
	}
	for _, w := range c.widgets {
		w.attachToForm(w, form)
	}
	if c.contextMenu != nil {
		c.contextMenu.attachToForm(c.contextMenu, form)
	}
}

func (c *Widget) RemoveWidget(w Widgeter) {
	for i, widget := range c.widgets {
		widgeter := widget
		if widgeter.Id() == w.Id() {
			w.SetParentWidgetId("")
			c.widgets = append(c.widgets[:i], c.widgets[i+1:]...)
			return
		}
	}
	c.updateLayout(0, 0, 0, 0)
}

func (c *Widget) RemoveAllWidgets() {
	for _, w := range c.widgets {
		w.SetParentWidgetId("")
	}
	c.widgets = make([]Widgeter, 0)
	c.updateLayout(0, 0, 0, 0)
	c.form.Update()
}

func (c *Widget) FindWidgetByName(name string) Widgeter {
	for _, w := range c.widgets {
		if w.Name() == name {
			return w
		}
	}

	// recursive search in child widgets
	for _, w := range c.widgets {
		childWidget := w.FindWidgetByName(name)
		if childWidget != nil {
			return childWidget
		}
	}
	return nil
}

func (c *Widget) NextGridColumn() int {
	if len(c.widgets) == 0 {
		return 0
	}

	maxX := 0
	for _, w := range c.widgets {
		if w.GridX() >= maxX {
			maxX = w.GridX() + 1
		}
	}
	return maxX
}

func (c *Widget) NextGridRow() int {
	if len(c.widgets) == 0 {
		return 0
	}

	maxY := 0
	for _, w := range c.widgets {
		if w.GridY() >= maxY {
			maxY = w.GridY() + 1
		}
	}
	return maxY
}

func (c *Widget) SetForegroundColor(col color.Color) {
	c.foregroundColor = col
}

func (c *Widget) SetBackgroundColor(col color.Color) {
	c.backgroundColor = col
}

func (c *Widget) SetAllowScroll(allowX bool, allowY bool) {
	c.allowScrollX = allowX
	c.allowScrollY = allowY
}

func MouseCursorToString(c MouseCursor) string {
	switch c {
	case MouseCursorNotDefined:
		return ""
	case MouseCursorArrow:
		return "arrow"
	case MouseCursorPointer:
		return "pointer"
	case MouseCursorResizeHor:
		return "resize-hor"
	case MouseCursorResizeVer:
		return "resize-ver"
	case MouseCursorIBeam:
		return "ibeam"
	}
	return ""
}

func MouseCursorFromString(s string) MouseCursor {
	switch s {
	case "arrow":
		return MouseCursorArrow
	case "pointer":
		return MouseCursorPointer
	case "resize-hor":
		return MouseCursorResizeHor
	case "resize-ver":
		return MouseCursorResizeVer
	case "ibeam":
		return MouseCursorIBeam
	}
	return MouseCursorArrow
}

func (c *Widget) SetMouseCursor(cursor MouseCursor) {
	c.SetProp("cursor", MouseCursorToString(cursor))
}

func (c *Widget) MouseCursor() MouseCursor {
	return MouseCursorFromString(c.GetPropString("cursor", ""))
}

func (c *Widget) X() int {
	return c.x
}

func (c *Widget) Y() int {
	return c.y
}

func (c *Widget) Width() int {
	return c.w
}

func (c *Widget) Height() int {
	return c.h
}

func (c *Widget) InnerWidth() int {
	if c.innerWidth == 0 {
		return c.w
	}
	return c.innerWidth
}

func (c *Widget) InnerHeight() int {
	if c.innerHeight == 0 {
		return c.h
	}
	return c.innerHeight
}

func (c *Widget) SetInnerSize(width, height int) {
	c.innerWidth = width
	c.innerHeight = height
}

///////////////////////////////////////////////////////////
// Properties
///////////////////////////////////////////////////////////

func (c *Widget) SetProp(key string, value any) {
	c.props[key] = value

	if c.form != nil {
		w := c.form.WidgetById(c.Id())
		if w != nil {
			w.ProcessPropChange(key, value)
		}
	}
}

func (c *Widget) ProcessPropChange(key string, value interface{}) {
	// Default implementation does nothing
}

func (c *Widget) SetPropFunction(key string, f func()) {
	c.propsFunctions[key] = f
}

func (c *Widget) GetProp(key string) any {
	if value, ok := c.props[key]; ok {
		return value
	}
	return nil
}

func (c *Widget) GetPropFunction(key string) func() {
	if f, ok := c.propsFunctions[key]; ok {
		return f
	}
	return nil
}

func (c *Widget) GetPropString(key string, defaultValue string) string {
	v := c.GetProp(key)
	if v != nil {
		return fmt.Sprint(v)
	}
	return defaultValue
}

func (c *Widget) GetPropInt(key string, defaultValue int) int {
	v := c.GetProp(key)
	if v != nil {
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(rv.Uint())
		case reflect.Float32, reflect.Float64:
			return int(math.Round(rv.Float()))
		case reflect.String:
			i, err := strconv.Atoi(rv.String())
			if err == nil {
				return i
			}
		}
	}
	return defaultValue
}

func (c *Widget) GetPropFloat64(key string, defaultValue float64) float64 {
	v := c.GetProp(key)
	if v != nil {
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Float32, reflect.Float64:
			return rv.Float()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return float64(rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return float64(rv.Uint())
		case reflect.String:
			var f float64
			f, err := strconv.ParseFloat(rv.String(), 64)
			if err == nil {
				return f
			}
		}
	}
	return defaultValue
}

func (c *Widget) GetPropBool(key string, defaultValue bool) bool {
	v := c.GetProp(key)
	if v != nil {
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Bool:
			return rv.Bool()
		case reflect.String:
			strVal := rv.String()
			strVal = strings.ToLower(strVal)
			if strVal == "true" || strVal == "1" {
				return true
			} else if strVal == "false" || strVal == "0" {
				return false
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return rv.Int() != 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return rv.Uint() != 0
		case reflect.Float32, reflect.Float64:
			return rv.Float() != 0.0
		}
	}
	return defaultValue
}

func (c *Widget) GetPropColor(key string, defaultValue string) color.Color {
	v := c.GetProp(key)
	if v != nil {
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.String {
			return ColorFromHex(rv.String())
		}
		return ColorFromHex(defaultValue)
	}
	return ColorFromHex(defaultValue)
}

func (c *Widget) GetHAlign(key string, defaultValue HAlign) HAlign {
	v := c.GetProp(key)
	if v != nil {
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.String:
			vStr := strings.ToLower(rv.String())
			switch vStr {
			case "left":
				return HAlignLeft
			case "center":
				return HAlignCenter
			case "right":
				return HAlignRight
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal := int(rv.Int())
			if intVal >= int(HAlignLeft) && intVal <= int(HAlignRight) {
				return HAlign(intVal)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uintVal := int(rv.Uint())
			if uintVal >= int(HAlignLeft) && uintVal <= int(HAlignRight) {
				return HAlign(uintVal)
			}
		}
	}
	return defaultValue
}

func (c *Widget) GetVAlign(key string, defaultValue VAlign) VAlign {
	v := c.GetProp(key)
	if v != nil {
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.String:
			vStr := strings.ToLower(rv.String())
			switch vStr {
			case "top":
				return VAlignTop
			case "center":
				return VAlignCenter
			case "bottom":
				return VAlignBottom
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal := int(rv.Int())
			if intVal >= int(VAlignTop) && intVal <= int(VAlignBottom) {
				return VAlign(intVal)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uintVal := int(rv.Uint())
			if uintVal >= int(VAlignTop) && uintVal <= int(VAlignBottom) {
				return VAlign(uintVal)
			}
		}
	}
	return defaultValue
}

////////////////////////////////////////////////////////////

func (c *Widget) SetOnFocused(onFocused func()) {
	c.onFocused = onFocused
}

func (c *Widget) SetOnFocusLost(onFocusLost func()) {
	c.onFocusLost = onFocusLost
}

func (c *Widget) Focus() {
	if !c.canBeFocused {
		return
	}
	if c.form == nil {
		return
	}
	// fmt.Println("Widget Focused", c.Name(), "Id:", c.Id(), "Type:", c.TypeName())
	widgetToFocus := c.form.WidgetById(c.Id())
	if widgetToFocus == nil {
		return
	}
	focusChanged := false
	previousFocusedWidget := c.form.focusedWidget

	if (c.form.focusedWidget != nil && c.form.focusedWidget.Id() != widgetToFocus.Id()) || previousFocusedWidget == nil {
		focusChanged = true
	}

	if focusChanged {
		if previousFocusedWidget != nil {
			previousFocusedWidget.ProcessFocusLost()
		}

		c.form.focusedWidget = widgetToFocus
		c.form.Update()

		widgetToFocus.ProcessFocused()
	}
}

func (c *Widget) ProcessFocused() {
	if c.onFocused != nil {
		c.onFocused()
	}
}

func (c *Widget) ProcessFocusLost() {
	if c.onFocusLost != nil {
		c.onFocusLost()
	}
}

func (c *Widget) IsFocused() bool {
	return c.form.focusedWidget == c.form.WidgetById(c.Id())
}

func (c *Widget) IsHovered() bool {
	return c.form.hoverWidget == c.form.WidgetById(c.Id())
}

func (c *Widget) Name() string {
	return c.name
}

func (c *Widget) SetPosition(x, y int) {
	c.x = x
	c.y = y
}

func (c *Widget) SetSize(w, h int) {
	oldW := c.w
	oldH := c.h
	c.w = w
	c.h = h
	c.updateLayout(oldW, oldH, w, h)
	c.checkScrolls()
}

func (c *Widget) SetMinSize(minWidth, minHeight int) {
	c.themeHeight = nil
	c.minWidth = minWidth
	c.minHeight = minHeight
	c.checkScrolls()
}

func (c *Widget) SetMaxSize(maxWidth, maxHeight int) {
	c.themeHeight = nil
	c.maxWidth = maxWidth
	c.maxHeight = maxHeight
	c.checkScrolls()
}

func (c *Widget) SetAnchors(left, top, right, bottom bool) {
	/*c.anchorLeft = left
	c.anchorTop = top
	c.anchorRight = right
	c.anchorBottom = bottom*/
}

func (c *Widget) SetOnClick(f func()) {
	c.SetPropFunction("onclick", f)
}

func (c *Widget) SetOnScrollChanged(f func(scrollX, scrollY int)) {
	c.onScrollChanged = f
}

func (c *Widget) SetOnPaint(f func(cnv *Canvas)) {
	c.onCustomPaint = f
}

func (c *Widget) SetOnPostPaint(f func(cnv *Canvas)) {
	c.onPostPaint = f
}

func (c *Widget) SetOnMouseDown(f func(button MouseButton, x int, y int, mods KeyModifiers) bool) {
	c.onMouseDown = f
}

func (c *Widget) SetOnMouseUp(f func(button MouseButton, x int, y int, mods KeyModifiers) bool) {
	c.onMouseUp = f
}

func (c *Widget) SetOnMouseDblClick(f func(button MouseButton, x int, y int, mods KeyModifiers) bool) {
	c.onMouseDblClick = f
}

func (c *Widget) SetOnMouseMove(f func(x int, y int, mods KeyModifiers) bool) {
	c.onMouseMove = f
}

func (c *Widget) SetOnMouseLeave(f func()) {
	c.onMouseLeave = f
}

func (c *Widget) SetOnMouseEnter(f func()) {
	c.onMouseEnter = f
}

func (c *Widget) SetOnKeyDown(f func(key Key, mods KeyModifiers) bool) {
	c.onKeyDown = f
}

func (c *Widget) SetOnKeyUp(f func(key Key, mods KeyModifiers) bool) {
	c.onKeyUp = f
}

func (c *Widget) SetOnChar(f func(char rune, mods KeyModifiers) bool) {
	c.onChar = f
}

func (c *Widget) SetOnMouseWheel(f func(deltaX, deltaY int) bool) {
	c.onMouseWheel = f
}

func (c *Widget) setScrollX(scrollX int) {
	if c.scrollX != scrollX {
		c.scrollX = scrollX
		if c.onScrollChanged != nil {
			c.onScrollChanged(c.scrollX, c.scrollY)
		}
	}
}

func (c *Widget) setScrollY(scrollY int) {
	if c.scrollY != scrollY {
		c.scrollY = scrollY
		if c.onScrollChanged != nil {
			c.onScrollChanged(c.scrollX, c.scrollY)
		}
	}
}

func (c *Widget) ScrollX() int {
	return c.scrollX
}

func (c *Widget) ScrollY() int {
	return c.scrollY
}

// SetScrollX scrolls the content horizontally to scrollX, kept within the content
func (c *Widget) SetScrollX(scrollX int) {
	c.setScrollX(scrollX)
	c.checkScrolls()
	c.form.Update()
}

// SetScrollY scrolls the content vertically to scrollY, kept within the content
func (c *Widget) SetScrollY(scrollY int) {
	c.setScrollY(scrollY)
	c.checkScrolls()
	c.form.Update()
}

func (c *Widget) ScrollEnsureVisible(x1, y1 int) {

	if y1 < c.scrollY {
		c.setScrollY(y1)
	}
	if y1 > c.scrollY+c.Height() {
		c.setScrollY(y1 - c.Height())
	}

	if x1 < c.scrollX {
		c.setScrollX(x1)
	}
	if x1 > c.scrollX+c.Width() {
		c.setScrollX(x1 - c.Width())
	}
}

func (c *Widget) getWidgetAt(x, y int) Widgeter {

	for _, w := range c.widgets {
		innerWidth := w.Width()
		innerHeight := w.Height()
		//innerWidth := w.InnerWidth()
		//innerHeight := w.InnerHeight()
		if x >= w.X() && x < w.X()+innerWidth && y >= w.Y() && y < w.Y()+innerHeight {
			//fmt.Println("Widget found at", w.Name(), "at position", w.X(), w.Y(), "with size", innerWidth, innerHeight)
			return w
		}
	}
	return nil
}

func (c *Widget) findWidgetAt(x, y int) Widgeter {
	// if it is the bar area, return self
	if c.allowScrollX && c.innerWidth > c.w && y >= c.h-c.scrollBarXSize {
		return c
	}
	if c.allowScrollY && c.innerHeight > c.h && x >= c.w-c.scrollBarYSize {
		return c
	}

	x += c.scrollX
	y += c.scrollY

	innerWidget := c.getWidgetAt(x, y)
	if innerWidget != nil {
		return innerWidget.findWidgetAt(x-innerWidget.X(), y-innerWidget.Y())
	}
	return c.form.WidgetById(c.Id())
	//return c
}

func (c *Widget) ProcessPaint(cnv *Canvas) {
	// Draw the background color if set
	autoFillBackground := c.GetPropBool("autofillbackground", false)
	if autoFillBackground {
		backgroundColor := c.BackgroundColor()
		_, _, _, a := backgroundColor.RGBA()
		if a > 0 {
			cnv.SetColor(backgroundColor)
			cnv.FillRect(0, 0, c.w, c.h, backgroundColor)
		}
	}

	// Draw using the custom paint function if set
	cnv.Save()

	cnv.state.translateX -= c.scrollX
	cnv.state.translateY -= c.scrollY

	if c.onCustomPaint != nil {
		c.onCustomPaint(cnv)
	}

	// Draw all child widgets
	for _, w := range c.widgets {

		cnv.Save()
		cnv.TranslateAndClip(w.X(), w.Y(), w.Width(), w.Height())
		w.ProcessPaint(cnv)
		cnv.Restore()
	}

	if c.onPostPaint != nil {
		c.onPostPaint(cnv)
	}

	cnv.Restore()

	// Draw ScrollBarX
	if !c.hideScrollbarX && c.allowScrollX && c.innerWidth > c.w {
		scrollBarWidth := c.w * c.w / c.innerWidth
		scrollBarX := c.scrollX * (c.w - scrollBarWidth) / (c.innerWidth - c.w)

		hovered := c.lastMouseAbsPosY >= c.h-c.scrollBarXSize && c.lastMouseAbsPosY < c.h
		drawScrollBarThumb(cnv, scrollBarX, c.h-c.scrollBarXSize, scrollBarWidth, c.scrollBarXSize, hovered)
	}

	// Draw ScrollBarY
	if !c.hideScrollbarY && c.allowScrollY && c.innerHeight > c.h {
		scrollBarHeight := c.h * c.h / c.innerHeight
		scrollBarY := c.scrollY * (c.h - scrollBarHeight) / (c.innerHeight - c.h)

		hovered := c.lastMouseAbsPosX >= c.w-c.scrollBarYSize && c.lastMouseAbsPosX < c.w
		drawScrollBarThumb(cnv, c.w-c.scrollBarYSize, scrollBarY, c.scrollBarYSize, scrollBarHeight, hovered)
	}

	/*if !c.Enabled() {
		backgroundColor := color.RGBA{R: 55, G: 55, B: 55, A: 55}
		_, _, _, a := backgroundColor.RGBA()
		if a > 0 {
			cnv.SetColor(backgroundColor)
			cnv.FillRect(0, 0, c.w, c.h, backgroundColor)
		}
	}*/
}

func (c *Widget) ProcessMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	// Determine if the click is within the horizontal scroll bar area
	if c.allowScrollX && c.innerWidth > c.w && y >= c.h-c.scrollBarXSize {
		isLeftBar := x < c.w*c.scrollX/c.innerWidth
		if isLeftBar {
			// Clicked in the left part of the scroll bar
			pageSize := c.w * c.w / c.innerWidth
			c.scrollX -= pageSize // Scroll left
			if c.scrollX < 0 {
				c.setScrollX(0)
			}
			c.checkScrolls()
			return true
		}

		isRightBar := x >= c.w*(c.scrollX+c.w)/c.innerWidth
		if isRightBar {
			// Clicked in the right part of the scroll bar
			pageSize := c.w * c.w / c.innerWidth
			c.scrollX += pageSize // Scroll right
			if c.scrollX > c.innerWidth-c.w {
				c.setScrollX(c.innerWidth - c.w)
			}
			c.checkScrolls()
			return true
		}

		// Clicked in the scroll bar
		scrollBarWidth := c.w * c.w / c.innerWidth
		scrollBarX := c.scrollX * (c.w - scrollBarWidth) / (c.innerWidth - c.w)
		if x >= scrollBarX && x < scrollBarX+scrollBarWidth {
			c.scrollingX = true
			c.scrollingXInitial = c.scrollX
			c.scrollingXInitialMousePos = x
			fmt.Println("Started scrollingX", c.scrollingXInitial, c.scrollingXInitialMousePos)
			return true
		}
	}

	// Determine if the click is within the vertical scroll bar area
	if c.allowScrollY && c.innerHeight > c.h && x >= c.w-c.scrollBarYSize {
		isUpperBar := y < c.h*c.scrollY/c.innerHeight
		if isUpperBar {
			// Clicked in the upper part of the scroll bar
			pageSize := c.h * c.h / c.innerHeight
			c.scrollY -= pageSize // Scroll up
			if c.scrollY < 0 {
				c.setScrollY(0)
			}
			c.checkScrolls()
			return true
		}

		isLowerBar := y >= c.h*(c.scrollY+c.h)/c.innerHeight
		if isLowerBar {
			// Clicked in the lower part of the scroll bar
			pageSize := c.h * c.h / c.innerHeight
			c.scrollY += pageSize // Scroll down
			if c.scrollY > c.innerHeight-c.h {
				c.setScrollY(c.innerHeight - c.h)
			}
			c.checkScrolls()
			return true
		}

		// Clicked in the scroll bar
		scrollBarHeight := c.h * c.h / c.innerHeight
		scrollBarY := c.scrollY * (c.h - scrollBarHeight) / (c.innerHeight - c.h)
		if y >= scrollBarY && y < scrollBarY+scrollBarHeight {
			c.scrollingY = true
			c.scrollingYInitial = c.scrollY
			c.scrollingYInitialMousePos = y
			fmt.Println("Started scrollingY", c.scrollingYInitial, c.scrollingYInitialMousePos)
			return true
		}
	}

	// Apply scrolling
	x += c.scrollX
	y += c.scrollY

	// Delegate the mouse down event to the widgets
	processed := false
	childHit := false

	for _, w := range c.widgets {
		if x >= w.X() && x < w.X()+w.Width() && y >= w.Y() && y < w.Y()+w.Height() {
			processed = w.ProcessMouseDown(button, x-w.X(), y-w.Y(), mods)
			if processed {
				break
			}
			childHit = true
		}
	}

	// A right click on a child that didn't handle it opens this widget's
	// context menu: the click goes up to the nearest parent with a menu
	if !processed {
		contextMenuFound := false
		//if event.Button == MouseButtonRight {
		if button == MouseButtonRight {
			wX, wY := c.RectClientAreaOnWindow()
			if c.ContextMenu() != nil {
				c.ContextMenu().ShowMenu(wX+x-c.scrollX, wY+y-c.scrollY)
				contextMenuFound = true
			} /*else {
				if c.OnContextMenuNeed != nil {
					m := c.OnContextMenuNeed(me.X, me.Y)
					if m != nil {
						m.ShowMenu(wX+me.X-c.ScrollOffsetX(), wY+me.Y-c.ScrollOffsetY())
						contextMenuFound = true
					}
				}
			}*/
		}
		if contextMenuFound {
			processed = true
		}
	}

	// Other clicks on a child don't reach this widget's own handler
	if childHit {
		processed = true
	}

	if !processed && c.onMouseDown != nil {
		processed = c.onMouseDown(button, x, y, mods)
	}

	// "onclick" is fired on mouse-up (see Button.buttonProcessMouseUp), not
	// here on mouse-down, so a press can still be cancelled by dragging off
	// the widget before releasing - the standard button click semantics.

	return processed
}

func (c *Widget) ProcessMouseUp(button MouseButton, x int, y int, mods KeyModifiers, onlyForWidgetId string) bool {
	// If scrolling is active, stop it
	if c.scrollingX {
		c.scrollingX = false
		return true
	}

	// If scrolling is active, stop it
	if c.scrollingY {
		c.scrollingY = false
		return true
	}

	x += c.scrollX
	y += c.scrollY

	for _, w := range c.widgets {
		w.ProcessMouseUp(button, x-w.X(), y-w.Y(), mods, onlyForWidgetId)
	}

	if c.onMouseUp != nil && onlyForWidgetId == c.Id() {
		c.onMouseUp(button, x, y, mods)
	}

	return false
}

func (c *Widget) ProcessMouseMove(x int, y int, mods KeyModifiers) bool {
	if c.form == nil {
		return false
	}

	if c.scrollingX {
		if c.allowScrollX && c.innerWidth > c.w {
			k := float64(c.innerWidth) / float64(c.w)
			newScrollFloat64 := float64(c.scrollingXInitial) + float64(x-c.scrollingXInitialMousePos)*k
			c.setScrollX(int(newScrollFloat64))
			c.checkScrolls()
			return true
		}
		return true
	}

	if c.scrollingY {
		if c.allowScrollY && c.innerHeight > c.h {
			k := float64(c.innerHeight) / float64(c.h)
			newScrollFloat64 := float64(c.scrollingYInitial) + float64(y-c.scrollingYInitialMousePos)*k
			c.setScrollY(int(newScrollFloat64))
			c.checkScrolls()
			return true
		}
		return true
	}

	c.lastMouseAbsPosX = x
	c.lastMouseAbsPosY = y

	x += c.scrollX
	y += c.scrollY

	c.lastMouseX = x
	c.lastMouseY = y

	processed := false

	for _, w := range c.widgets {
		// Temporary process in the all widgets - perrormance issue
		//inWidget := true

		inWidget := x >= w.X() && x < w.X()+w.Width() && y >= w.Y() && y < w.Y()+w.Height()
		if c.form.mouseLeftButtonPressed && c.form.mouseLeftButtonPressedWidget != nil {
			pathToPressedWidget := c.form.mouseLeftButtonPressedWidget.FullPath()
			itemsInPathAsSet := make(map[string]bool)
			for _, item := range pathToPressedWidget {
				itemsInPathAsSet[item] = true
			}
			if _, ok := itemsInPathAsSet[w.Id()]; ok {
				inWidget = true // If the widget is in the path of the pressed widget, process it
			}
		}

		if inWidget {
			processed = w.ProcessMouseMove(x-w.X(), y-w.Y(), mods)
			if processed {
				break
			}
		}
	}

	if !processed && c.onMouseMove != nil {
		processed = c.onMouseMove(x, y, mods)
		return processed
	}

	return processed
}

func (c *Widget) ProcessMouseLeave() bool {
	if c.onMouseLeave != nil {
		c.onMouseLeave()
	}
	c.form.Update()
	return true
}

func (c *Widget) ProcessMouseEnter() bool {
	if c.onMouseEnter != nil {
		c.onMouseEnter()
	}
	c.form.Update()
	return true
}

func (c *Widget) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	processed := false

	if !processed && c.onKeyDown != nil {
		processed = c.onKeyDown(key, mods)
	}

	return processed
}

func (c *Widget) ProcessKeyUp(key Key, mods KeyModifiers) bool {
	processed := false

	/*for _, w := range c.widgets {
		processed = w.ProcessKeyUp(key, mods)
		if processed {
			break
		}
	}*/

	if !processed && c.onKeyUp != nil {
		processed = c.onKeyUp(key, mods)
	}

	return processed
}

func (c *Widget) ProcessMouseDblClick(button MouseButton, x int, y int, mods KeyModifiers) bool {
	fmt.Println("Widget Mouse Double Click", x, y)

	x += c.scrollX
	y += c.scrollY

	processed := false

	for _, w := range c.widgets {
		if x >= w.X() && x < w.X()+w.Width() && y >= w.Y() && y < w.Y()+w.Height() {
			processed = w.ProcessMouseDblClick(button, x-w.X(), y-w.Y(), mods)
			if processed {
				break
			}
			processed = true
		}
	}

	if !processed && c.onMouseDblClick != nil {
		processed = c.onMouseDblClick(button, x, y, mods)
	}

	return processed
}

func (c *Widget) ProcessChar(char rune, mods KeyModifiers) bool {
	processed := false

	/*for _, w := range c.widgets {
		processed = w.ProcessChar(char, mods)
		if processed {
			break
		}
	}*/

	if !processed && c.onChar != nil {
		processed = c.onChar(char, mods)
	}

	return processed
}

func (c *Widget) ProcessMouseWheel(deltaX, deltaY int) bool {
	hoverWidget := c.getWidgetAt(c.lastMouseX, c.lastMouseY)
	if hoverWidget != nil {
		processed := hoverWidget.ProcessMouseWheel(deltaX, deltaY)
		if processed {
			return true
		}
	}

	if c.onMouseWheel != nil {
		c.onMouseWheel(deltaX, deltaY)
		return true
	}

	if deltaY != 0 && c.allowScrollY && c.InnerHeight() > c.h {
		c.scrollY -= deltaY * 30 // Adjust the scroll speed as needed
		c.checkScrolls()
		return true
	}

	if deltaX != 0 && c.allowScrollX && c.InnerWidth() > c.w {
		c.scrollX -= deltaX * 30 // Adjust the scroll speed as needed
		c.checkScrolls()
		return true
	}

	/*if (c.allowScrollX || c.allowScrollY) && (c.innerWidth > c.w || c.innerHeight > c.h) {
		if c.allowScrollX {
			c.scrollX -= deltaX * 30
		}
		if c.allowScrollY {
			c.scrollY -= deltaY * 30
			fmt.Println("WidgetName:", c.name, "ScrollY:", c.scrollY)
		}
		c.checkScrolls()
		return true
	}*/

	return false
}

func (c *Widget) ProcessTimer() {
	for _, t := range c.timers {
		t.tick()
	}

	for _, w := range c.widgets {
		w.ProcessTimer()
	}
}

func (c *Widget) checkScrolls() {
	if c.allowScrollX {
		if c.scrollX > c.innerWidth-c.w {
			c.setScrollX(c.innerWidth - c.w)
		}
		if c.scrollX < 0 {
			c.setScrollX(0)
		}
		if c.innerWidth < c.w {
			c.setScrollX(0)
		}
	}

	if c.allowScrollY {
		if c.scrollY > c.innerHeight-c.h {
			c.setScrollY(c.innerHeight - c.h)
		}
		if c.scrollY < 0 {
			c.setScrollY(0)
		}
		if c.innerHeight < c.h {
			c.setScrollY(0)
		}
	}
}

/*func (c *Widget) Anchors() (left, top, right, bottom bool) {
	return c.anchorLeft, c.anchorTop, c.anchorRight, c.anchorBottom
}*/

func (c *Widget) SetAbsolutePositioning(absolute bool) {
	c.absolutePositioning = absolute
	c.SetProp("absolutepositioning", absolute)
}

func (c *Widget) SetXExpandable(expandable bool) {
	//c.xExpandable = expandable
	c.SetProp("xexpandable", expandable)
}

func (c *Widget) SetYExpandable(expandable bool) {
	//c.yExpandable = expandable
	c.SetProp("yexpandable", expandable)
}

func (c *Widget) SetMinWidth(minWidth int) {
	c.minWidth = minWidth
}

func (c *Widget) SetMinHeight(minHeight int) {
	c.themeHeight = nil
	c.minHeight = minHeight
}

func (c *Widget) SetMaxWidth(maxWidth int) {
	c.maxWidth = maxWidth
}

func (c *Widget) SetMaxHeight(maxHeight int) {
	c.themeHeight = nil
	c.maxHeight = maxHeight
}

func (c *Widget) AppendPopupWidget(w Widgeter) {
	if w != nil {
		w.setPreviousFocusedWidget(c.form.focusedWidget)
		c.PopupWidgets = append(c.PopupWidgets, w)
		w.SetParentWidgetId(c.form.Panel().Id())
		registerWidget(w)
		w.attachToForm(w, w.Form())
	}
	c.form.syncPopupWindows()
	c.form.Update()
}

func (c *Widget) CloseAfterPopupWidget(w Widgeter) {
	foundIndex := -1
	for index, popupWidget := range c.PopupWidgets {
		if popupWidget.Id() == w.Id() {
			foundIndex = index
			break
		}
	}

	if foundIndex > -1 {
		foundIndex++

		for i := foundIndex; i < len(c.PopupWidgets); i++ {
			popupWidget := c.PopupWidgets[i]
			previousFocusedWidget := popupWidget.getPreviousFocusedWidget()
			popupWidget.ProcessClosePopup()
			if previousFocusedWidget != nil {
				previousFocusedWidget.Focus()
			}
		}

		if foundIndex < len(c.PopupWidgets) {
			c.PopupWidgets = append(c.PopupWidgets[:foundIndex], c.PopupWidgets[foundIndex+1:]...)
		}
		c.ClearFocus()
		c.form.syncPopupWindows()
		c.form.updateHover()
		c.form.Update()
	}
}

func (c *Widget) CloseAllPopup() {
	for _, popupWidget := range c.PopupWidgets {
		previousFocusedWidget := popupWidget.getPreviousFocusedWidget()
		popupWidget.ProcessClosePopup()
		if previousFocusedWidget != nil {
			previousFocusedWidget.Focus()
		}
	}

	c.ClearFocus()

	c.PopupWidgets = make([]Widgeter, 0)
	c.form.syncPopupWindows()
	c.form.updateHover()
	c.form.Update()
}

func (c *Widget) CloseTopPopup() {
	if len(c.PopupWidgets) == 0 {
		return

	}

	c.ClearFocus()

	previousFocusedWidget := c.PopupWidgets[len(c.PopupWidgets)-1].getPreviousFocusedWidget()
	c.PopupWidgets[len(c.PopupWidgets)-1].ProcessClosePopup()
	c.PopupWidgets = c.PopupWidgets[:len(c.PopupWidgets)-1]
	c.form.syncPopupWindows()
	if previousFocusedWidget != nil {
		previousFocusedWidget.Focus()
	}
	// The mouse is now over what was under the popup
	c.form.updateHover()
}

func (c *Widget) ProcessClosePopup() {
}

func (c *Widget) ClearFocus() {
	if c.form.focusedWidget != nil {
		c.form.focusedWidget.ProcessFocusLost()
		c.form.focusedWidget = nil
		c.form.Update()
	}
}

// var updateLayoutStack int

func (c *Widget) updateLayout(oldWidth, oldHeight, newWidth, newHeight int) {
	//fmt.Println("Begin Widget", c.name, "layout updated:", "Width:", c.w, "Height:", c.h, "InnerWidth:", c.innerWidth, "InnerHeight:", c.innerHeight)

	if c.form == nil {
		return
	}

	if c.form.layoutingBlockStack > 0 {
		return
	}

	for _, popupWidget := range c.PopupWidgets {
		popupWidget.updateLayout(oldWidth, oldHeight, newWidth, newHeight)
	}

	if c.absolutePositioning {
		for _, w := range c.widgets {
			newX := w.X()
			newY := w.Y()
			newW := w.Width()
			newH := w.Height()
			w.SetSize(newW, newH)
			w.SetPosition(newX, newY)
		}
	} else {
		fullWidth := c.w
		fullHeight := c.h

		panelPadding := c.GetPropInt("padding", 2)
		cellPadding := c.GetPropInt("spacing", 2)

		_, minX, maxX, allCellPaddingX := c.makeColumnsInfo(fullWidth)
		columnsInfo, _, _, _ := c.makeColumnsInfo(fullWidth - (panelPadding + allCellPaddingX + panelPadding))

		_, minY, maxY, allCellPaddingY := c.makeRowsInfo(fullHeight)
		rowsInfo, _, _, _ := c.makeRowsInfo(fullHeight - (panelPadding + allCellPaddingY + panelPadding + c.insetTop))

		/*if strings.Contains(c.name, "Top") {
			fmt.Println("RowsInfo:")
			for yy := minY; yy <= maxY; yy++ {
				if rowInfo, ok := rowsInfo[yy]; ok {
					fmt.Printf("Row %d: minHeight=%d, maxHeight=%d, expandable=%t, height=%d, collapsed=%t\n",
						yy, rowInfo.minHeight, rowInfo.maxHeight, rowInfo.expandable, rowInfo.height, rowInfo.collapsed)
				}
			}
		}*/

		xOffset := panelPadding //+ c.LeftBorderWidth()
		for x := minX; x <= maxX; x++ {
			if colInfo, ok := columnsInfo[x]; ok {
				yOffset := panelPadding + c.insetTop
				for y := minY; y <= maxY; y++ {
					if rowInfo, ok := rowsInfo[y]; ok {
						w := c.getWidgetInGridCell(x, y)
						if w != nil {

							cX := xOffset
							cY := yOffset

							wWidth := colInfo.width
							if wWidth > w.MaxWidth() {
								wWidth = w.MaxWidth()
							}
							wHeight := rowInfo.height
							if wHeight > w.MaxHeight() {
								wHeight = w.MaxHeight()
							}

							// Place widget in the center of the cell
							//cX += (colInfo.width - wWidth) / 2
							//cY += (rowInfo.height - wHeight) / 2

							w.SetPosition(cX, cY)

							if w.IsVisible() {
								w.SetSize(wWidth, wHeight)
							} else {
								w.SetSize(0, 0)
							}
						}

						yOffset += rowInfo.height
						if rowInfo.height > 0 && y < maxY {
							yOffset += cellPadding
						}
					}
				}

				xOffset += colInfo.width
				if colInfo.width > 0 && x < maxX {
					xOffset += cellPadding
				}
			}
		}

		for _, w := range c.widgets {
			if !w.IsVisible() {
				w.SetSize(0, 0)
			}
		}

		if len(c.widgets) > 0 {
			// Set InnerSize
			innerWidth := 0
			innerHeight := 0

			for _, w := range c.widgets {
				if w.IsVisible() {
					if w.X()+w.Width() > innerWidth {
						innerWidth = w.X() + w.Width()
					}
					if w.Y()+w.Height() > innerHeight {
						innerHeight = w.Y() + w.Height()
					}
				}
			}

			if innerWidth < c.w {
				innerWidth = c.w
			}
			if innerHeight < c.h {
				innerHeight = c.h
			}
			c.innerWidth = innerWidth
			c.innerHeight = innerHeight
			c.checkScrolls()
		}

	}

	/*duration := time.Since(dt)
	prefix := ""
	for i := 0; i < updateLayoutStack; i++ {
		prefix += "."
	}
	fmt.Println(prefix+"Widget", c.name, "layout updated:", "type", c.typeName, "Width:", c.w, "Height:", c.h, "InnerWidth:", c.innerWidth, "InnerHeight:", c.innerHeight, "Duration:", duration)*/
}

func (c *Widget) makeColumnsInfo(fullWidth int) (map[int]*ContainerGridColumnInfo, int, int, int) {
	//fmt.Println("makeColumnsInfo", makeColumnsInfoCounter)

	// panelPadding := c.GetPropInt("padding", 2)
	cellPadding := c.GetPropInt("spacing", 2)

	minX := MaxInt
	minY := MaxInt

	maxX := MinInt
	maxY := MinInt

	// Detect range of grid coordinates
	for _, w := range c.widgets {
		if w.GridX() < minX {
			minX = w.GridX()
		}
		if w.GridX() > maxX {
			maxX = w.GridX()
		}
		if w.GridY() < minY {
			minY = w.GridY()
		}
		if w.GridY() > maxY {
			maxY = w.GridY()
		}
	}

	columnsInfo := make(map[int]*ContainerGridColumnInfo)
	hasExpandableColumns := false

	// Fill columnsInfo
	for x := minX; x <= maxX; x++ {
		var colInfo ContainerGridColumnInfo
		colInfo.minWidth = MinInt
		colInfo.maxWidth = MaxInt
		colInfo.expandable = false
		found := false

		for y := minY; y <= maxY; y++ {
			w := c.getWidgetInGridCell(x, y)
			if w != nil {
				if w.XExpandable() {
					colInfo.expandable = true // Found expandable by X
					hasExpandableColumns = true
				}
				found = true
			}
		}

		if colInfo.expandable {
			colInfo.minWidth = MinInt
			colInfo.maxWidth = MinInt

			for y := minY; y <= maxY; y++ {
				w := c.getWidgetInGridCell(x, y)
				if w != nil {
					wMinWidth := w.MinWidth()
					if wMinWidth > colInfo.minWidth {
						colInfo.minWidth = wMinWidth
					}
					wMaxWidth := w.MaxWidth()
					if wMaxWidth > colInfo.maxWidth {
						colInfo.maxWidth = wMaxWidth
					}
				}
			}

		} else {
			colInfo.minWidth = MinInt
			colInfo.maxWidth = MinInt

			for y := minY; y <= maxY; y++ {
				w := c.getWidgetInGridCell(x, y)
				if w != nil {
					wMinWidth := w.MinWidth()
					if wMinWidth > colInfo.minWidth {
						colInfo.minWidth = w.MinWidth()
					}
					if wMinWidth > colInfo.maxWidth {
						colInfo.maxWidth = w.MaxWidth()
					}
					/*if w.MaxWidth() < colInfo.maxWidth {
						colInfo.maxWidth = w.MaxWidth()
					}*/
				}
			}
		}

		if found {
			columnsInfo[x] = &colInfo
		}
	}

	if hasExpandableColumns {
		hasNonExpandable := false
		for _, colInfo := range columnsInfo {
			if !colInfo.expandable {
				hasNonExpandable = true
				break
			}
		}
		if hasNonExpandable {
			for _, colInfo := range columnsInfo {
				if !colInfo.expandable {
					colInfo.width = colInfo.minWidth
					colInfo.collapsed = true
				}
			}
		}
	}

	width := fullWidth

	for {
		readyWidth := 0
		for _, colInfo := range columnsInfo {
			readyWidth += colInfo.width
		}
		deltaWidth := width - readyWidth
		countOfColumnCanChange := 0
		for _, colInfo := range columnsInfo {
			if deltaWidth > 0 {
				if colInfo.width < colInfo.maxWidth {
					if !colInfo.collapsed {
						countOfColumnCanChange++
					}
				}
			} else {
				if deltaWidth < 0 {
					if colInfo.width > colInfo.minWidth {
						if !colInfo.collapsed {
							countOfColumnCanChange++
						}
					}
				}
			}
		}

		if countOfColumnCanChange > 0 && deltaWidth != 0 {
			pixForOne := deltaWidth / countOfColumnCanChange
			if math.Abs(float64(pixForOne)) < 1 {
				break
			}
			for _, colInfo := range columnsInfo {
				if !colInfo.collapsed {
					colInfo.width += pixForOne
				}
			}
		} else {
			break
		}

		for _, colInfo := range columnsInfo {
			if colInfo.width > colInfo.maxWidth {
				colInfo.width = colInfo.maxWidth
			}
			if colInfo.width < colInfo.minWidth {
				colInfo.width = colInfo.minWidth
			}
		}
	}

	allCellPadding := len(columnsInfo) - 1
	allCellPadding *= cellPadding
	if allCellPadding < 0 {
		allCellPadding = 0
	}

	return columnsInfo, minX, maxX, allCellPadding

}

func (c *Widget) makeRowsInfo(fullHeight int) (map[int]*ContainerGridRowInfo, int, int, int) {
	cellPadding := c.GetPropInt("spacing", 2)

	// Определяем минимальный и максимальный индекс строк
	minX := MaxInt
	minY := MaxInt
	maxX := MinInt
	maxY := MinInt
	for _, w := range c.widgets {
		if w.GridX() < minX {
			minX = w.GridX()
		}
		if w.GridX() > maxX {
			maxX = w.GridX()
		}
		if w.GridY() < minY {
			minY = w.GridY()
		}
		if w.GridY() > maxY {
			maxY = w.GridY()
		}
	}

	// Подготовка
	rowsInfo := make(map[int]*ContainerGridRowInfo)
	hasExpandableRows := false

	// Главный цикл по строкам
	for y := minY; y <= maxY; y++ {
		var rowInfo ContainerGridRowInfo
		rowInfo.minHeight = MinInt // Минимальная высота строки пока 0
		rowInfo.maxHeight = MaxInt // Максимальная высота строки пока ... максимум
		rowInfo.expandable = false // Пока думаем, что строка не мажорная
		found := false             // Признак того, что вообще есть в строке контролы

		// If any widget in the row is expandable, set the expandable flag for the row
		for x := minX; x <= maxX; x++ {
			w := c.getWidgetInGridCell(x, y)
			if w != nil {
				if w.YExpandable() {
					rowInfo.expandable = true // Found expandable by Y
					hasExpandableRows = true
				}
				found = true
			}
		}

		if rowInfo.expandable {
			rowInfo.minHeight = MinInt
			rowInfo.maxHeight = MinInt

			for x := minX; x <= maxX; x++ {
				w := c.getWidgetInGridCell(x, y)
				if w != nil {
					wMinHeight := w.MinHeight()
					if wMinHeight > rowInfo.minHeight {
						rowInfo.minHeight = wMinHeight
					}
					wMaxHeight := w.MaxHeight()
					if wMaxHeight > rowInfo.maxHeight {
						rowInfo.maxHeight = wMaxHeight
					}
				}
			}

		} else {
			rowInfo.minHeight = MinInt
			rowInfo.maxHeight = MinInt

			for x := minX; x <= maxX; x++ {
				w := c.getWidgetInGridCell(x, y)
				if w != nil {
					wMinHeight := w.MinHeight()
					if wMinHeight > rowInfo.minHeight {
						rowInfo.minHeight = wMinHeight
					}
					if wMinHeight > rowInfo.maxHeight {
						rowInfo.maxHeight = w.MaxHeight()
					}
					/*if w.MaxWidth() < colInfo.maxWidth {
						colInfo.maxWidth = w.MaxWidth()
					}*/
				}
			}
		}

		if found {
			rowsInfo[y] = &rowInfo
		}
	}

	if hasExpandableRows {
		hasNonExpandable := false
		for _, rowInfo := range rowsInfo {
			if !rowInfo.expandable {
				hasNonExpandable = true
				break
			}
		}
		if hasNonExpandable {
			for _, rowsInfo := range rowsInfo {
				if !rowsInfo.expandable {
					rowsInfo.height = rowsInfo.minHeight
					rowsInfo.collapsed = true
				}
			}
		}
	}

	height := fullHeight

	for {
		readyHeight := 0
		for _, rowInfo := range rowsInfo {
			readyHeight += rowInfo.height
		}
		deltaHeight := height - readyHeight
		countOfRowCanChange := 0
		for _, rowInfo := range rowsInfo {
			if deltaHeight > 0 {
				if rowInfo.height < rowInfo.maxHeight {
					if !rowInfo.collapsed {
						countOfRowCanChange++
					}
				}
			} else {
				if deltaHeight < 0 {
					if rowInfo.height > rowInfo.minHeight {
						if !rowInfo.collapsed {
							countOfRowCanChange++
						}
					}
				}
			}
		}

		if countOfRowCanChange > 0 && deltaHeight != 0 {
			pixForOne := deltaHeight / countOfRowCanChange
			if math.Abs(float64(pixForOne)) < 1 {
				break
			}
			for _, rowInfo := range rowsInfo {
				if !rowInfo.collapsed {
					rowInfo.height += pixForOne
				}
			}
		} else {
			break
		}

		for _, rowInfo := range rowsInfo {
			if rowInfo.height > rowInfo.maxHeight {
				rowInfo.height = rowInfo.maxHeight
			}
			if rowInfo.height < rowInfo.minHeight {
				rowInfo.height = rowInfo.minHeight
			}
		}
	}

	allCellPadding := len(rowsInfo) - 1
	allCellPadding *= cellPadding
	if allCellPadding < 0 {
		allCellPadding = 0
	}

	return rowsInfo, minY, maxY, allCellPadding
}

func (c *Widget) getWidgetInGridCell(x, y int) Widgeter {
	for _, w := range c.widgets {
		if w.GridX() == x && w.GridY() == y {
			if w.IsVisible() {
				return w
			}
		}
	}
	return nil
}

func (c *Widget) XExpandable() bool {
	if c.GetPropBool("xexpandable", false) {
		return true
	}

	if len(c.widgets) == 0 {
		return c.GetPropBool("xexpandable", false)
	}

	if c.layoutCacheXExpandableValid {
		return c.layoutCacheXExpandable
	}

	colsInfo, _, _, _ := c.makeColumnsInfo(1000)
	for _, ci := range colsInfo {
		if ci.expandable {
			c.layoutCacheXExpandableValid = true
			c.layoutCacheXExpandable = true
			return true
		}
	}

	c.layoutCacheXExpandableValid = true
	c.layoutCacheXExpandable = false

	return false
}

func (c *Widget) YExpandable() bool {
	if c.GetPropBool("yexpandable", false) {
		return true
	}

	if len(c.widgets) == 0 {
		return c.GetPropBool("yexpandable", false)
	}

	if c.layoutCacheYExpandableValid {
		return c.layoutCacheYExpandable
	}

	rowsInfo, _, _, _ := c.makeRowsInfo(1000)
	for _, ri := range rowsInfo {
		if ri.expandable {
			c.layoutCacheYExpandableValid = true
			c.layoutCacheYExpandable = true
			return true
		}
	}

	c.layoutCacheYExpandableValid = true
	c.layoutCacheYExpandable = false

	return false
}

// SetFontFamily sets the font of the widget, e.g. FontFamilyMono; an empty
// family returns the theme font.
func (c *Widget) SetFontFamily(family string) {
	c.SetProp("fontfamily", family)
}

func (c *Widget) FontFamily() string {
	if family, ok := c.GetProp("fontfamily").(string); ok && family != "" {
		return family
	}
	return ThemeFontFamily()
}

func (c *Widget) SetFontSize(fontSize float64) {
	c.SetProp("fontsize", fontSize)
}

func (c *Widget) FontSize() float64 {
	if c.GetProp("fontsize") != nil {
		if fontSize, ok := c.GetProp("fontsize").(float64); ok {
			return fontSize
		}
	}
	return ThemeFontSize()
}

func (c *Widget) ForegroundColor() color.Color {
	if c.foregroundColor != nil {
		return c.foregroundColor
	}
	return ThemeForegroundColor(c.Role())
}

func (c *Widget) ForegroundColorDisabled() color.Color {
	return ThemeForegroundColorDisabled()
}

func (c *Widget) CurrentElevation() int {
	summedElevation := 0
	for _, wId := range c.FullPath() {
		wId := c.form.WidgetById(wId)
		if wId != nil {
			summedElevation += wId.Elevation()
		}
	}
	return summedElevation
}

func (c *Widget) BackgroundColorWithAddElevation(elevation int) color.Color {
	if c.backgroundColor != nil {
		return c.backgroundColor
	}
	summedElevation := elevation
	for _, wId := range c.FullPath() {
		wId := c.form.WidgetById(wId)
		if wId != nil {
			summedElevation += wId.Elevation()
		}
	}
	return ThemeBackgroundColor(summedElevation, c.Role())
}

func (c *Widget) BackgroundColor() color.Color {
	if c.backgroundColor != nil {
		return c.backgroundColor
	}
	summedElevation := 0
	for _, wId := range c.FullPath() {
		wId := c.form.WidgetById(wId)
		if wId != nil {
			summedElevation += wId.Elevation()
		}
	}
	return ThemeBackgroundColor(summedElevation, c.Role())
}

func (c *Widget) BackgroundColorForRole(role string) color.Color {
	return ThemeBackgroundColor(c.CurrentElevation(), role)
}

func (c *Widget) SetContextMenu(menu *ContextMenu) {
	c.contextMenu = menu
	c.contextMenu.attachToForm(c.contextMenu, c.form)
}

func (c *Widget) ContextMenu() *ContextMenu {
	return c.contextMenu
}

func (c *Widget) ParentWidget() Widgeter {
	parentWidgetId := c.parentWidgetId
	if parentWidgetId == "" {
		return nil
	}
	return c.form.WidgetById(parentWidgetId)
}

func (c *Widget) RectClientAreaOnWindow() (x, y int) {
	x = c.X()
	y = c.Y()
	// A popup widget is positioned in the form's client coordinates, not in
	// those of the form's panel it's registered under (the panel is below
	// the menu bar)
	if c.isOpenPopupWidget() {
		return x, y
	}
	parentWidget := c.ParentWidget()
	if parentWidget != nil {
		xx, yy := parentWidget.RectClientAreaOnWindow()
		x += xx
		y += yy

		x -= parentWidget.ScrollX()
		y -= parentWidget.ScrollY()
	}

	return x, y
}

func (c *Widget) isOpenPopupWidget() bool {
	if c.form == nil || c.form.topWidget == nil {
		return false
	}
	for _, w := range c.form.topWidget.PopupWidgets {
		if w.Id() == c.id {
			return true
		}
	}
	return false
}

func (c *Widget) ClearLayoutCache() {
	c.layoutCacheXExpandableValid = false
	c.layoutCacheYExpandableValid = false
	c.layoutCacheMinWidthValid = false
	c.layoutCacheMinHeightValid = false

	for _, w := range c.Widgets() {
		w.ClearLayoutCache()
	}

	for _, popupWidget := range c.PopupWidgets {
		popupWidget.ClearLayoutCache()
	}
}

type uiNode struct {
	XMLName   xml.Name
	Attrs     []xml.Attr `xml:",any,attr"`
	Nodes     []uiNode   `xml:",any"`
	InnerText string     `xml:",chardata"`
}

func (c *uiNode) GetChildByName(name string) *uiNode {
	for index, child := range c.Nodes {
		if child.XMLName.Local == name {
			return &c.Nodes[index]
		}
	}
	return nil
}

func (c *uiNode) GetAttrByName(name string) *xml.Attr {
	for index, attr := range c.Attrs {
		if attr.Name.Local == name {
			return &c.Attrs[index]
		}
	}
	return nil
}

func (c *uiNode) GetAttrValueByName(name string, defaultValue string) string {
	for _, attr := range c.Attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return defaultValue
}

func (c *uiNode) GetAttrValueByNameInt(name string, defaultValue int) int {
	for _, attr := range c.Attrs {
		if attr.Name.Local == name {
			if intValue, err := strconv.Atoi(attr.Value); err == nil {
				return intValue
			}
			return defaultValue
		}
	}
	return defaultValue
}

func (c *Widget) SetLayout(layoutAsXml string, eventProcessor interface{}, widgets map[string]Widgeter) error {
	var n uiNode
	err := xml.Unmarshal([]byte(layoutAsXml), &n)
	if err != nil {
		return fmt.Errorf("failed to parse layout XML: %v", err)
	}
	c.buildNode(&n, c, 0, 0, eventProcessor, widgets)
	return nil
}

func (c *Widget) buildNode(n *uiNode, parent Widgeter, row int, col int, eventProcessor interface{}, widgets map[string]Widgeter) {
	var w Widgeter
	isRow := false
	switch n.XMLName.Local {
	case "frame":
		w = NewFrame()
	case "panel":
		w = NewPanel()
	case "column":
		w = NewPanel()
	case "row":
		w = NewPanel()
		isRow = true
	case "label":
		w = NewLabel("")
	case "button":
		w = NewButton("")
	case "hspacer":
		w = NewHSpacer()
	case "vspacer":
		w = NewVSpacer()
	case "space":
		w = NewSpace()
	case "textbox":
		w = NewTextBox()
	case "imagebox":
		w = NewImageBox()
	case "checkbox":
		w = NewCheckbox("")
	case "radiobutton":
		w = NewRadioButton("")
	case "combobox":
		w = NewComboBox()
	case "table":
		{
			w = NewTable()
			table := w.(*Table)
			table.TableSetLayoutXml(n)
		}
	case "tabwidget":
		{
			w = NewTabWidget()
			tabWidget := w.(*TabWidget)
			tabWidget.SetLayoutXml(n, eventProcessor, widgets)
		}
	case "scrollarea":
		w = NewScrollArea()
	case "progressbar":
		w = NewProgressBar(0, 100, 0)
	case "widget":
		{
			// <widget id="InnerWidget" />
			var widgetId string
			for _, attr := range n.Attrs {
				if attr.Name.Local == "id" {
					widgetId = attr.Value
					break
				}
			}
			if widgetId != "" {
				if existingWidget, ok := widgets[widgetId]; ok {
					w = existingWidget
				} else {
					fmt.Printf("Widget with id %s not found in widgets map\n", widgetId)
					return
				}
			} else {
				fmt.Println("Widget tag without id attribute")
				return
			}
		}
	default:
		return
	}

	if parent != nil {
		parent.AddWidget(row, col, w)
	}

	// Set attributes - only after adding to parent
	for _, attr := range n.Attrs {
		if attr.Name.Local == "id" && n.XMLName.Local != "widget" {
			w.SetName(attr.Value)
			continue
		}
		if strings.HasPrefix(attr.Name.Local, "on") {
			fmt.Println("Set ON:", attr.Value)
			if attr.Value == "BtnUsualButton" {
				fmt.Println("Found BtnUsualButton event handler")
			}

			// If event processor is map[string]func(), then get function from map
			if eventProcessorMap, ok := eventProcessor.(map[string]func()); ok {
				if eventHandler, ok := eventProcessorMap[attr.Value]; ok {
					w.SetPropFunction(attr.Name.Local, eventHandler)
					continue
				}
			}

			if eventProcessor != nil {
				// Get function by name from eventProcessor
				method := reflect.ValueOf(eventProcessor).MethodByName(attr.Value)
				if method.IsValid() {
					w.SetPropFunction(attr.Name.Local, method.Interface().(func()))
				} else {
					fmt.Printf("Event handler %s not found in eventProcessor\n", attr.Value)
				}
			}
		} else {
			w.SetProp(attr.Name.Local, attr.Value)
		}
	}

	index := 0
	// Recursively build child nodes
	for _, childNode := range n.Nodes {
		if isRow {
			c.buildNode(&childNode, w, 0, index, eventProcessor, widgets)
			index++
		} else {
			c.buildNode(&childNode, w, index, 0, eventProcessor, widgets)
			index++
		}
	}
}

func (c *Widget) AllChildren() []Widgeter {
	var all []Widgeter
	for _, w := range c.widgets {
		all = append(all, w)
		if len(w.Widgets()) > 0 {
			all = append(all, w.AllChildren()...)
		}
	}
	return all
}

func (c *Widget) nextFocus(reverse bool) {
	fmt.Println("Widget::netFocus")
	children := c.AllChildren()
	if len(children) == 0 {
		return
	}

	focusedWidgetIndex := -1
	var focusableWidgets []Widgeter
	for _, w := range children {
		if w.IsCanBeFocused() && w.IsVisible() {
			focusableWidgets = append(focusableWidgets, w)
			if c.form.FocusedWidget().Id() == w.Id() {
				focusedWidgetIndex = len(focusableWidgets) - 1
			}
		}
	}

	fmt.Println("Focused widget index:", focusedWidgetIndex)

	for i, w := range focusableWidgets {
		fmt.Println("Focusable widget:", i, w.Id(), "Type:", w.TypeName())
	}

	if len(focusableWidgets) == 0 {
		return
	}

	nextIndexToFocus := focusedWidgetIndex
	for {
		if reverse {
			nextIndexToFocus--
			if nextIndexToFocus < 0 {
				nextIndexToFocus = len(focusableWidgets) - 1
			}
		} else {
			nextIndexToFocus++
			if nextIndexToFocus >= len(focusableWidgets) {
				nextIndexToFocus = 0
			}
		}
		if nextIndexToFocus == focusedWidgetIndex {
			return
		}
		if focusableWidgets[nextIndexToFocus].IsVisible() {
			focusableWidgets[nextIndexToFocus].Focus()
			return
		}
	}

}

// left, right, up, down focus navigation
func (c *Widget) nextFocusByDirection(dir string) {
	children := c.AllChildren()
	if len(children) == 0 {
		return
	}

	focusWidget := c.form.FocusedWidget()

	var focusableWidgets []Widgeter
	for _, w := range children {
		if w.IsCanBeFocused() && w.IsVisible() {
			focusableWidgets = append(focusableWidgets, w)
		}
	}

	if focusWidget == nil || len(focusableWidgets) == 0 {
		return
	}

	focusedWidgetClientX, focusedWidgetClientY := focusWidget.RectClientAreaOnWindow()
	focusXAvg := focusedWidgetClientX + focusWidget.Width()/2
	focusYAvg := focusedWidgetClientY + focusWidget.Height()/2

	// find nearest focusable widget above the currently focused widget
	var nearestWidget Widgeter
	minDistance := 1000000000.0
	for _, w := range focusableWidgets {
		if w.Id() == focusWidget.Id() {
			continue
		}
		wClientX, wClientY := w.RectClientAreaOnWindow()
		wXAvg := wClientX + w.Width()/2
		wYAvg := wClientY + w.Height()/2

		distance := math.Sqrt(math.Pow(float64(wXAvg-focusXAvg), 2) + math.Pow(float64(wYAvg-focusYAvg), 2))
		if dir == "up" && wYAvg < focusYAvg && distance < minDistance {
			if wXAvg > focusedWidgetClientX && wXAvg < focusedWidgetClientX+focusWidget.Width() {
				minDistance = distance
				nearestWidget = w
			}
		}
		if dir == "down" && wYAvg > focusYAvg && distance < minDistance {
			if wXAvg > focusedWidgetClientX && wXAvg < focusedWidgetClientX+focusWidget.Width() {
				minDistance = distance
				nearestWidget = w
			}
		}
		if dir == "left" && wXAvg < focusXAvg && distance < minDistance {
			if wYAvg > focusedWidgetClientY && wYAvg < focusedWidgetClientY+focusWidget.Height() {
				minDistance = distance
				nearestWidget = w
			}
		}
		if dir == "right" && wXAvg > focusXAvg && distance < minDistance {
			if wYAvg > focusedWidgetClientY && wYAvg < focusedWidgetClientY+focusWidget.Height() {
				minDistance = distance
				nearestWidget = w
			}
		}
	}

	if nearestWidget != nil {
		nearestWidget.Focus()
	}
}

func (c *Widget) setPreviousFocusedWidget(w Widgeter) {
	c.previousFocusedWidget = w
}

func (c *Widget) getPreviousFocusedWidget() Widgeter {
	return c.previousFocusedWidget
}
