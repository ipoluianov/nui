# nui core runtime audit: findings

Scope: ui/signal.go, uithread.go, async.go, timer.go, form*.go, widget.go (grid layout), boxes.go, splitter.go, dock.go, tabwidget*.go, dialog.go, messagebox.go, toast.go, tooltip.go, action.go, context_menu*.go, menu_bar.go.

Each finding has a reproducer, and every one of them was run. The reproducer files are in
`doc/audit/_repro/core/`
(`zz_audit_core*_test.go`, package `ui`, internal tests). To run one, copy the files into `ui/` and run
`go test -run '^TestName$' ./ui/ -count=1 -timeout 60s`. Run them one at a time: several of them panic or hang on purpose. The headless tests call `platforms.SetHeadless()` themselves.
Helpers: `newTestForm` is from controls_test.go. `auditHeadless(t)`/`auditWnd(f)` (core3) show a form in headless mode and return its `*platforms.HeadlessWindow`. `auditRegistrySize()` (core1) is `len(allwidgets)`. `auditWithin(t, d, f)` (core6) runs f and fails if it panics or does not return within d.

---

## 1. Grid layout: infinite loop when a column/row with a larger stretch hits its max size
- Severity: **high** (the UI thread hangs forever, with ordinary API use)
- File: ui/widget.go:2755-2767 (makeColumnsInfo), the same at ui/widget.go:3000-3012 (makeRowsInfo)
- Root cause: in the distribution loop, `weights` adds up only the stretch of the columns that *can still change*. But `pix := deltaWidth * colInfo.stretch / weights` is then applied to *every* non-collapsed column, including one already at its maxWidth. Take a column at its max with stretch >= weights. It gets `pix != 0`, so `moved = true`, and then it is clamped back to its max. The changeable columns get 0 from the integer division. The state never changes and the loop never ends.
- Reproducer (core6 `TestAuditStretchLoop`; it hangs at stretch 2, width 107):
```go
form := NewForm()
form.processResize(107, 300)
a := NewPanel(); a.SetXExpandable(true); a.SetMaxWidth(40)
form.Panel().AddWidget(0, 0, a)
form.Panel().SetColumnStretch(0, 2)
for i := 1; i <= 2; i++ { p := NewPanel(); p.SetXExpandable(true); form.Panel().AddWidget(0, i, p) }
form.layoutNow() // never returns
```
- Actual: `zz_audit_core6_test.go:64: did not return within 3s` (stretch 2 width 107)
- Expected: the layout ends. The rounding remainder goes to some column, or the loop stops when nothing changed.
- Fix: share only among the changeable columns (the ones counted in `weights`). Set `moved` only when the width actually changed after the clamp. Or give the remainder `deltaWidth - sum(pix)` to the first changeable column. Apply the same fix to rows.

## 2. Grid layout: integer divide-by-zero / overflow with large stretch weights
- Severity: low
- File: ui/widget.go:2759 (and 3004); SetColumnStretch/SetRowStretch at ui/widget.go:3479/3488 accept any int
- Root cause: `weights += colInfo.stretch` overflows. 4 columns with stretch 1<<62 add up to exactly 0, so `deltaWidth*stretch/weights` panics. Even when the sum does not wrap to 0, `deltaWidth*stretch` overflows and gives garbage widths.
- Reproducer (core6 `TestAuditHugeStretch`): 4 labels in row 0, `SetColumnStretch(i, 1<<62)` for each, then `form.layoutNow()`
- Actual: `panic: runtime error: integer divide by zero`
- Fix: clamp the weight in SetColumnStretch/SetRowStretch to a sane maximum (e.g. 1<<16).

