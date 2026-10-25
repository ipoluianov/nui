//go:build linux

package ui

import (
	"fmt"
	"image"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/godbus/dbus/v5"

	"github.com/ipoluianov/nui/internal/platforms"
)

// The accessible objects on Linux: AT-SPI, the accessibility bus of the
// desktop (D-Bus), which Orca and the other assistive tools read. The
// application registers its root object with the registry of the bus
// (Socket.Embed); every form and widget is an object at a path of its own,
// answering the AT-SPI interfaces (Accessible, Component, Action, Text,
// EditableText, Value) from the snapshot of accessible.go, made on the UI
// thread for each question. A focus change is sent as an event.
//
// As Qt does, the bridge starts only when the desktop has accessibility on
// (org.a11y.Status.IsEnabled: a screen reader runs, or the setting is on),
// at once or when it is turned on later; NUI_ACCESSIBILITY=1 starts it
// always, NUI_ACCESSIBILITY=0 never.

const (
	atspiPathPrefix = "/org/a11y/atspi/accessible/"
	atspiRootPath   = dbus.ObjectPath(atspiPathPrefix + "root")
	atspiNullPath   = dbus.ObjectPath("/org/a11y/atspi/null")
	atspiRegistry   = "org.a11y.atspi.Registry"
)

// atspiRef is an object reference of AT-SPI: the bus name and the path
type atspiRef struct {
	Name string
	Path dbus.ObjectPath
}

type atspiBridge struct {
	conn *dbus.Conn

	mu      sync.Mutex
	desktop atspiRef // the parent of the root, given by the registry

	serials map[string]uint64 // widget id -> the number in its path
	widgets map[uint64]string
	forms   map[uint64]*Form
	formIDs map[*Form]uint64
	next    uint64

	// focused is the object focused last, for the focus events
	focused dbus.ObjectPath
}

var (
	// atspi is the bridge once it runs; set by the goroutine that starts it
	atspi          atomic.Pointer[atspiBridge]
	atspiStartOnce sync.Once
)

// startAccessibility starts the bridge, if accessibility is on; called
// when the first form is shown
func startAccessibility() {
	if platforms.IsHeadless() {
		return
	}
	atspiStartOnce.Do(func() {
		switch os.Getenv("NUI_ACCESSIBILITY") {
		case "0":
			return
		case "1":
			go startATSPI()
			return
		}
		go watchAccessibilityStatus()
	})
}

