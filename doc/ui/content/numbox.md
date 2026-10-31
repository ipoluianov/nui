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

Defaults: 2 decimals, no limits (`min` and `max` are infinite), value 0.

## Change events

```go
nb.OnValueChanged.Connect(func(v float64) { // any change, also by SetValue
  fmt.Println("changed to", v)
})
nb.OnActivated.Connect(func(v float64) { // the user's: typed and left the box or pressed Enter, arrows, buttons, wheel
  save(v)
})
```

The old `SetOnValueChanged(func())` / `SetOnChanged(func())` still work. See [events](events.md).


## Disabled

```go
nb.SetEnabled(false) // grayed out: ignores the mouse, the keyboard and focus, keeps its value
```
