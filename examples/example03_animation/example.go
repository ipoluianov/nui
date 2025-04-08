package example03animation

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"time"

	"github.com/ipoluianov/nui/nui"
	"github.com/ipoluianov/nui/nuicanvas"
	"github.com/ipoluianov/nui/nuikey"
)

func fullRectOnRGBA(rgba *image.RGBA, x, y, w, h int, c color.Color) {
	for i := x; i < x+w; i++ {
		for j := y; j < y+h; j++ {
			rgba.Set(i, j, c)
		}
	}
}

func Run() {
	drawTimes := make([]int, 20)
	drawTimesOffset := 0
	lastDrawTime := time.Now()

	calcMinTime := func() int {
		min := 1000000
		for i := 0; i < len(drawTimes); i++ {
			if drawTimes[i] < min {
				min = drawTimes[i]
			}
		}
		return min
	}

	calcMaxTime := func() int {
		max := 0
		for i := 0; i < len(drawTimes); i++ {
			if drawTimes[i] > max {
				max = drawTimes[i]
			}
		}
		return max
	}

	totalCounter := 0
	counter := 0
	speed := float64(0)
	offset := 0
	wnd := nui.CreateWindow("App", 800, 600, true)
	wnd.Show()
	wnd.OnPaint(func(rgba *image.RGBA) {
		posX := offset
		fullRectOnRGBA(rgba, posX, 10, 100, 100, color.RGBA{255, 0, 0, 255})
		cnv := nuicanvas.NewCanvas(rgba)
		cnv.SetColor(color.RGBA{200, 200, 200, 255})
		counterStr := "Counter: " + strconv.FormatInt(int64(counter), 10)
		cnv.DrawFixedString(10, 120, counterStr, 2)
		speedStr := "Speed: " + strconv.FormatFloat(speed, 'f', 2, 64)
		cnv.DrawFixedString(10, 140, speedStr, 2)

		minStr := "MinTime: " + fmt.Sprint(calcMinTime())
		cnv.DrawFixedString(10, 160, minStr, 2)

		maxStr := "MaxTime: " + fmt.Sprint(calcMaxTime())
		cnv.DrawFixedString(10, 180, maxStr, 2)

		drawTimeStr := "DrawTime:" + fmt.Sprint(wnd.DrawTimeUs()/1000)
		cnv.DrawFixedString(10, 200, drawTimeStr, 2)

	})
	wnd.OnKeyDown(func(keyCode nuikey.Key, keyModifiers nuikey.KeyModifiers) {
		wnd.Resize(800, 600)
	})
	dtBegin := time.Now()
	lastTotalCounter := 0

	dtLastTimer := time.Now()

	wnd.OnTimer(func() {
		dt := time.Since(dtLastTimer)
		if dt.Milliseconds() < 50 {
			return
		}
		dtLastTimer = time.Now()

		delta := int(time.Since(lastDrawTime).Milliseconds())
		fmt.Println(time.Now().Format("15:05:06.999"), "\t", "delta", delta)
		drawTimes[drawTimesOffset] = delta
		drawTimesOffset++
		if drawTimesOffset >= len(drawTimes) {
			drawTimesOffset = 0
		}
		lastDrawTime = time.Now()

		offset++

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
	})
	wnd.EventLoop()
}
