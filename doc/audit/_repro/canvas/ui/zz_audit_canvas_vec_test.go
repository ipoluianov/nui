package ui

import (
	"image"
	"image/draw"
	"testing"

	"golang.org/x/image/vector"
)

func TestAuditVectorDirect(t *testing.T) {
	for _, sz := range [][2]int{{400, 300}, {64, 300}, {600, 600}} {
		z := vector.NewRasterizer(sz[0], sz[1])
		z.MoveTo(0, 0)
		z.LineTo(400, 37.3)
		z.LineTo(10, 290)
		z.ClosePath()
		m := image.NewAlpha(image.Rect(0, 0, sz[0], sz[1]))
		z.DrawOp = draw.Src
		z.Draw(m, m.Rect, image.Opaque, image.Point{})
		t.Logf("vector %v: pixel(9,288)=%d", sz, m.AlphaAt(9, 288).A)
	}
	// nui unbanded: a canvas narrow enough
	c, img := newTestCanvas(100, 300)
	p := NewPath()
	p.Polygon(PointF{0, 0}, PointF{400, 37.3}, PointF{10, 290})
	c.FillPath(p, auditWhite)
	t.Logf("nui 100x300: pixel(9,288)=%d", img.RGBAAt(9, 288).R)
	c, img = newTestCanvas(400, 300)
	c.FillPath(p, auditWhite)
	t.Logf("nui 400x300: pixel(9,288)=%d", img.RGBAAt(9, 288).R)
}
