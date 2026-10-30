package ui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// truncateOld is the truncation before graphemes: a rune at a time from the
// end
func truncateOld(fontFamily string, fontSize float64, text string, maxWidth int) string {
	width, _, _ := MeasureText(fontFamily, fontSize, text)
	if maxWidth <= 0 || width <= maxWidth {
		return text
	}
	const ellipsis = "..."
	if w, _, _ := MeasureText(fontFamily, fontSize, ellipsis); w > maxWidth {
		return ellipsis
	}
	runes := []rune(text)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if w, _, _ := MeasureText(fontFamily, fontSize, string(runes)+ellipsis); w <= maxWidth {
			return string(runes) + ellipsis
		}
	}
	return ellipsis
}

func TestTruncateAsBefore(t *testing.T) {
	for _, text := range []string{"The quick brown fox jumps over the lazy dog", "Съешь же ещё этих мягких булок", "WWWW iiii", "a", "中文字符的测试"} {
		w, _, _ := MeasureText(FontFamilySans, 13, text)
		for max := 1; max <= w+5; max++ {
			if got, want := truncateTextToWidth(FontFamilySans, 13, text, max), truncateOld(FontFamilySans, 13, text, max); got != want {
				t.Fatalf("%q in %d: %q, before %q", text, max, got, want)
			}
		}
	}
}

func TestTruncateGraphemes(t *testing.T) {
	haveScripts(t, "नमस्ते")
	for _, text := range []string{"नमस्ते दुनिया नमस्ते", "café crème brûlée"} {
		text := strings.ReplaceAll(text, "é", "é")
		bounds := graphemeBounds(text)
		w, _, _ := MeasureText(FontFamilySans, 14, text)
		for max := 10; max < w; max += 3 {
			got := truncateTextToWidth(FontFamilySans, 14, text, max)
			cut := utf8.RuneCountInString(strings.TrimSuffix(got, "..."))
			if !containsInt(bounds, cut) {
				t.Fatalf("%q in %d: cut at %d, inside a grapheme (%v)", text, max, cut, bounds)
			}
			if gw, _, _ := MeasureText(FontFamilySans, 14, got); gw > max && got != "..." {
				t.Fatalf("%q in %d: %q is %d wide", text, max, got, gw)
			}
		}
	}
}

func TestTruncateRightToLeft(t *testing.T) {
	haveScripts(t, "שלום")
	text := "שלום עולם ומלואו"
	w, _, _ := MeasureText(FontFamilySans, 14, text)
	got := truncateTextToWidth(FontFamilySans, 14, text, w*2/3)
	if !strings.HasPrefix(got, "שלום") || !strings.HasSuffix(got, "...") {
		t.Fatalf("%q", got)
	}
	// The ellipsis is at the left: the caret after it is at the left edge
	n := utf8.RuneCountInString(got)
	if x := TextCaretX(FontFamilySans, 14, got, n); x > 2 {
		t.Fatalf("the end of %q at %d, not at the left", got, x)
	}
	dots := TextSelectionSpans(FontFamilySans, 14, got, n-3, n)
	start := TextCaretX(FontFamilySans, 14, got, 0)
	if len(dots) != 1 || dots[0][1] > start/2 {
		t.Fatalf("the ellipsis at %v, the start of the text at %d", dots, start)
	}
}
