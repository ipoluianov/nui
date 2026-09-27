# Button

Clickable button.

## Create

```go
btn := ui.NewButton("Save")
```

## Click handler

```go
btn.SetOnClick(func() {
  // handle click
})
```

## Notes

- Button also reacts to `Enter` / `Space` when focused.
