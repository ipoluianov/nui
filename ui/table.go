package ui

import (
	"image"
	"image/color"
	"sort"

	"github.com/nfnt/resize"
)

type Table struct {
	Widget

	// Content of the table
	headerRowsCount  int
	headerRows       map[int]*tableHeaderRow
	headerRowHeights map[int]int
	columnsWidths    map[int]int
	rows             map[int]*tableRow

	rowHeight1         int  // Can be changed
	customRowHeight    bool // set by SetRowHeight, else rowHeight1 follows the theme font
	defaultColumnWidth int  // Default width for columns if not set
	// stretchLastColumn widens the last column to the right edge of the table
	stretchLastColumn bool

	rowCount    int
	columnCount int

	columnResizingIndex int

	cellBorderWidth int

	cellBorderColorOverrided *color.RGBA

	cellPadding int

	showSelection bool
	multiselect   bool
	selectingRows bool

	clearSelectionOnEmptyClick bool
	hotTracking                bool // the current cell follows the mouse (a drop-down list)
	cellEditor                 *TextBox
	cellEditorCancel           func()

	// Selection
	currentCellX int
	currentCellY int

	selectedRows  map[int]bool
	selectedCells map[TableCellPos]bool

	selectionAnchorRow int
	selectionAnchorCol int

	selectionDragging      bool
	selectionDragStarted   bool // the mouse moved past tableDragThreshold
	selectionDragStartX    int
	selectionDragStartY    int
	selectionDragBaseRows  map[int]bool
	selectionDragBaseCells map[TableCellPos]bool

	onSelectionChanged  func(row int, col int)
	onCellChanged       func(row int, col int, text string, data interface{}) bool
	onCellMouseDblClick func()
	onCellMouseDown     func(button MouseButton, row int, col int, x int, y int, mods KeyModifiers)
	onCellMouseUp       func(button MouseButton, row int, col int, x int, y int, mods KeyModifiers)

	headerWidget  *tableHeader
	editorTextBox *TextBox
	innerWidgets  []*innerWidget

	editTriggerDoubleClick bool
	editTriggerEnter       bool
	editTriggerF2          bool
	editTriggerKeyDown     bool

	onColumnResize func(col int, newWidth int)
	onColumnClick  func(col int)

	modeLoading     bool
	modeLoadingText string

	previousCurrentCellX int
	previousCurrentCellY int
}

// TableCellPos identifies a single cell by row and column index.
type TableCellPos struct {
	Row int
	Col int
}

type innerWidget struct {
	widget        Widgeter
	posCellCol    int
	posCellRow    int
	widthInCells  int
	heightInCells int
}

type tableRow struct {
	cells map[int]*tableCell
}

type tableHeaderRow struct {
	cells map[int]*tableHeaderCell
}

type tableHeaderCell struct {
	name   string
	hAlign HAlign

	image      image.Image
	imageWidth int

	spanCol int
	spanRow int
}

func (c *tableHeaderCell) SpanCol() int {
	if c.spanCol <= 0 {
		return 1
	}
	return c.spanCol
}

func (c *tableHeaderCell) SpanRow() int {
	if c.spanRow <= 0 {
		return 1
	}
	return c.spanRow
}

func (c *tableHeaderCell) setImage(img image.Image, imgWidth int) {
	c.image = img
	c.imageWidth = imgWidth
}

type tableCell struct {
	text   string
	color  color.Color
	hAlign HAlign
	vAlign VAlign
	data   interface{}

	displayTextExists bool
	displayText       string

	image      image.Image
	imageWidth int

	contraction bool

	editTriggerDoubleClick bool
	editTriggerEnter       bool
	editTriggerF2          bool
	editTriggerKeyDown     bool

	onDraw func(cnv *Canvas)

	selectionDisabled bool
}

// tableDragThreshold is how far the mouse has to move with the button held
// before it starts selecting cells by dragging
const tableDragThreshold = 4

func NewTable() *Table {
	var c Table
	c.InitWidget()
	c.SetAbsolutePositioning(true)
	c.SetTypeName("Table")
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	c.SetAllowScroll(true, true)
	c.scrollBarInset = 1 // inside the border

	c.SetAutoFillBackground(true)
	c.SetRole("base")

	// Events
	c.SetOnPaint(c.draw)
	c.SetOnPostPaint(c.drawPost)
	c.SetOnMouseDown(c.onMouseDown)
	c.SetOnMouseUp(c.onMouseUp)
	c.SetOnMouseDblClick(c.onMouseDblClick)
	//c.SetOnKeyDown(c.onKeyDown)
	c.SetOnKeyUp(c.onKeyUp)
	c.SetOnMouseMove(c.onMouseMove)
	c.SetOnFocused(c.onFocused)

	c.SetOnScrollChanged(func(scrollX, scrollY int) {
		c.updateInnerWidgetsLayout()
	})

	// Init runtime
	c.SetCanBeFocused(true)
	c.rows = make(map[int]*tableRow)
	c.rowHeight1 = ThemeRowHeight()
	c.headerRows = make(map[int]*tableHeaderRow)
	c.columnsWidths = make(map[int]int)
	c.headerRowHeights = make(map[int]int)
	c.defaultColumnWidth = 200

	c.headerRowsCount = 1

	c.columnResizingIndex = -1

	c.cellBorderWidth = 1
	//c.cellBorderColor = c.
	c.cellPadding = 3

	c.showSelection = true
	c.selectingRows = true
	c.multiselect = false
	c.clearSelectionOnEmptyClick = true
	c.selectedRows = make(map[int]bool)
	c.selectedCells = make(map[TableCellPos]bool)

	c.headerWidget = newTableHeader()
	c.headerWidget.OnHeaderMouseDown = func(button MouseButton, x, y int, mods KeyModifiers) bool {
		return c.onMouseDown(button, x+c.scrollX, y+c.scrollY, mods)
	}
	c.headerWidget.OnHeaderMouseUp = func(button MouseButton, x, y int, mods KeyModifiers) bool {
		return c.onMouseUp(button, x+c.scrollX, y+c.scrollY, mods)
	}
	c.headerWidget.OnHeaderMouseMove = func(x, y int, mods KeyModifiers) MouseCursor {
		return c.onMouseMoveHeader(x+c.scrollX, y+c.scrollY, mods)
	}
	c.AddWidget(0, 0, c.headerWidget)

	c.innerWidgets = make([]*innerWidget, 0)

	c.updateInnerWidgetsLayout()

	return &c
}

func (c *Table) TableSetLayoutXml(n *uiNode) {
	// columns
	columnsNode := n.GetChildByName("columns")
	if columnsNode != nil {
		type columnInfo struct {
			text  string
			width int
		}
		columnsInfos := make([]*columnInfo, 0)
		columnIndex := 0
		for _, columnNode := range columnsNode.Nodes {
			if columnNode.XMLName.Local == "column" {
				var colInfo columnInfo
				colInfo.text = columnNode.GetAttrValueByName("text", "")
				colInfo.width = columnNode.GetAttrValueByNameInt("width", c.defaultColumnWidth)
				columnsInfos = append(columnsInfos, &colInfo)
				columnIndex++
			}
		}

		c.SetColumnCount(columnIndex)
		for index, colInfo := range columnsInfos {
			c.SetColumnName(index, colInfo.text)
			c.SetColumnWidth(index, colInfo.width)
		}
	}

	rowsNode := n.GetChildByName("rows")
	if rowsNode != nil {
		rowIndex := 0
		for _, rowNode := range rowsNode.Nodes {
			if rowNode.XMLName.Local == "row" {
				cellIndex := 0
				for _, cellNode := range rowNode.Nodes {
					if cellNode.XMLName.Local == "cell" {
						c.SetCellText2(rowIndex, cellIndex, cellNode.GetAttrValueByName("text", ""))
						cellIndex++
					}
				}
				rowIndex++
			}
		}
		c.SetRowCount(rowIndex)
	}
}

func (c *Table) PreviousCurrentCellX() int {
	return c.previousCurrentCellX
}

func (c *Table) PreviousCurrentCellY() int {
	return c.previousCurrentCellY
}

func (c *Table) SetRowHeight(height int) {
	c.customRowHeight = true
	// At least a pixel: the rows under a point are found dividing by it
	c.rowHeight1 = max(height, 1)
	c.updateInnerSize()
	c.form.UpdateLayout()
	c.form.Update()
}

// SetCellPadding sets the space between the text of the cells (and of the
// header) and their edges
func (c *Table) SetCellPadding(padding int) {
	c.cellPadding = padding
	c.form.Update()
}

func (c *Table) CellPadding() int {
	return c.cellPadding
}

func (c *Table) SetCellOnDraw(row int, col int, onDraw func(cnv *Canvas)) {
	cellObj := c.getCellObj(row, col)
	cellObj.onDraw = onDraw
	c.form.Update()
}

func (c *Table) SetCellBorderColor(col color.RGBA) {
	c.cellBorderColorOverrided = &col
}

func (c *Table) CellBorderColor() color.Color {
	if c.cellBorderColorOverrided != nil {
		return *c.cellBorderColorOverrided
	}
	return CurrentPalette().Divider
}

