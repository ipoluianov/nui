package nui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics
#include "window.h"
*/
import "C"
import "fmt"

func DDD() {
	fmt.Println("DDD1")
	C.InitWindow()
	fmt.Println("DDD2")
	C.RunEventLoop()
}
