package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
	"unicode/utf8"
)

// haveScripts skips the test if the system has no font of the texts
func haveScripts(t *testing.T, texts ...string) {
	t.Helper()
	for _, text := range texts {
		ensureFallbackFonts(text)
		for _, r := range text {
			if r > 0x0590 && !anyFontHas(r) && !isCJKRune(r) {
				t.Skipf("no font of %q on this system", r)
			}
		}
	}
}

func TestNeedsShaping(t *testing.T) {
	for text, want := range map[string]bool{
		"Hello, world": false, "Привет": false, "Γειά": false, "中文 日本語": false, "": false, "→ ★ ✓": false,
		"שלום": true, "مرحبا": true, "नमस्ते": true, "ไทย": true, "é": true, "a‏b": true,
		"Hello עולם": true,
	} {
		if got := needsShaping(text); got != want {
			t.Errorf("%q: %v, want %v", text, got, want)
		}
	}
}

func TestGraphemes(t *testing.T) {
	check := func(text string, want []int) {
		t.Helper()
		got := graphemeBounds(text)
		if len(got) != len(want) {
			t.Fatalf("%q: %v, want %v", text, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%q: %v, want %v", text, got, want)
			}
		}
	}
	check("abc", []int{0, 1, 2, 3})
	check("éx", []int{0, 2, 3})
	// न म स्ते: the conjunct with its vowel sign is one
	check("नमस्ते", []int{0, 1, 2, 6})
	if nextGrapheme("नमस्ते", 2) != 6 || prevGrapheme("नमस्ते", 6) != 2 || prevGrapheme("नमस्ते", 0) != 0 {
		t.Fatal("next and previous graphemes")
	}
	if snapToGrapheme("नमस्ते", 4) != 2 && snapToGrapheme("नमस्ते", 4) != 6 {
		t.Fatal("snapped into a grapheme")
	}
}

func TestShapedMeasureAndPositions(t *testing.T) {
	haveScripts(t, "שלום", "مرحبا")
	family, size := FontFamilySans, 16.0
	for _, text := range []string{"שלום עולם", "مرحبا بالعالم", "abc שלום def"} {
		w, h, err := MeasureText(family, size, text)
		lw, _ := shapedLine(text, family, size*textScale()).Size()
		if err != nil || math.Abs(float64(w)-lw/textScale()) > 1 {
			t.Errorf("%q: width %d, the layout %v", text, w, lw)
		}
		if _, h0, _ := MeasureText(family, size, "Ag"); h != h0 {
			t.Errorf("%q: height %d, of Latin %d", text, h, h0)
		}
		pos, _ := GetCharPositions(family, size, text)
		if len(pos) != utf8.RuneCountInString(text)+1 || pos[len(pos)-1] != w {
			t.Errorf("%q: %d positions, the last %d, the width %d", text, len(pos), pos[len(pos)-1], w)
		}
	}
	// Hebrew: the start at the right, the end at the left; the positions
	// end with the width, as for any text
	pos, _ := GetCharPositions(family, size, "שלום")
	w, _, _ := MeasureText(family, size, "שלום")
	if abs(pos[0]-w) > 1 || pos[4] != w || pos[3] >= pos[2] {
		t.Fatalf("right to left positions %v, width %d", pos, w)
	}
	if x := TextCaretX(family, size, "שלום", 4); x != 0 {
		t.Fatalf("the caret at the end of Hebrew at %d, not at the left", x)
	}
	if x := TextCaretX(family, size, "abc", 3); x != pos3("abc") {
		t.Fatalf("the caret at the end of Latin at %d", x)
	}
}

func TestTextIndexAtAndSpans(t *testing.T) {
	haveScripts(t, "שלום")
	family, size := FontFamilySans, 15.0
	for _, text := range []string{"Hello world", "שלום עולם", "abc שלום def", "नमस्ते दुनिया"} {
		caret := func(i int) int { return TextCaretX(family, size, text, i) }
		for _, i := range graphemeBounds(text) {
			j := TextIndexAt(family, size, text, caret(i))
			// At the meeting of two directions two indexes share a place
			if abs(caret(j)-caret(i)) > 1 {
				t.Errorf("%q: the click at the caret of %d (x %d) gives %d (x %d)", text, i, caret(i), j, caret(j))
			}
		}
		w, _, _ := MeasureText(family, size, text)
		spans := TextSelectionSpans(family, size, text, 0, utf8.RuneCountInString(text))
		total := 0
		for _, s := range spans {
			if s[1] <= s[0] {
				t.Errorf("%q: an empty span %v", text, s)
			}
			total += s[1] - s[0]
		}
		if abs(total-w) > len(spans) {
			t.Errorf("%q: the selection of all is %d wide in %v, the text %d", text, total, spans, w)
		}
	}
	// Part of a mixed line: the selection of the Hebrew word is within its
	// place, not from its logical start to the end of the line
	text := "abc שלום def"
	spans := TextSelectionSpans(family, size, text, 4, 8)
	left, right := TextCaretX(family, size, text, 3), TextCaretX(family, size, text, 9)
	if len(spans) != 1 || spans[0][0] < left || spans[0][1] > right {
		t.Fatalf("the Hebrew word: %v, between %d and %d", spans, left, right)
	}
}

