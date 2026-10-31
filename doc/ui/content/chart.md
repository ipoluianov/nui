# Chart

`Chart` is the simple chart of the earlier versions of nui, kept for the
code that uses it: one line of `[]ui.ChartPoint`. It is a `ChartView` with
one line; new code takes `ui.NewLineChart()` or another widget of the
`ChartView` family, which has more lines, bars, pies, time axes and the
rest: see [Charts](charts.md).

## Create

```go
ch := ui.NewChart()
ch.SetProp("xlabel", "time")
ch.SetProp("ylabel", "value")
ch.SetData([]ui.ChartPoint{
  {X: 0, Y: 1},
  {X: 1, Y: 3},
})
```

`Data()` returns the points; `SetData` replaces them and repaints the chart.
The points are drawn in the order of X.

## Useful props

- `xlabel`, `ylabel` (string): the titles of the axes
- `padding` (int): the space around the plot, 6 by default

The ticks are picked for the size of the chart so their labels never
overlap. Being a `ChartView`, the chart zooms with the mouse, shows a
tooltip and exports to PNG, SVG and CSV.
