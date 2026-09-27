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
