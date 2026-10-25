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

// registerNativeFont hands the file of the face (a faceName) to the system's
// text engine, where there is one that takes files (Windows)
func registerNativeFont(family string, data []byte, f *sfnt.Font) {
	family = strings.ToLower(family)
	fd, err := platforms.RegisterFontData(data)
	systemFontsMu.Lock()
	defer systemFontsMu.Unlock()
	// The faces synthesized from another face of the family may now have
	// their own file
	base, _, _ := strings.Cut(family, ":")
	for key := range nativeFaces {
		if isFaceOf(key.family, base) {
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
// Widget.SetFontFamily or the theme, in every style: the bold and italic
// faces of the system font, or the system's synthesized ones.
// ErrSystemFontNotFound if the system has no such font.
func RegisterSystemFont(family, name string) error {
	family, _ = parseFontFamily(family)
	family = strings.ToLower(family)
	size := ThemeFontSize()
	f, err := platforms.OpenSystemFont(name, size, false, false)
	if errors.Is(err, platforms.ErrSystemTextNotSupported) {
		return registerSystemFontFile(family, name)
	}
	if err != nil {
		return err
	}
	systemFontsMu.Lock()
	systemFamilies[family] = name
	for key := range systemFaces {
		if isFaceOf(key.family, family) {
			delete(systemFaces, key)
		}
	}
	systemFaces[faceKey{family, size}] = f
	systemFontsMu.Unlock()
	clearAllRenderedTexts()
	return nil
}

// isFaceOf tells whether the face name is of the family
func isFaceOf(face, family string) bool {
	return face == family || strings.HasPrefix(face, family+":")
}

// isSystemFamily tells whether the family (lowercase) is drawn with a
// system font
func isSystemFamily(family string) bool {
	systemFontsMu.Lock()
	defer systemFontsMu.Unlock()
	_, ok := systemFamilies[family]
	return ok
}

// forgetFoundSystemFamily drops the system font found for a family nui
// didn't know (see findSystemFamily) once the application registers a font
// file of the family: the file is what it asked for
func forgetFoundSystemFamily(family string) {
	systemLookupMu.Lock()
	defer systemLookupMu.Unlock()
	if !systemLookups[family] {
		return
	}
	delete(systemLookups, family)
	systemFontsMu.Lock()
	delete(systemFamilies, family)
	for key := range systemFaces {
		if isFaceOf(key.family, family) {
			delete(systemFaces, key)
		}
	}
	systemFontsMu.Unlock()
}

// registerSystemFontFile registers the files of the system font's faces,
// drawn by nui's rasterizer (macOS). A face the font has no font of its own
// for is synthesized from the regular one.
func registerSystemFontFile(family, name string) error {
	for _, style := range []FontStyle{FontRegular, FontBold, FontItalic, FontBold | FontItalic} {
		path, postScriptName, err := platforms.SystemFontFile(name, style&FontBold != 0, style&FontItalic != 0)
		if err != nil {
			if style == FontRegular {
				return err
			}
			continue
		}
		f, exact, err := openFontFile(path, postScriptName)
		if style == FontRegular && err != nil {
			return err
		}
		// A face that is an instance of a variable font (the system font
		// SF) isn't a font of its own in the file: nui's rasterizer doesn't
		// apply the variations, so it is synthesized from the regular face
		if style != FontRegular && (err != nil || !exact) {
			continue
		}
		setFont(faceName(family, style), f)
	}
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

// systemFontFor returns the system font that draws text in the family (it
// may name a style) at the size: the system font registered as the family,
// or the file of the family in the system's text engine if it has all the
// characters of text. nil if nui's rasterizer draws the text.
func systemFontFor(family string, size float64, text string) platforms.SystemFont {
	r := resolveFont(family)
	systemFontsMu.Lock()
	defer systemFontsMu.Unlock()
	if name, ok := systemFamilies[r.family]; ok {
		key := faceKey{faceName(r.family, r.style), size}
		if f, ok := systemFaces[key]; ok {
			return f
		}
		f, err := platforms.OpenSystemFont(name, size, r.style&FontBold != 0, r.style&FontItalic != 0)
		if err != nil {
			return nil
		}
		systemFaces[key] = f
		return f
	}

	if !NativeFontRendering {
		return nil
	}
	family = r.family
	n, synth := nativeFontFor(family, r.style)
	if n == nil {
		// An unknown family gets the default font, as in withFace
		if isRegisteredFont(family) {
			return nil
		}
		family = defaultFontKey
		if n, synth = nativeFontFor(family, r.style); n == nil {
			return nil
		}
	}
	if !n.hasAll(text) {
		return nil
	}
	key := faceKey{faceName(family, r.style), size}
	if f, ok := nativeFaces[key]; ok {
		return f
	}
	// The style the family has no file for is synthesized by the system
	fd := n.data
	if synth&FontBold != 0 {
		fd.Weight = 700
	}
	if synth&FontItalic != 0 {
		fd.Italic = true
	}
	f, err := platforms.OpenFontData(fd, size)
	if err != nil {
		return nil
	}
	nativeFaces[key] = f
	return f
}

// nativeFontFor returns the file in the system's text engine of the face of
// the family nearest to the style, and the style to synthesize; must be
// called with systemFontsMu held
func nativeFontFor(family string, style FontStyle) (*nativeFont, FontStyle) {
	for _, s := range styleCandidates(style) {
		if n, ok := nativeFamilies[faceName(family, s)]; ok {
			return n, style &^ s
		}
	}
	return nil, 0
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
