package ui

import (
	"image/color"
	"math"
	"reflect"
	"sort"
	"unicode/utf8"

	gtbidi "github.com/go-text/typesetting/bidi"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	xbidi "golang.org/x/text/unicode/bidi"
)

// TextStyle is how a piece of the text of a TextLayout looks
type TextStyle struct {
	// Font is the family; it may name a style ("sans:bold"). Empty is the
	// default font.
	Font string
	// Size in logical pixels; 0 is DefaultFontSize
	Size float64
	// Color of the letters; the zero value is the color of the canvas
	// (SetColor) at the time of drawing
	Color                    color.RGBA
	Bold, Italic             bool
	Underline, Strikethrough bool
	// Background, if not transparent, is filled behind the text: a mark
	Background color.RGBA
	// LetterSpacing is added between the letters, in logical pixels
	LetterSpacing float64
}

func (s TextStyle) size() float64 {
	if s.Size <= 0 {
		return DefaultFontSize
	}
	return s.Size
}

func (s TextStyle) fontStyle() FontStyle {
	var st FontStyle
	if s.Bold {
		st |= FontBold
	}
	if s.Italic {
		st |= FontItalic
	}
	return st
}

// TextLayout is text laid out in lines: of several styles, wrapped at a
// width, aligned, of any script - shaped by HarfBuzz (ligatures, kerning,
// Arabic, Hebrew, Indic scripts, right to left mixed with left to right).
// It is placed in fractional logical coordinates and drawn sharp under
// any transform of the canvas. It tells where each character is: the
// caret, the selection, the character under the pointer.
//
//	l := ui.NewTextLayout("Hello, ", ui.TextStyle{Size: 16})
//	l.Append("world", ui.TextStyle{Size: 16, Bold: true, Color: red})
//	l.SetWidth(200)
//	cnv.DrawTextLayout(l, 10.5, 20.25)
//
// The indexes of the characters are byte offsets in Text(). A TextLayout
// is laid out once, when it is first measured or drawn after a change, and
// may be drawn many times; it is not safe for concurrent use.
type TextLayout struct {
	text        string
	spans       []textSpan
	width       float64
	align       HAlign
	lineSpacing float64
	maxLines    int
	rtl         int // 0 by the text, 1 left to right, 2 right to left

	laidOut bool
	runes   []rune
	offsets []int // the byte offset of each rune, and len(text) last
	lines   []layoutLine
	w, h    float64
}

type textSpan struct {
	end   int // the rune after the span
	style TextStyle
}

// layoutLine is a line laid out: its runs left to right
type layoutLine struct {
	runs           []layoutRun
	start, end     int // the runes of the line; end is before a line break
	y              float64
	ascent, height float64
	x, width       float64
	rtl            bool
	clusters       []caretCluster // left to right
}

// layoutRun is a run of glyphs of a font and a style
type layoutRun struct {
	out   shaping.Output
	style TextStyle
	synth FontStyle
	x     float64 // from the left of the line
	base  int     // the rune of the text the run's rune 0 is
}

// caretCluster is a group of glyphs that stands for a group of runes
type caretCluster struct {
	start, count int // runes of the text
	x0, x1       float64
	rtl          bool
}

// NewTextLayout returns a layout of the text in the style
func NewTextLayout(text string, style TextStyle) *TextLayout {
	l := &TextLayout{lineSpacing: 1}
	l.Append(text, style)
	return l
}

// Append adds text in the style at the end
func (l *TextLayout) Append(text string, style TextStyle) {
	if text == "" {
		return
	}
	l.text += text
	l.spans = append(l.spans, textSpan{end: utf8.RuneCountInString(l.text), style: style})
	l.laidOut = false
}

// SetText replaces the text by one of a style
func (l *TextLayout) SetText(text string, style TextStyle) {
	l.text, l.spans = "", l.spans[:0]
	l.Append(text, style)
	if text == "" {
		l.spans = append(l.spans, textSpan{style: style})
	}
}

// Text returns the text
func (l *TextLayout) Text() string { return l.text }

// SetWidth wraps the lines at the width in logical pixels; 0 does not
// wrap them
func (l *TextLayout) SetWidth(w float64) {
	if w != l.width {
		l.width, l.laidOut = max(0, w), false
	}
}

// SetAlign aligns the lines in the width (or in the widest line, if there
// is no width): left, center, right. Right-to-left text is aligned to the
// right by HAlignLeft, as the start of its lines is there.
func (l *TextLayout) SetAlign(a HAlign) {
	if a != l.align {
		l.align, l.laidOut = a, false
	}
}

