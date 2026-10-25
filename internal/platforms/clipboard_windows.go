//go:build windows

package platforms

import (
	"bytes"
	"errors"
	"image/png"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// The Win32 clipboard: the text in it as CF_UNICODETEXT; an image as the
// "PNG" format (applications that keep the alpha read it) and CF_DIB (the
// others, e.g. Paint); other MIME types as formats registered by their names

const (
	c_CF_TEXT        = 1
	c_CF_BITMAP      = 2
	c_CF_DIB         = 8
	c_CF_UNICODETEXT = 13
	c_CF_DIBV5       = 17

	// Another application may hold the clipboard open for a moment
	clipboardOpenAttempts = 10
	clipboardOpenDelay    = 10 * time.Millisecond
)

var (
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procEmptyClipboard             = user32.NewProc("EmptyClipboard")
	procSetClipboardData           = user32.NewProc("SetClipboardData")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGlobalAllocClip            = kernel32.NewProc("GlobalAlloc")
	procGlobalFreeClip             = kernel32.NewProc("GlobalFree")
	procGlobalLockClip             = kernel32.NewProc("GlobalLock")
	procGlobalUnlockClip           = kernel32.NewProc("GlobalUnlock")
	procGlobalSizeClip             = kernel32.NewProc("GlobalSize")
	procRegisterClipboardFormatW   = user32.NewProc("RegisterClipboardFormatW")
	procEnumClipboardFormats       = user32.NewProc("EnumClipboardFormats")
	procGetClipboardFormatNameW    = user32.NewProc("GetClipboardFormatNameW")
)

var errClipboardBusy = errors.New("clipboard: another application keeps the clipboard open")

func openClipboard() error {
	for i := 0; i < clipboardOpenAttempts; i++ {
		if r, _, _ := procOpenClipboard.Call(0); r != 0 {
			return nil
		}
		time.Sleep(clipboardOpenDelay)
	}
	return errClipboardBusy
}

func clipboardGetText() (string, error) {
	if r, _, _ := procIsClipboardFormatAvailable.Call(c_CF_UNICODETEXT); r == 0 {
		return "", nil
	}
	if err := openClipboard(); err != nil {
		return "", err
	}
	defer procCloseClipboard.Call()
	h, _, _ := procGetClipboardData.Call(c_CF_UNICODETEXT)
	if h == 0 {
		return "", nil
	}
	p, _, _ := procGlobalLockClip.Call(h)
	if p == 0 {
		return "", nil
	}
	defer procGlobalUnlockClip.Call(h)
	size, _, _ := procGlobalSizeClip.Call(h)
	u := unsafe.Slice((*uint16)(unsafe.Pointer(p)), size/2)
	// The text ends at the first NUL, which may come before the end of the memory
	for i, c := range u {
		if c == 0 {
			u = u[:i]
			break
		}
	}
	return string(utf16.Decode(u)), nil
}

func indexNUL(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return i
		}
	}
	return len(s)
}

// clipboardFormat returns the format of the MIME type, registered by its
// name ("PNG" for an image)
func clipboardFormat(mime string) uintptr {
	name := mime
	if mime == MIMEPNG {
		name = "PNG"
	}
	u, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0
	}
	f, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(u)))
	return f
}

// globalFromBytes copies the data to movable global memory, for SetClipboardData
func globalFromBytes(data []byte) (uintptr, error) {
	h, _, _ := procGlobalAllocClip.Call(c_GMEM_MOVEABLE, uintptr(max(len(data), 1)))
	if h == 0 {
		return 0, errors.New("clipboard: out of memory")
	}
	p, _, _ := procGlobalLockClip.Call(h)
	if p == 0 {
		procGlobalFreeClip.Call(h)
		return 0, errors.New("clipboard: out of memory")
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(p)), len(data)), data)
	procGlobalUnlockClip.Call(h)
	return h, nil
}

