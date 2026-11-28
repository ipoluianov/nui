# Charts audit (ui/chart*.go, ui/timechart*.go)

Every finding below was reproduced with a test in package `ui`. The tests are in
`doc/audit/_repro/charts/`
(`zz_audit_charts*_test.go`). Copy them into `ui/` to run them. Helpers used below (`auditTry`, `auditPaint`,
`auditInteract`, `auditStackAfter`) are defined in `zz_audit_charts_test.go`, `zz_audit_charts3_test.go` and
`zz_audit_charts5_test.go`:

- `auditTry` runs f in a goroutine and reports a panic, or a HANG after 10 s.
- `auditPaint` paints the ChartView twice into an RGBA image.
- `auditStackAfter(d)` dumps all goroutines and exits after d.

**Run the hang/OOM tests one at a time and under a memory cap**, e.g. `go test -c -o /tmp/x/ui.test ./ui/`
then `ulimit -v 1500000; /tmp/x/ui.test -test.run 'TestAuditRadarInfOOM$'`. Some of them allocate without
bound until the machine runs out of memory.

`TestAuditChartsSmoke` (`zz_audit_charts3_test.go`) is a matrix: 45 chart builders x 16 datasets (empty, single,
equal, zero, negative, NaN, Inf, huge, MaxFloat64, denormal, 70000 points) x 3 sizes, each with full mouse,
wheel and keyboard interaction, SaveState/RestoreState, CSV, SVG and Render. Use `AUDIT_SKIP=prefix1,prefix2` to
skip the known hangs. Most findings below came from it.

