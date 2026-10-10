package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ipoluianov/nui/internal/platforms"
)

type Form struct {
	wnd   platforms.Window
	title string
	icon  *image.RGBA

	posX int
	posY int

	width  int
	height int

	lastMouseX      int
	lastMouseY      int
	lastMouseCursor MouseCursor

	lastKeyboardModifiers KeyModifiers

	mouseLeftButtonPressed       bool
	mouseLeftButtonPressedWidget Widgeter

	topWidget     *Panel
	hoverWidget   Widgeter
	focusedWidget Widgeter

	// menuBar is the main menu above topWidget, nil if none (see SetMenuBar)
	menuBar *MenuBar

	tooltip tooltipState

	// hidden is set by Hide
	hidden bool

	// See AddShortcut
	shortcuts []formShortcut

	// The drag in progress, see SetDragSource
	drag           dragState
	onFilesDropped func(files []string, x, y int)

	// The toasts shown now, oldest first, see ShowToast
	toasts                 []*toast
	toastPopupsUnavailable bool

	// Native windows of the open popup widgets, see form_popup.go
	popupHosts              []*popupHost
	freePopupWindows        []platforms.PopupWindow
	popupWindowsUnavailable bool
	// popupUnderMouse is the popup window the last mouse event came from,
	// nil when it came from the form's own window
	popupUnderMouse *popupHost
	// mouseDownPopup is the popup window where a mouse button was pressed,
	// so the release goes there too
	mouseDownPopup *popupHost

	onGlobalKeyDown func(keyCode Key, mods KeyModifiers) bool

	// active: the window has the keyboard focus; the callbacks are called
	// when it changes
	active        bool
	onActivated   func()
	onDeactivated func()

	needUpdate         bool
	lastFreeMemoryTime time.Time

	lastUpdateTime time.Time

	updateBlockStack    int
	layoutingBlockStack int

	allowMinimize bool
	allowMaximize bool

	// parentForm is the form this one was shown modally over (see ShowModal),
	// kept around so MoveToCenterOfParent can re-center later - e.g. after
	// OnDialogShow resizes the dialog to fit its actual content.
	parentForm *Form

	acceptButton *Button // The button that is triggered when the user accepts the form (e.g., presses Enter)
	cancelButton *Button // The button that is triggered when the user cancels the form (e.g., presses Esc)

	OnClose func() bool

	// See SetTitleFunc and SetOnLanguageChanged
	titleFunc         func() string
	onLanguageChanged func()
}

var nextWidgetId int64

// allwidgets holds the widgets of all the forms by ID. The forms all live on
// the UI thread, but widgets may be created on other goroutines too (e.g. a
// form built before it is shown), so the map is locked.
var (
	allwidgetsMtx sync.RWMutex
	allwidgets    map[string]Widgeter
)

func init() {
	nextWidgetId = 0
	allwidgets = make(map[string]Widgeter)
}

func registerWidget(w Widgeter) {
	allwidgetsMtx.Lock()
	allwidgets[w.Id()] = w
	allwidgetsMtx.Unlock()
}

func unregisterWidget(id string) {
	allwidgetsMtx.Lock()
	delete(allwidgets, id)
	allwidgetsMtx.Unlock()
}

func lookupWidget(id string) (Widgeter, bool) {
	allwidgetsMtx.RLock()
	defer allwidgetsMtx.RUnlock()
	w, ok := allwidgets[id]
	return w, ok
}

func PrintAllWidgets() {
	ws := make([]Widgeter, 0)
	allwidgetsMtx.RLock()
	for _, w := range allwidgets {
		ws = append(ws, w)
	}
	allwidgetsMtx.RUnlock()
	sort.Slice(ws, func(i, j int) bool {
		return ws[i].Id() < ws[j].Id()
	})
	fmt.Println("")
	fmt.Println("WIDGETS:")
	for _, w := range ws {
		fmt.Println(w.Id(), w.TypeName())
	}
	fmt.Println("")
}

func (c *Form) SystemHandle() any {
	if c.wnd != nil {
		return c.wnd.SystemHandle()
	}
	return nil
}

func (c *Form) Maximize() {
	if c.wnd != nil {
		c.wnd.MaximizeWindow()
	}
}

// Restore returns a maximized window to its normal size
func (c *Form) Restore() {
	if c.wnd != nil {
		c.wnd.RestoreWindow()
	}
}

func (c *Form) Minimize() {
	if c.wnd != nil {
		c.wnd.MinimizeWindow()
	}
}

// SetAlwaysOnTop keeps the window above the other windows; call it once the form is shown
func (c *Form) SetAlwaysOnTop(onTop bool) {
	if c.wnd != nil {
		c.wnd.SetAlwaysOnTop(onTop)
	}
}

// RequestAttention marks the window in the taskbar/dock until the user looks at it
func (c *Form) RequestAttention() {
	if c.wnd != nil {
		c.wnd.RequestAttention()
	}
}

// Beep plays the system alert sound
func (c *Form) Beep() {
	if c.wnd != nil {
		c.wnd.Beep()
	}
}

