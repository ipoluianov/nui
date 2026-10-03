package ui

import (
	"runtime"
	"testing"
)

func TestParseShortcut(t *testing.T) {
	sc := MustParseShortcut("Ctrl+Shift+S")
	if sc.Key != KeyS || !sc.Ctrl || !sc.Shift || sc.Alt {
		t.Errorf("Ctrl+Shift+S: %+v", sc)
	}
	if runtime.GOOS != "darwin" && sc.String() != "Ctrl+Shift+S" {
		t.Errorf("String: %q", sc.String())
	}
	for s, key := range map[string]Key{"F5": KeyF5, "delete": KeyDelete, "Alt+F4": KeyF4, "Ctrl++": KeyEqual, "ctrl+pgdn": KeyPageDown} {
		if got, err := ParseShortcut(s); err != nil || got.Key != key {
			t.Errorf("%q: %+v %v", s, got, err)
		}
	}
	mod := MustParseShortcut("Mod+O")
	if (runtime.GOOS == "darwin") != mod.Cmd || (runtime.GOOS != "darwin") != mod.Ctrl {
		t.Errorf("Mod+O: %+v", mod)
	}
	for _, bad := range []string{"", "Ctrl+", "Hyper+S", "Ctrl+Foo"} {
		if _, err := ParseShortcut(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if !sc.Matches(KeyS, KeyModifiers{Ctrl: true, Shift: true}) || sc.Matches(KeyS, KeyModifiers{Ctrl: true}) {
		t.Error("Matches: the modifiers must be exactly the same")
	}
}

func TestParseMnemonic(t *testing.T) {
	cases := []struct {
		in, shown string
		m         rune
		index     int
	}{
		{"&File", "File", 'f', 0},
		{"Save &As", "Save As", 'a', 5},
		{"Drag & Drop", "Drag & Drop", 0, -1},
		{"Tom && &Jerry", "Tom & Jerry", 'j', 6},
		{"Plain", "Plain", 0, -1},
	}
	for _, c := range cases {
		shown, m, index := parseMnemonic(c.in)
		if shown != c.shown || m != c.m || index != c.index {
			t.Errorf("%q: %q %q %d", c.in, shown, m, index)
		}
	}
}

func newMenuTestForm(t *testing.T) (*Form, *TextBox, map[string]int) {
	t.Helper()
	form := NewForm()
	form.processResize(500, 300)
	box := NewTextBox()
	form.Panel().AddWidget(0, 0, box)
	clicks := map[string]int{}
	bar := NewMenuBar()
	file := bar.AddMenu("&File")
	file.AddItem("&New", func() { clicks["new"]++ }).SetShortcut("Ctrl+N")
	file.AddItem("&Save", func() { clicks["save"]++ }).SetShortcut("Ctrl+S")
	file.AddSeparator()
	recent := NewContextMenu(nil)
	recent.AddItem("&a.txt", func() { clicks["a.txt"]++ }).SetShortcut("Ctrl+Shift+1")
	file.AddItemWithSubmenu("&Recent", recent)
	off := file.AddItem("&Disabled", func() { clicks["disabled"]++ })
	off.SetShortcut("Ctrl+D")
	off.SetEnabled(false)
	edit := bar.AddMenu("&Edit")
	edit.AddItem("&Copy", func() { clicks["copy"]++ }).SetShortcut("Ctrl+C")
	edit.AddItem("De&lete", func() { clicks["delete"]++ }).SetShortcut("Delete")
	form.SetMenuBar(bar)
	form.UpdateLayout()
	return form, box, clicks
}

func TestMenuShortcuts(t *testing.T) {
	form, box, clicks := newMenuTestForm(t)
	ctrl := KeyModifiers{Ctrl: true}

	form.processKeyDown(KeyS, ctrl)
	form.processKeyDown(KeyN, ctrl)
	form.processKeyDown(Key1, KeyModifiers{Ctrl: true, Shift: true})
	form.processKeyDown(KeyD, ctrl)
	if clicks["save"] != 1 || clicks["new"] != 1 || clicks["a.txt"] != 1 || clicks["disabled"] != 0 {
		t.Fatalf("shortcuts: %v", clicks)
	}

	// With the text box focused: Ctrl+S still saves, but Ctrl+C and Delete
	// are the text box's
	box.Focus()
	box.SetText("hello")
	form.processKeyDown(KeyS, ctrl)
	form.processKeyDown(KeyC, ctrl)
	form.processKeyDown(KeyDelete, KeyModifiers{})
	if clicks["save"] != 2 || clicks["copy"] != 0 || clicks["delete"] != 0 {
		t.Errorf("with the text box focused: %v", clicks)
	}
	// Without it they are the menu's
	box.ClearFocus()
	form.processKeyDown(KeyC, ctrl)
	form.processKeyDown(KeyDelete, KeyModifiers{})
	if clicks["copy"] != 1 || clicks["delete"] != 1 {
		t.Errorf("without a focused text box: %v", clicks)
	}

	// The form's own shortcuts
	f5 := 0
	form.AddShortcut("F5", func() { f5++ })
	form.processKeyDown(KeyF5, KeyModifiers{})
	form.RemoveShortcut("F5")
	form.processKeyDown(KeyF5, KeyModifiers{})
	if f5 != 1 {
		t.Errorf("form shortcut: %d", f5)
	}
}

func TestMenuKeyboard(t *testing.T) {
	form, _, clicks := newMenuTestForm(t)
	alt := KeyModifiers{Alt: true}
	file := form.menuBar.items[0].menu

	// Alt+F opens File with its first item chosen
	form.processKeyDown(KeyF, alt)
	if form.TopPopupWidget() != Widgeter(file) || file.activeItem() == nil || file.activeItem().Text() != "New" {
		t.Fatalf("Alt+F: top popup %v, active %v", form.TopPopupWidget(), file.activeItem())
	}
	// Down skips the separator and the disabled item, and goes around
	form.processKeyDown(KeyArrowDown, KeyModifiers{})
	form.processKeyDown(KeyArrowDown, KeyModifiers{})
	if file.activeItem().Text() != "Recent" {
		t.Fatalf("Down Down: %q", file.activeItem().Text())
	}
	form.processKeyDown(KeyArrowDown, KeyModifiers{})
	if file.activeItem().Text() != "New" {
		t.Fatalf("around the end: %q", file.activeItem().Text())
	}
	form.processKeyDown(KeyArrowUp, KeyModifiers{})
	// Right opens the submenu, Enter clicks its item and closes the menus
	form.processKeyDown(KeyArrowRight, KeyModifiers{})
	recent := file.items[3].innerMenu
	if form.TopPopupWidget() != Widgeter(recent) {
		t.Fatalf("Right didn't open the submenu")
	}
	form.processKeyDown(KeyEnter, KeyModifiers{})
	if clicks["a.txt"] != 1 || len(form.topWidget.PopupWidgets) != 0 {
		t.Fatalf("Enter: %v, popups %d", clicks, len(form.topWidget.PopupWidgets))
	}

	// A mnemonic letter in the open menu clicks its item; Right on an item
	// without a submenu goes to the next main menu
	form.processKeyDown(KeyF, alt)
	form.processKeyDown(KeyArrowRight, KeyModifiers{})
	edit := form.menuBar.items[1].menu
	if form.TopPopupWidget() != Widgeter(edit) {
		t.Fatalf("Right didn't open Edit")
	}
	form.processKeyDown(KeyL, KeyModifiers{})
	if clicks["delete"] != 1 || len(form.topWidget.PopupWidgets) != 0 {
		t.Errorf("mnemonic L: %v", clicks)
	}
}

func TestShortcutPunctuationAndNumpad(t *testing.T) {
	for s, want := range map[string]Shortcut{
		`Ctrl+\`:      {Key: KeyBackslash, Ctrl: true},
		"Alt+NumPlus": {Key: KeyNumpadPlus, Alt: true},
		"NumMinus":    {Key: KeyNumpadMinus},
		"NumMultiply": {Key: KeyNumpadAsterisk},
		"NumDivide":   {Key: KeyNumpadSlash},
		"Ctrl+.":      {Key: KeyDot, Ctrl: true},
	} {
		got, err := ParseShortcut(s)
		if err != nil || got != want {
			t.Errorf("%s: %+v %v", s, got, err)
		}
	}
	if runtime.GOOS != "darwin" {
		if got := MustParseShortcut("Ctrl+NumPlus").String(); got != "Ctrl+Num +" {
			t.Errorf("String: %q", got)
		}
		if got := MustParseShortcut(`Ctrl+\`).String(); got != `Ctrl+\` {
			t.Errorf("String: %q", got)
		}
	}
}

func TestTypingKeysStayInTextBox(t *testing.T) {
	for _, k := range []Key{KeyDot, KeyNumpadPlus, KeyA, KeySpace, KeyNumpad5} {
		if !textEditingKey(k, KeyModifiers{}) {
			t.Errorf("%v: not a text editing key", k)
		}
		if !textEditingKey(k, KeyModifiers{Shift: true}) {
			t.Errorf("Shift+%v: not a text editing key", k)
		}
	}
	if textEditingKey(KeyDot, KeyModifiers{Ctrl: true}) || textEditingKey(KeyF5, KeyModifiers{}) {
		t.Error("Ctrl+. and F5 are shortcuts")
	}
}
