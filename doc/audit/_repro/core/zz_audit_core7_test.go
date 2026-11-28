package ui

import (
	"path/filepath"
	"testing"

	"github.com/ipoluianov/nui/internal/platforms"
)

// A form with RememberState shown again after it was closed (a tray
// application) saves its state when it closes the second time
func TestAuditRememberStateReshown(t *testing.T) {
	platforms.SetHeadless()
	s, err := OpenSettingsFile(filepath.Join(t.TempDir(), "s.json"))
	if err != nil {
		t.Fatal(err)
	}
	f := NewForm()
	f.SetSize(400, 300)
	f.RememberState(s, "main")
	f.Show()
	f.Close()
	var st FormState
	s.Get("main", &st)
	t.Logf("after first close: %+v", st)
	f.Show()
	f.SystemHandle().(*platforms.HeadlessWindow).UserResize(640, 480)
	f.Close()
	s.Get("main", &st)
	if st.Width != 640 || st.Height != 480 {
		t.Errorf("after the second close the saved state is %+v, want 640x480", st)
	}
}

// The same for a DockLayout's RememberState
func TestAuditDockRememberStateReshown(t *testing.T) {
	platforms.SetHeadless()
	s, err := OpenSettingsFile(filepath.Join(t.TempDir(), "s.json"))
	if err != nil {
		t.Fatal(err)
	}
	f := NewForm()
	f.SetSize(800, 600)
	docks := NewDockLayout(NewPanel())
	d := docks.AddDock("Files", NewPanel(), DockLeft)
	docks.RememberState(s, "docks")
	f.Panel().AddWidget(0, 0, docks)
	f.Show()
	f.Close()
	f.Show()
	d.SetShown(false)
	f.Close()
	var st dockState
	s.Get("docks", &st)
	if st.Docks["Files"].Shown {
		t.Errorf("after the second close the saved dock state is %+v, want Files hidden", st.Docks)
	}
}

// A dock whose saved area is out of range (corrupted settings file)
func TestAuditDockBadArea(t *testing.T) {
	platforms.SetHeadless()
	s, err := OpenSettingsFile(filepath.Join(t.TempDir(), "s.json"))
	if err != nil {
		t.Fatal(err)
	}
	s.Set("docks", dockState{Docks: map[string]dockWidgetState{"Files": {Area: 7, Shown: true}}})
	f := NewForm()
	f.SetSize(800, 600)
	content := NewLabel("files")
	docks := NewDockLayout(NewPanel())
	d := docks.AddDock("Files", content, DockLeft)
	docks.RememberState(s, "docks")
	f.Panel().AddWidget(0, 0, docks)
	f.Show()
	defer f.Close()
	d.SetShown(false)
	d.SetShown(true)
	if d.Shown() && content.Form() == nil {
		t.Errorf("dock reports shown (area %d) but its content is in no form: it can never be shown again", d.Area())
	}
}