func (c *Table) SetCellBorderWidth(width int) {
	c.cellBorderWidth = width
}

func (c *Table) CellBorderWidth() int {
	return c.cellBorderWidth
}

func (c *Table) SetModeLoading(loading bool, text string) {
	c.modeLoading = loading
	c.modeLoadingText = text
	c.form.Update()
}

func (c *Table) SetOnColumnResize(callback func(col int, newWidth int)) {
	c.onColumnResize = callback
}

func (c *Table) SetOnColumnClick(callback func(col int)) {
	c.onColumnClick = callback
}

func (c *Table) SetOnCellChanged(callback func(row int, col int, text string, data interface{}) bool) {
	c.onCellChanged = callback
}

func (c *Table) SetEditTriggerDoubleClick(enabled bool) {
	c.editTriggerDoubleClick = enabled
}

func (c *Table) SetEditTriggerEnter(enabled bool) {
	c.editTriggerEnter = enabled
}

func (c *Table) SetEditTriggerF2(enabled bool) {
	c.editTriggerF2 = enabled
}

func (c *Table) SetEditTriggerKeyDown(enabled bool) {
	c.editTriggerKeyDown = enabled
}

func (c *Table) SetOnSelectionChanged(callback func(x int, y int)) {
	c.onSelectionChanged = callback
}

func (c *Table) AddWidgetOnTable(widget Widgeter, posCellRow int, posCellCol int, widthInCells int, heightInCells int) {
	if posCellCol < 0 || posCellRow < 0 || posCellCol >= c.columnCount || posCellRow >= c.rowCount {
		return
	}
	if posCellCol+widthInCells > c.columnCount || posCellRow+heightInCells > c.rowCount {
		return
	}

	var inWidget innerWidget
	inWidget.widget = widget
	inWidget.posCellCol = posCellCol
	inWidget.posCellRow = posCellRow
	inWidget.widthInCells = widthInCells
	inWidget.heightInCells = heightInCells
	c.innerWidgets = append(c.innerWidgets, &inWidget)
	c.AddWidget(0, 0, widget)
	c.updateInnerWidgetsLayout()
}

func (c *Table) rowOffset(row int) int {
	if row < 0 || row >= c.rowCount {
		return 0
	}
	return c.headerHeight() + row*c.rowHeight1
}

func (c *Table) headerCell2(rowIndex int, colIndex int) *tableHeaderCell {
	if colIndex < 0 || colIndex >= c.columnCount {
		return &tableHeaderCell{name: ""}
	}

	if rowIndex < 0 || rowIndex >= c.headerRowsCount {
		return &tableHeaderCell{name: ""}
	}

	headerRow, exists := c.headerRows[rowIndex]
	if !exists {
		headerRow = &tableHeaderRow{cells: make(map[int]*tableHeaderCell)}
		c.headerRows[rowIndex] = headerRow
	}

	cell, exists := headerRow.cells[colIndex]
	if !exists {
		cell = &tableHeaderCell{name: ""}
		headerRow.cells[colIndex] = cell
	}

	return cell
}

func (c *Table) headerCellShadowed2(rowIndex int, colIndex int) bool {
	result := false

	for headerRowIndex := 0; headerRowIndex < c.headerRowsCount; headerRowIndex++ {
		for headerColIndex := 0; headerColIndex < c.columnCount; headerColIndex++ {
			if headerRowIndex == rowIndex && headerColIndex == colIndex {
				continue
			}
			headerCell := c.headerCell2(headerRowIndex, headerColIndex)
			spanX := headerCell.SpanCol()
			spanY := headerCell.SpanRow()
			if spanX > 1 || spanY > 1 {
				cellSpanX1 := headerColIndex
				cellSpanX2 := headerColIndex + spanX - 1
				cellSpanY1 := headerRowIndex
				cellSpanY2 := headerRowIndex + spanY - 1
				if colIndex >= cellSpanX1 && colIndex <= cellSpanX2 &&
					rowIndex >= cellSpanY1 && rowIndex <= cellSpanY2 {
					result = true
					break
				}
			}
		}
	}

	return result
}

/*func (c *Table) columnWidth(col int) int {
	if col < 0 || col >= c.columnCount {
		return c.defaultColumnWidth
	}
	colWidth, exists := c.columnsWidths[col]
	if !exists {
		return c.defaultColumnWidth
	}
	return colWidth
}*/

func (c *Table) updateInnerWidgetsLayout() {
	c.headerWidget.SetPosition(c.scrollX, c.scrollY)
	c.headerWidget.SetSize(c.innerWidth, c.headerHeight())

	for _, inWidget := range c.innerWidgets {
		widgetPosInPixelsX := c.columnOffset(inWidget.posCellCol)
		widgetPosInPixelsY := c.rowOffset(inWidget.posCellRow)
		widgetWidthInPixels := 0
		for i := 0; i < inWidget.widthInCells; i++ {
			widgetWidthInPixels += c.columnWidth(inWidget.posCellCol + i)
		}
		widgetHeightInPixels := inWidget.heightInCells * c.rowHeight1
		inWidget.widget.SetPosition(widgetPosInPixelsX, widgetPosInPixelsY)
		inWidget.widget.SetSize(widgetWidthInPixels, widgetHeightInPixels)
	}
}

func (c *Table) RowCount() int {
	return c.rowCount
}

// ClearRows removes all the rows, as in a new table: the cells with their
// text, data, images and colors, the selection and the current cell.
// RowCount becomes 0. An open cell editor is closed without applying it.
//
// SetRowCount only changes how many rows are shown: the cells of the rows
// beyond the count are kept, and come back when the count grows again. To
// load other data, call ClearRows first, then fill the cells and set the
// count.
func (c *Table) ClearRows() {
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	// The editors, without their commit on the focus loss
	c.CancelCellEditor()
	if editor := c.editorTextBox; editor != nil {
		hadFocus := c.form != nil && c.form.FocusedWidget() == Widgeter(editor)
		c.editorTextBox = nil
		c.removeInnerWidget(editor)
		if hadFocus {
			c.Focus()
		}
	}

	hadSelection := c.CurrentRow() >= 0 || len(c.selectedRows) > 0 || len(c.selectedCells) > 0
	c.rows = make(map[int]*tableRow)
	c.rowCount = 0
	c.previousCurrentCellX, c.previousCurrentCellY = c.currentCellX, c.currentCellY
	c.currentCellX, c.currentCellY = 0, 0
	c.selectionAnchorRow, c.selectionAnchorCol = 0, 0
	c.selectionDragging = false
	c.clearSelectionSets()
	c.updateInnerSize()
	c.updateInnerWidgetsLayout()

	c.form.Update()
	if hadSelection && c.onSelectionChanged != nil {
		c.onSelectionChanged(c.CurrentRow(), c.CurrentColumn())
	}
}

func (c *Table) SetRowCount(count int) {
	c.rowCount = count
	c.updateInnerSize()
	c.updateInnerWidgetsLayout()

	// Preserve the historical default of the first cell/row being selected
	// as soon as data exists, as long as nothing has been selected yet.
	if len(c.selectedRows) == 0 && len(c.selectedCells) == 0 {
		c.syncSelectionToCurrent()
	}
}

func (c *Table) SetColumnCount(count int) {
	c.columnCount = count
	c.updateInnerSize()
	c.updateInnerWidgetsLayout()
}

func (c *Table) SetColumnWidth(col int, width int) {
	c.columnsWidths[col] = width

	c.updateInnerSize()
	c.updateInnerWidgetsLayout()

	if c.onColumnResize != nil {
		c.onColumnResize(col, width)
	}
}

func (c *Table) SetColumnImage(col int, img image.Image, imgWidth int) {
	if col < 0 || col >= c.columnCount {
		return
	}
	colHeader := c.headerCell2(0, col)
	colHeader.setImage(img, imgWidth)
	c.form.Update()
}

func (c *Table) ColumnWidth(col int) int {
	if col < 0 || col >= c.columnCount {
		return c.defaultColumnWidth
	}
	colWidth, exists := c.columnsWidths[col]
	if !exists {
		return c.defaultColumnWidth
	}
	return colWidth
}

func (c *Table) RowHeight() int {
	return c.rowHeight1
}

func (c *Table) ColumnCount() int {
	return c.columnCount
}

func (c *Table) SetColumnCellName2(row int, col int, name string) {
	if col < 0 || col >= c.columnCount {
		return
	}
	headerCell := c.headerCell2(row, col)
	headerCell.name = name
	c.updateInnerSize()
}

func (c *Table) SetColumnName(col int, name string) {
	if col < 0 || col >= c.columnCount {
		return
	}
	headerCell := c.headerCell2(0, col)
	headerCell.name = name
	c.updateInnerSize()
}

// SetColumnHAlign aligns the column's header text, e.g. to the right over numbers
// aligned with SetCellHAlign
func (c *Table) SetColumnHAlign(col int, align HAlign) {
	if col < 0 || col >= c.columnCount {
		return
	}
	c.headerCell2(0, col).hAlign = align
	c.form.Update()
}

