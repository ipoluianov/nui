//go:build linux

package platforms

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

// The tray icon is a StatusNotifierItem
// (https://www.freedesktop.org/wiki/Specifications/StatusNotifierItem/) on
// the session D-Bus: KDE Plasma, GNOME (with the AppIndicator extension),
// XFCE, Cinnamon and others show these. The application exports the item
// object and registers it with the tray (org.kde.StatusNotifierWatcher). The
// menu is a com.canonical.dbusmenu object the tray draws itself.
//
// D-Bus calls arrive on godbus' goroutines; the application's callbacks are
// posted to the UI thread.

const (
	sniPath      = dbus.ObjectPath("/StatusNotifierItem")
	sniInterface = "org.kde.StatusNotifierItem"
	menuPath     = dbus.ObjectPath("/MenuBar")
	menuIface    = "com.canonical.dbusmenu"
	watcherName  = "org.kde.StatusNotifierWatcher"
	watcherPath  = dbus.ObjectPath("/StatusNotifierWatcher")
)

var trayCounter atomic.Int32

// sniPixmap is one size of an icon: ARGB32 pixels in network byte order
type sniPixmap struct {
	Width, Height int32
	Data          []byte
}

type sniToolTip struct {
	IconName    string
	IconPixmaps []sniPixmap
	Title       string
	Description string
}

// menuLayout is a dbusmenu item with its children (variants of menuLayout)
type menuLayout struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

type menuItemProps struct {
	ID    int32
	Props map[string]dbus.Variant
}

type menuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

type trayIcon struct {
	conn     *dbus.Conn
	busName  string
	props    *prop.Properties
	appName  string
	icon     *image.RGBA
	notifyID uint32

	mu       sync.Mutex
	onClick  func()
	menu     []TrayMenuItem
	menuByID map[int32]*TrayMenuItem
	revision uint32
	closed   bool
}

func createTrayIcon() (TrayIcon, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTrayNotSupported, err)
	}
	var hasWatcher bool
	if err := conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, watcherName).Store(&hasWatcher); err != nil || !hasWatcher {
		conn.Close()
		return nil, ErrTrayNotSupported
	}

	t := &trayIcon{
		conn:     conn,
		busName:  fmt.Sprintf("org.kde.StatusNotifierItem-%d-%d", os.Getpid(), trayCounter.Add(1)),
		appName:  filepath.Base(os.Args[0]),
		menuByID: map[int32]*TrayMenuItem{},
	}
	if err := t.export(); err != nil {
		conn.Close()
		return nil, err
	}
	if reply, err := conn.RequestName(t.busName, dbus.NameFlagDoNotQueue); err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		conn.Close()
		return nil, fmt.Errorf("%w: can't own %s", ErrTrayNotSupported, t.busName)
	}
	if err := t.register(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("%w: %v", ErrTrayNotSupported, err)
	}
	t.watchTrayRestarts()
	return t, nil
}

// export puts the item and menu objects on the bus
func (t *trayIcon) export() error {
	var err error
	t.props, err = prop.Export(t.conn, sniPath, prop.Map{
		sniInterface: {
			"Category":            {Value: "ApplicationStatus", Emit: prop.EmitFalse},
			"Id":                  {Value: t.appName, Emit: prop.EmitFalse},
			"Title":               {Value: t.appName, Emit: prop.EmitFalse},
			"Status":              {Value: "Active", Emit: prop.EmitFalse},
			"WindowId":            {Value: int32(0), Emit: prop.EmitFalse},
			"IconName":            {Value: "", Emit: prop.EmitFalse},
			"IconPixmap":          {Value: []sniPixmap{}, Emit: prop.EmitFalse},
			"IconThemePath":       {Value: "", Emit: prop.EmitFalse},
			"OverlayIconName":     {Value: "", Emit: prop.EmitFalse},
			"OverlayIconPixmap":   {Value: []sniPixmap{}, Emit: prop.EmitFalse},
			"AttentionIconName":   {Value: "", Emit: prop.EmitFalse},
			"AttentionIconPixmap": {Value: []sniPixmap{}, Emit: prop.EmitFalse},
			"AttentionMovieName":  {Value: "", Emit: prop.EmitFalse},
			"ToolTip":             {Value: sniToolTip{IconPixmaps: []sniPixmap{}}, Emit: prop.EmitFalse},
			"ItemIsMenu":          {Value: false, Emit: prop.EmitFalse},
			"Menu":                {Value: menuPath, Emit: prop.EmitFalse},
		},
	})
	if err != nil {
		return err
	}
	if err := t.conn.Export(sniItem{t}, sniPath, sniInterface); err != nil {
		return err
	}
	if _, err := prop.Export(t.conn, menuPath, prop.Map{
		menuIface: {
			"Version":       {Value: uint32(3), Emit: prop.EmitFalse},
			"TextDirection": {Value: "ltr", Emit: prop.EmitFalse},
			"Status":        {Value: "normal", Emit: prop.EmitFalse},
			"IconThemePath": {Value: []string{}, Emit: prop.EmitFalse},
		},
	}); err != nil {
		return err
	}
	if err := t.conn.Export(dbusMenu{t}, menuPath, menuIface); err != nil {
		return err
	}

	t.conn.Export(introspect.Introspectable(sniIntrospection), sniPath, "org.freedesktop.DBus.Introspectable")
	t.conn.Export(introspect.Introspectable(menuIntrospection), menuPath, "org.freedesktop.DBus.Introspectable")
	return nil
}

