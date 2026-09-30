package ui

import "testing"

func TestExpander(t *testing.T) {
	e := NewExpander("Advanced")
	lbl := e.Content().AddLabel(0, 0, "Timeout")
	form := newTestForm(t, e)
	changes := 0
	e.SetOnExpandedChanged(func() { changes++ })

	header := e.headerHeight()
	if e.Expanded() || e.Height() != header {
		t.Fatalf("collapsed: expanded %v, height %d, want %d", e.Expanded(), e.Height(), header)
	}
	if lbl.Height() != 0 {
		t.Errorf("collapsed content is laid out: label height %d", lbl.Height())
	}

	// A click on the header expands it
	form.processMouseDown(MouseButtonLeft, e.X()+10, e.Y()+header/2)
	form.processMouseUp(MouseButtonLeft, e.X()+10, e.Y()+header/2)
	if !e.Expanded() || changes != 1 {
		t.Fatalf("click: expanded %v, changes %d", e.Expanded(), changes)
	}
	if e.Height() <= header || lbl.Height() == 0 || lbl.Y()+e.Content().Y() < header {
		t.Errorf("expanded layout: height %d, label y %d h %d", e.Height(), lbl.Y(), lbl.Height())
	}
	// Not stretched by the spacer below: as tall as its content
	if e.Height() != e.MinHeight() {
		t.Errorf("expanded height %d, min %d", e.Height(), e.MinHeight())
	}

	// Keys: Left collapses, Right expands; SetExpanded calls no callback
	e.ProcessKeyDown(KeyArrowLeft, KeyModifiers{})
	if e.Expanded() || changes != 2 {
		t.Errorf("Left: expanded %v, changes %d", e.Expanded(), changes)
	}
	e.SetExpanded(true)
	if !e.Expanded() || changes != 2 {
		t.Errorf("SetExpanded: expanded %v, changes %d", e.Expanded(), changes)
	}
}

func TestAccordion(t *testing.T) {
	acc := NewAccordion()
	a := acc.AddSection("A")
	b := acc.AddSection("B")
	c := acc.AddSection("C")
	for _, s := range acc.Sections() {
		s.Content().AddLabel(0, 0, "content")
	}
	form := newTestForm(t, acc)
	var changed []int
	acc.SetOnSectionChanged(func(i int) { changed = append(changed, i) })

	if acc.ExpandedIndex() != 0 {
		t.Fatalf("first section not expanded: %d", acc.ExpandedIndex())
	}
	// Expanding B by the user collapses A
	b.toggleByUser(true)
	if a.Expanded() || !b.Expanded() || c.Expanded() || len(changed) != 1 || changed[0] != 1 {
		t.Fatalf("exclusive: a %v b %v c %v, changed %v", a.Expanded(), b.Expanded(), c.Expanded(), changed)
	}
	form.UpdateLayout()
	if c.Y() <= b.Y()+b.headerHeight() {
		t.Errorf("C isn't below B's content: b.y %d, c.y %d", b.Y(), c.Y())
	}

	acc.SetExclusive(false)
	a.toggleByUser(true)
	if !a.Expanded() || !b.Expanded() {
		t.Errorf("non-exclusive: a %v b %v", a.Expanded(), b.Expanded())
	}
	acc.Expand(-1)
	if acc.ExpandedIndex() != -1 {
		t.Errorf("Expand(-1): %d", acc.ExpandedIndex())
	}
}
