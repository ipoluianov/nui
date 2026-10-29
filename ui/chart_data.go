package ui

import (
	"math"
	"sort"
	"sync"
)

// ChartSample is one sample of a series at X, or a bucket of samples merged
// together. For a raw sample the four values are equal; for a bucket First
// and Last are the first and the last values in it and High and Low its
// extremes, so a signal that jitters inside a bucket is drawn as a band
// instead of a flat average. The same shape is an OHLC bar.
//
// HasGood and HasBad are the quality of the data, as in TimeChartPoint: a
// sample without HasGood has no value and breaks the line; HasBad marks bad
// data (drawn as a hatch); a sample with neither is a gap, a break in the
// data.
type ChartSample struct {
	X                      float64
	First, Last, High, Low float64
	HasGood, HasBad        bool
}

// ChartValue returns a good sample of the value at x
func ChartValue(x, v float64) ChartSample {
	return ChartSample{X: x, First: v, Last: v, High: v, Low: v, HasGood: true}
}

// ChartOHLC returns a good sample of a bar: open, high, low, close
func ChartOHLC(x, open, high, low, close float64) ChartSample {
	return ChartSample{X: x, First: open, Last: close, High: high, Low: low, HasGood: true}
}

// ChartBad returns a sample of bad quality without a value at x
func ChartBad(x float64) ChartSample {
	return ChartSample{X: x, HasBad: true}
}

// ChartGap returns a break in the data at x
func ChartGap(x float64) ChartSample {
	return ChartSample{X: x}
}

// HasValue reports whether the values of the sample are meaningful
func (s ChartSample) HasValue() bool { return s.HasGood }

// IsGap reports whether the sample is a break in the data
func (s ChartSample) IsGap() bool { return !s.HasGood && !s.HasBad }

// ChartSource gives the samples of a series along a numeric or a time axis
// (seconds, see ChartTime).
//
// GetData is called when the chart is painted, for the visible range [from,
// to]; group is the part of the axis one pixel takes: the source returns
// its data merged into buckets of that size (ChartDownsample does it), so
// about a sample per pixel is drawn whatever the amount of data. The
// samples are sorted by X; the nearest one on each side outside the range
// lets the line go on to the edges.
//
// GetData runs on the UI thread and must be quick. A source that loads its
// data in the background returns what it has and asks for a repaint
// (Widget.RequestRepaint) when more comes.
type ChartSource interface {
	GetData(from, to, group float64) []ChartSample
}

// ChartSourceExtent is implemented by a source that knows the range of X of
// all its data: an axis that fits the data uses it
type ChartSourceExtent interface {
	Extent() (min, max float64, ok bool)
}

// ChartSourceVersion is implemented by a source whose data changes: the
// version grows with every change, so the chart knows what it drew is old
type ChartSourceVersion interface {
	Version() uint64
}

// ChartSourceFunc is a ChartSource of a function
type ChartSourceFunc func(from, to, group float64) []ChartSample

// GetData implements ChartSource
func (f ChartSourceFunc) GetData(from, to, group float64) []ChartSample { return f(from, to, group) }

// ChartMemorySource is a ChartSource over samples in memory, sorted by X. It
// is safe to add samples from another goroutine: the chart that shows it
// repaints by itself (see SetMaxPoints for a live feed).
type ChartMemorySource struct {
	mtx       sync.RWMutex
	samples   []ChartSample
	maxPoints int
	version   uint64
	// The widgets to repaint on a change, see watch
	watchers []func()

	// cache is the result of the last GetData: a repaint for the pointer
	// asks for the same again
	cacheMtx                    sync.Mutex
	cacheFrom, cacheTo, cacheGr float64
	cacheVersion                uint64
	cache                       []ChartSample
	cacheValid                  bool
	// levels are the samples merged by 4, by 16, by 64...: a range of
	// millions of samples is merged from the level a few samples per
	// bucket, not from all of them (see pyramidMin)
	levels []chartLevel
}

// chartLevel is a level of the pyramid: the samples of the level under it
// merged by 4, done up to the source sample consumed
type chartLevel struct {
	samples []ChartSample
	// consumed: the samples of the level below merged up to the start of
	// the last group; outLen: the samples of this level made from them
	consumed, outLen int
}

// pyramidMin is the number of samples from which a source builds its
// pyramid of merged levels
const pyramidMin = 65536

