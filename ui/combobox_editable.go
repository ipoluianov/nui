package ui

import "strings"

// EditableComboBox is a text field with a list of suggestions: typing shows
// the items that contain the typed text (the ones that start with it
// first, the match highlighted); the arrow button or Alt+Down / F4 shows
// all of them. Up/Down move in the list, Enter or a click takes the item,
// Escape closes the list. Any text can be entered, not only an item.
//
//	city := ui.NewEditableComboBox()
//	city.SetItems([]string{"Amsterdam", "Berlin", "London", "Paris"})
//	city.SetOnItemSelected(func(index int, text string) { ... })
type EditableComboBox struct {
	Widget
	edit  *TextBox
	items []string

	autoComplete   bool
	settingText    bool
	popup          *completionPopup
	hoverButton    bool
	onTextChanged  func()
	onItemSelected func(index int, text string)
	onAccept       func(text string)
}

func NewEditableComboBox() *EditableComboBox {
	var c EditableComboBox
	c.InitWidget()
	c.SetTypeName("EditableComboBox")
	c.SetAbsolutePositioning(true)
	c.SetMinWidth(DefaultComboBoxMinWidth)
	c.SetMaxWidth(10000)
	c.SetXExpandable(true)
	c.setThemeHeight(ThemeControlHeight, true)
	c.SetOnPaint(c.drawButton)
	c.SetOnMouseDown(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		if button == MouseButtonLeft && c.Enabled() && x >= c.buttonX() {
			c.edit.Focus()
			if c.popup != nil {
				c.closePopup()
			} else {
				c.showSuggestions(false)
			}
		}
		return true
	})
	c.SetOnMouseMove(func(x, y int, mods KeyModifiers) bool {
		hover := x >= c.buttonX()
		if hover != c.hoverButton {
			c.hoverButton = hover
			c.form.Update()
		}
		return true
	})
	c.SetOnMouseLeave(func() { c.hoverButton = false })
	c.autoComplete = true

	c.edit = NewTextBox()
	c.edit.SetOnTextChanged(c.textChanged)
	c.edit.SetOnTextBoxKeyDown(c.editKeyDown)
	c.AddWidget(0, 0, c.edit)
	return &c
}

// SetItems sets the suggestions.
func (c *EditableComboBox) SetItems(items []string) {
	c.items = append([]string(nil), items...)
	if c.popup != nil {
		c.showSuggestions(c.popup.filtered)
	}
}

func (c *EditableComboBox) AddItem(text string) {
	c.items = append(c.items, text)
}

func (c *EditableComboBox) Items() []string {
	return c.items
}

func (c *EditableComboBox) Text() string {
	return c.edit.Text()
}

// SetText sets the text without showing suggestions or calling
// SetOnTextChanged's function.
func (c *EditableComboBox) SetText(text string) {
	c.settingText = true
	c.edit.SetText(text)
	c.edit.MoveCursorToEnd()
	c.settingText = false
}

// SetHint sets the gray text shown while the field is empty.
func (c *EditableComboBox) SetHint(hint string) {
	c.edit.SetHint(hint)
}

// SetAutoComplete turns the suggestions while typing on (the default) or
// off; the arrow button still shows the list.
func (c *EditableComboBox) SetAutoComplete(enabled bool) {
	c.autoComplete = enabled
}

// SetOnTextChanged sets the function called when the user changes the text.
func (c *EditableComboBox) SetOnTextChanged(f func()) {
	c.onTextChanged = f
}

// SetOnItemSelected sets the function called when the user takes an item
// from the list; index is its position in Items.
func (c *EditableComboBox) SetOnItemSelected(f func(index int, text string)) {
	c.onItemSelected = f
}

// SetOnAccept sets the function called on Enter while no suggestion is chosen.
func (c *EditableComboBox) SetOnAccept(f func(text string)) {
	c.onAccept = f
}

// TextBox returns the field, e.g. to make it read-only or focus it.
func (c *EditableComboBox) TextBox() *TextBox {
	return c.edit
}

func (c *EditableComboBox) IsPopupOpen() bool {
	return c.popup != nil
}

func (c *EditableComboBox) Focus() {
	c.edit.Focus()
}

func (c *EditableComboBox) SetEnabled(enabled bool) {
	c.Widget.SetEnabled(enabled)
	c.edit.SetEnabled(enabled)
}

func (c *EditableComboBox) buttonX() int {
	return c.Width() - c.Height()
}

// SetSize lays the field out next to the arrow button.
func (c *EditableComboBox) SetSize(w, h int) {
	c.Widget.SetSize(w, h)
	c.edit.SetPosition(0, 0)
	c.edit.SetSize(max(0, w-h+1), h)
}

// matches returns the indexes of the items containing text, the ones that
// start with it first; all of them for an empty text.
func (c *EditableComboBox) matches(text string) []int {
	needle := strings.ToLower(strings.TrimSpace(text))
	var prefix, contains []int
	for i, item := range c.items {
		lower := strings.ToLower(item)
		switch {
		case needle == "" || strings.HasPrefix(lower, needle):
			prefix = append(prefix, i)
		case strings.Contains(lower, needle):
			contains = append(contains, i)
		}
	}
	return append(prefix, contains...)
}

