package ui

import (
	"image/color"
)

type ComboBox struct {
	Widget
	items         []*ComboBoxItem
	selectedIndex int

	onSelectedIndexChanged func()
}

type ComboBoxItem struct {
	text string
	data interface{}
}

// DefaultComboBoxMinWidth keeps a bare ComboBox (no width ever set by the
// caller) from collapsing to an unusably thin trigger.
const DefaultComboBoxMinWidth = 150

func NewComboBox() *ComboBox {
	var c ComboBox
	c.InitWidget()
	// Height is fixed (min == max): without a max, a ComboBox alone in a
	// row - with nothing else in the page actually YExpandable - absorbs
	// all leftover vertical space via the layout's "grow every row"
	// fallback, ballooning into a huge empty box instead of a compact
	// dropdown trigger.
	c.SetMinWidth(DefaultComboBoxMinWidth)
	c.SetMaxWidth(10000)
	c.setThemeHeight(ThemeControlHeight, true)

	c.SetOnMouseDown(func(button MouseButton, x int, y int, mods KeyModifiers) bool {
		if button == MouseButtonLeft {
			c.OpenPopup()
			return true
		}
		return false
	})

	c.SetTypeName("ComboBox")
	c.SetMouseCursor(MouseCursorPointer)
	c.SetOnPaint(c.draw)
	return &c
}

func (c *ComboBox) AddItem(text string, data interface{}) {
	var item ComboBoxItem
	item.text = text
	item.data = data
	c.items = append(c.items, &item)
}

// SetItemText changes the text of the item, e.g. for another language;
// the item keeps its data and the selection doesn't change.
func (c *ComboBox) SetItemText(index int, text string) {
	if index < 0 || index >= len(c.items) {
		return
	}
	c.items[index].text = text
	c.form.Update()
}

// SetSelectedIndex selects the item without calling the
// SetOnSelectedIndexChanged function.
func (c *ComboBox) SetSelectedIndex(index int) {
	if index < 0 || index >= len(c.items) {
		return
	}
	c.selectedIndex = index
	c.form.Update()
}

func (c *ComboBox) SelectedIndex() int {
	return c.selectedIndex
}

func (c *ComboBox) ItemCount() int {
	return len(c.items)
}

// SetOnSelectedIndexChanged sets the function called when the user picks
// another item.
func (c *ComboBox) SetOnSelectedIndexChanged(f func()) {
	c.onSelectedIndexChanged = f
}

// selectByUser selects the item the user picked
func (c *ComboBox) selectByUser(index int) {
	changed := index != c.selectedIndex
	c.SetSelectedIndex(index)
	if changed && c.onSelectedIndexChanged != nil {
		c.onSelectedIndexChanged()
	}
}

func (c *ComboBox) SelectedItemText() string {
	if c.selectedIndex < 0 || c.selectedIndex >= len(c.items) {
		return ""
	}
	return c.items[c.selectedIndex].text
}

func (c *ComboBox) SelectedItemData() interface{} {
	if c.selectedIndex < 0 || c.selectedIndex >= len(c.items) {
		return nil
	}
	return c.items[c.selectedIndex].data
}

func (c *ComboBox) OpenPopup() {
	popup := NewComboBoxPopup()
	// The popup is a fresh widget, never added via AddWidget, so nothing
	// else ever attaches it to a form - ShowPopup's c.form.Panel() would
	// nil-panic without this (the same class of bug as an unattached
	// ContextMenu submenu).
	popup.attachToForm(popup, c.form)
	// The dropdown should never look narrower than the control it drops
	// from, even though it's free to grow wider to fit long item text.
	popup.triggerWidth = c.Width()
	// So the popup can highlight whichever item is currently selected.
	popup.selectedIndex = c.selectedIndex
	for _, item := range c.items {
		popup.AddItem(item.text, func(index int) {
			c.selectByUser(index)
			c.form.Update()
		})
	}
	x, y := c.RectClientAreaOnWindow()
	popup.triggerTop = y
	popup.ShowPopup(x, y+c.Height())
}

// Size and placement of the dropdown arrow drawn at the right of the control:
// the submenu arrow of ContextMenuItem, turned downwards.
const (
	comboBoxArrowWidth   = contextMenuArrowHalfHeight * 2
	comboBoxArrowHeight  = contextMenuArrowWidth
	comboBoxArrowPadding = 10
)

