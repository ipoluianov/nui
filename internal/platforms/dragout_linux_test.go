//go:build linux

package platforms

import (
	"os"
	"testing"
	"time"
)

// needX skips the test without an X server: it needs DISPLAY and libX11.
// Run it on a throwaway server, e.g. Xvfb :97 and DISPLAY=:97.
func needX(t *testing.T) {
	t.Helper()
	// Opt-in: the test shows a window and grabs the pointer, which a run on
	// the developer's own desktop should not do. Run it on Xvfb:
	// NUI_TEST_X11=1 DISPLAY=:97 go test ./internal/platforms
	if os.Getenv("NUI_TEST_X11") != "1" {
		t.Skip("set NUI_TEST_X11=1 (with DISPLAY of an Xvfb) to run")
	}
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	if x11LoadError != nil {
		t.Skip(x11LoadError)
	}
	if xDisplay == 0 {
		probe := xOpenDisplay(0)
		if probe == 0 {
			t.Skip("can't open the display")
		}
		xCloseDisplay(probe)
	}
}

// A drag of files out of a window whose window is closed during the drag
// (Close from any goroutine, or the close button through the window
// manager) ends as a cancel. The window closed in the drag's own loop and
// its grabs went with it, so the button release went to another window and
// the drag loop, waiting for it, never returned: the UI thread hung.
func TestDragOutEndsWhenWindowCloses(t *testing.T) {
	needX(t)
	w := createWindow("drag", 0, 0, 300, 300, false, false)
	w.Show()
	time.Sleep(300 * time.Millisecond) // mapped: a pointer grab needs a viewable window
	processXEvents()

	type result struct {
		effect           DragEffect
		dropped, started bool
	}
	ret := make(chan result, 1)
	go func() {
		effect, dropped, started := w.startFileDragSync([]string{"/tmp/dragged.txt"})
		ret <- result{effect, dropped, started}
	}()
	time.Sleep(300 * time.Millisecond)
	w.Close()

	select {
	case r := <-ret:
		if !r.started {
			t.Skip("the pointer grab was refused")
		}
		if r.effect != DragEffectNone || r.dropped {
			t.Errorf("effect %v, dropped %v; want a cancel", r.effect, r.dropped)
		}
		if !w.platform.closed {
			t.Error("the window is not closed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the drag did not end 5 s after its window was closed")
	}
}
