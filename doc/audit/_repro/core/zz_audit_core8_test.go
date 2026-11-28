package ui

import (
	"strings"
	"testing"

	"github.com/ipoluianov/nui/internal/platforms"
)

// Enter after the dialog button box was taken out of the form
func TestAuditRemovedAcceptButton(t *testing.T) {
	ok := 0
	box := NewDialogButtonBox()
	box.AddOK(func() { ok++ })
	form := newTestForm(t, box)
	form.Panel().RemoveWidget(box)
	form.processKeyDown(KeyEnter, KeyModifiers{})
	if ok != 0 {
		t.Errorf("Enter clicked the OK button of a removed DialogButtonBox (%d clicks)", ok)
	}
}

// A toast of a long text (a log, a stack trace) gets a popup as tall as all
// of its lines
func TestAuditToastHuge(t *testing.T) {
	platforms.SetHeadless()
	f := NewForm()
	f.Show()
	defer f.Close()
	f.ShowToast(strings.Repeat("line\n", 100000), ToastError)
	tt := f.toasts[len(f.toasts)-1]
	t.Logf("toast %dx%d, %d lines", tt.w, tt.h, len(tt.lines))
	if tt.h > 2000 {
		t.Errorf("toast popup height %d px (%d MB of pixels for its window)", tt.h, tt.w*tt.h*4>>20)
	}
}

// A shared action in menus built per showing: every menu ever shown keeps a
// handler on the action
func TestAuditActionHandlersGrow(t *testing.T) {
	form := newTestForm(t, NewLabel("x"))
	a := NewAction("Copy", func() {})
	for i := 0; i < 100; i++ {
		m := NewContextMenu(nil)
		m.AddAction(a)
		form.ShowContextMenu(m, 10, 10)
		form.CloseTopPopup()
	}
	t.Logf("handlers on the action: %d", len(a.OnChanged.handlers))
	if len(a.OnChanged.handlers) > 1 {
		t.Errorf("%d handlers left on the action after 100 menus were shown and closed", len(a.OnChanged.handlers))
	}
}
