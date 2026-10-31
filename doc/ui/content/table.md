# Table

Grid/table widget with selection and editing support.

## Create

```go
t := ui.NewTable()
```

`Table` embeds `Widget`, so all generic widget methods (position, size, colors,
focus, etc.) are available too. The methods below are the ones that belong to
`Table` specifically.

## Signals

```go
t.OnSelectionChanged.Connect(func(c ui.TableCell) { show(c.Row) })          // any change of the current cell or the selection
t.OnCellDoubleClick.Connect(func(e *ui.TableCellEvent) { open(e.Row); e.Handled = true }) // Handled: no editor
t.OnCellEdited.Connect(func(c ui.TableCell) { save(c.Row, c.Col, c.Text) }) // the user accepted an edit
```

- `OnSelectionChanged(TableCell)` - the current cell or the selection changed, by the user or the code.
- `OnCellChanged(TableCell)` - the text of a cell changed: `SetCellText2` or an edit.
- `OnCellEdited(TableCell)` - the user edited a cell and accepted the text.
- `OnCellDoubleClick(*TableCellEvent)` - `Row`, `Col`, `Button`, `Mods`; `Handled = true` keeps the table from editing the cell.
- `OnColumnClick(int)` - a header column was clicked.

See [events](events.md). The `SetOn…` setters below still work.

## A model

`SetModel(model ui.ItemModel)` shows a model instead of the table's own cells: the table keeps no cells then, and a model of a million rows is as fast as one of ten. The model gives the rows, the columns (named by a `ColumnNamer`), the texts (styled by an `ItemStyler`); `SetCellText2` and the user's edits go to an `ItemEditor`. A model with `ui.ModelSignal` is followed by itself; after changing another one, call `RefreshModel()`. See [ListView and models](listview.md).

```go
t.SetModel(ui.NewFuncModel(func() int { return len(rows) }, 3,
	func(row, col int) string { return rows[row][col] }))
```

## Rows & columns

- `SetRowCount(count int)` / `RowCount() int` - how many rows are shown. The cells of the rows beyond the count are kept and come back when the count grows again.
- `ClearRows()` - removes all the rows (cells, selection, current cell), as in a new table; `RowCount()` becomes 0. To load other data: `ClearRows()`, fill the cells, `SetRowCount(n)`.
- `SetColumnCount(count int)` / `ColumnCount() int`
- `SetColumnName(col int, name string)` / `ColumnName(col int) string` - header text for column `col` (header row 0).
- `SetColumnCellName2(row, col int, name string)` - header text for a specific header row/column, for use with multi-row headers.
- `SetColumnWidth(col int, width int)` / `ColumnWidth(col int) int`
- `SetColumnImage(col int, img image.Image, imgWidth int)` - icon drawn in the column header.
- `SetColumnHAlign(col int, align HAlign)` - aligns the header text, e.g. to the right over numbers.
- `SetStretchLastColumn(true)` - the last column fills the table up to its right edge.
- `SetRowHeight(height int)` / `RowHeight() int` - row height applies to every row.
- `SetCellPadding(padding int)` / `CellPadding() int`
- `SetHeaderRowCount(count int)` - number of header rows (for multi-row headers).
- `SetHeaderCellSpan2(row, col, spanRow, spanCol int)` - merge header cells, like an HTML `colspan`/`rowspan`.

## Cell content

- `SetCellText2(row, col int, text string)` / `GetCellText2(row, col int) string`
- `SetCellDisplayText(row, col int, text string)` - text shown instead of the underlying value (e.g. formatted), without changing what editing starts from.
- `SetCellData2(row, col int, data interface{})` / `GetCellData2(row, col int) interface{}` - arbitrary payload attached to a cell.
- `SetCellImage(row, col int, img image.Image, imgWidth int)`
- `SetCellColor(row, col int, color color.Color)` - text color override.
- `SetCellHAlign(row, col int, align HAlign)` / `SetCellVAlign(row, col int, align VAlign)`
- `SetCellContraction(row, col int, contraction bool)` - truncate overflowing text with `..` instead of overflowing/clipping silently.
- `SetCellOnDraw(row, col int, onDraw func(cnv *Canvas))` - fully custom cell rendering; when set, the default text/image drawing for that cell is skipped.

## Editing

- `SetEditTriggerDoubleClick(enabled bool)` / `SetEditTriggerEnter(enabled bool)` / `SetEditTriggerF2(enabled bool)` / `SetEditTriggerKeyDown(enabled bool)` - table-wide edit triggers.
- `SetCellEditTriggerDoubleClick(row, col int, enabled bool)` / `SetCellEditTriggerEnter(...)` / `SetCellEditTriggerF2(...)` / `SetCellEditTriggerKeyDown(...)` - per-cell overrides of the above.
- `EditCurrentCell(enteredText string)` - opens the inline text editor on the current cell; pass `""` to start from the cell's existing text.
- `EditCell(row, col, colSpan int, text string, onDone func(text string, accepted bool)) *TextBox` - an editor over the cells with the text (in-place renaming); `onDone` gets the text (`accepted` false on Esc), the cell itself is not changed. `CancelCellEditor()`, `IsEditingCell()`.
- `SetOnCellChanged(func(row, col int, text string, data interface{}) bool)` - called when a cell gets a new text (an edit or `SetCellText2`); return `false` to reject the new text and revert it.

