# View2D

`View2D` shows a drawing of any size in its own units - millimeters, meters,
the coordinates of a map - and lets the user move around it as the views of
CAD programs do. The application draws in the drawing's coordinates; the
view zooms, pans, draws the grid and the crosshair, selects by a box, and
keeps the picture while only the overlay changes.

Run `go run . cad` for a small CAD program built on it.

```go
view := ui.NewView2D()
view.SetCrosshair(ui.CrosshairFull)
view.SetBoxSelection(true)
view.SetPainter(func(p *ui.View2DPaint) {
	line := ui.StrokeStyle{Width: 1, NonScaling: true}
	path := ui.NewPath()
	index.Search(p.Visible, func(s *Shape) bool {
		s.AddTo(path)
		return true
	})
	p.Canvas.StrokePath(path, line, ink)
})
view.SetExtents(drawingBounds)
form.Panel().AddWidget(0, 0, view)
form.Invoke(view.ZoomExtents) // once the view has its size
```

## Coordinates

y goes up, as in CAD and in mathematics (`SetYUp(false)`: down, as in
images). The view is a center and a scale:

| Method | |
|---|---|
| `Center`, `SetCenter(p)`, `Scale`, `SetScale(s)`, `SetView(center, scale)` | the point at the middle of the view; the pixels a unit takes |
| `ZoomAt(factor, x, y)` | zoom keeping the point of the widget in place |
| `ZoomToRect(r)`, `SetExtents(r)`, `ZoomExtents()` | show a rectangle; show all of the drawing |
| `StartZoomWindow()` | the next drag of the left button is the box to zoom to |
| `SetScaleRange(min, max)` | the limits of the zoom |
| `ToWorld(x, y)`, `ToScreen(p)`, `WorldMatrix()` | convert between the widget and the drawing |
| `VisibleRect()` | the part of the drawing seen |
| `Pointer()`, `PointerScreen()` | the pointer, in the drawing and in the widget |

The coordinates are `float64` all the way to the pixels: a drawing a
kilometer wide zoomed to a micrometer is drawn exactly (see
[canvas.md](canvas.md), "Large coordinates").

## The user

| Mouse and keys | |
|---|---|
| wheel | zoom towards the pointer |
| middle button drag | pan (`SetPanWithLeftButton(true)`: the left one too) |
| middle button double click | `ZoomExtents` |
| left drag (`SetBoxSelection(true)`) | a selection box: to the right solid, selects what is inside; to the left dashed, what it crosses - `OnBoxSelected` |
| arrows, `+`, `-`, Home | pan, zoom, `ZoomExtents` |
| Esc | cancel the box |

A press the application takes is not a box: connect to `view.OnMouseDown`
and set `e.Handled` when the press picks a shape or gives a point to a
tool. `SetControlsEnabled(false)` makes a view the user cannot move.

## Painting

- `SetPainter(f)` draws the drawing. `p.Canvas` is transformed to the
  drawing's coordinates; `p.Visible` is the part to draw (a strip of the
  view while it is dragged); `p.Scale`, `p.PixelSize()`, `p.ToScreen(pt)`.
  `p.InScreen(func(cnv))` draws in the widget's pixels: markers, grips,
  labels that keep their size.
- The picture is kept: it is painted again when the view moves or zooms,
  and after `Invalidate()` - call it when the drawing changes. When the
  view is dragged, the picture moves and only the strip that comes in is
  painted.
- `SetOverlay(f)` draws over the picture on every paint: the shape under
  the pointer, the rubber band of a tool, the snap marker. `Update()`
  repaints the overlay only.
- Lines: `StrokeStyle{NonScaling: true}` keeps their width and dashes in
  pixels at any zoom.
- Text: `view.AddText(path, at, text, font, height, angle, hAlign, vAlign)`
  adds the outlines of the text, `height` units tall, reading upright
  whichever way y goes; fill the path.
- `RenderImage(w, h, pixelScale)` and `RenderSVG(w, h)` paint the drawing
  around the view's center at its scale, without the grid, the axes and the
  overlay: to save, to copy, to print.

## The look

| Method | |
|---|---|
| `SetBackground(c)` | the sheet (zero: the theme's) |
| `SetGrid(ui.View2DGrid{Visible, Step, Major, Dots, Color, MajorColor})` | a step of 0 picks 1, 2 or 5 times a power of ten for the zoom |
| `SetAxesVisible(b)` | the axes through the origin, x red and y green |
| `SetCrosshair(ui.CrosshairNone / CrosshairSmall / CrosshairFull)`, `SetPickBox(n)` | the pointer: the arrow, a cross, or lines across the view with a pick box |

Signals: `OnViewChanged` (the visible rectangle), `OnPointerMoved` (the
point of the drawing under the pointer), `OnBoxSelected`
(`ui.BoxSelection{Rect, Crossing, Mods}`).

## RectF and SpatialIndex

`ui.RectF{MinX, MinY, MaxX, MaxY}` is a rectangle of the drawing:
`RectFromPoints`, `EmptyRectF`, `Contains`, `ContainsRect`, `Intersects`,
`Union`, `AddPoint`, `Inset`, `Center`, `Width`, `Height`.

`ui.SpatialIndex[T]` finds the shapes in an area among many - what to draw,
what is under the pointer, what is in a selection box - without looking at
the others:

```go
index := ui.NewSpatialIndex[*Shape]()
index.Insert(s, s.Bounds()) // again to move it; Remove(s)
index.Search(area, func(s *Shape) bool { ...; return true })
near := index.Nearest(point, tolerance, 16) // nearest boxes first
```

It is a quadtree that grows to hold items anywhere. Searching a view of
1000 × 600 among 100 000 shapes takes about 25 µs.

## What it costs

On a 4-core desktop CPU of 2014 (Core i5-4460), a view of 1060 × 700 pixels
of the CAD example:

| | |
|---|---|
| the sample drawing, all of it | 4-7 ms |
| 100 000 shapes, all in view (the smallest drawn as dots) | about 100 ms |
| 100 000 shapes, zoomed in | 1-4 ms |
| a drag of the view | the strip only: a few ms |
| a move of the pointer | the overlay only: the picture is copied |
