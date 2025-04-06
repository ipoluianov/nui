package example00demo

import "github.com/ipoluianov/nui/nui"

func Run() {
	win := nui.CreateWindow()
	win.Show()
	win.MoveToCenterOfScreen()
	win.EventLoop()
}
