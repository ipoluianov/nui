# Checkbox

Boolean input widget.

## Create

```go
cb := ui.NewCheckbox("Enable feature")
cb.SetChecked(true)
```

## Event (no params)

```go
cb.SetOnStateChanged(func() {
  ev := ui.CurrentEvent()
  if e, ok := ev.Parameter.(*ui.EventCheckboxStateChanged); ok {
    fmt.Println("checked:", e.Checked)
  }
})
```

