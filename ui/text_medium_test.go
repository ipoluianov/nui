package ui

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"strings"
	"testing"
	"time"
)

// txtNuiGlyphs draws the text of the test with nui's own rasterizer, hinted or
// not, as uitest does
func txtNuiGlyphs(t *testing.T, hinting bool) {
	native, hint := NativeFontRendering, TextHinting
	NativeFontRendering, TextHinting = false, hinting
	t.Cleanup(func() { NativeFontRendering, TextHinting = native, hint })
}

// txtInkCount counts the pixels of the picture that are not white
func txtInkCount(img *image.RGBA) int {
	n := 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 255 || img.Pix[i+1] != 255 || img.Pix[i+2] != 255 {
			n++
		}
	}
	return n
}

func txtWhiteRGBA(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	return img
}

// TXT-5: text larger than the masks are made for is filled as outlines
// within the clip: no mask as large as the glyph is made or kept, and the
// memory does not grow with the size. Absurd sizes are not drawn.
func TestDrawTextHugeFontSizeBounded(t *testing.T) {
	for _, hinting := range []bool{true, false} {
		txtNuiGlyphs(t, hinting)
		for _, size := range []float64{16000, 60000, 1e6, 1e9, math.Inf(1), math.NaN(), -5} {
			glyphMasksMu.Lock()
			clear(glyphMasks)
			glyphMasksBytes = 0
			glyphMasksMu.Unlock()
			runtime.GC()
			var m0, m1 runtime.MemStats
			runtime.ReadMemStats(&m0)
			cnv := txtCanvas(image.NewRGBA(image.Rect(0, 0, 100, 50)))
			cnv.SetFontSize(size)
			cnv.DrawText(0, 0, 100, 50, "ab")
			runtime.ReadMemStats(&m1)
			if mb := (m1.TotalAlloc - m0.TotalAlloc) >> 20; mb > 20 {
				t.Errorf("hinting %v, size %v: DrawText of \"ab\" on 100x50 allocated %d MB", hinting, size, mb)
			}
			glyphMasksMu.Lock()
			n := len(glyphMasks)
			glyphMasksMu.Unlock()
			if n != 0 {
				t.Errorf("hinting %v, size %v: %d glyph masks made", hinting, size, n)
			}
		}
	}
}

// TXT-5: a large glyph drawn as an outline is drawn where its mask would
// be: the ink of a size just over maxMaskSize covers about what the mask
// just under it covers, and an "H" seen through a small clip is drawn
func TestDrawTextLargeGlyphOutlines(t *testing.T) {
	for _, hinting := range []bool{true, false} {
		txtNuiGlyphs(t, hinting)
		draw := func(size float64) *image.RGBA {
			img := txtWhiteRGBA(400, 400)
			cnv := txtCanvas(img)
			cnv.SetColor(color.RGBA{0, 0, 0, 255})
			cnv.SetFontSize(size)
			cnv.DrawText(10, 10, 390, 390, "H")
			return img
		}
		under, over := draw(maxMaskSize-1), draw(maxMaskSize+1)
		a, b := txtInkCount(under), txtInkCount(over)
		if a == 0 || b < a*9/10 || b > a*12/10 {
			t.Errorf("hinting %v: ink of \"H\" at %d px (mask) %d, at %d px (outline) %d", hinting, maxMaskSize-1, a, maxMaskSize+1, b)
		}
		// The rows and columns of the ink are within a few pixels
		box := func(img *image.RGBA) image.Rectangle {
			r := image.Rectangle{}
			for y := 0; y < 400; y++ {
				for x := 0; x < 400; x++ {
					if img.Pix[img.PixOffset(x, y)] < 128 {
						r = r.Union(image.Rect(x, y, x+1, y+1))
					}
				}
			}
			return r
		}
		ba, bb := box(under), box(over)
		if abs(ba.Min.X-bb.Min.X) > 3 || abs(ba.Min.Y-bb.Min.Y) > 4 || abs(ba.Max.Y-bb.Max.Y) > 4 {
			t.Errorf("hinting %v: ink of the mask %v, of the outline %v", hinting, ba, bb)
		}
	}
}

