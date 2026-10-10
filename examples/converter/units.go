package converter

import (
	"math"
	"strconv"
	"strings"
)

// unit converts linearly to the base unit of its category:
// base = value*factor + offset (an offset only for temperatures).
type unit struct {
	name   func(u *UnitStrings) UnitName
	factor float64
	offset float64
}

func (u unit) toBase(v float64) float64   { return v*u.factor + u.offset }
func (u unit) fromBase(b float64) float64 { return (b - u.offset) / u.factor }

// convert converts the value between two units of a category.
func convert(v float64, from, to unit) float64 {
	return to.fromBase(from.toBase(v))
}

type category struct {
	name     func(c *CategoryStrings) string
	units    []unit
	from, to int // the units chosen at start
}

// categories in the order of the radio buttons; the base units are the
// metre, square metre, litre, kilogram, kelvin, metre per second and byte.
var categories = []category{
	{
		name: func(c *CategoryStrings) string { return c.Length },
		from: 7, to: 3,
		units: []unit{
			{name: func(u *UnitStrings) UnitName { return u.Millimetre }, factor: 0.001},
			{name: func(u *UnitStrings) UnitName { return u.Centimetre }, factor: 0.01},
			{name: func(u *UnitStrings) UnitName { return u.Metre }, factor: 1},
			{name: func(u *UnitStrings) UnitName { return u.Kilometre }, factor: 1000},
			{name: func(u *UnitStrings) UnitName { return u.Inch }, factor: 0.0254},
			{name: func(u *UnitStrings) UnitName { return u.Foot }, factor: 0.3048},
			{name: func(u *UnitStrings) UnitName { return u.Yard }, factor: 0.9144},
			{name: func(u *UnitStrings) UnitName { return u.Mile }, factor: 1609.344},
			{name: func(u *UnitStrings) UnitName { return u.NauticalMile }, factor: 1852},
		},
	},
	{
		name: func(c *CategoryStrings) string { return c.Area },
		from: 5, to: 2,
		units: []unit{
			{name: func(u *UnitStrings) UnitName { return u.SquareCentimetre }, factor: 1e-4},
			{name: func(u *UnitStrings) UnitName { return u.SquareMetre }, factor: 1},
			{name: func(u *UnitStrings) UnitName { return u.Hectare }, factor: 1e4},
			{name: func(u *UnitStrings) UnitName { return u.SquareKilometre }, factor: 1e6},
			{name: func(u *UnitStrings) UnitName { return u.SquareFoot }, factor: 0.09290304},
			{name: func(u *UnitStrings) UnitName { return u.Acre }, factor: 4046.8564224},
			{name: func(u *UnitStrings) UnitName { return u.SquareMile }, factor: 2589988.110336},
		},
	},
	{
		name: func(c *CategoryStrings) string { return c.Volume },
		from: 5, to: 1,
		units: []unit{
			{name: func(u *UnitStrings) UnitName { return u.Millilitre }, factor: 0.001},
			{name: func(u *UnitStrings) UnitName { return u.Litre }, factor: 1},
			{name: func(u *UnitStrings) UnitName { return u.CubicMetre }, factor: 1000},
			{name: func(u *UnitStrings) UnitName { return u.FluidOunce }, factor: 0.0295735295625},
			{name: func(u *UnitStrings) UnitName { return u.Cup }, factor: 0.2365882365},
			{name: func(u *UnitStrings) UnitName { return u.GallonUS }, factor: 3.785411784},
			{name: func(u *UnitStrings) UnitName { return u.GallonUK }, factor: 4.54609},
		},
	},
	{
		name: func(c *CategoryStrings) string { return c.Mass },
		from: 5, to: 2,
		units: []unit{
			{name: func(u *UnitStrings) UnitName { return u.Milligram }, factor: 1e-6},
			{name: func(u *UnitStrings) UnitName { return u.Gram }, factor: 1e-3},
			{name: func(u *UnitStrings) UnitName { return u.Kilogram }, factor: 1},
			{name: func(u *UnitStrings) UnitName { return u.Tonne }, factor: 1000},
			{name: func(u *UnitStrings) UnitName { return u.Ounce }, factor: 0.028349523125},
			{name: func(u *UnitStrings) UnitName { return u.Pound }, factor: 0.45359237},
			{name: func(u *UnitStrings) UnitName { return u.Stone }, factor: 6.35029318},
		},
	},
	{
		name: func(c *CategoryStrings) string { return c.Temperature },
		from: 0, to: 1,
		units: []unit{
			{name: func(u *UnitStrings) UnitName { return u.Celsius }, factor: 1, offset: 273.15},
			{name: func(u *UnitStrings) UnitName { return u.Fahrenheit }, factor: 5.0 / 9, offset: 273.15 - 32*5.0/9},
			{name: func(u *UnitStrings) UnitName { return u.Kelvin }, factor: 1},
		},
	},
	{
		name: func(c *CategoryStrings) string { return c.Speed },
		from: 1, to: 2,
		units: []unit{
			{name: func(u *UnitStrings) UnitName { return u.MetrePerSecond }, factor: 1},
			{name: func(u *UnitStrings) UnitName { return u.KilometrePerHour }, factor: 1 / 3.6},
			{name: func(u *UnitStrings) UnitName { return u.MilePerHour }, factor: 0.44704},
			{name: func(u *UnitStrings) UnitName { return u.Knot }, factor: 1852.0 / 3600},
			{name: func(u *UnitStrings) UnitName { return u.FootPerSecond }, factor: 0.3048},
		},
	},
	{
		name: func(c *CategoryStrings) string { return c.Data },
		from: 8, to: 7,
		units: []unit{
			{name: func(u *UnitStrings) UnitName { return u.Bit }, factor: 0.125},
			{name: func(u *UnitStrings) UnitName { return u.Byte }, factor: 1},
			{name: func(u *UnitStrings) UnitName { return u.Kilobyte }, factor: 1e3},
			{name: func(u *UnitStrings) UnitName { return u.Megabyte }, factor: 1e6},
			{name: func(u *UnitStrings) UnitName { return u.Gigabyte }, factor: 1e9},
			{name: func(u *UnitStrings) UnitName { return u.Terabyte }, factor: 1e12},
			{name: func(u *UnitStrings) UnitName { return u.Kibibyte }, factor: 1 << 10},
			{name: func(u *UnitStrings) UnitName { return u.Mebibyte }, factor: 1 << 20},
			{name: func(u *UnitStrings) UnitName { return u.Gibibyte }, factor: 1 << 30},
			{name: func(u *UnitStrings) UnitName { return u.Tebibyte }, factor: 1 << 40},
		},
	},
}

// unitText is the unit as it is shown in the lists: "Mile (mi)", or
// just "Фут" when the symbol is the word itself.
func unitText(s *Strings, u unit) string {
	n := u.name(&s.Units)
	if strings.EqualFold(n.Name, n.Symbol) {
		return n.Name
	}
	return n.Name + " (" + n.Symbol + ")"
}

// formatNumber writes v with the decimals and the separators of the
// language, grouping the thousands if asked.
func formatNumber(v float64, decimals int, group bool, s *Strings) string {
	if math.Abs(v) < 0.5*math.Pow10(-decimals) {
		v = 0 // no "-0.00"
	}
	text := strconv.FormatFloat(math.Abs(v), 'f', decimals, 64)
	whole, fraction, _ := strings.Cut(text, ".")
	if group {
		var b strings.Builder
		for i, digit := range whole {
			if i > 0 && (len(whole)-i)%3 == 0 {
				b.WriteString(s.ThousandsSeparator)
			}
			b.WriteRune(digit)
		}
		whole = b.String()
	}
	if v < 0 {
		whole = "-" + whole
	}
	if fraction == "" {
		return whole
	}
	return whole + s.DecimalSeparator + fraction
}
