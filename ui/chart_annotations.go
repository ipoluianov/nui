package ui

import (
	"image/color"
	"math"
	"sort"
)

// ChartAnnotationKind is the kind of an annotation
type ChartAnnotationKind int

const (
	// ChartAnnotationHLine is a line across the plot at a value of the
	// vertical axis: a threshold, a set point
	ChartAnnotationHLine ChartAnnotationKind = iota
	// ChartAnnotationVLine is a line down the plot at a value of the
	// horizontal axis: an event
	ChartAnnotationVLine
	// ChartAnnotationXZone is a band between two values of the horizontal
	// axis: a shift, the night, a maintenance window
	ChartAnnotationXZone
	// ChartAnnotationYZone is a band between two values of the vertical
	// axis: the normal range, a tolerance
	ChartAnnotationYZone
	// ChartAnnotationLabel is a text pinned to a point, with a mark
	// pointing at it: a peak, a detected event
	ChartAnnotationLabel
)

// ChartAnnotation is a line, a zone or a label over the series of a pane
type ChartAnnotation struct {
	anns   *ChartAnnotations
	kind   ChartAnnotationKind
	v1, v2 float64 // the values; X and Y for a label
	text   string
	col    *color.RGBA
	dash   []float64
	axis   string // the vertical axis; "" - "y"
	hidden bool
}

// SetColor sets the color of the annotation (a zone is filled with it
// translucent); without it the theme's color is used
func (a *ChartAnnotation) SetColor(col color.RGBA) *ChartAnnotation {
	a.col = &col
	a.anns.update()
	return a
}

// SetDash makes the line dashed
func (a *ChartAnnotation) SetDash(pattern ...float64) *ChartAnnotation {
	a.dash = pattern
	a.anns.update()
	return a
}

// SetText sets the label of the annotation
func (a *ChartAnnotation) SetText(text string) *ChartAnnotation {
	a.text = text
	a.anns.update()
	return a
}

// SetAxis puts the annotation on the vertical axis of the name
func (a *ChartAnnotation) SetAxis(name string) *ChartAnnotation {
	a.axis = name
	a.anns.update()
	return a
}

// SetValues moves the annotation: the value of a line, the ends of a zone,
// the point of a label
func (a *ChartAnnotation) SetValues(v1, v2 float64) *ChartAnnotation {
	a.v1, a.v2 = v1, v2
	a.anns.update()
	return a
}

// Values returns the values of the annotation
func (a *ChartAnnotation) Values() (v1, v2 float64) { return a.v1, a.v2 }

// SetVisible shows or hides the annotation
func (a *ChartAnnotation) SetVisible(visible bool) *ChartAnnotation {
	a.hidden = !visible
	a.anns.update()
	return a
}

// Remove takes the annotation away
func (a *ChartAnnotation) Remove() {
	a.anns.remove(a)
}

// annotationRange widens the range of the data on the vertical axis to the
// lines and the zones of the pane's annotations on it: a threshold or a
// control limit stays in view
func (p *ChartPane) annotationRange(a *ChartAxis, lo, hi float64) (float64, float64) {
	if p.annotations == nil {
		return lo, hi
	}
	for _, an := range p.annotations.items {
		if an.hidden || (an.kind != ChartAnnotationHLine && an.kind != ChartAnnotationYZone) {
			continue
		}
		axis := p.axes[0]
		if an.axis != "" {
			if ax := p.Axis(an.axis); ax != nil {
				axis = ax
			}
		}
		if axis != a {
			continue
		}
		lo, hi = math.Min(lo, an.v1), math.Max(hi, an.v1)
		if an.kind == ChartAnnotationYZone {
			lo, hi = math.Min(lo, an.v2), math.Max(hi, an.v2)
		}
	}
	return lo, hi
}

// ChartAnnotations is the layer of lines, zones and labels of a pane
// (ChartPane.Annotations), drawn over its series
type ChartAnnotations struct {
	pane  *ChartPane
	items []*ChartAnnotation
	added bool
}

func (as *ChartAnnotations) update() {
	if as.pane != nil {
		as.pane.view.Update()
	}
}

