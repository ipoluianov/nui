# Clipboard

The system clipboard holds one piece of data, in one or several formats. Each format is named by its MIME type.

```go
ui.ClipboardSetText("copied")
text, err := ui.ClipboardGetText() // "" if the clipboard has no text

ui.ClipboardSetImage(screenshot)
img, err := ui.ClipboardGetImage() // nil if the clipboard has no image

// Several formats at once: the text for any application, the shapes for this one
ui.ClipboardSet(
	ui.ClipboardItem{MIME: ui.MIMEText, Data: []byte("2 shapes")},
	ui.ClipboardItem{MIME: "application/x-myapp-shapes", Data: shapesJSON},
)
data, ok, err := ui.ClipboardGetData("application/x-myapp-shapes")

paste.SetEnabled(ui.ClipboardHasFormat(ui.MIMEPNG))
formats, err := ui.ClipboardFormats() // e.g. [text/plain;charset=utf-8 image/png]
```

All the functions are safe from any goroutine; the work is done on the UI thread. Text boxes, tables and tree views use the clipboard for Ctrl+C, Ctrl+X and Ctrl+V.

`ui.MIMEText` (UTF-8 text) and `ui.MIMEPNG` (an image) map to the standard formats of each platform, so other applications read them. Any other MIME type maps to a format with the same name: applications that know the name read it, and the rest ignore it.

No cgo is needed on any system, so a program builds with `CGO_ENABLED=0` and cross-compiles:

- **Linux**: the CLIPBOARD selection of X11, through Xlib like the windows.
  - The text is `UTF8_STRING` (also `STRING`, `TEXT` and `text/plain`). Other formats are targets named by their MIME types, as GTK and Qt name them.
  - The data stays in the application. When its last window closes, a clipboard manager (most desktops run one) is asked to keep it, so it can still be pasted after the program quits.
  - Large data is received in parts (INCR). Sending is limited to the largest request of the X server, usually 16 MB.
  - An application that does not answer is waited for 2 seconds.
- **macOS**: the general `NSPasteboard`. The text is `public.utf8-plain-text` and the image `public.png`. An image other applications put as `public.tiff` is read too.
- **Windows**: the Win32 clipboard.
  - The text is `CF_UNICODETEXT`.
  - An image is put as the `PNG` format (it keeps the alpha) and as `CF_DIB` (for Paint and older programs).
  - An image is read from `PNG`, else from `CF_DIBV5` or `CF_DIB`.
  - Other formats are registered by their MIME type names.
