package ui

import (
	"reflect"
	"testing"
	"time"
)

// newTestForm makes a form with the widget in it, laid out.
func newTestForm(t *testing.T, w Widgeter) *Form {
	t.Helper()
	form := NewForm()
	form.processResize(500, 400)
	form.Panel().AddWidget(0, 0, w)
	form.Panel().AddVSpacer(1, 0)
	form.UpdateLayout()
	return form
}

func TestSlider(t *testing.T) {
	s := NewSlider()
	newTestForm(t, s)
	changes := 0
	s.SetOnValueChanged(func() { changes++ })
	s.SetRange(0, 10)
	s.SetStep(2)
	s.SetValue(5.2)
	if s.Value() != 6 {
		t.Errorf("snapped value = %v, want 6", s.Value())
	}
	s.ProcessKeyDown(KeyArrowRight, KeyModifiers{})
	s.ProcessKeyDown(KeyEnd, KeyModifiers{})
	s.ProcessKeyDown(KeyArrowRight, KeyModifiers{}) // at the end: no change
	if s.Value() != 10 || changes != 2 {
		t.Errorf("keys: value %v, changes %d", s.Value(), changes)
	}
	// A click on the track jumps there, a drag follows
	s.mouseDown(MouseButtonLeft, s.posOf(2), s.Height()/2, KeyModifiers{})
	if s.Value() != 2 {
		t.Errorf("click: value %v", s.Value())
	}
	s.form.mouseLeftButtonPressed = true
	s.mouseMove(s.Width()+100, 0, KeyModifiers{})
	if s.Value() != 10 {
		t.Errorf("drag past the end: value %v", s.Value())
	}
	// Vertical: the minimum at the bottom
	s.SetVertical(true)
	s.SetSize(30, 200)
	if s.valueAt(s.Height()-1) != 0 || s.valueAt(0) != 10 {
		t.Errorf("vertical ends: bottom %v, top %v", s.valueAt(s.Height()-1), s.valueAt(0))
	}
}

func TestToggleSwitch(t *testing.T) {
	sw := NewToggleSwitch("Wi-Fi")
	form := newTestForm(t, sw)
	changes := 0
	sw.SetOnStateChanged(func() { changes++ })
	sw.SetChecked(true)
	if !sw.Checked() || changes != 0 || sw.knob != 1 {
		t.Fatal("SetChecked: wrong state, a callback or an animation")
	}
	sw.Focus()
	typeKey(form, KeySpace, ' ', KeyModifiers{})
	if sw.Checked() || changes != 1 {
		t.Fatalf("Space: checked %v, changes %d", sw.Checked(), changes)
	}
	for i := 0; i < 10; i++ {
		sw.animate()
	}
	if sw.knob != 0 {
		t.Errorf("the knob stopped at %v", sw.knob)
	}
	sw.onMouseUp(MouseButtonLeft, 5, 5, KeyModifiers{})
	if !sw.Checked() {
		t.Error("a click didn't flip it")
	}
}

func TestLink(t *testing.T) {
	clicks := 0
	link := NewLink("Help", func() { clicks++ })
	form := newTestForm(t, link)
	link.Focus()
	typeKey(form, KeyEnter, 0, KeyModifiers{})
	link.onMouseUp(MouseButtonLeft, 2, 2, KeyModifiers{})
	link.onMouseUp(MouseButtonLeft, link.textWidth()+50, 2, KeyModifiers{}) // past the text
	if clicks != 2 || !link.Visited() {
		t.Errorf("clicks = %d, visited %v", clicks, link.Visited())
	}
}

