# Events: signals

A widget tells what happens to it by **signals**: fields named `On…` of the type `ui.Signal[T]`. A signal takes any number of handlers, and each gets the data of the event as its argument:

```go
table.OnCellDoubleClick.Connect(func(e *ui.TableCellEvent) { open(e.Row) })

conn := search.OnTextChanged.Connect(func(text string) { filter(text) })
conn.Disconnect() // when the handler is no longer needed
```

The old setters `SetOn…` work as before: each keeps one handler, and a new call replaces the last. New code uses the signals.

## Three rules

1. **The signals of input come before the widget handles the input.** Their data is a pointer to the event; a handler that sets `Handled = true` cancels what the widget would do: a letter is not typed, a table does not open the editor of a cell.
2. **`On…Changed` fires on every change of the value**, by the user and by the code (`SetText`, `SetChecked`, `SetSelectedIndex`…). A value that did not change fires nothing.
3. **`OnActivated` (and `OnTextEdited` of TextBox) fires on an action of the user only.** So a handler that sets the value itself does not call itself, and flags like `updating`/`filling` are not needed.

All the signals fire in the UI thread. The handlers are called in the order they were connected. A handler connected while the signal fires is called from the next time; one disconnected while it fires is not called again.

## Signals of every widget

| Signal | Data | When |
|---|---|---|
| `OnMouseDown`, `OnMouseUp`, `OnMouseDoubleClick` | `*MouseEvent{Button, X, Y, Mods, Handled}` | a mouse button pressed, released, double-clicked. The release goes to the widget the button was pressed on, wherever the mouse is (for any button) |
| `OnMouseMove` | `*MouseEvent` | the mouse over the widget, or anywhere while a button pressed on it is down |
| `OnWheel` | `*WheelEvent{DX, DY, X, Y, Mods, Handled}` | the wheel over the widget, with the place of the mouse and the modifiers: for "Ctrl + wheel = zoom around the cursor" |
| `OnKeyDown`, `OnKeyUp` | `*KeyEvent{Key, Mods, Handled}` | a key on the focused widget, before the shortcuts of the form and the widget's own handling |
| `OnChar` | `*CharEvent{Char, Mods, Handled}` | a character typed |
| `OnClick` | `Widgeter` (who) | the left button pressed and released on the widget; a button is also pressed by Space, Enter and `Push` |
| `OnResize` | `Size{Width, Height}` | the layout gave the widget a new size, instead of a timer asking its width |
| `OnFocusChanged` | `bool` | the widget got (`true`) or lost the focus |

The coordinates of the mouse events are those of the widget's content, with its scrolling, as in its own handlers.

## Signals of the widgets

| Widget | Any change | Action of the user | Other |
|---|---|---|---|
| TextBox | `OnTextChanged(string)` | `OnTextEdited(string)` | `OnSubmit(string)`: Enter in a single-line box (while a handler is connected, Enter does not reach the default button of the form); `OnEditingFinished(string)`: Enter, or leaving the box after an edit |
| ComboBox | `OnSelectionChanged(int)` | `OnActivated(int)`: a choice of the user, even of the same item | - |
| Checkbox, ToggleSwitch | `OnCheckedChanged(bool)` | `OnActivated(bool)` | - |
| RadioButton | `OnCheckedChanged(bool)`, also when another button of the group took the check | `OnActivated(bool)` | - |
| NumBox, Slider | `OnValueChanged(float64)` | `OnActivated(float64)` | - |
| Table | `OnSelectionChanged(TableCell)`, `OnCellChanged(TableCell)` | `OnCellEdited(TableCell)` | `OnCellDoubleClick(*TableCellEvent)` (`Handled` cancels the editing), `OnColumnClick(int)` |
| TreeView | `OnSelectionChanged(*TreeNode)` | `OnNodeActivated(*TreeNode)`, `OnCellEdited(TreeCell)` | `OnExpanding(*TreeNode)`: before a node opens, the place to load its children; `OnCollapsed(*TreeNode)`; `OnColumnClick(int)` |

A signal of the application: `ui.OnThemeChanged(bool dark)`, the theme applied. The widgets repaint themselves; the handler recolors what the application colored itself, e.g. icons.

## Instead of `CurrentEvent()`

`ui.CurrentEvent()`, `PushEvent`, `PopEvent`, `TextBox.SetOnTextBoxKeyDown` and `Table.SetOnCellMouseDblClick` are deprecated.

| Was | Is |
|---|---|
| `tb.SetOnTextBoxKeyDown(func() { ev := ui.CurrentEvent().Parameter.(*ui.EventTextboxKeyDown); if ev.Key == ui.KeyEnter { …; ev.Processed = true } })` | `tb.OnKeyDown.Connect(func(e *ui.KeyEvent) { if e.Key == ui.KeyEnter { …; e.Handled = true } })`, or just `tb.OnSubmit.Connect(func(text string) { … })` |
| `table.SetOnCellMouseDblClick(func() { ev, _ := ui.CurrentEvent().Parameter.(*ui.EventTableCellMouseDblClick); open(ev.Row) })` | `table.OnCellDoubleClick.Connect(func(e *ui.TableCellEvent) { open(e.Row) })` |
| a timer asking the selection of a table, because its `SetOnSelectionChanged` is taken by its owner | `table.OnSelectionChanged.Connect(…)`: any number of handlers |
| a list of `themeListeners` of one's own | `ui.OnThemeChanged.Connect(…)` |
| a timer asking the width of a widget | `w.OnResize.Connect(…)` |
| the place of the mouse kept in `OnMouseMove`, and `Form().KeyModifiers()` in a handler of the wheel | `w.OnWheel.Connect(func(e *ui.WheelEvent) { zoomAt(e.X, e.Y, e.Mods.Ctrl) })` |

## A widget of one's own

A widget that embeds `ui.Widget` gets all the signals of the base widget. Its own signals are fields, fired with `Emit`:

```go
type ColorWell struct {
	ui.Widget
	OnColorChanged ui.Signal[color.RGBA] // any change
	OnActivated    ui.Signal[color.RGBA] // a choice of the user
}
```
