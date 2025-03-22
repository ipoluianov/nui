package nui

import (
	_ "embed"
	"runtime"
)

//go:embed test.png
var testPng []byte

func Init() {
	runtime.LockOSThread()
}