func (t *trayIcon) register() error {
	return t.conn.Object(watcherName, watcherPath).Call(watcherName+".RegisterStatusNotifierItem", 0, t.busName).Err
}

// watchTrayRestarts registers the item again when the tray comes back, e.g.
// after the desktop shell restarted
func (t *trayIcon) watchTrayRestarts() {
	t.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchArg(0, watcherName),
	)
	signals := make(chan *dbus.Signal, 8)
	t.conn.Signal(signals)
	go func() {
		for s := range signals {
			if s.Name != "org.freedesktop.DBus.NameOwnerChanged" || len(s.Body) < 3 {
				continue
			}
			if newOwner, _ := s.Body[2].(string); newOwner != "" {
				t.register()
			}
		}
	}()
}

func (t *trayIcon) emit(member string, args ...any) {
	t.conn.Emit(sniPath, sniInterface+"."+member, args...)
}

func (t *trayIcon) SetIcon(icon *image.RGBA) {
	t.icon = icon
	t.props.SetMust(sniInterface, "IconPixmap", []sniPixmap{toSNIPixmap(icon)})
	t.emit("NewIcon")
}

func (t *trayIcon) SetTooltip(text string) {
	t.props.SetMust(sniInterface, "Title", text)
	t.props.SetMust(sniInterface, "ToolTip", sniToolTip{IconPixmaps: []sniPixmap{}, Title: text})
	t.emit("NewTitle")
	t.emit("NewToolTip")
}

func (t *trayIcon) OnClick(f func()) {
	t.mu.Lock()
	t.onClick = f
	t.mu.Unlock()
}

func (t *trayIcon) SetMenu(items []TrayMenuItem) {
	t.mu.Lock()
	t.menu = items
	t.menuByID = trayMenuIndex(items)
	t.revision++
	revision := t.revision
	t.mu.Unlock()
	t.conn.Emit(menuPath, menuIface+".LayoutUpdated", revision, int32(0))
}

// ShowNotification shows a desktop notification (org.freedesktop.Notifications)
// with the tray icon's image. A new notification replaces the previous one.
func (t *trayIcon) ShowNotification(title, text string) error {
	hints := map[string]dbus.Variant{}
	if t.icon != nil {
		b := t.icon.Bounds()
		pix := make([]byte, 0, b.Dx()*b.Dy()*4)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			i := t.icon.PixOffset(b.Min.X, y)
			pix = append(pix, t.icon.Pix[i:i+b.Dx()*4]...)
		}
		// (iiibiiay): width, height, rowstride, has alpha, bits per sample, channels, data
		hints["image-data"] = dbus.MakeVariant(struct {
			W, H, Stride   int32
			Alpha          bool
			Bits, Channels int32
			Data           []byte
		}{int32(b.Dx()), int32(b.Dy()), int32(b.Dx() * 4), true, 8, 4, pix})
	}
	obj := t.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call("org.freedesktop.Notifications.Notify", 0,
		t.appName, t.notifyID, "", title, text, []string{}, hints, int32(-1))
	if call.Err != nil {
		return fmt.Errorf("%w: %v", ErrNotificationsNotSupported, call.Err)
	}
	return call.Store(&t.notifyID)
}

func (t *trayIcon) Close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()
	// Leaving the bus removes the item from the tray
	t.conn.Close()
}

// toSNIPixmap converts the image to ARGB32 in network byte order
func toSNIPixmap(img *image.RGBA) sniPixmap {
	b := img.Bounds()
	data := make([]byte, 0, b.Dx()*b.Dy()*4)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := img.PixOffset(x, y)
			r, g, bl, a := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
			data = append(data, a, r, g, bl)
		}
	}
	return sniPixmap{Width: int32(b.Dx()), Height: int32(b.Dy()), Data: data}
}