// watchAccessibilityStatus starts the bridge when the desktop turns
// accessibility on
func watchAccessibilityStatus() {
	session, err := dbus.ConnectSessionBus()
	if err != nil {
		return
	}
	enabled := func() bool {
		v, err := session.Object("org.a11y.Bus", "/org/a11y/bus").GetProperty("org.a11y.Status.IsEnabled")
		on, _ := v.Value().(bool)
		return err == nil && on
	}
	if enabled() {
		session.Close()
		startATSPI()
		return
	}
	session.AddMatchSignal(dbus.WithMatchObjectPath("/org/a11y/bus"),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"), dbus.WithMatchMember("PropertiesChanged"))
	signals := make(chan *dbus.Signal, 4)
	session.Signal(signals)
	for range signals {
		if enabled() {
			session.Close()
			startATSPI()
			return
		}
	}
}

// startATSPI connects to the accessibility bus and registers the root
func startATSPI() {
	session, err := dbus.ConnectSessionBus()
	if err != nil {
		return
	}
	var address string
	err = session.Object("org.a11y.Bus", "/org/a11y/bus").Call("org.a11y.Bus.GetAddress", 0).Store(&address)
	session.Close()
	if err != nil || address == "" {
		return
	}
	conn, err := dbus.Connect(address)
	if err != nil {
		return
	}
	b := &atspiBridge{conn: conn, serials: map[string]uint64{}, widgets: map[uint64]string{},
		forms: map[uint64]*Form{}, formIDs: map[*Form]uint64{}}
	prefix := dbus.ObjectPath(strings.TrimSuffix(atspiPathPrefix, "/"))
	for _, e := range []struct {
		v     any
		iface string
	}{
		{atspiAccessible{b}, "org.a11y.atspi.Accessible"},
		{atspiComponent{b}, "org.a11y.atspi.Component"},
		{atspiAction{b}, "org.a11y.atspi.Action"},
		{atspiText{b}, "org.a11y.atspi.Text"},
		{atspiEditableText{b}, "org.a11y.atspi.EditableText"},
		{atspiProperties{b}, "org.freedesktop.DBus.Properties"},
	} {
		if err := conn.ExportSubtree(e.v, prefix, e.iface); err != nil {
			conn.Close()
			return
		}
	}
	// The cache of the objects: empty, the clients ask the objects instead
	conn.Export(atspiCache{}, "/org/a11y/atspi/cache", "org.a11y.atspi.Cache")
	atspi.Store(b)
	var desktop atspiRef
	plug := atspiRef{conn.Names()[0], atspiRootPath}
	if err := conn.Object(atspiRegistry, atspiRootPath).Call("org.a11y.atspi.Socket.Embed", 0, plug).Store(&desktop); err == nil {
		b.mu.Lock()
		b.desktop = desktop
		b.mu.Unlock()
	}
}

// accessibilityFocusChanged sends the focus event when the focused object
// changed; the forms call it after each frame
func (c *Form) accessibilityFocusChanged() {
	b := atspi.Load()
	if b == nil || !c.active {
		return
	}
	var path dbus.ObjectPath
	if w := c.FocusedWidget(); w != nil {
		path = b.widgetPath(w)
		// A list or a table: its current item
		n := accessibleNodeOf(w)
		if items, ok := n.Widget.(accessibleItemsWidget); ok {
			for i := range min(items.accessibleItemCount(), maxAccessibleItems) {
				if accessibleItemNode(n.Widget, i).Focused {
					path = b.itemPath(n.Widget, i)
					break
				}
			}
		}
	} else {
		path = b.formPath(c)
	}
	b.mu.Lock()
	old := b.focused
	b.focused = path
	b.mu.Unlock()
	if old == path {
		return
	}
	if old != "" {
		b.emit(old, "StateChanged", "focused", 0)
	}
	b.emit(path, "StateChanged", "focused", 1)
}

// emit sends an Object event of the object
func (b *atspiBridge) emit(path dbus.ObjectPath, member, minor string, detail int32) {
	b.conn.Emit(path, "org.a11y.atspi.Event.Object."+member, minor, detail, int32(0),
		dbus.MakeVariant(int32(0)), map[string]dbus.Variant{})
}

// Paths: the root, f<n> a form, w<n> a widget, w<n>_<i> an item of it

func (b *atspiBridge) formPath(f *Form) dbus.ObjectPath {
	b.mu.Lock()
	defer b.mu.Unlock()
	id, ok := b.formIDs[f]
	if !ok {
		b.next++
		id = b.next
		b.formIDs[f], b.forms[id] = id, f
	}
	return dbus.ObjectPath(atspiPathPrefix + "f" + strconv.FormatUint(id, 10))
}

func (b *atspiBridge) widgetSerial(w Widgeter) uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	id, ok := b.serials[w.Id()]
	if !ok {
		b.next++
		id = b.next
		b.serials[w.Id()], b.widgets[id] = id, w.Id()
	}
	return id
}

func (b *atspiBridge) widgetPath(w Widgeter) dbus.ObjectPath {
	return dbus.ObjectPath(atspiPathPrefix + "w" + strconv.FormatUint(b.widgetSerial(w), 10))
}

func (b *atspiBridge) itemPath(w Widgeter, item int) dbus.ObjectPath {
	return dbus.ObjectPath(fmt.Sprintf("%sw%d_%d", atspiPathPrefix, b.widgetSerial(w), item))
}

func (b *atspiBridge) pathOf(n *AccessibleNode) dbus.ObjectPath {
	switch {
	case n == nil:
		return atspiNullPath
	case n.Widget == nil:
		return b.formPath(n.form)
	case n.Item >= 0:
		return b.itemPath(n.Widget, n.Item)
	}
	return b.widgetPath(n.Widget)
}

func (b *atspiBridge) ref(n *AccessibleNode) atspiRef {
	return atspiRef{b.conn.Names()[0], b.pathOf(n)}
}

// atspiObject is what a path names: the application root, or an object
type atspiObject struct {
	root bool
	node *AccessibleNode
}

