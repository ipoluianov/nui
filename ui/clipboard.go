package ui

import (
	"bytes"
	"image"
	"image/png"
	"slices"

	"github.com/ipoluianov/nui/internal/platforms"
)

// The system clipboard. It holds one piece of data in one or several
// formats, each named by its MIME type: a text, an image, the data of the
// application. All the functions are safe from any goroutine.
//
//	ui.ClipboardSetImage(screenshot)
//	ui.ClipboardSet(
//		ui.ClipboardItem{MIME: ui.MIMEText, Data: []byte(text)},
//		ui.ClipboardItem{MIME: "application/x-myapp-shapes", Data: shapesJSON},
//	)

// ClipboardItem is the data of the clipboard in one format
type ClipboardItem struct {
	MIME string
	Data []byte
}

const (
	// MIMEText is the text, UTF-8: the standard text of the platform
	MIMEText = platforms.MIMEText
	// MIMEPNG is an image: the standard image of the platform (on Windows
	// also CF_DIB, on macOS TIFF from other applications)
	MIMEPNG = platforms.MIMEPNG
)

// ClipboardSetText puts the text on the system clipboard
func ClipboardSetText(text string) {
	_ = platforms.ClipboardSetText(text)
}

// ClipboardGetText returns the text on the system clipboard, "" if there is
// none
func ClipboardGetText() (string, error) {
	return platforms.ClipboardGetText()
}

// ClipboardSet replaces the clipboard with the data in the formats: e.g. a
// text for any application and the application's own data for itself
func ClipboardSet(items ...ClipboardItem) error {
	list := make([]platforms.ClipboardItem, len(items))
	for i, item := range items {
		list[i] = platforms.ClipboardItem{MIME: item.MIME, Data: item.Data}
	}
	return platforms.ClipboardSet(list)
}

// ClipboardSetData replaces the clipboard with the data in one format
func ClipboardSetData(mime string, data []byte) error {
	return ClipboardSet(ClipboardItem{MIME: mime, Data: data})
}

// ClipboardGetData returns the data of the clipboard in the format; ok is
// false if it has none
func ClipboardGetData(mime string) (data []byte, ok bool, err error) {
	return platforms.ClipboardGet(mime)
}

// ClipboardFormats returns the formats of the data on the clipboard, as
// MIME types where known (MIMEText, MIMEPNG), else as the platform names them
func ClipboardFormats() ([]string, error) {
	return platforms.ClipboardFormats()
}

// ClipboardHasFormat reports whether the clipboard has data in the format:
// e.g. to enable Paste
func ClipboardHasFormat(mime string) bool {
	formats, _ := ClipboardFormats()
	return slices.Contains(formats, mime)
}

// ClipboardSetImage puts the image on the clipboard
func ClipboardSetImage(img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return ClipboardSetData(MIMEPNG, buf.Bytes())
}

// ClipboardGetImage returns the image on the clipboard, nil if there is none
func ClipboardGetImage() (image.Image, error) {
	data, ok, err := ClipboardGetData(MIMEPNG)
	if err != nil || !ok {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}
