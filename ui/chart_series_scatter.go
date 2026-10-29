package ui

import (
	"image"
	"image/color"
	"math"
	"sort"
	"strconv"
)

// ScatterPoint is a point of a ScatterSeries: where it is, and optionally
// its size (a bubble chart), a value its color is taken by and a label
type ScatterPoint struct {
	X, Y float64
	// Size of the bubble; 0 - the series' marker size
	Size float64
	// Value picks the color on the series' color scale; NaN - the series'
	// color (see SetColorScale)
	Value float64
	Label string
}

// ScatterSeries draws points as markers: a scatter chart, a bubble chart
// (SetBubbleSizes), points colored by a value (SetColorScale). With more
// points than the density threshold (20000) it draws how many points fall
// on each pixel instead, so a million points stay quick and readable.
//
//	s := chart.Pane(0).AddScatter("Measurements")
//	s.Add(1.2, 3.4).Add(2.5, 1.1)
type ScatterSeries struct {
	ChartSeriesBase
	points []ScatterPoint

	marker MarkerShape
	size   float64
	// The bubbles: the sizes of the points mapped to diameters
	bubble               bool
	bubbleMin, bubbleMax float64
	// The colors by value
	scale      []color.RGBA
	vmin, vmax float64
	density    int
	outline    bool
	// stems: a line from the base value up to each point
	stems    bool
	stemBase float64

	// The points drawn on the last paint, in pixels, for Nearest: a grid
	// of cells of the indexes in them
	drawn  []int
	px, py []float64
	// The extent of all the points, found once for them; the buffers of
	// the density
	extent      [4]float64
	extentValid bool
	counts      []uint32
	densityImg  *image.RGBA
	grid        map[[2]int][]int
	dense       bool
}

// NewScatterSeries makes points of the name
func NewScatterSeries(name string) *ScatterSeries {
	s := &ScatterSeries{marker: MarkerCircle, size: 6, density: 20000, bubbleMin: 4, bubbleMax: 40}
	s.name = name
	return s
}

// AddScatter adds points of the name to the pane
func (p *ChartPane) AddScatter(name string) *ScatterSeries {
	s := NewScatterSeries(name)
	p.AddSeries(s)
	return s
}

// Add adds the point (x, y)
func (s *ScatterSeries) Add(x, y float64) *ScatterSeries {
	s.extentValid = false
	s.points = append(s.points, ScatterPoint{X: x, Y: y, Value: math.NaN()})
	s.update()
	return s
}

// AddPoint adds a point with its size, value and label
func (s *ScatterSeries) AddPoint(p ScatterPoint) *ScatterSeries {
	s.extentValid = false
	s.points = append(s.points, p)
	s.update()
	return s
}

// SetPoints replaces the points
func (s *ScatterSeries) SetPoints(points []ScatterPoint) *ScatterSeries {
	s.extentValid = false
	s.points = append(s.points[:0], points...)
	s.update()
	return s
}

// SetXY replaces the points with the pairs of xs and ys
func (s *ScatterSeries) SetXY(xs, ys []float64) *ScatterSeries {
	s.extentValid = false
	n := min(len(xs), len(ys))
	s.points = s.points[:0]
	for i := 0; i < n; i++ {
		s.points = append(s.points, ScatterPoint{X: xs[i], Y: ys[i], Value: math.NaN()})
	}
	s.update()
	return s
}

// Points returns the points
func (s *ScatterSeries) Points() []ScatterPoint { return s.points }

// Clear removes the points
func (s *ScatterSeries) Clear() *ScatterSeries {
	s.extentValid = false
	s.points = s.points[:0]
	s.update()
	return s
}

// SetMarker sets the shape and the size of the markers
func (s *ScatterSeries) SetMarker(shape MarkerShape, size float64) *ScatterSeries {
	s.marker = shape
	if size > 0 {
		s.size = size
	}
	s.update()
	return s
}

// SetOutline draws the markers outlined with the background: they stay
// apart where they overlap (on by default for bubbles)
func (s *ScatterSeries) SetOutline(on bool) *ScatterSeries {
	s.outline = on
	s.update()
	return s
}

// SetBubbleSizes makes a bubble chart: the Size of the points is shown as
// the area of the markers, from min to max pixels across
func (s *ScatterSeries) SetBubbleSizes(min, max float64) *ScatterSeries {
	s.bubble = true
	s.outline = true
	if min > 0 {
		s.bubbleMin = min
	}
	if max > min {
		s.bubbleMax = max
	}
	s.update()
	return s
}

// SetColorScale colors the points by their Value on the scale of colors
// (ChartTheme.Sequential, say) from vmin to vmax; vmin == vmax - the range
// of the values
func (s *ScatterSeries) SetColorScale(colors []color.RGBA, vmin, vmax float64) *ScatterSeries {
	s.scale = colors
	s.vmin, s.vmax = vmin, vmax
	s.update()
	return s
}

