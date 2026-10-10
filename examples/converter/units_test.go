package converter

import (
	"math"
	"testing"
)

// find returns the unit of the category by its English symbol.
func find(t *testing.T, symbol string) unit {
	t.Helper()
	for _, cat := range categories {
		for _, u := range cat.units {
			if u.name(&en.Units).Symbol == symbol {
				return u
			}
		}
	}
	t.Fatalf("no unit %q", symbol)
	return unit{}
}

func TestConvert(t *testing.T) {
	tests := []struct {
		v        float64
		from, to string
		want     float64
	}{
		{100, "°C", "°F", 212},
		{-40, "°F", "°C", -40},
		{0, "°C", "K", 273.15},
		{1, "mi", "m", 1609.344},
		{1, "ft", "in", 12},
		{1, "GiB", "MiB", 1024},
		{1, "B", "bit", 8},
		{1, "ha", "m²", 10000},
		{1, "gal", "l", 3.785411784},
		{1, "lb", "oz", 16},
		{36, "km/h", "m/s", 10},
	}
	for _, tt := range tests {
		got := convert(tt.v, find(t, tt.from), find(t, tt.to))
		if math.Abs(got-tt.want) > 1e-9*math.Max(1, math.Abs(tt.want)) {
			t.Errorf("%v %s = %v %s, want %v", tt.v, tt.from, got, tt.to, tt.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		v        float64
		decimals int
		group    bool
		s        *Strings
		want     string
	}{
		{1609.344, 3, true, &en, "1,609.344"},
		{1609.344, 1, false, &en, "1609.3"},
		{-1234567, 0, true, &en, "-1,234,567"},
		{1234567.5, 2, true, &ru, "1 234 567,50"},
		{-0.0001, 2, true, &en, "0.00"},
		{999, 0, true, &en, "999"},
	}
	for _, tt := range tests {
		if got := formatNumber(tt.v, tt.decimals, tt.group, tt.s); got != tt.want {
			t.Errorf("formatNumber(%v, %d, %v) = %q, want %q", tt.v, tt.decimals, tt.group, got, tt.want)
		}
	}
}