func (c *Form) UpdateLayout() {
	if c != nil && c.Panel() != nil {
		c.Panel().ClearLayoutCache()
		// Lays out the panel too, below the menu bar
		c.layoutMenuBar()
		for _, popupWidget := range c.Panel().PopupWidgets {
			if popupWidget != nil {
				popupWidget.updateLayout(0, 0, 0, 0)
			}
		}
		c.Update()
	}
}

func (c *Form) WidgetById(id string) Widgeter {
	if widget, exists := lookupWidget(id); exists {
		return widget
	}
	return nil
}

func NewForm() *Form {
	var c Form

	c.title = "Form"
	c.posX = -1
	c.posY = -1
	c.width = 800
	c.height = 600
	c.allowMinimize = true
	c.allowMaximize = true
	topWidget := NewPanel()
	topWidget.form = &c
	topWidget.SetName("FormTopWidget")
	topWidget.SetPosition(0, 0)
	topWidget.SetSize(c.width, c.height)
	topWidget.SetAnchors(true, true, true, true)
	topWidget.SetAutoFillBackground(true)
	c.topWidget = topWidget
	registerWidget(topWidget)
	return &c
}

func newWidgetId() string {
	id := fmt.Sprint(atomic.AddInt64(&nextWidgetId, 1))
	for len(id) < 3 {
		id = "0" + id
	}
	return id
}

// OpenPopup shows w above the form's widgets, e.g. a dropdown of your own
// widget. Before calling it, give w its size and its position in the form's
// client coordinates - RectClientAreaOnWindow tells where a widget is:
//
//	x, y := field.RectClientAreaOnWindow()
//	picker.SetSize(200, 150)
//	picker.SetPosition(x, y+field.Height())
//	form.OpenPopup(picker)
//
// The popup is shown in its own window, so it can extend beyond the form,
// and is kept on the screen. Implement PopupPlacer to choose where it goes
// when it doesn't fit. Its children are laid out on the grid like in any panel.
//
// The popup closes on a click outside it (see SetCloseByClickOutside), on
// Escape, when the form loses activation or is moved, and by CloseTopPopup.
func (c *Form) OpenPopup(w Widgeter) {
	// A new widget isn't attached to any form yet, and without a form its
	// children weren't laid out when its size was set
	w.attachToForm(w, c)
	w.updateLayout(w.Width(), w.Height(), w.Width(), w.Height())
	c.topWidget.AppendPopupWidget(w)
	c.Update()
}

func (c *Form) TopPopupWidget() Widgeter {
	if len(c.topWidget.PopupWidgets) > 0 {
		return c.topWidget.PopupWidgets[len(c.topWidget.PopupWidgets)-1]
	}
	return nil
}

func (c *Form) CloseTopPopup() {
	c.topWidget.CloseTopPopup()
}

// Close closes the window at once, without calling OnClose - the same on
// every platform. It may be called from any goroutine (off the UI thread it
// is handed over to it, see Invoke). A window that has a modal dialog open
// closes as soon as the dialog does.
// To close the window as if the user clicked its close button, so OnClose
// can veto it, use RequestClose.
func (c *Form) Close() {
	if !platforms.IsUIThread() {
		platforms.Post(c.Close)
		return
	}
	c.tooltipClose()
	c.closeToasts()
	c.destroyPopupWindows()
	if c.wnd != nil {
		if !c.wnd.Close() {
			return // still open: keep it registered
		}
		c.wnd = nil
	}
	unregisterOpenForm(c)
}

// RequestClose closes the window as if the user clicked its close button:
// OnClose is called first and may keep the window open.
// Returns true if the window is closed. Call it on the UI thread (e.g. from
// a handler or via Invoke), like OnClose itself runs.
func (c *Form) RequestClose() bool {
	if !c.processWindowClose() {
		return false
	}
	c.Close()
	return true
}

func (c *Form) SetTitle(title string) {
	c.title = title
	if c.wnd != nil {
		c.wnd.SetTitle(title)
	}
}

// appIcon is the icon every new form gets unless it has its own (see SetIcon).
var appIcon *image.RGBA

// SetAppIcon sets the default icon for all windows of the application.
// Call it at startup, before showing any form - forms that are already shown
// keep their current icon. A nil image is ignored.
func SetAppIcon(img image.Image) {
	if img == nil {
		return
	}
	appIcon = toRGBA(img)
}

// SetIcon overrides the application icon (see SetAppIcon) for this form.
// Can be called before or after the window is shown. A nil image is ignored.
func (c *Form) SetIcon(img image.Image) {
	if img == nil {
		return
	}
	c.icon = toRGBA(img)
	if c.wnd != nil {
		c.wnd.SetAppIcon(c.icon)
	}
}

func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	return rgba
}

func (c *Form) SetSize(width, height int) {
	c.width = width
	c.height = height
	if c.wnd != nil {
		c.wnd.Resize(width, height)
	}
}

