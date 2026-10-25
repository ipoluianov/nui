# Actions, ToolBar, StatusBar

## Action

A command of the application — Save, Undo, Bold — with its text, icon, shortcut and state, shown by any number of menu items and toolbar buttons. They all show its state and run it: disable the action, and all its items are disabled.

```go
save := ui.NewAction("&Save", doSave).
	SetIcon(icons.Save(16)).
	SetShortcut("Mod+S").
	SetEnabledFunc(func() bool { return doc.Modified() })
wrap := ui.NewAction("&Word Wrap", applyWrap).SetCheckable(true)

fileMenu.AddAction(save)    // a menu item: text, icon, shortcut, state
toolbar.AddAction(save)     // a flat button: icon, tooltip "Save (Ctrl+S)"
viewMenu.AddAction(wrap)    // a check mark when checked
form.AddAction(save)        // the shortcut works when the action isn't in the menu bar
```

- `SetEnabled(bool)`, or `SetEnabledFunc(f)`: `f` is asked again before a menu opens and every few tenths of a second, so the items follow the application's state without code that updates them.
- `SetCheckable(true)`: running the action toggles `Checked()`; the menu items show a check mark, the buttons a pressed look.
- `Trigger()` runs it as a click would (nothing if disabled). `OnTriggered` and `OnChanged` are signals.

A menu item can be checkable without an action too: `item.SetCheckable(true)`, `item.SetChecked(...)`, `item.Checked()`.

## ToolBar

A row of flat buttons; the free width stays at its end.

```go
toolbar := ui.NewToolBar()
toolbar.AddAction(newAction)
toolbar.AddAction(saveAction)
toolbar.AddSeparator()
toolbar.AddAction(boldAction)
toolbar.Add(fontCombo)     // any widget
form.Panel().AddWidget(0, 0, toolbar)
```

An action without an icon is a text button.

## StatusBar

The line at the bottom of a window: a text at the left, sections at the right, messages shown for a while.

```go
status := ui.NewStatusBar()
position := status.AddLabel("Ln 1, Col 1") // position.SetText(...) later
status.Add(progressBar)                    // any widget
status.SetText("Ready")
status.ShowMessage("Saved", 3*time.Second) // then "Ready" is back
form.Panel().AddWidget(2, 0, status)
```
