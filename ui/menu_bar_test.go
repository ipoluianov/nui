package ui

import "testing"

// The menu bar takes the top of the form, the panel the rest; mouse events
// go to the one under the mouse, in its own coordinates
func TestMenuBarLayout(t *testing.T) {
	form := NewForm()
	form.processResize(400, 300)

	bar := NewMenuBar()
	bar.AddMenu("File")
	bar.AddMenu("Edit")
	form.SetMenuBar(bar)

	barHeight := bar.barHeight()
	if bar.Width() != 400 || bar.Height() != barHeight {
		t.Errorf("bar size = %dx%d, want 400x%d", bar.Width(), bar.Height(), barHeight)
	}
	panel := form.Panel()
	if panel.Y() != barHeight || panel.Height() != 300-barHeight || panel.Width() != 400 {
		t.Errorf("panel at y=%d size %dx%d, want y=%d size 400x%d", panel.Y(), panel.Width(), panel.Height(), barHeight, 300-barHeight)
	}

	items := bar.Items()
	if items[0].X() != 0 || items[1].X() != items[0].Width() || items[0].Width() <= 0 {
		t.Errorf("titles at x=%d (w=%d) and x=%d", items[0].X(), items[0].Width(), items[1].X())
	}

	if target, _, _ := form.mouseTarget(nil, 5, 2); target != Widgeter(bar) {
		t.Errorf("mouse over the bar goes to %s", target.TypeName())
	}
	target, x, y := form.mouseTarget(nil, 5, barHeight+3)
	if target != Widgeter(panel) || x != 5 || y != 3 {
		t.Errorf("mouse below the bar goes to %s at (%d, %d), want the panel at (5, 3)", target.TypeName(), x, y)
	}

	// Hidden titles take no place
	items[0].SetVisible(false)
	if items[1].X() != 0 {
		t.Errorf("title after a hidden one at x=%d, want 0", items[1].X())
	}

	form.processResize(500, 200)
	if bar.Width() != 500 || panel.Height() != 200-barHeight {
		t.Errorf("after resize: bar width %d, panel height %d", bar.Width(), panel.Height())
	}

	form.SetMenuBar(nil)
	if panel.Y() != 0 || panel.Height() != 200 {
		t.Errorf("without the bar: panel at y=%d height %d", panel.Y(), panel.Height())
	}
	if target, _, _ := form.mouseTarget(nil, 5, 2); target != Widgeter(panel) {
		t.Errorf("without the bar the mouse goes to %s", target.TypeName())
	}
}
