package ui

import (
	"image"
	"image/color"
	"math"
)

// evenOddRasterizer computes how much of every pixel a polygon covers by
// FillRuleEvenOdd or, with nonZero, by FillRuleNonZero. Each line adds its
// signed area to the cells it crosses; the sum along a row is then how many
// times the path winds around the pixel, of which the even-odd rule takes
// the fractional parity and the nonzero one the magnitude up to 1.
//
// It is in float32 for both rules: the fixed-point math x/image/vector uses
// for the masks up to 512 pixels gives the pixels near a sharp vertex too
// much coverage.
type evenOddRasterizer struct {
	w, h    int
	nonZero bool
	acc     []float32 // (w + 2) per row: a line may reach past the last pixel
	// span is the first and the last cell of each row a line added to;
	// before it the sum is 0, after it the sum does not change
	span       [][2]int32
	penX, penY float32
	startX     float32
	startY     float32
}

func (r *evenOddRasterizer) reset(w, h int, nonZero bool) *evenOddRasterizer {
	r.w, r.h = w, h
	r.nonZero = nonZero
	n := (w + 2) * h
	if cap(r.acc) < n {
		r.acc = make([]float32, n)
	}
	r.acc = r.acc[:n]
	clear(r.acc)
	if cap(r.span) < h {
		r.span = make([][2]int32, h)
	}
	r.span = r.span[:h]
	for i := range r.span {
		r.span[i] = [2]int32{math.MaxInt32, -1}
	}
	return r
}

// touch widens the span of the row y to the cells lo..hi
func (r *evenOddRasterizer) touch(y, lo, hi int) {
	sp := &r.span[y]
	sp[0] = min(sp[0], int32(lo))
	sp[1] = max(sp[1], int32(hi))
}

func (r *evenOddRasterizer) MoveTo(x, y float32) {
	r.penX, r.penY = x, y
	r.startX, r.startY = x, y
}

func (r *evenOddRasterizer) ClosePath() {
	r.LineTo(r.startX, r.startY)
}

// LineTo adds the line from the pen to (x, y): for every row it crosses,
// the part of its height in the row, split between the cells by the area
// on the right of the line
func (r *evenOddRasterizer) LineTo(bx, by float32) {
	ax, ay := r.penX, r.penY
	r.penX, r.penY = bx, by
	// A line of the same height covers nothing; one with a NaN or an
	// infinite end has no place in the rows (int(NaN) indexes out of them)
	if ay == by || ax-ax != 0 || ay-ay != 0 || bx-bx != 0 || by-by != 0 {
		return
	}
	dir := float32(1)
	if ay > by {
		dir, ax, ay, bx, by = -1, bx, by, ax, ay
	}
	fw := float32(r.w)
	dxdy := (bx - ax) / (by - ay)
	x := ax
	y0 := int(math.Floor(float64(ay)))
	if y0 < 0 {
		x -= ay * dxdy
		y0 = 0
	}
	yEnd := min(r.h, int(math.Ceil(float64(by))))
	stride := r.w + 2
	for y := y0; y < yEnd; y++ {
		row := r.acc[y*stride : (y+1)*stride]
		dy := min(float32(y+1), by) - max(float32(y), ay)
		xNext := x + dxdy*dy
		d := dy * dir
		x0, x1 := x, xNext
		if x1 < x0 {
			x0, x1 = x1, x0
		}
		// The part of the line left of the first pixel adds all of its
		// height to it, as the sum along the row carries it on; the part
		// past the last one is not seen. Only the rest is split between
		// the cells: clamping the ends would spread the whole height over
		// the part within the row and give the wrong area.
		if x0 < 0 && x1 > 0 {
			left := d * (-x0 / (x1 - x0))
			row[0] += left
			r.touch(y, 0, 0)
			d -= left
			x0 = 0
		}
		if x0 < fw && x1 > fw {
			d *= (fw - x0) / (x1 - x0)
			x1 = fw
		}
		x0 = max(0, min(x0, fw))
		x1 = max(0, min(x1, fw))
		x0i := int(x0)
		x0f := x0 - float32(x0i)
		x1i := int(math.Ceil(float64(x1)))
		if x1i <= x0i+1 {
			xmf := (x0+x1)/2 - float32(x0i)
			row[x0i] += d - d*xmf
			row[x0i+1] += d * xmf
			r.touch(y, x0i, x0i+1)
		} else {
			s := 1 / (x1 - x0)
			a0 := 0.5 * s * (1 - x0f) * (1 - x0f)
			x1f := x1 - float32(x1i) + 1
			am := 0.5 * s * x1f * x1f
			row[x0i] += d * a0
			if x1i == x0i+2 {
				row[x0i+1] += d * (1 - a0 - am)
			} else {
				a1 := s * (1.5 - x0f)
				row[x0i+1] += d * (a1 - a0)
				for xi := x0i + 2; xi < x1i-1; xi++ {
					row[xi] += d * s
				}
				a2 := a1 + float32(x1i-x0i-3)*s
				row[x1i-1] += d * (1 - a2 - am)
			}
			row[x1i] += d * am
			r.touch(y, x0i, x1i)
		}
		x = xNext
	}
}

