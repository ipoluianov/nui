package platforms

// The text clipboard of the system. Each platform talks to it in pure Go, so
// the library needs no cgo: X11 selections on Linux (clipboard_linux.go),
// NSPasteboard on macOS (clipboard_darwin.go), the Win32 clipboard on Windows
// (clipboard_windows.go).

// ClipboardSetText puts the text on the clipboard. Safe from any goroutine:
// the work is done on the UI thread.
func ClipboardSetText(text string) error {
	var err error
	RunOnUIThread(func() { err = clipboardSetText(text) })
	return err
}

// ClipboardGetText returns the text on the clipboard, "" if there is none.
// Safe from any goroutine: the work is done on the UI thread.
func ClipboardGetText() (string, error) {
	var text string
	var err error
	RunOnUIThread(func() { text, err = clipboardGetText() })
	return text, err
}
