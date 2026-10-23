//go:build windows

package platforms

import (
	"errors"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// The Win32 clipboard, the text in it as CF_UNICODETEXT

const (
	c_CF_UNICODETEXT = 13

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

func clipboardSetText(text string) error {
	u, err := syscall.UTF16FromString(text)
	if err != nil {
		// A NUL in the text: what is before it is copied
		u = append(utf16.Encode([]rune(text[:indexNUL(text)])), 0)
	}
	size := uintptr(len(u) * 2)
	h, _, _ := procGlobalAllocClip.Call(c_GMEM_MOVEABLE, size)
	if h == 0 {
		return errors.New("clipboard: out of memory")
	}
	p, _, _ := procGlobalLockClip.Call(h)
	if p == 0 {
		procGlobalFreeClip.Call(h)
		return errors.New("clipboard: out of memory")
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u)), u)
	procGlobalUnlockClip.Call(h)

	if err := openClipboard(); err != nil {
		procGlobalFreeClip.Call(h)
		return err
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	// The system owns the memory once it is set
	if r, _, _ := procSetClipboardData.Call(c_CF_UNICODETEXT, h); r == 0 {
		procGlobalFreeClip.Call(h)
		return errors.New("clipboard: the text was not put on the clipboard")
	}
	return nil
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
