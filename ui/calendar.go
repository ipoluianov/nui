package ui

import (
	"fmt"
	"time"
)

// Calendar shows a month as a grid of days to choose a date. The arrows in
// the header (and PageUp/PageDown, the mouse wheel) change the month; the
// arrow keys move the selection by a day or a week, Ctrl+PageUp/PageDown by
// a year, Home/End to the month's first/last day. Month and day names,
// and the first day of the week, follow the language (UIText,
// FirstDayOfWeek).
//
//	cal := ui.NewCalendar()
//	cal.SetOnDateChanged(func(d time.Time) { status.SetText(d.Format("2006-01-02")) })
type Calendar struct {
	Widget
	date     time.Time // selected, at midnight
	month    time.Time // the first day of the month shown
	minDate  time.Time // zero: no limit
	maxDate  time.Time
	firstDay time.Weekday
	// firstDaySet: SetFirstDayOfWeek was called, else FirstDayOfWeek()
	firstDaySet bool

	hoverDay    time.Time
	hoverPart   calendarPart
	pickOnClick bool // a click activates too (DatePicker's dropdown)

	onDateChanged   func(date time.Time)
	onDateActivated func(date time.Time)
}

type calendarPart int

const (
	calendarPartNone calendarPart = iota
	calendarPartPrev
	calendarPartNext
	calendarPartDay
	calendarPartToday
)

const calendarPadding = 6

// DateOnly returns the date of t at midnight in the local time zone.
func DateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func NewCalendar() *Calendar {
	var c Calendar
	c.InitWidget()
	c.SetTypeName("Calendar")
	c.SetCanBeFocused(true)
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.mouseDown)
	c.SetOnMouseDblClick(c.mouseDblClick)
	c.SetOnMouseMove(c.mouseMove)
	c.SetOnMouseLeave(func() {
		c.hoverPart = calendarPartNone
		c.form.Update()
	})
	c.SetOnMouseWheel(func(deltaX, deltaY int) bool {
		c.ShowMonth(c.month.Year(), c.month.Month()-time.Month(deltaY))
		return true
	})
	c.date = DateOnly(time.Now())
	c.month = firstOfMonth(c.date)
	c.updateSize()
	return &c
}

func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
}

// Date returns the selected date, at midnight local time.
func (c *Calendar) Date() time.Time {
	return c.date
}

// SetDate selects the date (kept within SetMinDate/SetMaxDate) and shows
// its month, without calling SetOnDateChanged's function.
func (c *Calendar) SetDate(date time.Time) {
	c.date = c.clamp(DateOnly(date))
	c.month = firstOfMonth(c.date)
	c.form.Update()
}

// SetMinDate and SetMaxDate limit the dates that can be chosen; a zero
// time removes the limit.
func (c *Calendar) SetMinDate(date time.Time) {
	c.minDate = zeroOrDate(date)
	c.SetDate(c.date)
}

func (c *Calendar) SetMaxDate(date time.Time) {
	c.maxDate = zeroOrDate(date)
	c.SetDate(c.date)
}

func zeroOrDate(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return DateOnly(t)
}

// ShowMonth shows the month without changing the selection; the month may
// be out of 1..12 (it's normalized, as by time.Date).
func (c *Calendar) ShowMonth(year int, month time.Month) {
	c.month = time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	c.form.Update()
}

// ShownMonth returns the first day of the month shown.
func (c *Calendar) ShownMonth() time.Time {
	return c.month
}

// SetFirstDayOfWeek overrides the language's first day of the week.
func (c *Calendar) SetFirstDayOfWeek(day time.Weekday) {
	c.firstDay, c.firstDaySet = day, true
	c.form.Update()
}

func (c *Calendar) firstDayOfWeek() time.Weekday {
	if c.firstDaySet {
		return c.firstDay
	}
	return FirstDayOfWeek()
}

// SetOnDateChanged sets the function called when the user selects another
// date: by a click, the keyboard or the Today button.
func (c *Calendar) SetOnDateChanged(f func(date time.Time)) {
	c.onDateChanged = f
}

// SetOnDateActivated sets the function called on a double click on a day
// or Enter.
func (c *Calendar) SetOnDateActivated(f func(date time.Time)) {
	c.onDateActivated = f
}

func (c *Calendar) inRange(d time.Time) bool {
	return (c.minDate.IsZero() || !d.Before(c.minDate)) && (c.maxDate.IsZero() || !d.After(c.maxDate))
}

func (c *Calendar) clamp(d time.Time) time.Time {
	if !c.minDate.IsZero() && d.Before(c.minDate) {
		return c.minDate
	}
	if !c.maxDate.IsZero() && d.After(c.maxDate) {
		return c.maxDate
	}
	return d
}

// selectDate is a selection made by the user.
func (c *Calendar) selectDate(d time.Time) {
	d = c.clamp(DateOnly(d))
	c.month = firstOfMonth(d)
	c.form.Update()
	if d.Equal(c.date) {
		return
	}
	c.date = d
	if c.onDateChanged != nil {
		c.onDateChanged(d)
	}
}

