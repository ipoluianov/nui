package example00demo

import (
	"image"
	"image/color"
	"time"

	"github.com/ipoluianov/nui/nui"
	"github.com/ipoluianov/nui/nuicanvas"
)

var logItems = make([]string, 0)

func log(s string) {
	dtStr := time.Now().Format("15:04:05.999")
	for len(dtStr) < 12 {
		dtStr += "0"
	}

	s = dtStr + " " + s
	logItems = append(logItems, s)
	if len(logItems) > 40 {
		logItems = logItems[1:]
	}
}

func Run() {
	win := nui.CreateWindow()

	win.OnKeyDown = func(keyCode nui.Key, modifiers nui.KeyModifiers) {
		modStr := modifiers.String()
		if len(modStr) > 0 {
			modStr = " + " + modStr
		}
		log("OnKeyDown: " + keyCode.String() + modStr)
		switch keyCode {
		case nui.KeyEsc:
			logItems = nil
		case nui.KeyF1:
			win.MaximizeWindow()
		case nui.KeyF2:
			win.MinimizeWindow()
		}
		win.Update()
	}

	win.OnKeyUp = func(keyCode nui.Key, modifiers nui.KeyModifiers) {
		modStr := modifiers.String()
		if len(modStr) > 0 {
			modStr = " + " + modStr
		}
		log("OnKeyUp: " + keyCode.String() + modStr)
		win.Update()
	}

	win.OnChar = func(char rune) {
		log("OnChar: " + string(char))
		win.Update()
	}

	win.OnPaint = func(rgba *image.RGBA) {
		cnv := nuicanvas.NewCanvas(rgba)
		cnv.SetColor(color.RGBA{0, 255, 0, 255})
		for i, s := range logItems {
			cnv.DrawFixedString(600, float64(10+20*i), s, 2)
		}
	}

	win.Show()
	win.MoveToCenterOfScreen()
	win.EventLoop()
}