// Root cause 3 (CHART-9, MISC-2): text at a place near ±2^63 (int(NaN) of
// a caller, a far point converted to int) is not drawn and does not panic
func TestDrawTextFarCoordinates(t *testing.T) {
	nan := math.NaN()
	far := []int{int(nan), math.MinInt64, math.MaxInt64, math.MaxInt64 - 150, math.MinInt64 + 150, int(math.Round(nan)) - 150, 1 << 40, -(1 << 40)}
	for _, hinting := range []bool{true, false} {
		txtNuiGlyphs(t, hinting)
		for _, text := range []string{"x", "label", "😀 שלום"} {
			for _, v := range far {
				for _, p := range [][2]int{{v, 10}, {10, v}, {v, v}} {
					img := txtWhiteRGBA(64, 64)
					func() {
						defer func() {
							if r := recover(); r != nil {
								t.Errorf("hinting %v, DrawText(%q) at %v: panic %v", hinting, text, p, r)
							}
						}()
						DrawText(img, text, color.Black, FontFamilySans, 12, p[0], p[1], 0, 0, 64, 64)
						cnv := txtCanvas(img)
						cnv.DrawText(p[0], p[1], 300, 20, text)
						l := NewTextLayout(text, TextStyle{Size: 12})
						cnv.DrawTextLayout(l, float64(p[0]), float64(p[1]))
					}()
					if n := txtInkCount(img); n != 0 {
						t.Errorf("hinting %v, DrawText(%q) at %v: %d pixels drawn", hinting, text, p, n)
					}
				}
			}
		}
	}
	// The blitter itself
	m := &glyphMask{x: -2, y: -10, w: 8, h: 12, pix: make([]uint8, 8*12)}
	img := txtWhiteRGBA(64, 64)
	for _, v := range far {
		blendGlyphMask(img, img.Rect, m, v, 20, 0, 0, 0, 255, 0)
		blendGlyphMask(img, img.Rect, m, 20, v, 0, 0, 0, 255, 0)
	}
}

// TXT-6: stepping a value too large to round to the decimals keeps it
// finite, and an infinite value is ignored as NaN is
func TestNumBoxStaysFinite(t *testing.T) {
	n := NewNumBox()
	form := newTestForm(t, n)
	n.SetDecimals(2)
	n.SetValue(1e307)
	n.Focus()
	typeKey(form, KeyArrowUp, 0, KeyModifiers{})
	if v := n.Value(); v != 1e307 {
		t.Errorf("decimals 2, value 1e307, Up: value %v (text %q), want 1e307", v, n.Text())
	}
	n.SetValue(math.MaxFloat64)
	n.SetStep(math.MaxFloat64)
	typeKey(form, KeyArrowUp, 0, KeyModifiers{})
	if v := n.Value(); math.IsInf(v, 0) || math.IsNaN(v) {
		t.Errorf("MaxFloat64 stepped up by MaxFloat64: value %v", v)
	}
	n.SetValue(5)
	for _, v := range []float64{math.Inf(1), math.Inf(-1), math.NaN()} {
		n.SetValue(v)
		if n.Value() != 5 {
			t.Errorf("SetValue(%v): value %v (text %q), want 5 kept", v, n.Value(), n.Text())
		}
		n.SetProp("value", v)
		if n.Value() != 5 {
			t.Errorf("SetProp(\"value\", %v): value %v, want 5 kept", v, n.Value())
		}
	}
}

// TXT-7: a click anywhere on a line puts the cursor at a nearest caret: no
// pixel between two characters is left out
func TestTextBoxClickEveryPixel(t *testing.T) {
	box := NewTextBox()
	newTestForm(t, box)
	line := "line 52 Wim .,;il|"
	box.SetText(line)
	box.Focus()
	cp, _ := GetCharPositions(box.FontFamily(), box.FontSize(), line)
	lx := box.lineX(0)
	for x := lx - 3; x < lx+cp[len(cp)-1]+3; x++ {
		// The cursor starts far from the click, at the other end
		start := 0
		if x-lx < cp[len(cp)-1]/2 {
			start = len(cp) - 1
		}
		box.SetCursorPosition(TextPos{Col: start})
		box.MouseDown(MouseButtonLeft, x, 5, KeyModifiers{})
		box.MouseUp(MouseButtonLeft, x, 5, KeyModifiers{})
		got := box.CursorPosition()
		best := math.MaxInt
		for _, p := range cp {
			best = min(best, abs(p-(x-lx)))
		}
		if got.Line != 0 || abs(cp[got.Col]-(x-lx)) != best {
			t.Fatalf("click at x %d (caret xs %v + %d): cursor %+v, %d px off, nearest %d px", x, cp, lx, got, abs(cp[got.Col]-(x-lx)), best)
		}
	}
}

