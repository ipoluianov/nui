//go:build linux

package platforms

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A file dialog is modal over its window: the input of the window is
// dropped while the helper runs and comes back after it; kdialog is made
// transient for the window and placed in its middle
func TestFileDialogModal(t *testing.T) {
	p := &nativeWindow{windowPosX: 100, windowPosY: 50, windowWidth: 1200, windowHeight: 800}
	p.platform.window = 0x6800001
	dir := t.TempDir()
	// A kdialog that tells whether its window takes input: the test sees it from the helper
	os.WriteFile(filepath.Join(dir, "kdialog"), []byte("#!/bin/sh\necho \"$@\"\n"), 0o755)
	t.Setenv("PATH", dir)
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	blocked := false
	var args []string
	err := withHelper(p, func(kind, bin string, attach []string) error {
		blocked = p.inputBlocked()
		args = attach
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !blocked {
		t.Error("the window takes input while its dialog is open")
	}
	if p.inputBlocked() {
		t.Error("the window takes no input after its dialog")
	}
	want := "--attach 109051905 --geometry 900x620+250+140"
	if got := strings.Join(args, " "); got != want {
		t.Errorf("kdialog %q, not %q", got, want)
	}
	// The whole call: the arguments reach kdialog
	out, err := selectDirectoryDialog(p, SelectDirectoryDialogOptions{Title: "T", DefaultDirectory: "/tmp"})
	if err != nil || !strings.HasPrefix(out, want+" --getexistingdirectory /tmp") {
		t.Errorf("kdialog ran with %q (%v)", out, err)
	}
}
