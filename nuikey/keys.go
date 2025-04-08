package nuikey

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
	KeyEsc = 0x1B
	KeyF1  = 0x70
	KeyF2  = 0x71
	KeyF3  = 0x72
	KeyF4  = 0x73
	KeyF5  = 0x74
	KeyF6  = 0x75
	KeyF7  = 0x76
	KeyF8  = 0x77
	KeyF9  = 0x78
	KeyF10 = 0x79
	KeyF11 = 0x7A
	KeyF12 = 0x7B
	KeyF13 = 0x7C
	KeyF14 = 0x7D
	KeyF15 = 0x7E
	KeyF16 = 0x7F
	KeyF17 = 0x80
	KeyF18 = 0x81
	KeyF19 = 0x82
	KeyF20 = 0x83
	KeyF21 = 0x84
	KeyF22 = 0x85
	KeyF23 = 0x86
	KeyF24 = 0x87

	KeyGrave     = 0xC0
	Key1         = 0x31
	Key2         = 0x32
	Key3         = 0x33
	Key4         = 0x34
	Key5         = 0x35
	Key6         = 0x36
	Key7         = 0x37
	Key8         = 0x38
	Key9         = 0x39
	Key0         = 0x30
	KeyMinus     = 0xBD
	KeyEqual     = 0xBB
	KeyBackspace = 0x08

	KeyTab          = 0x09
	KeyQ            = 0x51
	KeyW            = 0x57
	KeyE            = 0x45
	KeyR            = 0x52
	KeyT            = 0x54
	KeyY            = 0x59
	KeyU            = 0x55
	KeyI            = 0x49
	KeyO            = 0x4F
	KeyP            = 0x50
	KeyLeftBracket  = 0xDB
	KeyRightBracket = 0xDD
	KeyBackslash    = 0xDC

	KeyCapsLock   = 0x14
	KeyA          = 0x41
	KeyS          = 0x53
	KeyD          = 0x44
	KeyF          = 0x46
	KeyG          = 0x47
	KeyH          = 0x48
	KeyJ          = 0x4A
	KeyK          = 0x4B
	KeyL          = 0x4C
	KeySemicolon  = 0xBA
	KeyApostrophe = 0xDE
	KeyEnter      = 0x0D

	KeyShift = 0x10
	KeyZ     = 0x5A
	KeyX     = 0x58
	KeyC     = 0x43
	KeyV     = 0x56
	KeyB     = 0x42
	KeyN     = 0x4E
	KeyM     = 0x4D
	KeyComma = 0xBC
	KeyDot   = 0xBE
	KeySlash = 0xBF

	KeyCtrl        = 0x11
	KeyWin         = 0x5B
	KeyAlt         = 0x12
	KeySpace       = 0x20
	KeyContextMenu = 0x5D

	KeyPrintScreen = 0x2C
	KeyScrollLock  = 0x91
	KeyPauseBreak  = 0x13
	KeyInsert      = 0x2D
	KeyHome        = 0x24
	KeyPageUp      = 0x21
	KeyDelete      = 0x2E
	KeyEnd         = 0x23
	KeyPageDown    = 0x22

	KeyArrowUp    = 0x26
	KeyArrowLeft  = 0x25
	KeyArrowDown  = 0x28
	KeyArrowRight = 0x27

	KeyNumLock        = 0x90
	KeyNumpadSlash    = 0x6F
	KeyNumpadAsterisk = 0x6A
	KeyNumpadMinus    = 0x6D
	KeyNumpadPlus     = 0x6B
	KeyNumpad1        = 0x61
	KeyNumpad2        = 0x62
	KeyNumpad3        = 0x63
	KeyNumpad4        = 0x64
	KeyNumpad5        = 0x65
	KeyNumpad6        = 0x66
	KeyNumpad7        = 0x67
	KeyNumpad8        = 0x68
	KeyNumpad9        = 0x69
	KeyNumpad0        = 0x60
	KeyNumpadDot      = 0x6E

	// Mac OS
	KeyCommand  = 0xCC01
	KeyFunction = 0xCC03
)

