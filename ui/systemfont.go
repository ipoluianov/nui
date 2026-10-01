package ui

import (
	"errors"
	"strings"
	"sync"

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
)

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

// systemFontFor returns the system font of the family at the size, nil if the
// family isn't a system font
func systemFontFor(family string, size float64) platforms.SystemFont {
	family = strings.ToLower(family)
	systemFontsMu.Lock()
	defer systemFontsMu.Unlock()
	name, ok := systemFamilies[family]
	if !ok {
		return nil
	}
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

// UseBuiltinFont draws the interface with the font built into nui (Noto
// Sans) again, after UseSystemFont
func UseBuiltinFont() {
	Theme["fontFamily"] = FontFamilySans
	ApplyBaseFontSize(ThemeFontSize())
}