func (c *Form) Position() (int, int) {
	if c.wnd != nil {
		return c.wnd.PosX(), c.wnd.PosY()
	}
	return 0, 0
}

func (c *Form) Size() (int, int) {
	return c.width, c.height
}

// KeyModifiers returns which of Shift, Ctrl, Alt (and Cmd on macOS) are held
// right now - not only inside a key event. A program can show what the
// keys do with them, like the function key bar of a file manager that changes
// while Shift is held: read it in a timer. While another window or application
// is active nothing is held for this window.
// Call it on the UI thread. Before the window is shown it returns the state of
// the last key event.
func (c *Form) KeyModifiers() KeyModifiers {
	if c.wnd != nil {
		return c.wnd.KeyModifiers()
	}
	return c.lastKeyboardModifiers
}

func (c *Form) IsMaximized() bool {
	if c.wnd != nil {
		return c.wnd.IsMaximized()
	}
	return false
}

func (c *Form) SetAllowMinimize(allow bool) {
	c.allowMinimize = allow
	if c.wnd != nil {
		c.wnd.SetAllowMinimize(allow)
	}
}

func (c *Form) SetAllowMaximize(allow bool) {
	c.allowMaximize = allow
	if c.wnd != nil {
		c.wnd.SetAllowMaximize(allow)
	}
}

// SetOnActivated sets the function called when the window becomes active:
// shown, or the user came back to it from another window or application
func (c *Form) SetOnActivated(f func()) {
	c.onActivated = f
}

// SetOnDeactivated sets the function called when the window stops being
// active: the user switched to another window or application
func (c *Form) SetOnDeactivated(f func()) {
	c.onDeactivated = f
}

// IsActive reports whether the window has the keyboard focus
func (c *Form) IsActive() bool {
	return c.active
}

func (c *Form) SetOnGlobalKeyDown(onGlobalKeyDown func(keyCode Key, mods KeyModifiers) bool) {
	c.onGlobalKeyDown = onGlobalKeyDown
}

func (c *Form) SetMainWidget(w Widgeter) {
	c.topWidget.RemoveAllWidgets()
	c.topWidget.AddWidget(0, 0, w)
}

func (c *Form) Panel() *Panel {
	return c.topWidget
}

// SetMenuBar shows the main menu at the top of the form, above Panel().
// nil removes it.
func (c *Form) SetMenuBar(bar *MenuBar) {
	if c.menuBar != nil {
		c.closePopups()
		c.menuBar.attachToForm(c.menuBar, nil)
	}
	c.menuBar = bar
	if bar != nil {
		bar.attachToForm(bar, c)
	}
	c.layoutMenuBar()
	c.Update()
}

func (c *Form) MenuBar() *MenuBar {
	return c.menuBar
}

// layoutMenuBar places the menu bar at the top of the client area and the
// top widget in the rest of it.
func (c *Form) layoutMenuBar() {
	barHeight := 0
	if c.menuBar != nil {
		barHeight = min(c.menuBar.barHeight(), c.height)
		c.menuBar.SetPosition(0, 0)
		c.menuBar.SetSize(c.width, barHeight)
		c.menuBar.rebuildVisualElements()
	}
	c.topWidget.SetPosition(0, barHeight)
	c.topWidget.SetSize(c.width, c.height-barHeight)
}

func (c *Form) createWindow(maximized bool) {
	// No explicit position and no parent to center on (ShowModal already
	// resolves posX/posY via centerOnForm before this runs) - center on the
	// screen instead of leaving it to the OS/window manager's default spot.
	centerOnScreen := c.posX < 0 && c.posY < 0
	c.wnd = platforms.CreateWindow(c.title, c.posX, c.posY, c.width, c.height, centerOnScreen, maximized)
	c.wnd.OnPaint(c.processPaint)
	c.wnd.OnResize(c.processResize)
	c.wnd.OnMouseButtonDown(func(button MouseButton, x, y int) {
		c.popupUnderMouse = nil
		c.processMouseDown(button, x, y)
	})
	c.wnd.OnMouseButtonUp(c.processMouseUp)
	c.wnd.OnMouseButtonDblClick(func(button MouseButton, x, y int) {
		c.popupUnderMouse = nil
		c.processMouseDblClick(button, x, y)
	})
	c.wnd.OnMouseMove(func(x, y int) {
		c.popupUnderMouse = nil
		c.processMouseMove(x, y)
	})
	c.wnd.OnMouseWheel(c.processMouseWheel)
	c.wnd.OnMouseLeave(func() {
		// The mouse went into a popup window: it's still over the form's
		// widgets. Popup windows report their own leave.
		if c.popupUnderMouse == nil {
			c.processMouseLeave()
		}
	})
	c.wnd.OnMouseEnter(c.processMouseEnter)
	c.wnd.OnKeyDown(c.processKeyDown)
	c.wnd.OnKeyUp(c.processKeyUp)
	c.wnd.OnChar(c.processChar)
	c.wnd.OnTimer(c.processTimer)
	c.wnd.OnMove(c.processWindowMove)
	c.wnd.OnDeactivate(c.processDeactivate)
	c.wnd.OnActivate(c.processActivate)
	c.wnd.OnCloseRequest(c.processWindowClose)
	c.wnd.OnFilesDropped(c.processFilesDropped)
	c.wnd.SetAllowMinimize(c.allowMinimize)
	c.wnd.SetAllowMaximize(c.allowMaximize)
	if c.icon != nil {
		c.wnd.SetAppIcon(c.icon)
	} else if appIcon != nil {
		c.wnd.SetAppIcon(appIcon)
	}
	if c.posX >= 0 && c.posY >= 0 {
		c.wnd.Move(c.posX, c.posY)
	}
	c.wnd.SetDarkMode(IsDarkTheme)
	c.wnd.SetBackgroundColor(currentPalette.Window)
	registerOpenForm(c)
}

