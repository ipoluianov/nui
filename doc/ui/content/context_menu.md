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
