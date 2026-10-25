package ui

import "github.com/ipoluianov/nui/internal/platforms"

// Threading. Every form lives on the UI thread - the main OS thread, where
// the program's main goroutine runs Form.Exec. All the handlers (button
// clicks, OnClose, timers, painting, ...) of all the forms are called on that
// one thread, one at a time, so a handler of one form may freely use the
// widgets of another form.
//
// Other goroutines must not touch the forms and widgets directly: they hand
// the work over to the UI thread with Invoke (or InvokeSync, Form.Invoke).

// Invoke runs f on the UI thread and returns at once; the forms are
// repainted after it, as after a handler. Safe to call from any goroutine;
// called on the UI thread, f runs after the current handler.
func Invoke(f func()) {
	if f == nil {
		return
	}
	platforms.Post(func() {
		f()
		repaintOpenForms()
	})
}

// InvokeSync runs f on the UI thread and waits for it to return; the forms
// are repainted after it. On the UI thread it just calls f. Don't call it
// from a goroutine the UI thread is waiting for: that would deadlock.
func InvokeSync(f func()) {
	if f == nil {
		return
	}
	platforms.RunOnUIThread(func() {
		f()
		repaintOpenForms()
	})
}

// repaintOpenForms has the open forms repainted with the next frame: f may
// have changed any of them
func repaintOpenForms() {
	for _, form := range openForms() {
		form.needUpdate = true
	}
}

// IsUIThread reports whether the caller runs on the UI thread
func IsUIThread() bool {
	return platforms.IsUIThread()
}
