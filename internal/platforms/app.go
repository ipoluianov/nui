package platforms

import (
	"image"
	"image/color"
	"sync"

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

func CreateWindow(title string, posX int, posY int, width int, height int, center bool, maximized bool) Window {
	w := createWindow(title, posX, posY, width, height, center, maximized)
	w.SetAppIcon(makeDefaultIcon())
	return w
}

func CreateDefaultWindow() Window {
	w := CreateWindow("App", 100, 100, 800, 600, true, false)
	return w
}

// Run shows each window and blocks until all of them have been closed.
// Additional windows created later (e.g. from a callback) don't need Run at
// all: Show() (or ShowModal()) already makes a window live entirely on its
// own, hiding whatever goroutine it needs internally - call Exec() yourself
// afterward only if you specifically need to block until that window closes.
//
// The first window's Show()/Exec() run on the calling goroutine rather than
// a spawned one: on Cocoa the real event loop must start on the process's
// original thread, so callers must never wrap their first Run() call in
// `go` themselves.
func Run(windows ...Window) {
	if len(windows) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, w := range windows[1:] {
		wg.Add(1)
		go func(w Window) {
			defer wg.Done()
			w.Show()
			w.Exec()
		}(w)
	}
	windows[0].Show()
	windows[0].Exec()
	wg.Wait()
}
