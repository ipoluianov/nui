package ui

import (
	"image"
	"image/color"
	"math"
	"sort"
	"sync"
	"unicode"
	"unicode/utf8"

	gtseg "github.com/go-text/typesetting/segmenter"
	"golang.org/x/image/font"
)

// Text that is not drawn right a character after another - written right
// to left, of a script whose letters join or combine, with combining marks
// - is drawn, measured and edited through a TextLayout: shaped and in the
// order of the bidi algorithm. DrawText, MeasureText and GetCharPositions
// take that way for such a line, so labels, tables, menus and buttons show
// it right; the rest of the text is drawn as before, character by
// character.

// needsShaping tells whether the text is to be laid out by the shaper
func needsShaping(text string) bool {
	for _, r := range text {
		if r < 0x0300 {
			continue
		}
		switch {
		case r <= 0x036F: // combining diacritical marks
			return true
		case r >= 0x0590 && r <= 0x08FF: // Hebrew, Arabic, Syriac, Thaana, N'Ko...
			return true
		case r >= 0x0900 && r <= 0x0DFF: // the Indic scripts
			return true
		case r >= 0x0E00 && r <= 0x0FFF: // Thai, Lao, Tibetan
			return true
		case r >= 0x1000 && r <= 0x109F: // Myanmar
			return true
		case r >= 0x1780 && r <= 0x18AF: // Khmer, Mongolian
			return true
		case r >= 0x1A00 && r <= 0x1CFF: // Buginese... Balinese, Javanese
			return true
		case r == 0x200C || r == 0x200D || (r >= 0x200E && r <= 0x200F) || (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069):
			return true // joiners and the marks of direction
		case r >= 0xA800 && r <= 0xABFF: // Syloti... Meetei Mayek
			return true
		case r >= 0xFB1D && r <= 0xFDFF, r >= 0xFE70 && r <= 0xFEFF: // presentation forms
			return true
		case r >= 0x10800 && r <= 0x10FFF, r >= 0x11000 && r <= 0x11FFF, r >= 0x1E800 && r <= 0x1EFFF:
			return true // the historic and other scripts of the SMP
		}
		if unicode.In(r, unicode.Mn, unicode.Mc, unicode.Me) {
			return true
		}
	}
	return false
}

// The layouts of the lines drawn and measured through the shaper, by the
// size in pixels
var (
	shapedLinesMu sync.Mutex
	shapedLines   = map[shapedLineKey]*TextLayout{}
)

type shapedLineKey struct {
	text, font string
	size       float64
}

// shapedLine returns the layout of the line in the font at the size
func shapedLine(text, family string, size float64) *TextLayout {
	key := shapedLineKey{text, family, size}
	shapedLinesMu.Lock()
	l, ok := shapedLines[key]
	shapedLinesMu.Unlock()
	if ok {
		return l
	}
	l = NewTextLayout(text, TextStyle{Font: family, Size: size})
	l.layout()
	shapedLinesMu.Lock()
	if len(shapedLines) >= 2048 {
		clear(shapedLines)
	}
	shapedLines[key] = l
	shapedLinesMu.Unlock()
	return l
}

// clearShapedLines forgets the layouts: the fonts changed
func clearShapedLines() {
	shapedLinesMu.Lock()
	clear(shapedLines)
	shapedLinesMu.Unlock()
}

// textBaseline returns where DrawText puts the baseline of a line whose
// box starts at y: under the ascent of the font, in pixels
func textBaseline(family string, size float64, y int) int {
	baseline := y
	withFace(family, size, func(face font.Face) {
		m := face.Metrics()
		baseline = y + (m.Ascent + m.Descent).Ceil() - m.Descent.Ceil()
	})
	if ascent, _, ok := nativeMetrics(family, size); ok {
		baseline = y + ascent
	}
	return baseline
}

// drawShapedText draws a line as DrawText does, in pixels of dst, through
// the shaper: on the baseline DrawText would draw it on
func drawShapedText(dst *image.RGBA, text string, col color.Color, family string, size float64, x, y int, clip image.Rectangle, lcd int) {
	l := shapedLine(text, family, size)
	if len(l.lines) == 0 {
		return
	}
	cnv := &Canvas{rgba: dst, scale: 1, lcd: lcd,
		state: &canvasState{col: col, clipX: clip.Min.X, clipY: clip.Min.Y, clipW: clip.Dx(), clipH: clip.Dy()}}
	baseline := textBaseline(family, size, y)
	cnv.DrawTextLayout(l, float64(x), float64(baseline)-l.lines[0].ascent)
}

// shapedWidth returns the width of the line in pixels, rounded up
func shapedWidth(text, family string, size float64) int {
	w, _ := shapedLine(text, family, size).Size()
	return int(math.Ceil(w - 1e-6))
}

// shapedPositions returns the x of the caret before each character of the
// line, in pixels, and its width last, as charPositions does. In text
// written right to left they are not in order.
func shapedPositions(text, family string, size float64) []int {
	l := shapedLine(text, family, size)
	out := make([]int, 0, utf8.RuneCountInString(text)+1)
	for i := range text {
		out = append(out, int(math.Round(l.CaretRect(i).MinX)))
	}
	out = append(out, shapedWidth(text, family, size))
	return out
}

