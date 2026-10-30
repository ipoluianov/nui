// Package canvasdemo shows what the canvas of nui draws: transforms,
// paths read from SVG, fill rules, splines, gradients, patterns, shadows,
// layers and blur, clips to shapes, blend modes, text laid out, in tiles
// of a custom widget, some of them moving.
package canvasdemo

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// NewForm returns the Canvas demo
func NewForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Canvas")
	form.SetSize(1000, 700)
	form.Panel().AddWidget(0, 0, newDemo())
	return form
}

// demo is the widget that draws the tiles
type demo struct {
	ui.Widget
	start   time.Time
	t       float64 // seconds since the start, for the moving tiles
	pattern image.Image
	heart   *ui.Path
	photo   *ui.MipImage
	label   *ui.TextLayout
	// The mouse, and the origin of the tile being drawn, in the widget
	mouseX, mouseY float64
	tileX, tileY   float64
}

const tileW, tileH, gap = 230, 200, 12

func newDemo() *demo {
	d := &demo{start: time.Now(), mouseX: -1, mouseY: -1}
	d.InitWidget()
	d.SetTypeName("CanvasDemo")
	d.SetXExpandable(true)
	d.SetYExpandable(true)
	d.SetOnPaint(d.paint)
	d.SetOnMouseMove(func(x, y int, _ ui.KeyModifiers) bool {
		d.mouseX, d.mouseY = float64(x), float64(y)
		return false
	})
	d.pattern = dots()
	d.photo = ui.NewMipImage(photo())
	d.label = ui.NewTextLayout("Shaped ", ui.TextStyle{Size: 15})
	d.label.Append("and wrapped", ui.TextStyle{Size: 15, Bold: true, Color: blue})
	d.label.Append(" text: office, AV, ", ui.TextStyle{Size: 15})
	d.label.Append("שלום", ui.TextStyle{Size: 15, Color: orange})
	d.label.Append(" and مرحبا 2024.", ui.TextStyle{Size: 15, Italic: true})
	d.label.SetWidth(tileW - 30)
	// A heart of an SVG icon (24 x 24)
	d.heart, _ = ui.ParseSVGPath("M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z")
	d.AddTimer(33, func() {
		d.t = time.Since(d.start).Seconds()
		d.Update()
	})
	return d
}

var (
	ink    = color.RGBA{40, 44, 52, 255}
	blue   = color.RGBA{37, 99, 235, 255}
	orange = color.RGBA{234, 88, 12, 255}
	green  = color.RGBA{22, 163, 74, 255}
	purple = color.RGBA{147, 51, 234, 255}
	paper  = color.RGBA{250, 250, 252, 255}
)

func (d *demo) paint(cnv *ui.Canvas) {
	cnv.FillRect(0, 0, d.Width(), d.Height(), color.RGBA{232, 234, 238, 255})
	tiles := []struct {
		title string
		draw  func(cnv *ui.Canvas)
	}{
		{"Transforms", d.transforms},
		{"SVG path, even-odd", d.svgAndRules},
		{"Splines and arcs", d.splines},
		{"Gradients", d.gradients},
		{"Patterns", d.patterns},
		{"Shadow and layers", d.shadows},
		{"Text on a transform", d.text},
		{"Path under the mouse", d.hitTest},
		{"Clip to a shape", d.clips},
		{"Blend modes, eraser", d.blends},
		{"Text laid out", d.layout},
		{"Gradient line, faded", d.fades},
	}
	cols := max(1, (d.Width()-gap)/(tileW+gap))
	for i, tile := range tiles {
		x := gap + (i%cols)*(tileW+gap)
		y := gap + (i/cols)*(tileH+gap)
		cnv.DrawShadow(float64(x), float64(y+2), tileW, tileH, 8, 10, color.RGBA{0, 0, 0, 60})
		cnv.SetColor(paper)
		cnv.FillRoundedRect(x, y, tileW, tileH, 8)
		cnv.SetColor(ink)
		cnv.SetFontSize(12)
		cnv.SetFontBold(true)
		cnv.SetHAlign(ui.HAlignLeft)
		cnv.SetVAlign(ui.VAlignTop)
		cnv.DrawText(x+12, y+8, tileW-24, 20, tile.title)
		cnv.SetFontBold(false)
		cnv.Save()
		cnv.TranslateAndClip(x, y+30, tileW, tileH-30)
		d.tileX, d.tileY = float64(x), float64(y+30)
		tile.draw(cnv)
		cnv.Restore()
	}
}

