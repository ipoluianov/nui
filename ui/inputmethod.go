package ui

import "github.com/ipoluianov/nui/internal/platforms"

// UseMacInputMethod turns on the input method of macOS (Chinese, Japanese,
// Korean, and the accents shown when a key is held) for the forms shown
// after it. Experimental, off by default until checked on more systems: the
// keys go through a hidden NSTextView of the window. Windows has its input
// method always; on Linux nothing changes.
func UseMacInputMethod(use bool) {
	platforms.SetInputMethodEnabled(use)
}
