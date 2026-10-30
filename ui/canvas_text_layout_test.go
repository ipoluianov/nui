package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

// inkOf returns the sum of the darkness of the pixels of the white
// picture and the x of its center
func inkOf(img *image.RGBA, r image.Rectangle) (sum, cx float64) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			v := 255 - float64(img.RGBAAt(x, y).R)
			sum += v
			cx += v * float64(x)
		}
	}
	if sum > 0 {
		cx /= sum
	}
	return sum, cx
}

func newWhiteCanvas(w, h int, scale float64) (*Canvas, *image.RGBA) {
	cnv, img := newScaledTestCanvas(w, h, scale)
	cnv.FillRect(0, 0, w, h, white)
	cnv.SetColor(black)
	return cnv, img
}

func TestTextLayoutLines(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog, again and again."
	l := NewTextLayout(text, TextStyle{Size: 14})
	w, h := l.Size()
	if len(l.Lines()) != 1 || w < 300 || h < 14 || h > 24 {
		t.Fatalf("one line: %d lines, %v × %v", len(l.Lines()), w, h)
	}
	l.SetWidth(150)
	lines := l.Lines()
	if len(lines) < 3 {
		t.Fatalf("wrapped in %d lines", len(lines))
	}
	pos := 0
	for i, ln := range lines {
		if ln.Width > 150+0.01 {
			t.Errorf("line %d: %v wide", i, ln.Width)
		}
		if ln.Start != pos {
			t.Errorf("line %d starts at %d, want %d", i, ln.Start, pos)
		}
		if i > 0 && ln.Y <= lines[i-1].Y {
			t.Errorf("line %d not under the one before", i)
		}
		pos = ln.End
	}
	if pos != len(text) {
		t.Fatalf("the lines end at %d of %d", pos, len(text))
	}

	l = NewTextLayout("a\n\nb", TextStyle{})
	if n := len(l.Lines()); n != 3 {
		t.Fatalf("%d lines of a paragraph break", n)
	}
	if l.Lines()[1].Height <= 0 {
		t.Fatal("an empty line has no height")
	}
	if _, h := NewTextLayout("", TextStyle{}).Size(); h <= 0 {
		t.Fatal("no text: no height")
	}

	l = NewTextLayout(text+" "+text, TextStyle{})
	l.SetWidth(120)
	l.SetMaxLines(2)
	lines = l.Lines()
	if len(lines) != 2 {
		t.Fatalf("max 2 lines: %d", len(lines))
	}
}

func TestTextLayoutCaret(t *testing.T) {
	text := "Office fit, ﬁne: tea."
	l := NewTextLayout(text, TextStyle{Size: 20})
	prev := -1.0
	for i := range text {
		c := l.CaretRect(i)
		if c.MinX <= prev {
			t.Fatalf("the caret at %d (%v) not right of the one before (%v)", i, c.MinX, prev)
		}
		prev = c.MinX
		if j := l.IndexAt(c.MinX+0.01, c.MinY+5); j != i {
			t.Errorf("IndexAt the caret of %d: %d", i, j)
		}
	}
	w, _ := l.Size()
	if end := l.CaretRect(len(text)); math.Abs(end.MinX-w) > 0.01 {
		t.Fatalf("the caret at the end %v, the width %v", end.MinX, w)
	}
	if l.IndexAt(-10, 5) != 0 || l.IndexAt(w+10, 5) != len(text) {
		t.Fatal("the pointer out of the text")
	}
	rects := l.SelectionRects(0, len(text))
	if len(rects) != 1 || math.Abs(rects[0].MaxX-w) > 0.01 || rects[0].MinX != 0 {
		t.Fatalf("selection of all %v", rects)
	}

	l.SetWidth(80)
	rects = l.SelectionRects(0, len(text))
	if len(rects) != len(l.Lines()) {
		t.Fatalf("%d rectangles for %d lines", len(rects), len(l.Lines()))
	}
	// A caret on a wrapped line is on that line
	last := l.Lines()[len(l.Lines())-1]
	if c := l.CaretRect(last.Start + 1); c.MinY != last.Y {
		t.Fatalf("the caret on the last line at y %v, the line at %v", c.MinY, last.Y)
	}
}

func TestTextLayoutRightToLeft(t *testing.T) {
	ensureFallbackFonts("שלום")
	if !anyFontHas('ש') {
		t.Skip("no Hebrew font on this system")
	}
	text := "שלום עולם"
	l := NewTextLayout(text, TextStyle{Size: 20})
	w, _ := l.Size()
	if !l.Lines()[0].RightToLeft {
		t.Fatal("not right to left")
	}
	// The start of the text is at the right
	if c := l.CaretRect(0); math.Abs(c.MinX-w) > 0.01 {
		t.Fatalf("the caret of the start at %v, the width %v", c.MinX, w)
	}
	if c := l.CaretRect(len(text)); math.Abs(c.MinX) > 0.01 {
		t.Fatalf("the caret of the end at %v", c.MinX)
	}
	if l.IndexAt(w-0.5, 5) != 0 {
		t.Fatal("the pointer at the right is not at the start")
	}
	// Aligned to the right in a width by HAlignLeft: the start of its lines
	l.SetWidth(300)
	if x := l.Lines()[0].X; math.Abs(x+w-300) > 0.01 {
		t.Fatalf("right to left aligned at %v", x)
	}
}

func TestVisualOrder(t *testing.T) {
	// English, Hebrew, Arabic, digits within them, English
	got := visualOrder([]int{0, 1, 1, 2, 0})
	want := []int{0, 3, 2, 1, 4}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%v, want %v", got, want)
		}
	}
}

