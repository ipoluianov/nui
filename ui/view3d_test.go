package ui_test

import (
	"image/color"
	"math"
	"os"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func newView3DForm(t *testing.T) (*ui.View3D, *uitest.Window) {
	scene := ui.NewScene3D()
	scene.Background = color.RGBA{10, 10, 10, 255}
	box := scene.Add(ui.NewBoxMesh(1, 1, 1), ui.Material{Color: color.RGBA{220, 60, 60, 255}})
	box.Name = "box"
	view := ui.NewView3D(scene)
	form := ui.NewForm()
	form.SetSize(400, 300)
	form.Panel().AddWidget(0, 0, view)
	w := uitest.Show(t, form)
	return view, w
}

// The view shows the scene, the mouse turns, moves and zooms it, a click
// picks, the keys work
func TestView3DControls(t *testing.T) {
	view, w := newView3DForm(t)
	img := w.Screenshot()
	cx, cy := uitest.Center(view).X, uitest.Center(view).Y
	if c := img.RGBAAt(cx, cy); c.R < 60 || c.R <= c.G {
		t.Fatalf("the box at the center is %v", c)
	}

	_, d0, yaw0, pitch0 := view.View()
	w.Drag(uitest.Center(view), uitest.At(view, view.Width()/2+60, view.Height()/2+30))
	_, _, yaw1, pitch1 := view.View()
	if yaw1 >= yaw0 || pitch1 <= pitch0 {
		t.Fatalf("the drag did not turn the view: yaw %v -> %v, pitch %v -> %v", yaw0, yaw1, pitch0, pitch1)
	}

	w.Wheel(uitest.Center(view), 0, 2)
	if _, d1, _, _ := view.View(); d1 >= d0 {
		t.Fatalf("the wheel up did not zoom in: %v -> %v", d0, d1)
	}

	// A right drag moves the target
	t0, _, _, _ := view.View()
	a, b := uitest.Center(view), uitest.At(view, view.Width()/2+40, view.Height()/2)
	w.Move(a)
	w.Press(ui.MouseButtonRight, a)
	w.Move(b)
	w.Release(ui.MouseButtonRight, b)
	if t1, _, _, _ := view.View(); t1 == t0 {
		t.Fatal("the right drag did not pan")
	}

	view.ResetView()
	var picked string
	view.OnPick.Connect(func(h ui.Hit3D) { picked = h.Object.Name })
	w.Click(uitest.Center(view))
	if picked != "box" {
		t.Fatalf("the click picked %q", picked)
	}
	picked = ""
	w.Click(uitest.At(view, 3, 3))
	if picked != "" {
		t.Fatalf("a click on nothing picked %q", picked)
	}

	// Keys: the top view, the orthographic projection, reset
	w.Key(ui.Key7, ui.KeyModifiers{})
	if _, _, _, p := view.View(); p < math.Pi/2-0.01 {
		t.Fatalf("7 is not the top view: pitch %v", p)
	}
	w.Key(ui.KeyO, ui.KeyModifiers{})
	if !view.Orthographic() {
		t.Fatal("O did not switch to orthographic")
	}
	w.Key(ui.KeyHome, ui.KeyModifiers{})
	if _, _, y, p := view.View(); y != yaw0 || p != pitch0 {
		t.Fatal("Home did not reset the view")
	}
	img = w.Screenshot()
	if c := img.RGBAAt(cx, cy); c.R < 60 {
		t.Fatalf("the orthographic box at the center is %v", c)
	}
}

func TestView3DRenderImage(t *testing.T) {
	view, _ := newView3DForm(t)
	img := view.RenderImage(120, 90)
	if c := img.RGBAAt(60, 45); c.R < 60 {
		t.Fatalf("the picture's center %v", c)
	}
	x, y, ok := view.Project(ui.V3(0, 0, 0))
	if !ok || math.Abs(x-float64(view.Width())/2) > 1 || math.Abs(y-float64(view.Height())/2) > 1 {
		t.Fatalf("the target is projected to %v, %v", x, y)
	}
}

func TestView3DXML(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/tri.obj"
	if err := os.WriteFile(path, []byte("v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	form := ui.NewForm()
	err := form.Panel().SetLayout(`<column>
		<view3d id="shape" shape="torus" color="#40a0e0" axes="true" grid="true" ortho="true"/>
		<view3d id="model" model="`+path+`" wireframe="true"/>
	</column>`, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	shape := form.Panel().FindWidgetByName("shape").(*ui.View3D)
	if !shape.Orthographic() || len(shape.Scene().Objects) != 1 || shape.Scene().Objects[0].Material.Color != (color.RGBA{0x40, 0xa0, 0xe0, 255}) {
		t.Fatalf("view3d attributes: %+v", shape.Scene().Objects[0].Material)
	}
	if len(shape.Scene().Lines) == 0 || len(shape.Scene().Labels) != 3 {
		t.Fatal("no axes or grid")
	}
	model := form.Panel().FindWidgetByName("model").(*ui.View3D)
	if m := model.Scene().Objects[0].Mesh; m == nil || m.TriangleCount() != 1 {
		t.Fatal("the model was not loaded")
	}
	err = form.Panel().SetLayout(`<view3d shape="blob"/>`, nil, nil)
	if err == nil {
		t.Fatal("an unknown shape gave no error")
	}
}

// The view draws the scene again only when something it shows changed
func TestView3DCache(t *testing.T) {
	view, w := newView3DForm(t)
	r := view.Renderer()
	w.Screenshot()
	first := r.Stats().Duration
	view.Update()
	w.Settle()
	w.Screenshot()
	if r.Stats().Duration != first {
		t.Fatal("drawn again with nothing changed")
	}
	obj := view.Scene().Objects[0]
	obj.Material.Color = color.RGBA{0, 200, 0, 255}
	view.Update()
	img := w.Screenshot()
	c := uitest.Center(view)
	if px := img.RGBAAt(c.X, c.Y); px.G < 60 || px.R > px.G {
		t.Fatalf("the new color is not shown: %v", px)
	}
	// A mesh changed in place, told by Changed
	m := obj.Mesh
	for i := range m.Positions {
		m.Positions[i] = m.Positions[i].Scale(0.1)
	}
	m.Changed()
	view.Update()
	img = w.Screenshot()
	if px := img.RGBAAt(c.X+40, c.Y); px.G > 60 && px.G > px.R {
		t.Fatalf("the shrunk box still covers %v", px)
	}
}

// A click on an axis of the gizmo looks along it, a second one from the
// other side
func TestView3DGizmo(t *testing.T) {
	view, w := newView3DForm(t)
	view.SetAxesGizmo(true)
	// The red axis X is drawn in the lower left corner
	img := w.Screenshot()
	at := uitest.At(view, 0, 0)
	found := false
	for y := view.Height() - 84; y < view.Height(); y++ {
		for x := 0; x < 84; x++ {
			if c := img.RGBAAt(at.X+x, at.Y+y); c.R > 200 && c.G < 80 && c.B < 80 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no red axis drawn")
	}
	view.SetView(ui.Vec3{}, 5, 0.3, 0.2)
	w.Screenshot()
	p := gizmoEnd(view, 0)
	w.Click(uitest.At(view, int(p.X), int(p.Y)))
	if _, _, yaw, pitch := view.View(); math.Abs(yaw-math.Pi/2) > 1e-9 || pitch != 0 {
		t.Fatalf("X: yaw %v pitch %v", yaw, pitch)
	}
	w.Screenshot()
	p = gizmoEnd(view, 0)
	w.Click(uitest.At(view, int(p.X), int(p.Y)))
	if _, _, yaw, _ := view.View(); math.Abs(yaw-3*math.Pi/2) > 1e-9 {
		t.Fatalf("X again: yaw %v", yaw)
	}
}

// gizmoEnd returns where the end of the axis is drawn, as the gizmo does
func gizmoEnd(view *ui.View3D, axis int) ui.PointF {
	dir := [3]ui.Vec3{{X: 1}, {Y: 1}, {Z: 1}}[axis]
	d := view.Camera().ViewMatrix().MulDir(dir)
	return ui.PointF{X: 42 + d.X*28, Y: float64(view.Height()) - 42 - d.Y*28}
}