func (c *Calendar) activate() {
	if c.onDateActivated != nil {
		c.onDateActivated(c.date)
	}
}

// ---------------------------------------------------------------- Geometry

func (c *Calendar) cellSize() int {
	return ThemeRowHeight() + 4
}

func (c *Calendar) headerHeight() int {
	return ThemeRowHeight() + 6
}

func (c *Calendar) weekdaysTop() int {
	return calendarPadding + c.headerHeight()
}

func (c *Calendar) gridTop() int {
	return c.weekdaysTop() + ThemeRowHeight()
}

func (c *Calendar) footerTop() int {
	return c.gridTop() + 6*c.cellSize() + 2
}

func (c *Calendar) gridLeft() int {
	return (c.Width() - 7*c.cellSize()) / 2
}

func (c *Calendar) updateSize() {
	w := 7*c.cellSize() + calendarPadding*2
	h := c.footerTop() + ThemeRowHeight() + calendarPadding
	c.SetMinSize(w, h)
	c.SetMaxSize(w, h)
}

// gridStart is the date in the first cell: the first day of the week on or
// before the month's first day.
func (c *Calendar) gridStart() time.Time {
	offset := (int(c.month.Weekday()) - int(c.firstDayOfWeek()) + 7) % 7
	return c.month.AddDate(0, 0, -offset)
}

func (c *Calendar) arrowRects() (prevX, nextX, y, size int) {
	size = c.headerHeight() - 4
	y = calendarPadding + 2
	return c.gridLeft(), c.gridLeft() + 7*c.cellSize() - size, y, size
}

func (c *Calendar) partAt(x, y int) (calendarPart, time.Time) {
	prevX, nextX, ay, size := c.arrowRects()
	if y >= ay && y < ay+size {
		if x >= prevX && x < prevX+size {
			return calendarPartPrev, time.Time{}
		}
		if x >= nextX && x < nextX+size {
			return calendarPartNext, time.Time{}
		}
	}
	cell := c.cellSize()
	if x >= c.gridLeft() && x < c.gridLeft()+7*cell && y >= c.gridTop() && y < c.gridTop()+6*cell {
		col, row := (x-c.gridLeft())/cell, (y-c.gridTop())/cell
		return calendarPartDay, c.gridStart().AddDate(0, 0, row*7+col)
	}
	if tx, tw := c.todayRect(); y >= c.footerTop() && y < c.footerTop()+ThemeRowHeight() && x >= tx && x < tx+tw {
		return calendarPartToday, time.Time{}
	}
	return calendarPartNone, time.Time{}
}

func (c *Calendar) todayRect() (x, w int) {
	textW, _, _ := MeasureText(c.FontFamily(), c.FontSize(), UIText().Today)
	w = textW + 16
	return (c.Width() - w) / 2, w
}

// ---------------------------------------------------------------- Input

func (c *Calendar) mouseDown(button MouseButton, x, y int, mods KeyModifiers) bool {
	c.Focus()
	if button != MouseButtonLeft {
		return true
	}
	part, day := c.partAt(x, y)
	switch part {
	case calendarPartPrev:
		c.ShowMonth(c.month.Year(), c.month.Month()-1)
	case calendarPartNext:
		c.ShowMonth(c.month.Year(), c.month.Month()+1)
	case calendarPartDay:
		if c.inRange(day) {
			c.selectDate(day)
			if c.pickOnClick {
				c.activate()
			}
		}
	case calendarPartToday:
		if today := DateOnly(time.Now()); c.inRange(today) {
			c.selectDate(today)
			if c.pickOnClick {
				c.activate()
			}
		}
	}
	return true
}

func (c *Calendar) mouseDblClick(button MouseButton, x, y int, mods KeyModifiers) bool {
	part, day := c.partAt(x, y)
	switch part {
	case calendarPartPrev, calendarPartNext:
		return c.mouseDown(button, x, y, mods) // the second click of a fast double click
	case calendarPartDay:
		if c.inRange(day) {
			c.selectDate(day)
			c.activate()
		}
	}
	return true
}

func (c *Calendar) mouseMove(x, y int, mods KeyModifiers) bool {
	part, day := c.partAt(x, y)
	if part != c.hoverPart || !day.Equal(c.hoverDay) {
		c.hoverPart, c.hoverDay = part, day
		c.form.Update()
	}
	cursor := MouseCursorArrow
	if part != calendarPartNone && (part != calendarPartDay || c.inRange(day)) {
		cursor = MouseCursorPointer
	}
	c.SetMouseCursor(cursor)
	return true
}

func (c *Calendar) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	if c.onKeyDown != nil && c.onKeyDown(key, mods) {
		return true
	}
	d := c.date
	switch key {
	case KeyArrowLeft:
		d = d.AddDate(0, 0, -1)
	case KeyArrowRight:
		d = d.AddDate(0, 0, 1)
	case KeyArrowUp:
		d = d.AddDate(0, 0, -7)
	case KeyArrowDown:
		d = d.AddDate(0, 0, 7)
	case KeyPageUp, KeyPageDown:
		months := -1
		if key == KeyPageDown {
			months = 1
		}
		if mods.Ctrl || mods.Shift {
			months *= 12
		}
		d = addMonthsClamped(d, months)
	case KeyHome:
		d = firstOfMonth(d)
	case KeyEnd:
		d = firstOfMonth(d).AddDate(0, 1, -1)
	case KeyEnter:
		c.activate()
		return true
	default:
		return false
	}
	c.selectDate(d)
	return true
}

