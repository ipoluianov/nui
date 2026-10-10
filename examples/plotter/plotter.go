// Package plotter is a function plotter: type a formula of x and see its
// graph.
package plotter

import (
	"fmt"
	"math"
	"strings"

	"github.com/ipoluianov/nui/ui"
)

var presets = []string{
	"sin(x) * exp(-x/10)",
	"a*sin(b*x)",
	"sin(x)/x",
	"x^2 - 3",
	"a*x^3 - b*x",
	"sqrt(abs(x)) * cos(b*x)",
	"floor(x) + a*sin(b*x)/4",
	"exp(-x^2/a)",
}

// Plotter is the main window's content.
type Plotter struct {
	ui.Widget

	formula *ui.EditableComboBox
	status  *ui.Label
	chart   *ui.Chart
	values  *ui.Table

	a, b         *ui.Slider
	aValue       *ui.Label
	bValue       *ui.Label
	xFrom, xTo   *ui.NumBox
	pointCount   *ui.NumBox
	animate      *ui.ToggleSwitch
	animateSpeed float64 // change of b per tick; the sign is the direction

	points []ui.ChartPoint
}

// NewForm returns the Function Plotter: a formula typed into an
// EditableComboBox (with presets and history) is parsed and drawn on a
// Chart, with the parse errors in a red Label. Sliders change the
// parameters a and b, NumBoxes the range and the number of points; the
// "Animate" ToggleSwitch sweeps b with a widget timer. A Table shows a few
// values (with a context menu to copy them), and "Export CSV..." saves the
// curve via the system save dialog.
func NewForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Function Plotter")
	form.SetSize(1040, 720)
	form.Panel().AddWidget(0, 0, newPlotter())
	return form
}

func newPlotter() *Plotter {
	var c Plotter
	c.InitWidget()
	c.animateSpeed = 0.02

	// Formula line
	top := c.AddPanel(0, 0)
	top.AddLabel(0, 0, "f(x) =")
	c.formula = ui.NewEditableComboBox()
	c.formula.SetHint("Type a formula of x, a and b, e.g. a*sin(b*x)")
	c.formula.SetItems(presets)
	c.formula.SetAutoComplete(false) // the drop-down button shows the presets
	c.formula.SetText(presets[0])
	c.formula.SetOnTextChanged(func() { c.replot(false) })
	c.formula.SetOnAccept(func(string) { c.replot(true) })
	top.AddWidget(0, 1, c.formula)
	top.AddButton(0, 2, "Plot", func() { c.replot(true) })
	// A Label sizes itself to its text; the fixed-width panel around it
	// keeps the formula box from jumping as the message changes
	statusBox := top.AddPanel(0, 3)
	statusBox.SetMinWidth(240)
	statusBox.SetMaxWidth(240)
	c.status = statusBox.AddLabel(0, 0, "")

	body := c.AddPanel(1, 0)

	c.chart = ui.NewChart()
	c.chart.SetProp("xlabel", "x")
	c.chart.SetProp("ylabel", "f(x)")
	c.chart.SetProp("tickcount", 9)
	body.AddWidget(0, 0, c.chart)

	side := body.AddPanel(0, 1)
	side.SetMinWidth(300)
	side.SetMaxWidth(300)

	// Parameters
	params := ui.NewGroupBox("Parameters")
	side.AddWidget(0, 0, params)
	c.a, c.aValue = c.addParameter(params, 0, "a", 1)
	c.b, c.bValue = c.addParameter(params, 1, "b", 2)
	c.animate = ui.NewToggleSwitch("Animate b")
	c.animate.SetTooltip("Sweeps the parameter b back and forth")
	params.AddWidget(2, 1, c.animate)

	// Range
	rng := ui.NewGroupBox("Range")
	side.AddWidget(1, 0, rng)
	c.xFrom = c.addNumber(rng, 0, "X from", -10, 1)
	c.xTo = c.addNumber(rng, 1, "X to", 20, 1)
	c.pointCount = c.addNumber(rng, 2, "Points", 400, 50)
	c.pointCount.SetDecimals(0)
	c.pointCount.SetMin(2)
	c.pointCount.SetMax(10000)

	// Values
	side.AddLabel(2, 0, "Values")
	c.values = ui.NewTable()
	c.values.SetColumnCount(2)
	c.values.SetColumnName(0, "x")
	c.values.SetColumnName(1, "f(x)")
	c.values.SetColumnWidth(0, 120)
	c.values.SetColumnWidth(1, 100)
	c.values.SetColumnHAlign(0, ui.HAlignRight)
	c.values.SetColumnHAlign(1, ui.HAlignRight)
	c.values.SetStretchLastColumn(true)
	menu := ui.NewContextMenu(c.values)
	menu.AddItem("Copy value", c.values.CopySelectionToClipboard)
	menu.AddItem("Copy table", c.copyTable)
	c.values.SetContextMenu(menu)
	side.AddWidget(3, 0, c.values)

	side.AddButton(4, 0, "Export CSV...", c.exportCSV)

	c.AddTimer(50, c.onAnimationTick)
	c.replot(true)
	return &c
}

