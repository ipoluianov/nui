package ui

import (
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strings"
	"testing"
)

// drawLayers draws layers of every kind SVG has, and an eraser it has not
func drawLayers(cnv *Canvas) {
	cnv.FillRect(0, 0, 300, 150, white)
	cnv.FillRect(0, 0, 120, 150, black)
	for i, col := range []color.RGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}} {
		cnv.BeginLayerStyle(LayerStyle{Blend: BlendScreen})
		cnv.FillPath(pathOf(func(p *Path) { p.Circle(45+float64(i%2)*30, 55+float64(i/2)*30, 28) }), col)
		cnv.EndLayer()
	}
	cnv.BeginLayerStyle(LayerStyle{Opacity: 0.5})
	cnv.FillPath(pathOf(func(p *Path) { p.Rect(140, 10, 60, 60) }), red)
	cnv.EndLayer()
	cnv.Save()
	cnv.Translate(140, 80)
	cnv.BeginLayerStyle(LayerStyle{Mask: LinearGradient{X1: 0, X2: 140, Stops: []GradientStop{{0, black}, {1, color.RGBA{}}}}})
	cnv.FillPath(pathOf(func(p *Path) { p.Rect(0, 0, 140, 50) }), color.RGBA{0, 0, 255, 255})
	cnv.EndLayer()
	cnv.Restore()
	cnv.BeginLayerStyle(LayerStyle{Blend: BlendErase})
	cnv.FillPath(pathOf(func(p *Path) { p.Rect(0, 140, 300, 150) }), black)
	cnv.EndLayer()
	cnv.FillPath(pathOf(func(p *Path) { p.Circle(270, 40, 15) }), color.RGBA{0, 160, 0, 255})
}

func TestLayersSVG(t *testing.T) {
	cnv, img := newScaledTestCanvas(300, 150, 1)
	rec := NewSVGRecorder(300, 150)
	cnv.SetRecorder(rec)
	cnv.Save()
	cnv.TranslateAndClip(0, 0, 300, 150)
	cnv.ClipPath(pathOf(func(p *Path) { p.RoundedRect(0, 0, 300, 150, 20) }))
	drawLayers(cnv)
	cnv.Restore()
	svg := rec.String()

	// Well formed: the groups of the layers and of the clips nest
	dec := xml.NewDecoder(strings.NewReader(svg))
	for {
		if _, err := dec.Token(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("not well formed: %v\n%s", err, svg)
		}
	}
	for _, want := range []string{"mix-blend-mode:screen", `opacity="0.5"`, `<mask id=`, `mask-type="alpha"`, `mask="url(#`, "<linearGradient"} {
		if !strings.Contains(svg, want) {
			t.Errorf("no %s in\n%s", want, svg)
		}
	}
	// The eraser is not recorded: no black square over all
	if strings.Contains(svg, `d="M0 140L300 140L300 290L0 290Z"`) {
		t.Errorf("the eraser recorded as ink:\n%s", svg)
	}
	// Recorded after the eraser again
	if !strings.Contains(svg, `fill="#00a000"`) {
		t.Errorf("the drawing after the eraser is not recorded")
	}
	if dir := os.Getenv("NUI_SNAPSHOT_DIR"); dir != "" {
		os.WriteFile(dir+"/layers.svg", []byte(svg), 0o644)
		f, _ := os.Create(dir + "/layers.png")
		png.Encode(f, img)
		f.Close()
	}
	_ = image.Rect
}
