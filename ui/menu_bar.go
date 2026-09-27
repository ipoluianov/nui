package ui

// MenuBar is the main menu of a form: a row of titles at the top of the
// window, each opening a ContextMenu that drops down below it. Install it
// with Form.SetMenuBar:
//
//	bar := ui.NewMenuBar()
//	file := bar.AddMenu("File")
//	file.AddItem("Open", onOpen)
//	file.AddSeparator()
//	file.AddItem("Exit", func() { form.Close() })
//	form.SetMenuBar(bar)
//
// While one of its menus is open, moving the mouse to another title opens
// that title's menu instead, as in the usual desktop menus.
type MenuBar struct {
	Widget
	items []*MenuBarItem
	// openItem is the item whose menu was opened last; its menu may have
	// been closed since, see isOpen
	openItem *MenuBarItem
}

// menuBarItemPadding is the space left and right of a title
const menuBarItemPadding = 10

func NewMenuBar() *MenuBar {
	var c MenuBar
	c.InitWidget()
	c.SetAbsolutePositioning(true)
	c.SetTypeName("MenuBar")
	c.SetName("MenuBar")
	c.SetAutoFillBackground(true)
	c.SetOnPostPaint(c.drawBorder)
	return &c
}

// drawBorder separates the bar from the form's content below it.
func (c *MenuBar) drawBorder(cnv *Canvas) {
	cnv.FillRect(0, c.Height()-1, c.Width(), 1, CurrentPalette().Divider)
}

// AddMenu adds a title and returns its menu, to be filled like any
// ContextMenu (items, separators, submenus, icons).
func (c *MenuBar) AddMenu(text string) *ContextMenu {
	menu := NewContextMenu(c)
	c.AddMenuItem(text, menu)
	return menu
}

// AddMenuItem adds a title that opens an existing menu. Returns the title,
// e.g. to give it a text that follows the language (SetTextFunc).
func (c *MenuBar) AddMenuItem(text string, menu *ContextMenu) *MenuBarItem {
	item := newMenuBarItem(c, text, menu)
	c.items = append(c.items, item)
	c.AddWidget(0, 0, item)
	// AddWidget has laid the bar out again
	if c.form != nil {
		menu.attachToForm(menu, c.form)
	}
	return item
}

// Items returns the titles of the bar.
func (c *MenuBar) Items() []*MenuBarItem {
	return c.items
}

// barHeight is the height the bar takes at the top of the form.
func (c *MenuBar) barHeight() int {
	return ThemeRowHeight() + 1 // + the bottom border
}

// rebuildVisualElements places the titles one after another.
func (c *MenuBar) rebuildVisualElements() {
	x := 0
	h := c.barHeight() - 1
	for _, item := range c.items {
		if !item.IsVisible() {
			item.SetPosition(x, 0)
			item.SetSize(0, 0)
			continue
		}
		w := item.contentWidth()
		item.SetPosition(x, 0)
		item.SetSize(w, h)
		x += w
	}
}

// isOpen reports whether the item's menu is open now: it may have been
// closed by a click outside, Escape or a chosen item.
func (c *MenuBar) isOpen(item *MenuBarItem) bool {
	return item != nil && c.openItem == item && c.form != nil && c.form.isPopupWidgetOpen(item.menu)
}

// hasOpenMenu reports whether any of the bar's menus is open.
func (c *MenuBar) hasOpenMenu() bool {
	return c.isOpen(c.openItem)
}

// openMenu opens the item's menu below it, closing any other open popup.
func (c *MenuBar) openMenu(item *MenuBarItem) {
	if c.form == nil || item.menu == nil {
		return
	}
	c.form.closePopups()
	c.openItem = item
	x, y := item.RectClientAreaOnWindow()
	item.menu.dropDownFrom(x, y, item.Width(), item.Height())
	c.form.Update()
}

func (c *MenuBar) attachToForm(self Widgeter, form *Form) {
	c.Widget.attachToForm(self, form)
	// The menus aren't children of the bar
	for _, item := range c.items {
		if item.menu != nil {
			item.menu.attachToForm(item.menu, form)
		}
	}
}

// MenuBarItem is a title in a MenuBar.
type MenuBarItem struct {
	Widget
	bar  *MenuBar
	text string
	menu *ContextMenu
}

func newMenuBarItem(bar *MenuBar, text string, menu *ContextMenu) *MenuBarItem {
	var c MenuBarItem
	c.InitWidget()
	c.SetTypeName("MenuBarItem")
	c.bar = bar
	c.text = text
	c.menu = menu
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.mouseDown)
	c.SetOnMouseEnter(c.mouseEnter)
	return &c
}

func (c *MenuBarItem) Text() string {
	return c.text
}

func (c *MenuBarItem) SetText(text string) {
	c.text = text
	if c.form != nil {
		c.form.layoutMenuBar()
		c.form.Update()
	}
}

func (c *MenuBarItem) SetTextFunc(f func() string) {
	c.setTextFunc(f, c.SetText)
}

// Menu returns the menu the title opens.
func (c *MenuBarItem) Menu() *ContextMenu {
	return c.menu
}

// SetVisible hides or shows the title; the titles after it move over.
func (c *MenuBarItem) SetVisible(visible bool) {
	c.Widget.SetVisible(visible)
	if c.form != nil {
		c.form.layoutMenuBar()
	}
}

// applyLanguage also updates the menu, which isn't a child of the title.
func (c *MenuBarItem) applyLanguage() {
	c.Widget.applyLanguage()
	if c.menu != nil {
		applyLanguageTree(c.menu)
	}
}

func (c *MenuBarItem) contentWidth() int {
	textWidth, _, err := MeasureText(c.FontFamily(), c.FontSize(), c.text)
	if err != nil {
		textWidth = 0
	}
	return textWidth + menuBarItemPadding*2
}

// draw: the title of the open menu in the accent color, a hovered title
// in a softer one.
func (c *MenuBarItem) draw(ctx *Canvas) {
	p := CurrentPalette()
	textColor := p.WindowText
	switch {
	case c.bar.isOpen(c):
		ctx.FillRect(0, 0, c.Width(), c.Height(), p.Highlight)
		textColor = p.HighlightedText
	case c.IsHovered() && c.Enabled():
		ctx.FillRect(0, 0, c.Width(), c.Height(), p.Selection)
	}
	if !c.Enabled() {
		textColor = p.DisabledText
	}

	ctx.SetHAlign(HAlignCenter)
	ctx.SetVAlign(VAlignCenter)
	ctx.SetColor(textColor)
	ctx.SetFontFamily(c.FontFamily())
	ctx.SetFontSize(c.FontSize())
	ctx.DrawText(0, 0, c.Width(), c.Height(), c.text)
}

// mouseDown opens the menu. A click on the title of the open menu doesn't
// get here: the form closes its popups on a click outside them, so the
// click closes the menu.
func (c *MenuBarItem) mouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft || !c.Enabled() {
		return true
	}
	c.bar.openMenu(c)
	return true
}

// mouseEnter switches to this title's menu while another one is open.
func (c *MenuBarItem) mouseEnter() {
	if c.Enabled() && c.bar.hasOpenMenu() && !c.bar.isOpen(c) {
		c.bar.openMenu(c)
	}
}
