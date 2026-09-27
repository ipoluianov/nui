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
sp.SetOnSplitChanged(func() {
	saveWidth(sp.SecondSize())
})
```
