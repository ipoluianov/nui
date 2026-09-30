package platforms

import (
	_ "embed"
	"runtime"
)

func init() {
	// The main goroutine stays on the main OS thread for the life of the
	// program. That thread is the UI thread: every window is created there,
	// its event loop runs there and every callback (OnPaint, OnKeyDown,
	// OnTimer, ...) is called there, on all the platforms. Cocoa requires the
	// main thread outright; on Linux and Windows it makes all the windows share
	// one thread, so the handlers of different windows never run concurrently.
	runtime.LockOSThread()
	initUIThread()
}

// IsUIThread reports whether the caller runs on the UI thread - the main OS
// thread, where all the window callbacks are called.
func IsUIThread() bool {
	return isUIThread()
}

// Post schedules f to run on the UI thread and returns at once. It is safe to
// call from any goroutine, including the UI thread itself (f then runs after
// the current callback returns). Functions run in the order they were posted.
//
// f runs only while the UI thread is in an event loop (Run, Exec), so the
// program's main goroutine must eventually call one of them.
func Post(f func()) {
	if f == nil {
		return
	}
	postToUIThread(f)
}

// RunOnUIThread runs f on the UI thread and waits for it to return. Called on
// the UI thread, it runs f at once. Called from another goroutine, it blocks
// until the UI thread picks f up, so it must not be used while the UI thread
// itself waits for the calling goroutine.
func RunOnUIThread(f func()) {
	if f == nil {
		return
	}
	if isUIThread() {
		f()
		return
	}
	done := make(chan struct{})
	postToUIThread(func() {
		defer close(done)
		f()
	})
	<-done
}