// transforms: shapes turned, stretched and slanted by the canvas
func (d *demo) transforms(cnv *ui.Canvas) {
	p := ui.NewPath()
	p.RoundedRect(-22, -22, 44, 44, 8)
	for i, col := range []color.RGBA{blue, orange, green} {
		cnv.Save()
		cnv.Translate(45+float64(i)*70, 80)
		switch i {
		case 0:
			cnv.Rotate(d.t)
		case 1:
			s := 0.7 + 0.3*math.Sin(d.t*2)
			cnv.ScaleBy(s, 1/s)
		case 2:
			cnv.Concat(ui.SkewMatrix(math.Sin(d.t)*0.5, 0))
		}
		cnv.FillPath(p, col)
		cnv.StrokePath(p, ui.StrokeStyle{Width: 2}, ink)
		cnv.Restore()
	}
}

// svgAndRules: an SVG icon, and a star filled by both rules
func (d *demo) svgAndRules(cnv *ui.Canvas) {
	cnv.Save()
	cnv.Translate(10, 20)
	cnv.ScaleBy(3, 3)
	cnv.FillPath(d.heart, color.RGBA{220, 38, 38, 255})
	cnv.Restore()
	for i, rule := range []ui.FillRule{ui.FillRuleNonZero, ui.FillRuleEvenOdd} {
		p := ui.NewPath()
		cx, cy := 140+float64(i)*55, 85.0
		for k := 0; k < 5; k++ {
			a := -math.Pi/2 + float64(k)*4*math.Pi/5
			if k == 0 {
				p.MoveTo(cx+28*math.Cos(a), cy+28*math.Sin(a))
			} else {
				p.LineTo(cx+28*math.Cos(a), cy+28*math.Sin(a))
			}
		}
		p.Close()
		p.SetFillRule(rule)
		cnv.FillPath(p, purple)
	}
}

// splines: a smooth curve through points, an arc tangent to two lines
func (d *demo) splines(cnv *ui.Canvas) {
	pts := make([]ui.PointF, 7)
	for i := range pts {
		pts[i] = ui.PointF{X: 20 + float64(i)*32, Y: 85 + 40*math.Sin(float64(i)*1.1+d.t)}
	}
	p := ui.NewPath()
	p.Spline(pts, false, 0)
	cnv.StrokePath(p, ui.StrokeStyle{Width: 3, Cap: ui.LineCapRound}, blue)
	for _, pt := range pts {
		cnv.DrawMarker(ui.MarkerCircle, pt.X, pt.Y, 7, paper, blue)
	}
	q := ui.NewPath()
	q.MoveTo(20, 150)
	q.ArcTo(115, 120, 210, 150, 25)
	q.LineTo(210, 150)
	cnv.StrokePath(q, ui.StrokeStyle{Width: 2, Dash: []float64{6, 4}}, orange)
}

// gradients: a color wheel, stripes of a repeated gradient, a sphere
func (d *demo) gradients(cnv *ui.Canvas) {
	wheel := ui.NewPath()
	wheel.Sector(60, 80, 25, 50, 0, 2*math.Pi)
	stops := []ui.GradientStop{}
	for i := 0; i <= 6; i++ {
		h := float64(i) / 6
		stops = append(stops, ui.GradientStop{Offset: h, Color: hue(h)})
	}
	cnv.FillPathPaint(wheel, ui.ConicGradient{CX: 60, CY: 80, Angle: d.t, Stops: stops})
	bar := ui.NewPath()
	bar.RoundedRect(125, 30, 90, 40, 6)
	cnv.FillPathPaint(bar, ui.LinearGradient{X1: 125, Y1: 30, X2: 140, Y2: 45, Spread: ui.SpreadReflect,
		Stops: []ui.GradientStop{{Offset: 0, Color: blue}, {Offset: 1, Color: color.RGBA{147, 197, 253, 255}}}})
	ball := ui.NewPath()
	ball.Circle(170, 120, 32)
	cnv.FillPathPaint(ball, ui.RadialGradient{CX: 158, CY: 108, R: 45,
		Stops: []ui.GradientStop{{Offset: 0, Color: color.RGBA{255, 255, 255, 255}}, {Offset: 1, Color: green}}})
}

// patterns: an image repeated in a shape, a hatch
func (d *demo) patterns(cnv *ui.Canvas) {
	p := ui.NewPath()
	p.Star(65, 85, 55, 25, 6)
	cnv.FillPathPaint(p, ui.ImagePattern{Image: d.pattern, Repeat: true, X: d.t * 8})
	cnv.StrokePath(p, ui.StrokeStyle{Width: 1.5, Join: ui.LineJoinRound}, ink)
	q := ui.NewPath()
	q.RoundedRect(135, 35, 80, 100, 10)
	cnv.FillPathPaint(q, ui.HatchPaint{Color: color.RGBA{220, 38, 38, 160}, Angle: math.Pi / 4, Spacing: 7, Width: 2})
	cnv.StrokePath(q, ui.StrokeStyle{Width: 1.5}, color.RGBA{220, 38, 38, 255})
}

