//go:build linux

package platforms

import (
	"errors"
	"fmt"
	"slices"
	"time"
	"unsafe"
)

// The clipboard is the CLIPBOARD selection of X11 (ICCCM, section 2), served
// through Xlib like the rest of the window system, so it needs no cgo.
//
// Copying makes a hidden window of the application the owner of the
// selection and keeps the text; the applications that paste ask that window
// for it (SelectionRequest), handled in the event loop with the other
// events. Pasting asks the owner to convert the selection to UTF8_STRING
// into a property of the hidden window and waits for the answer
// (SelectionNotify); large texts come in parts (INCR).
//
// Other formats (an image as PNG, the data of an application) are targets
// named by their MIME types, as GTK and Qt name them.
//
// The data lives in the application: when it quits, a clipboard manager
// (most desktops run one) is asked to keep it, see clipboardHandOver.

const (
	xSelectionClear   = 29
	xSelectionRequest = 30
	xInputOnly        = 2
	xXAString         = 31
	xPropertyNewValue = 0
	xCurrentTime      = 0

	// How long an owner has to answer: a hung application must not hang this one
	clipboardTimeout = 2 * time.Second
)

// XSelectionRequestEvent, field order per Xlib.h
type xSelectionRequestEvent struct {
	Type      int32
	Serial    uintptr
	SendEvent int32
	Display   uintptr
	Owner     uintptr
	Requestor uintptr
	Selection uintptr
	Target    uintptr
	Property  uintptr
	Time      uintptr
}

// XSelectionClearEvent, field order per Xlib.h
type xSelectionClearEvent struct {
	Type      int32
	Serial    uintptr
	SendEvent int32
	Display   uintptr
	Window    uintptr
	Selection uintptr
	Time      uintptr
}

var errClipboardTimeout = errors.New("clipboard: the owner of the clipboard does not answer")

var clip clipboardState

type clipboardState struct {
	window uintptr
	// The data copied here, while this application owns the clipboard: the
	// text, if any, and the other formats by their atoms
	text    string
	hasText bool
	items   map[uintptr][]byte
	owned   bool

	clipboard, targets, utf8, textAtom, plain, incr, property uintptr
	plainBare                                                 uintptr
	manager, saveTargets, multiple, timestamp                 uintptr
}

// clipboardWindow returns the hidden window that owns and receives the
// selections, creating it with the display on the first call. UI thread only.
func clipboardWindow() (display, window uintptr, err error) {
	if x11LoadError != nil {
		return 0, 0, x11LoadError
	}
	if xDisplay == 0 {
		if err := tryOpenDisplay(); err != nil {
			return 0, 0, err
		}
	}
	if clip.window != 0 {
		return xDisplay, clip.window, nil
	}
	intern := func(name string) uintptr { return xInternAtom(xDisplay, name, xFalse) }
	clip.clipboard = intern("CLIPBOARD")
	clip.targets = intern("TARGETS")
	clip.utf8 = intern("UTF8_STRING")
	clip.textAtom = intern("TEXT")
	clip.plain = intern("text/plain;charset=utf-8")
	clip.plainBare = intern("text/plain")
	clip.incr = intern("INCR")
	clip.property = intern("NUI_CLIPBOARD")
	clip.manager = intern("CLIPBOARD_MANAGER")
	clip.multiple = intern("MULTIPLE")
	clip.timestamp = intern("TIMESTAMP")
	clip.saveTargets = intern("SAVE_TARGETS")

	root := xRootWindow(xDisplay, xScreen)
	clip.window = xCreateWindow(xDisplay, root, -10, -10, 1, 1, 0, 0, xInputOnly, 0, 0, nil)
	if clip.window == 0 {
		return 0, 0, errors.New("clipboard: cannot create the window of the clipboard")
	}
	// PropertyNotify brings the parts of a large text
	xSelectInput(xDisplay, clip.window, xPropertyChangeMask)
	return xDisplay, clip.window, nil
}

// tryOpenDisplay opens the display, an error instead of the panic of openDisplay
func tryOpenDisplay() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("clipboard: %v", r)
		}
	}()
	openDisplay()
	return nil
}