## 3. Grid layout: hang with a huge row/column span
- Severity: medium (a span can come from a layout file: malicious or corrupt input)
- File: ui/widget.go:2675 and 2920 (`for line := first; line < first+span; line++`), ui/widget.go:2491 (extent)
- Root cause: occupiedLines was changed so the layout does not walk over empty lines, but span handling still iterates over every line number in `[first, first+span)`. The span comes from SetGridSpan/AddWidgetSpan and is never capped. `first+span` can also overflow.
- Reproducer (core6 `TestAuditHugeSpan`):
```go
form := NewForm(); form.processResize(500, 400)
form.Panel().AddWidgetSpan(0, 0, 1, 1<<40, NewLabel("x"))
form.Panel().AddWidget(0, 1, NewLabel("y"))
form.layoutNow() // hangs
```
- Actual: `did not return within 10s`
- Fix: iterate over the sorted occupied lines (`lines.cols`/`lines.rows`, or `sortedKeys(columnsInfo)`) that fall within `[first, first+span)`. Clamp the span so `first+span` cannot overflow.

## 4. processKeyDown: nil dereference when the focused widget's key handler removes it
- Severity: medium (panic on the UI thread)
- File: ui/form.go:1489
- Root cause: after `c.focusedWidget.ProcessKeyDown(...)` returns false, the code calls `c.focusedWidget.ParentWidgetId()` again. If the handler removed the widget (RemoveWidget → forgetWidgets sets focusedWidget = nil), this is a nil interface call.
- Reproducer (core1 `TestAuditKeyDownRemovesFocused`):
```go
p := NewPanel(); p.SetCanBeFocused(true); p.SetMinSize(50, 50)
form := newTestForm(t, p); p.Focus()
p.SetOnKeyDown(func(Key, KeyModifiers) bool { form.Panel().RemoveWidget(p); return false })
form.processKeyDown(KeyA, KeyModifiers{})
```
- Actual: `panic: runtime error: invalid memory address or nil pointer dereference ... (*Form).processKeyDown form.go:1489`
- Fix: keep `w := c.focusedWidget` in a local before calling ProcessKeyDown. Use it for the parent walk, or return when `c.focusedWidget` became nil.

## 5. processMouseMove: nil dereference when the drag handler opens a modal dialog or removes the pressed widget
- Severity: **high** (a realistic pattern: a drag handler shows a message box, and that crashes the app)
- File: ui/form.go:1281-1284
- Root cause: `c.mouseLeftButtonPressedWidget.ProcessMouseMove(...)` can clear `c.mouseLeftButtonPressedWidget`. ShowModal → giveMouseToModal → cancelMousePress → processMouseUp sets it to nil. So do RemoveWidget → forgetWidgets. The next line then calls `.widgetBase().Update()` on a nil interface.
- Reproducer A (core9 `TestAuditModalFromDragMove`, headless, through the real window path):
```go
f := auditHeadless(t)
p := NewPanel(); p.SetMinSize(200, 200); f.Panel().AddWidget(0, 0, p); f.UpdateLayout()
w := auditWnd(f)
w.MouseDown(MouseButtonLeft, 20, 20)
shown := false
p.SetOnMouseMove(func(x, y int, m KeyModifiers) bool {
	if !shown { shown = true; ShowMessageBox(p, "Drag", "Can't drop here") }
	return true
})
w.MouseMove(30, 30)
```
- Reproducer B (core1 `TestAuditMouseMoveRemovesPressed`): `SetOnMouseMove` that calls `form.Panel().RemoveWidget(p)`, then `processMouseDown(Left,10,10)` and `processMouseMove(20,20)`.
- Actual: `panic: runtime error: invalid memory address or nil pointer dereference ... (*Form).processMouseMove form.go:1284`
- Fix: `pw := c.mouseLeftButtonPressedWidget` into a local, then call `pw.ProcessMouseMove` and `pw.widgetBase().Update()`.