// Show opens the form as a non-modal window and returns at once; a hidden
// form (see Hide) comes back. The window lives on the UI thread: called from
// another goroutine, Show opens it there and waits for it (see InvokeSync).
func (c *Form) Show() {
	platforms.RunOnUIThread(func() {
		if c.wnd != nil {
			c.wnd.Show()
			c.hidden = false
			c.forceUpdate()
			return
		}
		c.createWindow(false)
		c.wnd.Show()
		c.processResize(c.width, c.height)
	})
}

// Hide hides the form's window, taskbar button and all, until the next Show.
// The form stays open: Exec keeps waiting, the timers keep running. Useful
// with a TrayIcon, to keep the application in the tray.
func (c *Form) Hide() {
	platforms.RunOnUIThread(func() {
		if c.wnd == nil {
			return
		}
		c.tooltipHide()
		c.closePopups()
		c.closeToasts()
		c.wnd.Hide()
		c.hidden = true
	})
}

// IsHidden reports whether the form is hidden by Hide
func (c *Form) IsHidden() bool {
	return c.hidden
}

// ShowModal opens the form as a modal dialog over parent, which gets no input
// until the dialog is closed. It returns at once (on macOS, once the dialog
// is closed). Like Show, it opens the window on the UI thread.
func (c *Form) ShowModal(parent *Form) {
	if parent == nil {
		panic("parent form cannot be nil for ShowModal")
	}
	platforms.RunOnUIThread(func() {
		c.parentForm = parent
		if c.posX < 0 && c.posY < 0 {
			c.centerOnForm(parent)
		}
		c.createWindow(false)
		c.wnd.ShowModal(parent.wnd)
		c.processResize(c.width, c.height)
	})
}

// centerOnForm positions the window at the center of the given parent form.
func (c *Form) centerOnForm(parent *Form) {
	parentX, parentY := parent.Position()
	parentWidth, parentHeight := parent.Size()
	c.posX = parentX + (parentWidth-c.width)/2
	c.posY = parentY + (parentHeight-c.height)/2
	if c.posX < 0 {
		c.posX = 0
	}
	if c.posY < 0 {
		c.posY = 0
	}
}

// MoveToCenterOfParent re-centers the window over the form it was shown
// modally over (see ShowModal). Useful when a dialog only knows its real
// size once it's building itself (e.g. from OnDialogShow, after SetSize),
// i.e. too late for ShowModal's own initial centering to use it. Does
// nothing if the form was not shown via ShowModal.
func (c *Form) MoveToCenterOfParent() {
	if c.parentForm == nil {
		return
	}
	c.centerOnForm(c.parentForm)
	c.Move(c.posX, c.posY)
}

func (c *Form) ShowMaximized() {
	platforms.RunOnUIThread(func() {
		c.createWindow(true)
		c.wnd.Show()
		c.wnd.MaximizeWindow()
		c.processResize(c.width, c.height)
	})
}

// Exec waits until the form is closed. Called from the main goroutine (the
// UI thread), it runs the event loop of all the forms meanwhile - this is how
// a program runs its main form:
//
//	form.Show()
//	form.Exec()
//
// Called from a handler, it runs a nested event loop, so the other forms keep
// working. Called from another goroutine, it just waits.
func (c *Form) Exec() {
	if c.wnd == nil {
		panic("window is not created. Call Show or ShowModal first")
	}
	c.wnd.Exec()
}

func (c *Form) realUpdate() {
	if c.wnd != nil && c.needUpdate {
		c.wnd.Update()
		c.updatePopupWindows()
		c.needUpdate = false
		c.lastUpdateTime = time.Now()
	}
}

// minUpdateInterval is the least time between two repaints of a form: the
// updates asked for sooner are joined and painted by the timer. It is a
// frame of a 60 Hz screen, so drawing with the mouse follows it closely.
const minUpdateInterval = 16 * time.Millisecond

func (c *Form) Update() {
	if c == nil {
		return
	}
	if c.wnd == nil {
		return
	}
	if c.updateBlockStack > 0 {
		c.needUpdate = true
		return
	}
	c.needUpdate = true
	if time.Since(c.lastUpdateTime) > minUpdateInterval {
		c.realUpdate()
	}
}

