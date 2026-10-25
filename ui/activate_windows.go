package ui

import "syscall"

var (
	procIsIconic            = syscall.NewLazyDLL("user32.dll").NewProc("IsIconic")
	procShowWindow          = syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow")
	procSetForegroundWindow = syscall.NewLazyDLL("user32.dll").NewProc("SetForegroundWindow")
)

// activateWindow restores a minimized window and brings it to the
// foreground. Restore would also turn a maximized window to normal, so it
// is restored only when minimized.
func activateWindow(c *Form) {
	hwnd, ok := c.SystemHandle().(syscall.Handle)
	if !ok {
		c.Show()
		return
	}
	const swRestore = 9
	if minimized, _, _ := procIsIconic.Call(uintptr(hwnd)); minimized != 0 {
		procShowWindow.Call(uintptr(hwnd), swRestore)
	}
	procSetForegroundWindow.Call(uintptr(hwnd))
}
