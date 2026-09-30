# Drag and drop

Any widget can be a place to drag data from (a drag source) or to drop it on (a drop target).
The drag starts when the mouse moves a few pixels with the left button down; the target under
the mouse is highlighted when it accepts the data, and the dragged text follows the mouse.
Escape cancels. The release after a drag doesn't click the widget it started on.

```go
item.SetDragSource(func(x, y int) *ui.DragData {
	return &ui.DragData{Text: "Apple", Value: fruit} // nil: no drag from (x, y)
})

basket.SetDropTarget(
	func(d *ui.DragData, x, y int) bool { _, ok := d.Value.(Fruit); return ok }, // nil accepts all
	func(d *ui.DragData, x, y int) { basket.Add(d.Value.(Fruit)) },
)
```

`DragData`: `Text` (shown while dragging), `Files`, `Value` (any application data), `Source`
(the widget the drag started from). Coordinates are in the widget's own coordinates. The widget
under the mouse or its nearest parent that accepts the data is the target.

## Files from the system

Files dragged from the file manager (Explorer, Finder, Dolphin/Nautilus...) come to the drop
targets as `DragData.Files`. Files dropped where no target accepts them go to the form:

```go
form.SetOnFilesDropped(func(files []string, x, y int) {
	for _, f := range files { open(f) }
})
```

Platforms: XDND on Linux (X11), `WM_DROPFILES` on Windows, `NSDraggingDestination` on macOS.