// NewChartMemorySource makes an empty source
func NewChartMemorySource() *ChartMemorySource {
	return &ChartMemorySource{}
}

// SetSamples replaces the data; the samples must be sorted by X
func (s *ChartMemorySource) SetSamples(samples []ChartSample) {
	s.mtx.Lock()
	s.samples = append(s.samples[:0:0], samples...)
	s.levels = nil
	s.trimLocked()
	s.version++
	s.mtx.Unlock()
	s.changed()
}

// Append adds samples after the last one: their X must not be below its X.
// A sample out of order is put in its place, slower.
func (s *ChartMemorySource) Append(samples ...ChartSample) {
	s.mtx.Lock()
	for _, p := range samples {
		n := len(s.samples)
		if n == 0 || p.X >= s.samples[n-1].X {
			s.samples = append(s.samples, p)
			continue
		}
		i := sort.Search(n, func(i int) bool { return s.samples[i].X > p.X })
		s.samples = append(s.samples, ChartSample{})
		copy(s.samples[i+1:], s.samples[i:])
		s.samples[i] = p
		s.levels = nil
	}
	s.trimLocked()
	s.version++
	s.mtx.Unlock()
	s.changed()
}

// AppendValue adds a good sample of the value at x
func (s *ChartMemorySource) AppendValue(x, v float64) {
	s.Append(ChartValue(x, v))
}

// Clear removes all the samples
func (s *ChartMemorySource) Clear() {
	s.mtx.Lock()
	s.samples = s.samples[:0]
	s.levels = nil
	s.version++
	s.mtx.Unlock()
	s.changed()
}

// SetMaxPoints keeps at most n samples, the last ones: older samples are
// dropped as new ones come. 0 keeps all.
func (s *ChartMemorySource) SetMaxPoints(n int) {
	s.mtx.Lock()
	s.maxPoints = max(n, 0)
	s.trimLocked()
	s.version++
	s.mtx.Unlock()
	s.changed()
}

// trimLocked drops the oldest samples over maxPoints. The slice is only
// moved forward: the next append that runs out of its array copies the
// samples into a new one, so a live feed costs no copying per sample.
func (s *ChartMemorySource) trimLocked() {
	if s.maxPoints <= 0 || len(s.samples) <= s.maxPoints {
		return
	}
	s.samples = s.samples[len(s.samples)-s.maxPoints:]
	s.levels = nil
}

// Len returns the number of samples
func (s *ChartMemorySource) Len() int {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return len(s.samples)
}

// Samples returns a copy of the samples
func (s *ChartMemorySource) Samples() []ChartSample {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return append([]ChartSample(nil), s.samples...)
}

// Last returns the last sample; ok is false when there is none
func (s *ChartMemorySource) Last() (ChartSample, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	if len(s.samples) == 0 {
		return ChartSample{}, false
	}
	return s.samples[len(s.samples)-1], true
}

// GetData implements ChartSource. The result of a call is kept: the same
// range of the same data is not merged again. The slice returned is shared:
// it is not to be changed.
func (s *ChartMemorySource) GetData(from, to, group float64) []ChartSample {
	s.mtx.RLock()
	version := s.version
	s.mtx.RUnlock()
	s.cacheMtx.Lock()
	if s.cacheValid && s.cacheVersion == version && s.cacheFrom == from && s.cacheTo == to && s.cacheGr == group {
		res := s.cache
		s.cacheMtx.Unlock()
		return res
	}
	s.cacheMtx.Unlock()

	s.mtx.Lock()
	res := s.getDataLocked(from, to, group)
	version = s.version
	s.mtx.Unlock()

	s.cacheMtx.Lock()
	s.cache, s.cacheValid = res, true
	s.cacheFrom, s.cacheTo, s.cacheGr, s.cacheVersion = from, to, group, version
	s.cacheMtx.Unlock()
	return res
}

