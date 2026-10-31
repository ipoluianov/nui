package ui

import (
	"bytes"
	"image"
	"math"
	"testing"
	"time"
)

// A chart following "now" moves by a fraction of a pixel on every update:
// the line has to move with it smoothly, not jump by whole pixels and change
// its shape
func TestTimeChartScrollsSmoothly(t *testing.T) {
	// ClearType fringes of the labels would count as the blue series
	NativeFontRendering = false
	defer func() { NativeFontRendering = true }()
	const w, h = 600, 300
	chart := NewTimeChart()
	chart.SetSize(w, h)
	src := NewTimeChartMemorySource()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	spike := base.Add(150 * time.Second)
	for s := 0; s <= 300; s++ {
		dt := base.Add(time.Duration(s) * time.Second)
		v := 0.1
		if dt.Equal(spike) {
			v = 1
		}
		src.AddPoint(NewTimeChartValue(dt, v))
	}
	chart.AddArea().AddSeries("s", src)

	// The weighted column of the series pixels in the upper half of the
	// plot, where only the spike is
	spikeX := func(from, to time.Time) float64 {
		chart.SetDefaultTimeRange(from, to)
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		cnv := NewCanvas(img)
		cnv.SetDirectTranslateAndClip(0, 0, w, h)
		chart.draw(cnv) // the first paint sets the plot width
		cnv = NewCanvas(img)
		cnv.SetDirectTranslateAndClip(0, 0, w, h)
		chart.draw(cnv)
		var sum, weight float64
		for y := h / 5; y < h/2; y++ {
			for x := 0; x < w; x++ {
				p := img.RGBAAt(x, y)
				// The series is blue; the grid and the text are gray
				if d := float64(p.B) - float64(p.R); d > 40 {
					sum += float64(x) * d
					weight += d
				}
			}
		}
		if weight == 0 {
			t.Fatal("no spike drawn")
		}
		return sum / weight
	}

	from, to := base.Add(10*time.Second), base.Add(290*time.Second)
	x0 := spikeX(from, to)
	pixel := to.Sub(from) / time.Duration(chart.lastPlotW)
	for k := 1; k <= 8; k++ {
		shift := pixel * time.Duration(k) / 4
		x := spikeX(from.Add(shift), to.Add(shift))
		want := x0 - float64(k)/4
		if math.Abs(x-want) > 0.1 {
			t.Errorf("shifted by %d/4 px: the spike is at %.2f, want %.2f", k, x, want)
		}
	}
}

// The labels of the vertical axis change their width with the data, and so
// does the plot: a paint draws the data for the width of the plot it comes
// to, not for that of the last paint. Otherwise a frame that repaints only
// a part of the window, the chart left out, leaves the chart drawn for the
// wrong width until it is repainted.
func TestTimeChartPaintAsNext(t *testing.T) {
	const w, h = 700, 360
	chart := NewTimeChart()
	chart.SetSize(w, h)
	src := NewTimeChartMemorySource()
	chart.AddArea().AddSeries("s", src)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	chart.SetDefaultTimeRange(base, base.Add(3000*time.Second))
	paint := func() *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		cnv := NewCanvas(img)
		cnv.SetDirectTranslateAndClip(0, 0, w, h)
		chart.draw(cnv)
		return img
	}
	// Narrow labels, then wider and wider ones: on each paint the plot is
	// narrower than on the last
	for _, scale := range []float64{1, 1000, 1e6, 1, 1e7} {
		points := make([]TimeChartPoint, 3000)
		for i := range points {
			v := scale * (1 + math.Sin(float64(i)*0.37)*math.Cos(float64(i)*0.011))
			points[i] = NewTimeChartValue(base.Add(time.Duration(i)*time.Second), v)
		}
		src.SetPoints(points)
		first, next := paint(), paint()
		if !bytes.Equal(first.Pix, next.Pix) {
			t.Errorf("data of scale %g: the paint differs from the next one", scale)
		}
	}
}
