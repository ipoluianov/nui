# Canvas

A widget draws in its `SetOnPaint` handler on a `*ui.Canvas`. Its
coordinates are logical pixels: on a screen of the scale 2 everything is
drawn twice as dense, the code stays the same.

The canvas draws 2D shapes, text and images with antialiasing, and 3D scenes
(see [3d.md](3d.md)). Run `go run . canvas` for a demo of most of what is
described here.

## Rectangles, lines, text, images

`FillRect`, `DrawRect`, `FillRoundedRect`, `DrawRoundedRect`, `FillFrame`,
`DrawLine`, `DrawLineF`, `FillTriangle`, `DrawText` (with `SetHAlign`,
`SetVAlign`, `SetFontSize`...), `DrawImage`, `DrawImageRect`, and
`Save`/`Restore`, `TranslateAndClip`.

These methods take whole logical pixels and stay pixel-sharp: a rectangle
filled at whole coordinates covers whole pixels.

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
| `ArcTo(x1, y1, x2, y2, r)` | a rounded corner between two lines, as `arcTo` of the HTML canvas |
| `SVGArcTo(rx, ry, rotation, largeArc, sweep, x, y)` | the `A` command of SVG |
| `Rect`, `RoundedRect`, `Circle`, `Ellipse` | closed shapes |
| `Polygon`, `Polyline` | through points |
| `RegularPolygon(cx, cy, r, n, rotation)`, `Star(cx, cy, outer, inner, points)` | polygons and stars |
| `Spline(pts, closed, tension)` | a smooth curve through the points (Catmull-Rom) |
| `Sector(cx, cy, inner, outer, start, end)` | a slice of a pie, a donut, a gauge |
| `AddPath(q)` | the subpaths of another path |

`StrokeStyle` sets the width, the joins (`ui.LineJoinMiter`, `Round`,
`Bevel`), the ends (`ui.LineCapButt`, `Round`, `Square`), the miter limit and
the dashes. A translucent line that crosses itself is blended once.
`NonScaling: true` keeps the width and the dashes in pixels under the
canvas' transform: the path is transformed, the line is not - the lines of
a drawing stay as thick at any zoom (SVG's
`vector-effect="non-scaling-stroke"`).

Arcs, circles and ellipses are kept as arcs, not as curves near them: they
are exact at any scale and under any transform.

Shortcuts: `FillCircle`, `DrawCircle`, `FillPolygon`, `DrawPolyline`.

### Fill rules

Where the subpaths of a path overlap, its fill rule decides what is inside:

- `ui.FillRuleNonZero` (the default): a subpath drawn the other way round
  inside another one is a hole;
- `ui.FillRuleEvenOdd`: every subpath inside another one is a hole,
  whatever its direction (SVG `fill-rule="evenodd"`).

```go
ring := ui.NewPath()
ring.Circle(50, 50, 40)
ring.Circle(50, 50, 20)
ring.SetFillRule(ui.FillRuleEvenOdd)
cnv.FillPath(ring, col)
```

### SVG path data

`ui.ParseSVGPath(d)` reads the `d` attribute of an SVG `<path>`: all the
commands, absolute and relative. Icons of icon sets are drawn so:

```go
icon, err := ui.ParseSVGPath("M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 ...z")
cnv.Save()
cnv.Translate(x, y)
cnv.ScaleBy(2, 2) // a 24 x 24 icon drawn 48 x 48
cnv.FillPath(icon, red)
cnv.Restore()
```

### Measures and hit tests

- `Bounds()` — the box of the path;
- `Length()` and `PointAtLength(d)` — the length, and the point and the
  direction at a distance along the path: to move something along it;
- `Contains(x, y)` — whether the point is inside, by the fill rule: what
  the mouse is over;
- `StrokeContains(x, y, width)` — whether the point is on the line;
- `Transform(m)`, `Clone()`, `CurrentPoint()`.

### Text as outlines

