package ui

import (
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/nfnt/resize"
)

// TreeView shows a hierarchy of nodes in rows, with columns: the first
// column holds the tree (indentation, expand arrow, icon, text), the other
// columns hold more texts of the node - like a file manager's detail view.
//
//	tree := ui.NewTreeView()
//	tree.SetColumnCount(2)
//	tree.SetColumnName(0, "Name")
//	tree.SetColumnName(1, "Size")
//	docs := tree.AddNode(nil, "Documents")
//	docs.AddChild("report.txt").SetText(1, "12 KB")
//	docs.Expand()
//
// Selection, keyboard navigation and editing work like in Table: see
// SetMultiselect, SetEditTriggerF2 and SetOnCellChanged.
type TreeView struct {
	Widget

	nodes []*TreeNode // top-level nodes

	columnCount        int
	columnsWidths      map[int]int
	columnsNames       map[int]string
	columnsNameFuncs   map[int]func() string
	columnsReadOnly    map[int]bool
	defaultColumnWidth int
	stretchLastColumn  bool
	headerVisible      bool

	rowHeight       int
	customRowHeight bool
	indent          int
	cellPadding     int
	gridLines       bool

	// rows are the nodes shown now, top to bottom: the ones whose parents are
	// all expanded. Rebuilt lazily when rowsValid is false.
	rows      []treeRow
	rowIndex  map[*TreeNode]int
	rowsValid bool

	// Selection
	current       *TreeNode
	currentColumn int
	anchor        *TreeNode
	selected      map[*TreeNode]bool
	multiselect   bool

	selectionDragging bool
	dragBase          map[*TreeNode]bool

	hoverRow int

	// Type-ahead search: the letters typed within treeTypeAheadTimeout
	typeAhead     string
	typeAheadTime time.Time

	columnResizingIndex int
	headerWidget        *tableHeader

	// Editing
	editorTextBox *TextBox
	editNode      *TreeNode
	editColumn    int

	editTriggerDoubleClick bool
	editTriggerEnter       bool
	editTriggerF2          bool
	editTriggerKeyDown     bool

	// Callbacks
	onSelectionChanged func(node *TreeNode)
	onCellChanged      func(node *TreeNode, col int, text string) bool
	onNodeActivated    func(node *TreeNode)
	onExpand           func(node *TreeNode)
	onCollapse         func(node *TreeNode)
	onColumnClick      func(col int)
	onColumnResize     func(col int, newWidth int)
}

type treeRow struct {
	node  *TreeNode
	level int
}

// TreeViewIconSize is the size node icons are scaled down to.
const TreeViewIconSize = 16

const (
	treeTypeAheadTimeout = time.Second
	treeMinColumnWidth   = 30
	treeResizeGrip       = 5
)

func NewTreeView() *TreeView {
	var c TreeView
	c.InitWidget()
	c.SetAbsolutePositioning(true)
	c.SetTypeName("TreeView")
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	c.SetAllowScroll(true, true)
	c.scrollBarInset = 1 // inside the border
	c.SetAutoFillBackground(true)
	c.SetRole("base")
	c.SetCanBeFocused(true)

	c.SetOnPaint(c.draw)
	c.SetOnPostPaint(c.drawPost)
	c.SetOnMouseDown(c.onMouseDown)
	c.SetOnMouseUp(c.onMouseUp)
	c.SetOnMouseDblClick(c.onMouseDblClick)
	c.SetOnMouseMove(c.onMouseMove)
	c.SetOnMouseLeave(func() { c.hoverRow = -1 })
	c.SetOnChar(c.onChar)
	c.SetOnScrollChanged(func(scrollX, scrollY int) { c.layoutChildren() })

	c.columnCount = 1
	c.columnsWidths = make(map[int]int)
	c.columnsNames = make(map[int]string)
	c.columnsReadOnly = make(map[int]bool)
	c.defaultColumnWidth = 200
	c.stretchLastColumn = true
	c.headerVisible = true
	c.rowHeight = ThemeRowHeight()
	c.cellPadding = 3
	c.selected = make(map[*TreeNode]bool)
	c.hoverRow = -1
	c.columnResizingIndex = -1

	c.headerWidget = newTableHeader()
	c.headerWidget.OnHeaderMouseDown = func(button MouseButton, x, y int, mods KeyModifiers) bool {
		return c.onMouseDown(button, x+c.scrollX, y+c.scrollY, mods)
	}
	c.headerWidget.OnHeaderMouseUp = func(button MouseButton, x, y int, mods KeyModifiers) bool {
		return c.onMouseUp(button, x+c.scrollX, y+c.scrollY, mods)
	}
	c.headerWidget.OnHeaderMouseMove = func(x, y int, mods KeyModifiers) MouseCursor {
		return c.onMouseMoveHeader(x+c.scrollX, y+c.scrollY)
	}
	c.AddWidget(0, 0, c.headerWidget)

	c.structureChanged()
	return &c
}

// ---------------------------------------------------------------- Columns

func (c *TreeView) SetColumnCount(count int) {
	if count < 1 {
		count = 1
	}
	c.cancelEdit()
	c.columnCount = count
	if c.currentColumn >= count {
		c.currentColumn = count - 1
	}
	c.updateInnerSize()
}

func (c *TreeView) ColumnCount() int {
	return c.columnCount
}

// SetColumnName sets the header text of the column.
func (c *TreeView) SetColumnName(col int, name string) {
	c.columnsNames[col] = name
	c.form.Update()
}

// SetColumnNameFunc makes the header text come from f, now and on every
// language change.
func (c *TreeView) SetColumnNameFunc(col int, f func() string) {
	if c.columnsNameFuncs == nil {
		c.columnsNameFuncs = make(map[int]func() string)
	}
	c.columnsNameFuncs[col] = f
	c.SetColumnName(col, f())
}

func (c *TreeView) ColumnName(col int) string {
	return c.columnsNames[col]
}

func (c *TreeView) SetColumnWidth(col int, width int) {
	if width < treeMinColumnWidth {
		width = treeMinColumnWidth
	}
	c.columnsWidths[col] = width
	c.updateInnerSize()
	if c.onColumnResize != nil {
		c.onColumnResize(col, width)
	}
}

// ColumnWidth returns the width set for the column; the last column may be
// shown wider, see SetStretchLastColumn.
func (c *TreeView) ColumnWidth(col int) int {
	if w, ok := c.columnsWidths[col]; ok {
		return w
	}
	return c.defaultColumnWidth
}

// SetStretchLastColumn makes the last column fill the width left by the
// other columns (the default).
func (c *TreeView) SetStretchLastColumn(stretch bool) {
	c.stretchLastColumn = stretch
	c.updateInnerSize()
}

// SetColumnReadOnly excludes the column from editing, whatever the edit
// triggers are; see also TreeNode.SetReadOnly.
func (c *TreeView) SetColumnReadOnly(col int, readOnly bool) {
	c.columnsReadOnly[col] = readOnly
}

