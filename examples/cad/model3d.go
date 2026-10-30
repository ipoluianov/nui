package cad

import (
	"fmt"
	"image/color"
	"math"

	"github.com/ipoluianov/nui/ui"
)

// The 3D model of the part: the plate extruded from the outline of its
// front view, a bushing turned from its half section, four bolts; cut by
// a plane to look inside

func (a *app) build3DPage() ui.Widgeter {
	page := ui.NewPanel()
	page.SetPanelPadding(0)
	a.scene = ui.NewScene3D()
	a.buildScene()
	a.view3d = ui.NewView3D(a.scene)
	a.view3d.SetAxesGizmo(true)
	a.view3d.SetView(ui.V3(0, 10, 0), 300, math.Pi/5, math.Pi/6)
	a.view3d.FitToScene()
	a.view3d.OnPick.Connect(a.pick3D)
	a.view3d.SetOverlay(a.overlay3D)
	page.AddWidget(0, 0, a.view3d)

	side := page.AddPanel(0, 1)
	side.SetMinWidth(210)
	side.SetMaxWidth(210)
	cut := ui.NewGroupBox("Section")
	side.AddWidget(0, 0, cut)
	a.section = ui.NewCheckbox("Cut along the plate")
	a.section.SetOnStateChanged(a.updateSection)
	cut.AddWidget(0, 0, a.section)
	a.sectionAt = ui.NewSlider()
	a.sectionAt.SetRange(-60, 60)
	a.sectionAt.SetStep(1)
	a.sectionAt.SetValue(0)
	a.sectionAt.SetOnValueChanged(a.updateSection)
	cut.AddWidget(1, 0, a.sectionAt)

	look := ui.NewGroupBox("View")
	side.AddWidget(1, 0, look)
	a.edges3d = ui.NewCheckbox("Edges")
	a.edges3d.SetChecked(true)
	a.edges3d.SetOnStateChanged(func() { a.buildScene(); a.view3d.Update() })
	look.AddWidget(0, 0, a.edges3d)
	a.ortho3 = ui.NewCheckbox("Orthographic")
	a.ortho3.SetOnStateChanged(func() { a.view3d.SetOrthographic(a.ortho3.Checked()) })
	look.AddWidget(1, 0, a.ortho3)
	views := look.AddPanel(2, 0)
	views.SetPanelPadding(0)
	for i, v := range []struct {
		name       string
		yaw, pitch float64
	}{{"Front", 0, 0}, {"Top", 0, math.Pi / 2}, {"Right", math.Pi / 2, 0}, {"Iso", math.Pi / 4, math.Asin(1 / math.Sqrt(3))}} {
		views.AddButton(i/2, i%2, v.name, func() {
			target, dist, _, _ := a.view3d.View()
			a.view3d.SetView(target, dist, v.yaw, v.pitch)
		})
	}
	look.AddButton(3, 0, "Fit", a.view3d.FitToScene)
	side.AddLabel(2, 0, "Left: turn   Right: move\nWheel: zoom   Click: pick\nThe axes at the corner:\nclick one to look along it")
	side.AddVSpacer(3, 0)
	return page
}

// The parts, made once
var (
	plateMesh, bushingMesh, boltMesh *ui.Mesh
)