func (c *Form) forceUpdate() {
	c.needUpdate = true
	c.realUpdate()
}

func (c *Form) processWindowClose() bool {
	if c.OnClose != nil && !c.OnClose() {
		return false
	}
	c.closeToasts()
	unregisterOpenForm(c)
	return true
}

// applyTheme repaints the form in the current theme, see ApplyPalette.
func (c *Form) applyTheme() {
	if c.wnd == nil {
		return
	}
	c.wnd.SetDarkMode(IsDarkTheme)
	c.wnd.SetBackgroundColor(currentPalette.Window)
	c.forceUpdate()
}

func (c *Form) processPaint(rgba *image.RGBA) {
	if c.wnd == nil {
		return // closed: the native window goes away with the next turn of the loop
	}
	c.paintScaled(rgba, c.wnd.Scale())
}

// paintScaled paints the form on rgba, scale pixels of it per logical pixel
func (c *Form) paintScaled(rgba *image.RGBA, scale float64) {
	cnv := NewCanvasScaled(rgba, scale)
	cnv.SetDirectTranslateAndClip(0, 0, c.width, c.height)
	if c.menuBar != nil {
		cnv.Save()
		cnv.TranslateAndClip(c.menuBar.X(), c.menuBar.Y(), c.menuBar.Width(), c.menuBar.Height())
		c.menuBar.ProcessPaint(cnv)
		cnv.Restore()
	}
	cnv.Save()
	cnv.TranslateAndClip(c.topWidget.X(), c.topWidget.Y(), c.topWidget.Width(), c.topWidget.Height())
	c.topWidget.ProcessPaint(cnv)
	cnv.Restore()
	c.dragPaint(cnv)
	c.toastsPaint(cnv)
	c.tooltipPaint(cnv)
	if c.hoverWidget != nil {
		//c.DrawWidgetDebugInfo(c.hoverWidget, cnv)
	}
}

func (c *Form) DrawWidgetDebugInfo(w Widgeter, cnv *Canvas) {
	if w == nil {
		return
	}
	lines := make([]string, 0)
	posX := c.lastMouseX + 16
	posY := c.lastMouseY + 16
	col := color.RGBA{R: 0, G: 200, B: 200, A: 255}
	lines = append(lines, fmt.Sprintf("ID: %s", w.Id()))
	lines = append(lines, fmt.Sprintf("Name: %s", w.Name()))
	lines = append(lines, fmt.Sprintf("Type: %s", w.TypeName()))
	lines = append(lines, fmt.Sprintf("Position: (%d, %d)", w.X(), w.Y()))
	lines = append(lines, fmt.Sprintf("Size: %dx%d", w.Width(), w.Height()))
	lines = append(lines, fmt.Sprintf("Inner Size: %dx%d", w.InnerWidth(), w.InnerHeight()))
	lines = append(lines, fmt.Sprintf("Grid Position: (%d, %d)", w.GridX(), w.GridY()))
	lines = append(lines, fmt.Sprintf("Expandable: %t %t", w.XExpandable(), w.YExpandable()))
	lines = append(lines, fmt.Sprintf("Min Size: %dx%d", w.MinWidth(), w.MinHeight()))
	lines = append(lines, fmt.Sprintf("Max Size: %dx%d", w.MaxWidth(), w.MaxHeight()))

	for _, line := range lines {
		cnv.FillRect(posX, posY, 200, 20, color.RGBA{R: 0, G: 0, B: 0, A: 150})
		//cnv.DrawText(posX, posY, line, "roboto", 16, col, false)
		cnv.SetHAlign(HAlignLeft)
		cnv.SetVAlign(VAlignTop)
		cnv.SetColor(col)
		cnv.SetFontFamily("roboto")
		cnv.SetFontSize(12)
		cnv.DrawText(posX, posY, 200, 20, line)
		//fmt.Println("PosX:", posX, "PosY:", posY, "Line:", line)
		posY += 20
	}

	/*for y := 0; y < c.height; y += 10 {
		cnv.DrawLine(0, y, c.width, y, 1, color.RGBA{R: 0, G: 100, B: 0, A: 150})
		if y%50 == 0 {
			cnv.DrawText(0, y, fmt.Sprintf("%d", y), "roboto", 12, color.RGBA{R: 0, G: 200, B: 0, A: 255}, false)
		}
	}*/

}

func (c *Form) processResize(width, height int) {
	c.width = width
	c.height = height
	c.layoutMenuBar()
	c.layoutToasts()
	c.forceUpdate()
}

