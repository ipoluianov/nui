//go:build linux

package platforms

import (
	"net/url"
	"strings"
	"unsafe"
)

// Files dropped from other applications: the receiving side of the XDND
// protocol (version 5), https://freedesktop.org/wiki/Specifications/XDND/.
//
// The source announces a drag over the window with XdndEnter (the data types
// it offers), reports the pointer with XdndPosition (answered with
// XdndStatus: accept or not) and ends with XdndDrop. The window then asks for
// the data - a text/uri-list - through the XdndSelection selection; it
// arrives in a SelectionNotify event, and XdndFinished tells the source the
// drop is done.

const (
	xdndVersion      = 5
	xSelectionNotify = 31
	xAnyPropertyType = 0
)

// XSelectionEvent, field order per Xlib.h
type xSelectionEvent struct {
	Type      int32
	Serial    uintptr
	SendEvent int32
	Display   uintptr
	Requestor uintptr
	Selection uintptr
	Target    uintptr
	Property  uintptr
	Time      uintptr
}

var xdnd struct {
	aware, enter, position, status, leave, drop, finished uintptr
	selection, typeList, actionCopy, uriList, property    uintptr
}

// initXdndAtoms interns the XDND atoms once the display is open
func initXdndAtoms(display uintptr) {
	intern := func(name string) uintptr { return xInternAtom(display, name, xFalse) }
	xdnd.aware = intern("XdndAware")
	xdnd.enter = intern("XdndEnter")
	xdnd.position = intern("XdndPosition")
	xdnd.status = intern("XdndStatus")
	xdnd.leave = intern("XdndLeave")
	xdnd.drop = intern("XdndDrop")
	xdnd.finished = intern("XdndFinished")
	xdnd.selection = intern("XdndSelection")
	xdnd.typeList = intern("XdndTypeList")
	xdnd.actionCopy = intern("XdndActionCopy")
	xdnd.uriList = intern("text/uri-list")
	xdnd.property = intern("NUI_XDND_DATA")
}

// dndState is a drag from another application over the window
type dndState struct {
	source  uintptr
	hasURIs bool
	x, y    int
}

// enableFileDrop tells the drag sources the window takes XDND drops
func (c *nativeWindow) enableFileDrop() {
	version := uintptr(xdndVersion)
	xChangeProperty(c.platform.display, c.platform.window, xdnd.aware, xXAAtom, 32, xPropModeReplace, unsafe.Pointer(&version), 1)
}

// processXdnd handles an XDND client message; returns false for other messages
func (c *nativeWindow) processXdnd(m *xClientMessageEvent) bool {
	d := &c.platform.dnd
	switch m.MessageType {
	case xdnd.enter:
		*d = dndState{source: uintptr(m.dataLong(0))}
		if m.dataLong(1)&1 != 0 {
			// More than three types: they are in the source's XdndTypeList
			d.hasURIs = c.windowHasAtomInList(d.source, xdnd.typeList, xdnd.uriList)
		} else {
			for i := 2; i <= 4; i++ {
				if uintptr(m.dataLong(i)) == xdnd.uriList {
					d.hasURIs = true
				}
			}
		}
	case xdnd.position:
		rootX := int32(m.dataLong(2) >> 16 & 0xFFFF)
		rootY := int32(m.dataLong(2) & 0xFFFF)
		var x, y int32
		var child uintptr
		root := xRootWindow(c.platform.display, c.platform.screen)
		xTranslateCoordinates(c.platform.display, root, c.platform.window, rootX, rootY, unsafe.Pointer(&x), unsafe.Pointer(&y), unsafe.Pointer(&child))
		d.x, d.y = int(x), int(y)
		c.sendXdndStatus(d.source, d.hasURIs && c.onFilesDropped != nil && !c.inputBlocked())
	case xdnd.leave:
		*d = dndState{}
	case xdnd.drop:
		if !d.hasURIs || c.onFilesDropped == nil || c.inputBlocked() {
			c.sendXdndFinished(d.source, false)
			*d = dndState{}
			break
		}
		// The data comes in a SelectionNotify, see processXdndSelection
		xConvertSelection(c.platform.display, xdnd.selection, xdnd.uriList, xdnd.property, c.platform.window, uintptr(m.dataLong(2)))
	default:
		return false
	}
	return true
}

