# ListView

A scrolled list of the rows of a model. Only the visible rows are drawn: a list of a million rows is as fast as one of ten.

```go
list := ui.NewListView()
list.SetItems([]string{"one", "two", "three"})
list.OnActivated.Connect(func(row int) { open(row) })       // double click, Enter
list.OnSelectionChanged.Connect(func(row int) { show(row) }) // any change
```

## Models

A model is the data: `ui.ItemModel` (`RowCount`, `ColumnCount`, `Text(row, col)`). The list shows the first column; a `Table` shows the same models (`table.SetModel`).

```go
// The data stays in the application
logs := ui.NewFuncModel(func() int { return len(entries) }, 1,
	func(row, col int) string { return entries[row].Message })
list.SetModel(logs)
entries = append(entries, e)
logs.Changed() // the views follow
```

- `ui.NewListModel(texts...)` keeps texts with data: `Append`, `Insert`, `Remove`, `Set`, `SetItems`, `Data(row)`; it tells its views of the changes itself.
- Your own model: implement `ItemModel`; embed `ui.ModelSignal` and call `Changed()` to tell the views. A model without it: call `list.Refresh()` after changing it.
- Optional: `ItemStyler` (`Style(row, col) ui.ItemStyle`: color, bold, italic, alignment, icon), `ColumnNamer` (Table headers), `ItemEditor` (cells the user edits in a Table).

## Selection

- `CurrentRow()`, `SetCurrentRow(row)` (-1 clears), `SelectedRows()`, `IsRowSelected(row)`.
- `SetMultiselect(true)`: Ctrl+click toggles a row, Shift+click and Shift+keys select a range, Ctrl+A all (`SelectAll`).
- Keys: arrows, PageUp/PageDown, Home/End, Enter.

## Look

- `SetRowHeight(h)`: one height for all the rows.
- `SetEmptyText("No results")`: shown in an empty list.
- `SetItemPainter(func(p *ui.ItemPaint))`: paint the rows yourself (`p.Canvas` clipped to the row; `p.Row`, `p.Width`, `p.Height`, `p.Selected`, `p.Current`, `p.Focused`).
- `ItemRect(row)`, `RowAt(x, y)`, `EnsureVisible(row)`: where the rows are.
