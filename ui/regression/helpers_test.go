// Package regression holds the tests of the behavior of nui as an
// application sees it, kept across the changes of the library: the bugs
// fixed and the rules around them, by area (layout_test.go,
// events_test.go...).
package regression

import (
	"github.com/ipoluianov/nui/ui"
)

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
