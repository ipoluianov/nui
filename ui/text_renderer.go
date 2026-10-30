package ui

import (
	"compress/gzip"
	"embed"
	"fmt"
	"image"
	"image/color"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Font families registered by default
const (
	// FontFamilySans is the font of the interface: Noto Sans
	FontFamilySans = "notosans"
	// FontFamilyMono is a monospaced font for code, logs and aligned
	// columns of numbers: JetBrains Mono (see Widget.SetFontFamily)
	FontFamilyMono = "jetbrainsmono"
)

// The regular faces of the built-in families are registered at start; the
// other faces are gzipped and unpacked when first needed (see
// loadBuiltinFaces): unpacking takes milliseconds, and most applications
// use only the regular faces.

//go:embed "fonts/NotoSans-Regular.ttf"
var fontNotoSans []byte

//go:embed "fonts/JetBrainsMono-Regular.ttf"
var fontJetBrainsMono []byte

//go:embed fonts/*.ttf.gz
var builtinFontFiles embed.FS

var builtinFaces = map[string]string{
	FontFamilySans + ":bold":        "NotoSans-Bold",
	FontFamilySans + ":italic":      "NotoSans-Italic",
	FontFamilySans + ":bold:italic": "NotoSans-BoldItalic",
	FontFamilyMono + ":bold":        "JetBrainsMono-Bold",
	FontFamilyMono + ":italic":      "JetBrainsMono-Italic",
	FontFamilyMono + ":bold:italic": "JetBrainsMono-BoldItalic",
}

var (
	builtinFacesMu     sync.Mutex
	builtinFacesLoaded = map[string]bool{}
)

// loadBuiltinFaces registers the built-in faces of the family (lowercase,
// without a style) the style may be drawn with, if not yet. A face the
// application registered itself stays.
func loadBuiltinFaces(family string, style FontStyle) {
	// Held while a face is registered: text of the style waits for the face
	// instead of being drawn with a synthesized one meanwhile
	builtinFacesMu.Lock()
	defer builtinFacesMu.Unlock()
	for _, s := range styleCandidates(style) {
		name := faceName(family, s)
		file, ok := builtinFaces[name]
		if !ok || builtinFacesLoaded[name] {
			continue
		}
		builtinFacesLoaded[name] = true
		fontsMu.RLock()
		_, registered := fonts[name]
		fontsMu.RUnlock()
		if !registered {
			mustRegisterFont(name, builtinFontData(file))
		}
	}
}

// builtinFontData unpacks the built-in font file
func builtinFontData(file string) []byte {
	f, err := builtinFontFiles.Open("fonts/" + file + ".ttf.gz")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return data
}

type renderedText struct {
	key          string
	lastAccessDT time.Time
	textImage    *image.RGBA
}

var renderedTextsMu sync.Mutex
var renderedTexts = make(map[string]*renderedText)
var renderedTextLastClearDT time.Time

// renderedTextsBytes is the size of the pixels of renderedTexts. Above
// renderedTextsMaxBytes the least recently drawn texts go: a text that
// changes on every frame (a counter, a clock, axis labels) adds a new image
// each time, too fast for the eviction by age alone.
var renderedTextsBytes int

const renderedTextsMaxBytes = 64 << 20

// Registered in a variable initializer: those run before any init(), and
// init() in theme.go already measures text
var _ = mustRegisterFont(FontFamilySans, fontNotoSans)
var _ = mustRegisterFont(FontFamilyMono, fontJetBrainsMono)

// The built-in mono font is the first fallback: it has the symbols Noto
// Sans lacks, e.g. the keys of the shortcuts on macOS (⌘ ⇧ ⌃ ⌥) and arrows
var _ = func() bool { AddFallbackFont(FontFamilyMono); return true }()

func mustRegisterFont(family string, data []byte) bool {
	if err := RegisterFont(family, data); err != nil {
		panic(err)
	}
	return true
}

// clearAllRenderedTexts drops the rendered texts, e.g. after the fonts changed.
func clearAllRenderedTexts() {
	renderedTextsMu.Lock()
	renderedTexts = make(map[string]*renderedText)
	renderedTextsBytes = 0
	renderedTextsMu.Unlock()
	// The layouts of DrawTextF, in the fonts before
	textFLayoutsMu.Lock()
	clear(textFLayouts)
	textFLayoutsMu.Unlock()
}

// clearRenderedTexts must be called with renderedTextsMu held.
func clearRenderedTexts() {
	if renderedTextLastClearDT.IsZero() {
		renderedTextLastClearDT = time.Now()
		return
	}
	if time.Since(renderedTextLastClearDT) < 1*time.Minute {
		return
	}
	renderedTextLastClearDT = time.Now()
	for k, rt := range renderedTexts {
		if time.Since(rt.lastAccessDT) > 30*time.Second {
			deleteRenderedText(k, rt)
		}
	}
}

// deleteRenderedText must be called with renderedTextsMu held.
func deleteRenderedText(key string, rt *renderedText) {
	delete(renderedTexts, key)
	renderedTextsBytes -= len(rt.textImage.Pix)
}

// trimRenderedTexts drops the least recently drawn texts down to 3/4 of
// renderedTextsMaxBytes, if over it: the sort is rare that way. Must be
// called with renderedTextsMu held.
func trimRenderedTexts() {
	if renderedTextsBytes <= renderedTextsMaxBytes {
		return
	}
	all := make([]*renderedText, 0, len(renderedTexts))
	for _, rt := range renderedTexts {
		all = append(all, rt)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].lastAccessDT.Before(all[j].lastAccessDT) })
	for _, rt := range all {
		if renderedTextsBytes <= renderedTextsMaxBytes*3/4 {
			break
		}
		deleteRenderedText(rt.key, rt)
	}
}