// SetStretchLastColumn makes the last column fill the table up to its right
// edge, so there is no empty strip after the columns. Its width set with
// SetColumnWidth stays the minimum; the table doesn't scroll to show the rest.
func (c *Table) SetStretchLastColumn(stretch bool) {
	c.stretchLastColumn = stretch
	c.form.Update()
}

func (c *Table) ColumnName(col int) string {
	if col < 0 || col >= c.columnCount {
		return ""
	}
	headerCell := c.headerCell2(0, col)
	return headerCell.name
}

func (c *Table) newTableCell() *tableCell {
	return &tableCell{
		text:   "",
		color:  nil,
		data:   nil,
		hAlign: HAlignLeft,
		vAlign: VAlignCenter,
	}
}

func (c *Table) getCellObj(row int, col int) *tableCell {
	rowObj, exists := c.rows[row]
	if !exists {
		rowObj = &tableRow{cells: make(map[int]*tableCell)}
		c.rows[row] = rowObj
	}
	cellObj, exists := rowObj.cells[col]
	if !exists {
		cellObj = c.newTableCell()
		rowObj.cells[col] = cellObj
	}
	return cellObj
}

func (c *Table) SetCellText2(row int, col int, text string) {
	cellObj := c.getCellObj(row, col)

	originalText := cellObj.text
	cellObj.text = text
	cellObj.displayTextExists = false

	if c.onCellChanged != nil {
		b := c.onCellChanged(row, col, cellObj.text, cellObj.data)
		if !b {
			cellObj.text = originalText
		}
	}

	c.form.Update()
}

func (c *Table) SetCellDisplayText(row int, col int, text string) {
	cellObj := c.getCellObj(row, col)

	cellObj.displayText = text
	cellObj.displayTextExists = true

	c.form.Update()
}

func (c *Table) SetCellImage(row int, col int, img image.Image, imgWidth int) {
	cellObj := c.getCellObj(row, col)
	cellObj.image = img
	cellObj.imageWidth = imgWidth
	c.form.Update()
}

func (c *Table) SetCellContraction(row int, col int, contraction bool) {
	cellObj := c.getCellObj(row, col)
	cellObj.contraction = contraction
	c.form.Update()
}

func (c *Table) SetCellData2(row int, col int, data interface{}) {
	cellObj := c.getCellObj(row, col)
	cellObj.data = data
	c.form.Update()
}

func (c *Table) SetCellColor(row int, col int, color color.Color) {
	cellObj := c.getCellObj(row, col)
	cellObj.color = color
	c.form.Update()
}

func (c *Table) SetCellHAlign(row int, col int, align HAlign) {
	cellObj := c.getCellObj(row, col)
	cellObj.hAlign = align
	c.form.Update()
}

func (c *Table) SetCellVAlign(row int, col int, align VAlign) {
	cellObj := c.getCellObj(row, col)
	cellObj.vAlign = align
	c.form.Update()
}

func (c *Table) SetCellEditTriggerDoubleClick(row int, col int, enabled bool) {
	cellObj := c.getCellObj(row, col)
	cellObj.editTriggerDoubleClick = enabled
	c.form.Update()
}

func (c *Table) SetCellEditTriggerEnter(row int, col int, enabled bool) {
	cellObj := c.getCellObj(row, col)
	cellObj.editTriggerEnter = enabled
	c.form.Update()
}

func (c *Table) SetCellEditTriggerF2(row int, col int, enabled bool) {
	cellObj := c.getCellObj(row, col)
	cellObj.editTriggerF2 = enabled
	c.form.Update()
}

func (c *Table) SetCellEditTriggerKeyDown(row int, col int, enabled bool) {
	cellObj := c.getCellObj(row, col)
	cellObj.editTriggerKeyDown = enabled
	c.form.Update()
}

func (c *Table) SetCellSelectionDisabled(row int, col int, disabled bool) {
	cellObj := c.getCellObj(row, col)
	cellObj.selectionDisabled = disabled
	c.form.Update()
}

func (c *Table) SetCurrentCell2(row int, col int) {
	c.form.LayoutingBlockPush()
	defer c.form.LayoutingBlockPop()
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	c.previousCurrentCellX = c.currentCellX
	c.previousCurrentCellY = c.currentCellY

	cellObj := c.getCellObj(row, col)
	if cellObj.selectionDisabled {
		return
	}

	if row < 0 || row >= c.rowCount || col < 0 || col >= c.columnCount {
		return
	}
	c.currentCellX = col
	c.currentCellY = row

	c.selectionAnchorRow = row
	c.selectionAnchorCol = col
	c.syncSelectionToCurrent()

	c.ScrollToCell2(c.currentCellY, c.currentCellX)

	c.form.Update()
	if c.onSelectionChanged != nil {
		c.onSelectionChanged(c.currentCellY, c.currentCellX)
	}
}

// moveCurrentCellForSelection updates the active cell the same way SetCurrentCell2
// does (position, scroll, onSelectionChanged), but without resetting the
// multi-selection - used while extending a selection via Shift/Ctrl or drag.
func (c *Table) moveCurrentCellForSelection(row int, col int) {
	c.form.LayoutingBlockPush()
	defer c.form.LayoutingBlockPop()
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	c.previousCurrentCellX = c.currentCellX
	c.previousCurrentCellY = c.currentCellY

	c.currentCellX = col
	c.currentCellY = row

	c.ScrollToCell2(row, col)

	c.form.Update()
	if c.onSelectionChanged != nil {
		c.onSelectionChanged(row, col)
	}
}

// navigateTo moves the active cell for keyboard navigation. When extend is
// true and multiselect is enabled, it extends the selection range from the
// anchor instead of resetting it - the Shift+Arrow/Home/End/PageUp/PageDown
// behavior, mirroring Shift+click.
func (c *Table) navigateTo(row int, col int, extend bool) {
	if row < 0 || row >= c.rowCount || col < 0 || col >= c.columnCount {
		return
	}
	cellObj := c.getCellObj(row, col)
	if cellObj.selectionDisabled {
		return
	}

	if extend && c.multiselect {
		c.applyRangeSelection(c.selectionAnchorRow, c.selectionAnchorCol, row, col, nil, nil)
		c.moveCurrentCellForSelection(row, col)
		return
	}

	c.SetCurrentCell2(row, col)
}

// clearSelectionSets empties the multi-selection (row and cell sets).
func (c *Table) clearSelectionSets() {
	c.selectedRows = make(map[int]bool)
	c.selectedCells = make(map[TableCellPos]bool)
}

// syncSelectionToCurrent collapses the multi-selection down to the single
// active cell/row - the standard behavior on a plain click or keyboard nav.
func (c *Table) syncSelectionToCurrent() {
	c.clearSelectionSets()
	if c.currentCellY < 0 || c.currentCellY >= c.rowCount {
		return
	}
	if c.selectingRows {
		c.selectedRows[c.currentCellY] = true
	} else if c.currentCellX >= 0 && c.currentCellX < c.columnCount {
		c.selectedCells[TableCellPos{Row: c.currentCellY, Col: c.currentCellX}] = true
	}
}

func (c *Table) cloneSelectedRows() map[int]bool {
	result := make(map[int]bool, len(c.selectedRows))
	for k, v := range c.selectedRows {
		result[k] = v
	}
	return result
}

func (c *Table) cloneSelectedCells() map[TableCellPos]bool {
	result := make(map[TableCellPos]bool, len(c.selectedCells))
	for k, v := range c.selectedCells {
		result[k] = v
	}
	return result
}

// applyRangeSelection sets the selection to baseRows/baseCells plus the
// rectangular range between (fromRow,fromCol) and (toRow,toCol) - a row
// range when selectingRows is on, a cell rectangle otherwise. The base maps
// are copied, never mutated.
func (c *Table) applyRangeSelection(fromRow int, fromCol int, toRow int, toCol int, baseRows map[int]bool, baseCells map[TableCellPos]bool) {
	rowsSet := make(map[int]bool, len(baseRows))
	for k, v := range baseRows {
		rowsSet[k] = v
	}
	cellsSet := make(map[TableCellPos]bool, len(baseCells))
	for k, v := range baseCells {
		cellsSet[k] = v
	}

	r1, r2 := fromRow, toRow
	if r1 > r2 {
		r1, r2 = r2, r1
	}

	if c.selectingRows {
		for r := r1; r <= r2; r++ {
			rowsSet[r] = true
		}
	} else {
		c1, c2 := fromCol, toCol
		if c1 > c2 {
			c1, c2 = c2, c1
		}
		for r := r1; r <= r2; r++ {
			for cc := c1; cc <= c2; cc++ {
				cellsSet[TableCellPos{Row: r, Col: cc}] = true
			}
		}
	}

	c.selectedRows = rowsSet
	c.selectedCells = cellsSet
}

// toggleSelectionItem adds or removes the given row/cell from the selection.
func (c *Table) toggleSelectionItem(row int, col int) {
	if c.selectingRows {
		if c.selectedRows[row] {
			delete(c.selectedRows, row)
		} else {
			c.selectedRows[row] = true
		}
		return
	}

	pos := TableCellPos{Row: row, Col: col}
	if c.selectedCells[pos] {
		delete(c.selectedCells, pos)
	} else {
		c.selectedCells[pos] = true
	}
}

