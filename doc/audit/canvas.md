# Audit: 2D canvas, paths, SVG, effects (ui/canvas*.go, internal/canvas)

Every finding below was reproduced with a test, unless it says otherwise. The reproducer files are in
`doc/audit/_repro/canvas/`
(`ui/zz_audit_canvas*_test.go` and `internal/canvas/zz_audit_canvas_test.go`). Copy them into the package to run them.
They use `newTestCanvas(w, h)` from `ui/canvas_test.go`, plus a helper `auditRun(t, name, f)` that runs `f` in a goroutine
and reports a panic (with stack) or a hang (>5 s). It is defined in `zz_audit_canvas_test.go`.

Line numbers refer to commit 4342672.

---

## 1. A dashed stroke disappears entirely when a dash ends exactly at a vertex, and so does a zero-length dash ("dotted" pattern)

- **Severity:** high (common inputs give a wrong result, and nothing is drawn)
- **Where:** `ui/canvas_path.go:1427` `dashPolyline`, `ui/canvas_path.go:1321` `strokePolyline` (`dir`, line 1333), `ui/canvas_path.go:969` `rasterizePolys`
- **Root cause:** When a dash ends exactly at a vertex (`segLen-pos == left`), `dashPolyline` appends the vertex `b` to `cur`. On the next segment, `pos=0` gives `pt == a == b`, and that point is appended again. The dash handed to `strokePolyline` then holds a repeated point: `[{0 10} {10 10} {10 10}]`. A zero-length dash (`Dash{0, 4}`) gives `[p, p]` in the same way. `strokePolyline` does not drop repeats; only `strokeSubpath` does. `dir()` divides by `l == 0`, so the quads and joins get NaN corners. `rasterizePolys` folds every polygon box into `all` with `math.Min/Max`, so a single NaN turns `all` into NaN. `boxPixels` then returns an empty rectangle and **the whole stroke is dropped**, including all the other dashes.
- **Reproducer:**
```go
cnv, img := newTestCanvas(40, 40)
p := NewPath(); p.Rect(10, 10, 20, 20)
cnv.StrokePath(p, StrokeStyle{Width: 2, Dash: []float64{4, 4}}, white)   // 20 = 2.5 periods: a dash ends at every corner
// inkBounds(img) == (0,0)-(0,0), inkSum == 0
cnv.DrawPolyline([]PointF{{0,10},{10,10},{10,30}}, StrokeStyle{Width: 2, Dash: []float64{10, 5}}, white) // nothing drawn
cnv.StrokePath(p, StrokeStyle{Width: 2, Dash: []float64{0, 4}, Cap: LineCapRound}, white) // dotted line: nothing drawn
```
  Output: `dash=[4 4]: ink=(0,0)-(0,0) sum=0.0`, `dash=[0 4] cap=round: sum=0.0`, `polyline dash {10,5}: sum=0.0`. With `{9,5}` it draws (sum=40). The stroke polygons contain `{NaN NaN}` (see `TestAuditDashCornerRootCause`).
- **Expected:** The dashes are drawn. Zero-length dashes with round or square caps draw dots, as in SVG.
- **Fix direction:** Drop repeated consecutive points inside `dashPolyline`, or in `strokePolyline`. For a dash that collapses to a single point, draw only its caps, as `strokeSubpath` does for one point. Optionally make `rasterizePolys`/`polyBuilder.add` skip polygons with non-finite points, so that one bad polygon cannot erase the rest.

## 2. Inf coordinates panic the rasterizer (integer divide by zero / index out of range)

