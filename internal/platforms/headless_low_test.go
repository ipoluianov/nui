package platforms

import (
	"math"
	"slices"
	"testing"
)

// useHeadless switches the platform to the headless mode for the test
func useHeadless(t *testing.T) {
	t.Helper()
	was := headless.Load()
	headless.Store(true)
	headlessMu.Lock()
	saved := headlessClipboard
	headlessMu.Unlock()
	t.Cleanup(func() {
		headless.Store(was)
		headlessMu.Lock()
		headlessClipboard = saved
		headlessMu.Unlock()
	})
}

// The clipboard kept the caller's slice: changing the buffer after copying
// changed what was pasted, and changing what was pasted changed the
// clipboard
func TestHeadlessClipboardCopiesData(t *testing.T) {
	useHeadless(t)
	data := []byte("first")
	if err := ClipboardSet([]ClipboardItem{{"application/x-test", data}}); err != nil {
		t.Fatal(err)
	}
	copy(data, "XXXXX")
	got, ok, _ := ClipboardGet("application/x-test")
	if !ok || string(got) != "first" {
		t.Fatalf("after the buffer changed: %q, %v; want \"first\"", got, ok)
	}
	got[0] = 'Z'
	if again, _, _ := ClipboardGet("application/x-test"); string(again) != "first" {
		t.Errorf("after the result changed: %q; want \"first\"", again)
	}
}

// "text/plain" is the text, as on X11, macOS and Windows; the headless
// clipboard compared the MIME exactly, and ClipboardGetText found nothing
func TestHeadlessClipboardTextMIMEVariants(t *testing.T) {
	useHeadless(t)
	for _, mime := range []string{"text/plain", "text/plain;charset=UTF-8", MIMEText} {
		if err := ClipboardSet([]ClipboardItem{{mime, []byte("hello")}}); err != nil {
			t.Fatal(err)
		}
		if text, _ := ClipboardGetText(); text != "hello" {
			t.Errorf("set as %q: ClipboardGetText = %q", mime, text)
		}
		if data, ok, _ := ClipboardGet("text/plain"); !ok || string(data) != "hello" {
			t.Errorf("set as %q: ClipboardGet(text/plain) = %q, %v", mime, data, ok)
		}
		if formats, _ := ClipboardFormats(); !slices.Equal(formats, []string{MIMEText}) {
			t.Errorf("set as %q: formats %q", mime, formats)
		}
	}
}

// max(NaN, 1) is NaN and max(+Inf, 1) is +Inf: either scale made Render
// panic (image: NewRGBA Rectangle has huge or negative dimensions)
func TestHeadlessInvalidScale(t *testing.T) {
	headlessMu.Lock()
	saved, savedW, savedH := headlessScale, headlessScreenW, headlessScreenH
	headlessMu.Unlock()
	defer func() {
		headlessMu.Lock()
		headlessScale, headlessScreenW, headlessScreenH = saved, savedW, savedH
		headlessMu.Unlock()
	}()
	for _, s := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, -2, 1e300} {
		w := &HeadlessWindow{width: 10, height: 10, scale: 1}
		w.MoveToScreen(s)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("MoveToScreen(%v): Render panics: %v", s, r)
				}
			}()
			w.Render()
		}()
		if !(w.Scale() >= 1 && w.Scale() <= 16) {
			t.Errorf("MoveToScreen(%v): scale %v", s, w.Scale())
		}
		SetHeadlessScreen(100, 100, s)
		headlessMu.Lock()
		got := headlessScale
		headlessMu.Unlock()
		if !(got >= 1 && got <= 16) {
			t.Errorf("SetHeadlessScreen(%v): scale %v", s, got)
		}
	}
	w := &HeadlessWindow{width: 10, height: 10, scale: 1}
	if w.MoveToScreen(2); w.Scale() != 2 {
		t.Errorf("MoveToScreen(2): scale %v", w.Scale())
	}
}
