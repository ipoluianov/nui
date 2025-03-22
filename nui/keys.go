package nui

type Key int

const (
	KeyEsc            = 0x01
	Key1              = 0x02
	Key2              = 0x03
	Key3              = 0x04
	Key4              = 0x05
	Key5              = 0x06
	Key6              = 0x07
	Key7              = 0x08
	Key8              = 0x09
	Key9              = 0x0A
	Key0              = 0x0B
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
	KeyLeftCtrl       = 0x1D
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
	KeyLeftShift      = 0x2A
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
	KeyRightShift     = 0x36
	KeyNumpadAsterisk = 0x37
	KeyLeftAlt        = 0x38
	KeySpace          = 0x39
	KeyCapsLock       = 0x3A
	KeyF1             = 0x3B
	KeyF2             = 0x3C
	KeyF3             = 0x3D
	KeyF4             = 0x3E
	KeyF5             = 0x3F
	KeyF6             = 0x40
	KeyF7             = 0x41
	KeyF8             = 0x42
	KeyF9             = 0x43
	KeyF10            = 0x44
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
	KeyF11            = 0x57
	KeyF12            = 0x58

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

	// Modifiers - right side
	KeyRightCtrl = 0xE01D
	KeyRightAlt  = 0xE038
	KeyRightWin  = 0xE05C
	KeyLeftWin   = 0xE05B
	KeyApps      = 0xE05D // Menu key

	// Numpad - special keys
	KeyNumpadEnter = 0xE01C
	KeyNumpadSlash = 0xE035

	// Special keys
	KeyPause       = 0xE11D45 // Very special: composite scan-code
	KeyPrintScreen = 0xE037   // Requires shift-logic
)
