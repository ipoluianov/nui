// Package chartgallery shows the charts of nui: every kind on a page of its
// own, the same data in the light and the dark theme, the gestures of each
// profile, the export of a chart to PNG, SVG and CSV.
package chartgallery

import (
	"errors"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// gallery is the window's content: a tab per kind of chart
type gallery struct {
	ui.Widget
	tabs   *ui.TabWidget
	charts []*ui.ChartView // of each page, the one the toolbar works on
	status *ui.Label
}

// NewForm returns the Chart Gallery
func NewForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Chart Gallery")
	form.SetSize(1100, 740)
	form.Panel().AddWidget(0, 0, newGallery())
	return form
}

func newGallery() *gallery {
	g := &gallery{}
	g.InitWidget()

	bar := g.AddPanel(0, 0)
	dark := ui.NewToggleSwitch("Dark theme")
	dark.SetChecked(ui.IsDarkTheme)
	dark.SetOnStateChanged(func() {
		if dark.Checked() {
			ui.ApplyDarkTheme()
		} else {
			ui.ApplyLightTheme()
		}
	})
	bar.AddWidget(0, 0, dark)
	bar.AddLabel(0, 1, "Mouse:")
	profiles := ui.NewComboBox()
	profiles.SetItems([]string{"Scientific", "Time series", "Finance", "Dashboard", "Static"})
	profiles.SetSelectedIndex(-1)
	profiles.SetOnSelectedIndexChanged(func() {
		p := []ui.ChartProfile{ui.ChartProfileScientific, ui.ChartProfileTimeSeries, ui.ChartProfileFinance,
			ui.ChartProfileDashboard, ui.ChartProfileStatic}[profiles.SelectedIndex()]
		if c := g.current(); c != nil {
			c.Interaction().UseProfile(p)
			g.showBindings()
		}
	})
	bar.AddWidget(0, 2, profiles)
	bar.AddButton(0, 3, "Reset zoom", func() {
		if c := g.current(); c != nil {
			c.ResetZoom()
		}
	})
	bar.AddButton(0, 4, "Copy image", func() {
		if c := g.current(); c != nil {
			c.CopyImage()
		}
	})
	bar.AddButton(0, 5, "Save SVG...", func() { g.save("svg") })
	bar.AddButton(0, 6, "Save CSV...", func() { g.save("csv") })
	bar.AddHSpacer(0, 7)

	g.tabs = ui.NewTabWidget()
	g.AddWidget(1, 0, g.tabs)
	g.addPage("Lines", g.linesPage())
	g.addPage("Areas", g.areasPage())
	g.addPage("Bars", g.barsPage())
	g.addPage("Pie", g.piePage())
	g.addPage("Scatter", g.scatterPage())
	g.addPage("Finance", g.financePage())
	g.addPage("Time series", g.timePage())
	g.addPage("Dashboard", g.dashboardPage())

	g.status = g.AddLabel(2, 0, "")
	g.tabs.OnCurrentChanged.Connect(func(int) { g.showBindings() })
	g.showBindings()
	return g
}

// addPage adds a tab; the page's chart was given to track before
func (g *gallery) addPage(name string, page ui.Widgeter) {
	g.tabs.AddPage(name, page)
}

// current returns the chart of the current page
func (g *gallery) current() *ui.ChartView {
	i := g.tabs.CurrentIndex()
	if i < 0 || i >= len(g.charts) {
		return nil
	}
	return g.charts[i]
}

// showBindings writes the gestures of the current chart in the status line
func (g *gallery) showBindings() {
	c := g.current()
	if c == nil {
		return
	}
	var parts []string
	for gesture, action := range c.Interaction().Bindings() {
		if strings.Contains(gesture, "Drag") || strings.Contains(gesture, "Wheel") {
			parts = append(parts, gesture+": "+action)
		}
	}
	sort.Strings(parts)
	g.status.SetText(strings.Join(parts, "   "))
}

