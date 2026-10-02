package platforms

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// System fonts on Windows are drawn by GDI with ClearType, like the text of
// the native controls. ClearType blends each color channel with what is
// under the text, so the text isn't drawn into a mask: the pixels under it
// are copied into a GDI bitmap, GDI draws the text there in its color, and
// the result is copied back. GDI does all the blending (and its gamma)
// itself - the text looks exactly as in the other applications.

var (
	procCreateFontW           = gdi32.NewProc("CreateFontW")
	procCreateCompatibleDC    = gdi32.NewProc("CreateCompatibleDC")
	procSelectObject          = gdi32.NewProc("SelectObject")
	procDeleteObject          = gdi32.NewProc("DeleteObject")
	procDeleteDC              = gdi32.NewProc("DeleteDC")
	procGetTextMetricsW       = gdi32.NewProc("GetTextMetricsW")
	procGetTextFaceW          = gdi32.NewProc("GetTextFaceW")
	procGetTextExtentExPointW = gdi32.NewProc("GetTextExtentExPointW")
	procCreateDIBSection      = gdi32.NewProc("CreateDIBSection")
	procSetBkMode             = gdi32.NewProc("SetBkMode")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procExtTextOutW           = gdi32.NewProc("ExtTextOutW")
	procGdiFlush              = gdi32.NewProc("GdiFlush")
	procSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
)

const (
	c_FW_NORMAL               = 400
	c_DEFAULT_CHARSET         = 1
	c_CLEARTYPE_QUALITY       = 5
	c_TRANSPARENT             = 1
	c_SPI_GETNONCLIENTMETRICS = 0x0029
	// NONCLIENTMETRICSW: its size and where lfMessageFont.lfFaceName is
	nonClientMetricsSize = 504
	messageFontFaceName  = 408 + 28
)

type textMetricW struct {
	tmHeight, tmAscent, tmDescent int32
	rest                          [48]byte
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

type windowsFont struct {
	mu      sync.Mutex
	dc      uintptr
	font    uintptr
	ascent  int
	descent int

	// The bitmap the text is drawn in, grown as needed
	dib        uintptr
	dibBits    unsafe.Pointer
	dibW, dibH int
}

func systemUIFontName() string {
	buf := make([]byte, nonClientMetricsSize)
	*(*uint32)(unsafe.Pointer(&buf[0])) = nonClientMetricsSize
	ret, _, _ := procSystemParametersInfoW.Call(c_SPI_GETNONCLIENTMETRICS, nonClientMetricsSize, uintptr(unsafe.Pointer(&buf[0])), 0)
	if ret == 0 {
		return "Segoe UI"
	}
	name := unsafe.Slice((*uint16)(unsafe.Pointer(&buf[messageFontFaceName])), 32)
	if s := syscall.UTF16ToString(name); s != "" {
		return s
	}
	return "Segoe UI"
}

func openSystemFont(name string, pixelSize float64) (SystemFont, error) {
	return openGDIFont(name, pixelSize, c_FW_NORMAL, false)
}

func openGDIFont(name string, pixelSize float64, weight int, italic bool) (SystemFont, error) {
	face, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	dc, _, _ := procCreateCompatibleDC.Call(0)
	if dc == 0 {
		return nil, fmt.Errorf("%w: CreateCompatibleDC failed", ErrSystemTextNotSupported)
	}
	// A negative height is the em size, as the size of the other fonts
	height := -int32(math.Round(pixelSize))
	var italicFlag uintptr
	if italic {
		italicFlag = 1
	}
	hfont, _, _ := procCreateFontW.Call(uintptr(height), 0, 0, 0, uintptr(weight), italicFlag, 0, 0,
		c_DEFAULT_CHARSET, 0, 0, c_CLEARTYPE_QUALITY, 0, uintptr(unsafe.Pointer(face)))
	if hfont == 0 {
		procDeleteDC.Call(dc)
		return nil, fmt.Errorf("%w: CreateFontW failed", ErrSystemTextNotSupported)
	}
	procSelectObject.Call(dc, hfont)

	// GDI substitutes another font for an unknown name
	var got [64]uint16
	procGetTextFaceW.Call(dc, uintptr(len(got)), uintptr(unsafe.Pointer(&got[0])))
	if !strings.EqualFold(syscall.UTF16ToString(got[:]), name) {
		procDeleteDC.Call(dc)
		procDeleteObject.Call(hfont)
		return nil, fmt.Errorf("%w: %q", ErrSystemFontNotFound, name)
	}

	var tm textMetricW
	procGetTextMetricsW.Call(dc, uintptr(unsafe.Pointer(&tm)))
	procSetBkMode.Call(dc, c_TRANSPARENT)
	return &windowsFont{dc: dc, font: hfont, ascent: int(tm.tmAscent), descent: int(tm.tmDescent)}, nil
}

func (f *windowsFont) Metrics() (int, int) {
	return f.ascent, f.descent
}

// extents returns the x after each UTF-16 unit of s
func (f *windowsFont) extents(s []uint16) ([]int32, int) {
	if len(s) == 0 {
		return nil, 0
	}
	dx := make([]int32, len(s))
	var size struct{ cx, cy int32 }
	procGetTextExtentExPointW.Call(f.dc, uintptr(unsafe.Pointer(&s[0])), uintptr(len(s)), 0, 0,
		uintptr(unsafe.Pointer(&dx[0])), uintptr(unsafe.Pointer(&size)))
	return dx, int(size.cx)
}

func (f *windowsFont) Positions(text string) []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	runes := []rune(text)
	s := utf16.Encode(runes)
	dx, width := f.extents(s)
	positions := make([]int, len(runes)+1)
	unit := 0
	for i, r := range runes {
		if unit > 0 {
			positions[i] = int(dx[unit-1])
		}
		// utf16.Encode writes an invalid rune as one U+FFFD
		n := utf16.RuneLen(r)
		if n < 1 {
			n = 1
		}
		unit += n
	}
	positions[len(runes)] = width
	return positions
}