func (c *Form) processMouseDown(button MouseButton, x int, y int) {
	// fmt.Println("Mouse CLK at:", x, y, "Button:", button)
	if button == MouseButtonLeft {
		c.mouseLeftButtonPressed = true
	}
	c.tooltipSuppress()
	if c.popupUnderMouse == nil && c.toastsProcessMouseDown(x, y) {
		c.mouseLeftButtonPressed = false
		return
	}
	if c.closePopupsByClickOutside() {
		c.Update()
		return
	}
	c.mouseDownPopup = c.popupUnderMouse
	target, targetX, targetY := c.mouseTarget(c.popupUnderMouse, x, y)
	widgetAtCoords := c.widgetUnderMouse(x, y)
	if c.mouseLeftButtonPressed {
		c.mouseLeftButtonPressedWidget = widgetAtCoords
		if button == MouseButtonLeft && c.popupUnderMouse == nil {
			c.dragMouseDown(widgetAtCoords, x, y)
		}
	}
	// A click on the menu bar keeps the focus, so the chosen menu item acts
	// on the focused widget
	if widgetAtCoords != nil && target != Widgeter(c.menuBar) {
		if widgetAtCoords.IsCanBeFocused() {
			widgetAtCoords.Focus()
		} else {
			if c.FocusedWidget() != nil {
				c.FocusedWidget().ClearFocus()
			}
		}
	}
	target.ProcessMouseDown(button, targetX, targetY, c.lastKeyboardModifiers)
	c.Update()
}

func (c *Form) processMouseDblClick(button MouseButton, x int, y int) {
	if button == MouseButtonLeft {
		c.mouseLeftButtonPressed = true
	}
	if c.closePopupsByClickOutside() {
		c.Update()
		return
	}
	c.mouseDownPopup = c.popupUnderMouse
	target, targetX, targetY := c.mouseTarget(c.popupUnderMouse, x, y)
	widgetAtCoords := c.widgetUnderMouse(x, y)
	if c.mouseLeftButtonPressed {
		c.mouseLeftButtonPressedWidget = widgetAtCoords
	}
	if widgetAtCoords != nil && target != Widgeter(c.menuBar) {
		widgetAtCoords.Focus()
	}
	target.ProcessMouseDblClick(button, targetX, targetY, c.lastKeyboardModifiers)
	c.Update()
}

func (c *Form) processMouseUp(button MouseButton, x int, y int) {
	mouseLeftButtonPressedWidgetId := ""
	if c.mouseLeftButtonPressedWidget != nil {
		mouseLeftButtonPressedWidgetId = c.mouseLeftButtonPressedWidget.Id()
	}

	if button == MouseButtonLeft {
		c.mouseLeftButtonPressed = false
		c.mouseLeftButtonPressedWidget = nil
		// After a drag the release goes nowhere: it must not click the
		// widget the drag started on
		if c.dragMouseUp(x, y) {
			x, y = dragCancelledPos, dragCancelledPos
		}
	}

	// The release goes where the button was pressed, wherever the mouse is now
	downPopup := c.mouseDownPopup
	c.mouseDownPopup = nil
	if downPopup != nil && !c.isPopupHostOpen(downPopup) {
		c.Update() // the press closed that popup
		return
	}
	if downPopup != nil {
		target, targetX, targetY := c.mouseTarget(downPopup, x, y)
		target.ProcessMouseUp(button, targetX, targetY, c.lastKeyboardModifiers, mouseLeftButtonPressedWidgetId)
	} else {
		// Wherever the mouse is now, both get it: only the pressed widget
		// handles it (see onlyForWidgetId)
		if c.menuBar != nil {
			c.menuBar.ProcessMouseUp(button, x-c.menuBar.X(), y-c.menuBar.Y(), c.lastKeyboardModifiers, mouseLeftButtonPressedWidgetId)
		}
		c.topWidget.ProcessMouseUp(button, x-c.topWidget.X(), y-c.topWidget.Y(), c.lastKeyboardModifiers, mouseLeftButtonPressedWidgetId)
	}

	c.Update()
}

func (c *Form) processMouseMove(x int, y int) {
	if c.dragMouseMove(x, y) {
		return
	}
	if c.mouseLeftButtonPressedWidget != nil {
		wX, wY := c.mouseLeftButtonPressedWidget.RectClientAreaOnWindow()
		c.mouseLeftButtonPressedWidget.ProcessMouseMove(x-wX, y-wY, c.lastKeyboardModifiers)
		if c.mouseDownPopup != nil {
			c.followMouseInPopups(x, y)
		}
		c.Update()
		return
	}

	target, targetX, targetY := c.mouseTarget(c.popupUnderMouse, x, y)
	target.ProcessMouseMove(targetX, targetY, c.lastKeyboardModifiers)

	c.lastMouseX = x
	c.lastMouseY = y
	c.updateHover()

	c.Update()
}

