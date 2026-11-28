package regression

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"runtime"
	"slices"
	"strings"
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

// pngWithSize is a valid little PNG whose header claims the size: the
// decoder allocates the image from the header before it reads the pixels
func pngWithSize(width, height uint32) []byte {
	chunk := func(typ string, data []byte) []byte {
		b := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
		b = append(b, typ...)
		b = append(b, data...)
		return binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(append([]byte(typ), data...)))
	}
	ihdr := binary.BigEndian.AppendUint32(nil, width)
	ihdr = binary.BigEndian.AppendUint32(ihdr, height)
	ihdr = append(ihdr, 8, 6, 0, 0, 0) // 8 bits, RGBA
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	zw.Write(make([]byte, 100))
	zw.Close()
	p := []byte("\x89PNG\r\n\x1a\n")
	p = append(p, chunk("IHDR", ihdr)...)
	p = append(p, chunk("IDAT", z.Bytes())...)
	return append(p, chunk("IEND", nil)...)
}

// An image another application put on the clipboard with a header that
// claims a huge size is refused with an error, without allocating it: an
// 89-byte PNG of 1000000x1000000 used to kill the process on Paste
// (a fatal out of memory, which recover can't catch).
func TestClipboardImageHugeHeader(t *testing.T) {
	uitest.Show(t, newForm(100, 100))
	for _, size := range [][2]uint32{{1000000, 1000000}, {30000, 30000}, {40000, 1}, {1, 40000}} {
		ui.ClipboardSetData(ui.MIMEPNG, pngWithSize(size[0], size[1]))
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		img, err := ui.ClipboardGetImage()
		runtime.ReadMemStats(&after)
		if err == nil || img != nil {
			t.Errorf("%dx%d: image %v, error %v; want an error", size[0], size[1], img, err)
		}
		if grown := after.TotalAlloc - before.TotalAlloc; grown > 16<<20 {
			t.Errorf("%dx%d: %d MB allocated", size[0], size[1], grown>>20)
		}
	}
	// An image within the limits is still decoded (its pixels are missing,
	// so the decoder fails, but after reading the header)
	ui.ClipboardSetData(ui.MIMEPNG, pngWithSize(100, 100))
	if _, err := ui.ClipboardGetImage(); err == nil || strings.Contains(err.Error(), "too large") {
		t.Errorf("100x100: %v, want a decoding error", err)
	}
}