func DrawText(rgba *image.RGBA, text string, textColor color.Color, fontFamily string, fontSize float64, x int, y int, clipX int, clipY int, clipWidth int, clipHeight int) {
	// A system font is drawn straight onto the pixels under it: its subpixel
	// antialiasing depends on them, so there is no image to cache
	if sf := systemFontFor(fontFamily, fontSize, text); sf != nil {
		sf.Draw(rgba, text, colorToRGBA(textColor), x, y, image.Rect(clipX, clipY, clipX+clipWidth, clipY+clipHeight))
		return
	}

	var textImage *image.RGBA
	key := renderedTextKey(fontFamily, fontSize, textColor, text)

	renderedTextsMu.Lock()
	img, ok := renderedTexts[key]
	if ok {
		img.lastAccessDT = time.Now()
	}
	renderedTextsMu.Unlock()

	if ok {
		textImage = img.textImage
	} else {
		textImage = renderText(text, textColor, fontFamily, fontSize)
		if textImage == nil {
			return
		}
		var img renderedText
		img.key = key
		img.lastAccessDT = time.Now()
		img.textImage = textImage

		renderedTextsMu.Lock()
		if old, ok := renderedTexts[key]; ok { // drawn meanwhile by another goroutine
			deleteRenderedText(key, old)
		}
		renderedTexts[key] = &img
		renderedTextsBytes += len(textImage.Pix)
		clearRenderedTexts()
		trimRenderedTexts()
		renderedTextsMu.Unlock()
	}

	textBounds := textImage.Bounds()
	if textBounds.Dx() <= 0 || textBounds.Dy() <= 0 {
		return
	}
	// On the baseline of the system font the other text is drawn with
	if ascent, _, ok := nativeMetrics(fontFamily, fontSize); ok {
		withFace(fontFamily, fontSize, func(face font.Face) {
			m := face.Metrics()
			y += ascent - ((m.Ascent + m.Descent).Ceil() - m.Descent.Ceil())
		})
	}

	// The image's x 0 is the start of the text (see renderText)
	dstRect := textBounds.Add(image.Pt(x, y))
	clipRect := image.Rect(clipX, clipY, clipX+clipWidth, clipY+clipHeight)
	dstRect = dstRect.Intersect(clipRect)

	if dstRect.Empty() {
		return
	}

	srcStart := dstRect.Min.Sub(image.Pt(x, y))
	drawOver(rgba, dstRect, textImage, srcStart)
}

