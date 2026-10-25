package ui

import "math"

// ScrollBar is a scroll bar of its own, for a view that scrolls in units of
// its own - the rows of a hex editor, the samples of a wave - rather than
// in pixels of a widget's content (those have their scroll bars built in).
// The value goes from Min to Max, Max being the first unit of the last
// page; the arrows move it by the step, the track by the page.
//
//	bar := ui.NewScrollBar(true)
//	bar.SetRange(0, rows-visibleRows)
//	bar.SetPageStep(visibleRows)
//	bar.OnValueChanged.Connect(func(row int) { view.SetFirstRow(row) })
type ScrollBar struct {
	Widget

	// OnValueChanged: the value changed, by the user or by SetValue
	OnValueChanged Signal[int]
	// OnActivated: the user moved the bar
	OnActivated Signal[int]

	vertical       bool
	min, max, page int
	step, value    int
	applying       bool // the value is being shown: no signal of it
}

// NewScrollBar makes a vertical or a horizontal bar, of the range 0..100
func NewScrollBar(vertical bool) *ScrollBar {
	c := &ScrollBar{vertical: vertical, max: 100, page: 10, step: 1}
	c.InitWidget()
	c.SetTypeName("ScrollBar")
	if vertical {
		c.SetAllowScroll(false, true)
		c.SetMinSize(c.scrollBarYSize, 3*c.scrollBarYSize)
		c.SetMaxSize(c.scrollBarYSize, MaxInt/2)
		c.SetYExpandable(true)
	} else {
		c.SetAllowScroll(true, false)
		c.SetMinSize(3*c.scrollBarXSize, c.scrollBarXSize)
		c.SetMaxSize(MaxInt/2, c.scrollBarXSize)
		c.SetXExpandable(true)
	}
	c.SetOnScrollChanged(func(scrollX, scrollY int) { c.scrolled(scrollX, scrollY) })
	c.SetOnMouseWheel(func(deltaX, deltaY int) bool {
		delta := deltaY
		if !c.vertical && deltaX != 0 {
			delta = deltaX
		}
		c.setValue(c.value-delta*3*c.step, true)
		return true
	})
	c.OnResize.Connect(func(Size) { c.apply() })
	c.apply()
	return c
}

// SetRange sets the smallest and the largest value
func (c *ScrollBar) SetRange(lo, hi int) {
	c.min, c.max = lo, max(lo, hi)
	c.apply()
	c.setValue(c.value, false)
}

func (c *ScrollBar) Min() int { return c.min }
func (c *ScrollBar) Max() int { return c.max }

// SetPageStep sets the page: the units shown at once, what the track moves
// by and the thumb's share of the bar
func (c *ScrollBar) SetPageStep(page int) {
	c.page = max(1, page)
	c.apply()
}

func (c *ScrollBar) PageStep() int { return c.page }

// SetStep sets what the arrows move the value by
func (c *ScrollBar) SetStep(step int) {
	c.step = max(1, step)
	c.apply()
}

// SetValue moves the bar to the value, kept in the range
func (c *ScrollBar) SetValue(value int) {
	c.setValue(value, false)
}

func (c *ScrollBar) Value() int { return c.value }

func (c *ScrollBar) setValue(value int, byUser bool) {
	value = max(c.min, min(value, c.max))
	if value == c.value {
		return
	}
	c.value = value
	c.apply()
	c.OnValueChanged.Emit(value)
	if byUser {
		c.OnActivated.Emit(value)
	}
}

// unit is the pixels a unit of the value takes on the bar
func (c *ScrollBar) unit() float64 {
	length := c.h
	if !c.vertical {
		length = c.w
	}
	return float64(max(1, length)) / float64(c.page)
}

// apply shows the range and the value as the content and the scroll of the
// built-in bar: the bar's length is a page
func (c *ScrollBar) apply() {
	u := c.unit()
	content := int(math.Round(float64(c.max-c.min+c.page) * u))
	scroll := int(math.Round(float64(c.value-c.min) * u))
	c.scrollLineStep = max(1, int(math.Round(float64(c.step)*u)))
	c.applying = true
	if c.vertical {
		c.innerHeight, c.innerWidth = content, 0
		c.setScrollY(scroll)
	} else {
		c.innerWidth, c.innerHeight = content, 0
		c.setScrollX(scroll)
	}
	c.applying = false
	c.form.Update()
}

// scrolled: the user moved the built-in bar
func (c *ScrollBar) scrolled(scrollX, scrollY int) {
	if c.applying {
		return
	}
	pos := scrollY
	if !c.vertical {
		pos = scrollX
	}
	c.setValue(c.min+int(math.Round(float64(pos)/c.unit())), true)
}