var keyNames = map[Key]string{
	KeyEsc: "Esc",
	KeyF1:  "F1",
	KeyF2:  "F2",
	KeyF3:  "F3",
	KeyF4:  "F4",
	KeyF5:  "F5",
	KeyF6:  "F6",
	KeyF7:  "F7",
	KeyF8:  "F8",
	KeyF9:  "F9",
	KeyF10: "F10",
	KeyF11: "F11",
	KeyF12: "F12",
	KeyF13: "F13",
	KeyF14: "F14",
	KeyF15: "F15",
	KeyF16: "F16",
	KeyF17: "F17",
	KeyF18: "F18",
	KeyF19: "F19",
	KeyF20: "F20",
	KeyF21: "F21",
	KeyF23: "F22",
	KeyF24: "F23",

	KeyGrave:     "Grave",
	Key1:         "1",
	Key2:         "2",
	Key3:         "3",
	Key4:         "4",
	Key5:         "5",
	Key6:         "6",
	Key7:         "7",
	Key8:         "8",
	Key9:         "9",
	Key0:         "0",
	KeyMinus:     "-",
	KeyEqual:     "=",
	KeyBackspace: "Backspace",

	KeyTab:          "Tab",
	KeyQ:            "Q",
	KeyW:            "W",
	KeyE:            "E",
	KeyR:            "R",
	KeyT:            "T",
	KeyY:            "Y",
	KeyU:            "U",
	KeyI:            "I",
	KeyO:            "O",
	KeyP:            "P",
	KeyLeftBracket:  "[",
	KeyRightBracket: "]",
	KeyBackslash:    "Backslash",

	KeyCapsLock:   "CapsLock",
	KeyA:          "A",
	KeyS:          "S",
	KeyD:          "D",
	KeyF:          "F",
	KeyG:          "G",
	KeyH:          "H",
	KeyJ:          "J",
	KeyK:          "K",
	KeyL:          "L",
	KeySemicolon:  ";",
	KeyApostrophe: "'",
	KeyEnter:      "Enter",

	KeyShift: "Shift",
	KeyZ:     "Z",
	KeyX:     "X",
	KeyC:     "C",
	KeyV:     "V",
	KeyB:     "B",
	KeyN:     "N",
	KeyM:     "M",
	KeyComma: ",",
	KeyDot:   ".",
	KeySlash: "/",

	KeyCtrl:        "Ctrl",
	KeyWin:         "Win",
	KeyAlt:         "Alt",
	KeySpace:       "Space",
	KeyContextMenu: "ContextMenu",

	KeyPrintScreen: "PrintScreen",
	KeyScrollLock:  "ScrollLock",
	KeyPauseBreak:  "PauseBreak",

	KeyInsert:   "Insert",
	KeyHome:     "Home",
	KeyPageUp:   "PageUp",
	KeyDelete:   "Delete",
	KeyEnd:      "End",
	KeyPageDown: "PageDown",

	KeyArrowUp:    "ArrowUp",
	KeyArrowLeft:  "ArrowLeft",
	KeyArrowDown:  "ArrowDown",
	KeyArrowRight: "ArrowRight",

	KeyNumLock:        "NumLock",
	KeyNumpadSlash:    "NumpadSlash",
	KeyNumpadAsterisk: "NumpadAsterisk",
	KeyNumpadMinus:    "NumpadMinus",
	KeyNumpadPlus:     "NumpadPlus",

	KeyNumpad1:   "Numpad1",
	KeyNumpad2:   "Numpad2",
	KeyNumpad3:   "Numpad3",
	KeyNumpad4:   "Numpad4",
	KeyNumpad5:   "Numpad5",
	KeyNumpad6:   "Numpad6",
	KeyNumpad7:   "Numpad7",
	KeyNumpad8:   "Numpad8",
	KeyNumpad9:   "Numpad9",
	KeyNumpad0:   "Numpad0",
	KeyNumpadDot: "NumpadDot",

	// Mac OS
	KeyCommand:  "Command",
	KeyFunction: "Function",
}

func (c Key) String() string {
	if name, ok := keyNames[c]; ok {
		return name
	}
	return "Unknown"
}