// SetHeaderVisible shows or hides the row of column names.
func (c *TreeView) SetHeaderVisible(visible bool) {
	c.headerVisible = visible
	c.updateInnerSize()
}

func (c *TreeView) HeaderVisible() bool {
	return c.headerVisible
}

func (c *TreeView) SetOnColumnClick(callback func(col int)) {
	c.onColumnClick = callback
}

func (c *TreeView) SetOnColumnResize(callback func(col int, newWidth int)) {
	c.onColumnResize = callback
}

// ---------------------------------------------------------------- Appearance

func (c *TreeView) SetRowHeight(height int) {
	c.customRowHeight = true
	// At least a pixel: the rows under a point are found dividing by it
	c.rowHeight = max(height, 1)
	c.updateInnerSize()
}

func (c *TreeView) RowHeight() int {
	return c.rowHeight
}

// SetIndent sets how far each level is shifted to the right; 0 (the
// default) follows the row height.
func (c *TreeView) SetIndent(indent int) {
	c.indent = indent
	c.updateInnerSize()
}

func (c *TreeView) Indent() int {
	if c.indent > 0 {
		return c.indent
	}
	return max(c.rowHeight-4, 12)
}

// SetGridLines draws lines between the rows and columns.
func (c *TreeView) SetGridLines(gridLines bool) {
	c.gridLines = gridLines
	c.form.Update()
}

// ---------------------------------------------------------------- Nodes

// AddNode adds a node with the text in the first column at the end of
// parent's children, or of the top-level nodes when parent is nil.
func (c *TreeView) AddNode(parent *TreeNode, text string) *TreeNode {
	if parent != nil {
		return parent.AddChild(text)
	}
	return c.InsertNode(nil, len(c.nodes), text)
}

// InsertNode adds a node at the index among parent's children (the
// top-level nodes when parent is nil).
func (c *TreeView) InsertNode(parent *TreeNode, index int, text string) *TreeNode {
	if parent != nil {
		return parent.InsertChild(index, text)
	}
	node := newTreeNode(c, nil, text)
	index = max(0, min(index, len(c.nodes)))
	c.nodes = append(c.nodes[:index], append([]*TreeNode{node}, c.nodes[index:]...)...)
	c.structureChanged()
	return node
}

// Nodes returns the top-level nodes.
func (c *TreeView) Nodes() []*TreeNode {
	return c.nodes
}

// RemoveNode removes the node with all its children.
func (c *TreeView) RemoveNode(node *TreeNode) {
	if node != nil && node.tree == c {
		node.Remove()
	}
}

// Clear removes all the nodes.
func (c *TreeView) Clear() {
	c.cancelEdit()
	for _, n := range c.nodes {
		n.detach()
	}
	c.nodes = nil
	c.current, c.anchor = nil, nil
	c.selected = make(map[*TreeNode]bool)
	c.structureChanged()
	c.fireSelectionChanged()
}

// ExpandAll expands every node that has children. Nodes whose children
// aren't loaded yet (SetHasChildren) stay collapsed: loading them could
// never end, e.g. for a tree of folders.
func (c *TreeView) ExpandAll() {
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()
	for _, n := range c.nodes {
		n.expandLoaded()
	}
}

// CollapseAll collapses every node.
func (c *TreeView) CollapseAll() {
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()
	var collapse func(nodes []*TreeNode)
	collapse = func(nodes []*TreeNode) {
		for _, n := range nodes {
			collapse(n.children)
			n.Collapse()
		}
	}
	collapse(c.nodes)
}

// SetOnExpand sets the function called before a node expands. Children
// can be added there, e.g. to load a folder only when it's opened; mark
// such nodes with SetHasChildren so they show the expand arrow.
func (c *TreeView) SetOnExpand(f func(node *TreeNode)) {
	c.onExpand = f
}

// SetOnCollapse sets the function called after a node collapses.
func (c *TreeView) SetOnCollapse(f func(node *TreeNode)) {
	c.onCollapse = f
}

// SetOnNodeActivated sets the function called on a double click on a node
// or Enter, when they don't start editing.
func (c *TreeView) SetOnNodeActivated(f func(node *TreeNode)) {
	c.onNodeActivated = f
}

// ScrollToNode expands the node's parents and scrolls the node into view.
func (c *TreeView) ScrollToNode(node *TreeNode) {
	if node == nil || node.tree != c {
		return
	}
	node.expandParents()
	row := c.rowOf(node)
	if row < 0 {
		return
	}
	top := c.headerHeight() + row*c.rowHeight
	viewHeight := c.ViewportHeight()
	if top < c.scrollY+c.headerHeight() {
		c.setScrollY(top - c.headerHeight())
	} else if top+c.rowHeight > c.scrollY+viewHeight {
		c.setScrollY(top + c.rowHeight - viewHeight)
	}
	c.checkScrolls()
	c.form.Update()
}

// ---------------------------------------------------------------- Selection

// CurrentNode returns the node with the keyboard focus, nil if none.
func (c *TreeView) CurrentNode() *TreeNode {
	return c.current
}

// CurrentColumn returns the column that editing starts in: the one clicked
// last, or moved to with Ctrl+Left/Right.
func (c *TreeView) CurrentColumn() int {
	return c.currentColumn
}

// SetCurrentNode makes the node current and the only selected one,
// expanding its parents and scrolling it into view. nil clears the selection.
func (c *TreeView) SetCurrentNode(node *TreeNode) {
	if node != nil && node.tree != c {
		return
	}
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()
	c.current = node
	c.anchor = node
	c.selected = make(map[*TreeNode]bool)
	if node != nil {
		c.selected[node] = true
		c.ScrollToNode(node)
	}
	c.form.Update()
	c.fireSelectionChanged()
}

// SetCurrentColumn sets the column editing starts in.
func (c *TreeView) SetCurrentColumn(col int) {
	if col >= 0 && col < c.columnCount {
		c.currentColumn = col
		c.form.Update()
	}
}

// SetMultiselect allows selecting several nodes: Shift+click and
// Shift+arrows select a range, Ctrl+click and Ctrl+Space toggle a node,
// Ctrl+A selects all shown nodes. Turning it off keeps only the current node.
func (c *TreeView) SetMultiselect(enabled bool) {
	if c.multiselect == enabled {
		return
	}
	c.multiselect = enabled
	if !enabled {
		c.selectionDragging = false
		c.selectOnly(c.current)
		c.form.Update()
	}
}

func (c *TreeView) Multiselect() bool {
	return c.multiselect
}

// SelectedNodes returns the selected nodes, top to bottom.
func (c *TreeView) SelectedNodes() []*TreeNode {
	result := make([]*TreeNode, 0, len(c.selected))
	c.walk(func(n *TreeNode) {
		if c.selected[n] {
			result = append(result, n)
		}
	})
	return result
}

