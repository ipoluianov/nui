# TimeChart

Interactive time-series chart: one or more stacked **areas** that share a time
axis, each with its own Y axis and any number of **series**. Every series reads
its points from a **data source** you provide, so the same chart works for live
monitoring, historical data from a server and trading (OHLC) data.

`TimeChart` embeds `Widget`, so all generic widget methods are available too.
See `examples/ex06timechart` for a runnable demo (live ping to several servers,
downsampling, candles).

## Quick start

```go
src := ui.NewTimeChartMemorySource()
now := time.Now()
for i := 0; i < 3600; i++ {
  t := now.Add(time.Duration(i-3600) * time.Second)
  src.AddPoint(ui.NewTimeChartValue(t, math.Sin(float64(i)/100)))
}

chart := ui.NewTimeChart()
chart.AddArea().AddSeries("Signal", src)
chart.SetDefaultTimeRange(now.Add(-10*time.Minute), now)

form.Panel().AddWidget(0, 0, chart)
```

## Structure

```
TimeChart            time axis, zoom, selection
 └─ TimeChartArea    one horizontal band with its own Y axis, legend, markers
     └─ TimeChartSeries   a line or candles drawn from a TimeChartDataSource
```

Areas are stacked top to bottom with equal heights. Above each area there is a
legend strip with the names of its series.

## Points

```go
type TimeChartPoint struct {
  DT      time.Time
  First   float64
  Last    float64
  High    float64
  Low     float64
  HasGood bool
  HasBad  bool
}
```

A point is either one raw sample (all four values equal) or a bucket of
samples merged together: `First`/`Last` are the first and last values in the
bucket and `High`/`Low` its extremes. This is also an OHLC bar, so the same
type serves trading charts.

Keeping `High`/`Low` matters when data is downsampled: a signal that jitters
inside a bucket is drawn as a vertical band instead of collapsing into a flat
average line.

Data quality:

- `HasBad` - the point (or some sample in the bucket) is bad, for example a
  sensor was disconnected. Bad spans are drawn as a light reddish hatch.
- `HasGood` - at least one sample carries a valid value. `First`/`Last`/
  `High`/`Low` describe only the good samples.
- A point without `HasGood` has no value: the line breaks there and the
  point is ignored by Y auto-scaling. Set `HasGood` when building points
  as struct literals.
- A point with neither flag is a gap: no data at all, for example where data
  acquisition was stopped or started. The line breaks there without the
  hatch.

Helpers:

- `NewTimeChartValue(dt time.Time, value float64) TimeChartPoint` - a good raw sample.
- `NewTimeChartBad(dt time.Time) TimeChartPoint` - a bad sample without a value.
- `NewTimeChartGap(dt time.Time) TimeChartPoint` - a break in the data.
- `(p TimeChartPoint) HasValue() bool` - whether the values are meaningful (`HasGood`).
- `(p TimeChartPoint) IsGap() bool` - whether the point is a gap (neither flag).

## Data sources

```go
type TimeChartDataSource interface {
  GetData(from, to time.Time, groupDuration time.Duration) []TimeChartPoint
}
```

`GetData` is called on every paint for the visible range. `groupDuration` is
the time covered by one screen pixel: return the data aggregated into buckets
of that size (about one point per pixel). Rules:

- Points must be sorted by `DT`.
- Include the nearest point on each side outside `[from, to]`, so the line
  continues to the edges of the plot.
- The call happens on the UI thread during painting, so it must be fast. A
  source that loads data asynchronously (e.g. from a server) returns what it
  has now and calls `Form().Update()` on the chart when more data arrives.

### TimeChartMemorySource

Ready-made source over an in-memory slice. Safe to add points from another
goroutine.

- `NewTimeChartMemorySource() *TimeChartMemorySource`
- `SetPoints(points []TimeChartPoint)` - replaces the data; points must be sorted by `DT`.
- `AddPoint(p TimeChartPoint)` - appends a point; its `DT` must not be earlier than the last one.
- `GetData(from, to, groupDuration)` - implements `TimeChartDataSource` via `TimeChartDownsample`.

### TimeChartDownsample

```go
func TimeChartDownsample(points []TimeChartPoint, from, to time.Time, groupDuration time.Duration) []TimeChartPoint
```

