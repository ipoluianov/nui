package ui_test

import (
	"bytes"
	"encoding/xml"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func TestScatterAndBubbles(t *testing.T) {
	ch := ui.NewXYChart()
	pts := ch.Pane(0).AddScatter("Measurements")
	for i := 0; i < 60; i++ {
		x := float64(i)
		pts.Add(x, 20+15*math.Sin(x/7)+5*math.Sin(x*3.1))
	}
	bub := ch.Pane(0).AddScatter("Cities").SetBubbleSizes(6, 36)
	bub.SetColorScale(ui.CurrentChartTheme().Sequential, 0, 0)
	for i := 0; i < 12; i++ {
		bub.AddPoint(ui.ScatterPoint{X: float64(i*5 + 2), Y: 30 + float64(i%4)*6, Size: float64(1 + i*i), Value: float64(i)})
	}
	w := showChart(t, ch, 520, 320)
	w.MatchGolden("chart-scatter-bubbles")
}

// A million points are drawn as their density, quickly
func TestScatterDensity(t *testing.T) {
	ch := ui.NewXYChart()
	s := ch.Pane(0).AddScatter("Noise")
	xs, ys := make([]float64, 200000), make([]float64, 200000)
	for i := range xs {
		// Two clouds of a deterministic noise
		a := float64(i) * 0.61803398875
		r := math.Sqrt(float64(i%1000) / 1000)
		cx := 0.0
		if i%3 == 0 {
			cx = 3
		}
		xs[i] = cx + r*math.Cos(a*6.283)
		ys[i] = r * math.Sin(a*6.283) * 0.7
	}
	s.SetXY(xs, ys)
	w := showChart(t, ch, 480, 300)
	w.MatchGolden("chart-scatter-density")
}

func TestStackedPercent(t *testing.T) {
	ch := ui.NewLineChart()
	for i, name := range []string{"Coal", "Gas", "Solar"} {
		s := ch.Series(name).SetStack("energy").SetFill(true, 0.8, false).SetLineWidth(1)
		for x := 0; x < 12; x++ {
			s.AddXY(float64(x), float64(10+i*5)+float64(x*(i+1))*0.8)
		}
	}
	ch.Pane(0).SetStackPercent("energy", true)
	ch.Pane(0).YAxis().Scale().(*ui.LinearScale).Unit = "%"
	w := showChart(t, &ch.ChartView, 480, 300)
	w.MatchGolden("chart-stacked-percent")
	if _, hi, _ := ch.Pane(0).YAxis().Range(); math.Abs(hi-105) > 1 {
		t.Fatalf("the top of a 100%% stack at %v", hi)
	}
}

func TestSparkline(t *testing.T) {
	form := ui.NewForm()
	form.SetSize(300, 120)
	line := ui.NewSparkline().SetValues([]float64{3, 4, 2, 5, 7, 6, 8, 7, 9, 12, 10})
	bars := ui.NewSparkline().SetKind(ui.ChartSparkBars).SetValues([]float64{3, 4, -2, 5, 7, -6, 8})
	wl := ui.NewSparkline().SetKind(ui.ChartSparkWinLoss).SetValues([]float64{1, -1, 1, 1, -1, 0, 1, -1})
	form.Panel().AddWidget(0, 0, line)
	form.Panel().AddWidget(1, 0, bars)
	form.Panel().AddWidget(2, 0, wl)
	w := showForm(t, form)
	w.MatchGolden("chart-sparklines")
}

func TestChartExports(t *testing.T) {
	bar := ui.NewBarChart()
	bar.SetTitle("Sales & <costs>")
	bar.Add("Jan", 120).Add("Feb", 95)
	bar.Series("Plan").Set("Jan", 100).Set("Feb", 110)
	showChart(t, bar, 400, 260)

	img := bar.Render(400, 260, 2, nil)
	if img.Bounds() != image.Rect(0, 0, 800, 520) {
		t.Fatalf("render %v", img.Bounds())
	}
	var svg bytes.Buffer
	if err := bar.WriteSVG(&svg, 400, 260, ui.ChartThemeOf(&ui.LightPalette, false)); err != nil {
		t.Fatal(err)
	}
	// Well-formed XML with the shapes and the texts
	dec := xml.NewDecoder(bytes.NewReader(svg.Bytes()))
	paths, texts := 0, []string{}
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok {
			switch se.Name.Local {
			case "path":
				paths++
			case "text":
				var s string
				dec.DecodeElement(&s, &se)
				texts = append(texts, s)
			}
		}
	}
	all := strings.Join(texts, "|")
	if paths < 4 || !strings.Contains(all, "Sales & <costs>") || !strings.Contains(all, "Jan") || !strings.Contains(all, "Plan") {
		t.Fatalf("svg: %d paths, texts %q", paths, all)
	}
	var csvOut strings.Builder
	if err := bar.WriteCSV(&csvOut, true); err != nil {
		t.Fatal(err)
	}
	want := "series,x,value,open,high,low,close\n,Jan,120,,,,\n,Feb,95,,,,\nPlan,Jan,100,,,,\nPlan,Feb,110,,,,\n"
	if csvOut.String() != want {
		t.Fatalf("csv:\n%s\nwant\n%s", csvOut.String(), want)
	}
	var pdf bytes.Buffer
	if err := bar.PrintJob("Sales").WritePDF(&pdf, nil); err != nil || !bytes.HasPrefix(pdf.Bytes(), []byte("%PDF")) {
		t.Fatalf("pdf: %v, %q", err, pdf.Bytes()[:min(8, pdf.Len())])
	}
}