- **Severity:** high (a panic in paint; Inf comes easily from data, e.g. `log(0)` in a chart, or from overflow under a transform)
- **Where:** `ui/canvas_path.go:1169` `clipPolygon` (`cross`, lines 1202-1214), then `x/image/vector` (nonzero) or `ui/canvas_path_fill.go:75-77` `evenOddRasterizer.LineTo` (even-odd)
- **Root cause:** `rasterizePolys` clamps the box (`boxPixels` limits it to ±1e9), so a polygon with an `±Inf` vertex passes. `clipPolygon.cross` then computes `t = (edge-p1.X)/(Inf-p1.X) = 0` and `p1 + Inf*0 = NaN`, which puts NaN vertices into the rasterizer. `vector.Rasterizer` panics with an integer divide by zero. In `evenOddRasterizer.LineTo`, `max(0, min(NaN, fw))` stays NaN and `int(NaN)` is `-9223372036854775808`, so `row[x0i]` panics. (A NaN vertex alone does not panic: it makes the box NaN and the shape is silently skipped.) Finite paths also overflow to Inf under a transform: `ParseSVGPath("M0 0L1e308 1e308")` drawn after `ScaleBy(2, 0.5)` panics in `strokePathXf`. `StrokeStyle{Width: +Inf}` panics too.
- **Reproducer:**
```go
cnv, _ := newTestCanvas(50, 50)
p := NewPath(); p.Polygon(PointF{0,0}, PointF{math.Inf(1),10}, PointF{0,40}, PointF{30,30})
cnv.FillPath(p, white)                 // panic: runtime error: integer divide by zero (rasterizeRect -> vector)
p.SetFillRule(FillRuleEvenOdd)
cnv.FillPath(p, white)                 // panic: index out of range [-9223372036854775808] (evenOddRasterizer.LineTo)
q := NewPath(); q.Rect(0, 0, math.Inf(1), 30); cnv.FillPath(q, white) // divide by zero
r := NewPath(); r.Rect(1,1,10,10); cnv.StrokePath(r, StrokeStyle{Width: math.Inf(1)}, white) // divide by zero
```
  The same panic happens for `{-Inf,10}` and `{10,+Inf}`.
- **Expected:** No panic. Draw what is finite (or nothing), consistently with NaN.
- **Fix direction:** In `clipPolygon`/`flatten`, clamp coordinates to a finite range (e.g. ±1e9 px, the range `boxPixels` already uses) before clipping, or drop non-finite points. Also guard `evenOddRasterizer.LineTo` against NaN (`if !(x0 >= 0)`…).

## 3. `Path.Bounds()` loops forever on an arc with an infinite or huge start angle (hang in ClipPath, SVG recorder, glyphs)

- **Severity:** high (an infinite loop on the UI thread)
- **Where:** `ui/canvas_path.go:316-340` `arcBounds`, loop `for ; t <= to; t += 2 * math.Pi` (line 336)
- **Root cause:** `EllipticalArc` clamps the sweep, but not `start`. With `start = ±Inf`, `from` and `to` are ±Inf, `t` becomes ±Inf, and `t += 2π` never changes it. With `|start| > ~2^54` (e.g. 1e17), `2π` is below half an ulp, so `t += 2π` is again a no-op. `Bounds` is called by `Canvas.ClipPath` (canvas_clip.go:47), by `SVGRecorder.paintAttr` for custom paints (canvas_svg.go:270) and by text_glyph.go:261.
- **Reproducer:**
```go
p := NewPath(); p.Arc(0, 0, 10, math.Inf(1), 0); p.Bounds()      // never returns
p2 := NewPath(); p2.Arc(0, 0, 10, 1e17, 1e17+16); p2.Bounds()    // never returns
```
  Output: `HANG (>5s)` for both.
- **Expected:** It returns: a NaN box or no arc for Inf, and a correct box for a huge finite angle.
- **Fix direction:** In `EllipticalArc`, reduce `start` modulo 2π (`math.Mod`) and reject a non-finite `start`/`end`. In `arcBounds`, bound the loop, which has at most 4 iterations per extremum since `|sweep| <= 8π`, and break if `t` does not grow.

## 4. `DrawShadow` panics on NaN/Inf/huge blur, NaN radius, and non-finite or huge position

- **Severity:** medium/high (a panic; the blur radius often comes from animation or config)
- **Where:** `ui/canvas_effects.go:140-185` (`pad`, line 147; the overflowing `mw*mh` check, line 155; `row := mask.Pix[(py-oy)*mw:]`, line 178), `ui/canvas_frame.go:340-342` `roundedRectPoints`, `ui/canvas_frame.go:202` `addPoints`
- **Root cause:**
  - `blur = Inf/NaN` (`max(NaN, 0)` is NaN in Go): `pad = int(NaN)+1` wraps around and the offsets become garbage. Result: `slice bounds out of range [-9223372036854775795:]`.
  - `blur = 1e12`: `mw*mh` overflows int and passes the `> 1<<24` guard, so `ras.reset` gets a negative size: `slice bounds out of range [:-412788226847481687]`.
  - `x` or `y` = Inf/NaN/-1e300: `int(math.Floor(x0))` overflows, and huge float32 coordinates reach `vector.Rasterizer`: integer divide by zero in `addPoints`.
  - `radius = NaN`: in `roundedRectPoints`, `max(0, min(NaN, …))` is NaN, so `steps = min(4+int(NaN)/2, 32)` is negative and `quarterTrig(steps)` panics with `index out of range [-4611686018427387900]`.
