# TimePicker

A time of day, 24-hour, edited in segments: `HH:MM` or `HH:MM:SS`.

```go
start := ui.NewTimePicker()
start.SetTime(9, 30, 0)
start.OnActivated.Connect(func(d time.Duration) { meeting.Start = d }) // the user changed it
```

| Input | Action |
|---|---|
| Left / Right, a click | choose the segment |
| Up / Down, the wheel, the arrows at the right | change it by 1 (wrapping around) |
| PageUp / PageDown | by 10 |
| digits | type it: two digits, or one no two-digit value starts with ("7" in hours), move to the next segment |
| `:` `.` space | next segment |
| Home / End, Backspace / Delete | the minimum / maximum, 0 |

- `OnTimeChanged` (any change, `SetTime` too) and `OnActivated` (the user's): the time since midnight.
- `SetTime(h, m, s)` (the values are clamped, seconds are 0 without `SetShowSeconds(true)`), `Hour()`, `Minute()`, `Second()`, `Text()`
- `Duration()` / `SetDuration(d)` - the time since midnight.
- `SetShowSeconds(bool)`; the old `SetOnTimeChanged(func())` - called when the user changes the time.
