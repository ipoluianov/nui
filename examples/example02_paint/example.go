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

	wnd.OnPaint = func(rgba *image.RGBA) {
		cnv := nuicanvas.NewCanvas(rgba)
		_ = cnv
		//cnv.Clear(color.RGBA{0, 0, 0, 255})
		cnv.DrawRect(10, 10, 100, 100, color.RGBA{255, 0, 0, 255})
		counterStr := "Counter: " + strconv.FormatInt(int64(counter), 10)
		cnv.DrawFixedString(10, 120, counterStr, 2, color.RGBA{200, 200, 200, 255})
	}

	wnd.OnTimer = func() {
		counter++
		wnd.Update()
	}

	wnd.Show()
	wnd.MoveToCenterOfScreen()
	wnd.Resize(900, 900)
	wnd.MoveToCenterOfScreen()
	wnd.MaximizeWindow()
	wnd.EventLoop()
}
