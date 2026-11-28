package ui

import (
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// Close while the accept goroutine of OnActivate handles a connection must
// stop the goroutine cleanly: it used to read the listener field again,
// which Close had set to nil (a data race and a nil dereference panic).
// Run with -race to see the race itself.
func TestInstanceCloseWhileAccepting(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	for i := 0; i < 50; i++ {
		dir := t.TempDir()
		first, ok := acquireSingleInstance(dir, nil)
		if !ok || first.listener == nil {
			t.Skip("no local socket")
		}
		first.OnActivate(func([]string) {})
		sock := instanceSocketPath(dir)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for j := 0; j < 20; j++ {
				if c, err := net.DialTimeout("unix", sock, time.Second); err == nil {
					c.Close()
				}
			}
		}()
		time.Sleep(time.Millisecond)
		first.Close()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("the dialing goroutine did not finish")
		}
	}
}
