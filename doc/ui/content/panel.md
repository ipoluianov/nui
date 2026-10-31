# Panel

`Panel` is a basic container widget: it lays its children out in a grid (see [Layouts](layouts.md)).

## Create

```go
p := ui.NewPanel()
p.SetXExpandable(true)
```

## Add children

```go
p.AddWidget(0, 0, ui.NewLabel("Name"))
p.AddWidget(0, 1, ui.NewTextBox())
p.AddButton(1, 1, "Save", save) // AddLabel, AddButton, AddPanel, AddHSpacer, AddVSpacer
p.AddVSpacer(2, 0)              // the free height stays below
```

Useful helpers:

- `NextGridRow()`, `NextGridColumn()` on `Widget` to append rows/columns: `p.AddWidget(p.NextGridRow(), 0, w)`.
- `SetPanelPadding(pixels)` (the room around the grid) and `SetCellPadding(pixels)` (between the cells), the `padding`/`spacing` props.
- `SetColumnStretch(col, n)`, `SetRowStretch(row, n)`, `AddWidgetSpan(row, col, rowSpan, colSpan, w)`.
- `RemoveWidget(w)`, `RemoveAllWidgets()`, `Widgets()`.
- `SetAutoFillBackground(true)` paints the panel's background; `SetElevation(n)` makes it lighter (darker in the light theme), as a [Frame](frame.md).
