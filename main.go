package main

import (
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
	}

	counter := 0
	testPng := nui.GetRGBATestImage()

	wnd.OnPaint = func(rgba *image.RGBA) {
		counter++

		_ = testPng
		// full with black
		dataSize := rgba.Stride * rgba.Rect.Dy()
		for i := 0; i < dataSize; i++ {
			rgba.Pix[i] = 0
		}
		draw.Draw(rgba, rgba.Rect, testPng, image.Point{0, 0}, draw.Src)
		//fmt.Println("Paint event", counter, "width", width, "height", height)
		//
	}

	wnd.Show()
	wnd.EventLoop()
}
