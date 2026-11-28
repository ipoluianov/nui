package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// A 3D label at a NaN, infinite or very far point is skipped: before, its
// projected point went to DrawText as an int near ±2^63 and the glyph
// blitter panicked (slice bounds out of range)
func TestMisc3DLabelFarPointSkipped(t *testing.T) {
	for _, p := range []Vec3{
		V3(math.NaN(), 0, 0),
		V3(math.Inf(1), 0, 0),
		V3(1e300, 0, 0),
		V3(0, -1e300, 0),
		V3(1e12, 1e12, 0),
	} {
		sc := NewScene3D()
		sc.AddLabel(p, "x", color.RGBA{0, 0, 0, 255})
		sc.AddLabel(V3(0, 0, 0), "o", color.RGBA{0, 0, 0, 255})
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("label at %v: render panicked: %v", p, r)
				}
			}()
			NewRenderer3D().RenderImage(64, 64, sc, NewCamera(V3(0, 0, 5), V3(0, 0, 0)))
		}()
	}
	for _, c := range []struct {
		v    float64
		want bool
	}{{math.NaN(), false}, {math.Inf(-1), false}, {1e300, false}, {-labelFar - 1, false}, {-10, true}, {70, true}} {
		if got := labelInReach(c.v, 64); got != c.want {
			t.Errorf("labelInReach(%v, 64) = %v, want %v", c.v, got, c.want)
		}
	}
}

func faceCacheLen() int {
	fontsMu.RLock()
	defer fontsMu.RUnlock()
	return len(faceCache)
}

// The face cache stays bounded: text drawn under a smoothly changing
// transform asked for a new face at each scale, and each NaN size added one
func TestMiscFaceCacheBounded(t *testing.T) {
	cnv, _ := NewImageCanvas(200, 100, 1)
	for i := 0; i < 2000; i++ {
		cnv.Save()
		k := 1 + float64(i)*0.0005
		cnv.SetTransform(ScalingMatrix(k, k))
		cnv.SetFontSize(12)
		cnv.DrawText(0, 0, 100, 20, "Label")
		cnv.Restore()
		MeasureText(FontFamilySans, 10+float64(i)*0.001, "Label")
	}
	if n := faceCacheLen(); n > maxFaceCache {
		t.Errorf("faceCache has %d faces after a zoom animation, want at most %d", n, maxFaceCache)
	}

	before := faceCacheLen()
	for i := 0; i < 100; i++ {
		if _, err := getFaceEntry(FontFamilySans, FontRegular, math.NaN()); err == nil {
			t.Fatal("getFaceEntry(NaN) gave no error")
		}
		MeasureText(FontFamilySans, math.NaN(), "Label")
		MeasureText(FontFamilySans, math.Inf(1), "Label")
	}
	if n := faceCacheLen(); n > before {
		t.Errorf("NaN and Inf sizes added %d faces", n-before)
	}

	// Sizes that draw the same (1/64 px apart at most) share a face
	a, err2 := getFaceEntry(FontFamilySans, FontRegular, 13)
	if err2 != nil {
		t.Fatal(err2)
	}
	b, err3 := getFaceEntry(FontFamilySans, FontRegular, 13+1e-6)
	if err3 != nil {
		t.Fatal(err3)
	}
	if a != b {
		t.Error("sizes 13 and 13+1e-6 got different faces")
	}
}

// ToolButton.SetEnabled(false) under a disabled parent is kept: before, it
// compared with Enabled() (false already, from the parent) and returned
func TestMiscToolButtonDisabledUnderDisabledParent(t *testing.T) {
	panel := NewPanel()
	b := NewToolButton(image.NewRGBA(image.Rect(0, 0, 16, 16)), "tip", func() {})
	panel.AddWidget(0, 0, b)
	newTestForm(t, panel)
	panel.SetEnabled(false)
	b.SetEnabled(false)
	panel.SetEnabled(true)
	if b.Enabled() {
		t.Fatal("ToolButton.SetEnabled(false) under a disabled parent was ignored")
	}
	b.SetEnabled(true)
	if !b.Enabled() {
		t.Fatal("SetEnabled(true) did not enable the button again")
	}
	// The image follows the parent at paint time
	panel.SetEnabled(false)
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	b.Form().paintScaled(img, 1)
	if b.ButtonImage.img != b.imgDisabled {
		t.Error("the button under a disabled parent paints the enabled image")
	}
	panel.SetEnabled(true)
	b.Form().paintScaled(img, 1)
	if b.ButtonImage.img != b.img {
		t.Error("the button under an enabled parent paints the disabled image")
	}
}