// SetSelectedNodes selects the nodes (all of them with multiselect, else
// the first); the first one becomes current.
func (c *TreeView) SetSelectedNodes(nodes []*TreeNode) {
	valid := make([]*TreeNode, 0, len(nodes))
	for _, n := range nodes {
		if n != nil && n.tree == c {
			valid = append(valid, n)
		}
	}
	if len(valid) == 0 {
		c.SetCurrentNode(nil)
		return
	}
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()
	c.SetCurrentNode(valid[0])
	if !c.multiselect {
		return
	}
	for _, n := range valid[1:] {
		n.expandParents()
		c.selected[n] = true
	}
	c.fireSelectionChanged()
}

func (c *TreeView) IsNodeSelected(node *TreeNode) bool {
	return c.selected[node]
}

// SelectAll selects all the shown nodes; only with multiselect.
func (c *TreeView) SelectAll() {
	if !c.multiselect {
		return
	}
	c.ensureRows()
	for _, r := range c.rows {
		c.selected[r.node] = true
	}
	if c.current == nil && len(c.rows) > 0 {
		c.current = c.rows[0].node
		c.anchor = c.current
	}
	c.form.Update()
	c.fireSelectionChanged()
}

// SetOnSelectionChanged sets the function called when the current node or
// the set of selected nodes changes.
func (c *TreeView) SetOnSelectionChanged(f func(node *TreeNode)) {
	c.onSelectionChanged = f
}

func (c *TreeView) fireSelectionChanged() {
	if c.onSelectionChanged != nil {
		c.onSelectionChanged(c.current)
	}
}

func (c *TreeView) selectOnly(node *TreeNode) {
	c.selected = make(map[*TreeNode]bool)
	if node != nil {
		c.selected[node] = true
	}
}

// selectRange selects base plus the shown nodes between the anchor and node.
func (c *TreeView) selectRange(node *TreeNode, base map[*TreeNode]bool) {
	c.ensureRows()
	from := c.rowOf(c.anchor)
	to := c.rowOf(node)
	if from < 0 {
		from = to
	}
	if from > to {
		from, to = to, from
	}
	selected := make(map[*TreeNode]bool, len(base)+to-from+1)
	for n := range base {
		selected[n] = true
	}
	for i := from; i <= to && i >= 0; i++ {
		selected[c.rows[i].node] = true
	}
	c.selected = selected
}

// moveTo makes the node current for keyboard navigation and mouse drags:
// extend selects the range from the anchor (with multiselect), keepSelection
// only moves the focus (Ctrl+arrows).
func (c *TreeView) moveTo(node *TreeNode, extend bool, keepSelection bool) {
	if node == nil {
		return
	}
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()
	switch {
	case extend && c.multiselect:
		c.current = node
		c.selectRange(node, nil)
	case keepSelection && c.multiselect:
		c.current = node
	default:
		c.current = node
		c.anchor = node
		c.selectOnly(node)
	}
	c.ScrollToNode(node)
	c.form.Update()
	c.fireSelectionChanged()
}

// ---------------------------------------------------------------- Editing

func (c *TreeView) SetEditTriggerDoubleClick(enabled bool) { c.editTriggerDoubleClick = enabled }
func (c *TreeView) SetEditTriggerEnter(enabled bool)       { c.editTriggerEnter = enabled }
func (c *TreeView) SetEditTriggerF2(enabled bool)          { c.editTriggerF2 = enabled }

// SetEditTriggerKeyDown starts editing when a letter is typed, replacing
// the text; without it typing jumps to the next node starting with the
// typed letters.
func (c *TreeView) SetEditTriggerKeyDown(enabled bool) { c.editTriggerKeyDown = enabled }

// SetOnCellChanged sets the function called when an edit is committed,
// with the new text; returning false rejects it.
func (c *TreeView) SetOnCellChanged(f func(node *TreeNode, col int, text string) bool) {
	c.onCellChanged = f
}

// IsCellEditable reports whether the cell can be edited: its column and
// node aren't read-only.
func (c *TreeView) IsCellEditable(node *TreeNode, col int) bool {
	return node != nil && node.tree == c && col >= 0 && col < c.columnCount &&
		!c.columnsReadOnly[col] && !node.IsReadOnly(col)
}

// IsEditing reports whether the inline editor is open.
func (c *TreeView) IsEditing() bool {
	return c.editorTextBox != nil
}

// EditCurrentCell opens the editor on the current node's current column.
// With enteredText empty the editor starts from the cell's text, all
// selected; otherwise it starts from enteredText.
func (c *TreeView) EditCurrentCell(enteredText string) {
	c.EditCell(c.current, c.currentColumn, enteredText)
}

// EditCell opens the editor on the cell, see EditCurrentCell. Does nothing
// for a cell that isn't editable (IsCellEditable).
func (c *TreeView) EditCell(node *TreeNode, col int, enteredText string) {
	if !c.IsCellEditable(node, col) {
		return
	}
	c.cancelEdit()
	c.ScrollToNode(node)
	c.ensureColumnVisible(col)

	editor := NewTextBox()
	c.editorTextBox, c.editNode, c.editColumn = editor, node, col
	if enteredText == "" {
		editor.SetText(node.Text(col))
		editor.MoveCursorToEnd()
		editor.SelectAllText()
	} else {
		editor.SetText(enteredText)
		editor.MoveCursorToEnd()
	}
	editor.SetOnTextBoxKeyDown(func() {
		ev := CurrentEvent().Parameter.(*EventTextboxKeyDown)
		switch ev.Key {
		case KeyEnter:
			c.finishEdit(true, true)
			ev.Processed = true
		case KeyEsc:
			c.finishEdit(false, true)
			ev.Processed = true
		}
	})
	// A click elsewhere commits; the focus stays where it went
	editor.SetOnFocusLost(func() { c.finishEdit(true, false) })
	c.AddWidget(0, 0, editor)
	c.layoutChildren()
	editor.Focus()
}

// CommitEdit applies the editor's text (unless SetOnCellChanged rejects
// it) and closes the editor.
func (c *TreeView) CommitEdit() {
	c.finishEdit(true, true)
}

// CancelEdit closes the editor keeping the cell's text.
func (c *TreeView) CancelEdit() {
	c.finishEdit(false, true)
}

func (c *TreeView) cancelEdit() {
	c.finishEdit(false, true)
}

// finishEdit closes the editor, applying its text if commit; refocus gives
// the focus back to the tree if the editor had it.
func (c *TreeView) finishEdit(commit bool, refocus bool) {
	editor := c.editorTextBox
	if editor == nil {
		return
	}
	node, col, text := c.editNode, c.editColumn, editor.Text()
	hadFocus := c.form != nil && c.form.FocusedWidget() == Widgeter(editor)
	c.editorTextBox, c.editNode = nil, nil
	editor.SetOnFocusLost(nil)
	c.RemoveWidget(editor)
	if refocus && hadFocus {
		c.Focus()
	}
	if commit && node.tree == c && (c.onCellChanged == nil || c.onCellChanged(node, col, text)) {
		node.SetText(col, text)
	}
	c.form.Update()
}

// ---------------------------------------------------------------- Clipboard

