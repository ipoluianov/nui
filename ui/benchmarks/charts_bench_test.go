package benchmarks

import (
	"image"
	"math"
	"testing"

	"github.com/ipoluianov/nui/ui"
)

// renderOnce paints the chart into an image of the size, as a frame does
func renderFrame(c *ui.ChartView, w, h int) *image.RGBA {
	return c.Render(w, h, 1, nil)
}

// BenchmarkChartLines: 10 lines of n points each in memory, the whole of
// them in a chart 1920 pixels wide: a frame (the goal: 16 ms)
func BenchmarkChartLines(b *testing.B) {
	for _, n := range []int{100_000, 1_000_000} {
		b.Run(itoa(n), func(b *testing.B) {
			if n > 100_000 && testing.Short() {
				b.Skip("-short")
			}
			ch := ui.NewChartView()
			for s := 0; s < 10; s++ {
				src := ui.NewChartMemorySource()
				samples := make([]ui.ChartSample, n)
				for i := range samples {
					samples[i] = ui.ChartValue(float64(i), math.Sin(float64(i)/5000+float64(s))*10+math.Sin(float64(i)*1.7))
				}
				src.SetSamples(samples)
				ch.Pane(0).AddLine("s").SetSource(src).SetLineWidth(1)
			}
			renderFrame(ch, 1920, 600)
			b.ResetTimer()
			for b.Loop() {
				renderFrame(ch, 1920, 600)
			}
		})
	}
}

// BenchmarkChartLive: 20 lines of a live feed, a sample appended to each
// and a frame painted, keeping the last 3600 samples
func BenchmarkChartLive(b *testing.B) {
	ch := ui.NewChartView()
	var lines []*ui.LineSeries
	for s := 0; s < 20; s++ {
		l := ch.Pane(0).AddLine("s").SetMaxPoints(3600).SetLineWidth(1)
		lines = append(lines, l)
	}
	x := 0.0
	for ; x < 3600; x++ {
		for i, l := range lines {
			l.AddXY(x, math.Sin(x/50+float64(i)))
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		for i, l := range lines {
			l.AddXY(x, math.Sin(x/50+float64(i)))
		}
		x++
		renderFrame(ch, 1280, 500)
	}
}

// BenchmarkChartScatterMillion: a million points drawn as their density
// (the goal: 50 ms)
func BenchmarkChartScatterMillion(b *testing.B) {
	if testing.Short() {
		b.Skip("-short")
	}
	ch := ui.NewXYChart()
	xs, ys := make([]float64, 1_000_000), make([]float64, 1_000_000)
	for i := range xs {
		a := float64(i) * 0.6180339887
		r := math.Sqrt(float64(i%10000) / 10000)
		xs[i], ys[i] = r*math.Cos(a*6.283), r*math.Sin(a*6.283)
	}
	ch.Pane(0).AddScatter("p").SetXY(xs, ys)
	renderFrame(ch, 1280, 800)
	b.ResetTimer()
	for b.Loop() {
		renderFrame(ch, 1280, 800)
	}
}

// BenchmarkChartHeatmap: a grid of 2000×2000 values (the goal: 30 ms)
func BenchmarkChartHeatmap(b *testing.B) {
	ch := ui.NewXYChart()
	grid := make([][]float64, 2000)
	for r := range grid {
		grid[r] = make([]float64, 2000)
		for c := range grid[r] {
			grid[r][c] = math.Sin(float64(r)/100) * math.Cos(float64(c)/70)
		}
	}
	ch.Pane(0).AddHeatmap("h").SetGrid(grid, 0, 1, 0, 1)
	renderFrame(ch, 1280, 800)
	b.ResetTimer()
	for b.Loop() {
		renderFrame(ch, 1280, 800)
	}
}

func itoa(n int) string {
	switch {
	case n >= 1_000_000:
		return string(rune('0'+n/1_000_000)) + "M"
	case n >= 1000:
		s := ""
		for v := n / 1000; v > 0; v /= 10 {
			s = string(rune('0'+v%10)) + s
		}
		return s + "k"
	}
	return "n"
}
