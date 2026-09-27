# Canvas API (`internal/canvas`)

2D drawing helper on top of `*image.RGBA`. Used inside `win.OnPaint(func(rgba *image.RGBA) {...})`.

## Create

```go
cnv := canvas.NewCanvas(rgba) // rgba is the *image.RGBA passed to OnPaint
cnv.Width() int
cnv.Height() int
cnv.RGBA() *image.RGBA // underlying buffer
```

## Color and style state

```go
cnv.SetColor(color.RGBA{R, G, B, A})
cnv.SetLineWidth(w float64)
cnv.Transform(dx, dy float64) // adds to current translation offset
cnv.Clip(x, y, w, h float64)  // sets clip rect (default: whole canvas)

cnv.Save()    // push current state (color/transform/clip)
cnv.Restore() // pop it
```

State (color, line width, transform, clip) is stacked by `Save`/`Restore`. Always pair them.

## Clearing

```go
cnv.Clear(color.RGBA{0, 0, 0, 255})
```

## Pixels

```go
cnv.SetPixel(x, y float64, alpha float64)      // uses current color
cnv.BlendPixel(x, y int, col color.Color)      // alpha-blends col onto existing pixel, any color.Color
```

Out-of-bounds coordinates are ignored (no panic).

## Shapes

```go
cnv.DrawLine(x0, y0, x1, y1 int, alpha float64)
cnv.DrawRect(x, y, w, h float64)               // outline
cnv.FillRect(x, y, w, h int, alpha float64)    // filled
cnv.DrawCircle(x0, y0, r int)                  // outline
cnv.FillCircle(x, y, radius int)               // filled
```

All shapes use the color set by `SetColor`. `alpha` (0..1) is applied on top of the current color's alpha for `FillRect`/`DrawLine`.

## Text

```go
cnv.DrawFixedString(x, y float64, str string, pixelSize float64)
cnv.DrawFixedChar(x, y float64, ch byte, pixelSize float64)
```

Built-in fixed-width bitmap font (5x7 per glyph). `pixelSize` scales each font pixel (e.g. `2` = each font pixel becomes a 2x2 block). No external font files needed.

## Example

```go
win.OnPaint(func(rgba *image.RGBA) {
	cnv := canvas.NewCanvas(rgba)
	cnv.Clear(color.RGBA{20, 20, 20, 255})
	cnv.SetColor(color.RGBA{0, 200, 255, 255})
	cnv.FillRect(10, 10, 100, 40, 1)
	cnv.DrawCircle(200, 100, 30)
	cnv.DrawFixedString(10, 60, "Score: 42", 2)
})
```
