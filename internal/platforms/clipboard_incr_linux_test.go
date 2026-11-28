//go:build linux

package platforms

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
	"unsafe"
)

// The X11 tests here need an X server, opt-in like needX says:
// NUI_TEST_X11=1 DISPLAY=:98 go test -run TestClipboardIncr ./internal/platforms
// on a throwaway Xvfb. Each one in its own run is the safest: they share the
// process-wide connection of the clipboard.

// A text larger than one request of the server used to be written whole
// into the requestor's property: the server refused it (BadLength) while
// the requestor was told it got it, and read nothing. It goes in parts
// (INCR) now.
func TestClipboardIncrServesLargeText(t *testing.T) {
	needX(t)
	if _, _, err := clipboardWindow(); err != nil {
		t.Fatal(err)
	}
	size := maxPropertySize(xDisplay) + 3<<20 + 17
	text := strings.Repeat("abcdefgh", size/8+1)[:size]
	if err := clipboardSet([]ClipboardItem{{MIMEText, []byte(text)}}); err != nil {
		t.Fatal(err)
	}

	// A second client pastes, in parts
	d2 := xOpenDisplay(0)
	if d2 == 0 {
		t.Fatal("no display")
	}
	defer xCloseDisplay(d2)
	root := xRootWindow(d2, xDefaultScreen(d2))
	win := xCreateWindow(d2, root, 0, 0, 1, 1, 0, 0, xInputOnly, 0, 0, nil)
	xSelectInput(d2, win, xPropertyChangeMask)
	clipAtom := xInternAtom(d2, "CLIPBOARD", xFalse)
	utf8 := xInternAtom(d2, "UTF8_STRING", xFalse)
	incr := xInternAtom(d2, "INCR", xFalse)
	prop := xInternAtom(d2, "NUI_TEST_PROP", xFalse)
	xConvertSelection(d2, clipAtom, utf8, prop, win, xCurrentTime)
	xFlush(d2)

	read := func() ([]byte, uintptr) {
		var typ uintptr
		var format int32
		var n, after uintptr
		var data unsafe.Pointer
		if xGetWindowProperty(d2, win, prop, 0, 1<<26, xTrue, xAnyPropertyType, unsafe.Pointer(&typ),
			unsafe.Pointer(&format), unsafe.Pointer(&n), unsafe.Pointer(&after), unsafe.Pointer(&data)) != xSuccess || data == nil {
			return nil, typ
		}
		defer xFree(data)
		if format != 8 {
			return nil, typ
		}
		xFlush(d2)
		return bytes.Clone(unsafe.Slice((*byte)(data), int(n))), typ
	}

	var got []byte
	notified, incrStarted, done := false, false, false
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline) && !done; {
		for xPending(xDisplay) > 0 {
			var ev xEvent
			xNextEvent(xDisplay, unsafe.Pointer(&ev))
			processClipboardEvent(&ev)
		}
		var ev xEvent
		if !notified && xCheckTypedWindowEvent(d2, win, xSelectionNotify, unsafe.Pointer(&ev)) != 0 {
			notified = true
			if (*xSelectionEvent)(unsafe.Pointer(&ev)).Property == xNone {
				t.Fatal("the paste was refused")
			}
			data, typ := read()
			if typ != incr {
				t.Fatalf("the property came whole, type %d, %d bytes", typ, len(data))
			}
			incrStarted = true
			continue
		}
		if incrStarted && xCheckTypedWindowEvent(d2, win, xPropertyNotify, unsafe.Pointer(&ev)) != 0 {
			if p := (*xPropertyEvent)(unsafe.Pointer(&ev)); p.Atom != prop || p.State != xPropertyNewValue {
				continue
			}
			part, typ := read()
			if typ == xNone {
				continue
			}
			if typ != utf8 {
				t.Fatalf("a part of type %d", typ)
			}
			if len(part) == 0 {
				done = true
				break
			}
			got = append(got, part...)
			continue
		}
		time.Sleep(time.Millisecond)
	}
	if !done {
		t.Fatalf("the transfer did not end: notified=%v, %d of %d bytes", notified, len(got), len(text))
	}
	if string(got) != text {
		t.Errorf("got %d bytes, want %d", len(got), len(text))
	}
	if len(incrTransfers) != 0 {
		t.Errorf("%d transfers left", len(incrTransfers))
	}
}

