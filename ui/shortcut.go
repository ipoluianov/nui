package ui

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Shortcut is a key with modifiers, e.g. Ctrl+S. It is matched by the key's
// place on the keyboard, so Ctrl+S works in any keyboard layout.
type Shortcut struct {
	Key                   Key
	Ctrl, Shift, Alt, Cmd bool
}

// shortcutKeys are the key names ParseShortcut takes, upper-cased
var shortcutKeys = func() map[string]Key {
	m := map[string]Key{}
	for _, k := range []Key{
		KeyA, KeyB, KeyC, KeyD, KeyE, KeyF, KeyG, KeyH, KeyI, KeyJ, KeyK, KeyL, KeyM,
		KeyN, KeyO, KeyP, KeyQ, KeyR, KeyS, KeyT, KeyU, KeyV, KeyW, KeyX, KeyY, KeyZ,
		Key0, Key1, Key2, Key3, Key4, Key5, Key6, Key7, Key8, Key9,
		KeyF1, KeyF2, KeyF3, KeyF4, KeyF5, KeyF6, KeyF7, KeyF8, KeyF9, KeyF10, KeyF11, KeyF12,
		KeyEsc, KeyTab, KeyEnter, KeySpace, KeyBackspace, KeyDelete, KeyInsert,
		KeyHome, KeyEnd, KeyPageUp, KeyPageDown,
		KeyArrowUp, KeyArrowDown, KeyArrowLeft, KeyArrowRight,
		KeyMinus, KeyEqual, KeyComma, KeyDot, KeySlash, KeySemicolon, KeyApostrophe,
		KeyLeftBracket, KeyRightBracket, KeyBackslash, KeyGrave,
	} {
		m[strings.ToUpper(k.String())] = k
	}
	for name, k := range map[string]Key{
		"ESCAPE": KeyEsc, "RETURN": KeyEnter, "DEL": KeyDelete, "INS": KeyInsert,
		"PGUP": KeyPageUp, "PGDN": KeyPageDown, "UP": KeyArrowUp, "DOWN": KeyArrowDown,
		"LEFT": KeyArrowLeft, "RIGHT": KeyArrowRight, "PLUS": KeyEqual, "MINUS": KeyMinus,
		// The keys of the numeric keypad
		"NUMPLUS": KeyNumpadPlus, "NUMMINUS": KeyNumpadMinus, "NUMMULTIPLY": KeyNumpadAsterisk,
		"NUMSTAR": KeyNumpadAsterisk, "NUMDIVIDE": KeyNumpadSlash, "NUMSLASH": KeyNumpadSlash,
		// The punctuation keys by their characters
		`\`: KeyBackslash, "/": KeySlash, ",": KeyComma, ".": KeyDot, ";": KeySemicolon,
		"'": KeyApostrophe, "[": KeyLeftBracket, "]": KeyRightBracket, "`": KeyGrave,
	} {
		m[name] = k
	}
	return m
}()

// punctuationKeys are shown as their characters
var punctuationKeys = map[Key]string{
	KeyBackslash: `\`, KeySlash: "/", KeyComma: ",", KeyDot: ".", KeySemicolon: ";",
	KeyApostrophe: "'", KeyLeftBracket: "[", KeyRightBracket: "]", KeyGrave: "`",
}

// ParseShortcut parses a shortcut like "Ctrl+S", "Ctrl+Shift+Z", "Alt+F4",
// "F5" or "Delete". "Mod" is Cmd on macOS and Ctrl elsewhere: "Mod+S" saves
// with the usual key of each system.
func ParseShortcut(s string) (Shortcut, error) {
	var sc Shortcut
	parts := strings.Split(s, "+")
	// "Ctrl++": the key is "+"
	if strings.HasSuffix(s, "++") {
		parts = append(strings.Split(strings.TrimSuffix(s, "++"), "+"), "PLUS")
	}
	for i, part := range parts {
		name := strings.ToUpper(strings.TrimSpace(part))
		if i < len(parts)-1 {
			switch name {
			case "CTRL", "CONTROL":
				sc.Ctrl = true
			case "SHIFT":
				sc.Shift = true
			case "ALT", "OPTION":
				sc.Alt = true
			case "CMD", "COMMAND", "META":
				sc.Cmd = true
			case "MOD":
				if runtime.GOOS == "darwin" {
					sc.Cmd = true
				} else {
					sc.Ctrl = true
				}
			default:
				return Shortcut{}, fmt.Errorf("shortcut %q: unknown modifier %q", s, part)
			}
			continue
		}
		key, ok := shortcutKeys[name]
		if !ok {
			return Shortcut{}, fmt.Errorf("shortcut %q: unknown key %q", s, part)
		}
		sc.Key = key
	}
	return sc, nil
}