func showForm(t *testing.T, form *ui.Form) *uitest.Window {
	t.Helper()
	return uitest.Show(t, form)
}

// Every kind of chart in one picture, in the light and the dark theme
func TestChartKindsBothThemes(t *testing.T) {
	defer ui.ApplyLightTheme()
	for _, theme := range []string{"light", "dark"} {
		if theme == "dark" {
			ui.ApplyDarkTheme()
		} else {
			ui.ApplyLightTheme()
		}
		form := ui.NewForm()
		form.SetSize(900, 520)
		p := form.Panel()

		line := ui.NewLineChart()
		line.SetTitle("Lines")
		a, b := line.Series("A").SetShape(ui.ChartLineSpline), line.Series("B").SetShape(ui.ChartLineStepAfter)
		for i := 0; i < 12; i++ {
			a.AddXY(float64(i), 5+3*math.Sin(float64(i)/2))
			b.AddXY(float64(i), 4+float64(i%3))
		}
		area := ui.NewLineChart()
		area.SetTitle("Areas")
		for k, name := range []string{"X", "Y"} {
			s := area.Series(name).SetStack("s").SetFill(true, 0.7, false).SetLineWidth(1)
			for i := 0; i < 12; i++ {
				s.AddXY(float64(i), float64(3+k)+2*math.Sin(float64(i+k*3)/3))
			}
		}
		bar := ui.NewBarChart()
		bar.SetTitle("Bars")
		bar.Series("2025").Set("Q1", 4).Set("Q2", 6).Set("Q3", 5)
		bar.Series("2026").Set("Q1", 5).Set("Q2", 7).Set("Q3", 3)
		pie := ui.NewPieChart()
		pie.SetTitle("Pie")
		pie.SetDonut(0.5)
		pie.Add("One", 5).Add("Two", 3).Add("Three", 2)
		sc := ui.NewXYChart()
		sc.SetTitle("Scatter")
		pts := sc.Pane(0).AddScatter("P")
		for i := 0; i < 40; i++ {
			pts.Add(float64(i%10), float64(i)/4+math.Sin(float64(i)))
		}
		cd := ui.NewXYChart()
		cd.SetTitle("Candles")
		c := cd.Pane(0).AddCandles("C")
		v := 10.0
		for i := 0; i < 20; i++ {
			o := v
			v += math.Sin(float64(i) * 1.9)
			c.AddOHLC(float64(i), o, math.Max(o, v)+0.4, math.Min(o, v)-0.4, v)
		}
		p.AddWidget(0, 0, line)
		p.AddWidget(0, 1, area)
		p.AddWidget(0, 2, bar)
		p.AddWidget(1, 0, pie)
		p.AddWidget(1, 1, sc)
		p.AddWidget(1, 2, cd)
		w := uitest.Show(t, form)
		w.MatchGolden("chart-kinds-" + theme)
		form.Close()
	}
}
