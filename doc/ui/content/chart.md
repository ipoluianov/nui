# Chart

Simple line chart for `[]ui.ChartPoint`.

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

- `tickcount` (int): number of ticks (default `5`)
- `xlabel`, `ylabel` (string)
- `padding` (int)

