package ui

import "testing"

// Space flips a checkbox like a click, through the tristate cycle; Enter is
// left to the form, and a disabled checkbox doesn't change
func TestCheckboxKeyboard(t *testing.T) {
	cb := NewCheckbox("Option")
	form := newTestForm(t, cb)
	changes := 0
	cb.SetOnStateChanged(func() { changes++ })
	cb.Focus()

	typeKey(form, KeySpace, ' ', KeyModifiers{})
	if !cb.Checked() || changes != 1 {
		t.Fatalf("Space: checked %v, changes %d", cb.Checked(), changes)
	}
	if cb.ProcessKeyDown(KeyEnter, KeyModifiers{}) {
		t.Error("Enter was taken: the accept button of a dialog wouldn't get it")
	}
	if cb.ProcessKeyDown(KeySpace, KeyModifiers{Ctrl: true}) {
		t.Error("Ctrl+Space was taken")
	}

	cb.SetTristate(true)
	typeKey(form, KeySpace, ' ', KeyModifiers{})
	if !cb.Indeterminate() {
		t.Fatal("tristate: Space on checked didn't go to indeterminate")
	}
	typeKey(form, KeySpace, ' ', KeyModifiers{})
	if cb.Checked() || cb.Indeterminate() {
		t.Fatal("tristate: Space on indeterminate didn't uncheck")
	}

	cb.SetEnabled(false)
	if cb.ProcessKeyDown(KeySpace, KeyModifiers{}) || cb.Checked() {
		t.Error("a disabled checkbox changed")
	}

	// The application's own key handler goes first
	cb.SetEnabled(true)
	cb.SetOnKeyDown(func(key Key, mods KeyModifiers) bool { return key == KeySpace })
	typeKey(form, KeySpace, ' ', KeyModifiers{})
	if cb.Checked() {
		t.Error("the OnKeyDown handler didn't get Space first")
	}
}

// Space checks a radio button and unchecks the others of its group
func TestRadioButtonKeyboard(t *testing.T) {
	panel := NewPanel()
	a := NewRadioButton("A")
	b := NewRadioButton("B")
	panel.AddWidget(0, 0, a)
	panel.AddWidget(1, 0, b)
	form := newTestForm(t, panel)
	a.SetChecked(true)
	b.Focus()
	typeKey(form, KeySpace, ' ', KeyModifiers{})
	if !b.Checked() || a.Checked() {
		t.Errorf("Space: a %v, b %v", a.Checked(), b.Checked())
	}
}

func newTestComboBox(t *testing.T) (*ComboBox, *Form, *int) {
	t.Helper()
	cb := NewComboBox()
	for _, s := range []string{"One", "Two", "Three"} {
		cb.AddItem(s, s)
	}
	cb.SetSelectedIndex(0)
	form := newTestForm(t, cb)
	changes := 0
	cb.SetOnSelectedIndexChanged(func() { changes++ })
	return cb, form, &changes
}

// Tab reaches a combo box; Up/Down/Home/End pick an item in place and call
// the callback; Enter is left to the form
func TestComboBoxKeyboard(t *testing.T) {
	cb, form, changes := newTestComboBox(t)
	form.processKeyDown(KeyTab, KeyModifiers{})
	if form.FocusedWidget() != Widgeter(cb) {
		t.Fatalf("Tab focused %v", form.FocusedWidget())
	}

	form.processKeyDown(KeyArrowDown, KeyModifiers{})
	form.processKeyDown(KeyArrowDown, KeyModifiers{})
	form.processKeyDown(KeyArrowDown, KeyModifiers{}) // at the end: stays
	if cb.SelectedIndex() != 2 || *changes != 2 {
		t.Fatalf("Down x3: index %d, changes %d", cb.SelectedIndex(), *changes)
	}
	form.processKeyDown(KeyHome, KeyModifiers{})
	if cb.SelectedIndex() != 0 {
		t.Fatalf("Home: index %d", cb.SelectedIndex())
	}
	form.processKeyDown(KeyEnd, KeyModifiers{})
	form.processKeyDown(KeyArrowUp, KeyModifiers{})
	if cb.SelectedIndex() != 1 || *changes != 5 {
		t.Fatalf("End, Up: index %d, changes %d", cb.SelectedIndex(), *changes)
	}
	if cb.ProcessKeyDown(KeyEnter, KeyModifiers{}) {
		t.Error("Enter was taken")
	}
	if len(form.topWidget.PopupWidgets) != 0 {
		t.Error("the arrows opened the list")
	}
}