// showSuggestions opens (or updates) the list: the matching items when
// filtered, else all of them with the current text's match chosen.
func (c *EditableComboBox) showSuggestions(filtered bool) {
	if c.form == nil {
		return
	}
	var indexes []int
	chosen := -1
	if filtered {
		indexes = c.matches(c.Text())
		if len(indexes) > 0 {
			chosen = 0
		}
	} else {
		indexes = c.matches("")
		for i, index := range indexes {
			if strings.EqualFold(c.items[index], c.Text()) {
				chosen = i
			}
		}
	}
	if len(indexes) == 0 || (filtered && len(indexes) == 1 && c.items[indexes[0]] == c.Text()) {
		c.closePopup()
		return
	}

	if c.popup == nil {
		c.popup = newCompletionPopup(c)
		c.popup.setItems(indexes, chosen, filtered)
		x, y := c.RectClientAreaOnWindow()
		c.popup.anchorX, c.popup.anchorY = x, y
		c.popup.SetPosition(x, y+c.Height())
		c.form.OpenPopup(c.popup)
		return
	}
	c.popup.setItems(indexes, chosen, filtered)
	c.form.Update()
}

func (c *EditableComboBox) closePopup() {
	if c.popup != nil && c.form.TopPopupWidget() == Widgeter(c.popup) {
		c.form.CloseTopPopup()
	}
}

// take puts the item into the field.
func (c *EditableComboBox) take(index int) {
	c.closePopup()
	c.SetText(c.items[index])
	if c.onTextChanged != nil {
		c.onTextChanged()
	}
	if c.onItemSelected != nil {
		c.onItemSelected(index, c.items[index])
	}
}

func (c *EditableComboBox) textChanged() {
	if c.settingText {
		return
	}
	if c.onTextChanged != nil {
		c.onTextChanged()
	}
	if !c.autoComplete {
		return
	}
	if strings.TrimSpace(c.Text()) == "" {
		c.closePopup()
		return
	}
	c.showSuggestions(true)
}

func (c *EditableComboBox) editKeyDown() {
	ev := CurrentEvent().Parameter.(*EventTextboxKeyDown)
	open := c.popup != nil
	switch {
	case (ev.Key == KeyArrowDown && ev.Mods.Alt) || ev.Key == KeyF4:
		if open {
			c.closePopup()
		} else {
			c.showSuggestions(false)
		}
	case ev.Key == KeyArrowDown:
		if open {
			c.popup.move(1)
		} else {
			c.showSuggestions(false)
		}
	case ev.Key == KeyArrowUp && open:
		c.popup.move(-1)
	case ev.Key == KeyPageDown && open:
		c.popup.move(completionMaxVisible - 1)
	case ev.Key == KeyPageUp && open:
		c.popup.move(-(completionMaxVisible - 1))
	case ev.Key == KeyEnter:
		if open && c.popup.chosen >= 0 {
			c.take(c.popup.indexes[c.popup.chosen])
		} else {
			c.closePopup()
			if c.onAccept != nil {
				c.onAccept(c.Text())
			}
		}
	default:
		return
	}
	ev.Processed = true
}

func (c *EditableComboBox) drawButton(cnv *Canvas) {
	p := CurrentPalette()
	x, size := c.buttonX(), c.Height()
	fill, border := p.Button, p.Border
	foreColor := p.ButtonText
	switch {
	case !c.Enabled():
		foreColor = p.DisabledText
	case c.hoverButton || c.popup != nil:
		fill = hoverColor(fill, foreColor)
	}
	if c.popup != nil {
		border = p.Highlight
	}
	cnv.FillFrame(x, 0, size, size, themeControlRadius, fill, border)
	ax := x + (size-comboBoxArrowWidth)/2
	ay := (size - comboBoxArrowHeight) / 2
	cnv.FillTriangle(ax, ay, ax+comboBoxArrowWidth, ay, ax+comboBoxArrowWidth/2, ay+comboBoxArrowHeight, foreColor)
}

// ---------------------------------------------------------------- List

const completionMaxVisible = 8

// completionPopup is the list of suggestions. It doesn't take the focus:
// typing goes on in the field while it's open.
type completionPopup struct {
	Widget
	combo    *EditableComboBox
	indexes  []int // into combo.items
	chosen   int   // in indexes, -1 none
	filtered bool  // the matches of the text, highlighted
	first    int   // the first visible row
	hover    int
	anchorX  int
	anchorY  int
}

