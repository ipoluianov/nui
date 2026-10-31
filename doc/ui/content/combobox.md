# ComboBox

Dropdown list.

## Keyboard

Tab reaches the combo box. Up/Down pick the previous or the next item in place, Home/End the
first or the last one; Space, F4 or Alt+Down open the list. In the open list Up/Down/Home/End
choose an item, Enter or Space picks it, Esc closes the list without a change. Enter on the
closed combo box is left to the form (the accept button of a dialog).

## Create

```go
cb := ui.NewComboBox()
cb.AddItem("One", 1)
cb.AddItem("Two", 2)
cb.SetSelectedIndex(0)
```

## Read selection

```go
index := cb.SelectedIndex()
text := cb.SelectedItemText()
data := cb.SelectedItemData()
```

## Items

- `AddItem(text, data)`, `InsertItem(index, text, data)`, `RemoveItem(index)`, `Clear()`
- `SetItems([]string{...})`: replaces the items (no data); the item with the selected text stays selected, else the first
- `ItemCount()`, `ItemText(i)`, `ItemData(i)`, `SetItemText(i, text)` (e.g. for another language)
- `FindText(text)`, `FindData(data)`: the index, -1 if none

The first item is selected by default.

## Selection changes

```go
cb.OnSelectionChanged.Connect(func(index int) { // any change, also SetSelectedIndex
	fmt.Println("now", cb.ItemText(index))
})
cb.OnActivated.Connect(func(index int) { // the user picked an item, even the selected one
	apply(cb.SelectedItemData())
})
```

The old `SetOnSelectedIndexChanged(func())` still works: it is called when the user picks another item; `SetSelectedIndex` doesn't call it.