// sniItem holds the org.kde.StatusNotifierItem methods
type sniItem struct{ t *trayIcon }

func (s sniItem) Activate(x, y int32) *dbus.Error {
	s.t.mu.Lock()
	f := s.t.onClick
	s.t.mu.Unlock()
	if f != nil {
		Post(f)
	}
	return nil
}

func (s sniItem) SecondaryActivate(x, y int32) *dbus.Error { return nil }

// ContextMenu is called by trays that don't show the Menu object themselves
func (s sniItem) ContextMenu(x, y int32) *dbus.Error { return nil }

func (s sniItem) Scroll(delta int32, orientation string) *dbus.Error { return nil }

// dbusMenu holds the com.canonical.dbusmenu methods
type dbusMenu struct{ t *trayIcon }

func (m dbusMenu) GetLayout(parentID, depth int32, names []string) (uint32, menuLayout, *dbus.Error) {
	m.t.mu.Lock()
	defer m.t.mu.Unlock()
	root := menuLayout{ID: 0, Props: map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}}
	var id int32 = 1
	root.Children = buildMenuLayout(m.t.menu, &id)
	if parentID != 0 {
		if found, ok := findMenuLayout(root, parentID); ok {
			return m.t.revision, found, nil
		}
		return m.t.revision, menuLayout{ID: parentID, Props: map[string]dbus.Variant{}}, nil
	}
	return m.t.revision, root, nil
}

// buildMenuLayout numbers the items the same way as trayMenuIndex
func buildMenuLayout(items []TrayMenuItem, id *int32) []dbus.Variant {
	children := []dbus.Variant{}
	for i := range items {
		item := &items[i]
		layout := menuLayout{ID: *id, Props: menuItemProperties(item)}
		*id++
		layout.Children = buildMenuLayout(item.Items, id)
		children = append(children, dbus.MakeVariant(layout))
	}
	return children
}

func findMenuLayout(l menuLayout, id int32) (menuLayout, bool) {
	if l.ID == id {
		return l, true
	}
	for _, child := range l.Children {
		if cl, ok := child.Value().(menuLayout); ok {
			if found, ok := findMenuLayout(cl, id); ok {
				return found, true
			}
		}
	}
	return menuLayout{}, false
}

func menuItemProperties(item *TrayMenuItem) map[string]dbus.Variant {
	if item.Separator {
		return map[string]dbus.Variant{"type": dbus.MakeVariant("separator")}
	}
	props := map[string]dbus.Variant{
		// An underscore marks a mnemonic in dbusmenu labels
		"label":   dbus.MakeVariant(strings.ReplaceAll(item.Text, "_", "__")),
		"enabled": dbus.MakeVariant(!item.Disabled),
		"visible": dbus.MakeVariant(true),
	}
	if item.Checkable {
		state := int32(0)
		if item.Checked {
			state = 1
		}
		props["toggle-type"] = dbus.MakeVariant("checkmark")
		props["toggle-state"] = dbus.MakeVariant(state)
	}
	if len(item.Items) > 0 {
		props["children-display"] = dbus.MakeVariant("submenu")
	}
	return props
}

func (m dbusMenu) GetGroupProperties(ids []int32, names []string) ([]menuItemProps, *dbus.Error) {
	m.t.mu.Lock()
	defer m.t.mu.Unlock()
	result := []menuItemProps{}
	for _, id := range ids {
		if item, ok := m.t.menuByID[id]; ok {
			result = append(result, menuItemProps{ID: id, Props: menuItemProperties(item)})
		}
	}
	return result, nil
}

func (m dbusMenu) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	m.t.mu.Lock()
	defer m.t.mu.Unlock()
	if item, ok := m.t.menuByID[id]; ok {
		if v, ok := menuItemProperties(item)[name]; ok {
			return v, nil
		}
	}
	return dbus.MakeVariant(""), nil
}

func (m dbusMenu) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventID != "clicked" {
		return nil
	}
	m.t.mu.Lock()
	item, ok := m.t.menuByID[id]
	m.t.mu.Unlock()
	if ok && item.OnClick != nil && !item.Disabled {
		Post(item.OnClick)
	}
	return nil
}

func (m dbusMenu) EventGroup(events []menuEvent) ([]int32, *dbus.Error) {
	for _, e := range events {
		m.Event(e.ID, e.EventID, e.Data, e.Timestamp)
	}
	return []int32{}, nil
}

