//go:build darwin

package platforms

import (
	"sync"
	"unicode/utf16"

	"github.com/ebitengine/purego/objc"
)

// The input method of macOS (Chinese, Japanese, Korean; also the accents
// held on a key) with SetInputMethodEnabled(true): each window gets a
// hidden NSTextView, which implements NSTextInputClient itself (its methods
// return NSRange and NSRect structs, which callbacks of purego can't).
// The text view is the first responder, at the caret (SetTextInputRect):
//
//   - a key goes to nui first, as without the input method (shortcuts, the
//     arrows, Backspace); the keys nui leaves, and all the keys while a
//     composition goes on, go to the text view and so to the input method;
//   - the text committed lands in the text view: textDidChange: hands it
//     to nui as characters. The composition itself shows in the input
//     method's window, not in nui's text box.
//   - the last character stays in the text view: a key held shows the
//     accents of the character before the caret (e -> é è ê), and the one
//     picked replaces it there. A replacement goes to nui as Backspace and
//     the new character. The view is emptied when nui's caret may have
//     moved: a key nui took, a click.

var (
	selSetTextColor           = objc.RegisterName("setTextColor:")
	selSetInsertionPointColor = objc.RegisterName("setInsertionPointColor:")
	selClearColor             = objc.RegisterName("clearColor")
	selOrderFrontCharPalette  = objc.RegisterName("orderFrontCharacterPalette:")
	selSetFrame               = objc.RegisterName("setFrame:")
	selAddSubview             = objc.RegisterName("addSubview:")
	selMakeFirstResponder     = objc.RegisterName("makeFirstResponder:")
	selHasMarkedText          = objc.RegisterName("hasMarkedText")
	selString                 = objc.RegisterName("string")
	selSetString              = objc.RegisterName("setString:")
	selSetRichText            = objc.RegisterName("setRichText:")
	selSetDrawsBackground     = objc.RegisterName("setDrawsBackground:")
	selTextDidChange          = objc.RegisterName("textDidChange:")
	selSuperview              = objc.RegisterName("superview")
)

var (
	inputMethodEnabled bool

	textInputClassOnce sync.Once
	textInputClass     objc.Class
	textInputDelegate  objc.ID
	// textInputViews are the hidden text views by window
	textInputViews = map[int]objc.ID{}
	// textInputKept is the text left in each text view (see above)
	textInputKept = map[objc.ID][]rune{}
)

func setInputMethodEnabled(enabled bool) {
	inputMethodEnabled = enabled
}

func registerTextInputClass() {
	var err error
	textInputClass, err = objc.RegisterClass(
		"NUITextInputView",
		objc.GetClass("NSTextView"),
		nil, nil,
		[]objc.MethodDef{
			{Cmd: selKeyDown, Fn: nuiTextInputKeyDown},
			{Cmd: selKeyUp, Fn: nuiViewKeyUp},
			{Cmd: selFlagsChanged, Fn: nuiViewFlagsChanged},
		},
	)
	if err != nil {
		panic(err)
	}
	delegateClass, err := objc.RegisterClass(
		"NUITextInputDelegate",
		objc.GetClass("NSObject"),
		nil, nil,
		[]objc.MethodDef{{Cmd: selTextDidChange, Fn: nuiTextInputDidChange}},
	)
	if err != nil {
		panic(err)
	}
	textInputDelegate = objc.ID(delegateClass).Send(selAlloc).Send(selInit)
}

// installTextInput gives the window its hidden text view, the first
// responder; called when the window is made, if the input method is on
func installTextInput(win, view objc.ID) {
	if !inputMethodEnabled {
		return
	}
	textInputClassOnce.Do(registerTextInputClass)
	tv := objc.ID(textInputClass).Send(selAlloc).Send(selInitWithFrame, nsRect{Size: nsSize{1, 20}})
	tv.Send(selSetRichText, false)
	// Seen but drawn in no color: a transparent view (alpha 0) counts as
	// hidden, and the input method shows no accents of a held key for it
	clear := objc.ID(objc.GetClass("NSColor")).Send(selClearColor)
	tv.Send(selSetDrawsBackground, false)
	tv.Send(selSetTextColor, clear)
	tv.Send(selSetInsertionPointColor, clear)
	tv.Send(selSetDelegate, textInputDelegate)
	view.Send(selAddSubview, tv)
	tv.Send(selRelease) // the view keeps it
	win.Send(selMakeFirstResponder, tv)
	textInputViews[int(objc.Send[int](win, selWindowNumber))] = tv
}

