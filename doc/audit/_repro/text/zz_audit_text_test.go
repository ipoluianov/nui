package ui

// Reproducers of the text audit. Each test logs (t.Logf) what it shows;
// run: go test ./ui -run TestAudit -count=1 -v

import (
	"fmt"
	"image"
	"math"
	"runtime"
	"strings"
	"testing"
	"time"
)

func catch(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint("PANIC: ", r)
		}
	}()
	f()
	return "ok"
}

func paintForm(form *Form) { form.paintScaled(image.NewRGBA(image.Rect(0, 0, 500, 400)), 1) }

// F1: a bidi paragraph separator in shaped text
func TestAuditF1ParagraphSeparator(t *testing.T) {
	for _, s := range []string{"😀 done\r", "שלום\rעולם", "e\u0301\u2029x", "😀\u0085x"} {
		t.Logf("MeasureText(%q): %s", s, catch(func() { MeasureText(FontFamilySans, 12, s) }))
	}
	for _, s := range []string{"a\r", "a\x1cb", "a\u2029", "a\r\n"} {
		t.Logf("TextLayout(%q).SetWidth(1).Size(): %s", s, catch(func() {
			l := NewTextLayout(s, TextStyle{Size: 12})
			l.SetWidth(1)
			l.Size()
		}))
	}
	t.Logf("NewLabel(\"😀 done\\r\"): %s", catch(func() { NewLabel("😀 done\r") }))
	t.Logf("TextBox.SetText(\"e\\u0301\\u2029x\"): %s", catch(func() {
		box := NewTextBox()
		newTestForm(t, box)
		box.SetText("e\u0301\u2029x")
	}))
	t.Logf("typing e, U+0301, U+2029, x into a TextBox: %s", catch(func() {
		box := NewTextBox()
		form := newTestForm(t, box)
		box.Focus()
		for _, r := range "e\u0301\u2029x" {
			form.processChar(r)
		}
	}))
}

// F2: SetMaxLines on a text of several paragraphs
func TestAuditF2MaxLines(t *testing.T) {
	t.Logf("NewTextLayout(\"a b\\nc\").SetMaxLines(1).Size(): %s", catch(func() {
		l := NewTextLayout("a b\nc", TextStyle{Size: 12})
		l.SetMaxLines(1)
		l.Size()
	}))
}

// F3: the cache of shaped lines has no budget of memory
func TestAuditF3ShapedCacheMemory(t *testing.T) {
	clearShapedLines()
	runtime.GC()
	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("😀" + strings.Repeat("x", 20000))
	box.Focus()
	box.MoveCursorToEnd()
	t0 := time.Now()
	for i := 0; i < 200; i++ {
		form.processChar('y')
	}
	runtime.GC()
	runtime.ReadMemStats(&m1)
	t.Logf("200 chars typed at the end of a 20001-rune line with one emoji: %v, live heap +%d MB, shapedLines entries %d (limit 2048)",
		time.Since(t0), (int64(m1.HeapAlloc)-int64(m0.HeapAlloc))>>20, len(shapedLines))
}

// F4: a password box copies its text
func TestAuditF4PasswordCopy(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetIsPassword(true)
	box.SetText("s3cret")
	box.Focus()
	box.SelectAllText()
	typeKey(form, KeyC, 0, KeyModifiers{Ctrl: true})
	got, _ := ClipboardGetText()
	t.Logf("clipboard after Ctrl+A, Ctrl+C in a password box: %q", got)
}

// F5: DrawText at a large font size
func TestAuditF5BigFontSize(t *testing.T) {
	for _, size := range []float64{4000, 16000} {
		runtime.GC()
		var m0, m1 runtime.MemStats
		runtime.ReadMemStats(&m0)
		t0 := time.Now()
		cnv := NewCanvasScaled(image.NewRGBA(image.Rect(0, 0, 100, 50)), 1)
		cnv.SetFontSize(size)
		cnv.DrawText(0, 0, 100, 50, "ab")
		runtime.ReadMemStats(&m1)
		t.Logf("DrawText(\"ab\") size %v on a 100x50 canvas: %v, allocated %d MB", size, time.Since(t0), (m1.TotalAlloc-m0.TotalAlloc)>>20)
	}
	t.Logf("size 1e9: %s", catch(func() {
		cnv := NewCanvasScaled(image.NewRGBA(image.Rect(0, 0, 100, 50)), 1)
		cnv.SetFontSize(1e9)
		cnv.DrawText(0, 0, 100, 50, "ab")
	}))
}