// shadows: a card with a shadow, a translucent layer, a blurred area
func (d *demo) shadows(cnv *ui.Canvas) {
	cnv.DrawShadow(20, 28, 80, 60, 8, 12, color.RGBA{0, 0, 0, 110})
	cnv.SetColor(color.RGBA{255, 255, 255, 255})
	cnv.FillRoundedRect(20, 24, 80, 60, 8)
	// Two circles in a half transparent layer: no darker overlap
	cnv.BeginLayer(0.5)
	cnv.FillCircle(150, 55, 30, blue)
	cnv.FillCircle(185, 55, 30, blue)
	cnv.EndLayer()
	for i := 0; i < 6; i++ {
		cnv.FillRect(20+i*33, 110, 16, 50, []color.RGBA{blue, orange, green, purple, ink, blue}[i])
	}
	cnv.Blur(110, 105, 110, 60, 4+3*math.Sin(d.t*2))
}

// text: a label turned and stretched by the transform
func (d *demo) text(cnv *ui.Canvas) {
	cnv.Save()
	cnv.Translate(115, 85)
	cnv.Rotate(math.Sin(d.t) * 0.6)
	cnv.ScaleBy(1.4, 1.4)
	cnv.SetColor(ink)
	cnv.SetFontSize(16)
	cnv.SetHAlign(ui.HAlignCenter)
	cnv.SetVAlign(ui.VAlignCenter)
	cnv.DrawText(-80, -15, 160, 30, "Hello, canvas")
	cnv.Restore()
}

// hitTest: the shape under the mouse is lit
func (d *demo) hitTest(cnv *ui.Canvas) {
	shapes := []*ui.Path{ui.NewPath(), ui.NewPath(), ui.NewPath()}
	shapes[0].RegularPolygon(55, 80, 40, 6, 0)
	shapes[1].Star(160, 70, 45, 20, 5)
	shapes[2].Ellipse(150, 135, 60, 18)
	mx, my := cnv.FromCanvas(d.mouseX-d.tileX, d.mouseY-d.tileY)
	for i, p := range shapes {
		col := []color.RGBA{blue, orange, green}[i]
		if p.Contains(mx, my) {
			col = color.RGBA{250, 204, 21, 255}
		}
		cnv.FillPath(p, col)
		cnv.StrokePath(p, ui.StrokeStyle{Width: 1.5, Join: ui.LineJoinRound}, ink)
	}
}

// clips: a picture in a turning star, a mipmapped thumbnail in a circle
func (d *demo) clips(cnv *ui.Canvas) {
	star := ui.NewPath()
	star.Star(70, 85, 60, 30, 5)
	cnv.Save()
	cnv.RotateAround(d.t/2, 70, 85)
	cnv.ClipPath(star)
	cnv.ResetTransform()
	cnv.DrawImageRect(10, 25, 120, 120, d.photo, d.photo.Bounds(), true)
	cnv.Restore()
	circle := ui.NewPath()
	circle.Circle(175, 85, 40)
	cnv.Save()
	cnv.ClipPath(circle)
	// The picture of 512 pixels drawn 80 wide: smooth, from its mipmaps
	cnv.DrawImageRect(135, 45, 80, 80, d.photo, d.photo.Bounds(), true)
	cnv.Restore()
	cnv.StrokePath(circle, ui.StrokeStyle{Width: 2}, ink)
}

// blends: light added on black, inks multiplied on paper, rubbed out by
// an eraser in their layer
func (d *demo) blends(cnv *ui.Canvas) {
	cnv.FillRect(10, 10, 100, 130, color.RGBA{20, 20, 24, 255})
	cnv.Save()
	cnv.TranslateAndClip(10, 10, 100, 130)
	for i, col := range []color.RGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}} {
		cnv.BeginLayerStyle(ui.LayerStyle{Blend: ui.BlendScreen})
		a := d.t + float64(i)*2*math.Pi/3
		cnv.FillCircle(50+14*math.Cos(a), 65+14*math.Sin(a), 28, col)
		cnv.EndLayer()
	}
	cnv.Restore()
	cnv.Save()
	cnv.TranslateAndClip(120, 10, 100, 130)
	// The inks in a layer of their own: the eraser rubs them out, not the
	// paper
	cnv.BeginLayerStyle(ui.LayerStyle{Isolated: true})
	for i, col := range []color.RGBA{{0, 200, 255, 255}, {255, 0, 200, 255}, {255, 230, 0, 255}} {
		cnv.BeginLayerStyle(ui.LayerStyle{Blend: ui.BlendMultiply})
		a := -d.t + float64(i)*2*math.Pi/3
		cnv.FillCircle(50+14*math.Cos(a), 65+14*math.Sin(a), 28, col)
		cnv.EndLayer()
	}
	cnv.BeginLayerStyle(ui.LayerStyle{Blend: ui.BlendErase})
	line := ui.NewPath()
	line.MoveTo(15, 65+30*math.Sin(d.t*1.3))
	line.LineTo(85, 65-30*math.Sin(d.t*1.3))
	cnv.StrokePath(line, ui.StrokeStyle{Width: 8, Cap: ui.LineCapRound}, ink)
	cnv.EndLayer()
	cnv.EndLayer()
	cnv.Restore()
}

