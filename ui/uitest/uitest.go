// Package uitest tests nui forms without a display: importing it switches
// nui to the headless mode, where the windows are kept in memory. A test
// shows its form, acts on it as the user would - clicks, types, presses
// shortcuts - and checks the widgets, or compares the screenshot with a
// golden image:
//
//	func TestLogin(t *testing.T) {
//		form := NewLoginForm()
//		w := uitest.Show(t, form)
//		w.Click(w.Find("user"))
//		w.Type("admin")
//		w.Shortcut("Ctrl+Enter")
//		w.MatchGolden("login-error")
//	}
//
// Everything runs on the test's goroutine, which is the UI thread: the
// functions posted with Invoke run when the test acts or calls Settle.
// Timers fire only on Tick, so the screenshots don't depend on the time
// (e.g. on the blinking of the text cursor).
//
// For the same pixels everywhere the fonts are drawn by nui's own rasterizer
// (NativeFontRendering is off) and the theme is the light one.
package uitest

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/ipoluianov/nui/internal/platforms"
	"github.com/ipoluianov/nui/ui"
)

func init() {
	platforms.SetHeadless()
	ui.NativeFontRendering = false
	ui.ApplyLightTheme()
}

// Window is a shown form under test
type Window struct {
	t    testing.TB
	Form *ui.Form
	wnd  *platforms.HeadlessWindow

	mouseX, mouseY int

	// NoFrames turns off the painting of a frame before each action (see
	// frame): for benchmarks of the input alone
	NoFrames bool
}

// Show shows the form with its size and returns it for the test; the form
// is closed when the test ends
func Show(t testing.TB, form *ui.Form) *Window {
	t.Helper()
	form.Show()
	w := wrap(t, form)
	t.Cleanup(func() {
		if !w.wnd.Closed() {
			form.Close()
		}
	})
	w.Settle()
	w.frame()
	return w
}

// frame paints the window, as the event loop does between the events: some
// widgets work out where their parts are when they paint (the tabs of a tab
// widget), and the user sees the window before acting on it
func (w *Window) frame() {
	w.Settle()
	if !w.NoFrames {
		w.Form.Size()
		w.wnd.Render()
	}
}

func wrap(t testing.TB, form *ui.Form) *Window {
	t.Helper()
	wnd, ok := form.SystemHandle().(*platforms.HeadlessWindow)
	if !ok {
		t.Fatalf("uitest: the form has no headless window (shown before uitest was imported?)")
	}
	return &Window{t: t, Form: form, wnd: wnd}
}

// SetScreen sets the size of the screen the windows are on, in logical
// pixels, and its scale: 2 draws the screenshots as on a Retina screen.
// For the forms shown after it.
func SetScreen(width, height int, scale float64) {
	platforms.SetHeadlessScreen(width, height, scale)
}

// Settle runs the functions posted to the UI thread (Invoke, Form.Invoke)
// and those they post, until none is left
func (w *Window) Settle() {
	platforms.RunHeadlessPosted()
}

// RunPosted runs the functions posted to the UI thread, as Settle, and
// returns how many ran
func RunPosted() int {
	return platforms.RunHeadlessPosted()
}

// Tick fires the timers of the open forms once, as the event loop does
// every few milliseconds, and settles. A widget's timer runs only if its
// interval has passed: see Wait.
func (w *Window) Tick() {
	platforms.TickHeadlessTimers()
	w.Settle()
}

// Wait lets the time pass with the timers ticking, e.g. for an animation,
// a tooltip or a debounce to happen
func (w *Window) Wait(d time.Duration) {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		time.Sleep(5 * time.Millisecond)
		w.Tick()
	}
	w.Tick()
}

// Resize resizes the window as the user would, by its frame
func (w *Window) Resize(width, height int) {
	w.wnd.UserResize(width, height)
	w.Settle()
}

// Size returns the size of the window's client area
func (w *Window) Size() (int, int) {
	return w.wnd.Size()
}

// Title returns the title of the window
func (w *Window) Title() string {
	return w.wnd.Title()
}

// Closed reports whether the window is closed
func (w *Window) Closed() bool {
	return w.wnd.Closed()
}

// Blocked reports whether a modal window shown over this one takes its input
func (w *Window) Blocked() bool {
	return w.wnd.Blocked()
}

// Cursor returns the mouse cursor the window shows now
func (w *Window) Cursor() ui.MouseCursor {
	return w.wnd.Cursor()
}

// Accessible returns the accessible objects of the window, as a screen
// reader sees them (see ui.Form.Accessible)
func (w *Window) Accessible() *ui.AccessibleNode {
	w.frame()
	return w.Form.Accessible()
}

// TextInputRect returns where the window told the input method the caret
// is, in the client area
func (w *Window) TextInputRect() (x, y, width, height int) {
	return w.wnd.TextInputRect()
}

// TextInputActive tells whether the window was told the focus is in a text
// box: the input method of macOS takes the keys only then
func (w *Window) TextInputActive() bool {
	return w.wnd.TextInputActive()
}

