package ui

import (
	"testing"
	"time"
)

func auditWithin(t *testing.T, d time.Duration, f func()) {
	t.Helper()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		f()
	}()
	select {
	case r := <-done:
		if r != nil {
			t.Fatalf("panic: %v", r)
		}
	case <-time.After(d):
		t.Fatalf("did not return within %v", d)
	}
}

// A widget spanning a huge number of columns (e.g. from a layout file)
func TestAuditHugeSpan(t *testing.T) {
	form := NewForm()
	form.processResize(500, 400)
	l := NewLabel("x")
	form.Panel().AddWidgetSpan(0, 0, 1, 1<<40, l)
	form.Panel().AddWidget(0, 1, NewLabel("y"))
	auditWithin(t, 10*time.Second, func() { form.layoutNow(); form.Panel().X() })
}

// Stretch weights large enough to overflow their sum
func TestAuditHugeStretch(t *testing.T) {
	form := NewForm()
	form.processResize(500, 400)
	for i := 0; i < 4; i++ {
		form.Panel().AddWidget(0, i, NewLabel("x"))
		form.Panel().SetColumnStretch(i, 1<<62)
	}
	auditWithin(t, 10*time.Second, func() { form.layoutNow() })
}

// Columns limited by a max width with a larger stretch than the others:
// the layout loop must end for every width
func TestAuditStretchLoop(t *testing.T) {
	for _, stretch := range []int{2, 3, 5, 10} {
		for width := 100; width < 700; width += 7 {
			form := NewForm()
			form.processResize(width, 300)
			a := NewPanel()
			a.SetXExpandable(true)
			a.SetMaxWidth(40)
			form.Panel().AddWidget(0, 0, a)
			form.Panel().SetColumnStretch(0, stretch)
			for i := 1; i <= 2; i++ {
				p := NewPanel()
				p.SetXExpandable(true)
				form.Panel().AddWidget(0, i, p)
			}
			t.Logf("stretch %d width %d", stretch, width)
			auditWithin(t, 3*time.Second, func() { form.layoutNow() })
		}
	}
}
