# DatePicker and Calendar

## Calendar

A month as a grid of days. The header arrows, PageUp/PageDown and the mouse wheel change the month;
arrows move the selection by a day / a week, Ctrl+PageUp/PageDown by a year, Home/End to the first /
last day of the month. "Today" selects today.

```go
cal := ui.NewCalendar()
cal.SetOnDateChanged(func(d time.Time) { ... })   // a click, the keyboard, Today
cal.SetOnDateActivated(func(d time.Time) { ... }) // a double click, Enter
```

- `Date()` / `SetDate(t)` - dates are at midnight local time (`ui.DateOnly(t)`).
- `SetMinDate(t)`, `SetMaxDate(t)` - a zero time removes the limit; days out of range are grayed out.
- `ShowMonth(year, month)`, `ShownMonth()`, `SetFirstDayOfWeek(time.Weekday)`

## DatePicker

A field showing a date; a click, Enter, Space, F4 or Alt+Down drops a calendar down (with the focus
in it): a click on a day or Enter chooses, Escape closes without changing. With the dropdown closed
Up/Down change the date by a day, PageUp/PageDown by a month.

```go
due := ui.NewDatePicker()
due.SetDate(time.Now().AddDate(0, 0, 7))
due.SetOnDateChanged(func(d time.Time) { task.Due = d })
```

- `Date()` / `SetDate(t)`, `SetMinDate`, `SetMaxDate`, `Text()` - the date as shown.
- `SetFormat(layout)` - a `time.Format` layout; `""` (default) follows the language: `ui.DefaultDateLayout()`.
- `OpenPopup()`, `ClosePopup()`, `IsPopupOpen()`

## Language

Month and day names come from `ui.UIText()` (`MonthNames`, `WeekdaysShort`, `Today`; English,
Russian and Chinese built in, others via `ui.RegisterUIStrings`). The first day of the week is
`ui.FirstDayOfWeek()`: Sunday for English (US), Japanese, Korean and a few others, Monday elsewhere.
