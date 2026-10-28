//go:build darwin

package platforms

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ebitengine/purego/objc"
)

// The menu bar of macOS: NSApp's main menu. The first menu is the
// application menu; each of the others is an NSMenu whose delegate (an
// NUIMenuTarget) fills it from NativeMenu.Items in menuNeedsUpdate:, which
// AppKit calls before showing the menu and before matching its key
// equivalents. The items' action comes to the same target; the item's tag
// is its index in the items of its menu.

var (
	selSetMainMenu                  = objc.RegisterName("setMainMenu:")
	selRemoveAllItems               = objc.RegisterName("removeAllItems")
	selSetKeyEquivalentModifierMask = objc.RegisterName("setKeyEquivalentModifierMask:")
	selMenu                         = objc.RegisterName("menu")
	selHide                         = objc.RegisterName("hide:")
	selHideOtherApplications        = objc.RegisterName("hideOtherApplications:")
	selUnhideAllApplications        = objc.RegisterName("unhideAllApplications:")
	selMenuNeedsUpdate              = objc.RegisterName("menuNeedsUpdate:")
	selNuiMenuItemClicked           = objc.RegisterName("nuiMenuItemClicked:")
	selNuiQuit                      = objc.RegisterName("nuiQuit:")
	selAutorelease                  = objc.RegisterName("autorelease")
)

var (
	menuTargetClassOnce sync.Once
	menuTargetClass     objc.Class
	menuTarget          objc.ID
	// nativeMenus are the menus by their NSMenu, and the items each shows now
	nativeMenus = map[objc.ID]*nativeMenuState{}
	mainMenu    objc.ID
)

type nativeMenuState struct {
	menu  *NativeMenu
	items []NativeMenuItem
	// submenus are the NSMenus of the items, forgotten when it is filled again
	submenus []objc.ID
}

func registerMenuTargetClass() {
	var err error
	menuTargetClass, err = objc.RegisterClass(
		"NUIMenuTarget",
		objc.GetClass("NSObject"),
		nil, nil,
		[]objc.MethodDef{
			{Cmd: selMenuNeedsUpdate, Fn: nuiMenuNeedsUpdate},
			{Cmd: selNuiMenuItemClicked, Fn: nuiMenuItemClicked},
			{Cmd: selNuiQuit, Fn: nuiQuit},
		},
	)
	if err != nil {
		panic(err)
	}
	menuTarget = objc.ID(menuTargetClass).Send(selAlloc).Send(selInit)
}

func setNativeMenuBar(appName string, menus []*NativeMenu) bool {
	menuTargetClassOnce.Do(registerMenuTargetClass)
	withAutoreleasePool(func() {
		// The menus of the old bar go with it
		clear(nativeMenus)
		bar := newNSMenu("")
		appItem := newNSMenuItem("", 0, "")
		app := newNSMenu(appName)
		app.Send(selAddItem, newNSMenuItem("Hide "+appName, selHide, "h"))
		others := newNSMenuItem("Hide Others", selHideOtherApplications, "h")
		others.Send(selSetKeyEquivalentModifierMask, uint(nsEventModifierFlagCommand|nsEventModifierFlagOption))
		app.Send(selAddItem, others)
		app.Send(selAddItem, newNSMenuItem("Show All", selUnhideAllApplications, ""))
		app.Send(selAddItem, objc.ID(objc.GetClass("NSMenuItem")).Send(selSeparatorItem))
		quit := newNSMenuItem("Quit "+appName, selNuiQuit, "q")
		quit.Send(selSetTarget, menuTarget)
		app.Send(selAddItem, quit)
		appItem.Send(selSetSubmenu, app)
		bar.Send(selAddItem, appItem)

		for _, m := range menus {
			item := newNSMenuItem(m.Title, 0, "")
			item.Send(selSetSubmenu, newDynamicMenu(m))
			bar.Send(selAddItem, item)
		}
		nsApp := objc.ID(clsNSApplication).Send(selSharedApplication)
		nsApp.Send(selSetMainMenu, bar)
		if mainMenu != 0 {
			mainMenu.Send(selRelease)
		}
		mainMenu = bar.Send(selRetain)
	})
	return true
}