// updateHover finds the widget under the mouse and sets its cursor.
// Called on mouse moves and when the widgets under the mouse change
// without a move, e.g. a popup menu is closed by a click.
func (c *Form) updateHover() {
	if c == nil || c.topWidget == nil {
		return
	}
	hoverWidget := c.widgetUnderMouse(c.lastMouseX, c.lastMouseY)

	if hoverWidget != c.hoverWidget {
		if c.hoverWidget != nil {
			c.hoverWidget.ProcessMouseLeave()
		}
		c.hoverWidget = hoverWidget
		if c.hoverWidget != nil {
			c.hoverWidget.ProcessMouseEnter()
		}
		c.tooltipHoverChanged()
	}

	newCursor := MouseCursorArrow
	if c.hoverWidget != nil {
		if c.hoverWidget.MouseCursor() != MouseCursorNotDefined {
			newCursor = c.hoverWidget.MouseCursor()
		}
	}

	// The cursor belongs to the window the mouse is over
	if h := c.popupUnderMouse; h != nil {
		if h.cursor != newCursor {
			h.wnd.SetMouseCursor(newCursor)
			h.cursor = newCursor
		}
		return
	}

	if c.lastMouseCursor != newCursor {
		// fmt.Println("Set mouse cursor:", newCursor)
		if c.wnd != nil {
			c.wnd.SetMouseCursor(newCursor)
		}
		c.lastMouseCursor = newCursor
	}
}

func (c *Form) processMouseLeave() {
	if c.menuBar != nil {
		c.menuBar.ProcessMouseLeave()
	}
	c.topWidget.ProcessMouseLeave()
	c.tooltipHide()

	if c.hoverWidget != nil {
		c.hoverWidget.ProcessMouseLeave()
		c.hoverWidget = nil
	}

	c.Update()
}

func (c *Form) processMouseEnter() {
	if c.menuBar != nil {
		c.menuBar.ProcessMouseEnter()
	}
	c.topWidget.ProcessMouseEnter()
}

func (c *Form) FocusedWidget() Widgeter {
	return c.focusedWidget
}

func (c *Form) processKeyDown(keyCode Key, mods KeyModifiers) bool {
	c.tooltipSuppress()

	if c.lastKeyboardModifiers != mods {
		c.lastKeyboardModifiers = mods
	}

	// The keys that edit text go to a focused text field (see handlesKey)
	// before the application's global handler too, not only before the
	// shortcuts: Ctrl+Z in a search box undoes its typing, not the document
	// of the application; the keys it does not take go on as usual
	if c.focusedWidget != nil && len(c.topWidget.PopupWidgets) == 0 && c.focusedWidget.Enabled() {
		if h, ok := c.focusedWidget.(keyHandler); ok && h.handlesKey(keyCode, mods) {
			if c.focusedWidget.ProcessKeyDown(keyCode, mods) {
				c.Update()
				return true
			}
		}
	}

	if c.onGlobalKeyDown != nil {
		if c.onGlobalKeyDown(keyCode, mods) {
			c.Update()
			return true
		}
	}

	if keyCode == KeyEsc && c.dragCancel() {
		return true
	}
	// Alt shows the mnemonics of the main menu
	if keyCode == KeyAlt {
		c.Update()
	}

	// Escape closes the top popup (a submenu closes before its menu) instead
	// of reaching the widgets or the form's cancel button
	if keyCode == KeyEsc && len(c.topWidget.PopupWidgets) > 0 {
		if canceler, ok := c.TopPopupWidget().(PopupCanceler); ok {
			canceler.CancelPopup()
		}
		c.topWidget.CloseTopPopup()
		c.Update()
		return true
	}

	if len(c.topWidget.PopupWidgets) > 0 {
		// An open menu takes the arrows, Enter and the mnemonic letters
		if c.processMenuKeys(keyCode, mods) {
			return true
		}
	} else {
		if c.processMenuBarMnemonic(keyCode, mods) {
			return true
		}
		// The shortcuts with modifiers and the function keys go before the
		// focused widget; the others only reach the shortcuts if it doesn't
		// take them
		if isCommandKey(keyCode, mods) && c.processShortcut(keyCode, mods) {
			return true
		}
	}

	if c.focusedWidget != nil {
		if c.focusedWidget.ProcessKeyDown(keyCode, mods) {
			c.Update()
			return true
		}

		parentWidgetId := c.focusedWidget.ParentWidgetId()
		for parentWidgetId != "" {
			parentWidget := c.WidgetById(parentWidgetId)
			if parentWidget != nil {
				if parentWidget.ProcessKeyDown(keyCode, mods) {
					c.Update()
					return true
				}
				parentWidgetId = parentWidget.ParentWidgetId()
			} else {
				break
			}
		}
	}

	if len(c.topWidget.PopupWidgets) == 0 && !isCommandKey(keyCode, mods) && c.processShortcut(keyCode, mods) {
		return true
	}

	if !c.topWidget.ProcessKeyDown(keyCode, mods) {
		if keyCode == KeyTab {
			reverse := mods.Shift
			if len(c.topWidget.PopupWidgets) > 0 {
				c.topWidget.PopupWidgets[0].nextFocus(reverse)
			} else {
				c.topWidget.nextFocus(reverse)
			}
		}
		if keyCode == KeyArrowUp {
			if len(c.topWidget.PopupWidgets) > 0 {
				c.topWidget.PopupWidgets[0].nextFocusByDirection("up")
			} else {
				c.topWidget.nextFocusByDirection("up")
			}
		}
		if keyCode == KeyArrowDown {
			if len(c.topWidget.PopupWidgets) > 0 {
				c.topWidget.PopupWidgets[0].nextFocusByDirection("down")
			} else {
				c.topWidget.nextFocusByDirection("down")
			}
		}
		if keyCode == KeyArrowLeft {
			if len(c.topWidget.PopupWidgets) > 0 {
				c.topWidget.PopupWidgets[0].nextFocusByDirection("left")
			} else {
				c.topWidget.nextFocusByDirection("left")
			}
		}
		if keyCode == KeyArrowRight {
			if len(c.topWidget.PopupWidgets) > 0 {
				c.topWidget.PopupWidgets[0].nextFocusByDirection("right")
			} else {
				c.topWidget.nextFocusByDirection("right")
			}
		}
	}

	if keyCode == KeyEsc {
		if c.cancelButton != nil {
			c.cancelButton.Push()
		}
	}

	if keyCode == KeyEnter {
		if c.acceptButton != nil {
			c.acceptButton.Push()
		}
	}

	c.Update()
	return false
}

