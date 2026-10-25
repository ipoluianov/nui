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
//     to nui as characters and empties the view. The composition itself
//     shows in the input method's window, not in nui's text box.

var (
	selSetAlphaValue      = objc.RegisterName("setAlphaValue:")
	selSetFrame           = objc.RegisterName("setFrame:")
	selAddSubview         = objc.RegisterName("addSubview:")
	selMakeFirstResponder = objc.RegisterName("makeFirstResponder:")
	selHasMarkedText      = objc.RegisterName("hasMarkedText")
	selString             = objc.RegisterName("string")
	selSetString          = objc.RegisterName("setString:")
	selSetRichText        = objc.RegisterName("setRichText:")
	selSetDrawsBackground = objc.RegisterName("setDrawsBackground:")
	selTextDidChange      = objc.RegisterName("textDidChange:")
	selSuperview          = objc.RegisterName("superview")
)

var (
	inputMethodEnabled bool

	textInputClassOnce sync.Once
	textInputClass     objc.Class
	textInputDelegate  objc.ID
	// textInputViews are the hidden text views by window
	textInputViews = map[int]objc.ID{}
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
	tv.Send(selSetDrawsBackground, false)
	tv.Send(selSetAlphaValue, 0.0)
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
	if go_on_key_down(id, int(objc.Send[uint16](event, selKeyCode))) {
		return
	}
	flags := objc.Send[uint64](event, selModifierFlags)
	if flags&(nsEventModifierFlagCommand|nsEventModifierFlagControl) != 0 {
		return
	}
	self.SendSuper(cmd, event)
}

// nuiTextInputDidChange hands the text committed to nui
func nuiTextInputDidChange(self objc.ID, _ objc.SEL, notification objc.ID) {
	tv := objc.Send[objc.ID](notification, selObject)
	if tv == 0 || objc.Send[bool](tv, selHasMarkedText) {
		return
	}
	text := nsStringToGo(objc.Send[objc.ID](tv, selString))
	if text == "" {
		return
	}
	tv.Send(selSetString, goStringToNS(""))
	id := wndID(objc.Send[objc.ID](tv, selWindow))
	for _, r := range text {
		if r == '\n' || r == '\r' || r == '\t' || utf16.IsSurrogate(r) {
			continue // the keys of these went to nui
		}
		go_on_char(id, int(r))
	}
}

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
	delete(textInputViews, id)
}