// renderedTextKey is the key of the text's image in renderedTexts
func renderedTextKey(fontFamily string, fontSize float64, textColor color.Color, text string) string {
	b := make([]byte, 0, len(fontFamily)+len(text)+32)
	b = append(b, fontFamily...)
	b = append(b, '_')
	b = strconv.AppendFloat(b, fontSize, 'g', -1, 64)
	b = append(b, '_')
	if c, ok := textColor.(color.RGBA); ok {
		b = append(b, c.R, c.G, c.B, c.A)
	} else {
		// Another type with the same components is another color
		b = fmt.Appendf(b, "%T%v", textColor, textColor)
	}
	b = append(b, '_')
	b = append(b, text...)
	return string(b)
}

// drawOver draws src over dst in r, from sp of src: the same as draw.Draw
// with draw.Over, to the bit, but faster for the image of a text, most of
// whose pixels are transparent or opaque. src is premultiplied, as
// image.RGBA is.
func drawOver(dst *image.RGBA, r image.Rectangle, src *image.RGBA, sp image.Point) {
	// Clipped to both images, as draw.Draw does
	orig := r.Min
	r = r.Intersect(dst.Rect).Intersect(src.Rect.Add(orig.Sub(sp)))
	if r.Empty() {
		return
	}
	sp = sp.Add(r.Min.Sub(orig))

	const m = 1<<16 - 1
	width := r.Dx() * 4
	for y := 0; y < r.Dy(); y++ {
		d0 := dst.PixOffset(r.Min.X, r.Min.Y+y)
		s0 := src.PixOffset(sp.X, sp.Y+y)
		dpix := dst.Pix[d0 : d0+width : d0+width]
		spix := src.Pix[s0 : s0+width : s0+width]
		for i := 0; i < width; i += 4 {
			s := spix[i : i+4 : i+4]
			switch {
			case s[3] == 0 && s[0]|s[1]|s[2] == 0:
				// Transparent: draw.Over leaves the pixel as it is
			case s[3] == 255:
				// Opaque: draw.Over gives the source pixel
				copy(dpix[i:i+4], s)
			default:
				// image/draw's drawCopyOver
				sr := uint32(s[0]) * 0x101
				sg := uint32(s[1]) * 0x101
				sb := uint32(s[2]) * 0x101
				sa := uint32(s[3]) * 0x101
				a := (m - sa) * 0x101
				d := dpix[i : i+4 : i+4]
				d[0] = uint8((uint32(d[0])*a/m + sr) >> 8)
				d[1] = uint8((uint32(d[1])*a/m + sg) >> 8)
				d[2] = uint8((uint32(d[2])*a/m + sb) >> 8)
				d[3] = uint8((uint32(d[3])*a/m + sa) >> 8)
			}
		}
	}
}

// GetCharPositions returns the x of each character of text and, last, the
// width of the whole text. Advances and kerning are added up the same way
// as in MeasureText and in drawing, so the cursor stays on the characters.
// On a scaled screen they are measured in its pixels (see textScale).
func GetCharPositions(fontFamily string, fontSize float64, text string) ([]int, error) {
	s := textScale()
	if s == 1 {
		return charPositions(fontFamily, fontSize, text)
	}
	positions, err := charPositions(fontFamily, fontSize*s, text)
	for i, p := range positions {
		positions[i] = toLogicalText(p, s)
	}
	return positions, err
}

