# NumBox

`NumBox` is a single-line `float64` input with:

- fixed decimals formatting
- `min/max` clamping
- step buttons (▲/▼), keyboard arrows, and mouse wheel
- text selection + editing

## Create

```go
nb := ui.NewNumBox()
nb.SetDecimals(3)
nb.SetMin(-10)
nb.SetMax(10)
nb.SetValue(1.234)
```

## Value API

- `SetValue(v float64)`
- `Value() float64` / `GetValue() float64`

## Step API

```go
nb.SetStep(0.1) // optional
```

If `step == 0`, `Step()` defaults to \(10^{-decimals}\) (or `1` when `decimals == 0`).

## Change events (no params)

```go
nb.SetOnChanged(func() {
  if ev := ui.CurrentEvent(); ev != nil {
    if e, ok := ev.Parameter.(*ui.EventNumBoxValueChanged); ok {
      fmt.Println("changed to", e.Value)
    }
  }
})
```


## Disabled

```go
nb.SetEnabled(false) // grayed out: ignores the mouse, the keyboard and focus, keeps its value
```
