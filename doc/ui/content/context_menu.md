# ContextMenu

Popup menu widget.

## Create and show

```go
menu := ui.NewContextMenu(form.Panel())
menu.AddItem("Open", func() { /* ... */ })
menu.AddItem("Quit", func() { form.Close() })
menu.ShowMenu(100, 100)
```

## Submenus

```go
sub := ui.NewContextMenu(form.Panel())
sub.AddItem("Item", func() {})
menu.AddItemWithSubmenu("More", sub)
```


## Icons

```go
menu.AddItem("Edit", onEdit).SetImage(editIcon)
menu.AddItem("Remove", onRemove).SetImage(removeIcon)
```

An icon is drawn left of the text; images larger than `ui.ContextMenuItemIconSize` (16px)
are scaled down, so for crisp icons pass 16x16 images. When any item has an icon,
the texts of all the items are aligned after the icon column.

## Showing a menu from code

```go
menu := ui.NewContextMenu(nil)
menu.AddItem("Recent dir", onClick)
form.ShowContextMenu(menu, x, y) // x, y in the form's client area
```

The first item is active, so the keyboard (Up/Down, Enter, Esc) works at once.

## Keyboard

The Menu key and Shift+F10 open, from the keyboard, the menu a right click would: the one set
with `SetContextMenu` on the focused widget or on its nearest parent that has one. It opens
under the focused widget (a table: under its current cell) with the first item chosen. The key
goes to the application first: a widget's `SetOnKeyDown`, `SetOnGlobalKeyDown` and a shortcut
added with `form.AddShortcut` take it before the menu opens.

In an open menu Up/Down choose an item, Home/End the first or the last one, Enter or Space
click it, Right/Left open and close a submenu, a mnemonic letter clicks its item, Esc closes
the menu. Other keys stay in the menu and don't reach the widget behind it; shortcuts with
Ctrl (Cmd) still work.
