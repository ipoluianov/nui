package platforms

import (
	"syscall"
	"unsafe"
)

var (
	procGetUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
	procLCIDToLocaleName         = kernel32.NewProc("LCIDToLocaleName")
)

// systemLanguage returns the display language of Windows, e.g. "ru-RU" (not
// the regional format, which may differ).
func systemLanguage() string {
	langID, _, _ := procGetUserDefaultUILanguage.Call()
	var buf [85]uint16 // LOCALE_NAME_MAX_LENGTH
	n, _, _ := procLCIDToLocaleName.Call(langID, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:])
}
