package platforms

// SetInputMethodEnabled turns the input method of macOS on or off (on by
// default) for the windows made after it, see ime_darwin.go; elsewhere it
// does nothing (Windows has it always, Linux composes in place)
func SetInputMethodEnabled(enabled bool) {
	setInputMethodEnabled(enabled)
}
