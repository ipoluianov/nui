package nui

type Key int

type KeyModifiers struct {
	Shift bool
	Ctrl  bool
	Alt   bool
	Cmd   bool
}

func (c KeyModifiers) String() string {
	str := ""
	if c.Shift {
		str += "Shift"
	}
	if c.Ctrl {
		if str != "" {
			str += " "
		}
		str += "Ctrl"
	}
	if c.Alt {
		if str != "" {
			str += " "
		}
		str += "Alt"
	}
	if c.Cmd {
		if str != "" {
			str += " "
		}
		str += "Cmd"
	}
	return str
}

const (
	KeyEsc = 0x01

	KeyF1  = 0x3B
	KeyF2  = 0x3C
	KeyF3  = 0x3D
	KeyF4  = 0x3E
	KeyF5  = 0x3F
	KeyF6  = 0x40
	KeyF7  = 0x41
	KeyF8  = 0x42
	KeyF9  = 0x43
	KeyF10 = 0x44
	KeyF11 = 0x57
	KeyF12 = 0x58

	Key1 = 0x02
	Key2 = 0x03
	Key3 = 0x04
	Key4 = 0x05
	Key5 = 0x06
	Key6 = 0x07
	Key7 = 0x08
	Key8 = 0x09
	Key9 = 0x0A
	Key0 = 0x0B

	KeyMinus          = 0x0C
	KeyEqual          = 0x0D
	KeyBackspace      = 0x0E
	KeyTab            = 0x0F
	KeyQ              = 0x10
	KeyW              = 0x11
	KeyE              = 0x12
	KeyR              = 0x13
	KeyT              = 0x14
	KeyY              = 0x15
	KeyU              = 0x16
	KeyI              = 0x17
	KeyO              = 0x18
	KeyP              = 0x19
	KeyLeftBracket    = 0x1A
	KeyRightBracket   = 0x1B
	KeyEnter          = 0x1C
	KeyCtrl           = 0x1D
	KeyA              = 0x1E
	KeyS              = 0x1F
	KeyD              = 0x20
	KeyF              = 0x21
	KeyG              = 0x22
	KeyH              = 0x23
	KeyJ              = 0x24
	KeyK              = 0x25
	KeyL              = 0x26
	KeySemicolon      = 0x27
	KeyApostrophe     = 0x28
	KeyGrave          = 0x29
	KeyShift          = 0x2A
	KeyBackslash      = 0x2B
	KeyZ              = 0x2C
	KeyX              = 0x2D
	KeyC              = 0x2E
	KeyV              = 0x2F
	KeyB              = 0x30
	KeyN              = 0x31
	KeyM              = 0x32
	KeyComma          = 0x33
	KeyDot            = 0x34
	KeySlash          = 0x35
	KeyNumpadAsterisk = 0x37
	KeyAlt            = 0x38
	KeySpace          = 0x39
	KeyCapsLock       = 0x3A
	KeyNumLock        = 0x45
	KeyScrollLock     = 0x46
	KeyNumpad7        = 0x47
	KeyNumpad8        = 0x48
	KeyNumpad9        = 0x49
	KeyNumpadMinus    = 0x4A
	KeyNumpad4        = 0x4B
	KeyNumpad5        = 0x4C
	KeyNumpad6        = 0x4D
	KeyNumpadPlus     = 0x4E
	KeyNumpad1        = 0x4F
	KeyNumpad2        = 0x50
	KeyNumpad3        = 0x51
	KeyNumpad0        = 0x52
	KeyNumpadDot      = 0x53
	KeyNumpadDivide   = 0x5A
	KeyNumpadMultiply = 0x5F

	KeyInsert     = 0xE052
	KeyDelete     = 0xE053
	KeyHome       = 0xE047
	KeyEnd        = 0xE04F
	KeyPageUp     = 0xE049
	KeyPageDown   = 0xE051
	KeyArrowUp    = 0xE048
	KeyArrowLeft  = 0xE04B
	KeyArrowDown  = 0xE050
	KeyArrowRight = 0xE04D

	// Numpad - special keys
	KeyNumpadEnter = 0xE01C
	KeyNumpadSlash = 0xE035

	// Special keys
	KeyPause       = 0xE11D45 // Very special: composite scan-code
	KeyPrintScreen = 0xE037   // Requires shift-logic

	// Mac OS
	KeyCommand     = 0xCC37
	KeyOption      = 0xCC3A
	KeyRightOption = 0xCC3D
	KeyFunction    = 0xCC3F
	KeyVolumeUp    = 0xCC48
	KeyVolumeDown  = 0xCC49
	KeyMute        = 0xCC4A
	KeyF13         = 0xCC69
	KeyF14         = 0xCC6B
	KeyF15         = 0xCC71
	KeyF16         = 0xCC6A
	KeyF17         = 0xCC40
	KeyF18         = 0xCC4F
	KeyF19         = 0xCC50
	KeyF20         = 0xCC5A
	KeyHelp        = 0xCC72
)

