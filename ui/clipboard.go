package ui

import "github.com/ipoluianov/nui/internal/platforms"

// ClipboardSetText puts the text on the system clipboard. Safe from any goroutine.
func ClipboardSetText(text string) {
	_ = platforms.ClipboardSetText(text)
}

// ClipboardGetText returns the text on the system clipboard, "" if there is
// none. Safe from any goroutine.
func ClipboardGetText() (string, error) {
	return platforms.ClipboardGetText()
}
