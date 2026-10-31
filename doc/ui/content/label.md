# Label

Text output widget. A text with `\n` takes several lines.

## Create

```go
lbl := ui.NewLabel("Hello")
```

## Text and styling

```go
lbl.SetText("Updated")
lbl.SetUnderline(true)
lbl.SetTextAlign(ui.HAlignCenter)
```

## For screen readers

```go
name := ui.NewTextBox()
ui.NewLabel("Name:").SetBuddy(name) // the field is read as "Name: text field"
```

`FormLayout.AddRow` does it for its rows.
