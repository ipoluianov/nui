package ui

import "slices"

type TabWidget struct {
	Widget

	// OnCurrentChanged: another page became the current one (its index), by
	// the user or by SetCurrentIndex
	OnCurrentChanged Signal[int]
	// OnActivated: the user clicked a tab (its index), also the current one
	OnActivated Signal[int]

	pages        []tabWidgetPage
	panelTop     *tabWidgetHeader
	panelContent *tabWidgetContent
	currentPage  int

	headerHeight       int
	headerItemMinWidth int
}

type tabWidgetPage struct {
	name     string
	nameFunc func() string // see SetPageNameFunc
	widget   Widgeter
}

func NewTabWidget() *TabWidget {
	var c TabWidget
	c.InitWidget()
	c.SetPanelPadding(0)
	c.SetCellPadding(0)

	c.headerHeight = ThemeControlHeight()
	c.headerItemMinWidth = 100

	c.SetTypeName("TabWidget")

	c.SetXExpandable(true)
	c.SetYExpandable(true)

	c.pages = make([]tabWidgetPage, 0)

	c.panelTop = newTabWidgetHeader(c.headerItemMinWidth, c.headerHeight)
	c.panelTop.setThemeHeight(ThemeControlHeight, true)
	c.panelTop.onTabChanged = c.onTabChanged
	c.AddWidget(0, 0, c.panelTop)

	c.panelContent = newTabWidgetContent(&c)
	c.AddWidget(1, 0, c.panelContent)

	c.SetOnPostPaint(c.drawPost)

	return &c
}

// tabWidgetContent shows the current page. Its minimum size is of the
// largest page, not only of the current one, as in Qt: the window doesn't
// change its size when another tab is chosen.
type tabWidgetContent struct {
	Panel
	tabs *TabWidget
}

func newTabWidgetContent(tabs *TabWidget) *tabWidgetContent {
	var c tabWidgetContent
	c.InitWidget()
	c.SetTypeName("Panel")
	c.tabs = tabs
	return &c
}

func (c *tabWidgetContent) MinWidth() int {
	result := c.Panel.MinWidth()
	padding := c.GetPropInt("padding", 2)
	for _, page := range c.tabs.pages {
		if page.widget.IsVisible() {
			result = max(result, page.widget.MinWidth()+padding*2)
		}
	}
	return result
}

func (c *tabWidgetContent) MinHeight() int {
	result := c.Panel.MinHeight()
	padding := c.GetPropInt("padding", 2)
	for _, page := range c.tabs.pages {
		if page.widget.IsVisible() {
			result = max(result, page.widget.MinHeight()+padding*2)
		}
	}
	return result
}

// ClearLayoutCache clears the pages not shown now too: their sizes count
func (c *tabWidgetContent) ClearLayoutCache() {
	c.Panel.ClearLayoutCache()
	for _, page := range c.tabs.pages {
		page.widget.ClearLayoutCache()
	}
}

// onTabChanged: the user clicked the tab
func (c *TabWidget) onTabChanged(index int) {
	c.SetCurrentIndex(index)
	c.OnActivated.Emit(index)
}

// CurrentIndex returns the index of the current page, -1 if there are none
func (c *TabWidget) CurrentIndex() int {
	if len(c.pages) == 0 {
		return -1
	}
	return c.currentPage
}

// SetCurrentIndex shows the page; OnCurrentChanged is emitted if it is
// another one
func (c *TabWidget) SetCurrentIndex(index int) {
	if index < 0 || index >= len(c.pages) {
		return
	}
	changed := index != c.currentPage
	c.currentPage = index
	c.rebuildInterface()
	if changed {
		c.OnCurrentChanged.Emit(index)
	}
}

// CurrentPage returns the widget of the current page, nil if there are none
func (c *TabWidget) CurrentPage() Widgeter {
	return c.Page(c.CurrentIndex())
}

// PageCount returns the number of the pages
func (c *TabWidget) PageCount() int {
	return len(c.pages)
}