// ensureDIB makes the bitmap at least w x h
func (f *windowsFont) ensureDIB(w, h int) bool {
	if f.dib != 0 && w <= f.dibW && h <= f.dibH {
		return true
	}
	w, h = max(w, f.dibW, 256), max(h, f.dibH, 32)
	bi := bitmapInfoHeader{biWidth: int32(w), biHeight: -int32(h), biPlanes: 1, biBitCount: 32}
	bi.biSize = uint32(unsafe.Sizeof(bi))
	var bits unsafe.Pointer
	dib, _, _ := procCreateDIBSection.Call(f.dc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if dib == 0 {
		return false
	}
	procSelectObject.Call(f.dc, dib)
	if f.dib != 0 {
		procDeleteObject.Call(f.dib)
	}
	f.dib, f.dibBits, f.dibW, f.dibH = dib, bits, w, h
	return true
}

func (f *windowsFont) Draw(dst *image.RGBA, text string, col color.RGBA, x, y int, clip image.Rectangle) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := utf16.Encode([]rune(text))
	_, width := f.extents(s)
	if len(s) == 0 {
		return
	}
	// ClearType and overhanging glyphs reach a little beyond the advances
	const margin = 3
	r := image.Rect(x-margin, y, x+width+margin, y+f.ascent+f.descent).Intersect(clip).Intersect(dst.Rect)
	if r.Empty() || !f.ensureDIB(r.Dx(), r.Dy()) {
		return
	}
	stride := f.dibW * 4
	bits := unsafe.Slice((*byte)(f.dibBits), stride*f.dibH)

	if !opaque(dst, r) {
		f.drawMask(dst, s, col, x, y, r, bits, stride)
		return
	}

	// The pixels under the text into the bitmap, RGBA to BGRA
	for py := 0; py < r.Dy(); py++ {
		src := dst.Pix[dst.PixOffset(r.Min.X, r.Min.Y+py):]
		row := bits[py*stride:]
		for px := 0; px < r.Dx(); px++ {
			row[px*4], row[px*4+1], row[px*4+2], row[px*4+3] = src[px*4+2], src[px*4+1], src[px*4], 0
		}
	}

	cr, cg, cb, ca := straightColor(col)
	colorRef := uintptr(cr) | uintptr(cg)<<8 | uintptr(cb)<<16
	procSetTextColor.Call(f.dc, colorRef)
	procExtTextOutW.Call(f.dc, uintptr(int32(x-r.Min.X)), uintptr(int32(y-r.Min.Y)), 0, 0,
		uintptr(unsafe.Pointer(&s[0])), uintptr(len(s)), 0)
	procGdiFlush.Call()

	// And back; GDI draws opaque text, a translucent color is mixed in here
	for py := 0; py < r.Dy(); py++ {
		d := dst.Pix[dst.PixOffset(r.Min.X, r.Min.Y+py):]
		row := bits[py*stride:]
		for px := 0; px < r.Dx(); px++ {
			nr, ng, nb := int(row[px*4+2]), int(row[px*4+1]), int(row[px*4])
			if ca < 255 {
				nr = int(d[px*4]) + (nr-int(d[px*4]))*ca/255
				ng = int(d[px*4+1]) + (ng-int(d[px*4+1]))*ca/255
				nb = int(d[px*4+2]) + (nb-int(d[px*4+2]))*ca/255
			}
			d[px*4], d[px*4+1], d[px*4+2] = byte(nr), byte(ng), byte(nb)
		}
	}
}

// opaque tells whether all the pixels of r are opaque
func opaque(img *image.RGBA, r image.Rectangle) bool {
	for py := r.Min.Y; py < r.Max.Y; py++ {
		row := img.Pix[img.PixOffset(r.Min.X, py):]
		for px := 0; px < r.Dx(); px++ {
			if row[px*4+3] != 255 {
				return false
			}
		}
	}
	return true
}

// drawMask draws the text where the pixels under it aren't all opaque
// (e.g. an image with a transparent background): ClearType has nothing to
// blend with there, so GDI draws white text on black, which is the coverage,
// and the text is blended in grayscale with it as the alpha.
func (f *windowsFont) drawMask(dst *image.RGBA, s []uint16, col color.RGBA, x, y int, r image.Rectangle, bits []byte, stride int) {
	for py := 0; py < r.Dy(); py++ {
		clear(bits[py*stride : py*stride+r.Dx()*4])
	}
	procSetTextColor.Call(f.dc, 0xFFFFFF)
	procExtTextOutW.Call(f.dc, uintptr(int32(x-r.Min.X)), uintptr(int32(y-r.Min.Y)), 0, 0,
		uintptr(unsafe.Pointer(&s[0])), uintptr(len(s)), 0)
	procGdiFlush.Call()

	mask := make([]byte, r.Dx()*r.Dy())
	for py := 0; py < r.Dy(); py++ {
		row := bits[py*stride:]
		for px := 0; px < r.Dx(); px++ {
			mask[py*r.Dx()+px] = byte((int(row[px*4]) + int(row[px*4+1]) + int(row[px*4+2])) / 3)
		}
	}
	blendMask(dst, col, mask, r.Dx(), r.Dy(), r.Dx(), false, r.Min.X, r.Min.Y, r)
}
