package ui

import (
	"os"

	"golang.org/x/sys/windows"
)

var procAllowSetForegroundWindow = windows.NewLazySystemDLL("user32.dll").NewProc("AllowSetForegroundWindow")

// tryLockFile takes the lock on the file without waiting; the system
// releases it when the process ends
func tryLockFile(f *os.File) bool {
	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{})
	return err == nil
}

// allowSetForeground lets any process bring its window to the foreground
// (ASFW_ANY): the running copy, asked to show itself
func allowSetForeground() {
	const asfwAny = ^uintptr(0)
	procAllowSetForegroundWindow.Call(asfwAny)
}
