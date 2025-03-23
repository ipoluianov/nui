package example02paint

import (
	"image"
	"image/color"
	"strconv"

	"github.com/ipoluianov/nui/nui"
	"github.com/ipoluianov/nui/nuicanvas"
)

func fullRectOnRGBA(rgba *image.RGBA, x, y, w, h int, c color.Color) {
	for i := x; i < x+w; i++ {
		for j := y; j < y+h; j++ {
			rgba.Set(i, j, c)
		}
	}
}

func Run() {
	nui.Init()
	wnd := nui.CreateWindow()

	counter := 0

	scrollPosX := float64(0)
	scrollPosY := float64(0)

	wnd.OnPaint = func(rgba *image.RGBA) {
		cnv := nuicanvas.NewCanvas(rgba)
		_ = cnv
		//cnv.Clear(color.RGBA{0, 0, 0, 255})
		cnv.DrawRect(10, 10, 100, 100, color.RGBA{255, 0, 0, 255})
		counterStr := "Counter: " + strconv.FormatInt(int64(counter), 10)
		cnv.DrawFixedString(10, 120, counterStr, 2, color.RGBA{200, 200, 200, 255})

		scrollXStr := "ScrollX: " + strconv.FormatFloat(scrollPosX, 'f', 2, 64)
		scrollYStr := "ScrollY: " + strconv.FormatFloat(scrollPosY, 'f', 2, 64)

		cnv.DrawFixedString(10, 140, scrollXStr, 2, color.RGBA{200, 200, 200, 255})
		cnv.DrawFixedString(10, 160, scrollYStr, 2, color.RGBA{200, 200, 200, 255})
	}

	wnd.OnMouseWheel = func(deltaX float64, deltaY float64) {
		scrollPosX += float64(deltaX)
		scrollPosY += float64(deltaY)
		wnd.Update()
	}

	wnd.OnTimer = func() {
		counter++
		wnd.Update()
	}

	wnd.Show()
	//wnd.MoveToCenterOfScreen()
	wnd.Resize(300, 300)
	//wnd.MoveToCenterOfScreen()
	//wnd.MaximizeWindow()
	wnd.EventLoop()
}
