package ui

import (
	"compress/gzip"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"sort"
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
	key := fontFamily + "_" + fmt.Sprint(fontSize) + "_" + fmt.Sprintf("%v", textColor) + "_" + text

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
	draw.Draw(rgba, dstRect, textImage, srcStart, draw.Over)
}

// GetCharPositions returns the x of each character of text and, last, the
// width of the whole text. Advances and kerning are added up the same way
// as in MeasureText and in drawing, so the cursor stays on the characters.
func GetCharPositions(fontFamily string, fontSize float64, text string) ([]int, error) {
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
			if a, ok := face.GlyphAdvance(r); ok {
				advance += a
			}
			prev = r
			runeIndex++
		}
		positions[runeIndex] = advance.Round()
	})
	return positions, err
}

func MeasureText(fontFamily string, fontSize float64, text string) (int, int, error) {
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

		d := &font.Drawer{
			Dst:  rgba,
			Src:  image.NewUniform(textColor),
			Face: face,
			Dot:  fixed.P(0, textHeight-metrics.Descent.Ceil()),
		}
		d.DrawString(text)
	})
	return rgba
}
