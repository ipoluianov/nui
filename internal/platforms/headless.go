package platforms

import (
	"image"
	"image/color"
	"image/draw"
	"sync"
	"sync/atomic"
)

// Headless mode: the windows are kept in memory, nothing is shown, and the
// program drives them itself - sends the mouse and the keyboard events,
// ticks the timers, runs the posted functions and paints the windows into
// images (see HeadlessWindow). It is how the forms are tested without a
// display, on any platform. The goroutine that drives the windows is the UI
// thread: Post queues the functions until RunHeadlessPosted.
//
// SetHeadless must be called before any window is created; there is no way
// back. The clipboard is the process's own, the file dialogs are cancelled,
// there are no printers and the tray icon is invisible.

var headless atomic.Bool

// SetHeadless switches the platform to the headless mode
func SetHeadless() {
	headless.Store(true)
}

// IsHeadless reports whether the platform is in the headless mode
func IsHeadless() bool {
	return headless.Load()
}

// The screen of the headless mode: its size and the pixels a logical pixel
// takes
var (
	headlessMu      sync.Mutex
	headlessScreenW = 1920
	headlessScreenH = 1080
	headlessScale   = 1.0

	headlessWindows []*HeadlessWindow // open, in creation order
	headlessPopups  []*headlessPopup  // shown, in showing order
	headlessActive  *HeadlessWindow

	headlessPosted    []func()
	headlessClipboard []ClipboardItem
)

// SetHeadlessScreen sets the size of the headless screen in logical pixels
// and its scale (2 for a Retina-like screen); for the windows created after
func SetHeadlessScreen(width, height int, scale float64) {
	headlessMu.Lock()
	defer headlessMu.Unlock()
	headlessScreenW, headlessScreenH = width, height
	headlessScale = max(scale, 1)
}

// RunHeadlessPosted runs the functions posted so far (see Post), and those
// they post, until none is left; it returns how many ran. A function that
// keeps posting itself stops it after a limit.
func RunHeadlessPosted() int {
	n := 0
	for range 1000 {
		headlessMu.Lock()
		queue := headlessPosted
		headlessPosted = nil
		headlessMu.Unlock()
		if len(queue) == 0 {
			break
		}
		for _, f := range queue {
			f()
		}
		n += len(queue)
	}
	return n
}

// TickHeadlessTimers calls the timer handler of every open window once, as
// the event loop does every few milliseconds
func TickHeadlessTimers() {
	for _, w := range HeadlessWindows() {
		if w.onTimer != nil {
			w.onTimer()
		}
	}
}

// HeadlessWindows returns the open headless windows, the oldest first
func HeadlessWindows() []*HeadlessWindow {
	headlessMu.Lock()
	defer headlessMu.Unlock()
	return append([]*HeadlessWindow(nil), headlessWindows...)
}

func postHeadless(f func()) {
	headlessMu.Lock()
	headlessPosted = append(headlessPosted, f)
	headlessMu.Unlock()
}

// HeadlessWindow is a window of the headless mode. Its methods besides the
// Window ones drive it as the system would: they call its handlers.
type HeadlessWindow struct {
	title         string
	x, y          int
	width, height int
	minW, minH    int
	scale         float64
	textInputRect [4]int
	visible       bool
	closed        bool
	maximized     bool
	modalParent   *HeadlessWindow
	modalChildren int
	mods          KeyModifiers
	cursor        MouseCursor

	// The window or popup that got the button press: it gets the moves and
	// the release (the system's mouse grab); and the one under the mouse
	grab  any
	hover any

	onKeyDown         func(keyCode Key, mods KeyModifiers) bool
	onKeyUp           func(keyCode Key, mods KeyModifiers)
	onChar            func(char rune)
	onMouseEnter      func()
	onMouseLeave      func()
	onMouseMove       func(x, y int)
	onMouseButtonDown func(btn MouseButton, x int, y int)
	onMouseButtonUp   func(btn MouseButton, x int, y int)
	onMouseDblClick   func(btn MouseButton, x int, y int)
	onMouseWheel      func(deltaX int, deltaY int)
	onCreated         func()
	onPaint           func(rgba *image.RGBA)
	onMove            func(x, y int)
	onResize          func(width, height int)
	onCloseRequest    func() bool
	onTimer           func()
	onDeactivate      func()
	onActivate        func()
	onFilesDropped    func(files []string, x, y int)
}

