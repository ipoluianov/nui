# TreeView

Hierarchy of nodes shown in rows, with columns - like a file manager's detail view. The
first column holds the tree (indentation, expand arrow, icon, text); the other columns hold
more texts of the node. Selection, keyboard navigation and editing work like in [Table](table.md).

## Create

```go
tree := ui.NewTreeView()
tree.SetColumnCount(2)
tree.SetColumnName(0, "Name")
tree.SetColumnName(1, "Size")
tree.SetColumnWidth(0, 250)

docs := tree.AddNode(nil, "Documents") // nil parent: a top-level node
report := docs.AddChild("report.txt")
report.SetText(1, "12 KB")
report.SetImage(fileIcon)
docs.Expand()
```

## Columns

- `SetColumnCount(n)` / `ColumnCount()` - at least 1.
- `SetColumnName(col, name)` / `SetColumnNameFunc(col, f)` (follows the language) / `ColumnName(col)`
- `SetColumnWidth(col, width)` / `ColumnWidth(col)` - the user resizes columns by dragging the header borders.
- `SetStretchLastColumn(bool)` - the last column fills the remaining width (default `true`).
- `SetColumnReadOnly(col, bool)` - the column can't be edited.
- `SetHeaderVisible(bool)`, `SetOnColumnClick(func(col))`, `SetOnColumnResize(func(col, width))`

## Nodes

`*ui.TreeNode`:

- `AddChild(text)`, `InsertChild(index, text)`, `Remove()`, `RemoveChildren()`
- `Text(col)` / `SetText(col, text)` / `SetTextFunc(col, f)` (follows the language)
- `SetImage(img)` - icon before the text, scaled down to `ui.TreeViewIconSize` (16px).
- `SetColor(color)` - text color; `SetData(any)` / `Data()` - your own value.
- `SetReadOnly(col, bool)` / `IsReadOnly(col)`
- `Expand()`, `Collapse()`, `Toggle()`, `SetExpanded(bool)`, `IsExpanded()`
- `ExpandAll()` - expands the node (loading its children if needed) and the nodes below it that already have children.
- `Parent()`, `Children()`, `ChildCount()`, `Index()`, `Level()`, `Tree()` (nil after removal)
- `SetHasChildren(true)` - shows the expand arrow before the children exist (lazy loading, see below).

On the tree: `AddNode(parent, text)`, `InsertNode(parent, index, text)`, `Nodes()` (top level),
`RemoveNode(node)`, `Clear()`, `ExpandAll()`, `CollapseAll()`, `ScrollToNode(node)`.

### Loading children on demand

```go
folder := tree.AddNode(nil, "C:")
folder.SetHasChildren(true)
tree.SetOnExpand(func(node *ui.TreeNode) {
	if node.ChildCount() == 0 {
		for _, name := range readDir(node) {
			node.AddChild(name).SetHasChildren(true)
		}
	}
})
```

`SetOnExpand` is called before a node expands; `SetOnCollapse` after it collapses.

`TreeView.ExpandAll()` expands only the nodes that already have children and doesn't load any:
for a tree of folders loading everything could take forever. Nodes marked with `SetHasChildren`
but not loaded yet stay collapsed.

Adding, removing, expanding and collapsing nodes is cheap: the shown rows are rebuilt once, before
the next paint, so a tree of 100 000 nodes can be filled with plain `AddChild` calls.

## Selection

- `CurrentNode()` / `SetCurrentNode(node)` - the node with the keyboard focus; setting it selects only it, expands its parents and scrolls it into view.
- `CurrentColumn()` / `SetCurrentColumn(col)` - the column editing starts in (the column clicked last).
- `SetMultiselect(bool)` - Shift+click / Shift+arrows select a range, Ctrl+click and Ctrl+Space toggle a node, Ctrl+arrows move the focus without changing the selection, Ctrl+A selects all shown nodes, dragging selects a range.
- `SelectedNodes()` (top to bottom), `SetSelectedNodes(nodes)`, `IsNodeSelected(node)`, `SelectAll()`
- `SetOnSelectionChanged(func(node))` - the current node or the selection changed.

Collapsing a node moves the focus out of its children to the node; removing the current node moves it
to the next sibling (or the previous one, or the parent). A right click selects the node under the
mouse (keeping a multi-selection it is in), so a context menu (`SetContextMenu`) acts on it.

## Editing

The same triggers as in `Table`:

- `SetEditTriggerDoubleClick(bool)`, `SetEditTriggerEnter(bool)`, `SetEditTriggerF2(bool)`
- `SetEditTriggerKeyDown(bool)` - typing a letter starts editing with that letter.
- `EditCurrentCell(text)` / `EditCell(node, col, text)` - open the editor (`""` starts from the cell's text, all selected).
- `CommitEdit()`, `CancelEdit()`, `IsEditing()`, `IsCellEditable(node, col)`
- `SetOnCellChanged(func(node, col, text) bool)` - called on commit; return `false` to reject the text.

In the editor Enter commits, Escape cancels, a click elsewhere or Tab commits.

## Keyboard & mouse

| Input | Action |
|---|---|
| Up / Down, Home / End, PageUp / PageDown | move (Shift extends the selection) |
| Right | expand; on an expanded node - go to its first child |
| Left | collapse; on a collapsed node - go to its parent |
| Numpad + / - / * | expand / collapse / expand the branch (its loaded part, see `ExpandAll`) |
| Ctrl+Left / Ctrl+Right | previous / next column for editing |
| Enter, double click | edit (if the trigger is on), else expand/collapse and `SetOnNodeActivated` |
| letters | jump to the next node starting with the typed text (unless `SetEditTriggerKeyDown`) |
| Ctrl+C | copy the selected nodes (columns tab-separated) - `CopySelectionToClipboard()` |
| click on the arrow | expand/collapse without selecting |

## Appearance

`SetRowHeight(h)`, `SetIndent(px)` (0 follows the row height), `SetGridLines(bool)`.
A tree without the focus keeps its selection in a softer color.

See `examples/files` (a folder tree loaded on expand).
