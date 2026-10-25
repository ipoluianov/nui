//go:build linux

package platforms

import "testing"

func TestParseDesktopScale(t *testing.T) {
	tests := []struct {
		nui, resources, gdk string
		want                float64
	}{
		{"", "", "", 1},
		{"", "Xft.antialias:\t1\nXft.dpi:\t192\n", "", 2},
		{"", "Xft.dpi: 120", "", 1.25},
		{"", "Xft.dpi: 144", "2", 1.5}, // KDE sets both, Xft.dpi is the whole scale
		{"", "Xft.dpi: 96", "2", 2},
		{"", "Xft.dpi: 72", "", 1},
		{"1.5", "Xft.dpi: 192", "2", 1.5},
		{"0.5", "", "", 1},
		{"9", "", "", 4},
		{"abc", "", "", 1},
		{"", "Xft.dpi: 97", "", 1}, // 1.0104 is 1 in steps of 1/8
	}
	for _, tt := range tests {
		if got := parseDesktopScale(tt.nui, tt.resources, tt.gdk); got != tt.want {
			t.Errorf("parseDesktopScale(%q, %q, %q) = %v, want %v", tt.nui, tt.resources, tt.gdk, got, tt.want)
		}
	}
}