Takes a sorted slice, keeps the points in `[from, to]` plus one neighbour on
each side and merges them into buckets of `groupDuration`:

| Bucket contains      | HasGood | HasBad | Values                  |
|----------------------|---------|--------|-------------------------|
| only good samples    | yes     |        | from all samples        |
| good and bad samples | yes     | yes    | from the good ones only |
| only bad samples     |         | yes    | none                    |

Gap points are not merged into buckets: each one stays a separate point and
starts a new bucket after it, so a break in the data remains visible at any
zoom level.

### TimeChartAggregator

The same aggregation, fed one point at a time - for a source with many points
(e.g. a day of samples) that should not build a slice of all of them on every paint:

```go
agg := ui.NewTimeChartAggregator(groupDuration, 0)
for _, s := range samplesIn(from, to) { // sorted, plus one neighbour on each side
	agg.Add(toPoint(s))
}
return agg.Points()
```

Buckets are aligned to multiples of `groupDuration` since the Unix epoch, so
panning does not change how points are grouped. It is the same aggregation a
server should do before sending data to the chart. With `groupDuration <= 0`
the points are returned as they are.

### Custom source

```go
type serverSource struct{ client *MyClient }

func (s *serverSource) GetData(from, to time.Time, group time.Duration) []ui.TimeChartPoint {
  // The server aggregates into buckets of `group` and returns
  // DT/First/Last/High/Low/HasGood/HasBad for each bucket.
  return s.client.Query(from, to, group)
}
```

## TimeChart

### Create

- `NewTimeChart() *TimeChart` - expands in both directions. The initial and
  default range is the last hour.

### Areas

- `AddArea() *TimeChartArea` - adds an area at the bottom.
- `Areas() []*TimeChartArea`
- `RemoveAllAreas()`

### Time range and the default view

The application owns the **default range**; the user moves away from it by
panning and zooming and comes back to it with Esc or by zooming back.

- `SetDefaultTimeRange(from, to time.Time)` - sets the default range. Call it
  as often as needed (e.g. on every new sample of a live feed): while the chart
  is in the default view the new range is shown at once; after the user has
  panned or zoomed it is only remembered.
- `DefaultTimeRange() (from, to time.Time)`
- `IsDefaultView() bool` - whether the chart shows the default range and follows `SetDefaultTimeRange`.
- `ResetTimeRange()` - switches to the default view (keeps manual Y ranges).
- `TimeRange() (from, to time.Time)` - the visible range.
- `SetTimeRange(from, to time.Time)` - shows the given range and leaves the
  default view. The span is limited to 10 ms ... 200 years.
- `SetOnTimeRangeChanged(f func())` - called when the visible range or the
  default-view state changes (including while panning).

Live chart, "last 5 minutes":

```go
chart.AddTimer(1000, func() {
  now := time.Now()
  src.AddPoint(ui.NewTimeChartValue(now, readSensor()))
  chart.SetDefaultTimeRange(now.Add(-5*time.Minute), now)
})
```

The user can pan into the history at any time; Esc brings back the live
window.

### Zoom

Each zoom-in (right drag, Ctrl + right drag) saves the current view - time
range, default-view state and Y ranges of all areas - on a stack, so you can
drill into details and come back step by step.

- `ZoomBack()` - restores the view saved by the last zoom-in; with an empty
  stack resets to the default view. If the saved level was the default view,
  the *current* default range is shown.
- `ZoomDepth() int` - how many levels `ZoomBack` can go back. Shown on the
  chart as "Zoom level N".
- `ResetZoom()` - clears the stack, turns on Y auto-scaling in every area and
  switches to the default view.

### Selection

A time range the user marks with Shift + left drag. It is drawn over all
areas with edge lines and a header showing its duration and a close button.

- `Selection() (from, to time.Time, ok bool)` - `ok` is `false` when nothing is selected.
- `SetSelection(from, to time.Time)`
- `ClearSelection()`
- `SetOnSelectionChanged(f func())` - called on every change, including each
  mouse move while the selection is created, moved or resized.

```go
chart.SetOnSelectionChanged(func() {
  if from, to, ok := chart.Selection(); ok {
    status.SetText(fmt.Sprintf("%s - %s", from.Format("15:04:05"), to.Format("15:04:05")))
  }
})
```

