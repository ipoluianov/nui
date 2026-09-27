# ComboBox

Dropdown list.

## Create

```go
cb := ui.NewComboBox()
cb.AddItem("One", 1)
cb.AddItem("Two", 2)
cb.SetSelectedIndex(0)
```

## Read selection

```go
text := cb.SelectedItemText()
data := cb.SelectedItemData()
```

## Notes

Current `ComboBox` opens a popup on left click. It does not expose a dedicated `OnChanged` callback yet; update your UI after selection if needed.

