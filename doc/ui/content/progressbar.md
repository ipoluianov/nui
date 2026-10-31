# ProgressBar

Value indicator between min/max.

## Create

```go
pb := ui.NewProgressBar(0, 100, 25) // min, max, value
pb.SetText("25%")                   // drawn in the middle; "" (default) - none
```

## Update value

```go
pb.SetValue(50) // kept in [min, max]
```

`Value()`, `Text()`, `SetMinValue(v)`, `SetMaxValue(v)`.
