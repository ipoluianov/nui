package platforms

// The main menu in the system's menu bar (macOS): the application gives
// the titles and, for each, a function returning the items, asked again
// every time the menu opens (and before its shortcuts are looked up), so
// the items follow the state of the application.

// NativeMenu is a menu of the bar, or a submenu
type NativeMenu struct {
	Title string
	Items func() []NativeMenuItem
}

// NativeMenuItem is an item of a NativeMenu
type NativeMenuItem struct {
	Title     string
	Separator bool
	Disabled  bool
	Checked   bool
	// KeyEquivalent is the key of the shortcut as macOS names it: a letter
	// (lower case), a digit, a sign, or a character of a function key
	// (NSF1FunctionKey...); "" if none
	KeyEquivalent         string
	Shift, Ctrl, Alt, Cmd bool
	Submenu               *NativeMenu
	OnClick               func()
}

// SetNativeMenuBar puts the menus in the system's menu bar, after the
// application menu (Hide, Quit...) named appName; nil menus leave the
// application menu only. Returns false where there is no system menu bar
// (Windows, Linux): the forms show their own then.
func SetNativeMenuBar(appName string, menus []*NativeMenu) bool {
	if IsHeadless() {
		return false
	}
	ok := false
	RunOnUIThread(func() { ok = setNativeMenuBar(appName, menus) })
	return ok
}