// SetStems draws a line from the base value up (or down) to each point: a
// stem plot of discrete samples of a signal
func (s *ScatterSeries) SetStems(on bool, base float64) *ScatterSeries {
	s.stems, s.stemBase = on, base
	s.update()
	return s
}

// SetDensityThreshold sets how many points are drawn as markers; with more
// the series draws their density (0 - always markers)
func (s *ScatterSeries) SetDensityThreshold(n int) *ScatterSeries {
	s.density = n
	s.update()
	return s
}

// Range implements ChartSeries
func (s *ScatterSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if !s.extentValid {
		e := [4]float64{math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)}
		for _, p := range s.points {
			if chartFinite(p.X) && chartFinite(p.Y) {
				e[0], e[1] = math.Min(e[0], p.X), math.Max(e[1], p.X)
				e[2], e[3] = math.Min(e[2], p.Y), math.Max(e[3], p.Y)
			}
		}
		s.extent, s.extentValid = e, true
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	if s.stems && !axis.Horizontal() {
		lo, hi = s.stemBase, s.stemBase
	}
	xmin, xmax := math.Inf(-1), math.Inf(1)
	if !axis.Horizontal() && v != nil {
		xmin, xmax = v.XRange()
	}
	// All the points in view: the extent found before
	if axis.Horizontal() {
		return math.Min(lo, s.extent[0]), math.Max(hi, s.extent[1]), s.extent[0] <= s.extent[1]
	}
	if xmin <= s.extent[0] && xmax >= s.extent[1] {
		return math.Min(lo, s.extent[2]), math.Max(hi, s.extent[3]), s.extent[2] <= s.extent[3]
	}
	for _, p := range s.points {
		if !chartFinite(p.X) || !chartFinite(p.Y) {
			continue
		}
		if axis.Horizontal() {
			lo, hi = math.Min(lo, p.X), math.Max(hi, p.X)
		} else if p.X >= xmin && p.X <= xmax {
			lo, hi = math.Min(lo, p.Y), math.Max(hi, p.Y)
		}
	}
	return lo, hi, lo <= hi
}

// Prepare implements ChartSeries
func (s *ScatterSeries) Prepare(v *ChartViewport) {}

// chartXMargin asks the horizontal axis for a margin: the markers at the
// ends of the data are not cut at the edges of the plot
func (s *ScatterSeries) chartXMargin() float64 { return 0.04 }

