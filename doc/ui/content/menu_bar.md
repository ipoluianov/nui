# MenuBar

Main menu of a form: a row of titles at the top of the window. Each title opens a
[ContextMenu](context_menu.md) that drops down below it, so the menus support everything
context menus do: separators, submenus, icons, hidden items, `SetOnShow`.

## Create

```go
bar := ui.NewMenuBar()

file := bar.AddMenu("File") // returns *ui.ContextMenu
file.AddItem("Open", onOpen)
file.AddItemWithSubmenu("Open Recent", recentMenu)
file.AddSeparator()
file.AddItem("Exit", func() { form.Close() })

edit := bar.AddMenu("Edit")
edit.AddItem("Copy", onCopy)

form.SetMenuBar(bar) // nil removes it
```

`form.Panel()` is laid out below the bar; widgets are added to it as usual.

## Behavior

- A click on a title opens its menu; a click on the title again (or anywhere outside) closes it.
- While a menu is open, moving the mouse to another title opens that title's menu.
- A menu that doesn't fit below the title opens above it.
- A click on the bar keeps the keyboard focus where it is, so menu items can act on the focused widget.
- Escape closes the open menu.

## Titles

`AddMenuItem(text, menu)` adds a title for an existing menu and returns `*ui.MenuBarItem`:

```go
item := bar.AddMenuItem("View", viewMenu)
item.SetTextFunc(func() string { return tr("View") }) // follows the language
item.SetVisible(false)                                // the titles after it move over
item.SetEnabled(false)                                // grayed out, doesn't open
```

`bar.Items()` returns the titles, `item.Menu()` the menu of a title.

## Keyboard: mnemonics and shortcuts

```go
file := bar.AddMenu("&File")                            // Alt+F opens it
file.AddItem("&Save", onSave).SetShortcut("Mod+S")      // S in the open menu, Ctrl+S (Cmd+S on macOS) anywhere
file.AddItem("Save &As...", onSaveAs).SetShortcut("Ctrl+Shift+S")
file.AddItem("E&xit", onExit)
form.AddShortcut("F5", refresh)                         // a shortcut without a menu item
```

- `&` before a letter or digit marks the **mnemonic**: Alt+letter opens a main menu; in an open
  menu the letter chooses the item. The letters are underlined (in the bar while Alt is held).
  `&&` is a `&`; a `&` before a space or a sign stays as it is ("Drag & Drop").
- `SetShortcut` runs the item without opening the menu and shows the keys on its right:
  `"Ctrl+S"`, `"Ctrl+Shift+Z"`, `"Alt+F4"`, `"F5"`, `"Delete"`; `"Mod"` is Cmd on macOS and Ctrl
  elsewhere. The shortcuts match the key's place on the keyboard, so they work in any layout.
  Disabled and hidden items don't run.
- In an open menu: Up/Down choose the item, Enter or Space clicks it, Right opens a submenu (or
  the next main menu), Left closes a submenu (or opens the previous main menu), Escape closes.
- A focused text field keeps its own keys (Ctrl+C/V/X/A/Z/Y, Delete, the arrows...): a menu item
  with the shortcut Ctrl+C runs only when no text field has the focus.
- Keys of the numeric keypad: `NumPlus`, `NumMinus`, `NumMultiply`, `NumDivide` (shown as
  "Num +", ...); punctuation keys by their characters: `Ctrl+\`, `Ctrl+.`, `Alt+/`.
- `ui.ParseShortcut(s)` / `ui.MustParseShortcut(s)` give a `ui.Shortcut` (`String()`, `Matches`).
