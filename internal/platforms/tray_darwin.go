package platforms

import (
	"image"
	"image/draw"
	"sync"

	"github.com/ebitengine/purego/objc"
)

// The tray icon is an NSStatusItem in the menu bar. With a menu, a click on
// it opens the menu (the usual on macOS); without one it calls OnClick. The
// clicks come to an NUITrayTarget object on the main thread, the UI thread.

var (
	selSystemStatusBar                  = objc.RegisterName("systemStatusBar")
	selStatusItemWithLength             = objc.RegisterName("statusItemWithLength:")
	selRemoveStatusItem                 = objc.RegisterName("removeStatusItem:")
	selButton                           = objc.RegisterName("button")
	selSetImage                         = objc.RegisterName("setImage:")
	selSetToolTip                       = objc.RegisterName("setToolTip:")
	selSetMenu                          = objc.RegisterName("setMenu:")
	selSetTarget                        = objc.RegisterName("setTarget:")
	selSetAction                        = objc.RegisterName("setAction:")
	selInitWithTitle                    = objc.RegisterName("initWithTitle:")
	selInitWithTitleActionKeyEquivalent = objc.RegisterName("initWithTitle:action:keyEquivalent:")
	selSetAutoenablesItems              = objc.RegisterName("setAutoenablesItems:")
	selAddItem                          = objc.RegisterName("addItem:")
	selSeparatorItem                    = objc.RegisterName("separatorItem")
	selSetSubmenu                       = objc.RegisterName("setSubmenu:")
	selSetTag                           = objc.RegisterName("setTag:")
	selTag                              = objc.RegisterName("tag")
	selSetEnabled                       = objc.RegisterName("setEnabled:")
	selSetState                         = objc.RegisterName("setState:")
	selNuiTrayClicked                   = objc.RegisterName("nuiTrayClicked:")
	selNuiTrayMenuItemClicked           = objc.RegisterName("nuiTrayMenuItemClicked:")
)

const (
	nsVariableStatusItemLength = -1.0
	nsControlStateValueOn      = 1
	// The menu bar icon size in points
	trayIconPoints = 18
)

var (
	trayTargetClassOnce sync.Once
	trayTargetClass     objc.Class
	// trayByTarget finds the icon of a click
	trayByTarget = map[objc.ID]*trayIcon{}
)

type trayIcon struct {
	item     objc.ID
	target   objc.ID
	onClick  func()
	menuByID map[int32]*TrayMenuItem
}

func registerTrayTargetClass() {
	var err error
	trayTargetClass, err = objc.RegisterClass(
		"NUITrayTarget",
		objc.GetClass("NSObject"),
		nil, nil,
		[]objc.MethodDef{
			{Cmd: selNuiTrayClicked, Fn: nuiTrayClicked},
			{Cmd: selNuiTrayMenuItemClicked, Fn: nuiTrayMenuItemClicked},
		},
	)
	if err != nil {
		panic(err)
	}
}

func nuiTrayClicked(self objc.ID, _ objc.SEL, _ objc.ID) {
	if t, ok := trayByTarget[self]; ok && t.onClick != nil {
		t.onClick()
	}
}

func nuiTrayMenuItemClicked(self objc.ID, _ objc.SEL, sender objc.ID) {
	t, ok := trayByTarget[self]
	if !ok {
		return
	}
	id := int32(objc.Send[int](sender, selTag))
	if item, ok := t.menuByID[id]; ok && item.OnClick != nil && !item.Disabled {
		item.OnClick()
	}
}

func createTrayIcon() (TrayIcon, error) {
	trayTargetClassOnce.Do(registerTrayTargetClass)
	bar := objc.ID(objc.GetClass("NSStatusBar")).Send(selSystemStatusBar)
	if bar == 0 {
		return nil, ErrTrayNotSupported
	}
	t := &trayIcon{}
	t.item = bar.Send(selStatusItemWithLength, float64(nsVariableStatusItemLength))
	t.item.Send(selRetain)
	t.target = objc.ID(trayTargetClass).Send(selAlloc).Send(selInit)
	trayByTarget[t.target] = t

	button := objc.Send[objc.ID](t.item, selButton)
	button.Send(selSetTarget, t.target)
	button.Send(selSetAction, selNuiTrayClicked)
	return t, nil
}

func (t *trayIcon) SetIcon(icon *image.RGBA) {
	b := icon.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), icon, b.Min, draw.Src)
	withAutoreleasePool(func() {
		img := newNSImageFromRGBA(rgba.Pix, b.Dx(), b.Dy(), trayIconPoints, trayIconPoints)
		if img == 0 {
			return
		}
		objc.Send[objc.ID](t.item, selButton).Send(selSetImage, img)
		img.Send(selRelease)
	})
}

func (t *trayIcon) SetTooltip(text string) {
	withAutoreleasePool(func() {
		objc.Send[objc.ID](t.item, selButton).Send(selSetToolTip, goStringToNS(text))
	})
}

func (t *trayIcon) OnClick(f func()) {
	t.onClick = f
}

func (t *trayIcon) SetMenu(items []TrayMenuItem) {
	t.menuByID = trayMenuIndex(items)
	if len(items) == 0 {
		t.item.Send(selSetMenu, objc.ID(0))
		return
	}
	withAutoreleasePool(func() {
		var id int32 = 1
		menu := t.buildMenu(items, &id)
		t.item.Send(selSetMenu, menu)
		menu.Send(selRelease)
	})
}

// buildMenu makes an NSMenu (+1 retained); the items are numbered the same
// way as trayMenuIndex, the number is the item's tag
func (t *trayIcon) buildMenu(items []TrayMenuItem, id *int32) objc.ID {
	menu := objc.ID(objc.GetClass("NSMenu")).Send(selAlloc).Send(selInitWithTitle, goStringToNS(""))
	menu.Send(selSetAutoenablesItems, false)
	for i := range items {
		item := &items[i]
		itemID := *id
		*id++
		if item.Separator {
			menu.Send(selAddItem, objc.ID(objc.GetClass("NSMenuItem")).Send(selSeparatorItem))
			continue
		}
		mi := objc.ID(objc.GetClass("NSMenuItem")).Send(selAlloc).Send(selInitWithTitleActionKeyEquivalent,
			goStringToNS(item.Text), selNuiTrayMenuItemClicked, goStringToNS(""))
		mi.Send(selSetTarget, t.target)
		mi.Send(selSetTag, int(itemID))
		mi.Send(selSetEnabled, !item.Disabled)
		if item.Checkable && item.Checked {
			mi.Send(selSetState, nsControlStateValueOn)
		}
		if len(item.Items) > 0 {
			submenu := t.buildMenu(item.Items, id)
			mi.Send(selSetSubmenu, submenu)
			submenu.Send(selRelease)
		}
		menu.Send(selAddItem, mi)
		mi.Send(selRelease)
	}
	return menu
}

// ShowNotification isn't available: macOS shows notifications only from
// signed application bundles (UNUserNotificationCenter)
func (t *trayIcon) ShowNotification(title, text string) error {
	return ErrNotificationsNotSupported
}

func (t *trayIcon) Close() {
	if t.item == 0 {
		return
	}
	objc.ID(objc.GetClass("NSStatusBar")).Send(selSystemStatusBar).Send(selRemoveStatusItem, t.item)
	t.item.Send(selRelease)
	delete(trayByTarget, t.target)
	t.target.Send(selRelease)
	t.item = 0
}
