package main

import (
	"fmt"
	"image"
	"image/draw"

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

		if key == nui.KeyF5 {
			wnd.SetMouseCursor(nui.MouseCursorPointer)
		}

	}

	wnd.OnMouseEnter = func() {
		fmt.Println("Mouse enter")
	}

	wnd.OnMouseLeave = func() {
		fmt.Println("Mouse leave")
	}

	wnd.OnMouseMove = func(x, y int) {
		//fmt.Printf("Mouse move: %d, %d\n", x, y)
	}

	counter := 0
	testPng := nui.GetRGBATestImage()
	wnd.OnPaint = func(rgba *image.RGBA) {
		counter++
		draw.Draw(rgba, rgba.Rect, testPng, image.Point{0, 0}, draw.Src)
	}

	wnd.Show()
	wnd.EventLoop()
}