`p.Text(x, y, text, font, size)` adds the outlines of the letters to the
path, the baseline starting at `(x, y)`; `p.TextAligned(x, y, text, font,
size, hAlign, vAlign)` aligns them to the point. Unlike `DrawText`, which
draws pictures of the letters at the screen's size, such text is sharp at
any scale, turns and slants with the transform, can be filled with a paint
or stroked, and is written to SVG as curves: the text of a drawing, a map,
a gauge. `ui.MeasureTextOutline(text, font, size)` returns its width, the
ascent and the descent. Kerning and the fallback fonts are used; a bold the
family lacks is not synthesized.

```go
label := ui.NewPath()
label.TextAligned(0, 0, "Ø40 H7", ui.FontFamilySans, 3.5, ui.HAlignCenter, ui.VAlignBottom)
cnv.FillPath(label, ink)
```

## Transforms

The canvas has a 2D affine transform, kept by `Save` and `Restore` as the
clip is. It moves, turns, stretches and slants everything drawn after it is
set: the paths, the rectangles, the lines, the images and the text.

```go
cnv.Save()
cnv.Translate(100, 80)   // the origin
cnv.Rotate(math.Pi / 6)  // radians, clockwise on the screen
cnv.ScaleBy(2, 1)        // the widths of the lines too
cnv.FillRect(-20, -10, 40, 20, blue)
cnv.Restore()
```

| Method | |
|---|---|
| `Translate(dx, dy)`, `Rotate(angle)`, `RotateAround(angle, cx, cy)`, `ScaleBy(sx, sy)` | add to the transform |
| `Concat(m)` | apply a `ui.Matrix` before the current transform |
| `SetTransform(m)`, `Transform()`, `ResetTransform()` | set, read, clear |
| `ToCanvas(x, y)`, `FromCanvas(x, y)` | map a point to the canvas and back, e.g. the mouse to the drawing |

`ui.Matrix` is the matrix of SVG (`A, B, C, D, E, F`), made by
`IdentityMatrix`, `TranslationMatrix`, `ScalingMatrix`, `RotationMatrix`,
`SkewMatrix`, combined by `Then`, and inverted by `Invert`.

Without a transform (the default), the methods draw exactly as before; the
whole-pixel methods are pixel-sharp. Under a turn or a stretch they are
antialiased shapes. The clip stays a rectangle of the canvas. `Blur`,
`DrawShadow`, `FloodFill`, `Clear`, `PixelAt` and `AdjustPixels` work on the
pixels of the canvas and are not transformed.

### Large coordinates

A drawing zoomed far in has coordinates of millions of pixels around a
view of a thousand. The canvas keeps them exact and quick:

- the coordinates are `float64` until each polygon is cut to the pixels it
  covers;
- a curve or an arc too long to be flattened at once is split, and its
  parts out of view are taken by their chords: what is seen is within a
  twentieth of a pixel, however large the radius;
- a dashed line is cut to the view before it is dashed: a line of a
  billion pixels costs as much as one across the view.

A circle of a radius of 10⁹ pixels, seen in part, is drawn in about a
millisecond, its edge where it should be.

## Paints

`FillPathPaint` fills with a color per point:

- `ui.LinearGradient{X1, Y1, X2, Y2, Stops, Spread}`;
- `ui.RadialGradient{CX, CY, R, Stops, Spread}`;
- `ui.ConicGradient{CX, CY, Angle, Stops}` — around a center: color wheels,
  gauges;
- `ui.ImagePattern{Image, X, Y, Scale, Repeat, Smooth}` — a picture, tiled
  when `Repeat` is set;
- `ui.HatchPaint{Color, Background, Angle, Spacing, Width}` — stripes, for
  unavailable areas and bad data;
- any type with `ColorAt(x, y float64) color.RGBA`.

`Spread` is what a gradient is past its ends: `ui.SpreadPad` (the end
colors), `ui.SpreadRepeat`, `ui.SpreadReflect`. The colors of the gradients
are computed once per fill, not at every pixel.

## Images

- `DrawImage(x, y, img)` and `DrawImageRect(x, y, w, h, img, src, smooth)`
  draw an image, or a part of it stretched;