// getDataLocked merges the range from the coarsest level of the pyramid
// that has eight samples or more in a bucket: the merged samples carry the
// first, the last, the high and the low of theirs, so the result is the
// same but for the edges of the buckets, a fraction of a pixel
func (s *ChartMemorySource) getDataLocked(from, to, group float64) []ChartSample {
	src := s.samples
	if group > 0 && len(src) >= pyramidMin && to > from {
		i0 := sort.Search(len(src), func(i int) bool { return src[i].X >= from })
		i1 := sort.Search(len(src), func(i int) bool { return src[i].X > to })
		perBucket := float64(i1-i0) / ((to - from) / group)
		if perBucket >= 32 {
			s.buildLevels()
			for k := len(s.levels) - 1; k >= 0; k-- {
				if math.Pow(4, float64(k+1))*8 <= perBucket {
					src = s.levels[k].samples
					break
				}
			}
		}
	}
	return ChartDownsample(src, from, to, group)
}

// buildLevels brings the pyramid up to the samples: the levels exist while
// the samples are only appended, and grow by the new ones
func (s *ChartMemorySource) buildLevels() {
	below := s.samples
	for k := 0; ; k++ {
		if len(below) < 64 {
			s.levels = s.levels[:min(k, len(s.levels))]
			return
		}
		if k == len(s.levels) {
			s.levels = append(s.levels, chartLevel{})
		}
		lv := &s.levels[k]
		// Done again from the start of the last group, which may have been
		// partial
		lv.samples, lv.consumed, lv.outLen = mergeByFour(below, lv.samples, lv.consumed, lv.outLen)
		below = lv.samples
	}
}

// mergeByFour merges the samples of src from consumed on by four into out
// (cut to outLen, the samples made before consumed), keeping the gaps apart.
// It returns out, and where its last group starts in src and in out: that
// group, partial or not, is merged again next time.
func mergeByFour(src, out []ChartSample, consumed, outLen int) ([]ChartSample, int, int) {
	out = out[:outLen]
	i := consumed
	for i < len(src) {
		start := len(out)
		end := min(i+4, len(src))
		var m ChartSample
		has := false
		for j := i; j < end; j++ {
			p := src[j]
			if p.IsGap() {
				if has {
					out = append(out, m)
					has = false
				}
				out = append(out, p)
				continue
			}
			if !has {
				m, has = p, true
				continue
			}
			m.HasBad = m.HasBad || p.HasBad
			switch {
			case !p.HasGood:
			case !m.HasGood:
				m.First, m.Last, m.High, m.Low, m.HasGood = p.First, p.Last, p.High, p.Low, true
			default:
				m.Last = p.Last
				m.High = math.Max(m.High, p.High)
				m.Low = math.Min(m.Low, p.Low)
			}
		}
		if has {
			out = append(out, m)
		}
		if end == len(src) {
			// The last group: done again when more samples come
			return out, i, start
		}
		i = end
	}
	return out, i, len(out)
}

// Extent implements ChartSourceExtent
func (s *ChartMemorySource) Extent() (float64, float64, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	if len(s.samples) == 0 {
		return 0, 0, false
	}
	return s.samples[0].X, s.samples[len(s.samples)-1].X, true
}

// Version implements ChartSourceVersion
func (s *ChartMemorySource) Version() uint64 {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.version
}

// watch makes the source call f after every change; the chart repaints so
func (s *ChartMemorySource) watch(f func()) {
	s.mtx.Lock()
	s.watchers = append(s.watchers, f)
	s.mtx.Unlock()
}

func (s *ChartMemorySource) changed() {
	s.mtx.RLock()
	watchers := s.watchers
	s.mtx.RUnlock()
	for _, f := range watchers {
		f()
	}
}

// ChartDownsample returns the samples of the sorted slice within [from,
// to], with the nearest one on each side, merged into buckets of group: as
// TimeChartDownsample does for time. First and Last of a bucket come from
// its first and last good sample, High and Low are the extremes of the good
// ones; HasGood is set when the bucket has a good sample and HasBad when it
// has a bad one, so a short outage stays visible however far the chart is
// zoomed out. Gaps are never merged: each closes the bucket before it. The
// buckets are aligned to multiples of group, so panning does not regroup
// the data. With group 0 or below the samples are returned as they are.
func ChartDownsample(samples []ChartSample, from, to, group float64) []ChartSample {
	if to < from || len(samples) == 0 {
		return nil
	}
	i0 := sort.Search(len(samples), func(i int) bool { return samples[i].X >= from })
	i1 := sort.Search(len(samples), func(i int) bool { return samples[i].X > to })
	if i0 > 0 {
		i0--
	}
	if i1 < len(samples) {
		i1++
	}
	src := samples[i0:i1]
	if !(group > 0) || len(src) <= 2 {
		return append([]ChartSample(nil), src...)
	}
	// Few enough already: no merging
	if float64(len(src)) <= (to-from)/group+3 && sortedStepAtLeast(src, group) {
		return append([]ChartSample(nil), src...)
	}
	agg := NewChartAggregator(group, min(len(src), int((to-from)/group)+3))
	for _, p := range src {
		agg.Add(p)
	}
	return agg.Samples()
}

