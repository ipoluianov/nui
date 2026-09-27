package ui

import (
	"image/color"
	"time"
)

// DatePicker is a field showing a date; a click (or Enter, Space, F4)
// drops a Calendar down to choose another one. With the dropdown closed
// Up/Down change the date by a day, PageUp/PageDown by a month.
//
//	due := ui.NewDatePicker()
//	due.SetDate(time.Now().AddDate(0, 0, 7))
//	due.SetOnDateChanged(func(d time.Time) { task.Due = d })
type DatePicker struct {
	Widget
	date    time.Time
	layout  string // "" follows the language (DefaultDateLayout)
	minDate time.Time
	maxDate time.Time

	popup         *datePickerPopup
	onDateChanged func(date time.Time)
}

func NewDatePicker() *DatePicker {
	var c DatePicker
	c.InitWidget()
	c.SetTypeName("DatePicker")
	c.SetMinWidth(DefaultComboBoxMinWidth)
	c.SetMaxWidth(10000)
	c.setThemeHeight(ThemeControlHeight, true)
	c.SetCanBeFocused(true)
	c.SetMouseCursor(MouseCursorPointer)
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		if button == MouseButtonLeft && c.Enabled() {
			c.OpenPopup()
		}
		return true
	})
	c.date = DateOnly(time.Now())
	return &c
}

// Date returns the chosen date, at midnight local time.
func (c *DatePicker) Date() time.Time {
	return c.date
}

// SetDate sets the date (kept within the limits) without calling
// SetOnDateChanged's function.
func (c *DatePicker) SetDate(date time.Time) {
	c.date = c.clamp(DateOnly(date))
	c.form.Update()
}

// SetMinDate and SetMaxDate limit the dates; a zero time removes the limit.
func (c *DatePicker) SetMinDate(date time.Time) {
	c.minDate = zeroOrDate(date)
	c.SetDate(c.date)
}

func (c *DatePicker) SetMaxDate(date time.Time) {
	c.maxDate = zeroOrDate(date)
	c.SetDate(c.date)
}

// SetFormat sets the time.Format layout of the date shown; "" (the
// default) follows the language, see DefaultDateLayout.
func (c *DatePicker) SetFormat(layout string) {
	c.layout = layout
	c.form.Update()
}

// Text returns the date as shown.
func (c *DatePicker) Text() string {
	layout := c.layout
	if layout == "" {
		layout = DefaultDateLayout()
	}
	return c.date.Format(layout)
}

// SetOnDateChanged sets the function called when the user chooses another date.
func (c *DatePicker) SetOnDateChanged(f func(date time.Time)) {
	c.onDateChanged = f
}

func (c *DatePicker) IsPopupOpen() bool {
	return c.popup != nil
}

func (c *DatePicker) clamp(d time.Time) time.Time {
	if !c.minDate.IsZero() && d.Before(c.minDate) {
		return c.minDate
	}
	if !c.maxDate.IsZero() && d.After(c.maxDate) {
		return c.maxDate
	}
	return d
}

func (c *DatePicker) changeDate(d time.Time) {
	d = c.clamp(DateOnly(d))
	if d.Equal(c.date) {
		return
	}
	c.date = d
	c.form.Update()
	if c.onDateChanged != nil {
		c.onDateChanged(d)
	}
}

// OpenPopup drops the calendar down, with the keyboard focus in it.
func (c *DatePicker) OpenPopup() {
	if c.form == nil || c.popup != nil {
		return
	}
	popup := newDatePickerPopup(c)
	c.popup = popup
	x, y := c.RectClientAreaOnWindow()
	popup.anchorX, popup.anchorY, popup.anchorW = x, y, c.Width()
	popup.SetPosition(x, y+c.Height())
	c.form.OpenPopup(popup)
	popup.calendar.Focus()
}

// ClosePopup closes the dropdown without changing the date.
func (c *DatePicker) ClosePopup() {
	if c.popup != nil && c.form.TopPopupWidget() == Widgeter(c.popup) {
		c.form.CloseTopPopup()
	}
}

func (c *DatePicker) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	if c.onKeyDown != nil && c.onKeyDown(key, mods) {
		return true
	}
	if !c.Enabled() {
		return false
	}
	switch key {
	case KeyEnter, KeySpace, KeyF4:
		c.OpenPopup()
	case KeyArrowDown:
		if mods.Alt {
			c.OpenPopup()
		} else {
			c.changeDate(c.date.AddDate(0, 0, -1))
		}
	case KeyArrowUp:
		c.changeDate(c.date.AddDate(0, 0, 1))
	case KeyPageUp:
		c.changeDate(addMonthsClamped(c.date, 1))
	case KeyPageDown:
		c.changeDate(addMonthsClamped(c.date, -1))
	default:
		return false
	}
	return true
}

