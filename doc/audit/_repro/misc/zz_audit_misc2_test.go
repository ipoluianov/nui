package ui

import (
	"image/color"
	"math"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

func firstLines(s string, n int) string {
	l := strings.Split(s, "\n")
	if len(l) > n {
		l = l[:n]
	}
	return strings.Join(l, "\n")
}

func TestAuditView2DGridFar(t *testing.T) {
	v := NewView2D()
	f := newTestForm(t, v)
	v.SetView(PointF{1e10, 0}, 1e9)
	done := make(chan struct{})
	go func() {
		defer close(done)
		auditPaint(t, f)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("View2D paint hangs: grid loop at center 1e10, scale 1e9")
	}
}

func TestAuditView2DGridFar2(t *testing.T) {
	v := NewView2D()
	f := newTestForm(t, v)
	v.SetGrid(View2DGrid{Visible: true, Step: 1})
	v.SetView(PointF{0, 3e16}, 10)
	done := make(chan struct{})
	go func() {
		defer close(done)
		auditPaint(t, f)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("View2D paint hangs: grid loop at center y=3e16, step 1")
	}
}

func TestAuditZoomViewNaN(t *testing.T) {
	z := NewZoomView()
	f := newTestForm(t, z)
	z.SetContentSize(5000, 5000)
	z.SetZoom(math.NaN())
	z.ActualSize()
	cx, cy := z.ToContent(10, 10)
	t.Logf("after SetZoom(NaN)+ActualSize: zoom=%v content at (10,10)=%v,%v", z.Zoom(), cx, cy)
	auditPaint(t, f)
	z2 := NewZoomView()
	newTestForm(t, z2)
	z2.SetContentSize(5000, 5000)
	z2.SetZoomRange(0, 64)
	for i := 0; i < 5000; i++ {
		z2.SetZoom(z2.Zoom() / 1.25)
	}
	z2.ActualSize()
	cx, cy = z2.ToContent(10, 10)
	t.Logf("minZoom 0: zoom=%v content at (10,10)=%v,%v", z2.Zoom(), cx, cy)
}

func TestAuditView3DKeys(t *testing.T) {
	sc := NewScene3D()
	sc.Add(NewBoxMesh(1, 1, 1), Material{})
	v := NewView3D(sc)
	f := newTestForm(t, v)
	for i := 0; i < 5000; i++ {
		v.keyDown(KeyMinus, KeyModifiers{})
	}
	_, d, _, _ := v.View()
	t.Logf("distance after 5000 minus: %v", d)
	v.keyDown(KeyArrowLeft, KeyModifiers{Shift: true})
	tg, d, _, _ := v.View()
	t.Logf("after pan: target %v distance %v", tg, d)
	for i := 0; i < 20; i++ {
		v.keyDown(KeyEqual, KeyModifiers{})
	}
	tg, d, _, _ = v.View()
	t.Logf("after 20 plus: target %v distance %v", tg, d)
	auditPaint(t, f)
	v2 := NewView3D(sc)
	newTestForm(t, v2)
	for i := 0; i < 6000; i++ {
		v2.keyDown(KeyEqual, KeyModifiers{})
	}
	_, d, _, _ = v2.View()
	t.Logf("distance after 6000 plus: %v cam %+v", d, v2.Camera())
}

func TestAuditFaceCacheZoomedText(t *testing.T) {
	fontsMu.RLock()
	n0 := len(faceCache)
	fontsMu.RUnlock()
	cnv, _ := NewImageCanvas(200, 100, 1)
	for i := 0; i < 2000; i++ {
		cnv.Save()
		k := 1 + float64(i)*0.0005 // a smooth zoom animation
		cnv.SetTransform(ScalingMatrix(k, k))
		cnv.SetFontSize(12)
		cnv.DrawText(0, 0, 100, 20, "Label")
		cnv.Restore()
	}
	fontsMu.RLock()
	n1 := len(faceCache)
	fontsMu.RUnlock()
	t.Logf("zoomed DrawText: faceCache %d -> %d entries", n0, n1)
}

func TestAuditFaceCacheGrowth(t *testing.T) {
	var ms0, ms1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms0)
	fontsMu.RLock()
	n0 := len(faceCache)
	fontsMu.RUnlock()
	for i := 0; i < 20000; i++ {
		MeasureText(FontFamilySans, 10+float64(i)*0.001, "Ag")
	}
	for i := 0; i < 1000; i++ {
		MeasureText(FontFamilySans, math.NaN(), "Ag")
	}
	runtime.GC()
	runtime.ReadMemStats(&ms1)
	fontsMu.RLock()
	n1 := len(faceCache)
	fontsMu.RUnlock()
	t.Logf("faceCache %d -> %d entries, heap %d -> %d MB", n0, n1, ms0.HeapAlloc>>20, ms1.HeapAlloc>>20)
}