func clipboardSet(items []ClipboardItem) error {
	display, window, err := clipboardWindow()
	if err != nil {
		return err
	}
	clip.text, clip.hasText, clip.items = "", false, map[uintptr][]byte{}
	for _, item := range items {
		if isTextMIME(item.MIME) {
			clip.text, clip.hasText = string(item.Data), true
		} else {
			clip.items[xInternAtom(display, item.MIME, xFalse)] = item.Data
		}
	}
	xSetSelectionOwner(display, clip.clipboard, window, xCurrentTime)
	clip.owned = xGetSelectionOwner(display, clip.clipboard) == window
	xFlush(display)
	if !clip.owned {
		clip.text, clip.hasText, clip.items = "", false, nil
		return errors.New("clipboard: another application keeps the clipboard")
	}
	return nil
}

func clipboardGet(mime string) ([]byte, bool, error) {
	if isTextMIME(mime) {
		formats, err := clipboardFormats()
		if err != nil || !slices.Contains(formats, MIMEText) {
			return nil, false, err
		}
		text, err := clipboardGetText()
		return []byte(text), err == nil, err
	}
	display, window, err := clipboardWindow()
	if err != nil {
		return nil, false, err
	}
	owner := xGetSelectionOwner(display, clip.clipboard)
	if owner == xNone {
		return nil, false, nil
	}
	target := xInternAtom(display, mime, xFalse)
	if owner == window && clip.owned {
		data, ok := clip.items[target]
		return data, ok, nil
	}
	data, ok, err := convertClipboard(display, window, target)
	return []byte(data), ok, err
}

func clipboardFormats() ([]string, error) {
	display, window, err := clipboardWindow()
	if err != nil {
		return nil, err
	}
	owner := xGetSelectionOwner(display, clip.clipboard)
	if owner == xNone {
		return nil, nil
	}
	var targets []uintptr
	if owner == window && clip.owned {
		targets = clip.targetAtoms()
	} else {
		xConvertSelection(display, clip.clipboard, clip.targets, clip.property, window, xCurrentTime)
		xFlush(display)
		var event xEvent
		if !waitClipboardEvent(display, window, xSelectionNotify, &event, nil) {
			return nil, errClipboardTimeout
		}
		if (*xSelectionEvent)(unsafe.Pointer(&event)).Property == xNone {
			return nil, nil
		}
		targets = readClipboardAtoms(display, window)
	}
	var formats []string
	for _, atom := range targets {
		switch atom {
		case clip.targets, clip.multiple, clip.timestamp, clip.saveTargets:
			continue
		case clip.utf8, clip.plain, clip.plainBare, clip.textAtom, xXAString:
			if !slices.Contains(formats, MIMEText) {
				formats = append(formats, MIMEText)
			}
			continue
		}
		if name := atomName(display, atom); name != "" {
			formats = append(formats, name)
		}
	}
	return formats, nil
}

// targetAtoms are the targets the data copied here is offered as
func (c *clipboardState) targetAtoms() []uintptr {
	atoms := []uintptr{c.targets}
	if c.hasText {
		atoms = append(atoms, c.utf8, c.plain, c.textAtom, xXAString)
	}
	for atom := range c.items {
		atoms = append(atoms, atom)
	}
	return atoms
}

// readClipboardAtoms reads and deletes the list of atoms the owner wrote to
// the property (TARGETS)
func readClipboardAtoms(display, window uintptr) []uintptr {
	var typ uintptr
	var actualFormat int32
	var nitems, bytesAfter uintptr
	var prop unsafe.Pointer
	status := xGetWindowProperty(display, window, clip.property, 0, 4096, xTrue, xAnyPropertyType,
		unsafe.Pointer(&typ), unsafe.Pointer(&actualFormat), unsafe.Pointer(&nitems), unsafe.Pointer(&bytesAfter), unsafe.Pointer(&prop))
	if status != xSuccess || prop == nil {
		return nil
	}
	defer xFree(prop)
	if actualFormat != 32 || nitems == 0 {
		return nil
	}
	// Xlib gives the 32-bit items as longs
	return slices.Clone(unsafe.Slice((*uintptr)(prop), int(nitems)))
}

// atomName returns the name of the atom
func atomName(display, atom uintptr) string {
	p := xGetAtomName(display, atom)
	if p == nil {
		return ""
	}
	defer xFree(unsafe.Pointer(p))
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(p, n))
}

// maxPropertySize is the most data written to a property at once: the
// largest request of the server. Larger data would need the INCR protocol.
func maxPropertySize(display uintptr) int {
	words := xExtendedMaxRequestSize(display)
	if words == 0 {
		words = xMaxRequestSize(display)
	}
	return int(words)*4 - 1024
}

