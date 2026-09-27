# Widget (base)

Most controls embed `ui.Widget`, which provides:

- **Layout**: size/position, min/max size, anchors, expand flags
- **Child widgets**: `AddWidget`, `AddWidgetOnGrid`, `RemoveWidget`, `RemoveAllWidgets`
- **Properties**: `SetProp(...)` / `GetProp...(...)`
- **Callbacks**: `SetOnPaint`, mouse/keyboard handlers, focus handlers
- **Scrolling**: internal scroll offsets and scrollbars (when enabled)

## Properties

Properties are simple key-value pairs stored inside a widget:

```go
w.SetProp("role", "primary")
w.SetProp("padding", 8)
text := w.GetPropString("text", "")
```

When `SetProp` is used, `ProcessPropChange(key, value)` is called on that widget.

## Events / callbacks

Callbacks are registered via `SetOn...` methods (or directly with `SetPropFunction`):

```go
btn := ui.NewButton("OK")
btn.SetOnClick(func() {
  // ...
})
```

Some widgets push an event payload into `ui.CurrentEvent()` before calling the callback.

## Common callbacks

- **Paint**: `SetOnPaint(func(*ui.Canvas))`
- **Mouse**: `SetOnMouseDown/Up/Move`, `SetOnMouseWheel`, `SetOnMouseEnter/Leave`
- **Keyboard**: `SetOnKeyDown/Up`, `SetOnChar`
- **Focus**: `SetOnFocused`, `SetOnFocusLost`

