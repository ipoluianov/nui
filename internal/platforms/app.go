package platforms

import (
	"image"
	"image/color"

	"github.com/ipoluianov/nui/internal/canvas"
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
	cnv := canvas.NewCanvas(icon)
	cnv.SetColor(color.RGBA{0, 128, 255, 255})

	cnv.FillRect(x1, y1, rectSize, rectSize, 1)
	cnv.FillRect(x2, y1, rectSize, rectSize, 1)
	cnv.FillRect(x1, y2, rectSize, rectSize, 1)
	cnv.FillRect(x2, y2, rectSize, rectSize, 1)

	return icon
}

// CreateWindow creates a hidden window. Windows live on the UI thread: called
// from another goroutine, it creates the window there and waits for it (see
// RunOnUIThread).
func CreateWindow(title string, posX int, posY int, width int, height int, center bool, maximized bool) Window {
	var w *nativeWindow
	RunOnUIThread(func() {
		w = createWindow(title, posX, posY, width, height, center, maximized)
		w.SetAppIcon(makeDefaultIcon())
	})
	return w
}

func CreateDefaultWindow() Window {
	w := CreateWindow("App", 100, 100, 800, 600, true, false)
	return w
}

// Run shows the windows and runs the event loop until all of them are
// closed. Call it from the main goroutine: the event loop of every window -
// these ones and the ones opened later, e.g. from a callback - runs on the UI
// thread (the main OS thread), and so does every window callback.
//
// Windows opened later need no Run of their own: Show() is enough, the
// running event loop serves them too.
func Run(windows ...Window) {
	for _, w := range windows {
		w.Show()
	}
	for _, w := range windows {
		w.Exec()
	}
}
