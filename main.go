package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/ipoluianov/nui/nui"
)

func fullRectOnRGBA(rgba *image.RGBA, x, y, w, h int, c color.Color) {
	for i := x; i < x+w; i++ {
		for j := y; j < y+h; j++ {
			rgba.Set(i, j, c)
		}
	}
}

func main() {
	nui.Init()
	wnd := nui.CreateWindow()
	wnd.OnKeyDown = func(key nui.Key) {
		fmt.Println("Key down:", key.String())

		if key == nui.KeyEsc {
			wnd.Close()
		}

		if key == nui.KeyF1 {
			wnd.SetTitle("F1 pressed")
		}

		if key == nui.KeyF2 {
			wnd.Resize(200, 100)
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

		if key == nui.KeyF6 {
			wnd.MaximizeWindow()
		}

		if key == nui.KeyF7 {
			wnd.MinimizeWindow()
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

	wnd.OnCloseRequest = func() bool {
		fmt.Println("Close request")
		return true
	}

	counter := 0
	testPng := nui.GetRGBATestImage()
	wnd.OnPaint = func(rgba *image.RGBA) {
		counter++
		draw.Draw(rgba, rgba.Rect, testPng, image.Point{0, 0}, draw.Src)
		fullRectOnRGBA(rgba, 100, 100, 100, 100, color.RGBA{255, 0, 0, 255})
		fullRectOnRGBA(rgba, 200, 200, 100, 100, color.RGBA{0, 255, 0, 255})
		fullRectOnRGBA(rgba, 300, 300, 100, 100, color.RGBA{0, 0, 255, 255})
	}

	wnd.Show()
	wnd.EventLoop()
}
