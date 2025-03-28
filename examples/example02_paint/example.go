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
	if len(logItems) > 40 {
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
		cnv.SetColor(color.RGBA{200, 200, 200, 255})
		cnv.SetPixel(110, 110, 1)
		cnv.SetPixel(113.1, 113.1, 1)

		cnv.DrawLineSDF(20, 20, 220, 150, 10)

		//cnv.Clear(color.RGBA{0, 0, 0, 255})
		cnv.DrawRect(0, 0, 100, 100)
		//cnv.DrawCircleAA(50, 50, 50)
		counterStr := "Counter: " + strconv.FormatInt(int64(counter), 10)
		cnv.DrawFixedString(10, 120, counterStr, 2)

		scrollXStr := "ScrollX: " + strconv.FormatFloat(scrollPosX, 'f', 2, 64)
		scrollYStr := "ScrollY: " + strconv.FormatFloat(scrollPosY, 'f', 2, 64)

		cnv.DrawFixedString(10, 140, scrollXStr, 2)
		cnv.DrawFixedString(10, 160, scrollYStr, 2)

		mouseXStr := "MouseX: " + strconv.FormatInt(int64(lastMousePosX), 10)
		mouseYStr := "MouseY: " + strconv.FormatInt(int64(lastMousePosY), 10)

		cnv.DrawFixedString(10, 180, mouseXStr, 2)
		cnv.DrawFixedString(10, 200, mouseYStr, 2)

		mouseButtonLeftStr := "Mouse Button Left: "
		if mouseLeftButtonStatus {
			mouseButtonLeftStr += "pressed"
		}
		cnv.DrawFixedString(10, 220, mouseButtonLeftStr, 2)

		mouseButtonMiddleStr := "Mouse Button Middle: "
		if mouseMiddleButtonStatus {
			mouseButtonMiddleStr += "pressed"
		}
		cnv.DrawFixedString(10, 240, mouseButtonMiddleStr, 2)

		mouseButtonRightStr := "Mouse Button Right: "
		if mouseRightButtonStatus {
			mouseButtonRightStr += "pressed"
		}
		cnv.DrawFixedString(10, 260, mouseButtonRightStr, 2)

		winPosX := wnd.PosX()
		winPosY := wnd.PosY()
		windowPosXStr := "Window PosX: " + strconv.FormatInt(int64(winPosX), 10)
		cnv.DrawFixedString(10, 280, windowPosXStr, 2)
		windowPosYStr := "Window PosY: " + strconv.FormatInt(int64(winPosY), 10)
		cnv.DrawFixedString(10, 300, windowPosYStr, 2)

		winWidth := wnd.Width()
		winHeight := wnd.Height()
		windowWidthStr := "Window Width: " + strconv.FormatInt(int64(winWidth), 10)
		cnv.DrawFixedString(10, 320, windowWidthStr, 2)
		windowHeightStr := "Window Height: " + strconv.FormatInt(int64(winHeight), 10)
		cnv.DrawFixedString(10, 340, windowHeightStr, 2)

		mods := wnd.KeyModifiers()

		keyModifiersShift := "Shift:" + strconv.FormatBool(mods.Shift)
		keyModifiersCtrl := "Ctrl:" + strconv.FormatBool(mods.Ctrl)
		keyModifiersAlt := "Alt:" + strconv.FormatBool(mods.Alt)
		keyModifiersCmd := "Cmd:" + strconv.FormatBool(mods.Cmd)

		cnv.DrawFixedString(10, 360, keyModifiersShift, 2)
		cnv.DrawFixedString(10, 380, keyModifiersCtrl, 2)
		cnv.DrawFixedString(10, 400, keyModifiersAlt, 2)
		cnv.DrawFixedString(10, 420, keyModifiersCmd, 2)

		for i, s := range logItems {
			cnv.DrawFixedString(600, float64(10+20*i), s, 2)
		}
	}

	wnd.OnMove = func(x, y int) {
		log("Window moved: " + strconv.FormatInt(int64(x), 10) + " " + strconv.FormatInt(int64(y), 10))
	}

	wnd.OnResize = func(w, h int) {
		log("Window resized: " + strconv.FormatInt(int64(w), 10) + " " + strconv.FormatInt(int64(h), 10))
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