func newHeadlessWindow(title string, posX, posY, width, height int, center, maximized bool) *HeadlessWindow {
	headlessMu.Lock()
	defer headlessMu.Unlock()
	w := &HeadlessWindow{title: title, x: posX, y: posY, width: width, height: height, scale: headlessScale}
	if center || posX < 0 || posY < 0 {
		w.x, w.y = (headlessScreenW-width)/2, (headlessScreenH-height)/2
	}
	if maximized {
		w.maximized = true
		w.x, w.y, w.width, w.height = 0, 0, headlessScreenW, headlessScreenH
	}
	headlessWindows = append(headlessWindows, w)
	return w
}

// Title returns the title of the window
func (w *HeadlessWindow) Title() string { return w.title }

// Visible reports whether the window is shown
func (w *HeadlessWindow) Visible() bool { return w.visible && !w.closed }

// Closed reports whether the window is closed
func (w *HeadlessWindow) Closed() bool { return w.closed }

// Modal reports whether the window is shown modally (see ShowModal)
func (w *HeadlessWindow) Modal() bool { return w.modalParent != nil }

// Blocked reports whether a modal window shown over this one takes its input
func (w *HeadlessWindow) Blocked() bool { return w.modalChildren > 0 }

// Cursor returns the mouse cursor set last
func (w *HeadlessWindow) Cursor() MouseCursor { return w.cursor }

// Render paints the window, with its popups on top, as the screen shows it:
// Scale pixels a logical pixel
func (w *HeadlessWindow) Render() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w.px(w.width), w.px(w.height)))
	if w.onPaint != nil {
		w.onPaint(img)
	}
	for _, p := range w.popups() {
		pimg := image.NewRGBA(image.Rect(0, 0, w.px(p.width), w.px(p.height)))
		if p.onPaint != nil {
			p.onPaint(pimg)
		}
		at := image.Pt(w.px(p.x-w.x), w.px(p.y-w.y))
		draw.Draw(img, pimg.Bounds().Add(at), pimg, image.Point{}, draw.Over)
	}
	return img
}

func (w *HeadlessWindow) px(v int) int {
	return int(float64(v)*w.scale + 0.5)
}

// popups returns the shown popups of the window, the topmost last
func (w *HeadlessWindow) popups() []*headlessPopup {
	headlessMu.Lock()
	defer headlessMu.Unlock()
	var list []*headlessPopup
	for _, p := range headlessPopups {
		if p.owner == w && p.visible {
			list = append(list, p)
		}
	}
	return list
}

// target returns the window or the popup at the client point, and the point
// in its coordinates
func (w *HeadlessWindow) target(x, y int) (any, int, int) {
	sx, sy := x+w.x, y+w.y
	popups := w.popups()
	for i := len(popups) - 1; i >= 0; i-- {
		p := popups[i]
		if sx >= p.x && sx < p.x+p.width && sy >= p.y && sy < p.y+p.height {
			return p, sx - p.x, sy - p.y
		}
	}
	return w, x, y
}

// local converts the client point to the coordinates of the target
func (w *HeadlessWindow) local(t any, x, y int) (int, int) {
	if p, ok := t.(*headlessPopup); ok {
		return x + w.x - p.x, y + w.y - p.y
	}
	return x, y
}

// SetModifiers sets the keys held, as KeyModifiers reports them
func (w *HeadlessWindow) SetModifiers(mods KeyModifiers) { w.mods = mods }

// MouseMove moves the mouse to the client point: the window or the popup
// under it gets the move, or the one that got the button press while the
// button is held
func (w *HeadlessWindow) MouseMove(x, y int) {
	if w.Blocked() {
		return
	}
	t, tx, ty := w.target(x, y)
	if w.grab != nil {
		t = w.grab
		tx, ty = w.local(t, x, y)
	}
	w.setHover(t)
	switch t := t.(type) {
	case *headlessPopup:
		if t.onMouseMove != nil {
			t.onMouseMove(tx, ty)
		}
	case *HeadlessWindow:
		if t.onMouseMove != nil {
			t.onMouseMove(tx, ty)
		}
	}
}

func (w *HeadlessWindow) setHover(t any) {
	if w.hover == t {
		return
	}
	switch old := w.hover.(type) {
	case *headlessPopup:
		if old.onMouseLeave != nil {
			old.onMouseLeave()
		}
	case *HeadlessWindow:
		if old.onMouseLeave != nil {
			old.onMouseLeave()
		}
	}
	w.hover = t
	if t == any(w) && w.onMouseEnter != nil {
		w.onMouseEnter()
	}
}

