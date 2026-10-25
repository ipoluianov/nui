package platforms

import "slices"

// The clipboard of the system. Each platform talks to it in pure Go, so
// the library needs no cgo: X11 selections on Linux (clipboard_linux.go),
// NSPasteboard on macOS (clipboard_darwin.go), the Win32 clipboard on Windows
// (clipboard_windows.go).
//
// The clipboard holds one piece of data in several formats, each named by
// its MIME type: a text, an image as PNG, an application's own data. Each
// platform maps the types to its own names: the text and PNG to the standard
// formats other applications read, the other types to formats by the same
// name.

// ClipboardItem is the data of the clipboard in one format
type ClipboardItem struct {
	MIME string
	Data []byte
}

const (
	// MIMEText is the text, UTF-8
	MIMEText = "text/plain;charset=utf-8"
	// MIMEPNG is an image as PNG
	MIMEPNG = "image/png"
)

// ClipboardSetText puts the text on the clipboard. Safe from any goroutine:
// the work is done on the UI thread.
func ClipboardSetText(text string) error {
	return ClipboardSet([]ClipboardItem{{MIMEText, []byte(text)}})
}

// ClipboardGetText returns the text on the clipboard, "" if there is none.
// Safe from any goroutine: the work is done on the UI thread.
func ClipboardGetText() (string, error) {
	if IsHeadless() {
		data, _, err := ClipboardGet(MIMEText)
		return string(data), err
	}
	var text string
	var err error
	RunOnUIThread(func() { text, err = clipboardGetText() })
	return text, err
}

// ClipboardSet replaces the clipboard with the data in the formats
func ClipboardSet(items []ClipboardItem) error {
	if IsHeadless() {
		headlessMu.Lock()
		headlessClipboard = slices.Clone(items)
		headlessMu.Unlock()
		return nil
	}
	var err error
	RunOnUIThread(func() { err = clipboardSet(items) })
	return err
}

// ClipboardGet returns the data of the clipboard in the format; ok is false
// if it has none
func ClipboardGet(mime string) (data []byte, ok bool, err error) {
	if IsHeadless() {
		headlessMu.Lock()
		defer headlessMu.Unlock()
		for _, item := range headlessClipboard {
			if item.MIME == mime {
				return item.Data, true, nil
			}
		}
		return nil, false, nil
	}
	RunOnUIThread(func() { data, ok, err = clipboardGet(mime) })
	return data, ok, err
}

// ClipboardFormats returns the formats of the data on the clipboard: MIME
// types, MIMEText for a text; the names of the formats of other applications
// as the platform gives them
func ClipboardFormats() ([]string, error) {
	if IsHeadless() {
		headlessMu.Lock()
		defer headlessMu.Unlock()
		var formats []string
		for _, item := range headlessClipboard {
			formats = append(formats, item.MIME)
		}
		return formats, nil
	}
	var formats []string
	var err error
	RunOnUIThread(func() { formats, err = clipboardFormats() })
	return formats, err
}

// isTextMIME reports whether the type is the text of MIMEText
func isTextMIME(mime string) bool {
	return mime == MIMEText || mime == "text/plain" || mime == "text/plain;charset=UTF-8"
}
