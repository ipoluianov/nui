package platforms

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/image/font/sfnt"
)

// Font files of the application (e.g. the fonts built into nui) are added to
// GDI for this process only, so they are drawn like the system fonts: with
// ClearType and the hinting of the file, which nui's rasterizer doesn't run.

var (
	procAddFontMemResourceEx    = gdi32.NewProc("AddFontMemResourceEx")
	procRemoveFontMemResourceEx = gdi32.NewProc("RemoveFontMemResourceEx")
)

type registeredFontData struct {
	handle uintptr
	data   []byte
}

var (
	fontDataMu sync.Mutex
	// GDI tells the fonts apart by the name, the weight and the style only:
	// another file of the same font replaces the previous one
	fontDataRegistered = map[FontData]registeredFontData{}
)

func registerFontData(data []byte) (FontData, error) {
	fd, err := fontDataInfo(data)
	if err != nil {
		return FontData{}, err
	}
	fontDataMu.Lock()
	defer fontDataMu.Unlock()
	if prev, ok := fontDataRegistered[fd]; ok {
		if bytes.Equal(prev.data, data) {
			return fd, nil
		}
		procRemoveFontMemResourceEx.Call(prev.handle)
		delete(fontDataRegistered, fd)
	}
	if procAddFontMemResourceEx.Find() != nil {
		return FontData{}, ErrSystemTextNotSupported
	}
	var count uint32
	h, _, _ := procAddFontMemResourceEx.Call(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 0, uintptr(unsafe.Pointer(&count)))
	if h == 0 || count == 0 {
		return FontData{}, fmt.Errorf("%w: AddFontMemResourceEx failed", ErrSystemTextNotSupported)
	}
	fontDataRegistered[fd] = registeredFontData{handle: h, data: data}
	return fd, nil
}

func openFontData(fd FontData, pixelSize float64) (SystemFont, error) {
	return openGDIFont(fd.Name, pixelSize, fd.Weight, fd.Italic)
}

// fontDataInfo reads the family name GDI knows the font by (the name ID 1)
// and its weight and style (the OS/2 table)
func fontDataInfo(data []byte) (FontData, error) {
	f, err := sfnt.Parse(data)
	if err != nil {
		return FontData{}, err
	}
	name, err := f.Name(nil, sfnt.NameIDFamily)
	if err != nil || name == "" {
		return FontData{}, fmt.Errorf("%w: the font has no family name", ErrSystemTextNotSupported)
	}
	fd := FontData{Name: name, Weight: c_FW_NORMAL}
	if os2 := sfntTable(data, "OS/2"); len(os2) >= 64 {
		if w := int(binary.BigEndian.Uint16(os2[4:])); w > 0 {
			fd.Weight = w
		}
		fd.Italic = binary.BigEndian.Uint16(os2[62:])&1 != 0
	}
	return fd, nil
}

// sfntTable returns the table of the tag, nil if the font has none
func sfntTable(data []byte, tag string) []byte {
	if len(data) < 12 {
		return nil
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	for i := 0; i < n; i++ {
		rec := 12 + i*16
		if rec+16 > len(data) {
			return nil
		}
		if string(data[rec:rec+4]) != tag {
			continue
		}
		off := int(binary.BigEndian.Uint32(data[rec+8:]))
		size := int(binary.BigEndian.Uint32(data[rec+12:]))
		if off < 0 || size < 0 || off+size > len(data) {
			return nil
		}
		return data[off : off+size]
	}
	return nil
}
