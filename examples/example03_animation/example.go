package example03animation

import (
	"image"
	"image/color"
	"strconv"
	"time"

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
	totalCounter := 0
	counter := 0
	speed := float64(0)
	nui.Init()
	wnd := nui.CreateWindow()
	wnd.Show()
	wnd.OnPaint = func(rgba *image.RGBA) {
		posX := 2 * int(time.Now().UnixMilli()%10000) / 20
		fullRectOnRGBA(rgba, posX, 10, 100, 100, color.RGBA{255, 0, 0, 255})
		cnv := nuicanvas.NewCanvas(rgba)
		counterStr := "Counter: " + strconv.FormatInt(int64(counter), 10)
		cnv.DrawFixedString(10, 120, counterStr, 2, color.RGBA{200, 200, 200, 255})
		speedStr := "Speed: " + strconv.FormatFloat(speed, 'f', 2, 64)
		cnv.DrawFixedString(10, 140, speedStr, 2, color.RGBA{200, 200, 200, 255})
	}
	wnd.OnKeyDown = func(keyCode nui.Key, keyModifiers nui.KeyModifiers) {
		wnd.Resize(800, 600)
	}
	dtBegin := time.Now()
	lastTotalCounter := 0
	wnd.OnTimer = func() {
		counter++
		totalCounter++
		if counter > 100 {
			counter = 0
			dtEnd := time.Now()
			dt := dtEnd.Sub(dtBegin)
			duration := dt.Seconds()
			_ = duration

			speed = float64(totalCounter-lastTotalCounter) / duration
			lastTotalCounter = totalCounter

			dtBegin = time.Now()
		}
		wnd.Update()
	}
	wnd.EventLoop()
}
