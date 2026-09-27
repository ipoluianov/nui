package ui

import (
	"fmt"
	"image/color"
	"sync"
)

// ThemePalette is the set of colors every widget is drawn with. Each color
// has a role, like QPalette in Qt: widgets ask for "the background of an
// input field" or "the border of a control", not for a particular gray, so
// the light and the dark themes are just two palettes.
//
// Hover and pressed states aren't in the palette: they are mixed from the
// background and the text color (see hoverColor), which works in both themes.
type ThemePalette struct {
	// Window is the background of forms and panels, WindowText the text on it
	Window     color.RGBA
	WindowText color.RGBA

	// Base is the background of input fields, lists and tables, Text the
	// text in them
	Base color.RGBA
	Text color.RGBA

	// Button is the background of buttons, combo boxes and table headers
	Button     color.RGBA
	ButtonText color.RGBA

	// Border frames the controls; Divider is a softer line inside them:
	// table grid, menu separators
	Border  color.RGBA
	Divider color.RGBA

	// Highlight is the accent: focus, checked boxes, chosen items, primary
	// buttons; HighlightedText is the text on it
	Highlight       color.RGBA
	HighlightedText color.RGBA

	// Selection is the background of selected text, softer than Highlight
	// so the text keeps its own color
	Selection color.RGBA

	DisabledText    color.RGBA
	PlaceholderText color.RGBA

	// PopupBase is the background of menus and dropdowns
	PopupBase color.RGBA

	ToolTipBase color.RGBA
	ToolTipText color.RGBA

	Link    color.RGBA
	Error   color.RGBA
	Warning color.RGBA
	Success color.RGBA
}

// LightPalette is the light theme: flat, in the spirit of Qt's Fusion style.
var LightPalette = ThemePalette{
	Window:          ColorFromHex("#EFEFEF"),
	WindowText:      ColorFromHex("#1F1F1F"),
	Base:            ColorFromHex("#FFFFFF"),
	Text:            ColorFromHex("#1F1F1F"),
	Button:          ColorFromHex("#FAFAFA"),
	ButtonText:      ColorFromHex("#1F1F1F"),
	Border:          ColorFromHex("#B5B5B5"),
	Divider:         ColorFromHex("#DCDCDC"),
	Highlight:       ColorFromHex("#2A7DC1"),
	HighlightedText: ColorFromHex("#FFFFFF"),
	Selection:       ColorFromHex("#BFD9EE"),
	DisabledText:    ColorFromHex("#A0A0A0"),
	PlaceholderText: ColorFromHex("#8C8C8C"),
	PopupBase:       ColorFromHex("#FFFFFF"),
	ToolTipBase:     ColorFromHex("#FFFFFF"),
	ToolTipText:     ColorFromHex("#1F1F1F"),
	Link:            ColorFromHex("#1D6FB5"),
	Error:           ColorFromHex("#C62828"),
	Warning:         ColorFromHex("#B26A00"),
	Success:         ColorFromHex("#2E7D32"),
}

// DarkPalette is the dark theme: the same structure as the light one, with
// depth shown by the lightness of the surfaces instead of shadows. Inputs
// are darker than the window, buttons and popups lighter.
var DarkPalette = ThemePalette{
	Window:          ColorFromHex("#2B2B2B"),
	WindowText:      ColorFromHex("#E3E3E3"),
	Base:            ColorFromHex("#1F1F1F"),
	Text:            ColorFromHex("#E3E3E3"),
	Button:          ColorFromHex("#383838"),
	ButtonText:      ColorFromHex("#E3E3E3"),
	Border:          ColorFromHex("#505050"),
	Divider:         ColorFromHex("#3A3A3A"),
	Highlight:       ColorFromHex("#3584C8"),
	HighlightedText: ColorFromHex("#FFFFFF"),
	Selection:       ColorFromHex("#264F78"),
	DisabledText:    ColorFromHex("#7A7A7A"),
	PlaceholderText: ColorFromHex("#8A8A8A"),
	PopupBase:       ColorFromHex("#333333"),
	ToolTipBase:     ColorFromHex("#3A3A3A"),
	ToolTipText:     ColorFromHex("#E3E3E3"),
	Link:            ColorFromHex("#5AA9E6"),
	Error:           ColorFromHex("#EF5350"),
	Warning:         ColorFromHex("#FFB74D"),
	Success:         ColorFromHex("#66BB6A"),
}