// RequestClose clicks the close button of the window; it returns whether
// the window closed (OnClose may keep it open)
func (w *Window) RequestClose() bool {
	closed := w.wnd.RequestClose()
	w.Settle()
	return closed
}

// Screenshot paints the window, with its open popups on top
func (w *Window) Screenshot() *image.RGBA {
	w.Settle()
	w.Form.Size() // the window grows to its widgets before it is painted
	return w.wnd.Render()
}

// Top returns the window shown last that is still open: a dialog or a
// message box the form opened, or the form itself
func (w *Window) Top() *Window {
	w.t.Helper()
	windows := platforms.HeadlessWindows()
	for i := len(windows) - 1; i >= 0; i-- {
		for _, form := range ui.OpenForms() {
			if form.SystemHandle() == any(windows[i]) {
				return wrap(w.t, form)
			}
		}
	}
	w.t.Fatal("uitest: no open window")
	return nil
}

// Dialog returns the modal window shown over this one, failing the test if
// there is none
func (w *Window) Dialog() *Window {
	w.t.Helper()
	top := w.Top()
	if top.Form == w.Form || !top.wnd.Modal() {
		w.t.Fatal("uitest: no dialog over the window")
	}
	return top
}

// Mouse

// Point is a point in the window's client coordinates
type Point struct{ X, Y int }

// Center returns the center of the widget in the window's client
// coordinates, where a click on it goes
func Center(widget ui.Widgeter) Point {
	x, y := widget.RectClientAreaOnWindow()
	return Point{x + widget.Width()/2, y + widget.Height()/2}
}

// At returns the point (dx, dy) from the top-left corner of the widget
func At(widget ui.Widgeter, dx, dy int) Point {
	x, y := widget.RectClientAreaOnWindow()
	return Point{x + dx, y + dy}
}

// target is where an action on a widget or a point goes
func (w *Window) target(on any) Point {
	w.t.Helper()
	switch on := on.(type) {
	case Point:
		return on
	case ui.Widgeter:
		if on == nil {
			w.t.Fatal("uitest: nil widget")
		}
		if !on.IsVisible() || on.Width() <= 0 || on.Height() <= 0 {
			w.t.Fatalf("uitest: %s %q is not visible", on.TypeName(), on.Name())
		}
		return Center(on)
	}
	w.t.Fatalf("uitest: want a widget or a Point, got %T", on)
	return Point{}
}

// Move moves the mouse to the widget's center or to the point
func (w *Window) Move(to any) {
	p := w.target(to)
	w.frame()
	w.mouseX, w.mouseY = p.X, p.Y
	w.wnd.MouseMove(p.X, p.Y)
	w.Settle()
}

// Press presses the mouse button over the widget's center or the point
func (w *Window) Press(button ui.MouseButton, at any) {
	p := w.target(at)
	w.frame()
	w.mouseX, w.mouseY = p.X, p.Y
	w.wnd.MouseDown(button, p.X, p.Y)
	w.Settle()
}

// Release releases the mouse button over the widget's center or the point
func (w *Window) Release(button ui.MouseButton, at any) {
	p := w.target(at)
	w.frame()
	w.mouseX, w.mouseY = p.X, p.Y
	w.wnd.MouseUp(button, p.X, p.Y)
	w.Settle()
}

// Click clicks the left button on the widget's center or on the point
func (w *Window) Click(on any) {
	w.ClickButton(ui.MouseButtonLeft, on)
}

// RightClick clicks the right button, e.g. for a context menu
func (w *Window) RightClick(on any) {
	w.ClickButton(ui.MouseButtonRight, on)
}

// ClickButton clicks the mouse button on the widget's center or the point
func (w *Window) ClickButton(button ui.MouseButton, on any) {
	w.t.Helper()
	p := w.target(on)
	w.Move(p)
	w.Press(button, p)
	w.Release(button, p)
}

// DoubleClick double-clicks the left button, as the system reports it: a
// click, the second press with the double click event, the release
func (w *Window) DoubleClick(on any) {
	w.t.Helper()
	p := w.target(on)
	w.Click(p)
	w.wnd.MouseDown(ui.MouseButtonLeft, p.X, p.Y)
	w.wnd.MouseDblClick(ui.MouseButtonLeft, p.X, p.Y)
	w.wnd.MouseUp(ui.MouseButtonLeft, p.X, p.Y)
	w.Settle()
}

// Drag presses the left button at from, moves the mouse to to in steps,
// and releases it there
func (w *Window) Drag(from, to any) {
	w.t.Helper()
	a, b := w.target(from), w.target(to)
	w.Move(a)
	w.Press(ui.MouseButtonLeft, a)
	const steps = 8
	for i := 1; i <= steps; i++ {
		w.Move(Point{a.X + (b.X-a.X)*i/steps, a.Y + (b.Y-a.Y)*i/steps})
	}
	w.Release(ui.MouseButtonLeft, b)
}

// Wheel turns the mouse wheel over the widget's center or the point;
// a positive dy scrolls up, as the systems report it
func (w *Window) Wheel(at any, dx, dy int) {
	w.Move(at)
	w.wnd.MouseWheel(dx, dy)
	w.Settle()
}