// draw looks like a button: the same frame, with the text on the left
func (c *ComboBox) draw(cnv *Canvas) {
	p := CurrentPalette()
	fill, border := p.Button, p.Border
	if c.backgroundColor != nil {
		fill = colorToRGBA(c.backgroundColor)
	}
	foreColor := colorToRGBA(c.ForegroundColor())
	switch {
	case !c.Enabled():
		foreColor = p.DisabledText
	case c.IsHovered():
		fill = hoverColor(fill, foreColor)
	}
	if c.IsFocused() && c.Enabled() {
		border = p.Highlight
	}
	cnv.FillFrame(0, 0, c.Width(), c.Height(), themeControlRadius, fill, border)

	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(foreColor)
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	textAreaWidth := c.Width() - themeTextInset - comboBoxArrowPadding*2 - comboBoxArrowWidth
	cnv.DrawText(themeTextInset, 0, textAreaWidth, c.Height(), c.SelectedItemText())

	c.drawArrow(cnv, foreColor)
}

// drawArrow paints a small downward-pointing triangle - the usual dropdown
// indicator - at the right edge of the control.
func (c *ComboBox) drawArrow(cnv *Canvas, arrowColor color.Color) {
	x := c.Width() - comboBoxArrowPadding - comboBoxArrowWidth
	y := (c.Height() - comboBoxArrowHeight) / 2
	cnv.FillTriangle(x, y, x+comboBoxArrowWidth, y, x+comboBoxArrowWidth/2, y+comboBoxArrowHeight, arrowColor)
}

// Adaptive popup width bounds: never wider than this, regardless of how
// long an item's text is.
const comboBoxPopupMaxWidth = 420
const comboBoxItemPadding = 10

// comboBoxPopupMaxVisibleItems limits the dropdown's height; longer lists scroll
const comboBoxPopupMaxVisibleItems = 12

// comboBoxPopupWheelItems is how many items one mouse wheel step scrolls
const comboBoxPopupWheelItems = 3

type comboBoxPopup struct {
	Widget

	items []*comboBoxPopupItem
	// triggerWidth is the owning ComboBox's own width - the floor for the
	// popup's width, set by ComboBox.OpenPopup before ShowPopup runs.
	triggerWidth int
	// selectedIndex is the owning ComboBox's current selection, also set by
	// ComboBox.OpenPopup, so the matching item can be highlighted.
	selectedIndex int
	// triggerTop is the owning ComboBox's top in client coordinates, where
	// the dropdown ends when it opens upwards (see PopupFlipped).
	triggerTop int
}

func NewComboBoxPopup() *comboBoxPopup {
	var c comboBoxPopup
	c.InitWidget()
	c.SetTypeName("ComboBoxPopup")
	c.SetAbsolutePositioning(true)
	c.scrollBarInset = 1 // inside the border
	c.SetRole("popup")
	c.SetAutoFillBackground(true)
	c.SetOnPostPaint(c.drawBorder)
	c.SetOnMouseWheel(c.processWheel)
	return &c
}

// drawBorder matches ContextMenu's treatment - a subtle outline so the
// dropdown reads as a distinct surface instead of blending into whatever is
// behind it.
func (c *comboBoxPopup) drawBorder(cnv *Canvas) {
	cnv.SetColor(CurrentPalette().Border)
	// Post-paint is translated by the scroll offset; the border must not scroll
	cnv.DrawRect(0, c.ScrollY(), c.Width(), c.Height())
}

// PopupFlipped opens the dropdown above the ComboBox when it doesn't fit
// below, and aligns it to the ComboBox's right edge when it doesn't fit to
// the right.
func (c *comboBoxPopup) PopupFlipped() (int, int) {
	return c.X() + c.triggerWidth - c.Width(), c.triggerTop - c.Height()
}

func (c *comboBoxPopup) ShowPopup(x int, y int) {
	c.SetPosition(x, y)
	c.rebuildVisualElements()
	c.form.Panel().AppendPopupWidget(c)
	c.form.Update()
}

func (c *comboBoxPopup) AddItem(text string, onClick func(index int)) {
	index := len(c.items)
	item := newComboBoxPopupItem(index, text)
	item.parentWidgetId = c.Id()
	item.OnClick = onClick
	item.selected = index == c.selectedIndex
	c.items = append(c.items, item)
	c.AddWidget(0, 0, item)
}

func (c *comboBoxPopup) rebuildVisualElements() {
	width := c.contentWidth()
	itemHeight := ThemeRowHeight()
	contentHeight := len(c.items) * itemHeight
	height := min(contentHeight, comboBoxPopupMaxVisibleItems*itemHeight)

	c.SetAllowScroll(false, true)
	c.SetSize(width, height)
	c.SetInnerSize(width, contentHeight)

	// The items take the view: the scroll bar has its own room
	itemWidth := c.ViewportWidth()
	yOffset := 0
	for _, item := range c.items {
		item.SetPosition(0, yOffset)
		item.SetSize(itemWidth, itemHeight)
		yOffset += itemHeight
	}
	c.SetInnerSize(itemWidth, contentHeight)
	c.scrollToSelected()
}