// handleSelectionMouseDown applies standard multi-select click semantics:
// plain click replaces the selection, Shift extends a range from the anchor,
// Ctrl toggles a single item (and Ctrl+Shift extends additively), and any of
// them arms drag-to-extend for the following mouse moves.
func (c *Table) handleSelectionMouseDown(row int, col int, mods KeyModifiers) {
	cellObj := c.getCellObj(row, col)
	if cellObj.selectionDisabled {
		return
	}

	// Cmd is the toggle modifier on macOS, Ctrl elsewhere
	toggle := mods.Ctrl || mods.Cmd
	if c.multiselect && mods.Shift {
		if toggle {
			c.selectionDragBaseRows = c.cloneSelectedRows()
			c.selectionDragBaseCells = c.cloneSelectedCells()
		} else {
			c.selectionDragBaseRows = make(map[int]bool)
			c.selectionDragBaseCells = make(map[TableCellPos]bool)
		}
		c.applyRangeSelection(c.selectionAnchorRow, c.selectionAnchorCol, row, col, c.selectionDragBaseRows, c.selectionDragBaseCells)
		c.moveCurrentCellForSelection(row, col)
		c.selectionDragging = true
		return
	}

	if c.multiselect && toggle {
		c.toggleSelectionItem(row, col)
		c.selectionAnchorRow = row
		c.selectionAnchorCol = col
		c.selectionDragBaseRows = c.cloneSelectedRows()
		c.selectionDragBaseCells = c.cloneSelectedCells()
		c.moveCurrentCellForSelection(row, col)
		c.selectionDragging = true
		return
	}

	// Plain click: reset the selection to this single item; it becomes the
	// drag anchor for a following Shift-click or mouse drag. Dragging is
	// armed even without multiselect, so the single selection follows the
	// mouse while the button is held.
	c.SetCurrentCell2(row, col)
	c.selectionDragBaseRows = make(map[int]bool)
	c.selectionDragBaseCells = make(map[TableCellPos]bool)
	c.selectionDragging = true
}

// ClearSelection removes the selection entirely: nothing is selected and
// CurrentRow() returns -1 until a row is clicked or navigated to again
// (Arrow Down / Home then start from the first row).
func (c *Table) ClearSelection() {
	if c.currentCellY < 0 && len(c.selectedRows) == 0 && len(c.selectedCells) == 0 {
		return
	}

	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	c.previousCurrentCellX = c.currentCellX
	c.previousCurrentCellY = c.currentCellY
	c.currentCellY = -1
	c.selectionAnchorRow = 0
	c.selectionAnchorCol = max(c.currentCellX, 0)
	c.selectionDragging = false
	c.clearSelectionSets()

	c.form.Update()
	if c.onSelectionChanged != nil {
		c.onSelectionChanged(c.CurrentRow(), c.CurrentColumn())
	}
}

// SelectAll selects every row (in row-selection mode) or every cell (in
// cell-selection mode). No-op unless multiselect is enabled.
func (c *Table) SelectAll() {
	if !c.multiselect || c.rowCount <= 0 || c.columnCount <= 0 {
		return
	}

	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	rows := make(map[int]bool)
	cells := make(map[TableCellPos]bool)
	if c.selectingRows {
		for r := 0; r < c.rowCount; r++ {
			rows[r] = true
		}
	} else {
		for r := 0; r < c.rowCount; r++ {
			for cc := 0; cc < c.columnCount; cc++ {
				cells[TableCellPos{Row: r, Col: cc}] = true
			}
		}
	}
	c.selectedRows = rows
	c.selectedCells = cells
	c.form.Update()
	if c.onSelectionChanged != nil {
		c.onSelectionChanged(c.currentCellY, c.currentCellX)
	}
}

// SetSelectedRows selects the rows (in row-selection mode), e.g. to keep a
// selection after the rows were reloaded; the first one becomes the current row.
// Rows out of range are skipped. Without multiselect only the first row is selected.
func (c *Table) SetSelectedRows(rows []int) {
	valid := make([]int, 0, len(rows))
	for _, r := range rows {
		if r >= 0 && r < c.rowCount {
			valid = append(valid, r)
		}
	}
	if len(valid) == 0 {
		return
	}
	sort.Ints(valid)
	c.SetCurrentCell2(valid[0], max(c.currentCellX, 0))
	if !c.multiselect || !c.selectingRows {
		return
	}
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()
	selected := make(map[int]bool, len(valid))
	for _, r := range valid {
		selected[r] = true
	}
	c.selectedRows = selected
	c.form.Update()
	if c.onSelectionChanged != nil {
		c.onSelectionChanged(c.currentCellY, c.currentCellX)
	}
}

// SelectedRows returns the sorted list of currently selected row indices.
// Meaningful when SelectingRows() is true.
func (c *Table) SelectedRows() []int {
	result := make([]int, 0, len(c.selectedRows))
	for r := range c.selectedRows {
		if r < 0 || r >= c.rowCount {
			continue
		}
		result = append(result, r)
	}
	sort.Ints(result)
	return result
}

// SelectedCells returns the sorted list of currently selected cells.
// Meaningful when SelectingRows() is false.
func (c *Table) SelectedCells() []TableCellPos {
	result := make([]TableCellPos, 0, len(c.selectedCells))
	for pos := range c.selectedCells {
		if pos.Row < 0 || pos.Row >= c.rowCount || pos.Col < 0 || pos.Col >= c.columnCount {
			continue
		}
		result = append(result, pos)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Row != result[j].Row {
			return result[i].Row < result[j].Row
		}
		return result[i].Col < result[j].Col
	})
	return result
}

// IsRowSelected reports whether the given row is selected. Only meaningful
// when SelectingRows() is true.
func (c *Table) IsRowSelected(row int) bool {
	return c.selectingRows && c.selectedRows[row]
}

// IsCellSelected reports whether the given cell is selected. Only meaningful
// when SelectingRows() is false.
func (c *Table) IsCellSelected(row int, col int) bool {
	return !c.selectingRows && c.selectedCells[TableCellPos{Row: row, Col: col}]
}

func (c *Table) SetHeaderRowCount(count int) {
	c.headerRowsCount = count
	c.updateInnerSize()
	c.updateInnerWidgetsLayout()
}

func (c *Table) SetHeaderCellSpan2(row int, col int, spanRow int, spanCol int) {
	if col < 0 || col >= c.columnCount || row < 0 || row >= c.headerRowsCount {
		return
	}
	headerCell := c.headerCell2(row, col)
	headerCell.spanCol = spanCol
	headerCell.spanRow = spanRow
	c.updateInnerSize()
	c.updateInnerWidgetsLayout()
}

func (c *Table) CurrentRow() int {
	if c.currentCellY < 0 || c.currentCellY >= c.rowCount {
		return -1
	}
	return c.currentCellY
}

func (c *Table) CurrentColumn() int {
	if c.currentCellX < 0 || c.currentCellX >= c.columnCount {
		return -1
	}
	return c.currentCellX
}

func (c *Table) GetCellText2(row int, col int) string {
	if row < 0 || row >= c.rowCount || col < 0 || col >= c.columnCount {
		return ""
	}
	rowObj, exists := c.rows[row]
	if !exists {
		return ""
	}
	cellObj, exists := rowObj.cells[col]
	if !exists {
		return ""
	}
	return cellObj.text
}

func (c *Table) GetCellData2(row int, col int) interface{} {
	if row < 0 || row >= c.rowCount || col < 0 || col >= c.columnCount {
		return nil
	}
	rowObj, exists := c.rows[row]
	if !exists {
		return nil
	}
	cellObj, exists := rowObj.cells[col]
	if !exists {
		return nil
	}
	return cellObj.data
}

func (c *Table) ScrollToCell2(row, col int) {
	c.form.LayoutingBlockPush()
	defer c.form.LayoutingBlockPop()
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	if row < 0 || row >= c.rowCount || col < 0 || col >= c.columnCount {
		return
	}

	leftTopX := c.columnOffset(col)
	leftTopY := c.rowOffset(row)
	c.ScrollEnsureVisible(leftTopX, leftTopY-c.headerHeight())

	rightBottomX := leftTopX + c.columnWidth(col)
	rightBottomY := leftTopY + c.rowHeight1
	c.ScrollEnsureVisible(rightBottomX, rightBottomY)
	c.form.Update()
}

