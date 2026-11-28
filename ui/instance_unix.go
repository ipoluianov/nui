//go:build !windows

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// tryLockFile takes the lock on the file without waiting; the system
// releases it when the process ends
func tryLockFile(f *os.File) bool {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) == nil
}

func allowSetForeground() {}

// privateTempDir returns a directory in the temporary one only the user can
// use: nui-<uid>, made with mode 0700, and checked to be a real directory of
// the user that no one else can write to (another user may have made it
// first)
func privateTempDir() (string, error) {
	return privateDirIn(os.TempDir())
}

func privateDirIn(parent string) (string, error) {
	uid := os.Getuid()
	dir := filepath.Join(parent, "nui-"+strconv.Itoa(uid))
	if err := os.Mkdir(dir, 0o700); err != nil && !os.IsExist(err) {
		return "", err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || !ok || int(st.Uid) != uid {
		return "", fmt.Errorf("%s is not a directory of the user", dir)
	}
	if info.Mode().Perm()&0o077 != 0 {
		// The user's own directory with a wider mode: made private
		if err := os.Chmod(dir, 0o700); err != nil {
			return "", err
		}
	}
	return dir, nil
}
