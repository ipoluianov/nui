package ui

import (
	"testing"

	"github.com/ipoluianov/nui/internal/platforms"
)

// A drag handler that opens a message box (e.g. "can't drop here")
func TestAuditModalFromDragMove(t *testing.T) {
	f := auditHeadless(t)
	p := NewPanel()
	p.SetMinSize(200, 200)
	f.Panel().AddWidget(0, 0, p)
	f.UpdateLayout()
	shown := false
	w := auditWnd(f)
	w.MouseDown(MouseButtonLeft, 20, 20)
	// a drag: the moves with the button held
	p.SetOnMouseMove(func(x, y int, mods KeyModifiers) bool {
		if !shown {
			shown = true
			ShowMessageBox(p, "Drag", "Can't drop here")
		}
		return true
	})
	t.Logf("pressed widget %v", f.mouseLeftButtonPressedWidget)
	w.MouseMove(30, 30)
	platforms.RunHeadlessPosted()
	t.Logf("shown %v, forms %d", shown, len(openForms()))
}

// A menu bar item without a menu, added once the bar is in a form
func TestAuditMenuBarItemNilMenu(t *testing.T) {
	form := newTestForm(t, NewLabel("x"))
	bar := NewMenuBar()
	form.SetMenuBar(bar)
	bar.AddMenuItem("Help", nil)
}

// A message box for a widget that isn't in a form yet
func TestAuditMessageBoxWidgetNoForm(t *testing.T) {
	platforms.SetHeadless()
	ShowMessageBox(NewLabel("x"), "T", "text")
}
