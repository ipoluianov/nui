# Audit: item views and models (Table, ListView, TreeView, ComboBox, PropertyGrid, LogView)

Every finding was reproduced by a package-internal test. The reproducer files are in
`doc/audit/_repro/views/`
(`zz_audit_views_test.go`, `zz_audit_views2_test.go`, `zz_audit_views3_test.go`). Copy them into `ui/` and run
`go test ./ui -run TestAudit -count=1 -v`. They use the existing `newTestForm` helper (ui/controls_test.go) and
this paint helper:

```go
func auditPaint(form *Form) {
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	form.paintScaled(img, 1)
}
```

These were checked and found to work: scroll offsets are clamped after a model shrinks (ListView, Table, TreeView), negative or zero column widths do not panic, ListView icons at row heights 1..5 do not panic, a zero-height cell image does not panic, and concurrent `LogView.Write` passes under `-race`.

---

## 1. Table paint panics when the image height is negative (small row height / large padding / negative image width)

- **Severity:** high (panic in paint, so the app crashes)
- **Where:** ui/table.go:1736-1749 (cells), ui/table.go:1888-1901 (header images), ui/table.go:2497 (model icon width)
- **Root cause:** `imgHeight := c.rowHeight1 - c.cellPadding*2` and `imgWidth` are not clamped. They are passed as
  `uint(imgHeight)` / `uint(imgWidth)` to `resize.Resize`. A negative value wraps to about 2^64, and
  `image.NewRGBA` panics. `modelCell` also sets `cell.imageWidth = c.rowHeight1 - c.cellPadding*2`, which goes
  negative in the same case. Each of these triggers it: `SetRowHeight(h)` with `h < 2*padding` (default padding is 3,
  so h <= 5), `SetCellPadding(p)` with `2p > rowHeight`, `SetCellImage(r, c, img, -1)`, or `SetColumnImage(col, img, -1)`.
- **Reproducer:**
  ```go
  type auditIconModel struct{ ModelSignal; rows int; icon image.Image }
  func (m *auditIconModel) RowCount() int { return m.rows }
  func (m *auditIconModel) ColumnCount() int { return 2 }
  func (m *auditIconModel) Text(row, col int) string { return "x" }
  func (m *auditIconModel) Style(row, col int) ItemStyle { return ItemStyle{Icon: m.icon} }

  func TestAuditTraceIcon(t *testing.T) {
  	table := NewTable()
  	form := newTestForm(t, table)
  	table.SetModel(&auditIconModel{rows: 3, icon: image.NewRGBA(image.Rect(0, 0, 16, 16))})
  	table.SetRowHeight(4)
  	auditPaint(form)
  }
  // also: table.SetColumnCount(1); table.SetRowCount(1);
  //       table.SetCellImage(0, 0, image.NewRGBA(image.Rect(0,0,10,10)), -1); auditPaint(form)
  ```
- **Actual:**
  ```
  panic: image: NewRGBA Rectangle has huge or negative dimensions
  github.com/nfnt/resize.Resize(0x0, 0xfffffffffffffffe, ...)  resize.go:114
  github.com/ipoluianov/nui/ui.(*Table).draw(...)                ui/table.go:1749
  ```
- **Expected:** no panic. The image is skipped, or drawn at size <= 0 as nothing.
- **Fix direction:** in `draw` and `drawPost`, skip the image when `imgWidth <= 0 || imgHeight <= 0` (or clamp both with
  `max(0, …)`). Also guard `b.Dy() == 0` before computing the aspect ratio. Clamp the width in `SetCellImage`/`SetColumnImage` and in
  `modelCell`. Optionally cache the resized image instead of calling `resize.Resize` for every visible cell on every paint.

---

## 2. TreeView: lazy loading in OnExpanding/SetOnExpand that selects a child recurses forever (stack overflow)

- **Severity:** high (the stack overflow is fatal and cannot be recovered; the common lazy-load pattern triggers it)
- **Where:** ui/treeview.go:1783 (`TreeNode.Expand`), via `SetCurrentNode` -> `ScrollToNode` -> `expandParents` (1842)
- **Root cause:** `Expand` emits `OnExpanding` and calls `onExpand` *before* it sets `n.expanded = true`. A handler that
  loads children and then calls `tree.SetCurrentNode(child)`, `tree.ScrollToNode(child)`, `SetSelectedNodes`,
  `child.expandParents()` or `n.Expand()` re-enters `n.Expand()`. The node is still not expanded, so the handler runs again,
  adds children again, and so on with no end.