## TimeChartArea

### Series

- `AddSeries(name string, source TimeChartDataSource) *TimeChartSeries` - adds a
  series; the name is shown in the legend. Series are drawn in the order they
  were added (later ones on top).
- `Series() []*TimeChartSeries`

### Y axis

By default the Y axis auto-scales to the visible data of all series of the
area (with a 5% margin).

- `SetYRange(yMin, yMax float64)` - fixed Y range (also set by Ctrl + right drag).
- `YRange() (yMin, yMax float64, ok bool)` - `ok` is `false` while the area auto-scales.
- `ResetYRange()` - back to auto-scaling.

### Markers

Text notes pinned to a point of the area, e.g. detected peaks or events.

```go
type TimeChartMarker struct {
  DT    time.Time
  Value float64     // position on the area's Y axis
  Text  string
  Color color.Color // nil = theme's marker color
}
```

- `AddMarker(m TimeChartMarker)` - markers may be added in any order.
- `SetMarkers(markers []TimeChartMarker)` - replaces all markers.
- `Markers() []TimeChartMarker`
- `ClearMarkers()`

A marker is a small triangle pointing at the value with the text above it
(below when there is no room). Texts that would overlap an already drawn one
are skipped, so a zoomed-out view stays readable; zoom in to see them all.

```go
area.AddMarker(ui.TimeChartMarker{
  DT:    t,
  Value: v,
  Text:  fmt.Sprintf("peak %.0f ms", v),
})
```

## TimeChartSeries

- `Name() string`
- `Source() TimeChartDataSource`
- `SetType(seriesType TimeChartSeriesType)`:
  - `TimeChartSeriesLine` (default) - a line; at each point a vertical bar
    from `Low` to `High`, and `Last` of a point is joined to `First` of the next.
  - `TimeChartSeriesCandles` - OHLC candles, green when `Last >= First`, red otherwise.
- `SetPaletteColor(index int)` - color number `index` of the chart palette. The
  palette has a dark and a light variant, so the series stays readable in both
  themes. Without a call, series get palette colors in the order they are added.
- `SetColor(col color.Color)` - a fixed color that does not change with the theme.

```go
area := chart.AddArea()
area.AddSeries("Ping, ms", pingSrc).SetPaletteColor(0)
area.AddSeries("Average", avgSrc).SetPaletteColor(1)

prices := chart.AddArea()
prices.AddSeries("BTC", ohlcSrc).SetType(ui.TimeChartSeriesCandles)
```

## Mouse and keyboard

| Action                                   | Result                                      |
|------------------------------------------|---------------------------------------------|
| Left drag                                | pan horizontally                            |
| Wheel                                    | zoom time around the pointer                |
| Right drag to the right                  | zoom into the dragged time range            |
| Ctrl + right drag to the right           | zoom into the dragged rectangle (time and Y of that area) |
| Right drag to the left, double click     | zoom back one level (`ZoomBack`)            |
| Esc                                      | reset zoom (`ResetZoom`)                    |
| Shift + left drag                        | select a time range                         |
| Left drag on the selection header        | move the selection                          |
| Left drag on a selection edge            | resize the selection                        |
| Click on × in the selection header       | clear the selection                         |

Esc works after the chart has been clicked (it takes focus on click). When
there is nothing to reset, Esc is passed on to the form, so it still closes
dialogs.

## Appearance

- Colors follow the application theme (`ui.ApplyDarkTheme()` /
  `ui.ApplyLightTheme()`); call `Form().Update()` after switching.
- The time axis uses a slightly smaller font: tick times on the first row and
  a ribbon of dates below it. Each date is centered on the visible part of its
  day, so a range crossing midnight shows both dates. On larger scales the
  ribbon switches to months and years.
- Props: `padding` (int, default `6`) - outer padding.

## Notes

- Call chart methods from the UI thread (for example from a widget timer).
  Only `TimeChartMemorySource.AddPoint`/`SetPoints` may be called from other
  goroutines; trigger the repaint from the UI thread.
- The form captures the mouse only for the left button. While zooming with the
  right button, the zoom band stops following the pointer when it leaves the
  chart; the release is still handled.