// CopySelectionToClipboard copies the selected nodes, one per line, their
// columns separated by tabs (Ctrl+C).
func (c *TreeView) CopySelectionToClipboard() {
	nodes := c.SelectedNodes()
	if len(nodes) == 0 {
		return
	}
	lines := make([]string, 0, len(nodes))
	for _, n := range nodes {
		texts := make([]string, c.columnCount)
		for col := range texts {
			texts[col] = n.Text(col)
		}
		lines = append(lines, strings.TrimRight(strings.Join(texts, "\t"), "\t"))
	}
	ClipboardSetText(strings.Join(lines, "\n"))
}

// ---------------------------------------------------------------- Geometry

func (c *TreeView) headerHeight() int {
	if !c.headerVisible {
		return 0
	}
	return c.rowHeight
}

// columnWidth is the shown width: the last column may stretch to the
// view's edge.
func (c *TreeView) columnWidth(col int) int {
	w := c.ColumnWidth(col)
	if c.stretchLastColumn && col == c.columnCount-1 {
		w = max(w, c.ViewportWidth()-c.columnOffset(col))
	}
	return w
}

func (c *TreeView) columnOffset(col int) int {
	x := 0
	for i := 0; i < col; i++ {
		x += c.ColumnWidth(i)
	}
	return x
}

func (c *TreeView) columnsWidth() int {
	return c.columnOffset(c.columnCount-1) + c.columnWidth(c.columnCount-1)
}

func (c *TreeView) columnAt(x int) int {
	for col := 0; col < c.columnCount; col++ {
		offset := c.columnOffset(col)
		if x >= offset && x < offset+c.columnWidth(col) {
			return col
		}
	}
	return -1
}

// rowAt returns the row at the content y, -1 if none.
func (c *TreeView) rowAt(y int) int {
	c.ensureRows()
	y -= c.headerHeight()
	if y < 0 {
		return -1
	}
	row := y / c.rowHeight
	if row >= len(c.rows) {
		return -1
	}
	return row
}

// rowAtClamped is rowAt for a point that may be above or below the rows,
// e.g. while dragging a selection past the edge.
func (c *TreeView) rowAtClamped(y int) int {
	c.ensureRows()
	if len(c.rows) == 0 {
		return -1
	}
	row := (y - c.headerHeight()) / c.rowHeight
	if y < c.headerHeight() {
		row = 0
	}
	return max(0, min(row, len(c.rows)-1))
}

// arrowRect is the expand arrow's area of a row in the first column.
func (c *TreeView) arrowX(level int) int {
	return c.columnOffset(0) + level*c.Indent()
}

// textX returns where the text of a cell starts, relative to the column.
func (c *TreeView) textX(node *TreeNode, level int, col int) int {
	if col != 0 {
		return c.cellPadding
	}
	x := (level+1)*c.Indent() + c.cellPadding
	if node.image != nil {
		x += TreeViewIconSize + c.cellPadding
	}
	return x
}

func (c *TreeView) visibleRows() (first, last int) {
	c.ensureRows()
	first = max(0, c.scrollY/c.rowHeight-1)
	last = min(len(c.rows), first+c.ViewportHeight()/c.rowHeight+3)
	return
}

func (c *TreeView) updateInnerSize() {
	if c.rowsValid {
		c.applyContentSize()
	} else {
		c.ensureRows() // applies the size too
	}
	c.form.Update()
}

// applyContentSize sets the scrollable size from the columns and the rows.
// The width is of the columns as they are set: the stretch of the last one
// only fills the view, it doesn't make the content wider.
func (c *TreeView) applyContentSize() {
	c.SetInnerSize(c.columnOffset(c.columnCount), c.headerHeight()+len(c.rows)*c.rowHeight)
	c.checkScrolls()
	c.layoutChildren()
}

// SetScrollX, SetScrollY, ProcessMouseWheel and InnerHeight bring the
// rows up to date first: the scrollable size depends on them.

func (c *TreeView) SetScrollX(scrollX int) {
	c.ensureRows()
	c.Widget.SetScrollX(scrollX)
}

func (c *TreeView) SetScrollY(scrollY int) {
	c.ensureRows()
	c.Widget.SetScrollY(scrollY)
}

func (c *TreeView) ProcessMouseWheel(deltaX, deltaY int) bool {
	c.ensureRows()
	return c.Widget.ProcessMouseWheel(deltaX, deltaY)
}

func (c *TreeView) InnerHeight() int {
	c.ensureRows()
	return c.Widget.InnerHeight()
}

// ProcessPaint brings the rows up to date before the content and the
// scroll bars are drawn.
func (c *TreeView) ProcessPaint(cnv *Canvas) {
	c.ensureRows()
	c.Widget.ProcessPaint(cnv)
}

// SetSize also stretches the last column to the new width.
func (c *TreeView) SetSize(w, h int) {
	c.Widget.SetSize(w, h)
	c.updateInnerSize()
}

// layoutChildren keeps the header at the top of the view and the editor
// over its cell.
func (c *TreeView) layoutChildren() {
	c.headerWidget.SetPosition(c.scrollX, c.scrollY)
	c.headerWidget.SetSize(c.ViewportWidth(), c.headerHeight())

	if c.editorTextBox != nil {
		row := c.rowOf(c.editNode)
		if row < 0 {
			return
		}
		x := c.columnOffset(c.editColumn)
		textX := c.textX(c.editNode, c.rows[row].level, c.editColumn) - c.cellPadding
		c.editorTextBox.SetPosition(x+textX, c.headerHeight()+row*c.rowHeight)
		c.editorTextBox.SetSize(max(c.columnWidth(c.editColumn)-textX, treeMinColumnWidth), c.rowHeight)
	}
}

func (c *TreeView) ensureColumnVisible(col int) {
	x := c.columnOffset(col)
	if x < c.scrollX {
		c.setScrollX(x)
	} else if x+c.columnWidth(col) > c.scrollX+c.ViewportWidth() {
		c.setScrollX(min(x, x+c.columnWidth(col)-c.ViewportWidth()))
	}
	c.checkScrolls()
}

// ---------------------------------------------------------------- Rows

// structureChanged is called when nodes are added, removed, expanded or
// collapsed: the shown rows are rebuilt once, on the next use (painting,
// a mouse event...), however many changes come before it.
func (c *TreeView) structureChanged() {
	c.rowsValid = false
	c.form.Update()
}

func (c *TreeView) ensureRows() {
	if c.rowsValid {
		return
	}
	c.rows = c.rows[:0]
	c.rowIndex = make(map[*TreeNode]int, len(c.rows))
	var add func(nodes []*TreeNode, level int)
	add = func(nodes []*TreeNode, level int) {
		for _, n := range nodes {
			c.rowIndex[n] = len(c.rows)
			c.rows = append(c.rows, treeRow{node: n, level: level})
			if n.expanded {
				add(n.children, level+1)
			}
		}
	}
	add(c.nodes, 0)
	c.rowsValid = true
	c.applyContentSize()
}

func (c *TreeView) rowOf(node *TreeNode) int {
	if node == nil {
		return -1
	}
	c.ensureRows()
	if row, ok := c.rowIndex[node]; ok {
		return row
	}
	return -1
}

