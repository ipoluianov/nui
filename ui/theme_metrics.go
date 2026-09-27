package ui

import "sync"

// Sizes derived from the theme font, so that controls fit their text at
// any font size (see ApplyBaseFontSize).

const (
	// Space above and below the text of a control: button, input field...
	controlPaddingY = 4
	// Space above and below the text of a table row or a menu item
	rowPaddingY = 3
	// Space between a check box or radio button indicator and its text
	indicatorTextGap = 8
)

var (
	lineHeightMu    sync.Mutex
	lineHeightKey   faceKey
	lineHeightValue int
)

// ThemeLineHeight returns the height of a line of text in the theme font.
func ThemeLineHeight() int {
	key := faceKey{ThemeFontFamily(), ThemeFontSize()}
	lineHeightMu.Lock()
	defer lineHeightMu.Unlock()
	if key != lineHeightKey || lineHeightValue == 0 {
		_, height, err := MeasureText(key.family, key.size, "Ag")
		if err != nil {
			height = int(key.size * 1.4)
		}
		lineHeightKey, lineHeightValue = key, height
	}
	return lineHeightValue
}

// ThemeControlHeight returns the height of buttons, input fields, combo
// boxes, check boxes and other single-line controls.
func ThemeControlHeight() int {
	return ThemeLineHeight() + controlPaddingY*2
}

// ThemeRowHeight returns the height of table rows and of menu and dropdown items.
func ThemeRowHeight() int {
	return ThemeLineHeight() + rowPaddingY*2
}

// ThemeIndicatorSize returns the size of a check box or radio button indicator.
func ThemeIndicatorSize() int {
	return ThemeLineHeight() * 4 / 5
}

// setThemeHeight makes the widget's minimum height (and maximum, if fixed)
// follow the theme font. Setting a height explicitly (SetMinSize,
// SetMinHeight...) turns it off.
func (c *Widget) setThemeHeight(height func() int, fixed bool) {
	c.themeHeight = height
	c.themeHeightFixed = fixed
	c.applyThemeHeight()
}

func (c *Widget) applyThemeHeight() {
	if c.themeHeight == nil {
		return
	}
	height := c.themeHeight()
	c.minHeight = height
	if c.themeHeightFixed {
		c.maxHeight = height
	}
}

// applyThemeMetrics updates the widget for a new theme font size. Widgets
// with more sizes derived from the font override it.
func (c *Widget) applyThemeMetrics() {
	c.applyThemeHeight()
}

// applyThemeMetricsTree updates the widget and all its children.
func applyThemeMetricsTree(w Widgeter) {
	w.applyThemeMetrics()
	for _, child := range w.Widgets() {
		applyThemeMetricsTree(child)
	}
}

// applyFontSize lays the form out again for a new theme font size.
func (c *Form) applyFontSize() {
	if c.topWidget == nil {
		return
	}
	// Popups are sized when opened
	c.closePopups()
	applyThemeMetricsTree(c.topWidget)
	c.UpdateLayout()
	c.forceUpdate()
}