func (m dbusMenu) AboutToShow(id int32) (bool, *dbus.Error) {
	return false, nil
}

func (m dbusMenu) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	return []int32{}, []int32{}, nil
}

const sniIntrospection = `<node>
	<interface name="org.kde.StatusNotifierItem">
		<method name="Activate"><arg name="x" type="i" direction="in"/><arg name="y" type="i" direction="in"/></method>
		<method name="SecondaryActivate"><arg name="x" type="i" direction="in"/><arg name="y" type="i" direction="in"/></method>
		<method name="ContextMenu"><arg name="x" type="i" direction="in"/><arg name="y" type="i" direction="in"/></method>
		<method name="Scroll"><arg name="delta" type="i" direction="in"/><arg name="orientation" type="s" direction="in"/></method>
		<signal name="NewTitle"/>
		<signal name="NewIcon"/>
		<signal name="NewAttentionIcon"/>
		<signal name="NewOverlayIcon"/>
		<signal name="NewToolTip"/>
		<signal name="NewStatus"><arg name="status" type="s"/></signal>
		<property name="Category" type="s" access="read"/>
		<property name="Id" type="s" access="read"/>
		<property name="Title" type="s" access="read"/>
		<property name="Status" type="s" access="read"/>
		<property name="WindowId" type="i" access="read"/>
		<property name="IconName" type="s" access="read"/>
		<property name="IconPixmap" type="a(iiay)" access="read"/>
		<property name="IconThemePath" type="s" access="read"/>
		<property name="OverlayIconName" type="s" access="read"/>
		<property name="OverlayIconPixmap" type="a(iiay)" access="read"/>
		<property name="AttentionIconName" type="s" access="read"/>
		<property name="AttentionIconPixmap" type="a(iiay)" access="read"/>
		<property name="AttentionMovieName" type="s" access="read"/>
		<property name="ToolTip" type="(sa(iiay)ss)" access="read"/>
		<property name="ItemIsMenu" type="b" access="read"/>
		<property name="Menu" type="o" access="read"/>
	</interface>` + introspect.IntrospectDataString + prop.IntrospectDataString + `</node>`

const menuIntrospection = `<node>
	<interface name="com.canonical.dbusmenu">
		<method name="GetLayout">
			<arg type="i" name="parentId" direction="in"/>
			<arg type="i" name="recursionDepth" direction="in"/>
			<arg type="as" name="propertyNames" direction="in"/>
			<arg type="u" name="revision" direction="out"/>
			<arg type="(ia{sv}av)" name="layout" direction="out"/>
		</method>
		<method name="GetGroupProperties">
			<arg type="ai" name="ids" direction="in"/>
			<arg type="as" name="propertyNames" direction="in"/>
			<arg type="a(ia{sv})" name="properties" direction="out"/>
		</method>
		<method name="GetProperty">
			<arg type="i" name="id" direction="in"/>
			<arg type="s" name="name" direction="in"/>
			<arg type="v" name="value" direction="out"/>
		</method>
		<method name="Event">
			<arg type="i" name="id" direction="in"/>
			<arg type="s" name="eventId" direction="in"/>
			<arg type="v" name="data" direction="in"/>
			<arg type="u" name="timestamp" direction="in"/>
		</method>
		<method name="EventGroup">
			<arg type="a(isvu)" name="events" direction="in"/>
			<arg type="ai" name="idErrors" direction="out"/>
		</method>
		<method name="AboutToShow">
			<arg type="i" name="id" direction="in"/>
			<arg type="b" name="needUpdate" direction="out"/>
		</method>
		<method name="AboutToShowGroup">
			<arg type="ai" name="ids" direction="in"/>
			<arg type="ai" name="updatesNeeded" direction="out"/>
			<arg type="ai" name="idErrors" direction="out"/>
		</method>
		<signal name="ItemsPropertiesUpdated">
			<arg type="a(ia{sv})" name="updatedProps"/>
			<arg type="a(ias)" name="removedProps"/>
		</signal>
		<signal name="LayoutUpdated">
			<arg type="u" name="revision"/>
			<arg type="i" name="parent"/>
		</signal>
		<property name="Version" type="u" access="read"/>
		<property name="TextDirection" type="s" access="read"/>
		<property name="Status" type="s" access="read"/>
		<property name="IconThemePath" type="as" access="read"/>
	</interface>` + introspect.IntrospectDataString + prop.IntrospectDataString + `</node>`
