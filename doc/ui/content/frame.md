# Frame

A container with a background fill and elevation (looks like a card/panel surface). Its children are laid out on the grid like in a [Panel](panel.md).

## Create

```go
f := ui.NewFrame()
f.AddLabel(0, 0, "Inside frame")
f.AddWidget(1, 0, ui.NewTextBox())
```

A frame is a panel with `SetElevation(1)` and `SetAutoFillBackground(true)`; `SetElevation(2)` raises it more. `SetPanelPadding` and `SetCellPadding` set the room inside.
