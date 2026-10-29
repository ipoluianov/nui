package ui

import (
	"image/color"
	"math"
	"sort"
	"strconv"
)

// ---------------------------------------------------------------------------
// Histogram

// ChartHistogramMode is what the bars of a histogram show
type ChartHistogramMode int

const (
	// ChartHistogramCount: how many values fall into each bin
	ChartHistogramCount ChartHistogramMode = iota
	// ChartHistogramDensity: the part of the values per unit of x, so the
	// area of the bars is 1 and histograms of different sizes compare
	ChartHistogramDensity
	// ChartHistogramCumulative: how many values are up to the end of the bin
	ChartHistogramCumulative
)

// HistogramSeries counts raw values into bins and draws them as bars
// touching each other: the distribution of a measurement. The bins are
// picked by the Freedman–Diaconis rule unless SetBins fixes them.
//
//	h := chart.Pane(0).AddHistogram("Response time")
//	h.SetValues(samples)
type HistogramSeries struct {
	ChartSeriesBase
	values []float64
	bins   int
	width  float64
	mode   ChartHistogramMode

	// The bins of the last Prepare: their edges and heights
	edges  []float64
	counts []float64
}

// NewHistogramSeries makes a histogram of the name
func NewHistogramSeries(name string) *HistogramSeries {
	s := &HistogramSeries{}
	s.name = name
	return s
}

// AddHistogram adds a histogram of the name to the pane
func (p *ChartPane) AddHistogram(name string) *HistogramSeries {
	s := NewHistogramSeries(name)
	p.AddSeries(s)
	p.YAxis().SetIncludeZero(true)
	return s
}

// SetValues replaces the raw values
func (s *HistogramSeries) SetValues(values []float64) *HistogramSeries {
	s.values = append(s.values[:0], values...)
	s.update()
	return s
}

// AddValues adds raw values
func (s *HistogramSeries) AddValues(values ...float64) *HistogramSeries {
	s.values = append(s.values, values...)
	s.update()
	return s
}

// SetBins fixes the number of the bins (0 - picked from the data); width
// above 0 fixes their width instead
func (s *HistogramSeries) SetBins(n int, width float64) *HistogramSeries {
	s.bins, s.width = max(n, 0), math.Max(width, 0)
	s.update()
	return s
}

// SetMode sets what the bars show
func (s *HistogramSeries) SetMode(mode ChartHistogramMode) *HistogramSeries {
	s.mode = mode
	s.update()
	return s
}

// Bins returns the edges of the bins (one more than the bins) and the
// heights of the bars as of the last paint
func (s *HistogramSeries) Bins() (edges, heights []float64) { return s.edges, s.counts }

