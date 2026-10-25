package platforms

// SetInputMethodEnabled turns on the input method of macOS for the windows
// made after it, see ime_darwin.go; elsewhere it does nothing (Windows has
// it always, Linux composes in place)
func SetInputMethodEnabled(enabled bool) {
	setInputMethodEnabled(enabled)
}
