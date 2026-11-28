//go:build linux

package platforms

import (
	"image"
	"os"
	"strings"
	"testing"
	"time"
	"unsafe"
)

// Run with DISPLAY pointing at a throwaway Xvfb, e.g. DISPLAY=:97.
func auditNeedX(t *testing.T) {
	if os.Getenv("NUI_AUDIT_X") == "" {
		t.Skip("set NUI_AUDIT_X=1 and DISPLAY to an Xvfb")
	}
}

// A 2048x2048 icon is 4194306 32-bit words in one ChangeProperty request:
// over the BIG-REQUESTS maximum of the server (4194303 words by default).
func TestAuditBigIcon(t *testing.T) {
	auditNeedX(t)
	w := createWindow("icon", 0, 0, 200, 200, false, false)
	t.Logf("window created")
	icon := image.NewRGBA(image.Rect(0, 0, 2048, 2048))
	w.SetAppIcon(icon)
	xSync(w.platform.display, xFalse)
	t.Logf("survived SetAppIcon 2048x2048")
}

// The text copied here is written whole into the requestor's property, without
// the size check the other formats get (maxPropertySize) or INCR.
func TestAuditBigClipboardText(t *testing.T) {
	auditNeedX(t)
	if err := clipboardSet([]ClipboardItem{{MIMEText, []byte(strings.Repeat("a", 20<<20))}}); err != nil {
		t.Fatal(err)
	}
	// A second client pastes
	d2 := xOpenDisplay(0)
	if d2 == 0 {
		t.Fatal("no display")
	}
	root := xRootWindow(d2, xDefaultScreen(d2))
	win := xCreateWindow(d2, root, 0, 0, 1, 1, 0, 0, xInputOnly, 0, 0, nil)
	clipAtom := xInternAtom(d2, "CLIPBOARD", xFalse)
	utf8 := xInternAtom(d2, "UTF8_STRING", xFalse)
	prop := xInternAtom(d2, "AUDIT_PROP", xFalse)
	xConvertSelection(d2, clipAtom, utf8, prop, win, xCurrentTime)
	xFlush(d2)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for xPending(xDisplay) > 0 {
			var ev xEvent
			xNextEvent(xDisplay, unsafe.Pointer(&ev))
			processClipboardEvent(&ev)
		}
		var ev xEvent
		if xCheckTypedWindowEvent(d2, win, xSelectionNotify, unsafe.Pointer(&ev)) != 0 {
			t.Logf("requestor got SelectionNotify, property=%d", (*xSelectionEvent)(unsafe.Pointer(&ev)).Property)
			var typ uintptr
			var format int32
			var n, after uintptr
			var data unsafe.Pointer
			st := xGetWindowProperty(d2, win, prop, 0, 1<<26, xTrue, xAnyPropertyType, unsafe.Pointer(&typ), unsafe.Pointer(&format), unsafe.Pointer(&n), unsafe.Pointer(&after), unsafe.Pointer(&data))
			t.Logf("requestor reads property: status=%d type=%d format=%d nitems=%d (want 20971520)", st, typ, format, n)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	xSync(xDisplay, xFalse)
	t.Logf("owner survived serving 20 MB text")
}

// auditIncrOwner owns CLIPBOARD on its own connection and answers with INCR:
// utf8 requests are refused when refuseUTF8; chunks() gives the parts.
func auditIncrOwner(t *testing.T, refuseUTF8 bool, chunks func(i int) ([]byte, time.Duration, bool), stop <-chan struct{}) {
	d2 := xOpenDisplay(0)
	if d2 == 0 {
		t.Error("no display")
		return
	}
	root := xRootWindow(d2, xDefaultScreen(d2))
	win := xCreateWindow(d2, root, 0, 0, 1, 1, 0, 0, xInputOnly, 0, 0, nil)
	clipAtom := xInternAtom(d2, "CLIPBOARD", xFalse)
	utf8 := xInternAtom(d2, "UTF8_STRING", xFalse)
	incr := xInternAtom(d2, "INCR", xFalse)
	xSetSelectionOwner(d2, clipAtom, win, xCurrentTime)
	xFlush(d2)
	for {
		select {
		case <-stop:
			return
		default:
		}
		var ev xEvent
		if xCheckTypedWindowEvent(d2, win, xSelectionRequest, unsafe.Pointer(&ev)) == 0 {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		r := *(*xSelectionRequestEvent)(unsafe.Pointer(&ev))
		var reply xEvent
		n := (*xSelectionEvent)(unsafe.Pointer(&reply))
		n.Type, n.Requestor, n.Selection, n.Target, n.Time, n.Property = xSelectionNotify, r.Requestor, r.Selection, r.Target, r.Time, xNone
		if r.Target == utf8 && refuseUTF8 || (r.Target != utf8 && r.Target != xXAString) {
			xSendEvent(d2, r.Requestor, xFalse, 0, unsafe.Pointer(&reply))
			xFlush(d2)
			continue
		}
		xSelectInput(d2, r.Requestor, xPropertyChangeMask)
		size := uintptr(1)
		xChangeProperty(d2, r.Requestor, r.Property, incr, 32, xPropModeReplace, unsafe.Pointer(&size), 1)
		n.Property = r.Property
		xSendEvent(d2, r.Requestor, xFalse, 0, unsafe.Pointer(&reply))
		xFlush(d2)
		waitDelete := func() bool {
			for {
				select {
				case <-stop:
					return false
				default:
				}
				var pe xEvent
				if xCheckTypedWindowEvent(d2, r.Requestor, xPropertyNotify, unsafe.Pointer(&pe)) != 0 {
					p := (*xPropertyEvent)(unsafe.Pointer(&pe))
					if p.Atom == r.Property && p.State == 1 {
						return true
					}
					continue
				}
				time.Sleep(time.Millisecond)
			}
		}
		for i := 0; ; i++ {
			if !waitDelete() {
				return
			}
			data, delay, more := chunks(i)
			time.Sleep(delay)
			if !more {
				data = nil
			}
			buf := append(data[:len(data):len(data)], 0)
			xChangeProperty(d2, r.Requestor, r.Property, r.Target, 8, xPropModeReplace, unsafe.Pointer(&buf[0]), int32(len(buf)-1))
			xFlush(d2)
			if !more {
				break
			}
		}
	}
}

// Latin-1 STRING sent in INCR parts is not converted to UTF-8
func TestAuditIncrLatin1(t *testing.T) {
	auditNeedX(t)
	if _, _, err := clipboardWindow(); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	defer close(stop)
	go auditIncrOwner(t, true, func(i int) ([]byte, time.Duration, bool) {
		return []byte{'c', 'a', 'f', 0xE9}, 0, i == 0
	}, stop)
	time.Sleep(300 * time.Millisecond)
	text, err := clipboardGetText()
	t.Logf("text=%q err=%v (want \"café\")", text, err)
	if text != "café" {
		t.Errorf("INCR STRING not converted from Latin-1: %q", text)
	}
}

// An owner that sends its parts slowly holds the UI thread with no bound:
// clipboardTimeout (2 s) applies per part only
func TestAuditIncrSlowDrip(t *testing.T) {
	auditNeedX(t)
	if _, _, err := clipboardWindow(); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	defer close(stop)
	go auditIncrOwner(t, false, func(i int) ([]byte, time.Duration, bool) {
		return []byte("x"), 1500 * time.Millisecond, i < 6
	}, stop)
	time.Sleep(300 * time.Millisecond)
	start := time.Now()
	text, err := clipboardGetText()
	t.Logf("clipboardGetText returned after %v: len=%d err=%v (clipboardTimeout=%v)", time.Since(start), len(text), err, clipboardTimeout)
	if time.Since(start) > 2*clipboardTimeout {
		t.Errorf("UI thread blocked %v by one paste", time.Since(start))
	}
}
