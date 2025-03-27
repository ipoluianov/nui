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

/*
#define VK_LBUTTON 0x01
#define VK_RBUTTON 0x02
#define VK_CANCEL 0x03
#define VK_MBUTTON 0x04
#define VK_XBUTTON1 0x05
#define VK_XBUTTON2 0x06
#define VK_BACK 0x08
#define VK_TAB 0x09
#define VK_CLEAR 0x0C
#define VK_RETURN 0x0D
#define VK_SHIFT 0x10
#define VK_CONTROL 0x11
#define VK_MENU 0x12
#define VK_PAUSE 0x13
#define VK_CAPITAL 0x14
#define VK_KANA 0x15
#define VK_HANGEUL 0x15
#define VK_HANGUL 0x15
#define VK_IME_ON 0x16
#define VK_JUNJA 0x17
#define VK_FINAL 0x18
#define VK_HANJA 0x19
#define VK_KANJI 0x19
#define VK_IME_OFF 0x1A
#define VK_ESCAPE 0x1B
#define VK_CONVERT 0x1C
#define VK_NONCONVERT 0x1D
#define VK_ACCEPT 0x1E
#define VK_MODECHANGE 0x1F
#define VK_SPACE 0x20
#define VK_PRIOR 0x21
#define VK_NEXT 0x22
#define VK_END 0x23
#define VK_HOME 0x24
#define VK_LEFT 0x25
#define VK_UP 0x26
#define VK_RIGHT 0x27
#define VK_DOWN 0x28
#define VK_SELECT 0x29
#define VK_PRINT 0x2A
#define VK_EXECUTE 0x2B
#define VK_SNAPSHOT 0x2C
#define VK_INSERT 0x2D
#define VK_DELETE 0x2E
#define VK_HELP 0x2F

#define VK_LWIN 0x5B
#define VK_RWIN 0x5C
#define VK_APPS 0x5D
#define VK_SLEEP 0x5F
#define VK_NUMPAD0 0x60
#define VK_NUMPAD1 0x61
#define VK_NUMPAD2 0x62
#define VK_NUMPAD3 0x63
#define VK_NUMPAD4 0x64
#define VK_NUMPAD5 0x65
#define VK_NUMPAD6 0x66
#define VK_NUMPAD7 0x67
#define VK_NUMPAD8 0x68
#define VK_NUMPAD9 0x69
#define VK_MULTIPLY 0x6A
#define VK_ADD 0x6B
#define VK_SEPARATOR 0x6C
#define VK_SUBTRACT 0x6D
#define VK_DECIMAL 0x6E
#define VK_DIVIDE 0x6F
#define VK_F1 0x70
#define VK_F2 0x71
#define VK_F3 0x72
#define VK_F4 0x73
#define VK_F5 0x74
#define VK_F6 0x75
#define VK_F7 0x76
#define VK_F8 0x77
#define VK_F9 0x78
#define VK_F10 0x79
#define VK_F11 0x7A
#define VK_F12 0x7B
#define VK_F13 0x7C
#define VK_F14 0x7D
#define VK_F15 0x7E
#define VK_F16 0x7F
#define VK_F17 0x80
#define VK_F18 0x81
#define VK_F19 0x82
#define VK_F20 0x83
#define VK_F21 0x84
#define VK_F22 0x85
#define VK_F23 0x86
#define VK_F24 0x87

#define VK_NAVIGATION_VIEW 0x88
#define VK_NAVIGATION_MENU 0x89
#define VK_NAVIGATION_UP 0x8A
#define VK_NAVIGATION_DOWN 0x8B
#define VK_NAVIGATION_LEFT 0x8C
#define VK_NAVIGATION_RIGHT 0x8D
#define VK_NAVIGATION_ACCEPT 0x8E
#define VK_NAVIGATION_CANCEL 0x8F

#define VK_NUMLOCK 0x90
#define VK_SCROLL 0x91
#define VK_OEM_NEC_EQUAL 0x92
#define VK_OEM_FJ_JISHO 0x92
#define VK_OEM_FJ_MASSHOU 0x93
#define VK_OEM_FJ_TOUROKU 0x94
#define VK_OEM_FJ_LOYA 0x95
#define VK_OEM_FJ_ROYA 0x96
#define VK_LSHIFT 0xA0
#define VK_RSHIFT 0xA1
#define VK_LCONTROL 0xA2
#define VK_RCONTROL 0xA3
#define VK_LMENU 0xA4
#define VK_RMENU 0xA5
#define VK_BROWSER_BACK 0xA6
#define VK_BROWSER_FORWARD 0xA7
#define VK_BROWSER_REFRESH 0xA8
#define VK_BROWSER_STOP 0xA9
#define VK_BROWSER_SEARCH 0xAA
#define VK_BROWSER_FAVORITES 0xAB
#define VK_BROWSER_HOME 0xAC
#define VK_VOLUME_MUTE 0xAD
#define VK_VOLUME_DOWN 0xAE
#define VK_VOLUME_UP 0xAF
#define VK_MEDIA_NEXT_TRACK 0xB0
#define VK_MEDIA_PREV_TRACK 0xB1
#define VK_MEDIA_STOP 0xB2
#define VK_MEDIA_PLAY_PAUSE 0xB3
#define VK_LAUNCH_MAIL 0xB4
#define VK_LAUNCH_MEDIA_SELECT 0xB5
#define VK_LAUNCH_APP1 0xB6
#define VK_LAUNCH_APP2 0xB7
#define VK_OEM_1 0xBA
#define VK_OEM_PLUS 0xBB
#define VK_OEM_COMMA 0xBC
#define VK_OEM_MINUS 0xBD
#define VK_OEM_PERIOD 0xBE
#define VK_OEM_2 0xBF
#define VK_OEM_3 0xC0
#if _WIN32_WINNT >= 0x0604
#define VK_GAMEPAD_A 0xC3
#define VK_GAMEPAD_B 0xC4
#define VK_GAMEPAD_X 0xC5
#define VK_GAMEPAD_Y 0xC6
#define VK_GAMEPAD_RIGHT_SHOULDER 0xC7
#define VK_GAMEPAD_LEFT_SHOULDER 0xC8
#define VK_GAMEPAD_LEFT_TRIGGER 0xC9
#define VK_GAMEPAD_RIGHT_TRIGGER 0xCA
#define VK_GAMEPAD_DPAD_UP 0xCB
#define VK_GAMEPAD_DPAD_DOWN 0xCC
#define VK_GAMEPAD_DPAD_LEFT 0xCD
#define VK_GAMEPAD_DPAD_RIGHT 0xCE
#define VK_GAMEPAD_MENU 0xCF
#define VK_GAMEPAD_VIEW 0xD0
#define VK_GAMEPAD_LEFT_THUMBSTICK_BUTTON 0xD1
#define VK_GAMEPAD_RIGHT_THUMBSTICK_BUTTON 0xD2
#define VK_GAMEPAD_LEFT_THUMBSTICK_UP 0xD3
#define VK_GAMEPAD_LEFT_THUMBSTICK_DOWN 0xD4
#define VK_GAMEPAD_LEFT_THUMBSTICK_RIGHT 0xD5
#define VK_GAMEPAD_LEFT_THUMBSTICK_LEFT 0xD6
#define VK_GAMEPAD_RIGHT_THUMBSTICK_UP 0xD7
#define VK_GAMEPAD_RIGHT_THUMBSTICK_DOWN 0xD8
#define VK_GAMEPAD_RIGHT_THUMBSTICK_RIGHT 0xD9
#define VK_GAMEPAD_RIGHT_THUMBSTICK_LEFT 0xDA

#define VK_OEM_4 0xDB
#define VK_OEM_5 0xDC
#define VK_OEM_6 0xDD
#define VK_OEM_7 0xDE
#define VK_OEM_8 0xDF
#define VK_OEM_AX 0xE1
#define VK_OEM_102 0xE2
#define VK_ICO_HELP 0xE3
#define VK_ICO_00 0xE4
#define VK_PROCESSKEY 0xE5
#define VK_ICO_CLEAR 0xE6
#define VK_PACKET 0xE7
#define VK_OEM_RESET 0xE9
#define VK_OEM_JUMP 0xEA
#define VK_OEM_PA1 0xEB
#define VK_OEM_PA2 0xEC
#define VK_OEM_PA3 0xED
#define VK_OEM_WSCTRL 0xEE
#define VK_OEM_CUSEL 0xEF
#define VK_OEM_ATTN 0xF0
#define VK_OEM_FINISH 0xF1
#define VK_OEM_COPY 0xF2
#define VK_OEM_AUTO 0xF3
#define VK_OEM_ENLW 0xF4
#define VK_OEM_BACKTAB 0xF5
#define VK_ATTN 0xF6
#define VK_CRSEL 0xF7
#define VK_EXSEL 0xF8
#define VK_EREOF 0xF9
#define VK_PLAY 0xFA
#define VK_ZOOM 0xFB
#define VK_NONAME 0xFC
#define VK_PA1 0xFD
#define VK_OEM_CLEAR 0xFE
*/

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
	KeyOption   = 0xCC02
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
	KeyOption:   "Option",
	KeyFunction: "Function",
}

func (c Key) String() string {
	if name, ok := keyNames[c]; ok {
		return name
	}
	return "Unknown"
}
