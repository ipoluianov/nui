# ZoomView and ScrollBar

## ZoomView

Content larger or smaller than the view, zoomed and moved by the user: an image, a drawing, a map, a diagram.

```go
view := ui.NewZoomView()
view.SetImage(photo)     // or SetContentSize + SetPainter for a drawing
view.SetSmooth(false)    // sharp pixels: pixel art, a bitmap being edited
view.FitToView()
view.OnZoomChanged.Connect(func(zoom float64) { status.SetText(fmt.Sprintf("%.0f%%", zoom*100)) })
```

- The wheel scrolls (Shift+wheel sideways), **Ctrl+wheel zooms around the mouse**: the point under it stays.
- The middle button drags the content; with `SetPanWithLeftButton(true)` the left one too.
- `+`, `-` zoom, `0` shows the actual size.
- `SetZoom`, `ZoomAt(zoom, x, y)`, `FitToView`, `ActualSize`, `CenterOn(x, y)`, `SetZoomRange`.
- `ToContent(x, y)` and `ToView(x, y)` convert between the view and the content (a click on a pixel of the image: `view.ToContent(e.X, e.Y)`).

A painter draws in the view's canvas, converting the content's coordinates:

```go
view.SetContentSize(1000, 800)
view.SetPainter(func(p *ui.ZoomPaint) {
	x, y := p.ToView(100, 100)
	p.Canvas.FillRect(x, y, int(50*p.Zoom), int(50*p.Zoom), red)
})
```

`Canvas.DrawImageRect(x, y, w, h, img, src, smooth)` draws a part of an image stretched to a rectangle, computing the visible part only.

## ScrollBar

A scroll bar of its own, for a view that scrolls in units of its own — the rows of a hex editor, the samples of a wave — rather than in the pixels of a widget's content (those have their bars built in).

```go
bar := ui.NewScrollBar(true)              // vertical
bar.SetRange(0, rows-visibleRows)         // the value: the first row shown
bar.SetPageStep(visibleRows)
bar.OnValueChanged.Connect(func(row int) { view.SetFirstRow(row) })
```

The arrows move the value by `SetStep` (1), the track by the page, the wheel by three steps. `OnValueChanged` is emitted on any change, `OnActivated` on the user's.
