package ui

import (
	"path/filepath"
	"strings"
	"testing"
)

// A long settings directory (a long home or application name on macOS,
// where there is no XDG_RUNTIME_DIR) still gives a socket path the system
// takes
func TestInstanceSocketPathLength(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	long := filepath.Join(t.TempDir(), "Library", "Application Support", strings.Repeat("com.company.product.", 4))
	if path := instanceSocketPath(long); len(path) > maxSocketPath {
		t.Errorf("%d bytes: %s", len(path), path)
	}
	if instanceSocketPath(long+"2") == instanceSocketPath(long) {
		t.Error("different directories share a socket")
	}

	short := filepath.Join("/home/u/.config", "app")
	if got := instanceSocketPath(short); got != filepath.Join(short, "instance.sock") {
		t.Errorf("a short directory keeps the socket in it: %s", got)
	}
}
