# Table

Grid/table widget with selection and editing support.

## Create

```go
t := ui.NewTable()
```

`Table` embeds `Widget`, so all generic widget methods (position, size, colors,
focus, etc.) are available too. The methods below are the ones that belong to
`Table` specifically.

## Rows & columns

- `SetRowCount(count int)` / `RowCount() int` - how many rows are shown. The cells of the rows beyond the count are kept and come back when the count grows again.
- `ClearRows()` - removes all the rows (cells, selection, current cell), as in a new table; `RowCount()` becomes 0. To load other data: `ClearRows()`, fill the cells, `SetRowCount(n)`.
- `SetColumnCount(count int)` / `ColumnCount() int`
- `SetColumnName(col int, name string)` / `ColumnName(col int) string` - header text for column `col` (header row 0).
- `SetColumnCellName2(row, col int, name string)` - header text for a specific header row/column, for use with multi-row headers.
- `SetColumnWidth(col int, width int)` / `ColumnWidth(col int) int`
- `SetColumnImage(col int, img image.Image, imgWidth int)` - icon drawn in the column header.
- `SetRowHeight(height int)` / `RowHeight() int` - row height applies to every row.
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
- `SetOnCellChanged(func(row, col int, text string, data interface{}) bool)` - called after an edit commits; return `false` to reject the new text and revert it.

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
- `SetCurrentCell2(row, col int)` - move the active cell/row programmatically (scrolls it into view, resets the selection to just this item, fires `SetOnSelectionChanged`).
- `CurrentRow() int` / `CurrentColumn() int` - the active cell, or `-1` if none.
- `PreviousCurrentCellX() int` / `PreviousCurrentCellY() int` - active cell before the last change, useful inside an `OnSelectionChanged` callback.
- `SetCellSelectionDisabled(row, col int, disabled bool)` - excludes a cell/row from selection (clicks on it are ignored).
- `SetShowSelection(show bool)` / `ShowSelection() bool` - whether the selection highlight is drawn at all.
- `ScrollToCell2(row, col int)` - scroll so the given cell is visible.
- `CopySelectionToClipboard()` - copies the current cell's text to the clipboard.
- `SetOnSelectionChanged(func(row, col int))` - called whenever the active cell/row moves (click, drag, keyboard nav, `Ctrl+A`).

See `examples/contacts` (sorting, filtering, multiselect, context menu) for a full demo.

## Mouse callbacks

- `SetOnCellMouseDown(func(button ui.MouseButton, row, col, x, y int, mods ui.KeyModifiers))`
- `SetOnCellMouseDblClick(func())` - inspect `ui.CurrentEvent().Parameter.(*ui.EventTableCellMouseDblClick)` for `Row`/`Col`/`Table`, and set `.Processed = true` to suppress the default double-click edit trigger.
- `SetOnColumnClick(func(col int))` - fired when a header column is clicked.
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
- `EventTableCellMouseDblClick{Table *Table, Row, Col int, Processed bool}` - the event passed through `SetOnCellMouseDblClick`.

## Internal / framework-invoked

- `ProcessKeyDown(key ui.Key, mods ui.KeyModifiers) bool` - part of the `Widgeter` interface; the form dispatches key events to the focused widget through this. Not normally called directly.
- `TableSetLayoutXml(n *uiNode)` - used by the XML form loader to populate columns/rows from markup; `uiNode` is unexported, so this isn't callable from outside the `ui` package.
