package ui

import "math"

// strokeSubpathCulled strokes the subpath as strokeSubpath does; a dashed
// one that goes out of cull is cut to the pieces in it first, each dashed
// from where it is along the subpath. Otherwise a line of a drawing zoomed
// far in would be dashed along all its length, millions of dashes out of
// sight.
func strokeSubpathCulled(pb *polyBuilder, pts []PointF, closed bool, st StrokeStyle, cull [4]float64) {
	if len(st.Dash) == 0 || len(pts) < 2 {
		strokeSubpath(pb, pts, closed, st)
		return
	}
	inside := true
	for _, pt := range pts {
		if pt.X < cull[0] || pt.Y < cull[1] || pt.X > cull[2] || pt.Y > cull[3] {
			inside = false
			break
		}
	}
	if inside {
		strokeSubpath(pb, pts, closed, st)
		return
	}
	n := len(pts) - 1
	if closed {
		n++
	}
	run := pb.run[:0]
	runStart := 0.0 // the length along the subpath where the run starts
	along := 0.0
	flush := func() {
		if len(run) > 1 {
			rs := st
			rs.DashOffset = st.DashOffset + runStart
			strokeSubpath(pb, run, false, rs)
		}
		run = run[:0]
	}
	for i := 0; i < n; i++ {
		a, b := pts[i], pts[(i+1)%len(pts)]
		length := math.Hypot(b.X-a.X, b.Y-a.Y)
		t0, t1, ok := clipSegment(a, b, cull)
		if !ok {
			flush()
			along += length
			continue
		}
		p0 := PointF{a.X + (b.X-a.X)*t0, a.Y + (b.Y-a.Y)*t0}
		p1 := PointF{a.X + (b.X-a.X)*t1, a.Y + (b.Y-a.Y)*t1}
		if t0 > 0 || len(run) == 0 {
			flush()
			runStart = along + length*t0
			run = append(run, p0)
		}
		run = append(run, p1)
		if t1 < 1 {
			flush()
		}
		along += length
	}
	flush()
	pb.run = run
}

// clipSegment returns the part of the segment a-b in the rectangle (minX,
// minY, maxX, maxY), as the parameters t0 <= t1 of a + (b-a)*t; false if
// none (Liang-Barsky)
func clipSegment(a, b PointF, r [4]float64) (t0, t1 float64, ok bool) {
	t0, t1 = 0, 1
	dx, dy := b.X-a.X, b.Y-a.Y
	for _, e := range [4][2]float64{{-dx, a.X - r[0]}, {dx, r[2] - a.X}, {-dy, a.Y - r[1]}, {dy, r[3] - a.Y}} {
		p, q := e[0], e[1]
		if p == 0 {
			if q < 0 {
				return 0, 0, false
			}
			continue
		}
		t := q / p
		if p < 0 {
			if t > t1 {
				return 0, 0, false
			}
			t0 = max(t0, t)
		} else {
			if t < t0 {
				return 0, 0, false
			}
			t1 = min(t1, t)
		}
	}
	return t0, t1, true
}
