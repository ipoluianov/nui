package ui

import "image/color"

// timeChartTheme holds every color the TimeChart draws with. The chart gets
// it on each paint (see currentTimeChartTheme), so switching the application
// theme only needs a repaint.
type timeChartTheme struct {
	background color.Color
	border     color.Color
	axis       color.Color
	text       color.Color
	grid       color.Color

	// Automatic series colors, in the order series are added.
	palette []color.Color

	candleUp   color.Color
	candleDown color.Color

	marker        color.Color
	markerLabelBg color.Color

	selection       color.Color
	selectionDrag   color.Color
	selectionEdge   color.Color
	selectionHeader color.Color
	headerText      color.Color
	closeHover      color.Color
	zoomBand        color.Color

	// Hatch over the spans of bad quality
	badHatch color.Color
}

var timeChartDarkTheme = timeChartTheme{
	palette: []color.Color{
		ColorFromHex("#4fc3f7"),
		ColorFromHex("#f0883e"),
		ColorFromHex("#a371f7"),
		ColorFromHex("#3fb950"),
		ColorFromHex("#f778ba"),
		ColorFromHex("#d29922"),
	},
	candleUp:   ColorFromHex("#3fb950"),
	candleDown: ColorFromHex("#f85149"),
	marker:     ColorFromHex("#e3b341"),
	closeHover: timeChartAlpha("#ffffff", 60),
	zoomBand:   timeChartAlpha("#ffffff", 40),
	badHatch:   timeChartAlpha("#f85149", 110),
}

var timeChartLightTheme = timeChartTheme{
	palette: []color.Color{
		ColorFromHex("#0969da"),
		ColorFromHex("#d1570a"),
		ColorFromHex("#8250df"),
		ColorFromHex("#1a7f37"),
		ColorFromHex("#bf3989"),
		ColorFromHex("#9a6700"),
	},
	candleUp:   ColorFromHex("#1a7f37"),
	candleDown: ColorFromHex("#cf222e"),
	marker:     ColorFromHex("#9a6700"),
	closeHover: timeChartAlpha("#ffffff", 70),
	zoomBand:   timeChartAlpha("#000000", 25),
	badHatch:   timeChartAlpha("#cf222e", 100),
}

// currentTimeChartTheme returns the theme for the current palette: the
// surfaces, lines and selection follow the application's palette, so the
// chart matches the other widgets; the series colors are picked for each theme.
func currentTimeChartTheme() *timeChartTheme {
	th := timeChartLightTheme
	if IsDarkTheme {
		th = timeChartDarkTheme
	}

	p := CurrentPalette()
	th.background = p.Base
	th.border = p.Border
	th.axis = MixColors(p.Base, p.Text, 0.45)
	th.text = MixColors(p.Base, p.Text, 0.65)
	th.grid = MixColors(p.Base, p.Text, 0.08)
	th.markerLabelBg = withAlpha(p.Base, 215)
	th.selection = withAlpha(p.Highlight, 40)
	th.selectionDrag = withAlpha(p.Highlight, 60)
	th.selectionEdge = withAlpha(p.Highlight, 210)
	th.selectionHeader = withAlpha(p.Highlight, 190)
	th.headerText = p.HighlightedText
	return &th
}

// timeChartAlpha returns a translucent color for Canvas.MixPixel (used by
// FillRect and axis-aligned DrawLine), which blends the RGB channels as
// given: they must be straight, not premultiplied as color.RGBA normally is.
// A color.NRGBA would be premultiplied by its RGBA() method and come out
// darker than intended.
func timeChartAlpha(hex string, a uint8) color.RGBA {
	c := ColorFromHex(hex)
	c.A = a
	return c
}
