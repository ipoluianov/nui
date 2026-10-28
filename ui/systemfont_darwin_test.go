package ui

import "testing"

// Helvetica Neue maps its characters only in a cmap subtable of Unicode
// 1.1, which sfnt rejects unless read as Unicode BMP (see
// cmapEncodingPatches); its bold is a real face of the collection
func TestHelveticaNeue(t *testing.T) {
	if err := registerSystemFontFile("test-helvetica-neue", "Helvetica Neue"); err != nil {
		t.Fatal(err)
	}
	const sample = "Fox jumps over the lazy dog"
	regular, _, _ := MeasureText("test-helvetica-neue", 15, sample)
	bold, _, _ := MeasureText("test-helvetica-neue:bold", 15, sample)
	sans, _, _ := MeasureText(FontFamilySans, 15, sample)
	if regular == sans {
		t.Errorf("Helvetica Neue as wide as Noto Sans (%d): not the system font", regular)
	}
	if bold <= regular {
		t.Errorf("bold %d, regular %d", bold, regular)
	}
}
