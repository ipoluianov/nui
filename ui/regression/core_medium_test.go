package regression

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// A widget spanning a huge number of columns or rows (a span from a layout
// file) is laid out over the lines that exist: the layout used to walk over
// every number of the span (a hang), and a span past math.MaxInt-first
// wrapped around and covered no line at all
func TestHugeGridSpan(t *testing.T) {
	done := make(chan struct{})
	var form *ui.Form
	var wide, tall, next, below *ui.Panel
	go func() {
		defer close(done)
		form = newForm(500, 400)
		p := form.Panel()
		panel := func(w, h int) *ui.Panel {
			x := ui.NewPanel()
			x.SetMinSize(w, h)
			return x
		}
		wide, tall = panel(20, 20), panel(20, 20)
		next, below = panel(80, 30), panel(30, 80)
		p.AddWidgetSpan(0, 1, 1, 1<<40, wide)
		p.AddWidget(0, 2, next)
		p.AddWidgetSpan(1, 0, math.MaxInt, 1, tall)
		p.AddWidget(2, 0, below)
		p.AddVSpacer(1<<20, 0)
		form.Panel().X()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the layout did not end within 10s")
	}
	uitest.Show(t, form)
	if right := next.X() + next.Width(); wide.X()+wide.Width() < right {
		t.Errorf("the spanning widget ends at x %d, the column it spans at %d", wide.X()+wide.Width(), right)
	}
	if bottom := below.Y() + below.Height(); tall.Y()+tall.Height() < bottom {
		t.Errorf("the spanning widget ends at y %d, the row it spans at %d", tall.Y()+tall.Height(), bottom)
	}
}

// A mouse handler of one of the overlapping widgets that removes the others:
// the loop over the children read the shrunk list by index (a panic)
func TestMouseHandlerRemovesSiblings(t *testing.T) {
	for _, double := range []bool{false, true} {
		form := newForm(300, 200)
		p := form.Panel()
		var panels []*ui.Panel
		for i := 0; i < 3; i++ {
			x := ui.NewPanel()
			x.SetMinSize(100, 100)
			p.AddWidget(0, 0, x)
			panels = append(panels, x)
		}
		top := panels[2]
		removeOthers := func() {
			p.RemoveWidget(panels[0])
			p.RemoveWidget(panels[1])
		}
		top.SetOnMouseDown(func(ui.MouseButton, int, int, ui.KeyModifiers) bool {
			removeOthers()
			return false
		})
		top.SetOnMouseDblClick(func(ui.MouseButton, int, int, ui.KeyModifiers) bool {
			removeOthers()
			return false
		})
		w := uitest.Show(t, form)
		if double {
			w.DoubleClick(uitest.At(top, 10, 10))
		} else {
			w.Click(uitest.At(top, 10, 10))
		}
		if panels[0].Form() != nil || panels[1].Form() != nil || top.Form() == nil {
			t.Errorf("double %v: the others are still in the form, or the top one is gone", double)
		}
		form.Close()
	}
}

// A context menu built for one show leaves the form when it closes: every
// menu shown used to stay in the registry of the widgets (and its items
// linked to their actions) until the form closed
func TestShownContextMenuLeavesForm(t *testing.T) {
	form := newForm(300, 200)
	label := ui.NewLabel("x")
	form.Panel().AddWidget(0, 0, label)
	w := uitest.Show(t, form)
	a := ui.NewAction("Copy", func() {})
	for i := 0; i < 3; i++ {
		menu := ui.NewContextMenu(nil)
		item := menu.AddAction(a)
		form.ShowContextMenu(menu, 10, 10)
		if w.Popup() == nil || form.WidgetById(item.Id()) == nil {
			t.Fatal("the menu shown is not open")
		}
		form.CloseTopPopup()
		if menu.Form() != nil || form.WidgetById(item.Id()) != nil {
			t.Fatalf("show %d: the closed menu is still in the form", i)
		}
	}

	// A menu kept by the application is shown again, and its item works
	ran := 0
	menu := ui.NewContextMenu(nil)
	menu.AddItem("Run", func() { ran++ })
	for i := 0; i < 2; i++ {
		form.ShowContextMenu(menu, 10, 10)
		w.Click(w.Find("Run"))
	}
	if ran != 2 || menu.Form() != nil {
		t.Errorf("ran %d times, want 2; in the form after the close: %v", ran, menu.Form() != nil)
	}
}