func partMeshes() {
	if plateMesh != nil {
		return
	}
	// The outline of the front view, centered: the corners rounded
	var outline []ui.PointF
	corner := func(cx, cy, from float64) {
		for i := 0; i <= 12; i++ {
			t := from + float64(i)/12*math.Pi/2
			outline = append(outline, ui.PointF{X: cx + plateR*math.Cos(t), Y: cy + plateR*math.Sin(t)})
		}
	}
	hw, hh := plateW/2-plateR, plateH/2-plateR
	corner(hw, -hh, -math.Pi/2)
	corner(hw, hh, 0)
	corner(-hw, hh, math.Pi/2)
	corner(-hw, -hh, math.Pi)
	hole := func(cx, cy, r float64, n int) []ui.PointF {
		pts := make([]ui.PointF, n)
		for i := range pts {
			t := 2 * math.Pi * float64(i) / float64(n)
			pts[i] = ui.PointF{X: cx + r*math.Cos(t), Y: cy + r*math.Sin(t)}
		}
		return pts
	}
	holes := [][]ui.PointF{hole(0, 0, boreR, 64)}
	for _, sx := range []float64{-1, 1} {
		for _, sy := range []float64{-1, 1} {
			holes = append(holes, hole(sx*boltX, sy*boltY, boltR, 32))
		}
	}
	plateMesh = ui.NewExtrudeMesh(outline, holes, plateT)
	// The sketch is in x, y; the plate lies on the floor, its thickness up
	plateMesh.Transform(ui.RotateX4(-math.Pi / 2))

	// The bushing: the half section, counterclockwise (radius right,
	// height up): a sleeve through the bore with a collar on the plate
	bushingMesh = ui.NewLatheMesh([]ui.PointF{
		{X: 12, Y: -8}, {X: 19.5, Y: -8}, {X: 19.5, Y: plateT}, {X: 28, Y: plateT},
		{X: 28, Y: plateT + 8}, {X: 26, Y: plateT + 10}, {X: 12, Y: plateT + 10}, {X: 12, Y: -8},
	}, 96)

	// A bolt: a hexagon head and a shank, turned
	bolt := ui.NewLatheMesh([]ui.PointF{
		{X: 0, Y: -14}, {X: 5.5, Y: -14}, {X: 5.8, Y: -13.5}, {X: 5.8, Y: plateT}, {X: 0, Y: plateT},
	}, 32)
	head := ui.NewCylinderMesh(10, 10, 7, 6, true)
	head.Transform(ui.Translate4(ui.V3(0, plateT+3.5, 0)))
	bolt.Append(head)
	boltMesh = bolt
}

// buildScene makes the scene of the part
func (a *app) buildScene() {
	partMeshes()
	s := a.scene
	s.Clear()
	edge := color.RGBA{}
	if a.edges3d == nil || a.edges3d.Checked() {
		edge = color.RGBA{25, 28, 32, 255}
	}
	plate := s.Add(plateMesh, ui.Material{Color: color.RGBA{170, 176, 186, 255}, Specular: 0.35, EdgeColor: edge})
	plate.Name = "Plate 160×100×12"
	bushing := s.Add(bushingMesh, ui.Material{Color: color.RGBA{210, 165, 70, 255}, Specular: 0.6, Shininess: 40, EdgeColor: edge})
	bushing.Name = "Bushing Ø40/Ø24"
	for _, sx := range []float64{-1, 1} {
		for _, sy := range []float64{-1, 1} {
			b := s.Add(boltMesh, ui.Material{Color: color.RGBA{90, 96, 108, 255}, Specular: 0.5, EdgeColor: edge})
			// The plate's y went to -z
			b.Transform = ui.Translate4(ui.V3(sx*boltX, 0, -sy*boltY))
			b.Name = "Bolt M12"
		}
	}
	s.AddGrid(200, 10, color.RGBA{128, 128, 128, 90})
	// The floor under the bolts, not in the plane of the plate's bottom
	for i := range s.Lines {
		s.Lines[i].A.Y, s.Lines[i].B.Y = -15, -15
	}
	s.SectionColor = color.RGBA{220, 70, 60, 255}
	a.picked = nil
	a.updateSection()
}

// updateSection cuts the model by the plane square to x at the slider
func (a *app) updateSection() {
	a.scene.ClipPlanes = nil
	if a.section != nil && a.section.Checked() {
		x := a.sectionAt.Value()
		a.scene.ClipPlanes = []ui.Plane3D{{Point: ui.V3(x, 0, 0), Normal: ui.V3(-1, 0, 0)}}
	}
	if a.view3d != nil {
		a.view3d.Update()
	}
}

// pick3D marks the part clicked
func (a *app) pick3D(hit ui.Hit3D) {
	if a.picked == hit.Object {
		a.picked = nil
	} else {
		a.picked = hit.Object
	}
	for _, o := range a.scene.Objects {
		o.Material.EdgeWidth = 1
		if o == a.picked {
			o.Material.EdgeWidth = 3
		}
	}
	a.view3d.Update()
}

func (a *app) overlay3D(cnv *ui.Canvas) {
	st := a.view3d.Renderer().Stats()
	text := fmt.Sprintf("%d triangles   %.1f ms", st.Triangles, float64(st.Duration.Microseconds())/1000)
	if a.picked != nil {
		text = a.picked.Name + "   " + text
	}
	tag(cnv, 10, 10, text, color.RGBA{230, 230, 230, 255})
}
