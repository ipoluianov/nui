package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
)

// formRepaint is what the next frame of a form paints again. The window's
// buffer keeps the last frame, so a frame may paint only the rectangles of
// the widgets that asked for it (Widget.Update) over it; the rest of the
// window is left as it is.
//
// The whole window is painted when the form asked for it (Form.Update),
// when the widgets were laid out, when the buffer is not the one of the
// last frame (a new size, another screen) and when nothing asked for the
// frame: the system repaints the window by itself, e.g. uncovered.
type formRepaint struct {
	all   bool
	dirty []image.Rectangle // in the form's client coordinates
	// moved are the parents of the widgets moved or resized: repainted
	// where they are when the frame is painted, the layout done
	moved []*Widget

	// The buffer of the last frame, as the window gave it
	pix    *uint8
	rect   image.Rectangle
	stride int
	scale  float64
	// bg is the color of the window's background, under the widgets
	bg color.RGBA

	// The frames painted, and of them those painted in part
	frames, partialFrames int

	// The widgets asking for a repaint off the UI thread, see RequestRepaint
	mu     sync.Mutex
	posted []*Widget
}

// maxDirtyRects is how many rectangles a frame paints apart at most: each
// is a pass over the widgets
const maxDirtyRects = 6

// add adds the rectangle to those to paint: joined with those it touches,
// as is the pair that grows the least when there are too many
func (r *formRepaint) add(rect image.Rectangle) {
	if rect.Empty() {
		return
	}
	for i := 0; i < len(r.dirty); {
		// Touching, or a few pixels apart: one pass paints both
		if near := r.dirty[i].Inset(-4); near.Overlaps(rect) {
			rect = rect.Union(r.dirty[i])
			r.dirty = slices.Delete(r.dirty, i, i+1)
			i = 0
			continue
		}
		i++
	}
	r.dirty = append(r.dirty, rect)
	for len(r.dirty) > maxDirtyRects {
		bi, bj, best := 0, 1, -1
		for i := range r.dirty {
			for j := i + 1; j < len(r.dirty); j++ {
				grow := area(r.dirty[i].Union(r.dirty[j])) - area(r.dirty[i]) - area(r.dirty[j])
				if best < 0 || grow < best {
					bi, bj, best = i, j, grow
				}
			}
		}
		r.dirty[bi] = r.dirty[bi].Union(r.dirty[bj])
		r.dirty = slices.Delete(r.dirty, bj, bj+1)
	}
}

func area(r image.Rectangle) int { return r.Dx() * r.Dy() }

// repaintCheck returns the environment variable NUI_CHECK_REPAINT: with it
// each frame painted in part is compared with the whole window painted
// again, and the difference is reported - a widget that changed without
// asking for a repaint of its rectangle. "1" writes it to the standard
// error, "panic" panics (for the tests). With NUI_CHECK_REPAINT_DIR the
// two pictures are saved there too.
func repaintCheck() string { return os.Getenv("NUI_CHECK_REPAINT") }

// updateRect repaints the rectangle of the form's client area with the next
// frame
func (c *Form) updateRect(r image.Rectangle) {
	if c == nil || c.wnd == nil || r.Empty() {
		return
	}
	if !c.repaint.all {
		c.repaint.add(r)
	}
	c.requestFrame()
}

// RepaintStats returns how many frames the form painted, and how many of
// them only in part: the rectangles of the widgets that changed
func (c *Form) RepaintStats() (frames, partial int) {
	return c.repaint.frames, c.repaint.partialFrames
}

// setWindowBackground gives the window the background of the theme
func (c *Form) setWindowBackground() {
	c.repaint.bg = colorToRGBA(currentPalette.Window)
	c.wnd.SetBackgroundColor(c.repaint.bg)
}

// paintFrame paints the window's buffer: what changed since the last frame
// or all of it
func (c *Form) paintFrame(rgba *image.RGBA, scale float64) {
	r := &c.repaint
	for _, w := range r.moved {
		if !r.all {
			r.add(w.windowRect())
		}
	}
	r.moved = r.moved[:0]
	partial := !r.all && len(r.dirty) > 0 && len(rgba.Pix) > 0 && &rgba.Pix[0] == r.pix &&
		rgba.Rect == r.rect && rgba.Stride == r.stride && scale == r.scale
	parts := []image.Rectangle{rgba.Rect}
	if partial {
		parts = parts[:0]
		for _, d := range r.dirty {
			// The pixels the rectangle touches, and one more around: what
			// a widget there paints is all within them
			p := image.Rect(
				int(math.Floor(float64(d.Min.X)*scale))-1, int(math.Floor(float64(d.Min.Y)*scale))-1,
				int(math.Ceil(float64(d.Max.X)*scale))+1, int(math.Ceil(float64(d.Max.Y)*scale))+1,
			).Intersect(rgba.Rect)
			if !p.Empty() {
				parts = append(parts, p)
			}
		}
	}
	// Reset before painting: what asks for a repaint while the frame is
	// painted is in the next one
	r.all, r.dirty = false, r.dirty[:0]
	r.pix, r.rect, r.stride, r.scale = nil, rgba.Rect, rgba.Stride, scale
	if len(rgba.Pix) > 0 {
		r.pix = &rgba.Pix[0]
	}
	if len(parts) == 0 || parts[0].Empty() {
		return
	}
	r.frames++
	if partial {
		r.partialFrames++
	}
	for _, p := range parts {
		fillPixels(rgba, p, r.bg)
		c.paintClipped(rgba, scale, p)
	}
	if partial && repaintCheck() != "" {
		c.checkRepaint(rgba, scale, parts)
	}
}