// TextCaretX returns the x of the caret before the character i (in runes)
// of a line of text, measured as GetCharPositions measures; i past the
// last character is the end of the text: at the right of text written left
// to right, at the left of text written right to left
func TextCaretX(fontFamily string, fontSize float64, text string, i int) int {
	if !needsShaping(text) {
		positions, err := GetCharPositions(fontFamily, fontSize, text)
		if err != nil || len(positions) == 0 {
			return 0
		}
		return positions[max(0, min(i, len(positions)-1))]
	}
	s := textScale()
	l := shapedLine(text, fontFamily, fontSize*s)
	return toLogicalText(int(math.Round(l.CaretRect(runeOffset(text, max(0, i))).MinX)), s)
}

// TextIndexAt returns the index of the character (in runes) whose caret is
// nearest the x of a line of text, measured as GetCharPositions measures:
// where a click lands. It knows text written right to left.
func TextIndexAt(fontFamily string, fontSize float64, text string, x int) int {
	if !needsShaping(text) {
		positions, err := GetCharPositions(fontFamily, fontSize, text)
		if err != nil || len(positions) == 0 {
			return 0
		}
		best := 0
		for i, p := range positions {
			if abs(p-x) < abs(positions[best]-x) {
				best = i
			}
		}
		return best
	}
	s := textScale()
	l := shapedLine(text, fontFamily, fontSize*s)
	i := l.IndexAt(float64(x)*s, 0)
	return utf8.RuneCountInString(text[:i])
}

// TextSelectionSpans returns the ranges of x a selection of the characters
// [from, to) (in runes) of a line covers, measured as GetCharPositions
// measures: one range, or a few where text written right to left is mixed
// in
func TextSelectionSpans(fontFamily string, fontSize float64, text string, from, to int) [][2]int {
	if from >= to {
		return nil
	}
	if !needsShaping(text) {
		positions, err := GetCharPositions(fontFamily, fontSize, text)
		if err != nil || len(positions) == 0 {
			return nil
		}
		from, to = max(0, from), min(to, len(positions)-1)
		return [][2]int{{positions[from], positions[to]}}
	}
	s := textScale()
	l := shapedLine(text, fontFamily, fontSize*s)
	a, b := runeOffset(text, from), runeOffset(text, to)
	var out [][2]int
	for _, r := range l.SelectionRects(a, b) {
		out = append(out, [2]int{toLogicalText(int(math.Round(r.MinX)), s), toLogicalText(int(math.Round(r.MaxX)), s)})
	}
	return out
}

// runeOffset returns the byte offset of the rune i of the text
func runeOffset(text string, i int) int {
	n := 0
	for off := range text {
		if n == i {
			return off
		}
		n++
	}
	return len(text)
}

// graphemeBounds returns the indexes (in runes) where the graphemes of the
// text start, and its length last: the places a caret stops at, not within
// a letter and its marks or a conjunct of an Indic script
func graphemeBounds(text string) []int {
	runes := []rune(text)
	if !needsShaping(text) {
		out := make([]int, len(runes)+1)
		for i := range out {
			out[i] = i
		}
		return out
	}
	var seg gtseg.Segmenter
	seg.Init(runes)
	it := seg.GraphemeIterator()
	var out []int
	for it.Next() {
		out = append(out, it.Grapheme().Offset)
	}
	return append(out, len(runes))
}

// nextGrapheme returns the start of the grapheme after the one at i
func nextGrapheme(text string, i int) int {
	b := graphemeBounds(text)
	k := sort.SearchInts(b, i+1)
	if k >= len(b) {
		return b[len(b)-1]
	}
	return b[k]
}

// prevGrapheme returns the start of the grapheme before i
func prevGrapheme(text string, i int) int {
	b := graphemeBounds(text)
	k := sort.SearchInts(b, i) - 1
	if k < 0 {
		return 0
	}
	return b[k]
}

// snapToGrapheme returns the start of a grapheme nearest i
func snapToGrapheme(text string, i int) int {
	b := graphemeBounds(text)
	k := sort.SearchInts(b, i)
	if k < len(b) && b[k] == i {
		return i
	}
	if k == 0 {
		return b[0]
	}
	if k >= len(b) {
		return b[len(b)-1]
	}
	if i-b[k-1] <= b[k]-i {
		return b[k-1]
	}
	return b[k]
}

// drawLayoutAsText draws a layout of one line where DrawText(x, y, w, h)
// draws a line of text in the family and the size: on the same baseline,
// aligned up and down as the canvas aligns
func (c *Canvas) drawLayoutAsText(l *TextLayout, x, y, h int, family string, size float64) {
	l.layout()
	if len(l.lines) == 0 {
		return
	}
	_, textHeight, err := MeasureText(family, size, "Йg")
	if err != nil {
		return
	}
	switch c.state.vAlign {
	case VAlignCenter:
		y += h/2 - textHeight/2
	case VAlignBottom:
		y += h - textHeight
	}
	py := c.state.translateY + c.scaled(y)
	baseline := float64(textBaseline(family, size*c.scale, py)-c.state.translateY) / c.scale
	c.DrawTextLayout(l, float64(x), baseline-l.lines[0].ascent)
}