// resolve finds the object of the path, nil if it is gone (a closed form,
// a removed widget). On the UI thread.
func (b *atspiBridge) resolve(path dbus.ObjectPath) *atspiObject {
	if path == atspiRootPath {
		return &atspiObject{root: true}
	}
	name := strings.TrimPrefix(string(path), atspiPathPrefix)
	if strings.HasPrefix(name, "f") {
		id, _ := strconv.ParseUint(name[1:], 10, 64)
		b.mu.Lock()
		f := b.forms[id]
		b.mu.Unlock()
		if f == nil || f.wnd == nil {
			return nil
		}
		return &atspiObject{node: f.accessibleRoot()}
	}
	if !strings.HasPrefix(name, "w") {
		return nil
	}
	serial, item, hasItem := strings.Cut(name[1:], "_")
	id, _ := strconv.ParseUint(serial, 10, 64)
	b.mu.Lock()
	widgetID, ok := b.widgets[id]
	b.mu.Unlock()
	if !ok {
		return nil
	}
	w, ok := lookupWidget(widgetID)
	if !ok || w.Form() == nil {
		return nil
	}
	if hasItem {
		i, _ := strconv.Atoi(item)
		items, ok := w.(accessibleItemsWidget)
		if !ok || i < 0 || i >= items.accessibleItemCount() {
			return nil
		}
		return &atspiObject{node: accessibleItemNode(w, i)}
	}
	return &atspiObject{node: accessibleNodeOf(w)}
}

// parentOf is the object above n: the next widget up that is an object, or
// the form
func parentOf(n *AccessibleNode) *AccessibleNode {
	if n.Widget == nil {
		return nil
	}
	if n.Item >= 0 {
		return accessibleNodeOf(n.Widget)
	}
	for p := n.Widget.widgetBase().ParentWidget(); p != nil; p = p.widgetBase().ParentWidget() {
		if pn := accessibleNodeOf(p); pn.Role != RoleNone && !accessibleLeaf(pn.Role) {
			return pn
		}
	}
	return n.Widget.Form().accessibleRoot()
}

// with runs f with the object of the message's path on the UI thread
func (b *atspiBridge) with(msg dbus.Message, f func(o *atspiObject)) *dbus.Error {
	path, _ := msg.Headers[dbus.FieldPath].Value().(dbus.ObjectPath)
	var gone bool
	platforms.RunOnUIThread(func() {
		o := b.resolve(path)
		if o == nil {
			gone = true
			return
		}
		f(o)
	})
	if gone {
		return dbus.MakeFailedError(fmt.Errorf("the object %s is gone", path))
	}
	return nil
}

// childrenOf are the objects below o
func (b *atspiBridge) childrenOf(o *atspiObject) []*AccessibleNode {
	if o.root {
		var roots []*AccessibleNode
		for _, f := range openForms() {
			if f.wnd != nil {
				roots = append(roots, f.accessibleRoot())
			}
		}
		slices.SortFunc(roots, func(a, c *AccessibleNode) int { return strings.Compare(string(b.pathOf(a)), string(b.pathOf(c))) })
		return roots
	}
	return accessibleChildren(o.node)
}

// atspiRoles are the AT-SPI roles of the roles
var atspiRoles = map[AccessibleRole]uint32{
	RoleNone: 39, RoleWindow: 23, RoleDialog: 16, RoleGroup: 39, RoleButton: 43, RoleCheckBox: 7,
	RoleRadioButton: 44, RoleSwitch: 62, RoleLabel: 29, RoleLink: 88, RoleTextField: 61,
	RolePasswordField: 40, RoleTextArea: 61, RoleSpinButton: 52, RoleSlider: 51, RoleProgressBar: 42,
	RoleComboBox: 11, RoleList: 98, RoleListItem: 32, RoleTable: 55, RoleTableRow: 90, RoleTree: 65,
	RoleTreeItem: 91, RoleTabList: 38, RoleTab: 37, RoleMenuBar: 34, RoleMenu: 33, RoleMenuItem: 35,
	RoleToolBar: 63, RoleStatusBar: 54, RoleScrollBar: 48, RoleImage: 27, RoleCanvas: 6,
	RoleSplitter: 53, RoleCalendar: 5,
}

const atspiRoleApplication = 75

