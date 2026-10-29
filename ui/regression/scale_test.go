package regression

import (
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// Two windows on screens of different scales, e.g. a monitor at 150% next
// to one at 100%: the text is measured for the screen of each form. With
// one scale for all, the paint of one window laid the other out again for
// its scale and repainted it, for ever, and the texts of both jumped.
func TestWindowsOnScreensOfDifferentScales(t *testing.T) {
	show := func(scale float64) *uitest.Window {
		uitest.SetScreen(1920, 1080, scale)
		defer uitest.SetScreen(1920, 1080, 1)
		form := newForm(400, 200)
		tb := ui.NewTextBox()
		tb.SetText("Съешь ещё этих мягких французских булок")
		form.Panel().AddWidget(0, 0, tb)
		form.Panel().AddVSpacer(1, 0)
		return uitest.Show(t, form)
	}
	a, b := show(1), show(1.5)
	paint := func() {
		for range 3 {
			a.Screenshot()
			b.Screenshot()
		}
	}
	paint()
	ua, ub := a.Updates(), b.Updates()
	paint()
	if a.Updates() != ua || b.Updates() != ub {
		t.Fatalf("painting each other repainted them: %d and %d repaints", a.Updates()-ua, b.Updates()-ub)
	}

	// Dragged to the other screen, a window is laid out for it once
	a.MoveToScreen(1.5)
	paint()
	ua = a.Updates()
	paint()
	if a.Updates() != ua || b.Updates() != ub {
		t.Fatalf("after the move: %d and %d repaints", a.Updates()-ua, b.Updates()-ub)
	}
}
