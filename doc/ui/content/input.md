# Keyboard & mouse types

Defined in the native layer and re-exported by `ui` ([ui/input.go](../../../ui/input.go)),
so applications only import `ui`.

```go
type Key          // key code; key.String() -> "Enter", "F1", "A"
type KeyModifiers // struct{ Shift, Ctrl, Alt, Cmd bool }; mods.String() -> "Shift Ctrl"
type MouseButton  // ui.MouseButtonLeft / Middle / Right
type MouseCursor  // ui.MouseCursorArrow / Pointer / ResizeHor / ResizeVer / IBeam / NotDefined
```

## Usage

```go
form.SetOnGlobalKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool {
	if key == ui.KeyS && mods.Ctrl {
		save()
		return true
	}
	return false
})
```

## Key constants (selected)

```
KeyEsc, KeyEnter, KeyTab, KeyBackspace, KeySpace
KeyF1..KeyF24
Key0..Key9
KeyA..KeyZ
KeyArrowUp, KeyArrowDown, KeyArrowLeft, KeyArrowRight
KeyInsert, KeyDelete, KeyHome, KeyEnd, KeyPageUp, KeyPageDown
KeyShift, KeyCtrl, KeyAlt, KeyWin, KeyContextMenu
KeyNumpad0..KeyNumpad9, KeyNumpadSlash, KeyNumpadAsterisk, KeyNumpadMinus, KeyNumpadPlus, KeyNumpadDot
KeyCommand, KeyFunction // macOS only
```

Full list: [ui/input.go](../../../ui/input.go).
