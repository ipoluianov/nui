# Widget (base)

Most controls embed `ui.Widget`, which provides:

- **Layout**: size/position, min/max size (`SetMinSize`, `SetMaxSize`, `SetMinWidth`...), anchors, expand flags (`SetXExpandable`, `SetYExpandable`), see [Layouts](layouts.md)
- **Child widgets**: `AddWidget(row, col, w)`, `AddWidgetSpan`, `RemoveWidget`, `RemoveAllWidgets`, `Widgets()`, and helpers `AddLabel`, `AddButton`, `AddPanel`, `AddHSpacer`, `AddVSpacer`
- **State**: `SetEnabled`, `SetVisible`, `SetTooltip`, `SetContextMenu`, `SetMouseCursor`, `SetName` (found by `FindWidgetByName`)
- **Fonts and colors**: `SetFontFamily`, `SetFontSize`, `SetFontBold`, `SetForegroundColor`, `SetBackgroundColor`
- **Properties**: `SetProp(...)` / `GetProp...(...)`
- **Signals**: `OnClick`, `OnMouseDown`, `OnKeyDown`, `OnResize`... (see [Events](events.md))
- **Timers**: `AddTimer`, `NewTimer`, `AfterFunc` (see [Timer](timer.md))
- **Drag and drop**: `SetDragSource`, `SetDropTarget` (see [Drag and drop](dragdrop.md))
- **Scrolling**: internal scroll offsets and scrollbars (`SetAllowScroll(x, y)`, `ScrollX()`, `SetScrollY(...)`)

## Properties

Properties are simple key-value pairs stored inside a widget:

```go
w.SetProp("role", "primary")
w.SetProp("padding", 8)
text := w.GetPropString("text", "")
```

When `SetProp` is used, `ProcessPropChange(key, value)` is called on that widget.

## Events

A widget tells what happens to it by signals, fields `On…` of the type `ui.Signal[T]`; any number of handlers can be connected (see [Events](events.md)):

```go
btn := ui.NewButton("OK")
btn.OnClick.Connect(func(ui.Widgeter) { save() })
panel.OnResize.Connect(func(s ui.Size) { relayout(s.Width) })
```

The old setters still work; each keeps one handler:

- **Click**: `SetOnClick(func())`
- **Mouse**: `SetOnMouseDown/Up/Move`, `SetOnMouseDblClick`, `SetOnMouseWheel`, `SetOnMouseEnter/Leave`
- **Keyboard**: `SetOnKeyDown/Up`, `SetOnChar`
- **Focus**: `SetOnFocused`, `SetOnFocusLost`. Tab and Shift+Tab go over the visible and enabled widgets that can be focused (`SetCanBeFocused`); `Focus()` gives the focus to a widget.

## Painting

`SetOnPaint(func(*ui.Canvas))` draws the widget (see [Canvas](canvas.md)); `SetOnPostPaint` draws over its children.

A frame repaints only the rectangles of the widgets that asked for it, over the last frame. After a click, a key or a new layout the whole form is repainted; after a mouse move, a wheel turn or a timer only the widgets that asked. So a widget that draws its own data calls `Update()` when the data changes:

```go
func (c *Gauge) SetValue(v float64) {
	c.value = v
	c.Update() // repaint the widget with the next frame
}
```

The setters of the nui widgets do this themselves. `Update()` is for the UI thread; from another goroutine use `RequestRepaint()`. `Form().Update()` repaints the whole form.
