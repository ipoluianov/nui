package ui

import (
	"image"
	"image/color"
	"math"
	"sort"
	"strconv"
)

// HeatmapSeries draws a grid of values as colored cells: a correlation
// matrix, the load by hour and weekday, a temperature field, a spectrogram.
// The cells cover the rectangle from (x0, y0) to (x1, y1) of the axes; on
// category axes the cell (row, col) is at the categories col and row.
//
//	h := chart.Pane(0).AddHeatmap("Load")
//	h.SetGrid(values, 0, 24, 0, 7) // 24 hours by 7 days
type HeatmapSeries struct {
	ChartSeriesBase
	grid           [][]float64 // [row][col], row 0 at y0
	x0, x1, y0, y1 float64
	colors         []color.RGBA
	vmin, vmax     float64
	labels         bool
	colorBar       bool
	// hex: the cells of a Histogram2DSeries are hexagons of this radius in
	// pixels
	hex float64
	// counts of a hexagonal histogram, by the cells of the last paint
	hexCells map[[2]int]float64
	hexPts   func() []ChartPoint
	// gridRange is the range of the values of the grid, found by SetGrid
	gridRange [2]float64
}

// NewHeatmapSeries makes a heat map of the name
func NewHeatmapSeries(name string) *HeatmapSeries {
	s := &HeatmapSeries{colorBar: true, gridRange: [2]float64{math.Inf(1), math.Inf(-1)}}
	s.name = name
	return s
}

// AddHeatmap adds a heat map of the name to the pane, its axes without
// grids (the cells are the grid)
func (p *ChartPane) AddHeatmap(name string) *HeatmapSeries {
	s := NewHeatmapSeries(name)
	p.AddSeries(s)
	p.xAxis.SetGrid(false, false)
	p.YAxis().SetGrid(false, false).SetMargin(0)
	return s
}

// SetGrid sets the values, rows by columns, and the rectangle they cover;
// with x0 == x1 the cells are at the categories (or the numbers) 0, 1, 2...
func (s *HeatmapSeries) SetGrid(values [][]float64, x0, x1, y0, y1 float64) *HeatmapSeries {
	s.grid = values
	cols := 0
	for _, r := range values {
		cols = max(cols, len(r))
	}
	if x0 == x1 {
		x0, x1 = -0.5, float64(cols)-0.5
	}
	if y0 == y1 {
		y0, y1 = -0.5, float64(len(values))-0.5
	}
	s.x0, s.x1, s.y0, s.y1 = x0, x1, y0, y1
	s.gridRange = [2]float64{math.Inf(1), math.Inf(-1)}
	for _, r := range values {
		for _, v := range r {
			if v == v && !math.IsInf(v, 0) {
				if v < s.gridRange[0] {
					s.gridRange[0] = v
				}
				if v > s.gridRange[1] {
					s.gridRange[1] = v
				}
			}
		}
	}
	s.update()
	return s
}

// SetColors sets the scale of colors from the lowest value to the highest
// (nil - the theme's Sequential) and its range (vmin == vmax - of the data)
func (s *HeatmapSeries) SetColors(colors []color.RGBA, vmin, vmax float64) *HeatmapSeries {
	s.colors, s.vmin, s.vmax = colors, vmin, vmax
	s.update()
	return s
}

// SetLabels writes the value in each cell big enough for it
func (s *HeatmapSeries) SetLabels(show bool) *HeatmapSeries {
	s.labels = show
	s.update()
	return s
}

// SetColorBar shows or hides the scale of colors at the right of the plot
func (s *HeatmapSeries) SetColorBar(show bool) *HeatmapSeries {
	s.colorBar = show
	s.update()
	return s
}

func (s *HeatmapSeries) cols() int {
	n := 0
	for _, r := range s.grid {
		n = max(n, len(r))
	}
	return n
}

