// Package viewer3d is a 3D model viewer: the shapes of nui, the models of
// OBJ and STL files, a surface of a function and a cloud of points, seen
// in a View3D that the mouse turns, moves and zooms.
package viewer3d

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/ipoluianov/nui/ui"
)

// models are the contents of the Model list, in its order
var models = []string{"Torus", "Sphere", "Box", "Cylinder", "Cone", "Surface", "Point cloud", "Scene"}

// viewer is the window's content: the view and the controls beside it
type viewer struct {
	ui.Widget

	view   *ui.View3D
	scene  *ui.Scene3D
	status *ui.Label

	model                   *ui.ComboBox
	colors                  *ui.ComboBox
	wireframe, edges, flat  *ui.Checkbox
	texture, grid, axes     *ui.Checkbox
	ortho, smooth, rotate   *ui.Checkbox
	shine                   *ui.Slider
	file                    *ui.Mesh // the model opened from a file, if any
	fileName                string
	checker                 image.Image
	selected                *ui.Object3D
	selectedMat             ui.Material
	lastTriangles, lastTime string
}

var palette = []struct {
	name string
	col  color.RGBA
}{
	{"Orange", color.RGBA{230, 120, 40, 255}},
	{"Blue", color.RGBA{60, 130, 230, 255}},
	{"Green", color.RGBA{70, 180, 100, 255}},
	{"Gray", color.RGBA{190, 190, 195, 255}},
	{"Glass", color.RGBA{120, 180, 240, 110}},
}

// NewForm returns the 3D Viewer
func NewForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("3D Viewer")
	form.SetSize(1100, 740)
	form.Panel().AddWidget(0, 0, newViewer())
	return form
}

func newViewer() *viewer {
	v := &viewer{}
	v.InitWidget()
	v.checker = checkerImage()

	v.scene = ui.NewScene3D()
	v.view = ui.NewView3D(v.scene)
	v.view.SetOverlay(v.drawOverlay)
	v.view.OnPick.Connect(v.pick)
	v.AddWidget(0, 0, v.view)

	side := v.AddPanel(0, 1)
	side.SetMinWidth(250)
	side.SetMaxWidth(250)

	shape := ui.NewGroupBox("Model")
	side.AddWidget(0, 0, shape)
	v.model = ui.NewComboBox()
	v.model.SetItems(models)
	v.model.SetSelectedIndex(0)
	v.model.SetOnSelectedIndexChanged(func() { v.file = nil; v.rebuild(true) })
	shape.AddWidget(0, 0, v.model)
	shape.AddButton(1, 0, "Open OBJ or STL...", v.open)

	look := ui.NewGroupBox("Material")
	side.AddWidget(1, 0, look)
	v.colors = ui.NewComboBox()
	names := make([]string, len(palette))
	for i, p := range palette {
		names[i] = p.name
	}
	v.colors.SetItems(names)
	v.colors.SetSelectedIndex(0)
	v.colors.SetOnSelectedIndexChanged(func() { v.rebuild(false) })
	look.AddWidget(0, 0, v.colors)
	v.wireframe = v.check(look, 1, "Wireframe", false)
	v.edges = v.check(look, 2, "Edges", false)
	v.flat = v.check(look, 3, "Flat shading", false)
	v.texture = v.check(look, 4, "Checker texture", false)
	look.AddLabel(5, 0, "Shine")
	v.shine = ui.NewSlider()
	v.shine.SetRange(0, 1)
	v.shine.SetStep(0.05)
	v.shine.SetValue(0.5)
	v.shine.SetOnValueChanged(func() { v.rebuild(false) })
	look.AddWidget(6, 0, v.shine)

	scn := ui.NewGroupBox("View")
	side.AddWidget(2, 0, scn)
	v.grid = v.check(scn, 0, "Grid", true)
	v.axes = v.check(scn, 1, "Axes", true)
	v.ortho = v.check(scn, 2, "Orthographic", false)
	v.smooth = v.check(scn, 3, "Smooth edges", true)
	v.rotate = v.check(scn, 4, "Turn around", false)
	buttons := scn.AddPanel(5, 0)
	buttons.AddButton(0, 0, "Fit", v.view.FitToScene)
	buttons.AddButton(0, 1, "Reset", v.view.ResetView)
	scn.AddButton(6, 0, "Save picture...", v.save)

	side.AddLabel(3, 0, "Left: turn   Right: move\nWheel: zoom   Click: pick\nDouble click: center\nKeys: arrows, + -, 1 3 7, O, Home")
	side.AddVSpacer(4, 0)
	v.status = v.AddLabel(1, 0, "")

	v.AddTimer(500, v.showStats)
	v.rebuild(true)
	return v
}

