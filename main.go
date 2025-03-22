package main

import (
	"fmt"
	"image"

	"github.com/ipoluianov/nui/nui"
)

func main() {
	nui.Init()
	wnd := nui.CreateWindow()

	wnd.OnKeyDown = func(key nui.Key) {
		if key == nui.KeyEsc {
			wnd.Close()
		}

		if key == nui.KeyF1 {
			wnd.SetTitle("F1 pressed")
		}

		if key == nui.KeyF2 {
			wnd.Resize(1000, 800)
		}

		if key == nui.KeyF3 {
			wnd.Move(100, 100)
		}

		if key == nui.KeyF4 {
			wnd.SetMouseCursor(nui.MouseCursorArrow)
		}
	}

	counter := 0

	wnd.OnPaint = func(width int, height int) *image.RGBA {
		counter++
		fmt.Println("Paint event", counter, "width", width, "height", height)
		testPng := nui.GetRGBATestImage()
		return testPng
	}

	wnd.EventLoop()
}
