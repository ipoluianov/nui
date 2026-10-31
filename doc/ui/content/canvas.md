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
antialiased shapes. The clip of `TranslateAndClip` stays a rectangle of
the canvas; `ClipPath` clips to any shape. `Blur`,
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

## Clipping to a shape

`ClipPath(p)` clips everything drawn after it to the inside of the path,
under the transform, with antialiased edges, until the `Restore` of the
`Save` before it:

```go
cnv.Save()
card := ui.NewPath()
card.RoundedRect(0, 0, w, h, 12)
cnv.ClipPath(card)
cnv.DrawImageRect(0, 0, w, h, photo, photo.Bounds(), true) // the corners cut off
cnv.Restore()
```

- Clips nest: a `ClipPath` after another clips to the inside of both.
- Everything is clipped: paths, rectangles, text, images, 3D scenes,
  `Blur`, `Clear`, the widgets painted inside.
- The fill rule of the path counts: an even-odd ring clips to a ring.
- It is done at `Restore`: the drawings go on the canvas, and `Restore`
  puts back the pixels they changed outside the shape. Call it between
  `Save` and `Restore`; until then `PixelAt` reads the pixels as drawn.
  `Blur` inside the clip blurs what is under it, as frosted glass should.
- It costs a rasterization of the shape and two copies of the pixels of
  its box: about 0.1 ms for a card of 300 × 200.
- The SVG recorder writes it as a `<clipPath>`.

## Paints

`FillPathPaint` fills with a color per point, `StrokePathPaint(p, style,
paint)` draws a line so — along a gradient, or colored by a value of the
application's own (a track colored by speed):

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
  slanted, at any scale, with antialiased edges;
- `SetImageOpacity(a)` draws the images that follow at the opacity 0..1
  (kept by `Save` and `Restore`): a picture fading in, a disabled icon.

### Images drawn small

An image drawn much smaller than it is - a photo as a thumbnail, a map or
a page zoomed out - samples a few of its pixels for each pixel of the
canvas and looks grainy. `ui.NewMipImage(img)` keeps copies of it each
half the size of the one before (mipmaps); drawn smaller than half its
size with `smooth`, it is sampled from the copy of the nearest size, smooth
and as quick as a small image:

```go
photo := ui.NewMipImage(img)
cnv.DrawImageRect(0, 0, 160, 120, photo, photo.Bounds(), true)
```

The copies are made when first needed and take a third more memory than
the image. After the pixels of the image change, call `Invalidate()`. A
`MipImage` is an `image.Image`: drawn at its size, it is the image itself.

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

### Blend modes and masks

`BeginLayerStyle(ui.LayerStyle{Opacity, Blend, Mask})` starts a layer that
`EndLayer` puts on the canvas by its style:

```go
// A layer of a drawing over the paper, and an eraser in it
cnv.BeginLayerStyle(ui.LayerStyle{Isolated: true})
paintStrokes(cnv)
cnv.BeginLayerStyle(ui.LayerStyle{Blend: ui.BlendErase})
cnv.StrokePath(eraser, ui.StrokeStyle{Width: 20, Cap: ui.LineCapRound, Join: ui.LineJoinRound}, black)
cnv.EndLayer()
cnv.EndLayer()

// The end of a list faded out
cnv.BeginLayerStyle(ui.LayerStyle{Mask: ui.LinearGradient{Y1: h - 40, Y2: h,
	Stops: []ui.GradientStop{{0, opaque}, {1, transparent}}}})
paintRows(cnv)
cnv.EndLayer()
```

| `Blend` | |
|---|---|
| `BlendNormal` | over (the zero value) |
| `BlendMultiply`, `BlendScreen`, `BlendOverlay`, `BlendDarken`, `BlendLighten`, `BlendColorDodge`, `BlendColorBurn`, `BlendHardLight`, `BlendSoftLight`, `BlendDifference`, `BlendExclusion`, `BlendHue`, `BlendSaturation`, `BlendColor`, `BlendLuminosity` | the blend modes of CSS and SVG |
| `BlendAdd` | the colors added: light, glow |
| `BlendErase` | clears the canvas as much as the layer is opaque: an eraser |
| `BlendKeepInside` | keeps the canvas only where the layer is drawn; the rest of the layer's area is cleared |
| `BlendAtop` | the layer only over what is drawn: a texture over a shape |