// DropFiles drops the files on the widget's center or the point
func (w *Window) DropFiles(at any, files ...string) {
	p := w.target(at)
	w.wnd.DropFiles(files, p.X, p.Y)
	w.Settle()
}

// Keyboard

// Key presses and releases the key with the modifiers
func (w *Window) Key(key ui.Key, mods ui.KeyModifiers) {
	w.frame()
	w.wnd.KeyDown(key, mods)
	w.wnd.KeyUp(key, mods)
	w.Settle()
}

// Shortcut presses the keys of the shortcut, e.g. "Ctrl+S", "Shift+F3",
// "Mod+Z" (Cmd on macOS, Ctrl elsewhere): see ui.ParseShortcut
func (w *Window) Shortcut(shortcut string) {
	w.t.Helper()
	s, err := ui.ParseShortcut(shortcut)
	if err != nil {
		w.t.Fatalf("uitest: %v", err)
	}
	w.Key(s.Key, ui.KeyModifiers{Ctrl: s.Ctrl, Shift: s.Shift, Alt: s.Alt, Cmd: s.Cmd})
}

// Type types the text as the keyboard would: a character at a time, Enter
// for "\n" and Tab for "\t"
func (w *Window) Type(text string) {
	w.frame()
	for _, ch := range text {
		switch ch {
		case '\n':
			w.wnd.KeyDown(ui.KeyEnter, ui.KeyModifiers{})
			w.wnd.KeyUp(ui.KeyEnter, ui.KeyModifiers{})
		case '\t':
			w.wnd.KeyDown(ui.KeyTab, ui.KeyModifiers{})
			w.wnd.KeyUp(ui.KeyTab, ui.KeyModifiers{})
		default:
			// As the system does: the key press comes before the character
			// (a text field takes the character, not the press)
			key := keyOf(ch)
			mods := ui.KeyModifiers{Shift: ch >= 'A' && ch <= 'Z'}
			w.wnd.KeyDown(key, mods)
			w.wnd.Char(ch)
			w.wnd.KeyUp(key, mods)
		}
		w.Settle()
	}
}

// keyOf returns the key of a Latin letter or a digit, 0 for the others
func keyOf(ch rune) ui.Key {
	upper := strings.ToUpper(string(ch))
	if s, err := ui.ParseShortcut(upper); err == nil && len(upper) == 1 {
		return s.Key
	}
	return 0
}

// Finding widgets

// Find returns the widget of the name (see Widget.SetName), or the one that
// shows the text (a button, a label, a menu item...): in the form, its menu
// bar and its open popups. It fails the test if there is none.
func (w *Window) Find(nameOrText string) ui.Widgeter {
	w.t.Helper()
	if found := w.FindAll(func(widget ui.Widgeter) bool { return widget.Name() == nameOrText }); len(found) > 0 {
		return found[0]
	}
	found := w.FindAll(func(widget ui.Widgeter) bool {
		t, ok := widget.(interface{ Text() string })
		return ok && cleanText(t.Text()) == cleanText(nameOrText)
	})
	if len(found) == 0 {
		w.t.Fatalf("uitest: no widget named or showing %q", nameOrText)
	}
	return found[0]
}

// cleanText drops the mnemonic marks ("&Open" shows "Open")
func cleanText(s string) string {
	s = strings.ReplaceAll(s, "&&", "\x00")
	s = strings.ReplaceAll(s, "&", "")
	return strings.ReplaceAll(s, "\x00", "&")
}

// FindAll returns the widgets match accepts, the open popups' first (they
// are on top), then the menu bar's, then the form's, each in tree order
func (w *Window) FindAll(match func(ui.Widgeter) bool) []ui.Widgeter {
	var found []ui.Widgeter
	visit := func(root ui.Widgeter) {
		if match(root) {
			found = append(found, root)
		}
		for _, widget := range root.AllChildren() {
			if match(widget) {
				found = append(found, widget)
			}
		}
	}
	popups := w.Form.Panel().PopupWidgets
	for i := len(popups) - 1; i >= 0; i-- {
		visit(popups[i])
	}
	if bar := w.Form.MenuBar(); bar != nil {
		visit(bar)
	}
	visit(w.Form.Panel())
	return found
}

// Popup returns the topmost open popup (a menu, a drop-down list), nil if
// none. Its items may be painted rather than widgets of their own: click
// them with At(popup, x, y), or choose them with the keys.
func (w *Window) Popup() ui.Widgeter {
	popups := w.Form.Panel().PopupWidgets
	if len(popups) == 0 {
		return nil
	}
	return popups[len(popups)-1]
}

// Focused returns the widget with the keyboard focus, nil if none
func (w *Window) Focused() ui.Widgeter {
	return w.Form.FocusedWidget()
}

// String describes the window for the messages of a test
func (w *Window) String() string {
	width, height := w.Size()
	return fmt.Sprintf("window %q %dx%d", w.Title(), width, height)
}