// save writes the current chart to a file of the format
func (g *gallery) save(format string) {
	c := g.current()
	if c == nil {
		return
	}
	opts := ui.SaveFileDialogOptions{
		Title:           "Save " + strings.ToUpper(format),
		DefaultFileName: "chart." + format,
		Filters:         []ui.FileDialogFilter{{DisplayName: strings.ToUpper(format) + " files", Patterns: []string{"*." + format}}},
	}
	g.Form().ShowSaveFileDialog(opts, func(path string, err error) {
		if errors.Is(err, ui.ErrNoFileDialog) || path == "" || err != nil {
			return
		}
		f, err := os.Create(path)
		if err != nil {
			ui.ShowMessageBox(g, "Save", err.Error())
			return
		}
		defer f.Close()
		if format == "svg" {
			err = c.WriteSVG(f, c.Width(), c.Height(), nil)
		} else {
			err = c.WriteCSV(f, true)
		}
		if err != nil {
			ui.ShowMessageBox(g, "Save", err.Error())
		}
	})
}

// track makes the chart the one the toolbar works on for the page added next
func (g *gallery) track(c *ui.ChartView) {
	g.charts = append(g.charts, c)
}

func (g *gallery) linesPage() ui.Widgeter {
	ch := ui.NewLineChart()
	ch.SetTitle("Daily temperature")
	plan := ch.Series("Forecast").SetShape(ui.ChartLineSpline).SetDash(6, 4)
	fact := ch.Series("Measured")
	steps := ch.Series("Heating on").SetShape(ui.ChartLineStepAfter)
	steps.SetAxisNames("x", "state")
	state := ch.Pane(0).AddAxis("state", ui.ChartAxisRight, nil)
	state.SetTitle("Heating").SetRange(-0.2, 3)
	for h := 0; h <= 48; h++ {
		x := float64(h) / 2
		plan.AddXY(x, 18+6*math.Sin((x-9)*math.Pi/12))
		fact.AddXY(x, 18+6*math.Sin((x-9.5)*math.Pi/12)+math.Sin(x*2.3)*0.8)
		on := 0.0
		if x < 7 || x > 19 {
			on = 1
		}
		steps.AddXY(x, on)
	}
	ch.XAxis().SetTitle("Hour")
	ch.Pane(0).YAxis().SetTitle("°C")
	ann := ch.Pane(0).Annotations()
	ann.AddHLine(23, "Comfort limit").SetDash(4, 3)
	ann.AddXZone(0, 6, "Night")
	ann.AddLabel(15, 24.1, "Peak")
	g.track(&ch.ChartView)
	return ch
}

func (g *gallery) areasPage() ui.Widgeter {
	ch := ui.NewLineChart()
	ch.SetTitle("Energy mix, a year")
	sources := []string{"Coal", "Gas", "Wind", "Solar"}
	for i, name := range sources {
		s := ch.Series(name).SetStack("mix").SetFill(true, 0.75, false).SetLineWidth(1)
		for m := 0; m < 12; m++ {
			v := 20 + 10*float64(i) + 8*math.Sin(float64(m+i*3)/2)
			if name == "Solar" {
				v = 10 + 25*math.Sin(float64(m)*math.Pi/11)
			}
			s.AddXY(float64(m+1), v)
		}
	}
	ch.XAxis().SetTitle("Month")
	page := ui.NewPanel()
	percent := ui.NewToggleSwitch("100%")
	percent.SetOnStateChanged(func() { ch.Pane(0).SetStackPercent("mix", percent.Checked()) })
	page.AddWidget(0, 0, percent)
	page.AddWidget(1, 0, ch)
	g.track(&ch.ChartView)
	return page
}