// layout: styled, shaped text wrapped in a width; the caret follows the
// mouse
func (d *demo) layout(cnv *ui.Canvas) {
	x, y := 15.0, 12.0
	cnv.DrawTextLayout(d.label, x, y)
	mx, my := d.mouseX-d.tileX-x, d.mouseY-d.tileY-y
	w, h := d.label.Size()
	if mx >= 0 && my >= 0 && mx <= w && my <= h {
		at := d.label.IndexAt(mx, my)
		for _, r := range d.label.SelectionRects(0, at) {
			p := ui.NewPath()
			p.Rect(x+r.MinX, y+r.MinY, r.Width(), r.Height())
			cnv.FillPath(p, color.RGBA{37, 99, 235, 40})
		}
		c := d.label.CaretRect(at)
		caret := ui.NewPath()
		caret.Rect(x+c.MinX-0.5, y+c.MinY, 1, c.Height())
		cnv.FillPath(caret, ink)
	}
	cnv.SetColor(ink)
	cnv.SetFontSize(11)
	cnv.DrawTextF(x, y+h+8, "at x 15, the caret under the mouse")
}

// fades: a spiral colored along its length, stripes faded out at the right
func (d *demo) fades(cnv *ui.Canvas) {
	spiral := ui.NewPath()
	for i := 0; i <= 200; i++ {
		a := float64(i) / 200 * 4 * math.Pi
		r := 5 + float64(i)/200*55
		px, py := 70+r*math.Cos(a+d.t), 85+r*math.Sin(a+d.t)
		if i == 0 {
			spiral.MoveTo(px, py)
		} else {
			spiral.LineTo(px, py)
		}
	}
	stops := []ui.GradientStop{}
	for i := 0; i <= 6; i++ {
		stops = append(stops, ui.GradientStop{Offset: float64(i) / 6, Color: hue(float64(i) / 6)})
	}
	cnv.StrokePathPaint(spiral, ui.StrokeStyle{Width: 4, Cap: ui.LineCapRound, Join: ui.LineJoinRound},
		ui.ConicGradient{CX: 70, CY: 85, Angle: d.t, Stops: stops})
	cnv.Save()
	cnv.TranslateAndClip(140, 20, 80, 130)
	cnv.BeginLayerStyle(ui.LayerStyle{Mask: ui.LinearGradient{X1: 20, X2: 80,
		Stops: []ui.GradientStop{{Offset: 0, Color: color.RGBA{0, 0, 0, 255}}, {Offset: 1}}}})
	for i := 0; i < 8; i++ {
		cnv.FillRect(0, i*16, 80, 10, []color.RGBA{blue, orange, green, purple}[i%4])
	}
	cnv.EndLayer()
	cnv.Restore()
}

// photo is a picture of 512 pixels with fine detail: rings that shimmer
// when it is drawn small without its mipmaps
func photo() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	for y := 0; y < 512; y++ {
		for x := 0; x < 512; x++ {
			dx, dy := float64(x-256), float64(y-256)
			r := math.Hypot(dx, dy)
			v := 0.5 + 0.5*math.Sin(r*0.6)
			c := hue(math.Mod(math.Atan2(dy, dx)/(2*math.Pi)+1, 1))
			img.SetRGBA(x, y, color.RGBA{uint8(float64(c.R) * v), uint8(float64(c.G) * v), uint8(float64(c.B) * v), 255})
		}
	}
	return img
}

// hue returns the color of the hue 0..1 at full saturation
func hue(h float64) color.RGBA {
	f := func(n float64) uint8 {
		k := math.Mod(n+h*6, 6)
		return uint8(255 * (1 - math.Max(0, math.Min(math.Min(k, 4-k), 1))))
	}
	return color.RGBA{f(5), f(3), f(1), 255}
}

// dots is the image of the pattern: a dot in a square
func dots() image.Image {
	cnv, img := ui.NewImageCanvas(12, 12, 1)
	cnv.Clear(color.RGBA{255, 247, 237, 255})
	cnv.FillCircle(6, 6, 3.2, orange)
	return img
}
