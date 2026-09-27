package ui

import (
	"math"
	"sort"
	"sync"
	"time"
)

// TimeChartMemorySource is a TimeChartDataSource over an in-memory slice of
// points sorted by DT. It is safe to add points from another goroutine.
type TimeChartMemorySource struct {
	mtx    sync.RWMutex
	points []TimeChartPoint
}

func NewTimeChartMemorySource() *TimeChartMemorySource {
	return &TimeChartMemorySource{}
}

// SetPoints replaces the data. The points must be sorted by DT.
func (c *TimeChartMemorySource) SetPoints(points []TimeChartPoint) {
	c.mtx.Lock()
	c.points = points
	c.mtx.Unlock()
}

// AddPoint appends a point. Its DT must not be earlier than the last one.
func (c *TimeChartMemorySource) AddPoint(p TimeChartPoint) {
	c.mtx.Lock()
	c.points = append(c.points, p)
	c.mtx.Unlock()
}

func (c *TimeChartMemorySource) GetData(from, to time.Time, groupDuration time.Duration) []TimeChartPoint {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return TimeChartDownsample(c.points, from, to, groupDuration)
}

// TimeChartDownsample returns the points of the sorted slice that fall into
// [from, to], plus the nearest point on each side, merged into buckets of
// groupDuration: First/Last of a bucket come from its first and last good
// point, High/Low are the extremes of the good points, HasGood is set when
// the bucket has any good point and HasBad when it has any bad one - so a
// short outage stays visible however far the chart is zoomed out. Gap points
// (neither flag) are never merged: each one is kept as a separate point and
// closes the current bucket, so a break stays visible at any zoom. Buckets
// are aligned to multiples of groupDuration since the Unix epoch, so panning
// does not change how points are grouped. It is the same aggregation a server should do before sending
// data to the chart.
func TimeChartDownsample(points []TimeChartPoint, from, to time.Time, groupDuration time.Duration) []TimeChartPoint {
	i0 := sort.Search(len(points), func(i int) bool { return !points[i].DT.Before(from) })
	i1 := sort.Search(len(points), func(i int) bool { return points[i].DT.After(to) })
	if i0 > 0 {
		i0--
	}
	if i1 < len(points) {
		i1++
	}
	src := points[i0:i1]

	if groupDuration <= 0 {
		res := make([]TimeChartPoint, len(src))
		copy(res, src)
		return res
	}

	agg := NewTimeChartAggregator(groupDuration, min(len(src), int(to.Sub(from)/groupDuration)+3))
	for _, p := range src {
		agg.Add(p)
	}
	return agg.Points()
}

// TimeChartAggregator merges sorted points into buckets of groupDuration one by
// one, the same way as TimeChartDownsample. A source with many points can feed
// them straight from its storage without building a slice of all of them first.
type TimeChartAggregator struct {
	group    int64
	res      []TimeChartPoint
	bucket   int64
	inBucket bool
}

// NewTimeChartAggregator creates an aggregator; capacity is a hint for the number of result points
func NewTimeChartAggregator(groupDuration time.Duration, capacity int) *TimeChartAggregator {
	return &TimeChartAggregator{
		group: max(int64(groupDuration), 1),
		res:   make([]TimeChartPoint, 0, max(capacity, 0)),
	}
}

// Add adds the next point; points must come sorted by DT
func (c *TimeChartAggregator) Add(p TimeChartPoint) {
	g := c.group
	ns := p.DT.UnixNano()
	b := ns / g
	if ns < 0 && ns%g != 0 {
		b--
	}
	if p.IsGap() {
		c.res = append(c.res, NewTimeChartGap(time.Unix(0, b*g)))
		c.inBucket = false
		return
	}
	good := p.HasValue()
	if c.inBucket && b == c.bucket {
		last := &c.res[len(c.res)-1]
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
	c.bucket = b
	c.inBucket = true
	q := TimeChartPoint{DT: time.Unix(0, b*g), HasBad: p.HasBad}
	if good {
		q.First, q.Last, q.High, q.Low = p.First, p.Last, p.High, p.Low
		q.HasGood = true
	}
	c.res = append(c.res, q)
}

// Points returns the aggregated points
func (c *TimeChartAggregator) Points() []TimeChartPoint {
	return c.res
}