// Corner radius of buttons, input fields and other controls
const themeControlRadius = 3

// Distance from the left edge of an input field or a combo box to its text
const themeTextInset = 8

var Theme map[string]interface{}

const DefaultFontSize = 14.0

var IsDarkTheme = true

var currentPalette ThemePalette

// CurrentPalette returns the colors of the current theme.
func CurrentPalette() *ThemePalette {
	return &currentPalette
}

func ApplyDarkTheme() {
	ApplyPalette(DarkPalette, true)
}

func ApplyLightTheme() {
	ApplyPalette(LightPalette, false)
}

// ApplyPalette switches to the palette, e.g. a copy of LightPalette with
// another Highlight, and repaints the open forms. dark tells whether it's a
// dark theme, for the window title bars and for the charts.
func ApplyPalette(palette ThemePalette, dark bool) {
	currentPalette = palette
	IsDarkTheme = dark

	// Kept for code that reads the colors from the Theme map
	Theme["background.surface"] = palette.Window
	Theme["background.primary"] = palette.Highlight
	Theme["background.secondary"] = palette.Highlight
	Theme["foreground.surface"] = palette.WindowText
	Theme["foreground.primary"] = palette.HighlightedText
	Theme["foreground.secondary"] = palette.HighlightedText

	for _, form := range openForms() {
		form.Invoke(form.applyTheme)
	}
}

// ApplyBaseFontSize sets the size of the theme font and lays out the open
// forms again: the heights of controls follow the font (see ThemeControlHeight).
func ApplyBaseFontSize(fontSize float64) {
	Theme["fontSize"] = fontSize
	DefaultUiLineHeight = ThemeControlHeight()
	for _, form := range openForms() {
		form.Invoke(form.applyFontSize)
	}
}

func ColorFromHex(hexStr string) color.RGBA {
	var r, g, b, a uint8
	a = 255
	if len(hexStr) == 7 {
		_, err := fmt.Sscanf(hexStr, "#%02x%02x%02x", &r, &g, &b)
		if err != nil {
			return color.RGBA{0, 0, 0, 255}
		}
	} else if len(hexStr) == 9 {
		_, err := fmt.Sscanf(hexStr, "#%02x%02x%02x%02x", &r, &g, &b, &a)
		if err != nil {
			return color.RGBA{0, 0, 0, 255}
		}
	} else {
		return color.RGBA{0, 0, 0, 255}
	}
	return color.RGBA{r, g, b, a}
}

func ColorToHex(col color.Color) string {
	r, g, b, a := col.RGBA()
	return fmt.Sprintf("#%02X%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
}

// MixColors returns the color t of the way from a to b: 0 is a, 1 is b.
func MixColors(a, b color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	mix := func(x, y uint8) uint8 {
		return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5)
	}
	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), mix(a.A, b.A)}
}

// colorToRGBA converts any color to an opaque color.RGBA.
func colorToRGBA(col color.Color) color.RGBA {
	if rgba, ok := col.(color.RGBA); ok {
		return rgba
	}
	r, g, b, _ := col.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
}

// hoverColor and pressedColor shade a background towards the text color
// drawn on it: darker in the light theme, lighter in the dark one.
func hoverColor(background, text color.Color) color.RGBA {
	return MixColors(colorToRGBA(background), colorToRGBA(text), 0.06)
}

func pressedColor(background, text color.Color) color.RGBA {
	return MixColors(colorToRGBA(background), colorToRGBA(text), 0.13)
}

// withAlpha returns the color with the alpha, for Canvas.FillRect and
// DrawLine, which blend straight (not premultiplied) colors.
func withAlpha(col color.RGBA, a uint8) color.RGBA {
	col.A = a
	return col
}

func init() {
	Theme = make(map[string]interface{})

	ApplyDarkTheme()

	Theme["fontFamily"] = FontFamilySans
	Theme["fontSize"] = DefaultFontSize
	DefaultUiLineHeight = ThemeControlHeight()
}

