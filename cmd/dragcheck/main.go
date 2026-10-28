// dragcheck checks dragging files out of the window (to Finder, Explorer,
// a file manager, a terminal, another copy of dragcheck) and into it; see
// doc/platform-checks.md
//
//	go run ./cmd/dragcheck
package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipoluianov/nui/ui"
)

var logView *ui.LogView

// logf prints to the console (for the report) and to the window
func logf(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	fmt.Println(line)
	ui.Invoke(func() { logView.AppendLine(line) })
}

// makeFiles writes the two files that are dragged, one with a space in the
// name
func makeFiles() ([]string, error) {
	dir, err := os.MkdirTemp("", "dragcheck-")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, name := range []string{"dragcheck.txt", "drag check ä.txt"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("dragcheck: "+name+"\n"), 0o644); err != nil {
			return nil, err
		}
		files = append(files, path)
	}
	return files, nil
}

// area is a frame with a caption: the place to drag from or to drop on
func area(text string, background color.Color) *ui.Frame {
	f := ui.NewFrame()
	f.SetBackgroundColor(background)
	f.SetMinHeight(140)
	f.AddVSpacer(0, 0)
	l := f.AddLabel(1, 0, text)
	l.SetTextAlign(ui.HAlignCenter)
	l.SetForegroundColor(color.RGBA{0x20, 0x20, 0x20, 0xff}) // readable in the dark theme too
	f.AddVSpacer(2, 0)
	return f
}

func main() {
	files, err := makeFiles()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	form := ui.NewForm()
	form.SetTitle("nui drag check")
	form.SetSize(700, 560)
	p := form.Panel()

	drags := 0
	source := area("Drag the two files from here out of the window", color.RGBA{0xc8, 0xe6, 0xc9, 0xff})
	source.SetDragSource(func(x, y int) *ui.DragData {
		drags++
		logf("--- drag %d started: %s", drags, strings.Join(files, ", "))
		return &ui.DragData{
			Text:  "2 files",
			Files: files,
			DroppedOutside: func(effect ui.DragEffect) {
				logf("DroppedOutside: %s", effectName(effect))
				for _, f := range files {
					if _, err := os.Stat(f); err != nil {
						logf("  %s is gone (moved)", f)
					}
				}
			},
		}
	})
	p.AddWidget(0, 0, source)

	target := area("Drop files here (from Finder, Explorer or another dragcheck)", color.RGBA{0xbb, 0xde, 0xfb, 0xff})
	target.SetDropTarget(nil, func(data *ui.DragData, x, y int) {
		if len(data.Files) == 0 {
			logf("dropped without files: %q", data.Text)
			return
		}
		logf("dropped %d files at %d, %d:", len(data.Files), x, y)
		for _, f := range data.Files {
			logf("  %s", f)
		}
	})
	p.AddWidget(1, 0, target)

	// The button shows that the window takes clicks after a drag ended
	// outside (dropped nowhere, Esc): it must not stay pressed
	clicks := 0
	p.AddButton(2, 0, "Click me after a drag", func() {
		clicks++
		logf("click %d", clicks)
	})

	logView = ui.NewLogView()
	p.AddWidget(3, 0, logView)
	logf("the files: %s", filepath.Dir(files[0]))

	form.Show()
	form.Exec()
}

func effectName(effect ui.DragEffect) string {
	switch effect {
	case ui.DragEffectCopy:
		return "Copy"
	case ui.DragEffectMove:
		return "Move"
	case ui.DragEffectLink:
		return "Link"
	case ui.DragEffectNone:
		return "None"
	}
	return fmt.Sprintf("effect %d", effect)
}