// addMonthsClamped moves by months keeping the day within the new month
// (Jan 31 + 1 month = Feb 28/29, not Mar 3).
func addMonthsClamped(d time.Time, months int) time.Time {
	first := time.Date(d.Year(), d.Month()+time.Month(months), 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(d.Day(), last), 0, 0, 0, 0, time.Local)
}

// ---------------------------------------------------------------- Painting

func (c *Calendar) draw(cnv *Canvas) {
	p := CurrentPalette()
	texts := UIText()
	focused := c.IsFocused()
	cell := c.cellSize()
	left := c.gridLeft()

	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.SetVAlign(VAlignCenter)

	// Header: arrows and "Month Year"
	prevX, nextX, ay, size := c.arrowRects()
	for _, a := range []struct {
		x    int
		part calendarPart
		next bool
	}{{prevX, calendarPartPrev, false}, {nextX, calendarPartNext, true}} {
		if c.hoverPart == a.part {
			cnv.FillRoundedRectAA(a.x, ay, size, size, themeControlRadius, hoverColor(p.Base, p.Text))
		}
		cx, cy := a.x+size/2, ay+size/2
		const half = 4
		if a.next {
			cnv.FillTriangle(cx-half/2, cy-half, cx-half/2, cy+half, cx+half/2+1, cy, p.Text)
		} else {
			cnv.FillTriangle(cx+half/2, cy-half, cx+half/2, cy+half, cx-half/2-1, cy, p.Text)
		}
	}
	cnv.SetHAlign(HAlignCenter)
	cnv.SetColor(p.Text)
	title := fmt.Sprintf("%s %d", texts.MonthNames[c.month.Month()-1], c.month.Year())
	cnv.DrawText(prevX+size, calendarPadding, nextX-prevX-size, c.headerHeight(), title)

	// Weekday names
	first := c.firstDayOfWeek()
	weekdayColor := MixColors(p.Text, p.Base, 0.4)
	for i := 0; i < 7; i++ {
		cnv.SetColor(weekdayColor)
		cnv.DrawText(left+i*cell, c.weekdaysTop(), cell, ThemeRowHeight(), texts.WeekdaysShort[(int(first)+i)%7])
	}
	cnv.FillRect(left, c.gridTop()-1, 7*cell, 1, p.Divider)

	// Days
	today := DateOnly(time.Now())
	day := c.gridStart()
	for row := 0; row < 6; row++ {
		for col := 0; col < 7; col++ {
			x, y := left+col*cell, c.gridTop()+row*cell
			const inset = 2
			inMonth := day.Month() == c.month.Month()
			enabled := c.inRange(day)
			selected := day.Equal(c.date)

			textColor := p.Text
			switch {
			case !enabled:
				textColor = p.DisabledText
			case !inMonth:
				textColor = MixColors(p.Text, p.Base, 0.55)
			}
			switch {
			case selected && focused:
				cnv.FillRoundedRectAA(x+inset, y+inset, cell-inset*2, cell-inset*2, themeControlRadius+1, p.Highlight)
				textColor = p.HighlightedText
			case selected:
				cnv.FillRoundedRectAA(x+inset, y+inset, cell-inset*2, cell-inset*2, themeControlRadius+1, p.Selection)
			case enabled && c.hoverPart == calendarPartDay && day.Equal(c.hoverDay):
				cnv.FillRoundedRectAA(x+inset, y+inset, cell-inset*2, cell-inset*2, themeControlRadius+1, hoverColor(p.Base, p.Text))
			}
			if day.Equal(today) && !(selected && focused) {
				cnv.SetColor(p.Highlight)
				cnv.DrawRect(x+inset, y+inset, cell-inset*2, cell-inset*2)
			}
			cnv.SetColor(textColor)
			cnv.DrawText(x, y, cell, cell, fmt.Sprint(day.Day()))
			day = day.AddDate(0, 0, 1)
		}
	}

	// Today
	tx, tw := c.todayRect()
	todayColor := p.Link
	if !c.inRange(today) {
		todayColor = p.DisabledText
	}
	if c.hoverPart == calendarPartToday {
		cnv.FillRoundedRectAA(tx, c.footerTop(), tw, ThemeRowHeight(), themeControlRadius, hoverColor(p.Base, p.Text))
	}
	cnv.SetColor(todayColor)
	cnv.DrawText(tx, c.footerTop(), tw, ThemeRowHeight(), texts.Today)

	if focused {
		cnv.SetColor(withAlpha(p.Highlight, 90))
		cnv.DrawRect(0, 0, c.Width(), c.Height())
	}
}

func (c *Calendar) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.updateSize()
}