## 6. Widget.ProcessMouseDown / ProcessMouseDblClick: index out of range when a handler removes siblings
- Severity: medium
- File: ui/widget.go:1891-1892 (ProcessMouseDown), the same loop at ui/widget.go:2113-2114 (ProcessMouseDblClick)
- Root cause: the loop runs `for i := len(c.widgets)-1; i >= 0; i--` and reads `c.widgets[i]` live. A child's handler that removes two or more children shrinks the slice, so the next `c.widgets[i]` is out of range. This needs overlapping widgets, which rules.md documents ("Several widgets in one cell lie on top of each other").
- Reproducer (core1 `TestAuditMouseDownRemovesSiblings`): panels a, b, c all at (0,0), min 100x100. c's `SetOnMouseDown` removes a and b and returns false. Then `form.processMouseDown(Left, 10, 10)`.
- Actual: `panic: runtime error: index out of range [1] with length 1 ... (*Widget).ProcessMouseDown widget.go:1892`
- Fix: iterate over a snapshot (`slices.Clone(c.widgets)`), and skip widgets whose parent is no longer c.

## 7. ShowModal / ShowMaximized on an already open form create a second native window (leak; the parent stays blocked forever)
- Severity: **high** (after the dialog closes, the parent window gets no input ever again)
- File: ui/form.go:813-830 (ShowModal), 858-865 (ShowMaximized); createWindow at 696 sets `c.wnd = nil` without closing the old window
- Root cause: unlike Show, ShowModal and ShowMaximized do not check `c.wnd != nil && !c.closed`. createWindow overwrites c.wnd. The old native window stays open with its callbacks bound to the form, and its modal disable/counter on the parent is never undone.
- Reproducer (core3 `TestAuditShowModalTwice`, `TestAuditShowMaximizedTwice`):
```go
parent := auditHeadless(t)
dlg := NewForm(); dlg.SetSize(200, 100)
dlg.ShowModal(parent); dlg.ShowModal(parent)   // e.g. a "show settings" command run twice
dlg.Close()
auditWnd(parent).Blocked() // true
```
- Actual: `windows: 1 -> 3`, `parent still blocked after the dialog closed; open windows 2`. ShowMaximized: `windows: 1 -> 2 -> after Close 1` (an orphan window is left).
- Expected: a second ShowModal on an open dialog activates it, or does nothing. ShowMaximized on a shown form maximizes the existing window.
- Fix: in both, when `c.wnd != nil && !c.closed`, reuse the window (Show/Activate/MaximizeWindow) instead of calling createWindow. Or have createWindow close an existing open window first.

## 8. Context menus shown and closed stay in the global widget registry (and keep action handlers) until the form closes
- Severity: medium (unbounded memory growth in a long-lived main window that builds a menu per right click; every Action.OnChanged emit also calls all the dead items)
- File: ui/widget.go:2267-2285 (AppendPopupWidget registers the popup); CloseTopPopup/CloseAfterPopupWidget (ui/widget.go:2287-2352) never unregister it. Also ui/action.go linkItem (the link stays made while registered).
- Root cause: AppendPopupWidget calls `registerWidget(w)` and `attachToForm(w, form)`. Closing the popup only drops it from PopupWidgets. The menu and its items stay in `allwidgets`, and the links stay made, until unregisterFormWidgets at form close.
- Reproducer (core1 `TestAuditContextMenuRegistryLeak`, core8 `TestAuditActionHandlersGrow`):
```go
form := newTestForm(t, NewLabel("x"))
a := NewAction("Copy", func() {})
for i := 0; i < 100; i++ {
	m := NewContextMenu(nil); m.AddItem("Copy", nil); m.AddItem("Paste", nil) // or m.AddAction(a)
	form.ShowContextMenu(m, 10, 10); form.CloseTopPopup()
}
```
- Actual: `registry before 3 after 303`; `100 handlers left on the action after 100 menus were shown and closed`
- Fix: when a popup is closed (CloseTopPopup, CloseAfterPopupWidget, closePopups), detach it unless it is owned elsewhere (a widget's contextMenu, a menu bar item's menu): `w.attachToForm(w, nil)`. A menu owned by a widget is re-attached on its next show anyway (ShowContextMenu/SetContextMenu attach it).