// scrollToSelected scrolls the selected item to the middle of the dropdown.
func (c *comboBoxPopup) scrollToSelected() {
	if c.selectedIndex < 0 || c.selectedIndex >= len(c.items) {
		return
	}
	visibleItems := c.Height() / ThemeRowHeight()
	c.scrollToItem(c.selectedIndex - visibleItems/2)
}

// processWheel scrolls by whole items, so no item is cut at the top.
func (c *comboBoxPopup) processWheel(deltaX, deltaY int) bool {
	c.scrollToItem(c.ScrollY()/ThemeRowHeight() - deltaY*comboBoxPopupWheelItems)
	return true
}

// scrollToItem makes the item with the given index the first visible one,
// as far as the list allows.
func (c *comboBoxPopup) scrollToItem(index int) {
	c.setScrollY(index * ThemeRowHeight())
	c.checkScrolls()
	c.form.Update()
}

// contentWidth is never narrower than triggerWidth (the owning ComboBox's
// own width), grows to fit the widest item text when that text wouldn't
// otherwise fit, and never exceeds comboBoxPopupMaxWidth.
func (c *comboBoxPopup) contentWidth() int {
	// A long list scrolls: the scroll bar takes room next to the text
	barRoom := 0
	if len(c.items) > comboBoxPopupMaxVisibleItems {
		barRoom = c.scrollBarYRoom(true)
	}
	width := c.triggerWidth
	for _, item := range c.items {
		textWidth, _, err := MeasureText(item.FontFamily(), item.FontSize(), item.text)
		if err != nil {
			continue
		}
		itemWidth := comboBoxItemPadding*2 + textWidth + barRoom
		if itemWidth > width {
			width = itemWidth
		}
	}
	if width > comboBoxPopupMaxWidth {
		width = comboBoxPopupMaxWidth
	}
	if width < c.triggerWidth {
		width = c.triggerWidth
	}
	return width
}

type comboBoxPopupItem struct {
	Widget
	index    int
	text     string
	selected bool
	OnClick  func(index int)

	// pressed is set by a press on the item; the release over it picks it
	pressed bool
}

func newComboBoxPopupItem(index int, text string) *comboBoxPopupItem {
	var item comboBoxPopupItem
	item.InitWidget()
	item.SetTypeName("ComboBoxPopupItem")
	item.SetAbsolutePositioning(true)
	item.SetMouseCursor(MouseCursorPointer)
	item.SetOnPaint(item.Draw)
	item.SetOnMouseDown(item.mouseDownHandler)
	item.SetOnMouseUp(item.mouseUpHandler)

	item.index = index
	item.text = text
	return &item
}

// Draw: the hovered item in the accent color, the current one softer
func (c *comboBoxPopupItem) Draw(ctx *Canvas) {
	p := CurrentPalette()
	backColor, textColor := p.PopupBase, p.Text
	switch {
	case c.IsHovered():
		backColor, textColor = p.Highlight, p.HighlightedText
	case c.selected:
		backColor = p.Selection
	}
	ctx.FillRect(0, 0, c.InnerWidth(), c.InnerHeight(), backColor)
	ctx.SetHAlign(HAlignLeft)
	ctx.SetVAlign(VAlignCenter)
	ctx.SetColor(textColor)
	ctx.SetFontFamily(c.FontFamily())
	ctx.SetFontSize(c.FontSize())
	ctx.DrawText(comboBoxItemPadding, 0, c.Width()-comboBoxItemPadding*2, c.Height(), c.text)
}

// The item is picked on the release over it, like in the system dropdowns
func (c *comboBoxPopupItem) mouseDownHandler(button MouseButton, x int, y int, mods KeyModifiers) bool {
	c.pressed = button == MouseButtonLeft
	return true
}

func (c *comboBoxPopupItem) mouseUpHandler(button MouseButton, x int, y int, mods KeyModifiers) bool {
	pressed := c.pressed
	c.pressed = false
	if !pressed || button != MouseButtonLeft || x < 0 || x >= c.Width() || y < 0 || y >= c.Height() {
		return true
	}
	if c.OnClick != nil {
		c.OnClick(c.index)
	}
	c.form.Panel().CloseTopPopup()
	return true
}