// valueRange returns the range of the values the colors are spread over
func (s *ScatterSeries) valueRange() (float64, float64) {
	if s.vmax > s.vmin {
		return s.vmin, s.vmax
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range s.points {
		if chartFinite(p.Value) {
			lo, hi = math.Min(lo, p.Value), math.Max(hi, p.Value)
		}
	}
	return lo, hi
}

// sizeRange returns the range of the sizes of the bubbles
func (s *ScatterSeries) sizeRange() (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range s.points {
		if p.Size > 0 && chartFinite(p.Size) {
			lo, hi = math.Min(lo, p.Size), math.Max(hi, p.Size)
		}
	}
	return lo, hi
}

// diameter returns the width of the marker of the point
func (s *ScatterSeries) diameter(p ScatterPoint, lo, hi float64) float64 {
	if !s.bubble || !(p.Size > 0) {
		return s.size
	}
	if hi <= lo {
		return (s.bubbleMin + s.bubbleMax) / 2
	}
	// The area grows with the size
	t := math.Sqrt((p.Size - lo) / (hi - lo))
	return s.bubbleMin + t*(s.bubbleMax-s.bubbleMin)
}

// Draw implements ChartSeries
func (s *ScatterSeries) Draw(cnv *Canvas, v *ChartViewport) {
	col := s.Color(v.Theme)
	s.drawn, s.px, s.py = s.drawn[:0], s.px[:0], s.py[:0]
	vlo, vhi, slo, shi := 0.0, 0.0, 0.0, 0.0
	if s.scale != nil {
		vlo, vhi = s.valueRange()
	}
	if s.bubble {
		slo, shi = s.sizeRange()
	}
	for i, p := range s.points {
		x, y := v.PX(p.X), v.PY(p.Y)
		if !chartFinite(x) || !chartFinite(y) || x < v.X-40 || x > v.X+v.W+40 || y < v.Y-40 || y > v.Y+v.H+40 {
			continue
		}
		s.drawn = append(s.drawn, i)
		s.px = append(s.px, x)
		s.py = append(s.py, y)
	}
	s.dense = s.density > 0 && len(s.drawn) > s.density
	s.grid = nil
	if s.dense {
		s.drawDensity(cnv, v, col)
		return
	}
	// The grid of the points for Nearest: only of the points drawn as markers
	s.grid = make(map[[2]int][]int, len(s.drawn)/4+1)
	for k := range s.drawn {
		cell := [2]int{int(s.px[k]) / 16, int(s.py[k]) / 16}
		s.grid[cell] = append(s.grid[cell], k)
	}
	// Big bubbles first, the small ones over them
	order := make([]int, len(s.drawn))
	for k := range order {
		order[k] = k
	}
	if s.bubble {
		sort.SliceStable(order, func(a, b int) bool {
			return s.diameter(s.points[s.drawn[order[a]]], slo, shi) > s.diameter(s.points[s.drawn[order[b]]], slo, shi)
		})
	}
	var outline color.Color
	if s.outline {
		outline = v.Theme.PlotBackground
	}
	if s.stems {
		p := pathPool.Get().(*Path)
		p.Reset()
		base := v.PY(s.stemBase)
		for k := range s.drawn {
			x := chartCrisp(s.px[k])
			p.MoveTo(x, base)
			p.LineTo(x, s.py[k])
		}
		cnv.StrokePath(p, StrokeStyle{Width: 1}, col)
		pathPool.Put(p)
	}
	for _, k := range order {
		p := s.points[s.drawn[k]]
		c := col
		if s.scale != nil && chartFinite(p.Value) && vhi > vlo {
			c = ColorScale(s.scale, (p.Value-vlo)/(vhi-vlo))
		}
		fill := c
		if s.bubble {
			fill.A = uint8(float64(c.A) * 0.75)
		}
		cnv.DrawMarker(s.marker, s.px[k], s.py[k], s.diameter(p, slo, shi), fill, outline)
	}
}

// drawDensity draws how many points fall on each pixel of the plot: the
// color of the series, more opaque where there are more
func (s *ScatterSeries) drawDensity(cnv *Canvas, v *ChartViewport, col color.RGBA) {
	scale := cnv.Scale()
	w, h := int(math.Ceil(v.W*scale)), int(math.Ceil(v.H*scale))
	if w <= 0 || h <= 0 {
		return
	}
	if cap(s.counts) < w*h {
		s.counts = make([]uint32, w*h)
	}
	counts := s.counts[:w*h]
	clear(counts)
	maxCount := uint32(0)
	for k := range s.drawn {
		ix := int((s.px[k] - v.X) * scale)
		iy := int((s.py[k] - v.Y) * scale)
		if ix < 0 || iy < 0 || ix >= w || iy >= h {
			continue
		}
		counts[iy*w+ix]++
		if counts[iy*w+ix] > maxCount {
			maxCount = counts[iy*w+ix]
		}
	}
	if maxCount == 0 {
		return
	}
	img := s.densityImg
	if img == nil || img.Rect.Dx() != w || img.Rect.Dy() != h {
		img = image.NewRGBA(image.Rect(0, 0, w, h))
		s.densityImg = img
	} else {
		clear(img.Pix)
	}
	logMax := math.Log1p(float64(maxCount))
	// The color of each count: a single point is still seen, the densest
	// pixels are opaque; a table for the usual counts
	shade := func(n uint32) [4]uint8 {
		a := 0.25 + 0.75*math.Log1p(float64(n))/logMax
		pa := uint32(float64(col.A) * a)
		return [4]uint8{uint8(uint32(col.R) * pa / 255), uint8(uint32(col.G) * pa / 255), uint8(uint32(col.B) * pa / 255), uint8(pa)}
	}
	var table [256][4]uint8
	for n := 1; n < len(table) && uint32(n) <= maxCount; n++ {
		table[n] = shade(uint32(n))
	}
	for i, n := range counts {
		if n == 0 {
			continue
		}
		var px [4]uint8
		if n < uint32(len(table)) {
			px = table[n]
		} else {
			px = shade(n)
		}
		copy(img.Pix[i*4:i*4+4], px[:])
	}
	cnv.drawPixels(v.X, v.Y, img)
}

// Nearest implements ChartSeries
func (s *ScatterSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	best, bestD := -1, math.Inf(1)
	check := func(k int) {
		d := math.Abs(s.px[k] - px)
		if !byX {
			d = math.Hypot(s.px[k]-px, s.py[k]-py)
		}
		if d < bestD {
			best, bestD = k, d
		}
	}
	if byX || s.grid == nil {
		for k := range s.drawn {
			check(k)
		}
	} else {
		cx, cy := int(px)/16, int(py)/16
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				for _, k := range s.grid[[2]int{cx + dx, cy + dy}] {
					check(k)
				}
			}
		}
	}
	if best < 0 {
		return ChartHit{}, false
	}
	p := s.points[s.drawn[best]]
	yl := v.YAxis.scale.Format(p.Y)
	if chartFinite(p.Value) {
		yl += "  (" + trimChartZeros(strconv.FormatFloat(p.Value, 'f', 3, 64)) + ")"
	}
	name := s.name
	if p.Label != "" {
		name = p.Label
	}
	return ChartHit{
		Series: s, Index: s.drawn[best], X: p.X, Y: p.Y, PX: s.px[best], PY: s.py[best],
		XLabel: v.XAxis.scale.Format(p.X), YLabel: yl, Name: name, Color: s.Color(v.Theme), dist: bestD,
	}, true
}

// Legend implements ChartSeries
func (s *ScatterSeries) Legend(th *ChartTheme) []ChartLegendItem {
	items := s.legendItem(th, ChartLegendMarker)
	items[0].Marker = s.marker
	return items
}