- **Reproducer:**
```go
cnv, _ := newTestCanvas(40, 40)
cnv.DrawShadow(0, 0, 10, 10, 2, math.Inf(1), white)   // slice bounds out of range
cnv.DrawShadow(0, 0, 10, 10, 2, math.NaN(), white)    // same
cnv.DrawShadow(0, 0, 10, 10, 2, 1e12, white)          // rasterizer.reset: slice bounds out of range [:-4127...]
cnv.DrawShadow(math.Inf(1), 0, 10, 10, 2, 3, white)   // integer divide by zero (addPoints)
cnv.DrawShadow(-1e300, 0, 10, 10, 2, 3, white)        // integer divide by zero
cnv.DrawShadow(0, 0, 10, 10, math.NaN(), 3, white)    // quarterTrig: index out of range
```
- **Expected:** No panic. Non-finite input draws nothing; a huge blur is limited.
- **Fix direction:** Return early when any argument is non-finite. Clamp `sigma` (e.g. to the size of the image), compute `mw*mh` in float64 (or check `mw > limit/mh`), and treat a NaN radius as 0 in `roundedRectPoints`.

## 5. `Canvas.Blur` hangs on a large radius: the cost is O(radius) per row, and the uint32 sum overflows

- **Severity:** medium
- **Where:** `ui/canvas_effects.go:224-266` `boxBlurH`/`boxBlurV` (initial window loop `for x := -r; x <= r; x++`, line 232; `var sum uint32`, lines 231 and 251)
- **Root cause:** The doc says "the cost does not grow with the radius". However, each row and each channel first sums `2r+1` clamped samples, so the cost is O(r). With `radius = 1e7`, `r` is about 5e6, and a 40x40 region takes far more than 5 s (with 1e9 it is practically forever). `sum` is `uint32`. It holds up to `255*(2r+1)`, which overflows when `r > ~8.4e6`, and the result is garbage.
- **Reproducer:**
```go
cnv, _ := newTestCanvas(40, 40)
cnv.FillRect(10, 10, 10, 10, white)
cnv.Blur(0, 0, 40, 40, 1e7)    // HANG (>5s)
cnv.Blur(0, 0, 40, 40, 1e12)   // HANG (>5s)
```
- **Expected:** It returns quickly. A radius larger than the region gives the mean.
- **Fix direction:** Clamp `r` to `max(w, h)`, since with clamped edges a larger window only adds copies of the edge pixels. Compute the initial sum in closed form: `(r - x + 1)*edge + sum(...)`, or with `min(r, w-1)` and the remaining edge count multiplied. Use a `uint64` sum.

## 6. Even-odd fill gives wrong coverage in the first column of the rasterized rectangle (clip edges, band edges)

- **Severity:** medium (visible wrong pixels; error up to 2x)
- **Where:** `ui/canvas_path_fill.go:44-106` `evenOddRasterizer.LineTo` (the clamp `x0 = max(0, min(x0, fw))` at line 75)
- **Root cause:** Within one row, a line that crosses x=0 (the left edge of the rectangle being rasterized) has its x clamped to `[0, w]` before the area split. The part of the line left of 0 should add its full `d` (its share of dy) to cell 0. Instead, the clamp treats the whole `dy` as if it lay between 0 and `x1`, which gives the wrong area. Polygons are clipped to `[-1, w+1]` (`rasterizeRect`), so such lines always occur when a shape crosses the left edge of a widget's clip, and at every 64-px band boundary in `rasterizePolys`. The nonzero rasterizer (x/image/vector) gives the right value.
- **Reproducer:**
```go
for _, eo := range []bool{false, true} {
    c, img := newTestCanvas(40, 40)
    c.SetDirectTranslateAndClip(10, 0, 30, 40)
    p := NewPath()
    p.Polygon(PointF{-10, 0}, PointF{20.5, 0}, PointF{-19.5, 40}, PointF{-10, 40})
    if eo { p.SetFillRule(FillRuleEvenOdd) }
    c.FillPath(p, white)
    // img.RGBAAt(10, 20).R: nonzero 32 (exact: 0.125*255=32), evenodd 64
}
```
  A random test (`TestAuditEvenOddVsNonZero`, 3000 simple polygons) also found pixel (0,18): exact coverage 161 (by supersampling), nonzero 161, evenodd 93.
