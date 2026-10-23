//go:build darwin

package platforms

import (
	"errors"

	"github.com/ebitengine/purego/objc"
)

// The clipboard is the general NSPasteboard, the text in it of the type
// NSPasteboardTypeString ("public.utf8-plain-text")

var (
	selGeneralPasteboard = objc.RegisterName("generalPasteboard")
	selClearContents     = objc.RegisterName("clearContents")
	selSetStringForType  = objc.RegisterName("setString:forType:")
	selStringForType     = objc.RegisterName("stringForType:")
)

const nsPasteboardTypeString = "public.utf8-plain-text"

func generalPasteboard() objc.ID {
	return objc.ID(objc.GetClass("NSPasteboard")).Send(selGeneralPasteboard)
}

func clipboardSetText(text string) error {
	ok := false
	withAutoreleasePool(func() {
		pb := generalPasteboard()
		pb.Send(selClearContents)
		ok = objc.Send[bool](pb, selSetStringForType, goStringToNS(text), goStringToNS(nsPasteboardTypeString))
	})
	if !ok {
		return errors.New("clipboard: the text was not put on the pasteboard")
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
