package example02paint

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

var logItems = make([]string, 0)

func log(s string) {
	dtStr := time.Now().Format("2006-01-02 15:04:05.999")
	if len(dtStr) < 23 {
		dtStr += "0"
	}

	s = dtStr + " " + s
	logItems = append(logItems, s)
	if len(logItems) > 10 {
		logItems = logItems[1:]
	}
}

func Run() {
	nui.Init()
	wnd := nui.CreateWindow()

	log("started")

	counter := 0

	lastMousePosX := int(0)
	lastMousePosY := int(0)

	scrollPosX := float64(0)
	scrollPosY := float64(0)

	mouseLeftButtonStatus := false
	mouseMiddleButtonStatus := false
	mouseRightButtonStatus := false

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

		mouseXStr := "MouseX: " + strconv.FormatInt(int64(lastMousePosX), 10)
		mouseYStr := "MouseY: " + strconv.FormatInt(int64(lastMousePosY), 10)

		cnv.DrawFixedString(10, 180, mouseXStr, 2, color.RGBA{200, 200, 200, 255})
		cnv.DrawFixedString(10, 200, mouseYStr, 2, color.RGBA{200, 200, 200, 255})

		mouseButtonLeftStr := "Mouse Button Left: "
		if mouseLeftButtonStatus {
			mouseButtonLeftStr += "pressed"
		}
		cnv.DrawFixedString(10, 220, mouseButtonLeftStr, 2, color.RGBA{200, 200, 200, 255})

		mouseButtonMiddleStr := "Mouse Button Middle: "
		if mouseMiddleButtonStatus {
			mouseButtonMiddleStr += "pressed"
		}
		cnv.DrawFixedString(10, 240, mouseButtonMiddleStr, 2, color.RGBA{200, 200, 200, 255})

		mouseButtonRightStr := "Mouse Button Right: "
		if mouseRightButtonStatus {
			mouseButtonRightStr += "pressed"
		}
		cnv.DrawFixedString(10, 260, mouseButtonRightStr, 2, color.RGBA{200, 200, 200, 255})

		winWidth := wnd.Width()
		winHeight := wnd.Height()

		windowWidthStr := "Window Width: " + strconv.FormatInt(int64(winWidth), 10)
		cnv.DrawFixedString(10, 280, windowWidthStr, 2, color.RGBA{200, 200, 200, 255})
		windowHeightStr := "Window Height: " + strconv.FormatInt(int64(winHeight), 10)
		cnv.DrawFixedString(10, 300, windowHeightStr, 2, color.RGBA{200, 200, 200, 255})

		for i, s := range logItems {
			cnv.DrawFixedString(10, 340+20*i, s, 2, color.RGBA{200, 200, 200, 255})
		}

		mods := wnd.KeyModifiers()

		keyModifiersShift := "Shift:" + strconv.FormatBool(mods.Shift)
		keyModifiersCtrl := "Ctrl:" + strconv.FormatBool(mods.Ctrl)
		keyModifiersAlt := "Alt:" + strconv.FormatBool(mods.Alt)
		keyModifiersCmd := "Cmd:" + strconv.FormatBool(mods.Cmd)

		cnv.DrawFixedString(600, 0, keyModifiersShift, 2, color.RGBA{200, 200, 200, 255})
		cnv.DrawFixedString(600, 20, keyModifiersCtrl, 2, color.RGBA{200, 200, 200, 255})
		cnv.DrawFixedString(600, 40, keyModifiersAlt, 2, color.RGBA{200, 200, 200, 255})
		cnv.DrawFixedString(600, 60, keyModifiersCmd, 2, color.RGBA{200, 200, 200, 255})
	}

	wnd.OnMouseWheel = func(deltaX int, deltaY int) {
		scrollPosX += float64(deltaX)
		scrollPosY += float64(deltaY)
		log("Mouse wheel: " + strconv.FormatInt(int64(deltaX), 10) + " " + strconv.FormatInt(int64(deltaY), 10))
	}

	wnd.OnMouseButtonDown = func(button nui.MouseButton, x, y int) {
		log("Mouse button down: " + button.String())
		switch button {
		case nui.MouseButtonLeft:
			mouseLeftButtonStatus = true
		case nui.MouseButtonMiddle:
			mouseMiddleButtonStatus = true
		case nui.MouseButtonRight:
			mouseRightButtonStatus = true
		}
	}

	wnd.OnMouseButtonUp = func(button nui.MouseButton, x, y int) {
		log("Mouse button up: " + button.String())
		switch button {
		case nui.MouseButtonLeft:
			mouseLeftButtonStatus = false
		case nui.MouseButtonMiddle:
			mouseMiddleButtonStatus = false
		case nui.MouseButtonRight:
			mouseRightButtonStatus = false
		}
	}

	wnd.OnChar = func(char rune) {
		log("Char: " + string(char))
	}

	wnd.OnKeyDown = func(key nui.Key, mods nui.KeyModifiers) {
		log("Key down: " + key.String() + " " + mods.String())
	}

	wnd.OnKeyUp = func(key nui.Key, mods nui.KeyModifiers) {
		log("Key up: " + key.String() + " " + mods.String())
	}

	wnd.OnMouseMove = func(x, y int) {
		lastMousePosX = x
		lastMousePosY = y
	}

	wnd.OnTimer = func() {
		counter++
		wnd.Update()
	}

	wnd.Show()
	//wnd.MoveToCenterOfScreen()
	wnd.Resize(800, 600)
	//wnd.MoveToCenterOfScreen()
	//wnd.MaximizeWindow()
	wnd.EventLoop()
}