var keyNames = map[Key]string{
	KeyEsc:            "Esc",
	Key1:              "1",
	Key2:              "2",
	Key3:              "3",
	Key4:              "4",
	Key5:              "5",
	Key6:              "6",
	Key7:              "7",
	Key8:              "8",
	Key9:              "9",
	Key0:              "0",
	KeyMinus:          "-",
	KeyEqual:          "=",
	KeyBackspace:      "Backspace",
	KeyTab:            "Tab",
	KeyQ:              "Q",
	KeyW:              "W",
	KeyE:              "E",
	KeyR:              "R",
	KeyT:              "T",
	KeyY:              "Y",
	KeyU:              "U",
	KeyI:              "I",
	KeyO:              "O",
	KeyP:              "P",
	KeyLeftBracket:    "[",
	KeyRightBracket:   "]",
	KeyEnter:          "Enter",
	KeyCtrl:           "Ctrl",
	KeyA:              "A",
	KeyS:              "S",
	KeyD:              "D",
	KeyF:              "F",
	KeyG:              "G",
	KeyH:              "H",
	KeyJ:              "J",
	KeyK:              "K",
	KeyL:              "L",
	KeySemicolon:      ";",
	KeyApostrophe:     "'",
	KeyGrave:          "`",
	KeyShift:          "Shift",
	KeyBackslash:      "\\",
	KeyZ:              "Z",
	KeyX:              "X",
	KeyC:              "C",
	KeyV:              "V",
	KeyB:              "B",
	KeyN:              "N",
	KeyM:              "M",
	KeyComma:          ",",
	KeyDot:            ".",
	KeySlash:          "/",
	KeyNumpadAsterisk: "NumpadAsterisk",
	KeyAlt:            "Alt",
	KeySpace:          "Space",
	KeyCapsLock:       "CapsLock",
	KeyF1:             "F1",
	KeyF2:             "F2",
	KeyF3:             "F3",
	KeyF4:             "F4",
	KeyF5:             "F5",
	KeyF6:             "F6",
	KeyF7:             "F7",
	KeyF8:             "F8",
	KeyF9:             "F9",
	KeyF10:            "F10",
	KeyNumLock:        "NumLock",
	KeyScrollLock:     "ScrollLock",
	KeyNumpad7:        "Numpad7",
	KeyNumpad8:        "Numpad8",
	KeyNumpad9:        "Numpad9",
	KeyNumpadMinus:    "NumpadMinus",
	KeyNumpad4:        "Numpad4",
	KeyNumpad5:        "Numpad5",
	KeyNumpad6:        "Numpad6",
	KeyNumpadPlus:     "NumpadPlus",
	KeyNumpad1:        "Numpad1",
	KeyNumpad2:        "Numpad2",
	KeyNumpad3:        "Numpad3",
	KeyNumpad0:        "Numpad0",
	KeyNumpadDot:      "NumpadDot",
	KeyF11:            "F11",
	KeyF12:            "F12",
	KeyInsert:         "Insert",
	KeyDelete:         "Delete",
	KeyHome:           "Home",
	KeyEnd:            "End",
	KeyPageUp:         "PageUp",
	KeyPageDown:       "PageDown",
	KeyArrowUp:        "ArrowUp",
	KeyArrowLeft:      "ArrowLeft",
	KeyArrowDown:      "ArrowDown",
	KeyArrowRight:     "ArrowRight",
	KeyNumpadEnter:    "NumpadEnter",
	KeyNumpadSlash:    "NumpadSlash",
	KeyPause:          "Pause",
	KeyPrintScreen:    "PrintScreen",
}

func (c Key) String() string {
	if name, ok := keyNames[c]; ok {
		return name
	}
	return "Unknown"
}
