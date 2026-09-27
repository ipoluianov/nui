package ui

import (
	"fmt"
	"image"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Font registry. Each font file is parsed once, when registered. Faces (a
// font at a size) are cached; a face isn't safe for concurrent use and forms
// paint on their own goroutines, so every use of a face holds its mutex (see
// withFace).
//
// A face falls back to other fonts for the characters its own font doesn't
// have (see AddFallbackFont), e.g. to a Chinese font for Chinese text.

var (
	fontsMu        sync.RWMutex
	fonts          = make(map[string]*sfnt.Font)
	fallbackFonts  []string
	faceCache      = make(map[faceKey]*faceEntry)
	defaultFontKey = FontFamilySans
)

type faceKey struct {
	family string
	size   float64
}

type faceEntry struct {
	mu   sync.Mutex
	face *fallbackFace
}

// RegisterFont adds a font family from a TrueType or OpenType file. The
// family name is case-insensitive; registering it again replaces the font.
func RegisterFont(family string, data []byte) error {
	f, err := opentype.Parse(data)
	if err != nil {
		return fmt.Errorf("font %q: %w", family, err)
	}
	setFont(family, f)
	return nil
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

func setFont(family string, f *sfnt.Font) {
	fontsMu.Lock()
	fonts[strings.ToLower(family)] = f
	// Faces of any family may use this font as a fallback
	faceCache = make(map[faceKey]*faceEntry)
	fontsMu.Unlock()
	clearAllRenderedTexts()
}

// withFace calls fn with the face of the family at the size, holding the
// face's lock. An unknown family gets the default font.
func withFace(family string, size float64, fn func(face font.Face)) error {
	entry, err := getFaceEntry(strings.ToLower(family), size)
	if err != nil {
		return err
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	fn(entry.face)
	return nil
}

func getFaceEntry(family string, size float64) (*faceEntry, error) {
	key := faceKey{family, size}

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

	primary, ok := fonts[family]
	if !ok {
		primary, ok = fonts[defaultFontKey]
		if !ok {
			return nil, fmt.Errorf("font %q is not registered", family)
		}
	}

	chain := []*sfnt.Font{primary}
	for _, name := range fallbackFonts {
		if f, ok := fonts[name]; ok && f != primary {
			chain = append(chain, f)
		}
	}

	faces := make([]font.Face, 0, len(chain))
	for _, f := range chain {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			return nil, err
		}
		faces = append(faces, face)
	}

	entry = &faceEntry{face: newFallbackFace(faces)}
	faceCache[key] = entry
	return entry, nil
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