func (as *ChartAnnotations) add(a *ChartAnnotation) *ChartAnnotation {
	a.anns = as
	as.items = append(as.items, a)
	if !as.added {
		as.added = true
		as.pane.AddLayer(as, true)
	}
	as.update()
	return a
}

func (as *ChartAnnotations) remove(a *ChartAnnotation) {
	for i, o := range as.items {
		if o == a {
			as.items = append(as.items[:i:i], as.items[i+1:]...)
			as.update()
			return
		}
	}
}

// AddHLine adds a line across the plot at the value of the vertical axis
func (as *ChartAnnotations) AddHLine(value float64, text string) *ChartAnnotation {
	return as.add(&ChartAnnotation{kind: ChartAnnotationHLine, v1: value, text: text})
}

// AddVLine adds a line down the plot at the value of the horizontal axis
func (as *ChartAnnotations) AddVLine(x float64, text string) *ChartAnnotation {
	return as.add(&ChartAnnotation{kind: ChartAnnotationVLine, v1: x, text: text})
}

// AddXZone adds a band between the values of the horizontal axis
func (as *ChartAnnotations) AddXZone(x1, x2 float64, text string) *ChartAnnotation {
	return as.add(&ChartAnnotation{kind: ChartAnnotationXZone, v1: x1, v2: x2, text: text})
}

// AddYZone adds a band between the values of the vertical axis
func (as *ChartAnnotations) AddYZone(y1, y2 float64, text string) *ChartAnnotation {
	return as.add(&ChartAnnotation{kind: ChartAnnotationYZone, v1: y1, v2: y2, text: text})
}

// AddLabel adds a text pinned to the point (x, y), with a small triangle
// pointing at it. Texts that would overlap one drawn before are skipped, so
// a zoomed-out chart stays readable.
func (as *ChartAnnotations) AddLabel(x, y float64, text string) *ChartAnnotation {
	return as.add(&ChartAnnotation{kind: ChartAnnotationLabel, v1: x, v2: y, text: text})
}

// Items returns the annotations
func (as *ChartAnnotations) Items() []*ChartAnnotation { return as.items }

// Clear removes all the annotations
func (as *ChartAnnotations) Clear() {
	as.items = nil
	as.update()
}

// RemoveKind removes the annotations of the kind: all the labels, say,
// before adding the new peaks
func (as *ChartAnnotations) RemoveKind(kind ChartAnnotationKind) {
	keep := as.items[:0]
	for _, a := range as.items {
		if a.kind != kind {
			keep = append(keep, a)
		}
	}
	as.items = keep
	as.update()
}

