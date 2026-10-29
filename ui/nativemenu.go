package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ipoluianov/nui/internal/platforms"
)

// The system menu bar (macOS): with UseNativeMenuBar(true), the menu bar of
// the active form goes to the top of the screen, after the application menu
// (Hide, Quit), and the form shows none of its own. The menus stay the
// form's: their items, actions, check marks, shortcuts, OnShow and the
// enabled functions of the actions work as in the form's own bar.

var nativeMenuBars bool

// nativeMenuOwner is the form whose menus are in the system menu bar now
var nativeMenuOwner *Form

// UseNativeMenuBar puts the menu bars of the forms in the system's menu
// bar where there is one (macOS); elsewhere the forms keep their own
func UseNativeMenuBar(use bool) {
	nativeMenuBars = use
}

// installNativeMenuBar gives the form's menu bar to the system, if wanted
// and possible: the bar takes no room in the form then. A form without a
// menu bar leaves the application menu alone there: the menus of the form
// active before mustn't stay, they would act on it.
func (c *Form) installNativeMenuBar() {
	if !nativeMenuBars || c.wnd == nil {
		return
	}
	nativeMenuOwner = c
	if c.menuBar == nil {
		platforms.SetNativeMenuBar(applicationName(), nil)
		return
	}
	var menus []*platforms.NativeMenu
	for _, item := range c.menuBar.Items() {
		if !item.IsVisible() {
			continue
		}
		menus = append(menus, nativeMenuOf(stripMnemonic(item.Text()), item.Menu()))
	}
	if platforms.SetNativeMenuBar(applicationName(), menus) && !c.menuBar.native {
		c.menuBar.native = true
		c.menuBar.SetVisible(false)
		c.UpdateLayout()
	}
}

// releaseNativeMenuBar takes the menus of the closing form out of the
// system menu bar, if they are there: the application menu stays until the
// form active next puts its own
func (c *Form) releaseNativeMenuBar() {
	if nativeMenuOwner != c {
		return
	}
	nativeMenuOwner = nil
	platforms.SetNativeMenuBar(applicationName(), nil)
}

// nativeMenuOf is the menu for the system: its items are taken anew every
// time it opens
func nativeMenuOf(title string, menu *ContextMenu) *platforms.NativeMenu {
	return &platforms.NativeMenu{Title: title, Items: func() []platforms.NativeMenuItem {
		if menu.onShow != nil {
			menu.onShow()
		}
		menu.updateActions()
		var items []platforms.NativeMenuItem
		for _, item := range menu.items {
			if !item.IsVisible() {
				continue
			}
			if item.separator {
				items = append(items, platforms.NativeMenuItem{Separator: true})
				continue
			}
			n := platforms.NativeMenuItem{
				Title:    stripMnemonic(item.text),
				Disabled: !item.Enabled(),
				Checked:  item.checkable && item.checked,
			}
			if item.innerMenu != nil {
				n.Submenu = nativeMenuOf(n.Title, item.innerMenu)
			} else {
				n.OnClick = func() {
					item.trigger()
					if f := item.Form(); f != nil {
						f.Update()
					}
				}
			}
			n.KeyEquivalent = keyEquivalent(item.shortcut.Key)
			if n.KeyEquivalent != "" {
				s := item.shortcut
				n.Shift, n.Ctrl, n.Alt, n.Cmd = s.Shift, s.Ctrl, s.Alt, s.Cmd
			}
			items = append(items, n)
		}
		return items
	}}
}

// keyEquivalent is the key as macOS names it in a menu's shortcut
func keyEquivalent(key Key) string {
	switch {
	case key >= KeyA && key <= KeyZ:
		return string(rune('a' + key - KeyA))
	case key >= Key0 && key <= Key9:
		return string(rune('0' + key - Key0))
	case key >= KeyF1 && key <= KeyF12:
		return string(rune(0xF704 + key - KeyF1)) // NSF1FunctionKey...
	}
	switch key {
	case KeyEnter:
		return "\r"
	case KeyEsc:
		return "\x1b"
	case KeyTab:
		return "\t"
	case KeySpace:
		return " "
	case KeyBackspace:
		return "\x08"
	case KeyDelete:
		return string(rune(0xF728)) // NSDeleteFunctionKey
	case KeyArrowUp:
		return string(rune(0xF700))
	case KeyArrowDown:
		return string(rune(0xF701))
	case KeyArrowLeft:
		return string(rune(0xF702))
	case KeyArrowRight:
		return string(rune(0xF703))
	case KeyHome:
		return string(rune(0xF729))
	case KeyEnd:
		return string(rune(0xF72B))
	case KeyPageUp:
		return string(rune(0xF72C))
	case KeyPageDown:
		return string(rune(0xF72D))
	case KeyMinus, KeyNumpadMinus:
		return "-"
	case KeyEqual:
		return "="
	case KeyNumpadPlus:
		return "+"
	case KeyComma:
		return ","
	case KeyDot:
		return "."
	case KeySlash:
		return "/"
	}
	return ""
}

// stripMnemonic removes the "&" of a mnemonic ("&&" is an "&")
func stripMnemonic(text string) string {
	if !strings.Contains(text, "&") {
		return text
	}
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(text, "&&", "\x00"), "&", ""), "\x00", "&")
}

// applicationName is the name of the program, for the application menu
func applicationName() string {
	exe, err := os.Executable()
	if err != nil {
		return "Application"
	}
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
}
