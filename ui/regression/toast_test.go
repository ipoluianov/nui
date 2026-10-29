package regression

import (
	"bytes"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// A form hidden in the tray shows no toasts: the toast's own window was
// shown over the other applications, with the form's window not on the
// screen (the reminders of the Tasks example)
func TestNoToastOnHiddenForm(t *testing.T) {
	form := newForm(400, 300)
	w := uitest.Show(t, form)
	before := w.Screenshot()

	form.Hide()
	form.ShowToastFor("Reminder", ui.ToastWarning, time.Hour)
	w.Settle()
	form.Show()
	after := w.Screenshot()
	if !bytes.Equal(before.Pix, after.Pix) {
		t.Fatal("the toast shown while the form was hidden is on the screen")
	}

	// A shown form shows them
	form.ShowToastFor("Saved", ui.ToastSuccess, time.Hour)
	if bytes.Equal(before.Pix, w.Screenshot().Pix) {
		t.Fatal("no toast on the shown form")
	}
}