func charPositions(fontFamily string, fontSize float64, text string) ([]int, error) {
	if sf := systemFontFor(fontFamily, fontSize, text); sf != nil {
		return sf.Positions(text), nil
	}
	stringLenInRunes := len([]rune(text))

	positions := make([]int, stringLenInRunes+1) // на 1 больше, чтобы последняя позиция = ширине всей строки

	ensureFallbackFonts(text)
	err := withFace(fontFamily, fontSize, func(face font.Face) {
		var advance fixed.Int26_6
		prev := rune(-1)
		runeIndex := 0
		for _, r := range text {
			if prev >= 0 {
				advance += face.Kern(prev, r)
			}
			positions[runeIndex] = advance.Round()
			// A character no font has is drawn as the box of the font
			// (.notdef), and measured with it: its width counts here too,
			// ok or not
			a, _ := face.GlyphAdvance(r)
			advance += a
			prev = r
			runeIndex++
		}
		positions[runeIndex] = advance.Round()
	})
	return positions, err
}

// MeasureText returns the width and the height of text, a single line. On
// a scaled screen it is measured in its pixels (see textScale).
func MeasureText(fontFamily string, fontSize float64, text string) (int, int, error) {
	s := textScale()
	if s == 1 {
		return measureText(fontFamily, fontSize, text)
	}
	width, height, err := measureText(fontFamily, fontSize*s, text)
	return toLogicalText(width, s), toLogicalText(height, s), err
}

func measureText(fontFamily string, fontSize float64, text string) (int, int, error) {
	if sf := systemFontFor(fontFamily, fontSize, text); sf != nil {
		positions := sf.Positions(text)
		ascent, descent := sf.Metrics()
		return positions[len(positions)-1], ascent + descent, nil
	}
	var textWidth, textHeight int
	ensureFallbackFonts(text)
	err := withFace(fontFamily, fontSize, func(face font.Face) {
		metrics := face.Metrics()
		textWidth = font.MeasureString(face, text).Ceil()
		textHeight = (metrics.Ascent + metrics.Descent).Ceil()
	})
	if ascent, descent, ok := nativeMetrics(fontFamily, fontSize); ok {
		textHeight = ascent + descent
	}
	return textWidth, textHeight, err
}

// textLines splits the text into lines at "\n" or "\r\n", as
// Canvas.DrawText draws them
func textLines(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}

// measureMultilineTextWidth returns the width of the widest line of text
// (see textLines)
func measureMultilineTextWidth(fontFamily string, fontSize float64, text string) (int, error) {
	maxWidth := 0
	for _, line := range textLines(text) {
		lineWidth, _, err := MeasureText(fontFamily, fontSize, line)
		if err != nil {
			return 0, err
		}
		if lineWidth > maxWidth {
			maxWidth = lineWidth
		}
	}
	return maxWidth, nil
}

func renderText(text string, textColor color.Color, fontFamily string, fontSize float64) *image.RGBA {
	var rgba *image.RGBA
	ensureFallbackFonts(text)
	withFace(fontFamily, fontSize, func(face font.Face) {
		// The image is as wide as the ink: an italic letter may stand out
		// of its advance, e.g. the top of the last one. x 0 is the pen's
		// start, where DrawText puts it.
		bounds, advance := font.BoundString(face, text)
		minX := min(0, bounds.Min.X.Floor())
		maxX := max(advance.Ceil(), bounds.Max.X.Ceil())
		metrics := face.Metrics()
		textHeight := (metrics.Ascent + metrics.Descent).Ceil()

		rgba = image.NewRGBA(image.Rect(minX, 0, maxX, textHeight))

		// The canvas takes color.RGBA as straight alpha (see
		// straightRGBA8); image/draw would read it as premultiplied
		src := textColor
		if c, ok := textColor.(color.RGBA); ok {
			src = color.NRGBA(c)
		}
		d := &font.Drawer{
			Dst:  rgba,
			Src:  image.NewUniform(src),
			Face: face,
			Dot:  fixed.P(0, textHeight-metrics.Descent.Ceil()),
		}
		d.DrawString(text)
	})
	return rgba
}