// The states of AT-SPI, bits of two uint32
const (
	atspiActive        = 1
	atspiChecked       = 4
	atspiCollapsed     = 5
	atspiEditable      = 7
	atspiEnabled       = 8
	atspiExpandable    = 9
	atspiExpanded      = 10
	atspiFocusable     = 11
	atspiFocused       = 12
	atspiMultiLine     = 17
	atspiSelectable    = 22
	atspiSelected      = 23
	atspiSensitive     = 24
	atspiShowing       = 25
	atspiSingleLine    = 26
	atspiVisible       = 30
	atspiCheckable     = 41
	atspiReadOnly      = 43
	atspiSelectableTxt = 38
)

func atspiStates(n *AccessibleNode) []uint32 {
	states := []uint32{0, 0}
	set := func(bit int) { states[bit/32] |= 1 << (bit % 32) }
	set(atspiVisible)
	set(atspiShowing)
	if n.Enabled {
		set(atspiEnabled)
		set(atspiSensitive)
	}
	if n.Focusable {
		set(atspiFocusable)
	}
	if n.Focused {
		set(atspiFocused)
	}
	if n.Widget == nil && n.Focused {
		set(atspiActive)
	}
	if n.Checkable {
		set(atspiCheckable)
		if n.Checked {
			set(atspiChecked)
		}
	}
	if n.Selectable {
		set(atspiSelectable)
		if n.Selected {
			set(atspiSelected)
		}
	}
	if n.Expandable {
		set(atspiExpandable)
		if n.Expanded {
			set(atspiExpanded)
		} else {
			set(atspiCollapsed)
		}
	}
	switch n.Role {
	case RoleTextField, RolePasswordField, RoleTextArea:
		set(atspiSelectableTxt)
		if n.Editable {
			set(atspiEditable)
		} else {
			set(atspiReadOnly)
		}
		if n.Multiline {
			set(atspiMultiLine)
		} else {
			set(atspiSingleLine)
		}
	}
	return states
}

// atspiInterfaces are the interfaces an object answers
func atspiInterfaces(n *AccessibleNode) []string {
	ifaces := []string{"org.a11y.atspi.Accessible", "org.a11y.atspi.Component"}
	if len(n.Actions) > 0 {
		ifaces = append(ifaces, "org.a11y.atspi.Action")
	}
	switch n.Role {
	case RoleTextField, RolePasswordField, RoleTextArea, RoleLabel:
		ifaces = append(ifaces, "org.a11y.atspi.Text")
		if n.Editable {
			ifaces = append(ifaces, "org.a11y.atspi.EditableText")
		}
	case RoleSlider, RoleSpinButton, RoleProgressBar, RoleScrollBar:
		ifaces = append(ifaces, "org.a11y.atspi.Value")
	}
	return ifaces
}

// org.a11y.atspi.Accessible

type atspiAccessible struct{ b *atspiBridge }

func (a atspiAccessible) GetChildAtIndex(msg dbus.Message, index int32) (atspiRef, *dbus.Error) {
	ref := atspiRef{a.b.conn.Names()[0], atspiNullPath}
	err := a.b.with(msg, func(o *atspiObject) {
		if children := a.b.childrenOf(o); index >= 0 && int(index) < len(children) {
			ref = a.b.ref(children[index])
		}
	})
	return ref, err
}

func (a atspiAccessible) GetChildren(msg dbus.Message) ([]atspiRef, *dbus.Error) {
	var refs []atspiRef
	err := a.b.with(msg, func(o *atspiObject) {
		for _, c := range a.b.childrenOf(o) {
			refs = append(refs, a.b.ref(c))
		}
	})
	return refs, err
}

func (a atspiAccessible) GetIndexInParent(msg dbus.Message) (int32, *dbus.Error) {
	index := int32(-1)
	err := a.b.with(msg, func(o *atspiObject) {
		if o.root {
			return
		}
		var siblings []*AccessibleNode
		if p := parentOf(o.node); p != nil {
			siblings = accessibleChildren(p)
		} else {
			siblings = a.b.childrenOf(&atspiObject{root: true})
		}
		path := a.b.pathOf(o.node)
		for i, s := range siblings {
			if a.b.pathOf(s) == path {
				index = int32(i)
			}
		}
	})
	return index, err
}

