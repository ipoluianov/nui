//go:build linux || windows

package platforms

import "sync"

// The functions posted to the UI thread (see Post). The platform event loop
// runs them when wakeUIThread wakes it: Linux polls a pipe next to the X
// connection, Windows handles a message sent to a hidden window.
var (
	postedMu sync.Mutex
	posted   []func()
)

func postToUIThread(f func()) {
	postedMu.Lock()
	posted = append(posted, f)
	postedMu.Unlock()
	wakeUIThread()
}

// runPosted runs the functions posted so far. UI thread only. A function
// posted while these run waits for the next call, so a function that keeps
// re-posting itself can't lock the event loop up.
func runPosted() {
	postedMu.Lock()
	queue := posted
	posted = nil
	postedMu.Unlock()
	for _, f := range queue {
		f()
	}
}
