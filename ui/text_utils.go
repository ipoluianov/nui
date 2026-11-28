package ui

import (
	"strings"
	"unicode/utf8"
)

// truncateTextToWidth shortens text with a trailing "..." so its rendered
// width fits within maxWidth pixels for the given font. Returns text
// unchanged if it already fits, or if maxWidth/measurement isn't usable.
// It is cut between graphemes - a letter keeps its marks, a conjunct of an
// Indic script is not split - and in text written right to left the
// ellipsis is at the end of its reading, on the left.
func truncateTextToWidth(fontFamily string, fontSize float64, text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	width, _, err := MeasureText(fontFamily, fontSize, text)
	if err != nil || width <= maxWidth {
		return text
	}

	const ellipsis = "..."
	ellipsisWidth, _, err := MeasureText(fontFamily, fontSize, ellipsis)
	if err != nil {
		return text
	}
	if ellipsisWidth > maxWidth {
		return ellipsis
	}

	// The longest start that fits with the ellipsis: the width grows with
	// the start, so it is searched by halves
	runes := []rune(text)
	bounds := graphemeBounds(text)
	fits := func(k int) bool {
		w, _, err := MeasureText(fontFamily, fontSize, string(runes[:bounds[k]])+ellipsis)
		return err == nil && w <= maxWidth
	}
	lo, hi := 0, len(bounds)-2 // bounds[lo] fits (the empty start), bounds[hi+1] is all of it
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if fits(mid) {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return string(runes[:bounds[lo]]) + ellipsis
}

func MakeLinesFromStringWithWordWrapping(text string, maxChars int) []string {
	// Strings are split and sliced, not built rune by rune: that was
	// quadratic in the length of a word
	isSpace := func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f'
	}

	// If maxChars <= 0, do not wrap — just preserve existing newlines
	if maxChars <= 0 {
		return strings.Split(text, "\n")
	}

	result := make([]string, 0)

	// Split on mandatory newlines first
	for _, p := range strings.Split(text, "\n") {
		words := strings.FieldsFunc(p, isSpace)
		// An empty paragraph corresponds to an empty line
		if len(words) == 0 {
			result = append(result, "")
			continue
		}

		var current strings.Builder
		currentLen := 0 // in runes
		for _, w := range words {
			wLen := utf8.RuneCountInString(w)

			if currentLen > 0 && currentLen+1+wLen > maxChars {
				// push current and start new line with word
				result = append(result, current.String())
				current.Reset()
				currentLen = 0
			}
			if currentLen > 0 {
				current.WriteByte(' ')
				currentLen++
			}
			// A word longer than maxChars has a line of its own
			current.WriteString(w)
			currentLen += wLen
		}
		result = append(result, current.String())
	}

	return result
}