func clipboardGetText() (string, error) {
	display, window, err := clipboardWindow()
	if err != nil {
		return "", err
	}
	owner := xGetSelectionOwner(display, clip.clipboard)
	if owner == xNone {
		return "", nil
	}
	if owner == window && clip.owned {
		return clip.text, nil
	}
	// UTF8_STRING is what everyone gives today; STRING (Latin-1) is for the old ones
	for _, target := range []uintptr{clip.utf8, xXAString} {
		text, ok, err := convertClipboard(display, window, target)
		if err != nil || ok {
			return text, err
		}
	}
	return "", nil
}

// convertClipboard asks the owner for the text as target; ok is false when
// it has no text of that kind
func convertClipboard(display, window, target uintptr) (text string, ok bool, err error) {
	xConvertSelection(display, clip.clipboard, target, clip.property, window, xCurrentTime)
	xFlush(display)
	var event xEvent
	if !waitClipboardEvent(display, window, xSelectionNotify, &event, nil) {
		return "", false, errClipboardTimeout
	}
	n := (*xSelectionEvent)(unsafe.Pointer(&event))
	if n.Property == xNone {
		return "", false, nil
	}
	data, typ := readClipboardProperty(display, window)
	if typ == clip.incr {
		data, err = readClipboardIncr(display, window)
	}
	if typ == xXAString {
		data = latin1ToUTF8(data)
	}
	return string(data), true, err
}

// readClipboardIncr gathers a large text: the owner puts it into the
// property part by part, each one deleted when read; an empty part ends it
func readClipboardIncr(display, window uintptr) ([]byte, error) {
	var result []byte
	newValue := func(e *xEvent) bool {
		p := (*xPropertyEvent)(unsafe.Pointer(e))
		return p.Atom == clip.property && p.State == xPropertyNewValue
	}
	for {
		var event xEvent
		if !waitClipboardEvent(display, window, xPropertyNotify, &event, newValue) {
			return result, errClipboardTimeout
		}
		part, typ := readClipboardProperty(display, window)
		// No property: an old event, e.g. of the INCR header written before;
		// the end is a part of zero length
		if typ == xNone {
			continue
		}
		if len(part) == 0 {
			return result, nil
		}
		result = append(result, part...)
	}
}

// readClipboardProperty reads and deletes the property the owner wrote to
func readClipboardProperty(display, window uintptr) (data []byte, typ uintptr) {
	var actualFormat int32
	var nitems, bytesAfter uintptr
	var prop unsafe.Pointer
	status := xGetWindowProperty(display, window, clip.property, 0, 1<<26, xTrue, xAnyPropertyType,
		unsafe.Pointer(&typ), unsafe.Pointer(&actualFormat), unsafe.Pointer(&nitems), unsafe.Pointer(&bytesAfter), unsafe.Pointer(&prop))
	if status != xSuccess || prop == nil {
		return nil, typ
	}
	defer xFree(prop)
	if actualFormat == 8 && nitems > 0 {
		data = append([]byte(nil), unsafe.Slice((*byte)(prop), int(nitems))...)
	}
	return data, typ
}

// waitClipboardEvent waits for an event of the type for the window, leaving
// the other events queued for the event loop. match, if set, skips the
// events it returns false for. False on a timeout.
func waitClipboardEvent(display, window uintptr, eventType int32, event *xEvent, match func(*xEvent) bool) bool {
	deadline := time.Now().Add(clipboardTimeout)
	for {
		for xCheckTypedWindowEvent(display, window, eventType, unsafe.Pointer(event)) != 0 {
			if match == nil || match(event) {
				return true
			}
		}
		left := time.Until(deadline)
		if left <= 0 {
			return false
		}
		waitForFds([]int{int(xConnectionNumber(display))}, min(left, 20*time.Millisecond))
	}
}

func latin1ToUTF8(b []byte) []byte {
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return []byte(string(r))
}

// processClipboardEvent handles an event of the clipboard window; false if
// the event is for another window. UI thread only.
func processClipboardEvent(event *xEvent) bool {
	if clip.window == 0 || event.window() != clip.window {
		return false
	}
	switch event.eventType() {
	case xSelectionClear:
		// Another application copied something: the text here is not the clipboard any more
		if e := (*xSelectionClearEvent)(unsafe.Pointer(event)); e.Selection == clip.clipboard {
			clip.owned = false
			clip.text, clip.hasText, clip.items = "", false, nil
		}
	case xSelectionRequest:
		answerSelectionRequest((*xSelectionRequestEvent)(unsafe.Pointer(event)))
	}
	return true
}

