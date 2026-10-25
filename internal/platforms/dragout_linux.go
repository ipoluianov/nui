//go:build linux

package platforms

import (
	"net/url"
	"time"
	"unsafe"
)

// Files dragged out to other applications: the source side of XDND (see
// dnd_linux.go for the receiving one). The window grabs the pointer; under
// the pointer it looks for a window that takes XDND drops (XdndAware),
// announces the drag to it (XdndEnter), reports the pointer (XdndPosition,
// answered with XdndStatus) and drops (XdndDrop). The target asks for the
// files as text/uri-list through the XdndSelection selection, which this
// window owns meanwhile, and ends with XdndFinished.

const (
	xGrabModeAsync = 1
	xGrabSuccess   = 0

	// How long the target has to take the files after the drop
	dragFinishTimeout = 5 * time.Second
)

// dragOut is a drag of files out of a window
type dragOut struct {
	c       *nativeWindow
	uriList []byte
	time    uintptr

	target        uintptr // the window under the pointer that takes drops, 0 if none
	targetVersion int
	accepted      bool
	action        uintptr
	// waiting: an XdndPosition is not answered yet; moved: the pointer moved
	// meanwhile, a new position goes with the answer
	waiting bool
	moved   bool
	rootX   int32
	rootY   int32
}

func (c *nativeWindow) startFileDragSync(files []string) (effect DragEffect, dropped bool, started bool) {
	if c.platform.closed || len(files) == 0 {
		return DragEffectNone, false, false
	}
	display, window := c.platform.display, c.platform.window
	d := &dragOut{c: c}
	for _, f := range files {
		d.uriList = append(d.uriList, (&url.URL{Scheme: "file", Path: f}).String()+"\r\n"...)
	}
	const mask = xButtonPressMask | xButtonReleaseMask | xPointerMotionMask
	if xGrabPointer(display, window, xFalse, mask, xGrabModeAsync, xGrabModeAsync, xNone,
		fontCursor(display, xCursorShape(MouseCursorPointer)), xCurrentTime) != xGrabSuccess {
		return DragEffectNone, false, false
	}
	// Esc cancels the drag
	xGrabKeyboard(display, window, xFalse, xGrabModeAsync, xGrabModeAsync, xCurrentTime)
	defer func() {
		xUngrabPointer(display, xCurrentTime)
		xUngrabKeyboard(display, xCurrentTime)
		xFlush(display)
	}()
	xSetSelectionOwner(display, xdnd.selection, window, xCurrentTime)

	// The pointer where it is now: the first target
	var root, child uintptr
	var rootX, rootY, winX, winY int32
	var state uint32
	xQueryPointer(display, xRootWindow(display, c.platform.screen), unsafe.Pointer(&root), unsafe.Pointer(&child),
		unsafe.Pointer(&rootX), unsafe.Pointer(&rootY), unsafe.Pointer(&winX), unsafe.Pointer(&winY), unsafe.Pointer(&state))
	d.move(rootX, rootY)

	for {
		var event xEvent
		if !d.nextEvent(&event, time.Time{}) {
			continue
		}
		switch event.eventType() {
		case xMotionNotify:
			m := (*xMotionEvent)(unsafe.Pointer(&event))
			d.time = m.Time
			d.move(m.XRoot, m.YRoot)
		case xButtonRelease:
			if b := (*xButtonEvent)(unsafe.Pointer(&event)); b.Button == 1 {
				d.time = b.Time
				effect, dropped = d.drop()
				return effect, dropped, true
			}
		case xKeyPress:
			if ConvertLinuxKeyToNuiKey(int((*xKeyEvent)(unsafe.Pointer(&event)).Keycode)) == KeyEsc {
				d.leave()
				return DragEffectNone, false, true
			}
		case xClientMessage:
			d.status((*xClientMessageEvent)(unsafe.Pointer(&event)))
		}
	}
}