// SetLineSpacing sets the height of the lines to k times their font's;
// 1 is the default
func (l *TextLayout) SetLineSpacing(k float64) {
	if k <= 0 {
		k = 1
	}
	if k != l.lineSpacing {
		l.lineSpacing, l.laidOut = k, false
	}
}

// SetMaxLines keeps the first n lines, the last ending with an ellipsis
// if the text goes on; 0 keeps all
func (l *TextLayout) SetMaxLines(n int) {
	if n != l.maxLines {
		l.maxLines, l.laidOut = max(0, n), false
	}
}

// SetRightToLeft sets the direction of the paragraphs: true right to left
// (Arabic, Hebrew), false left to right. By default each paragraph takes
// the direction of its first letter of a direction.
func (l *TextLayout) SetRightToLeft(rtl bool) {
	v := 1
	if rtl {
		v = 2
	}
	if v != l.rtl {
		l.rtl, l.laidOut = v, false
	}
}

// Size returns the width of the widest line and the height of all
func (l *TextLayout) Size() (w, h float64) {
	l.layout()
	return l.w, l.h
}

// TextLine is a line of a layout, in its logical coordinates
type TextLine struct {
	// Start and End are the byte offsets of its text: End is before the
	// line break, if it ends with one
	Start, End int
	// X, Y are its top left corner; Width, Height its size
	X, Y, Width, Height float64
	// Baseline is the y of its baseline
	Baseline float64
	// RightToLeft is the direction of its paragraph
	RightToLeft bool
}

// Lines returns the lines laid out
func (l *TextLayout) Lines() []TextLine {
	l.layout()
	out := make([]TextLine, len(l.lines))
	for i, ln := range l.lines {
		out[i] = TextLine{Start: l.offsets[ln.start], End: l.offsets[ln.end], X: ln.x, Y: ln.y,
			Width: ln.width, Height: ln.height, Baseline: ln.y + ln.ascent, RightToLeft: ln.rtl}
	}
	return out
}

// spanAt returns the style of the rune i
func (l *TextLayout) styleAt(i int) TextStyle {
	for _, s := range l.spans {
		if i < s.end {
			return s.style
		}
	}
	if len(l.spans) > 0 {
		return l.spans[len(l.spans)-1].style
	}
	return TextStyle{}
}

// layout lays the text out, if it changed
func (l *TextLayout) layout() {
	if l.laidOut {
		return
	}
	l.laidOut = true
	l.runes = []rune(l.text)
	l.offsets = l.offsets[:0]
	for i := range l.text {
		l.offsets = append(l.offsets, i)
	}
	l.offsets = append(l.offsets, len(l.text))
	l.lines = l.lines[:0]
	l.w, l.h = 0, 0

	ensureFallbackFonts(l.text)
	shapeMu.Lock()
	defer shapeMu.Unlock()
	start := 0
	for {
		end := start
		for end < len(l.runes) && l.runes[end] != '\n' {
			end++
		}
		if l.maxLines > 0 && len(l.lines) >= l.maxLines {
			break
		}
		l.layoutParagraph(start, end, end < len(l.runes))
		if end >= len(l.runes) {
			break
		}
		start = end + 1
	}
	// The lines one under another, aligned
	y := 0.0
	for i := range l.lines {
		ln := &l.lines[i]
		ln.y = y
		y += ln.height
		l.w = max(l.w, ln.width)
	}
	l.h = y
	box := l.width
	if box == 0 {
		box = l.w
	}
	for i := range l.lines {
		ln := &l.lines[i]
		switch {
		case l.align == HAlignCenter:
			ln.x = (box - ln.width) / 2
		case (l.align == HAlignRight) != ln.rtl:
			ln.x = box - ln.width
		}
	}
}

// toFixed is a length in logical pixels as the shaper's 26.6
func toFixed(v float64) fixed.Int26_6 { return fixed.Int26_6(math.Round(v * 64)) }

func fromFixed(v fixed.Int26_6) float64 { return float64(v) / 64 }

// paragraphRTL tells the direction of the paragraph by its first letter
// of a direction
func (l *TextLayout) paragraphRTL(start, end int) bool {
	if l.rtl != 0 {
		return l.rtl == 2
	}
	for _, r := range l.runes[start:end] {
		p, _ := xbidi.LookupRune(r)
		switch p.Class() {
		case xbidi.L:
			return false
		case xbidi.R, xbidi.AL:
			return true
		}
	}
	return false
}