// processXdndSelection reads the dropped file list
func (c *nativeWindow) processXdndSelection(e *xSelectionEvent) {
	d := c.platform.dnd
	if d.source == 0 || e.Selection != xdnd.selection {
		return
	}
	c.platform.dnd = dndState{}
	if e.Property == xNone {
		c.sendXdndFinished(d.source, false)
		return
	}

	var actualType uintptr
	var actualFormat int32
	var nitems, bytesAfter uintptr
	var prop unsafe.Pointer
	status := xGetWindowProperty(c.platform.display, c.platform.window, xdnd.property, 0, 1<<24, xTrue, xAnyPropertyType,
		unsafe.Pointer(&actualType), unsafe.Pointer(&actualFormat), unsafe.Pointer(&nitems), unsafe.Pointer(&bytesAfter), unsafe.Pointer(&prop))
	var files []string
	if status == xSuccess && prop != nil {
		if actualFormat == 8 && nitems > 0 {
			files = parseURIList(string(unsafe.Slice((*byte)(prop), int(nitems))))
		}
		xFree(prop)
	}
	c.sendXdndFinished(d.source, len(files) > 0)
	if len(files) > 0 && c.onFilesDropped != nil {
		c.onFilesDropped(files, d.x, d.y)
	}
}

// windowHasAtomInList reports whether the atom list property of window holds atom
func (c *nativeWindow) windowHasAtomInList(window, property, atom uintptr) bool {
	var actualType uintptr
	var actualFormat int32
	var nitems, bytesAfter uintptr
	var prop unsafe.Pointer
	status := xGetWindowProperty(c.platform.display, window, property, 0, 1024, xFalse, xXAAtom,
		unsafe.Pointer(&actualType), unsafe.Pointer(&actualFormat), unsafe.Pointer(&nitems), unsafe.Pointer(&bytesAfter), unsafe.Pointer(&prop))
	if status != xSuccess || prop == nil {
		return false
	}
	defer xFree(prop)
	if actualFormat != 32 {
		return false
	}
	for _, a := range unsafe.Slice((*uintptr)(prop), int(nitems)) {
		if a == atom {
			return true
		}
	}
	return false
}

// sendXdndMessage sends an XDND client message about the window to the source
func (c *nativeWindow) sendXdndMessage(source, messageType uintptr, data [5]int64) {
	// XSendEvent reads a whole XEvent, so the message is built in one
	var event xEvent
	m := (*xClientMessageEvent)(unsafe.Pointer(&event))
	m.Type = xClientMessage
	m.Window = source
	m.MessageType = messageType
	m.Format = 32
	m.Data = data
	xSendEvent(c.platform.display, source, xFalse, 0, unsafe.Pointer(&event))
	xFlush(c.platform.display)
}

func (c *nativeWindow) sendXdndStatus(source uintptr, accept bool) {
	// Bit 1: send XdndPosition for every move, not only outside a rectangle
	data := [5]int64{int64(c.platform.window), 2, 0, 0, 0}
	if accept {
		data[1] |= 1
		data[4] = int64(xdnd.actionCopy)
	}
	c.sendXdndMessage(source, xdnd.status, data)
}

func (c *nativeWindow) sendXdndFinished(source uintptr, accepted bool) {
	if source == 0 {
		return
	}
	data := [5]int64{int64(c.platform.window), 0, 0, 0, 0}
	if accepted {
		data[1] = 1
		data[2] = int64(xdnd.actionCopy)
	}
	c.sendXdndMessage(source, xdnd.finished, data)
}

// parseURIList returns the local file paths of a text/uri-list
func parseURIList(list string) []string {
	var files []string
	for _, line := range strings.Split(list, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, err := url.Parse(line)
		if err != nil || u.Scheme != "file" || u.Path == "" {
			continue
		}
		files = append(files, u.Path)
	}
	return files
}