// MouseDown presses the button at the client point
func (w *HeadlessWindow) MouseDown(btn MouseButton, x, y int) {
	if w.Blocked() {
		return
	}
	w.MouseMove(x, y)
	t, tx, ty := w.target(x, y)
	w.grab = t
	switch t := t.(type) {
	case *headlessPopup:
		if t.onMouseButtonDown != nil {
			t.onMouseButtonDown(btn, tx, ty)
		}
	case *HeadlessWindow:
		if t.onMouseButtonDown != nil {
			t.onMouseButtonDown(btn, tx, ty)
		}
	}
}

// MouseUp releases the button at the client point: the window or the popup
// that got the press gets it
func (w *HeadlessWindow) MouseUp(btn MouseButton, x, y int) {
	if w.Blocked() {
		return
	}
	t := w.grab
	w.grab = nil
	if t == nil {
		t, _, _ = w.target(x, y)
	}
	tx, ty := w.local(t, x, y)
	switch t := t.(type) {
	case *headlessPopup:
		if t.visible && t.onMouseButtonUp != nil {
			t.onMouseButtonUp(btn, tx, ty)
		}
	case *HeadlessWindow:
		if t.onMouseButtonUp != nil {
			t.onMouseButtonUp(btn, tx, ty)
		}
	}
}

// MouseDblClick sends the double click the system sends after the second
// press of a double click (the press itself comes before, with MouseDown)
func (w *HeadlessWindow) MouseDblClick(btn MouseButton, x, y int) {
	if w.Blocked() {
		return
	}
	if t, tx, ty := w.target(x, y); t == any(w) && w.onMouseDblClick != nil {
		w.onMouseDblClick(btn, tx, ty)
	}
}

// MouseWheel turns the wheel over the window or the popup under the mouse
func (w *HeadlessWindow) MouseWheel(deltaX, deltaY int) {
	if w.Blocked() {
		return
	}
	switch t := w.hover.(type) {
	case *headlessPopup:
		if t.onMouseWheel != nil {
			t.onMouseWheel(deltaX, deltaY)
		}
	default:
		if w.onMouseWheel != nil {
			w.onMouseWheel(deltaX, deltaY)
		}
	}
}

// MouseLeave moves the mouse out of the window
func (w *HeadlessWindow) MouseLeave() {
	w.setHover(nil)
}

// KeyDown presses the key with the modifiers held; it returns what the
// handler returned (whether the key was taken)
func (w *HeadlessWindow) KeyDown(key Key, mods KeyModifiers) bool {
	if w.Blocked() {
		return false
	}
	w.mods = mods
	if w.onKeyDown != nil {
		return w.onKeyDown(key, mods)
	}
	return false
}

// KeyUp releases the key
func (w *HeadlessWindow) KeyUp(key Key, mods KeyModifiers) {
	if w.Blocked() {
		return
	}
	w.mods = mods
	if w.onKeyUp != nil {
		w.onKeyUp(key, mods)
	}
}

// Char types the character, as the system sends it after a key press
func (w *HeadlessWindow) Char(ch rune) {
	if w.Blocked() {
		return
	}
	if w.onChar != nil {
		w.onChar(ch)
	}
}

// DropFiles drops the files at the client point
func (w *HeadlessWindow) DropFiles(files []string, x, y int) {
	if w.onFilesDropped != nil {
		w.onFilesDropped(files, x, y)
	}
}

// RequestClose clicks the close button of the window: it closes if the
// handler lets it; it returns whether it did
func (w *HeadlessWindow) RequestClose() bool {
	if w.onCloseRequest != nil && !w.onCloseRequest() {
		return false
	}
	return w.Close()
}

// UserResize resizes the window as the user would, by its frame
func (w *HeadlessWindow) UserResize(width, height int) {
	w.Resize(width, height)
}

// Window

func (w *HeadlessWindow) Show() {
	w.visible = true
	w.activate()
}

func (w *HeadlessWindow) Hide() {
	w.visible = false
}

func (w *HeadlessWindow) ShowModal(parent Window) {
	if p, ok := parent.(*HeadlessWindow); ok {
		w.modalParent = p
		p.modalChildren++
	}
	w.Show()
}

