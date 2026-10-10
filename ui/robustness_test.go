package ui

import (
	"image"
	"math"
	"testing"
	"time"
)

// Tab with nothing focused (a fresh form, or after a click on a label)
// focuses the first focusable widget instead of panicking
func TestTabWithoutFocus(t *testing.T) {
	tb := NewTextBox()
	form := newTestForm(t, tb)
	form.Panel().AddWidget(2, 0, NewLabel("label"))
	if form.FocusedWidget() != nil {
		t.Fatal("a fresh form has a focused widget")
	}
	form.processKeyDown(KeyTab, KeyModifiers{})
	if form.FocusedWidget() != Widgeter(tb) {
		t.Errorf("focused after Tab: %v", form.FocusedWidget())
	}
}

// A removed widget loses the focus: keys don't go to it any more, and the
// next click elsewhere doesn't touch it
func TestRemoveFocusedWidget(t *testing.T) {
	label := NewLabel("label")
	form := newTestForm(t, label)
	box := NewPanel()
	tb := NewTextBox()
	box.AddWidget(0, 0, tb)
	form.Panel().AddWidget(2, 0, box)
	form.UpdateLayout()

	focusLost := 0
	tb.SetOnFocusLost(func() { focusLost++ })
	tb.Focus()
	form.hoverWidget = tb
	form.mouseLeftButtonPressedWidget = tb

	// The text box goes with its parent
	form.Panel().RemoveWidget(box)
	if form.FocusedWidget() != nil || form.hoverWidget != nil || form.mouseLeftButtonPressedWidget != nil {
		t.Fatalf("the form still refers to the removed widget: focused %t, hover %t, pressed %t",
			form.FocusedWidget() != nil, form.hoverWidget != nil, form.mouseLeftButtonPressedWidget != nil)
	}
	form.processChar('x')
	if tb.Text() != "" {
		t.Errorf("a key went to the removed text box: %q", tb.Text())
	}
	form.processMouseDown(MouseButtonLeft, label.X()+2, label.Y()+2)
	form.processMouseUp(MouseButtonLeft, label.X()+2, label.Y()+2)
	if focusLost != 0 {
		t.Errorf("onFocusLost of the removed widget called %d times", focusLost)
	}
	if tb.IsFocused() || tb.IsHovered() {
		t.Error("the removed widget reports focus or hover")
	}
	tb.ClearFocus() // detached: does nothing
}

// Esc in the table's cell editor keeps the cell's text, Enter applies it:
// removing the focused editor must not run its focus-lost commit
func TestTableEditorEscAndEnter(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	table.SetColumnCount(2)
	table.SetRowCount(3)
	table.SetCellText2(1, 0, "old")
	table.SetCurrentCell2(1, 0)

	table.EditCurrentCell("new")
	form.processKeyDown(KeyEsc, KeyModifiers{})
	if got := table.GetCellText2(1, 0); got != "old" {
		t.Errorf("after Esc: %q, want old", got)
	}
	if form.FocusedWidget() != Widgeter(table) {
		t.Errorf("after Esc the focus is on %v, want the table", form.FocusedWidget())
	}

	table.EditCurrentCell("new")
	form.processKeyDown(KeyEnter, KeyModifiers{})
	if got := table.GetCellText2(1, 0); got != "new" {
		t.Errorf("after Enter: %q, want new", got)
	}
}

// A Y range narrower than the float resolution at its magnitude used to
// hang the paint: v += step did not move v
func TestTimeChartTinyYRange(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		setup func(c *TimeChart)
	}{
		{"manual 1e17", func(c *TimeChart) {
			c.AddArea().SetYRange(1e17, 1e17+16)
		}},
		{"manual empty", func(c *TimeChart) {
			c.AddArea().SetYRange(5, 5)
		}},
		{"manual inverted", func(c *TimeChart) {
			c.AddArea().SetYRange(5, 1)
		}},
		{"data a few ULPs apart", func(c *TimeChart) {
			src := NewTimeChartMemorySource()
			src.AddPoint(NewTimeChartValue(base, 1e6))
			src.AddPoint(NewTimeChartValue(base.Add(time.Second), math.Nextafter(1e6, 2e6)))
			c.AddArea().AddSeries("s", src)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chart := NewTimeChart()
			chart.SetSize(400, 300)
			chart.SetDefaultTimeRange(base.Add(-time.Minute), base.Add(time.Minute))
			tc.setup(chart)
			done := make(chan struct{})
			go func() {
				defer close(done)
				img := image.NewRGBA(image.Rect(0, 0, 400, 300))
				cnv := NewCanvas(img)
				cnv.SetDirectTranslateAndClip(0, 0, 400, 300)
				chart.draw(cnv)
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("draw hangs")
			}
		})
	}
}