// layoutParagraph lays out the runes start..end, a paragraph, in lines
func (l *TextLayout) layoutParagraph(start, end int, broken bool) {
	rtl := l.paragraphRTL(start, end)
	dir := di.DirectionLTR
	if rtl {
		dir = di.DirectionRTL
	}
	para := l.runes[start:end]
	firstStyle := l.styleAt(start)
	if len(para) == 0 {
		l.emptyLine(start, firstStyle, rtl)
		return
	}
	// By direction and script over the whole paragraph, by the fonts of
	// the styles in each span
	firstChain := shapeChain(firstStyle.Font, firstStyle.fontStyle())
	if firstChain == nil {
		return
	}
	// The embedding levels of the runes, by the bidi algorithm: the runs
	// are cut where they change, and the runs of a line ordered by them
	var bp gtbidi.Paragraph
	def := gtbidi.LeftToRight
	if rtl {
		def = gtbidi.RightToLeft
	}
	bruns := bp.Segment(para, def)
	levels := runeLevels(&bruns, len(para), rtl)
	levelAt := func(r int) (int, int) {
		end := r + 1
		for end < len(levels) && levels[end] == levels[r] {
			end++
		}
		return levels[r], end
	}
	var runs []shaping.Output
	var styles []TextStyle
	var synths []FontStyle
	whole := shaping.Input{Text: para, RunStart: 0, RunEnd: len(para), Direction: dir, Size: toFixed(firstStyle.size())}
	parts := append([]shaping.Input(nil), segmenter.Split(whole, firstChain)...)
	for _, part := range parts {
		for a := part.RunStart; a < part.RunEnd; {
			// The end of the span of the rune a, within the part
			st, b := l.spanOf(start + a)
			_, levelEnd := levelAt(a)
			b = min(b-start, part.RunEnd, levelEnd)
			chain := shapeChain(st.Font, st.fontStyle())
			in := part
			in.RunStart, in.RunEnd = a, b
			in.Size = toFixed(st.size())
			for _, sub := range splitByFace(in, chain) {
				out := shaper.Shape(sub)
				synth := chain.synthOf(sub.Face)
				if synth&FontBold != 0 {
					embolden(&out, st.size())
				}
				if st.LetterSpacing != 0 {
					out.AddLetterSpacing(toFixed(st.LetterSpacing), false, false)
				}
				runs = append(runs, out)
				styles = append(styles, st)
				synths = append(synths, synth)
			}
			a = b
		}
	}
	cfg := shaping.WrapConfig{Direction: dir}
	if l.maxLines > 0 {
		cfg.TruncateAfterLines = l.maxLines - len(l.lines)
		cfg.TextContinues = broken
		cfg = cfg.WithTruncator(&shaper, shaping.Input{Text: []rune("…"), RunEnd: 1, Direction: dir,
			Face: firstChain.ResolveFace('…'), Size: toFixed(firstStyle.size())})
	}
	maxWidth := fixed.Int26_6(math.MaxInt32)
	if l.width > 0 {
		maxWidth = toFixed(l.width)
	}
	var wrapper shaping.LineWrapper
	lines, _ := wrapper.WrapParagraphF(cfg, maxWidth, para, shaping.NewSliceIterator(runs))
	for li, line := range lines {
		ln := layoutLine{rtl: rtl, start: math.MaxInt, end: 0}
		levels := make([]int, len(line))
		for i, out := range line {
			levels[i], _ = levelAt(out.Runes.Offset)
		}
		vis := visualOrder(levels)
		var asc, desc float64
		x := 0.0
		for _, i := range vis {
			out := line[i]
			// The style of the run: of the shaped run it was cut from
			k := runOf(runs, out.Runes.Offset)
			st, synth := firstStyle, FontStyle(0)
			if k >= 0 {
				st, synth = styles[k], synths[k]
			}
			if out.Runes.Count > 0 {
				ln.start = min(ln.start, start+out.Runes.Offset)
				ln.end = max(ln.end, start+out.Runes.Offset+out.Runes.Count)
			}
			r := layoutRun{out: out, style: st, synth: synth, x: x, base: start}
			ln.clusters = appendClusters(ln.clusters, r)
			ln.runs = append(ln.runs, r)
			x += fromFixed(out.Advance)
			asc = max(asc, fromFixed(out.LineBounds.Ascent))
			desc = max(desc, -fromFixed(out.LineBounds.Descent))
		}
		if ln.start == math.MaxInt {
			ln.start, ln.end = start, start
		}
		if li == len(lines)-1 {
			ln.end = end // the spaces trimmed at its end are its own
		}
		ln.width = x
		ln.ascent = asc
		ln.height = (asc + desc) * l.lineSpacing
		ln.ascent += (ln.height - (asc + desc)) / 2
		l.lines = append(l.lines, ln)
		if l.maxLines > 0 && len(l.lines) >= l.maxLines {
			break
		}
	}
	// A wrapped line ends where the next one starts
	n := len(lines)
	for i := len(l.lines) - n; i >= 0 && i < len(l.lines)-1; i++ {
		l.lines[i].end = max(l.lines[i].end, l.lines[i+1].start)
	}
}