// accumulate writes the coverage of every pixel to the mask, of the
// rasterizer's size
func (r *evenOddRasterizer) accumulate(mask *image.Alpha) {
	stride := r.w + 2
	for y := 0; y < r.h; y++ {
		row := r.acc[y*stride : y*stride+r.w]
		out := mask.Pix[y*mask.Stride : y*mask.Stride+r.w]
		out = out[:len(row)]
		// Before the span of the row the pixels are 0, after it the last
		// pixel of it again
		lo, hi := int(r.span[y][0]), min(int(r.span[y][1]), len(row)-1)
		if hi < lo {
			clear(out)
			continue
		}
		clear(out[:lo])
		var sum float32
		var last uint8
		for x := lo; x <= hi; {
			// Within a shape and outside it most cells add nothing: the
			// run of them repeats the pixel before it, written as a block
			if row[x] == 0 {
				end := x + 1
				for end <= hi && row[end] == 0 {
					end++
				}
				fillBytes(out[x:end], last)
				x = end
				continue
			}
			sum += row[x]
			last = r.coverage(sum)
			out[x] = last
			x++
		}
		fillBytes(out[hi+1:], last)
	}
}

// coverage is the mask value of a pixel whose winding sum is sum, without
// float64 math in the common case: it runs for many pixels of every fill
func (r *evenOddRasterizer) coverage(sum float32) uint8 {
	a := sum
	if a < 0 {
		a = -a
	}
	if r.nonZero {
		a = min(a, 1)
	} else {
		// The parity: 0 at an even winding, 1 at an odd one, the fraction
		// between them at the edges
		if a >= 2 {
			a -= 2 * float32(math.Floor(float64(a/2)))
		}
		if a > 1 {
			a = 2 - a
		}
	}
	return uint8(a*255 + 0.5)
}

// fillBytes sets every byte of b to v
func fillBytes(b []byte, v byte) {
	if len(b) == 0 {
		return
	}
	if v == 0 {
		clear(b)
		return
	}
	b[0] = v
	for n := 1; n < len(b); n *= 2 {
		copy(b[n:], b[:n])
	}
}

// strokePathXf strokes the path under the canvas' transform: the outline is
// made in the drawing coordinates, so the width, the joins and the dashes
// are transformed with the path, and then its polygons are transformed to
// the pixels
func (c *Canvas) strokePathXf(p *Path, style StrokeStyle, in pathInk, col color.Color, paint Paint) {
	m := c.pixelMatrix()
	k := m.meanScale()
	if k == 0 || math.IsNaN(k) || math.IsInf(k, 0) {
		return
	}
	if c.rec != nil {
		xf := c.state.xf
		c.withPathXf(p, func(tp *Path) {
			st := style
			if st.Width <= 0 {
				st.Width = 1
			}
			ks := xf.meanScale()
			st.Width *= ks
			if len(st.Dash) > 0 {
				st.Dash = append([]float64(nil), st.Dash...)
				for i := range st.Dash {
					st.Dash[i] *= ks
				}
				st.DashOffset *= ks
			}
			if paint != nil {
				c.rec.paintXf = &xf
				defer func() { c.rec.paintXf = nil }()
			}
			c.rec.strokePath(c, tp, st, col, paint)
		})
	}
	// Flattened at the density of the pixels: k of them per unit
	fp := flatPaths.Get().(*flatPath)
	defer flatPaths.Put(fp)
	st := strokeStyleInPixels(style, k)
	// From k per unit to the pixels
	pm := ScalingMatrix(1/k, 1/k).Then(m)
	// What is seen, in k per unit: the box of the clip taken back, with
	// the reach of the line around it
	cull, culled := [4]float64{}, false
	if inv, ok := pm.Invert(); ok {
		// The clip of the whole frame, see shapeClip
		clip := c.shapeClip().Intersect(c.rgba.Rect)
		cull = [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
		for _, q := range [4]PointF{{float64(clip.Min.X), float64(clip.Min.Y)}, {float64(clip.Max.X), float64(clip.Min.Y)},
			{float64(clip.Min.X), float64(clip.Max.Y)}, {float64(clip.Max.X), float64(clip.Max.Y)}} {
			q = inv.ApplyPoint(q)
			cull = [4]float64{min(cull[0], q.X), min(cull[1], q.Y), max(cull[2], q.X), max(cull[3], q.Y)}
		}
		// Two pixels taken back are at most this long
		px := 2 * math.Sqrt(inv.A*inv.A+inv.B*inv.B+inv.C*inv.C+inv.D*inv.D)
		margin := st.Width/2*st.MiterLimit + px
		cull = [4]float64{cull[0] - margin, cull[1] - margin, cull[2] + margin, cull[3] + margin}
		culled = true
	}
	fp.culling, fp.cull = culled, cull
	p.flatten(fp, 0, 0, k)

	pb := polyBuilders.Get().(*polyBuilder)
	defer polyBuilders.Put(pb)
	pb.reset()
	for _, sub := range fp.subs {
		if culled {
			strokeSubpathCulled(pb, fp.pts[sub.from:sub.to], sub.closed, st, cull)
		} else {
			strokeSubpath(pb, fp.pts[sub.from:sub.to], sub.closed, st)
		}
	}
	for i, pt := range pb.pts {
		pb.pts[i] = pm.ApplyPoint(pt)
	}
	c.rasterizePolys(pb, func(off int, cover []uint8, px, py int) { c.blendInk(off, cover, px, py, &in) })
}
