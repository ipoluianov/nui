package example01base

import "github.com/ipoluianov/nui/nui"

func Run() {
	nui.Init()
	wnd := nui.CreateWindow()
	wnd.Show()
	wnd.EventLoop()
}
