package platforms

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"math/bits"
)

// Images on the Windows clipboard are DIBs (CF_DIB, CF_DIBV5): a
// BITMAPINFOHEADER (or the larger V4, V5 headers) and the pixels, without
// the file header of a .bmp. Platform independent, so it is tested anywhere.

// dibFromImage makes a CF_DIB: 32 bits per pixel, the rows bottom-up, the
// alpha in the fourth byte (applications that ignore it see the colors)
func dibFromImage(img image.Image) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dib := make([]byte, 40+w*h*4)
	le := binary.LittleEndian
	le.PutUint32(dib[0:], 40)
	le.PutUint32(dib[4:], uint32(w))
	le.PutUint32(dib[8:], uint32(h))
	le.PutUint16(dib[12:], 1)  // planes
	le.PutUint16(dib[14:], 32) // bits per pixel
	le.PutUint32(dib[20:], uint32(w*h*4))
	px := dib[40:]
	for y := 0; y < h; y++ {
		row := px[(h-1-y)*w*4:]
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = c.B, c.G, c.R, c.A
		}
	}
	return dib
}

const (
	biRGB       = 0
	biBitfields = 3
)

var errDIB = errors.New("clipboard: an image of an unsupported format")

// imageFromDIB reads a CF_DIB or CF_DIBV5 of 24 or 32 bits per pixel
func imageFromDIB(dib []byte) (image.Image, error) {
	le := binary.LittleEndian
	if len(dib) < 40 {
		return nil, errDIB
	}
	headerSize := int(le.Uint32(dib[0:]))
	w := int(int32(le.Uint32(dib[4:])))
	h := int(int32(le.Uint32(dib[8:])))
	bpp := int(le.Uint16(dib[14:]))
	compression := le.Uint32(dib[16:])
	if headerSize < 40 || headerSize > len(dib) || w <= 0 || h == 0 || w > 1<<15 || h > 1<<15 || h < -(1<<15) {
		return nil, errDIB
	}
	topDown := h < 0
	if topDown {
		h = -h
	}
	masks := [4]uint32{0xFF0000, 0xFF00, 0xFF, 0}
	offset := headerSize
	switch {
	case compression == biBitfields && bpp == 32:
		if headerSize >= 52 { // V4, V5: the masks are in the header
			masks = [4]uint32{le.Uint32(dib[40:]), le.Uint32(dib[44:]), le.Uint32(dib[48:]), 0}
			if headerSize >= 56 {
				masks[3] = le.Uint32(dib[52:])
			}
		} else if len(dib) >= 52 { // after the header
			masks = [4]uint32{le.Uint32(dib[40:]), le.Uint32(dib[44:]), le.Uint32(dib[48:]), 0}
			offset += 12
		}
	case compression == biRGB && (bpp == 32 || bpp == 24):
		if bpp == 32 {
			masks[3] = 0xFF000000
		}
	default:
		return nil, errDIB
	}
	stride := (w*bpp/8 + 3) &^ 3
	if len(dib) < offset+stride*h {
		return nil, errDIB
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	channel := func(v, mask uint32) uint8 {
		if mask == 0 {
			return 0xFF
		}
		v = (v & mask) >> bits.TrailingZeros32(mask)
		if n := bits.OnesCount32(mask); n < 8 {
			return uint8(v * 255 / (1<<n - 1))
		} else {
			return uint8(v >> (n - 8))
		}
	}
	anyAlpha := false
	for y := 0; y < h; y++ {
		src := dib[offset+(h-1-y)*stride:]
		if topDown {
			src = dib[offset+y*stride:]
		}
		dst := img.Pix[y*img.Stride:]
		for x := 0; x < w; x++ {
			var v uint32
			if bpp == 32 {
				v = le.Uint32(src[x*4:])
			} else {
				v = uint32(src[x*3]) | uint32(src[x*3+1])<<8 | uint32(src[x*3+2])<<16
			}
			r, g, b := channel(v, masks[0]), channel(v, masks[1]), channel(v, masks[2])
			a := uint8(0xFF)
			if masks[3] != 0 {
				a = channel(v, masks[3])
				anyAlpha = anyAlpha || a != 0
			}
			dst[x*4], dst[x*4+1], dst[x*4+2], dst[x*4+3] = r, g, b, a
		}
	}
	// A 32-bit DIB whose alpha is all 0 has no alpha: many applications
	// leave the fourth byte empty
	if masks[3] != 0 && !anyAlpha {
		for i := 3; i < len(img.Pix); i += 4 {
			img.Pix[i] = 0xFF
		}
	}
	return img, nil
}