func (c *TreeView) nodeAtRow(row int) *TreeNode {
	c.ensureRows()
	if row < 0 || row >= len(c.rows) {
		return nil
	}
	return c.rows[row].node
}

// walk calls f for every node, top to bottom, shown or not.
func (c *TreeView) walk(f func(n *TreeNode)) {
	var visit func(nodes []*TreeNode)
	visit = func(nodes []*TreeNode) {
		for _, n := range nodes {
			f(n)
			visit(n.children)
		}
	}
	visit(c.nodes)
}

// nodeCollapsed moves the focus and the selection out of the collapsed
// node's children, which aren't shown anymore.
func (c *TreeView) nodeCollapsed(node *TreeNode) {
	if c.editNode != nil && c.editNode.isDescendantOf(node) {
		c.cancelEdit()
	}
	changed := false
	for n := range c.selected {
		if n.isDescendantOf(node) {
			delete(c.selected, n)
			changed = true
		}
	}
	if c.current != nil && c.current.isDescendantOf(node) {
		c.current = node
		c.selected[node] = true
		changed = true
	}
	if c.anchor != nil && c.anchor.isDescendantOf(node) {
		c.anchor = node
	}
	if changed {
		c.fireSelectionChanged()
	}
}

// nodeRemoved forgets the removed node and its children: the focus goes to
// the next shown node (or the previous one, or the parent).
func (c *TreeView) nodeRemoved(node *TreeNode, parent *TreeNode, index int) {
	if c.editNode != nil && (c.editNode == node || c.editNode.isDescendantOf(node)) {
		c.cancelEdit()
	}
	changed := false
	for n := range c.selected {
		if n == node || n.isDescendantOf(node) {
			delete(c.selected, n)
			changed = true
		}
	}
	if c.anchor == node || (c.anchor != nil && c.anchor.isDescendantOf(node)) {
		c.anchor = nil
	}
	if c.current == node || (c.current != nil && c.current.isDescendantOf(node)) {
		siblings := c.nodes
		if parent != nil {
			siblings = parent.children
		}
		switch {
		case index < len(siblings):
			c.current = siblings[index]
		case index > 0:
			c.current = siblings[index-1]
		default:
			c.current = parent
		}
		c.anchor = c.current
		if c.current != nil {
			c.selected[c.current] = true
		}
		changed = true
	}
	if changed {
		c.fireSelectionChanged()
	}
}

// ---------------------------------------------------------------- Mouse

func (c *TreeView) onMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	// The header isn't focusable: a click on it would take the focus away
	c.Focus()
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	if y < c.scrollY+c.headerHeight() {
		if button != MouseButtonLeft {
			return true
		}
		if border := c.headerBorderAt(x); border >= 0 {
			c.columnResizingIndex = border
			return true
		}
		if col := c.columnAt(x); col >= 0 && c.onColumnClick != nil {
			c.onColumnClick(col)
		}
		return true
	}

	row := c.rowAt(y)
	if row < 0 {
		return true
	}
	r := c.rows[row]
	if col := c.columnAt(x); col >= 0 {
		c.currentColumn = col
	}

	if button == MouseButtonLeft && c.onArrow(r, x) {
		r.node.Toggle()
		return true
	}

	if button == MouseButtonRight {
		// Keep a multi-selection the click is in, so a context menu acts on it
		if !c.selected[r.node] {
			c.moveTo(r.node, false, false)
		}
		return true
	}
	if button != MouseButtonLeft {
		return true
	}

	switch {
	case c.multiselect && mods.Shift:
		base := map[*TreeNode]bool(nil)
		if mods.Ctrl {
			base = c.cloneSelected()
		}
		c.current = r.node
		c.selectRange(r.node, base)
		c.dragBase = base
		c.fireSelectionChanged()
	case c.multiselect && mods.Ctrl:
		if c.selected[r.node] {
			delete(c.selected, r.node)
		} else {
			c.selected[r.node] = true
		}
		c.current, c.anchor = r.node, r.node
		c.dragBase = c.cloneSelected()
		c.fireSelectionChanged()
	default:
		c.moveTo(r.node, false, false)
		c.dragBase = nil
	}
	c.selectionDragging = true
	c.form.Update()
	return true
}

// ProcessMouseDown handles the right click before a context menu opens,
// so the menu acts on the node under the mouse.
func (c *TreeView) ProcessMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if button == MouseButtonRight && y >= c.headerHeight() {
		if row := c.rowAt(y + c.scrollY); row >= 0 && !c.selected[c.rows[row].node] {
			if col := c.columnAt(x + c.scrollX); col >= 0 {
				c.currentColumn = col
			}
			c.moveTo(c.rows[row].node, false, false)
		}
	}
	return c.Widget.ProcessMouseDown(button, x, y, mods)
}

func (c *TreeView) cloneSelected() map[*TreeNode]bool {
	result := make(map[*TreeNode]bool, len(c.selected))
	for n := range c.selected {
		result[n] = true
	}
	return result
}

func (c *TreeView) onArrow(r treeRow, x int) bool {
	if !r.node.HasChildren() {
		return false
	}
	arrowX := c.arrowX(r.level)
	return x >= arrowX && x < arrowX+c.Indent()
}

func (c *TreeView) onMouseUp(button MouseButton, x int, y int, mods KeyModifiers) bool {
	c.columnResizingIndex = -1
	c.selectionDragging = false
	return true
}

func (c *TreeView) onMouseDblClick(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft || y < c.scrollY+c.headerHeight() {
		return true
	}
	row := c.rowAt(y)
	if row < 0 {
		return true
	}
	r := c.rows[row]
	if c.onArrow(r, x) {
		r.node.Toggle() // the second click of a fast double click on the arrow
		return true
	}
	col := c.columnAt(x)
	if col >= 0 {
		c.currentColumn = col
	}
	c.moveTo(r.node, false, false)
	if c.editTriggerDoubleClick && c.IsCellEditable(r.node, c.currentColumn) {
		c.EditCurrentCell("")
		return true
	}
	c.activate(r.node)
	return true
}

// activate: a node with children expands or collapses, and
// SetOnNodeActivated's function is called.
func (c *TreeView) activate(node *TreeNode) {
	if node.HasChildren() {
		node.Toggle()
	}
	if c.onNodeActivated != nil {
		c.onNodeActivated(node)
	}
}

func (c *TreeView) onMouseMove(x int, y int, mods KeyModifiers) bool {
	c.SetMouseCursor(MouseCursorArrow)
	hover := c.rowAt(y)
	if y < c.scrollY+c.headerHeight() {
		hover = -1
	}
	if hover != c.hoverRow {
		c.hoverRow = hover
		c.form.Update()
	}

	if c.selectionDragging {
		row := c.rowAtClamped(y)
		if node := c.nodeAtRow(row); node != nil && node != c.current {
			if c.multiselect {
				c.current = node
				c.selectRange(node, c.dragBase)
				c.ScrollToNode(node)
				c.fireSelectionChanged()
			} else {
				c.moveTo(node, false, false)
			}
			c.form.Update()
		}
	}
	return true
}