func (a atspiAccessible) GetRelationSet(msg dbus.Message) ([]struct {
	Type    uint32
	Targets []atspiRef
}, *dbus.Error) {
	return nil, nil
}

func (a atspiAccessible) GetRole(msg dbus.Message) (uint32, *dbus.Error) {
	var role uint32
	err := a.b.with(msg, func(o *atspiObject) {
		if o.root {
			role = atspiRoleApplication
		} else {
			role = atspiRoles[o.node.Role]
		}
	})
	return role, err
}

func (a atspiAccessible) GetRoleName(msg dbus.Message) (string, *dbus.Error) {
	var name string
	err := a.b.with(msg, func(o *atspiObject) {
		if o.root {
			name = "application"
		} else {
			name = o.node.Role.String()
		}
	})
	return name, err
}

func (a atspiAccessible) GetLocalizedRoleName(msg dbus.Message) (string, *dbus.Error) {
	return a.GetRoleName(msg)
}

func (a atspiAccessible) GetState(msg dbus.Message) ([]uint32, *dbus.Error) {
	states := []uint32{0, 0}
	err := a.b.with(msg, func(o *atspiObject) {
		if !o.root {
			states = atspiStates(o.node)
		}
	})
	return states, err
}

func (a atspiAccessible) GetAttributes(msg dbus.Message) (map[string]string, *dbus.Error) {
	attrs := map[string]string{"toolkit": "nui"}
	err := a.b.with(msg, func(o *atspiObject) {
		if !o.root && o.node.Role == RoleTreeItem {
			attrs["level"] = strconv.Itoa(o.node.Level + 1)
		}
	})
	return attrs, err
}

func (a atspiAccessible) GetApplication(msg dbus.Message) (atspiRef, *dbus.Error) {
	return atspiRef{a.b.conn.Names()[0], atspiRootPath}, nil
}

func (a atspiAccessible) GetInterfaces(msg dbus.Message) ([]string, *dbus.Error) {
	var ifaces []string
	err := a.b.with(msg, func(o *atspiObject) {
		if o.root {
			ifaces = []string{"org.a11y.atspi.Accessible", "org.a11y.atspi.Application"}
		} else {
			ifaces = atspiInterfaces(o.node)
		}
	})
	return ifaces, err
}

// org.a11y.atspi.Component

type atspiComponent struct{ b *atspiBridge }

const (
	atspiCoordScreen = 0
	atspiCoordWindow = 1
	atspiCoordParent = 2
)

// extents is the rectangle of the object in the coordinates asked
func (b *atspiBridge) extents(n *AccessibleNode, coordType uint32) image.Rectangle {
	r := n.Rect
	form := n.form
	if n.Widget != nil {
		form = n.Widget.Form()
	}
	switch coordType {
	case atspiCoordScreen:
		if form != nil && form.wnd != nil {
			x, y := form.wnd.ClientToScreen(r.Min.X, r.Min.Y)
			r = r.Add(image.Pt(x-r.Min.X, y-r.Min.Y))
		}
	case atspiCoordParent:
		if p := parentOf(n); p != nil && p.Widget != nil {
			r = r.Sub(p.Rect.Min)
		}
	}
	return r
}

func (c atspiComponent) GetExtents(msg dbus.Message, coordType uint32) (struct{ X, Y, W, H int32 }, *dbus.Error) {
	var e struct{ X, Y, W, H int32 }
	err := c.b.with(msg, func(o *atspiObject) {
		if o.root {
			return
		}
		r := c.b.extents(o.node, coordType)
		e.X, e.Y, e.W, e.H = int32(r.Min.X), int32(r.Min.Y), int32(r.Dx()), int32(r.Dy())
	})
	return e, err
}

func (c atspiComponent) GetPosition(msg dbus.Message, coordType uint32) (int32, int32, *dbus.Error) {
	e, err := c.GetExtents(msg, coordType)
	return e.X, e.Y, err
}

func (c atspiComponent) GetSize(msg dbus.Message) (int32, int32, *dbus.Error) {
	e, err := c.GetExtents(msg, atspiCoordWindow)
	return e.W, e.H, err
}

