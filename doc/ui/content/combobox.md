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

## Selection changes

```go
cb.SetOnSelectedIndexChanged(func() {
	fmt.Println("picked", cb.SelectedItemText())
})
```

Called when the user picks another item in the popup; `SetSelectedIndex` doesn't call it.
`ItemCount()` returns the number of items.