func (c *TreeView) onMouseMoveHeader(x int, y int) MouseCursor {
	if c.columnResizingIndex >= 0 {
		c.SetColumnWidth(c.columnResizingIndex, x-c.columnOffset(c.columnResizingIndex))
		return MouseCursorResizeHor
	}
	if c.hoverRow != -1 {
		c.hoverRow = -1
		c.form.Update()
	}
	if c.headerBorderAt(x) >= 0 {
		return MouseCursorResizeHor
	}
	if c.onColumnClick != nil {
		return MouseCursorPointer
	}
	return MouseCursorArrow
}

// headerBorderAt returns the column whose right border is at x, -1 if none.
// The stretched last column has no border to drag.
func (c *TreeView) headerBorderAt(x int) int {
	last := c.columnCount - 1
	if c.stretchLastColumn {
		last--
	}
	for col := 0; col <= last; col++ {
		border := c.columnOffset(col) + c.ColumnWidth(col)
		if x >= border-treeResizeGrip && x < border+treeResizeGrip {
			return col
		}
	}
	return -1
}

// ---------------------------------------------------------------- Keyboard

func (c *TreeView) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	// The keys the editor doesn't handle come here as to its parent. The
	// navigation keys and Ctrl shortcuts are kept from moving the selection
	// away from the edited node; the others must stay unhandled, or the
	// platform doesn't make the typed character. Tab moves the focus on,
	// which commits the edit.
	if c.editorTextBox != nil {
		switch key {
		case KeyArrowUp, KeyArrowDown, KeyArrowLeft, KeyArrowRight,
			KeyHome, KeyEnd, KeyPageUp, KeyPageDown, KeyF2:
			return true
		}
		return mods.Ctrl && key != KeyTab
	}
	if c.onKeyDown != nil && c.onKeyDown(key, mods) {
		return true
	}
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	c.ensureRows()
	row := c.rowOf(c.current)
	pageRows := max(1, (c.ViewportHeight()-c.headerHeight())/c.rowHeight-1)
	extend, keepSelection := mods.Shift, mods.Ctrl && !mods.Shift

	goRow := func(target int) {
		if len(c.rows) == 0 {
			return
		}
		c.moveTo(c.rows[max(0, min(target, len(c.rows)-1))].node, extend, keepSelection)
	}

	switch key {
	case KeyArrowUp:
		if row < 0 {
			goRow(0)
		} else {
			goRow(row - 1)
		}
	case KeyArrowDown:
		goRow(row + 1)
	case KeyHome:
		goRow(0)
	case KeyEnd:
		goRow(len(c.rows) - 1)
	case KeyPageUp:
		goRow(row - pageRows)
	case KeyPageDown:
		goRow(row + pageRows)

	case KeyArrowRight:
		if mods.Ctrl {
			c.SetCurrentColumn(min(c.currentColumn+1, c.columnCount-1))
			c.ensureColumnVisible(c.currentColumn)
		} else if n := c.current; n != nil && n.HasChildren() {
			if !n.expanded {
				n.Expand()
			} else if len(n.children) > 0 {
				c.moveTo(n.children[0], false, false)
			}
		} else if n == nil {
			goRow(0)
		}
	case KeyArrowLeft:
		if mods.Ctrl {
			c.SetCurrentColumn(max(c.currentColumn-1, 0))
			c.ensureColumnVisible(c.currentColumn)
		} else if n := c.current; n != nil {
			if n.expanded && n.HasChildren() {
				n.Collapse()
			} else if n.parent != nil {
				c.moveTo(n.parent, false, false)
			}
		}
	case KeyNumpadPlus:
		if c.current != nil {
			c.current.Expand()
		}
	case KeyNumpadMinus:
		if c.current != nil {
			c.current.Collapse()
		}
	case KeyNumpadAsterisk:
		if c.current != nil {
			c.current.ExpandAll()
		}

	case KeySpace:
		if !mods.Ctrl || !c.multiselect || c.current == nil {
			return false
		}
		if c.selected[c.current] {
			delete(c.selected, c.current)
		} else {
			c.selected[c.current] = true
		}
		c.anchor = c.current
		c.fireSelectionChanged()

	case KeyEnter:
		if c.current == nil {
			return false
		}
		if c.editTriggerEnter && c.IsCellEditable(c.current, c.currentColumn) {
			c.EditCurrentCell("")
		} else {
			c.activate(c.current)
		}
	case KeyF2:
		col := c.f2Column()
		if col < 0 {
			return false
		}
		c.EditCell(c.current, col, "")

	case KeyA:
		if !mods.Ctrl || !c.multiselect {
			return false
		}
		c.SelectAll()
	case KeyC:
		if !mods.Ctrl {
			return false
		}
		c.CopySelectionToClipboard()

	default:
		return false
	}
	c.form.Update()
	return true
}

// onChar starts editing (SetEditTriggerKeyDown) or jumps to the next node
// whose text starts with the letters typed.
func (c *TreeView) onChar(char rune, mods KeyModifiers) bool {
	if char < ' ' || char == 0x7F || mods.Ctrl || mods.Alt || mods.Cmd {
		return false
	}
	if c.editTriggerKeyDown && c.IsCellEditable(c.current, c.currentColumn) {
		c.EditCurrentCell(string(char))
		return true
	}

	now := time.Now()
	sameSearch := now.Sub(c.typeAheadTime) < treeTypeAheadTimeout
	if char == ' ' && !sameSearch {
		return false // a space starts no search
	}
	c.typeAheadTime = now
	if sameSearch {
		c.typeAhead += string(char)
	} else {
		c.typeAhead = string(char)
	}

	c.ensureRows()
	if len(c.rows) == 0 {
		return true
	}
	prefix := strings.ToLower(c.typeAhead)
	start := max(c.rowOf(c.current), 0)
	if !sameSearch || len([]rune(c.typeAhead)) == 1 {
		start++ // a new search starts after the current node
	}
	for i := 0; i < len(c.rows); i++ {
		n := c.rows[(start+i)%len(c.rows)].node
		if strings.HasPrefix(strings.ToLower(n.Text(0)), prefix) {
			c.moveTo(n, false, false)
			break
		}
	}
	return true
}

// ---------------------------------------------------------------- Painting

