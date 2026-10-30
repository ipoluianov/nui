package ui

import (
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Text as outlines: the letters of a font as curves of a Path. Unlike
// DrawText, which draws pictures of the letters at the size of the screen,
// such text is sharp at any scale, turns and slants with the canvas'
// transform, is filled and stroked as any path, and is written to SVG as
// curves - the text of a drawing, of a map, of a label on a 3D view.

// Text adds the outlines of the text to the path: the start of its first
// line's baseline at (x, y), the letters size units tall (as a font size:
// the height of the em), y down. Lines are split at "\n". The family is
// as for DrawText ("notosans", "jetbrainsmono:bold"); the characters its
// font lacks come from the fallback fonts. A bold the family lacks is not
// synthesized; an italic is slanted.
func (p *Path) Text(x, y float64, text, fontFamily string, size float64) {
	if size <= 0 || text == "" {
		return
	}
	chain, italic := outlineFonts(fontFamily)
	if len(chain) == 0 {
		return
	}
	buf := sfntBuffers.Get().(*sfnt.Buffer)
	defer sfntBuffers.Put(buf)
	lineHeight := outlineLineHeight(buf, chain[0], size)
	for i, line := range strings.Split(text, "\n") {
		p.textLine(buf, chain, italic, x, y+float64(i)*lineHeight, line, size)
	}
}

// TextAligned adds the outlines of the text as Text does, aligned to the
// point (x, y) as DrawText aligns to its rectangle: HAlignLeft starts the
// lines at x, HAlignCenter centers them on it, HAlignRight ends them there;
// VAlignTop puts the top of the first line at y, VAlignCenter the middle of
// the lines, VAlignBottom the bottom of the last one.
func (p *Path) TextAligned(x, y float64, text, fontFamily string, size float64, hAlign HAlign, vAlign VAlign) {
	if size <= 0 || text == "" {
		return
	}
	chain, _ := outlineFonts(fontFamily)
	if len(chain) == 0 {
		return
	}
	buf := sfntBuffers.Get().(*sfnt.Buffer)
	ascent, descent, lineHeight := outlineMetrics(buf, chain[0], size)
	sfntBuffers.Put(buf)
	lines := strings.Split(text, "\n")
	height := ascent + descent + float64(len(lines)-1)*lineHeight
	top := y
	switch vAlign {
	case VAlignCenter:
		top = y - height/2
	case VAlignBottom:
		top = y - height
	}
	for i, line := range lines {
		lx := x
		if hAlign != HAlignLeft {
			w, _, _ := MeasureTextOutline(line, fontFamily, size)
			if hAlign == HAlignCenter {
				lx -= w / 2
			} else {
				lx -= w
			}
		}
		p.Text(lx, top+ascent+float64(i)*lineHeight, line, fontFamily, size)
	}
}

// MeasureTextOutline returns the width of the widest line of the text as
// Path.Text adds it, and the ascent and the descent of its font: how far
// its letters reach above and below the baseline
func MeasureTextOutline(text, fontFamily string, size float64) (width, ascent, descent float64) {
	chain, _ := outlineFonts(fontFamily)
	if len(chain) == 0 || size <= 0 {
		return 0, 0, 0
	}
	buf := sfntBuffers.Get().(*sfnt.Buffer)
	defer sfntBuffers.Put(buf)
	ascent, descent, _ = outlineMetrics(buf, chain[0], size)
	for _, line := range strings.Split(text, "\n") {
		w := 0.0
		walkOutlineGlyphs(buf, chain, line, size, func(*sfnt.Font, sfnt.GlyphIndex, float64) {}, &w)
		width = max(width, w)
	}
	return width, ascent, descent
}

// textLine adds one line of the text, its baseline starting at (x, y)
func (p *Path) textLine(buf *sfnt.Buffer, chain []*sfnt.Font, italic bool, x, y float64, line string, size float64) {
	var w float64
	walkOutlineGlyphs(buf, chain, line, size, func(f *sfnt.Font, g sfnt.GlyphIndex, pen float64) {
		o := glyphOutline(buf, f, g)
		if o == nil {
			return
		}
		// Font units to the size; an italic slants by 12 degrees
		k := size / float64(o.unitsPerEm)
		slant := 0.0
		if italic {
			slant = 0.21
		}
		at := func(q PointF) (float64, float64) {
			return x + pen + (q.X-q.Y*slant)*k, y + q.Y*k
		}
		open := false
		for _, s := range o.segs {
			switch s.op {
			case sfnt.SegmentOpMoveTo:
				if open {
					p.Close()
				}
				p.MoveTo(at(s.p[0]))
				open = true
			case sfnt.SegmentOpLineTo:
				p.LineTo(at(s.p[0]))
			case sfnt.SegmentOpQuadTo:
				cx, cy := at(s.p[0])
				ex, ey := at(s.p[1])
				p.QuadTo(cx, cy, ex, ey)
			case sfnt.SegmentOpCubeTo:
				c1x, c1y := at(s.p[0])
				c2x, c2y := at(s.p[1])
				ex, ey := at(s.p[2])
				p.CubicTo(c1x, c1y, c2x, c2y, ex, ey)
			}
		}
		if open {
			p.Close()
		}
	}, &w)
}

// walkOutlineGlyphs calls f with the font, the glyph and the pen position
// (from the start of the line) of every character of the line, kerned;
// *width is set to the width of the line
func walkOutlineGlyphs(buf *sfnt.Buffer, chain []*sfnt.Font, line string, size float64, f func(*sfnt.Font, sfnt.GlyphIndex, float64), width *float64) {
	pen := 0.0
	var prevFont *sfnt.Font
	var prev sfnt.GlyphIndex
	for _, r := range line {
		fnt, g := chain[0], sfnt.GlyphIndex(0)
		for _, c := range chain {
			if i, err := c.GlyphIndex(buf, r); err == nil && i != 0 {
				fnt, g = c, i
				break
			}
		}
		if g == 0 {
			g, _ = fnt.GlyphIndex(buf, r)
		}
		upem := float64(fnt.UnitsPerEm())
		ppem := fixed.Int26_6(fnt.UnitsPerEm()) << 6
		if prevFont == fnt {
			if kern, err := fnt.Kern(buf, prev, g, ppem, font.HintingNone); err == nil {
				pen += float64(kern) / 64 * size / upem
			}
		}
		f(fnt, g, pen)
		if adv, err := fnt.GlyphAdvance(buf, g, ppem, font.HintingNone); err == nil {
			pen += float64(adv) / 64 * size / upem
		}
		prevFont, prev = fnt, g
	}
	*width = pen
}

// outlineMetrics returns the ascent, the descent and the line height of
// the font at the size
func outlineMetrics(buf *sfnt.Buffer, f *sfnt.Font, size float64) (ascent, descent, lineHeight float64) {
	ppem := fixed.Int26_6(f.UnitsPerEm()) << 6
	m, err := f.Metrics(buf, ppem, font.HintingNone)
	if err != nil {
		return size * 0.8, size * 0.2, size * 1.2
	}
	k := size / float64(f.UnitsPerEm()) / 64
	return float64(m.Ascent) * k, float64(m.Descent) * k, float64(m.Height) * k
}

func outlineLineHeight(buf *sfnt.Buffer, f *sfnt.Font, size float64) float64 {
	_, _, h := outlineMetrics(buf, f, size)
	return h
}

// outlineFonts returns the fonts the text of the family is drawn with: the
// family's and the fallbacks, and whether it is to be slanted
func outlineFonts(family string) ([]*sfnt.Font, bool) {
	if family == "" {
		family = defaultFontKey
	}
	rf := resolveFont(family)
	fontsMu.RLock()
	defer fontsMu.RUnlock()
	primary, synth, ok := lookupFont(rf.family, rf.style)
	if !ok {
		if primary, synth, ok = lookupFont(defaultFontKey, rf.style); !ok {
			return nil, false
		}
	}
	chain := []*sfnt.Font{primary}
	for _, name := range fallbackFonts {
		if f, _, ok := lookupFont(name, rf.style); ok && f != primary {
			chain = append(chain, f)
		}
	}
	return chain, synth&FontItalic != 0
}

// glyphOutlineData is a glyph's outline in font units, y down
type glyphOutlineData struct {
	unitsPerEm int
	segs       []glyphSeg
}

type glyphSeg struct {
	op sfnt.SegmentOp
	p  [3]PointF
}

type glyphKey struct {
	f *sfnt.Font
	g sfnt.GlyphIndex
}

// glyphOutlines caches the outlines: loading one is slow, and the text of
// a drawing is drawn again on every frame
var glyphOutlines sync.Map // glyphKey -> *glyphOutlineData

var sfntBuffers = sync.Pool{New: func() any { return &sfnt.Buffer{} }}

// glyphOutline returns the outline of the glyph, nil if it has none
func glyphOutline(buf *sfnt.Buffer, f *sfnt.Font, g sfnt.GlyphIndex) *glyphOutlineData {
	key := glyphKey{f, g}
	if o, ok := glyphOutlines.Load(key); ok {
		return o.(*glyphOutlineData)
	}
	ppem := fixed.Int26_6(f.UnitsPerEm()) << 6
	segs, err := f.LoadGlyph(buf, g, ppem, nil)
	var o *glyphOutlineData
	if err == nil && len(segs) > 0 {
		o = &glyphOutlineData{unitsPerEm: int(f.UnitsPerEm()), segs: make([]glyphSeg, len(segs))}
		for i, s := range segs {
			o.segs[i].op = s.Op
			for k := range s.Args {
				o.segs[i].p[k] = PointF{float64(s.Args[k].X) / 64, float64(s.Args[k].Y) / 64}
			}
		}
	}
	glyphOutlines.Store(key, o)
	return o
}
