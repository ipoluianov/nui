package platforms

import (
	"bytes"
	"errors"
	"image"
	"image/png"
)

// Images on the clipboard come from any application, and a decoder
// allocates the whole image from the size in the header before it reads a
// pixel: a PNG of a hundred bytes may claim 1000000x1000000. The size is
// checked first, from the header alone.

const (
	// maxClipboardImageSide is the widest and tallest image read from the
	// clipboard (as for a DIB)
	maxClipboardImageSide = 1 << 15
	// maxClipboardImagePixels is the largest image read from the
	// clipboard: 64 Mpx, 256 MB as RGBA
	maxClipboardImagePixels = 1 << 26
)

// ErrClipboardImageTooLarge: the image on the clipboard claims a size
// beyond the limits, it is not decoded
var ErrClipboardImageTooLarge = errors.New("clipboard: the image is too large")

// checkClipboardImageSize returns ErrClipboardImageTooLarge for a size out
// of the limits
func checkClipboardImageSize(cfg image.Config) error {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil // the decoder rejects it itself
	}
	if cfg.Width > maxClipboardImageSide || cfg.Height > maxClipboardImageSide ||
		cfg.Width*cfg.Height > maxClipboardImagePixels {
		return ErrClipboardImageTooLarge
	}
	return nil
}

// DecodeClipboardPNG decodes a PNG of the clipboard, refusing one whose
// header claims a size beyond the limits before anything is allocated
func DecodeClipboardPNG(data []byte) (image.Image, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if err := checkClipboardImageSize(cfg); err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}
