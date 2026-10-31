# Checkbox

Boolean input widget. A click or Space flips it (with `SetTristate(true)` it goes on to the
indeterminate state); Enter is left to the form, where it presses the accept button of a dialog.

## Create

```go
cb := ui.NewCheckbox("Enable feature")
cb.SetChecked(true)
```

## Events

```go
cb.OnCheckedChanged.Connect(func(checked bool) { // any change, also SetChecked
  fmt.Println("checked:", checked)
})
cb.OnActivated.Connect(func(checked bool) { // a click or Space of the user
  save(checked)
})
```

The old `SetOnStateChanged(func())` still works. See [events](events.md).

## Three states

```go
cb.SetTristate(true)       // a click goes unchecked -> checked -> indeterminate
cb.SetIndeterminate(true)  // the dash; Checked() is false then
```

`Checked()`, `Indeterminate()`, `SetText(s)` / `Text()`.
