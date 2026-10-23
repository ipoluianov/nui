# Clipboard

The text clipboard of the system.

```go
ui.ClipboardSetText("copied")
text, err := ui.ClipboardGetText() // "" if the clipboard has no text
```

Both are safe from any goroutine: the work is done on the UI thread. The text
boxes, tables and tree views use them for Ctrl+C, Ctrl+X and Ctrl+V.

No cgo is needed on any system, so a program builds with `CGO_ENABLED=0` and
cross-compiles:

- **Linux** - the CLIPBOARD selection of X11, through Xlib like the windows.
  The text copied stays in the application; when its last window closes, a
  clipboard manager (most desktops run one) is asked to keep it, so it can
  still be pasted after the program quits. Large texts are received in parts
  (INCR). An application that does not answer is waited for 2 seconds.
- **macOS** - the general `NSPasteboard`.
- **Windows** - the Win32 clipboard (`CF_UNICODETEXT`).
