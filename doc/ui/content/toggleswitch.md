# ToggleSwitch

An on/off switch with a text, for settings that apply at once. A click, Space or Enter flips it;
the knob slides to its side.

```go
wifi := ui.NewToggleSwitch("Wi-Fi")
wifi.SetChecked(true)
wifi.OnActivated.Connect(func(on bool) { network.SetWifi(on) })
```

- `Checked()` / `SetChecked(bool)` - `SetChecked` doesn't animate; it fires `OnCheckedChanged`, not `OnActivated`.
- `Text()` / `SetText(text)` / `SetTextFunc(f)` (follows the language)
- `OnCheckedChanged(bool)` - any change; `OnActivated(bool)` - the user flipped it. See [events](events.md).
- `SetOnStateChanged(func())` - the old setter, called when the user flips it.
