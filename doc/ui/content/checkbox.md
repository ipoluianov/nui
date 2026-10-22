# Checkbox

Boolean input widget. A click or Space flips it (with `SetTristate(true)` it goes on to the
indeterminate state); Enter is left to the form, where it presses the accept button of a dialog.

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

