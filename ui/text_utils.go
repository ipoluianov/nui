package ui

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
	// Local helpers as closures to keep a single top-level function
	splitOnNewlines := func(s string) []string {
		res := make([]string, 0)
		cur := ""
		for _, r := range s {
			if r == '\n' {
				res = append(res, cur)
				cur = ""
			} else {
				cur += string(r)
			}
		}
		res = append(res, cur)
		return res
	}

	fields := func(s string) []string {
		res := make([]string, 0)
		cur := ""
		inField := false
		for _, r := range s {
			if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' {
				if inField {
					res = append(res, cur)
					cur = ""
					inField = false
				}
			} else {
				cur += string(r)
				inField = true
			}
		}
		if inField {
			res = append(res, cur)
		}
		return res
	}

	splitPreserveNewlines := func(s string) []string { return splitOnNewlines(s) }

	// If maxChars <= 0, do not wrap — just preserve existing newlines
	if maxChars <= 0 {
		return splitPreserveNewlines(text)
	}

	result := make([]string, 0)

	// Split on mandatory newlines first
	paragraphs := splitOnNewlines(text)
	for _, p := range paragraphs {
		// An empty paragraph corresponds to an empty line
		if p == "" {
			result = append(result, "")
			continue
		}

		words := fields(p)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}

		current := ""
		for _, w := range words {
			wLen := len([]rune(w))

			if current == "" {
				// start new line with the word (even if it exceeds maxChars)
				current = w
				continue
			}

			// length of current + 1 space + word
			if len([]rune(current))+1+wLen <= maxChars {
				current = current + " " + w
			} else {
				// push current and start new line with word
				result = append(result, current)
				current = w
			}
		}

		if current != "" {
			result = append(result, current)
		}
	}

	return result
}