// drawLine draws the text with DrawText on a white picture
func drawLine(text string, size float64, scale float64) *image.RGBA {
	cnv, img := newWhiteCanvas(240, 40, scale)
	cnv.SetFontSize(size)
	cnv.DrawText(4, 0, 230, 40, text)
	return img
}

func TestDrawTextShaped(t *testing.T) {
	haveScripts(t, "مرحبا")
	// Drawn as the layout draws it: joined
	for _, scale := range []float64{1, 1.5, 2} {
		img := drawLine("مرحبا", 16, scale)
		cnv, want := newWhiteCanvas(240, 40, scale)
		l := shapedLine("مرحبا", FontFamilySans, 16*scale)
		base := textBaseline(FontFamilySans, 16*scale, cnv.scaled(0))
		pc := &Canvas{rgba: want, scale: 1, state: &canvasState{col: black, clipW: want.Rect.Dx(), clipH: want.Rect.Dy()}}
		pc.DrawTextLayout(l, float64(cnv.scaled(4)), float64(base)-l.lines[0].ascent)
		if string(img.Pix) != string(want.Pix) {
			t.Fatalf("scale %v: DrawText of Arabic is not the shaped line", scale)
		}
	}
	// Joined Arabic is narrower than its letters one by one
	shaped, _, _ := MeasureText(FontFamilySans, 16, "مرحبا")
	sum := 0
	for _, r := range "مرحبا" {
		w, _, _ := MeasureText(FontFamilySans, 16, string(r))
		sum += w
	}
	if shaped >= sum {
		t.Fatalf("joined %d, the letters apart %d", shaped, sum)
	}
}

func TestShapedTextOnTheBaseline(t *testing.T) {
	haveScripts(t, "שלום")
	// The A of a mixed line where the A alone is: the same baseline, the
	// same place
	bottomOf := func(img *image.RGBA, x0, x1 int) (int, int) {
		bottom, left := -1, img.Rect.Dx()
		for y := 0; y < img.Rect.Dy(); y++ {
			for x := x0; x < x1; x++ {
				if img.RGBAAt(x, y).R < 128 {
					bottom, left = y, min(left, x)
				}
			}
		}
		return bottom, left
	}
	for _, scale := range []float64{1, 2} {
		a := drawLine("A", 16, scale)
		b := drawLine("A שלום", 16, scale)
		w := int(14 * scale)
		ba, la := bottomOf(a, 0, w)
		bb, lb := bottomOf(b, 0, w)
		if ba != bb || abs(la-lb) > 1 {
			t.Errorf("scale %v: the A alone ends at row %d, col %d; in the shaped line at %d, %d", scale, ba, la, bb, lb)
		}
	}
}

