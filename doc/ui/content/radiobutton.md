# RadioButton

Single-choice widget (exclusive within the same parent container). A click or Space checks it.

## Create

```go
rb1 := ui.NewRadioButton("A")
rb2 := ui.NewRadioButton("B")
rb1.SetChecked(true)
```

Checking a button unchecks the other `RadioButton`s of the same parent: put each group in its own panel.

## Events

```go
rb1.OnCheckedChanged.Connect(func(checked bool) {
  // got or lost the check: by the user, by SetChecked, or by another button of the group
})
rb1.OnActivated.Connect(func(bool) {
  // the user chose this button
})
```

The old `SetOnStateChanged(func(btn *ui.RadioButton, checked bool))` still works. `Checked()` reads the state.