func (c *Form) processKeyUp(keyCode Key, mods KeyModifiers) {
	if c.lastKeyboardModifiers != mods {
		c.lastKeyboardModifiers = mods
	}
	// The mnemonics of the main menu hide
	if keyCode == KeyAlt {
		c.lastKeyboardModifiers.Alt = false
		c.Update()
	}
	if c.focusedWidget != nil {
		c.focusedWidget.ProcessKeyUp(keyCode, mods)
		c.Update()
		return
	}
	c.topWidget.ProcessKeyUp(keyCode, mods)
	c.Update()
}

func (c *Form) processChar(char rune) {
	if c.focusedWidget != nil {
		c.focusedWidget.ProcessChar(char, c.lastKeyboardModifiers)
		c.Update()
		return
	}
	c.topWidget.ProcessChar(char, c.lastKeyboardModifiers)
}

func (c *Form) processMouseWheel(deltaX int, deltaY int) {
	if c.lastKeyboardModifiers.Shift {
		deltaX, deltaY = deltaY, deltaX // Swap for horizontal scrolling
	}
	target, _, _ := c.mouseTarget(c.popupUnderMouse, c.lastMouseX, c.lastMouseY)
	target.ProcessMouseWheel(deltaX, deltaY)
	// The content under the mouse may have scrolled
	c.updateHover()
	c.Update()
}

// Invoke runs f on the UI thread, then updates the form, and returns at once.
// Safe to call from any goroutine: it is how a background goroutine changes
// the widgets of the form. Called on the UI thread, f runs after the current
// handler. See also the package-level Invoke.
func (c *Form) Invoke(f func()) {
	if c == nil || f == nil {
		return
	}
	platforms.Post(func() {
		f()
		c.Update()
	})
}

// ParentForm returns the form this one was shown modally over, nil if none
func (c *Form) ParentForm() *Form {
	return c.parentForm
}

func (c *Form) processTimer() {
	if time.Since(c.lastFreeMemoryTime) > 30*time.Second {
		c.freeMemory()
		c.lastFreeMemoryTime = time.Now()
	}

	if c.menuBar != nil {
		c.menuBar.ProcessTimer()
	}
	c.topWidget.ProcessTimer()
	c.tooltipProcessTimer()
	c.toastsProcessTimer()

	for _, popupWidget := range c.topWidget.PopupWidgets {
		if popupWidget != nil {
			popupWidget.ProcessTimer()
		}
	}

	if c.needUpdate {
		c.realUpdate()
	}
}

func (c *Form) processWindowMove(x, y int) {
	c.tooltipHide()
	c.layoutToasts()
	// Popup windows don't move with the form
	c.closePopups()
	c.forceUpdate()
}

func (c *Form) Move(x, y int) {
	if c == nil {
		return
	}
	c.posX = x
	c.posY = y
	if c.wnd != nil {
		c.wnd.Move(x, y)
	}
}

func (c *Form) freeMemory() {
	runtime.GC()
	debug.FreeOSMemory()
}

func (c *Form) UpdateBlockPush() {
	if c == nil {
		return
	}
	c.updateBlockStack++
}

func (c *Form) UpdateBlockPop() {
	if c == nil {
		return
	}
	if c.updateBlockStack <= 0 {
		return
	}
	c.updateBlockStack--
	if c.updateBlockStack == 0 {
		c.Update()
	}
}

func (c *Form) LayoutingBlockPush() {
	if c == nil {
		return
	}
	c.layoutingBlockStack++
}

func (c *Form) LayoutingBlockPop() {
	if c == nil {
		return
	}

	if c.layoutingBlockStack <= 0 {
		return
	}
	c.layoutingBlockStack--
	if c.layoutingBlockStack == 0 {
		c.UpdateLayout()
	}
}

func (c *Form) SetAcceptButton(button *Button) {
	c.acceptButton = button
}

func (c *Form) SetCancelButton(button *Button) {
	c.cancelButton = button
}
