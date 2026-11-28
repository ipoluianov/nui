package ui

import (
	"runtime"
	"slices"
	"strings"
	"testing"
)

// TXT-9: an edit that changes nothing records no undo step and keeps Redo
func TestTextBoxNoOpEditKeepsHistory(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetText("abc")
	box.Focus()
	box.MoveCursorToEnd()
	typeText(form, "d")
	typeKey(form, KeyZ, 0, KeyModifiers{Ctrl: true})
	if box.Text() != "abc" || !box.CanRedo() {
		t.Fatalf("after typing d and Ctrl+Z: %q CanRedo=%v", box.Text(), box.CanRedo())
	}
	undo := len(box.undoStack)

	typeKey(form, KeyHome, 0, KeyModifiers{})
	typeKey(form, KeyBackspace, 0, KeyModifiers{}) // nothing before the cursor
	box.InsertText("")
	box.InsertText("\x01") // only a control character: nothing inserted
	typeKey(form, KeyEnd, 0, KeyModifiers{})
	typeKey(form, KeyDelete, 0, KeyModifiers{}) // nothing after the cursor

	if box.Text() != "abc" {
		t.Fatalf("text = %q", box.Text())
	}
	if !box.CanRedo() {
		t.Error("a no-op edit cleared the redo history")
	}
	if len(box.undoStack) != undo {
		t.Errorf("undo steps = %d, want %d", len(box.undoStack), undo)
	}
	typeKey(form, KeyY, 0, KeyModifiers{Ctrl: true})
	if box.Text() != "abcd" {
		t.Errorf("after Redo: %q, want \"abcd\"", box.Text())
	}
}

// TXT-9: a no-op Backspace does not break the grouping of the typing
// around it into one undo step
func TestTextBoxNoOpEditKeepsGrouping(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.Focus()
	typeKey(form, KeyBackspace, 0, KeyModifiers{})
	if box.CanUndo() {
		t.Fatal("Backspace in an empty box made an undo step")
	}
	typeText(form, "ab")
	typeKey(form, KeyZ, 0, KeyModifiers{Ctrl: true})
	if box.Text() != "" || box.CanUndo() {
		t.Errorf("after Ctrl+Z: %q CanUndo=%v, want \"\" and false", box.Text(), box.CanUndo())
	}
}

// TXT-10: an edit keeps the bytes of the text it does not touch; invalid
// UTF-8 is replaced once, by SetText
func TestTextBoxInvalidUTF8EditConsistent(t *testing.T) {
	box := NewTextBox()
	form := newTestForm(t, box)
	box.SetMultiline(true)
	box.SetText("a\xffb\nc\xfed")
	before := box.Text()
	if before != "a�b\nc�d" {
		t.Fatalf("SetText kept invalid UTF-8: %q", before)
	}
	box.Focus()
	box.MoveCursorToEnd() // the end of the first line
	typeText(form, "Z")
	if got, want := box.Text(), "a�b\nc�dZ"; got != want && got != "a�bZ\nc�d" {
		t.Errorf("after typing Z: %q", got)
	}
	// The edited line and the untouched one are in the same form
	for _, l := range box.lines {
		if strings.ContainsRune(l, 0xFF) || !strings.Contains(l, "�") {
			t.Errorf("line %q", l)
		}
	}
}

// TXT-11: a range past the end of the text selects nothing
func TestTextSelectionSpansPastEnd(t *testing.T) {
	for _, r := range [][2]int{{5, 9}, {2, 9}, {3, 3}} {
		var spans [][2]int
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("TextSelectionSpans(%d, %d) panicked: %v", r[0], r[1], p)
				}
			}()
			spans = TextSelectionSpans(FontFamilySans, 12, "ab", r[0], r[1])
		}()
		if spans != nil {
			t.Errorf("TextSelectionSpans(%d, %d) = %v, want nil", r[0], r[1], spans)
		}
	}
	// A range that overlaps the end is cut at the end
	whole := TextSelectionSpans(FontFamilySans, 12, "ab", 0, 2)
	over := TextSelectionSpans(FontFamilySans, 12, "ab", 0, 9)
	if len(whole) != 1 || !slices.Equal(whole, over) {
		t.Errorf("[0, 9) = %v, [0, 2) = %v", over, whole)
	}
}

// TXT-13: word wrapping is linear: a long word does not allocate its length
// squared
func TestWordWrappingLinear(t *testing.T) {
	word := strings.Repeat("x", 100000)
	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	lines := MakeLinesFromStringWithWordWrapping(word+" y", 80)
	runtime.ReadMemStats(&m1)
	if len(lines) != 2 || lines[0] != word || lines[1] != "y" {
		t.Fatalf("lines: %d", len(lines))
	}
	if alloc := m1.TotalAlloc - m0.TotalAlloc; alloc > 10<<20 {
		t.Errorf("wrapping a 100k word allocated %d bytes", alloc)
	}
}

// TXT-13: the rewrite wraps as before
func TestWordWrappingResult(t *testing.T) {
	cases := []struct {
		text string
		max  int
		want []string
	}{
		{"", 10, []string{""}},
		{"a\n\nb", 10, []string{"a", "", "b"}},
		{"one two three four", 9, []string{"one two", "three", "four"}},
		{"  one\ttwo  ", 7, []string{"one two"}},
		{"a b c", 3, []string{"a b", "c"}}, // NBSP is not a space here
		{"long-word x", 3, []string{"long-word", "x"}},
		{"   \nx", 5, []string{"", "x"}},
		{"a b\nc", 0, []string{"a b", "c"}},
		{"ää ää", 5, []string{"ää ää"}},
	}
	for _, c := range cases {
		if got := MakeLinesFromStringWithWordWrapping(c.text, c.max); !slices.Equal(got, c.want) {
			t.Errorf("wrap(%q, %d) = %q, want %q", c.text, c.max, got, c.want)
		}
	}
}

// TXT-14: the highlighted match is the original text of the match, even
// when lower-casing changes the byte lengths of letters before it
func TestComboBoxLowerMatch(t *testing.T) {
	cases := []struct {
		text, needle, want string
	}{
		{"Berlin", "erl", "erl"},
		{"Ⱥẞabc", "ßab", "ẞab"}, // Ⱥ grows (2 → 3 bytes), ẞ shrinks (3 → 2)
		{"ȺȺxY", "xy", "xY"},
		{"İstanbul", "stan", "stan"},
		{"abc", "zz", ""},
	}
	for _, c := range cases {
		start, end := lowerMatch(c.text, c.needle)
		got := ""
		if start >= 0 {
			got = c.text[start:end]
		}
		if got != c.want {
			t.Errorf("lowerMatch(%q, %q) = %q, want %q", c.text, c.needle, got, c.want)
		}
	}
}
