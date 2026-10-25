package regression

import (
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// The signals of the widgets (doc/production-readiness.md 3.2)

// Bug #3, fixed: the release of the right button reaches the widget, by
// the old handler and by the signal
func TestRightButtonUp(t *testing.T) {
	form := newForm(300, 200)
	target := ui.NewPanel()
	target.SetMinSize(100, 100)
	old, signal := 0, 0
	target.SetOnMouseUp(func(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
		if button == ui.MouseButtonRight {
			old++
		}
		return true
	})
	target.OnMouseUp.Connect(func(e *ui.MouseEvent) {
		if e.Button == ui.MouseButtonRight {
			signal++
		}
	})
	form.Panel().AddWidget(0, 0, target)
	w := uitest.Show(t, form)
	w.RightClick(target)
	if old != 1 || signal != 1 {
		t.Errorf("right button releases: handler %d, signal %d", old, signal)
	}
}

// A right click doesn't click a button or check a check box
func TestRightClickIsNoClick(t *testing.T) {
	form := newForm(300, 200)
	clicks := 0
	btn := form.Panel().AddButton(0, 0, "Button", func() { clicks++ })
	box := ui.NewCheckbox("Box")
	form.Panel().AddWidget(1, 0, box)
	w := uitest.Show(t, form)
	w.RightClick(btn)
	w.RightClick(box)
	if clicks != 0 || box.Checked() {
		t.Errorf("after right clicks: %d clicks, checked %v", clicks, box.Checked())
	}
}

// Any number of handlers; the button is clicked with the mouse, the keys
// and Push; Disconnect removes a handler
func TestButtonOnClick(t *testing.T) {
	form := newForm(300, 200)
	old := 0
	btn := form.Panel().AddButton(0, 0, "Button", func() { old++ })
	a, b := 0, 0
	btn.OnClick.Connect(func(w ui.Widgeter) {
		if w != ui.Widgeter(btn) {
			t.Errorf("sender %v", w)
		}
		a++
	})
	conn := btn.OnClick.Connect(func(ui.Widgeter) { b++ })
	w := uitest.Show(t, form)
	w.Click(btn)
	w.Key(ui.KeySpace, ui.KeyModifiers{}) // the click focused it
	btn.Push()
	if a != 3 || b != 3 || old != 3 {
		t.Errorf("clicks: %d, %d, old handler %d; want 3", a, b, old)
	}
	conn.Disconnect()
	w.Click(btn)
	if a != 4 || b != 3 {
		t.Errorf("after Disconnect: %d, %d", a, b)
	}
}

// Any widget can be clicked: a panel; a press dragged off isn't a click
func TestPanelOnClick(t *testing.T) {
	form := newForm(300, 200)
	panel := ui.NewPanel()
	panel.SetMinSize(100, 60)
	other := ui.NewPanel()
	other.SetMinSize(100, 60)
	form.Panel().AddWidget(0, 0, panel)
	form.Panel().AddWidget(1, 0, other)
	clicks := 0
	panel.OnClick.Connect(func(ui.Widgeter) { clicks++ })
	w := uitest.Show(t, form)
	w.Click(panel)
	w.Drag(panel, other)
	if clicks != 1 {
		t.Errorf("clicks %d, want 1", clicks)
	}
}

// OnKeyDown comes before the widget: a handled key isn't typed, Enter can
// be caught without CurrentEvent
func TestTextBoxOnKeyDown(t *testing.T) {
	form := newForm(300, 200)
	tb := ui.NewTextBox()
	form.Panel().AddWidget(0, 0, tb)
	entered := ""
	tb.OnKeyDown.Connect(func(e *ui.KeyEvent) {
		if e.Key == ui.KeyEnter {
			entered = tb.Text()
			e.Handled = true
		}
	})
	tb.OnChar.Connect(func(e *ui.CharEvent) {
		if e.Char == '#' {
			e.Handled = true // no hashes
		}
	})
	w := uitest.Show(t, form)
	w.Click(tb)
	w.Type("a#b\n")
	if tb.Text() != "ab" || entered != "ab" {
		t.Errorf("text %q, entered %q", tb.Text(), entered)
	}
}

// The wheel tells where the mouse is and which keys are held
func TestOnWheel(t *testing.T) {
	form := newForm(300, 200)
	canvas := ui.NewPanel()
	canvas.SetMinSize(200, 150)
	form.Panel().AddWidget(0, 0, canvas)
	var got ui.WheelEvent
	canvas.OnWheel.Connect(func(e *ui.WheelEvent) { got = *e; e.Handled = true })
	w := uitest.Show(t, form)
	w.Move(uitest.At(canvas, 30, 40))
	w.Shortcut("Ctrl+A") // holds Ctrl for the next event
	w.Wheel(uitest.At(canvas, 30, 40), 0, 1)
	if got.X != 30 || got.Y != 40 || got.DY != 1 {
		t.Errorf("wheel %+v", got)
	}
}

// The press and the moves while the button is held go to the pressed
// widget, wherever the mouse is
func TestMouseCapture(t *testing.T) {
	form := newForm(300, 200)
	a, b := ui.NewPanel(), ui.NewPanel()
	a.SetMinSize(100, 60)
	b.SetMinSize(100, 60)
	form.Panel().AddWidget(0, 0, a)
	form.Panel().AddWidget(1, 0, b)
	moves, ups := 0, 0
	lastY := 0
	a.OnMouseMove.Connect(func(e *ui.MouseEvent) { moves++; lastY = e.Y })
	a.OnMouseUp.Connect(func(e *ui.MouseEvent) { ups++ })
	w := uitest.Show(t, form)
	w.Drag(a, b)
	if ups != 1 || lastY <= a.Height() {
		t.Errorf("ups %d, last move at y %d (a is %d high)", ups, lastY, a.Height())
	}
	before := moves
	w.Move(b)
	if moves != before {
		t.Error("a gets the moves over b after the release")
	}
}

// OnResize tells the new size, from the layout
func TestOnResize(t *testing.T) {
	form := newForm(300, 200)
	panel := ui.NewPanel()
	panel.SetXExpandable(true)
	form.Panel().AddWidget(0, 0, panel)
	var got ui.Size
	panel.OnResize.Connect(func(s ui.Size) { got = s })
	w := uitest.Show(t, form)
	w.Resize(500, 200)
	if got.Width != panel.Width() || got.Width < 450 {
		t.Errorf("OnResize %+v, width %d", got, panel.Width())
	}
}

// Bug #4, fixed: a disabled container disables its children
func TestDisabledParent(t *testing.T) {
	form := newForm(300, 200)
	panel := form.Panel().AddPanel(0, 0)
	clicks := 0
	btn := panel.AddButton(0, 0, "Click", func() { clicks++ })
	panel.SetEnabled(false)
	w := uitest.Show(t, form)
	w.Click(btn)
	if clicks != 0 || btn.Enabled() {
		t.Errorf("in a disabled panel: %d clicks, enabled %v", clicks, btn.Enabled())
	}
	panel.SetEnabled(true)
	w.Click(btn)
	if clicks != 1 {
		t.Errorf("enabled again: %d clicks", clicks)
	}
}

// Bug #11, fixed: an open list takes the keys, the form's global handler
// gets them again once it closes
func TestGlobalKeysUnderPopup(t *testing.T) {
	form := newForm(300, 200)
	combo := ui.NewComboBox()
	combo.AddItem("One", nil)
	combo.AddItem("Two", nil)
	form.Panel().AddWidget(0, 0, combo)
	global := 0
	form.SetOnGlobalKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool {
		if key == ui.KeyF5 {
			global++
		}
		return false
	})
	w := uitest.Show(t, form)
	w.Click(combo)
	if w.Popup() == nil {
		t.Fatal("the list didn't open")
	}
	w.Key(ui.KeyF5, ui.KeyModifiers{})
	if global != 0 {
		t.Error("F5 reached the global handler over the open list")
	}
	w.Key(ui.KeyEsc, ui.KeyModifiers{})
	w.Key(ui.KeyF5, ui.KeyModifiers{})
	if global != 1 {
		t.Errorf("after the list closed: %d", global)
	}
}