// compute counts the values into the bins
func (s *HistogramSeries) compute() {
	s.edges, s.counts = nil, nil
	vals := make([]float64, 0, len(s.values))
	for _, v := range s.values {
		if chartFinite(v) {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return
	}
	sort.Float64s(vals)
	lo, hi := vals[0], vals[len(vals)-1]
	width := s.width
	switch {
	case width > 0:
	case s.bins > 0:
		width = (hi - lo) / float64(s.bins)
	default:
		// Freedman–Diaconis: 2 IQR / cbrt(n), at least a bin per 4 values
		iqr := quantileSorted(vals, 0.75) - quantileSorted(vals, 0.25)
		width = 2 * iqr / math.Cbrt(float64(len(vals)))
		if !(width > 0) {
			width = (hi - lo) / math.Max(1, math.Ceil(math.Sqrt(float64(len(vals)))))
		}
	}
	if !(width > 0) {
		width = 1
	}
	// Round edges: a nice step close to the width
	if s.width == 0 {
		width = niceStepAtLeast(width * 0.7)
	}
	start := math.Floor(lo/width) * width
	n := int(math.Floor((hi-start)/width)) + 1
	n = max(1, min(n, 10000))
	s.edges = make([]float64, n+1)
	for i := range s.edges {
		s.edges[i] = start + float64(i)*width
	}
	s.counts = make([]float64, n)
	for _, v := range vals {
		i := min(int((v-start)/width), n-1)
		s.counts[i]++
	}
	switch s.mode {
	case ChartHistogramDensity:
		for i := range s.counts {
			s.counts[i] /= float64(len(vals)) * width
		}
	case ChartHistogramCumulative:
		for i := 1; i < n; i++ {
			s.counts[i] += s.counts[i-1]
		}
	}
}

// quantileSorted returns the quantile q of the sorted values, interpolated
// between the two around it
func quantileSorted(vals []float64, q float64) float64 {
	if len(vals) == 0 {
		return math.NaN()
	}
	pos := q * float64(len(vals)-1)
	i := int(math.Floor(pos))
	if i >= len(vals)-1 {
		return vals[len(vals)-1]
	}
	f := pos - float64(i)
	return vals[i]*(1-f) + vals[i+1]*f
}

// Range implements ChartSeries
func (s *HistogramSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	s.compute()
	if len(s.counts) == 0 {
		return 0, 0, false
	}
	if axis.Horizontal() {
		return s.edges[0], s.edges[len(s.edges)-1], true
	}
	hi := 0.0
	for _, c := range s.counts {
		hi = math.Max(hi, c)
	}
	return 0, hi, true
}

// Prepare implements ChartSeries
func (s *HistogramSeries) Prepare(v *ChartViewport) { s.compute() }

// Draw implements ChartSeries
func (s *HistogramSeries) Draw(cnv *Canvas, v *ChartViewport) {
	col := s.Color(v.Theme)
	edge := v.Theme.PlotBackground
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	zero := v.PY(0)
	for i, c := range s.counts {
		if c <= 0 {
			continue
		}
		x1, x2 := math.Round(v.PX(s.edges[i])), math.Round(v.PX(s.edges[i+1]))
		y := math.Round(v.PY(c))
		p.Reset()
		p.Rect(x1, y, math.Max(1, x2-x1), zero-y)
		cnv.FillPath(p, col)
		// A line of the background between the bars, when they are wide enough
		if x2-x1 >= 4 {
			p.Reset()
			p.MoveTo(x2-0.5, y)
			p.LineTo(x2-0.5, zero)
			cnv.StrokePath(p, StrokeStyle{Width: 1}, edge)
		}
	}
}

// Nearest implements ChartSeries
func (s *HistogramSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	x := v.DataX(px)
	for i, c := range s.counts {
		if x >= s.edges[i] && x < s.edges[i+1] {
			f := v.XAxis.scale.Format
			return ChartHit{
				Series: s, Index: i, X: (s.edges[i] + s.edges[i+1]) / 2, Y: c,
				PX: v.PX((s.edges[i] + s.edges[i+1]) / 2), PY: v.PY(c),
				XLabel: f(s.edges[i]) + " … " + f(s.edges[i+1]), YLabel: trimChartZeros(strconv.FormatFloat(c, 'f', 3, 64)),
				Name: s.name, Color: s.Color(v.Theme),
			}, true
		}
	}
	return ChartHit{}, false
}

// Legend implements ChartSeries
func (s *HistogramSeries) Legend(th *ChartTheme) []ChartLegendItem {
	return s.legendItem(th, ChartLegendBox)
}

// ---------------------------------------------------------------------------
// Box plot

// BoxStats are the numbers of a box of a box plot
type BoxStats struct {
	Min, Q1, Median, Q3, Max float64
	// The whiskers end at the farthest values within 1.5 IQR of the box
	LowWhisker, HighWhisker float64
	Mean                    float64
	// Outliers are the values beyond the whiskers
	Outliers []float64
	N        int
}

// ComputeBoxStats returns the box of the values
func ComputeBoxStats(values []float64) BoxStats {
	vals := make([]float64, 0, len(values))
	sum := 0.0
	for _, v := range values {
		if chartFinite(v) {
			vals = append(vals, v)
			sum += v
		}
	}
	var b BoxStats
	b.N = len(vals)
	if b.N == 0 {
		return b
	}
	sort.Float64s(vals)
	b.Min, b.Max = vals[0], vals[len(vals)-1]
	b.Q1, b.Median, b.Q3 = quantileSorted(vals, 0.25), quantileSorted(vals, 0.5), quantileSorted(vals, 0.75)
	b.Mean = sum / float64(b.N)
	iqr := b.Q3 - b.Q1
	lo, hi := b.Q1-1.5*iqr, b.Q3+1.5*iqr
	b.LowWhisker, b.HighWhisker = b.Q1, b.Q3
	for _, v := range vals {
		if v >= lo {
			b.LowWhisker = math.Min(b.LowWhisker, v)
		}
		if v <= hi {
			b.HighWhisker = math.Max(b.HighWhisker, v)
		}
		if v < lo || v > hi {
			b.Outliers = append(b.Outliers, v)
		}
	}
	return b
}

// BoxPlotSeries draws a box with whiskers per category of a CategoryScale:
// the quartiles, the median, the whiskers within 1.5 IQR and the outliers;
// as a violin (SetViolin), the shape of the distribution around the box.
//
//	box := chart.Pane(0).AddBoxPlot("Latency")
//	box.Set("eu-west", samplesEU).Set("us-east", samplesUS)
type BoxPlotSeries struct {
	ChartSeriesBase
	values [][]float64
	stats  []BoxStats
	violin bool
	mean   bool

	rects []barRect
}

// NewBoxPlotSeries makes a box plot of the name
func NewBoxPlotSeries(name string) *BoxPlotSeries {
	s := &BoxPlotSeries{mean: true}
	s.name = name
	return s
}

// AddBoxPlot adds a box plot of the name to the pane; the pane's x axis
// becomes a CategoryScale if it is not one
func (p *ChartPane) AddBoxPlot(name string) *BoxPlotSeries {
	if _, ok := p.xAxis.scale.(*CategoryScale); !ok {
		p.xAxis.SetScale(NewCategoryScale())
		p.xAxis.SetGrid(false, false)
	}
	s := NewBoxPlotSeries(name)
	p.AddSeries(s)
	return s
}

// Set sets the raw values of the category, adding the category to the axis
// when it has none of the name
func (s *BoxPlotSeries) Set(category string, values []float64) *BoxPlotSeries {
	if s.pane == nil {
		return s
	}
	x, _ := s.pane.seriesAxes(s)
	cs, ok := x.scale.(*CategoryScale)
	if !ok {
		return s
	}
	i := cs.Index(category)
	if i < 0 {
		cs.SetCategories(append(append([]string(nil), cs.categories...), category))
		i = len(cs.categories) - 1
	}
	for len(s.values) <= i {
		s.values = append(s.values, nil)
		s.stats = append(s.stats, BoxStats{})
	}
	s.values[i] = append([]float64(nil), values...)
	s.stats[i] = ComputeBoxStats(values)
	s.update()
	return s
}

// SetViolin draws the shape of the distribution around a thin box
func (s *BoxPlotSeries) SetViolin(violin bool) *BoxPlotSeries {
	s.violin = violin
	s.update()
	return s
}

// SetMean shows the mean as a small diamond
func (s *BoxPlotSeries) SetMean(show bool) *BoxPlotSeries {
	s.mean = show
	s.update()
	return s
}

// Stats returns the box of the category number i
func (s *BoxPlotSeries) Stats(i int) BoxStats {
	if i < 0 || i >= len(s.stats) {
		return BoxStats{}
	}
	return s.stats[i]
}

// Range implements ChartSeries
func (s *BoxPlotSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if axis.Horizontal() {
		return -0.5, float64(len(s.stats)) - 0.5, len(s.stats) > 0
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, b := range s.stats {
		if b.N == 0 {
			continue
		}
		lo, hi = math.Min(lo, b.Min), math.Max(hi, b.Max)
	}
	return lo, hi, lo <= hi
}

// Prepare implements ChartSeries
func (s *BoxPlotSeries) Prepare(v *ChartViewport) {}

// Draw implements ChartSeries
func (s *BoxPlotSeries) Draw(cnv *Canvas, v *ChartViewport) {
	col := s.Color(v.Theme)
	fill := col
	fill.A = 70
	slot, slots := v.Pane.barSlotOf(s)
	band := math.Abs(v.PX(1) - v.PX(0))
	group := band * 0.7
	w := group / float64(slots)
	s.rects = s.rects[:0]
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	for i, b := range s.stats {
		if b.N == 0 {
			continue
		}
		center := v.PX(float64(i)) - group/2 + (float64(slot)+0.5)*w
		bw := w * 0.8
		if s.violin {
			s.drawViolin(cnv, v, p, s.values[i], center, bw/2, fill, col)
			bw = math.Min(bw*0.18, 10)
		}
		x1, x2 := center-bw/2, center+bw/2
		yq1, yq3 := v.PY(b.Q1), v.PY(b.Q3)
		s.rects = append(s.rects, barRect{x: x1, y: math.Min(yq1, yq3), w: bw, h: math.Abs(yq1 - yq3), index: i})
		// The whiskers with their caps
		p.Reset()
		p.MoveTo(chartCrisp(center), v.PY(b.HighWhisker))
		p.LineTo(chartCrisp(center), yq3)
		p.MoveTo(chartCrisp(center), yq1)
		p.LineTo(chartCrisp(center), v.PY(b.LowWhisker))
		if !s.violin {
			cw := bw / 3
			p.MoveTo(center-cw, chartCrisp(v.PY(b.HighWhisker)))
			p.LineTo(center+cw, chartCrisp(v.PY(b.HighWhisker)))
			p.MoveTo(center-cw, chartCrisp(v.PY(b.LowWhisker)))
			p.LineTo(center+cw, chartCrisp(v.PY(b.LowWhisker)))
		}
		cnv.StrokePath(p, StrokeStyle{Width: 1}, col)
		// The box and its median
		p.Reset()
		p.Rect(math.Round(x1)+0.5, math.Round(yq3)+0.5, math.Round(bw)-1, math.Round(yq1)-math.Round(yq3))
		if !s.violin {
			cnv.FillPath(p, fill)
		} else {
			cnv.FillPath(p, col)
		}
		cnv.StrokePath(p, StrokeStyle{Width: 1}, col)
		p.Reset()
		ym := chartCrisp(v.PY(b.Median))
		p.MoveTo(x1, ym)
		p.LineTo(x2, ym)
		medCol := col
		if s.violin {
			medCol = v.Theme.PlotBackground
		}
		cnv.StrokePath(p, StrokeStyle{Width: 2}, medCol)
		if s.mean && !s.violin {
			cnv.DrawMarker(MarkerDiamond, center, v.PY(b.Mean), 6, v.Theme.PlotBackground, col)
		}
		for _, o := range b.Outliers {
			cnv.DrawMarker(MarkerCircle, center, v.PY(o), 5, nil, col)
		}
	}
}

// drawViolin draws the density of the values (a Gaussian kernel, Silverman's
// bandwidth) mirrored around the center, half as wide as hw at its widest
func (s *BoxPlotSeries) drawViolin(cnv *Canvas, v *ChartViewport, p *Path, values []float64, center, hw float64, fill, col color.RGBA) {
	b := ComputeBoxStats(values)
	if b.N < 2 {
		return
	}
	sd := 0.0
	for _, x := range values {
		if chartFinite(x) {
			sd += (x - b.Mean) * (x - b.Mean)
		}
	}
	sd = math.Sqrt(sd / float64(b.N-1))
	bw := 0.9 * math.Min(sd, (b.Q3-b.Q1)/1.34) * math.Pow(float64(b.N), -0.2)
	if !(bw > 0) {
		bw = math.Max(sd, 1e-9) * 0.5
	}
	const steps = 48
	ys := make([]float64, steps+1)
	ds := make([]float64, steps+1)
	maxD := 0.0
	for k := 0; k <= steps; k++ {
		y := b.Min + (b.Max-b.Min)*float64(k)/steps
		d := 0.0
		for _, x := range values {
			if chartFinite(x) {
				z := (y - x) / bw
				d += math.Exp(-z * z / 2)
			}
		}
		ys[k], ds[k] = y, d
		maxD = math.Max(maxD, d)
	}
	if maxD == 0 {
		return
	}
	p.Reset()
	for k := 0; k <= steps; k++ {
		x := center + ds[k]/maxD*hw
		if k == 0 {
			p.MoveTo(x, v.PY(ys[k]))
		} else {
			p.LineTo(x, v.PY(ys[k]))
		}
	}
	for k := steps; k >= 0; k-- {
		p.LineTo(center-ds[k]/maxD*hw, v.PY(ys[k]))
	}
	p.Close()
	cnv.FillPath(p, fill)
	cnv.StrokePath(p, StrokeStyle{Width: 1}, col)
}

// Nearest implements ChartSeries
func (s *BoxPlotSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	for _, r := range s.rects {
		if px >= r.x-4 && px <= r.x+r.w+4 {
			b := s.stats[r.index]
			f := v.YAxis.scale.Format
			cs, _ := v.XAxis.scale.(*CategoryScale)
			cat := ""
			if cs != nil {
				cat = cs.name(r.index)
			}
			return ChartHit{
				Series: s, Index: r.index, X: float64(r.index), Y: b.Median, PX: r.x + r.w/2, PY: v.PY(b.Median),
				XLabel: cat, Name: s.name, Color: s.Color(v.Theme),
				YLabel: "median " + f(b.Median) + ", Q1 " + f(b.Q1) + ", Q3 " + f(b.Q3) + ", n " + strconv.Itoa(b.N),
			}, true
		}
	}
	return ChartHit{}, false
}

// Legend implements ChartSeries
func (s *BoxPlotSeries) Legend(th *ChartTheme) []ChartLegendItem {
	return s.legendItem(th, ChartLegendBox)
}

// barSlotOf returns the place of the series among the box plots of the
// pane: they stand side by side in a category
func (p *ChartPane) barSlotOf(s ChartSeries) (slot, slots int) {
	for _, o := range p.series {
		if _, ok := o.(*BoxPlotSeries); ok && o.Visible() {
			if o == s {
				slot = slots
			}
			slots++
		}
	}
	return slot, max(slots, 1)
}

// ---------------------------------------------------------------------------
// Bands and error bars

// BandSample is a point of a band or of an error bar: the value at X and
// the range around it
type BandSample struct {
	X, Y, Low, High float64
}

// BandSeries draws a range around a line: a corridor of min and max, ± σ of
// a measurement, the confidence interval of a forecast, the tolerance of a
// process. As error bars (SetErrorBars) it draws a whisker at each point.
//
//	b := chart.Pane(0).AddBand("Forecast")
//	b.Add(x, mean, mean-sd, mean+sd)
type BandSeries struct {
	ChartSeriesBase
	samples   []BandSample
	errorBars bool
	showLine  bool
	opacity   float64
}

// NewBandSeries makes a band of the name
func NewBandSeries(name string) *BandSeries {
	s := &BandSeries{showLine: true, opacity: 0.2}
	s.name = name
	return s
}

// AddBand adds a band of the name to the pane
func (p *ChartPane) AddBand(name string) *BandSeries {
	s := NewBandSeries(name)
	p.AddSeries(s)
	return s
}

// Add adds a point: the value y at x and its range from low to high; the
// points are kept sorted by x
func (s *BandSeries) Add(x, y, low, high float64) *BandSeries {
	b := BandSample{X: x, Y: y, Low: math.Min(low, high), High: math.Max(low, high)}
	i := sort.Search(len(s.samples), func(i int) bool { return s.samples[i].X > x })
	s.samples = append(s.samples, BandSample{})
	copy(s.samples[i+1:], s.samples[i:])
	s.samples[i] = b
	s.update()
	return s
}

// SetSamples replaces the points
func (s *BandSeries) SetSamples(samples []BandSample) *BandSeries {
	s.samples = append(s.samples[:0], samples...)
	sort.SliceStable(s.samples, func(i, j int) bool { return s.samples[i].X < s.samples[j].X })
	s.update()
	return s
}

// SetErrorBars draws a whisker at each point instead of a band
func (s *BandSeries) SetErrorBars(on bool) *BandSeries {
	s.errorBars = on
	s.update()
	return s
}

// SetLine shows or hides the line of the values
func (s *BandSeries) SetLine(show bool) *BandSeries {
	s.showLine = show
	s.update()
	return s
}

// SetOpacity sets how much of the series color the band takes (0.2)
func (s *BandSeries) SetOpacity(opacity float64) *BandSeries {
	s.opacity = math.Max(0, math.Min(1, opacity))
	s.update()
	return s
}

// Range implements ChartSeries
func (s *BandSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if len(s.samples) == 0 {
		return 0, 0, false
	}
	if axis.Horizontal() {
		return s.samples[0].X, s.samples[len(s.samples)-1].X, true
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	xmin, xmax := v.XRange()
	for _, b := range s.samples {
		if b.X >= xmin && b.X <= xmax {
			lo, hi = math.Min(lo, math.Min(b.Low, b.Y)), math.Max(hi, math.Max(b.High, b.Y))
		}
	}
	return lo, hi, lo <= hi
}

// Prepare implements ChartSeries
func (s *BandSeries) Prepare(v *ChartViewport) {}

// chartXMargin: whiskers at the ends are not cut
func (s *BandSeries) chartXMargin() float64 {
	if s.errorBars {
		return 0.03
	}
	return 0
}

// Draw implements ChartSeries
func (s *BandSeries) Draw(cnv *Canvas, v *ChartViewport) {
	if len(s.samples) == 0 {
		return
	}
	col := s.Color(v.Theme)
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	if s.errorBars {
		p.Reset()
		for _, b := range s.samples {
			x := chartCrisp(v.PX(b.X))
			yh, yl := v.PY(b.High), v.PY(b.Low)
			p.MoveTo(x, yh)
			p.LineTo(x, yl)
			p.MoveTo(x-4, chartCrisp(yh))
			p.LineTo(x+4, chartCrisp(yh))
			p.MoveTo(x-4, chartCrisp(yl))
			p.LineTo(x+4, chartCrisp(yl))
		}
		cnv.StrokePath(p, StrokeStyle{Width: 1.2}, col)
		for _, b := range s.samples {
			cnv.DrawMarker(MarkerCircle, v.PX(b.X), v.PY(b.Y), 6, col, v.Theme.PlotBackground)
		}
		return
	}
	fill := col
	fill.A = uint8(float64(col.A) * s.opacity)
	p.Reset()
	for i, b := range s.samples {
		if i == 0 {
			p.MoveTo(v.PX(b.X), v.PY(b.High))
		} else {
			p.LineTo(v.PX(b.X), v.PY(b.High))
		}
	}
	for i := len(s.samples) - 1; i >= 0; i-- {
		p.LineTo(v.PX(s.samples[i].X), v.PY(s.samples[i].Low))
	}
	p.Close()
	cnv.FillPath(p, fill)
	if s.showLine {
		p.Reset()
		for i, b := range s.samples {
			if i == 0 {
				p.MoveTo(v.PX(b.X), v.PY(b.Y))
			} else {
				p.LineTo(v.PX(b.X), v.PY(b.Y))
			}
		}
		cnv.StrokePath(p, StrokeStyle{Width: 1.5, Join: LineJoinRound}, col)
	}
}

// Nearest implements ChartSeries
func (s *BandSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	if len(s.samples) == 0 {
		return ChartHit{}, false
	}
	x := v.DataX(px)
	i := sort.Search(len(s.samples), func(i int) bool { return s.samples[i].X >= x })
	if i == len(s.samples) || (i > 0 && x-s.samples[i-1].X < s.samples[i].X-x) {
		i--
	}
	b := s.samples[i]
	f := v.YAxis.scale.Format
	hx, hy := v.PX(b.X), v.PY(b.Y)
	d := math.Abs(hx - px)
	if !byX {
		d = math.Hypot(hx-px, hy-py)
	}
	return ChartHit{
		Series: s, Index: i, X: b.X, Y: b.Y, PX: hx, PY: hy, dist: d,
		XLabel: v.XAxis.scale.Format(b.X), YLabel: f(b.Y) + " (" + f(b.Low) + " … " + f(b.High) + ")",
		Name: s.name, Color: s.Color(v.Theme),
	}, true
}

// Legend implements ChartSeries
func (s *BandSeries) Legend(th *ChartTheme) []ChartLegendItem {
	return s.legendItem(th, ChartLegendBox)
}
