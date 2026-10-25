//go:build darwin

package platforms

import (
	"bytes"
	"errors"
	"image/png"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"golang.org/x/image/tiff"
)

// The clipboard is the general NSPasteboard: the text in it of the type
// NSPasteboardTypeString ("public.utf8-plain-text"), an image as
// "public.png" (and read as "public.tiff" too, which many applications give
// only), the other MIME types as pasteboard types by the same names

var (
	selGeneralPasteboard = objc.RegisterName("generalPasteboard")
	selClearContents     = objc.RegisterName("clearContents")
	selSetStringForType  = objc.RegisterName("setString:forType:")
	selStringForType     = objc.RegisterName("stringForType:")
	selSetDataForType    = objc.RegisterName("setData:forType:")
	selDataForType       = objc.RegisterName("dataForType:")
	selTypes             = objc.RegisterName("types")
	selDataWithBytes     = objc.RegisterName("dataWithBytes:length:")
	selBytes             = objc.RegisterName("bytes")
)

const (
	nsPasteboardTypeString = "public.utf8-plain-text"
	nsPasteboardTypePNG    = "public.png"
	nsPasteboardTypeTIFF   = "public.tiff"
	nsPasteboardTypeHTML   = "public.html"
)

func generalPasteboard() objc.ID {
	return objc.ID(objc.GetClass("NSPasteboard")).Send(selGeneralPasteboard)
}

// pasteboardType is the type of the pasteboard for the MIME type
func pasteboardType(mime string) string {
	switch {
	case isTextMIME(mime):
		return nsPasteboardTypeString
	case mime == MIMEPNG:
		return nsPasteboardTypePNG
	case mime == "text/html":
		return nsPasteboardTypeHTML
	}
	return mime
}

func clipboardSet(items []ClipboardItem) error {
	ok := true
	withAutoreleasePool(func() {
		pb := generalPasteboard()
		pb.Send(selClearContents)
		for _, item := range items {
			typ := goStringToNS(pasteboardType(item.MIME))
			if isTextMIME(item.MIME) {
				ok = objc.Send[bool](pb, selSetStringForType, goStringToNS(string(item.Data)), typ) && ok
				continue
			}
			var p unsafe.Pointer
			if len(item.Data) > 0 {
				p = unsafe.Pointer(&item.Data[0])
			}
			data := objc.ID(objc.GetClass("NSData")).Send(selDataWithBytes, p, len(item.Data))
			ok = objc.Send[bool](pb, selSetDataForType, data, typ) && ok
		}
	})
	if !ok {
		return errors.New("clipboard: the data was not put on the pasteboard")
	}
	return nil
}

func clipboardGetText() (string, error) {
	var text string
	withAutoreleasePool(func() {
		s := generalPasteboard().Send(selStringForType, goStringToNS(nsPasteboardTypeString))
		text = nsStringToGo(s)
	})
	return text, nil
}

// pasteboardData returns the data of the type on the pasteboard
func pasteboardData(typ string) (data []byte, ok bool) {
	withAutoreleasePool(func() {
		d := generalPasteboard().Send(selDataForType, goStringToNS(typ))
		if d == 0 {
			return
		}
		ok = true
		n := objc.Send[int](d, selLength)
		if n > 0 {
			p := objc.Send[unsafe.Pointer](d, selBytes)
			data = append([]byte(nil), unsafe.Slice((*byte)(p), n)...)
		}
	})
	return data, ok
}

func clipboardGet(mime string) ([]byte, bool, error) {
	if isTextMIME(mime) {
		data, ok := pasteboardData(nsPasteboardTypeString)
		return data, ok, nil
	}
	if data, ok := pasteboardData(pasteboardType(mime)); ok {
		return data, true, nil
	}
	if mime != MIMEPNG {
		return nil, false, nil
	}
	data, ok := pasteboardData(nsPasteboardTypeTIFF)
	if !ok {
		return nil, false, nil
	}
	img, err := tiff.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false, nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, false, err
	}
	return buf.Bytes(), true, nil
}

func clipboardFormats() ([]string, error) {
	var formats []string
	add := func(name string) {
		for _, f := range formats {
			if f == name {
				return
			}
		}
		formats = append(formats, name)
	}
	withAutoreleasePool(func() {
		types := generalPasteboard().Send(selTypes)
		if types == 0 {
			return
		}
		for i := 0; i < objc.Send[int](types, selCount); i++ {
			switch typ := nsStringToGo(types.Send(selObjectAtIndex, i)); typ {
			case nsPasteboardTypeString:
				add(MIMEText)
			case nsPasteboardTypePNG, nsPasteboardTypeTIFF:
				add(MIMEPNG)
			case nsPasteboardTypeHTML:
				add("text/html")
			default:
				add(typ)
			}
		}
	})
	return formats, nil
}