// activate makes the window the active one, as the system does when it is
// shown: the active window before is deactivated
func (w *HeadlessWindow) activate() {
	headlessMu.Lock()
	prev := headlessActive
	headlessActive = w
	headlessMu.Unlock()
	if prev == w {
		return
	}
	if prev != nil && prev.onDeactivate != nil {
		prev.onDeactivate()
	}
	if w.onActivate != nil {
		w.onActivate()
	}
}

func (w *HeadlessWindow) Update() {}

func (w *HeadlessWindow) Close() bool {
	if w.closed {
		return true
	}
	w.closed, w.visible = true, false
	headlessMu.Lock()
	for i, o := range headlessWindows {
		if o == w {
			headlessWindows = append(headlessWindows[:i], headlessWindows[i+1:]...)
			break
		}
	}
	wasActive := headlessActive == w
	if wasActive {
		headlessActive = nil
	}
	headlessMu.Unlock()
	if p := w.modalParent; p != nil {
		p.modalChildren--
		w.modalParent = nil
		if wasActive {
			p.activate()
		}
	}
	return true
}

// Exec returns at once: the program drives the headless windows itself
func (w *HeadlessWindow) Exec() {}

func (w *HeadlessWindow) SystemHandle() any { return w }

func (w *HeadlessWindow) OnKeyDown(f func(keyCode Key, mods KeyModifiers) bool) { w.onKeyDown = f }
func (w *HeadlessWindow) OnKeyUp(f func(keyCode Key, mods KeyModifiers))        { w.onKeyUp = f }
func (w *HeadlessWindow) OnChar(f func(char rune))                              { w.onChar = f }
func (w *HeadlessWindow) OnMouseEnter(f func())                                 { w.onMouseEnter = f }
func (w *HeadlessWindow) OnMouseLeave(f func())                                 { w.onMouseLeave = f }
func (w *HeadlessWindow) OnMouseMove(f func(x, y int))                          { w.onMouseMove = f }
func (w *HeadlessWindow) OnMouseButtonDown(f func(btn MouseButton, x int, y int)) {
	w.onMouseButtonDown = f
}
func (w *HeadlessWindow) OnMouseButtonUp(f func(btn MouseButton, x int, y int)) {
	w.onMouseButtonUp = f
}
func (w *HeadlessWindow) OnMouseButtonDblClick(f func(btn MouseButton, x int, y int)) {
	w.onMouseDblClick = f
}
func (w *HeadlessWindow) OnMouseWheel(f func(deltaX int, deltaY int)) { w.onMouseWheel = f }
func (w *HeadlessWindow) OnCreated(f func())                          { w.onCreated = f }
func (w *HeadlessWindow) OnPaint(f func(rgba *image.RGBA))            { w.onPaint = f }
func (w *HeadlessWindow) OnMove(f func(x, y int))                     { w.onMove = f }
func (w *HeadlessWindow) OnResize(f func(width, height int))          { w.onResize = f }
func (w *HeadlessWindow) OnCloseRequest(f func() bool)                { w.onCloseRequest = f }
func (w *HeadlessWindow) OnTimer(f func())                            { w.onTimer = f }
func (w *HeadlessWindow) OnDeactivate(f func())                       { w.onDeactivate = f }
func (w *HeadlessWindow) OnActivate(f func())                         { w.onActivate = f }
func (w *HeadlessWindow) OnFilesDropped(f func(files []string, x, y int)) {
	w.onFilesDropped = f
}

func (w *HeadlessWindow) StartFileDrag(files []string, done func(DragEffect, bool)) bool {
	return false
}

