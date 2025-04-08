package example00demo

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/ipoluianov/nui/nui"
	"github.com/ipoluianov/nui/nuicanvas"
	"github.com/ipoluianov/nui/nuikey"
	"github.com/ipoluianov/nui/nuimouse"
)

var logItems = make([]string, 0)

func log(s string) {
	dtStr := time.Now().Format("15:04:05.999")
	for len(dtStr) < 12 {
		dtStr += "0"
	}

	s = dtStr + " " + s
	logItems = append(logItems, s)
	if len(logItems) > 20 {
		logItems = logItems[1:]
	}
}

func Run() {
	win := nui.CreateWindow("App", 800, 600, true)

	var timerCounter = 0
	var mousePosX, mousePosY = 0, 0
	var winPosX, winPosY = 0, 0
	var winWidth, winHeight = 0, 0
	var mouseWheelX, mouseWheelY = 0, 0
	var animationOffset = 0

	win.OnKeyDown(func(keyCode nuikey.Key, modifiers nuikey.KeyModifiers) {
		modStr := modifiers.String()
		if len(modStr) > 0 {
			modStr = " + " + modStr
		}
		log("OnKeyDown: " + keyCode.String() + modStr)
		switch keyCode {
		case nuikey.KeyEsc:
			logItems = nil
		case nuikey.KeyF1:
			win.MaximizeWindow()
		case nuikey.KeyF2:
			win.MinimizeWindow()
		case nuikey.KeyF3:
			win.SetTitle("Title: " + time.Now().Format("15:04:05"))
		case nuikey.KeyF4:
			win.Resize(640, 480)
		case nuikey.KeyF5:
			win.Move(100, 100)
		case nuikey.KeyF6:
			win.MoveToCenterOfScreen()
		case nuikey.KeyF7:
			{
				iconImg := image.NewRGBA(image.Rect(0, 0, 16, 16))
				cnv := nuicanvas.NewCanvas(iconImg)
				cnv.SetColor(color.RGBA{0, 0, 0, 255})
				cnv.FillRect(0, 0, 16, 16, 1)
				cnv.SetColor(color.RGBA{0, 150, 200, 255})
				cnv.DrawFixedString(0, 4, "NUI", 1)
				win.SetAppIcon(iconImg)
			}
		case nuikey.KeyF8:
			win.SetMouseCursor(nuimouse.MouseCursorArrow)
		case nuikey.KeyF9:
			win.SetMouseCursor(nuimouse.MouseCursorPointer)
		case nuikey.KeyF10:
			win.SetMouseCursor(nuimouse.MouseCursorIBeam)
		case nuikey.KeyF12:
			win.Close()
		case nuikey.Key1:
			win.SetBackgroundColor(color.RGBA{0, 0, 0, 255})
		case nuikey.Key2:
			win.SetBackgroundColor(color.RGBA{255, 0, 0, 255})
		case nuikey.Key3:
			win.SetBackgroundColor(color.RGBA{0, 255, 0, 255})
		case nuikey.Key4:
			win.SetBackgroundColor(color.RGBA{0, 0, 255, 255})
		case nuikey.Key5:
			win.SetBackgroundColor(color.RGBA{255, 255, 255, 255})
		}
		win.Update()
	})

	dtAnimation := time.Now()

	win.OnTimer(func() {
		timerCounter++
		if time.Since(dtAnimation) > 50*time.Millisecond {
			dtAnimation = time.Now()
			animationOffset += 1
			if animationOffset > win.Width() {
				animationOffset = 0
			}
		}
		win.Update()
	})

	win.OnKeyUp(func(keyCode nuikey.Key, modifiers nuikey.KeyModifiers) {
		modStr := modifiers.String()
		if len(modStr) > 0 {
			modStr = " + " + modStr
		}
		log("OnKeyUp: " + keyCode.String() + modStr)
		win.Update()
	})

	win.OnChar(func(char rune) {
		log("OnChar: " + string(char))
		win.Update()
	})

	win.OnMouseLeave(func() {
		log("OnMouseLeave")
		win.Update()
	})

	win.OnMouseEnter(func() {
		log("OnMouseEnter")
		win.Update()
	})

	win.OnMouseMove(func(x, y int) {
		mousePosX = x
		mousePosY = y
		win.Update()
	})

	win.OnMouseButtonDblClick(func(button nuimouse.MouseButton, x, y int) {
		log(fmt.Sprintf("OnMouseButtonDblClick: %s (%d, %d)", button.String(), x, y))
		win.Update()
	})

	win.OnMouseButtonDown(func(button nuimouse.MouseButton, x, y int) {
		log(fmt.Sprintf("OnMouseButtonDown: %s (%d, %d)", button.String(), x, y))
		win.Update()
	})

	win.OnMouseButtonUp(func(button nuimouse.MouseButton, x, y int) {
		log(fmt.Sprintf("OnMouseButtonUp: %s (%d, %d)", button.String(), x, y))
		win.Update()
	})

	win.OnMouseWheel(func(deltaX, deltaY int) {
		log(fmt.Sprintf("OnMouseWheel: %d %d", deltaX, deltaY))
		mouseWheelX += deltaX
		mouseWheelY += deltaY
		win.Update()
	})

	win.OnMove(func(x, y int) {
		winPosX = x
		winPosY = y
		log(fmt.Sprintf("OnMove: %d %d", x, y))
		win.Update()
	})

	win.OnResize(func(width, height int) {
		winWidth = width
		winHeight = height
		log(fmt.Sprintf("OnResize: %d %d", width, height))
		win.Update()
	})

	win.OnPaint(func(rgba *image.RGBA) {
		cnv := nuicanvas.NewCanvas(rgba)
		cnv.SetColor(color.RGBA{0, 255, 0, 255})

		// legend
		cnv.DrawFixedString(10, 10, "Press F1 to maximize window", 2)
		cnv.DrawFixedString(10, 30, "Press F2 to minimize window", 2)
		cnv.DrawFixedString(10, 50, "Press F3 to change title", 2)
		cnv.DrawFixedString(10, 70, "Press F4 to resize window", 2)
		cnv.DrawFixedString(10, 90, "Press F5 to move window", 2)
		cnv.DrawFixedString(10, 110, "Press F6 to center window", 2)
		cnv.DrawFixedString(10, 130, "Press F7 to set app icon", 2)
		cnv.DrawFixedString(10, 150, "Press F8 to set arrow cursor", 2)
		cnv.DrawFixedString(10, 170, "Press F9 to set pointer cursor", 2)
		cnv.DrawFixedString(10, 190, "Press F10 to set IBeam cursor", 2)
		cnv.DrawFixedString(10, 210, "Press F12 to close window", 2)

		cnv.DrawFixedString(10, 230, "Timer: "+fmt.Sprint(timerCounter), 2)
		cnv.DrawFixedString(10, 250, "MouseX: "+fmt.Sprint(mousePosX), 2)
		cnv.DrawFixedString(10, 270, "MouseY: "+fmt.Sprint(mousePosY), 2)
		cnv.DrawFixedString(10, 290, "WinX: "+fmt.Sprint(winPosX), 2)
		cnv.DrawFixedString(10, 310, "WinY: "+fmt.Sprint(winPosY), 2)
		cnv.DrawFixedString(10, 330, "WinW: "+fmt.Sprint(winWidth), 2)
		cnv.DrawFixedString(10, 350, "WinH: "+fmt.Sprint(winHeight), 2)
		cnv.DrawFixedString(10, 370, "MouseWheelX: "+fmt.Sprint(mouseWheelX), 2)
		cnv.DrawFixedString(10, 390, "MouseWheelY: "+fmt.Sprint(mouseWheelY), 2)
		cnv.DrawFixedString(10, 410, "DrawTimeMs: "+fmt.Sprint(win.DrawTimeUs()/1000), 2)

		cnv.DrawLine(5, 430, win.Width()-5, 430, 0.5)
		cnv.DrawLine(390, 5, 390, 425, 0.5)
		cnv.FillRect(animationOffset, 440, 20, 20, 0.5)

		cnv.DrawFixedString(400, 10, "Press Esc to clear log", 2)
		for i, s := range logItems {
			cnv.DrawFixedString(400, 20+float64(10+20*i), s, 2)
		}
	})

	win.Show()
	win.MoveToCenterOfScreen()

	winPosX = win.PosX()
	winPosY = win.PosY()
	winWidth = win.Width()
	winHeight = win.Height()

	win.EventLoop()
}
