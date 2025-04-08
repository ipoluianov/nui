package nui

import (
	"image"
	"image/color"

	"github.com/ipoluianov/nui/nuicanvas"
)

type application struct {
	windows map[windowId]*nativeWindow
}

var app *application

func newApp() *application {
	return &application{
		windows: make(map[windowId]*nativeWindow),
	}
}

func init() {
	app = newApp()
}

func makeDefaultIcon() *image.RGBA {
	size := 32
	padding := 1
	rectSize := size/2 - padding*4

	x1 := padding
	y1 := padding
	x2 := padding + rectSize + padding + padding + padding + padding
	y2 := padding + rectSize + padding + padding + padding + padding

	icon := image.NewRGBA(image.Rect(0, 0, size, size))
	nuicanvas := nuicanvas.NewCanvas(icon)
	nuicanvas.SetColor(color.RGBA{0, 128, 255, 255})

	nuicanvas.FillRect(x1, y1, rectSize, rectSize, 1)
	nuicanvas.FillRect(x2, y1, rectSize, rectSize, 1)
	nuicanvas.FillRect(x1, y2, rectSize, rectSize, 1)
	nuicanvas.FillRect(x2, y2, rectSize, rectSize, 1)

	return icon
}

func CreateWindow(title string, width int, height int, center bool) Window {
	w := createWindow(title, width, height, center)
	w.SetAppIcon(makeDefaultIcon())
	return w
}

func CreateDefaultWindow() Window {
	w := CreateWindow("App", 800, 600, true)
	return w
}
