package regression

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func TestSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app", "settings.json")
	s, err := ui.OpenSettingsFile(path)
	if err != nil {
		t.Fatal(err)
	}
	type point struct{ X, Y int }
	s.Set("theme", "dark")
	s.Set("size", 14)
	s.Set("where", point{3, 4})
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := ui.OpenSettingsFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var p point
	if again.String("theme", "") != "dark" || again.Int("size", 0) != 14 || !again.Get("where", &p) || p != (point{3, 4}) {
		t.Errorf("read back: %q %d %v", again.String("theme", ""), again.Int("size", 0), p)
	}
	if again.Bool("missing", true) != true || again.Get("missing", &p) {
		t.Error("a missing key")
	}
}

// The window opens where the user left it: its normal place and size, and
// maximized
func TestRememberState(t *testing.T) {
	uitest.SetScreen(1920, 1080, 1)
	path := filepath.Join(t.TempDir(), "settings.json")
	settings, _ := ui.OpenSettingsFile(path)

	form := newForm(400, 300)
	form.RememberState(settings, "main")
	form.Move(100, 120)
	w := uitest.Show(t, form)
	w.Resize(500, 350)
	form.Maximize()
	if got := form.State(); !got.Maximized || got.Width != 500 || got.X != 100 {
		t.Errorf("maximized state: %+v", got)
	}
	form.Close()

	settings, _ = ui.OpenSettingsFile(path)
	again := newForm(400, 300)
	again.RememberState(settings, "main")
	w2 := uitest.Show(t, again)
	if !again.IsMaximized() {
		t.Error("not maximized again")
	}
	again.Restore()
	if got := again.State(); got.Width != 500 || got.Height != 350 {
		t.Errorf("restored size: %+v", got)
	}
	_ = w2

	// A place off the screen (another monitor gone) comes back onto it
	off := newForm(400, 300)
	off.SetState(ui.FormState{X: 5000, Y: -300, Width: 400, Height: 300})
	uitest.Show(t, off)
	if x, y := off.Position(); x > 1920-400 || y < 0 {
		t.Errorf("off screen: at %d, %d", x, y)
	}
}

func TestSingleInstance(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	form := newForm(200, 100)
	w := uitest.Show(t, form)

	first, ok := ui.AcquireSingleInstance("nui-test-instance", nil)
	if !ok {
		t.Fatal("the first start must run")
	}
	var got []string
	first.OnActivate(func(args []string) { got = args })

	if _, ok := ui.AcquireSingleInstance("nui-test-instance", []string{"a.txt", "b c.txt"}); ok {
		t.Fatal("the second start must quit while the first runs")
	}
	deadline := time.Now().Add(3 * time.Second)
	for got == nil && time.Now().Before(deadline) {
		w.Wait(20 * time.Millisecond)
	}
	if len(got) != 2 || got[1] != "b c.txt" {
		t.Errorf("args passed: %q", got)
	}
	first.Close()
	again, ok := ui.AcquireSingleInstance("nui-test-instance", nil)
	if !ok {
		t.Fatal("a start after the first quit must run")
	}
	again.Close()
	form.Activate() // a shown form: nothing breaks
}

func newDockForm(settings *ui.Settings) (*ui.Form, *ui.DockLayout, map[string]*ui.DockWidget) {
	form := newForm(640, 420)
	editor := ui.NewTextBox()
	editor.SetMultiline(true)
	editor.SetText("the central widget")
	docks := ui.NewDockLayout(editor)
	byTitle := map[string]*ui.DockWidget{}
	files := ui.NewListView()
	files.SetItems([]string{"main.go", "dock.go"})
	byTitle["Files"] = docks.AddDock("Files", files, ui.DockLeft)
	byTitle["Outline"] = docks.AddDock("Outline", ui.NewLabel("func main"), ui.DockLeft)
	logs := ui.NewLogView()
	logs.AppendLine("built")
	byTitle["Log"] = docks.AddDock("Log", logs, ui.DockBottom)
	if settings != nil {
		docks.RememberState(settings, "docks")
	}
	form.Panel().AddWidget(0, 0, docks)
	return form, docks, byTitle
}

func TestDockLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	settings, _ := ui.OpenSettingsFile(path)
	form, docks, d := newDockForm(settings)
	w := uitest.Show(t, form)
	w.MatchGolden("docks")

	// Moved to the right from its menu, then closed and shown again
	w.Click(w.FindAll(func(x ui.Widgeter) bool { b, ok := x.(*ui.Button); return ok && b.Text() == "…" })[0])
	w.Click(w.Find("Dock Right"))
	if d["Files"].Area() != ui.DockRight {
		t.Errorf("Files in %v", d["Files"].Area())
	}
	if x, _ := d["Files"].Content().RectClientAreaOnWindow(); x < 320 || d["Files"].Content().Width() == 0 {
		t.Errorf("Files content at x %d, width %d: not at the right", x, d["Files"].Content().Width())
	}
	if x, _ := d["Outline"].Content().RectClientAreaOnWindow(); x > 100 || d["Outline"].Content().Width() == 0 {
		t.Errorf("Outline content at x %d, width %d: not at the left", x, d["Outline"].Content().Width())
	}
	w.MatchGolden("docks-moved")
	d["Log"].SetShown(false)
	if d["Log"].ToggleAction().Checked() {
		t.Error("the toggle action stays checked")
	}
	d["Log"].ToggleAction().Trigger()
	if !d["Log"].Shown() {
		t.Error("the toggle action doesn't show it")
	}
	d["Outline"].SetShown(false)
	docks.SetAreaSize(ui.DockRight, 200)
	form.Close()

	// The layout comes back
	settings, _ = ui.OpenSettingsFile(path)
	form2, docks2, d2 := newDockForm(settings)
	uitest.Show(t, form2)
	if d2["Files"].Area() != ui.DockRight || d2["Outline"].Shown() || !d2["Log"].Shown() {
		t.Errorf("restored: files %v, outline shown %v, log shown %v", d2["Files"].Area(), d2["Outline"].Shown(), d2["Log"].Shown())
	}
	if size := docks2.AreaSize(ui.DockRight); size < 190 || size > 210 {
		t.Errorf("right area %d, want 200", size)
	}
}
