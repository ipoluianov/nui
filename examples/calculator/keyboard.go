package calculator

import "github.com/ipoluianov/nui/ui"

// setupKeyboard routes the keys to the engine. The editing keys are taken
// by key code before the focused widget sees them; the digits and the
// operators come as characters, so they work in any keyboard layout and on
// the numeric keypad.
func (c *calculator) setupKeyboard() {
	c.form.SetOnGlobalKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool {
		// Leave the shortcuts and an open menu alone
		if mods.Ctrl || mods.Alt || mods.Cmd || c.form.TopPopupWidget() != nil {
			return false
		}
		e := c.engine
		switch key {
		case ui.KeyEnter:
			e.Equals()
		case ui.KeyBackspace:
			e.Backspace()
		case ui.KeyEsc:
			e.Clear()
		case ui.KeyDelete:
			e.ClearEntry()
		case ui.KeyF9:
			e.Negate()
		default:
			return false
		}
		c.refresh()
		return true
	})

	// A character goes to the focused widget, or to the form's panel when
	// nothing has the focus (the keypad keys never take it)
	c.form.Panel().SetOnChar(c.onChar)
	c.history.SetOnChar(c.onChar)
	c.historySwitch.SetOnChar(c.onChar)
}

func (c *calculator) onChar(ch rune, mods ui.KeyModifiers) bool {
	if mods.Ctrl || mods.Alt || mods.Cmd {
		return false
	}
	e := c.engine
	switch ch {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		e.Digit(int(ch - '0'))
	case '.', ',':
		e.Point()
	case '+':
		e.Operator(OpAdd)
	case '-':
		e.Operator(OpSub)
	case '*', 'x', 'X':
		e.Operator(OpMul)
	case '/', ':':
		e.Operator(OpDiv)
	case '=':
		e.Equals()
	case '%':
		e.Percent()
	// The keys of the Windows calculator
	case 'r', 'R':
		e.Reciprocal()
	case '@':
		e.Sqrt()
	case 'q', 'Q':
		e.Square()
	default:
		return false
	}
	c.refresh()
	return true
}
