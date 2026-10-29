# Canvas

A widget draws in its `SetOnPaint` handler on a `*ui.Canvas`. Its
coordinates are logical pixels: on a screen of the scale 2 everything is
drawn twice as dense, the code stays the same.

## Rectangles, lines, text, images

`FillRect`, `DrawRect`, `FillRoundedRect`, `DrawRoundedRect`, `FillFrame`,
`DrawLine`, `DrawLineF`, `FillTriangle`, `DrawText` (with `SetHAlign`,
`SetVAlign`, `SetFontSize`...), `DrawImage`, `DrawImageRect`, and
`Save`/`Restore`, `TranslateAndClip`.

## Paths

A `Path` is any shape of lines, curves and arcs, filled or stroked with
antialiasing. Its coordinates are fractional, and they are the edges of the
pixels, as in SVG: a line one wide along `y = 3.5` covers a row of whole
pixels.

```go
p := ui.NewPath()
p.MoveTo(10, 40)
p.LineTo(60, 10)
p.QuadTo(80, 0, 100, 40)
p.CubicTo(110, 60, 130, 60, 140, 40)
p.Arc(170, 40, 20, math.Pi, 2*math.Pi)
p.Close()
cnv.FillPath(p, fill)
cnv.StrokePath(p, ui.StrokeStyle{Width: 2, Join: ui.LineJoinRound, Dash: []float64{6, 3}}, line)
```

| Method | Shape |
|---|---|
| `MoveTo`, `LineTo`, `QuadTo`, `CubicTo`, `Close` | the pieces |
| `Arc`, `EllipticalArc` | an arc; angles in radians, growing clockwise |
| `Rect`, `RoundedRect`, `Circle`, `Ellipse` | closed shapes |
| `Polygon`, `Polyline` | through points |
| `Sector(cx, cy, inner, outer, start, end)` | a slice of a pie, a donut, a gauge |

`StrokeStyle` sets the width, the joins (`ui.LineJoinMiter`, `Round`,
`Bevel`), the ends (`ui.LineCapButt`, `Round`, `Square`), the miter limit and
the dashes. A translucent line that crosses itself is blended once. A
subpath drawn the other way round inside another one is a hole.

Shortcuts: `FillCircle`, `DrawCircle`, `FillPolygon`, `DrawPolyline`.

## Paints

`FillPathPaint` fills with a color per point:

- `ui.LinearGradient{X1, Y1, X2, Y2, Stops}` and `ui.RadialGradient{CX, CY, R, Stops}`;
- `ui.HatchPaint{Color, Background, Angle, Spacing, Width}` — stripes, for
  unavailable areas and bad data;
- any type with `ColorAt(x, y float64) color.RGBA`.

## Markers

`DrawMarker(shape, x, y, size, fill, stroke)` draws `ui.MarkerCircle`,
`MarkerSquare`, `MarkerDiamond`, `MarkerTriangleUp`, `MarkerTriangleDown`,
`MarkerCross`, `MarkerPlus`, `MarkerStar`: the points of a chart, the status
dots of a table, the entries of a legend.

## Turned text

`DrawTextRotated(x, y, angle, text)` draws a line of text turned around the
point that `SetHAlign` and `SetVAlign` name: `-math.Pi/2` reads upwards (the
title of a vertical axis). A quarter turn is exact, other angles smoothed.
`MeasureTextRotated` returns the box it takes.

## SVG

`SetRecorder(ui.NewSVGRecorder(w, h))` makes the canvas write what it draws
as SVG too: paths, fills with gradients and hatches, strokes with dashes,
rectangles, lines and texts. Paint a widget into such a canvas and write the
recorder out: a picture sharp at any size. Images are not recorded.
