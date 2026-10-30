package ui

import (
	"image/color"
	"math"
	"sync"

	gtfont "github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
)

// The drawing of the glyphs of a TextLayout. Text that is not turned is
// drawn from masks of the glyphs, made once for a font, a size and a
// quarter of a pixel across, and kept: as quick as the text of DrawText,
// at fractional places. Turned or slanted text, text too large for masks
// and text recorded as SVG is filled as the outlines of its glyphs.

// shapedGlyphKey is a glyph of a font
type shapedGlyphKey struct {
	font *gtfont.Font
	gid  gtfont.GID
}

// glyphShape is the outline of a glyph in ems, y down
type glyphShape struct {
	ops []glyphOp
}

type glyphOp struct {
	kind byte // 'M', 'L', 'Q', 'C'
	p    [3]PointF
}

var glyphShapes sync.Map // shapedGlyphKey -> *glyphShape

// shapeOf returns the outline of the glyph of the face
func shapeOf(face *gtfont.Face, gid gtfont.GID) *glyphShape {
	key := shapedGlyphKey{face.Font, gid}
	if v, ok := glyphShapes.Load(key); ok {
		return v.(*glyphShape)
	}
	shapeMu.Lock()
	data := face.GlyphData(gid)
	shapeMu.Unlock()
	g := &glyphShape{}
	if o, ok := data.(gtfont.GlyphOutline); ok {
		k := 1 / float64(face.Upem())
		pt := func(p ot.SegmentPoint) PointF { return PointF{float64(p.X) * k, -float64(p.Y) * k} }
		for _, s := range o.Segments {
			op := glyphOp{}
			switch s.Op {
			case ot.SegmentOpMoveTo:
				op.kind = 'M'
			case ot.SegmentOpLineTo:
				op.kind = 'L'
			case ot.SegmentOpQuadTo:
				op.kind = 'Q'
			case ot.SegmentOpCubeTo:
				op.kind = 'C'
			}
			for i, a := range s.ArgsSlice() {
				op.p[i] = pt(a)
			}
			g.ops = append(g.ops, op)
		}
	}
	glyphShapes.Store(key, g)
	return g
}

// addTo adds the outline at the size to the path, its origin at (x, y),
// slanted as italic if it is so
func (g *glyphShape) addTo(p *Path, x, y, size float64, italic bool) {
	shear := 0.0
	if italic {
		shear = italicShear
	}
	at := func(q PointF) (float64, float64) {
		return x + (q.X-q.Y*shear)*size, y + q.Y*size
	}
	open := false
	for _, op := range g.ops {
		switch op.kind {
		case 'M':
			if open {
				p.Close()
			}
			p.MoveTo(at(op.p[0]))
			open = true
		case 'L':
			p.LineTo(at(op.p[0]))
		case 'Q':
			x1, y1 := at(op.p[0])
			x2, y2 := at(op.p[1])
			p.QuadTo(x1, y1, x2, y2)
		case 'C':
			x1, y1 := at(op.p[0])
			x2, y2 := at(op.p[1])
			x3, y3 := at(op.p[2])
			p.CubicTo(x1, y1, x2, y2, x3, y3)
		}
	}
	if open {
		p.Close()
	}
}

// maxMaskSize is the size in pixels over which glyphs are filled as
// outlines, not kept as masks
const maxMaskSize = 160

// DrawTextLayout draws the text laid out with its top left corner at the
// point (x, y), in the logical coordinates of the canvas, under its
// transform. A style without a color takes the color of SetColor.
func (c *Canvas) DrawTextLayout(l *TextLayout, x, y float64) {
	l.layout()
	if c.clipEmpty() {
		return
	}
	def := color.RGBAModel.Convert(c.state.col).(color.RGBA)
	colorOf := func(st TextStyle) color.RGBA {
		if st.Color == (color.RGBA{}) {
			return def
		}
		return st.Color
	}
	m := c.pixelMatrix()
	masks := c.rec == nil && m.B == 0 && m.C == 0 && m.A == m.D && m.A > 0
	for li := range l.lines {
		ln := &l.lines[li]
		top := y + ln.y
		baseline := top + ln.ascent
		left := x + ln.x
		// The backgrounds first: under the text of the runs next to them
		for _, r := range ln.runs {
			if r.style.Background.A > 0 {
				p := NewPath()
				p.Rect(left+r.x, top, fromFixed(r.out.Advance), ln.height)
				c.FillPath(p, r.style.Background)
			}
		}
		for ri := range ln.runs {
			r := &ln.runs[ri]
			col := colorOf(r.style)
			size := r.style.size()
			if masks && size*m.A <= maxMaskSize {
				c.drawRunMasks(r, m, left, baseline, col)
			} else {
				c.drawRunOutlines(r, left, baseline, col)
			}
			c.drawRunLines(r, left, baseline, col)
		}
	}
}