func (c *Table) onMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	// The header is a separate child widget that isn't itself focusable, so
	// a click landing on it (e.g. to glance at a column name) makes Form's
	// generic pre-dispatch focus logic treat it as "clicked something
	// unfocusable" and clear the table's focus entirely - after which arrow
	// keys stop navigating cells until a data cell is clicked again. Header
	// clicks are forwarded here too (see OnHeaderMouseDown below), so
	// re-asserting focus on every click keeps the table keyboard-navigable
	// no matter where within it the user clicks.
	c.Focus()

	c.form.LayoutingBlockPush()
	defer c.form.LayoutingBlockPop()
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	headerColumnBorder := c.headerColumnBorderByPosition(x, y)
	if headerColumnBorder >= 0 {
		// fmt.Println("Header column border clicked:", headerColumnBorder)
		c.columnResizingIndex = headerColumnBorder
		return true
	}

	headerColumn := c.headerColumnByPosition(x, y)
	if headerColumn >= 0 {
		// fmt.Println("Header column clicked:", headerColumn)
		if c.onColumnClick != nil {
			c.onColumnClick(headerColumn)
		}
		return true
	}

	col, row := c.cellByPosition(x, y)
	if row >= 0 && col >= 0 && button == MouseButtonLeft {
		c.selectionDragStarted = false
		c.selectionDragStartX = x
		c.selectionDragStartY = y
		c.handleSelectionMouseDown(row, col, mods)
		//fmt.Println("Cell clicked:", col, row, " at ", x, y)
	} else if button == MouseButtonLeft && c.clearSelectionOnEmptyClick && !mods.Ctrl && !mods.Cmd && !mods.Shift {
		// A plain click on the empty area below the rows / right of the columns.
		c.ClearSelection()
	}

	if c.onCellMouseDown != nil {
		c.onCellMouseDown(button, row, col, x, y, mods)
	}

	return true
}

// ProcessMouseDown selects the row (cell) under a right click before the
// context menu is shown, so the menu acts on what was clicked. An already
// selected row keeps the selection, so a menu over a multi-selection acts on all of it.
func (c *Table) ProcessMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if button == MouseButtonRight && c.ContextMenu() != nil && y >= c.headerHeight() {
		col, row := c.cellByPosition(x+c.scrollX, y+c.scrollY)
		if row >= 0 && col >= 0 && !c.IsRowSelected(row) && !c.IsCellSelected(row, col) {
			c.SetCurrentCell2(row, col)
		}
	}
	return c.Widget.ProcessMouseDown(button, x, y, mods)
}

func (c *Table) onMouseUp(button MouseButton, x int, y int, mods KeyModifiers) bool {
	resizing := c.columnResizingIndex >= 0
	c.columnResizingIndex = -1
	c.selectionDragging = false
	if c.onCellMouseUp != nil && !resizing {
		col, row := c.cellByPosition(x, y)
		c.onCellMouseUp(button, row, col, x, y, mods)
	}
	return true
}

// SetOnCellMouseUp is called when a mouse button is released over the table;
// row and col are -1 outside the cells
func (c *Table) SetOnCellMouseUp(callback func(button MouseButton, row int, col int, x int, y int, mods KeyModifiers)) {
	c.onCellMouseUp = callback
}

// SetHotTracking makes the current cell follow the mouse, as in a drop-down
// list: the cell under the mouse is lit and the mouse is a hand over it
// (cells with the selection disabled are skipped)
func (c *Table) SetHotTracking(enabled bool) {
	c.hotTracking = enabled
}

type EventTableCellMouseDblClick struct {
	Table     *Table
	Row       int
	Col       int
	Processed bool
}

func (c *Table) SetOnCellMouseDblClick(callback func()) {
	c.onCellMouseDblClick = callback
}

func (c *Table) SetOnCellMouseDown(callback func(button MouseButton, row int, col int, x int, y int, mods KeyModifiers)) {
	c.onCellMouseDown = callback
}

func (c *Table) onMouseDblClick(button MouseButton, x int, y int, mods KeyModifiers) bool {
	col, row := c.cellByPosition(x, y)
	if row >= 0 && col >= 0 {
		// A quick Shift/Ctrl+click on another cell arrives as a double click;
		// it must extend or toggle the selection, not reset it
		if c.multiselect && (mods.Shift || mods.Ctrl || mods.Cmd) {
			c.handleSelectionMouseDown(row, col, mods)
			return true
		}
		c.SetCurrentCell2(row, col)

		if c.onCellMouseDblClick != nil {
			var event EventTableCellMouseDblClick
			event.Row = row
			event.Col = col
			event.Table = c
			PushEvent(&event)
			c.onCellMouseDblClick()
			PopEvent()
			if event.Processed {
				return true
			}
		}

		if c.editTriggerDoubleClick {
			c.EditCurrentCell("")
		}

		return true
	}
	return false
}

func (c *Table) onFocused() {
}

func (c *Table) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	c.form.LayoutingBlockPush()
	defer c.form.LayoutingBlockPop()
	c.form.UpdateBlockPush()
	defer c.form.UpdateBlockPop()

	processed := false

	if c.onKeyDown != nil {
		processed = c.onKeyDown(key, mods)
	}

	if processed {
		return processed
	}

	if key == KeyA && (mods.Ctrl || mods.Cmd) && c.multiselect {
		c.SelectAll()
		c.form.Update()
		return true
	}

	if key == KeyArrowLeft {
		if c.currentCellX > 0 {
			c.navigateTo(c.currentCellY, c.currentCellX-1, mods.Shift)
			c.form.Update()
		}
		processed = true
	}

	if key == KeyArrowRight {
		if c.currentCellX < c.columnCount-1 {
			c.navigateTo(c.currentCellY, c.currentCellX+1, mods.Shift)
			c.form.Update()
		}
		processed = true
	}

	if key == KeyArrowUp {
		if c.currentCellY > 0 {
			selectRowIndex := c.currentCellY - 1
			if selectRowIndex >= c.rowCount {
				selectRowIndex = c.rowCount - 1
			}
			c.navigateTo(selectRowIndex, c.currentCellX, mods.Shift)
			c.form.Update()
		}
		processed = true
	}

	if key == KeyArrowDown {
		if c.currentCellY < c.rowCount-1 {
			selectRowIndex := c.currentCellY + 1
			if selectRowIndex >= c.rowCount {
				selectRowIndex = c.rowCount - 1
			}
			c.navigateTo(selectRowIndex, c.currentCellX, mods.Shift)
			c.form.Update()
		}
		processed = true
	}

	if key == KeyHome {
		c.navigateTo(0, c.currentCellX, mods.Shift)
		c.form.Update()
		processed = true
	}

	if key == KeyEnd {
		c.navigateTo(c.rowCount-1, c.currentCellX, mods.Shift)
		c.form.Update()
		processed = true
	}

	if (key == KeyEnter || key == KeyF2) && c.editKey(key) {
		c.EditCurrentCell("")
		c.form.Update()
		processed = true
	}

	if key == KeyPageUp {
		pageSizeInRows := c.pageSizeInRows()
		targetRow := c.currentCellY - pageSizeInRows
		if targetRow < 0 {
			targetRow = 0
		}
		if targetRow != c.currentCellY {
			c.navigateTo(targetRow, c.currentCellX, mods.Shift)
			c.form.Update()
		}
		processed = true
	}

	if key == KeyPageDown {
		pageSizeInRows := c.pageSizeInRows()
		targetRow := c.currentCellY + pageSizeInRows
		if targetRow >= c.rowCount {
			targetRow = c.rowCount - 1
		}
		if targetRow != c.currentCellY {
			c.navigateTo(targetRow, c.currentCellX, mods.Shift)
			c.form.Update()
		}
		processed = true
	}

	return processed
}

func (c *Table) onKeyUp(key Key, mods KeyModifiers) bool {
	return true
}

func (c *Table) onMouseMoveHeader(x int, y int, _ KeyModifiers) MouseCursor {
	if c.columnResizingIndex >= 0 {
		if c.columnResizingIndex < 0 || c.columnResizingIndex >= c.columnCount {
			return MouseCursorResizeHor
		}
		/*colInfo, exists := c.cols[c.columnResizingIndex]
		if !exists {
			colInfo = &tableColumn{name: "", width: c.defaultColumnWidth}
			c.cols[c.columnResizingIndex] = colInfo
		}*/

		colWidth := c.columnWidth(c.columnResizingIndex)

		newWidth := x - c.columnOffset(c.columnResizingIndex)
		if newWidth < 50 {
			newWidth = 50
		}
		if newWidth != colWidth {
			/*c.columnsWidths[c.columnResizingIndex] = newWidth
			c.updateInnerSize()
			c.updateInnerWidgetsLayout()*/
			c.SetColumnWidth(c.columnResizingIndex, newWidth)
			c.SetMouseCursor(MouseCursorResizeHor)
			c.form.Update()
		}
		return MouseCursorResizeHor
	}

	headerColumnBorder := c.headerColumnBorderByPosition(x, y)
	if headerColumnBorder >= 0 {
		return MouseCursorResizeHor
	}
	if c.onColumnClick != nil {
		return MouseCursorPointer
	}
	return MouseCursorArrow
}