func (v *viewer) check(box *ui.GroupBox, row int, text string, on bool) *ui.Checkbox {
	c := ui.NewCheckbox(text)
	c.SetChecked(on)
	c.SetOnStateChanged(func() { v.rebuild(false) })
	box.AddWidget(row, 0, c)
	return c
}

// material returns the material the controls describe
func (v *viewer) material() ui.Material {
	m := ui.Material{
		Color:       palette[max(0, v.colors.SelectedIndex())].col,
		Specular:    v.shine.Value(),
		Shininess:   48,
		Flat:        v.flat.Checked(),
		Wireframe:   v.wireframe.Checked(),
		DoubleSided: true,
	}
	if v.edges.Checked() {
		m.EdgeColor = color.RGBA{20, 20, 20, 255}
	}
	if v.texture.Checked() {
		m.Texture = v.checker
		m.Color = color.RGBA{255, 255, 255, m.Color.A}
	}
	return m
}

// rebuild makes the scene again of the controls; fit aims the view at it
func (v *viewer) rebuild(fit bool) {
	v.scene.Clear()
	v.selected = nil
	mat := v.material()
	name := models[max(0, v.model.SelectedIndex())]
	switch {
	case v.file != nil:
		name = v.fileName
		v.scene.Add(v.file, mat).Name = name
	case name == "Surface":
		m := ui.NewSurfaceMesh(-3, 3, -3, 3, 90, 90, func(x, z float64) float64 {
			r := math.Hypot(x, z)*2.5 + 1e-9
			return math.Sin(r) / r * 1.5
		})
		m.ColorByHeight([]ui.GradientStop{
			{Offset: 0, Color: color.RGBA{40, 50, 150, 255}},
			{Offset: 0.5, Color: color.RGBA{60, 190, 160, 255}},
			{Offset: 1, Color: color.RGBA{250, 220, 90, 255}},
		})
		if !v.texture.Checked() {
			mat.Color = color.RGBA{255, 255, 255, mat.Color.A}
		}
		v.scene.Add(m, mat).Name = "sin(r)/r"
	case name == "Point cloud":
		rnd := rand.New(rand.NewSource(1))
		for i := 0; i < 20000; i++ {
			// A spiral galaxy: arms of points around the center
			arm := float64(i%3) * 2 * math.Pi / 3
			r := rnd.Float64() * 2.5
			a := arm + r*1.8 + rnd.NormFloat64()*0.25
			p := ui.V3(r*math.Cos(a), rnd.NormFloat64()*0.08*(2.6-r), r*math.Sin(a))
			t := r / 2.5
			c := color.RGBA{uint8(255 - 120*t), uint8(200 - 60*t), uint8(120 + 135*t), 255}
			v.scene.AddPoint(p, c, 2)
		}
	case name == "Scene":
		v.addScene(mat)
	default:
		v.scene.Add(ui.NewShapeMesh(name), mat).Name = name
	}
	if v.grid.Checked() {
		v.scene.AddGrid(6, 0.5, color.RGBA{128, 128, 128, 120})
	}
	if v.axes.Checked() {
		v.scene.AddAxes(1.8)
	}
	v.view.SetOrthographic(v.ortho.Checked())
	aa := 1
	if v.smooth.Checked() {
		aa = 2
	}
	v.view.SetAntialias(aa)
	if v.rotate.Checked() {
		v.view.SetAutoRotate(0.6)
	} else {
		v.view.SetAutoRotate(0)
	}
	if fit {
		v.view.SetScene(v.scene)
	}
	v.view.Update()
}

