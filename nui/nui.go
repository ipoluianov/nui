package nui

import (
	_ "embed"
	"runtime"
)

func init() {
	// Lock the OS thread to prevent it from being moved to another thread
	// This is important for GUI applications to ensure that the GUI
	// is always run on the same thread
	runtime.LockOSThread()
}

const (
	defaultWindowTitle = "NUI Window"
)
