# Rules of nui

A short set of rules the signatures do not show and that must not be broken. For a developer and for an AI assistant writing code on nui: read it before the first change.

## The UI thread

- **All the forms and widgets live in one UI thread**, the main thread of the program, where `main` calls `form.Exec()`. The handlers of events, the signals, the timers and the painting are called in it, one at a time, so a handler of one form may touch the widgets of another freely.
- **Other goroutines do not touch widgets.** They pass work to the UI thread with `ui.Invoke(f)` (returns at once) or `ui.InvokeSync(f)` (waits). After `f` the forms repaint themselves.
- **A repaint from a goroutine** without a change of widgets is asked by `widget.RequestRepaint()`: many requests before a frame give one repaint.
- **Long work** is `ui.RunAsync(owner, work, done)`: `work` in a goroutine with a context of cancellation (cancelled also when the owner's form closes) and reports of progress, `done` in the UI thread. With a dialog of progress, `ui.RunWithProgress`.
- **Commands** are `ui.Action`: one command in a menu, on a toolbar and on a shortcut; whether it is enabled is `SetEnabledFunc`, without updating the items by hand.

## Layout and repaint

- **The repaint is automatic, but not of the whole window.** A frame repaints only the rectangles of the widgets that asked for it (`Widget.Update()`), over the last frame. The whole window is repainted after a click, a key, a character typed, an `Invoke`, a layout that moved a widget without a parent, and `Form().Update()`. After a mouse move, the wheel and a timer only the widgets whose handlers got them are repainted, and the owner of the timer (for a timer of the form's panel that is the whole window). The setters of the widgets (`SetText`, `SetValue`…) ask for the repaint themselves. A widget that paints data of its own in `OnPaint` needs `widget.Update()` after the data change, not `Form().Update()`, which repaints the whole window.
- **A paint draws what the next one would.** A widget that works out its layout while painting (from what it measures) settles it within the paint: a part of the window repainted later draws with that later layout.
- **Debugging the repaint:** `NUI_CHECK_REPAINT=1` compares each frame repainted in part with a whole repaint and writes to stderr where they differ and which widgets lie there. A difference means a widget changed without asking to be repainted. `NUI_CHECK_REPAINT_DIR=dir` saves both pictures.
- **Pixels written by hand.** What writes into `Canvas.RGBA()` itself places the drawing from `PixelOrigin()` and writes only within `ClipRect()`: in a frame that repaints a part of the window the rest of the image holds the pixels of other widgets.
- **The layout is deferred.** `AddWidget`, `RemoveWidget`, `SetVisible`, `SetMinSize`/`SetMaxSize` and a new text of a `Label` ask for it themselves. The form is laid out once: before a frame, before input, or when the geometry is first asked (`X()`, `Width()`, `MinWidth()`…), so a place read right after a change is already right. `Form.UpdateLayout()` is needed only when the minimum size of a widget of one's own changed so that nui did not learn of it.
- **The grid.** `AddWidget(row, column, w)`: the row first. Without expandable widgets the spare room is shared between the rows, so a form has `AddVSpacer(n, 0)` at the bottom and a row has `AddHSpacer` on the right. Expandable is set by `SetXExpandable`/`SetYExpandable`.
- **Several widgets in one cell** lie on top of each other, as in Qt; a click goes to the one added last.
- **One widget, one parent.** `AddWidget` silently does nothing when the widget is already in a form: to move a widget, `RemoveWidget` it first.
- **Sizes are in logical pixels.** On a screen of scale 2 (Retina, 200%) the window and the widgets are drawn twice as dense, and the coordinates and sizes in the code stay the same. The scale comes from the system: the monitor's on Windows, Retina on macOS, `Xft.dpi` / 96 (set by GNOME and KDE) or `GDK_SCALE` on Linux. `NUI_SCALE` (e.g. `NUI_SCALE=1.5`) sets the scale on Linux by hand.
- **`SetEnabled(false)` disables the widget and all that is in it.** `Enabled()` takes the parents into account.

## Events

In detail: [events.md](ui/content/events.md).

- **Signals:** `w.OnSomething.Connect(f)`. Any number of handlers, the data comes as the argument.
- **`On…Changed` fires on every change of the value**, by the code too (`SetText`, `SetChecked`, `SetSelectedIndex`…), and only when the value did change.
- **`OnActivated` (and `OnTextEdited`) fires on an action of the user only.** A handler that sets the value itself does not call itself through them.
- **The signals of input** (`OnKeyDown`, `OnMouseDown`…) come before the widget's own handling; `Handled = true` cancels it.
- **The old `SetOn…`** keep one handler, each call replaces the last. `CurrentEvent()` is deprecated.

## Windows and dialogs

- **`form.Show()` and `ShowModal(parent)` return at once**, except `ShowModal` on macOS, which waits until the dialog closes. Do not write code that reads the result after `ShowModal`: the result of a dialog comes to a handler. Dialogs of one's own are built on `ui.DialogContent` and `ShowDialog` ([dialog.md](ui/content/dialog.md)).
- **A modal window blocks the input of its parent** until it closes.
- **`form.Close()` closes the window and frees its widgets**; `form.OnClose` may keep open a window the user closes (`RequestClose`).
- **A question before data is lost** is asked with `ShowQuestionMessageBoxYesNoCancel`: Esc and the close button in it mean "cancel", not "no".
- **File dialogs** (`form.ShowOpenFileDialog` and the others) give the result to a handler.
- **Standard dialogs:** `ShowInputDialog`, `ShowProgressDialog`, `ShowAboutDialog`, the questions `ShowQuestionMessageBox…`, with the result in a handler too.
- **The place and size of a window between runs** is `form.RememberState(settings, key)`, one copy of the application is `ui.AcquireSingleInstance`, bringing a window to the front is `form.Activate()`.

## Text

- **Positions in a text are in runes**, not bytes: `TextPos{Line, Col}` of `TextBox`, `PosAt`/`OffsetOf` for offsets in `Text()`. A line break is one rune, `"\n"`.
- **`TextBox.SetText` starts the undo history again.** An edit the user can undo is made with `InsertText` (after `SelectAllText`, a replacement of the whole text).
- **`Canvas.DrawText` breaks lines at `"\n"` and `"\r\n"`.**
- **Input through an IME** (Chinese, Japanese, Korean): the window of composition opens at the caret of the TextBox. On Windows it always works; on macOS it is on by default too (`ui.UseMacInputMethod(false)` before the forms are shown turns it off), with the accents by holding a key, the dead keys and the emoji panel. Outside of text boxes the keys go to the form as they are.
- **The font of a widget** is `SetFontFamily` (the built-in `ui.FontFamilySans`, `ui.FontFamilyMono` or the name of a system font), `SetFontSize`, `SetFontBold`, `SetFontItalic`. The family string may carry the style: `"jetbrainsmono:bold"`.

## Timers

- **`w.AddTimer(ms, f)`** repeats, the first time on the next tick. **`w.NewTimer(d, f)`** makes a stopped one, **`w.AfterFunc(d, f)`** fires once. All return a `*ui.Timer` with `Start`/`Stop`.
- **A timer runs while its widget is in an open form.** `Start` again counts the interval anew: a debounce ([timer.md](ui/content/timer.md)).
- **Do not ask with a timer** what a signal tells (the selection, the size, the theme).

## A widget of one's own

- **Embed `ui.Widget`** (or `ui.Panel`, `ui.DialogContent`) and call `InitWidget()` in the constructor. `ui.Widgeter` cannot be implemented otherwise.
- **Painting** is `SetOnPaint(func(cnv *ui.Canvas))`; input, the signals of the base widget or `SetOnMouseDown`/`SetOnKeyDown`.
- **Only the methods of the `Widgeter` interface can be overridden**: `ProcessPaint`, `ProcessKeyDown`, `MinWidth`, `MinHeight`, `XExpandable`, `ProcessPropChange` and the others. nui calls them through the interface. A method of `ui.Widget` that calls another method of `ui.Widget` from inside gets the base version, not yours: Go has no virtual methods.
- **Signals of one's own** are fields of `ui.Signal[T]`, fired with `Emit`.
- **Compare widgets by `Id()`**, not by interface values: a value got from the base `Widget` may carry the type `*ui.Widget`, not yours.

## Tests

- **Forms are tested without a display** by the package `ui/uitest`: clicks, typing, menus, dialogs, golden pictures ([testing.md](testing.md)).
- **In tests the time does not run by itself:** the timers fire on `Tick`/`Wait`, the pending `Invoke` calls on `Settle` and after each action.
