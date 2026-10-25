package ui

import (
	"image"
	"image/color"
	"slices"

	xdraw "golang.org/x/image/draw"
)

// ListView shows the rows of a model (its first column) in a scrolled list,
// drawing only the visible rows: a list of a million items is as fast as
// one of ten. The rows are one height. The user selects with the mouse and
// the keys (Ctrl and Shift with SetMultiselect), activates with a double
// click or Enter.
//
//	list := ui.NewListView()
//	list.SetItems([]string{"one", "two", "three"}) // or SetModel
//	list.OnActivated.Connect(func(row int) { open(row) })
type ListView struct {
	Widget

	// OnSelectionChanged: the current row or the selection changed, by the
	// user or by the code; the current row (-1 if none)
	OnSelectionChanged Signal[int]
	// OnActivated: the user double-clicked a row or pressed Enter on it
	OnActivated Signal[int]

	model       ItemModel
	modelConn   Connection
	rowHeight   int
	current     int
	anchor      int // where a Shift selection starts
	selected    map[int]bool
	multiselect bool
	emptyText   string
	painter     func(p *ItemPaint)
	icons       map[image.Image]*image.RGBA // icons scaled to the row height
}

// ItemPaint is a row of a ListView to paint by the function of
// SetItemPainter: the canvas is clipped to the row, its origin at the row's
// top-left corner
type ItemPaint struct {
	Canvas        *Canvas
	Row           int
	Width, Height int
	Selected      bool
	Current       bool // the row of the keyboard cursor
	Focused       bool // the list has the focus
}

// NewListView makes an empty list
func NewListView() *ListView {
	var c ListView
	c.InitWidget()
	c.SetTypeName("ListView")
	c.current = -1
	c.anchor = -1
	c.selected = map[int]bool{}
	c.rowHeight = ThemeControlHeight()
	c.SetCanBeFocused(true)
	c.SetAllowScroll(false, true)
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	c.SetMinSize(100, 60)
	c.scrollBarInset = 1
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.mouseDown)
	c.SetOnMouseDblClick(c.mouseDoubleClick)
	c.SetOnKeyDown(c.keyDown)
	c.SetModel(NewListModel())
	return &c
}

// SetModel shows the model; a ModelNotifier is followed
func (c *ListView) SetModel(model ItemModel) {
	c.modelConn.Disconnect()
	c.model = model
	if n, ok := model.(ModelNotifier); ok {
		c.modelConn = n.ModelChanged().Connect(func(struct{}) { c.Refresh() })
	}
	c.Refresh()
}

// Model returns the model shown
func (c *ListView) Model() ItemModel {
	return c.model
}

// SetItems shows the texts (a ListModel of them)
func (c *ListView) SetItems(texts []string) {
	c.SetModel(NewListModel(texts...))
}

// Refresh shows the model again, after it changed (a ModelNotifier does it
// by itself): the rows past its end leave the selection
func (c *ListView) Refresh() {
	n := c.RowCount()
	changed := false
	for row := range c.selected {
		if row >= n {
			delete(c.selected, row)
			changed = true
		}
	}
	if c.current >= n {
		c.current = n - 1
		changed = true
	}
	c.updateInnerSize()
	c.form.Update()
	if changed {
		c.OnSelectionChanged.Emit(c.current)
	}
}

// RowCount returns the number of the rows of the model
func (c *ListView) RowCount() int {
	if c.model == nil {
		return 0
	}
	return c.model.RowCount()
}

// SetRowHeight sets the height of the rows
func (c *ListView) SetRowHeight(height int) {
	c.rowHeight = max(1, height)
	c.icons = nil
	c.updateInnerSize()
	c.form.Update()
}

func (c *ListView) RowHeight() int {
	return c.rowHeight
}

// SetMultiselect lets the user select several rows: Ctrl+click adds or
// removes one, Shift+click and Shift+keys select a range, Ctrl+A all
func (c *ListView) SetMultiselect(multiselect bool) {
	c.multiselect = multiselect
}

// SetEmptyText sets the text shown in an empty list, e.g. "No results"
func (c *ListView) SetEmptyText(text string) {
	c.emptyText = text
	c.form.Update()
}

// SetItemPainter paints the rows with the function instead of the text of
// the model: a row of its own design (an icon, two lines, a badge)
func (c *ListView) SetItemPainter(paint func(p *ItemPaint)) {
	c.painter = paint
	c.form.Update()
}

