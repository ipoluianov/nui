//go:build !windows

package ui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Without XDG_RUNTIME_DIR and with a long settings path the socket used to
// go straight into the shared temporary directory, under a name another
// user could predict and take first. Now it goes into nui-<uid> there,
// made private, and a directory of that name that is not the user's own
// (or not a directory) gives no socket at all.
func TestInstanceSocketPrivateTempDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("XDG_RUNTIME_DIR", "")
	long := filepath.Join(tmp, "Library", "Application Support", strings.Repeat("com.company.product.", 4))
	private := filepath.Join(tmp, "nui-"+strconv.Itoa(os.Getuid()))

	path := instanceSocketPath(long)
	if filepath.Dir(path) != private {
		t.Fatalf("socket %s, want it in %s", path, private)
	}
	info, err := os.Lstat(private)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("private dir %v %v", info, err)
	}

	// The user's own directory with a wide mode is made private
	os.Chmod(private, 0o777)
	instanceSocketPath(long)
	if info, _ := os.Lstat(private); info.Mode().Perm() != 0o700 {
		t.Errorf("mode %v left", info.Mode().Perm())
	}

	// A link (to a directory someone else controls, say) is not used
	os.Remove(private)
	other := t.TempDir()
	if err := os.Symlink(other, private); err != nil {
		t.Skip("no symlinks:", err)
	}
	if path := instanceSocketPath(long); path != "" {
		t.Errorf("socket through a link: %s", path)
	}
	// And with no socket the copy still runs, as the only one
	inst, ok := acquireSingleInstance(t.TempDir(), nil)
	if !ok {
		t.Fatal("not the running copy")
	}
	inst.Close()
}
