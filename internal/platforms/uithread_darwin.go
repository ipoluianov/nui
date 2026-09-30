package platforms

// Cocoa already runs every window on the main thread, the UI thread: its one
// shared run loop serves them all. Posted functions go to the main queue.

func initUIThread() {}

func isUIThread() bool {
	return isMainThread()
}

func postToUIThread(f func()) {
	dispatchAsyncMain(f)
}