// CurrentRow returns the row of the keyboard cursor, -1 if none
func (c *ListView) CurrentRow() int {
	return c.current
}

// SetCurrentRow makes the row current and the only selected one, and
// scrolls it into view; -1 clears the selection
func (c *ListView) SetCurrentRow(row int) {
	c.selectRow(row, KeyModifiers{})
}

// SelectedRows returns the selected rows, in order
func (c *ListView) SelectedRows() []int {
	rows := make([]int, 0, len(c.selected))
	for row := range c.selected {
		rows = append(rows, row)
	}
	slices.Sort(rows)
	return rows
}

// IsRowSelected reports whether the row is selected
func (c *ListView) IsRowSelected(row int) bool {
	return c.selected[row]
}

// SelectAll selects all the rows (with SetMultiselect)
func (c *ListView) SelectAll() {
	if !c.multiselect {
		return
	}
	for row := range c.RowCount() {
		c.selected[row] = true
	}
	c.form.Update()
	c.OnSelectionChanged.Emit(c.current)
}

// EnsureVisible scrolls the row into view
func (c *ListView) EnsureVisible(row int) {
	if row < 0 {
		return
	}
	c.ScrollEnsureVisible(0, row*c.rowHeight)
	c.ScrollEnsureVisible(0, (row+1)*c.rowHeight-1)
	c.form.Update()
}

// ItemRect returns where the row is, in the widget's coordinates: scrolled,
// possibly out of the view
func (c *ListView) ItemRect(row int) (x, y, width, height int) {
	vw, _ := c.viewportSize()
	return 0, row*c.rowHeight - c.scrollY, vw, c.rowHeight
}

// RowAt returns the row at the point of the widget, -1 if none
func (c *ListView) RowAt(x, y int) int {
	return c.rowAtContent(y + c.scrollY)
}

func (c *ListView) rowAtContent(y int) int {
	if y < 0 {
		return -1
	}
	row := y / c.rowHeight
	if row >= c.RowCount() {
		return -1
	}
	return row
}

func (c *ListView) updateInnerSize() {
	vw, _ := c.viewportSize()
	c.innerWidth = vw
	c.innerHeight = c.RowCount() * c.rowHeight
	c.checkScrolls()
}

// selectRow makes the row current: alone selected, or with Ctrl toggled,
// or with Shift the range from the anchor (with multiselect)
func (c *ListView) selectRow(row int, mods KeyModifiers) {
	n := c.RowCount()
	if row >= n {
		row = n - 1
	}
	if row < 0 {
		changed := c.current != -1 || len(c.selected) > 0
		c.current, c.anchor = -1, -1
		clear(c.selected)
		if changed {
			c.form.Update()
			c.OnSelectionChanged.Emit(-1)
		}
		return
	}
	toggle := c.multiselect && (mods.Ctrl || mods.Cmd)
	extend := c.multiselect && mods.Shift && c.anchor >= 0
	switch {
	case extend:
		clear(c.selected)
		for r := min(c.anchor, row); r <= max(c.anchor, row); r++ {
			c.selected[r] = true
		}
	case toggle:
		if c.selected[row] {
			delete(c.selected, row)
		} else {
			c.selected[row] = true
		}
		c.anchor = row
	default:
		clear(c.selected)
		c.selected[row] = true
		c.anchor = row
	}
	c.current = row
	c.EnsureVisible(row)
	c.form.Update()
	c.OnSelectionChanged.Emit(row)
}

func (c *ListView) mouseDown(button MouseButton, x, y int, mods KeyModifiers) bool {
	if !c.Enabled() {
		return false
	}
	c.Focus()
	row := c.rowAtContent(y)
	if row < 0 {
		return true
	}
	if button == MouseButtonRight && c.selected[row] {
		return false // the context menu acts on the selection
	}
	c.selectRow(row, mods)
	return button == MouseButtonLeft
}

func (c *ListView) mouseDoubleClick(button MouseButton, x, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft || !c.Enabled() {
		return false
	}
	if row := c.rowAtContent(y); row >= 0 {
		c.OnActivated.Emit(row)
	}
	return true
}