func TestGroupBox(t *testing.T) {
	group := NewGroupBox("Connection")
	label := group.AddLabel(0, 0, "Host")
	newTestForm(t, group)
	if label.Y() < group.titleHeight() {
		t.Errorf("the child is under the title: y=%d, title height %d", label.Y(), group.titleHeight())
	}
	if group.MinHeight() < group.titleHeight()+label.MinHeight() {
		t.Errorf("min height %d doesn't fit the title and the child", group.MinHeight())
	}
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func TestCalendar(t *testing.T) {
	cal := NewCalendar()
	newTestForm(t, cal)
	var changed []time.Time
	cal.SetOnDateChanged(func(d time.Time) { changed = append(changed, d) })
	cal.SetDate(date(2024, time.January, 31))

	cal.SetFirstDayOfWeek(time.Monday) // Jan 1 2024 is a Monday
	if !cal.gridStart().Equal(date(2024, time.January, 1)) {
		t.Errorf("grid start (Monday) = %v", cal.gridStart())
	}
	cal.SetFirstDayOfWeek(time.Sunday)
	if !cal.gridStart().Equal(date(2023, time.December, 31)) {
		t.Errorf("grid start (Sunday) = %v", cal.gridStart())
	}

	cal.ProcessKeyDown(KeyPageDown, KeyModifiers{}) // Jan 31 -> Feb 29 (leap year)
	cal.ProcessKeyDown(KeyArrowRight, KeyModifiers{})
	cal.ProcessKeyDown(KeyPageUp, KeyModifiers{Ctrl: true}) // a year back
	want := []time.Time{date(2024, time.February, 29), date(2024, time.March, 1), date(2023, time.March, 1)}
	if !reflect.DeepEqual(changed, want) {
		t.Errorf("keyboard: %v", changed)
	}
	if cal.ShownMonth().Month() != time.March || cal.ShownMonth().Year() != 2023 {
		t.Errorf("shown month %v", cal.ShownMonth())
	}

	cal.SetMinDate(date(2023, time.March, 10))
	if !cal.Date().Equal(date(2023, time.March, 10)) {
		t.Errorf("min date: selection %v", cal.Date())
	}
	cal.ProcessKeyDown(KeyArrowLeft, KeyModifiers{})
	if !cal.Date().Equal(date(2023, time.March, 10)) {
		t.Errorf("moved before the min date: %v", cal.Date())
	}

	// A click on a day in the grid
	cell := cal.cellSize()
	start := cal.gridStart()
	target := date(2023, time.March, 15)
	offset := int(target.Sub(start).Hours() / 24)
	cal.mouseDown(MouseButtonLeft, cal.gridLeft()+(offset%7)*cell+cell/2, cal.gridTop()+(offset/7)*cell+cell/2, KeyModifiers{})
	if !cal.Date().Equal(target) {
		t.Errorf("click: %v, want %v", cal.Date(), target)
	}
}

func TestDatePicker(t *testing.T) {
	picker := NewDatePicker()
	form := newTestForm(t, picker)
	picker.SetDate(date(2024, time.May, 10))
	var changed []time.Time
	picker.SetOnDateChanged(func(d time.Time) { changed = append(changed, d) })

	picker.SetFormat("2006-01-02")
	if picker.Text() != "2024-05-10" {
		t.Errorf("text = %q", picker.Text())
	}
	picker.Focus()
	typeKey(form, KeyArrowUp, 0, KeyModifiers{})
	if !picker.Date().Equal(date(2024, time.May, 11)) {
		t.Errorf("Up: %v", picker.Date())
	}

	// Open, move in the calendar, Escape: nothing changes
	typeKey(form, KeyEnter, 0, KeyModifiers{})
	if !picker.IsPopupOpen() || form.FocusedWidget() != Widgeter(picker.popup.calendar) {
		t.Fatal("Enter didn't open the calendar with the focus in it")
	}
	typeKey(form, KeyArrowDown, 0, KeyModifiers{})
	typeKey(form, KeyEsc, 0, KeyModifiers{})
	if picker.IsPopupOpen() || !picker.Date().Equal(date(2024, time.May, 11)) || form.FocusedWidget() != Widgeter(picker) {
		t.Fatalf("after Escape: open %v, date %v", picker.IsPopupOpen(), picker.Date())
	}
	// Open, a week on, Enter: chosen
	typeKey(form, KeyF4, 0, KeyModifiers{})
	typeKey(form, KeyArrowDown, 0, KeyModifiers{})
	typeKey(form, KeyEnter, 0, KeyModifiers{})
	if picker.IsPopupOpen() || !picker.Date().Equal(date(2024, time.May, 18)) {
		t.Errorf("after Enter: open %v, date %v", picker.IsPopupOpen(), picker.Date())
	}
	if len(changed) != 2 {
		t.Errorf("changes = %v", changed)
	}

	SetLanguage("ru")
	defer SetLanguage("en")
	picker.SetFormat("")
	if picker.Text() != "18.05.2024" {
		t.Errorf("Russian text = %q", picker.Text())
	}
}

func TestTimePicker(t *testing.T) {
	tp := NewTimePicker()
	form := newTestForm(t, tp)
	changes := 0
	tp.SetOnTimeChanged(func() { changes++ })
	tp.Focus()
	for _, ch := range "0930" {
		typeKey(form, KeyA, ch, KeyModifiers{})
	}
	if tp.Text() != "09:30" {
		t.Errorf("typed 0930: %q", tp.Text())
	}
	// A digit no two-digit hour starts with moves on at once
	typeKey(form, KeyArrowLeft, 0, KeyModifiers{})
	typeKey(form, KeyA, '7', KeyModifiers{})
	if tp.Hour() != 7 || tp.segment != 1 {
		t.Errorf("typed 7: hour %d, segment %d", tp.Hour(), tp.segment)
	}
	// Up/Down wrap around
	typeKey(form, KeyArrowDown, 0, KeyModifiers{}) // minutes 30 -> 29
	typeKey(form, KeyArrowLeft, 0, KeyModifiers{})
	typeKey(form, KeyHome, 0, KeyModifiers{})
	typeKey(form, KeyArrowDown, 0, KeyModifiers{}) // hours 0 -> 23
	if tp.Text() != "23:29" {
		t.Errorf("after the arrows: %q", tp.Text())
	}
	tp.SetShowSeconds(true)
	tp.SetDuration(90*time.Minute + 5*time.Second)
	if tp.Text() != "01:30:05" || tp.Duration() != 90*time.Minute+5*time.Second {
		t.Errorf("duration: %q", tp.Text())
	}
	if changes == 0 {
		t.Error("no change reported")
	}
}

func TestEditableComboBox(t *testing.T) {
	combo := NewEditableComboBox()
	form := newTestForm(t, combo)
	combo.SetItems([]string{"Amsterdam", "Berlin", "Bern", "Lisbon", "London", "Oberhausen"})
	var selected []string
	combo.SetOnItemSelected(func(index int, text string) { selected = append(selected, text) })
	accepted := ""
	combo.SetOnAccept(func(text string) { accepted = text })
	combo.Focus()

	for _, ch := range "ber" {
		typeKey(form, KeyA, ch, KeyModifiers{})
	}
	if !combo.IsPopupOpen() {
		t.Fatal("typing didn't show suggestions")
	}
	var shown []string
	for _, i := range combo.popup.indexes {
		shown = append(shown, combo.items[i])
	}
	if !reflect.DeepEqual(shown, []string{"Berlin", "Bern", "Oberhausen"}) {
		t.Errorf("suggestions for \"ber\" = %v (starting with it first)", shown)
	}
	if form.FocusedWidget() != Widgeter(combo.edit) {
		t.Fatal("the list took the focus")
	}
	typeKey(form, KeyArrowDown, 0, KeyModifiers{})
	typeKey(form, KeyEnter, 0, KeyModifiers{})
	if combo.Text() != "Bern" || combo.IsPopupOpen() || !reflect.DeepEqual(selected, []string{"Bern"}) {
		t.Errorf("Down+Enter: text %q, open %v, selected %v", combo.Text(), combo.IsPopupOpen(), selected)
	}

	// Enter without a list accepts the text as it is
	typeKey(form, KeyEnter, 0, KeyModifiers{})
	if accepted != "Bern" {
		t.Errorf("accepted %q", accepted)
	}
	// F4 shows all, the current text chosen; Escape closes
	typeKey(form, KeyF4, 0, KeyModifiers{})
	if !combo.IsPopupOpen() || len(combo.popup.indexes) != 6 || combo.items[combo.popup.indexes[combo.popup.chosen]] != "Bern" {
		t.Fatal("F4 didn't show all the items with the current one chosen")
	}
	typeKey(form, KeyEsc, 0, KeyModifiers{})
	if combo.IsPopupOpen() || form.FocusedWidget() != Widgeter(combo.edit) {
		t.Error("Escape didn't close the list or lost the focus")
	}
	// No match: no list
	combo.SetText("")
	for _, ch := range "xyz" {
		typeKey(form, KeyA, ch, KeyModifiers{})
	}
	if combo.IsPopupOpen() {
		t.Error("a list without matches")
	}
}