// nextEvent waits for an event of the drag, answering the requests for the
// data and handing the other events to their windows; false if the
// deadline (if any) passes first
func (d *dragOut) nextEvent(event *xEvent, deadline time.Time) bool {
	display := d.c.platform.display
	for xPending(display) == 0 {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return false
		}
		// The other windows are painted meanwhile
		for _, w := range openWindows() {
			w.idle(false)
		}
		waitForFds([]int{int(xConnectionNumber(display))}, 20*time.Millisecond)
	}
	xNextEvent(display, unsafe.Pointer(event))
	switch event.eventType() {
	case xMotionNotify, xButtonPress, xButtonRelease, xKeyPress, xKeyRelease:
		return true
	case xClientMessage:
		if (*xClientMessageEvent)(unsafe.Pointer(event)).MessageType == xdnd.status ||
			(*xClientMessageEvent)(unsafe.Pointer(event)).MessageType == xdnd.finished {
			return true
		}
	case xSelectionRequest:
		if r := (*xSelectionRequestEvent)(unsafe.Pointer(event)); r.Selection == xdnd.selection {
			d.answerRequest(r)
			return false
		}
	}
	if processClipboardEvent(event) {
		return false
	}
	if w := GetNativeWindowByHandle(event.window()); w != nil && !w.platform.closed {
		w.processEvent(event)
	} else if p := getPopupByWindow(event.window()); p != nil {
		p.processEvent(event)
	}
	return false
}

// move finds the target under the pointer and tells it where the pointer is
func (d *dragOut) move(rootX, rootY int32) {
	d.rootX, d.rootY = rootX, rootY
	target, version := d.targetAt(rootX, rootY)
	if target != d.target {
		d.leave()
		d.target, d.targetVersion = target, version
		if target != 0 {
			d.send(xdnd.enter, [5]int64{0, int64(min(version, xdndVersion)) << 24, int64(xdnd.uriList), 0, 0})
		}
	}
	if d.target == 0 {
		return
	}
	if d.waiting {
		d.moved = true
		return
	}
	d.sendPosition()
}

func (d *dragOut) sendPosition() {
	d.waiting, d.moved = true, false
	d.send(xdnd.position, [5]int64{0, 0, int64(d.rootX)<<16 | int64(d.rootY)&0xFFFF, int64(d.time), int64(xdnd.actionCopy)})
}

// status takes the target's answer to a position
func (d *dragOut) status(m *xClientMessageEvent) {
	if m.MessageType != xdnd.status || d.target == 0 || uintptr(m.dataLong(0)) != d.target {
		return
	}
	d.waiting = false
	d.accepted = m.dataLong(1)&1 != 0
	d.action = uintptr(m.dataLong(4))
	if d.moved {
		d.sendPosition()
	}
}

// leave tells the target the drag left it
func (d *dragOut) leave() {
	if d.target != 0 {
		d.send(xdnd.leave, [5]int64{})
	}
	d.target, d.accepted, d.waiting, d.moved = 0, false, false, false
}

// drop drops the files on the target that accepts them, and waits until
// it took them
func (d *dragOut) drop() (DragEffect, bool) {
	// The answer to the last position decides
	deadline := time.Now().Add(dragFinishTimeout)
	for d.target != 0 && d.waiting {
		var event xEvent
		if !d.nextEvent(&event, deadline) {
			break
		}
		if event.eventType() == xClientMessage {
			d.status((*xClientMessageEvent)(unsafe.Pointer(&event)))
		}
	}
	if d.target == 0 || !d.accepted {
		d.leave()
		return DragEffectNone, false
	}
	d.send(xdnd.drop, [5]int64{0, 0, int64(d.time), 0, 0})
	for {
		var event xEvent
		if !d.nextEvent(&event, deadline) {
			if time.Now().After(deadline) {
				return DragEffectNone, false
			}
			continue
		}
		m := (*xClientMessageEvent)(unsafe.Pointer(&event))
		if event.eventType() != xClientMessage || m.MessageType != xdnd.finished || uintptr(m.dataLong(0)) != d.target {
			continue
		}
		action := d.action
		if d.targetVersion >= 5 {
			if m.dataLong(1)&1 == 0 {
				return DragEffectNone, false
			}
			action = uintptr(m.dataLong(2))
		}
		return dragEffectOf(action), true
	}
}