func (g *gallery) barsPage() ui.Widgeter {
	bar := ui.NewBarChart()
	bar.SetTitle("Sales by quarter")
	for i, year := range []string{"2024", "2025", "2026"} {
		s := bar.Series(year)
		for q, quarter := range []string{"Q1", "Q2", "Q3", "Q4"} {
			s.Set(quarter, 80+float64(i*15)+30*math.Sin(float64(q+i)))
		}
	}
	bar.Series("2026").SetLabels(true)
	page := ui.NewPanel()
	row := page.AddPanel(0, 0)
	horizontal := ui.NewToggleSwitch("Horizontal")
	horizontal.SetOnStateChanged(func() { bar.SetHorizontal(horizontal.Checked()) })
	stacked := ui.NewToggleSwitch("Stacked")
	stacked.SetOnStateChanged(func() { bar.SetStacked(stacked.Checked()) })
	row.AddWidget(0, 0, horizontal)
	row.AddWidget(0, 1, stacked)
	row.AddHSpacer(0, 2)
	page.AddWidget(1, 0, bar)
	g.track(&bar.ChartView)
	return page
}

func (g *gallery) piePage() ui.Widgeter {
	page := ui.NewPanel()
	pie := ui.NewPieChart()
	pie.SetTitle("Desktop OS")
	pie.Add("Windows", 71).Add("macOS", 15).Add("Linux", 4).Add("ChromeOS", 2).Add("Unknown", 8)
	donut := ui.NewPieChart()
	donut.SetTitle("Disk usage")
	donut.SetDonut(0.55)
	donut.Pie().SetLabels(ui.ChartPieLabelNamePercent)
	donut.Add("Photos", 120).Add("Video", 310).Add("Apps", 80).Add("Free", 290)
	gauge := ui.NewPieChart()
	gauge.SetTitle("CPU load")
	gauge.SetLegend(ui.ChartLegendNone)
	gauge.Pie().SetAngles(math.Pi, math.Pi).SetInnerRadius(0.7).SetLabels(ui.ChartPieLabelNone)
	gauge.Pie().SetSliceColor("Load", ui.CurrentChartTheme().Warning)
	gauge.Add("Load", 72).Add("Idle", 28)
	page.AddWidget(0, 0, pie)
	page.AddWidget(0, 1, donut)
	page.AddWidget(0, 2, gauge)
	g.track(&pie.ChartView)
	return page
}

func (g *gallery) scatterPage() ui.Widgeter {
	ch := ui.NewXYChart()
	ch.SetTitle("Height and weight")
	men := ch.Pane(0).AddScatter("Men")
	women := ch.Pane(0).AddScatter("Women").SetMarker(ui.MarkerDiamond, 7)
	for i := 0; i < 150; i++ {
		a := float64(i) * 2.39996
		r := math.Sqrt(float64(i) / 150)
		men.Add(178+9*r*math.Cos(a), 80+12*r*math.Sin(a)+0.4*(9*r*math.Cos(a)))
		women.Add(165+8*r*math.Sin(a), 63+10*r*math.Cos(a)+0.4*(8*r*math.Sin(a)))
	}
	ch.XAxis().SetTitle("Height, cm")
	ch.Pane(0).YAxis().SetTitle("Weight, kg")
	g.track(ch)
	return ch
}

func (g *gallery) financePage() ui.Widgeter {
	ch := ui.NewChartView()
	ch.SetTitle("ACME, hourly")
	ch.XAxis().SetScale(ui.NewTimeScale(time.Unix(0, 0), time.Unix(1, 0)))
	ch.Interaction().UseProfile(ui.ChartProfileFinance)
	price := ch.Pane(0)
	price.SetWeight(3)
	candles := price.AddCandles("ACME")
	ma := price.AddLine("Average 20")
	ma.SetMarkers(ui.ChartMarkersNever, ui.MarkerCircle, 0)
	volume := ch.AddPane().AddBars("Volume")
	volume.SetUpDownColors(true)
	vol := ui.NewChartMemorySource()
	start := time.Date(2026, 3, 2, 9, 0, 0, 0, time.Local)
	p := 100.0
	var closes []float64
	for i := 0; i < 160; i++ {
		x := ui.ChartTime(start.Add(time.Duration(i) * time.Hour))
		open := p
		p += 2.2*math.Sin(float64(i)*1.7) + 0.6*math.Sin(float64(i)/9)
		hi, lo := math.Max(open, p)+1+math.Abs(math.Sin(float64(i))), math.Min(open, p)-1-math.Abs(math.Cos(float64(i)))
		candles.AddOHLC(x, open, hi, lo, p)
		v := 1000 + 600*math.Abs(math.Sin(float64(i)*0.9))
		vol.Append(ui.ChartSample{X: x, First: open, Last: v, High: v, Low: v, HasGood: true})
		closes = append(closes, p)
		if len(closes) >= 20 {
			sum := 0.0
			for _, c := range closes[len(closes)-20:] {
				sum += c
			}
			ma.AddXY(x, sum/20)
		}
	}
	// The volume bars are colored by their candle: First is the open, the
	// direction the close took is in Last vs First of the candle's sample
	volume.SetSource(ui.ChartSourceFunc(func(from, to, group float64) []ui.ChartSample {
		samples := vol.GetData(from, to, 0)
		cs := candles.Source().GetData(from, to, 0)
		for i := range samples {
			if i < len(cs) {
				up := cs[i].Last >= cs[i].First
				samples[i].First = 0
				if !up {
					samples[i].First = samples[i].Last + 1
				}
			}
		}
		return samples
	}))
	g.track(ch)
	return ch
}

