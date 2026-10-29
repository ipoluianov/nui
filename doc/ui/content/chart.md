# Chart

Simple line chart for `[]ui.ChartPoint`: a `ChartView` with one line. For
more lines, bars, pies and the rest see [Charts](charts.md).

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

## Useful props

- `xlabel`, `ylabel` (string): the titles of the axes
- `padding` (int)

The ticks are picked for the size of the chart so their labels never
overlap. Being a `ChartView`, the chart zooms with the mouse, shows a
tooltip and exports to PNG, SVG and CSV.
