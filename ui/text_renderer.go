package ui

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
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

//go:embed "fonts/NotoSans-Regular.ttf"
var fontNotoSans []byte

//go:embed "fonts/JetBrainsMono-Regular.ttf"
var fontJetBrainsMono []byte

type renderedText struct {
	key          string
	lastAccessDT time.Time
	textImage    *image.RGBA
}

var renderedTextsMu sync.Mutex
var renderedTexts = make(map[string]*renderedText)
var renderedTextLastClearDT time.Time

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
			delete(renderedTexts, k)
		}
	}
}

func DrawText(rgba *image.RGBA, text string, textColor color.Color, fontFamily string, fontSize float64, x int, y int, clipX int, clipY int, clipWidth int, clipHeight int) {
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
		renderedTexts[key] = &img
		clearRenderedTexts()
		renderedTextsMu.Unlock()
	}

	textBounds := textImage.Bounds()
	if textBounds.Dx() <= 0 || textBounds.Dy() <= 0 {
		return
	}

	dstRect := image.Rect(x, y, x+textBounds.Dx(), y+textBounds.Dy())
	clipRect := image.Rect(clipX, clipY, clipX+clipWidth, clipY+clipHeight)
	dstRect = dstRect.Intersect(clipRect)

	if dstRect.Empty() {
		return
	}

	srcStart := textBounds.Min.Add(dstRect.Min.Sub(image.Pt(x, y)))
	draw.Draw(rgba, dstRect, textImage, srcStart, draw.Over)
}

// GetCharPositions returns the x of each character of text and, last, the
// width of the whole text. Advances and kerning are added up the same way
// as in MeasureText and in drawing, so the cursor stays on the characters.
func GetCharPositions(fontFamily string, fontSize float64, text string) ([]int, error) {
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
	var textWidth, textHeight int
	ensureFallbackFonts(text)
	err := withFace(fontFamily, fontSize, func(face font.Face) {
		metrics := face.Metrics()
		textWidth = font.MeasureString(face, text).Ceil()
		textHeight = (metrics.Ascent + metrics.Descent).Ceil()
	})
	return textWidth, textHeight, err
}

// measureMultilineTextWidth returns the width of the widest line of text,
// where lines are separated by "\r\n" - the same separator Canvas.DrawText
// splits on when rendering multiline text.
func measureMultilineTextWidth(fontFamily string, fontSize float64, text string) (int, error) {
	maxWidth := 0
	for _, line := range strings.Split(text, "\r\n") {
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
		textWidth := font.MeasureString(face, text).Ceil()
		metrics := face.Metrics()
		textHeight := (metrics.Ascent + metrics.Descent).Ceil()

		rgba = image.NewRGBA(image.Rect(0, 0, textWidth, textHeight))

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
