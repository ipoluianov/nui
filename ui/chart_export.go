package ui

import (
	"encoding/csv"
	"image"
	"image/png"
	"io"
	"math"
	"strconv"
)

// ChartThemeOf returns the chart theme of the palette: the light one for
// LightPalette, the dark one for DarkPalette. Printing and exporting use the
// light theme whatever the application's theme is.
func ChartThemeOf(p *ThemePalette, dark bool) *ChartTheme {
	th := chartLightTheme
	if dark {
		th = chartDarkTheme
	}
	fillChartTheme(&th, p)
	return &th
}

// Render draws the chart into an image of the logical size at the scale
// (2 for a sharp picture), without the crosshair and the tooltip of the
// pointer: for a file, the clipboard, a report. th is the theme of the
// picture; nil - the chart's own.
func (c *ChartView) Render(width, height int, scale float64, th *ChartTheme) *image.RGBA {
	if scale <= 0 {
		scale = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, int(math.Ceil(float64(width)*scale)), int(math.Ceil(float64(height)*scale))))
	c.renderTo(width, height, th, func() *Canvas { return NewCanvasScaled(img, scale) })
	return img
}

// renderTo paints the chart at the size on the canvases made by newCanvas:
// twice, the first time to lay it out at that size
func (c *ChartView) renderTo(width, height int, th *ChartTheme, newCanvas func() *Canvas) {
	savedTheme, savedMouse, savedW := c.fixedTheme, c.mouseIn, c.lastPlotW
	savedHits := c.hits
	if th != nil {
		c.fixedTheme = th
	}
	c.mouseIn = false
	c.renderW, c.renderH = width, height
	for pass := 0; pass < 2; pass++ {
		cnv := newCanvas()
		cnv.SetDirectTranslateAndClip(0, 0, width, height)
		c.paintFunc(cnv)
	}
	c.renderW, c.renderH = 0, 0
	c.fixedTheme, c.mouseIn, c.lastPlotW, c.hits = savedTheme, savedMouse, savedW, savedHits
	// The layout of the picture is not the widget's: the next paint lays it out again
	c.laidOut = false
	c.Update()
}

// size returns the size the chart is painted at: the widget's, or the
// picture's of Render
func (c *ChartView) size() (float64, float64) {
	if c.renderW > 0 {
		return float64(c.renderW), float64(c.renderH)
	}
	return float64(c.Width()), float64(c.Height())
}

// WritePNG writes the chart as a PNG picture of the logical size at the scale
func (c *ChartView) WritePNG(w io.Writer, width, height int, scale float64, th *ChartTheme) error {
	return png.Encode(w, c.Render(width, height, scale, th))
}

// WriteSVG writes the chart as an SVG picture of the logical size: lines,
// shapes and texts, sharp at any size, for a document or a web page
func (c *ChartView) WriteSVG(w io.Writer, width, height int, th *ChartTheme) error {
	rec := NewSVGRecorder(float64(width), float64(height))
	scratch := image.NewRGBA(image.Rect(0, 0, width, height))
	pass := 0
	c.renderTo(width, height, th, func() *Canvas {
		cnv := NewCanvas(scratch)
		// The second pass, laid out, is the one recorded
		if pass == 1 {
			cnv.SetRecorder(rec)
		}
		pass++
		return cnv
	})
	_, err := rec.WriteTo(w)
	return err
}

// CopyImage puts the picture of the chart, as it is on the screen, at twice
// its size into the clipboard
func (c *ChartView) CopyImage() error {
	return ClipboardSetImage(c.Render(c.Width(), c.Height(), 2, nil))
}

// WriteCSV writes the data of the visible series as CSV: a row per sample
// (series, x, value; open, high, low, close for candles), per category of
// bars, per slice of a pie, per point of a scatter. With all it writes all
// the data of the series' sources, not only the visible range.
func (c *ChartView) WriteCSV(w io.Writer, all bool) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"series", "x", "value", "open", "high", "low", "close"}); err != nil {
		return err
	}
	num := func(v float64) string {
		if !chartFinite(v) {
			return ""
		}
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
	for _, p := range c.panes {
		for _, s := range p.series {
			if !s.Visible() {
				continue
			}
			xa, _ := p.seriesAxes(s)
			xText := func(x float64) string {
				if _, ok := xa.scale.(*TimeScale); ok {
					return xa.scale.Format(x)
				}
				return num(x)
			}
			samples := func(src ChartSource, drawn []ChartSample) []ChartSample {
				if !all || src == nil {
					return drawn
				}
				if e, ok := src.(ChartSourceExtent); ok {
					if lo, hi, ok := e.Extent(); ok {
						return src.GetData(lo, hi, 0)
					}
				}
				return drawn
			}
			var rows [][]string
			switch ss := s.(type) {
			case *LineSeries:
				for _, sm := range samples(ss.source, ss.samples) {
					if sm.HasValue() {
						rows = append(rows, []string{ss.name, xText(sm.X), num(sm.Last), "", "", "", ""})
					}
				}
			case *CandleSeries:
				for _, sm := range samples(ss.source, ss.samples) {
					if sm.HasValue() {
						rows = append(rows, []string{ss.name, xText(sm.X), num(sm.Last), num(sm.First), num(sm.High), num(sm.Low), num(sm.Last)})
					}
				}
			case *BarSeries:
				if ss.byCategory() {
					cs := ss.categories()
					for i, v := range ss.values {
						if chartFinite(v) && cs != nil {
							rows = append(rows, []string{ss.name, cs.name(i), num(v), "", "", "", ""})
						}
					}
				} else {
					for _, sm := range samples(ss.source, ss.samples) {
						if sm.HasValue() {
							rows = append(rows, []string{ss.name, xText(sm.X), num(sm.Last), "", "", "", ""})
						}
					}
				}
			case *PieSeries:
				for _, sl := range ss.slices {
					rows = append(rows, []string{ss.name, sl.name, num(sl.value), "", "", "", ""})
				}
			case *ScatterSeries:
				for _, pt := range ss.points {
					rows = append(rows, []string{ss.name, num(pt.X), num(pt.Y), "", "", "", ""})
				}
			}
			if err := cw.WriteAll(rows); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

// PrintJob returns a job that prints the chart on a page, in the light
// theme, as wide as the page within margins of 15 mm and as tall as the
// chart's shape on the screen: start it with Form.Print
func (c *ChartView) PrintJob(title string) *PrintJob {
	return &PrintJob{
		Title: title,
		Pages: 1,
		DrawPage: func(p *PrintPage) {
			margin := p.MM(15)
			w := p.Width - 2*margin
			aspect := 0.6
			if c.Width() > 0 && c.Height() > 0 {
				aspect = float64(c.Height()) / float64(c.Width())
			}
			h := min(int(float64(w)*aspect), p.Height-2*margin)
			// A logical pixel of the chart is a pixel of a 96 dpi screen
			scale := p.DPI / 96
			img := c.Render(int(float64(w)/scale), int(float64(h)/scale), scale, ChartThemeOf(&LightPalette, false))
			p.Canvas.DrawImage(margin, margin, img)
		},
	}
}