func (g *gallery) timePage() ui.Widgeter {
	ch := ui.NewTimeChart()
	src, avg := ui.NewTimeChartMemorySource(), ui.NewTimeChartMemorySource()
	area := ch.AddArea()
	area.AddSeries("Signal", src)
	area.AddSeries("Average", avg)
	t0 := time.Now().Add(-10 * time.Minute)
	n := 0
	sum := 0.0
	add := func() {
		dt := t0.Add(time.Duration(n) * time.Second)
		v := 50 + 20*math.Sin(float64(n)/40) + 8*math.Sin(float64(n)*1.3)
		if n%170 == 85 {
			v += 35
			area.AddMarker(ui.TimeChartMarker{DT: dt, Value: v, Text: "spike"})
		}
		src.AddPoint(ui.NewTimeChartValue(dt, v))
		sum = sum*0.95 + v*0.05
		avg.AddPoint(ui.NewTimeChartValue(dt, sum/(1-math.Pow(0.95, float64(n+1)))))
		n++
	}
	for i := 0; i < 600; i++ {
		add()
	}
	ch.SetDefaultTimeRange(t0, t0.Add(10*time.Minute))
	// A new sample a second: the chart follows it until panned or zoomed
	ch.AddTimer(1000, func() {
		add()
		last := t0.Add(time.Duration(n) * time.Second)
		ch.SetDefaultTimeRange(last.Add(-10*time.Minute), last)
	})
	g.track(&ch.ChartView)
	return ch
}

func (g *gallery) dashboardPage() ui.Widgeter {
	page := ui.NewPanel()
	names := []string{"CPU", "Memory", "Requests/s", "Errors"}
	for i, name := range names {
		page.AddLabel(i, 0, name)
		sp := ui.NewSparkline()
		vals := make([]float64, 40)
		for k := range vals {
			vals[k] = 50 + 30*math.Sin(float64(k+i*7)/5) + 10*math.Sin(float64(k*3+i))
		}
		switch i {
		case 2:
			sp.SetKind(ui.ChartSparkBars)
		case 3:
			sp.SetKind(ui.ChartSparkWinLoss)
			for k := range vals {
				vals[k] = math.Sin(float64(k * (i + 2)))
			}
		}
		sp.SetValues(vals)
		sp.SetMinWidth(220)
		page.AddWidget(i, 1, sp)
		page.AddLabel(i, 2, ui.FormatChartNumber(vals[len(vals)-1], 1, false))
	}
	bar := ui.NewBarChart()
	bar.SetTitle("Tickets by team")
	bar.SetHorizontal(true)
	for _, t := range [][2]any{{"Platform", 34.0}, {"Mobile", 21.0}, {"Web", 45.0}, {"Data", 12.0}} {
		bar.Add(t[0].(string), t[1].(float64))
	}
	bar.Series("").SetLabels(true)
	page.AddWidgetSpan(len(names), 0, 1, 3, bar)
	g.track(&bar.ChartView)
	return page
}
