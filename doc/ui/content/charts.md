# Charts

All charts of nui are one widget, `ChartView`, set up for their kind: line
and area charts, columns and bars, pies and donuts, scatter and bubble
charts, candles, sparklines, time series. They share the axes, the legend,
the tooltip, the zoom, the selection, the themes and the export.

## A chart in three lines

```go
ch := ui.NewLineChart()
ch.AddValues(3, 5, 4, 7, 6, 9, 8, 10, 9, 12)
form.Panel().AddWidget(0, 0, ch)
```

No data model is needed for a small chart: the ready-made widgets take the
points straight away and keep them in memory.

| Widget | Methods | Notes |
|---|---|---|
| `ui.NewLineChart()` | `AddValues(ys...)`, `SetValues`, `AddXY(x, y)`, `AddPoints([]ui.ChartPoint)`, `SetPoints`, `Clear`, `SetMaxPoints(n)`, `Series(name)` | `Series` makes a line per name |
| `ui.NewBarChart()` | `Add(category, value)`, `Series(name).Set(category, value)`, `SetCategories`, `SetHorizontal`, `SetStacked` | categories are added as they are named |
| `ui.NewPieChart()` | `Add(name, value)`, `SetDonut(hole)`, `Pie()` | legend on the right, a click on an entry hides the slice |
| `ui.NewSparkline()` | `SetValues`, `AddValue`, `SetMaxValues`, `SetKind` | no axes, no frame; `ui.DrawSparkline` draws one in a table cell |
| `ui.NewXYChart()` | a `ChartView` of linear axes | add any series to its panes |
| `ui.NewTimeChart()` | see [TimeChart](timechart.md) | a `ChartView` of a time axis |
| `ui.NewChart()` | `SetData([]ui.ChartPoint)` | the simple chart of the earlier versions: see [Chart](chart.md) |

```go
ch.Series("Plan").AddXY(1, 10).AddXY(2, 12)
ch.Series("Fact").AddPoints([]ui.ChartPoint{{1, 9}, {2, 13}})

bar := ui.NewBarChart()
bar.Add("Jan", 120).Add("Feb", 95).Add("Mar", 140)

pie := ui.NewPieChart()
pie.Add("Linux", 60).Add("Windows", 30).Add("macOS", 10)
```

Changes of the data repaint the chart by themselves; the memory sources may
be written from any goroutine. In the XML layout:
`<linechart values="3,5,4,7" title="Load"/>`, `<barchart items="Jan:120|Feb:95"/>`,
`<piechart items="Linux:60|Windows:30" donut="0.5"/>`.

## Structure

```
ChartView          the widget: title, legend, tooltip, gestures, export
 └─ ChartPane      a plot with its axes; panes are stacked, sharing the x axis
     ├─ ChartAxis  scale, side, title, grid, visible range
     ├─ series     LineSeries, BarSeries, ScatterSeries, CandleSeries, PieSeries...
     └─ layers     annotations and the drawing of the application, over or under the series
```

```go
ch := ui.NewChartView()
price := ch.Pane(0)
price.SetWeight(3)              // three quarters of the height
price.AddCandles("ACME").SetSource(src)
price.AddLine("Average 20").SetSource(average)
volume := ch.AddPane().AddBars("Volume")
volume.SetSource(volumes)
```

A pane may have more vertical axes: `pane.AddAxis("y2", ui.ChartAxisRight, nil)`
and `series.SetAxisNames("x", "y2")` put a series of other units on the right.

## Axes and scales

| Scale | For |
|---|---|
| `ui.NewLinearScale(min, max)` | ordinary values; `Unit`, `SI` (1.5k, 20M), `Integer`, `Formatter` |
| `ui.NewLogScale(min, max)` | many orders of magnitude; values above 0 only |
| `ui.NewSymlogScale(min, max)` | both signs over many orders of magnitude |
| `ui.NewTimeScale(from, to)` | time: values are `ui.ChartTime(t)`, seconds since the Unix epoch; ticks on the calendar, a ribbon of dates under them |
| `ui.NewCategoryScale(names...)` | named categories; category i is at the value i |

The ticks are picked for the length of the axis so their labels never
overlap. The visible range of an axis is, in this order: the one set by
`SetRange` or by the user's zoom, the default one of `SetDefaultRange` (the
last minutes of a live chart), or the one that fits the data, with
`SetMargin` and `SetIncludeZero`. `SetInverted`, `SetLabelAngle`,
`SetTitle`, `SetGrid(major, minor)`, `SetPosition` set up the rest.