// Page returns the widget of the page, nil if there is no such page
func (c *TabWidget) Page(index int) Widgeter {
	if index < 0 || index >= len(c.pages) {
		return nil
	}
	return c.pages[index].widget
}

// PageIndex returns the index of the page of the widget, -1 if none
func (c *TabWidget) PageIndex(w Widgeter) int {
	return slices.IndexFunc(c.pages, func(p tabWidgetPage) bool { return p.widget == w })
}

// SetPageName renames the page
func (c *TabWidget) SetPageName(index int, name string) {
	if index < 0 || index >= len(c.pages) {
		return
	}
	c.pages[index].name = name
	c.pages[index].nameFunc = nil
	c.rebuildInterface()
}

// InsertPage puts a page before the page index (at the end for an index
// past them); the current page stays current
func (c *TabWidget) InsertPage(index int, name string, w Widgeter) {
	index = max(0, min(index, len(c.pages)))
	c.pages = slices.Insert(c.pages, index, tabWidgetPage{name: name, widget: w})
	if len(c.pages) > 1 && index <= c.currentPage {
		c.currentPage++
		c.rebuildInterface()
		c.OnCurrentChanged.Emit(c.currentPage)
		return
	}
	c.rebuildInterface()
}

// RemovePage removes the page. The current page stays current; if it's the
// one removed, the next one becomes current, or the last.
func (c *TabWidget) RemovePage(index int) {
	if index < 0 || index >= len(c.pages) {
		return
	}
	c.pages = slices.Delete(c.pages, index, index+1)
	if index > c.currentPage {
		c.rebuildInterface()
		return
	}
	if index < c.currentPage {
		c.currentPage-- // the same page, one place to the left
	}
	c.currentPage = max(0, min(c.currentPage, len(c.pages)-1))
	c.rebuildInterface()
	if len(c.pages) > 0 {
		c.OnCurrentChanged.Emit(c.currentPage)
	}
}

func (c *TabWidget) AddPage(name string, widgeter Widgeter) {
	if c.pages == nil {
		c.pages = make([]tabWidgetPage, 0)
	}

	var p tabWidgetPage
	p.name = name
	p.widget = widgeter
	c.pages = append(c.pages, p)

	c.rebuildInterface()
}

func (c *TabWidget) rebuildInterface() {
	// Build buttons
	c.panelTop.RemoveAllWidgets()
	items := make([]string, len(c.pages))
	for i, page := range c.pages {
		items[i] = page.name
	}
	c.panelTop.SetItems(items)
	c.panelTop.currentIndex = c.currentPage

	// Build content
	c.panelContent.RemoveAllWidgets()
	c.panelContent.SetPanelPadding(0)
	//c.panelContent.SetAutoFillBackground(true)
	//c.panelContent.SetBackgroundColor(ColorFromHex("#222222"))
	if c.currentPage >= 0 && c.currentPage < len(c.pages) {
		c.panelContent.AddWidget(0, 0, c.pages[c.currentPage].widget)
	}
	c.form.Update()
}

func (c *TabWidget) drawPost(cnv *Canvas) {
	// Draw border
	borderColor := CurrentPalette().Border

	// Border left
	cnv.DrawLine(0, c.headerHeight, 0, c.Height(), 1, borderColor)
	// Border bottom
	cnv.DrawLine(0, c.Height()-1, c.Width(), c.Height()-1, 1, borderColor)
	// Border right
	cnv.DrawLine(c.Width()-1, c.headerHeight, c.Width()-1, c.Height(), 1, borderColor)
}

type tabWidgetHeader struct {
	Widget
	items        []string
	minWidth     int
	height       int
	currentIndex int

	itemsWidths []int

	onTabChanged       func(index int)
	onTabChangedCalled bool
}

func newTabWidgetHeader(minWidth int, height int) *tabWidgetHeader {
	var c tabWidgetHeader
	c.InitWidget()
	c.minWidth = minWidth
	c.height = height
	c.SetTypeName("TabWidgetHeader")
	c.SetOnPaint(c.draw)

	c.SetOnMouseDown(c.onMouseDown)
	c.SetOnMouseUp(c.onMouseUp)
	c.SetOnMouseMove(c.onMouseMove)
	return &c
}