## Selection

- `SetSelectingRows(selectingRows bool)` / `SelectingRows() bool` - selection unit: whole rows (`true`, the default) or individual cells (`false`).
- `SetMultiselect(enabled bool)` / `Multiselect() bool` - allow selecting more than one row/cell at once (default `false`). When enabled: drag to select a range, Shift+click/Shift+Arrow/Home/End/PageUp/PageDown extends a range from the anchor, Ctrl+click toggles a single item, Ctrl+Shift+click adds a range, Ctrl+A selects everything. Only the left mouse button drives selection. When disabled, a plain drag still moves the single selection to follow the mouse.
- `SelectAll()` - selects every row/cell; no-op unless `Multiselect()` is `true`.
- `ClearSelection()` - removes the selection; `CurrentRow()` returns `-1` until a row is clicked or navigated to again.
- `SetClearSelectionOnEmptyClick(enabled bool)` / `ClearSelectionOnEmptyClick() bool` - whether a plain left click on the empty area of the table (outside any cell) clears the selection (default `true`).
- `SelectedRows() []int` - sorted selected row indices; meaningful when `SelectingRows()` is `true`.
- `SetSelectedRows(rows []int)` - selects the rows (e.g. to keep a selection after reloading them); the first one becomes current.
- `SelectedCells() []TableCellPos` - sorted selected cells; meaningful when `SelectingRows()` is `false`.
- `IsRowSelected(row int) bool` / `IsCellSelected(row, col int) bool`
- `SetCurrentCell2(row, col int)` - move the active cell/row programmatically (scrolls it into view, resets the selection to just this item, fires `OnSelectionChanged`).
- `CurrentRow() int` / `CurrentColumn() int` - the active cell, or `-1` if none.
- `PreviousCurrentCellX() int` / `PreviousCurrentCellY() int` - active cell before the last change, useful inside an `OnSelectionChanged` callback.
- `SetCellSelectionDisabled(row, col int, disabled bool)` - excludes a cell/row from selection (clicks on it are ignored).
- `SetShowSelection(show bool)` / `ShowSelection() bool` - whether the selection highlight is drawn at all.
- `ScrollToCell2(row, col int)` - scroll so the given cell is visible.
- `CopySelectionToClipboard()` - copies the selection as text spreadsheets paste: rows one per line, cells separated by tabs; without a selection, the current cell.
- `SetOnSelectionChanged(func(row, col int))` - the old setter of `OnSelectionChanged`: called whenever the active cell/row moves (click, drag, keyboard nav, `Ctrl+A`).

See `examples/contacts` (sorting, filtering, multiselect, context menu) for a full demo.

## Mouse callbacks

- `SetOnCellMouseDown(func(button ui.MouseButton, row, col, x, y int, mods ui.KeyModifiers))`
- `SetOnCellMouseUp(...)` - the same arguments, on release; row and col are -1 outside the cells.
- `OnCellDoubleClick` (see Signals) - a double click on a cell. `SetOnCellMouseDblClick(func())` with `ui.CurrentEvent()` is deprecated.
- `SetOnColumnClick(func(col int))` - fired when a header column is clicked (or `OnColumnClick`).
- `CellAt(x, y int) (row, col int)` - the cell at a point of the table, -1, -1 for none.
- `SetHotTracking(true)` - the current cell follows the mouse, as in a drop-down list.
- `SetOnColumnResize(func(col, newWidth int))` - fired while a header column border is dragged to resize it.
- `SetContextMenu(menu *ContextMenu)` - shown on a right click. The click first selects the row (cell) under the mouse, unless it is already selected, so the menu acts on what was clicked and a multi-selection is kept.

## Appearance

- `SetCellBorderColor(col color.RGBA)` / `CellBorderColor() color.Color`
- `SetCellBorderWidth(width int)` / `CellBorderWidth() int`
- `SetModeLoading(loading bool, text string)` - replaces the table body with a centered loading message.

## Embedding widgets in cells

- `AddWidgetOnTable(widget Widgeter, posCellRow, posCellCol, widthInCells, heightInCells int)` - places any widget on top of the grid, spanning the given number of cells; it's repositioned/resized automatically as columns resize or the table scrolls.

## Types

- `TableCellPos{Row, Col int}` - returned by `SelectedCells()`.
- `TableCell{Row, Col int, Text string}` - the data of the signals.
- `TableCellEvent{Row, Col, Button, Mods, Handled}` - the data of `OnCellDoubleClick`.
- `EventTableCellMouseDblClick{Table *Table, Row, Col int, Processed bool}` - deprecated, the event of `SetOnCellMouseDblClick`.

## Internal / framework-invoked

- `ProcessKeyDown(key ui.Key, mods ui.KeyModifiers) bool` - part of the `Widgeter` interface; the form dispatches key events to the focused widget through this. Not normally called directly.
- `TableSetLayoutXml(n *uiNode)` - used by the XML form loader to populate columns/rows from markup; `uiNode` is unexported, so this isn't callable from outside the `ui` package.