- **Expected:** For a simple polygon, even-odd matches nonzero.
- **Fix direction:** Before splitting, split the row's line at x=0 (and at x=w). Add the part with x<0 as `d_part` to `row[0]`, and area-split only the part inside `[0, w]`. Alternatively, keep a "left carry" accumulator.

## 7. `FillTriangle`, `DrawLine` (width > 1) and `fillPolygonPixels` panic at moderately large integer coordinates (>= 2^25)

- **Severity:** medium (a panic; 33M px is reachable with zoomed charts or with unclipped data positions)
- **Where:** `ui/canvas_frame.go:236-261` `fillPolygonPixels` (also `addPoints`, line 202)
- **Root cause:** The rasterizer box is clipped, but the polygon is not. The points are passed as `float32(p - ox)` to `vector.Rasterizer`, whose fixed-point path overflows. Result: integer divide by zero. `FillPath`/`FillPolygon` do not have this problem because they clip with `clipPolygon`.
- **Reproducer:**
```go
cnv, _ := newTestCanvas(40, 40)
cnv.FillTriangle(-1<<25, 0, 1<<25, 10, 0, 30, white)   // panic: integer divide by zero
cnv.DrawLine(-1<<26, 0, 1<<26, 17, 3, white)           // panic: integer divide by zero (drawWideLine -> fillPolygonPixels)
```
- **Expected:** It draws the visible part.
- **Fix direction:** Clip the polygon to the box with a margin (reuse `clipPolygon`) before feeding the rasterizer in `fillPolygonPixels`.

## 8. `DrawText` under a large zoom panics in `image.NewRGBA` (int overflow bypasses the size guard)

- **Severity:** medium (a CAD-like zoom of 2^32 makes the paint panic)
- **Where:** `ui/canvas_transform.go:500-513` `drawTextXf` (`if pw*ph > 1<<20`, line 511)
- **Root cause:** `pw*ph` overflows int. With `k = 2^32` and a 1x1 box, `pw = ph = 2^32` and `pw*ph = 2^64`, which wraps to 0. That passes the guard, and `image.NewRGBA(2^32 x 2^32)` panics.
- **Reproducer:**
```go
cnv, _ := newTestCanvas(40, 40)
cnv.ScaleBy(1<<32, 1<<32)
cnv.DrawText(0, 0, 1, 1, "A")   // panic: image: NewRGBA Rectangle has huge or negative dimensions
```
- **Expected:** It falls back to `drawTextOutlines`.
- **Fix direction:** Compare in float64: `float64(width)*k*float64(height)*k > 1<<20`, or check each side first.

## 9. `ClipPath` with very large coordinates clips everything away (float to int overflow)

- **Severity:** medium (wrong result: nothing is drawn inside a clip that covers the widget)
- **Where:** `ui/canvas_clip.go:47-53`
- **Root cause:** `int(math.Floor(maxX*s+ox))+1` with `maxX = 1e300` or `+Inf` gives `MinInt64+1`, because a float to int conversion out of range is implementation-defined (on amd64 it gives 0x8000000000000000). For the left side, `int(...)-1` wraps to `MaxInt64`. `image.Rect` then swaps the edges, and the intersection with the image is empty. `FillPath` with the same path draws correctly (finite 1e300), so only the clip is wrong.
- **Reproducer:**
```go
cnv, img := newTestCanvas(40, 40)
cnv.Save()
p := NewPath(); p.Rect(0, 0, 1e300, 30)   // also Rect(-1e300, 0, 1e300+30, 30), or +Inf
cnv.ClipPath(p)
cnv.FillRect(0, 0, 40, 40, white)
cnv.Restore()
// img.RGBAAt(5,5) == {0 0 0 255}, want white
```
- **Expected:** Pixels inside the shape are drawn.
- **Fix direction:** Clamp the float bounds to ±1e9 (as `boxPixels` does) before converting to int. Reusing `boxPixels` is simplest.