- `Opacity` 0..1, 0 is 1. `Mask` is a `Paint` whose alpha is the opacity
  at each point, in the coordinates of the drawing at `BeginLayerStyle`:
  a gradient fades the layer out, an `ImagePattern` cuts it by a picture.
- A layer covers the clip at its start. A layer of `BlendNormal` is drawn
  on the canvas, as `BeginLayer`; one of another blend, or with
  `Isolated: true`, on a transparent picture of its own, so `Blur` and
  `PixelAt` inside read the layer alone, and an eraser in it rubs out the
  layer, not what is under it: the layer of a drawing program.
- `ClipPath` begun in a layer ends with it at the latest.
- A blend costs about 8 ns a pixel of the layer's area on the old desktop
  CPU (0.5 ms for 300 × 200); keep the clip of such a layer to what it
  covers.
- The SVG recorder writes a layer as a group: its opacity, its blend as
  `mix-blend-mode` (`BlendAdd` as `plus-lighter`), its isolation, its mask
  as a `<mask>` of the paint. `BlendErase`, `BlendKeepInside` and
  `BlendAtop` have no counterpart in SVG: what is drawn in such a layer is
  not written, rather than written as ink.

## Pixels

- `PixelAt(x, y)` — the color of a logical pixel;
- `AdjustPixels(x, y, w, h, f)` — replace every color of the rectangle by
  `f(color)`: a filter of your own (gray, sepia, inverted);
- `FloodFill(x, y, color, tolerance)` — fill the area of one color, as the
  bucket of a paint program;
- `Clear(color)` — set every pixel of the clip, its alpha too.

A painter of its own writes the pixels of `RGBA()` itself, for an image
copied as is or a scene rendered by other code. Two things place it:

- `PixelOrigin()` is the pixel where the logical (0, 0) lands: a logical
  point is at `PixelOrigin()` plus the point times `Scale()`;
- `ClipRect()` is the pixels it may write. In a frame that repaints only a
  part of the window that is the part, the rest of the image holds the
  pixels of other widgets. It is a bound, not a place: the drawing starts
  at `PixelOrigin()` wherever the part starts.

```go
w.SetOnPaint(func(cnv *ui.Canvas) {
	dst, at := cnv.RGBA(), cnv.PixelOrigin()
	r := scene.Rect.Add(at).Intersect(cnv.ClipRect())
	draw.Draw(dst, r, scene, r.Min.Sub(at), draw.Src)
})
```

Under a transform of the canvas the logical point is not there; draw such
a picture with `DrawImage`.

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

## Painting less

What a widget paints costs on every frame. Two things keep it to what
changed:

- `cnv.ClipBounds()` is the part of the drawing coordinates the clip shows
  (under the transform): draw only what reaches into it. A view of a part
  of a large picture, or a strip of it being painted again, costs that
  part.
- `ui.PaintCache` keeps the picture of an expensive drawing and puts it on
  the canvas again until it is invalidated; what changes every frame - a
  cursor, a selection, a hover - is drawn over it. `InvalidateRect(r)`
  paints only that part again, the rest is kept:

```go
cache := ui.NewPaintCache()
w.SetOnPaint(func(cnv *ui.Canvas) {
	cache.Draw(cnv, 0, 0, w.Width(), w.Height(), func(cnv *ui.Canvas) {
		visible := cnv.ClipBounds() // all of it, or the invalid part
		paintMap(cnv, visible)
	})
	paintCursor(cnv)
})
// A marker moved: its old and new boxes painted again
cache.InvalidateRect(oldBox)
cache.InvalidateRect(newBox)
w.Update()
```

The picture is made again when the size or the scale of the screen
changes, and `Invalidate()` paints all of it again. Under a transform of
the canvas, or recorded as SVG, the drawing is painted directly. Putting a
kept picture of 800 × 600 on the canvas takes about 0.6 ms. (A `View2D`
keeps its picture so itself, and moves it when the view is dragged.)

A frame of a mouse move, a wheel or a timer paints only the widgets that
called `Update`, over the last frame, and only their rectangles (a layout,
a click, a key paint the whole window): a painter is clipped to the part of the window
repainted, and `ClipBounds()` is that part. With the expensive parts kept,
a frame is the copy of their pictures and the drawing of the rest.

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

## Text laid out

`DrawText` puts text at whole pixels, in one style, each character drawn
with its own glyph. A `TextLayout` is text as a document or a label of a
chart needs it:

