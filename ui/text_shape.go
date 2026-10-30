package ui

import (
	"bytes"
	"os"
	"sync"
	"unicode"

	"github.com/go-text/typesetting/harfbuzz"

	gtfont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/font/sfnt"
)

// The fonts of the shaper (go-text/typesetting, a port of HarfBuzz): the
// same files as the fonts of nui, opened again by it when text is first
// laid out in them. Shaping turns the characters into the glyphs of the
// font and their places: ligatures, kerning, marks, Arabic joined, Indic
// letters reordered.

// fontSource is where the file of a font is: its data, or the path of the
// file and the index of the font in it
type fontSource struct {
	data  []byte
	path  string
	index int
}

var (
	// fontSources is the source of each font of nui
	fontSources sync.Map // *sfnt.Font -> fontSource
	// shapeFaces are the fonts opened for the shaper
	shapeFaces sync.Map // *sfnt.Font -> *gtfont.Face (nil if it could not be opened)
	// shapeMu is held while the shaper or the faces are used: they are not
	// safe for concurrent use, and text is laid out once and kept
	shapeMu   sync.Mutex
	shaper    shaping.HarfbuzzShaper
	segmenter shaping.Segmenter
)

// setFontSource records where the font comes from
func setFontSource(f *sfnt.Font, src fontSource) {
	if f != nil {
		fontSources.Store(f, src)
	}
}

// shapeFace returns the face of the shaper for the font, nil if its file
// cannot be read by it. Called with shapeMu held.
func shapeFace(f *sfnt.Font) *gtfont.Face {
	if v, ok := shapeFaces.Load(f); ok {
		face, _ := v.(*gtfont.Face)
		return face
	}
	var face *gtfont.Face
	if v, ok := fontSources.Load(f); ok {
		src := v.(fontSource)
		var res gtfont.Resource
		if src.data != nil {
			res = bytes.NewReader(src.data)
		} else if file, err := os.Open(src.path); err == nil {
			// Kept open while the font is in use: the application's lifetime
			res = file
		}
		if res != nil {
			if faces, err := gtfont.ParseTTC(res); err == nil && src.index < len(faces) {
				face = faces[src.index]
			}
		}
	}
	shapeFaces.Store(f, face)
	return face
}

// shapeFont is a font of a family for the shaper, with the styles to
// synthesize on it
type shapeFont struct {
	face  *gtfont.Face
	synth FontStyle
}

// fontChain is a family's fonts for the shaper: its own and the fallbacks
type fontChain struct {
	fonts []shapeFont
}

// ResolveFace returns the first face of the chain that has the character,
// the first face if none does (it draws its "missing" glyph)
func (c *fontChain) ResolveFace(r rune) *gtfont.Face {
	// An emoji in color, if a font of color emoji has it: some fonts of
	// text have black ones (a flag in letters in boxes)
	if isEmojiRune(r) {
		for _, f := range c.fonts {
			if _, ok := f.face.NominalGlyph(r); ok && isColorFace(f.face) {
				return f.face
			}
		}
	}
	for _, f := range c.fonts {
		if _, ok := f.face.NominalGlyph(r); ok {
			return f.face
		}
	}
	return c.fonts[0].face
}

// textFace returns the first face of the chain that has the character and
// is not of color emoji: the face of a space between emoji
func (c *fontChain) textFace(r rune) *gtfont.Face {
	for _, f := range c.fonts {
		if _, ok := f.face.NominalGlyph(r); ok && !isColorFace(f.face) {
			return f.face
		}
	}
	return c.ResolveFace(r)
}

// synthOf returns the styles to synthesize on the face of the chain
func (c *fontChain) synthOf(face *gtfont.Face) FontStyle {
	for _, f := range c.fonts {
		if f.face == face {
			return f.synth
		}
	}
	return 0
}

// shapeChain returns the fonts of the family (it may name a style) for the
// shaper; nil if none can be opened. Called with shapeMu held.
func shapeChain(family string, style FontStyle) *fontChain {
	if family == "" {
		family = defaultFontKey
	}
	rf := resolveFont(family)
	rf.style |= style
	fontsMu.RLock()
	primary, synth, ok := lookupFont(rf.family, rf.style)
	if !ok {
		primary, synth, ok = lookupFont(defaultFontKey, rf.style)
	}
	type styled struct {
		f     *sfnt.Font
		synth FontStyle
	}
	var chain []styled
	if ok {
		chain = append(chain, styled{primary, synth})
	}
	for _, name := range fallbackFonts {
		if f, s, ok := lookupFont(name, rf.style); ok && f != primary {
			chain = append(chain, styled{f, s})
		}
	}
	fontsMu.RUnlock()
	c := &fontChain{}
	for _, s := range chain {
		if face := shapeFace(s.f); face != nil {
			c.fonts = append(c.fonts, shapeFont{face, s.synth})
		}
	}
	if len(c.fonts) == 0 {
		return nil
	}
	return c
}

// splitByFace splits the input into runs of the faces of the chain, as
// shaping.SplitByFace does - a space, a joiner, a selector keeps the face
// of the run it is in, so a run is not cut for it (Pango's rule) - but a
// space after emoji takes the face of the text: the spaces of the fonts
// of emoji are as wide as an emoji
func splitByFace(input shaping.Input, chain *fontChain) []shaping.Input {
	var out []shaping.Input
	cur := input
	cur.Face = nil
	for i := input.RunStart; i < input.RunEnd; i++ {
		r := input.Text[i]
		keep := unicode.In(r, unicode.Cc, unicode.Cs, unicode.Zl, unicode.Zp) ||
			(unicode.Is(unicode.Zs, r) && r != '\u1680') || harfbuzz.IsDefaultIgnorable(r)
		space := unicode.Is(unicode.Zs, r) && cur.Face != nil && isColorFace(cur.Face)
		if space {
			keep = false
		}
		if keep && (cur.Face != nil || i < input.RunEnd-1) {
			continue
		}
		face := chain.ResolveFace(r)
		if space {
			face = chain.textFace(r)
		}
		if cur.Face == nil {
			cur.Face = face
		}
		if cur.Face == face {
			continue
		}
		if i != input.RunStart {
			cur.RunEnd = i
			out = append(out, cur)
		}
		cur = input
		cur.RunStart = i
		cur.Face = face
	}
	cur.RunEnd = input.RunEnd
	if cur.Face == nil {
		cur.Face = chain.fonts[0].face
	}
	return append(out, cur)
}
