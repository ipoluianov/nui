package main

import "github.com/ipoluianov/nui/nui"

func main() {
	nui.Init()
	wnd := nui.CreateWindow()
	wnd.EventLoop()
}
