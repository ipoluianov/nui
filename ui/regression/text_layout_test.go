package regression

import (
	"fmt"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// noPanic runs f and fails the test, naming the case, if it panics
func noPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s: panic: %v", name, r)
		}
	}()
	f()
}

// The separators of paragraphs of the bidi algorithm other than "\n" -
// "\r", U+001C..U+001E, U+0085, U+2029 - in a text that is shaped (an
// emoji, Hebrew, Arabic, a combining mark) or wrapped do not crash the
// text layout. In a line of DrawText they are drawn as nothing; a
// TextLayout breaks its lines at them, "\r\n" being one break.
func TestTextParagraphSeparators(t *testing.T) {
	shaped := []string{"😀 done\r", "שלום\rעולם", "e\u0301\u2029x", "😀\u0085x",
		"א\r0", "aא\rb", "א\u2029b", "א\u0085b", "0־\x1e0", "0ا\x1e0", "a\x1cb😀", "😀\x1d"}
	for _, s := range shaped {
		noPanic(t, fmt.Sprintf("MeasureText(%q)", s), func() { ui.MeasureText(ui.FontFamilySans, 12, s) })
		noPanic(t, fmt.Sprintf("NewLabel(%q)", s), func() {
			form := newForm(300, 100)
			form.Panel().AddWidget(0, 0, ui.NewLabel(s))
			uitest.Show(t, form).Screenshot()
			form.Close()
		})
		noPanic(t, fmt.Sprintf("TextBox.SetText(%q)", s), func() {
			w, tb := textBoxForm(t, false)
			tb.SetText(s)
			w.Screenshot()
		})
	}
	for _, s := range append(shaped, "a\r", "a\x1cb", "a\u2029", "a\r\n", "a\rb", "a\u0085b") {
		for _, width := range []float64{0, 1, 100} {
			noPanic(t, fmt.Sprintf("TextLayout(%q) at width %v", s, width), func() {
				l := ui.NewTextLayout(s, ui.TextStyle{Size: 12})
				l.SetWidth(width)
				l.Size()
			})
		}
	}
	// The user types a combining accent and U+2029
	noPanic(t, "typing", func() {
		w, _ := textBoxForm(t, false)
		w.Type("e\u0301\u2029x")
		w.Screenshot()
	})

	// In a line of DrawText a lone "\r" is drawn as nothing
	w1, _, _ := ui.MeasureText(ui.FontFamilySans, 12, "😀 done\r")
	w2, _, _ := ui.MeasureText(ui.FontFamilySans, 12, "😀 done")
	if w1 != w2 {
		t.Errorf(`"😀 done\r" is %d wide, "😀 done" %d`, w1, w2)
	}
	// A TextLayout breaks its lines at the separators
	for s, want := range map[string]int{"a\rb": 2, "a\r\nb": 2, "a\u2029b\u0085c": 3, "שלום\rעולם": 2, "a\r": 2} {
		var lines []ui.TextLine
		noPanic(t, fmt.Sprintf("Lines(%q)", s), func() { lines = ui.NewTextLayout(s, ui.TextStyle{Size: 12}).Lines() })
		if len(lines) != want {
			t.Errorf("%q: %d lines, want %d", s, len(lines), want)
		}
	}
}

// TextLayout.SetMaxLines on a text of several paragraphs keeps the first
// lines, the last ending with an ellipsis
func TestTextLayoutMaxLinesSeveralParagraphs(t *testing.T) {
	for _, s := range []string{"a b\nc", "a b\nc\nd", "שלום\nעולם"} {
		var lines []ui.TextLine
		noPanic(t, fmt.Sprintf("%q", s), func() {
			l := ui.NewTextLayout(s, ui.TextStyle{Size: 12})
			l.SetMaxLines(1)
			lines = l.Lines()
		})
		if len(lines) != 1 {
			t.Errorf("%q: %d lines, want 1", s, len(lines))
			continue
		}
		first, _ := ui.NewTextLayout(s[:lines[0].End], ui.TextStyle{Size: 12}).Size()
		if lines[0].Width <= first {
			t.Errorf("%q: the line is %v wide, its text alone %v: no ellipsis", s, lines[0].Width, first)
		}
	}
}
