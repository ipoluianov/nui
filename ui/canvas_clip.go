package ui

import (
	"image"
	"math"
)

// The clip by a shape: ClipPath keeps the pixels of the area it may change
// and the coverage of the shape there; the drawings go on the canvas as
// usual, and Restore puts back what was drawn outside the shape. So every
// drawing is clipped - paths, text, images, 3D, blur - and what reads the
// canvas (Blur, PixelAt) reads what is there.

// clipLayer is a clip by a shape not restored yet
type clipLayer struct {
	rgba  *image.RGBA
	rect  image.Rectangle
	mask  []uint8 // the coverage of the shape, a byte a pixel of rect
	saved []uint8 // the pixels of rect at the time of the clip
}

// grow returns b of the length n, its memory reused if it is enough
func grow(b []uint8, n int) []uint8 {
	if cap(b) < n {
		return make([]uint8, n)
	}
	return b[:n]
}

// ClipPath clips the drawings that follow to the inside of the path, under
// the transform, with antialiased edges, until the Restore of the Save
// before it: an image in a circle, the content of a card with rounded
// corners. Clips nest: each one is within the ones before.
//
//	cnv.Save()
//	cnv.ClipPath(ui.NewPath().RoundedRect(0, 0, w, h, 12))
//	cnv.DrawImageRect(0, 0, w, h, photo, photo.Bounds(), true)
//	cnv.Restore() // the corners of the photo are cut off here
//
// What is drawn outside the shape is taken back at Restore, so call it
// between Save and Restore; PixelAt reads the pixels as drawn until then.
func (c *Canvas) ClipPath(p *Path) {
	if c.state.hasXf {
		c.withPathXf(p, c.ClipPath)
		return
	}
	minX, minY, maxX, maxY, ok := p.Bounds()
	ox, oy, s := float64(c.state.translateX), float64(c.state.translateY), c.scale
	r := image.Rectangle{}
	if ok {
		r = image.Rect(int(math.Floor(minX*s+ox))-1, int(math.Floor(minY*s+oy))-1,
			int(math.Ceil(maxX*s+ox))+1, int(math.Ceil(maxY*s+oy))+1)
	}
	// The clip of the whole frame (see shapeClip): drawn within a part of
	// it, the mask is set within the part, and the pixels outside are left
	r = r.Intersect(c.shapeClip()).Intersect(c.rgba.Rect)
	if c.rec != nil {
		c.rec.clipPath(c, p)
	}
	c.state.clipX, c.state.clipY, c.state.clipW, c.state.clipH = r.Min.X, r.Min.Y, r.Dx(), r.Dy()
	if r.Empty() {
		// Nothing is drawn, not even by a widget given its own clip
		c.clips = append(c.clips, clipLayer{})
		c.state.clipDepth = len(c.clips)
		c.state.clipBound = image.Rectangle{}
		return
	}
	n := r.Dx() * r.Dy()
	// The buffers of a clip ended before at this depth reused
	var l clipLayer
	if len(c.clips) < cap(c.clips) {
		l = c.clips[: len(c.clips)+1 : len(c.clips)+1][len(c.clips)]
	}
	l = clipLayer{rgba: c.rgba, rect: r, mask: grow(l.mask, n), saved: grow(l.saved, n*4)}
	clear(l.mask)
	w := r.Dx()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		off := c.rgba.PixOffset(r.Min.X, y)
		copy(l.saved[(y-r.Min.Y)*w*4:(y-r.Min.Y+1)*w*4], c.rgba.Pix[off:off+w*4])
	}
	c.fillPathWith(p, func(_ int, cover []uint8, px, py int) {
		copy(l.mask[(py-r.Min.Y)*w+px-r.Min.X:], cover)
	})
	c.clips = append(c.clips, l)
	c.state.clipDepth = len(c.clips)
	c.state.clipBound = r
}

// endClips puts back the pixels outside the shapes of the clips past the
// first depth ones, the last first
func (c *Canvas) endClips(depth int) {
	for len(c.clips) > depth {
		l := c.clips[len(c.clips)-1]
		c.clips = c.clips[:len(c.clips)-1]
		w := l.rect.Dx()
		for y := l.rect.Min.Y; y < l.rect.Max.Y; y++ {
			off := l.rgba.PixOffset(l.rect.Min.X, y)
			cur := l.rgba.Pix[off : off+w*4 : off+w*4]
			mask := l.mask[(y-l.rect.Min.Y)*w : (y-l.rect.Min.Y+1)*w]
			old := l.saved[(y-l.rect.Min.Y)*w*4 : (y-l.rect.Min.Y+1)*w*4]
			for x, m := range mask {
				switch m {
				case 255:
				case 0:
					copy(cur[x*4:x*4+4], old[x*4:x*4+4])
				default:
					k, ik := uint32(m), uint32(255-m)
					for i := x * 4; i < x*4+4; i++ {
						v := uint32(cur[i])*k + uint32(old[i])*ik + 128
						cur[i] = uint8((v + v>>8) >> 8)
					}
				}
			}
		}
	}
}