// answerSelectionRequest gives the data to an application that pastes it:
// the list of the formats it is offered as (TARGETS), or the data itself,
// written to the property of the requestor and announced with SelectionNotify
func answerSelectionRequest(r *xSelectionRequestEvent) {
	var reply xEvent
	n := (*xSelectionEvent)(unsafe.Pointer(&reply))
	n.Type = xSelectionNotify
	n.Requestor = r.Requestor
	n.Selection = r.Selection
	n.Target = r.Target
	n.Time = r.Time
	n.Property = xNone

	// An old requestor gives no property: the target is the property then
	property := r.Property
	if property == xNone {
		property = r.Target
	}
	if clip.owned && r.Selection == clip.clipboard {
		data, isItem := clip.items[r.Target]
		switch {
		case r.Target == clip.targets:
			atoms := clip.targetAtoms()
			xChangeProperty(xDisplay, r.Requestor, property, xXAAtom, 32, xPropModeReplace, unsafe.Pointer(&atoms[0]), int32(len(atoms)))
			n.Property = property
		case isItem:
			// Larger than a request: refused, the INCR protocol isn't done
			if len(data) <= maxPropertySize(xDisplay) {
				data = append(data[:len(data):len(data)], 0)
				xChangeProperty(xDisplay, r.Requestor, property, r.Target, 8, xPropModeReplace, unsafe.Pointer(&data[0]), int32(len(data)-1))
				n.Property = property
			}
		case clip.hasText && (r.Target == clip.utf8 || r.Target == clip.plain || r.Target == clip.textAtom || r.Target == xXAString):
			typ := r.Target
			if typ == clip.textAtom {
				typ = clip.utf8
			}
			data := []byte(clip.text)
			if typ == xXAString {
				data = utf8ToLatin1(clip.text)
			}
			// The data pointer must point somewhere even for an empty text
			data = append(data, 0)
			xChangeProperty(xDisplay, r.Requestor, property, typ, 8, xPropModeReplace, unsafe.Pointer(&data[0]), int32(len(data)-1))
			n.Property = property
		}
	}
	xSendEvent(xDisplay, r.Requestor, xFalse, 0, unsafe.Pointer(&reply))
	xFlush(xDisplay)
}

// utf8ToLatin1 is the text for STRING: the characters out of Latin-1 become '?'
func utf8ToLatin1(s string) []byte {
	b := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 0xFF {
			r = '?'
		}
		b = append(b, byte(r))
	}
	return b
}

// clipboardHandOver asks the clipboard manager to keep the text copied here,
// so it can still be pasted after the application quits. Called when the
// last window is closed. Without a manager the text goes with the application.
func clipboardHandOver() {
	if clip.window == 0 || !clip.owned || xDisplay == 0 {
		return
	}
	display, window := xDisplay, clip.window
	if xGetSelectionOwner(display, clip.manager) == xNone {
		return
	}
	// The formats to keep, as ICCCM asks: the manager then requests each of them
	atoms := clip.targetAtoms()[1:]
	if len(atoms) == 0 {
		return
	}
	xChangeProperty(display, window, clip.property, xXAAtom, 32, xPropModeReplace, unsafe.Pointer(&atoms[0]), int32(len(atoms)))
	xConvertSelection(display, clip.manager, clip.saveTargets, clip.property, window, xCurrentTime)
	xFlush(display)

	// The manager asks for the text meanwhile: the requests are answered
	// until it says it is done
	deadline := time.Now().Add(clipboardTimeout)
	for time.Now().Before(deadline) {
		var event xEvent
		for xCheckTypedWindowEvent(display, window, xSelectionRequest, unsafe.Pointer(&event)) != 0 {
			answerSelectionRequest((*xSelectionRequestEvent)(unsafe.Pointer(&event)))
		}
		if xCheckTypedWindowEvent(display, window, xSelectionNotify, unsafe.Pointer(&event)) != 0 {
			if (*xSelectionEvent)(unsafe.Pointer(&event)).Selection == clip.manager {
				return
			}
		}
		waitForFds([]int{int(xConnectionNumber(display))}, 20*time.Millisecond)
	}
}