func (c *Table) onMouseMove(x int, y int, mods KeyModifiers) bool {
	c.SetMouseCursor(MouseCursorArrow)

	if c.selectionDragging && !c.selectionDragStarted {
		// The hand shakes a bit while clicking: near a cell's edge that
		// would already extend the selection to the neighbor
		dx, dy := x-c.selectionDragStartX, y-c.selectionDragStartY
		c.selectionDragStarted = dx*dx+dy*dy > tableDragThreshold*tableDragThreshold
	}
	if c.hotTracking && !c.selectionDragging {
		col, row := c.cellByPosition(x, y)
		if row >= 0 && col >= 0 && !c.getCellObj(row, col).selectionDisabled {
			// A hand over the items that can be chosen, as over links
			c.SetMouseCursor(MouseCursorPointer)
			if row != c.currentCellY || col != c.currentCellX {
				c.SetCurrentCell2(row, col)
			}
		}
	}
	if c.selectionDragging && c.selectionDragStarted {
		col, row := c.cellByPositionClamped(x, y)
		if row >= 0 && col >= 0 {
			cellObj := c.getCellObj(row, col)
			if !cellObj.selectionDisabled {
				if c.multiselect {
					c.applyRangeSelection(c.selectionAnchorRow, c.selectionAnchorCol, row, col, c.selectionDragBaseRows, c.selectionDragBaseCells)
					c.moveCurrentCellForSelection(row, col)
				} else {
					// No multiselect: the single selection just follows the mouse.
					c.SetCurrentCell2(row, col)
				}
			}
		}
	}

	return true
}

func (c *Table) SetShowSelection(show bool) {
	c.showSelection = show
}

func (c *Table) ShowSelection() bool {
	return c.showSelection
}

// SetMultiselect enables or disables selecting more than one row/cell at a
// time (drag, Shift+click, Ctrl+click, Ctrl+A). Disabling it collapses any
// existing multi-selection down to the current cell/row.
func (c *Table) SetMultiselect(enabled bool) {
	if c.multiselect == enabled {
		return
	}
	c.multiselect = enabled
	if !enabled {
		c.selectionDragging = false
		c.syncSelectionToCurrent()
		c.form.Update()
	}
}

func (c *Table) Multiselect() bool {
	return c.multiselect
}

// SetClearSelectionOnEmptyClick controls whether a plain left click on the
// empty area of the table (outside any cell) clears the selection (default true).
func (c *Table) SetClearSelectionOnEmptyClick(enabled bool) {
	c.clearSelectionOnEmptyClick = enabled
}

func (c *Table) ClearSelectionOnEmptyClick() bool {
	return c.clearSelectionOnEmptyClick
}

// SetSelectingRows chooses the selection unit: true selects whole rows,
// false selects individual cells.
func (c *Table) SetSelectingRows(selectingRows bool) {
	if c.selectingRows == selectingRows {
		return
	}
	c.selectingRows = selectingRows
	c.selectionDragging = false
	c.syncSelectionToCurrent()
	c.form.Update()
}

func (c *Table) SelectingRows() bool {
	return c.selectingRows
}

func (c *Table) draw(cnv *Canvas) {
	if c.modeLoading {
		cnv.SetFontFamily(c.FontFamily())
		cnv.SetFontSize(c.FontSize())
		cnv.SetColor(c.ForegroundColor())
		cnv.SetHAlign(HAlignCenter)
		cnv.SetVAlign(VAlignCenter)
		cnv.DrawText(c.scrollX, c.scrollY, c.ViewportWidth(), c.ViewportHeight(), c.modeLoadingText)
		return
	}

	yOffset := 0
	yOffset += c.headerHeight()

	visibleRow1, visibleRow2 := c.visibleRows()
	//fmt.Println("Visible rows:", visibleRow1, visibleRow2)

	yOffset += visibleRow1 * c.rowHeight1

	// Every other row a shade darker (lighter in the dark theme), unless
	// the table has its own background
	p := CurrentPalette()
	rowBackColor := colorToRGBA(c.BackgroundColor())
	altRowBackColor := rowBackColor
	if c.backgroundColor == nil {
		altRowBackColor = MixColors(rowBackColor, p.Text, 0.035)
	}

	for rowIndex := visibleRow1; rowIndex < visibleRow2; rowIndex++ {
		rowObj1, rowExists := c.rows[rowIndex]
		{
			_ = rowExists
			for colIndex := 0; colIndex < c.columnCount; colIndex++ {
				var cellObj *tableCell
				var cellExists bool
				if rowObj1 != nil {
					cellObj, cellExists = rowObj1.cells[colIndex]
				}
				{
					_ = cellExists
					x := c.columnOffset(colIndex)
					y := yOffset

					columnWidth := c.columnWidth(colIndex)

					rowIsSelected := c.IsRowSelected(rowIndex)
					cellIsSelected := c.IsCellSelected(rowIndex, colIndex)

					selected := c.showSelection && (rowIsSelected || cellIsSelected)
					var backColor color.Color = rowBackColor
					if rowIndex%2 == 1 {
						backColor = altRowBackColor
					}
					if selected {
						backColor = c.GetPropColor("background_selected_cell", ColorToHex(p.Highlight))
					}
					cnv.FillRect(x, y, columnWidth, c.rowHeight1, backColor)

					hAlign := HAlignLeft
					vAlign := VAlignCenter

					var customCellDrawFunc func(cnv *Canvas)

					if cellObj != nil {
						if cellObj.onDraw != nil {
							customCellDrawFunc = cellObj.onDraw
						}
					}

					if customCellDrawFunc != nil {
						cnv.Save()
						cnv.TranslateAndClip(x, y, columnWidth, c.rowHeight1)
						customCellDrawFunc(cnv)
						cnv.Restore()
						continue
					}

					cellText := ""
					if cellObj != nil {
						cellText = cellObj.text
						if cellObj.displayTextExists {
							cellText = cellObj.displayText
						}
						hAlign = cellObj.hAlign
						vAlign = cellObj.vAlign
					}

					imgWidth := 0
					imgHeight := c.rowHeight1 - c.cellPadding*2
					if cellObj != nil && cellObj.image != nil {
						imgWidth = cellObj.imageWidth
						b := cellObj.image.Bounds()
						aspRatioImg := float64(b.Max.X) / float64(b.Max.Y)
						aspRationWidget := float64(imgWidth) / float64(imgHeight)
						if aspRatioImg > aspRationWidget {
							image := resize.Resize(uint(imgWidth), 0, cellObj.image, resize.Bicubic)
							b := image.Bounds()
							offsetX := (imgWidth-b.Max.X)/2 + x
							offsetY := (imgHeight-b.Max.Y)/2 + y
							cnv.DrawImage(offsetX+c.cellPadding, offsetY+c.cellPadding, image)
						} else {
							image := resize.Resize(0, uint(imgHeight), cellObj.image, resize.Bicubic)
							b := image.Bounds()
							offsetX := (imgWidth-b.Max.X)/2 + x
							offsetY := (imgHeight-b.Max.Y)/2 + y
							cnv.DrawImage(offsetX+c.cellPadding, offsetY+c.cellPadding, image)
						}

						imgWidth += c.cellPadding
					}

					cnv.SetHAlign(hAlign)
					cnv.SetVAlign(vAlign)
					cnv.SetFontFamily(c.FontFamily())
					cnv.SetFontSize(c.FontSize())
					col := c.ForegroundColor()
					if selected {
						col = p.HighlightedText
					}
					if cellObj != nil {
						if cellObj.color != nil {
							col = cellObj.color
						}
					}
					cnv.SetColor(col)

					drawContractionDots := false
					if cellObj != nil && cellObj.contraction {
						textW, textH, err := MeasureText(c.FontFamily(), c.FontSize(), cellText)
						if err == nil {
							_ = textH
							if textW > columnWidth-c.cellPadding*2-imgWidth {
								drawContractionDots = true
							}
						}
					}

					_ = cellText

					if drawContractionDots {
						cnv.SetHAlign(HAlignLeft)
					}

					cnv.DrawText(x+c.cellPadding+imgWidth, y+c.cellPadding, columnWidth-c.cellPadding*2-imgWidth, c.rowHeight1-c.cellPadding*2, cellText)
					if drawContractionDots {
						dots := ".."
						dotsW, _, err := MeasureText(c.FontFamily(), c.FontSize(), dots)
						if err == nil {
							// fill background on dots place to hide last part of text
							var backColorDots color.RGBA
							{
								r, g, b, a := backColor.RGBA()
								backColorDots = color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
								if backColorDots.A > 0 {
									backColorDots.A = 240
								}
							}

							backPadding := 5

							cnv.FillRect(x+columnWidth-dotsW-c.cellPadding-backPadding, y+c.cellPadding, dotsW+backPadding, c.rowHeight1-c.cellPadding*2, backColorDots)
							cnv.DrawText(x+columnWidth-dotsW-c.cellPadding, y+c.cellPadding, dotsW, c.rowHeight1-c.cellPadding*2, dots)
						}
					}

				}
			}
		}

		yOffset += c.rowHeight1
	}

	// Draw cell borders
	if c.cellBorderWidth > 0 {
		cnv.Save()
		cnv.TranslateAndClip(c.scrollX, c.scrollY+c.headerHeight(), c.ViewportWidth(), c.ViewportHeight()-c.headerHeight())
		for rowIndex := visibleRow1; rowIndex < visibleRow2+1; rowIndex++ {
			x1 := 0
			y1 := rowIndex*c.rowHeight1 - c.scrollY
			x2 := c.contentWidth()
			y2 := y1
			cnv.DrawLine(x1, y1, x2, y2, c.cellBorderWidth, c.CellBorderColor())
		}

		for colIndex := 0; colIndex < c.columnCount+1; colIndex++ {
			x1 := c.columnOffset(colIndex) - c.scrollX
			if colIndex == c.columnCount {
				x1 = c.contentWidth() - c.scrollX // after the stretched column
			}
			y1 := visibleRow1*c.rowHeight1 - c.scrollY
			x2 := x1
			y2 := visibleRow2*c.rowHeight1 - c.scrollY
			cnv.DrawLine(x1, y1, x2, y2, c.cellBorderWidth, c.CellBorderColor())
		}
		cnv.Restore()
	}
}

