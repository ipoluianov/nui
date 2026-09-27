# Slider

A number in a range, chosen by dragging a thumb along a track.

```go
volume := ui.NewSlider()
volume.SetRange(0, 100)
volume.SetStep(1)
volume.SetTickInterval(10)
volume.SetOnValueChanged(func() { player.SetVolume(volume.Value()) })
```

- `SetRange(min, max)`, `SetMin`, `SetMax`, `Min()`, `Max()`
- `SetValue(v)` / `Value()` - kept in the range and on the step; `SetValue` doesn't call the callback.
- `SetStep(step)` - the value is a multiple of step from the minimum; 0 (default) allows any value.
- `SetPageStep(step)` - PageUp/PageDown; 0 (default) is 1/10 of the range.
- `SetTickInterval(v)` - ticks along the track; 0 (default) none.
- `SetVertical(bool)` - upright, the minimum at the bottom.
- `SetOnValueChanged(func())` - every change by the user, also while dragging.

Mouse: drag the thumb; a click on the track moves the thumb there and starts dragging; the wheel
moves by a step. Keys: arrows - a step, PageUp/PageDown - a page, Home/End - the ends.