// runeLevels returns the embedding level of every rune. go-text's runs
// are of a direction, not of a level - digits at level 2 in an Arabic
// sentence and the English after it at 0 are one run - so the levels are
// read from the runs, where they are kept; if they cannot be, the level of
// the run is taken.
func runeLevels(runs *gtbidi.Runs, n int, rtl bool) []int {
	out := make([]int, n)
	if f := reflect.ValueOf(runs).Elem().FieldByName("levels"); f.IsValid() && f.Kind() == reflect.Slice && f.Len() == n {
		switch f.Type().Elem().Kind() {
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
			for i := range out {
				out[i] = int(f.Index(i).Int())
			}
			return out
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			for i := range out {
				out[i] = int(f.Index(i).Uint())
			}
			return out
		}
	}
	base := 0
	if rtl {
		base = 1
	}
	for i := range out {
		out[i] = base
	}
	for i := 0; i < runs.NumRuns(); i++ {
		run := runs.Run(i)
		for j := run.Start; j < min(run.End, n); j++ {
			out[j] = int(run.Level)
		}
	}
	return out
}

// visualOrder returns the runs of the levels left to right, by the rule L2
// of the bidi algorithm: from the highest level to the lowest odd one,
// every sequence of runs at that level or higher is reversed
func visualOrder(levels []int) []int {
	vis := make([]int, len(levels))
	hi, lo := 0, math.MaxInt
	for i, lv := range levels {
		vis[i] = i
		hi = max(hi, lv)
		if lv%2 == 1 {
			lo = min(lo, lv)
		}
	}
	for lv := hi; lv >= lo && lv > 0; lv-- {
		for i := 0; i < len(vis); {
			if levels[vis[i]] < lv {
				i++
				continue
			}
			j := i
			for j < len(vis) && levels[vis[j]] >= lv {
				j++
			}
			for a, b := i, j-1; a < b; a, b = a+1, b-1 {
				vis[a], vis[b] = vis[b], vis[a]
			}
			i = j
		}
	}
	return vis
}

// spanOf returns the style of the rune i and the end of its span
func (l *TextLayout) spanOf(i int) (TextStyle, int) {
	for _, s := range l.spans {
		if i < s.end {
			return s.style, s.end
		}
	}
	return l.styleAt(i), len(l.runes)
}

// runOf returns the shaped run that has the rune, -1 if none
func runOf(runs []shaping.Output, r int) int {
	for i, o := range runs {
		if r >= o.Runes.Offset && r < o.Runes.Offset+o.Runes.Count {
			return i
		}
	}
	return -1
}

// emptyLine adds a line without text, as tall as its font's lines
func (l *TextLayout) emptyLine(at int, st TextStyle, rtl bool) {
	asc, desc := 0.8*st.size(), 0.2*st.size()
	if chain := shapeChain(st.Font, st.fontStyle()); chain != nil {
		if ext, ok := chain.fonts[0].face.FontHExtents(); ok {
			k := st.size() / float64(chain.fonts[0].face.Upem())
			asc, desc = float64(ext.Ascender)*k, -float64(ext.Descender)*k
		}
	}
	h := (asc + desc) * l.lineSpacing
	l.lines = append(l.lines, layoutLine{start: at, end: at, ascent: asc + (h-asc-desc)/2, height: h, rtl: rtl})
}

// embolden widens the glyphs of a synthesized bold face, as the bold
// drawn by DrawText: by a 24th of the size
func embolden(out *shaping.Output, size float64) {
	b := toFixed(size / 24)
	for i := range out.Glyphs {
		out.Glyphs[i].Advance += b
		out.Glyphs[i].XAdvance += b
	}
	out.RecomputeAdvance()
}