func TestAuditLabelFar(t *testing.T) {
	for _, p := range []Vec3{{1e12, 0, 0}, {1e17, 0, 0}, {1e300, 0, 0}, {1, 0, 4.9999999999}} {
		sc := NewScene3D()
		sc.AddLabel(p, "x", color.RGBA{0, 0, 0, 255})
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("label at %v: panic: %v", p, r)
				}
			}()
			NewRenderer3D().RenderImage(64, 64, sc, NewCamera(V3(0, 0, 5), V3(0, 0, 0)))
		}()
	}
}

func TestAuditRenderNaNParts(t *testing.T) {
	v := math.NaN()
	blk := color.RGBA{0, 0, 0, 255}
	cases := map[string]func(sc *Scene3D){
		"mesh": func(sc *Scene3D) {
			sc.Add(&Mesh{Positions: []Vec3{{0, 0, 0}, {1, 0, 0}, {v, 1, 0}}, Indices: []int{0, 1, 2}}, Material{})
		},
		"meshEdges": func(sc *Scene3D) {
			sc.Add(&Mesh{Positions: []Vec3{{0, 0, 0}, {1, 0, 0}, {v, 1, 0}}, Indices: []int{0, 1, 2}}, Material{EdgeColor: blk})
		},
		"line":    func(sc *Scene3D) { sc.AddLine(V3(0, 0, 0), V3(v, 0, 0), blk, 1) },
		"point":   func(sc *Scene3D) { sc.AddPoint(V3(v, 0, 0), blk, 3) },
		"label":   func(sc *Scene3D) { sc.AddLabel(V3(v, 0, 0), "x", blk) },
		"lineInf": func(sc *Scene3D) { sc.AddLine(V3(0, 0, 0), V3(math.Inf(1), 0, 0), blk, 1) },
		"lineBig": func(sc *Scene3D) { sc.AddLine(V3(0, 0, 0), V3(1e30, 0, 0), blk, 1) },
		"pointBigSize": func(sc *Scene3D) {
			sc.AddPoint(V3(0, 0, 0), blk, 1e30)
		},
		"pointNaNSize": func(sc *Scene3D) {
			sc.AddPoint(V3(0, 0, 0), blk, math.NaN())
		},
		"lineNaNWidth": func(sc *Scene3D) { sc.AddLine(V3(0, 0, 0), V3(1, 0, 0), blk, math.NaN()) },
		"lineHugeWidth": func(sc *Scene3D) { sc.AddLine(V3(0, 0, 0), V3(1, 0, 0), blk, 1e30) },
	}
	for name, add := range cases {
		for ci, cam := range []Camera{NewCamera(V3(3, 3, 3), V3(0, 0, 0)), NewCamera(V3(0, 0, 5), V3(0, 0, 0))} {
			sc := NewScene3D()
			sc.Add(NewBoxMesh(0.2, 0.2, 0.2), Material{})
			add(sc)
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s cam%d: panic: %v\n%s", name, ci, r, firstLines(string(debug.Stack()), 16))
					}
				}()
				NewRenderer3D().RenderImage(64, 64, sc, cam)
			}()
		}
	}
}