// valueRange returns the range of the colors: of the grid's values, found
// once for a grid
func (s *HeatmapSeries) valueRange() (float64, float64) {
	if s.vmax > s.vmin {
		return s.vmin, s.vmax
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	if s.gridRange[0] <= s.gridRange[1] {
		lo, hi = s.gridRange[0], s.gridRange[1]
	}
	for _, v := range s.hexCells {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	if lo == hi {
		hi = lo + 1
	}
	return lo, hi
}

func (s *HeatmapSeries) palette(th *ChartTheme) []color.RGBA {
	if s.colors != nil {
		return s.colors
	}
	return th.Sequential
}

// Range implements ChartSeries
func (s *HeatmapSeries) Range(axis *ChartAxis, v *ChartViewport) (float64, float64, bool) {
	if s.hexPts != nil {
		pts := s.hexPts()
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, p := range pts {
			c := p.Y
			if axis.Horizontal() {
				c = p.X
			}
			if chartFinite(c) {
				lo, hi = math.Min(lo, c), math.Max(hi, c)
			}
		}
		return lo, hi, lo <= hi
	}
	if len(s.grid) == 0 {
		return 0, 0, false
	}
	if axis.Horizontal() {
		return s.x0, s.x1, true
	}
	return s.y0, s.y1, true
}

// Prepare implements ChartSeries
func (s *HeatmapSeries) Prepare(v *ChartViewport) {}

// Draw implements ChartSeries
func (s *HeatmapSeries) Draw(cnv *Canvas, v *ChartViewport) {
	if s.hex > 0 {
		s.drawHex(cnv, v)
		return
	}
	rows, cols := len(s.grid), s.cols()
	if rows == 0 || cols == 0 {
		return
	}
	lo, hi := s.valueRange()
	pal := s.palette(v.Theme)
	dx, dy := (s.x1-s.x0)/float64(cols), (s.y1-s.y0)/float64(rows)
	cellW := math.Abs(v.PX(s.x0+dx) - v.PX(s.x0))
	cellH := math.Abs(v.PY(s.y0+dy) - v.PY(s.y0))
	if cellW < 3 || cellH < 3 {
		s.drawRaster(cnv, v, lo, hi, pal)
	} else {
		p := pathPool.Get().(*Path)
		defer pathPool.Put(p)
		gap := 0.0
		if cellW >= 8 && cellH >= 8 {
			gap = 1
		}
		for r, row := range s.grid {
			for c, val := range row {
				if !chartFinite(val) {
					continue
				}
				xa, xb := math.Round(v.PX(s.x0+float64(c)*dx)), math.Round(v.PX(s.x0+float64(c+1)*dx))
				ya, yb := math.Round(v.PY(s.y0+float64(r)*dy)), math.Round(v.PY(s.y0+float64(r+1)*dy))
				x, y := math.Min(xa, xb), math.Min(ya, yb)
				w, h := math.Abs(xb-xa)-gap, math.Abs(yb-ya)-gap
				if x+w < v.X || x > v.X+v.W || y+h < v.Y || y > v.Y+v.H {
					continue
				}
				col := ColorScale(pal, (val-lo)/(hi-lo))
				p.Reset()
				p.Rect(x, y, math.Max(w, 1), math.Max(h, 1))
				cnv.FillPath(p, col)
				if s.labels {
					text := chartValueLabel(NewLinearScale(lo, hi), val)
					if chartTextWidth(cnv, text)+4 <= w {
						chartDrawLabel(cnv, text, x+w/2, y+h/2, HAlignCenter, VAlignCenter, chartContrastText(col))
					}
				}
			}
		}
	}
}

// outsideWidth is the room of the color bar on the right of the plot
func (s *HeatmapSeries) outsideWidth(cnv *Canvas) float64 {
	if !s.colorBar {
		return 0
	}
	lo, hi := s.valueRange()
	f := NewLinearScale(lo, hi)
	w := math.Max(chartTextWidth(cnv, chartValueLabel(f, lo)), chartTextWidth(cnv, chartValueLabel(f, hi)))
	return 8 + 10 + 4 + w + 4
}

// drawOutside draws the color bar on the right of the plot
func (s *HeatmapSeries) drawOutside(cnv *Canvas, v *ChartViewport) {
	if s.colorBar {
		s.drawColorBar(cnv, v)
	}
}

// drawRaster draws a grid of cells smaller than pixels: each pixel of the
// plot takes the color of the cell under its middle
func (s *HeatmapSeries) drawRaster(cnv *Canvas, v *ChartViewport, lo, hi float64, pal []color.RGBA) {
	scale := cnv.Scale()
	w, h := int(math.Ceil(v.W*scale)), int(math.Ceil(v.H*scale))
	if w <= 0 || h <= 0 {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rows, cols := len(s.grid), s.cols()
	dx, dy := (s.x1-s.x0)/float64(cols), (s.y1-s.y0)/float64(rows)
	colIdx := make([]int, w)
	for px := 0; px < w; px++ {
		x := v.DataX(v.X + (float64(px)+0.5)/scale)
		colIdx[px] = int(math.Floor((x - s.x0) / dx))
	}
	// The colors of the scale in 256 steps, premultiplied
	var lut [256][4]uint8
	for i := range lut {
		col := ColorScale(pal, float64(i)/255)
		a := uint32(col.A)
		lut[i] = [4]uint8{uint8(uint32(col.R) * a / 255), uint8(uint32(col.G) * a / 255), uint8(uint32(col.B) * a / 255), col.A}
	}
	k := 255 / (hi - lo)
	for py := 0; py < h; py++ {
		y := v.DataY(v.Y + (float64(py)+0.5)/scale)
		r := int(math.Floor((y - s.y0) / dy))
		if r < 0 || r >= rows {
			continue
		}
		row := s.grid[r]
		line := img.Pix[py*img.Stride : py*img.Stride+w*4]
		for px, c := range colIdx {
			if c < 0 || c >= len(row) {
				continue
			}
			val := row[c]
			if val != val || math.IsInf(val, 0) {
				continue
			}
			t := (val - lo) * k
			idx := 0
			if t >= 255 {
				idx = 255
			} else if t > 0 {
				idx = int(t + 0.5)
			}
			copy(line[px*4:px*4+4], lut[idx][:])
		}
	}
	cnv.drawPixels(v.X, v.Y, img)
}

// drawColorBar draws the scale of colors with its ends at the right edge of
// the plot
func (s *HeatmapSeries) drawColorBar(cnv *Canvas, v *ChartViewport) {
	lo, hi := s.valueRange()
	pal := s.palette(v.Theme)
	const bw = 10.0
	h := math.Min(v.H, 200)
	if h < 30 {
		return
	}
	// Past the axes on the right of the plot
	x, y := v.X+v.W+8, v.Y
	for _, a := range v.Pane.axes {
		if a.position == ChartAxisRight {
			x += a.bandW
		}
	}
	p := pathPool.Get().(*Path)
	defer pathPool.Put(p)
	p.Reset()
	p.Rect(x, y, bw, h)
	// From the lowest value at the bottom to the highest at the top
	stops := make([]GradientStop, len(pal))
	for i, c := range pal {
		stops[i] = GradientStop{Offset: float64(i) / float64(max(len(pal)-1, 1)), Color: c}
	}
	cnv.FillPathPaint(p, LinearGradient{X1: 0, Y1: y + h, X2: 0, Y2: y, Stops: stops})
	cnv.StrokePath(p, StrokeStyle{Width: 1}, v.Theme.Axis)
	cnv.Save()
	cnv.SetFontSize(v.FontSize * 0.8)
	f := NewLinearScale(lo, hi)
	for _, e := range []struct {
		val float64
		y   float64
		va  VAlign
	}{{hi, y, VAlignTop}, {lo, y + h, VAlignBottom}} {
		chartDrawLabel(cnv, chartValueLabel(f, e.val), x+bw+4, e.y, HAlignLeft, e.va, v.Theme.Text)
	}
	cnv.Restore()
}

// Nearest implements ChartSeries: the cell under the pointer
func (s *HeatmapSeries) Nearest(v *ChartViewport, px, py float64, byX bool) (ChartHit, bool) {
	if s.hex > 0 {
		return s.nearestHex(v, px, py)
	}
	rows, cols := len(s.grid), s.cols()
	if rows == 0 || cols == 0 {
		return ChartHit{}, false
	}
	x, y := v.DataX(px), v.DataY(py)
	c := int(math.Floor((x - s.x0) / (s.x1 - s.x0) * float64(cols)))
	r := int(math.Floor((y - s.y0) / (s.y1 - s.y0) * float64(rows)))
	if r < 0 || r >= rows || c < 0 || c >= len(s.grid[r]) || !chartFinite(s.grid[r][c]) {
		return ChartHit{}, false
	}
	dx, dy := (s.x1-s.x0)/float64(cols), (s.y1-s.y0)/float64(rows)
	cx, cy := s.x0+(float64(c)+0.5)*dx, s.y0+(float64(r)+0.5)*dy
	val := s.grid[r][c]
	return ChartHit{
		Series: s, Index: r*cols + c, X: cx, Y: cy, PX: v.PX(cx), PY: v.PY(cy),
		XLabel: v.XAxis.scale.Format(cx) + ", " + v.YAxis.scale.Format(cy),
		YLabel: trimChartZeros(strconv.FormatFloat(val, 'f', 3, 64)), Name: s.name,
		Color: ColorScale(s.palette(v.Theme), func() float64 { lo, hi := s.valueRange(); return (val - lo) / (hi - lo) }()),
	}, true
}

// Legend implements ChartSeries
func (s *HeatmapSeries) Legend(th *ChartTheme) []ChartLegendItem {
	pal := s.palette(th)
	items := s.legendItem(th, ChartLegendBox)
	if len(pal) > 0 {
		items[0].Color = pal[len(pal)*2/3]
	}
	return items
}

// ---------------------------------------------------------------------------
// 2D histogram

// Histogram2DSeries counts points into the cells of a grid (or of
// hexagons, SetHex) and colors each by how many fell in: the density of a
// scatter of very many points, with the counts readable.
type Histogram2DSeries struct {
	HeatmapSeries
	points     []ChartPoint
	cols, rows int
}

// AddHistogram2D adds a 2D histogram of the name to the pane
func (p *ChartPane) AddHistogram2D(name string) *Histogram2DSeries {
	s := &Histogram2DSeries{cols: 40, rows: 30}
	s.HeatmapSeries = *NewHeatmapSeries(name)
	p.AddSeries(s)
	p.xAxis.SetGrid(false, false)
	p.YAxis().SetGrid(false, false)
	return s
}

// SetPoints replaces the points and counts them into the cells
func (s *Histogram2DSeries) SetPoints(points []ChartPoint) *Histogram2DSeries {
	s.points = append(s.points[:0], points...)
	s.bin()
	return s
}

// SetBins sets how many cells across and down the grid has
func (s *Histogram2DSeries) SetBins(cols, rows int) *Histogram2DSeries {
	s.cols, s.rows = max(cols, 1), max(rows, 1)
	s.bin()
	return s
}

// SetHex counts the points into hexagons of the radius in pixels instead of
// the cells of a grid: they are counted anew as the chart is zoomed
func (s *Histogram2DSeries) SetHex(radius float64) *Histogram2DSeries {
	s.hex = math.Max(radius, 0)
	s.hexPts = func() []ChartPoint { return s.points }
	s.update()
	return s
}

// bin counts the points into the grid over their range
func (s *Histogram2DSeries) bin() {
	if len(s.points) == 0 {
		s.SetGrid(nil, 0, 0, 0, 0)
		return
	}
	x0, x1, y0, y1 := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range s.points {
		x0, x1 = math.Min(x0, p.X), math.Max(x1, p.X)
		y0, y1 = math.Min(y0, p.Y), math.Max(y1, p.Y)
	}
	if x1 == x0 {
		x1 = x0 + 1
	}
	if y1 == y0 {
		y1 = y0 + 1
	}
	grid := make([][]float64, s.rows)
	for r := range grid {
		grid[r] = make([]float64, s.cols)
		for c := range grid[r] {
			grid[r][c] = math.NaN()
		}
	}
	for _, p := range s.points {
		c := min(int((p.X-x0)/(x1-x0)*float64(s.cols)), s.cols-1)
		r := min(int((p.Y-y0)/(y1-y0)*float64(s.rows)), s.rows-1)
		if math.IsNaN(grid[r][c]) {
			grid[r][c] = 0
		}
		grid[r][c]++
	}
	s.SetGrid(grid, x0, x1, y0, y1)
}

// drawHex counts the points into hexagons of the plot and draws them
func (s *HeatmapSeries) drawHex(cnv *Canvas, v *ChartViewport) {
	if s.hexPts == nil {
		return
	}
	r := s.hex
	w := math.Sqrt(3) * r
	s.hexCells = map[[2]int]float64{}
	for _, p := range s.hexPts() {
		x, y := v.PX(p.X)-v.X, v.PY(p.Y)-v.Y
		if x < -w || y < -r || x > v.W+w || y > v.H+r {
			continue
		}
		s.hexCells[hexCell(x, y, r)]++
	}
	lo, hi := s.valueRange()
	pal := s.palette(v.Theme)
	path := pathPool.Get().(*Path)
	defer pathPool.Put(path)
	// In a fixed order: the edges of the neighbors blend the same every time
	cells := make([][2]int, 0, len(s.hexCells))
	for cell := range s.hexCells {
		cells = append(cells, cell)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i][1] != cells[j][1] {
			return cells[i][1] < cells[j][1]
		}
		return cells[i][0] < cells[j][0]
	})
	for _, cell := range cells {
		n := s.hexCells[cell]
		cx, cy := hexCenter(cell, r)
		path.Reset()
		for k := 0; k < 6; k++ {
			a := math.Pi/6 + float64(k)*math.Pi/3
			px, py := v.X+cx+r*math.Cos(a), v.Y+cy+r*math.Sin(a)
			if k == 0 {
				path.MoveTo(px, py)
			} else {
				path.LineTo(px, py)
			}
		}
		path.Close()
		cnv.FillPath(path, ColorScale(pal, (n-lo)/(hi-lo)))
	}
}

// hexCell returns the hexagon (pointy-top, of the radius) the point is in
func hexCell(x, y, r float64) [2]int {
	q := (math.Sqrt(3)/3*x - y/3) / r
	rr := (2.0 / 3 * y) / r
	// Cube rounding
	cx, cz := q, rr
	cy := -cx - cz
	rx, ry, rz := math.Round(cx), math.Round(cy), math.Round(cz)
	dx, dy, dz := math.Abs(rx-cx), math.Abs(ry-cy), math.Abs(rz-cz)
	if dx > dy && dx > dz {
		rx = -ry - rz
	} else if dy <= dz {
		rz = -rx - ry
	}
	return [2]int{int(rx), int(rz)}
}

// hexCenter returns the middle of the hexagon in pixels of the plot
func hexCenter(cell [2]int, r float64) (float64, float64) {
	q, rr := float64(cell[0]), float64(cell[1])
	return r * math.Sqrt(3) * (q + rr/2), r * 1.5 * rr
}

func (s *HeatmapSeries) nearestHex(v *ChartViewport, px, py float64) (ChartHit, bool) {
	cell := hexCell(px-v.X, py-v.Y, s.hex)
	n, ok := s.hexCells[cell]
	if !ok {
		return ChartHit{}, false
	}
	cx, cy := hexCenter(cell, s.hex)
	return ChartHit{
		Series: s, Index: -1, PX: v.X + cx, PY: v.Y + cy, X: v.DataX(v.X + cx), Y: v.DataY(v.Y + cy),
		XLabel: v.XAxis.scale.Format(v.DataX(v.X+cx)) + ", " + v.YAxis.scale.Format(v.DataY(v.Y+cy)),
		YLabel: strconv.Itoa(int(n)), Name: s.name, Color: ColorScale(s.palette(v.Theme), 0.7),
	}, true
}