// clipboardData copies the data of the format from the open clipboard
func clipboardData(format uintptr) ([]byte, bool) {
	h, _, _ := procGetClipboardData.Call(format)
	if h == 0 {
		return nil, false
	}
	p, _, _ := procGlobalLockClip.Call(h)
	if p == 0 {
		return nil, false
	}
	defer procGlobalUnlockClip.Call(h)
	size, _, _ := procGlobalSizeClip.Call(h)
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(p)), size)...), true
}

func clipboardSet(items []ClipboardItem) error {
	type entry struct {
		format uintptr
		data   []byte
	}
	var entries []entry
	for _, item := range items {
		if isTextMIME(item.MIME) {
			u, err := syscall.UTF16FromString(string(item.Data))
			if err != nil {
				u = append(utf16.Encode([]rune(string(item.Data[:indexNUL(string(item.Data))]))), 0)
			}
			entries = append(entries, entry{c_CF_UNICODETEXT, unsafe.Slice((*byte)(unsafe.Pointer(&u[0])), len(u)*2)})
			continue
		}
		if item.MIME == MIMEPNG {
			if img, err := png.Decode(bytes.NewReader(item.Data)); err == nil {
				entries = append(entries, entry{c_CF_DIB, dibFromImage(img)})
			}
		}
		if f := clipboardFormat(item.MIME); f != 0 {
			entries = append(entries, entry{f, item.Data})
		}
	}
	if err := openClipboard(); err != nil {
		return err
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	for _, e := range entries {
		h, err := globalFromBytes(e.data)
		if err != nil {
			return err
		}
		// The system owns the memory once it is set
		if r, _, _ := procSetClipboardData.Call(e.format, h); r == 0 {
			procGlobalFreeClip.Call(h)
			return errors.New("clipboard: the data was not put on the clipboard")
		}
	}
	return nil
}

func clipboardGet(mime string) ([]byte, bool, error) {
	if isTextMIME(mime) {
		if r, _, _ := procIsClipboardFormatAvailable.Call(c_CF_UNICODETEXT); r == 0 {
			return nil, false, nil
		}
		text, err := clipboardGetText()
		return []byte(text), err == nil, err
	}
	if err := openClipboard(); err != nil {
		return nil, false, err
	}
	defer procCloseClipboard.Call()
	if f := clipboardFormat(mime); f != 0 {
		if data, ok := clipboardData(f); ok {
			return data, true, nil
		}
	}
	if mime != MIMEPNG {
		return nil, false, nil
	}
	// An image of an application that gives a DIB only (Windows makes a
	// CF_DIB of a CF_BITMAP and of a CF_DIBV5 itself)
	for _, f := range []uintptr{c_CF_DIBV5, c_CF_DIB} {
		data, ok := clipboardData(f)
		if !ok {
			continue
		}
		img, err := imageFromDIB(data)
		if err != nil {
			continue
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, false, err
		}
		return buf.Bytes(), true, nil
	}
	return nil, false, nil
}

func clipboardFormats() ([]string, error) {
	if err := openClipboard(); err != nil {
		return nil, err
	}
	defer procCloseClipboard.Call()
	var formats []string
	add := func(name string) {
		for _, f := range formats {
			if f == name {
				return
			}
		}
		formats = append(formats, name)
	}
	for f := uintptr(0); ; {
		f, _, _ = procEnumClipboardFormats.Call(f)
		if f == 0 {
			break
		}
		switch f {
		case c_CF_TEXT, c_CF_UNICODETEXT:
			add(MIMEText)
			continue
		case c_CF_BITMAP, c_CF_DIB, c_CF_DIBV5:
			add(MIMEPNG)
			continue
		}
		var buf [256]uint16
		n, _, _ := procGetClipboardFormatNameW.Call(f, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 {
			continue // a standard format of no use here
		}
		name := string(utf16.Decode(buf[:n]))
		if name == "PNG" {
			name = MIMEPNG
		}
		add(name)
	}
	return formats, nil
}
