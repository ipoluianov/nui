// Package regression holds the tests of the bugs of nui
// (doc/production-readiness.md, section 5) and of the behavior around them.
//
// A known bug gets a test that reproduces it with knownBug: while the bug is
// there, the test is skipped with its number; once the bug is fixed, the
// test fails, asking to turn it into an ordinary test (drop the knownBug
// call and keep the check). So a fix is noticed, and a bug that comes back
// is caught. All the bugs with such tests so far are fixed: their tests are
// in the files of their areas (layout_test.go, events_test.go...).
package regression

import (
	"testing"

	"github.com/ipoluianov/nui/ui"
)

// knownBug checks the known bug number n: reproduced, it skips the test;
// not reproduced, it fails it
func knownBug(t *testing.T, n int, reproduced bool, what string) {
	t.Helper()
	if reproduced {
		t.Skipf("known bug #%d: %s", n, what)
	}
	t.Errorf("known bug #%d seems fixed (%s): make this an ordinary test", n, what)
}

var _ = knownBug // for the next known bug

func newForm(width, height int) *ui.Form {
	form := ui.NewForm()
	form.SetSize(width, height)
	return form
}

// paintBox is a widget the test paints itself
func paintBox(paint func(cnv *ui.Canvas)) *ui.Widget {
	w := &ui.Widget{}
	w.InitWidget()
	w.SetMinSize(100, 60)
	w.SetOnPaint(paint)
	return w
}
