package ui

// Expander is a section with a clickable header that shows or hides its
// content. The content is a Panel: add the widgets to Content(), laid out on
// its grid as usual.
//
//	adv := ui.NewExpander("Advanced")
//	adv.Content().AddLabel(0, 0, "Timeout")
//	adv.Content().AddWidget(0, 1, timeoutBox)
//
// A click on the header, Space or Enter toggles it; Left collapses and Right
// expands it.
type Expander struct {
	Widget
	title    string
	expanded bool
	content  *Panel

	onExpandedChanged func()
	// onUserToggled lets an Accordion close the other sections
	onUserToggled func()
}

const (
	expanderArrowArea    = 24
	expanderContentInset = 8
)

func NewExpander(title string) *Expander {
	var c Expander
	c.InitWidget()
	c.SetTypeName("Expander")
	c.SetPanelPadding(0)
	c.SetCellPadding(0)
	c.SetCanBeFocused(true)
	c.SetMouseCursor(MouseCursorPointer)
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		return y < c.headerHeight()
	})
	c.SetOnMouseUp(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		if button == MouseButtonLeft && x >= 0 && x < c.Width() && y >= 0 && y < c.headerHeight() && c.Enabled() {
			c.toggleByUser(!c.expanded)
		}
		return true
	})

	c.content = NewPanel()
	c.content.SetPanelPadding(expanderContentInset)
	c.Widget.AddWidget(0, 0, c.content)
	c.content.SetVisible(false)

	c.insetTop = c.headerHeight()
	c.SetTitle(title)
	return &c
}

// Content is the panel holding the widgets shown when the expander is expanded
func (c *Expander) Content() *Panel {
	return c.content
}

// AddWidget adds the widget to the content panel (see Content)
func (c *Expander) AddWidget(gridRow int, gridColumn int, w Widgeter) {
	c.content.AddWidget(gridRow, gridColumn, w)
}

func (c *Expander) Title() string {
	return c.title
}

func (c *Expander) SetTitle(title string) {
	c.title = title
	c.updateMinWidth()
	c.form.Update()
}

func (c *Expander) SetTitleFunc(f func() string) {
	c.setTextFunc(f, c.SetTitle)
}

func (c *Expander) Expanded() bool {
	return c.expanded
}

// SetExpanded expands or collapses the expander without calling the
// SetOnExpandedChanged function.
func (c *Expander) SetExpanded(expanded bool) {
	if c.expanded == expanded {
		return
	}
	c.expanded = expanded
	c.content.SetVisible(expanded)
	c.ClearLayoutCache()
	c.form.UpdateLayout()
}

// SetOnExpandedChanged sets the function called when the user expands or
// collapses the expander.
func (c *Expander) SetOnExpandedChanged(f func()) {
	c.onExpandedChanged = f
}

func (c *Expander) toggleByUser(expanded bool) {
	if c.expanded == expanded {
		return
	}
	c.SetExpanded(expanded)
	if c.onUserToggled != nil {
		c.onUserToggled()
	}
	if c.onExpandedChanged != nil {
		c.onExpandedChanged()
	}
}

// MaxHeight keeps a collapsed expander at its header's height, and an
// expanded one at the height of its content unless the content stretches -
// so extra space in the parent goes to the widgets that can use it.
func (c *Expander) MaxHeight() int {
	if !c.expanded {
		return c.headerHeight()
	}
	if c.content.YExpandable() {
		return c.Widget.MaxHeight()
	}
	return c.MinHeight()
}

func (c *Expander) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	if c.onKeyDown != nil && c.onKeyDown(key, mods) {
		return true
	}
	if !c.Enabled() {
		return false
	}
	switch key {
	case KeySpace, KeyEnter:
		c.toggleByUser(!c.expanded)
		return true
	case KeyArrowLeft:
		c.toggleByUser(false)
		return true
	case KeyArrowRight:
		c.toggleByUser(true)
		return true
	}
	return false
}

func (c *Expander) headerHeight() int {
	return ThemeControlHeight()
}

func (c *Expander) updateMinWidth() {
	textWidth, _, err := MeasureText(c.FontFamily(), c.FontSize(), c.title)
	if err != nil {
		textWidth = 0
	}
	c.SetMinWidth(expanderArrowArea + textWidth + 12)
}

