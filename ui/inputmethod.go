package ui

import "github.com/ipoluianov/nui/internal/platforms"

// UseMacInputMethod turns the input method of macOS (Chinese, Japanese,
// Korean, the accents shown when a key is held, dead keys, the emoji
// picker) on or off for the forms shown after it; it is on by default.
// While the focus is in a text box, the keys go through a hidden NSTextView
// of the window; elsewhere (a button, a list) they come to the form as they
// are. UseMacInputMethod(false), before the forms, takes the keys straight
// in text boxes too. Windows has its input method always; on Linux nothing
// changes.
func UseMacInputMethod(use bool) {
	platforms.SetInputMethodEnabled(use)
}