func (c *TreeView) draw(cnv *Canvas) {
	p := CurrentPalette()
	first, last := c.visibleRows()
	focused := c.IsFocused() || (c.editorTextBox != nil && c.editorTextBox.IsFocused())
	width := max(c.columnsWidth(), c.ViewportWidth())

	backColor := colorToRGBA(c.BackgroundColor())
	selectedBack, selectedText := p.Highlight, p.HighlightedText
	if !focused {
		// The selection of an inactive tree is kept, but doesn't draw attention
		selectedBack, selectedText = p.Selection, p.Text
	}
	hoverBack := MixColors(backColor, p.Highlight, 0.12)
	arrowColor := MixColors(backColor, p.Text, 0.6)

	for row := first; row < last; row++ {
		r := c.rows[row]
		y := c.headerHeight() + row*c.rowHeight
		selected := c.selected[r.node]

		textColor := color.Color(p.Text)
		switch {
		case selected:
			cnv.FillRect(0, y, width, c.rowHeight, selectedBack)
			textColor = selectedText
		case row == c.hoverRow:
			cnv.FillRect(0, y, width, c.rowHeight, hoverBack)
		}
		if r.node.color != nil && !(selected && focused) {
			textColor = r.node.color
		}

		for col := 0; col < c.columnCount; col++ {
			x := c.columnOffset(col)
			w := c.columnWidth(col)
			cnv.Save()
			cnv.TranslateAndClip(x, y, w, c.rowHeight)
			if col == 0 {
				c.drawTreePart(cnv, r, arrowColor, textColor)
			}
			textX := c.textX(r.node, r.level, col)
			text := truncateTextToWidth(c.FontFamily(), c.FontSize(), r.node.Text(col), w-textX-c.cellPadding)
			cnv.SetHAlign(HAlignLeft)
			cnv.SetVAlign(VAlignCenter)
			cnv.SetFontFamily(c.FontFamily())
			cnv.SetFontSize(c.FontSize())
			cnv.SetColor(textColor)
			cnv.DrawText(textX, 0, w-textX-c.cellPadding, c.rowHeight, text)
			cnv.Restore()
		}

		// The focus frame: which node the keyboard moves from, useful when
		// it isn't the only selected one
		if focused && r.node == c.current && c.multiselect && len(c.selected) != 1 {
			frameColor := p.Highlight
			if selected {
				frameColor = p.HighlightedText
			}
			cnv.SetColor(frameColor)
			cnv.DrawRect(0, y, width, c.rowHeight)
		}
	}

	if c.gridLines {
		divider := p.Divider
		bottom := c.headerHeight() + len(c.rows)*c.rowHeight
		for row := first; row <= last; row++ {
			y := c.headerHeight() + row*c.rowHeight
			if y > bottom {
				break
			}
			cnv.FillRect(0, y, width, 1, divider)
		}
		for col := 1; col < c.columnCount; col++ {
			x := c.columnOffset(col)
			cnv.FillRect(x, c.headerHeight()+first*c.rowHeight, 1, (last-first)*c.rowHeight, divider)
		}
	}
}

// drawTreePart draws the expand arrow and the icon, in the first column's
// coordinates.
func (c *TreeView) drawTreePart(cnv *Canvas, r treeRow, arrowColor color.RGBA, textColor color.Color) {
	indent := c.Indent()
	arrowX := r.level * indent
	midY := c.rowHeight / 2
	if r.node.HasChildren() {
		if rgba, ok := textColor.(color.RGBA); ok && c.selected[r.node] {
			arrowColor = rgba
		}
		const half = 4
		cx := arrowX + indent/2
		if r.node.expanded {
			cnv.FillTriangle(cx-half, midY-half/2, cx+half, midY-half/2, cx, midY+half/2+1, arrowColor)
		} else {
			cnv.FillTriangle(cx-half/2, midY-half, cx-half/2, midY+half, cx+half/2+1, midY, arrowColor)
		}
	}
	if img := r.node.image; img != nil {
		b := img.Bounds()
		x := (r.level+1)*indent + c.cellPadding + (TreeViewIconSize-b.Dx())/2
		cnv.DrawImage(x, (c.rowHeight-b.Dy())/2, img)
	}
}

func (c *TreeView) drawPost(cnv *Canvas) {
	p := CurrentPalette()
	if c.headerVisible {
		// The header is in the view: not over the scroll bars
		cnv.Save()
		c.clipToViewport(cnv)
		y := c.scrollY
		h := c.headerHeight()
		cnv.FillRect(c.scrollX, y, c.ViewportWidth(), h, p.Button)
		for col := 0; col < c.columnCount; col++ {
			x := c.columnOffset(col)
			w := c.columnWidth(col)
			cnv.Save()
			cnv.TranslateAndClip(x, y, w, h)
			cnv.SetHAlign(HAlignLeft)
			cnv.SetVAlign(VAlignCenter)
			cnv.SetFontFamily(c.FontFamily())
			cnv.SetFontSize(c.FontSize())
			cnv.SetColor(p.ButtonText)
			name := truncateTextToWidth(c.FontFamily(), c.FontSize(), c.columnsNames[col], w-c.cellPadding*2)
			cnv.DrawText(c.cellPadding, 0, w-c.cellPadding*2, h, name)
			cnv.Restore()
			if col < c.columnCount-1 || !c.stretchLastColumn {
				cnv.FillRect(x+w-1, y+3, 1, h-6, p.Divider)
			}
		}
		cnv.FillRect(c.scrollX, y+h-1, c.ViewportWidth(), 1, p.Border)
		cnv.Restore()
	}

	cnv.SetColor(p.Border)
	cnv.DrawRect(c.scrollX, c.scrollY, c.Width(), c.Height())
}

func (c *TreeView) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	if !c.customRowHeight {
		c.rowHeight = ThemeRowHeight()
		c.updateInnerSize()
	}
}

// applyLanguage also updates the node texts that follow the language.
func (c *TreeView) applyLanguage() {
	c.Widget.applyLanguage()
	c.walk(func(n *TreeNode) {
		for col, f := range n.textFuncs {
			n.SetText(col, f())
		}
	})
	for col, f := range c.columnsNameFuncs {
		c.columnsNames[col] = f()
	}
}

// ---------------------------------------------------------------- TreeNode

// TreeNode is a row of a TreeView: texts per column, an icon, children.
type TreeNode struct {
	tree     *TreeView
	parent   *TreeNode
	children []*TreeNode

	texts     map[int]string
	textFuncs map[int]func() string
	data      any
	image     image.Image
	color     color.Color
	readOnly  map[int]bool

	expanded    bool
	hasChildren bool
}

func newTreeNode(tree *TreeView, parent *TreeNode, text string) *TreeNode {
	return &TreeNode{tree: tree, parent: parent, texts: map[int]string{0: text}}
}

// Tree returns the tree view the node is in, nil after it was removed.
func (n *TreeNode) Tree() *TreeView {
	return n.tree
}

func (n *TreeNode) Parent() *TreeNode {
	return n.parent
}

func (n *TreeNode) Children() []*TreeNode {
	return n.children
}

func (n *TreeNode) ChildCount() int {
	return len(n.children)
}

// Index returns the position of the node among its siblings.
func (n *TreeNode) Index() int {
	siblings := n.siblings()
	for i := len(siblings) - 1; i >= 0; i-- {
		if siblings[i] == n {
			return i
		}
	}
	return -1
}

// Level is 0 for a top-level node, 1 for its children and so on.
func (n *TreeNode) Level() int {
	level := 0
	for p := n.parent; p != nil; p = p.parent {
		level++
	}
	return level
}