func newCompletionPopup(combo *EditableComboBox) *completionPopup {
	var c completionPopup
	c.InitWidget()
	c.SetTypeName("CompletionPopup")
	c.SetAbsolutePositioning(true)
	c.SetRole("popup")
	c.SetAutoFillBackground(true)
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		if row := c.rowAt(y); row >= 0 && button == MouseButtonLeft {
			c.combo.take(c.indexes[row])
		}
		return true
	})
	c.SetOnMouseMove(func(x, y int, mods KeyModifiers) bool {
		if row := c.rowAt(y); row != c.hover {
			c.hover = row
			c.form.Update()
		}
		return true
	})
	c.SetOnMouseLeave(func() { c.hover = -1 })
	c.SetOnMouseWheel(func(deltaX, deltaY int) bool {
		c.scrollTo(c.first - deltaY*3)
		return true
	})
	c.combo = combo
	c.hover = -1
	return &c
}

func (c *completionPopup) setItems(indexes []int, chosen int, filtered bool) {
	c.indexes, c.chosen, c.filtered = indexes, chosen, filtered
	c.first = 0
	c.hover = -1
	rows := min(len(indexes), completionMaxVisible)
	c.SetSize(max(c.combo.Width(), 80), rows*ThemeRowHeight()+2)
	if chosen >= 0 {
		c.scrollTo(chosen - rows/2)
	}
}

// PopupFlipped opens the list above the field when it doesn't fit below.
func (c *completionPopup) PopupFlipped() (int, int) {
	return c.anchorX, c.anchorY - c.Height()
}

func (c *completionPopup) ProcessClosePopup() {
	c.combo.popup = nil
	c.combo.form.Update()
}

func (c *completionPopup) visibleRows() int {
	return min(len(c.indexes), completionMaxVisible)
}

func (c *completionPopup) rowAt(y int) int {
	row := (y-1)/ThemeRowHeight() + c.first
	if y < 1 || row >= len(c.indexes) || row >= c.first+c.visibleRows() {
		return -1
	}
	return row
}

func (c *completionPopup) scrollTo(first int) {
	c.first = max(0, min(first, len(c.indexes)-c.visibleRows()))
	c.form.Update()
}

// move moves the chosen row, keeping it visible.
func (c *completionPopup) move(delta int) {
	if len(c.indexes) == 0 {
		return
	}
	if c.chosen < 0 {
		c.chosen = 0
		if delta < 0 {
			c.chosen = len(c.indexes) - 1
		}
	} else {
		c.chosen = max(0, min(len(c.indexes)-1, c.chosen+delta))
	}
	if c.chosen < c.first {
		c.scrollTo(c.chosen)
	} else if c.chosen >= c.first+c.visibleRows() {
		c.scrollTo(c.chosen - c.visibleRows() + 1)
	}
	c.form.Update()
}

func (c *completionPopup) draw(cnv *Canvas) {
	p := CurrentPalette()
	rowH := ThemeRowHeight()
	needle := strings.ToLower(strings.TrimSpace(c.combo.Text()))
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.SetVAlign(VAlignCenter)
	cnv.SetHAlign(HAlignLeft)
	scrollbar := len(c.indexes) > c.visibleRows()
	width := c.Width() - 2
	if scrollbar {
		width -= 6
	}

	for i := 0; i < c.visibleRows(); i++ {
		row := c.first + i
		y := 1 + i*rowH
		text := c.combo.items[c.indexes[row]]
		textColor, matchColor := p.Text, p.Link
		switch {
		case row == c.chosen:
			cnv.FillRect(1, y, width, rowH, p.Highlight)
			textColor, matchColor = p.HighlightedText, p.HighlightedText
		case row == c.hover:
			cnv.FillRect(1, y, width, rowH, p.Selection)
		}
		x := comboBoxItemPadding
		cnv.SetColor(textColor)
		start := -1
		if c.filtered && needle != "" {
			start = strings.Index(strings.ToLower(text), needle)
		}
		if start < 0 || len(strings.ToLower(text)) != len(text) {
			cnv.DrawText(x, y, width-x, rowH, truncateTextToWidth(c.FontFamily(), c.FontSize(), text, width-x*2))
			continue
		}
		// The matched part in the accent color, underlined
		parts := []string{text[:start], text[start : start+len(needle)], text[start+len(needle):]}
		for pi, part := range parts {
			w, _, _ := MeasureText(c.FontFamily(), c.FontSize(), part)
			col := textColor
			if pi == 1 {
				col = matchColor
			}
			cnv.SetColor(col)
			cnv.DrawText(x, y, width-x, rowH, part)
			if pi == 1 {
				cnv.FillRect(x, y+rowH-5, w, 1, col)
			}
			x += w
		}
	}

	if scrollbar {
		trackH := c.Height() - 2
		thumbH := max(12, trackH*c.visibleRows()/len(c.indexes))
		thumbY := 1 + (trackH-thumbH)*c.first/max(1, len(c.indexes)-c.visibleRows())
		cnv.FillRoundedRectAA(c.Width()-6, thumbY, 4, thumbH, 2, withAlpha(p.Text, 90))
	}
	cnv.SetColor(p.Border)
	cnv.DrawRect(0, 0, c.Width(), c.Height())
}
