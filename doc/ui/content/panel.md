# Panel

`Panel` is a basic container widget (grid layout by default).

## Create

```go
p := ui.NewPanel()
p.SetXExpandable(true)
```

## Add children

```go
p.AddWidgetOnGrid(ui.NewLabel("Name"), 0, 0)
p.AddWidgetOnGrid(ui.NewTextBox(), 0, 1)
```

Useful helpers:

- `NextGridRow()`, `NextGridColumn()` on `Widget` to append rows/columns.
- `SetPanelPadding(pixels)` and `SetCellPadding(pixels)` (aliases for `padding`/`spacing` props).