// TXT-7: the same for the carets of a NumBox, and for the zones themselves
func TestNearestCaretNoGaps(t *testing.T) {
	cases := [][]int{{0}, {0, 9}, {0, 4, 8, 17, 25, 29, 37, 45}, {3, 4, 5, 12, 13, 21}}
	for _, cp := range cases {
		for x := cp[0] - 5; x < cp[len(cp)-1]+5; x++ {
			i := nearestCaret(cp, x)
			best := math.MaxInt
			for _, p := range cp {
				best = min(best, abs(p-x))
			}
			if abs(cp[i]-x) != best {
				t.Errorf("carets %v, x %d: caret %d (%d px off), nearest %d px", cp, x, i, abs(cp[i]-x), best)
			}
		}
	}
	n := NewNumBox()
	newTestForm(t, n)
	n.SetDecimals(3)
	n.SetValue(-1234.517)
	cp, _ := GetCharPositions(n.FontFamily(), n.FontSize(), n.text)
	for x := -3; x < cp[len(cp)-1]+3; x++ {
		i := n.cursorPosFromPixel(x)
		best := math.MaxInt
		for _, p := range cp {
			best = min(best, abs(p-x))
		}
		if i < 0 || i >= len(cp) || abs(cp[i]-x) != best {
			t.Errorf("NumBox %q, x %d: cursor %d, nearest caret %d px away", n.text, x, i, best)
		}
	}
}

// TXT-8: a single-line box holds one line: the breaks of SetText are
// dropped, as typing drops them, and a box made single-line joins its lines
func TestTextBoxSingleLineSetText(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("visible\nhidden\r\nmore")
	box.Focus()
	typeKey(form, KeyArrowDown, 0, KeyModifiers{})
	typeText(form, "X")
	if box.LineCount() != 1 || strings.ReplaceAll(box.Text(), "X", "") != "visiblehiddenmore" {
		t.Errorf("single-line box after SetText with breaks, Down, X: %d lines, %q", box.LineCount(), box.Text())
	}
	box.SetProp("text", "a\nb")
	if box.LineCount() != 1 || box.Text() != "ab" {
		t.Errorf("SetProp(\"text\", \"a\\nb\"): %d lines, %q", box.LineCount(), box.Text())
	}

	multi := NewTextBox()
	newTestForm(t, multi)
	multi.SetMultiline(true)
	multi.SetText("one\ntwo")
	if multi.LineCount() != 2 {
		t.Errorf("multiline box: %d lines, want 2", multi.LineCount())
	}
	multi.SetMultiline(false)
	if multi.LineCount() != 1 || multi.Text() != "onetwo" {
		t.Errorf("box made single-line: %d lines, %q", multi.LineCount(), multi.Text())
	}

	cb := NewEditableComboBox()
	newTestForm(t, cb)
	cb.SetText("x\ny")
	if cb.Text() != "xy" {
		t.Errorf("EditableComboBox.SetText(\"x\\ny\"): %q", cb.Text())
	}
}

// TXT-12: the carets of a line at once are those of caretX one by one
func TestCaretXsMatchCaretX(t *testing.T) {
	texts := []string{"😀 abc", "שלום world", "éx́ y", "abc שלום def 😀", "a\nשלום\n\nb😀", "", "ffi fl"}
	for _, text := range texts {
		l := NewTextLayout(text, TextStyle{Size: 13})
		l.layout()
		for li := range l.lines {
			ln := &l.lines[li]
			xs := ln.caretXs()
			for r := ln.start; r <= ln.end; r++ {
				if got, want := xs[r-ln.start], ln.caretX(r); got != want {
					t.Errorf("%q line %d rune %d: caretXs %v, caretX %v", text, li, r, got, want)
				}
			}
		}
		all := l.caretXsAll()
		for r := 0; r <= len(l.runes); r++ {
			if got, want := all[r], l.CaretRect(l.offsets[r]).MinX; got != want {
				t.Errorf("%q rune %d: caretXsAll %v, CaretRect %v", text, r, got, want)
			}
		}
		if !strings.Contains(text, "\n") {
			xs := textCaretXs(FontFamilySans, 12, text)
			for i := range xs {
				if want := TextCaretX(FontFamilySans, 12, text, i); xs[i] != want {
					t.Errorf("%q: textCaretXs[%d] %d, TextCaretX %d", text, i, xs[i], want)
				}
			}
		}
	}
}

// TXT-12: a click in a long shaped line costs time linear in its length,
// not quadratic (64000 runes took seconds)
func TestTextIndexAtLongShapedLine(t *testing.T) {
	line := "😀" + strings.Repeat("a", 64000)
	TextCaretX(FontFamilySans, 12, line, 0) // laid out once, cached
	t0 := time.Now()
	i := TextIndexAt(FontFamilySans, 12, line, 1000)
	if d := time.Since(t0); d > 300*time.Millisecond {
		t.Errorf("TextIndexAt on a %d-rune shaped line: %v", len(line), d)
	}
	if x := TextCaretX(FontFamilySans, 12, line, i); abs(x-1000) > 10 {
		t.Errorf("TextIndexAt(1000) = %d, its caret at %d", i, x)
	}
}

// txtCanvas is a canvas of the whole picture
func txtCanvas(img *image.RGBA) *Canvas {
	cnv := NewCanvasScaled(img, 1)
	cnv.SetDirectTranslateAndClip(0, 0, img.Rect.Dx(), img.Rect.Dy())
	return cnv
}