// A NaN zoom, a zoom range with 0 or a NaN point leave the offset finite
func TestMiscZoomViewNaN(t *testing.T) {
	z := NewZoomView()
	newTestForm(t, z)
	z.SetContentSize(5000, 5000)
	z.SetZoom(math.NaN())
	if math.IsNaN(z.Zoom()) {
		t.Fatal("SetZoom(NaN) made the zoom NaN")
	}
	z.ActualSize()
	if cx, cy := z.ToContent(10, 10); !finiteValue(cx) || !finiteValue(cy) {
		t.Fatalf("after SetZoom(NaN): content at (10,10) = %v, %v", cx, cy)
	}
	z.SetZoom(0)
	z.SetZoom(-2)
	if z.Zoom() != 1 {
		t.Errorf("SetZoom(0), SetZoom(-2) changed the zoom to %v", z.Zoom())
	}
	z.SetZoom(math.Inf(1))
	if z.Zoom() != 64 {
		t.Errorf("SetZoom(+Inf) = %v, want the largest zoom 64", z.Zoom())
	}

	z2 := NewZoomView()
	newTestForm(t, z2)
	z2.SetContentSize(5000, 5000)
	z2.SetZoomRange(0, 64)
	z2.SetZoomRange(math.NaN(), 64)
	z2.SetZoomRange(4, 2)
	for i := 0; i < 5000; i++ {
		z2.SetZoom(z2.Zoom() / 1.25)
	}
	if !(z2.Zoom() > 0) {
		t.Fatalf("zoom reached %v", z2.Zoom())
	}
	z2.ActualSize()
	if cx, cy := z2.ToContent(10, 10); !finiteValue(cx) || !finiteValue(cy) {
		t.Fatalf("after zooming out with SetZoomRange(0, 64): content at (10,10) = %v, %v", cx, cy)
	}
	z2.CenterOn(math.NaN(), math.Inf(1))
	if cx, cy := z2.ToContent(10, 10); !finiteValue(cx) || !finiteValue(cy) {
		t.Fatalf("after CenterOn(NaN, Inf): content at (10,10) = %v, %v", cx, cy)
	}
	z2.SetContentSize(math.NaN(), 100)
	if cx, cy := z2.ToContent(10, 10); !finiteValue(cx) || !finiteValue(cy) {
		t.Fatalf("after SetContentSize(NaN, 100): content at (10,10) = %v, %v", cx, cy)
	}
}

// The slider ignores NaN values, and NaN or infinite range ends and steps,
// also from an assistive technology (AT-SPI CurrentValue = NaN)
func TestMiscSliderNaN(t *testing.T) {
	s := NewSlider()
	f := newTestForm(t, s)
	s.SetRange(0, 10)
	s.SetValue(5)
	var emitted []float64
	s.OnValueChanged.Connect(func(v float64) { emitted = append(emitted, v) })

	s.SetValue(math.NaN())
	if s.Value() != 5 {
		t.Errorf("SetValue(NaN): value %v, want 5", s.Value())
	}
	s.ProcessKeyDown(KeyArrowRight, KeyModifiers{})
	if math.IsNaN(s.Value()) || s.Value() <= 5 {
		t.Errorf("after the right arrow: value %v", s.Value())
	}
	s.SetStep(math.Inf(1))
	s.SetStep(math.NaN())
	if s.Step() != 0 {
		t.Errorf("SetStep(Inf/NaN): step %v, want 0", s.Step())
	}
	s.SetValue(5)
	if s.Value() != 5 {
		t.Errorf("after SetStep(Inf): SetValue(5) gives %v", s.Value())
	}
	s.SetRange(math.NaN(), 20)
	s.SetRange(0, math.Inf(1))
	if s.Min() != 0 || s.Max() != 10 {
		t.Errorf("SetRange with NaN/Inf changed the range to %v..%v", s.Min(), s.Max())
	}

	n := f.Accessible().Find(RoleSlider, "")
	if n == nil {
		t.Fatal("no accessible slider")
	}
	if n.SetCurrent(math.NaN()) {
		t.Error("SetCurrent(NaN) returned true")
	}
	if s.Value() != 5 {
		t.Errorf("after accessible SetCurrent(NaN): value %v", s.Value())
	}
	if !n.SetCurrent(math.Inf(1)) || s.Value() != 10 {
		t.Errorf("SetCurrent(+Inf): value %v, want the maximum 10", s.Value())
	}
	for _, v := range emitted {
		if math.IsNaN(v) {
			t.Fatal("OnValueChanged got NaN")
		}
	}
}

// A mesh with a negative index skips that triangle, as one with too large
// an index: before, it panicked the renderer and ComputeNormals
func TestMiscMeshNegativeIndex(t *testing.T) {
	m := NewBoxMesh(1, 1, 1)
	m.Indices = append(m.Indices, -1, 0, 1)
	sc := NewScene3D()
	sc.Add(m, Material{})
	cam := NewCamera(V3(3, 3, 3), V3(0, 0, 0))
	img := NewRenderer3D().RenderImage(64, 64, sc, cam)
	if img == nil {
		t.Fatal("no image")
	}
	if _, ok := sc.Pick(cam.Ray(32, 32, 64, 64)); !ok {
		t.Error("Pick at the center of the box hit nothing")
	}
	if e := m.wireEdges(true); len(e) == 0 {
		t.Error("no edges")
	}

	tri := &Mesh{Positions: []Vec3{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}, Indices: []int{0, 1, -2}}
	tri.ComputeNormals()
	if len(tri.Normals) != 3 {
		t.Errorf("ComputeNormals made %d normals", len(tri.Normals))
	}

	// Appended, the bad triangle doesn't turn into one of the corners of
	// the mesh before it
	all := NewBoxMesh(1, 1, 1)
	count := all.TriangleCount()
	all.Append(tri)
	if all.TriangleCount() != count {
		t.Errorf("Append added %d triangles of a mesh with only a bad one", all.TriangleCount()-count)
	}
}
