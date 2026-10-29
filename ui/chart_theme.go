package ui

import "image/color"

// ChartTheme is the set of colors a chart is drawn with. A chart takes the
// theme of the application's palette on every paint (CurrentChartTheme), so
// switching between ApplyDarkTheme and ApplyLightTheme only needs a
// repaint; ChartView.SetTheme fixes a theme of its own.
type ChartTheme struct {
	Background     color.RGBA // the widget
	PlotBackground color.RGBA // inside the axes
	Border         color.RGBA // around the widget
	Axis           color.RGBA // the axis lines and the ticks
	Text           color.RGBA // the labels
	Title          color.RGBA // the title and the axis titles
	Grid           color.RGBA
	GridMinor      color.RGBA

	// Palette is the colors of the series, in the order they are added: 10
	// colors told apart by most people with a color vision deficiency too
	Palette []color.RGBA

	// Up and Down color the candles and the bars of a waterfall
	Up, Down color.RGBA

	// The states of a value: normal, warning, critical (gauges, zones)
	Good, Warning, Critical color.RGBA

	// Sequential is the color scale of a heat map from low to high,
	// Diverging the one around a middle value
	Sequential []color.RGBA
	Diverging  []color.RGBA

	Marker        color.RGBA // annotations
	MarkerLabelBg color.RGBA

	Selection       color.RGBA
	SelectionDrag   color.RGBA
	SelectionEdge   color.RGBA
	SelectionHeader color.RGBA
	HeaderText      color.RGBA
	CloseHover      color.RGBA
	ZoomBand        color.RGBA

	Crosshair     color.RGBA
	TooltipBg     color.RGBA
	TooltipText   color.RGBA
	TooltipBorder color.RGBA

	// BadHatch hatches the spans of bad data
	BadHatch color.RGBA
}

// SeriesColor returns the color number i of the palette, going round it
func (t *ChartTheme) SeriesColor(i int) color.RGBA {
	if len(t.Palette) == 0 {
		return t.Text
	}
	i %= len(t.Palette)
	if i < 0 {
		i += len(t.Palette)
	}
	return t.Palette[i]
}

// chartDarkTheme and chartLightTheme are the colors picked for each theme;
// CurrentChartTheme adds the surfaces and lines of the palette
var chartDarkTheme = ChartTheme{
	Palette: []color.RGBA{
		ColorFromHex("#4fc3f7"), ColorFromHex("#f0883e"), ColorFromHex("#a371f7"),
		ColorFromHex("#3fb950"), ColorFromHex("#f778ba"), ColorFromHex("#d29922"),
		ColorFromHex("#39c5cf"), ColorFromHex("#ff7b72"), ColorFromHex("#a5d6ff"),
		ColorFromHex("#8b949e"),
	},
	Up:         ColorFromHex("#3fb950"),
	Down:       ColorFromHex("#f85149"),
	Good:       ColorFromHex("#3fb950"),
	Warning:    ColorFromHex("#d29922"),
	Critical:   ColorFromHex("#f85149"),
	Sequential: []color.RGBA{ColorFromHex("#0d1b2a"), ColorFromHex("#1b4965"), ColorFromHex("#2a9d8f"), ColorFromHex("#e9c46a"), ColorFromHex("#fdf0d5")},
	Diverging:  []color.RGBA{ColorFromHex("#4393c3"), ColorFromHex("#2b2b2b"), ColorFromHex("#d6604d")},
	Marker:     ColorFromHex("#e3b341"),
	CloseHover: chartAlpha("#ffffff", 60),
	ZoomBand:   chartAlpha("#ffffff", 40),
	BadHatch:   chartAlpha("#f85149", 110),
}

var chartLightTheme = ChartTheme{
	Palette: []color.RGBA{
		ColorFromHex("#0969da"), ColorFromHex("#d1570a"), ColorFromHex("#8250df"),
		ColorFromHex("#1a7f37"), ColorFromHex("#bf3989"), ColorFromHex("#9a6700"),
		ColorFromHex("#1b7c83"), ColorFromHex("#cf222e"), ColorFromHex("#54aeff"),
		ColorFromHex("#6e7781"),
	},
	Up:         ColorFromHex("#1a7f37"),
	Down:       ColorFromHex("#cf222e"),
	Good:       ColorFromHex("#1a7f37"),
	Warning:    ColorFromHex("#bf8700"),
	Critical:   ColorFromHex("#cf222e"),
	Sequential: []color.RGBA{ColorFromHex("#f7fbff"), ColorFromHex("#c6dbef"), ColorFromHex("#6baed6"), ColorFromHex("#2171b5"), ColorFromHex("#08306b")},
	Diverging:  []color.RGBA{ColorFromHex("#2166ac"), ColorFromHex("#f7f7f7"), ColorFromHex("#b2182b")},
	Marker:     ColorFromHex("#9a6700"),
	CloseHover: chartAlpha("#ffffff", 70),
	ZoomBand:   chartAlpha("#000000", 25),
	BadHatch:   chartAlpha("#cf222e", 100),
}

// CurrentChartTheme returns the chart theme of the current palette: the
// surfaces, lines and selection follow the application's palette, so a
// chart matches the other widgets; the series colors are picked for the
// light and the dark theme each.
func CurrentChartTheme() *ChartTheme {
	th := chartLightTheme
	if IsDarkTheme {
		th = chartDarkTheme
	}
	fillChartTheme(&th, CurrentPalette())
	return &th
}

// fillChartTheme sets the colors of the theme that follow the palette
func fillChartTheme(th *ChartTheme, p *ThemePalette) {
	th.Background = p.Base
	th.PlotBackground = p.Base
	th.Border = p.Border
	th.Axis = MixColors(p.Base, p.Text, 0.45)
	th.Text = MixColors(p.Base, p.Text, 0.65)
	th.Title = p.Text
	th.Grid = MixColors(p.Base, p.Text, 0.08)
	th.GridMinor = MixColors(p.Base, p.Text, 0.04)
	th.MarkerLabelBg = withAlpha(p.Base, 215)
	th.Selection = withAlpha(p.Highlight, 40)
	th.SelectionDrag = withAlpha(p.Highlight, 60)
	th.SelectionEdge = withAlpha(p.Highlight, 210)
	th.SelectionHeader = withAlpha(p.Highlight, 190)
	th.HeaderText = p.HighlightedText
	th.Crosshair = MixColors(p.Base, p.Text, 0.4)
	th.TooltipBg = withAlpha(p.ToolTipBase, 240)
	th.TooltipText = p.ToolTipText
	th.TooltipBorder = p.Border
}

// chartAlpha returns the color of the hex with the alpha, straight (not
// premultiplied), as the canvas blends it
func chartAlpha(hex string, a uint8) color.RGBA {
	c := ColorFromHex(hex)
	c.A = a
	return c
}

// ColorScale returns the color at t (0..1) of the scale of colors, mixed
// between the two around it: for heat maps and colored scatter points
func ColorScale(colors []color.RGBA, t float64) color.RGBA {
	switch len(colors) {
	case 0:
		return color.RGBA{}
	case 1:
		return colors[0]
	}
	if t != t || t <= 0 { // NaN too
		return colors[0]
	}
	if t >= 1 {
		return colors[len(colors)-1]
	}
	f := t * float64(len(colors)-1)
	i := int(f)
	return mixRGBA(colors[i], colors[i+1], f-float64(i))
}
