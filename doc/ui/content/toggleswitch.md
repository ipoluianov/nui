# ToggleSwitch

An on/off switch with a text, for settings that apply at once. A click, Space or Enter flips it;
the knob slides to its side.

```go
wifi := ui.NewToggleSwitch("Wi-Fi")
wifi.SetChecked(true)
wifi.SetOnStateChanged(func() { network.SetWifi(wifi.Checked()) })
```

- `Checked()` / `SetChecked(bool)` - `SetChecked` doesn't call the callback and doesn't animate.
- `Text()` / `SetText(text)` / `SetTextFunc(f)` (follows the language)
- `SetOnStateChanged(func())` - called when the user flips it.