// sortedStepAtLeast reports whether the samples are at least step apart
func sortedStepAtLeast(src []ChartSample, step float64) bool {
	for i := 1; i < len(src); i++ {
		if src[i].X-src[i-1].X < step {
			return false
		}
	}
	return true
}

// ChartAggregator merges sorted samples into buckets of a size one by one,
// as ChartDownsample does: a source with many samples feeds them from its
// storage without making a slice of them first. It is the aggregation a
// server does before sending the data of a range to the chart.
type ChartAggregator struct {
	group    float64
	res      []ChartSample
	bucket   float64
	inBucket bool
}

// NewChartAggregator makes an aggregator of buckets of the group; capacity
// is a hint of how many buckets there will be
func NewChartAggregator(group float64, capacity int) *ChartAggregator {
	if !(group > 0) {
		group = 0
	}
	return &ChartAggregator{group: group, res: make([]ChartSample, 0, max(capacity, 0))}
}

// Add adds the next sample; the samples must come sorted by X
func (a *ChartAggregator) Add(p ChartSample) {
	if a.group == 0 {
		a.res = append(a.res, p)
		return
	}
	b := math.Floor(p.X / a.group)
	if p.IsGap() {
		a.res = append(a.res, ChartGap(b*a.group))
		a.inBucket = false
		return
	}
	good := p.HasValue()
	if a.inBucket && b == a.bucket {
		last := &a.res[len(a.res)-1]
		last.HasBad = last.HasBad || p.HasBad
		switch {
		case !good:
		case !last.HasGood:
			last.First, last.Last, last.High, last.Low = p.First, p.Last, p.High, p.Low
			last.HasGood = true
		default:
			last.Last = p.Last
			last.High = math.Max(last.High, p.High)
			last.Low = math.Min(last.Low, p.Low)
		}
		return
	}
	a.bucket = b
	a.inBucket = true
	q := ChartSample{X: b * a.group, HasBad: p.HasBad}
	if good {
		q.First, q.Last, q.High, q.Low = p.First, p.Last, p.High, p.Low
		q.HasGood = true
	}
	a.res = append(a.res, q)
}

// Samples returns the merged samples
func (a *ChartAggregator) Samples() []ChartSample {
	return a.res
}

// ChartLTTB thins the points down to n by Largest-Triangle-Three-Buckets:
// the points that keep the shape of a smooth line, unlike the min/max of
// ChartDownsample, which keeps every peak. The first and the last points
// are kept; fewer than n points are returned as they are.
func ChartLTTB(points []ChartPoint, n int) []ChartPoint {
	if n >= len(points) || n < 3 {
		return append([]ChartPoint(nil), points...)
	}
	res := make([]ChartPoint, 0, n)
	res = append(res, points[0])
	every := float64(len(points)-2) / float64(n-2)
	a := 0
	for i := 0; i < n-2; i++ {
		// The average of the next bucket
		nextStart := int(math.Floor(float64(i+1)*every)) + 1
		nextEnd := min(int(math.Floor(float64(i+2)*every))+1, len(points))
		var avgX, avgY float64
		for j := nextStart; j < nextEnd; j++ {
			avgX += points[j].X
			avgY += points[j].Y
		}
		if cnt := float64(nextEnd - nextStart); cnt > 0 {
			avgX /= cnt
			avgY /= cnt
		}
		// The point of this bucket with the largest triangle
		start := int(math.Floor(float64(i)*every)) + 1
		end := int(math.Floor(float64(i+1)*every)) + 1
		best, bestArea := start, -1.0
		for j := start; j < end; j++ {
			area := math.Abs((points[a].X-avgX)*(points[j].Y-points[a].Y) - (points[a].X-points[j].X)*(avgY-points[a].Y))
			if area > bestArea {
				best, bestArea = j, area
			}
		}
		res = append(res, points[best])
		a = best
	}
	return append(res, points[len(points)-1])
}
