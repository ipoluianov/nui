//go:build linux

package platforms

import (
	"unicode/utf8"
	"unsafe"
)

// Text input on Linux goes through an X input method (XIM): the one of
// XMODIFIERS (ibus, fcitx: Chinese, Japanese, Korean input), or Xlib's own
// local one, which does the keyboard layout's characters in UTF-8 (Cyrillic,
// Greek...), dead keys and Compose. Each window has an input context; the
// event loop gives every event to XFilterEvent first, and the text of a key
// press comes from Xutf8LookupString. XLookupString, without an input
// method, gives Latin-1 only.

const (
	xFocusIn       = 9
	xMappingNotify = 34

	ximPreeditNothing = 0x0008
	ximStatusNothing  = 0x0400

	xLookupChars    = 2
	xLookupBoth     = 4
	xBufferOverflow = -1
)

var (
	xSetLocaleModifiers func(modifiers string) uintptr
	xOpenIM             func(display, rdb, resName, resClass uintptr) uintptr
	// XCreateIC is variadic: called with three name/value pairs and the
	// terminating NULL, which the Linux calling conventions pass like fixed
	// arguments
	xCreateIC               func(im uintptr, name1 string, value1 uintptr, name2 string, value2 uintptr, name3 string, value3 uintptr, end uintptr) uintptr
	xDestroyIC              func(ic uintptr)
	xSetICFocus             func(ic uintptr)
	xUnsetICFocus           func(ic uintptr)
	xFilterEvent            func(event unsafe.Pointer, window uintptr) int32
	xutf8LookupString       func(ic uintptr, event unsafe.Pointer, buffer unsafe.Pointer, bytes int32, keysym unsafe.Pointer, status unsafe.Pointer) int32
	xRefreshKeyboardMapping func(event unsafe.Pointer) int32

	// xim is the input method of the display, 0 if none could be opened
	xim uintptr
)

// openInputMethod opens the input method of XMODIFIERS, or else the local one
func openInputMethod(display uintptr) {
	xSetLocaleModifiers("")
	xim = xOpenIM(display, 0, 0, 0)
	if xim == 0 {
		// The input method server of XMODIFIERS isn't running
		xSetLocaleModifiers("@im=none")
		xim = xOpenIM(display, 0, 0, 0)
	}
}

// createInputContext makes the window's input context
func (c *nativeWindow) createInputContext() {
	if xim == 0 {
		return
	}
	c.platform.ic = xCreateIC(xim,
		"inputStyle", ximPreeditNothing|ximStatusNothing,
		"clientWindow", c.platform.window,
		"focusWindow", c.platform.window,
		0)
}

func (c *nativeWindow) destroyInputContext() {
	if c.platform.ic != 0 {
		xDestroyIC(c.platform.ic)
		c.platform.ic = 0
	}
}

func (c *nativeWindow) setInputFocus(focused bool) {
	if c.platform.ic == 0 {
		return
	}
	if focused {
		xSetICFocus(c.platform.ic)
	} else {
		xUnsetICFocus(c.platform.ic)
	}
}

// filterEvent gives the event to the input method; true if it took it (a
// dead key, a key of a composition)
func filterEvent(event *xEvent) bool {
	return xim != 0 && xFilterEvent(unsafe.Pointer(event), 0) != 0
}

// keyText returns the text a key press types
func (c *nativeWindow) keyText(event *xEvent) string {
	var sym uintptr
	if c.platform.ic != 0 {
		buf := make([]byte, 64)
		var status int32
		n := xutf8LookupString(c.platform.ic, unsafe.Pointer(event), unsafe.Pointer(&buf[0]), int32(len(buf)), unsafe.Pointer(&sym), unsafe.Pointer(&status))
		if status == xBufferOverflow {
			buf = make([]byte, n+1)
			n = xutf8LookupString(c.platform.ic, unsafe.Pointer(event), unsafe.Pointer(&buf[0]), int32(len(buf)), unsafe.Pointer(&sym), unsafe.Pointer(&status))
		}
		if (status == xLookupChars || status == xLookupBoth) && n > 0 && utf8.Valid(buf[:n]) {
			return string(buf[:n])
		}
		return ""
	}

	// No input method: Latin-1
	var buf [32]byte
	n := xLookupString(unsafe.Pointer(event), unsafe.Pointer(&buf[0]), int32(len(buf)), unsafe.Pointer(&sym), 0)
	runes := make([]rune, 0, n)
	for _, b := range buf[:max(n, 0)] {
		runes = append(runes, rune(b))
	}
	return string(runes)
}
