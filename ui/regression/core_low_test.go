package regression

import (
	"path/filepath"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// Huge stretch weights: the layout adds them and multiplies the free room by
// them, so four columns of 1<<62 summed to 0 (a divide-by-zero panic) and
// smaller ones overflowed into garbage widths. The weights are capped, and
// equal weights still share the room equally.
func TestHugeStretchWeights(t *testing.T) {
	form := newForm(500, 400)
	p := form.Panel()
	var cells []*ui.Panel
	for i := 0; i < 4; i++ {
		c := ui.NewPanel()
		c.SetMinSize(10, 10)
		p.AddWidget(0, i, c)
		p.SetColumnStretch(i, 1<<62)
		p.SetRowStretch(0, 1<<62)
		cells = append(cells, c)
	}
	uitest.Show(t, form) // panicked with a divide by zero
	w := cells[0].Width()
	for i, c := range cells {
		if c.Width() < 50 || c.Width() > w+1 || c.Width() < w-1 {
			t.Errorf("column %d is %d wide, column 0 %d: equal weights share the width equally", i, c.Width(), w)
		}
	}
}

// A title added to a menu bar already in a form without a menu: the bar
// attached the nil menu to the form (a panic)
func TestMenuBarItemWithoutMenu(t *testing.T) {
	form := newForm(300, 200)
	bar := ui.NewMenuBar()
	form.SetMenuBar(bar)
	uitest.Show(t, form)
	item := bar.AddMenuItem("Help", nil)
	if item == nil || len(bar.Items()) != 1 {
		t.Fatalf("the title is not in the bar: %v, %d items", item, len(bar.Items()))
	}
}

// A dock area out of range in the saved settings (a corrupt file) put the
// panel in no area while it said shown, and it could never be shown again.
// The saved area is ignored; AddDock and MoveTo don't take one either.
func TestDockCorruptSavedArea(t *testing.T) {
	settings, err := ui.OpenSettingsFile(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	settings.Set("docks", map[string]any{
		"Docks": map[string]any{"Files": map[string]any{"Area": 7, "Shown": true}},
	})
	form := newForm(640, 420)
	content := ui.NewLabel("files")
	docks := ui.NewDockLayout(ui.NewPanel())
	d := docks.AddDock("Files", content, ui.DockRight)
	docks.RememberState(settings, "docks")
	form.Panel().AddWidget(0, 0, docks)
	uitest.Show(t, form)
	if d.Area() != ui.DockRight || content.Form() == nil {
		t.Errorf("after a saved area 7: area %d, in a form %v; want the right area, shown", d.Area(), content.Form() != nil)
	}

	d.MoveTo(ui.DockArea(-1))
	if d.Area() != ui.DockRight || content.Form() == nil {
		t.Errorf("after MoveTo(-1): area %d, in a form %v; want the right area, shown", d.Area(), content.Form() != nil)
	}

	bad := ui.NewLabel("bad")
	b := docks.AddDock("Bad", bad, ui.DockArea(42))
	if b.Area() != ui.DockLeft || bad.Form() == nil {
		t.Errorf("AddDock with area 42: area %d, in a form %v; want the left area, shown", b.Area(), bad.Form() != nil)
	}
}

// A message box for a parent widget that is not in a form yet: it was shown
// modal over a nil form (a panic). It is shown as a window of its own.
func TestMessageBoxParentNotInForm(t *testing.T) {
	open := map[*ui.Form]bool{}
	for _, f := range ui.OpenForms() {
		open[f] = true
	}
	ui.ShowMessageBox(ui.NewLabel("x"), "T", "text") // panicked
	var shown []*ui.Form
	for _, f := range ui.OpenForms() {
		if !open[f] {
			shown = append(shown, f)
		}
	}
	for _, f := range shown {
		f.Close()
	}
	if len(shown) != 1 {
		t.Errorf("%d new windows after the message box, want 1", len(shown))
	}
}