// ThemeBackgroundColorDarkTheme is ThemeBackgroundColor for the dark palette.
func ThemeBackgroundColorDarkTheme(elevation int, role string) color.RGBA {
	return themeBackground(&DarkPalette, elevation, role)
}

// ThemeBackgroundColorLightTheme is ThemeBackgroundColor for the light palette.
func ThemeBackgroundColorLightTheme(elevation int, role string) color.RGBA {
	return themeBackground(&LightPalette, elevation, role)
}

// ThemeBackgroundColor returns the background of a widget with the role and
// the elevation (summed over its parents, see Widget.SetElevation).
//
// Roles: "" or "surface" - the window; "base" - input fields and lists;
// "popup" - menus and dropdowns; "primary" and "secondary" - the accent.
// A positive elevation shades the surface towards the text color, a
// negative one sinks it towards the input field background.
func ThemeBackgroundColor(elevation int, role string) color.RGBA {
	return themeBackground(&currentPalette, elevation, role)
}

func themeBackground(p *ThemePalette, elevation int, role string) color.RGBA {
	switch role {
	case "primary", "secondary":
		return MixColors(p.Highlight, p.HighlightedText, float64(elevation)*0.03)
	case "base":
		return shadeByElevation(p.Base, p.Text, p.Window, elevation)
	case "popup":
		return shadeByElevation(p.PopupBase, p.Text, p.Window, elevation)
	}
	return shadeByElevation(p.Window, p.WindowText, p.Base, elevation)
}

// shadeByElevation moves background towards text for a positive elevation
// and towards sunken (fully at -3) for a negative one.
func shadeByElevation(background, text, sunken color.RGBA, elevation int) color.RGBA {
	if elevation < 0 {
		return MixColors(background, sunken, float64(-elevation)/3)
	}
	return MixColors(background, text, min(float64(elevation)*0.025, 0.3))
}

func ThemeForegroundColor(role string) color.RGBA {
	p := &currentPalette
	switch role {
	case "primary", "secondary":
		return p.HighlightedText
	case "base", "popup":
		return p.Text
	}
	return p.WindowText
}

func ThemeForegroundColorDisabled() color.RGBA {
	return currentPalette.DisabledText
}

// Registry of the forms with a window, so a theme change reaches them all

var (
	openFormsMtx sync.Mutex
	openFormsSet = make(map[*Form]struct{})
)

func registerOpenForm(form *Form) {
	openFormsMtx.Lock()
	openFormsSet[form] = struct{}{}
	openFormsMtx.Unlock()
}

func unregisterOpenForm(form *Form) {
	openFormsMtx.Lock()
	delete(openFormsSet, form)
	openFormsMtx.Unlock()
}

func openForms() []*Form {
	openFormsMtx.Lock()
	defer openFormsMtx.Unlock()
	forms := make([]*Form, 0, len(openFormsSet))
	for form := range openFormsSet {
		forms = append(forms, form)
	}
	return forms
}

func ThemeFontFamily() string {
	if v, ok := Theme["fontFamily"]; ok {
		if fontFamily, ok := v.(string); ok {
			return fontFamily
		}
	}
	return FontFamilySans
}

func ThemeFontSize() float64 {
	if v, ok := Theme["fontSize"]; ok {
		if fontSize, ok := v.(float64); ok {
			return fontSize
		}
	}
	return DefaultFontSize
}

func GetThemeColor(name string, defaultColor color.RGBA) color.RGBA {
	if v, ok := Theme[name]; ok {
		if colorValue, ok := v.(color.RGBA); ok {
			return colorValue
		}
	}
	return defaultColor
}

func GetThemeString(name string, defaultValue string) string {
	if v, ok := Theme[name]; ok {
		if strValue, ok := v.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

func GetThemeInt(name string, defaultValue int) int {
	if v, ok := Theme[name]; ok {
		if intValue, ok := v.(int); ok {
			return intValue
		}
	}
	return defaultValue
}

func GetThemeFloat(name string, defaultValue float64) float64 {
	if v, ok := Theme[name]; ok {
		if floatValue, ok := v.(float64); ok {
			return floatValue
		}
	}
	return defaultValue
}
