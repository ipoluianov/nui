package ui

import (
	"image"
	"strings"
	"runtime/debug"
	"testing"
)

func TestAuditLayoutFuzz(t *testing.T) {
	pieces := []string{"a", "b ", " ", "\n", "\t", "\v", "\f", "שלום", "مرحبا", "123", "😀", "👨‍👩‍👧", "🇩🇪", "é", "́", "‍", "‮", "⁧", "क्षि", "\xff", "\xc3", "\x00", "…", "ﷺ", "\ufeff", "x̀́̂̃̄"}
	seed := uint32(1)
	rnd := func(n int) int { seed = seed*1664525 + 1013904223; return int(seed>>8) % n }
	for iter := 0; iter < 20000; iter++ {
		var b strings.Builder
		n := rnd(12)
		for i := 0; i < n; i++ {
			b.WriteString(pieces[rnd(len(pieces))])
		}
		text := b.String()
		if strings.ContainsAny(text, "\r\u0085\u2028\u2029") {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic for %q (iter %d): %v\n%s", text, iter, r, debug.Stack())
				}
			}()
			l := NewTextLayout(text, TextStyle{Size: float64(1 + rnd(30))})
			if rnd(2) == 0 {
				l.Append(pieces[rnd(len(pieces))], TextStyle{Size: 12, Bold: true, LetterSpacing: float64(rnd(5))})
			}
			if rnd(2) == 0 {
				l.SetWidth(float64(rnd(60)))
			}
			if rnd(2) == 0 {
				if !strings.Contains(l.Text(), "\n") { l.SetMaxLines(rnd(3)) }
			}
			if rnd(3) == 0 {
				l.SetRightToLeft(rnd(2) == 0)
			}
			l.SetAlign(HAlign(rnd(3)))
			l.Size()
			l.Lines()
			tl := len(l.Text())
			for i := -2; i <= tl+2; i++ {
				l.CaretRect(i)
				l.SelectionRects(i, tl-i)
			}
			l.IndexAt(float64(rnd(200)-20), float64(rnd(100)-10))
			cnv := NewCanvasScaled(image.NewRGBA(image.Rect(0, 0, 200, 100)), 1)
			cnv.DrawTextLayout(l, 3, 3)
			fs := float64(1 + rnd(30))
			TextCaretX(FontFamilySans, fs, text, rnd(20)-3)
			TextIndexAt(FontFamilySans, fs, text, rnd(200)-20)
			if nr := len([]rune(text)); nr > 0 { TextSelectionSpans(FontFamilySans, fs, text, rnd(nr+1)-1, rnd(nr+1)) }
			gb := graphemeBounds(text)
			if gb[len(gb)-1] != len([]rune(text)) {
				t.Errorf("graphemeBounds end %v for %q", gb, text)
			}
			GetCharPositions(FontFamilySans, fs, text)
			MeasureText(FontFamilySans, fs, text)
			truncateTextToWidth(FontFamilySans, fs, text, rnd(80))
		}()
	}
}
