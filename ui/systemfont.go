package ui

import (
	"errors"
	"strings"
	"sync"

	"golang.org/x/image/font/sfnt"

	"github.com/ipoluianov/nui/internal/platforms"
)

// System fonts: fonts of the operating system, not built into the
// application, drawn by the system's text engine so the text looks like in
// the other applications - with ClearType on Windows, with the desktop's
// FreeType settings (subpixel order, hinting) on Linux. On macOS, which
// draws text in grayscale, the system font's file is drawn by nui itself.
//
//	ui.UseSystemFont(ui.SystemUIFontName()) // e.g. "Segoe UI" on Windows
//
//	ui.RegisterSystemFont("code", "Consolas")
//	logView.SetFontFamily("code")

var (
	systemFontsMu  sync.Mutex
	systemFamilies = map[string]string{} // family -> the system font's name
	systemFaces    = map[faceKey]platforms.SystemFont{}

	nativeFamilies = map[string]*nativeFont{} // family -> its file in the system's text engine
	nativeFaces    = map[faceKey]platforms.SystemFont{}
)

// NativeFontRendering draws the fonts built into nui and those of
// RegisterFont with the system's text engine where it can take a font file:
// on Windows with GDI and ClearType, using the hinting of the file, as the
// system fonts are drawn. Text with characters the font lacks (e.g. Chinese,
// drawn with a fallback font) is drawn by nui's rasterizer. Set it to false
// before the first form is shown to draw everything with nui's rasterizer.
var NativeFontRendering = true

// nativeFont is a font file handed to the system's text engine
type nativeFont struct {
	data   platforms.FontData
	font   *sfnt.Font
	buf    sfnt.Buffer
	covers map[rune]bool
}

// registerNativeFont hands the file of the family to the system's text
// engine, where there is one that takes files (Windows)
func registerNativeFont(family string, data []byte, f *sfnt.Font) {
	family = strings.ToLower(family)
	fd, err := platforms.RegisterFontData(data)
	systemFontsMu.Lock()
	defer systemFontsMu.Unlock()
	for key := range nativeFaces {
		if key.family == family {
			delete(nativeFaces, key)
		}
	}
	if err != nil {
		delete(nativeFamilies, family)
		return
	}
	nativeFamilies[family] = &nativeFont{data: fd, font: f, covers: map[rune]bool{}}
}

// hasAll tells whether the font has all the characters of text; must be
// called with systemFontsMu held
func (n *nativeFont) hasAll(text string) bool {
	for _, r := range text {
		has, ok := n.covers[r]
		if !ok {
			i, err := n.font.GlyphIndex(&n.buf, r)
			has = err == nil && i != 0
			n.covers[r] = has
		}
		if !has {
			return false
		}
	}
	return true
}

// ErrSystemFontNotFound is returned when the system has no font of the name
var ErrSystemFontNotFound = platforms.ErrSystemFontNotFound

// SystemUIFontName returns the name of the font the system uses for its
// interface: "Segoe UI" on Windows, the desktop's sans-serif font on Linux,
// the system font on macOS
func SystemUIFontName() string {
	return platforms.SystemUIFontName()
}

// RegisterSystemFont makes the font family drawn with the system font of
// the name (e.g. "Segoe UI", "Consolas", "DejaVu Sans"), for
// Widget.SetFontFamily or the theme. ErrSystemFontNotFound if the system has
// no such font.
func RegisterSystemFont(family, name string) error {
	family = strings.ToLower(family)
	size := ThemeFontSize()
	f, err := platforms.OpenSystemFont(name, size)
	if errors.Is(err, platforms.ErrSystemTextNotSupported) {
		return registerSystemFontFile(family, name)
	}
	if err != nil {
		return err
	}
	systemFontsMu.Lock()
	systemFamilies[family] = name
	for key := range systemFaces {
		if key.family == family {
			delete(systemFaces, key)
		}
	}
	systemFaces[faceKey{family, size}] = f
	systemFontsMu.Unlock()
	clearAllRenderedTexts()
	return nil
}

// registerSystemFontFile registers the file of the system font, drawn by
// nui's rasterizer (macOS)
func registerSystemFontFile(family, name string) error {
	path, fileFamily, err := platforms.SystemFontFile(name)
	if err != nil {
		return err
	}
	f, err := openSystemFont(path, []string{fileFamily, name})
	if err != nil {
		return err
	}
	setFont(family, f)
	return nil
}

// UseSystemFont draws the whole interface with the system font of the name:
// it becomes the theme font, and the open forms are laid out again for it.
// ErrSystemFontNotFound if the system has no such font.
func UseSystemFont(name string) error {
	if err := RegisterSystemFont(name, name); err != nil {
		return err
	}
	Theme["fontFamily"] = name
	ApplyBaseFontSize(ThemeFontSize())
	return nil
}

// systemFontFor returns the system font that draws text in the family at
// the size: the system font registered as the family, or the file of the
// family in the system's text engine if it has all the characters of text.
// nil if nui's rasterizer draws the text.
func systemFontFor(family string, size float64, text string) platforms.SystemFont {
	family = strings.ToLower(family)
	systemFontsMu.Lock()
	defer systemFontsMu.Unlock()
	if name, ok := systemFamilies[family]; ok {
		key := faceKey{family, size}
		if f, ok := systemFaces[key]; ok {
			return f
		}
		f, err := platforms.OpenSystemFont(name, size)
		if err != nil {
			return nil
		}
		systemFaces[key] = f
		return f
	}

	if !NativeFontRendering {
		return nil
	}
	n, ok := nativeFamilies[family]
	if !ok {
		// An unknown family gets the default font, as in withFace
		if isRegisteredFont(family) {
			return nil
		}
		family = defaultFontKey
		if n, ok = nativeFamilies[family]; !ok {
			return nil
		}
	}
	if !n.hasAll(text) {
		return nil
	}
	key := faceKey{family, size}
	if f, ok := nativeFaces[key]; ok {
		return f
	}
	f, err := platforms.OpenFontData(n.data, size)
	if err != nil {
		return nil
	}
	nativeFaces[key] = f
	return f
}

// nativeMetrics returns the ascent and the descent of the line of the
// family's system font, if it has one: text drawn by nui's rasterizer (with
// characters the font lacks) takes them, so its lines are as high and its
// baseline is where the other text has it
func nativeMetrics(family string, size float64) (ascent, descent int, ok bool) {
	sf := systemFontFor(family, size, "")
	if sf == nil {
		return 0, 0, false
	}
	ascent, descent = sf.Metrics()
	return ascent, descent, true
}

// UseBuiltinFont draws the interface with the font built into nui (Noto
// Sans) again, after UseSystemFont
func UseBuiltinFont() {
	Theme["fontFamily"] = FontFamilySans
	ApplyBaseFontSize(ThemeFontSize())
}
