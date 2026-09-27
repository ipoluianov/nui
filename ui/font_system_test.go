package ui

import (
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
)

// Chinese text is measured with the system CJK font, not as "missing"
// boxes of the embedded font
func TestSystemCJKFont(t *testing.T) {
	width, _, err := MeasureText(FontFamilySans, 14, "中文")
	if err != nil {
		t.Fatal(err)
	}

	fontsMu.RLock()
	_, found := fonts[FontFamilyCJK]
	fontsMu.RUnlock()
	if !found {
		t.Skip("no system CJK font")
	}

	var cjkAdvance, boxAdvance int
	withFace(FontFamilySans, 14, func(face font.Face) {
		fb := face.(*fallbackFace)
		a, _ := fb.faces[len(fb.faces)-1].GlyphAdvance('中')
		cjkAdvance = a.Round()
		b, _ := fb.faces[0].GlyphAdvance('中')
		boxAdvance = b.Round()
	})
	if width != cjkAdvance*2 {
		t.Errorf("width of 中文 %d, CJK advance %d (the missing glyph box is %d)", width, cjkAdvance, boxAdvance)
	}

	// Latin text keeps the metrics of the main font
	_, h1, _ := MeasureText(FontFamilySans, 14, "Ag")
	_, h2, _ := MeasureText(FontFamilySans, 14, "中文")
	if h1 != h2 {
		t.Errorf("line height %d for Latin, %d for Chinese", h1, h2)
	}
}

// From the Noto CJK collection the simplified Chinese font is taken
func TestOpenSystemFontPicksFamily(t *testing.T) {
	const path = "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"
	f, err := openSystemFont(path, []string{"Noto Sans CJK SC"})
	if err != nil {
		t.Skip(err)
	}
	var buf sfnt.Buffer
	if name := fontName(f, &buf, sfnt.NameIDTypographicFamily, sfnt.NameIDFamily); name != "Noto Sans CJK SC" {
		t.Errorf("picked %q", name)
	}
}

func TestHasCJK(t *testing.T) {
	for text, want := range map[string]bool{"": false, "Hello, мир": false, "中": true, "日本語": true, "한국어": true, "，": true} {
		if got := hasCJK(text); got != want {
			t.Errorf("hasCJK(%q) = %v", text, got)
		}
	}
}

// The language picks the variant of the system CJK font
func TestCJKFontVariantFollowsLanguage(t *testing.T) {
	t.Cleanup(func() { setCJKVariant("sc") })
	var buf sfnt.Buffer
	for variant, family := range map[string]string{"jp": "Noto Sans CJK JP", "tc": "Noto Sans CJK TC", "sc": "Noto Sans CJK SC"} {
		setCJKVariant(variant)
		MeasureText(FontFamilySans, 14, "中文")
		fontsMu.RLock()
		f, ok := fonts[FontFamilyCJK]
		fontsMu.RUnlock()
		if !ok {
			t.Skip("no system CJK font")
		}
		name := fontName(f, &buf, sfnt.NameIDTypographicFamily, sfnt.NameIDFamily)
		if name != family && name[:min(len(name), 13)] == "Noto Sans CJK" {
			t.Errorf("%s: %q, want %q", variant, name, family)
		}
	}
}