// addParameter adds a row "name [slider] value" for a parameter in -5..5.
func (c *Plotter) addParameter(box *ui.GroupBox, row int, name string, value float64) (*ui.Slider, *ui.Label) {
	box.AddLabel(row, 0, name)
	slider := ui.NewSlider()
	slider.SetRange(-5, 5)
	slider.SetStep(0.01)
	slider.SetTickInterval(1)
	slider.SetValue(value)
	slider.SetXExpandable(true)
	box.AddWidget(row, 1, slider)
	label := box.AddLabel(row, 2, "")
	label.SetMinWidth(44)
	label.SetTextAlign(ui.HAlignRight)
	update := func() {
		label.SetText(fmt.Sprintf("%.2f", slider.Value()))
		c.replot(false)
	}
	slider.SetOnValueChanged(update)
	label.SetText(fmt.Sprintf("%.2f", value))
	return slider, label
}

func (c *Plotter) addNumber(box *ui.GroupBox, row int, name string, value, step float64) *ui.NumBox {
	box.AddLabel(row, 0, name)
	n := ui.NewNumBox()
	n.SetDecimals(2)
	n.SetStep(step)
	n.SetValue(value)
	n.SetOnValueChanged(func() { c.replot(false) })
	box.AddWidget(row, 1, n)
	return n
}

// replot parses the formula and redraws the chart and the values. An invalid
// formula only shows the error: the chart keeps the last good plot. accept
// is true for Enter and the Plot button, which also put the formula into the
// history of the combo box.
func (c *Plotter) replot(accept bool) {
	if c.formula == nil || c.pointCount == nil {
		return // still building the form
	}
	text := strings.TrimSpace(c.formula.Text())
	f, err := Parse(text)
	if err != nil {
		c.status.SetForegroundColor(ui.CurrentPalette().Error)
		c.status.SetText(err.Error())
		return
	}
	if accept {
		c.remember(text)
	}

	x0, x1 := c.xFrom.Value(), c.xTo.Value()
	vars := Vars{A: c.a.Value(), B: c.b.Value()}
	c.points = sample(f, vars, x0, x1, int(c.pointCount.Value()))
	c.chart.SetData(c.points)
	c.updateValues(f, vars, x0, x1)

	c.status.SetForegroundColor(nil)
	if len(c.points) == 0 {
		c.status.SetText("The function is not defined in the range")
	} else {
		c.status.SetText(fmt.Sprintf("%d points", len(c.points)))
	}
}

// remember puts a formula at the top of the drop-down list.
func (c *Plotter) remember(text string) {
	items := []string{text}
	for _, item := range c.formula.Items() {
		if item != text {
			items = append(items, item)
		}
	}
	if len(items) > 20 {
		items = items[:20]
	}
	c.formula.SetItems(items)
}

// sample evaluates f at n evenly spaced points. The points where f is not
// defined are skipped (the chart can't show gaps).
func sample(f *Formula, vars Vars, x0, x1 float64, n int) []ui.ChartPoint {
	n = max(n, 2)
	points := make([]ui.ChartPoint, 0, n)
	for i := range n {
		vars.X = x0 + (x1-x0)*float64(i)/float64(n-1)
		y := f.Eval(vars)
		if math.IsNaN(y) || math.IsInf(y, 0) {
			continue
		}
		points = append(points, ui.ChartPoint{X: vars.X, Y: y})
	}
	return points
}

func (c *Plotter) onAnimationTick() {
	if !c.animate.Checked() {
		return
	}
	b := c.b.Value() + c.animateSpeed
	if b > c.b.Max() || b < c.b.Min() {
		c.animateSpeed = -c.animateSpeed
		b = c.b.Value() + c.animateSpeed
	}
	c.b.SetValue(b)
	c.bValue.SetText(fmt.Sprintf("%.2f", c.b.Value()))
	c.replot(false)
}
