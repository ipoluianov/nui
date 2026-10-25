package ui

import (
	"fmt"
	"image"
	"log"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Font registry. Each font file is parsed once, when registered. Faces (a
// font at a size) are cached; a face isn't safe for concurrent use and text
// may be measured off the UI thread too (e.g. while a form is built), so every
// use of a face holds its mutex (see withFace).
//
// A family has up to four faces: regular, bold, italic and bold italic (see
// FontStyle). A face the family lacks is synthesized from the nearest one.
//
// A face falls back to other fonts for the characters its own font doesn't
// have (see AddFallbackFont), e.g. to a Chinese font for Chinese text.

var (
	fontsMu        sync.RWMutex
	fonts          = make(map[string]*sfnt.Font) // by faceName
	fallbackFonts  []string
	faceCache      = make(map[faceKey]*faceEntry)
	defaultFontKey = FontFamilySans
)

// faceKey is a face at a size; family is a faceName, with the style
type faceKey struct {
	family string
	size   float64
}

type faceEntry struct {
	mu   sync.Mutex
	face *fallbackFace
}

// RegisterFont adds a font family, or a face of it, from a TrueType or
// OpenType file: RegisterFont("inter", regular), then
// RegisterFont("inter:bold", bold) and so on (see FontStyle). The family name
// is case-insensitive; registering a face again replaces the font.
func RegisterFont(family string, data []byte) error {
	f, err := opentype.Parse(data)
	if err != nil {
		return fmt.Errorf("font %q: %w", family, err)
	}
	base, style := parseFontFamily(family)
	name := faceName(strings.ToLower(base), style)
	forgetFoundSystemFamily(strings.ToLower(base))
	setFont(name, f)
	registerNativeFont(name, data, f)
	return nil
}

// isRegisteredFont tells whether a font file is registered for a face of
// the family (lowercase, without a style)
func isRegisteredFont(family string) bool {
	fontsMu.RLock()
	defer fontsMu.RUnlock()
	for _, style := range styleCandidates(FontBold | FontItalic) {
		if _, ok := fonts[faceName(family, style)]; ok {
			return true
		}
	}
	return false
}

// AddFallbackFont adds the registered family to the end of the fallback
// chain: the fonts tried, in order, for characters the requested font
// doesn't have.
func AddFallbackFont(family string) {
	family = strings.ToLower(family)
	fontsMu.Lock()
	for _, f := range fallbackFonts {
		if f == family {
			fontsMu.Unlock()
			return
		}
	}
	fallbackFonts = append(fallbackFonts, family)
	faceCache = make(map[faceKey]*faceEntry)
	fontsMu.Unlock()
	clearAllRenderedTexts()
}

func setFont(name string, f *sfnt.Font) {
	fontsMu.Lock()
	fonts[name] = f
	// Faces of any family may use this font as a fallback
	faceCache = make(map[faceKey]*faceEntry)
	fontsMu.Unlock()
	clearAllRenderedTexts()
}

// lookupFont returns the font of the face nearest to the style the family
// has, and the style to synthesize on it. Must be called with fontsMu held.
func lookupFont(family string, style FontStyle) (*sfnt.Font, FontStyle, bool) {
	for _, s := range styleCandidates(style) {
		if f, ok := fonts[faceName(family, s)]; ok {
			return f, style &^ s, true
		}
	}
	// A family registered only with styled faces: no style is taken away
	for _, s := range styleCandidates(FontBold | FontItalic) {
		if f, ok := fonts[faceName(family, s)]; ok {
			return f, style &^ s, true
		}
	}
	return nil, 0, false
}

// withFace calls fn with the face of the family (it may name a style) at
// the size, holding the face's lock. An unknown family gets the default font.
func withFace(family string, size float64, fn func(face font.Face)) error {
	f := resolveFont(family)
	entry, err := getFaceEntry(f.family, f.style, size)
	if err != nil {
		return err
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	fn(entry.face)
	return nil
}

func getFaceEntry(family string, style FontStyle, size float64) (*faceEntry, error) {
	key := faceKey{faceName(family, style), size}

	fontsMu.RLock()
	entry, ok := faceCache[key]
	fontsMu.RUnlock()
	if ok {
		return entry, nil
	}

	fontsMu.Lock()
	defer fontsMu.Unlock()
	if entry, ok := faceCache[key]; ok {
		return entry, nil
	}

	primary, synth, ok := lookupFont(family, style)
	if !ok {
		primary, synth, ok = lookupFont(defaultFontKey, style)
		if !ok {
			return nil, fmt.Errorf("font %q is not registered", family)
		}
	}

	type styledFont struct {
		font  *sfnt.Font
		synth FontStyle
	}
	chain := []styledFont{{primary, synth}}
	for _, name := range fallbackFonts {
		if f, synth, ok := lookupFont(name, style); ok && f != primary {
			chain = append(chain, styledFont{f, synth})
		}
	}

	faces := make([]font.Face, 0, len(chain))
	for _, f := range chain {
		face, err := opentype.NewFace(f.font, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			return nil, err
		}
		faces = append(faces, newSyntheticFace(face, f.synth, size))
	}

	entry = &faceEntry{face: newFallbackFace(faces)}
	faceCache[key] = entry
	return entry, nil
}

// resolvedFont is a font family string taken apart: the family in
// lowercase and the style
type resolvedFont struct {
	family string
	style  FontStyle
}

// resolvedFonts caches resolveFont: text is measured and drawn often, with
// a few family strings
var resolvedFonts sync.Map // string -> resolvedFont

// resolveFont takes the family string apart and makes sure its faces are
// there: loads the built-in faces of the family the first time they are
// needed, and looks for a family nui doesn't know among the system fonts.
func resolveFont(s string) resolvedFont {
	if f, ok := resolvedFonts.Load(s); ok {
		return f.(resolvedFont)
	}
	base, style := parseFontFamily(s)
	f := resolvedFont{strings.ToLower(base), style}
	loadBuiltinFaces(f.family, f.style)
	if f.family != "" && !isRegisteredFont(f.family) && !isSystemFamily(f.family) {
		findSystemFamily(base)
	}
	resolvedFonts.Store(s, f)
	return f
}

// The families looked for among the system fonts, found or not
var (
	systemLookupMu sync.Mutex
	systemLookups  = map[string]bool{}
)

// findSystemFamily registers the system font of the name as the family of
// that name, if the system has one: a widget may name any font of the
// system, as in other toolkits. Otherwise the family gets the default font.
func findSystemFamily(name string) {
	family := strings.ToLower(name)
	systemLookupMu.Lock()
	defer systemLookupMu.Unlock()
	if _, ok := systemLookups[family]; ok || !SystemFallbackFonts {
		return
	}
	err := RegisterSystemFont(name, name)
	systemLookups[family] = err == nil
	if err != nil {
		log.Printf("nui: no font %q, neither registered nor of the system: the default font is used", name)
	}
}

// fallbackFace draws each character with the first face that has it; a
// character none has is drawn by the first face (as its "missing" glyph).
type fallbackFace struct {
	faces   []font.Face
	metrics font.Metrics
}

// newFallbackFace takes the line metrics from the first face only: the
// layout doesn't change when a fallback font appears. CJK fonts have larger
// ascents, but the extra is line spacing; the ideographs themselves fit
// within the line of Latin fonts.
func newFallbackFace(faces []font.Face) *fallbackFace {
	return &fallbackFace{faces: faces, metrics: faces[0].Metrics()}
}

func (f *fallbackFace) faceFor(r rune) font.Face {
	for _, face := range f.faces {
		if _, ok := face.GlyphAdvance(r); ok {
			return face
		}
	}
	return f.faces[0]
}

func (f *fallbackFace) Close() error {
	return nil
}

func (f *fallbackFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	return f.faceFor(r).Glyph(dot, r)
}

func (f *fallbackFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	return f.faceFor(r).GlyphBounds(r)
}

func (f *fallbackFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	return f.faceFor(r).GlyphAdvance(r)
}

// Kern only applies within one font
func (f *fallbackFace) Kern(r0, r1 rune) fixed.Int26_6 {
	face := f.faceFor(r0)
	if face != f.faceFor(r1) {
		return 0
	}
	return face.Kern(r0, r1)
}

func (f *fallbackFace) Metrics() font.Metrics {
	return f.metrics
}
