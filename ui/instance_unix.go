//go:build !windows

package ui

import (
	"os"
	"syscall"
)

// tryLockFile takes the lock on the file without waiting; the system
// releases it when the process ends
func tryLockFile(f *os.File) bool {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) == nil
}

func allowSetForeground() {}