// drawRunMasks draws the glyphs of the run from their masks; m takes the
// logical point to the pixels
func (c *Canvas) drawRunMasks(r *layoutRun, m Matrix, left, baseline float64, col color.RGBA) {
	cr, cg, cb, ca := straightRGBA8(col)
	if ca == 0 {
		return
	}
	size := r.style.size()
	pen := left + r.x
	for _, g := range r.out.Glyphs {
		gx := pen + fromFixed(g.XOffset)
		gy := baseline - fromFixed(g.YOffset)
		pen += fromFixed(g.Advance)
		px, py := m.Apply(gx, gy)
		ix := math.Floor(px)
		phase := int((px - ix) * 4)
		mask := layoutGlyphMask(r.out.Face, g.GlyphID, size*m.A, phase, r.synth, c.lcd)
		blendGlyphMask(c.rgba, c.clipRect(), mask, int(ix), int(math.Round(py)), cr, cg, cb, ca, c.lcd)
	}
}

// drawRunOutlines fills the outlines of the glyphs of the run
func (c *Canvas) drawRunOutlines(r *layoutRun, left, baseline float64, col color.RGBA) {
	size := r.style.size()
	p := NewPath()
	pen := left + r.x
	italic := r.synth&FontItalic != 0
	for _, g := range r.out.Glyphs {
		gx := pen + fromFixed(g.XOffset)
		gy := baseline - fromFixed(g.YOffset)
		pen += fromFixed(g.Advance)
		s := shapeOf(r.out.Face, g.GlyphID)
		s.addTo(p, gx, gy, size, italic)
		if r.synth&FontBold != 0 {
			s.addTo(p, gx+size/24, gy, size, italic)
		}
	}
	c.FillPath(p, col)
}

// drawRunLines draws the underline and the line through the run
func (c *Canvas) drawRunLines(r *layoutRun, left, baseline float64, col color.RGBA) {
	if !r.style.Underline && !r.style.Strikethrough {
		return
	}
	face := r.out.Face
	k := r.style.size() / float64(face.Upem())
	line := func(pos, thick gtfont.LineMetric, defPos float64) {
		shapeMu.Lock()
		y, t := float64(face.LineMetric(pos))*k, float64(face.LineMetric(thick))*k
		shapeMu.Unlock()
		if t <= 0 {
			t = r.style.size() / 14
		}
		if y == 0 {
			y = defPos
		}
		p := NewPath()
		p.Rect(left+r.x, baseline-y-t/2, fromFixed(r.out.Advance), t)
		c.FillPath(p, col)
	}
	if r.style.Underline {
		line(gtfont.UnderlinePosition, gtfont.UnderlineThickness, -r.style.size()/10)
	}
	if r.style.Strikethrough {
		line(gtfont.StrikethroughPosition, gtfont.StrikethroughThickness, r.style.size()/4)
	}
}

// The layouts of DrawTextF, kept for the text drawn again
var (
	textFLayoutsMu sync.Mutex
	textFLayouts   = map[textFKey]*TextLayout{}
)

type textFKey struct {
	text, font string
	size       float64
	underline  bool
	align      HAlign
}

// DrawTextF draws the text, in the font, size and color of the canvas,
// around the point (x, y) that SetHAlign and SetVAlign name: the top left
// corner of its box by default. The point is fractional and the text is
// drawn under the transform of the canvas as a TextLayout is: sharp at any
// scale or angle, shaped. Lines are broken at "\n".
func (c *Canvas) DrawTextF(x, y float64, text string) {
	l := c.textFLayout(text)
	w, h := l.Size()
	switch c.state.hAlign {
	case HAlignCenter:
		x -= w / 2
	case HAlignRight:
		x -= w
	}
	switch c.state.vAlign {
	case VAlignCenter:
		y -= h / 2
	case VAlignBottom:
		y -= h
	}
	c.DrawTextLayout(l, x, y)
}

// MeasureTextF returns the size of the text as DrawTextF draws it
func (c *Canvas) MeasureTextF(text string) (w, h float64) {
	return c.textFLayout(text).Size()
}

// textFLayout returns the layout of the text in the font of the canvas
func (c *Canvas) textFLayout(text string) *TextLayout {
	key := textFKey{text, c.state.fontFamily, c.state.fontSize, c.state.underline, c.state.hAlign}
	textFLayoutsMu.Lock()
	l, ok := textFLayouts[key]
	textFLayoutsMu.Unlock()
	if ok {
		return l
	}
	l = NewTextLayout(text, TextStyle{Font: c.state.fontFamily, Size: c.state.fontSize, Underline: c.state.underline})
	l.SetAlign(c.state.hAlign)
	l.layout()
	textFLayoutsMu.Lock()
	if len(textFLayouts) >= 2048 {
		clear(textFLayouts)
	}
	textFLayouts[key] = l
	textFLayoutsMu.Unlock()
	return l
}
