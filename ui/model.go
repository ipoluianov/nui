package ui

import (
	"image"
	"image/color"
	"slices"
)

// Models: the data a ListView or a Table shows, asked for when it is
// drawn. A view of a model holds none of the data itself: a million rows
// cost what the visible ones do.
//
//	logs := ui.NewFuncModel(func() int { return len(entries) }, 2,
//		func(row, col int) string { return entries[row].Field(col) })
//	table.SetModel(logs)
//	...
//	entries = append(entries, e)
//	logs.Changed() // the views show the new row
//
// A model can tell more with the optional interfaces: ItemStyler (the
// look of a cell), ItemEditor (the cells the user edits), ColumnNamer (the
// column headers).

// ItemModel is a table of texts: rows of cells. A list is a model of one
// column.
type ItemModel interface {
	RowCount() int
	ColumnCount() int
	Text(row, col int) string
}

// ItemStyle is how a cell looks; the zero value is the view's default
type ItemStyle struct {
	Color  color.Color // the text color
	Bold   bool
	Italic bool
	HAlign HAlign
	Icon   image.Image // drawn before the text, scaled to the row height
}

// ItemStyler is a model that styles its cells
type ItemStyler interface {
	Style(row, col int) ItemStyle
}

// ItemEditor is a model whose cells the user may edit (Table): SetText
// stores the text, or returns false to reject it
type ItemEditor interface {
	SetText(row, col int, text string) bool
}

// ColumnNamer is a model that names its columns (the header of a Table)
type ColumnNamer interface {
	ColumnName(col int) string
}

// ModelNotifier is a model that tells its views it changed: they connect
// to the signal returned
type ModelNotifier interface {
	ModelChanged() *Signal[struct{}]
}

// ModelSignal is the change notification for the models: embed it and call
// Changed after changing the data
type ModelSignal struct {
	changed Signal[struct{}]
}

// ModelChanged returns the signal the views connect to
func (m *ModelSignal) ModelChanged() *Signal[struct{}] {
	return &m.changed
}

// Changed tells the views the data changed: they show it again
func (m *ModelSignal) Changed() {
	m.changed.Emit(struct{}{})
}

// FuncModel is a model of functions: the data stays where the application
// keeps it
type FuncModel struct {
	ModelSignal
	rows    func() int
	columns int
	text    func(row, col int) string
}

// NewFuncModel makes a model of columns columns, whose number of rows and
// texts the functions return
func NewFuncModel(rows func() int, columns int, text func(row, col int) string) *FuncModel {
	return &FuncModel{rows: rows, columns: columns, text: text}
}

func (m *FuncModel) RowCount() int            { return m.rows() }
func (m *FuncModel) ColumnCount() int         { return m.columns }
func (m *FuncModel) Text(row, col int) string { return m.text(row, col) }

// ListModel is a list of texts, with their data
type ListModel struct {
	ModelSignal
	items []listItem
}

type listItem struct {
	text string
	data any
}

// NewListModel makes a list of the texts
func NewListModel(texts ...string) *ListModel {
	m := &ListModel{}
	m.SetItems(texts)
	return m
}

func (m *ListModel) RowCount() int    { return len(m.items) }
func (m *ListModel) ColumnCount() int { return 1 }

func (m *ListModel) Text(row, col int) string {
	if row < 0 || row >= len(m.items) {
		return ""
	}
	return m.items[row].text
}

// Data returns the data of the item, nil if none
func (m *ListModel) Data(row int) any {
	if row < 0 || row >= len(m.items) {
		return nil
	}
	return m.items[row].data
}

// SetItems replaces the items with the texts
func (m *ListModel) SetItems(texts []string) {
	m.items = make([]listItem, len(texts))
	for i, text := range texts {
		m.items[i].text = text
	}
	m.Changed()
}

// Append adds an item at the end
func (m *ListModel) Append(text string, data any) {
	m.items = append(m.items, listItem{text, data})
	m.Changed()
}

// Insert puts an item before the item row
func (m *ListModel) Insert(row int, text string, data any) {
	row = max(0, min(row, len(m.items)))
	m.items = slices.Insert(m.items, row, listItem{text, data})
	m.Changed()
}

// Remove removes the item
func (m *ListModel) Remove(row int) {
	if row < 0 || row >= len(m.items) {
		return
	}
	m.items = slices.Delete(m.items, row, row+1)
	m.Changed()
}

// Set changes the text of the item
func (m *ListModel) Set(row int, text string) {
	if row < 0 || row >= len(m.items) {
		return
	}
	m.items[row].text = text
	m.Changed()
}
