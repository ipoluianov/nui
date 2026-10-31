package regression

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func squareIcon(c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

// An action in a menu and on a toolbar: one state, shown by both
func TestActions(t *testing.T) {
	form := newForm(400, 300)
	saves, modified := 0, false
	save := ui.NewAction("&Save", func() { saves++ }).
		SetIcon(squareIcon(color.RGBA{0, 120, 215, 255})).
		SetShortcut("Ctrl+S").
		SetEnabledFunc(func() bool { return modified })
	wrap := ui.NewAction("&Word Wrap", nil).SetCheckable(true)

	bar := ui.NewMenuBar()
	file := bar.AddMenu("&File")
	saveItem := file.AddAction(save)
	view := bar.AddMenu("&View")
	wrapItem := view.AddAction(wrap)
	form.SetMenuBar(bar)

	toolbar := ui.NewToolBar()
	saveButton := toolbar.AddAction(save)
	wrapButton := toolbar.AddAction(wrap)
	form.Panel().AddWidget(0, 0, toolbar)
	form.Panel().AddVSpacer(1, 0)
	w := uitest.Show(t, form)

	// Disabled while nothing is modified: the menu item and the button too
	if save.Enabled() || saveItem.Enabled() || saveButton.Enabled() {
		t.Fatal("save enabled with nothing modified")
	}
	w.Shortcut("Ctrl+S")
	w.Click(saveButton)
	if saves != 0 {
		t.Fatalf("a disabled action ran %d times", saves)
	}
	// The enabled function is asked again on the timer
	modified = true
	w.Wait(200 * time.Millisecond)
	if !saveItem.Enabled() || !saveButton.Enabled() {
		t.Fatal("the items don't follow the enabled function")
	}
	w.Shortcut("Ctrl+S")
	w.Click(saveButton)
	w.Click(w.Find("File"))
	w.Click(w.Find("Save"))
	if saves != 3 {
		t.Errorf("saves: %d, want 3", saves)
	}
	if got := saveButton.(*ui.ToolButton); got == nil {
		t.Error("an action with an icon is a tool button")
	}

	// A checkable action: the menu item and the button show its state
	w.Click(w.Find("View"))
	w.Click(w.Find("Word Wrap"))
	if !wrap.Checked() || !wrapItem.Checked() {
		t.Errorf("after the menu: checked %v, item %v", wrap.Checked(), wrapItem.Checked())
	}
	w.Click(wrapButton)
	if wrap.Checked() || wrapItem.Checked() {
		t.Error("the toolbar button doesn't uncheck")
	}
	// The shortcut is shown in the names of the platform: ⌃S on macOS
	if want := "Save (" + ui.MustParseShortcut("Ctrl+S").String() + ")"; save.Tooltip() != want {
		t.Errorf("tooltip %q, want %q", save.Tooltip(), want)
	}
	w.Click(wrapButton)
	w.Click(w.Find("View"))
	w.MatchGolden("actions-menu")
}

// A plain checkable menu item, by its shortcut too
func TestCheckableMenuItem(t *testing.T) {
	form := newForm(300, 200)
	bar := ui.NewMenuBar()
	view := bar.AddMenu("&View")
	toggles := 0
	item := view.AddItem("&Status Bar", func() { toggles++ }).SetShortcut("F9")
	item.SetCheckable(true)
	form.SetMenuBar(bar)
	w := uitest.Show(t, form)
	w.Key(ui.KeyF9, ui.KeyModifiers{})
	if !item.Checked() || toggles != 1 {
		t.Errorf("F9: checked %v, toggles %d", item.Checked(), toggles)
	}
}

func TestStatusBar(t *testing.T) {
	form := newForm(400, 200)
	status := ui.NewStatusBar()
	position := status.AddLabel("Ln 1")
	form.Panel().AddVSpacer(0, 0)
	form.Panel().AddWidget(1, 0, status)
	w := uitest.Show(t, form)
	status.SetText("Ready")
	status.ShowMessage("Saved", 50*time.Millisecond)
	status.SetText("Ready, 2 files") // stays behind the message
	if w.Find("Saved") == nil {
		t.Fatal("no message")
	}
	w.Wait(120 * time.Millisecond)
	if w.FindAll(func(x ui.Widgeter) bool { l, ok := x.(*ui.Label); return ok && l.Text() == "Ready, 2 files" }) == nil {
		t.Error("the text isn't back after the message")
	}
	if px, _ := position.RectClientAreaOnWindow(); px < 300 {
		t.Errorf("the section at x %d: not at the right", px)
	}
}

// The shortcut of a menu item works right after its action became
// enabled, before the state is asked again by the timer
func TestActionShortcutRightAfterEnabled(t *testing.T) {
	form := newForm(400, 300)
	saves, modified := 0, false
	save := ui.NewAction("&Save", func() { saves++ }).
		SetShortcut("Ctrl+S").
		SetEnabledFunc(func() bool { return modified })
	bar := ui.NewMenuBar()
	bar.AddMenu("&File").AddAction(save)
	form.SetMenuBar(bar)
	form.Panel().AddVSpacer(0, 0)
	w := uitest.Show(t, form)

	modified = true
	w.Shortcut("Ctrl+S")
	if saves != 1 {
		t.Errorf("saves %d", saves)
	}
}
