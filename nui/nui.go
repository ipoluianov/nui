package nui

import (
	_ "embed"
	"runtime"
)

//go:embed test.png
var TestPng []byte

func Init() {
	runtime.LockOSThread()
}
