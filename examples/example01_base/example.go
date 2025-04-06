package example01base

import "github.com/ipoluianov/nui/nui"

func Run() {
	wnd := nui.CreateWindow()
	wnd.Show()
	wnd.EventLoop()
}