## Series

| Series | Kinds | Main settings |
|---|---|---|
| `LineSeries` | line, area, stacked area, 100% area, steps, spline | `SetShape`, `SetLineWidth`, `SetDash`, `SetMarkers`, `SetFill(fill, opacity, gradient)`, `SetStack(group)`, `SetGapMode` |
| `BarSeries` | columns, bars, grouped, stacked, 100%; a bar per sample along a numeric or time axis (volume) | `Set(category, v)`, `SetHorizontal`, `SetStack`, `SetGap`, `SetLabels`, `SetUpDownColors` |
| `ScatterSeries` | scatter, bubbles, colored by value, density of millions of points | `Add`, `AddPoint(ui.ScatterPoint)`, `SetMarker`, `SetBubbleSizes`, `SetColorScale`, `SetDensityThreshold` |
| `CandleSeries` | Japanese candles, OHLC bars | `AddOHLC`, `SetStyle`, `SetSource` |
| `PieSeries` | pie, donut, half circle | `Add`, `SetInnerRadius`, `SetAngles`, `SetLabels`, `SetOther`, `SetSliceColor` |

More series, added to a pane by `AddHistogram`, `AddBoxPlot`, `AddBand`,
`AddHeatmap`, `AddHistogram2D`, `AddWaterfall`, `AddFunnel`, `AddSpans`,
`AddDigital`, `AddEvents`, `AddRadar`, `AddPolar`:

| Series | For | Main settings |
|---|---|---|
| `HistogramSeries` | the distribution of raw values | `SetValues`, `SetBins`, `SetMode` (count, density, cumulative) |
| `BoxPlotSeries` | box plots and violins by category | `Set(category, values)`, `SetViolin`, `SetMean`; `ui.ComputeBoxStats` |
| `BandSeries` | a corridor around a line, error bars | `Add(x, y, low, high)`, `SetErrorBars` |
| `HeatmapSeries` | a grid of values as colors, with a color bar; a raster for big grids | `SetGrid`, `SetColors`, `SetLabels` |
| `Histogram2DSeries` | the counts of points in cells or hexagons | `SetPoints`, `SetBins`, `SetHex` |
| `WaterfallSeries` | a bridge of changes to a total | `AddStep`, `AddTotal` |
| `FunnelSeries` | the stages of a process with the conversion | `Add` |
| `SpanSeries` | Gantt charts, state timelines | `Add(ui.ChartSpan)`, `AddDependency`, `SetStateColor` |
| `DigitalSeries` | binary signals and buses | `AddState`, `SetBus`, `SetStateName` |
| `EventSeries` | events and alarms by source and severity | `Add(ui.ChartEvent)` |
| `RadarSeries`, `PolarSeries` | radar charts, polar lines, wind roses | `Set`, `Add`, `SetKind`, `SetStack` |

`ScatterSeries.SetStems` draws a stem plot. The ready-made widgets:
`ui.NewParetoChart()`, `ui.NewControlChart()` (a Shewhart chart with the
Western Electric rules), `ui.NewGauge(min, max)`,
`ui.NewLevelIndicator(min, max)`, `ui.NewKPITile(title)`;
`ui.GaugeZonesOf` makes the usual zones.

`ui.NewChartSyncGroup().Add(a, b, c)` ties the range, the crosshair and the
selection of charts; `SaveState` and `RestoreState` keep the view between
starts.

`ChartPane.SetStackPercent(group, true)` shows a stack group in percent of
its total. Every series has `SetColor`, `SetPaletteColor`, `SetVisible`,
`SetName`. A series of an application's own embeds `ChartSeriesBase` and
implements the rest of `ChartSeries`.

## Data sources

The series along a numeric or a time axis read their samples from a
`ChartSource`:

```go
type ChartSource interface {
	GetData(from, to, group float64) []ui.ChartSample
}
```

`GetData` is called on the paint for the visible range; `group` is the part
of the axis a pixel takes, so a source returns about a sample per pixel
whatever the amount of data (`ui.ChartDownsample` and `ui.ChartAggregator`
merge the samples into buckets keeping their first, last, high and low).
A `ChartSample` carries the quality of the data: `ui.ChartValue`,
`ui.ChartBad` (hatched), `ui.ChartGap` (the line breaks).
`ui.NewChartMemorySource()` keeps samples in memory (`Append`, `SetSamples`,
`SetMaxPoints` for a live feed); `ui.ChartSourceFunc` makes a source of a
function. `ui.ChartLTTB` thins a smooth line down to n points.