func (c atspiComponent) Contains(msg dbus.Message, x, y int32, coordType uint32) (bool, *dbus.Error) {
	var in bool
	err := c.b.with(msg, func(o *atspiObject) {
		if !o.root {
			in = image.Pt(int(x), int(y)).In(c.b.extents(o.node, coordType))
		}
	})
	return in, err
}

func (c atspiComponent) GetAccessibleAtPoint(msg dbus.Message, x, y int32, coordType uint32) (atspiRef, *dbus.Error) {
	ref := atspiRef{c.b.conn.Names()[0], atspiNullPath}
	err := c.b.with(msg, func(o *atspiObject) {
		if o.root {
			return
		}
		for _, child := range accessibleChildren(o.node) {
			if image.Pt(int(x), int(y)).In(c.b.extents(child, coordType)) {
				ref = c.b.ref(child)
			}
		}
	})
	return ref, err
}

func (c atspiComponent) GetLayer(msg dbus.Message) (uint32, *dbus.Error) {
	return 3, nil // ATSPI_LAYER_WIDGET
}

func (c atspiComponent) GetMDIZOrder(msg dbus.Message) (int16, *dbus.Error) {
	return 0, nil
}

func (c atspiComponent) GetAlpha(msg dbus.Message) (float64, *dbus.Error) {
	return 1, nil
}

func (c atspiComponent) GrabFocus(msg dbus.Message) (bool, *dbus.Error) {
	var ok bool
	path, _ := msg.Headers[dbus.FieldPath].Value().(dbus.ObjectPath)
	InvokeSync(func() {
		if o := c.b.resolve(path); o != nil && !o.root {
			ok = o.node.GrabFocus()
		}
	})
	return ok, nil
}

func (c atspiComponent) ScrollTo(msg dbus.Message, scrollType uint32) (bool, *dbus.Error) {
	return false, nil
}

// org.a11y.atspi.Action

type atspiAction struct{ b *atspiBridge }

func (a atspiAction) actions(msg dbus.Message) ([]string, *dbus.Error) {
	var actions []string
	err := a.b.with(msg, func(o *atspiObject) {
		if !o.root {
			actions = o.node.Actions
		}
	})
	return actions, err
}

func (a atspiAction) GetName(msg dbus.Message, index int32) (string, *dbus.Error) {
	actions, err := a.actions(msg)
	if index < 0 || int(index) >= len(actions) {
		return "", err
	}
	return actions[index], err
}

func (a atspiAction) GetLocalizedName(msg dbus.Message, index int32) (string, *dbus.Error) {
	return a.GetName(msg, index)
}

func (a atspiAction) GetDescription(msg dbus.Message, index int32) (string, *dbus.Error) {
	return "", nil
}

func (a atspiAction) GetKeyBinding(msg dbus.Message, index int32) (string, *dbus.Error) {
	return "", nil
}

func (a atspiAction) GetActions(msg dbus.Message) ([]struct{ Name, Description, KeyBinding string }, *dbus.Error) {
	actions, err := a.actions(msg)
	var list []struct{ Name, Description, KeyBinding string }
	for _, name := range actions {
		list = append(list, struct{ Name, Description, KeyBinding string }{Name: name})
	}
	return list, err
}

func (a atspiAction) DoAction(msg dbus.Message, index int32) (bool, *dbus.Error) {
	var ok bool
	path, _ := msg.Headers[dbus.FieldPath].Value().(dbus.ObjectPath)
	// After the reply: the action may open a modal dialog
	Invoke(func() {
		if o := a.b.resolve(path); o != nil && !o.root && index >= 0 && int(index) < len(o.node.Actions) {
			o.node.DoAction(o.node.Actions[index])
		}
	})
	ok = true
	return ok, nil
}

// org.a11y.atspi.Text: the value of a text field, the text of a label

type atspiText struct{ b *atspiBridge }

func (t atspiText) text(msg dbus.Message) (string, int, *dbus.Error) {
	var text string
	var caret int
	err := t.b.with(msg, func(o *atspiObject) {
		if o.root {
			return
		}
		text, caret = o.node.Value, o.node.Caret
		if o.node.Role == RoleLabel {
			text = o.node.Name
		}
	})
	return text, caret, err
}

