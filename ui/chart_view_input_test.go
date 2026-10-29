package ui_test

import (
	"math"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// dragButton drags with the button from one point to another in steps
func dragButton(w *uitest.Window, button ui.MouseButton, from, to uitest.Point) {
	w.Move(from)
	w.Press(button, from)
	const steps = 8
	for i := 1; i <= steps; i++ {
		w.Move(uitest.Point{X: from.X + (to.X-from.X)*i/steps, Y: from.Y + (to.Y-from.Y)*i/steps})
	}
	w.Release(button, to)
}

// plotPoint returns the point of the window at the part (fx, fy) of the
// first pane's plot
func plotPoint(ch *ui.ChartView, fx, fy float64) uitest.Point {
	x, y, pw, ph := ch.Pane(0).Plot()
	return uitest.At(ch, int(x+pw*fx), int(y+ph*fy))
}

// sineChart is a line chart of 0..100 with the time series bindings
func sineChart() *ui.LineChart {
	ch := ui.NewLineChart()
	s := ch.Series("sine")
	for i := 0; i <= 100; i++ {
		s.AddXY(float64(i), math.Sin(float64(i)/10))
	}
	return ch
}

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestChartZoomBoxAndBack(t *testing.T) {
	ch := sineChart()
	w := showChart(t, &ch.ChartView, 500, 300)
	// Scientific profile: a left drag zooms into the rectangle
	w.Drag(plotPoint(&ch.ChartView, 0.2, 0.2), plotPoint(&ch.ChartView, 0.6, 0.8))
	lo, hi := ch.XRange()
	if !near(lo, 20, 2) || !near(hi, 60, 2) {
		t.Fatalf("zoomed to %.1f..%.1f, want about 20..60", lo, hi)
	}
	if ch.ZoomDepth() != 1 || ch.IsDefaultView() {
		t.Fatalf("depth %d, default view %v", ch.ZoomDepth(), ch.IsDefaultView())
	}
	ymin, ymax, manual := ch.Pane(0).YAxis().Range()
	if !manual || ymax-ymin > 1.5 {
		t.Fatalf("y not zoomed: %v..%v manual %v", ymin, ymax, manual)
	}
	// Dragged to the left: back one level
	w.Drag(plotPoint(&ch.ChartView, 0.6, 0.5), plotPoint(&ch.ChartView, 0.3, 0.5))
	if lo, hi := ch.XRange(); lo != 0 || hi != 100 || ch.ZoomDepth() != 0 {
		t.Fatalf("after zooming back: %v..%v depth %d", lo, hi, ch.ZoomDepth())
	}
}

func TestChartWheelAndEsc(t *testing.T) {
	ch := sineChart()
	w := showChart(t, &ch.ChartView, 500, 300)
	at := plotPoint(&ch.ChartView, 0.5, 0.5)
	w.Wheel(at, 0, 2)
	lo, hi := ch.XRange()
	if hi-lo >= 100 || !near((lo+hi)/2, 50, 3) {
		t.Fatalf("wheel zoomed to %.1f..%.1f", lo, hi)
	}
	// Esc resets the zoom of the chart in focus (a click focuses it)
	w.Click(at)
	w.Key(ui.KeyEsc, ui.KeyModifiers{})
	if lo, hi := ch.XRange(); lo != 0 || hi != 100 {
		t.Fatalf("after Esc: %v..%v", lo, hi)
	}
}

func TestChartTimeSeriesProfile(t *testing.T) {
	ch := sineChart()
	ch.Interaction().UseProfile(ui.ChartProfileTimeSeries)
	w := showChart(t, &ch.ChartView, 500, 300)
	// A left drag pans: the data follows the pointer
	w.Drag(plotPoint(&ch.ChartView, 0.6, 0.5), plotPoint(&ch.ChartView, 0.4, 0.5))
	lo, hi := ch.XRange()
	if !near(lo, 20, 2) || !near(hi, 120, 2) {
		t.Fatalf("panned to %.1f..%.1f, want about 20..120", lo, hi)
	}
	// A right drag zooms into a range of x only
	ch.ResetZoom()
	dragButton(w, ui.MouseButtonRight, plotPoint(&ch.ChartView, 0.1, 0.5), plotPoint(&ch.ChartView, 0.3, 0.5))
	lo, hi = ch.XRange()
	if !near(lo, 10, 2) || !near(hi, 30, 2) {
		t.Fatalf("right drag zoomed to %.1f..%.1f", lo, hi)
	}
	if _, _, manual := ch.Pane(0).YAxis().Range(); manual {
		t.Fatal("a zoom of x changed y")
	}
	// Shift + left drag selects
	w.Hold(ui.KeyModifiers{Shift: true})
	w.Drag(plotPoint(&ch.ChartView, 0.25, 0.5), plotPoint(&ch.ChartView, 0.75, 0.5))
	w.Hold(ui.KeyModifiers{})
	from, to, ok := ch.Selection()
	if !ok || !near(from, 15, 1) || !near(to, 25, 1) {
		t.Fatalf("selection %v %.1f..%.1f", ok, from, to)
	}
	w.MatchGolden("chart-selection")
}

func TestChartBindings(t *testing.T) {
	ch := sineChart()
	in := ch.Interaction()
	if err := in.Bind("Ctrl+Shift+LeftDrag@plot", ui.ChartActionPan); err != nil {
		t.Fatal(err)
	}
	if a, ok := in.Binding("Shift+Ctrl+LeftDrag"); !ok || a.Name() != "pan" {
		t.Fatalf("binding %v %q", ok, a.Name())
	}
	for _, bad := range []string{"Hyper+LeftDrag", "LeftDrag@nowhere", "Esc@plot2", "Swipe", ""} {
		if err := in.Bind(bad, ui.ChartActionPan); err == nil {
			t.Errorf("%q bound", bad)
		}
	}
	// Unbound: a left drag does nothing
	in.Unbind("LeftDrag")
	w := showChart(t, &ch.ChartView, 500, 300)
	w.Drag(plotPoint(&ch.ChartView, 0.2, 0.2), plotPoint(&ch.ChartView, 0.6, 0.8))
	if lo, hi := ch.XRange(); lo != 0 || hi != 100 {
		t.Fatalf("an unbound drag zoomed to %v..%v", lo, hi)
	}
	if got := in.Bindings()["Wheel"]; got != "zoom.wheel" {
		t.Fatalf("Bindings()[Wheel] = %q", got)
	}
}

// An action of the application's own gets the values under the pointer
func TestChartBindFunc(t *testing.T) {
	ch := sineChart()
	var startX, endX float64
	moves := 0
	ch.Interaction().BindFunc("Alt+LeftDrag", ui.ChartGestureFunc{
		Begin: func(e ui.ChartGestureEvent) bool { startX = e.DataX; return true },
		Move:  func(e ui.ChartGestureEvent) { moves++ },
		End:   func(e ui.ChartGestureEvent) { endX = e.DataX },
	})
	w := showChart(t, &ch.ChartView, 500, 300)
	w.Hold(ui.KeyModifiers{Alt: true})
	w.Drag(plotPoint(&ch.ChartView, 0.1, 0.5), plotPoint(&ch.ChartView, 0.9, 0.5))
	w.Hold(ui.KeyModifiers{})
	if !near(startX, 10, 1.5) || !near(endX, 90, 1.5) || moves == 0 {
		t.Fatalf("start %.1f end %.1f moves %d", startX, endX, moves)
	}
	if lo, hi := ch.XRange(); lo != 0 || hi != 100 {
		t.Fatal("the custom gesture zoomed")
	}
}

// OnBeforeAction may cancel an action or change its range
func TestChartBeforeAction(t *testing.T) {
	ch := sineChart()
	ch.Interaction().UseProfile(ui.ChartProfileTimeSeries)
	ch.OnBeforeAction.Connect(func(e *ui.ChartActionEvent) {
		switch e.Action.Name() {
		case "pan":
			e.Cancel = true
		case "select.range":
			e.From, e.To = math.Round(e.From/10)*10, math.Round(e.To/10)*10
		}
	})
	w := showChart(t, &ch.ChartView, 500, 300)
	w.Drag(plotPoint(&ch.ChartView, 0.6, 0.5), plotPoint(&ch.ChartView, 0.4, 0.5))
	if lo, hi := ch.XRange(); lo != 0 || hi != 100 {
		t.Fatalf("a cancelled pan moved the view to %v..%v", lo, hi)
	}
	w.Hold(ui.KeyModifiers{Shift: true})
	w.Drag(plotPoint(&ch.ChartView, 0.23, 0.5), plotPoint(&ch.ChartView, 0.67, 0.5))
	w.Hold(ui.KeyModifiers{})
	if from, to, _ := ch.Selection(); from != 20 || to != 70 {
		t.Fatalf("selection %v..%v, want snapped to 20..70", from, to)
	}
}

// The limits keep the view within the data and a least span
func TestChartLimits(t *testing.T) {
	ch := sineChart()
	ch.Interaction().UseProfile(ui.ChartProfileTimeSeries)
	ch.Interaction().SetLimits(ui.ChartLimits{PanBounds: ui.ChartPanData, MinSpanX: 5})
	w := showChart(t, &ch.ChartView, 500, 300)
	w.Drag(plotPoint(&ch.ChartView, 0.6, 0.5), plotPoint(&ch.ChartView, 0.1, 0.5))
	if lo, hi := ch.XRange(); lo != 0 || hi != 100 {
		t.Fatalf("panned out of the data: %v..%v", lo, hi)
	}
	for i := 0; i < 30; i++ {
		w.Wheel(plotPoint(&ch.ChartView, 0.5, 0.5), 0, 1)
	}
	if lo, hi := ch.XRange(); hi-lo < 5-1e-9 {
		t.Fatalf("zoomed below the least span: %v..%v", lo, hi)
	}
}

func TestChartLegendToggle(t *testing.T) {
	ch := ui.NewLineChart()
	a := ch.Series("A").AddValues(1, 2, 3)
	b := ch.Series("B").AddValues(3, 2, 1)
	toggled := 0
	ch.OnSeriesToggled.Connect(func(ui.ChartSeries) { toggled++ })
	w := showChart(t, &ch.ChartView, 400, 260)
	// The entry of A: at the left end of the legend strip above the plot
	x, y, _, _ := ch.Pane(0).Plot()
	entry := uitest.At(ch, int(x)+8, int(y)-8)
	w.Click(entry)
	if a.Visible() || !b.Visible() || toggled != 1 {
		t.Fatalf("after a click on A: A %v B %v toggled %d", a.Visible(), b.Visible(), toggled)
	}
	w.Click(entry)
	if !a.Visible() {
		t.Fatal("A not shown again")
	}
}

func TestChartHoverAndClick(t *testing.T) {
	ch := ui.NewLineChart()
	ch.Series("Load").AddValues(3, 5, 4, 7, 6, 9, 8, 10, 9, 12)
	var clicked ui.ChartHit
	ch.OnPointClicked.Connect(func(h ui.ChartHit) { clicked = h })
	w := showChart(t, &ch.ChartView, 480, 300)
	// On the point 4 (x 4, value 6)
	x, y, pw, _ := ch.Pane(0).Plot()
	px := x + pw*4/9
	py := ch.Pane(0).YAxis().ToPixel(6)
	_ = y
	at := uitest.At(ch, int(px), int(py))
	w.Move(at)
	hits := ch.Hits()
	if len(hits) != 1 || hits[0].Index != 4 || hits[0].YLabel != "6.00" && hits[0].YLabel != "6.0" && hits[0].YLabel != "6" {
		t.Fatalf("hits %+v", hits)
	}
	w.MatchGolden("chart-hover")
	w.Click(at)
	if clicked.Series == nil || clicked.X != 4 {
		t.Fatalf("clicked %+v", clicked)
	}
}

// The keys pan and zoom the focused chart
func TestChartKeyboard(t *testing.T) {
	ch := sineChart()
	w := showChart(t, &ch.ChartView, 500, 300)
	w.Click(plotPoint(&ch.ChartView, 0.5, 0.5)) // focus
	w.Key(ui.KeyArrowRight, ui.KeyModifiers{})
	if lo, hi := ch.XRange(); !near(lo, 10, 0.5) || !near(hi, 110, 0.5) {
		t.Fatalf("Right moved the view to %v..%v", lo, hi)
	}
	w.Key(ui.KeyEqual, ui.KeyModifiers{})
	if lo, hi := ch.XRange(); !near(hi-lo, 80, 0.5) {
		t.Fatalf("Plus zoomed to %v..%v", lo, hi)
	}
	w.Key(ui.KeyHome, ui.KeyModifiers{})
	if lo, hi := ch.XRange(); lo != 0 || hi != 100 {
		t.Fatalf("Home: %v..%v", lo, hi)
	}
}

// A chart of the XML layout, the data in its attributes
func TestChartXMLLayout(t *testing.T) {
	form := ui.NewForm()
	form.SetSize(500, 300)
	err := form.Panel().SetLayout(`<column>
		<linechart id="load" values="3,5,4,7,6" title="Load" ytitle="%"/>
		<piechart id="os" items="Linux:60|Windows:30" donut="0.5"/>
	</column>`, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	uitest.Show(t, form)
	lc, ok := form.Panel().FindWidgetByName("load").(*ui.LineChart)
	if !ok {
		t.Fatalf("no line chart: %T", form.Panel().FindWidgetByName("load"))
	}
	if n := lc.Series("").Source().(*ui.ChartMemorySource).Len(); n != 5 {
		t.Fatalf("%d values", n)
	}
	if _, ok := form.Panel().FindWidgetByName("os").(*ui.PieChart); !ok {
		t.Fatal("no pie chart")
	}
}