// F6: NumBox stepped to +Inf
func TestAuditF6NumBoxInf(t *testing.T) {
	n := NewNumBox()
	form := newTestForm(t, n)
	n.SetDecimals(2)
	n.SetValue(1e307)
	n.Focus()
	typeKey(form, KeyArrowUp, 0, KeyModifiers{})
	t.Logf("decimals 2, value 1e307, Up: value=%v text=%q", n.Value(), n.Text())
	n.SetValue(math.Inf(1))
	t.Logf("SetValue(+Inf): value=%v text=%q", n.Value(), n.Text())
}

// F7: a click in a gap of moveCursorNearPoint is ignored
func TestAuditF7ClickGap(t *testing.T) {
	box := NewTextBox()
	newTestForm(t, box)
	box.SetText("line 52")
	box.Focus()
	cp, _ := GetCharPositions(box.FontFamily(), box.FontSize(), box.Line(0))
	lx := box.lineX(0)
	box.SetCursorPosition(TextPos{Col: 7})
	x := lx + 12 // between "i" (caret 4..8) and "n" (8..17): n is 9px wide
	box.MouseDown(MouseButtonLeft, x, 5, KeyModifiers{})
	box.MouseUp(MouseButtonLeft, x, 5, KeyModifiers{})
	t.Logf("caret xs %v (+lineX %d); click at x=%d: cursor %+v, nearest caret %d",
		cp, lx, x, box.CursorPosition(), TextIndexAt(box.FontFamily(), box.FontSize(), box.Line(0), x-lx))
}

// F8: a single-line box keeps the lines of SetText
func TestAuditF8SingleLineNewline(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("visible\nhidden")
	box.Focus()
	typeKey(form, KeyArrowDown, 0, KeyModifiers{})
	typeText(form, "X")
	t.Logf("single-line box: lines=%d, cursor %+v, text %q", box.LineCount(), box.CursorPosition(), box.Text())
}

// F9: an edit that changes nothing clears the redo history
func TestAuditF9NoopUndo(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("abc")
	box.Focus()
	box.MoveCursorToEnd()
	typeText(form, "d")
	typeKey(form, KeyZ, 0, KeyModifiers{Ctrl: true})
	t.Logf("after typing d and Ctrl+Z: %q CanRedo=%v", box.Text(), box.CanRedo())
	typeKey(form, KeyHome, 0, KeyModifiers{})
	typeKey(form, KeyBackspace, 0, KeyModifiers{})
	t.Logf("after Backspace at the start (nothing deleted): %q CanRedo=%v undo steps=%d", box.Text(), box.CanRedo(), len(box.undoStack))
	box.InsertText("")
	box.InsertText("\x01")
	t.Logf("after InsertText(\"\") and InsertText(\"\\x01\"): undo steps=%d", len(box.undoStack))
}

// F10: an edit rewrites the invalid bytes of the line
func TestAuditF10InvalidUTF8(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("a\xffb")
	box.Focus()
	box.MoveCursorToEnd()
	typeText(form, "Z")
	t.Logf("SetText(%q), typed Z at the end: Text()=%q", "a\xffb", box.Text())
}

// F11: TextSelectionSpans with from past the text
func TestAuditF11SelectionSpans(t *testing.T) {
	t.Logf("TextSelectionSpans(\"ab\", 5, 9): %s", catch(func() { TextSelectionSpans(FontFamilySans, 12, "ab", 5, 9) }))
}

// F12: IndexAt is quadratic
func TestAuditF12IndexAtQuadratic(t *testing.T) {
	for _, n := range []int{4000, 8000, 16000} {
		line := "😀" + strings.Repeat("a", n)
		TextCaretX(FontFamilySans, 12, line, 0)
		t0 := time.Now()
		TextIndexAt(FontFamilySans, 12, line, 100)
		t.Logf("TextIndexAt, %d-rune line with an emoji (laid out already): %v", n+1, time.Since(t0))
	}
}

// F13: MakeLinesFromStringWithWordWrapping is quadratic
func TestAuditF13WordWrapQuadratic(t *testing.T) {
	for _, n := range []int{25000, 50000, 100000} {
		t0 := time.Now()
		MakeLinesFromStringWithWordWrapping(strings.Repeat("x", n), 80)
		t.Logf("one %d-char word: %v", n, time.Since(t0))
	}
}

// F14: the highlight of the match in EditableComboBox's list
func TestAuditF14ComboHighlight(t *testing.T) {
	text, needle := "Ⱥẞabc", "ßab"
	start := strings.Index(strings.ToLower(text), needle)
	t.Logf("len(text)=%d len(lower)=%d: the part drawn as the match of %q is %q", len(text), len(strings.ToLower(text)), needle, text[start:start+len(needle)])
}