## 10. `CohenSutherland` (DrawLine width 1) overflows int: a visible line is reported as invisible

- **Severity:** low/medium
- **Where:** `ui/canvas.go:938-1018` (e.g. `y = y1 + (y2-y1)*(right-x1)/(x2-x1)`, line 998)
- **Root cause:** The intersection products `(y2-y1)*(right-x1)` overflow int64 for coordinates around ±3e9, which are reachable after `pixelCenter` from large logical ints.
- **Reproducer:**
```go
x1, y1, x2, y2, ok := CohenSutherland(-3000000000, -3000000000, 3000000000, 3000000000, 0, 0, 39, 39)
// -> 0 1518386171 3000000000 3000000000 false   (the diagonal y=x passes through the box)
cnv.DrawLine(-3000000000, -3000000000, 3000000000, 3000000000, 1, white) // draws nothing
```
- **Expected:** `ok == true`, endpoints (0,0)-(39,39), and the diagonal is drawn.
- **Fix direction:** Do the intersection math in float64 (`clipLineF` already exists), or clip with float Liang-Barsky and round afterwards.

## 11. SVG recorder: font family is not escaped, so attributes can be injected into the SVG

- **Severity:** medium (security: SVG/XML injection; an exported SVG may be opened in a browser)
- **Where:** `ui/canvas_svg.go:340-358` `svgFont` (only `'` is removed, line 349)
- **Root cause:** The family is written inside `font-family="…"` without escaping `"`, `<` or `&`.
- **Reproducer:**
```go
rec := NewSVGRecorder(100, 100)
cnv, _ := newTestCanvas(100, 100)
cnv.SetRecorder(rec)
cnv.SetFontFamily(`Evil" onload="alert(1)`)
cnv.DrawText(0, 0, 100, 20, "hi")
// rec.String() contains: <text x="0" y="0" font-family="'Evil" onload="alert(1)', sans-serif" ...>
```
  `encoding/xml` parses it as a `<text>` element with an `onload` attribute.
- **Expected:** `font-family="'Evil&quot; onload=&quot;alert(1)', sans-serif"`.
- **Fix direction:** Pass the family through `svgEscape`. Better, use an attribute escaper (`xml.EscapeText` plus `"`) for every attribute value built from strings.

## 12. SVG recorder: control characters in text produce an invalid SVG document

- **Severity:** low
- **Where:** `ui/canvas_svg.go:361-378` `svgEscape`
- **Root cause:** XML 1.0 forbids U+0000-U+0008, U+000B, U+000C, U+000E-U+001F (and U+FFFE/FFFF). `svgEscape` copies them through.
- **Reproducer:** `cnv.DrawText(0,0,100,20,"a\x01b\x00c\x1b")` with a recorder. `xml.Decoder` then fails with `illegal character code U+0001`.
- **Expected:** A well-formed SVG. Drop such runes or replace them with U+FFFD.
- **Fix direction:** In `svgEscape`, skip runes that are not valid XML characters.

## 13. `PaintCache.InvalidateRect` with an infinite or huge rectangle repaints nothing

- **Severity:** low
- **Where:** `ui/canvas_cache.go:121-129` (`int(math.Floor(d.MinX))`, line 124, `int(math.Ceil(d.MaxX))-x0`)
- **Root cause:** `RectF{-Inf,-Inf,+Inf,+Inf}` (a natural "invalidate everything") or ±1e300 converts to `MinInt64` for both ends, so the width is 0. The clip is empty, the dirty part is skipped, and the stale picture is shown.
- **Reproducer:**
```go
pc := NewPaintCache(); calls := 0
paint := func(*Canvas) { calls++ }
pc.Draw(cnv, 0, 0, 40, 40, paint)
pc.InvalidateRect(RectF{math.Inf(-1), math.Inf(-1), math.Inf(1), math.Inf(1)})
pc.Draw(cnv, 0, 0, 40, 40, paint)   // calls == 1, want 2
```
- **Fix direction:** Intersect `d` with `RectF{0,0,w,h}` (in float) before converting to int. Or, in `InvalidateRect`, treat a rectangle that covers the whole picture as `Invalidate()`.