// appendClusters adds the clusters of the run, left to right
func appendClusters(cs []caretCluster, r layoutRun) []caretCluster {
	rtl := r.out.Direction.Progression() == di.TowardTopLeft
	x := r.x
	g := r.out.Glyphs
	for i := 0; i < len(g); {
		j := i
		adv := 0.0
		for j < len(g) && g[j].ClusterIndex == g[i].ClusterIndex {
			adv += fromFixed(g[j].Advance)
			j++
		}
		cs = append(cs, caretCluster{start: r.base + g[i].ClusterIndex, count: max(1, g[i].RuneCount), x0: x, x1: x + adv, rtl: rtl})
		x += adv
		i = j
	}
	return cs
}

// lineOf returns the line of the rune index, the last if past the end
func (l *TextLayout) lineOf(r int) int {
	for i, ln := range l.lines {
		if r <= ln.end {
			return i
		}
	}
	return len(l.lines) - 1
}

// runeIndex returns the rune of the byte offset
func (l *TextLayout) runeIndex(i int) int {
	return sort.SearchInts(l.offsets, max(0, min(i, len(l.text))))
}

// caretX returns the x of the caret before the rune r in the line,
// from the line's left
func (ln *layoutLine) caretX(r int) float64 {
	for _, c := range ln.clusters {
		if r >= c.start && r < c.start+c.count {
			f := float64(r-c.start) / float64(c.count)
			if c.rtl {
				return c.x1 - f*(c.x1-c.x0)
			}
			return c.x0 + f*(c.x1-c.x0)
		}
	}
	// After the last rune: the trailing edge of the one before
	for _, c := range ln.clusters {
		if r-1 >= c.start && r-1 < c.start+c.count {
			if c.rtl {
				return c.x0
			}
			return c.x1
		}
	}
	if ln.rtl {
		return 0
	}
	return ln.width
}

// CaretRect returns the caret before the character at the byte offset:
// its x, its top, and the height of its line
func (l *TextLayout) CaretRect(i int) RectF {
	l.layout()
	if len(l.lines) == 0 {
		return RectF{}
	}
	r := l.runeIndex(i)
	ln := &l.lines[l.lineOf(r)]
	x := ln.x + ln.caretX(r)
	return RectF{MinX: x, MinY: ln.y, MaxX: x, MaxY: ln.y + ln.height}
}

// IndexAt returns the byte offset of the caret nearest the point: the
// character under the pointer
func (l *TextLayout) IndexAt(x, y float64) int {
	l.layout()
	if len(l.lines) == 0 {
		return 0
	}
	li := len(l.lines) - 1
	for i, ln := range l.lines {
		if y < ln.y+ln.height {
			li = i
			break
		}
	}
	ln := &l.lines[li]
	x -= ln.x
	best, bestD := ln.start, math.Inf(1)
	for r := ln.start; r <= ln.end; r++ {
		if d := math.Abs(ln.caretX(r) - x); d < bestD {
			best, bestD = r, d
		}
	}
	return l.offsets[best]
}

// SelectionRects returns the rectangles that cover the text between the
// byte offsets, a few a line where right-to-left text is mixed in
func (l *TextLayout) SelectionRects(from, to int) []RectF {
	l.layout()
	a, b := l.runeIndex(min(from, to)), l.runeIndex(max(from, to))
	var out []RectF
	for _, ln := range l.lines {
		if ln.end < a || ln.start > b || a == b {
			continue
		}
		var spans [][2]float64
		for _, c := range ln.clusters {
			lo, hi := max(a, c.start), min(b, c.start+c.count)
			if lo >= hi {
				continue
			}
			f0, f1 := float64(lo-c.start)/float64(c.count), float64(hi-c.start)/float64(c.count)
			w := c.x1 - c.x0
			x0, x1 := c.x0+f0*w, c.x0+f1*w
			if c.rtl {
				x0, x1 = c.x1-f1*w, c.x1-f0*w
			}
			if n := len(spans); n > 0 && math.Abs(spans[n-1][1]-x0) < 0.01 {
				spans[n-1][1] = x1
			} else {
				spans = append(spans, [2]float64{x0, x1})
			}
		}
		// A line break selected: a bit of the line past its text
		if b > ln.end && ln.end < len(l.runes) && l.runes[ln.end] == '\n' {
			x := ln.caretX(ln.end)
			spans = append(spans, [2]float64{x, x + ln.height/4})
		}
		for _, s := range spans {
			out = append(out, RectF{MinX: ln.x + s[0], MinY: ln.y, MaxX: ln.x + s[1], MaxY: ln.y + ln.height})
		}
	}
	return out
}
