package ui

import (
	"image/color"
	"math"
	"testing"
)

// A cutting plane shows the inside of a closed object in the section
// color, and what it cuts away is not picked
func TestRender3DClipPlanes(t *testing.T) {
	scene := NewScene3D()
	scene.Background = color.RGBA{0, 0, 0, 255}
	red := color.RGBA{200, 40, 40, 255}
	scene.Add(NewBoxMesh(1, 1, 1), Material{Color: red})
	cam := NewCamera(V3(0, 0, 5), V3(0, 0, 0))
	r := NewRenderer3D()
	r.Antialias = 1
	img := r.RenderImage(100, 100, scene, cam)
	front := img.RGBAAt(50, 50)
	if front.R < 100 || front.G > 80 {
		t.Fatalf("the front of the box is %v", front)
	}
	// Cut away the front half: the far wall is seen from inside
	section := color.RGBA{250, 220, 0, 255}
	scene.ClipPlanes = []Plane3D{{Point: V3(0, 0, 0), Normal: V3(0, 0, -1)}}
	scene.SectionColor = section
	img = r.RenderImage(100, 100, scene, cam)
	if c := img.RGBAAt(50, 50); c != section {
		t.Fatalf("the section is %v, want %v", c, section)
	}
	// Seen from the side, the cut box is half as wide
	side := NewCamera(V3(5, 0, 0), V3(0, 0, 0))
	side.Orthographic, side.OrthoHeight = true, 2
	img = r.RenderImage(100, 100, scene, side)
	// x of the picture is -z of the scene: z from -0.5 to 0 is x from 50
	// to 75
	if c := img.RGBAAt(60, 50); c.R < 100 {
		t.Fatalf("the kept half is %v", c)
	}
	if c := img.RGBAAt(35, 50); c != scene.Background {
		t.Fatalf("the cut half is %v", c)
	}
	hit, ok := scene.Pick(cam.Ray(50, 50, 100, 100))
	if !ok || math.Abs(hit.Point.Z+0.5) > 1e-9 {
		t.Fatalf("pick through the cut: %v %v", hit.Point, ok)
	}
	// The default section color is the object's, darker
	scene.SectionColor = color.RGBA{}
	img = r.RenderImage(100, 100, scene, cam)
	if c := img.RGBAAt(50, 50); c.R != 120 || c.G != 24 {
		t.Fatalf("the default section is %v", c)
	}
}

// The edges of a material outline the curved sides too, seen from where
// the camera is
func TestRender3DOutlineEdges(t *testing.T) {
	scene := NewScene3D()
	scene.Background = color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}
	o := scene.Add(NewCylinderMesh(1, 1, 2, 48, true), Material{Color: color.RGBA{120, 160, 220, 255}, EdgeColor: black, EdgeWidth: 2})
	cam := NewCamera(V3(0, 0, 10), V3(0, 0, 0))
	cam.Orthographic, cam.OrthoHeight = true, 4
	r := NewRenderer3D()
	r.Antialias = 1
	img := r.RenderImage(200, 200, scene, cam)
	// The sides of the cylinder at x = ±1: 50 pixels a unit
	dark := func(x, y int) bool {
		for dx := -2; dx <= 2; dx++ {
			if c := img.RGBAAt(x+dx, y); c.R < 60 && c.G < 60 && c.B < 60 {
				return true
			}
		}
		return false
	}
	if !dark(150, 100) || !dark(50, 100) {
		t.Fatalf("no outline at the sides: %v %v", img.RGBAAt(150, 100), img.RGBAAt(50, 100))
	}
	if dark(100, 100) {
		t.Fatal("an edge in the middle of the curved side")
	}
	// Turned, the outline follows
	o.Transform = RotateY4(0.7)
	img = r.RenderImage(200, 200, scene, cam)
	if !dark(150, 100) {
		t.Fatal("no outline of the turned cylinder")
	}
}
