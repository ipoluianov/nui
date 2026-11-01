# Splitter

Two panes with a draggable handle between them.
`NewHSplitter` puts the panes left and right, `NewVSplitter` top and bottom.

The pane whose size was set last (`SetFirstSize` / `SetSecondSize`) keeps it
when the splitter is resized; the other pane takes the rest.
Dragging respects the panes' minimum sizes. A hidden pane gives its space to the other one.

```go
sp := ui.NewHSplitter()
sp.SetWidgets(list, details)
sp.SetSecondSize(400) // details keep 400px when the window is resized
sp.OnActivated.Connect(func(size int) {
	saveWidth(size)
})
```

- `SetWidgets(first, second)`, `First()`, `Second()`.
- `SetFirstSize(px)` / `FirstSize()`, `SetSecondSize(px)` / `SecondSize()`: the size of a pane (the width for `NewHSplitter`, the height for `NewVSplitter`).
- `SetHandleSize(px)` / `HandleSize()`: the thickness of the handle.
- `OnActivated` (`int`): the user dragged the handle; the size of the fixed pane, e.g. to save it in the [settings](application.md). `SetFirstSize`/`SetSecondSize` don't fire it. The old `SetOnSplitChanged(f)` still works.
- `IsVertical()`: true for `NewVSplitter`.