Several panics end inside the canvas rasterizer. The canvas audit already covers that root cause
(canvas.md #2: Inf coordinates give integer divide by zero). They are listed here because the chart passes
non-finite or overflowing pixel coordinates from ordinary data values. The chart should filter them out
whatever the canvas fix is.

---

## 1. Radar/polar: the ring loop makes millions of rings for small values, and never ends (unbounded memory) for Inf/huge values

- **Severity:** high (process OOM or freeze from data values)
- **File:** ui/chart_polar.go:33-39 (`polarFrameOf`)
- **Root cause:**
  ```go
  step := niceStepAtLeast(top / 4)
  rimMax := math.Ceil(top/step-1e-9) * step
  for r := step; r <= rimMax+1e-9; r += step { rings = append(rings, r) }
  ```
  - The `+1e-9` epsilon is **absolute**. When the largest value `top` is far below 1e-9, the loop keeps going
    until `r` reaches 1e-9 in steps of `top/4`. At `top=1e-12` that is 2001 rings; at 1e-15 it is 2,000,002
    rings, each drawn with a label. At 1e-18 it is about 2e9 rings, which is an OOM. Small physical values
    such as currents in pA, seconds or probabilities are realistic radar inputs.
  - With `top=+Inf` (a single Inf value), `niceStepAtLeast(Inf)` returns 1 and `rimMax` is Inf. `r <= Inf`
    is always true, so the loop appends forever and memory grows until the OOM.
  - With `top=MaxFloat64`, `rimMax = 4*5e307` overflows to +Inf, with the same result.
  - `polarMax` does not filter non-finite values.
- **Reproducer** (zz_audit_charts6_test.go, zz_audit_charts5_test.go):
  ```go
  func TestAuditRadarSmallValues(t *testing.T) {
      for _, top := range []float64{1, 1e-12, 1e-14, 1e-15} {
          ch := NewXYChart()
          ch.Pane(0).AddRadar("r").Set("a", top).Set("b", top/2).Set("c", top/3)
          ch.SetSize(400, 300)
          auditPaint(ch, 400, 300)
          p := ch.Pane(0)
          f := p.polarFrameOf(p.viewport(p.xAxis, p.YAxis()), NewCanvas(image.NewRGBA(image.Rect(0, 0, 10, 10))))
          t.Logf("top %g: %d rings", top, len(f.rings))
      }
  }
  func TestAuditRadarInfOOM(t *testing.T) {
      auditStackAfter(4 * time.Second)
      ch := NewXYChart()
      ch.Pane(0).AddRadar("r").Set("a", 1).Set("b", math.Inf(1)).Set("c", 2)
      ch.Render(400, 300, 1, nil)
  }
  ```
  Output:
  ```
  top 1: 2 rings (rim 1)
  top 1e-12: 2001 rings (rim 1e-12)
  top 1e-14: 200001 rings
  top 1e-15: 2000002 rings          (whole test 22.8 s, nearly all spent painting ring labels)
  ```
  RadarInf under `ulimit -v 1500000`: `fatal error: out of memory ... ui/chart_polar.go:38`.
  The smoke case `radar/tiny` (values 1e-320, 2e-320) took the whole test binary down with
  `runtime: out of memory: cannot allocate 1149239296-byte block`. `TestAuditPolarMaxFloat`
  (`AddPolar("p").Add(0,1).Add(90, math.MaxFloat64)`) hangs.
- **Expected:** about 4 rings for any finite top. Non-finite values are ignored.
- **Fix direction:** make the epsilon relative (`r <= rimMax*(1+1e-9)`), or count the rings as an integer
  `n := int(math.Round(rimMax/step))` and use `r = step*float64(k)`. Cap n. Skip non-finite values in
  `polarMax`, and fall back when `rimMax` is not finite.

---

## 2. A dense line freezes the UI when a sample far outside the view is its neighbour (O(pixel distance) loop)

- **Severity:** high (UI hang: seconds per frame, or indefinitely)
- **File:** ui/chart_series_line.go:404-413 (`drawColumns`)
- **Root cause:** a thin straight line with about 1 sample per pixel is drawn by `drawColumns`. For each pair of
  samples more than 1.5 px apart, it fills `steps := int(math.Ceil(x - prevX))` one-pixel columns, **without
  clipping to the plot**. The source (by its contract) returns the nearest sample outside the visible range on
  each side. When that neighbour is far away, the loop runs once per *virtual* pixel up to it. Two common cases
  trigger it: a data gap (a device offline for a week or a year) or a strongly zoomed-in view. One example is
  1000 visible samples on a 400 px plot plus one sample at x=1e11, which is 4e10 px away.
- **Reproducer** (zz_audit_charts5_test.go / zz_audit_charts2_test.go):
  ```go
  func TestAuditHangFar(t *testing.T) {
      auditStackAfter(3 * time.Second)
      ch := NewLineChart()
      s := ch.Series("s")
      for i := 0; i <= 1000; i++ { s.AddXY(float64(i), float64(i%7)) }
      s.AddXY(1e11, 0)
      ch.SetXRange(0, 1000)
      ch.Render(400, 300, 1, nil)
  }
  ```
  Output: render time is 0.2 s with the far point at 1e7, 2.1 s at 1e8 and more than 10 s at 1e11 (HANG). The stack:
  ```
  ui.(*Canvas).fillRectF  canvas_path.go:1543
  ui.(*LineSeries).drawColumns  chart_series_line.go:413
  ui.(*LineSeries).Draw  chart_series_line.go:338
  ```
- **Expected:** constant cost per visible sample.
- **Fix direction:** clamp the interpolated run to the plot `[v.X-w, v.X+v.W+w]`. Iterate k only over the
  columns whose cx lies inside it, computing cy by interpolation. Alternatively, draw off-screen segments as one
  clipped line.

---

## 3. Histogram2D / hexbin: one NaN or Inf point panics (index out of range [-9223372036854775808])

- **Severity:** high (panic in `SetPoints` from a data value)
- **File:** ui/chart_series_heatmap.go:419-440 (`Histogram2DSeries.bin`)
- **Root cause:** the extent loop takes `math.Min/Max` over all points, so a NaN makes `x0` NaN. An Inf point
  makes `(p.X-x0)/(x1-x0)` NaN. `c := min(int(NaN), s.cols-1)` evaluates `int(NaN)`, which is MinInt64 on
  amd64, and `grid[r][c]` then panics. Points are not filtered for finiteness. `SetHex` uses the same `bin`
  through `SetPoints`.
- **Reproducer** (zz_audit_charts6_test.go):
  ```go
  ch := NewXYChart()
  ch.Pane(0).AddHistogram2D("h").SetPoints([]ChartPoint{{1, 1}, {math.NaN(), 2}, {3, 3}})
  // same with {math.Inf(1), 2}
  ```
  Output: `PANIC runtime error: index out of range [-9223372036854775808]` at `(*Histogram2DSeries).bin`
  chart_series_heatmap.go:440. The smoke cases `hexbin/nan`, `hexbin/inf`, `hexbin/huge` and `hexbin/maxf`
  panic the same way. The huge range `-1e308..1.7e308` overflows `x1-x0` to Inf.
- **Expected:** non-finite points are skipped. A range that overflows does not panic.
- **Fix direction:** skip points with `!chartFinite(p.X) || !chartFinite(p.Y)` in both loops. Compute the index
  as a float, clamp it to `[0, cols-1]`, and only then convert it to int. Guard `x1-x0 == Inf` by scaling,
  e.g. `(p.X/2 - x0/2)/(x1/2 - x0/2)`.

---

## 4. Histogram: a value far from the others panics (int overflow of the bin index), and values past bin 10000 pile into the last bin

- **Severity:** medium (panic from data and an explicit bin width; wrong counts)
- **File:** ui/chart_series_stats.go:127-137 (`HistogramSeries.compute`)
- **Root cause:**
  - `n` is capped to 10000 bins, but every value is still put at `i := min(int((v-start)/width), n-1)`.
  - When `(v-start)/width > 2^63` (for example width 1e-9 with values 0 and 1e10), or when `hi-lo` overflows to
    Inf (with values ±1e308), `int()` of the float gives MinInt64. `min(...)` keeps it and `s.counts[i]` panics.
  - When it does not overflow, all values beyond the 10000th bin are silently counted into the last bin. The
    histogram is then wrong, and the edges stop at `start+10000*width`, far below the data's max.
- **Reproducer** (zz_audit_charts6_test.go):
  ```go
  ch := NewXYChart()
  ch.Pane(0).AddHistogram("h").SetBins(0, 1e-9).SetValues([]float64{0, 0.5, 1e10})
  ch.Render(400, 300, 1, nil)                        // panic
  ch2 := NewXYChart()
  ch2.Pane(0).AddHistogram("h").SetValues([]float64{-1e308, 1.7e308})
  ch2.Render(400, 300, 1, nil)                       // panic
  ch3 := NewXYChart()
  h := ch3.Pane(0).AddHistogram("h").SetBins(0, 0.001).SetValues([]float64{0, 1, 2, 3, 100})
  ch3.Render(400, 300, 1, nil); e, c := h.Bins()
  ```
  Output:
  ```
  PANIC runtime error: index out of range [-9223372036854775808]
    ui.(*HistogramSeries).compute ... (*HistogramSeries).Range ... (*ChartView).fitXAxes
  bins 10000, last edge 10, last count 1          <- the value 100 is counted in [9.999,10]
  ```
- **Expected:** no panic. When the bin count would exceed the cap, the width is widened (e.g.
  `width = (hi-start)/10000` rounded to a nice step) so every value lands in its own bin.
- **Fix direction:** if `(hi-start)/width > 10000`, recompute the width instead of truncating n. Clamp the index
  in float space before converting it to int. Handle an Inf `hi-lo` by computing in halves.

---

## 5. Inf or overflowing values in a series panic the paint: integer divide by zero in the rasterizer (line fill, band, stacked polar, funnel, digital, percent stack)

- **Severity:** high (an Inf from a sensor or a division crashes the app on paint)
- **Files:**
  - ui/chart_series_line.go:549-603 (`drawFill`, the gradient fill at :600 and the plain fill at :603)
  - ui/chart_series_stats.go:675-690 (`BandSeries.Draw`)
  - ui/chart_series_waterfall.go:303 (`FunnelSeries.Draw`)
  - ui/chart_series_spans.go:456 (`DigitalSeries.Draw`)
  - ui/chart_polar.go (stacked polar with -Inf)
  - ui/chart_pane.go:325-326 (percent stack factor)
- **Root cause:**
  - The series turn every `HasGood` sample into pixels with `v.PY(value)` and add them to a path, without
    `chartFinite` checks. The plain stroke path happens to survive, but filled paths do not. An Inf (or 1e308,
    which overflows after scaling) reaches `rasterizePolys` -> `clipPolygon`, gives NaN vertices, and
    `x/image/vector` panics with an integer divide by zero (canvas.md #2).
  - In the percent stack, `if total != 0 { factor = 100/total }` gives `+Inf` for a denormal total, with the
    same result.
  - The y autoscale ignores such samples (see #6), so the rest of the chart is at 0..1 too.
- **Reproducer** (zz_audit_charts4_test.go, zz_audit_charts6_test.go):
  ```go
  ch := NewLineChart()
  ch.Series("a").SetFill(true, 0.3, false).SetValues([]float64{1, math.Inf(1), 3})
  ch.Render(400, 300, 1, nil)        // panic (also with gradient=true, and with values {1e308, -1e308, 1.7e308})

  b := NewXYChart()
  b.Pane(0).AddBand("b").Add(0, 1, 0, 2).Add(1, 2, 1, math.Inf(1)).Add(2, 1, 0, 2)
  b.Render(400, 300, 1, nil)         // panic

  s := NewLineChart()
  s.Series("a").SetStack("g").SetFill(true, 0.3, false).SetValues([]float64{1e-320, 2e-320})
  s.Series("b").SetStack("g").SetValues([]float64{1e-320, 2e-320})
  s.Pane(0).SetStackPercent("g", true)
  s.Render(400, 300, 1, nil)         // panic
  ```
  Output (each case):
  ```
  PANIC runtime error: integer divide by zero
    golang.org/x/image/vector/raster_fixed.go:121
    ui.(*Canvas).rasterizeRect canvas_path.go:1085
    ui.(*Canvas).rasterizePolys canvas_path.go:1031
    ui.(*LineSeries).drawFill chart_series_line.go:600   (BandSeries.Draw chart_series_stats.go:690, ...)
  ```
  The smoke cases `line-spline-fill/{inf,-inf,huge,maxf}`, `line-step-stack/tiny`, `band/{inf,-inf}`,
  `digital/{inf,-inf}` (Inf x), `polar/-inf` and `funnel/huge` panic the same way.
- **Expected:** a non-finite value is treated as a missing sample: it breaks the line or the fill, and is
  skipped in a band.
- **Fix direction:**
  - In each series' Draw, Range and Nearest, treat `!chartFinite(value)` as no value. Clamp pixel coordinates
    to e.g. ±1e7 around the plot before building paths.
  - In computeStacks, only use a factor when `100/total` is finite.
  - The canvas fix (canvas.md #2) is still needed as a backstop.

---

## 6. A single NaN or Inf sample breaks the y autoscale: the axis shows 0..1 and the data is off-screen

- **Severity:** medium (wrong display from common data: NaN is a usual "no reading")
- **Files:** ui/chart_series_line.go:282-298 (`LineSeries.Range`), ui/chart_series_candle.go:83-90,
  ui/chart_series_stats.go:631-636 (`BandSeries.Range`), ui/chart_series_spans.go:158-162 (`SpanSeries.Range`)
- **Root cause:**
  - `lo, hi = math.Min(lo, v), math.Max(hi, v)` with v = NaN makes lo and hi NaN. `lo <= hi` is then false,
    `fit` gets `ok=false` and falls back to 0..1.
  - With +Inf, hi is Inf. `fit` computes an Inf margin, and `widenDomain` returns 0..1.
  - Other series (bars, scatter, heatmap, sparkline) filter with `chartFinite`. These do not.
  - `SpanSeries.Range` even returns `ok=true` with a NaN bound.
- **Reproducer** (zz_audit_charts2_test.go):
  ```go
  ch := NewLineChart()
  ch.AddValues(100, 101, math.NaN(), 102, 103)
  newTestForm(t, ch)
  ch.Render(400, 300, 1, nil); ch.laidOut = true
  lo, hi, _ := ch.Pane(0).YAxis().Range()
  ```
  Output: `y range with a NaN: -0.05..1.05` and `y range with an Inf: 0..1`. The data at 100..103 is not visible.
- **Expected:** the axis is fitted to the finite values (100..103 plus the margin).
- **Fix direction:** skip `!chartFinite(l) || !chartFinite(h)` in the Range loops of LineSeries, CandleSeries,
  BandSeries and SpanSeries (and Funnel/Waterfall, if they do the same).

---

## 7. Zooming in far enough makes the view jump out to a huge range (10 years on a time axis)

- **Severity:** medium (wrong behaviour in normal interaction)
- **File:** ui/chart_scale.go:133-136 (`widenDomain`, used by every scale's SetDomain). It is reached from
  `zoomAxis`/`setAxisRange` in chart_view_input.go:693-734.
- **Root cause:** `widenDomain` treats any range shorter than `1e-12*|min|` as "no length" and widens it by
  `0.1*|min|`.
  - On a TimeScale at 1.7e9 s, a span below 1.7 ms counts as empty and is widened to ±1.7e8 s (about 5.4
    years). The wheel zoom in a plain ChartView has no minimum span (`MinSpanX` defaults to 0; only TimeChart
    sets 10 ms), so zooming in on millisecond data suddenly shows a decade. The next zoom-in starts from there.
  - The same happens on a linear axis with values around 1e6 at a 1e-6 span.
- **Reproducer** (zz_audit_charts6_test.go):
  ```go
  ch := NewLineChart()
  ch.XAxis().SetScale(&TimeScale{Location: time.UTC})
  s := ch.Series("a")
  for i := 0; i < 1000; i++ { s.AddXY(1.7e9+float64(i)*1e-3, math.Sin(float64(i)/10)) }
  ch.SetSize(500, 300); auditPaint(&ch.ChartView, 500, 300)
  prev := math.Inf(1)
  for i := 0; i < 60; i++ {
      ch.onWheel(&WheelEvent{DY: 3, X: 250, Y: 150}); auditPaint(&ch.ChartView, 500, 300)
      lo, hi := ch.XRange()
      if hi-lo > prev*1.01 { t.Fatalf("step %d: span grew from %g to %g (%f..%f)", i, prev, hi-lo, lo, hi) }
      prev = hi - lo
  }
  ```
  Output: `step 9: zoom in made the span grow from 0.0024154186248779297 s to 3.4000000009647393e+08 s
  (range 1530000000.428567..1870000000.525041)`
- **Expected:** the zoom stops at the smallest representable span, or stays where it is.
- **Fix direction:** in setAxisRange/zoomAxis, refuse (or clamp) a zoom that would go below a minimum span
  (e.g. `max(MinSpanX, |mid|*4e-12)`), keeping the previous range. Only `fit` (autoscale) should widen a
  zero range. Alternatively, give ChartView a default MinSpan derived from float precision.

---

## 8. TimeChart: dates before 1678 or after 2262 are bucketed at wrong times (UnixNano overflow), and nothing is drawn

- **Severity:** medium (historical or far-future data is silently wrong)
- **File:** ui/timechart_source.go:100-131 (`TimeChartAggregator.Add`, used by `TimeChartDownsample` and the
  memory source)
- **Root cause:** `ns := p.DT.UnixNano()` and `time.Unix(0, b*g)`. Go documents UnixNano as undefined outside
  1678..2262, and it wraps around. The ChartView layer (ChartTime, float seconds) supports such dates. Only the
  aggregation breaks.
- **Reproducer** (zz_audit_charts4_test.go):
  ```go
  base := time.Date(2300, 1, 1, 0, 0, 0, 0, time.UTC)
  var pts []TimeChartPoint
  for i := 0; i < 100; i++ { pts = append(pts, NewTimeChartValue(base.Add(time.Duration(i)*time.Minute), float64(i))) }
  res := TimeChartDownsample(pts, base, base.Add(100*time.Minute), 10*time.Minute)
  // res[0].DT
  ```
  Output:
  ```
  year 1600: first bucket at 2184-07-20 23:30:00 (want 1600-01-01)
  year 2300: first bucket at 1715-06-13 00:20:00
  year 3000: first bucket at 1830-11-23 00:50:00
  ```
  For a TimeChart with a memory source over year 2300 (TestAuditTimeChartFarDates), 457 samples are drawn and
  0 of them are within the visible range, so the plot is empty.
- **Expected:** buckets at the right times for any date that time.Time holds.
- **Fix direction:** bucket on `DT.Sub(epochRef)` with a reference near the data (e.g. `from`), or on
  `(Unix(), Nanosecond())` pairs: `b = floor((sec*1e9+nsec)/g)` computed with 128-bit math or in seconds when
  `g >= 1s`. Rebuild the time with `ref.Add(time.Duration(b*g))`.

---

## 9. A span with a NaN end and a label panics in the text drawing (int(NaN) position)

- **Severity:** medium (panic from data)
- **File:** ui/chart_series_spans.go:219-223 -> ui/chart_view.go:932-950 (`chartDrawLabel`) ->
  text_glyph.go:316
- **Root cause:**
  - `SpanSeries.Draw` checks visibility with `x2 < v.X-20 || x1 > v.X+v.W+20`. That check is false for NaN,
    so the span is drawn, and the label is written at `x2+4 = NaN`.
  - `chartDrawLabel` passes `int(math.Round(NaN))` = MinInt64 to `DrawText`. `blendGlyphMask` then computes
    `box := image.Rect(x+m.x, ...)`, which overflows, and the pixel slice index goes negative.
  - Any chart label with a non-finite position takes this path.
- **Reproducer** (zz_audit_charts6_test.go):
  ```go
  ch := NewXYChart()
  s := ch.Pane(0).AddSpans("s").SetLabels(true)
  s.Add(ChartSpan{ID: "a", Row: "r", Start: 0, End: 5, Label: "ok"})
  s.Add(ChartSpan{ID: "b", Row: "r", Start: 1, End: math.NaN(), Label: "x"})
  ch.Render(400, 300, 1, nil)
  ```
  Output:
  ```
  PANIC runtime error: slice bounds out of range [:-9223372036854775798]
    text_glyph.go:316 (blendGlyphMask) <- text_glyph.go:436 <- chart_view.go:950 (chartDrawLabel) <- chart_series_spans.go:223
  ```
- **Expected:** a span without finite bounds is skipped.
- **Fix direction:**
  - In SpanSeries.Draw/Range/Nearest, skip spans with non-finite Start/End.
  - In `chartDrawLabel`, return early when x or y is not finite or is beyond ±1e7.
  - As a backstop in the canvas, DrawText should reject coordinates whose glyph box overflows int.

---

## 10. Pie: a large start angle makes hovering hang (the angle normalization loop adds 2π per turn)

- **Severity:** low-medium (UI freeze; reachable with an accumulated rotation angle)
- **File:** ui/chart_series_pie.go:286-300 (`sliceAt`); `SetAngles` at :115 accepts any start
- **Root cause:** `for x < sl.a0 { x += 2*math.Pi }`. With a start of 1e12 this takes about 1.6e11 iterations,
  which is a hang of minutes. At a start of 1e20 or above, `x += 2π` no longer changes x, so the loop is
  infinite. Inf or NaN starts are also accepted.
- **Reproducer** (zz_audit_charts6_test.go):
  ```go
  ch := NewPieChart()
  ch.Add("a", 1).Add("b", 2)
  ch.Pie().SetAngles(1e20, 2*math.Pi)
  ch.SetSize(400, 300); auditPaint(&ch.ChartView, 400, 300)
  ch.onMouseMove(230, 180, KeyModifiers{}); ch.updateHits()
  ```
  Output: `pie hover with start 1e20: HANG (>10s)`. It also hangs with 1e12.
- **Expected:** hit testing works for any start angle.
- **Fix direction:** normalize with `math.Mod`: `x = sl.a0 + math.Mod(math.Mod(a-sl.a0, 2π)+2π, 2π)`.
  Normalize `start` in SetAngles with `math.Mod(start, 2π)` and reject non-finite values.

---

## 11. Symlog (and linear) domain at the float limit becomes infinite: Symlog Ticks loops forever (with -MaxFloat it also appends forever)

- **Severity:** low (needs values or a manual range near ±1.6e308, but the result is a hang or OOM)
- **File:** ui/chart_scale.go:126-137 (`widenDomain`), ui/chart_scale.go:547-555 (`SymlogScale.Ticks`)
- **Root cause:**
  - `widenDomain(min, max)` with `min == max` beyond about 1.6e308 computes `max + 0.1*|min|`, which is +Inf.
    The resulting domain is, for example, `1.6e308..+Inf`.
  - In SymlogScale.Ticks, `top = Inf` gives `kMax = Inf`, and `for k := kMax; k >= 0; k--` never ends
    because `Inf-1 == Inf`.
  - With `NewSymlogScale(-MaxFloat64, -MaxFloat64)`, the min is -Inf, so `add(-c*10^Inf)` keeps appending
    -Inf ticks without bound.
  - A LinearScale gets the same Inf domain and makes every value map to unit 0. That is wrong, but it does
    not hang.
- **Reproducer** (zz_audit_charts5_test.go / zz_audit_charts4_test.go):
  ```go
  ch := NewLineChart()
  ch.Series("a").SetValues([]float64{1, 2, 3})
  ch.Pane(0).YAxis().SetScale(NewSymlogScale(0, 1)).SetRange(math.MaxFloat64, math.MaxFloat64)
  ch.Render(400, 300, 1, nil)
  // and: NewSymlogScale(-math.MaxFloat64, -math.MaxFloat64).Ticks(300, measure)
  ```
  Output: `symlog chart: HANG (>10s)`, with the stack at chart_scale.go:550.
  `NewLinearScale(MaxFloat64, MaxFloat64).Domain() = 1.6179238213760842e+308..+Inf`.
- **Expected:** a finite domain is always kept.
- **Fix direction:** in widenDomain, check that the widened bounds are finite; if they are not, widen toward
  zero (`min - d` only) or return the previous or a default domain. In SymlogScale.Ticks, cap kMax (e.g.
  `min(kMax, 330)`) and return nil for a non-finite domain.

---

## 12. SaveState returns empty data when the selection or a manual range is non-finite; RestoreState of it fails

- **Severity:** low (silent loss of the whole saved view)
- **File:** ui/chart_sync.go:175-193 (`SaveState`)
- **Root cause:** `data, _ := json.Marshal(st)`. encoding/json fails on ±Inf or NaN (`unsupported value`),
  the error is dropped, and nil is returned. All zoom ranges and hidden series are lost along with the bad
  selection. Neither `SetSelection` nor `SetRange` (setManual) rejects non-finite values.
- **Reproducer** (zz_audit_charts4_test.go):
  ```go
  ch := NewLineChart()
  ch.Series("a").SetValues([]float64{1, 2, 3})
  ch.SetSelection(0, math.Inf(1))
  ch.Pane(0).YAxis().SetRange(0, 10)
  st := ch.SaveState()
  err := ch.RestoreState(st)
  ```
  Output: `SaveState = ""`, then `RestoreState(SaveState()) = unexpected end of JSON input`.
- **Expected:** the state is saved without the non-finite parts, or SetSelection/SetRange reject non-finite
  values.
- **Fix direction:** reject non-finite values in SetSelection and setManual. In SaveState, skip non-finite
  ranges or selections instead of failing the whole marshal.

---

## 13. Candles: an inverted candle (high < low) is dropped from the autoscale, and open/close outside high..low are cut off

- **Severity:** low (malformed but real feed data is drawn off-screen)
- **File:** ui/chart_series_candle.go:83-90 (`CandleSeries.Range`)
- **Root cause:** the range uses only `p.Low` for lo and `p.High` for hi.
  - With a single candle whose high is below its low, lo > hi, so `ok=false` and the axis is 0..1.
  - When open or close lie outside [low, high] (fields swapped by the feed), they are not included and are
    clipped. `Draw` does draw them.
- **Reproducer** (zz_audit_charts4_test.go):
  ```go
  ch := NewXYChart()
  ch.Pane(0).AddCandles("c").AddOHLC(0, 10, 9, 11, 10)  // high 9 < low 11
  ch.Render(400, 300, 1, nil)
  lo, hi := ch.Pane(0).YAxis().scale.Domain()
  ch2 := NewXYChart()
  ch2.Pane(0).AddCandles("c").AddOHLC(0, 100, 50, 40, 45).AddOHLC(1, 45, 50, 40, 48)
  ```
  Output: `y domain of a candle with high 9, low 11: -0.05..1.05` and `y domain with open 100 above high 50:
  39.5..50.5`.
- **Expected:** the axis covers all four values of every candle.
- **Fix direction:** use the min and max of First, Last, High and Low (finite only) per sample.

---

## 14. Render/WritePNG panics for a NaN or +Inf scale

- **Severity:** low
- **File:** ui/chart_export.go:28-31 (`Render`)
- **Root cause:** `if scale <= 0 { scale = 1 }` lets NaN and +Inf through. `int(math.Ceil(w*NaN))` is
  MinInt64, so `image.NewRGBA` panics. A scale computed from DPI or zero-size printer data can be NaN.
- **Reproducer** (zz_audit_charts4_test.go): `ch.Render(100, 100, math.NaN(), nil)` and
  `ch.Render(100, 100, math.Inf(1), nil)`.
  Output: `PANIC image: NewRGBA Rectangle has huge or negative dimensions`
- **Fix direction:** `if !(scale > 0) || math.IsInf(scale, 0) { scale = 1 }`. Also cap the image size.

---

## 15. ChartPane.SetWeight(NaN) gives all panes NaN heights: nothing is drawn

- **Severity:** low
- **File:** ui/chart_pane.go:64-65
- **Root cause:** `math.Max(NaN, 0.01)` is NaN. The weight sum becomes NaN, so every pane's h is NaN.
- **Reproducer** (zz_audit_charts4_test.go):
  ```go
  ch := NewLineChart(); ch.Series("a").SetValues([]float64{1, 2, 3})
  ch.AddPane().SetWeight(math.NaN()).AddLine("b").SetValues([]float64{1, 2})
  ch.Render(400, 300, 1, nil)   // ch.panes[0].h, ch.panes[1].h == NaN NaN
  ```
- **Fix direction:** `if !(weight > 0.01) || math.IsInf(weight, 0) { weight = 0.01 or 1 }`.

---

## 16. DrawSparkline / KPI trend: a constant series of large values (>= ~1e16) draws nothing

- **Severity:** low
- **File:** ui/chart_sparkline.go:139-141
- **Root cause:** for a zero range the code uses `lo, hi = lo-1, hi+1`. For |v| >= 2^53, `v±1 == v`, so
  `hi-lo` stays 0 and `(v-lo)/(hi-lo)` is NaN. The path is then NaN and the canvas drops it.
- **Reproducer** (zz_audit_charts6_test.go):
  ```go
  // count painted pixels of DrawSparkline(cnv, 0, 0, 100, 30, vals, ChartSparkLine, red, nil)
  ```
  Output: `painted pixels: constant 5 -> 1356, constant 5e16 -> 0`
- **Fix direction:** widen relatively, as ChartAxis.fit does: `d := math.Max(math.Abs(lo)*0.1, 1)`.

---

## 17. Quadratic builders: BarSeries.Set, PieSeries.Add, BandSeries.Add, out-of-order AddState

- **Severity:** low (performance with large counts)
- **Files:**
  - ui/chart_series_bar.go:91-103: `Set` copies the whole category list for every new category.
  - ui/chart_series_pie.go:71-80: `Add` scans the slices linearly by name.
  - ui/chart_series_stats.go:580-588: `Band.Add` inserts with copy.
  - ui/chart_data.go:144-160: `Append` out of order inserts with copy.
- **Measured** (zz_audit_charts4_test.go `TestAuditBarSetQuadratic`): 5000 categories by Set take 0.12 s;
  20000 categories take 2.6 s; 70000 categories exceed 8 s in the smoke test. The smoke test also timed out
  on `pie/many` (70000 Add) and `band/many` (error bars: 70000 markers drawn every paint without culling to
  the view).
- **Fix direction:** use an index map for category and slice names. Append the category in place. Use bulk
  setters. Cull error-bar markers to the visible x range.

---

Checked without finding a bug:

- empty, single-point and all-equal data on every chart type;
- log scale with zero or negative data;
- gauge, level and KPI with zero, inverted or NaN ranges;
- pie with zero total or negative values;
- heatmap with an empty or ragged matrix;
- waterfall;
- annotations with inverted zones;
- multi-pane charts with extra axes;
- CSV and SVG export of all of the above;
- the ChartSyncGroup with charts in two groups and handlers that reset zoom or selection;
- removing panes and series from OnHover, OnBeforeAction, OnViewChanged and OnSeriesToggled during
  interaction (TestAuditRemoveDuringInteraction);
- all scales (TestAuditScales: linear, log, symlog, time and category over extreme domains and lengths, with
  the symlog float-limit case excluded; see #11).