func TestTextLayoutBidi(t *testing.T) {
	ensureFallbackFonts("עברית العربية")
	if !anyFontHas('ע') || !anyFontHas('ع') {
		t.Skip("no Hebrew or Arabic font on this system")
	}
	text := "English עברית العربية 123 end"
	l := NewTextLayout(text, TextStyle{Size: 18})
	x := func(s string) float64 { return l.CaretRect(strings.Index(text, s)).MinX }
	// English, the digits, Arabic, Hebrew, English: left to right
	if !(x("English") < x("123") && x("123") < x("العربية") && x("العربية") < x("עברית") && x("עברית") < x("end")) {
		t.Fatalf("order: English %v, 123 %v, Arabic %v, Hebrew %v, end %v",
			x("English"), x("123"), x("العربية"), x("עברית"), x("end"))
	}
}

func TestDrawTextF(t *testing.T) {
	// As DrawText draws it
	a, ia := newWhiteCanvas(200, 30, 1)
	a.SetFontSize(16)
	a.DrawText(0, 0, 200, 30, "Hello, world")
	b, ib := newWhiteCanvas(200, 30, 1)
	b.SetFontSize(16)
	b.DrawTextF(0, 0, "Hello, world")
	sa, ca := inkOf(ia, ia.Rect)
	sb, cb := inkOf(ib, ib.Rect)
	if math.Abs(sa-sb)/sa > 0.06 || math.Abs(ca-cb) > 1 {
		t.Fatalf("DrawText ink %v at %v, DrawTextF %v at %v", sa, ca, sb, cb)
	}

	// Moved by a fraction of a pixel
	for _, dx := range []float64{0.25, 0.5, 0.75} {
		c, ic := newWhiteCanvas(200, 30, 1)
		c.SetFontSize(16)
		c.DrawTextF(dx, 0, "Hello, world")
		_, cc := inkOf(ic, ic.Rect)
		if math.Abs(cc-cb-dx) > 0.15 {
			t.Errorf("moved by %v: the center moved by %v", dx, cc-cb)
		}
	}

	// Turned: as much ink
	c, ic := newWhiteCanvas(200, 200, 1)
	c.SetFontSize(16)
	c.Translate(100, 100)
	c.Rotate(math.Pi / 4)
	c.SetHAlign(HAlignCenter)
	c.SetVAlign(VAlignCenter)
	c.DrawTextF(0, 0, "Hello, world")
	if s, _ := inkOf(ic, ic.Rect); math.Abs(s-sb)/sb > 0.08 {
		t.Fatalf("turned: ink %v, %v not turned", s, sb)
	}

	// Zoomed far: drawn, not dropped
	d, id := newWhiteCanvas(200, 200, 1)
	d.Translate(0, -300)
	d.ScaleBy(40, 40)
	d.DrawText(0, 0, 100, 20, "H")
	if s, _ := inkOf(id, id.Rect); s == 0 {
		t.Fatal("DrawText zoomed far is not drawn")
	}
}

func TestTextLayoutLooks(t *testing.T) {
	l := NewTextLayout("mark", TextStyle{Size: 20, Background: color.RGBA{255, 220, 0, 255}, Underline: true,
		Color: color.RGBA{200, 0, 0, 255}})
	cnv, img := newWhiteCanvas(100, 40, 2)
	cnv.DrawTextLayout(l, 5, 5)
	w, h := l.Size()
	if c := at(img, int(2*(5+w-1)), int(2*(5+2))); c != (color.RGBA{255, 220, 0, 255}) {
		t.Fatalf("background %v", c)
	}
	red := 0
	for y := 0; y < img.Rect.Dy(); y++ {
		if c := at(img, int(2*(5+w/2)), y); c.R > 150 && c.G < 60 {
			red++
		}
	}
	if red == 0 {
		t.Fatal("no underline across the middle")
	}
	_ = h

	// Recorded as SVG: the outlines of the letters
	cnv, _ = newWhiteCanvas(100, 40, 1)
	rec := NewSVGRecorder(100, 40)
	cnv.SetRecorder(rec)
	cnv.DrawTextF(5, 5, "SVG")
	if svg := rec.String(); strings.Count(svg, "<path") < 1 {
		t.Fatalf("%s", svg)
	}
}

func BenchmarkDrawText(b *testing.B) {
	cnv, _ := newWhiteCanvas(300, 30, 1)
	cnv.SetFontSize(14)
	for b.Loop() {
		cnv.DrawText(0, 0, 300, 30, "The quick brown fox jumps")
	}
}

func BenchmarkDrawTextF(b *testing.B) {
	cnv, _ := newWhiteCanvas(300, 30, 1)
	cnv.SetFontSize(14)
	for b.Loop() {
		cnv.DrawTextF(0.5, 0, "The quick brown fox jumps")
	}
}

func BenchmarkTextLayoutShape(b *testing.B) {
	for b.Loop() {
		l := NewTextLayout("The quick brown fox jumps over the lazy dog", TextStyle{})
		l.SetWidth(150)
		l.Size()
	}
}

func TestTextConcurrent(t *testing.T) {
	done := make(chan bool)
	for g := 0; g < 4; g++ {
		go func() {
			for i := 0; i < 20; i++ {
				cnv, _ := newWhiteCanvas(200, 40, 1+float64(i%3)/2)
				cnv.SetFontSize(12 + float64(i%4))
				cnv.DrawTextF(float64(i)/7, 0, "Concurrent text, fi ffl")
				l := NewTextLayout("Layout of its own", TextStyle{Bold: i%2 == 0})
				l.SetWidth(60)
				cnv.DrawTextLayout(l, 0, 20)
			}
			done <- true
		}()
	}
	for g := 0; g < 4; g++ {
		<-done
	}
}
