package platforms

import (
	"errors"
	"image"
	"image/color"
)

// ErrSystemFontNotFound is returned when the system has no font of the name
var ErrSystemFontNotFound = errors.New("nui: the system has no such font")

// ErrSystemTextNotSupported is returned where the platform's text engine
// isn't used (macOS: see SystemFontFile)
var ErrSystemTextNotSupported = errors.New("nui: system text rendering is not available")

// SystemFont is a font drawn by the system's own text engine - GDI with
// ClearType on Windows, FreeType with the fontconfig settings (subpixel
// order, hinting, LCD filter) on Linux - so the text looks like in the other
// applications. Safe for concurrent use.
type SystemFont interface {
	// Metrics returns the ascent and the descent of a line, in pixels
	Metrics() (ascent, descent int)
	// Positions returns the x of each character boundary of text: the start
	// of every character and, last, the width of the whole text
	Positions(text string) []int
	// Draw draws text in the color with the top-left corner of its line box
	// at (x, y), within clip. The pixels under the text must be opaque:
	// subpixel antialiasing blends each color channel on its own.
	// col is premultiplied, as color.RGBA is.
	Draw(dst *image.RGBA, text string, col color.RGBA, x, y int, clip image.Rectangle)
}

// OpenSystemFont opens the system font of the family name with the em size in
// pixels. ErrSystemFontNotFound if the system has no such font.
func OpenSystemFont(name string, pixelSize float64) (SystemFont, error) {
	return openSystemFont(name, pixelSize)
}

// SystemUIFontName returns the name of the font the system uses for its
// interface, e.g. "Segoe UI" on Windows
func SystemUIFontName() string {
	return systemUIFontName()
}

// straightColor undoes the premultiplication of color.RGBA
func straightColor(col color.RGBA) (r, g, b, a int) {
	r, g, b, a = int(col.R), int(col.G), int(col.B), int(col.A)
	if a > 0 && a < 255 {
		r, g, b = min(r*255/a, 255), min(g*255/a, 255), min(b*255/a, 255)
	}
	return r, g, b, a
}

// blendMask draws a coverage mask in the color onto dst: per channel for
// a subpixel mask (3 values per pixel, in R, G, B order), the same for all
// the channels for a grayscale one. The mask's top-left pixel goes to (x, y);
// only the pixels within clip are touched.
func blendMask(dst *image.RGBA, col color.RGBA, mask []byte, maskW, maskH, stride int, subpixel bool, x, y int, clip image.Rectangle) {
	r := image.Rect(x, y, x+maskW, y+maskH).Intersect(clip).Intersect(dst.Rect)
	if r.Empty() {
		return
	}
	cr, cg, cb, ca := straightColor(col)
	for py := r.Min.Y; py < r.Max.Y; py++ {
		row := mask[(py-y)*stride:]
		d := dst.Pix[dst.PixOffset(r.Min.X, py):]
		for px := r.Min.X; px < r.Max.X; px++ {
			mx := px - x
			var ar, ag, ab int
			if subpixel {
				ar, ag, ab = int(row[mx*3]), int(row[mx*3+1]), int(row[mx*3+2])
			} else {
				ar = int(row[mx])
				ag, ab = ar, ar
			}
			if ca != 255 {
				ar, ag, ab = ar*ca/255, ag*ca/255, ab*ca/255
			}
			if ar|ag|ab != 0 {
				d[0] = byte(int(d[0]) + (cr-int(d[0]))*ar/255)
				d[1] = byte(int(d[1]) + (cg-int(d[1]))*ag/255)
				d[2] = byte(int(d[2]) + (cb-int(d[2]))*ab/255)
				if a := max(ar, ag, ab); int(d[3]) < a {
					d[3] = byte(a)
				}
			}
			d = d[4:]
		}
	}
}

// FontData is a font file the application handed to the system's text
// engine (see RegisterFontData): the family name, the weight and the style
// the system knows it by
type FontData struct {
	Name   string
	Weight int
	Italic bool
}

// RegisterFontData hands a TrueType or OpenType file to the system's text
// engine, for this process only, so OpenFontData draws it like a system
// font (GDI with ClearType on Windows). ErrSystemTextNotSupported elsewhere.
func RegisterFontData(data []byte) (FontData, error) {
	return registerFontData(data)
}

// OpenFontData opens a font of RegisterFontData with the em size in pixels
func OpenFontData(fd FontData, pixelSize float64) (SystemFont, error) {
	return openFontData(fd, pixelSize)
}
