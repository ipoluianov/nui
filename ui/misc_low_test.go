package ui

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A step that doesn't divide the range: the values are the multiples of
// the step from the minimum and the maximum itself, the nearest is taken.
// Before, the maximum was reachable only by a value at or past it (9.9 by
// 4 in 0..10 gave 8, not the nearer 10).
func TestMiscSliderStepOffGridMax(t *testing.T) {
	s := NewSlider()
	newTestForm(t, s)
	s.SetRange(0, 10)
	s.SetStep(4)
	for _, c := range []struct{ in, want float64 }{
		{10, 10}, {9.9, 10}, {9.5, 10}, {9, 10}, {8.9, 8}, {8, 8}, {6.1, 8}, {5.9, 4}, {0.1, 0}, {-3, 0}, {42, 10},
	} {
		s.SetValue(c.in)
		if s.Value() != c.want {
			t.Errorf("SetValue(%v): value %v, want %v", c.in, s.Value(), c.want)
		}
	}
	s.SetValue(8)
	s.ProcessKeyDown(KeyArrowRight, KeyModifiers{})
	if s.Value() != 10 {
		t.Errorf("right arrow from 8: %v, want the maximum 10", s.Value())
	}
	s.ProcessKeyDown(KeyArrowLeft, KeyModifiers{})
	if s.Value() != 8 {
		t.Errorf("left arrow from 10: %v, want 8", s.Value())
	}
	s.ProcessKeyDown(KeyHome, KeyModifiers{})
	s.ProcessKeyDown(KeyEnd, KeyModifiers{})
	if s.Value() != 10 {
		t.Errorf("End: %v, want 10", s.Value())
	}
}

// The zoom keys and the wheel keep the distance finite and above 0: before,
// 5000 presses of - made it +Inf and a pan then made the target NaN
func TestMiscView3DZoomLimits(t *testing.T) {
	v := NewView3D(NewScene3D())
	newTestForm(t, v)
	finite := func(what string) {
		t.Helper()
		target, d, _, _ := v.View()
		if math.IsNaN(d) || math.IsInf(d, 0) || d <= 0 ||
			math.IsNaN(target.X+target.Y+target.Z) || math.IsInf(target.X+target.Y+target.Z, 0) {
			t.Fatalf("%s: target %v distance %v", what, target, d)
		}
	}
	for i := 0; i < 5000; i++ {
		v.keyDown(KeyMinus, KeyModifiers{})
	}
	finite("5000 x minus")
	v.keyDown(KeyArrowLeft, KeyModifiers{Shift: true})
	finite("a pan after zooming out")
	for i := 0; i < 6000; i++ {
		v.keyDown(KeyEqual, KeyModifiers{})
	}
	if _, d, _, _ := v.View(); d < minView3DZoomDistance {
		t.Errorf("6000 x plus: distance %v", d)
	}
	v.wheel(&WheelEvent{DY: -5000, X: 10, Y: 10})
	finite("a wheel of -5000")
	v.wheel(&WheelEvent{DY: 5000, X: 10, Y: 10})
	finite("a wheel of 5000")

	v.SetView(Vec3{}, 7, 0, 0)
	v.SetView(Vec3{}, math.NaN(), 0, 0)
	if _, d, _, _ := v.View(); d != 7 {
		t.Errorf("SetView with a NaN distance: distance %v, want 7", d)
	}
	v.SetView(Vec3{}, math.Inf(1), 0, 0)
	finite("SetView with +Inf")
}

// A negative role has a name as any unknown one: before, String panicked
func TestMiscAccessibleRoleNegative(t *testing.T) {
	if got := AccessibleRole(-1).String(); got != "role -1" {
		t.Errorf("AccessibleRole(-1) = %q", got)
	}
}

// A text STL: a repeated endfacet doesn't add the triangle again, a vertex
// outside a facet is an error, and a facet without a normal gets its own
func TestMiscSTLTextFacets(t *testing.T) {
	m, err := LoadSTL(strings.NewReader("solid\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 1 0 0\nvertex 0 1 0\nendloop\nendfacet\nendfacet\nendsolid\n"))
	if err != nil || len(m.Positions) != 3 {
		t.Errorf("repeated endfacet: %d positions, %v; want 3", len(m.Positions), err)
	}
	if _, err := LoadSTL(strings.NewReader("solid\nvertex 1 2 3\nvertex 1 2 3\nvertex 1 2 3\nendfacet\n")); err == nil {
		t.Error("vertices outside a facet are accepted")
	}
	// The second facet (clockwise seen from +z, so its normal is -z) has no
	// normal; before, it kept the first one's (0.6 0 -0.8, the same side),
	// now it gets the one made from its corners
	m, err = LoadSTL(strings.NewReader("solid\n" +
		"facet normal 0.6 0 -0.8\nvertex 0 0 0\nvertex 0 1 0\nvertex 1 0 0\nendfacet\n" +
		"facet\nvertex 0 0 0\nvertex 0 2 0\nvertex 2 0 0\nendfacet\nendsolid\n"))
	if err != nil || len(m.Normals) != 6 {
		t.Fatalf("%d normals, %v", len(m.Normals), err)
	}
	if n := m.Normals[3]; math.Abs(n.Z+1) > 1e-9 {
		t.Errorf("the facet without a normal: normal %v, want 0 0 -1", n)
	}
}

// openFontFile's discard closes the file of a font that isn't used: before,
// a font file without the PostScript name asked for (bold and italic of the
// macOS system font, script fonts without the character) stayed open
func TestMiscOpenFontFileDiscardCloses(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("counts the descriptors in /proc")
	}
	path, err := filepath.Abs("fonts/NotoSans-Regular.ttf")
	if err != nil {
		t.Fatal(err)
	}
	count := func() int {
		fds, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			t.Skip(err)
		}
		return len(fds)
	}
	before := count()
	for i := 0; i < 20; i++ {
		f, exact, discard, err := openFontFile(path, "no such name")
		if err != nil || f == nil || exact {
			t.Fatalf("openFontFile: %v %v %v", f, exact, err)
		}
		discard()
	}
	if after := count(); after > before+2 {
		t.Errorf("%d descriptors open before, %d after 20 discarded fonts", before, after)
	}
	if _, _, _, err := openFontFile(filepath.Join(t.TempDir(), "none.ttf"), ""); err == nil {
		t.Error("a missing file: no error")
	}
}