// MustParseShortcut is ParseShortcut that panics on a wrong shortcut
func MustParseShortcut(s string) Shortcut {
	sc, err := ParseShortcut(s)
	if err != nil {
		panic(err)
	}
	return sc
}

func (s Shortcut) IsZero() bool {
	return s.Key == 0
}

// Matches reports whether the key with exactly these modifiers is the shortcut
func (s Shortcut) Matches(key Key, mods KeyModifiers) bool {
	return s.Key != 0 && s.Key == key && s.Ctrl == mods.Ctrl && s.Shift == mods.Shift &&
		s.Alt == mods.Alt && s.Cmd == mods.Cmd
}

// String is the shortcut as menus show it: "Ctrl+Shift+S", "⇧⌘S" on macOS
func (s Shortcut) String() string {
	if s.Key == 0 {
		return ""
	}
	key := s.Key.String()
	switch s.Key {
	case KeyArrowUp:
		key = "Up"
	case KeyArrowDown:
		key = "Down"
	case KeyArrowLeft:
		key = "Left"
	case KeyArrowRight:
		key = "Right"
	case KeyEqual:
		key = "+"
	case KeyNumpadPlus:
		key = "Num +"
	case KeyNumpadMinus:
		key = "Num -"
	case KeyNumpadAsterisk:
		key = "Num *"
	case KeyNumpadSlash:
		key = "Num /"
	default:
		if ch, ok := punctuationKeys[s.Key]; ok {
			key = ch
		}
	}
	if runtime.GOOS == "darwin" {
		var b strings.Builder
		for _, m := range []struct {
			on  bool
			sym string
		}{{s.Ctrl, "⌃"}, {s.Alt, "⌥"}, {s.Shift, "⇧"}, {s.Cmd, "⌘"}} {
			if m.on {
				b.WriteString(m.sym)
			}
		}
		return b.String() + key
	}
	var parts []string
	for _, m := range []struct {
		on   bool
		name string
	}{{s.Ctrl, "Ctrl"}, {s.Alt, "Alt"}, {s.Shift, "Shift"}, {s.Cmd, "Cmd"}} {
		if m.on {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, key), "+")
}

// Mnemonics. "&File" is shown as "File" with F underlined: Alt+F opens the
// menu, or F chooses the item in an open menu. "&&" is a "&".

// parseMnemonic returns the text to show, the mnemonic letter (lower case, 0
// if none) and the index (in runes) of the underlined letter, -1 if none
func parseMnemonic(text string) (shown string, mnemonic rune, index int) {
	if !strings.Contains(text, "&") {
		return text, 0, -1
	}
	var b strings.Builder
	index = -1
	runes := []rune(text)
	n := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '&' && i+1 < len(runes) {
			next := runes[i+1]
			if next == '&' {
				b.WriteRune('&')
				n++
				i++
				continue
			}
			if mnemonic == 0 && (unicode.IsLetter(next) || unicode.IsDigit(next)) {
				mnemonic = unicode.ToLower(next)
				index = n
				continue
			}
		}
		b.WriteRune(r)
		n++
	}
	return b.String(), mnemonic, index
}

// mnemonicOfKey returns the letter or digit of the key, 0 if it has none
func mnemonicOfKey(key Key) rune {
	name := key.String()
	if utf8.RuneCountInString(name) != 1 {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(name)
	if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
		return 0
	}
	return unicode.ToLower(r)
}

