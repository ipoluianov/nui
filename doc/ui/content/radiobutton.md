# RadioButton

Single-choice widget (exclusive within the same parent container).

## Create

```go
rb1 := ui.NewRadioButton("A")
rb2 := ui.NewRadioButton("B")
rb1.SetChecked(true)
```

## Event

```go
rb1.SetOnStateChanged(func(btn *ui.RadioButton, checked bool) {
  // ...
})
```