// fillPixels fills the rectangle of the image with the color
func fillPixels(img *image.RGBA, r image.Rectangle, col color.RGBA) {
	r = r.Intersect(img.Rect)
	if r.Empty() {
		return
	}
	first := img.Pix[img.PixOffset(r.Min.X, r.Min.Y):][:r.Dx()*4]
	first[0], first[1], first[2], first[3] = col.R, col.G, col.B, col.A
	for n := 4; n < len(first); n *= 2 {
		copy(first[n:], first[:n])
	}
	for y := r.Min.Y + 1; y < r.Max.Y; y++ {
		copy(img.Pix[img.PixOffset(r.Min.X, y):][:len(first)], first)
	}
}

// checkRepaint paints the whole window again and reports the pixels that
// differ from the frame painted in parts (see repaintCheck)
func (c *Form) checkRepaint(frame *image.RGBA, scale float64, parts []image.Rectangle) {
	whole := image.NewRGBA(frame.Rect)
	fillPixels(whole, whole.Rect, c.repaint.bg)
	c.paintClipped(whole, scale, whole.Rect)
	var diff image.Rectangle
	n := 0
	for y := frame.Rect.Min.Y; y < frame.Rect.Max.Y; y++ {
		a := frame.Pix[frame.PixOffset(frame.Rect.Min.X, y):][:frame.Rect.Dx()*4]
		b := whole.Pix[whole.PixOffset(whole.Rect.Min.X, y):][:len(a)]
		for x := 0; x < len(a); x += 4 {
			if a[x] != b[x] || a[x+1] != b[x+1] || a[x+2] != b[x+2] || a[x+3] != b[x+3] {
				n++
				diff = diff.Union(image.Rect(frame.Rect.Min.X+x/4, y, frame.Rect.Min.X+x/4+1, y+1))
			}
		}
	}
	if n == 0 {
		return
	}
	// In logical pixels, with the widgets there
	logical := image.Rect(int(float64(diff.Min.X)/scale), int(float64(diff.Min.Y)/scale),
		int(math.Ceil(float64(diff.Max.X)/scale)), int(math.Ceil(float64(diff.Max.Y)/scale)))
	if c.repaintPending(logical) {
		// Changed since the frame (data from another goroutine, a widget
		// that asked for its repaint while painted): the next frame paints
		// it again
		return
	}
	msg := fmt.Sprintf("nui: repaint check: form %q: %d pixels differ from a whole repaint in %v (logical %v), the frame painted %v; widgets there: %s",
		c.title, n, diff, logical, parts, strings.Join(c.widgetsIn(logical), ", "))
	// A widget whose paint changes what it paints next is not to be found
	// by its rectangle: told apart
	again := image.NewRGBA(frame.Rect)
	fillPixels(again, again.Rect, c.repaint.bg)
	c.paintClipped(again, scale, again.Rect)
	if !bytes.Equal(again.Pix, whole.Pix) {
		msg += "; a second whole repaint differs from the first: a paint changes what the next one paints"
	}
	// The two pictures, to look at
	if dir := os.Getenv("NUI_CHECK_REPAINT_DIR"); dir != "" {
		for name, img := range map[string]*image.RGBA{"frame": frame, "whole": whole} {
			if f, err := os.Create(filepath.Join(dir, fmt.Sprintf("repaint-%d-%s.png", c.repaint.frames, name))); err == nil {
				png.Encode(f, img)
				f.Close()
			}
		}
	}
	if repaintCheck() == "panic" {
		panic(msg)
	}
	fmt.Fprintln(os.Stderr, msg)
}

// repaintPending reports whether the next frame repaints the rectangle of
// the client area: it was asked for since the frame, on the UI thread or
// off it (RequestRepaint)
func (c *Form) repaintPending(r image.Rectangle) bool {
	if c.repaint.all || c.layoutPending {
		return true
	}
	pending := slices.Clone(c.repaint.dirty)
	c.repaint.mu.Lock()
	for _, w := range c.repaint.posted {
		x, y := w.RectClientAreaOnWindow()
		pending = append(pending, image.Rect(x, y, x+w.w, y+w.h))
	}
	c.repaint.mu.Unlock()
	for _, p := range pending {
		if r.In(p) {
			return true
		}
	}
	return false
}

// widgetsIn names the deepest widgets over the rectangle of the client area
func (c *Form) widgetsIn(r image.Rectangle) []string {
	var names []string
	var walk func(w Widgeter, x, y int)
	walk = func(w Widgeter, x, y int) {
		wr := image.Rect(x, y, x+w.Width(), y+w.Height())
		if !w.IsVisible() || !wr.Overlaps(r) {
			return
		}
		deeper := len(names)
		for _, child := range w.Widgets() {
			walk(child, x+child.X()-w.ScrollX(), y+child.Y()-w.ScrollY())
		}
		if len(names) == deeper {
			name := w.TypeName()
			if w.Name() != "" {
				name += " " + w.Name()
			}
			names = append(names, fmt.Sprintf("%s %v", name, wr))
		}
	}
	if c.menuBar != nil {
		walk(c.menuBar, c.menuBar.X(), c.menuBar.Y())
	}
	walk(c.topWidget, c.topWidget.X(), c.topWidget.Y())
	sort.Strings(names)
	return names
}
