//go:build linux

package platforms

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Dragging out a relative path used to give file://docs/report.txt: the
// first folder became the host, and the receiver read it as /report.txt
// (or dropped file://report.txt). The paths go out absolute and come back
// the same.
func TestDragOutURIListRoundTrip(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	files := []string{"/tmp/a b#c?d%e.txt", "docs/report.txt", "report.txt", "./x/../y.txt"}
	want := []string{"/tmp/a b#c?d%e.txt", filepath.Join(wd, "docs/report.txt"),
		filepath.Join(wd, "report.txt"), filepath.Join(wd, "y.txt")}
	if got := parseURIList(string(fileURIList(files))); !slices.Equal(got, want) {
		t.Errorf("%q -> %q, want %q", files, got, want)
	}
	if got := fileURIList([]string{""}); len(got) != 0 {
		t.Errorf("an empty path gives %q", got)
	}
}

// A file URI of another machine is not the local file of the same path
func TestDropURIListRemoteHost(t *testing.T) {
	list := "file:///a\r\nfile://localhost/b\r\nfile://LOCALHOST/c\r\nfile://otherhost/etc/passwd\r\nfile://docs/report.txt\r\n"
	want := []string{"/a", "/b", "/c"}
	if name, err := os.Hostname(); err == nil && name != "" {
		list += "file://" + name + "/d\r\n"
		want = append(want, "/d")
	}
	if got := parseURIList(list); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