// executableName is the name of the application in its menu: the name of
// the program file
func executableName() string {
	exe, err := os.Executable()
	if err != nil {
		return "Application"
	}
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
}

// newDynamicMenu makes the NSMenu of a NativeMenu, filled now (for its
// key equivalents) and again before every use
func newDynamicMenu(m *NativeMenu) objc.ID {
	menu := newNSMenu(m.Title)
	menu.Send(selSetAutoenablesItems, false)
	menu.Send(selSetDelegate, menuTarget)
	state := &nativeMenuState{menu: m}
	nativeMenus[menu] = state
	fillNativeMenu(menu, state)
	return menu
}

// fillNativeMenu puts the items of the menu now into its NSMenu
func fillNativeMenu(menu objc.ID, state *nativeMenuState) {
	menu.Send(selRemoveAllItems)
	for _, sub := range state.submenus {
		forgetNativeMenu(sub)
	}
	state.items, state.submenus = nil, nil
	if state.menu.Items != nil {
		state.items = state.menu.Items()
	}
	for i, it := range state.items {
		if it.Separator {
			menu.Send(selAddItem, objc.ID(objc.GetClass("NSMenuItem")).Send(selSeparatorItem))
			continue
		}
		item := newNSMenuItem(it.Title, selNuiMenuItemClicked, it.KeyEquivalent)
		item.Send(selSetTarget, menuTarget)
		item.Send(selSetTag, i)
		item.Send(selSetEnabled, !it.Disabled)
		if it.KeyEquivalent != "" {
			var mask uint
			if it.Cmd {
				mask |= nsEventModifierFlagCommand
			}
			if it.Shift {
				mask |= nsEventModifierFlagShift
			}
			if it.Alt {
				mask |= nsEventModifierFlagOption
			}
			if it.Ctrl {
				mask |= nsEventModifierFlagControl
			}
			item.Send(selSetKeyEquivalentModifierMask, mask)
		}
		if it.Checked {
			item.Send(selSetState, nsControlStateValueOn)
		}
		if it.Submenu != nil {
			sub := newDynamicMenu(it.Submenu)
			state.submenus = append(state.submenus, sub)
			item.Send(selSetSubmenu, sub)
		}
		menu.Send(selAddItem, item)
	}
}

// forgetNativeMenu removes the menu and its submenus from nativeMenus
func forgetNativeMenu(menu objc.ID) {
	if state, ok := nativeMenus[menu]; ok {
		delete(nativeMenus, menu)
		for _, sub := range state.submenus {
			forgetNativeMenu(sub)
		}
	}
}

// newNSMenu makes an autoreleased NSMenu
func newNSMenu(title string) objc.ID {
	menu := objc.ID(objc.GetClass("NSMenu")).Send(selAlloc).Send(selInitWithTitle, goStringToNS(title))
	return menu.Send(selAutorelease)
}

// newNSMenuItem makes an autoreleased NSMenuItem
func newNSMenuItem(title string, action objc.SEL, key string) objc.ID {
	item := objc.ID(objc.GetClass("NSMenuItem")).Send(selAlloc).Send(selInitWithTitleActionKeyEquivalent,
		goStringToNS(title), action, goStringToNS(key))
	return item.Send(selAutorelease)
}

// nuiMenuNeedsUpdate fills the menu again before it is shown
func nuiMenuNeedsUpdate(self objc.ID, _ objc.SEL, menu objc.ID) {
	if state, ok := nativeMenus[menu]; ok {
		fillNativeMenu(menu, state)
	}
}

// nuiMenuItemClicked runs the item of the menu by its tag
func nuiMenuItemClicked(self objc.ID, _ objc.SEL, sender objc.ID) {
	state, ok := nativeMenus[objc.Send[objc.ID](sender, selMenu)]
	if !ok {
		return
	}
	i := objc.Send[int](sender, selTag)
	if i >= 0 && i < len(state.items) && state.items[i].OnClick != nil && !state.items[i].Disabled {
		state.items[i].OnClick()
	}
}

// nuiQuit is Quit in the application menu: see requestQuit
func nuiQuit(self objc.ID, _ objc.SEL, _ objc.ID) {
	requestQuit()
}
