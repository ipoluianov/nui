# Keyboard & mouse types

Defined in the native layer and re-exported by `ui` ([ui/input.go](../../../ui/input.go)),
so applications only import `ui`.

```go
type Key          // key code; key.String() -> "Enter", "F1", "A"
type KeyModifiers // struct{ Shift, Ctrl, Alt, Cmd bool }; mods.String() -> "Shift Ctrl"
type MouseButton  // ui.MouseButtonLeft / Middle / Right
type MouseCursor  // ui.MouseCursorArrow / Pointer / ResizeHor / ResizeVer / IBeam / Cross / Move / NotDefined
```

## Usage

For a shortcut, `form.AddShortcut` (or an [Action](actions.md)) is simpler: it takes a text like `"Ctrl+Shift+Z"`, `"F5"` or `"Mod+S"` ("Mod" is Cmd on macOS and Ctrl elsewhere) and matches the key by its place, in any keyboard layout. `ui.ParseShortcut` parses such a text into a `ui.Shortcut`.

```go
form.AddShortcut("Mod+S", save)
```

To see every key before the shortcuts:

```go
form.SetOnGlobalKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool {
	if key == ui.KeyS && mods.Ctrl {
		save()
		return true
	}
	return false
})
```

Inside a widget, the key and mouse signals carry these types: `OnKeyDown` gets a `*ui.KeyEvent{Key, Mods, Handled}`, `OnMouseDown` a `*ui.MouseEvent{Button, X, Y, Mods, Handled}` (see [Events](events.md)). `w.SetMouseCursor(ui.MouseCursorPointer)` sets the cursor over a widget.

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
