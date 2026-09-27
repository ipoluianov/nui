package ui

import ()

type TabWidget struct {
	Widget
	pages        []tabWidgetPage
	panelTop     *tabWidgetHeader
	panelContent *Panel
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

	c.panelContent = NewPanel()
	c.AddWidget(1, 0, c.panelContent)

	c.SetOnPostPaint(c.drawPost)

	return &c
}

func (c *TabWidget) onTabChanged(index int) {
	c.currentPage = index
	c.rebuildInterface()
}

func (c *TabWidget) SetLayoutXml(n *uiNode, eventProcessor interface{}, widgets map[string]Widgeter) {
	for _, child := range n.Nodes {
		if child.XMLName.Local == "tab" {
			childNode := child
			childNode.XMLName.Local = "panel"
			name := child.GetAttrValueByName("text", "")
			wContent := NewPanel()
			wContent.buildNode(&childNode, wContent, 0, 0, eventProcessor, widgets)
			c.AddPage(name, wContent)
		}
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
	//c.panelTop.AddWidgetOnGrid(NewHSpacer(), 0, len(c.pages))

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
