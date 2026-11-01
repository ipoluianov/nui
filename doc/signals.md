# Signals

How a widget tells the application what happens to it, and how to use that well. Which widget has which signal is in [events.md](ui/content/events.md); this page is about using them.

- [A signal](#a-signal)
- [Connect and disconnect](#connect-and-disconnect)
- [Changed and Activated](#changed-and-activated)
- [Input signals and Handled](#input-signals-and-handled)
- [Signals of your own](#signals-of-your-own)
- [How long a handler lives](#how-long-a-handler-lives)
- [Threads](#threads)
- [Patterns](#patterns)
- [The old setters](#the-old-setters)
- [Tests](#tests)

## A signal

A signal is a field of a widget named `On…`, of the type `ui.Signal[T]`; `T` is what it tells:

```go
type TextBox struct {
	...
	OnTextChanged Signal[string]
	OnSubmit      Signal[string]
}
```

A signal takes any number of handlers. When the event happens the widget emits the signal, and each handler is called with the data, in the order they were connected, right there: inside the setter or the input handling that caused it.

## Connect and disconnect

```go
conn := search.OnTextChanged.Connect(func(text string) { filter(text) })
...
conn.Disconnect() // a second call does nothing
```

Most handlers are never disconnected: they live as long as their form (see [How long a handler lives](#how-long-a-handler-lives)), and the `Connection` can be ignored.

While a signal is being emitted, a handler may connect or disconnect others, itself too: one connected meanwhile is called from the next emission on, one disconnected meanwhile is not called any more.

`Connected()` tells whether a signal has any handler, for a widget of your own that would compute its data only for someone.

## Changed and Activated

The value signals keep one rule:

| Signal | Fires on |
|---|---|
| `On…Changed` (`OnTextChanged`, `OnCheckedChanged`, `OnValueChanged`, `OnDateChanged`…) | every change of the value: the user's and the code's (`SetText`, `SetChecked`, `SetValue`…), only when it did change |
| `OnActivated` (and `OnTextEdited` of TextBox) | a change the user made |

Which to use:

- **To keep something in step with the value** (a preview, a label, an enabled state): `On…Changed`. It also fires when the code loads a document into the widgets, so the dependents follow.
- **To react to the user** (save a setting, mark a document modified, run a search the user asked for): `OnActivated`, `OnTextEdited`. Setting the value from the code doesn't fire them, so no handler calls itself and no `updating` flag is needed:

```go
// The document is modified by the user's edits, not by loading it
text.OnTextEdited.Connect(func(string) { doc.SetModified(true) })
text.SetText(loaded) // doesn't mark it modified

// Two widgets showing one value: each sets the other, no loop
slider.OnActivated.Connect(func(v float64) { box.SetValue(v) })
box.OnActivated.Connect(func(v float64) { slider.SetValue(v) })
```

Lists and calendars: `OnActivated` is a double click or Enter on an item (open it); the selection is `OnSelectionChanged` or `OnDateChanged`.

## Input signals and Handled

Every widget has the signals of the mouse and the keyboard: `OnMouseDown`, `OnMouseUp`, `OnMouseDoubleClick`, `OnMouseMove`, `OnWheel`, `OnKeyDown`, `OnKeyUp`, `OnChar`. They carry a pointer to the event and come **before** the widget does its own work; a handler that sets `Handled` stops it:

```go
// Only digits in the box
code.OnChar.Connect(func(e *ui.CharEvent) {
	if e.Char < '0' || e.Char > '9' {
		e.Handled = true
	}
})

// Ctrl + wheel zooms around the mouse instead of scrolling
view.OnWheel.Connect(func(e *ui.WheelEvent) {
	if e.Mods.Ctrl {
		zoomAt(e.X, e.Y, e.DY)
		e.Handled = true
	}
})
```

A key on the focused widget reaches its `OnKeyDown` before the shortcuts of the form; a handled key goes no further. The coordinates of the mouse are those of the widget's content, scrolled.

`OnClick` fires when the left button is pressed and released on the widget; a Button and a Link fire it for Enter, Space and `Push`/`Activate` too. `OnResize` tells the new size from the layout, `OnFocusChanged` the focus.

## Signals of your own

A widget of your own embeds `ui.Widget` and has all of its signals; its own are fields, emitted with `Emit`. Keep the rule of the value signals:

```go
type ColorWell struct {
	ui.Widget
	color color.RGBA
	// OnColorChanged: any change of the color
	OnColorChanged ui.Signal[color.RGBA]
	// OnActivated: the user chose a color
	OnActivated ui.Signal[color.RGBA]
}

func (c *ColorWell) SetColor(col color.RGBA) {
	if col == c.color {
		return // the same value fires nothing
	}
	c.color = col
	c.Update() // repaint this widget
	c.OnColorChanged.Emit(col)
}

// pick is the user's choice, from the mouse or the keyboard
func (c *ColorWell) pick(col color.RGBA) {
	c.SetColor(col)
	c.OnActivated.Emit(col)
}
```

Emit after the state is changed and the repaint is asked for: a handler may read the widget, or change it again. Any type can carry the data; a pointer to an event struct with a `Handled` field lets handlers cancel what follows.

## How long a handler lives

A handler keeps what its function refers to - usually widgets of its form - for as long as it is connected.

- **A signal of the form's own widgets** (a button, a box, a table of the form): the handler and the widgets refer to each other and go together when the form is closed and dropped. No disconnecting is needed, and this is nearly all handlers.
- **A signal that lives longer than the form**: `ui.OnThemeChanged`, a model shared by windows, a widget of another window, an object of the application. Connected with `Connect`, the handler would keep the closed form in memory and keep being called. Connect it with `ConnectWhileShown`, giving a widget of the form:

```go
ui.OnThemeChanged.ConnectWhileShown(form.Panel(), func(dark bool) {
	saveAction.SetIcon(saveIcon(dark))
})
hosts.ModelChanged().ConnectWhileShown(status, func(struct{}) { status.SetText(summary()) })
```

The handler is connected while the widget is in an open form: undone when the form closes or the widget is taken out of it, made again when the form is shown again. `Disconnect` on the returned connection removes it for good.

The library does the same for what it connects itself: a Table or ListView follows its model, menu items and toolbar buttons follow their `Action`, a chart follows its memory source, only while shown. A closed form is freed though the model, the action or the source live on; shown again, the views are brought up to date.

## Threads

Signals, like the widgets, are of the UI thread: `Connect`, `Disconnect` and `Emit` are called there, and every handler runs there. A goroutine does not emit a signal; it hands the work over:

```go
go func() {
	result := compute()
	ui.Invoke(func() { view.SetResult(result) }) // fires its signals on the UI thread
}()
```

## Patterns

**Wait until the typing pauses** before an expensive reaction: a single-shot timer started again on each change.

```go
debounce := panel.NewTimer(300*time.Millisecond, runSearch)
debounce.SetSingleShot(true)
search.OnTextChanged.Connect(func(string) { debounce.Start() })
```

**One handler for many widgets**: the data of `OnClick` is the widget clicked.

```go
for _, b := range []*ui.Button{red, green, blue} {
	b.OnClick.Connect(func(w ui.Widgeter) { choose(w.(*ui.Button).Text()) })
}
```

**A handler for a while**: keep the connection, disconnect when done.

```go
var conn ui.Connection
conn = table.OnSelectionChanged.Connect(func(ui.TableCell) {
	showHint()
	conn.Disconnect() // once
})
```

**Several listeners of one widget**: a table that its own form watches and a side panel watches too; each connects its handler, neither replaces the other's.

## The old setters

Before signals the widgets had setters: `SetOnClick`, `SetOnTextChanged`, `SetOnSelectionChanged`, `SetOnDateChanged`… They still work as they did: each keeps one handler, a new call replaces it, and some of them are called on the user's changes only (see the page of the widget). A signal and the setter of the same event are independent; both are called.

New code uses the signals. Replacing an old handler:

| Was | Is |
|---|---|
| `b.SetOnClick(f)` | `b.OnClick.Connect(func(ui.Widgeter) { f() })` |
| `w.SetOnMouseEnter(f)`, `w.SetOnMouseLeave(g)` | `w.OnHoverChanged.Connect(func(in bool) { … })` |
| `form.SetOnGlobalKeyDown(f)` | `form.OnGlobalKeyDown.Connect(func(e *ui.KeyEvent) { e.Handled = f(e.Key, e.Mods) })` |
| `form.SetOnLanguageChanged(f)` | `form.OnLanguageChanged.Connect(func(string) { f() })` |
| `menu.SetOnShow(f)` | `menu.OnShow.Connect(func(*ui.ContextMenu) { f() })` |
| `sp.SetOnSplitChanged(f)` | `sp.OnActivated.Connect(func(size int) { … })` |
| `tray.SetOnClick(f)` | `tray.OnClick.Connect(func(*ui.TrayIcon) { f() })` |
| `box.SetOnTextChanged(f)` | `box.OnTextChanged.Connect(f)` |
| `ui.CurrentEvent()` in a handler | the data of the signal: `OnKeyDown.Connect(func(e *ui.KeyEvent) {…})` |
| a timer polling a selection or a width | `OnSelectionChanged`, `OnResize` |

## Tests

In a test of `ui/uitest`, count the emissions and act as the user:

```go
changed, activated := 0, 0
box.OnCheckedChanged.Connect(func(bool) { changed++ })
box.OnActivated.Connect(func(bool) { activated++ })

box.SetChecked(true) // the code: changed only
w.Click(box)         // the user: both
if changed != 2 || activated != 1 {
	t.Errorf("changed %d, activated %d", changed, activated)
}
```

See [testing.md](testing.md).
