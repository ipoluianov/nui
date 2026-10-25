package regression

import (
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// Where there is no system menu bar (here: headless, as on Windows and
// Linux), UseNativeMenuBar leaves the form's own bar
func TestNativeMenuBarFallback(t *testing.T) {
	ui.UseNativeMenuBar(true)
	defer ui.UseNativeMenuBar(false)
	form := newForm(300, 200)
	bar := ui.NewMenuBar()
	opened := 0
	bar.AddMenu("&File").AddItem("&Open", func() { opened++ })
	form.SetMenuBar(bar)
	w := uitest.Show(t, form)
	w.Click(w.Find("File"))
	w.Click(w.Find("Open"))
	if opened != 1 {
		t.Errorf("Open ran %d times", opened)
	}
	if !bar.IsVisible() || bar.Height() == 0 {
		t.Error("the form's bar is hidden")
	}
}