- **Reproducer:**
  ```go
  func TestAuditTreeExpandingSelectsChild(t *testing.T) {
  	tree := NewTreeView()
  	newTestForm(t, tree)
  	folder := tree.AddNode(nil, "folder")
  	folder.SetHasChildren(true)
  	calls := 0
  	tree.OnExpanding.Connect(func(n *TreeNode) {
  		calls++
  		if calls > 50 { return } // stop the recursion for the test
  		child := n.AddChild("file")
  		tree.SetCurrentNode(child)
  	})
  	folder.Expand()
  	if calls != 1 { t.Errorf("OnExpanding called %d times for one Expand", calls) }
  	if n := folder.ChildCount(); n != 1 { t.Errorf("children loaded %d times", n) }
  }
  ```
- **Actual:** `OnExpanding called 51 times for one Expand`, `children loaded 50 times`. Without the guard the
  goroutine stack overflows.
- **Expected:** the handler runs once, and the nested `Expand` is a no-op.
- **Fix direction:** add an `expanding` flag on the node (return early while it is set), or set
  `n.expanded = true` before emitting and roll it back if needed. Apply the same fix in `expandParents`.

---

## 3. TreeView: re-entrant handlers during `TreeNode.Remove`: slice panic, or a removed node becomes current

- **Severity:** medium (a panic from code that looks correct in a signal handler)
- **Where:** ui/treeview.go:1660-1676 (`Remove`), 994 (`nodeRemoved`)
- **Root cause:** `Remove` unlinks the node from `parent.children` / `tree.nodes`, then calls `nodeRemoved`, which
  emits `OnSelectionChanged`. Only after that does it call `n.detach()`, which sets `tree = nil`. During the handler the removed node
  still reports `Tree() != nil`:
  - (a) `node.Remove()` called again from the handler: `Index()` returns -1, so `children[:index]` gives
    `slice bounds out of range [:-1]`.
  - (b) `tree.SetCurrentNode(removedNode)` passes the `node.tree == c` check. After `detach`, `CurrentNode()` is a node
    that is no longer in the tree.
- **Reproducer:**
  ```go
  func TestAuditTreeReentrantRemoveSame(t *testing.T) {
  	tree := NewTreeView(); newTestForm(t, tree)
  	a := tree.AddNode(nil, "a"); tree.AddNode(nil, "b")
  	tree.SetCurrentNode(a)
  	tree.OnSelectionChanged.Connect(func(n *TreeNode) {
  		if a.Tree() != nil { a.Remove() } // "remove a if still in the tree"
  	})
  	a.Remove()
  }
  func TestAuditTreeReentrantRemove(t *testing.T) {
  	tree := NewTreeView(); newTestForm(t, tree)
  	a := tree.AddNode(nil, "a"); tree.AddNode(nil, "b")
  	tree.SetCurrentNode(a)
  	once := false
  	tree.OnSelectionChanged.Connect(func(n *TreeNode) {
  		if !once { once = true; tree.SetCurrentNode(a) } // restore a remembered selection
  	})
  	a.Remove()
  	if cur := tree.CurrentNode(); cur != nil && cur.Tree() == nil { t.Errorf("current is a removed node") }
  }
  ```
- **Actual:** (a) `panic: runtime error: slice bounds out of range [:-1]` at treeview.go:1668. (b) `the current node is
  a removed node "a" (Tree()==nil)`.
- **Expected:** no panic. A removed node is never current or selected.
- **Fix direction:** call `n.detach()` (and `structureChanged`) before `nodeRemoved` emits, and fire the signal
  last. Make `Remove` return early when `Index() < 0`.

---

## 4. ListView: the Shift anchor is not clamped when the model shrinks, so rows past the end get selected

- **Severity:** medium (`SelectedRows()` returns indices >= RowCount(), and callers index their data with them)
- **Where:** ui/listview.go:105 (`Refresh`), 260 (`selectRow`, extend branch)
- **Root cause:** `Refresh` drops selected rows >= n and clamps `current`, but leaves `anchor` alone. The next Shift+key or
  Shift+click selects the range from the stale anchor, which includes rows that no longer exist.
- **Reproducer:**
  ```go
  func TestAuditListViewStaleAnchor(t *testing.T) {
  	list := NewListView(); newTestForm(t, list)
  	list.SetMultiselect(true)
  	model := NewListModel("a","b","c","d","e","f","g","h","i","j")
  	list.SetModel(model)
  	list.SetCurrentRow(9)
  	model.SetItems([]string{"a", "b", "c"})
  	list.ProcessKeyDown(KeyArrowUp, KeyModifiers{Shift: true})
  	// SelectedRows() must all be < RowCount()
  }
  ```