## 9. SetContextMenu: nil panics, and a replaced menu leaks
- Severity: medium (nil panic) / low (leak)
- File: ui/widget.go:3238-3241
- Root cause: `c.contextMenu.attachToForm(...)` is called on a nil menu, so `SetContextMenu(nil)` (the natural way to take a menu away) panics even on a widget that is not in a form. The old menu is never detached: it stays registered, with its links, while the form lives.
- Reproducer (core2 `TestAuditSetContextMenuNil`, `TestAuditReplaceContextMenuLeak`):
```go
l := NewLabel("x"); newTestForm(t, l)
l.SetContextMenu(NewContextMenu(nil)); l.SetContextMenu(nil) // panic
// and: 50x l.SetContextMenu(newMenuWithOneItem) -> registry 3 -> 103
```
- Actual: `nil pointer dereference ... (*Widget).SetContextMenu widget.go:3240`; `registry grew from 3 to 103`
- Fix: `if old := c.contextMenu; old != nil && old != menu { old.attachToForm(old, nil) }`, then attach the new one only when it is not nil.

## 10. MenuBar.AddMenuItem with a nil menu panics once the bar is in a form
- Severity: low
- File: ui/menu_bar.go:61-63
- Root cause: `menu.attachToForm(menu, c.form)` is called without a nil check. Elsewhere a nil menu is tolerated (MenuBar.attachToForm checks it, openMenu checks it).
- Reproducer (core9 `TestAuditMenuBarItemNilMenu`): `form.SetMenuBar(bar); bar.AddMenuItem("Help", nil)`
- Actual: `nil pointer dereference ... (*MenuBar).AddMenuItem menu_bar.go:62`
- Fix: `if c.form != nil && menu != nil`.

## 11. Form.RememberState / DockLayout.RememberState stop saving after the form is closed and shown again
- Severity: medium (tray-style apps lose the window and dock layout; rules.md explicitly supports showing a closed form again)
- File: ui/form.go:939-946 (runClosing clears `c.closing`), ui/form.go:419 (Close sets `c.closing = nil`); ui/formstate.go:52 registers once; ui/dock.go:281-285 (`c.saveHooked != c.form` prevents re-hooking on the same form)
- Root cause: the closing hooks run once and are then dropped. RememberState and hookSave register them only once, so after `Show()` of a closed form nothing saves the state at the next close.
- Reproducer (core7 `TestAuditRememberStateReshown`, `TestAuditDockRememberStateReshown`):
```go
f := NewForm(); f.SetSize(400, 300); f.RememberState(s, "main")
f.Show(); f.Close()
f.Show(); auditWnd(f).UserResize(640, 480); f.Close()
s.Get("main", &st) // still 400x300
```
- Actual: `after the second close the saved state is {X:760 Y:390 Width:400 Height:300}, want 640x480`; dock: `saved dock state is map[Files:{Area:0 Shown:true}], want Files hidden`
- Fix: make the persistent hooks (RememberState, dock save) survive a close. Keep them in a separate list that is not cleared, or re-register them in reopen/createWindow. Reset `saveHooked` when the form closes. Keep one-shot hooks (RunAsync) as they are.

## 12. DockLayout: a corrupt saved area makes a dock disappear for good
- Severity: low (malformed settings file)
- File: ui/dock.go:268-277 (RememberState trusts `s.Area`), ui/dock.go:126-155 (rebuild only places areas 0..2)
- Root cause: `d.area` is taken from the settings without validation. rebuild never places a dock whose area is out of range, while `Shown()` and the toggle action stay true. The dock's own menu ("Dock Left"...) is in its invisible title bar, so the user cannot bring it back.
- Reproducer (core7 `TestAuditDockBadArea`): settings `{"Docks":{"Files":{"Area":7,"Shown":true}}}`, then `RememberState`, Show, `SetShown(false)`, `SetShown(true)`
- Actual: `dock reports shown (area 7) but its content is in no form: it can never be shown again`
- Fix: ignore (or clamp to the default) saved areas outside `DockLeft..DockBottom`. Validate in AddDock/MoveTo too.

