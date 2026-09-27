package ui

type ContextMenu struct {
	Widget
	menuWidth  int
	menuHeight int
	items      []*ContextMenuItem
	CloseEvent func()
	parentMenu *ContextMenu
	onShow     func()

	// dropDown is set when the menu drops down from a rectangle (a MenuBar
	// title) instead of opening at a point, see dropDownFrom
	dropDown                                    bool
	anchorX, anchorY, anchorWidth, anchorHeight int
}

// Adaptive width bounds: the menu shrinks to fit short item text and grows
// for long item text, but never past these limits.
const contextMenuMinWidth = 140
const contextMenuMaxWidth = 420

func NewContextMenu(parent Widgeter) *ContextMenu {
	var c ContextMenu
	c.InitWidget()
	c.SetAbsolutePositioning(true)
	c.SetTypeName("ContextMenu")
	c.SetName("PopupMenuPanel")
	c.SetRole("popup")
	c.SetAutoFillBackground(true)
	c.SetOnPostPaint(c.drawBorder)
	return &c
}

// drawBorder gives the popup a subtle outline so it reads as a distinct
// surface instead of blending into whatever is behind it.
func (c *ContextMenu) drawBorder(cnv *Canvas) {
	cnv.SetColor(CurrentPalette().Border)
	cnv.DrawRect(0, 0, c.Width(), c.Height())
}

// SetOnShow sets the function called every time before the menu is shown,
// e.g. to hide the items that don't apply to what is selected now
// (ContextMenuItem.SetVisible).
func (c *ContextMenu) SetOnShow(f func()) {
	c.onShow = f
}

func (c *ContextMenu) ShowMenu(x int, y int) {
	c.dropDown = false
	c.show(x, y)
}

// dropDownFrom opens the menu below the rectangle (x, y, width, height) in
// the form's client coordinates, or above it when it doesn't fit below.
func (c *ContextMenu) dropDownFrom(x, y, width, height int) {
	c.dropDown = true
	c.anchorX, c.anchorY, c.anchorWidth, c.anchorHeight = x, y, width, height
	c.show(x, y+height)
}

func (c *ContextMenu) show(x int, y int) {
	c.parentMenu = nil
	if c.onShow != nil {
		c.onShow()
	}
	c.SetPosition(x, y)
	c.rebuildVisualElements()
	c.form.Panel().AppendPopupWidget(c)
	c.form.Update()
}

func (c *ContextMenu) showMenu(x int, y int, parentMenu *ContextMenu) {
	c.CloseAfterPopupWidget(parentMenu)
	c.parentMenu = parentMenu
	c.dropDown = false
	if c.onShow != nil {
		c.onShow()
	}
	c.SetPosition(x, y)
	c.rebuildVisualElements()
	//c.Window().AppendPopup(c)
	c.form.Panel().AppendPopupWidget(c)
}

// PopupFlipped opens a menu that doesn't fit on the screen to the left of
// (or above) the point it was opened at, a submenu to the left of its
// parent menu, and a drop-down menu above its title.
func (c *ContextMenu) PopupFlipped() (int, int) {
	if c.parentMenu != nil {
		return c.parentMenu.X() - c.Width(), c.Y()
	}
	if c.dropDown {
		return c.anchorX + c.anchorWidth - c.Width(), c.anchorY - c.Height()
	}
	return c.X() - c.Width(), c.Y() - c.Height()
}

func (c *ContextMenu) ClosePopup() {
	if c.CloseEvent != nil {
		c.CloseEvent()
	}
}

func (c *ContextMenu) AddItem(text string, onClick func()) *ContextMenuItem {
	item := NewContextMenuItem()
	item.parentWidgetId = c.Id()
	item.parentMenu = c
	item.SetText(text)
	item.OnClick = onClick

	c.items = append(c.items, item)
	c.AddWidget(0, 0, item)
	return item
}

// AddSeparator adds a line that separates groups of items
func (c *ContextMenu) AddSeparator() *ContextMenuItem {
	item := NewContextMenuItem()
	item.parentWidgetId = c.Id()
	item.parentMenu = c
	item.separator = true
	item.SetMouseCursor(MouseCursorArrow)
	c.items = append(c.items, item)
	c.AddWidget(0, 0, item)
	return item
}

func (c *ContextMenu) AddItemWithSubmenu(text string, innerMenu *ContextMenu) *ContextMenuItem {
	item := NewContextMenuItem()
	item.parentWidgetId = c.Id()
	item.parentMenu = c
	item.SetText(text)
	item.innerMenu = innerMenu
	c.items = append(c.items, item)
	c.AddWidget(0, 0, item)
	return item
}

func (c *ContextMenu) RemoveAllItems() {
	c.RemoveAllWidgets()
	c.rebuildVisualElements()
	c.form.Update()
}

func (c *ContextMenu) OnInit() {
	c.rebuildVisualElements()
}

func (c *ContextMenu) needToClose() {
	c.form.Panel().CloseTopPopup()
	if c.parentMenu != nil {
		c.parentMenu.needToClose()
	}
}

func (c *ContextMenu) rebuildVisualElements() {
	menuWidth := c.contentWidth()

	yOffset := 0
	for _, item := range c.items {
		item.needToClosePopupMenu = c.needToClose
		item.parentMenu = c
		// A hidden item takes no place
		if !item.IsVisible() {
			item.SetPosition(0, yOffset)
			item.SetSize(0, 0)
			continue
		}
		item.SetPosition(0, yOffset)
		item.SetSize(menuWidth, item.height())
		yOffset += item.height()
	}
	c.SetSize(menuWidth, yOffset)
	c.menuWidth = menuWidth
	c.menuHeight = yOffset
}

// hasImages reports whether any item has an icon, so the menu reserves the icon column
func (c *ContextMenu) hasImages() bool {
	for _, item := range c.items {
		if item.image != nil && item.IsVisible() {
			return true
		}
	}
	return false
}

// contentWidth measures the widest item text (reserving room for the
// submenu arrow on items that have one) and clamps the result between
// contextMenuMinWidth and contextMenuMaxWidth.
func (c *ContextMenu) contentWidth() int {
	width := contextMenuMinWidth
	for _, item := range c.items {
		if item.separator || !item.IsVisible() {
			continue
		}
		textWidth, _, err := MeasureText(item.FontFamily(), item.FontSize(), item.text)
		if err != nil {
			continue
		}
		itemWidth := item.textX() + textWidth + contextMenuItemPadding
		if item.innerMenu != nil {
			itemWidth += ThemeRowHeight() + contextMenuItemPadding
		}
		if itemWidth > width {
			width = itemWidth
		}
	}
	if width > contextMenuMaxWidth {
		width = contextMenuMaxWidth
	}
	return width
}