// drawMnemonicUnderline underlines the letter at index of text drawn with
// the canvas' font and alignment in (x, y, w, h)
func drawMnemonicUnderline(cnv *Canvas, x, y, w, h int, text string, index int) {
	if index < 0 {
		return
	}
	family, size := cnv.state.fontFamily, cnv.state.fontSize
	positions, err := GetCharPositions(family, size, text)
	if err != nil || index+1 >= len(positions) {
		return
	}
	textWidth := positions[len(positions)-1]
	_, textHeight, _ := MeasureText(family, size, "Ag")
	left := x
	switch cnv.state.hAlign {
	case HAlignCenter:
		left = x + w/2 - textWidth/2
	case HAlignRight:
		left = x + w - textWidth
	}
	top := y
	switch cnv.state.vAlign {
	case VAlignCenter:
		top = y + h/2 - textHeight/2
	case VAlignBottom:
		top = y + h - textHeight
	}
	lineY := top + textHeight - max(textHeight/6, 2)
	thickness := max(int(size/14), 1)
	cnv.FillRect(left+positions[index], lineY, positions[index+1]-positions[index], thickness, cnv.state.col)
}

// Form shortcuts

type formShortcut struct {
	shortcut Shortcut
	f        func()
}

// AddShortcut runs f when the shortcut is pressed in the form, whatever widget
// has the focus (except the keys a focused text field uses itself, like
// Ctrl+C). Panics on a wrong shortcut, see ParseShortcut.
//
//	form.AddShortcut("F5", refresh)
//	form.AddShortcut("Mod+F", func() { search.Focus() })
func (c *Form) AddShortcut(shortcut string, f func()) {
	sc := MustParseShortcut(shortcut)
	c.RemoveShortcut(shortcut)
	c.shortcuts = append(c.shortcuts, formShortcut{sc, f})
}

// RemoveShortcut removes a shortcut added by AddShortcut
func (c *Form) RemoveShortcut(shortcut string) {
	sc, err := ParseShortcut(shortcut)
	if err != nil {
		return
	}
	for i, s := range c.shortcuts {
		if s.shortcut == sc {
			c.shortcuts = append(c.shortcuts[:i], c.shortcuts[i+1:]...)
			return
		}
	}
}

// keyHandler is implemented by the widgets that take some keys before the
// shortcuts, e.g. a text field takes Ctrl+C and Delete
type keyHandler interface {
	handlesKey(key Key, mods KeyModifiers) bool
}

// textEditingKey reports whether a text field uses the key itself
func textEditingKey(key Key, mods KeyModifiers) bool {
	if mods.Alt {
		return false
	}
	ctrl := mods.Ctrl || mods.Cmd
	// A key that types a character is the text box's: a shortcut like "." or
	// "Num +" must not fire while the user types it
	if !ctrl && typingKey(key) {
		return true
	}
	switch key {
	case KeyA, KeyC, KeyV, KeyX, KeyZ, KeyY:
		return ctrl
	case KeyArrowLeft, KeyArrowRight, KeyArrowUp, KeyArrowDown, KeyHome, KeyEnd,
		KeyBackspace, KeyDelete:
		return true
	}
	return false
}

// typingKey: the keys that type a character (without Ctrl, Alt, Cmd)
func typingKey(key Key) bool {
	switch {
	case key >= KeyA && key <= KeyZ, key >= Key0 && key <= Key9,
		key >= KeyNumpad0 && key <= KeyNumpad9:
		return true
	}
	switch key {
	case KeySpace, KeyMinus, KeyEqual, KeyComma, KeyDot, KeySlash, KeySemicolon,
		KeyApostrophe, KeyLeftBracket, KeyRightBracket, KeyBackslash, KeyGrave,
		KeyNumpadPlus, KeyNumpadMinus, KeyNumpadAsterisk, KeyNumpadSlash, KeyNumpadDot:
		return true
	}
	return false
}

