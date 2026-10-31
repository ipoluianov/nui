# ScrollArea

A scrollable container: its children are laid out on the grid like in a [Panel](panel.md), and scrollbars appear when they need more room than the view. Both directions are enabled, and the area is expandable in X and Y.

## Create

```go
sa := ui.NewScrollArea()
sa.AddWidget(0, 0, bigWidget)
sa.SetAllowScroll(false, true) // only vertically
```

- `ScrollX()`, `ScrollY()`, `SetScrollX(x)`, `SetScrollY(y)`: the scroll offsets in pixels.
- `ScrollEnsureVisible(x, y)`: scroll so that the point of the content is in the view.
- `SetOnScrollChanged(func(scrollX, scrollY int))`: the user or the code scrolled.
- The mouse wheel over the area scrolls it; a child that scrolls itself (a text box, a list) takes the wheel first.
