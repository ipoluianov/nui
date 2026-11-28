package ui

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"os"
	"image/png"
	"math"
	"runtime"
	"testing"
)

// The clipboard keeps the caller's slice: changing the buffer after the copy
// changes what is pasted (X11 and headless; Windows copies)
func TestAuditClipboardAliasing(t *testing.T) {
	data := []byte("first")
	ClipboardSetData("application/x-audit", data)
	copy(data, "XXXXX")
	got, _, _ := ClipboardGetData("application/x-audit")
	t.Logf("pasted %q after the caller reused its buffer", got)
	if string(got) != "first" {
		t.Errorf("clipboard aliases the caller's buffer: %q", got)
	}
	got[0] = 'Z'
	again, _, _ := ClipboardGetData("application/x-audit")
	t.Logf("after modifying the returned slice: %q", again)

	// "text/plain" is a text MIME (isTextMIME) but the headless clipboard
	// does not see it as the text
	ClipboardSetData("text/plain", []byte("hello"))
	text, _ := ClipboardGetText()
	t.Logf("headless ClipboardGetText after Set(text/plain): %q", text)
	if text != "hello" {
		t.Errorf("headless text MIME aliases not unified: %q", text)
	}
}

// A 70-byte PNG claiming 1000000x1000000 RGBA: png.Decode allocates the
// whole image before reading the pixels. Run under `ulimit -v`.
func TestAuditClipboardImageHugeHeader(t *testing.T) {
	if os.Getenv("NUI_AUDIT_BOMB") == "" {
		t.Skip("set NUI_AUDIT_BOMB=1 (and ulimit -v) to run")
	}
	chunk := func(typ string, data []byte) []byte {
		b := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
		b = append(b, typ...)
		b = append(b, data...)
		return binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(append([]byte(typ), data...)))
	}
	ihdr := binary.BigEndian.AppendUint32(nil, 1000000)
	ihdr = binary.BigEndian.AppendUint32(ihdr, 1000000)
	ihdr = append(ihdr, 8, 6, 0, 0, 0)
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	zw.Write(make([]byte, 100))
	zw.Close()
	p := []byte("\x89PNG\r\n\x1a\n")
	p = append(p, chunk("IHDR", ihdr)...)
	p = append(p, chunk("IDAT", z.Bytes())...)
	p = append(p, chunk("IEND", nil)...)
	if err := ClipboardSetData(MIMEPNG, p); err != nil {
		t.Fatal(err)
	}
	t.Logf("clipboard payload %d bytes; pasting", len(p))
	_, err := ClipboardGetImage()
	t.Logf("survived: err=%v", err)
}

// ClipboardGetImage decodes whatever PNG another application put on the
// clipboard with no limit on its dimensions: a few KB allocate width*height
func TestAuditClipboardImageBomb(t *testing.T) {
	var buf bytes.Buffer
	png.Encode(&buf, image.NewGray(image.Rect(0, 0, 8000, 8000)))
	bomb := buf.Bytes()
	if err := ClipboardSetData(MIMEPNG, bomb); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	img, err := ClipboardGetImage()
	runtime.ReadMemStats(&after)
	t.Logf("clipboard payload %d bytes -> decoded %v, allocated %d MB, err=%v",
		len(bomb), img.Bounds(), (after.TotalAlloc-before.TotalAlloc)>>20, err)
	cfg, _ := png.DecodeConfig(bytes.NewReader(bomb))
	t.Logf("no size check: DecodeConfig %dx%d is accepted as is", cfg.Width, cfg.Height)
}

// AT-SPI's Properties.Set(Value, CurrentValue) passes any double of the
// D-Bus client to AccessibleNode.SetCurrent: NaN is not clamped
func TestAuditAccessibleSetCurrentNaN(t *testing.T) {
	s := NewSlider()
	newTestForm(t, s)
	s.SetRange(0, 10)
	s.SetValue(5)
	accessibleNodeOf(s).SetCurrent(math.NaN())
	t.Logf("slider value after SetCurrent(NaN): %v", s.Value())
	if math.IsNaN(s.Value()) {
		t.Errorf("slider value became NaN")
	}

	sb := NewScrollBar(true)
	newTestForm(t, sb)
	accessibleNodeOf(sb).SetCurrent(math.NaN())
	t.Logf("scrollbar value after SetCurrent(NaN): %v (min %d max %d)", sb.value, sb.min, sb.max)
	if sb.value < sb.min || sb.value > sb.max {
		t.Errorf("scrollbar value out of range: %d", sb.value)
	}

	nb := NewNumBox()
	newTestForm(t, nb)
	accessibleNodeOf(nb).SetCurrent(math.NaN())
	t.Logf("numbox value after SetCurrent(NaN): %v text %q", nb.Value(), nb.Text())
	if math.IsNaN(nb.Value()) {
		t.Errorf("numbox value became NaN")
	}
}