- **Actual:** `selected rows [1 2 3 4 5 6 7 8 9] past the end (3 rows)`
- **Expected:** `[1 2]`, or the anchor reset to the clamped current row.
- **Fix direction:** in `Refresh`, set `anchor` to -1 or `min(anchor, n-1)` when `anchor >= n`. Also clamp the range in
  `selectRow` to `[0, n-1]`.

---

## 5. Table: the current cell is not clamped when rows or columns shrink. No signal is emitted, the stale cell comes back, and keyboard navigation gets stuck

- **Severity:** medium
- **Where:** ui/table.go:583 (`SetRowCount`), `SetColumnCount`, 2456-2475 (`RefreshModel`, line 2472), 1426 (ArrowLeft),
  1485 (PageUp), 813-827 (`SetCurrentCell2` rejects out-of-range cells)
- **Root cause:** `SetRowCount`/`SetColumnCount` never adjust `currentCellY/X`, `selectionAnchor*` or the selection
  sets. `RefreshModel` tries `SetCurrentCell2(max(0, rows-1), max(0, currentCellX))`, but with `rows == 0`, or with the
  column also out of range, `SetCurrentCell2` rejects the cell, so nothing changes:
  - `CurrentRow()` goes from 5 to -1 and `OnSelectionChanged` is not emitted.
  - When the model grows again, the old current row and selection come back on their own.
  - After `SetRowCount(10)` with current row 90, PageUp computes target 80, `navigateTo` rejects it, and the key does nothing.
  - After the model drops from 10 to 2 columns with current column 8, ArrowLeft goes to `navigateTo(row, 7)`, which is rejected,
    so ArrowLeft and ArrowRight never work again until the user clicks.
- **Reproducer:**
  ```go
  type auditRowsModel struct{ ModelSignal; rows, cols int }
  func (m *auditRowsModel) RowCount() int { return m.rows }
  func (m *auditRowsModel) ColumnCount() int { return m.cols }
  func (m *auditRowsModel) Text(row, col int) string { return "x" }

  func TestAuditTableModelShrinkToZeroSignal(t *testing.T) {
  	table := NewTable(); newTestForm(t, table)
  	m := &auditRowsModel{rows: 10, cols: 2}
  	table.SetModel(m); table.SetCurrentCell2(5, 0)
  	var got []TableCell
  	table.OnSelectionChanged.Connect(func(c TableCell) { got = append(got, c) })
  	m.rows = 0; m.Changed()   // CurrentRow -1, got == [] (no signal)
  	m.rows = 10; m.Changed()  // CurrentRow == 5 again, SelectedRows == [5]
  }
  func TestAuditTableShrinkPageUpStuck(t *testing.T) {
  	table := NewTable(); newTestForm(t, table)
  	table.SetColumnCount(1); table.SetRowCount(100)
  	table.SetCurrentCell2(90, 0); table.SetRowCount(10)
  	table.ProcessKeyDown(KeyPageUp, KeyModifiers{}) // CurrentRow stays -1
  }
  func TestAuditTableColumnShrinkNavigation(t *testing.T) {
  	table := NewTable(); newTestForm(t, table)
  	m := &auditRowsModel{rows: 5, cols: 10}
  	table.SetModel(m); table.SetCurrentCell2(2, 8)
  	m.cols = 2; m.Changed()
  	for i := 0; i < 3; i++ { table.ProcessKeyDown(KeyArrowLeft, KeyModifiers{}) }
  	// CurrentColumn() == -1, currentCellX == 8
  }
  ```
- **Actual:**
  ```
  after shrink: CurrentRow=-1 currentCellY=5 signals=[]
  after regrow: CurrentRow=5 selected=[5]
  PageUp after SetRowCount(10) with current row 90: CurrentRow=-1 currentCellY=90
  after 3x Left: CurrentColumn=-1 currentCellX=8
  ```
- **Expected:** after a shrink, the current cell is clamped to the last row and column (or cleared when the table is empty),
  the selection sets and the anchor are pruned, and `OnSelectionChanged` is emitted once. Keyboard navigation starts from the clamped cell.
- **Fix direction:** add one `clampSelection()` and call it from `SetRowCount`, `SetColumnCount` and `RefreshModel`. It
  clamps `currentCellX/Y` and the anchor, deletes rows/cells >= count from `selectedRows`/`selectedCells`, and emits
  `selectionChanged` if anything changed. With 0 rows, set `currentCellY = -1`. In `ProcessKeyDown`, clamp the source
  row and column into range before computing the target.

---

## 6. Table: the cell editor stays open over the header after its row or column is removed

