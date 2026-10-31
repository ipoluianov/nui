# Button

Clickable button.

## Create

```go
btn := ui.NewButton("Save")
```

## Click handler

```go
btn.OnClick.Connect(func(w ui.Widgeter) {
  // handle click
})
```

The old `btn.SetOnClick(func() { ... })` still works (one handler).

## Notes

- Button also reacts to `Enter` / `Space` when focused.
- `Push()` clicks it from the code (nothing happens when it is disabled).
- `SetText(s)` / `Text()`; the button grows to fit its text (min width `ui.DefaultButtonMinWidth`).
- `SetRole("primary")` paints it in the accent color, for the main action of a dialog.
- In a dialog, `form.SetAcceptButton(btn)` / `form.SetCancelButton(btn)` make Enter / Esc press it.
- `SetEnabled(false)` grays it out and ignores clicks.
