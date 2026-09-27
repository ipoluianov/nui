# TextBox

Single-line or multi-line text input.

## Create

```go
tb := ui.NewTextBox()
tb.SetHint("Type here")
```

## Single vs multi-line

```go
tb.SetMultiline(false) // default
// tb.SetMultiline(true)
```

## Read-only / password

```go
tb.SetReadOnly(true)
tb.SetIsPassword(true)
```

## Events

```go
tb.SetOnTextChanged(func() {
  fmt.Println(tb.Text())
})
```