- `DrawImageTransformed(img, m, smooth)` draws it with a `ui.Matrix`: turned,
  slanted, at any scale, with antialiased edges.

## Shadows, layers, blur

```go
// A card with a soft shadow, lit from above
cnv.DrawShadow(x, y+2, w, h, radius, 10, color.RGBA{0, 0, 0, 80})
cnv.FillRoundedRect(...)

// Shapes seen through as one: their overlaps are not darker
cnv.BeginLayer(0.5)
cnv.FillCircle(40, 40, 20, red)
cnv.FillCircle(60, 40, 20, red)
cnv.EndLayer()

// Frosted glass behind a panel
cnv.Blur(x, y, w, h, 8)
```

`DrawShadow(x, y, w, h, radius, blur, color)` draws the blurred shape of a
rounded rectangle. `BeginLayer(opacity)` and `EndLayer()` group drawings and
put them on the canvas at the opacity; layers nest. `Blur(x, y, w, h,
radius)` blurs what is drawn in the rectangle (a Gaussian blur made of three
box blurs, so a large radius costs no more than a small one).

## Pixels

- `PixelAt(x, y)` — the color of a logical pixel;
- `AdjustPixels(x, y, w, h, f)` — replace every color of the rectangle by
  `f(color)`: a filter of your own (gray, sepia, inverted);
- `FloodFill(x, y, color, tolerance)` — fill the area of one color, as the
  bucket of a paint program;
- `Clear(color)` — set every pixel of the clip, its alpha too.

## Drawing off the screen

`ui.NewImageCanvas(w, h, scale)` returns a canvas on a new transparent
image. What is drawn there keeps its alpha (translucent pixels are blended
as `draw.Over` does), so the image can be saved, cached, or drawn over
something else later.

```go
cnv, img := ui.NewImageCanvas(256, 256, 1)
cnv.FillCircle(128, 128, 100, color.RGBA{37, 99, 235, 200})
png.Encode(f, img)
```

## Markers

`DrawMarker(shape, x, y, size, fill, stroke)` draws `ui.MarkerCircle`,
`MarkerSquare`, `MarkerDiamond`, `MarkerTriangleUp`, `MarkerTriangleDown`,
`MarkerCross`, `MarkerPlus`, `MarkerStar`: the points of a chart, the status
dots of a table, the entries of a legend.

## Turned text

`DrawTextRotated(x, y, angle, text)` draws a line of text turned around the
point that `SetHAlign` and `SetVAlign` name: `-math.Pi/2` reads upwards (the
title of a vertical axis). A quarter turn is exact, other angles smoothed.
`MeasureTextRotated` returns the box it takes. `DrawText` under a transform
of the canvas is turned and stretched with it.

## SVG

`SetRecorder(ui.NewSVGRecorder(w, h))` makes the canvas write what it draws
as SVG too: paths (with their fill rules and transforms), fills with
gradients and hatches, strokes with dashes, rectangles, lines and texts.
Paint a widget into such a canvas and write the recorder out: a picture
sharp at any size. Images and 3D scenes are not recorded.

## Performance

The canvas draws on the CPU. What a widget draws on every frame takes no
memory from the heap: the rasterizers and the masks are reused, a rounded
rectangle (the frame of every button) rasterizes only its corners and fills
the rest directly, translucent rectangles are blended row by row. Rough
costs on a 2014 desktop CPU (Core i5-4460):

| Operation | Time |
|---|---|
| `FillFrame` 120 × 30 (a button) | 3 µs |
| `FillRoundedRect` 120 × 30 | 5 µs |
| `FillCircle` radius 5 | 2.7 µs |
| `FillRect` 300 × 200, translucent | 0.2 ms |
| `FillPathPaint` 280 × 180 with a gradient | 1.5 ms |
| `StrokePath` of 1920 points, width 1.5 | 1.8–2.3 ms |

To keep a frame fast, reuse `Path`s (`Reset` keeps their memory) and draw
what does not change into an image once (`NewImageCanvas`). Many shapes of
the same style are quicker as one path: a call per layer of a drawing, not
per line. See [view2d.md](view2d.md) for views of large drawings.