func (c *Expander) draw(cnv *Canvas) {
	p := CurrentPalette()
	w := c.Width()
	hh := c.headerHeight()
	enabled := c.Enabled()

	fill := p.Button
	if enabled && c.IsHovered() && c.lastMouseAbsPosY < hh {
		fill = hoverColor(fill, p.ButtonText)
	}
	border := MixColors(p.Border, p.Window, 0.2)
	if c.IsFocused() && enabled {
		border = p.Highlight
	}
	cnv.FillFrame(0, 0, w, hh, themeControlRadius, fill, border)

	textColor := p.ButtonText
	if !enabled {
		textColor = p.DisabledText
	}

	// A right-pointing triangle when collapsed, a down-pointing one when expanded
	const half = 4
	cx, cy := expanderArrowArea/2+2, hh/2
	if c.expanded {
		cnv.FillTriangle(cx-half, cy-half/2, cx+half, cy-half/2, cx, cy+half/2+1, textColor)
	} else {
		cnv.FillTriangle(cx-half/2, cy-half, cx-half/2, cy+half, cx+half/2+1, cy, textColor)
	}

	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(textColor)
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	title := truncateTextToWidth(c.FontFamily(), c.FontSize(), c.title, w-expanderArrowArea-8)
	cnv.DrawText(expanderArrowArea, 0, w-expanderArrowArea-8, hh, title)
}

func (c *Expander) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.insetTop = c.headerHeight()
	c.updateMinWidth()
}

// Accordion is a column of expanders (sections). By default only one section
// is expanded at a time: expanding one collapses the others.
//
//	acc := ui.NewAccordion()
//	general := acc.AddSection("General")
//	general.Content().AddLabel(0, 0, "Name")
//	acc.AddSection("Network")
type Accordion struct {
	Widget
	sections  []*Expander
	exclusive bool

	onSectionChanged func(index int)
}

func NewAccordion() *Accordion {
	var c Accordion
	c.InitWidget()
	c.SetTypeName("Accordion")
	c.SetPanelPadding(0)
	c.SetCellPadding(2)
	c.exclusive = true
	return &c
}

// AddSection adds a collapsed section at the bottom and returns it. In an
// exclusive accordion the first section is expanded.
func (c *Accordion) AddSection(title string) *Expander {
	e := NewExpander(title)
	index := len(c.sections)
	c.sections = append(c.sections, e)
	e.onUserToggled = func() { c.sectionToggled(index) }
	c.Widget.AddWidget(index, 0, e)
	if c.exclusive && index == 0 {
		e.SetExpanded(true)
	}
	return e
}

func (c *Accordion) Sections() []*Expander {
	return c.sections
}

// SetExclusive sets whether only one section may be expanded at a time.
func (c *Accordion) SetExclusive(exclusive bool) {
	c.exclusive = exclusive
	if exclusive {
		c.Expand(c.ExpandedIndex())
	}
}

func (c *Accordion) Exclusive() bool {
	return c.exclusive
}

// Expand expands the section at index; in an exclusive accordion it collapses
// the others. A negative index collapses all the sections.
func (c *Accordion) Expand(index int) {
	for i, e := range c.sections {
		if i == index {
			e.SetExpanded(true)
		} else if c.exclusive || index < 0 {
			e.SetExpanded(false)
		}
	}
}

// ExpandedIndex returns the index of the first expanded section, -1 if none
func (c *Accordion) ExpandedIndex() int {
	for i, e := range c.sections {
		if e.Expanded() {
			return i
		}
	}
	return -1
}

// SetOnSectionChanged sets the function called when the user expands or
// collapses a section; index is the section the user clicked.
func (c *Accordion) SetOnSectionChanged(f func(index int)) {
	c.onSectionChanged = f
}

func (c *Accordion) sectionToggled(index int) {
	if c.exclusive && c.sections[index].Expanded() {
		for i, e := range c.sections {
			if i != index {
				e.SetExpanded(false)
			}
		}
	}
	if c.onSectionChanged != nil {
		c.onSectionChanged(index)
	}
}
