package platforms

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// macOS draws text in grayscale (it dropped subpixel antialiasing in 10.14),
// so the system fonts are not drawn by CoreText here: SystemFontFile finds
// the font file through CoreText, and nui draws it with its own rasterizer.

var ct struct {
	once sync.Once
	err  error

	cfStringCreateWithCString  func(alloc uintptr, s string, encoding uint32) uintptr
	cfStringGetCString         func(s uintptr, buf *byte, size int64, encoding uint32) bool
	cfURLGetFileSystemRep      func(url uintptr, resolve bool, buf *byte, size int64) bool
	cfRelease                  func(cf uintptr)
	ctFontCreateWithName       func(name uintptr, size float64, matrix uintptr) uintptr
	ctFontCreateUIFont         func(fontType uint32, size float64, language uintptr) uintptr
	ctFontCopyFamilyName       func(font uintptr) uintptr
	ctFontCopyAttribute        func(font, attribute uintptr) uintptr
	ctFontCopyPostScriptName   func(font uintptr) uintptr
	ctFontCreateCopyWithTraits func(font uintptr, size float64, matrix uintptr, traits, mask uint32) uintptr
	ctFontGetSymbolicTraits    func(font uintptr) uint32
	urlAttribute               uintptr
}

const (
	cfStringEncodingUTF8 = 0x08000100
	ctFontUIFontSystem   = 2
	ctFontItalicTrait    = 1 << 0
	ctFontBoldTrait      = 1 << 1
)

func loadCoreText() error {
	ct.once.Do(func() {
		cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			ct.err = err
			return
		}
		text, err := purego.Dlopen("/System/Library/Frameworks/CoreText.framework/CoreText", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			ct.err = err
			return
		}
		purego.RegisterLibFunc(&ct.cfStringCreateWithCString, cf, "CFStringCreateWithCString")
		purego.RegisterLibFunc(&ct.cfStringGetCString, cf, "CFStringGetCString")
		purego.RegisterLibFunc(&ct.cfURLGetFileSystemRep, cf, "CFURLGetFileSystemRepresentation")
		purego.RegisterLibFunc(&ct.cfRelease, cf, "CFRelease")
		purego.RegisterLibFunc(&ct.ctFontCreateWithName, text, "CTFontCreateWithName")
		purego.RegisterLibFunc(&ct.ctFontCreateUIFont, text, "CTFontCreateUIFontForLanguage")
		purego.RegisterLibFunc(&ct.ctFontCopyFamilyName, text, "CTFontCopyFamilyName")
		purego.RegisterLibFunc(&ct.ctFontCopyAttribute, text, "CTFontCopyAttribute")
		purego.RegisterLibFunc(&ct.ctFontCopyPostScriptName, text, "CTFontCopyPostScriptName")
		purego.RegisterLibFunc(&ct.ctFontCreateCopyWithTraits, text, "CTFontCreateCopyWithSymbolicTraits")
		purego.RegisterLibFunc(&ct.ctFontGetSymbolicTraits, text, "CTFontGetSymbolicTraits")
		sym, err := purego.Dlsym(text, "kCTFontURLAttribute")
		if err != nil {
			ct.err = err
			return
		}
		ct.urlAttribute = **(**uintptr)(unsafe.Pointer(&sym))
	})
	return ct.err
}

func cfString(s uintptr) string {
	if s == 0 {
		return ""
	}
	var buf [512]byte
	if !ct.cfStringGetCString(s, &buf[0], int64(len(buf)), cfStringEncodingUTF8) {
		return ""
	}
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return string(buf[:n])
}

// fontFile returns the family and the file of a CTFont
func fontFile(font uintptr) (family, path string) {
	name := ct.ctFontCopyFamilyName(font)
	family = cfString(name)
	if name != 0 {
		ct.cfRelease(name)
	}
	url := ct.ctFontCopyAttribute(font, ct.urlAttribute)
	if url == 0 {
		return family, ""
	}
	defer ct.cfRelease(url)
	var buf [1024]byte
	if !ct.cfURLGetFileSystemRep(url, true, &buf[0], int64(len(buf))) {
		return family, ""
	}
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return family, string(buf[:n])
}

// SystemFontFile returns the file of the face of the system font of the
// family name in the style, and the PostScript name of the face in it (for
// a font collection), on macOS. ErrSystemFontNotFound if the system has no
// such font, or the font has no face of the style.
func SystemFontFile(name string, bold, italic bool) (path, postScriptName string, err error) {
	if err := loadCoreText(); err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrSystemTextNotSupported, err)
	}
	cfName := ct.cfStringCreateWithCString(0, name, cfStringEncodingUTF8)
	defer ct.cfRelease(cfName)
	font := ct.ctFontCreateWithName(cfName, 14, 0)
	if font == 0 {
		return "", "", fmt.Errorf("%w: %q", ErrSystemFontNotFound, name)
	}
	defer ct.cfRelease(font)
	var traits uint32
	if bold {
		traits |= ctFontBoldTrait
	}
	if italic {
		traits |= ctFontItalicTrait
	}
	if traits != 0 {
		const mask = ctFontBoldTrait | ctFontItalicTrait
		styled := ct.ctFontCreateCopyWithTraits(font, 0, 0, traits, mask)
		if styled == 0 {
			return "", "", fmt.Errorf("%w: %q has no such face", ErrSystemFontNotFound, name)
		}
		defer ct.cfRelease(styled)
		if ct.ctFontGetSymbolicTraits(styled)&mask != traits {
			return "", "", fmt.Errorf("%w: %q has no such face", ErrSystemFontNotFound, name)
		}
		font = styled
	}
	family, path := fontFile(font)
	// CoreText gives another font for an unknown name
	if !strings.EqualFold(family, name) || path == "" {
		return "", "", fmt.Errorf("%w: %q", ErrSystemFontNotFound, name)
	}
	ps := ct.ctFontCopyPostScriptName(font)
	postScriptName = cfString(ps)
	if ps != 0 {
		ct.cfRelease(ps)
	}
	return path, postScriptName, nil
}

func openSystemFont(name string, pixelSize float64, bold, italic bool) (SystemFont, error) {
	return nil, ErrSystemTextNotSupported
}

func systemUIFontName() string {
	if loadCoreText() != nil {
		return "Helvetica Neue"
	}
	font := ct.ctFontCreateUIFont(ctFontUIFontSystem, 14, 0)
	if font == 0 {
		return "Helvetica Neue"
	}
	defer ct.cfRelease(font)
	family, _ := fontFile(font)
	if family == "" {
		return "Helvetica Neue"
	}
	return family
}

// systemSubpixelOrder: macOS draws text in grayscale
func systemSubpixelOrder() int { return 0 }
