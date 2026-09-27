package ui

import (
	"image/color"
	"testing"
)

func TestColorPickerConversions(t *testing.T) {
	for _, col := range []color.RGBA{
		{0, 0, 0, 255}, {255, 255, 255, 255}, {255, 0, 0, 255}, {0, 255, 0, 255},
		{0, 0, 255, 255}, {0x1E, 0x88, 0xE5, 255}, {0x80, 0x80, 0x80, 255}, {12, 200, 99, 255},
	} {
		h, s, v := rgbToHSV(col.R, col.G, col.B)
		r, g, b := hsvToRGB(h, s, v)
		if r != col.R || g != col.G || b != col.B {
			t.Errorf("%v -> hsv(%.1f %.2f %.2f) -> %d %d %d", col, h, s, v, r, g, b)
		}
	}

	cases := map[string]color.RGBA{
		"#1E88E5":   {0x1E, 0x88, 0xE5, 0xFF},
		"1e88e5":    {0x1E, 0x88, 0xE5, 0xFF},
		"#f00":      {0xFF, 0, 0, 0xFF},
		"#11223380": {0x11, 0x22, 0x33, 0x80},
		" #ABCDEF ": {0xAB, 0xCD, 0xEF, 0xFF},
	}
	for s, want := range cases {
		if got, ok := ParseHexColor(s); !ok || got != want {
			t.Errorf("ParseHexColor(%q) = %v, %v; want %v", s, got, ok, want)
		}
	}
	for _, bad := range []string{"", "#12", "#12345", "#GGGGGG", "#1234567"} {
		if _, ok := ParseHexColor(bad); ok {
			t.Errorf("ParseHexColor(%q) accepted", bad)
		}
	}
}

func newTestColorPicker(t *testing.T) (*ColorPicker, *Form, *[]color.RGBA) {
	t.Helper()
	form := NewForm()
	form.processResize(400, 300)
	picker := NewColorPicker()
	form.Panel().AddWidget(0, 0, picker)
	picker.SetColor(color.RGBA{0xFF, 0, 0, 0xFF})
	changes := &[]color.RGBA{}
	picker.SetOnColorChanged(func(col color.RGBA) { *changes = append(*changes, col) })
	return picker, form, changes
}

func TestColorPickerDragAndCancel(t *testing.T) {
	picker, form, changes := newTestColorPicker(t)
	picker.Focus()
	typeKey(form, KeyEnter, 0, KeyModifiers{})
	popup := picker.popup
	if popup == nil || form.TopPopupWidget() != Widgeter(popup) {
		t.Fatal("Enter didn't open the panel")
	}

	// Drag to the bottom-left corner of the square: black
	sx, sy, _, sh := popup.svRect()
	form.mouseLeftButtonPressed = true
	popup.mouseDown(MouseButtonLeft, sx+5, sy+5, KeyModifiers{})
	popup.mouseMove(sx-50, sy+sh+50, KeyModifiers{}) // past the edge: clamped
	form.mouseLeftButtonPressed = false
	if picker.Color() != (color.RGBA{0, 0, 0, 0xFF}) {
		t.Errorf("after the drag: %v", picker.Color())
	}
	if popup.hexBox.Text() != "#000000" {
		t.Errorf("hex = %q", popup.hexBox.Text())
	}
	// The hue survives black: the top right corner is red again
	popup.dragArea = cpAreaSV
	popup.dragTo(sx+popup.contentWidth(), sy)
	if picker.Color() != (color.RGBA{0xFF, 0, 0, 0xFF}) {
		t.Errorf("top right of the square = %v, want red", picker.Color())
	}

	// Escape returns to the color the panel was opened with
	popup.dragTo(sx, sy+sh)
	typeKey(form, KeyEsc, 0, KeyModifiers{})
	if picker.IsPopupOpen() || picker.Color() != (color.RGBA{0xFF, 0, 0, 0xFF}) {
		t.Errorf("after Escape: open %v, color %v", picker.IsPopupOpen(), picker.Color())
	}
	if len(*changes) == 0 || (*changes)[len(*changes)-1] != (color.RGBA{0xFF, 0, 0, 0xFF}) {
		t.Errorf("changes = %v", *changes)
	}
}

func TestColorPickerHexPaletteRecent(t *testing.T) {
	picker, form, _ := newTestColorPicker(t)
	picker.Focus()
	typeKey(form, KeySpace, 0, KeyModifiers{})
	popup := picker.popup
	if form.FocusedWidget() != Widgeter(popup.hexBox) {
		t.Fatal("the hex field isn't focused")
	}
	// Type a code (the whole text is replaced) and press Enter
	popup.hexBox.SetText("")
	for _, ch := range "#00ff80" {
		typeKey(form, KeyA, ch, KeyModifiers{})
	}
	if picker.Color() != (color.RGBA{0, 0xFF, 0x80, 0xFF}) {
		t.Fatalf("typed code: %v (text %q)", picker.Color(), popup.hexBox.Text())
	}
	typeKey(form, KeyEnter, 0, KeyModifiers{})
	if picker.IsPopupOpen() {
		t.Fatal("Enter didn't close the panel")
	}
	if recent := RecentColors(); len(recent) == 0 || recent[0] != (color.RGBA{0, 0xFF, 0x80, 0xFF}) {
		t.Errorf("recent = %v", recent)
	}

	// A palette swatch chooses its color and closes the panel
	picker.OpenPopup()
	popup = picker.popup
	paletteY, _ := popup.swatchesTop()
	size := popup.swatchSize()
	popup.mouseDown(MouseButtonLeft, cpPad+size+cpSwatchGap+2, paletteY+2, KeyModifiers{}) // the second color
	if picker.IsPopupOpen() || picker.Color() != DefaultColorPickerPalette[1] {
		t.Errorf("swatch: open %v, color %v", picker.IsPopupOpen(), picker.Color())
	}
	// The recent colors are offered in the next panel
	picker.OpenPopup()
	_, recentY := picker.popup.swatchesTop()
	if col, ok := picker.popup.swatchAt(cpPad+2, recentY+2); !ok || col != DefaultColorPickerPalette[1] {
		t.Errorf("first recent swatch = %v, %v", col, ok)
	}
	picker.ClosePopup()
}

func TestColorPickerAlpha(t *testing.T) {
	picker, _, _ := newTestColorPicker(t)
	picker.SetColor(color.RGBA{1, 2, 3, 0x40})
	if picker.Color().A != 0xFF {
		t.Error("alpha kept without SetAlphaEnabled")
	}
	picker.SetAlphaEnabled(true)
	picker.SetColor(color.NRGBA{10, 20, 30, 0x80})
	if picker.Color() != (color.RGBA{10, 20, 30, 0x80}) {
		t.Errorf("NRGBA converted to %v", picker.Color())
	}
	picker.OpenPopup()
	popup := picker.popup
	if popup.hexBox.Text() != "#0A141E80" {
		t.Errorf("hex with alpha = %q", popup.hexBox.Text())
	}
	ax, ay, _, _ := popup.alphaRect()
	popup.mouseDown(MouseButtonLeft, ax, ay+2, KeyModifiers{})
	if picker.Color().A != 0 {
		t.Errorf("alpha after a click at the left end = %d", picker.Color().A)
	}
	picker.ClosePopup()
}