// incrOwner owns CLIPBOARD on its own connection and answers UTF8_STRING
// in parts: chunk(i) gives the part i, the delay before it, and false for
// the end
func incrOwner(t *testing.T, chunk func(i int) ([]byte, time.Duration, bool), stop <-chan struct{}) {
	d2 := xOpenDisplay(0)
	if d2 == 0 {
		t.Error("no display")
		return
	}
	defer xCloseDisplay(d2)
	root := xRootWindow(d2, xDefaultScreen(d2))
	win := xCreateWindow(d2, root, 0, 0, 1, 1, 0, 0, xInputOnly, 0, 0, nil)
	clipAtom := xInternAtom(d2, "CLIPBOARD", xFalse)
	utf8 := xInternAtom(d2, "UTF8_STRING", xFalse)
	incr := xInternAtom(d2, "INCR", xFalse)
	xSetSelectionOwner(d2, clipAtom, win, xCurrentTime)
	xFlush(d2)
	stopped := func() bool {
		select {
		case <-stop:
			return true
		default:
			return false
		}
	}
	for !stopped() {
		var ev xEvent
		if xCheckTypedWindowEvent(d2, win, xSelectionRequest, unsafe.Pointer(&ev)) == 0 {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		r := *(*xSelectionRequestEvent)(unsafe.Pointer(&ev))
		var reply xEvent
		n := (*xSelectionEvent)(unsafe.Pointer(&reply))
		n.Type, n.Requestor, n.Selection, n.Target, n.Time, n.Property = xSelectionNotify, r.Requestor, r.Selection, r.Target, r.Time, xNone
		if r.Target != utf8 {
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
		for i := 0; ; i++ {
			// The requestor deletes the property to ask for the next part
			deleted := false
			for !deleted {
				if stopped() {
					return
				}
				var pe xEvent
				if xCheckTypedWindowEvent(d2, r.Requestor, xPropertyNotify, unsafe.Pointer(&pe)) != 0 {
					p := (*xPropertyEvent)(unsafe.Pointer(&pe))
					deleted = p.Atom == r.Property && p.State == xPropertyDelete
					continue
				}
				time.Sleep(time.Millisecond)
			}
			data, delay, more := chunk(i)
			time.Sleep(delay)
			if !more {
				data = nil
			}
			buf := append(data[:len(data):len(data)], 0)
			xChangeProperty(d2, r.Requestor, r.Property, utf8, 8, xPropModeReplace, unsafe.Pointer(&buf[0]), int32(len(buf)-1))
			xFlush(d2)
			if !more {
				break
			}
		}
	}
}

// An owner that sends its parts slowly used to hold the UI thread for as
// long as it went on: the timeout applied to each part only
func TestClipboardIncrSlowOwnerBounded(t *testing.T) {
	needX(t)
	if _, _, err := clipboardWindow(); err != nil {
		t.Fatal(err)
	}
	old := clipboardIncrTimeout
	clipboardIncrTimeout = 3 * time.Second
	defer func() { clipboardIncrTimeout = old }()
	stop := make(chan struct{})
	defer close(stop)
	go incrOwner(t, func(i int) ([]byte, time.Duration, bool) {
		return []byte("x"), time.Second, i < 20 // 20 s in all, each part in time
	}, stop)
	time.Sleep(300 * time.Millisecond)
	start := time.Now()
	text, err := clipboardGetText()
	took := time.Since(start)
	if took > clipboardIncrTimeout+clipboardTimeout/2 {
		t.Errorf("the paste held the UI thread %v (bound %v)", took, clipboardIncrTimeout)
	}
	if err == nil || text != "" {
		t.Errorf("text %q, err %v: want no text and an error", text, err)
	}
}

// An owner that never ends its parts used to fill the memory
func TestClipboardIncrSizeBounded(t *testing.T) {
	needX(t)
	if _, _, err := clipboardWindow(); err != nil {
		t.Fatal(err)
	}
	old := clipboardMaxSize
	clipboardMaxSize = 1000
	defer func() { clipboardMaxSize = old }()
	stop := make(chan struct{})
	defer close(stop)
	go incrOwner(t, func(i int) ([]byte, time.Duration, bool) {
		return []byte(strings.Repeat("y", 100)), 0, true // never ends
	}, stop)
	time.Sleep(300 * time.Millisecond)
	text, err := clipboardGetText()
	if !errors.Is(err, errClipboardTooLarge) || text != "" {
		t.Errorf("%d bytes, err %v: want errClipboardTooLarge", len(text), err)
	}
}