- **Severity:** low-medium (a visible glitch, and the typed text is then silently dropped)
- **Where:** ui/table.go:452 (`rowOffset` returns 0 for out-of-range rows), 522 (`updateInnerWidgetsLayout`), 2456
  (`RefreshModel` does not close the editor)
- **Root cause:** when the model shrinks below the edited row, the editor's `innerWidget` keeps `posCellRow = 5`.
  `rowOffset(5)` now returns 0, so the editor is laid out at y=0, on top of the header. The commit then checks
  `row < c.rowCount` and drops the text. `ClearRows` closes the editor, but `RefreshModel`/`SetRowCount` do not.
- **Reproducer:**
  ```go
  func TestAuditTableEditorRowRemoved(t *testing.T) {
  	table := NewTable(); newTestForm(t, table)
  	m := &auditRowsModel{rows: 10, cols: 2}
  	table.SetModel(m); table.SetCurrentCell2(5, 1)
  	table.EditCurrentCell("")
  	ed := table.editorTextBox
  	m.rows = 3; m.Changed()
  	// ed.Y() == 0 (< headerHeight), editor still open
  }
  ```
- **Actual:** `editor y before 156 after 0, still open true`
- **Expected:** the editor is cancelled (as in `ClearRows`) when its cell leaves the table.
- **Fix direction:** in `RefreshModel`/`SetRowCount`/`SetColumnCount`, cancel `editorTextBox` and `cellEditor` when their
  cell is out of range. Remove any `innerWidgets` whose cells fall outside the table, or hide them.

---

## 7. Table: `SetCurrentCell2` and other calls create cell objects for out-of-range or arbitrary cells (memory leak)

- **Severity:** low-medium (unbounded growth of `c.rows`; with a model, the table was supposed to keep no cells)
- **Where:** ui/table.go:822 (`SetCurrentCell2` calls `getCellObj` *before* the bounds check), 872 (`navigateTo`),
  985 (`handleSelectionMouseDown`), 1576 (hot tracking in `onMouseMove`), 2408 (`editKey` with `currentCellY == -1`
  after `ClearSelection`)
- **Root cause:** `getCellObj` inserts a `tableRow`/`tableCell` into the maps when none exists. It is used for a read-only check of
  `selectionDisabled`. Every call with a new (row, col) allocates for good, including (-1, x), rows past
  `rowCount`, and every hovered cell of a model-backed table with `SetHotTracking(true)`.
- **Reproducer:**
  ```go
  func TestAuditTableSetCurrentCellLeak(t *testing.T) {
  	table := NewTable(); newTestForm(t, table)
  	table.SetColumnCount(2); table.SetRowCount(2)
  	for i := 0; i < 1000; i++ {
  		table.SetCurrentCell2(100+i, 100+i)
  		table.SetCurrentCell2(-1-i, 0)
  	}
  	// len(table.rows) should be <= 2
  }
  ```
- **Actual:** `out-of-range SetCurrentCell2 made 2000 rows of cells`
- **Expected:** no allocation for reads. Out-of-range calls are no-ops.
- **Fix direction:** add a non-allocating `lookupCell(row, col) *tableCell` (nil when missing) and use it everywhere only
  `selectionDisabled`/edit triggers are read. In `SetCurrentCell2`, do the bounds check first.

---

## 8. ComboBox: a pick from an open popup after the items changed reports an index that does not exist

- **Severity:** medium (handlers get `OnActivated(i)` / `SetOnSelectedIndexChanged` for an index past the end; `ItemData(i)`
  is nil, and code that indexes its own slice panics)
- **Where:** ui/combobox.go:195 (`selectByUser`), 454 / 586 (the popup items call `OnClick(item.index)`)
- **Root cause:** the popup copies the items when it opens. If the combo box is cleared or shrunk while the popup is open (a timer or
  an `Invoke` reloading the list), picking an item calls `selectByUser(staleIndex)`.
  `SetSelectedIndex` ignores the index, but `selectByUser` still computes `changed := index != selectedIndex`, calls
  `onSelectedIndexChanged`, and emits `OnActivated(index)` without validating the index.
- **Reproducer:**
  ```go
  func TestAuditComboStalePopupPick(t *testing.T) {
  	cb := NewComboBox(); newTestForm(t, cb)
  	cb.AddItem("a", nil); cb.AddItem("b", nil); cb.AddItem("c", nil)
  	var activated []int; userChanged := 0
  	cb.OnActivated.Connect(func(i int) { activated = append(activated, i) })
  	cb.SetOnSelectedIndexChanged(func() { userChanged++ })
  	cb.OpenPopup()
  	pw := cb.form.Panel().PopupWidgets
  	popup := pw[len(pw)-1].(*comboBoxPopup)
  	cb.Clear()
  	popup.processKeys(KeyEnd); popup.processKeys(KeyEnter)
  }
  ```
