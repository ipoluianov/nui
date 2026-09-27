# GroupBox

A panel framed with a title, grouping related controls. Children are laid out on the grid like in
a `Panel` (`AddWidget`, `AddLabel`...), below the title.

```go
conn := ui.NewGroupBox("Connection")
conn.AddLabel(0, 0, "Host")
conn.AddWidget(0, 1, hostBox)
conn.AddLabel(1, 0, "Port")
conn.AddWidget(1, 1, portBox)
```

- `SetTitle(text)` / `SetTitleFunc(f)` (follows the language) / `Title()`
- `SetPanelPadding`, `SetCellPadding` - as for any container.
