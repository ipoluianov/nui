package ui

import (
	"testing"
)

// A key handler (SetOnKeyDown) of the focused widget that removes the widget
// and returns false
func TestAuditKeyDownRemovesFocused(t *testing.T) {
	p := NewPanel()
	p.SetCanBeFocused(true)
	p.SetMinSize(50, 50)
	form := newTestForm(t, p)
	p.Focus()
	if form.FocusedWidget() == nil {
		t.Fatal("not focused")
	}
	p.SetOnKeyDown(func(key Key, mods KeyModifiers) bool {
		form.Panel().RemoveWidget(p)
		return false
	})
	form.processKeyDown(KeyA, KeyModifiers{})
}

// A mouse-move handler of the widget being dragged removes the widget
func TestAuditMouseMoveRemovesPressed(t *testing.T) {
	p := NewPanel()
	p.SetMinSize(100, 100)
	form := newTestForm(t, p)
	form.UpdateLayout()
	p.SetOnMouseMove(func(x, y int, mods KeyModifiers) bool {
		form.Panel().RemoveWidget(p)
		return false
	})
	form.processMouseDown(MouseButtonLeft, 10, 10)
	if form.mouseLeftButtonPressedWidget == nil {
		t.Fatal("no pressed widget")
	}
	form.processMouseMove(20, 20)
}

// Overlapping widgets in one cell; the top one's mouse-down handler removes
// the others and returns false
func TestAuditMouseDownRemovesSiblings(t *testing.T) {
	form := NewForm()
	form.processResize(500, 400)
	a, b, c := NewPanel(), NewPanel(), NewPanel()
	for _, w := range []*Panel{a, b, c} {
		w.SetMinSize(100, 100)
		form.Panel().AddWidget(0, 0, w)
	}
	form.UpdateLayout()
	c.SetOnMouseDown(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		form.Panel().RemoveWidget(a)
		form.Panel().RemoveWidget(b)
		return false
	})
	form.processMouseDown(MouseButtonLeft, 10, 10)
}

func auditRegistrySize() int {
	allwidgetsMtx.RLock()
	defer allwidgetsMtx.RUnlock()
	return len(allwidgets)
}

// Menus made for one showing (a context menu built per right click) stay in
// the global registry while the form lives
func TestAuditContextMenuRegistryLeak(t *testing.T) {
	form := newTestForm(t, NewLabel("x"))
	before := auditRegistrySize()
	for i := 0; i < 100; i++ {
		m := NewContextMenu(nil)
		m.AddItem("Copy", func() {})
		m.AddItem("Paste", func() {})
		form.ShowContextMenu(m, 10, 10)
		form.CloseTopPopup()
	}
	if len(form.Panel().PopupWidgets) != 0 {
		t.Fatal("popups left")
	}
	t.Logf("registry before %d after %d", before, auditRegistrySize())
	if auditRegistrySize() > before+5 {
		t.Errorf("registry grew from %d to %d after 100 menus shown and closed", before, auditRegistrySize())
	}
}

// RemoveWidget, then AddWidget elsewhere moves the widget (rules.md)
func TestAuditRemoveThenAdd(t *testing.T) {
	l := NewLabel("x")
	form := newTestForm(t, l)
	other := NewPanel()
	form.Panel().AddWidget(3, 0, other)
	form.Panel().RemoveWidget(l)
	other.AddWidget(0, 0, l)
	if len(other.Widgets()) != 1 || l.Form() != form {
		t.Errorf("not moved: %d children, form %p", len(other.Widgets()), l.Form())
	}
}
