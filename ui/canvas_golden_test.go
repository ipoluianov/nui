package ui_test

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// drawShapesGallery draws every shape of the canvas: the golden image shows
// how they look, at each scale
func drawShapesGallery(cnv *ui.Canvas) {
	bg := color.RGBA{250, 250, 250, 255}
	ink := color.RGBA{40, 40, 40, 255}
	blue := color.RGBA{37, 99, 235, 255}
	orange := color.RGBA{234, 88, 12, 255}
	green := color.RGBA{22, 163, 74, 255}
	cnv.FillRect(0, 0, cnv.Width(), cnv.Height(), bg)

	// Joins and caps of a wide line
	p := ui.NewPath()
	for i, join := range []ui.LineJoin{ui.LineJoinMiter, ui.LineJoinRound, ui.LineJoinBevel} {
		x := 20 + float64(i)*60
		p.Reset()
		p.MoveTo(x, 60)
		p.LineTo(x+20, 20)
		p.LineTo(x+40, 60)
		cnv.StrokePath(p, ui.StrokeStyle{Width: 8, Join: join, Cap: ui.LineCap(i)}, blue)
		p.Reset()
		p.MoveTo(x, 60)
		p.LineTo(x+20, 20)
		p.LineTo(x+40, 60)
		cnv.StrokePath(p, ui.StrokeStyle{Width: 1}, orange)
	}

	// Dashes, a translucent line crossing itself
	p.Reset()
	p.MoveTo(200, 20.5)
	p.LineTo(380, 20.5)
	cnv.StrokePath(p, ui.StrokeStyle{Dash: []float64{6, 3}}, ink)
	p.Reset()
	p.MoveTo(200, 32.5)
	p.LineTo(380, 32.5)
	cnv.StrokePath(p, ui.StrokeStyle{Width: 2, Dash: []float64{8, 3, 2, 3}, Cap: ui.LineCapRound}, ink)
	p.Reset()
	p.MoveTo(200, 70)
	p.CubicTo(260, -10, 300, 120, 380, 45)
	p.LineTo(210, 45)
	cnv.StrokePath(p, ui.StrokeStyle{Width: 6, Join: ui.LineJoinRound}, color.RGBA{37, 99, 235, 110})

	// A pie, a donut, a gauge
	angles := []float64{0, 1.2, 2.9, 4.4, 2 * math.Pi}
	colors := []color.RGBA{blue, orange, green, {147, 51, 234, 255}}
	for i := 0; i < 4; i++ {
		p.Reset()
		p.Sector(60, 140, 0, 40, angles[i]-math.Pi/2, angles[i+1]-math.Pi/2)
		cnv.FillPath(p, colors[i])
		p.Reset()
		p.Sector(160, 140, 24, 40, angles[i]-math.Pi/2, angles[i+1]-math.Pi/2)
		cnv.FillPath(p, colors[i])
	}
	p.Reset()
	p.Sector(260, 160, 30, 40, math.Pi, 2*math.Pi)
	cnv.FillPath(p, color.RGBA{220, 220, 220, 255})
	p.Reset()
	p.Sector(260, 160, 30, 40, math.Pi, math.Pi*1.7)
	cnv.FillPath(p, green)
	cnv.DrawCircle(340, 140, 30, 2, ink)

	// A gradient under a line, a hatch
	area := ui.NewPath()
	line := make([]ui.PointF, 0, 50)
	for i := 0; i <= 48; i++ {
		x := 20 + float64(i)*3.5
		line = append(line, ui.PointF{X: x, Y: 250 - 25*math.Sin(float64(i)/6) - float64(i)/2})
	}
	area.Polyline(line...)
	area.LineTo(188, 290)
	area.LineTo(20, 290)
	area.Close()
	cnv.FillPathPaint(area, ui.LinearGradient{X1: 0, Y1: 210, X2: 0, Y2: 290, Stops: []ui.GradientStop{
		{Offset: 0, Color: color.RGBA{37, 99, 235, 160}}, {Offset: 1, Color: color.RGBA{37, 99, 235, 0}},
	}})
	cnv.DrawPolyline(line, ui.StrokeStyle{Width: 2, Join: ui.LineJoinRound}, blue)
	p.Reset()
	p.RoundedRect(210, 210, 80, 80, 10)
	cnv.FillPathPaint(p, ui.HatchPaint{Color: color.RGBA{220, 38, 38, 140}, Angle: math.Pi / 4, Spacing: 6, Width: 1.5})
	p.Reset()
	p.Circle(340, 250, 35)
	cnv.FillPathPaint(p, ui.RadialGradient{CX: 330, CY: 240, R: 40, Stops: []ui.GradientStop{
		{Offset: 0, Color: color.RGBA{255, 255, 255, 255}}, {Offset: 1, Color: orange},
	}})

	// Markers
	for i := ui.MarkerCircle; i <= ui.MarkerStar; i++ {
		x := 20 + float64(i)*22
		cnv.DrawMarker(i, x, 320, 10, blue, nil)
		cnv.DrawMarker(i, x, 342, 10, color.RGBA{255, 255, 255, 255}, ink)
	}

	// Turned text
	cnv.SetColor(ink)
	cnv.SetFontSize(12)
	cnv.SetHAlign(ui.HAlignCenter)
	cnv.SetVAlign(ui.VAlignCenter)
	cnv.DrawTextRotated(215, 330, -math.Pi/2, "Pressure")
	cnv.SetHAlign(ui.HAlignRight)
	cnv.SetVAlign(ui.VAlignTop)
	for i, label := range []string{"January", "February", "March"} {
		cnv.DrawTextRotated(260+float64(i)*40, 310, -math.Pi/4, label)
	}
	cnv.SetHAlign(ui.HAlignLeft)
	cnv.DrawTextRotated(370, 305, math.Pi/2, "Down")
}

// The shapes of the canvas look as in the golden images, at every scale
func TestCanvasShapesGolden(t *testing.T) {
	uitest.GoldenTolerance = 16
	defer func() { uitest.GoldenTolerance = 8 }()
	for _, scale := range []float64{1, 1.5, 2} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			w, h := 400, 370
			img := image.NewRGBA(image.Rect(0, 0, int(float64(w)*scale), int(float64(h)*scale)))
			cnv := ui.NewCanvasScaled(img, scale)
			cnv.SetDirectTranslateAndClip(0, 0, w, h)
			drawShapesGallery(cnv)
			uitest.MatchGoldenImage(t, fmt.Sprintf("canvas-shapes-%v", scale), img)
		})
	}
}
