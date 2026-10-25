package regression

import (
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func TestTimer(t *testing.T) {
	form := newForm(200, 100)
	panel := form.Panel()
	w := uitest.Show(t, form)

	// AddTimer: at the next tick, then every interval, until Stop
	runs := 0
	timer := panel.AddTimer(20, func() { runs++ })
	w.Tick()
	if runs != 1 {
		t.Fatalf("first tick: %d runs", runs)
	}
	w.Wait(110 * time.Millisecond)
	if runs < 3 || runs > 7 {
		t.Errorf("runs in 110 ms every 20 ms: %d", runs)
	}
	timer.Stop()
	before := runs
	w.Wait(60 * time.Millisecond)
	if runs != before || timer.Running() {
		t.Errorf("after Stop: %d runs, running %v", runs-before, timer.Running())
	}

	// AfterFunc: once
	once := 0
	panel.AfterFunc(20*time.Millisecond, func() { once++ })
	w.Wait(100 * time.Millisecond)
	if once != 1 {
		t.Errorf("AfterFunc ran %d times", once)
	}

	// AfterFunc(0): at the next tick, once
	zero := 0
	panel.AfterFunc(0, func() { zero++ })
	w.Tick()
	w.Tick()
	if zero != 1 {
		t.Errorf("AfterFunc(0) ran %d times", zero)
	}

	// A debounce: restarted before its interval, it runs once after the
	// last Start
	searches := 0
	search := panel.NewTimer(60*time.Millisecond, func() { searches++ })
	search.SetSingleShot(true)
	for range 4 {
		search.Start()
		w.Wait(20 * time.Millisecond)
	}
	if searches != 0 {
		t.Errorf("ran %d times while restarted", searches)
	}
	w.Wait(120 * time.Millisecond)
	if searches != 1 {
		t.Errorf("after the pause: %d runs", searches)
	}
}

// A timer stopping itself, or another one, in its function
func TestTimerStopsInFunction(t *testing.T) {
	form := newForm(200, 100)
	w := uitest.Show(t, form)
	var a, b *ui.Timer
	aRuns, bRuns := 0, 0
	a = form.Panel().AddTimer(10, func() { aRuns++; a.Stop(); b.Stop() })
	b = form.Panel().AddTimer(10, func() { bRuns++ })
	w.Wait(50 * time.Millisecond)
	if aRuns != 1 || bRuns != 0 {
		t.Errorf("a %d, b %d", aRuns, bRuns)
	}
}