## 13. The form keeps the accept/cancel button of a removed DialogButtonBox
- Severity: low/medium (Enter or Esc runs the OK/Cancel handler of a box that is no longer in the form)
- File: ui/boxes.go:225-247; ui/form.go:1553-1563; removal goes through Widget.SetParentWidgetId → base `attachToForm(c, nil)` (ui/widget.go:394-398), so the DialogButtonBox override is not called and nothing clears form.acceptButton/cancelButton. Removing a widget also never checks these two pointers in forgetWidgets (ui/form.go:1373).
- Reproducer (core8 `TestAuditRemovedAcceptButton`):
```go
ok := 0; box := NewDialogButtonBox(); box.AddOK(func() { ok++ })
form := newTestForm(t, box)
form.Panel().RemoveWidget(box)
form.processKeyDown(KeyEnter, KeyModifiers{}) // ok == 1
```
- Fix: in forgetWidgets, clear `acceptButton`/`cancelButton` when they are among the removed widgets.

## 14. Toast: no line limit, so a popup window millions of pixels tall
- Severity: medium (a toast of an error text or log can allocate GBs for the popup's backing buffer, an OOM or X error; ShowMessageBox caps at 20 lines, toasts do not)
- File: ui/toast.go:84-95 (`t.h = len(t.lines)*ThemeLineHeight() + ...`), shown with `popup.ShowAt(..., t.w, t.h)` at ui/toast.go:136. The tooltip has the same unbounded size (ui/tooltip.go:107-113, no wrap and no cap), but its text normally comes from the app.
- Reproducer (core8 `TestAuditToastHuge`): `f.ShowToast(strings.Repeat("line\n", 100000), ToastError)`
- Actual: `toast 57x2000040, 100001 lines` → `popup height 2000040 px (434 MB of pixels for its window)`; on a scale-2 screen that is 4x.
- Fix: cap the lines like newMessageBoxForm (`messageBoxMaxLines`, with an ellipsis on the last line), and cap the tooltip size to the screen work area.

## 15. ShowMessageBox* / ShowQuestion* panic for a parent widget that is not in a form
- Severity: low
- File: ui/messagebox.go:184-190 (showMessageBoxForm)
- Root cause: when `parentWidget != nil` but `parentWidget.Form() == nil`, it calls `ShowModal(nil)`, which panics. runOnParent in the same file handles `Form() == nil` explicitly, so the case is expected.
- Reproducer (core9 `TestAuditMessageBoxWidgetNoForm`): `ShowMessageBox(NewLabel("x"), "T", "text")`
- Actual: `panic: parent form cannot be nil for ShowModal ... showMessageBoxForm messagebox.go:187`
- Fix: `if parentWidget != nil && parentWidget.Form() != nil { ShowModal } else { Show }`.

---

### Checked, no bug found
- Signal Connect/Disconnect during Emit (snapshot plus the removed flag) is OK.
- Timer Stop/Start inside its own callback, Close from a timer, and AfterFunc(0) are OK.
- Close from an OnClick handler is OK. RunAsync with the owner form closed: done is not called, and the form is freed.
- TabWidget insert/remove index handling, Splitter clamping and Message box Enter (answers once) are OK.

### Side notes (not reproduced here, outside or at the edge of scope)
- In headless mode, `platforms.IsUIThread()` is true for every goroutine and `RunOnUIThread` runs f on the calling goroutine (internal/platforms/nui.go:21-26, 53). So `InvokeSync` from a test's background goroutine runs concurrently with the test goroutine: a data race inside tests that use uitest.
- ui/atspi_linux.go:208-218: `formPath` puts every form into `b.forms` / `b.formIDs` and nothing ever deletes them. With accessibility on (Linux), every closed dialog stays in memory. Verified by reading: no `delete(` on these maps.
- In headless mode, the last created window/form stays referenced until another one is created. It is bounded (one form), and existing leak tests must show and close a second form before they check (seen in TestAuditLeakBisect, core5).
