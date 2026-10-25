package regression

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func TestClipboardFormats(t *testing.T) {
	uitest.Show(t, newForm(100, 100))

	img := image.NewNRGBA(image.Rect(0, 0, 4, 3))
	img.Set(1, 2, color.NRGBA{10, 200, 30, 255})
	if err := ui.ClipboardSetImage(img); err != nil {
		t.Fatal(err)
	}
	got, err := ui.ClipboardGetImage()
	if err != nil || got == nil || got.Bounds() != img.Bounds() || got.At(1, 2) != img.At(1, 2) {
		t.Fatalf("image back: %v, %v", got, err)
	}
	if text, _ := ui.ClipboardGetText(); text != "" || ui.ClipboardHasFormat(ui.MIMEText) {
		t.Errorf("an image has a text %q", text)
	}

	ui.ClipboardSet(
		ui.ClipboardItem{MIME: ui.MIMEText, Data: []byte("two shapes")},
		ui.ClipboardItem{MIME: "application/x-shapes", Data: []byte("[1,2]")},
	)
	formats, _ := ui.ClipboardFormats()
	if !slices.Equal(formats, []string{ui.MIMEText, "application/x-shapes"}) {
		t.Errorf("formats %v", formats)
	}
	if text, _ := ui.ClipboardGetText(); text != "two shapes" {
		t.Errorf("text %q", text)
	}
	if data, ok, _ := ui.ClipboardGetData("application/x-shapes"); !ok || string(data) != "[1,2]" {
		t.Errorf("own data %q %v", data, ok)
	}
	if img, _ := ui.ClipboardGetImage(); img != nil {
		t.Error("the image stays after another copy")
	}
}
