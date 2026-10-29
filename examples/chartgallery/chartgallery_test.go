package chartgallery

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// Every page shows, in both themes, and answers the mouse; with
// NUI_SNAPSHOT_DIR set its pictures are saved there to be looked at
func TestGalleryPages(t *testing.T) {
	dir := os.Getenv("NUI_SNAPSHOT_DIR")
	defer ui.ApplyDarkTheme()
	for _, theme := range []string{"light", "dark"} {
		if theme == "light" {
			ui.ApplyLightTheme()
		} else {
			ui.ApplyDarkTheme()
		}
		form := NewForm()
		w := uitest.Show(t, form)
		g := form.Panel().Widgets()[0].(*gallery)
		for i := 0; i < len(g.charts); i++ {
			g.tabs.SetCurrentIndex(i)
			w.Settle()
			c := g.current()
			w.Move(uitest.Center(c))
			w.Wheel(uitest.Center(c), 0, 1)
			img := w.Screenshot()
			if dir != "" {
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("chartgallery-%s-%d.png", theme, i)))
				if err == nil {
					png.Encode(f, img)
					f.Close()
				}
			}
		}
		form.Close()
	}
}
