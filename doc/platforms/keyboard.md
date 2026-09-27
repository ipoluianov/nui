# Keyboard (`platforms`)

## Types

```go
type Key int

type KeyModifiers struct {
	Shift bool
	Ctrl  bool
	Alt   bool
	Cmd   bool // macOS Command key
}

mods.String() string // e.g. "Shift Ctrl"
key.String() string  // e.g. "Enter", "F1", "A"
```

## Usage

```go
win.OnKeyDown(func(key platforms.Key, mods platforms.KeyModifiers) bool {
	switch key {
	case platforms.KeyEsc:
		win.Close()
	case platforms.KeyS:
		if mods.Ctrl {
			save()
		}
	}
	return true // true = handled, don't pass to OS default handling
})

win.OnKeyUp(func(key platforms.Key, mods platforms.KeyModifiers) {})
win.OnChar(func(char rune) { /* text input, respects layout */ })
```

`win.KeyModifiers()` reads current modifier state at any time (not just inside a key event).

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
KeyCapsLock, KeyNumLock, KeyScrollLock, KeyPrintScreen, KeyPauseBreak
KeyMinus, KeyEqual, KeyLeftBracket, KeyRightBracket, KeyBackslash
KeySemicolon, KeyApostrophe, KeyComma, KeyDot, KeySlash, KeyGrave
KeyCommand, KeyFunction // macOS only
```

Full list: [internal/platforms/keys.go](../../internal/platforms/keys.go). Values match Windows virtual-key codes; other platforms translate to the same `Key` values.