// processShortcut runs the shortcut of the key: the form's, then the main
// menu's items. Returns true if there was one.
func (c *Form) processShortcut(key Key, mods KeyModifiers) bool {
	if handler, ok := c.focusedWidget.(keyHandler); ok && handler.handlesKey(key, mods) {
		return false
	}
	for _, s := range c.shortcuts {
		if s.shortcut.Matches(key, mods) && s.f != nil {
			s.f()
			c.Update()
			return true
		}
	}
	if c.menuBar != nil {
		for _, title := range c.menuBar.items {
			if !title.IsVisible() || !title.Enabled() || title.menu == nil {
				continue
			}
			if item := title.menu.itemWithShortcut(key, mods); item != nil {
				if item.OnClick != nil {
					item.OnClick()
				}
				c.Update()
				return true
			}
		}
	}
	return false
}

// itemWithShortcut finds the enabled item of the menu or its submenus with
// the shortcut
func (c *ContextMenu) itemWithShortcut(key Key, mods KeyModifiers) *ContextMenuItem {
	for _, item := range c.items {
		if item.separator || !item.IsVisible() || !item.Enabled() {
			continue
		}
		if item.innerMenu != nil {
			if found := item.innerMenu.itemWithShortcut(key, mods); found != nil {
				return found
			}
			continue
		}
		if item.shortcut.Matches(key, mods) {
			return item
		}
	}
	return nil
}

// processMenuBarMnemonic opens the main menu whose title has the mnemonic of
// Alt+key. Returns true if there was one.
func (c *Form) processMenuBarMnemonic(key Key, mods KeyModifiers) bool {
	if c.menuBar == nil || !mods.Alt || mods.Ctrl || mods.Cmd {
		return false
	}
	m := mnemonicOfKey(key)
	if m == 0 {
		return false
	}
	for _, title := range c.menuBar.items {
		if title.mnemonic == m && title.IsVisible() && title.Enabled() {
			c.menuBar.openMenu(title)
			title.menu.activateFirst()
			return true
		}
	}
	return false
}

// processMenuKeys moves through the open menu with the keyboard: Up and
// Down choose the item, Enter (Space) clicks it, Right opens a submenu (or
// the next main menu), Left closes a submenu (or opens the previous main
// menu), a letter clicks the item with the mnemonic. Returns true if the key
// was for the menu.
func (c *Form) processMenuKeys(key Key, mods KeyModifiers) bool {
	menu, ok := c.TopPopupWidget().(*ContextMenu)
	if !ok {
		return false
	}
	switch key {
	case KeyArrowDown:
		menu.moveActive(1)
	case KeyArrowUp:
		menu.moveActive(-1)
	case KeyEnter, KeySpace:
		if item := menu.activeItem(); item != nil {
			item.activateByKeyboard()
		}
	case KeyArrowRight:
		if item := menu.activeItem(); item != nil && item.innerMenu != nil {
			item.activateByKeyboard()
		} else {
			c.switchMenuBarMenu(1)
		}
	case KeyArrowLeft:
		if menu.parentMenu != nil {
			c.topWidget.CloseTopPopup()
		} else {
			c.switchMenuBarMenu(-1)
		}
	default:
		m := mnemonicOfKey(key)
		if m == 0 || mods.Ctrl || mods.Cmd {
			return false
		}
		for _, item := range menu.items {
			if item.mnemonic == m && item.selectable() {
				item.activateByKeyboard()
				break
			}
		}
	}
	c.Update()
	return true
}

// switchMenuBarMenu opens the next (or previous) main menu while one is open
func (c *Form) switchMenuBarMenu(step int) {
	bar := c.menuBar
	if bar == nil || !bar.hasOpenMenu() {
		return
	}
	n := len(bar.items)
	index := 0
	for i, title := range bar.items {
		if title == bar.openItem {
			index = i
		}
	}
	for i := 1; i <= n; i++ {
		title := bar.items[((index+step*i)%n+n)%n]
		if title.IsVisible() && title.Enabled() && title.menu != nil {
			bar.openMenu(title)
			title.menu.activateFirst()
			return
		}
	}
}

// isCommandKey reports whether the key is a command rather than typing:
// with Ctrl, Alt or Cmd, or a function key
func isCommandKey(key Key, mods KeyModifiers) bool {
	if mods.Ctrl || mods.Alt || mods.Cmd {
		return true
	}
	return key >= KeyF1 && key <= KeyF12
}
