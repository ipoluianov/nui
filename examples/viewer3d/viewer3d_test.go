package viewer3d

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// Every model shows and answers the mouse; with NUI_SNAPSHOT_DIR set the
// pictures are saved there to be looked at
func TestViewerModels(t *testing.T) {
	dir := os.Getenv("NUI_SNAPSHOT_DIR")
	form := NewForm()
	w := uitest.Show(t, form)
	v := form.Panel().Widgets()[0].(*viewer)
	for i := range models {
		v.model.SetSelectedIndex(i) // without the handler, as by the user
		v.rebuild(true)
		w.Settle()
		if len(v.scene.Objects)+len(v.scene.Points) == 0 {
			t.Fatalf("%s: an empty scene", models[i])
		}
		w.Drag(uitest.Center(v.view), uitest.At(v.view, v.view.Width()/2+40, v.view.Height()/2+20))
		w.Wheel(uitest.Center(v.view), 0, 1)
		img := w.Screenshot()
		if dir != "" {
			f, err := os.Create(filepath.Join(dir, fmt.Sprintf("viewer3d-%d.png", i)))
			if err == nil {
				png.Encode(f, img)
				f.Close()
			}
		}
	}
	// A click on an object of the scene picks it
	v.model.SetSelectedIndex(len(models) - 1)
	v.rebuild(true)
	v.view.SetView(ui.V3(0, 0.6, 0), 6, 0, 0.2)
	w.Click(uitest.Center(v.view))
	if v.selected == nil || v.selected.Name != "Cylinder" { // in front of the sphere, seen from +z
		t.Fatalf("picked %v", v.selected)
	}
	for _, c := range []*ui.Checkbox{v.wireframe, v.edges, v.flat, v.texture, v.ortho, v.smooth} {
		c.SetChecked(!c.Checked())
		v.rebuild(false)
		w.Settle()
	}
	form.Close()
}

func TestViewerLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pyramid.obj")
	os.WriteFile(path, []byte("v 0 1 0\nv -1 0 -1\nv 1 0 -1\nv 1 0 1\nv -1 0 1\nf 1 5 4\nf 1 4 3\nf 1 3 2\nf 1 2 5\nf 2 3 4 5\n"), 0o644)
	form := NewForm()
	w := uitest.Show(t, form)
	v := form.Panel().Widgets()[0].(*viewer)
	v.load(path)
	w.Settle()
	if v.file == nil || v.file.TriangleCount() != 6 || v.scene.Objects[0].Name != "pyramid.obj" {
		t.Fatal("the model was not loaded")
	}
	form.Close()
}
