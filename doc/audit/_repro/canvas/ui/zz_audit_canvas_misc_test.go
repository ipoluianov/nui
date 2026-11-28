package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestAuditMiscNoPanic(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	img0 := image.NewRGBA(image.Rect(0, 0, 0, 0))
	img1 := image.NewRGBA(image.Rect(3, 3, 4, 9))
	cases := map[string]func(c *Canvas){
		"layer empty clip": func(c *Canvas) {
			c.SetDirectTranslateAndClip(0, 0, 0, 0)
			c.BeginLayerStyle(LayerStyle{Blend: BlendMultiply})
			c.FillRect(0, 0, 10, 10, auditWhite)
			c.Blur(0, 0, 10, 10, 2)
			c.EndLayer()
		},
		"layer clip inside": func(c *Canvas) {
			c.BeginLayerStyle(LayerStyle{Blend: BlendScreen, Mask: LinearGradient{X2: 40, Stops: []GradientStop{{0, auditWhite}, {1, color.RGBA{}}}}})
			c.Save()
			p := NewPath()
			p.Circle(20, 20, 10)
			c.ClipPath(p)
			c.FillRect(0, 0, 40, 40, auditWhite)
			c.Restore()
			c.EndLayer()
		},
		"layer clip not restored": func(c *Canvas) {
			c.BeginLayerStyle(LayerStyle{Blend: BlendErase})
			p := NewPath()
			p.Circle(20, 20, 10)
			c.Save()
			c.ClipPath(p)
			c.FillRect(0, 0, 40, 40, auditWhite)
			c.EndLayer()
			c.Restore()
		},
		"layer opacity NaN": func(c *Canvas) {
			c.BeginLayer(nan)
			c.FillRect(0, 0, 40, 40, auditWhite)
			c.EndLayer()
		},
		"EndLayer twice":   func(c *Canvas) { c.BeginLayer(0.5); c.EndLayer(); c.EndLayer() },
		"Restore no Save":  func(c *Canvas) { c.Restore(); c.Restore() },
		"img xf NaN":       func(c *Canvas) { c.DrawImageTransformed(img1, Matrix{A: nan, D: 1}, true) },
		"img xf singular":  func(c *Canvas) { c.DrawImageTransformed(img1, Matrix{A: 1, B: 1, C: 1, D: 1}, true) },
		"img xf tiny":      func(c *Canvas) { c.DrawImageTransformed(img1, ScalingMatrix(1e-300, 1e-300), true) },
		"img xf huge":      func(c *Canvas) { c.DrawImageTransformed(img1, ScalingMatrix(1e300, 1e300), true) },
		"img xf inf E":     func(c *Canvas) { c.DrawImageTransformed(img1, Matrix{A: 1, D: 1, E: inf}, false) },
		"img empty":        func(c *Canvas) { c.DrawImage(0, 0, img0); c.DrawImageRect(0, 0, 10, 10, img0, img0.Rect, true) },
		"mip empty":        func(c *Canvas) { m := NewMipImage(img0); c.DrawImageRect(0, 0, 1, 1, m, m.Bounds(), true) },
		"mip offset small": func(c *Canvas) { m := NewMipImage(img1); c.DrawImageRect(0, 0, 1, 1, m, m.Bounds(), true) },
		"mip xf small":     func(c *Canvas) { m := NewMipImage(img1); c.DrawImageTransformed(m, ScalingMatrix(0.01, 0.01), true) },
		"img rect neg src": func(c *Canvas) { c.DrawImageRect(0, 0, 10, 10, img1, image.Rect(-100, -100, 100, 100), false) },
		"marker NaN":       func(c *Canvas) { c.DrawMarker(MarkerStar, nan, 10, 5, auditWhite, auditWhite) },
		"marker inf":       func(c *Canvas) { c.DrawMarker(MarkerCircle, 10, 10, inf, auditWhite, auditWhite) },
		"marker bad shape": func(c *Canvas) { c.DrawMarker(MarkerShape(99), 10, 10, 5, auditWhite, auditWhite) },
		"adjust neg":       func(c *Canvas) { c.AdjustPixels(30, 30, -50, -50, func(c color.RGBA) color.RGBA { return c }) },
		"blur neg":         func(c *Canvas) { c.Blur(30, 30, -50, -50, 3) },
		"flood":            func(c *Canvas) { c.FloodFill(5, 5, auditWhite, 300); c.FloodFill(-5, 5, auditWhite, -1) },
		"lineF NaN":        func(c *Canvas) { c.DrawLineF(nan, 0, 10, 10, auditWhite); c.DrawLineF(-inf, 5, inf, 5, auditWhite) },
		"lineF huge":       func(c *Canvas) { c.DrawLineF(-1e300, 5, 1e300, 6, auditWhite) },
		"line wide":        func(c *Canvas) { c.DrawLine(0, 0, 30, 17, 5, auditWhite); c.DrawLine(-1<<40, 0, 1<<40, 17, 3, auditWhite) },
		"rounded rect":     func(c *Canvas) { c.FillRoundedRect(0, 0, 30, 30, -5); c.DrawRoundedRect(0, 0, 30, 30, 1<<40) },
		"arc NaN":          func(c *Canvas) { p := NewPath(); p.Arc(nan, 0, 5, 0, 1); p.ArcTo(1, 1, 2, nan, 3); c.FillPath(p, auditWhite) },
		"svgarc":           func(c *Canvas) { p := NewPath(); p.MoveTo(0, 0); p.SVGArcTo(nan, inf, nan, true, true, 5, 5); c.StrokePath(p, StrokeStyle{}, auditWhite) },
		"stroke width NaN": func(c *Canvas) { p := NewPath(); p.Rect(1, 1, 10, 10); c.StrokePath(p, StrokeStyle{Width: nan, MiterLimit: nan}, auditWhite) },
		"stroke width inf": func(c *Canvas) { p := NewPath(); p.Rect(1, 1, 10, 10); c.StrokePath(p, StrokeStyle{Width: inf}, auditWhite) },
		"dash NaN":         func(c *Canvas) { p := NewPath(); p.Rect(1, 1, 10, 10); c.StrokePath(p, StrokeStyle{Dash: []float64{nan, 1}, DashOffset: inf}, auditWhite) },
		"dash tiny":        func(c *Canvas) { p := NewPath(); p.Rect(1, 1, 30, 30); c.StrokePath(p, StrokeStyle{Dash: []float64{1e-300, 1e-300}}, auditWhite) },
		"xf NaN fill":      func(c *Canvas) { c.SetTransform(Matrix{A: nan, D: 1}); c.FillRect(0, 0, 10, 10, auditWhite); c.DrawText(0, 0, 10, 10, "x") },
		"xf zero":          func(c *Canvas) { c.ScaleBy(0, 0); c.FillRect(0, 0, 10, 10, auditWhite); c.DrawText(0, 0, 10, 10, "x"); c.FromCanvas(1, 1) },
		"xf stroke tiny":   func(c *Canvas) { c.ScaleBy(1e-200, 1e-200); p := NewPath(); p.Rect(0, 0, 1e200, 1e200); c.StrokePath(p, StrokeStyle{Dash: []float64{1, 1}}, auditWhite) },
		"shadow ok":        func(c *Canvas) { c.DrawShadow(5, 5, 20, 20, 1e9, 4, auditWhite) },
		"clipbounds":       func(c *Canvas) { c.ScaleBy(0, 1); _ = c.ClipBounds() },
		"paint cache NaN":  func(c *Canvas) { pc := NewPaintCache(); pc.Draw(c, 0, 0, 10, 10, func(*Canvas) {}); pc.InvalidateRect(RectF{nan, nan, nan, nan}); pc.Draw(c, 0, 0, 10, 10, func(*Canvas) {}) },
	}
	for name, f := range cases {
		f := f
		auditRun(t, name, func() {
			cnv, _ := newTestCanvas(40, 40)
			f(cnv)
		})
	}
}