// AddChild adds a child with the text in the first column.
func (n *TreeNode) AddChild(text string) *TreeNode {
	return n.InsertChild(len(n.children), text)
}

// InsertChild adds a child at the index among the children.
func (n *TreeNode) InsertChild(index int, text string) *TreeNode {
	child := newTreeNode(n.tree, n, text)
	index = max(0, min(index, len(n.children)))
	n.children = append(n.children[:index], append([]*TreeNode{child}, n.children[index:]...)...)
	n.changed(true)
	return child
}

// Remove removes the node with its children from the tree.
func (n *TreeNode) Remove() {
	tree := n.tree
	if tree == nil {
		return
	}
	index := n.Index()
	parent := n.parent
	if parent != nil {
		parent.children = append(parent.children[:index], parent.children[index+1:]...)
	} else {
		tree.nodes = append(tree.nodes[:index], tree.nodes[index+1:]...)
	}
	tree.nodeRemoved(n, parent, index)
	n.detach()
	tree.structureChanged()
}

// RemoveChildren removes all the children.
func (n *TreeNode) RemoveChildren() {
	for len(n.children) > 0 {
		n.children[len(n.children)-1].Remove()
	}
}

// detach marks the node and its children as not in a tree anymore.
func (n *TreeNode) detach() {
	n.tree = nil
	for _, child := range n.children {
		child.detach()
	}
}

func (n *TreeNode) Text(col int) string {
	return n.texts[col]
}

func (n *TreeNode) SetText(col int, text string) {
	n.texts[col] = text
	n.changed(false)
}

// SetTextFunc makes the column's text come from f, now and on every
// language change.
func (n *TreeNode) SetTextFunc(col int, f func() string) {
	if n.textFuncs == nil {
		n.textFuncs = make(map[int]func() string)
	}
	n.textFuncs[col] = f
	n.SetText(col, f())
}

// Data returns the value attached to the node with SetData.
func (n *TreeNode) Data() any {
	return n.data
}

func (n *TreeNode) SetData(data any) {
	n.data = data
}

// SetImage sets the icon shown before the text; a larger image is scaled
// down to TreeViewIconSize. nil removes it. Returns the node for chaining.
func (n *TreeNode) SetImage(img image.Image) *TreeNode {
	if img != nil {
		b := img.Bounds()
		if b.Dx() > TreeViewIconSize || b.Dy() > TreeViewIconSize {
			img = resize.Thumbnail(TreeViewIconSize, TreeViewIconSize, img, resize.Lanczos3)
		}
	}
	n.image = img
	n.changed(false)
	return n
}

func (n *TreeNode) Image() image.Image {
	return n.image
}

// SetColor sets the color of the node's texts; nil returns to the default.
func (n *TreeNode) SetColor(col color.Color) {
	n.color = col
	n.changed(false)
}

// SetReadOnly excludes the node's cell in the column from editing.
func (n *TreeNode) SetReadOnly(col int, readOnly bool) {
	if n.readOnly == nil {
		n.readOnly = make(map[int]bool)
	}
	n.readOnly[col] = readOnly
}

func (n *TreeNode) IsReadOnly(col int) bool {
	return n.readOnly[col]
}

// SetHasChildren shows the expand arrow before the children are added,
// for children loaded in TreeView.SetOnExpand.
func (n *TreeNode) SetHasChildren(hasChildren bool) {
	n.hasChildren = hasChildren
	n.changed(false)
}

// HasChildren reports whether the node has children or is marked with
// SetHasChildren.
func (n *TreeNode) HasChildren() bool {
	return len(n.children) > 0 || n.hasChildren
}

func (n *TreeNode) IsExpanded() bool {
	return n.expanded
}

// SetExpanded expands or collapses the node.
func (n *TreeNode) SetExpanded(expanded bool) {
	if expanded {
		n.Expand()
	} else {
		n.Collapse()
	}
}

// Expand shows the children, calling the tree's SetOnExpand function first.
func (n *TreeNode) Expand() {
	if n.expanded {
		return
	}
	if n.tree != nil && n.tree.onExpand != nil {
		n.tree.onExpand(n)
	}
	n.expanded = true
	n.changed(true)
}

// Collapse hides the children.
func (n *TreeNode) Collapse() {
	if !n.expanded {
		return
	}
	n.expanded = false
	if n.tree != nil {
		n.tree.nodeCollapsed(n)
	}
	n.changed(true)
	if n.tree != nil && n.tree.onCollapse != nil {
		n.tree.onCollapse(n)
	}
}

// Toggle expands a collapsed node and collapses an expanded one.
func (n *TreeNode) Toggle() {
	n.SetExpanded(!n.expanded)
}

// ExpandAll expands the node (loading its children, see
// TreeView.SetOnExpand) and the nodes below it that have children; nodes
// below whose children aren't loaded yet stay collapsed.
func (n *TreeNode) ExpandAll() {
	n.Expand()
	for _, child := range n.children {
		child.expandLoaded()
	}
}

// expandLoaded expands the node and the nodes below it that already have
// children, without loading any.
func (n *TreeNode) expandLoaded() {
	if len(n.children) == 0 {
		return
	}
	n.Expand()
	for _, child := range n.children {
		child.expandLoaded()
	}
}

func (n *TreeNode) expandParents() {
	for p := n.parent; p != nil; p = p.parent {
		p.Expand()
	}
}

func (n *TreeNode) isDescendantOf(ancestor *TreeNode) bool {
	for p := n.parent; p != nil; p = p.parent {
		if p == ancestor {
			return true
		}
	}
	return false
}

func (n *TreeNode) siblings() []*TreeNode {
	if n.parent != nil {
		return n.parent.children
	}
	if n.tree != nil {
		return n.tree.nodes
	}
	return nil
}

// changed repaints the tree; structural changes also rebuild its rows.
func (n *TreeNode) changed(structural bool) {
	if n.tree == nil {
		return
	}
	if structural {
		n.tree.structureChanged()
		return
	}
	n.tree.layoutChildren()
	n.tree.form.Update()
}

// handlesKey: Enter and F2 that open the editor of the current cell go to
// the tree before the shortcuts (see Form.AddShortcut)
func (c *TreeView) handlesKey(key Key, mods KeyModifiers) bool {
	if mods.Ctrl || mods.Alt || mods.Shift || mods.Cmd || c.editorTextBox != nil {
		return false
	}
	switch key {
	case KeyEnter:
		return c.editTriggerEnter && c.IsCellEditable(c.current, c.currentColumn)
	case KeyF2:
		return c.f2Column() >= 0
	}
	return false
}

// f2Column is the column F2 edits: the current one, or the first one of the
// current node that can be edited (the name was clicked, the value is
// edited); -1 - none
func (c *TreeView) f2Column() int {
	if !c.editTriggerF2 || c.current == nil {
		return -1
	}
	if c.IsCellEditable(c.current, c.currentColumn) {
		return c.currentColumn
	}
	for col := 0; col < c.columnCount; col++ {
		if c.IsCellEditable(c.current, col) {
			return col
		}
	}
	return -1
}
