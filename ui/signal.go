package ui

// Signals. A widget tells what happens to it with signals, fields named
// On...: any number of handlers can be connected to one, each gets the data
// of the event as the argument.
//
//	table.OnCellDoubleClick.Connect(func(e *ui.TableCellEvent) { open(e.Row) })
//	conn := box.OnTextChanged.Connect(func(text string) { search(text) })
//	conn.Disconnect() // later
//
// The input signals (OnMouseDown, OnKeyDown...) come before the widget does
// its own work and carry a pointer to the event: a handler that sets
// Handled stops the widget's own handling, e.g. a key the text box would
// type.
//
// The rule of the value signals: On...Changed is emitted on every change of
// the value, by the user or by the code (SetText, SetChecked...);
// OnActivated (and the like: OnTextEdited) only on a change the user made,
// so a handler that sets the value itself doesn't call itself.
//
// A handler keeps what it refers to while connected. Handlers of a form
// connected to its own widgets go with the form; to a signal that lives
// longer, a form connects with ConnectWhileShown.
//
// The older SetOn... setters still work and keep their own behaviour: each
// sets one handler, a new one replaces it.

// Signal is an event handlers connect to; T is the data it carries. The
// zero value is ready to use. On the UI thread only, as the widgets.
type Signal[T any] struct {
	handlers []*signalHandler[T]
}

type signalHandler[T any] struct {
	f       func(T)
	removed bool
}

// Connection is a handler connected to a signal; Disconnect removes it
type Connection struct {
	disconnect func()
}

// Disconnect removes the handler from the signal; a second call does nothing
func (c Connection) Disconnect() {
	if c.disconnect != nil {
		c.disconnect()
	}
}

// Connect adds a handler, called after those connected before
func (s *Signal[T]) Connect(f func(T)) Connection {
	if f == nil {
		return Connection{}
	}
	h := &signalHandler[T]{f: f}
	s.handlers = append(s.handlers, h)
	return Connection{disconnect: func() {
		h.removed = true
		h.f = nil // what the handler refers to is let go at once
		for i, o := range s.handlers {
			if o == h {
				// A new array: an Emit in progress keeps its own, and the
				// old one, still holding h, isn't kept by an empty slice
				var handlers []*signalHandler[T]
				if len(s.handlers) > 1 {
					handlers = make([]*signalHandler[T], 0, len(s.handlers)-1)
					handlers = append(append(handlers, s.handlers[:i]...), s.handlers[i+1:]...)
				}
				s.handlers = handlers
				return
			}
		}
	}}
}

// ConnectWhileShown connects f while the widget w is in an open form: from
// now, or when w gets into one; undone when the form closes or w leaves it,
// made again when the form is shown again. For a signal that outlives the
// form - ui.OnThemeChanged, a model's, another window's - so the closed
// form is neither kept in memory nor called. Disconnect removes it for good.
func (s *Signal[T]) ConnectWhileShown(w Widgeter, f func(T)) Connection {
	if f == nil || w == nil {
		return Connection{}
	}
	remove := w.widgetBase().linkWhileShown(func() func() { return s.Connect(f).Disconnect })
	return Connection{disconnect: remove}
}

// Emit calls the handlers with v, in the order they were connected. A
// handler connected meanwhile is called from the next Emit on; one
// disconnected meanwhile is not called any more.
func (s *Signal[T]) Emit(v T) {
	for _, h := range s.handlers {
		if !h.removed {
			h.f(v)
		}
	}
}

// Connected reports whether any handler is connected
func (s *Signal[T]) Connected() bool {
	return len(s.handlers) > 0
}

// The data of the input signals

// MouseEvent is a mouse button press, release, double click or move, in the
// coordinates of the widget (its content, scrolled)
type MouseEvent struct {
	Button MouseButton // none for a move
	X, Y   int
	Mods   KeyModifiers
	// Handled set by a handler stops the widget's own handling
	Handled bool
}

// WheelEvent is a turn of the mouse wheel over the widget: DY > 0 scrolls up
// (away from the user), DX > 0 to the left, in lines; the mouse is at X, Y
type WheelEvent struct {
	DX, DY  int
	X, Y    int
	Mods    KeyModifiers
	Handled bool
}

// KeyEvent is a key press or release on the focused widget
type KeyEvent struct {
	Key     Key
	Mods    KeyModifiers
	Handled bool
}

// CharEvent is a character typed on the focused widget
type CharEvent struct {
	Char    rune
	Mods    KeyModifiers
	Handled bool
}

// Size is the size a widget got
type Size struct {
	Width, Height int
}
