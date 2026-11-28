package ui

import (
	"testing"
)

// A click between two characters whose left neighbor is an odd number of
// pixels wide falls in a 1px gap of moveCursorNearPoint and is ignored
func TestAuditClickGap(t *testing.T) {
	box := NewTextBox()
	newTestForm(t, box)
	box.SetText("line 52")
	box.Focus()
	cp, _ := GetCharPositions(box.FontFamily(), box.FontSize(), box.Line(0))
	lx := box.lineX(0)
	misses := 0
	for x := lx; x < lx+cp[len(cp)-1]; x++ {
		box.SetCursorPosition(TextPos{Col: 7})
		box.MouseDown(MouseButtonLeft, x, 5, KeyModifiers{})
		box.MouseUp(MouseButtonLeft, x, 5, KeyModifiers{})
		want := TextIndexAt(box.FontFamily(), box.FontSize(), box.Line(0), x-lx)
		if got := box.CursorPosition().Col; got != want {
			misses++
			if misses <= 5 {
				t.Logf("click at x=%d (positions %v + lineX %d): cursor stays at %d, nearest caret is %d", x, cp, lx, got, want)
			}
		}
	}
	t.Logf("TextBox: %d of %d pixels ignored", misses, cp[len(cp)-1])

	n := NewNumBox()
	newTestForm(t, n)
	n.SetText("1234.5678")
	cpn, _ := GetCharPositions(n.FontFamily(), n.FontSize(), n.Text())
	bad := 0
	for px := 0; px < cpn[len(cpn)-1]; px++ {
		got := n.cursorPosFromPixel(px)
		want := TextIndexAt(n.FontFamily(), n.FontSize(), n.Text(), px)
		if got != want {
			bad++
			if bad <= 3 {
				t.Logf("NumBox click at px=%d (positions %v): cursor %d, nearest caret %d", px, cpn, got, want)
			}
		}
	}
	t.Logf("NumBox: %d of %d pixels give a wrong position", bad, cpn[len(cpn)-1])
}