// Space opens the list; the arrows choose in it, Enter picks, Esc leaves
// the selection as it was
func TestComboBoxPopupKeyboard(t *testing.T) {
	cb, form, changes := newTestComboBox(t)
	cb.Focus()

	typeKey(form, KeySpace, ' ', KeyModifiers{})
	if _, ok := form.TopPopupWidget().(*comboBoxPopup); !ok {
		t.Fatalf("Space didn't open the list: %v", form.TopPopupWidget())
	}
	form.processKeyDown(KeyArrowDown, KeyModifiers{})
	form.processKeyDown(KeyArrowDown, KeyModifiers{})
	if cb.SelectedIndex() != 0 || *changes != 0 {
		t.Fatal("moving in the list changed the selection")
	}
	form.processKeyDown(KeyEnter, KeyModifiers{})
	if cb.SelectedIndex() != 2 || *changes != 1 || len(form.topWidget.PopupWidgets) != 0 {
		t.Fatalf("Enter: index %d, changes %d, popups %d", cb.SelectedIndex(), *changes, len(form.topWidget.PopupWidgets))
	}

	form.processKeyDown(KeyArrowDown, KeyModifiers{Alt: true})
	form.processKeyDown(KeyArrowUp, KeyModifiers{})
	form.processKeyDown(KeyEsc, KeyModifiers{})
	if cb.SelectedIndex() != 2 || len(form.topWidget.PopupWidgets) != 0 {
		t.Fatalf("Esc: index %d, popups %d", cb.SelectedIndex(), len(form.topWidget.PopupWidgets))
	}

	cb.SetEnabled(false)
	if cb.ProcessKeyDown(KeySpace, KeyModifiers{}) || len(form.topWidget.PopupWidgets) != 0 {
		t.Error("a disabled combo box opened")
	}
}

// Tab and Shift+Tab skip disabled widgets; a focused widget that got
// disabled is left for the next one
func TestTabSkipsDisabled(t *testing.T) {
	panel := NewPanel()
	first := NewButton("First")
	off := NewButton("Off")
	last := NewButton("Last")
	panel.AddWidget(0, 0, first)
	panel.AddWidget(0, 1, off)
	panel.AddWidget(0, 2, last)
	form := newTestForm(t, panel)
	off.SetEnabled(false)

	first.Focus()
	form.processKeyDown(KeyTab, KeyModifiers{})
	if form.FocusedWidget() != Widgeter(last) {
		t.Fatalf("Tab: %v", form.FocusedWidget())
	}
	form.processKeyDown(KeyTab, KeyModifiers{Shift: true})
	if form.FocusedWidget() != Widgeter(first) {
		t.Fatalf("Shift+Tab: %v", form.FocusedWidget())
	}

	off.SetEnabled(true)
	off.Focus()
	off.SetEnabled(false)
	form.processKeyDown(KeyTab, KeyModifiers{})
	if form.FocusedWidget() != Widgeter(last) {
		t.Errorf("Tab from a disabled focused widget: %v", form.FocusedWidget())
	}
}

func newContextMenuTestTable(t *testing.T) (*Table, *Form, map[string]int) {
	t.Helper()
	table := NewTable()
	table.SetColumnCount(2)
	table.SetRowCount(10)
	form := newTestForm(t, table)
	clicks := map[string]int{}
	menu := NewContextMenu(table)
	menu.AddItem("Edit", func() { clicks["edit"]++ })
	menu.AddItem("Remove", func() { clicks["remove"]++ })
	table.SetContextMenu(menu)
	table.SetCurrentCell2(3, 0)
	table.Focus()
	return table, form, clicks
}