- **Actual:** `OnActivated(2) for a combo box of 0 items; SelectedIndexChanged called 1`
- **Expected:** nothing is emitted (or the popup closes when the items change).
- **Fix direction:** in `selectByUser`, return early when `index < 0 || index >= len(c.items)`. Optionally close an open
  popup in `Clear`/`RemoveItem`/`SetItems`/`InsertItem`.

---

## 9. PropertyGrid: a float32 field is given ±Inf (or a value out of range) from the editor

- **Severity:** low-medium (silent data corruption; integer fields are already clamped to their type)
- **Where:** ui/propertygrid.go:520-525 (`addField` float case), 230-235 (`AddFloat` limits ±MaxFloat64), 573-575
  (`writeField` calls `fv.SetFloat(f)`)
- **Root cause:** the editor's range is ±MaxFloat64 for both float32 and float64 fields. `reflect.Value.SetFloat` on a
  float32 field converts a value above 3.4e38 to +Inf. Unlike the int/uint branches, the float case gets no
  type-based limit.
- **Reproducer:**
  ```go
  type auditF32 struct{ F float32 }
  func TestAuditPropertyGridFloat32Overflow(t *testing.T) {
  	grid := NewPropertyGrid(); newTestForm(t, grid)
  	obj := &auditF32{F: 1}
  	grid.SetObject(obj)
  	box := grid.Property("F").Editor().(*NumBox)
  	box.SetValue(1e300)
  	grid.Property("F").NotifyChanged()
  	// obj.F should be finite
  }
  ```
- **Actual:** `float32 field got +Inf from 1e300 (editor shows 1e+300)`
- **Expected:** the editor is limited to ±math.MaxFloat32 for float32 fields, so the field stays finite.
- **Fix direction:** in `addField`, for `reflect.Float32` use `lo/hi = max/min(tag, ∓math.MaxFloat32)`. In `writeField`,
  clamp to the field's range (`fv.OverflowFloat(f)`).

---

## 10. Quadratic or worse cost in Table and TreeView with many columns, selected nodes, or large selections

- **Severity:** medium (multi-second UI freezes from realistic sizes; the docs promise that "a model of a million rows is as fast as
  one of ten")
- **Where / measured (Linux amd64, the reproducers above):**
  - **Table with many columns.** ui/table.go:2003 `columnOffset(col)` is O(col) (a loop over the map). `draw` calls it for *every*
    column of every visible row (not only the visible columns), `cellByPosition` (2073) calls it in a loop, and
    `headerCellShadowed2` (483) is O(headerRows·cols) per header cell. One paint of 20 rows: 250 cols in 13 ms, 500 in 55 ms,
    1000 in 189 ms (quadratic). One `cellByPosition` (on every mouse move): 1000 cols in 13.6 ms.
  - **TreeView.RemoveChildren / Remove with a large selection.** ui/treeview.go:994-1003 `nodeRemoved` scans the whole
    `selected` map and calls `isDescendantOf` for every removal. `RemoveChildren` with all children selected: 2000 children in 35 ms,
    8000 children in 632 ms (quadratic; 32k is about 10 s).
  - **Table Ctrl+A in cell mode on a model.** ui/table.go:1052 `SelectAll` materialises rows×cols map entries. 1M×5 model:
    1.79 s and 5,000,000 map entries (hundreds of MB). Shift+End (`applyRangeSelection`) does the same, and so does ListView
    `SelectAll` (1 map entry per row).
  - **Header span is not clamped.** `SetHeaderCellSpan2(0, 0, 1, 50_000_000)` on a 2-column table makes one paint take 1.58 s
    (the loop `for i < cellSpanX` at table.go:1872). A larger span freezes the UI.
- **Reproducers:** `TestAuditTableManyColumnsPaint`, `TestAuditTreeRemoveChildrenSelected`,
  `TestAuditTableSelectAllBigModel` and `TestAuditTableHeaderSpanHuge` in the reproducer files (they log the timings).
- **Fix direction:** cache prefix sums of the column widths (invalidate them in `SetColumnWidth`/`SetColumnCount`) and binary-search
  them in `cellByPosition`. Draw only the columns that intersect the viewport. Precompute the shadowed header cells once per paint.
  In `nodeRemoved`, walk the removed subtree and delete those nodes from `selected` (O(subtree)), instead of scanning the
  selection. Represent "all selected" or row ranges as ranges, not maps. Clamp header spans to `columnCount-col` /
  `headerRowsCount-row`.