// runeSlice is the text from start to end, in runes; end -1 is the end
func runeSlice(text string, start, end int32) string {
	runes := []rune(text)
	if end < 0 || int(end) > len(runes) {
		end = int32(len(runes))
	}
	start = max(0, min(start, end))
	return string(runes[start:end])
}

func (t atspiText) GetText(msg dbus.Message, start, end int32) (string, *dbus.Error) {
	text, _, err := t.text(msg)
	return runeSlice(text, start, end), err
}

func (t atspiText) GetCharacterAtOffset(msg dbus.Message, offset int32) (int32, *dbus.Error) {
	text, _, err := t.text(msg)
	runes := []rune(text)
	if offset < 0 || int(offset) >= len(runes) {
		return 0, err
	}
	return runes[offset], err
}

// boundary returns the part of the text around the offset: the character,
// the word or the line (granularity 0, 1, 3, 4 of AT-SPI; 2 sentence is
// taken as the line)
func boundary(text string, offset int32, granularity uint32) (string, int32, int32) {
	runes := []rune(text)
	n := int32(len(runes))
	offset = max(0, min(offset, n))
	switch granularity {
	case 0: // char
		if offset >= n {
			return "", n, n
		}
		return string(runes[offset]), offset, offset + 1
	case 1: // word
		start, end := offset, offset
		isSpace := func(r rune) bool { return r == ' ' || r == '\n' || r == '\t' }
		for start > 0 && !isSpace(runes[start-1]) {
			start--
		}
		for end < n && !isSpace(runes[end]) {
			end++
		}
		return string(runes[start:end]), start, end
	case 4: // paragraph: the whole text
		return text, 0, n
	}
	start, end := offset, offset
	for start > 0 && runes[start-1] != '\n' {
		start--
	}
	for end < n && runes[end] != '\n' {
		end++
	}
	return string(runes[start:end]), start, end
}

func (t atspiText) GetStringAtOffset(msg dbus.Message, offset int32, granularity uint32) (string, int32, int32, *dbus.Error) {
	text, _, err := t.text(msg)
	s, start, end := boundary(text, offset, granularity)
	return s, start, end, err
}

// GetTextAtOffset is the older form of GetStringAtOffset: the boundary
// types 0 char, 1-2 word, 3-4 sentence, 5-6 line
func (t atspiText) GetTextAtOffset(msg dbus.Message, offset int32, boundaryType uint32) (string, int32, int32, *dbus.Error) {
	granularity := []uint32{0, 1, 1, 3, 3, 3, 3}[min(boundaryType, 6)]
	return t.GetStringAtOffset(msg, offset, granularity)
}

func (t atspiText) GetNSelections(msg dbus.Message) (int32, *dbus.Error) {
	return 0, nil
}

func (t atspiText) GetSelection(msg dbus.Message, index int32) (int32, int32, *dbus.Error) {
	return 0, 0, nil
}

func (t atspiText) SetCaretOffset(msg dbus.Message, offset int32) (bool, *dbus.Error) {
	return false, nil
}

func (t atspiText) GetCharacterExtents(msg dbus.Message, offset int32, coordType uint32) (int32, int32, int32, int32, *dbus.Error) {
	return 0, 0, 0, 0, nil
}

func (t atspiText) GetAttributeRun(msg dbus.Message, offset int32, includeDefaults bool) (map[string]string, int32, int32, *dbus.Error) {
	text, _, err := t.text(msg)
	return map[string]string{}, 0, int32(utf8.RuneCountInString(text)), err
}

func (t atspiText) GetDefaultAttributes(msg dbus.Message) (map[string]string, *dbus.Error) {
	return map[string]string{}, nil
}

// org.a11y.atspi.EditableText

type atspiEditableText struct{ b *atspiBridge }

func (e atspiEditableText) SetTextContents(msg dbus.Message, text string) (bool, *dbus.Error) {
	var ok bool
	path, _ := msg.Headers[dbus.FieldPath].Value().(dbus.ObjectPath)
	InvokeSync(func() {
		if o := e.b.resolve(path); o != nil && !o.root {
			ok = o.node.SetText(text)
		}
	})
	return ok, nil
}

// org.a11y.atspi.Value has properties only: see atspiProperties

// org.freedesktop.DBus.Properties of all the objects

type atspiProperties struct{ b *atspiBridge }

