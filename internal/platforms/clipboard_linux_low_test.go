//go:build linux

package platforms

import (
	"image"
	"image/color"
	"testing"
	"time"
	"unsafe"
)

// The X11 tests here are opt-in like needX says, on a throwaway Xvfb, each
// in its own run (they share the connection of the clipboard):
// NUI_TEST_X11=1 DISPLAY=:98 go test -run TestClipboardX11LowIncrLatin1 ./internal/platforms

// A Latin-1 text (STRING) sent in parts came back unconverted: the type
// looked at was INCR, that of the header, not STRING, that of the parts
func TestClipboardX11LowIncrLatin1(t *testing.T) {
	needX(t)
	if _, _, err := clipboardWindow(); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	defer close(stop)
	go incrOwnerOf(t, "STRING", func(i int) ([]byte, time.Duration, bool) {
		return []byte{'c', 'a', 'f', 0xE9}, 0, i == 0
	}, stop)
	time.Sleep(300 * time.Millisecond)
	text, err := clipboardGetText()
	if err != nil || text != "café" {
		t.Errorf("text %q, err %v; want \"café\"", text, err)
	}
}

// The data copied here was the caller's slice, served later to other
// applications: reusing the buffer after copying changed what they pasted
func TestClipboardX11LowCopiesData(t *testing.T) {
	needX(t)
	data := []byte("first")
	if err := clipboardSet([]ClipboardItem{{"application/x-nui-test", data}}); err != nil {
		t.Fatal(err)
	}
	copy(data, "XXXXX")
	got, ok, err := clipboardGet("application/x-nui-test")
	if err != nil || !ok || string(got) != "first" {
		t.Fatalf("after the buffer changed: %q, %v, %v; want \"first\"", got, ok, err)
	}
	got[0] = 'Z'
	if again, _, _ := clipboardGet("application/x-nui-test"); string(again) != "first" {
		t.Errorf("after the result changed: %q; want \"first\"", again)
	}
	if served := clip.items[xInternAtom(xDisplay, "application/x-nui-test", xFalse)]; string(served) != "first" {
		t.Errorf("served to the others: %q; want \"first\"", served)
	}
}

// An icon of 2048x2048 was written in one request over the limit of the
// server: refused (BadLength, only logged), the window had no icon. It is
// scaled down to fit now.
func TestSetAppIconX11LowLarge(t *testing.T) {
	needX(t)
	w := createWindow("icon", 0, 0, 200, 200, false, false)
	defer w.Close()
	icon := image.NewRGBA(image.Rect(0, 0, 2048, 2048))
	for i := range icon.Pix {
		icon.Pix[i] = 0xff
	}
	w.SetAppIcon(icon)
	d := w.platform.display
	xSync(d, xFalse)

	var typ uintptr
	var format int32
	var n, after uintptr
	var prop unsafe.Pointer
	atom := xInternAtom(d, "_NET_WM_ICON", xFalse)
	if xGetWindowProperty(d, w.platform.window, atom, 0, 1<<24, xFalse, xAnyPropertyType, unsafe.Pointer(&typ),
		unsafe.Pointer(&format), unsafe.Pointer(&n), unsafe.Pointer(&after), unsafe.Pointer(&prop)) != xSuccess || prop == nil {
		t.Fatal("no _NET_WM_ICON")
	}
	defer xFree(prop)
	if format != 32 || n < 3 {
		t.Fatalf("_NET_WM_ICON: format %d, %d items", format, n)
	}
	// Xlib gives the 32-bit items as longs
	words := unsafe.Slice((*uintptr)(prop), int(n))
	iw, ih := int(words[0]), int(words[1])
	if iw < 1000 || iw != ih || int(n) != 2+iw*ih {
		t.Fatalf("_NET_WM_ICON: %dx%d in %d items", iw, ih, n)
	}
	if px := uint32(words[2+iw*ih/2]); px != 0xffffffff {
		t.Errorf("a pixel %#x, want white", px)
	}
}

func TestIconSizeToFitLow(t *testing.T) {
	for _, c := range []struct{ w, h, max, ww, wh int }{
		{256, 256, 1 << 20, 256, 256},
		{2048, 2048, 4194303 - 258, 2047, 2047}, // the usual limit of BIG-REQUESTS
		{2048, 1024, 1 << 20, 1448, 724},
		{100000, 1, 10, 10, 1},
		{1, 100000, 10, 1, 10},
		{0, 0, 10, 0, 0},
	} {
		w, h := iconSizeToFit(c.w, c.h, c.max)
		if w != c.ww || h != c.wh || w*h > max(c.max, c.w*c.h) {
			t.Errorf("iconSizeToFit(%d, %d, %d) = %d, %d; want %d, %d", c.w, c.h, c.max, w, h, c.ww, c.wh)
		}
	}
	icon := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := range 20 {
		for x := range 40 {
			icon.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}
	got := fitIcon(icon, 200)
	if b := got.Bounds(); b.Dx() != 20 || b.Dy() != 10 {
		t.Fatalf("fitIcon: %v", b)
	}
	if c := got.RGBAAt(10, 5); c != (color.RGBA{0, 0, 255, 255}) {
		t.Errorf("fitIcon: pixel %v", c)
	}
	if fitIcon(icon, 800) != icon {
		t.Error("fitIcon scaled an icon that fits")
	}
}
