//go:build linux

package platforms

import (
	"testing"
	"time"

	"github.com/ebitengine/purego"
)

var (
	auditXTestMotion func(display uintptr, screen int32, x, y int32, delay uintptr) int32
	auditXTestButton func(display uintptr, button uint32, press int32, delay uintptr) int32
)

func auditLoadXTest(t *testing.T) {
	if auditXTestButton != nil {
		return
	}
	h, err := dlopenFirst("libXtst.so.6")
	if err != nil {
		t.Skip(err)
	}
	purego.RegisterLibFunc(&auditXTestMotion, h, "XTestFakeMotionEvent")
	purego.RegisterLibFunc(&auditXTestButton, h, "XTestFakeButtonEvent")
}

// auditDrag starts a drag of a file out of a shown window; closeFirst closes
// the window (Close is documented safe from any goroutine) before the user
// releases the button. Reports whether the drag returned.
func auditDrag(t *testing.T, closeFirst bool) bool {
	w := createWindow("drag", 0, 0, 300, 300, false, false)
	w.Show()
	time.Sleep(300 * time.Millisecond)
	processXEvents()

	d2 := xOpenDisplay(0)
	go func() {
		time.Sleep(300 * time.Millisecond)
		if closeFirst {
			w.Close()
			time.Sleep(300 * time.Millisecond)
		}
		auditXTestMotion(d2, 0, 150, 150, 0)
		auditXTestButton(d2, 1, 1, 0)
		auditXTestButton(d2, 1, 0, 0)
		xFlush(d2)
	}()
	ret := make(chan bool, 1)
	go func() {
		_, _, started := w.startFileDragSync([]string{"/tmp/audit.txt"})
		ret <- started
	}()
	select {
	case started := <-ret:
		t.Logf("drag returned (started=%v), window closed=%v", started, w.platform.closed)
		return true
	case <-time.After(4 * time.Second):
		t.Logf("drag still running 4 s after the button release; window closed=%v", w.platform.closed)
		return false
	}
}

func TestAuditDragControl(t *testing.T) {
	auditNeedX(t)
	auditLoadXTest(t)
	if !auditDrag(t, false) {
		t.Fatal("control: drag did not end on the button release")
	}
}

func TestAuditDragWindowClosedHangs(t *testing.T) {
	auditNeedX(t)
	auditLoadXTest(t)
	if !auditDrag(t, true) {
		t.Error("the drag loop never returns once its window is closed during the drag")
	}
}
