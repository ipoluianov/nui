package nui

import (
	_ "embed"
	"runtime"
)

//go:embed test.png
var TestPng []byte

const (
	DefaultWindowTitle = "NUI Window"
)

func Init() {
	runtime.LockOSThread()
}
