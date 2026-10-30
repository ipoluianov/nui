package canvasdemo

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/ipoluianov/nui/ui/uitest"
)

// The tiles draw, the mouse lights a shape; with NUI_SNAPSHOT_DIR set the
// picture is saved there to be looked at
func TestDemo(t *testing.T) {
	form := NewForm()
	w := uitest.Show(t, form)
	d := form.Panel().Widgets()[0].(*demo)
	w.Wait(100e6)
	w.Move(uitest.At(d, 3*(tileW+gap)+gap+55, tileH+2*gap+30+80))
	img := w.Screenshot()
	if dir := os.Getenv("NUI_SNAPSHOT_DIR"); dir != "" {
		f, err := os.Create(filepath.Join(dir, "canvasdemo.png"))
		if err == nil {
			png.Encode(f, img)
			f.Close()
		}
	}
	p := uitest.At(d, 3*(tileW+gap)+gap+55, tileH+2*gap+30+80)
	if c := img.RGBAAt(p.X, p.Y); c.R < 200 || c.G < 150 || c.B > 100 {
		t.Fatalf("the hexagon under the mouse is %v, not lit", c)
	}
	form.Close()
}