func TestTextBoxRightToLeft(t *testing.T) {
	haveScripts(t, "שלום")
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("abc שלום def")
	box.Focus()

	// The arrows go over the characters in the order of the text
	box.SetCursorPosition(TextPos{})
	for i := 1; i <= 12; i++ {
		typeKey(form, KeyArrowRight, 0, KeyModifiers{})
		if box.cursorPosX != i {
			t.Fatalf("Right %d: at %d", i, box.cursorPosX)
		}
	}
	// The caret where the layout puts it: in the Hebrew word, the start of
	// the word at its right
	text := box.Text()
	pos := make([]int, 13)
	for i := range pos {
		pos[i] = TextCaretX(box.FontFamily(), box.FontSize(), text, i)
	}
	if !(pos[4] > pos[5] && pos[5] > pos[6]) {
		t.Fatalf("the carets in the Hebrew word are not right to left: %v", pos[3:9])
	}
	box.SetCursorPosition(TextPos{Col: 5})
	x, _, _, _ := box.caretRect()
	if x != box.textX()+pos[5] {
		t.Fatalf("the caret at %d, the layout at %d", x, box.textX()+pos[5])
	}

	// A click lands on the character under it
	for _, i := range []int{0, 2, 5, 7, 10, 12} {
		box.MouseDown(MouseButtonLeft, box.textX()+pos[i], box.Height()/2, KeyModifiers{})
		box.MouseUp(MouseButtonLeft, box.textX()+pos[i], box.Height()/2, KeyModifiers{})
		if got := box.cursorPosX; abs(pos[got]-pos[i]) > 1 {
			t.Errorf("a click at the caret of %d gives %d", i, got)
		}
	}

	// The selection of the Hebrew word is painted over the word only
	box.SetSelection(TextPos{Col: 4}, TextPos{Col: 8})
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	form.paintScaled(img, 1)
	sel := CurrentPalette().Selection
	ox, oy := box.RectClientAreaOnWindow()
	selected := func(x int) bool {
		c := img.RGBAAt(ox+box.textX()+x, oy+box.Height()/2)
		return math.Abs(float64(c.R)-float64(sel.R))+math.Abs(float64(c.G)-float64(sel.G))+math.Abs(float64(c.B)-float64(sel.B)) < 30
	}
	lo, hi := min(pos[3], pos[9]), max(pos[3], pos[9])
	if selected(pos[1]) || selected(pos[11]) {
		t.Error("the selection is painted over the Latin words")
	}
	marked := 0
	for x := lo; x < hi; x++ {
		if selected(x) {
			marked++
		}
	}
	if marked < (hi-lo)/3 {
		t.Errorf("the Hebrew word %d..%d: %d columns marked", lo, hi, marked)
	}
	if box.SelectedText() != "שלום" {
		t.Errorf("selected %q", box.SelectedText())
	}
}

func TestTextBoxGraphemes(t *testing.T) {
	haveScripts(t, "नमस्ते")
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("नमस्ते")
	box.Focus()
	box.SetCursorPosition(TextPos{})
	var stops []int
	for i := 0; i < 5; i++ {
		typeKey(form, KeyArrowRight, 0, KeyModifiers{})
		stops = append(stops, box.cursorPosX)
	}
	if want := []int{1, 2, 6, 6, 6}; !equalInts(stops, want) {
		t.Fatalf("Right stops at %v, want %v", stops, want)
	}
	typeKey(form, KeyArrowLeft, 0, KeyModifiers{})
	if box.cursorPosX != 2 {
		t.Fatalf("Left from the end: %d", box.cursorPosX)
	}
	// Delete takes the conjunct with its vowel sign
	typeKey(form, KeyDelete, 0, KeyModifiers{})
	if box.Text() != "नम" {
		t.Fatalf("Delete: %q", box.Text())
	}
	// Backspace takes a code point, as the system editors do
	box.SetText("é")
	box.MoveCursorToEnd()
	typeKey(form, KeyBackspace, 0, KeyModifiers{})
	if box.Text() != "e" {
		t.Fatalf("Backspace: %q", box.Text())
	}
	// Up and down land at the start of a grapheme
	box.SetMultiline(true)
	box.SetText("abcdef\nनमस्ते")
	box.SetCursorPosition(TextPos{Line: 0, Col: 4})
	typeKey(form, KeyArrowDown, 0, KeyModifiers{})
	if b := graphemeBounds("नमस्ते"); !containsInt(b, box.cursorPosX) {
		t.Fatalf("Down into the middle of a grapheme: %d", box.cursorPosX)
	}
}

func TestTextBoxHighlightedShaped(t *testing.T) {
	haveScripts(t, "مرحبا")
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetMultiline(true)
	box.SetText("key = مرحبا")
	red := color.RGBA{220, 0, 0, 255}
	box.SetHighlighter(func(line int, text string) []TextSpan {
		return []TextSpan{{Start: 6, End: 11, Color: red, Background: color.RGBA{255, 240, 160, 255}}}
	})
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	form.paintScaled(img, 1)
	// The Arabic word is red, on its background, where the layout puts it
	ox, oy := box.RectClientAreaOnWindow()
	word := TextSelectionSpans(box.FontFamily(), box.FontSize(), "key = مرحبا", 6, 11)
	if len(word) != 1 {
		t.Fatalf("the Arabic word in %d pieces", len(word))
	}
	lo, hi := word[0][0], word[0][1]
	reds, backs := 0, 0
	for y := oy; y < oy+box.OneLineHeight()+2; y++ {
		for x := ox + box.textX() + lo; x < ox+box.textX()+hi; x++ {
			c := img.RGBAAt(x, y)
			if c.R > 150 && c.G < 80 && c.B < 80 {
				reds++
			}
			if c == (color.RGBA{255, 240, 160, 255}) {
				backs++
			}
		}
	}
	if reds < 10 || backs < 10 {
		t.Fatalf("the highlighted Arabic word: %d red pixels, %d of the background", reds, backs)
	}
}