```go
l := ui.NewTextLayout("Total: ", ui.TextStyle{Size: 15})
l.Append("1 250 ₽", ui.TextStyle{Size: 15, Bold: true, Color: green})
l.Append(" — paid", ui.TextStyle{Size: 13, Italic: true})
l.SetWidth(220)               // wrapped at the width; 0 does not wrap
l.SetAlign(ui.HAlignCenter)
w, h := l.Size()
cnv.DrawTextLayout(l, 10.5, 40.25) // fractional, under the transform
```

- **Styles.** `TextStyle{Font, Size, Color, Bold, Italic, Underline,
  Strikethrough, Background, LetterSpacing}` for each piece added by
  `Append`; a zero `Color` takes the color of `SetColor`.
- **Lines.** Paragraphs at `"\n"`, wrapped at `SetWidth` where Unicode
  allows (between words; anywhere in Chinese), `SetAlign`,
  `SetLineSpacing(k)`, `SetMaxLines(n)` with an ellipsis. `Lines()`
  returns each line: its text, its box, its baseline.
- **Scripts.** The text is shaped by HarfBuzz (go-text/typesetting):
  ligatures and kerning, Arabic letters joined, Indic letters combined,
  marks placed. Right-to-left text is laid out by the Unicode bidi
  algorithm, numbers and Latin words within it in their order; a paragraph
  takes the direction of its first letter (`SetRightToLeft` to set it).
  The fonts of the scripts come from the system (see
  [fonts.md](fonts.md)).
- **Where the characters are.** `CaretRect(i)` is the caret before the
  byte `i` of `Text()`, `IndexAt(x, y)` the caret nearest a point (a
  click), `SelectionRects(from, to)` the boxes of a selection, split where
  right-to-left text is mixed in: what a text editor needs.
- **Drawing.** Text that is not turned is drawn from masks of the glyphs
  kept for each quarter of a pixel, as quick as `DrawText`; turned,
  slanted or very large text, and text recorded as SVG, is filled as the
  outlines of its glyphs: sharp at any zoom.

`DrawTextF(x, y, text)` draws a line in the font, size, color and
underline of the canvas, as `DrawText` does, but at a fractional point
and shaped, around the point that `SetHAlign` and `SetVAlign` name;
`MeasureTextF` returns its size. Its layouts are kept, so text drawn again
is not laid out again.

A layout is laid out when it is first measured or drawn after a change:
about 50 µs for a short paragraph. Keep it and draw it many times. It is
not safe for concurrent use; the drawing of different layouts is.

## SVG

`SetRecorder(ui.NewSVGRecorder(w, h))` makes the canvas write what it draws
as SVG too: paths (with their fill rules and transforms), fills with
gradients and hatches, strokes with dashes, rectangles, lines and texts.
Paint a widget into such a canvas and write the recorder out: a picture
sharp at any size. Images and 3D scenes are not recorded; a `PaintCache`
paints directly into a recorded canvas, so its drawing is recorded.

## Colors in linear light

sRGB, the space of the colors of nui and of the screens, is not linear in
light: the middle between black and white by its numbers (128) is darker
than the middle in light (188). Browsers and nui mix in sRGB by default.
Two settings mix in linear light, as light mixes:

- `LinearRGB: true` on a `LinearGradient`, `RadialGradient` or
  `ConicGradient`: the colors between the stops (SVG's
  `color-interpolation="linearRGB"`); brighter between saturated colors, no
  dark band between red and green. Written to SVG as stops close enough
  for any viewer.
- `cnv.SetLinearBlending(true)` (kept by `Save` and `Restore`): the
  antialiased edges and the translucent fills of paths, rectangles and
  transformed images; thin lines keep their weight at any angle. Text keeps
  the mixing of the system, in sRGB, so it has the weight of the system's
  text; layers, blur, `DrawImage` and `DrawImageRect` mix in sRGB.

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
| `FillPath` of a rounded rectangle 300 × 200 | 0.16 ms |
| `ClipPath` + `Restore`, 300 × 200 | 0.1 ms |
| a layer of `BlendMultiply`, 300 × 200 | 0.5 ms |
| `DrawText` / `DrawTextF`, 25 characters | 11 µs / 10 µs |
| `StrokePath` of 1920 points, width 1.5 | 1.8–2.3 ms |

To keep a frame fast, reuse `Path`s (`Reset` keeps their memory) and draw
what does not change into an image once (`NewImageCanvas`). Many shapes of
the same style are quicker as one path: a call per layer of a drawing, not
per line. See [view2d.md](view2d.md) for views of large drawings.