func (c *Table) drawPost(cnv *Canvas) {
	// Draw header, in the view: not over the scroll bars
	cnv.Save()
	c.clipToViewport(cnv)
	for headerRowIndex := 0; headerRowIndex < c.headerRowsCount; headerRowIndex++ {
		for colIndex := 0; colIndex < c.columnCount; colIndex++ {
			needToDisplay := true
			if c.headerCellShadowed2(headerRowIndex, colIndex) {
				needToDisplay = false
			}

			if !needToDisplay {
				continue
			}

			headerCell := c.headerCell2(headerRowIndex, colIndex)

			cellSpanX := headerCell.SpanCol()
			cellSpanY := headerCell.SpanRow()

			headerRowOffset := c.headerRowOffset(headerRowIndex)
			//headerRowHeight := c.headerRowHeight(headerRowIndex)

			//cellWidth := c.columnWidth(colIndex)
			cellWidth := 0
			for i := 0; i < cellSpanX; i++ {
				cellWidth += c.columnWidth(colIndex + i)
			}

			cellHeight := 0
			for i := 0; i < cellSpanY; i++ {
				cellHeight += c.headerRowHeight(headerRowIndex + i)
			}
			//cellHeight := headerRowHeight

			x := c.columnOffset(colIndex)
			y := headerRowOffset + c.scrollY

			// Header Background
			cnv.FillRect(x, y, cellWidth, cellHeight, CurrentPalette().Button)

			imgWidth := 0
			imgHeight := c.rowHeight1 - c.cellPadding*2
			if headerCell != nil && headerCell.image != nil {
				imgWidth = headerCell.imageWidth
				b := headerCell.image.Bounds()
				aspRatioImg := float64(b.Max.X) / float64(b.Max.Y)
				aspRationWidget := float64(imgWidth) / float64(imgHeight)
				if aspRatioImg > aspRationWidget {
					image := resize.Resize(uint(imgWidth), 0, headerCell.image, resize.Bicubic)
					b := image.Bounds()
					offsetX := (imgWidth-b.Max.X)/2 + x
					offsetY := (imgHeight-b.Max.Y)/2 + y
					cnv.DrawImage(offsetX+c.cellPadding, offsetY+c.cellPadding, image)
				} else {
					image := resize.Resize(0, uint(imgHeight), headerCell.image, resize.Bicubic)
					b := image.Bounds()
					offsetX := (imgWidth-b.Max.X)/2 + x
					offsetY := (imgHeight-b.Max.Y)/2 + y
					cnv.DrawImage(offsetX+c.cellPadding, offsetY+c.cellPadding, image)
				}

				imgWidth += c.cellPadding
			}

			cnv.SetHAlign(headerCell.hAlign)
			cnv.SetVAlign(VAlignCenter)
			cnv.SetColor(CurrentPalette().ButtonText)
			cnv.SetFontFamily(c.FontFamily())
			cnv.SetFontSize(c.FontSize())
			cnv.DrawText(x+c.cellPadding+imgWidth, y+c.cellPadding, cellWidth-c.cellPadding*2-imgWidth, cellHeight-c.cellPadding*2, headerCell.name)

			cnv.SetColor(c.CellBorderColor())
			cnv.DrawRect(x, y, cellWidth+1, cellHeight+1)
		}
	}
	cnv.Restore()

	/*
		for colIndex := 0; colIndex < c.columnCount; colIndex++ {
			colObj, exists := c.cols[colIndex]
			if exists {
				x := c.columnOffset(colIndex)
				cnv.FillRect(x, c.scrollY, colObj.width, c.headerHeight(), color.RGBA{R: 70, G: 80, B: 90, A: 255})
				cnv.DrawTextMultiline(x+c.cellPadding, c.scrollY+c.cellPadding, colObj.width-c.cellPadding*2, c.headerHeight()-c.cellPadding*2, HAlignLeft, VAlignCenter, colObj.name, color.RGBA{R: 200, G: 200, B: 200, A: 255}, "robotomono", 16, false)
			}
		}*/

	/*for colIndex := 0; colIndex < c.columnCount+1; colIndex++ {
		x1 := c.columnOffset(colIndex)
		y1 := c.scrollY
		x2 := x1
		y2 := c.headerHeight() + c.scrollY
		cnv.DrawLine(x1, y1, x2, y2, c.cellBorderWidth, c.cellBorderColor)
	}*/

	// Draw table border
	cnv.SetColor(CurrentPalette().Border)
	cnv.DrawRect(c.scrollX, c.scrollY, c.Width(), c.Height())
}

// pageSizeInRows is how many rows PageUp and PageDown move by: the rows the
// view shows under the header
func (c *Table) pageSizeInRows() int {
	return max(1, (c.ViewportHeight()-c.headerHeight())/c.rowHeight1)
}

func (c *Table) visibleRows() (min int, max int) {
	min = c.scrollY / c.rowHeight1
	max = min + (c.ViewportHeight()-c.headerHeight())/c.rowHeight1
	min = min - 1
	max = max + 1
	if min < 0 {
		min = 0
	}
	if max > c.rowCount {
		max = c.rowCount
	}
	return
}

func (c *Table) columnWidth(col int) int {
	if col < 0 || col >= c.columnCount {
		return c.defaultColumnWidth
	}

	colWidth, exists := c.columnsWidths[col]
	if !exists {
		colWidth = c.defaultColumnWidth
	}

	if c.stretchLastColumn && col == c.columnCount-1 {
		colWidth = max(colWidth, c.visibleWidth()-c.columnOffset(col))
	}
	return colWidth
}

// visibleWidth is the width the cells can take: without the vertical scroll bar
func (c *Table) visibleWidth() int {
	return c.ViewportWidth()
}

// contentWidth is the width of all the columns, the stretched one included
func (c *Table) contentWidth() int {
	if c.columnCount == 0 {
		return 0
	}
	last := c.columnCount - 1
	return c.columnOffset(last) + c.columnWidth(last)
}

func (c *Table) columnOffset(col int) int {
	result := 0
	for i := 0; i < col; i++ {
		colWidth := c.defaultColumnWidth
		if colWidthValue, exists := c.columnsWidths[i]; exists {
			colWidth = colWidthValue
		}
		result += colWidth
	}
	return result
}

func (c *Table) updateInnerSize() {
	width := 0
	for i := 0; i < c.columnCount; i++ {
		colWidth := c.defaultColumnWidth
		if colWidthValue, exists := c.columnsWidths[i]; exists {
			colWidth = colWidthValue
		}
		width += colWidth
	}
	c.SetInnerSize(width, c.headerHeight()+c.rowCount*c.rowHeight1)
	c.checkScrolls()
}

func (c *Table) headerColumnBorderByPosition(x int, y int) int {
	if y < c.scrollY || y >= c.scrollY+c.headerHeight() {
		return -1
	}
	if x < 0 {
		return -1
	}
	if x >= c.contentWidth() {
		return -1
	}
	for col := 0; col < c.columnCount; col++ {
		// The stretched column ends at the edge of the table: nothing to drag
		if c.stretchLastColumn && col == c.columnCount-1 {
			break
		}
		colOffset := c.columnOffset(col)
		colWidth := c.columnWidth(col)
		rigthBorder := colOffset + colWidth
		if x >= rigthBorder-5 && x < rigthBorder+5 {
			return col
		}
	}
	return -1
}

func (c *Table) headerColumnByPosition(x int, y int) int {
	if y < c.scrollY || y >= c.scrollY+c.headerHeight() {
		return -1
	}
	if x < 0 {
		return -1
	}
	if x >= c.contentWidth() {
		return -1
	}
	for col := 0; col < c.columnCount; col++ {
		colOffset := c.columnOffset(col)
		colWidth := c.columnWidth(col)
		if x >= colOffset && x < colOffset+colWidth {
			return col
		}
	}
	return -1
}

func (c *Table) cellByPosition(x, y int) (row int, col int) {
	col = 0
	for col < c.columnCount {
		colOffset := c.columnOffset(col)
		colWidth := c.columnWidth(col)
		if x >= colOffset && x < colOffset+colWidth {
			break
		}
		col++
	}
	if col >= c.columnCount {
		return -1, -1
	}
	// The grid line drawn at the top of a row looks like the bottom edge of
	// the row above, so it belongs to that one; the line under the header
	// stays with the first row
	y -= c.headerHeight()
	if y < 0 {
		return -1, -1
	}
	row = max(y-c.cellBorderWidth, 0) / c.rowHeight1
	if row >= c.rowCount {
		return -1, -1
	}
	return col, row
}

