// clipcheck checks the clipboard: text, an image and the data of the
// application, between two copies of the program and with other
// applications; see doc/platform-checks.md
//
//	go run ./cmd/clipcheck -copy   # the first copy: copies, keeps the window
//	go run ./cmd/clipcheck -paste  # the second copy: reads it at the start
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/ipoluianov/nui/ui"
)

const (
	customMIME = "application/x-cliptest"
	sampleText = "clipcheck: Съешь ещё 中文 😀"
	customData = "custom data \x00\x01\x02 of clipcheck"
)

var logView *ui.LogView

// logf prints to the console (for the report) and to the window
func logf(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	fmt.Println(line)
	ui.Invoke(func() { logView.AppendLine(line) })
}

// sampleImage is a picture that shows swapped colors and a flipped image:
// red at the top left, green at the top right, blue at the bottom
func sampleImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 120, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 120; x++ {
			c := color.RGBA{0, 0, 0xff, 0xff}
			if y < 40 {
				c = color.RGBA{0xff, 0, 0, 0xff}
				if x >= 60 {
					c = color.RGBA{0, 0xff, 0, 0xff}
				}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// checkSample tells whether img is sampleImage as it was copied
func checkSample(img image.Image) string {
	b := img.Bounds()
	if b.Dx() != 120 || b.Dy() != 80 {
		return "not the image of clipcheck"
	}
	near := func(x, y int, want color.RGBA) bool {
		got := color.RGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.RGBA)
		d := func(a, b uint8) bool { return int(a)-int(b) >= -40 && int(a)-int(b) <= 40 }
		return d(got.R, want.R) && d(got.G, want.G) && d(got.B, want.B)
	}
	switch {
	case near(10, 10, color.RGBA{0xff, 0, 0, 0}) && near(110, 10, color.RGBA{0, 0xff, 0, 0}) && near(10, 70, color.RGBA{0, 0, 0xff, 0}):
		return "OK: the colors and the orientation are right"
	case near(10, 70, color.RGBA{0xff, 0, 0, 0}):
		return "BAD: flipped vertically"
	case near(10, 10, color.RGBA{0, 0, 0xff, 0}):
		return "BAD: red and blue swapped"
	}
	at := func(x, y int) color.RGBA {
		return color.RGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.RGBA)
	}
	return fmt.Sprintf("BAD: the colors are wrong: top left %v, top right %v, bottom %v", at(10, 10), at(110, 10), at(10, 70))
}

func copyAll() {
	var buf bytes.Buffer
	if err := png.Encode(&buf, sampleImage()); err != nil {
		logf("copy: %v", err)
		return
	}
	err := ui.ClipboardSet(
		ui.ClipboardItem{MIME: ui.MIMEText, Data: []byte(sampleText)},
		ui.ClipboardItem{MIME: ui.MIMEPNG, Data: buf.Bytes()},
		ui.ClipboardItem{MIME: customMIME, Data: []byte(customData)},
	)
	if err != nil {
		logf("copy: %v", err)
		return
	}
	logf("copied: text, image 120x80 (red, green / blue), %s", customMIME)
}

func copyImage() {
	if err := ui.ClipboardSetImage(sampleImage()); err != nil {
		logf("copy image: %v", err)
		return
	}
	logf("copied the image only (ClipboardSetImage)")
}

func paste(preview *ui.ImageBox) {
	logf("--- paste")
	formats, err := ui.ClipboardFormats()
	if err != nil {
		logf("formats: %v", err)
	} else {
		logf("formats (%d): %s", len(formats), strings.Join(formats, ", "))
	}

	text, err := ui.ClipboardGetText()
	switch {
	case err != nil:
		logf("text: %v", err)
	case text == sampleText:
		logf("text: OK, the text of clipcheck")
	default:
		logf("text: %q", shorten(text))
	}

	img, err := ui.ClipboardGetImage()
	switch {
	case err != nil:
		logf("image: %v", err)
		preview.SetImage(nil)
	case img == nil:
		logf("image: none")
		preview.SetImage(nil)
	default:
		logf("image: %dx%d, %s", img.Bounds().Dx(), img.Bounds().Dy(), checkSample(img))
		preview.SetImage(img)
	}

	data, ok, err := ui.ClipboardGetData(customMIME)
	switch {
	case err != nil:
		logf("%s: %v", customMIME, err)
	case !ok:
		logf("%s: none", customMIME)
	case string(data) == customData:
		logf("%s: OK, %d bytes", customMIME, len(data))
	default:
		logf("%s: BAD, %q", customMIME, shorten(string(data)))
	}
}

func shorten(s string) string {
	if r := []rune(s); len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return s
}

func main() {
	copyAtStart := flag.Bool("copy", false, "copy the text, the image and the data at the start")
	pasteAtStart := flag.Bool("paste", false, "read the clipboard at the start")
	flag.Parse()

	form := ui.NewForm()
	form.SetTitle("nui clipboard check")
	form.SetSize(800, 600)
	p := form.Panel()

	buttons := p.AddPanel(0, 0)
	buttons.SetPanelPadding(0)
	preview := ui.NewImageBox()
	preview.SetScaling(ui.ImageBoxScaleNoScaleAdjustBox)
	buttons.AddButton(0, 0, "Copy all", copyAll)
	buttons.AddButton(0, 1, "Copy image only", copyImage)
	buttons.AddButton(0, 2, "Copy text only", func() {
		ui.ClipboardSetText(sampleText)
		logf("copied the text only (ClipboardSetText)")
	})
	buttons.AddButton(0, 3, "Paste", func() { paste(preview) })
	buttons.AddHSpacer(0, 4)

	p.AddLabel(1, 0, "The image in the clipboard:")
	p.AddWidget(2, 0, preview)
	logView = ui.NewLogView()
	p.AddWidget(3, 0, logView)

	form.Show()
	if *copyAtStart {
		copyAll()
	}
	if *pasteAtStart {
		ui.Invoke(func() { paste(preview) })
	}
	form.Exec()
}