// DrawLayer implements ChartLayer
func (as *ChartAnnotations) DrawLayer(cnv *Canvas, v *ChartViewport) {
	th := v.Theme
	cnv.Save()
	defer cnv.Restore()
	cnv.SetFontSize(v.FontSize * 0.85)
	_, lineH := cnv.MeasureTextRotated("0", 0)
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)

	yOf := func(a *ChartAnnotation, val float64) float64 {
		y := v.YAxis
		if a.axis != "" {
			if ax := v.Pane.Axis(a.axis); ax != nil {
				y = ax
			}
		}
		return y.ToPixel(val)
	}
	colOf := func(a *ChartAnnotation, def color.RGBA) color.RGBA {
		if a.col != nil {
			return *a.col
		}
		return def
	}
	label := func(text string, x, y float64, h HAlign, vAlign VAlign, col color.RGBA) {
		if text == "" {
			return
		}
		cnv.SetHAlign(h)
		cnv.SetVAlign(vAlign)
		cnv.SetColor(col)
		cnv.DrawTextRotated(x, y, 0, text)
	}

	// The zones first, the lines over them, the labels last
	for _, a := range as.items {
		if a.hidden {
			continue
		}
		col := colOf(a, th.Marker)
		fill := col
		fill.A = uint8(float64(col.A) * 0.15)
		switch a.kind {
		case ChartAnnotationXZone:
			x1, x2 := v.PX(a.v1), v.PX(a.v2)
			x1, x2 = math.Min(x1, x2), math.Max(x1, x2)
			p.Reset()
			p.Rect(x1, v.Y, x2-x1, v.H)
			cnv.FillPath(p, fill)
			label(a.text, x1+4, v.Y+3, HAlignLeft, VAlignTop, col)
		case ChartAnnotationYZone:
			y1, y2 := yOf(a, a.v1), yOf(a, a.v2)
			y1, y2 = math.Min(y1, y2), math.Max(y1, y2)
			p.Reset()
			p.Rect(v.X, y1, v.W, y2-y1)
			cnv.FillPath(p, fill)
			label(a.text, v.X+v.W-4, y1+2, HAlignRight, VAlignTop, col)
		}
	}
	for _, a := range as.items {
		if a.hidden {
			continue
		}
		col := colOf(a, th.Marker)
		switch a.kind {
		case ChartAnnotationHLine:
			y := chartCrisp(yOf(a, a.v1))
			p.Reset()
			p.MoveTo(v.X, y)
			p.LineTo(v.X+v.W, y)
			cnv.StrokePath(p, StrokeStyle{Width: 1, Dash: a.dash}, col)
			label(a.text, v.X+v.W-4, y-2, HAlignRight, VAlignBottom, col)
		case ChartAnnotationVLine:
			x := chartCrisp(v.PX(a.v1))
			p.Reset()
			p.MoveTo(x, v.Y)
			p.LineTo(x, v.Y+v.H)
			cnv.StrokePath(p, StrokeStyle{Width: 1, Dash: a.dash}, col)
			label(a.text, x+3, v.Y+3, HAlignLeft, VAlignTop, col)
		}
	}
	as.drawLabels(cnv, v, lineH, yOf, colOf)
}

// drawLabels draws the labels pinned to points, in the order of x; one
// that would overlap a label drawn before is skipped
func (as *ChartAnnotations) drawLabels(cnv *Canvas, v *ChartViewport, lineH float64, yOf func(*ChartAnnotation, float64) float64, colOf func(*ChartAnnotation, color.RGBA) color.RGBA) {
	var labels []*ChartAnnotation
	for _, a := range as.items {
		if !a.hidden && a.kind == ChartAnnotationLabel {
			labels = append(labels, a)
		}
	}
	sort.SliceStable(labels, func(i, j int) bool { return labels[i].v1 < labels[j].v1 })
	type box struct{ x1, y1, x2, y2 float64 }
	var taken []box
	const tri = 5.0
	for _, a := range labels {
		x, y := v.PX(a.v1), yOf(a, a.v2)
		if x < v.X || x > v.X+v.W || y < v.Y || y > v.Y+v.H {
			continue
		}
		col := colOf(a, v.Theme.Marker)
		tw := chartTextWidth(cnv, a.text)
		// Above the point, or below it at the top of the plot
		above := y-tri-lineH-4 >= v.Y
		dir := -1.0
		if !above {
			dir = 1
		}
		bx := math.Max(v.X, math.Min(x-tw/2-3, v.X+v.W-tw-6))
		by := y + dir*(tri+2)
		if above {
			by -= lineH + 2
		}
		b := box{bx, by, bx + tw + 6, by + lineH + 2}
		overlaps := false
		for _, t := range taken {
			if b.x1 < t.x2 && b.x2 > t.x1 && b.y1 < t.y2 && b.y2 > t.y1 {
				overlaps = true
				break
			}
		}
		cnv.FillPolygon([]PointF{{x, y}, {x - tri, y + dir*tri}, {x + tri, y + dir*tri}}, col)
		if overlaps || a.text == "" {
			continue
		}
		taken = append(taken, b)
		bg := pathPool.Get().(*Path)
		bg.Reset()
		bg.RoundedRect(b.x1, b.y1, b.x2-b.x1, b.y2-b.y1, 3)
		cnv.FillPath(bg, v.Theme.MarkerLabelBg)
		pathPool.Put(bg)
		cnv.SetHAlign(HAlignLeft)
		cnv.SetVAlign(VAlignTop)
		cnv.SetColor(col)
		cnv.DrawTextRotated(b.x1+3, b.y1+1, 0, a.text)
	}
}