// cellByPositionClamped is like cellByPosition but clamps x/y into the
// table's content area first, so a point beyond the last row/column (e.g.
// while dragging a selection past the table's edge) still resolves to the
// nearest valid cell instead of (-1, -1).
func (c *Table) cellByPositionClamped(x, y int) (col int, row int) {
	if c.columnCount <= 0 || c.rowCount <= 0 {
		return -1, -1
	}

	if x < 0 {
		x = 0
	}
	maxX := c.columnOffset(c.columnCount) - 1
	if x > maxX {
		x = maxX
	}

	minY := c.headerHeight()
	if y < minY {
		y = minY
	}
	maxY := c.headerHeight() + c.rowCount*c.rowHeight1 - 1
	if y > maxY {
		y = maxY
	}

	return c.cellByPosition(x, y)
}

func (c *Table) headerRowOffset(headerRowIndex int) int {
	result := 0
	for i := 0; i < headerRowIndex; i++ {
		rowHeight := c.headerRowHeight(i)
		result += rowHeight
	}
	return result
}

func (c *Table) headerRowHeight(headerRowIndex int) int {
	result := c.rowHeight1
	if headerRowHeight, exists := c.headerRowHeights[headerRowIndex]; exists {
		result = headerRowHeight
	}
	return result
}

func (c *Table) headerHeight() int {
	result := 0
	for i := 0; i < c.headerRowsCount; i++ {
		rowHeight := c.headerRowHeight(i)
		result += rowHeight
	}
	return result
}

func (c *Table) EditCurrentCell(enteredText string) {
	if c.CurrentRow() < 0 || c.CurrentColumn() < 0 {
		return
	}

	if c.editorTextBox != nil {
		c.removeInnerWidget(c.editorTextBox)
		c.editorTextBox = nil
	}

	c.editorTextBox = NewTextBox()

	// The edit goes to the cell the editor is opened on, even if the current
	// cell changes meanwhile (e.g. code restoring the selection after a
	// refresh). It used to go to the current cell at the time of the commit.
	row, col := c.currentCellY, c.currentCellX
	commit := func(text string) {
		if row < c.rowCount && col < c.columnCount { // the cell may be gone
			c.SetCellText2(row, col, text)
		}
	}

	if len(enteredText) == 0 {
		enteredText = c.GetCellText2(row, col)
	}

	c.editorTextBox.SetText(enteredText)
	c.editorTextBox.MoveCursorToEnd()
	c.editorTextBox.SelectAllText()
	c.editorTextBox.SetOnTextBoxKeyDown(func() {
		ev := CurrentEvent().Parameter.(*EventTextboxKeyDown)
		if ev.Key == KeyEnter {
			if c.editorTextBox != nil {
				commit(c.editorTextBox.Text())
				c.removeInnerWidget(c.editorTextBox)
				c.editorTextBox = nil
				c.form.Update()
				c.Focus()
			}
			ev.Processed = true
			return
		}
		if ev.Key == KeyEsc {
			if c.editorTextBox != nil {
				c.removeInnerWidget(c.editorTextBox)
				c.editorTextBox = nil
				c.form.Update()
				c.Focus()
			}
			ev.Processed = true
			return
		}
	})
	c.editorTextBox.SetOnFocusLost(func() {
		if c.editorTextBox != nil {
			commit(c.editorTextBox.Text())
			c.removeInnerWidget(c.editorTextBox)
			c.editorTextBox = nil
			c.form.Update()
			c.Focus()
		}
	})
	c.AddWidgetOnTable(c.editorTextBox, row, col, 1, 1)
	c.editorTextBox.Focus()
}

// EditCell opens a one-line editor over the cells [col, col+colSpan) of the
// row, with the text in it (in-place renaming of file managers). onDone gets
// the text: accepted is true on Enter and when the editor loses the focus,
// false on Esc; the cell itself is not changed. Returns the editor (to select
// a part of the text, for example), nil if the cell is out of the table.
func (c *Table) EditCell(row, col, colSpan int, text string, onDone func(text string, accepted bool)) *TextBox {
	if row < 0 || row >= c.rowCount || col < 0 || colSpan < 1 || col+colSpan > c.columnCount {
		return nil
	}
	c.CancelCellEditor()

	editor := NewTextBox()
	editor.SetText(text)
	editor.MoveCursorToEnd()
	editor.SelectAllText()
	done := false
	// refocus: back to the table on Enter and Esc; not when the focus went elsewhere
	finish := func(accepted, refocus bool) {
		if done {
			return
		}
		done = true
		value := editor.Text()
		c.removeInnerWidget(editor)
		if c.cellEditor == editor {
			c.cellEditor = nil
		}
		c.form.Update()
		if refocus {
			c.Focus()
		}
		if onDone != nil {
			onDone(value, accepted)
		}
	}
	editor.SetOnTextBoxKeyDown(func() {
		ev := CurrentEvent().Parameter.(*EventTextboxKeyDown)
		switch ev.Key {
		case KeyEnter:
			finish(true, true)
			ev.Processed = true
		case KeyEsc:
			finish(false, true)
			ev.Processed = true
		}
	})
	editor.SetOnFocusLost(func() { finish(true, false) })
	c.cellEditorCancel = func() { finish(false, c.form.FocusedWidget() == editor) }
	c.cellEditor = editor

	c.ScrollEnsureVisible(c.columnOffset(col), c.rowOffset(row)+c.rowHeight1)
	c.AddWidgetOnTable(editor, row, col, colSpan, 1)
	editor.Focus()
	return editor
}

// CancelCellEditor closes the editor of EditCell, if one is open, as Esc does
func (c *Table) CancelCellEditor() {
	if c.cellEditor != nil && c.cellEditorCancel != nil {
		c.cellEditorCancel()
	}
}

// IsEditingCell tells whether the editor of EditCell is open
func (c *Table) IsEditingCell() bool {
	return c.cellEditor != nil
}

func (c *Table) removeInnerWidget(w Widgeter) {
	for i, in := range c.innerWidgets {
		if in.widget == w {
			c.innerWidgets = append(c.innerWidgets[:i], c.innerWidgets[i+1:]...)
			break
		}
	}
	c.RemoveWidget(w)
}

func (c *Table) CopySelectionToClipboard() {
	if c.currentCellY < 0 || c.currentCellY >= c.rowCount || c.currentCellX < 0 || c.currentCellX >= c.columnCount {
		return
	}

	text := c.GetCellText2(c.currentCellY, c.currentCellX)
	if len(text) == 0 {
		return
	}
	ClipboardSetText(text)
}

type tableHeader struct {
	Widget

	OnHeaderMouseDown func(button MouseButton, x int, y int, mods KeyModifiers) bool
	OnHeaderMouseUp   func(button MouseButton, x int, y int, mods KeyModifiers) bool
	OnHeaderMouseMove func(x int, y int, mods KeyModifiers) MouseCursor
}

func newTableHeader() *tableHeader {
	var c tableHeader
	c.InitWidget()
	c.SetOnMouseDown(c.onMouseDown)
	c.SetOnMouseUp(c.onMouseUp)
	c.SetOnMouseMove(c.onMouseMove)
	//c.SetBackgroundColor(color.RGBA{R: 240, G: 240, B: 240, A: 100})
	return &c
}

func (c *tableHeader) onMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if c.OnHeaderMouseDown != nil {
		if c.OnHeaderMouseDown(button, x, y, mods) {
			return true
		}
	}
	return true
}

func (c *tableHeader) onMouseUp(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if c.OnHeaderMouseUp != nil {
		if c.OnHeaderMouseUp(button, x, y, mods) {
			return true
		}
	}
	return true
}

func (c *tableHeader) onMouseMove(x int, y int, mods KeyModifiers) bool {
	if c.OnHeaderMouseMove != nil {
		cursor := c.OnHeaderMouseMove(x, y, mods)
		c.SetMouseCursor(cursor)
	}
	return true
}

func (c *Table) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	if !c.customRowHeight {
		c.rowHeight1 = ThemeRowHeight()
		c.updateInnerSize()
	}
}

// editKey reports whether the key (Enter or F2) opens the editor of the
// current cell: by the trigger of the table or of the cell
func (c *Table) editKey(key Key) bool {
	cellObj := c.getCellObj(c.currentCellY, c.currentCellX)
	switch key {
	case KeyEnter:
		return c.editTriggerEnter || cellObj != nil && cellObj.editTriggerEnter
	case KeyF2:
		return c.editTriggerF2 || cellObj != nil && cellObj.editTriggerF2
	}
	return false
}

// handlesKey: Enter and F2 that open the editor of the current cell go to
// the table before the shortcuts (see Form.AddShortcut)
func (c *Table) handlesKey(key Key, mods KeyModifiers) bool {
	if mods.Ctrl || mods.Alt || mods.Shift || mods.Cmd || c.editorTextBox != nil {
		return false
	}
	return c.CurrentRow() >= 0 && c.CurrentColumn() >= 0 && c.editKey(key)
}