## Mouse and keyboard

What the user's gestures do is a table of bindings, filled by a profile and
changed one gesture at a time:

```go
in := ch.Interaction()
in.UseProfile(ui.ChartProfileFinance)
in.Bind("Shift+LeftDrag", ui.ChartActionSelectRange)
in.Bind("Ctrl+Wheel", ui.ChartActionZoomWheel.OnAxes(ui.ChartAxesY))
in.Unbind("RightDrag")
in.SetLimits(ui.ChartLimits{MinSpanX: 1, PanBounds: ui.ChartPanData})
in.BindFunc("Alt+LeftDrag", ui.ChartGestureFunc{
	End: func(e ui.ChartGestureEvent) { addCut(e.StartDataX, e.DataX) },
})
```

A gesture is modifiers (`Ctrl`, `Shift`, `Alt`, `Cmd`, `Mod`) and one of
`LeftDrag`, `RightDrag`, `MiddleDrag`, `LeftClick`, `RightClick`,
`LeftDblClick`, `RightDblClick`, `Wheel`, `WheelX` or a key (`Esc`, `Home`,
`Left`, `Plus`...), optionally limited to a zone: `@plot` (the default),
`@axis`, `@xaxis`, `@yaxis`, `@legend`, `@any`.

| Gesture | Scientific (default) | TimeSeries (TimeChart) | Finance | Dashboard |
|---|---|---|---|---|
| LeftDrag | zoom into the rectangle | pan x | pan x and y | — |
| Shift+LeftDrag | select a range | select a range | select a range | — |
| RightDrag | pan | zoom into a range of x | — | — |
| Wheel | zoom x and y | zoom x | zoom x, the right edge kept | — |
| LeftDrag@axis | stretch the axis | — | stretch the axis | — |
| LeftDblClick | reset | zoom back | fit the axes | — |
| LeftClick | `OnPointClicked` | `OnPointClicked` | `OnPointClicked` | `OnPointClicked` |
| LeftClick@legend | hide the series | hide the series | hide the series | hide the series |
| Esc, Home | reset the zoom | reset the zoom | reset the zoom | — |

`ui.ChartProfileStatic` turns everything off. `SetTool` with
`ToolAction` gives a toolbar of pan / zoom / select buttons. Every action is
a method too: `ZoomBack`, `ResetZoom`, `SetXRange`, `SetSelection`,
`ClearSelection`. The signals: `OnViewChanged`, `OnSelectionChanged`,
`OnHover`, `OnPointClicked`, `OnSeriesToggled`, and `OnBeforeAction`, which
may cancel an action or change its range.

## Tooltip, legend, annotations

`SetHoverMode(ui.ChartHoverX, true)` shows the values of all the series at
the pointer with a crosshair; `ui.ChartHoverNearest` the nearest element;
`SetTooltipFunc` sets the rows. `SetLegend` puts the legend above each pane,
on the right or nowhere.

```go
ann := ch.Pane(0).Annotations()
ann.AddHLine(23, "Limit").SetDash(4, 3)
ann.AddXZone(0, 6, "Night")
ann.AddLabel(15, 24.1, "Peak")
```

The right-click menu resets the zoom, shows and hides the series, copies the
picture and the data; `SetContextMenuFunc` adds the application's items.

## Themes

The colors follow `ui.ApplyDarkTheme` and `ui.ApplyLightTheme`: the palette
of 10 series colors, the up/down, status and heat map colors have a variant
for each theme. `SetTheme(ui.ChartThemeOf(&ui.LightPalette, false))` fixes
the colors of a chart; `ui.ColorScale` picks a color on a scale.

## Export and printing

```go
img := ch.Render(800, 500, 2, nil)            // *image.RGBA, twice as sharp
ch.WritePNG(f, 800, 500, 2, nil)
ch.WriteSVG(f, 800, 500, nil)                 // vectors and texts
ch.WriteCSV(f, true)                          // the data of the series
ch.CopyImage()                                // into the clipboard
form.Print(ch.PrintJob("Sales"), nil)         // a page, in the light theme
```

The SVG comes from `ui.SVGRecorder`, which records what a `Canvas` draws:
see [Canvas](canvas.md).

## Performance

A chart draws about a sample per pixel: the sources merge the rest. A line
of 1920 points is stroked in about 2 ms, a scatter of more than 20000 points
is drawn as its density. `GetData` runs on the UI thread on every paint, so
a source over a database keeps the recent ranges at hand.