func (c *tabWidgetHeader) setCurrentPage(index int) {
	c.currentIndex = index
	if c.onTabChanged != nil {
		if !c.onTabChangedCalled {
			c.onTabChangedCalled = true
			c.onTabChanged(index)
			c.onTabChangedCalled = false
		}
	}
	c.form.Update()
}

func (c *tabWidgetHeader) SetItems(items []string) {
	c.items = items
	c.form.Update()
}

func (c *tabWidgetHeader) itemByCoords(x int, y int) int {
	if y < 0 || y > c.height {
		return -1
	}

	xOffset := 0
	for i, width := range c.itemsWidths {
		if x >= xOffset && x < xOffset+width {
			return i
		}
		xOffset += width
	}

	return -1
}

func (c *tabWidgetHeader) onMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	index := c.itemByCoords(x, y)
	if index >= 0 && index < len(c.items) {
		c.setCurrentPage(index)
		return true
	}
	return false
}

func (c *tabWidgetHeader) onMouseUp(button MouseButton, x int, y int, mods KeyModifiers) bool {
	return false
}

func (c *tabWidgetHeader) onMouseMove(x int, y int, mods KeyModifiers) bool {
	// hover - mouse cursor pointer
	index := c.itemByCoords(x, y)
	if index >= 0 && index < len(c.items) {
		c.SetMouseCursor(MouseCursorPointer)
	} else {
		c.SetMouseCursor(MouseCursorNotDefined)
	}
	c.form.Update()
	return true
}

// draw: the current tab has the page's background and joins the page; the
// other tabs are lower, shaded and have dimmer text
func (c *tabWidgetHeader) draw(cnv *Canvas) {
	p := CurrentPalette()
	borderColor := p.Border
	pageColor := colorToRGBA(c.BackgroundColor())
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())

	c.itemsWidths = make([]int, len(c.items))
	xOffset := 0
	for i, item := range c.items {
		width := c.minWidth

		// Measure text width
		textWidth, textHeight, err := MeasureText(c.FontFamily(), c.FontSize(), item)
		_ = textHeight
		if err == nil {
			textPadding := 20
			width = textWidth + textPadding
			width = max(width, c.minWidth)
		}

		c.itemsWidths[i] = width

		x := xOffset
		yOffset := 2
		textColor := p.WindowText
		if i == c.currentIndex {
			yOffset = 0
			cnv.FillRect(x, yOffset, width, c.height-yOffset, pageColor)
		} else {
			cnv.FillRect(x, yOffset, width, c.height-yOffset, MixColors(pageColor, p.WindowText, 0.05))
			textColor = MixColors(p.WindowText, pageColor, 0.35)
		}

		cnv.SetHAlign(HAlignCenter)
		cnv.SetVAlign(VAlignCenter)
		cnv.SetColor(textColor)
		cnv.DrawText(x, yOffset, width, c.height-yOffset, item)

		// Left Border
		cnv.DrawLine(x, yOffset, x, c.height, 1, borderColor)
		// Top Border
		cnv.DrawLine(x, yOffset, x+width, yOffset, 1, borderColor)
		// Right Border
		cnv.DrawLine(x+width, yOffset, x+width, c.height, 1, borderColor)

		if i != c.currentIndex {
			// Bottom Border
			cnv.DrawLine(x, c.height-1, x+width, c.height-1, 1, borderColor)
		}

		xOffset += width
	}

	// Fill remaining space with bottom border
	if xOffset < c.Width() {
		cnv.DrawLine(xOffset, c.height-1, c.Width(), c.height-1, 1, borderColor)
	}
}

// applyThemeMetrics also updates the pages that aren't shown, which aren't
// among the tab widget's children
func (c *TabWidget) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.headerHeight = ThemeControlHeight()
	c.panelTop.height = c.headerHeight
	for _, page := range c.pages {
		applyThemeMetricsTree(page.widget)
	}
}