// addScene adds a few objects to pick and to look at
func (v *viewer) addScene(mat ui.Material) {
	place := func(m *ui.Mesh, col color.RGBA, name string, at ui.Vec3) {
		mm := mat
		if mm.Texture == nil {
			mm.Color = col
		}
		o := v.scene.Add(m, mm)
		o.Name = name
		o.Transform = ui.Translate4(at)
	}
	place(ui.NewTorusMesh(0.7, 0.25, 48, 24), color.RGBA{230, 120, 40, 255}, "Torus", ui.V3(-1.6, 0.25, 0))
	place(ui.NewSphereMesh(0.6, 40, 24), color.RGBA{60, 130, 230, 255}, "Sphere", ui.V3(0, 0.6, 0))
	place(ui.NewBoxMesh(0.9, 0.9, 0.9), color.RGBA{70, 180, 100, 255}, "Box", ui.V3(1.6, 0.45, 0))
	place(ui.NewConeMesh(0.5, 1.2, 40), color.RGBA{200, 70, 160, 255}, "Cone", ui.V3(0, 0.6, -1.6))
	place(ui.NewCylinderMesh(0.4, 0.4, 1, 40, true), color.RGBA{220, 200, 70, 255}, "Cylinder", ui.V3(0, 0.5, 1.6))
}

// pick shows the object clicked by drawing its edges
func (v *viewer) pick(hit ui.Hit3D) {
	if v.selected != nil {
		v.selected.Material = v.selectedMat
	}
	if v.selected == hit.Object {
		v.selected = nil
		v.view.Update()
		return
	}
	v.selected, v.selectedMat = hit.Object, hit.Object.Material
	hit.Object.Material.EdgeColor = color.RGBA{255, 210, 0, 255}
	hit.Object.Material.EdgeWidth = 2
	v.view.Update()
}

// drawOverlay writes the name of the object picked over the view
func (v *viewer) drawOverlay(cnv *ui.Canvas) {
	if v.selected == nil {
		return
	}
	text := v.selected.Name
	cnv.SetFontSize(13)
	cnv.SetFontBold(true)
	w, _, _ := ui.MeasureText("", 13, text)
	cnv.DrawShadow(12, 12, float64(w+20), 28, 6, 8, color.RGBA{0, 0, 0, 120})
	cnv.SetColor(color.RGBA{30, 30, 34, 230})
	cnv.FillRoundedRect(10, 10, w+20, 28, 6)
	cnv.SetColor(color.RGBA{255, 210, 0, 255})
	cnv.SetVAlign(ui.VAlignCenter)
	cnv.DrawText(20, 10, w+10, 28, text)
}

func (v *viewer) showStats() {
	st := v.view.Renderer().Stats()
	text := fmt.Sprintf("%d triangles, %d lines, %d points   %.1f ms a frame",
		st.Triangles, st.Lines, st.Points, float64(st.Duration.Microseconds())/1000)
	v.status.SetText(text)
}

func (v *viewer) open() {
	opts := ui.OpenFileDialogOptions{
		Title:   "Open a 3D model",
		Filters: []ui.FileDialogFilter{{DisplayName: "3D models", Patterns: []string{"*.obj", "*.stl"}}},
	}
	v.Form().ShowOpenFileDialog(opts, func(paths []string, err error) {
		if errors.Is(err, ui.ErrNoFileDialog) || err != nil || len(paths) == 0 {
			return
		}
		v.load(paths[0])
	})
}

// load shows the model of the file
func (v *viewer) load(path string) {
	m, err := ui.LoadMeshFile(path)
	if err != nil {
		ui.ShowMessageBox(v, "Open", err.Error())
		return
	}
	m.Center(3)
	// On the floor
	lo, _ := m.Bounds()
	m.Transform(ui.Translate4(ui.V3(0, -lo.Y, 0)))
	v.file, v.fileName = m, filepath.Base(path)
	v.rebuild(true)
}

func (v *viewer) save() {
	opts := ui.SaveFileDialogOptions{
		Title:           "Save the picture",
		DefaultFileName: "view.png",
		Filters:         []ui.FileDialogFilter{{DisplayName: "PNG", Patterns: []string{"*.png"}}},
	}
	v.Form().ShowSaveFileDialog(opts, func(path string, err error) {
		if errors.Is(err, ui.ErrNoFileDialog) || err != nil || path == "" {
			return
		}
		img := v.view.RenderImage(v.view.Width()*2, v.view.Height()*2)
		f, err := os.Create(path)
		if err == nil {
			err = png.Encode(f, img)
			f.Close()
		}
		if err != nil {
			ui.ShowMessageBox(v, "Save", err.Error())
		}
	})
}

// checkerImage is the texture: squares of two colors with a grid
func checkerImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			c := color.RGBA{240, 240, 240, 255}
			if (x/32+y/32)%2 == 1 {
				c = color.RGBA{230, 90, 60, 255}
			}
			if x%32 == 0 || y%32 == 0 {
				c = color.RGBA{40, 40, 40, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}