// The context menu of a widget stays with it: closed, it opens again on the
// next right click, with its submenu
func TestWidgetContextMenuReused(t *testing.T) {
	form := newForm(300, 200)
	label := ui.NewLabel("target")
	label.SetMinSize(100, 40)
	form.Panel().AddWidget(0, 0, label)
	ran := 0
	sub := ui.NewContextMenu(nil)
	sub.AddItem("Inner", func() { ran++ })
	menu := ui.NewContextMenu(nil)
	menu.AddItem("Outer", func() { ran++ })
	menu.AddItemWithSubmenu("More", sub)
	label.SetContextMenu(menu)
	w := uitest.Show(t, form)
	for i := 0; i < 2; i++ {
		w.RightClick(label)
		w.Click(w.Find("Outer"))
	}
	w.RightClick(label)
	w.Click(w.Find("More"))
	w.Click(w.Find("Inner"))
	if ran != 3 || menu.Form() == nil {
		t.Errorf("ran %d times, want 3; the widget's menu in the form: %v", ran, menu.Form() != nil)
	}
}

// SetContextMenu(nil) takes the menu away (it panicked), and a menu replaced
// leaves the form; one shared by two widgets still opens on the other
func TestSetContextMenuReplace(t *testing.T) {
	form := newForm(300, 200)
	a, b := ui.NewLabel("a"), ui.NewLabel("b")
	a.SetMinSize(100, 40)
	b.SetMinSize(100, 40)
	form.Panel().AddWidget(0, 0, a)
	form.Panel().AddWidget(1, 0, b)
	w := uitest.Show(t, form)

	alone := ui.NewLabel("alone")
	alone.SetContextMenu(ui.NewContextMenu(nil))
	alone.SetContextMenu(nil)

	old := ui.NewContextMenu(nil)
	oldItem := old.AddItem("Old", nil)
	a.SetContextMenu(old)
	if form.WidgetById(oldItem.Id()) == nil {
		t.Fatal("the menu set is not in the form")
	}
	a.SetContextMenu(ui.NewContextMenu(nil))
	if old.Form() != nil || form.WidgetById(oldItem.Id()) != nil {
		t.Error("the menu replaced is still in the form")
	}
	a.SetContextMenu(nil)
	if a.ContextMenu() != nil {
		t.Error("SetContextMenu(nil) kept a menu")
	}

	ran := 0
	shared := ui.NewContextMenu(nil)
	shared.AddItem("Shared", func() { ran++ })
	a.SetContextMenu(shared)
	b.SetContextMenu(shared)
	a.SetContextMenu(nil)
	w.RightClick(b)
	w.Click(w.Find("Shared"))
	if ran != 1 {
		t.Errorf("the shared menu ran %d times on the other widget, want 1", ran)
	}
}

// A form closed and shown again (a tray application) saves its state and
// its dock layout at every close, not only at the first
func TestRememberStateShownAgain(t *testing.T) {
	uitest.SetScreen(1920, 1080, 1)
	path := filepath.Join(t.TempDir(), "settings.json")
	settings, _ := ui.OpenSettingsFile(path)
	form, docks, d := newDockForm(nil)
	form.RememberState(settings, "main")
	docks.RememberState(settings, "docks")
	uitest.Show(t, form)
	form.Close()

	w := uitest.Show(t, form)
	w.Resize(640, 480)
	d["Files"].SetShown(false)
	form.Close()

	settings, _ = ui.OpenSettingsFile(path)
	var saved ui.FormState
	if !settings.Get("main", &saved) || saved.Width != 640 || saved.Height != 480 {
		t.Errorf("saved after the second close: %+v, want 640x480", saved)
	}
	_, _, d2 := newDockForm(settings)
	if d2["Files"].Shown() {
		t.Error("the dock hidden before the second close is shown again")
	}
}

// The OK and Cancel of a dialog button box removed from the form are no
// longer pushed by Enter and Esc
func TestRemovedDialogButtonBox(t *testing.T) {
	form := newForm(300, 200)
	ok, cancel := 0, 0
	box := ui.NewDialogButtonBox()
	box.AddOK(func() { ok++ })
	box.AddCancel(func() { cancel++ })
	form.Panel().AddWidget(0, 0, box)
	w := uitest.Show(t, form)
	form.Panel().RemoveWidget(box)
	w.Key(ui.KeyEnter, ui.KeyModifiers{})
	w.Key(ui.KeyEsc, ui.KeyModifiers{})
	if ok != 0 || cancel != 0 {
		t.Errorf("the removed box: OK pushed %d, Cancel %d times", ok, cancel)
	}
	// Added again, it is the form's again
	form.Panel().AddWidget(0, 0, box)
	w.Key(ui.KeyEnter, ui.KeyModifiers{})
	if ok != 1 {
		t.Errorf("the box added again: OK pushed %d times, want 1", ok)
	}
}