// nuiTextInputKeyDown: nui first, then the input method
func nuiTextInputKeyDown(self objc.ID, cmd objc.SEL, event objc.ID) {
	if objc.Send[bool](self, selHasMarkedText) {
		self.SendSuper(cmd, event)
		return
	}
	id := wndID(objc.Send[objc.ID](self, selWindow))
	syncEventModifiers(id, event)
	if isEmojiKey(event) {
		// Emoji & Symbols, as AppKit's own item of the Edit menu: the
		// character picked comes to the text view (textDidChange)
		objc.ID(clsNSApplication).Send(selSharedApplication).Send(selOrderFrontCharPalette, objc.ID(0))
		return
	}
	if go_on_key_down(id, int(objc.Send[uint16](event, selKeyCode))) {
		emptyTextInput(self)
		return
	}
	flags := objc.Send[uint64](event, selModifierFlags)
	if flags&(nsEventModifierFlagCommand|nsEventModifierFlagControl) != 0 {
		emptyTextInput(self)
		return
	}
	self.SendSuper(cmd, event)
}

// macKeyCodeSpace is the key code of Space (kVK_Space)
const macKeyCodeSpace = 0x31

// isEmojiKey tells whether the event is Ctrl+Cmd+Space, the key of Emoji &
// Symbols. AppKit adds that item, with the key, to an Edit menu of the
// main menu only: nui's forms mostly have none there.
func isEmojiKey(event objc.ID) bool {
	const mods = nsEventModifierFlagCommand | nsEventModifierFlagControl | nsEventModifierFlagShift | nsEventModifierFlagOption
	flags := objc.Send[uint64](event, selModifierFlags) & mods
	return objc.Send[uint16](event, selKeyCode) == macKeyCodeSpace &&
		flags == nsEventModifierFlagCommand|nsEventModifierFlagControl
}

// emptyTextInput forgets the text kept in the text view: nui's caret may
// be elsewhere now
func emptyTextInput(tv objc.ID) {
	if len(textInputKept[tv]) == 0 {
		return
	}
	delete(textInputKept, tv)
	tv.Send(selSetString, goStringToNS(""))
}

// emptyTextInputOf empties the text view of the window (see emptyTextInput)
func emptyTextInputOf(id windowId) {
	if tv, ok := textInputViews[int(id)]; ok {
		emptyTextInput(tv)
	}
}

// nuiTextInputDidChange hands the text committed to nui
func nuiTextInputDidChange(self objc.ID, _ objc.SEL, notification objc.ID) {
	tv := objc.Send[objc.ID](notification, selObject)
	if tv == 0 || objc.Send[bool](tv, selHasMarkedText) {
		return
	}
	text := []rune(nsStringToGo(objc.Send[objc.ID](tv, selString)))
	kept := textInputKept[tv]
	same := 0
	for same < len(kept) && same < len(text) && kept[same] == text[same] {
		same++
	}
	id := wndID(objc.Send[objc.ID](tv, selWindow))
	// Replaced by the input method (an accent picked for the held key)
	for range kept[same:] {
		go_on_key_down(id, macKeyCodeDelete)
		go_on_key_up(id, macKeyCodeDelete)
	}
	for _, r := range text[same:] {
		if r == '\n' || r == '\r' || r == '\t' || utf16.IsSurrogate(r) {
			continue // the keys of these went to nui
		}
		go_on_char(id, int(r))
	}
	// The last character stays for the accents of a held key
	if len(text) > 1 {
		text = text[len(text)-1:]
		tv.Send(selSetString, goStringToNS(string(text)))
	}
	textInputKept[tv] = text
}

// macKeyCodeDelete is the key code of Backspace (kVK_Delete)
const macKeyCodeDelete = 0x33

// moveTextInput puts the hidden text view of the window at the caret: the
// input method shows its window by it
func (c *nativeWindow) moveTextInput() {
	tv, ok := textInputViews[int(c.hwnd)]
	if !ok {
		return
	}
	view := objc.Send[objc.ID](tv, selSuperview)
	if view == 0 {
		return
	}
	r := c.platform.textInputRect
	height := objc.Send[nsRect](view, selBounds).Size.Height
	// The view isn't flipped: y goes up from the bottom
	frame := nsRect{nsPoint{float64(r[0]), height - float64(r[1]+r[3])}, nsSize{1, float64(max(r[3], 1))}}
	tv.Send(selSetFrame, frame)
}

// forgetTextInput drops the text view of a closed window
func forgetTextInput(id int) {
	delete(textInputKept, textInputViews[id])
	delete(textInputViews, id)
}