func (c *ListView) keyDown(key Key, mods KeyModifiers) bool {
	n := c.RowCount()
	if n == 0 || !c.Enabled() {
		return false
	}
	_, vh := c.viewportSize()
	page := max(1, vh/c.rowHeight-1)
	moveMods := KeyModifiers{Shift: mods.Shift}
	switch key {
	case KeyArrowUp:
		c.selectRow(max(0, c.current-1), moveMods)
	case KeyArrowDown:
		c.selectRow(min(n-1, c.current+1), moveMods)
	case KeyPageUp:
		c.selectRow(max(0, c.current-page), moveMods)
	case KeyPageDown:
		c.selectRow(min(n-1, c.current+page), moveMods)
	case KeyHome:
		c.selectRow(0, moveMods)
	case KeyEnd:
		c.selectRow(n-1, moveMods)
	case KeyEnter:
		if c.current >= 0 {
			c.OnActivated.Emit(c.current)
		}
	case KeyA:
		if !(mods.Ctrl || mods.Cmd) || !c.multiselect {
			return false
		}
		c.SelectAll()
	default:
		return false
	}
	return true
}

// ProcessPaint draws the frame on the whole widget, under the rows and the
// scroll bar
func (c *ListView) ProcessPaint(cnv *Canvas) {
	fill, border := inputFrameColors(&c.Widget)
	cnv.FillFrame(0, 0, c.Width(), c.Height(), themeControlRadius, fill, border)
	c.Widget.ProcessPaint(cnv)
}

func (c *ListView) draw(cnv *Canvas) {
	p := CurrentPalette()
	vw, vh := c.viewportSize()
	n := c.RowCount()
	if n == 0 {
		if c.emptyText != "" {
			cnv.SetColor(p.PlaceholderText)
			cnv.SetFontFamily(c.FontFamily())
			cnv.SetFontSize(c.FontSize())
			cnv.SetHAlign(HAlignCenter)
			cnv.SetVAlign(VAlignCenter)
			cnv.DrawText(0, c.scrollY, vw, vh, c.emptyText)
		}
		return
	}
	first := max(0, c.scrollY/c.rowHeight)
	last := min(n-1, (c.scrollY+vh)/c.rowHeight)
	focused := c.IsFocused()
	styler, _ := c.model.(ItemStyler)
	for row := first; row <= last; row++ {
		y := row * c.rowHeight
		selected := c.selected[row]
		if c.painter != nil {
			cnv.Save()
			cnv.TranslateAndClip(0, y, vw, c.rowHeight)
			c.painter(&ItemPaint{Canvas: cnv, Row: row, Width: vw, Height: c.rowHeight,
				Selected: selected, Current: row == c.current, Focused: focused})
			cnv.Restore()
			continue
		}
		textColor := color.Color(p.Text)
		if selected {
			back := p.Selection
			if focused {
				back, textColor = p.Highlight, p.HighlightedText
			}
			cnv.FillRect(1, y, vw-2, c.rowHeight, back)
		}
		var style ItemStyle
		if styler != nil {
			style = styler.Style(row, 0)
		}
		if style.Color != nil && !(selected && focused) {
			textColor = style.Color
		}
		x := themeTextInset
		if style.Icon != nil {
			size := c.rowHeight - 4
			cnv.DrawImage(x, y+2, c.scaledIcon(style.Icon, size))
			x += size + themeTextInset
		}
		family := c.FontFamily()
		if style.Bold || style.Italic {
			family = StyledFont(family, fontStyleOf(style.Bold, style.Italic))
		}
		cnv.SetColor(textColor)
		cnv.SetFontFamily(family)
		cnv.SetFontSize(c.FontSize())
		cnv.SetHAlign(HAlignLeft)
		if style.HAlign != 0 {
			cnv.SetHAlign(style.HAlign)
		}
		cnv.SetVAlign(VAlignCenter)
		cnv.DrawText(x, y, vw-x-themeTextInset, c.rowHeight, c.model.Text(row, 0))
	}
}

// fontStyleOf makes the style of the flags
func fontStyleOf(bold, italic bool) FontStyle {
	var s FontStyle
	if bold {
		s |= FontBold
	}
	if italic {
		s |= FontItalic
	}
	return s
}

// scaledIcon returns the icon scaled to size x size, cached
func (c *ListView) scaledIcon(img image.Image, size int) image.Image {
	if b := img.Bounds(); b.Dx() == size && b.Dy() == size {
		return img
	}
	if scaled, ok := c.icons[img]; ok {
		return scaled
	}
	if c.icons == nil || len(c.icons) > 256 {
		c.icons = map[image.Image]*image.RGBA{}
	}
	scaled := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), xdraw.Over, nil)
	c.icons[img] = scaled
	return scaled
}

func (c *ListView) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.updateInnerSize()
}