func (w *HeadlessWindow) SetTitle(title string)             { w.title = title }
func (w *HeadlessWindow) SetAppIcon(icon *image.RGBA)       {}
func (w *HeadlessWindow) SetBackgroundColor(col color.RGBA) {}
func (w *HeadlessWindow) SetMouseCursor(cursor MouseCursor) { w.cursor = cursor }
func (w *HeadlessWindow) SetDarkMode(dark bool)             {}
func (w *HeadlessWindow) MoveToCenterOfScreen() {
	w.Move((headlessScreenW-w.width)/2, (headlessScreenH-w.height)/2)
}
func (w *HeadlessWindow) SetMinSize(width int, height int) { w.minW, w.minH = width, height }
func (w *HeadlessWindow) MinimizeWindow()                  {}
func (w *HeadlessWindow) MaximizeWindow() {
	w.maximized = true
	w.Move(0, 0)
	w.Resize(headlessScreenW, headlessScreenH)
}
func (w *HeadlessWindow) RestoreWindow()              { w.maximized = false }
func (w *HeadlessWindow) IsMaximized() bool           { return w.maximized }
func (w *HeadlessWindow) SetAlwaysOnTop(onTop bool)   {}
func (w *HeadlessWindow) RequestAttention()           {}
func (w *HeadlessWindow) Beep()                       {}
func (w *HeadlessWindow) SetAllowMinimize(allow bool) {}
func (w *HeadlessWindow) SetAllowMaximize(allow bool) {}
func (w *HeadlessWindow) Size() (int, int)            { return w.width, w.height }
func (w *HeadlessWindow) Pos() (int, int)             { return w.x, w.y }
func (w *HeadlessWindow) PosX() int                   { return w.x }
func (w *HeadlessWindow) PosY() int                   { return w.y }
func (w *HeadlessWindow) Width() int                  { return w.width }
func (w *HeadlessWindow) Height() int                 { return w.height }
func (w *HeadlessWindow) KeyModifiers() KeyModifiers  { return w.mods }
func (w *HeadlessWindow) DrawTimeUs() int64           { return 0 }
func (w *HeadlessWindow) Scale() float64              { return w.scale }

// SetTextInputRect keeps the rect, see TextInputRect
func (w *HeadlessWindow) SetTextInputRect(x, y, width, height int) {
	w.textInputRect = [4]int{x, y, width, height}
}

// TextInputRect returns what the form gave to SetTextInputRect last
func (w *HeadlessWindow) TextInputRect() (x, y, width, height int) {
	r := w.textInputRect
	return r[0], r[1], r[2], r[3]
}
func (w *HeadlessWindow) ClientToScreen(x, y int) (int, int) { return x + w.x, y + w.y }

func (w *HeadlessWindow) ScreenWorkArea(x, y int) (int, int, int, int) {
	return 0, 0, headlessScreenW, headlessScreenH
}

func (w *HeadlessWindow) Move(x, y int) {
	if x == w.x && y == w.y {
		return
	}
	w.x, w.y = x, y
	if w.onMove != nil {
		w.onMove(x, y)
	}
}

// Resize resizes the client area, not below the minimum size, and tells the
// window as the system does
func (w *HeadlessWindow) Resize(width, height int) {
	width, height = max(width, w.minW), max(height, w.minH)
	if width == w.width && height == w.height {
		return
	}
	w.width, w.height = width, height
	if w.onResize != nil {
		w.onResize(width, height)
	}
}

// headlessPopup is a popup window of the headless mode, at its place on the
// headless screen
type headlessPopup struct {
	popupCallbacks
	owner         *HeadlessWindow
	visible       bool
	x, y          int
	width, height int
}

func newHeadlessPopup(owner Window) PopupWindow {
	w, _ := owner.(*HeadlessWindow)
	p := &headlessPopup{owner: w}
	if w != nil {
		p.scale = w.scale
	}
	return p
}

func (p *headlessPopup) ShowAt(x, y, width, height int) {
	p.x, p.y, p.width, p.height = x, y, width, height
	headlessMu.Lock()
	defer headlessMu.Unlock()
	if !p.visible {
		p.visible = true
		headlessPopups = append(headlessPopups, p)
	}
}

func (p *headlessPopup) Hide() {
	headlessMu.Lock()
	defer headlessMu.Unlock()
	p.visible = false
	for i, o := range headlessPopups {
		if o == p {
			headlessPopups = append(headlessPopups[:i], headlessPopups[i+1:]...)
			break
		}
	}
}

func (p *headlessPopup) SetMouseCursor(cursor MouseCursor) {}
func (p *headlessPopup) Update()                           {}
func (p *headlessPopup) Close()                            { p.Hide() }

// headlessTray is the invisible tray icon of the headless mode
type headlessTray struct{}

func (headlessTray) SetIcon(icon *image.RGBA)                  {}
func (headlessTray) SetTooltip(text string)                    {}
func (headlessTray) SetMenu(items []TrayMenuItem)              {}
func (headlessTray) OnClick(f func())                          {}
func (headlessTray) ShowNotification(title, text string) error { return nil }
func (headlessTray) Close()                                    {}