// dragEffectOf is the effect of an XDND action
func dragEffectOf(action uintptr) DragEffect {
	switch action {
	case xdnd.actionMove:
		return DragEffectMove
	case xdnd.actionLink:
		return DragEffectLink
	}
	return DragEffectCopy
}

// send sends an XDND message about this window to the target
func (d *dragOut) send(messageType uintptr, data [5]int64) {
	data[0] = int64(d.c.platform.window)
	target := d.target
	var event xEvent
	m := (*xClientMessageEvent)(unsafe.Pointer(&event))
	m.Type = xClientMessage
	m.Window = target
	m.MessageType = messageType
	m.Format = 32
	m.Data = data
	xSendEvent(d.c.platform.display, target, xFalse, 0, unsafe.Pointer(&event))
	xFlush(d.c.platform.display)
}

// targetAt returns the window under the point that takes XDND drops, and
// its version of XDND; the windows of this application are not targets
// (the form handles a drag over itself)
func (d *dragOut) targetAt(rootX, rootY int32) (uintptr, int) {
	display := d.c.platform.display
	root := xRootWindow(display, d.c.platform.screen)
	w := root
	for range 16 {
		var x, y int32
		var child uintptr
		if xTranslateCoordinates(display, root, w, rootX, rootY, unsafe.Pointer(&x), unsafe.Pointer(&y), unsafe.Pointer(&child)) == 0 || child == 0 {
			return 0, 0
		}
		w = child
		if GetNativeWindowByHandle(w) != nil {
			return 0, 0
		}
		if version, ok := xdndAwareVersion(display, w); ok {
			return w, version
		}
	}
	return 0, 0
}

// xdndAwareVersion reads the XdndAware property of the window
func xdndAwareVersion(display, window uintptr) (int, bool) {
	var actualType uintptr
	var actualFormat int32
	var nitems, bytesAfter uintptr
	var prop unsafe.Pointer
	status := xGetWindowProperty(display, window, xdnd.aware, 0, 1, xFalse, xXAAtom,
		unsafe.Pointer(&actualType), unsafe.Pointer(&actualFormat), unsafe.Pointer(&nitems), unsafe.Pointer(&bytesAfter), unsafe.Pointer(&prop))
	if status != xSuccess || prop == nil {
		return 0, false
	}
	defer xFree(prop)
	if actualFormat != 32 || nitems == 0 {
		return 0, false
	}
	return int(*(*uintptr)(prop)), true
}

// answerRequest gives the files to the target: text/uri-list, or the list
// of the types (TARGETS)
func (d *dragOut) answerRequest(r *xSelectionRequestEvent) {
	var reply xEvent
	n := (*xSelectionEvent)(unsafe.Pointer(&reply))
	n.Type = xSelectionNotify
	n.Requestor = r.Requestor
	n.Selection = r.Selection
	n.Target = r.Target
	n.Time = r.Time
	property := r.Property
	if property == xNone {
		property = r.Target
	}
	display := d.c.platform.display
	switch r.Target {
	case xdnd.uriList:
		data := append(d.uriList[:len(d.uriList):len(d.uriList)], 0)
		xChangeProperty(display, r.Requestor, property, xdnd.uriList, 8, xPropModeReplace, unsafe.Pointer(&data[0]), int32(len(data)-1))
		n.Property = property
	case xdnd.targets:
		atoms := []uintptr{xdnd.targets, xdnd.uriList}
		xChangeProperty(display, r.Requestor, property, xXAAtom, 32, xPropModeReplace, unsafe.Pointer(&atoms[0]), int32(len(atoms)))
		n.Property = property
	}
	xSendEvent(display, r.Requestor, xFalse, 0, unsafe.Pointer(&reply))
	xFlush(display)
}