func (p atspiProperties) Get(msg dbus.Message, iface, name string) (dbus.Variant, *dbus.Error) {
	props, err := p.GetAll(msg, iface)
	if err != nil {
		return dbus.Variant{}, err
	}
	v, ok := props[name]
	if !ok {
		return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("no property %s.%s", iface, name))
	}
	return v, nil
}

func (p atspiProperties) GetAll(msg dbus.Message, iface string) (map[string]dbus.Variant, *dbus.Error) {
	props := map[string]dbus.Variant{}
	b := p.b
	err := b.with(msg, func(o *atspiObject) {
		if o.root {
			switch iface {
			case "org.a11y.atspi.Accessible":
				b.mu.Lock()
				parent := b.desktop
				b.mu.Unlock()
				if parent.Path == "" {
					parent = atspiRef{"", atspiNullPath}
				}
				props["Name"] = dbus.MakeVariant(applicationName())
				props["Description"] = dbus.MakeVariant("")
				props["Parent"] = dbus.MakeVariant(parent)
				props["ChildCount"] = dbus.MakeVariant(int32(len(b.childrenOf(o))))
				props["Locale"] = dbus.MakeVariant("")
				props["AccessibleId"] = dbus.MakeVariant("")
			case "org.a11y.atspi.Application":
				props["ToolkitName"] = dbus.MakeVariant("nui")
				props["Version"] = dbus.MakeVariant("1")
				props["AtspiVersion"] = dbus.MakeVariant("2.1")
				props["Id"] = dbus.MakeVariant(int32(0))
			}
			return
		}
		n := o.node
		switch iface {
		case "org.a11y.atspi.Accessible":
			parent := atspiRef{b.conn.Names()[0], atspiRootPath}
			if pn := parentOf(n); pn != nil {
				parent = b.ref(pn)
			}
			props["Name"] = dbus.MakeVariant(n.Name)
			props["Description"] = dbus.MakeVariant(n.Description)
			props["Parent"] = dbus.MakeVariant(parent)
			props["ChildCount"] = dbus.MakeVariant(int32(len(accessibleChildren(n))))
			props["Locale"] = dbus.MakeVariant("")
			props["AccessibleId"] = dbus.MakeVariant("")
			props["HelpText"] = dbus.MakeVariant("")
		case "org.a11y.atspi.Action":
			props["NActions"] = dbus.MakeVariant(int32(len(n.Actions)))
		case "org.a11y.atspi.Text":
			text := n.Value
			if n.Role == RoleLabel {
				text = n.Name
			}
			props["CharacterCount"] = dbus.MakeVariant(int32(utf8.RuneCountInString(text)))
			props["CaretOffset"] = dbus.MakeVariant(int32(n.Caret))
		case "org.a11y.atspi.Value":
			props["MinimumValue"] = dbus.MakeVariant(n.Min)
			props["MaximumValue"] = dbus.MakeVariant(n.Max)
			props["MinimumIncrement"] = dbus.MakeVariant(0.0)
			props["CurrentValue"] = dbus.MakeVariant(n.Current)
			props["Text"] = dbus.MakeVariant(n.Value)
		}
	})
	return props, err
}

func (p atspiProperties) Set(msg dbus.Message, iface, name string, value dbus.Variant) *dbus.Error {
	if iface != "org.a11y.atspi.Value" || name != "CurrentValue" {
		return dbus.MakeFailedError(fmt.Errorf("the property %s.%s can't be set", iface, name))
	}
	v, ok := value.Value().(float64)
	if !ok {
		return dbus.MakeFailedError(fmt.Errorf("CurrentValue is a double"))
	}
	path, _ := msg.Headers[dbus.FieldPath].Value().(dbus.ObjectPath)
	InvokeSync(func() {
		if o := p.b.resolve(path); o != nil && !o.root {
			o.node.SetCurrent(v)
		}
	})
	return nil
}

// org.a11y.atspi.Cache: libatspi asks it for all the objects at once; nui
// gives none, the objects are asked one by one
type atspiCache struct{}

func (atspiCache) GetItems() ([]struct {
	Ref, App, Parent  atspiRef
	Index, ChildCount int32
	Interfaces        []string
	Name              string
	Role              uint32
	Description       string
	States            []uint32
}, *dbus.Error) {
	return nil, nil
}