## 14. x/image/vector fixed-point rasterizer gives wrong coverage at sharp vertices (nonzero fills)

- **Severity:** low (third-party precision issue; visible as a too-bright pixel near acute tips)
- **Where:** `ui/canvas_path.go:1049` `rasterizeRect` uses `vector.Rasterizer`. For sizes ≤ 512 it uses fixed-point math, which is wrong here; the float math (size > 512) is right.
- **Reproducer:**
```go
z := vector.NewRasterizer(400, 300)   // fixed-point (<=512)
z.MoveTo(0, 0); z.LineTo(400, 37.3); z.LineTo(10, 290); z.ClosePath()
// mask pixel (9,288) = 108; with NewRasterizer(600,600) (float) = 13; exact (64x64 supersampling) = 13
```
  nui shows 108 at (9,288) for `FillPath` of this triangle on a 400x300 or 100x300 canvas, while the even-odd rasterizer gives the correct 13.
- **Fix direction:** Either accept it, or force floating-point math. The `vector` package chooses by size; a rasterizer at least 513 wide could be used with a sub-rectangle, or nui could always use its own accumulation rasterizer (like `evenOddRasterizer`, with a nonzero accumulate, once bug 6 is fixed).

## 15. internal/canvas: Save/Restore is a no-op, translucent colors blend too dark, Clear panics on non-RGBA colors, sub-images are drawn wrongly

- **Severity:** low (an internal package used only for the default window icon with opaque colors; these are latent)
- **Where:**
  - `internal/canvas/canvas_state.go:57-59` `Save` pushes the **same pointer** `c.CurrentState()`, so changes after `Save` change the saved state too.
  - `internal/canvas/canvas_setpixel.go:15-43` `BlendPixel` treats the premultiplied `col.RGBA()` as straight color and multiplies by alpha again: `RGBA{128,128,128,128}` over black gives 64 instead of 128.
  - `internal/canvas/canvas.go:33-41` `Clear` does `col.(color.RGBA)`, which panics for `color.White` (Gray16). It also assumes `Stride == 4*w` and `Rect.Min == 0`.
  - `BlendPixel` checks `x < Dx()` from 0 instead of `Rect.Min`, so on a `SubImage` nothing (or the wrong area) is drawn.
- **Reproducer:** `internal/canvas/zz_audit_canvas_test.go`:
  - `TestAuditSaveRestore`: the color after Save/SetColor(green)/Restore is green.
  - `TestAuditBlendPixelTranslucent`: 64, want ~128.
  - `TestAuditClearNonRGBA`: panic `interface conversion: color.Color is color.Gray16, not color.RGBA`.
  - `TestAuditSubImage`: pixel (9,9) is not filled.
- **Fix direction:** `Save` should push a copy (`s := *c.CurrentState(); append(&s)`). Blend premultiplied: `out = d*(1-a) + s`. Use `color.RGBAModel.Convert` in `Clear`. Use `image.Pt(x,y).In(rgba.Rect)`.

---

### Checked, no bug found
- `ParseSVGPath` was fuzzed for 150 s (about 1.07M execs), parsing and then fill (nonzero and even-odd), dashed stroke, Bounds, Length, PointAtLength, Contains, StrokeContains, transformed fill/stroke/ClipPath, and the SVG recorder. Inputs with |coord| > 1e15 were excluded because they are already covered by finding 2. No panics or hangs.
- Parallel off-screen rendering (`NewImageCanvas` in 8 goroutines: paths, gradients, layers, ClipPath, MipImage incl. Invalidate, Blur, DrawShadow, text, transforms) under `-race`: no races.
- Layers with an empty clip, clip-in-layer, NaN opacity, double EndLayer, Restore without Save, singular/NaN/tiny/huge image transforms, empty/offset MipImage, NaN markers, negative rects in AdjustPixels/Blur, FloodFill tolerance range, DrawLineF with NaN/Inf/huge values, NaN stroke width, NaN/Inf/tiny dash, NaN transform: no panics.