func TestLabelShaped(t *testing.T) {
	haveScripts(t, "مرحبا")
	l := NewLabel("مرحبا بالعالم")
	newTestForm(t, l)
	w, _, _ := MeasureText(l.FontFamily(), l.FontSize(), "مرحبا بالعالم")
	if l.MinWidth() < w {
		t.Fatalf("the label is %d wide, its text %d", l.MinWidth(), w)
	}
}

func pos3(text string) int {
	w, _, _ := MeasureText(FontFamilySans, 16, text)
	return w
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsInt(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

// On a scaled screen the carets, the clicks and the widths agree, measured
// in its pixels
func TestShapedOnScaledScreen(t *testing.T) {
	haveScripts(t, "שלום", "नमस्ते")
	defer textScaleBits.Store(textScaleBits.Load())
	for _, scale := range []float64{1.25, 1.5, 2} {
		textScaleBits.Store(math.Float64bits(scale))
		for _, text := range []string{"नमस्ते दुनिया", "abc שלום def"} {
			w, _, _ := MeasureText(FontFamilySans, 14, text)
			pos, _ := GetCharPositions(FontFamilySans, 14, text)
			if pos[len(pos)-1] != w {
				t.Errorf("scale %v %q: the last position %d, the width %d", scale, text, pos[len(pos)-1], w)
			}
			for _, i := range graphemeBounds(text) {
				x := TextCaretX(FontFamilySans, 14, text, i)
				if x < 0 || x > w+1 {
					t.Errorf("scale %v %q: the caret of %d at %d, out of 0..%d", scale, text, i, x, w)
				}
				j := TextIndexAt(FontFamilySans, 14, text, x)
				if abs(TextCaretX(FontFamilySans, 14, text, j)-x) > 1 {
					t.Errorf("scale %v %q: the click at the caret of %d gives %d", scale, text, i, j)
				}
			}
		}
	}
}

// A script below Hebrew the built-in fonts lack - Armenian - comes from
// the system too
func TestFontBelowHebrew(t *testing.T) {
	if !SystemFallbackFonts {
		t.Skip()
	}
	ensureFallbackFonts("Բարեւ")
	if !anyFontHas('Բ') {
		if p := fontconfigMatch(":charset=531"); p == "" {
			t.Skip("no Armenian font on this system")
		}
		t.Fatal("the Armenian font of the system is not taken")
	}
	// Drawn as letters, not as the boxes of the missing glyph: the boxes
	// of different letters are the same
	a, b := drawLine("Բ", 20, 1), drawLine("Ր", 20, 1)
	if string(a.Pix) == string(b.Pix) {
		t.Fatal("two Armenian letters drawn the same")
	}
}

func TestTextBoxAlign(t *testing.T) {
	haveScripts(t, "שלום")
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("שלום עולם")
	box.Focus()
	w, _, _ := MeasureText(box.FontFamily(), box.FontSize(), box.Text())
	right := box.Width() - box.leftAndRightPadding
	if x := box.lineX(0); x != box.textX() {
		t.Fatalf("aligned to the left by default: %d", x)
	}
	for _, tc := range []struct {
		align TextAlign
		text  string
		want  int // the start of the line
	}{
		{TextAlignStart, "שלום עולם", right - w},
		{TextAlignRight, "שלום עולם", right - w},
		{TextAlignCenter, "שלום עולם", box.textX() + (right-box.textX()-w)/2},
		{TextAlignStart, "Hello", box.textX()},
	} {
		box.SetText(tc.text)
		box.SetTextAlign(tc.align)
		if x := box.lineX(0); abs(x-tc.want) > 1 {
			t.Errorf("%v %q: the line at %d, want %d", tc.align, tc.text, x, tc.want)
		}
	}
	// Right to left at the right: the cursor at the start of the text is
	// at the right edge, a click lands where the characters are
	box.SetText("abc שלום def")
	box.SetTextAlign(TextAlignRight)
	text := box.Text()
	lx := box.lineX(0)
	for _, i := range []int{0, 3, 5, 8, 12} {
		x := lx + TextCaretX(box.FontFamily(), box.FontSize(), text, i)
		box.MouseDown(MouseButtonLeft, x, box.Height()/2, KeyModifiers{})
		box.MouseUp(MouseButtonLeft, x, box.Height()/2, KeyModifiers{})
		if got := box.cursorPosX; abs(TextCaretX(box.FontFamily(), box.FontSize(), text, got)+lx-x) > 1 {
			t.Errorf("a click at the caret of %d gives %d", i, got)
		}
		if cx, _, _, _ := box.caretRect(); abs(cx-x) > 1 && box.cursorPosX == i {
			t.Errorf("the caret of %d at %d, want %d", i, cx, x)
		}
	}
	// The selection painted where the line is
	box.SetSelection(TextPos{Col: 0}, TextPos{Col: 3})
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	form.paintScaled(img, 1)
	ox, oy := box.RectClientAreaOnWindow()
	sel := CurrentPalette().Selection
	at := img.RGBAAt(ox+lx+TextCaretX(box.FontFamily(), box.FontSize(), text, 1), oy+box.Height()/2)
	if math.Abs(float64(at.B)-float64(sel.B)) > 40 {
		t.Errorf("the selection of abc is not under abc: %v", at)
	}

	// A multiline box: each line at the side of its direction
	area := NewTextBox()
	newTestForm(t, area)
	area.SetMultiline(true)
	area.SetText("Hello\nשלום\n")
	area.SetTextAlign(TextAlignStart)
	if area.lineX(0) != area.textX() || area.lineX(1) <= area.textX() || area.lineX(2) != area.textX() {
		t.Fatalf("the lines at %d, %d, %d", area.lineX(0), area.lineX(1), area.lineX(2))
	}
}

func TestTextBoxVisualArrows(t *testing.T) {
	haveScripts(t, "שלום")
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("abc שלום def")
	box.Focus()
	caret := func() int { x, _, _, _ := box.caretRect(); return x }

	// In the order of the text: Left from the start of the Hebrew word goes
	// back to the space before it
	box.SetCursorPosition(TextPos{Col: 4})
	typeKey(form, KeyArrowLeft, 0, KeyModifiers{})
	if box.cursorPosX != 3 {
		t.Fatalf("logical Left: %d", box.cursorPosX)
	}
	// On the screen: Left always moves to the left, through all the line
	box.SetVisualCursorMovement(true)
	box.SetCursorPosition(TextPos{Col: 12})
	prev := caret()
	for k := 0; k < 14; k++ {
		typeKey(form, KeyArrowLeft, 0, KeyModifiers{})
		x := caret()
		if x > prev {
			t.Fatalf("Left %d moved right: %d -> %d", k, prev, x)
		}
		prev = x
	}
	if prev != box.textX() {
		t.Fatalf("Left does not reach the left edge: %d", prev)
	}
	for k := 0; k < 14; k++ {
		typeKey(form, KeyArrowRight, 0, KeyModifiers{})
		x := caret()
		if x < prev {
			t.Fatalf("Right %d moved left: %d -> %d", k, prev, x)
		}
		prev = x
	}
	w, _, _ := MeasureText(box.FontFamily(), box.FontSize(), box.Text())
	if abs(prev-box.textX()-w) > 1 {
		t.Fatalf("Right does not reach the right edge: %d", prev)
	}
	// Shift selects as it goes
	box.SetCursorPosition(TextPos{Col: 0})
	typeKey(form, KeyArrowRight, 0, KeyModifiers{Shift: true})
	if box.SelectedText() != "a" {
		t.Fatalf("Shift+Right selected %q", box.SelectedText())
	}
}

// A line placed to the right is drawn there, not cut by the width of the
// content
func TestTextBoxAlignedDrawn(t *testing.T) {
	haveScripts(t, "שלום")
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("שלום עולם")
	box.SetTextAlign(TextAlignStart)
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	form.paintScaled(img, 1)
	ox, oy := box.RectClientAreaOnWindow()
	w, _, _ := MeasureText(box.FontFamily(), box.FontSize(), box.Text())
	ink := 0
	bg := img.RGBAAt(ox+box.Width()/4, oy+box.Height()/2)
	for x := box.lineX(0); x < box.lineX(0)+w; x++ {
		for y := 4; y < box.Height()-4; y++ {
			if c := img.RGBAAt(ox+x, oy+y); abs(int(c.R)-int(bg.R)) > 80 {
				ink++
			}
		}
	}
	if ink < 30 {
		t.Fatalf("the right-aligned line is not drawn: %d pixels of ink", ink)
	}
}