func (c *DatePicker) draw(cnv *Canvas) {
	foreColor := drawPickerField(cnv, &c.Widget, c.popup != nil)
	iconSize := ThemeIndicatorSize()
	x := themeTextInset
	y := (c.Height() - iconSize) / 2
	drawCalendarIcon(cnv, x, y, iconSize, foreColor)

	textX := x + iconSize + 8
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(foreColor)
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.DrawText(textX, 0, c.Width()-textX-comboBoxArrowPadding*2-comboBoxArrowWidth, c.Height(), c.Text())
	drawPickerArrow(cnv, &c.Widget, foreColor)
}

// drawPickerField draws the frame of a field that drops something down
// (DatePicker, ColorPicker...) and returns the text color.
func drawPickerField(cnv *Canvas, w *Widget, open bool) (foreColor color.RGBA) {
	p := CurrentPalette()
	fill, border := p.Button, p.Border
	foreColor = colorToRGBA(w.ForegroundColor())
	switch {
	case !w.Enabled():
		foreColor = p.DisabledText
	case w.IsHovered() || open:
		fill = hoverColor(fill, foreColor)
	}
	if (w.IsFocused() || open) && w.Enabled() {
		border = p.Highlight
	}
	cnv.FillFrame(0, 0, w.Width(), w.Height(), themeControlRadius, fill, border)
	return foreColor
}

func drawPickerArrow(cnv *Canvas, w *Widget, col color.RGBA) {
	x := w.Width() - comboBoxArrowPadding - comboBoxArrowWidth
	y := (w.Height() - comboBoxArrowHeight) / 2
	cnv.FillTriangle(x, y, x+comboBoxArrowWidth, y, x+comboBoxArrowWidth/2, y+comboBoxArrowHeight, col)
}

// drawCalendarIcon draws a small calendar sheet.
func drawCalendarIcon(cnv *Canvas, x, y, size int, col color.RGBA) {
	cnv.SetColor(col)
	cnv.DrawRect(x, y+2, size, size-2)
	cnv.FillRect(x, y+2, size, 3, col)
	cnv.FillRect(x+3, y, 2, 4, col)
	cnv.FillRect(x+size-5, y, 2, 4, col)
	for row := 0; row < 2; row++ {
		for col2 := 0; col2 < 3; col2++ {
			cnv.FillRect(x+3+col2*(size-6)/3, y+7+row*(size-9)/2, 2, 2, col)
		}
	}
}

// datePickerPopup is the dropdown: a Calendar.
type datePickerPopup struct {
	Widget
	picker   *DatePicker
	calendar *Calendar
	anchorX  int
	anchorY  int
	anchorW  int
}

func newDatePickerPopup(picker *DatePicker) *datePickerPopup {
	var c datePickerPopup
	c.InitWidget()
	c.SetTypeName("DatePickerPopup")
	c.SetAbsolutePositioning(true)
	c.SetRole("popup")
	c.SetAutoFillBackground(true)
	c.SetOnPostPaint(func(cnv *Canvas) {
		cnv.SetColor(CurrentPalette().Border)
		cnv.DrawRect(0, 0, c.Width(), c.Height())
	})
	c.picker = picker

	cal := NewCalendar()
	cal.pickOnClick = true
	cal.SetMinDate(picker.minDate)
	cal.SetMaxDate(picker.maxDate)
	cal.SetDate(picker.date)
	cal.SetOnDateActivated(func(d time.Time) {
		picker.changeDate(d)
		picker.ClosePopup()
	})
	c.calendar = cal
	c.AddWidget(0, 0, cal)
	w, h := cal.MinWidth(), cal.MinHeight()
	cal.SetPosition(1, 1)
	cal.SetSize(w, h)
	c.SetSize(w+2, h+2)
	return &c
}

// PopupFlipped opens the calendar above the field when it doesn't fit below.
func (c *datePickerPopup) PopupFlipped() (int, int) {
	return c.anchorX + c.anchorW - c.Width(), c.anchorY - c.Height()
}

func (c *datePickerPopup) ProcessClosePopup() {
	c.picker.popup = nil
	c.picker.form.Update()
}