// The Menu key and Shift+F10 open the context menu of the focused widget
// under its current row, with the first item chosen
func TestContextMenuKey(t *testing.T) {
	table, form, clicks := newContextMenuTestTable(t)

	for _, k := range []struct {
		key  Key
		mods KeyModifiers
	}{{KeyContextMenu, KeyModifiers{}}, {KeyF10, KeyModifiers{Shift: true}}} {
		form.processKeyDown(k.key, k.mods)
		menu, ok := form.TopPopupWidget().(*ContextMenu)
		if !ok || menu != table.ContextMenu() {
			t.Fatalf("%v: top popup %v", k, form.TopPopupWidget())
		}
		if menu.activeItem() == nil || menu.activeItem().Text() != "Edit" {
			t.Fatalf("%v: active item %v", k, menu.activeItem())
		}
		_, tableY := table.RectClientAreaOnWindow()
		if want := tableY + table.rowOffset(3) + table.rowHeight1; menu.Y() != want {
			t.Errorf("%v: menu at y %d, want %d (under the current row)", k, menu.Y(), want)
		}
		form.processKeyDown(KeyEsc, KeyModifiers{})
	}

	form.processKeyDown(KeyF10, KeyModifiers{Shift: true})
	form.processKeyDown(KeyArrowDown, KeyModifiers{})
	form.processKeyDown(KeyEnter, KeyModifiers{})
	if clicks["remove"] != 1 || len(form.topWidget.PopupWidgets) != 0 {
		t.Errorf("Down, Enter: %v, popups %d", clicks, len(form.topWidget.PopupWidgets))
	}

	// F10 alone and Ctrl+Shift+F10 are not the menu keys
	form.processKeyDown(KeyF10, KeyModifiers{})
	form.processKeyDown(KeyF10, KeyModifiers{Shift: true, Ctrl: true})
	if len(form.topWidget.PopupWidgets) != 0 {
		t.Error("another key opened the menu")
	}
}

// The application keeps the key: its shortcut and the widget's own handler
// go first; a widget without a menu gets its parent's
func TestContextMenuKeyPriority(t *testing.T) {
	table, form, _ := newContextMenuTestTable(t)
	shortcut := 0
	form.AddShortcut("Shift+F10", func() { shortcut++ })
	form.processKeyDown(KeyF10, KeyModifiers{Shift: true})
	if shortcut != 1 || len(form.topWidget.PopupWidgets) != 0 {
		t.Fatalf("shortcut: called %d, popups %d", shortcut, len(form.topWidget.PopupWidgets))
	}

	table.SetOnKeyDown(func(key Key, mods KeyModifiers) bool { return key == KeyContextMenu })
	form.processKeyDown(KeyContextMenu, KeyModifiers{})
	if len(form.topWidget.PopupWidgets) != 0 {
		t.Fatal("the widget's handler didn't get the Menu key first")
	}

	panel := NewPanel()
	box := NewTextBox()
	panel.AddWidget(0, 0, box)
	form2 := newTestForm(t, panel)
	menu := NewContextMenu(panel)
	menu.AddItem("Paste", func() {})
	panel.SetContextMenu(menu)
	box.Focus()
	form2.processKeyDown(KeyContextMenu, KeyModifiers{})
	if form2.TopPopupWidget() != Widgeter(menu) {
		t.Errorf("parent's menu: top popup %v", form2.TopPopupWidget())
	}

	// No menu anywhere: the key goes on, nothing opens
	plain := NewTextBox()
	form3 := newTestForm(t, plain)
	plain.Focus()
	form3.processKeyDown(KeyContextMenu, KeyModifiers{})
	if len(form3.topWidget.PopupWidgets) != 0 {
		t.Error("a menu opened without one")
	}
}

// In an open menu Home/End choose the first and the last item, and the keys
// the menu doesn't use stay in it: the table behind keeps its current row
func TestOpenMenuKeepsKeys(t *testing.T) {
	table, form, clicks := newContextMenuTestTable(t)
	form.processKeyDown(KeyContextMenu, KeyModifiers{})
	menu := table.ContextMenu()
	form.processKeyDown(KeyEnd, KeyModifiers{})
	if menu.activeItem() == nil || menu.activeItem().Text() != "Remove" {
		t.Fatalf("End: active %v", menu.activeItem())
	}
	form.processKeyDown(KeyHome, KeyModifiers{})
	if menu.activeItem() == nil || menu.activeItem().Text() != "Edit" {
		t.Fatalf("Home: active %v", menu.activeItem())
	}
	for _, key := range []Key{KeyPageDown, KeyDelete, KeyArrowLeft, KeyF5} {
		form.processKeyDown(key, KeyModifiers{})
	}
	if table.CurrentRow() != 3 || form.TopPopupWidget() != Widgeter(menu) || len(clicks) != 0 {
		t.Errorf("keys leaked: row %d, top popup %v, clicks %v", table.CurrentRow(), form.TopPopupWidget(), clicks)
	}
}
